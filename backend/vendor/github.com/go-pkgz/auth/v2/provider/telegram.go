package provider

//go:generate moq --out telegram_moq_test.go . TelegramAPI

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-pkgz/repeater/v2"
	"github.com/go-pkgz/rest"
	"github.com/golang-jwt/jwt/v5"

	"github.com/go-pkgz/auth/v2/avatar"
	"github.com/go-pkgz/auth/v2/logger"
	authtoken "github.com/go-pkgz/auth/v2/token"
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
	// initialization
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
		if !ok { // no such token
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

		// avatarURL embeds the bot token in its path
		// (https://api.telegram.org/file/bot{TOKEN}/...). Never store it in
		// User.Picture: it would leak through avatar.Proxy.Put logs and, when
		// no avatar saver is configured, into the JWT and on to the client.
		// Fetch the bytes here and hand them to the avatar store directly.
		picture := th.saveTelegramAvatar(ctx, id, avatarURL)

		authRequest.confirmed = true
		authRequest.user = &authtoken.User{
			ID:      id,
			Name:    update.Message.Chat.Name,
			Picture: picture,
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
		// generate and send token
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

	// when saveTelegramAvatar already populated Picture with a local proxy
	// URL, skip the URL-fetching avatar pipeline. Letting setAvatar run
	// here would have it call Proxy.Put which re-fetches Picture; in
	// split-DNS / unreachable-internal-Opts.URL deployments that fetch
	// fails and the identicon fallback would silently overwrite the
	// stored Telegram bytes with an identicon at the same store path.
	u := *authUser
	if u.Picture == "" {
		u, err = setAvatar(th.AvatarSaver, *authUser, &http.Client{Timeout: 5 * time.Second})
		if err != nil {
			rest.SendErrorJSON(w, r, th.L, http.StatusInternalServerError, err, "failed to save avatar to proxy")
			return
		}
	}

	claims := authtoken.Claims{
		User: &u,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  []string{r.URL.Query().Get("site")},
			ID:        queryToken,
			Issuer:    th.ProviderName,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Minute)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
		},
		SessionOnly: false, // TODO review?
		AuthProvider: &authtoken.AuthProvider{
			Name: th.ProviderName,
		},
	}

	if _, err := th.TokenService.Set(w, claims); err != nil {
		rest.SendErrorJSON(w, r, th.L, http.StatusInternalServerError, err, "failed to set token")
		return
	}

	rest.RenderJSON(w, claims.User)

	// delete request
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

	// identifier of the first update to be requested.
	// should be equal to LastSeenUpdateID + 1
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
	// get profile pictures
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

	// user does not have profile picture set or it is hidden in privacy settings
	if len(profilePhotos.Result.Photos) == 0 || len(profilePhotos.Result.Photos[0]) == 0 {
		return "", nil
	}

	// get max possible picture size
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

func (tg *tgAPI) request(ctx context.Context, method string, data any) error {
	return repeater.NewFixed(3, time.Millisecond*50).Do(ctx, func() error {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", tg.token, method)

		req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", redactBotURLInErr(err))
		}

		resp, err := tg.client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", redactBotURLInErr(err))
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

// avatarContentSaver matches the optional method on AvatarSaver implementations
// that can store already-fetched bytes (avatar.Proxy provides one). Used by the
// Telegram provider to avoid passing a bot-token-bearing URL through the
// URL-fetching avatar pipeline.
type avatarContentSaver interface {
	PutContent(userID string, content io.Reader) (string, error)
}

// saveTelegramAvatar fetches the avatar bytes from a bot-token-bearing Telegram
// URL and stores them via th.AvatarSaver, returning a clean local proxy URL.
// The bot URL is consumed entirely inside this function so it never reaches
// User.Picture, JWT claims, or any debug log of the user object. Returns ""
// when the avatar cannot be saved (no URL, no compatible saver, or fetch
// failure) — the caller treats that as "no picture" and the avatar pipeline
// falls back to identicon as usual.
func (th *TelegramHandler) saveTelegramAvatar(ctx context.Context, userID, avatarURL string) string {
	if avatarURL == "" {
		return ""
	}
	// guard against typed-nil *avatar.Proxy. auth.go skips initializing
	// res.avatarProxy when Opts.AvatarStore is unset, so AvatarSaver can be
	// a non-nil interface wrapping a nil *avatar.Proxy. The type assertion
	// below would still succeed (interface satisfaction is structural), but
	// PutContent on a nil receiver panics on the first p.Store deref.
	if th.AvatarSaver == nil || th.AvatarSaver == (*avatar.Proxy)(nil) {
		th.Logf("[WARN] telegram avatar dropped: AvatarSaver is not configured")
		return ""
	}
	saver, ok := th.AvatarSaver.(avatarContentSaver)
	if !ok {
		// fallback intentionally drops the picture rather than expose the bot
		// token; warn so operators can wire a content-aware saver if they want
		// telegram avatars saved
		th.Logf("[WARN] telegram avatar dropped: configured AvatarSaver does not support direct content save")
		return ""
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, avatarURL, http.NoBody)
	if err != nil {
		th.Logf("[WARN] telegram avatar fetch request build failed: %v", redactBotURLInErr(err))
		return ""
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		th.Logf("[WARN] telegram avatar fetch failed: %v", redactBotURLInErr(err))
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		th.Logf("[WARN] telegram avatar fetch returned status %d", resp.StatusCode)
		return ""
	}
	// cap body size to protect PutContent from an unbounded upstream response.
	// Telegram caps photos at 5 MiB; 10 MiB is generous headroom while still
	// bounding worst-case memory.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxTelegramAvatarSize+1))
	if err != nil {
		th.Logf("[WARN] telegram avatar read failed: %v", err)
		return ""
	}
	if int64(len(body)) > maxTelegramAvatarSize {
		th.Logf("[WARN] telegram avatar dropped: body exceeds %d bytes", maxTelegramAvatarSize)
		return ""
	}
	picture, err := saver.PutContent(userID, bytes.NewReader(body))
	if err != nil {
		th.Logf("[WARN] telegram avatar save failed: %v", err)
		return ""
	}
	return picture
}

const maxTelegramAvatarSize = 10 << 20

// botTokenInURLPath matches the bot-token segment of a Telegram URL anchored
// between path slashes ("/botTOKEN/..."). The leading and trailing slashes
// avoid matching unrelated identifiers that happen to start with "bot" (e.g.
// the username "botFather" appearing elsewhere in a log line). Replacement
// preserves the slashes via "/bot<redacted>/" to keep surrounding URL
// structure intact for diagnostics.
var botTokenInURLPath = regexp.MustCompile(`/bot[A-Za-z0-9:_-]+/`)

// redactBotURLInErr returns the error with any embedded Telegram bot-token
// segment in URL paths replaced by "bot<redacted>". net/http's *url.Error
// stringifies as `Op "URL": Err`, so a transport failure on a URL like
// https://api.telegram.org/file/bot<TOKEN>/... otherwise prints the token
// verbatim.
func redactBotURLInErr(err error) error {
	if err == nil {
		return nil
	}
	redacted := botTokenInURLPath.ReplaceAllString(err.Error(), "/bot<redacted>/")
	if redacted == err.Error() {
		return err
	}
	return errors.New(redacted)
}
