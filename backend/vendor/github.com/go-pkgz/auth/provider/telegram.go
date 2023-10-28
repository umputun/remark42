package provider

//go:generate moq --out telegram_moq_test.go . TelegramAPI

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-pkgz/repeater"
	"github.com/go-pkgz/rest"
	"github.com/golang-jwt/jwt"

	"github.com/go-pkgz/auth/logger"
	authtoken "github.com/go-pkgz/auth/token"
)

// TelegramHandler implements login via telegram
type TelegramHandler struct {
	logger.L

	ProviderName         string
	ErrorMsg, SuccessMsg string

	TokenService TokenService
	AvatarSaver  AvatarSaver
	Telegram     TelegramAPI

	run      int32  // non-zero if Run goroutine has started
	username string // bot username
	requests struct {
		sync.RWMutex
		data map[string]tgAuthRequest
	}
}

type tgAuthRequest struct {
	confirmed bool // whether login request has been confirmed and user info set
	expires   time.Time
	user      *authtoken.User
}

// TelegramAPI is used for interacting with telegram API
type TelegramAPI interface {
	GetUpdates(ctx context.Context) (*telegramUpdate, error)
	Avatar(ctx context.Context, userID int) (string, error)
	Send(ctx context.Context, id int, text string) error
	BotInfo(ctx context.Context) (*botInfo, error)
}

// changed in tests
var apiPollInterval = time.Second * 5        // interval to check updates from Telegram API and answer to users
var expiredCleanupInterval = time.Minute * 5 // interval to check and clean up expired notification requests

// Run starts processing login requests sent in Telegram
// Blocks caller
func (th *TelegramHandler) Run(ctx context.Context) error {
	// Initialization
	atomic.AddInt32(&th.run, 1)
	info, err := th.Telegram.BotInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch bot info: %w", err)
	}
	th.username = info.Username

	th.requests.Lock()
	th.requests.data = make(map[string]tgAuthRequest)
	th.requests.Unlock()

	processUpdatedTicker := time.NewTicker(apiPollInterval)
	cleanupTicker := time.NewTicker(expiredCleanupInterval)

	for {
		select {
		case <-ctx.Done():
			processUpdatedTicker.Stop()
			cleanupTicker.Stop()
			atomic.AddInt32(&th.run, -1)
			return ctx.Err()
		case <-processUpdatedTicker.C:
			updates, err := th.Telegram.GetUpdates(ctx)
			if err != nil {
				th.Logf("Error while getting telegram updates: %v", err)
				continue
			}
			th.processUpdates(ctx, updates)
		case <-cleanupTicker.C:
			now := time.Now()
			th.requests.Lock()
			for key, req := range th.requests.data {
				if now.After(req.expires) {
					delete(th.requests.data, key)
				}
			}
			th.requests.Unlock()
		}
	}
}

// telegramUpdate contains update information, which is used from whole telegram API response
type telegramUpdate struct {
	Result []struct {
		UpdateID int `json:"update_id"`
		Message  struct {
			Chat struct {
				ID   int    `json:"id"`
				Name string `json:"first_name"`
				Type string `json:"type"`
			} `json:"chat"`
			Text string `json:"text"`
		} `json:"message"`
	} `json:"result"`
}

// ProcessUpdate is alternative to Run, it processes provided plain text update from Telegram
// so that caller could get updates and send it not only there but to multiple sources
func (th *TelegramHandler) ProcessUpdate(ctx context.Context, textUpdate string) error {
	if atomic.LoadInt32(&th.run) != 0 {
		return fmt.Errorf("Run goroutine should not be used with ProcessUpdate")
	}
	defer func() {
		// as Run goroutine is not running, clean up old requests on each update
		// even if we hit json decode error
		now := time.Now()
		th.requests.Lock()
		for key, req := range th.requests.data {
			if now.After(req.expires) {
				delete(th.requests.data, key)
			}
		}
		th.requests.Unlock()
	}()
	// initialize requests.data as usually it's initialized in Run
	th.requests.Lock()
	if th.requests.data == nil {
		th.requests.data = make(map[string]tgAuthRequest)
	}
	th.requests.Unlock()
	var updates telegramUpdate
	if err := json.Unmarshal([]byte(textUpdate), &updates); err != nil {
		return fmt.Errorf("failed to decode provided telegram update: %w", err)
	}
	th.processUpdates(ctx, &updates)
	return nil
}

