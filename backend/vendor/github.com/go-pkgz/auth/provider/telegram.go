package provider

//go:generate moq -out telegram_moq_test.go . TelegramAPI

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
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-pkgz/auth/logger"
	authtoken "github.com/go-pkgz/auth/token"
	"github.com/go-pkgz/repeater"
	"github.com/go-pkgz/rest"
	"github.com/pkg/errors"
)

// TelegramHandler implements login via telegram
type TelegramHandler struct {
	logger.L

	ProviderName         string
	ErrorMsg, SuccessMsg string

	TokenService TokenService
	AvatarSaver  AvatarSaver
	Telegram     TelegramAPI

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
var tgPollInterval = time.Second

// Run starts processing login requests sent in Telegram
// Blocks caller
func (th *TelegramHandler) Run(ctx context.Context) error {
	// Initialization
	info, err := th.Telegram.BotInfo(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to fetch bot info")
	}

	th.requests.Lock()
	th.requests.data = make(map[string]tgAuthRequest)
	th.requests.Unlock()

	th.username = info.Username

	ticker := time.NewTicker(tgPollInterval)

	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return ctx.Err()
		case <-ticker.C:
			err := th.processUpdates(ctx)
			if err != nil {
				th.Logf("Error while processing updates: %v", err)
				continue
			}

			// Purge expired requests
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

// processUpdates processes a batch of updates from telegram servers
// Returns offset for subsequent calls
func (th *TelegramHandler) processUpdates(ctx context.Context) error {
	updates, err := th.Telegram.GetUpdates(ctx)
	if err != nil {
		return err
	}

	for _, update := range updates.Result {
		if update.Message.Chat.Type != "private" {
			continue
		}

		if !strings.HasPrefix(update.Message.Text, "/start ") {
			err := th.Telegram.Send(ctx, update.Message.Chat.ID, th.ErrorMsg)
			if err != nil {
				th.Logf("failed to notify telegram peer: %v", err)
			}
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

	return nil
}

// Name of the provider
func (th *TelegramHandler) Name() string { return th.ProviderName }

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
		}

		th.requests.Lock()
		th.requests.data[token] = tgAuthRequest{
			expires: time.Now().Add(tgAuthRequestLifetime),
		}
		th.requests.Unlock()

		rest.RenderJSON(w, struct {
			Token string `json:"token"`
			Bot   string `json:"bot"`
		}{token, th.username})

		return
	}

	// GET /login?token=blah
	th.requests.RLock()
	authRequest, ok := th.requests.data[queryToken]
	th.requests.RUnlock()

	if !ok || time.Now().After(authRequest.expires) {
		th.requests.Lock()
		delete(th.requests.data, queryToken)
		th.requests.Unlock()

		rest.SendErrorJSON(w, r, nil, http.StatusNotFound, nil, "request expired")
		return
	}

	if !authRequest.confirmed {
		rest.SendErrorJSON(w, r, nil, http.StatusNotFound, nil, "request not yet confirmed")
		return
	}

	u, err := setAvatar(th.AvatarSaver, *authRequest.user, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		rest.SendErrorJSON(w, r, th.L, http.StatusInternalServerError, err, "failed to save avatar to proxy")
		return
	}

	claims := authtoken.Claims{
		User: &u,
		StandardClaims: jwt.StandardClaims{
			Id:     queryToken,
			Issuer: th.ProviderName,
		},
		SessionOnly: false, // TODO
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

// AuthHandler does nothing since we're don't have any callbacks
func (th *TelegramHandler) AuthHandler(w http.ResponseWriter, r *http.Request) {}

// LogoutHandler - GET /logout
func (th *TelegramHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
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
		return nil, errors.Wrap(err, "failed to fetch updates")
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

type botInfo struct {
	ID       int    `json:"id"`
	Name     string `json:"first_name"`
	Username string `json:"username"`
}

// BotInfo returns info about configured bot
func (tg *tgAPI) BotInfo(ctx context.Context) (*botInfo, error) {
	var resp = struct {
		Result *botInfo `json:"result"`
	}{}

	err := tg.request(ctx, "getMe", &resp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch bot info")
	}

	return resp.Result, nil
}

func (tg *tgAPI) request(ctx context.Context, method string, data interface{}) error {
	repeat := repeater.NewDefault(3, time.Millisecond*50)

	return repeat.Do(ctx, func() error {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", tg.token, method)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return errors.Wrap(err, "failed to create request")
		}

		resp, err := tg.client.Do(req)
		if err != nil {
			return errors.Wrap(err, "failed to send request")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return tg.parseError(resp.Body)
		}

		if err = json.NewDecoder(resp.Body).Decode(data); err != nil {
			return errors.Wrap(err, "failed to decode json response")
		}

		return nil
	})
}

func (tg *tgAPI) parseError(r io.Reader) error {
	var tgErr = struct {
		Description string `json:"description"`
	}{}

	if err := json.NewDecoder(r).Decode(&tgErr); err != nil {
		return errors.Wrap(err, "can't decode error")
	}

	return errors.Errorf("telegram returned error: %v", tgErr.Description)
}