// processUpdates processes a batch of updates from telegram servers
// Returns offset for subsequent calls
func (th *TelegramHandler) processUpdates(ctx context.Context, updates *telegramUpdate) {
	for _, update := range updates.Result {
		if update.Message.Chat.Type != "private" {
			continue
		}

		if !strings.HasPrefix(update.Message.Text, "/start ") {
			continue
		}

		token := strings.TrimPrefix(update.Message.Text, "/start ")

		th.requests.RLock()
		authRequest, ok := th.requests.data[token]
		if !ok { // No such token
			th.requests.RUnlock()
			err := th.Telegram.Send(ctx, update.Message.Chat.ID, th.ErrorMsg)
			if err != nil {
				th.Logf("failed to notify telegram peer: %v", err)
			}
			continue
		}
		th.requests.RUnlock()

		avatarURL, err := th.Telegram.Avatar(ctx, update.Message.Chat.ID)
		if err != nil {
			th.Logf("failed to get user avatar: %v", err)
			continue
		}

		id := th.ProviderName + "_" + authtoken.HashID(sha1.New(), fmt.Sprint(update.Message.Chat.ID))

		authRequest.confirmed = true
		authRequest.user = &authtoken.User{
			ID:      id,
			Name:    update.Message.Chat.Name,
			Picture: avatarURL,
		}

		th.requests.Lock()
		th.requests.data[token] = authRequest
		th.requests.Unlock()

		err = th.Telegram.Send(ctx, update.Message.Chat.ID, th.SuccessMsg)
		if err != nil {
			th.Logf("failed to notify telegram peer: %v", err)
		}
	}
}

// addToken adds token
func (th *TelegramHandler) addToken(token string, expires time.Time) error {
	th.requests.Lock()
	if th.requests.data == nil {
		th.requests.Unlock()
		return fmt.Errorf("run goroutine is not running")
	}
	th.requests.data[token] = tgAuthRequest{
		expires: expires,
	}
	th.requests.Unlock()
	return nil
}

// checkToken verifies incoming token, returns the user address if it's confirmed and empty string otherwise
func (th *TelegramHandler) checkToken(token string) (*authtoken.User, error) {
	th.requests.RLock()
	authRequest, ok := th.requests.data[token]
	th.requests.RUnlock()

	if !ok {
		return nil, fmt.Errorf("request is not found")
	}

	if time.Now().After(authRequest.expires) {
		th.requests.Lock()
		delete(th.requests.data, token)
		th.requests.Unlock()
		return nil, fmt.Errorf("request expired")
	}

	if !authRequest.confirmed {
		return nil, fmt.Errorf("request is not verified yet")
	}

	return authRequest.user, nil
}

// Name of the provider
func (th *TelegramHandler) Name() string { return th.ProviderName }

// String representation of the provider
func (th *TelegramHandler) String() string { return th.Name() }

// Default token lifetime. Changed in tests
var tgAuthRequestLifetime = time.Minute * 10

// LoginHandler generates and verifies login requests
func (th *TelegramHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	queryToken := r.URL.Query().Get("token")
	if queryToken == "" {
		// GET /login (No token supplied)
		// Generate and send token
		token, err := randToken()
		if err != nil {
			rest.SendErrorJSON(w, r, th.L, http.StatusInternalServerError, err, "failed to generate code")
			return
		}

		err = th.addToken(token, time.Now().Add(tgAuthRequestLifetime))
		if err != nil {
			rest.SendErrorJSON(w, r, th.L, http.StatusInternalServerError, err, "failed to process login request")
			return
		}

		// verify that we have a username, which is not set if Run was not used
		if th.username == "" {
			info, err := th.Telegram.BotInfo(r.Context())
			if err != nil {
				rest.SendErrorJSON(w, r, th.L, http.StatusInternalServerError, err, "failed to fetch bot username")
				return
			}
			th.username = info.Username
		}

		rest.RenderJSON(w, struct {
			Token string `json:"token"`
			Bot   string `json:"bot"`
		}{token, th.username})

		return
	}

	// GET /login?token=blah
	authUser, err := th.checkToken(queryToken)
	if err != nil {
		rest.SendErrorJSON(w, r, nil, http.StatusNotFound, err, err.Error())
		return
	}

	u, err := setAvatar(th.AvatarSaver, *authUser, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		rest.SendErrorJSON(w, r, th.L, http.StatusInternalServerError, err, "failed to save avatar to proxy")
		return
	}

	claims := authtoken.Claims{
		User: &u,
		StandardClaims: jwt.StandardClaims{
			Audience:  r.URL.Query().Get("site"),
			Id:        queryToken,
			Issuer:    th.ProviderName,
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
			NotBefore: time.Now().Add(-1 * time.Minute).Unix(),
		},
		SessionOnly: false, // TODO review?
	}

	if _, err := th.TokenService.Set(w, claims); err != nil {
		rest.SendErrorJSON(w, r, th.L, http.StatusInternalServerError, err, "failed to set token")
		return
	}

	rest.RenderJSON(w, claims.User)

	// Delete request
	th.requests.Lock()
	defer th.requests.Unlock()
	delete(th.requests.data, queryToken)
}

// AuthHandler does nothing since we don't have any callbacks
func (th *TelegramHandler) AuthHandler(_ http.ResponseWriter, _ *http.Request) {}

// LogoutHandler - GET /logout
func (th *TelegramHandler) LogoutHandler(w http.ResponseWriter, _ *http.Request) {
	th.TokenService.Reset(w)
}

// tgAPI implements TelegramAPI
type tgAPI struct {
	logger.L
	token  string
	client *http.Client

	// Identifier of the first update to be requested.
	// Should be equal to LastSeenUpdateID + 1
	// See https://core.telegram.org/bots/api#getupdates
	updateOffset int
}

// NewTelegramAPI returns initialized TelegramAPI implementation
func NewTelegramAPI(token string, client *http.Client) TelegramAPI {
	return &tgAPI{
		client: client,
		token:  token,
	}
}

// GetUpdates fetches incoming updates
func (tg *tgAPI) GetUpdates(ctx context.Context) (*telegramUpdate, error) {
	url := `getUpdates?allowed_updates=["message"]`
	if tg.updateOffset != 0 {
		url += fmt.Sprintf("&offset=%d", tg.updateOffset)
	}

	var result telegramUpdate

	err := tg.request(ctx, url, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updates: %w", err)
	}

	for _, u := range result.Result {
		if u.UpdateID >= tg.updateOffset {
			tg.updateOffset = u.UpdateID + 1
		}
	}

	return &result, err
}

// Send sends a message to telegram peer
func (tg *tgAPI) Send(ctx context.Context, id int, msg string) error {
	url := fmt.Sprintf("sendMessage?chat_id=%d&text=%s", id, neturl.PathEscape(msg))
	return tg.request(ctx, url, &struct{}{})
}

// Avatar returns URL to user avatar
func (tg *tgAPI) Avatar(ctx context.Context, id int) (string, error) {
	// Get profile pictures
	url := fmt.Sprintf(`getUserProfilePhotos?user_id=%d`, id)

	var profilePhotos = struct {
		Result struct {
			Photos [][]struct {
				ID string `json:"file_id"`
			} `json:"photos"`
		} `json:"result"`
	}{}

	if err := tg.request(ctx, url, &profilePhotos); err != nil {
		return "", err
	}

	// User does not have profile picture set or it is hidden in privacy settings
	if len(profilePhotos.Result.Photos) == 0 || len(profilePhotos.Result.Photos[0]) == 0 {
		return "", nil
	}

	// Get max possible picture size
	last := len(profilePhotos.Result.Photos[0]) - 1
	fileID := profilePhotos.Result.Photos[0][last].ID
	url = fmt.Sprintf(`getFile?file_id=%s`, fileID)

	var fileMetadata = struct {
		Result struct {
			Path string `json:"file_path"`
		} `json:"result"`
	}{}

	if err := tg.request(ctx, url, &fileMetadata); err != nil {
		return "", err
	}

	avatarURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", tg.token, fileMetadata.Result.Path)

	return avatarURL, nil
}

// botInfo structure contains information about telegram bot, which is used from whole telegram API response
type botInfo struct {
	Username string `json:"username"`
}

// BotInfo returns info about configured bot
func (tg *tgAPI) BotInfo(ctx context.Context) (*botInfo, error) {
	var resp = struct {
		Result *botInfo `json:"result"`
	}{}

	err := tg.request(ctx, "getMe", &resp)
	if err != nil {
		return nil, err
	}
	if resp.Result == nil {
		return nil, fmt.Errorf("received empty result")
	}

	return resp.Result, nil
}

func (tg *tgAPI) request(ctx context.Context, method string, data interface{}) error {
	return repeater.NewDefault(3, time.Millisecond*50).Do(ctx, func() error {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", tg.token, method)

		req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := tg.client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close() //nolint gosec // we don't care about response body

		if resp.StatusCode != http.StatusOK {
			return tg.parseError(resp.Body, resp.StatusCode)
		}

		if err = json.NewDecoder(resp.Body).Decode(data); err != nil {
			return fmt.Errorf("failed to decode json response: %w", err)
		}

		return nil
	})
}

func (tg *tgAPI) parseError(r io.Reader, statusCode int) error {
	tgErr := struct {
		Description string `json:"description"`
	}{}
	if err := json.NewDecoder(r).Decode(&tgErr); err != nil {
		return fmt.Errorf("unexpected telegram API status code %d", statusCode)
	}
	return fmt.Errorf("unexpected telegram API status code %d, error: %q", statusCode, tgErr.Description)
}
