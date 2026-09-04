package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater/v2"
	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
)

// TelegramParams contain settings for telegram notifications
type TelegramParams struct {
	Token                string        // token for telegram bot API interactions
	Timeout              time.Duration // http client timeout
	ErrorMsg, SuccessMsg string        // messages for successful and unsuccessful subscription requests to bot

	// APIURL points the client at something other than the public bot API, for a proxy standing in
	// front of Telegram or a test double answering as it. Empty means the public API. Every request
	// carries the bot token in its path, so this is validated rather than trusted: see
	// validateTelegramBaseURL for what is refused and why.
	APIURL string

	apiPrefix string // derived from APIURL, or the public API
}

// Telegram notifications client
type Telegram struct {
	TelegramParams

	// identifier of the first update to be requested.
	// should be equal to LastSeenUpdateID + 1
	// See https://core.telegram.org/bots/api#getupdates
	updateOffset           int
	apiPollInterval        time.Duration // interval to check updates from Telegram API and answer to users
	expiredCleanupInterval time.Duration // interval to check and clean up expired notification requests
	username               string        // bot username
	updates                struct {
		sync.RWMutex
		running bool // set while the Run goroutine is active, ProcessUpdate is not allowed then
	}
	requests struct {
		sync.Mutex
		data map[string]tgAuthRequest
	}
}

// telegramMsg is used to send message through Telegram bot API
type telegramMsg struct {
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type tgAuthRequest struct {
	confirmed  bool // whether login request has been confirmed and user info set
	expires    time.Time
	telegramID string
	user       string
	site       string
}

// TelegramBotInfo structure contains information about telegram bot, which is used from whole telegram API response
type TelegramBotInfo struct {
	Username string `json:"username"`
}

const telegramTimeOut = 5000 * time.Millisecond
const telegramAPIPrefix = "https://api.telegram.org/bot"
const tgPollInterval = time.Second * 5
const tgCleanupInterval = time.Minute * 5

// NewTelegram makes telegram bot for notifications
func NewTelegram(params TelegramParams) (*Telegram, error) {
	res := Telegram{TelegramParams: params}

	switch {
	case res.APIURL != "":
		base, err := validateTelegramBaseURL(res.APIURL)
		if err != nil {
			return nil, err
		}
		// the "bot" literal belongs to the API's own shape rather than to the operator's base, so
		// it is appended here: a caller passes https://proxy.example.com and the URLs come out as
		// https://proxy.example.com/bot<token>/<method>, which is what Telegram itself serves
		res.apiPrefix = base + "/bot"
	case res.apiPrefix == "":
		res.apiPrefix = telegramAPIPrefix
	default:
		// a prefix set directly, which the package's own tests do
	}
	if res.Timeout == 0 {
		res.Timeout = telegramTimeOut
	}

	if res.SuccessMsg == "" {
		res.SuccessMsg = "✅ You have successfully authenticated, check the web!"
	}

	res.apiPollInterval = tgPollInterval
	res.expiredCleanupInterval = tgCleanupInterval
	log.Printf("[DEBUG] create new telegram notifier for api=%s, timeout=%s", res.apiPrefix, res.Timeout)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	botInfo, err := res.botInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("can't retrieve bot info from Telegram API: %w", err)
	}
	res.username = botInfo.Username

	res.requests.data = make(map[string]tgAuthRequest)

	return &res, nil
}

// Send sends provided message to Telegram chat, with `parseMode` parsed from destination field (Markdown by default)
// with "telegram:" schema same way "mailto:" schema is constructed.
//
// Example:
//
// - telegram:channel
// - telegram:chatID // chatID is a number, like `-1001480738202`
// - telegram:channel?parseMode=HTML
func (t *Telegram) Send(ctx context.Context, destination, text string) error {
	chatID, parseMode, err := t.parseDestination(destination)
	if err != nil {
		return fmt.Errorf("problem parsing destination: %w", err)
	}

	body := telegramMsg{Text: text, ParseMode: parseMode}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("sendMessage?chat_id=%s&disable_web_page_preview=true", chatID)
	return t.Request(ctx, url, b, &struct{}{})
}

// TelegramSupportedHTML returns HTML with only tags allowed in Telegram HTML message payload, also trims ending newlines
//
// https://core.telegram.org/bots/api#html-style, https://core.telegram.org/api/entities#allowed-entities
func TelegramSupportedHTML(htmlText string) string {
	adjustedHTMLText := adjustHTMLTags(htmlText)
	p := bluemonday.NewPolicy()
	p.AllowElements("b", "strong", "i", "em", "u", "ins", "s", "strike", "del", "a", "code", "pre", "tg-spoiler", "tg-emoji", "blockquote")
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("class").OnElements("code")
	p.AllowAttrs("title").OnElements("tg-spoiler")
	p.AllowAttrs("emoji-id").OnElements("tg-emoji")
	p.AllowAttrs("language").OnElements("pre")
	return strings.TrimRight(p.Sanitize(adjustedHTMLText), "\n")
}

// EscapeTelegramText returns text sanitized of symbols not allowed inside other HTML tags in Telegram HTML message payload
//
// https://core.telegram.org/bots/api#html-style
func EscapeTelegramText(text string) string {
	// order is important
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	return text
}

// telegram not allow h1-h6 tags
// replace these tags with a combination of <b> and <i> for visual distinction
func adjustHTMLTags(htmlText string) string {
	buff := strings.Builder{}
	tokenizer := html.NewTokenizer(strings.NewReader(htmlText))
	for {
		if tokenizer.Next() == html.ErrorToken {
			return buff.String()
		}
		token := tokenizer.Token()
		switch token.Type {
		case html.StartTagToken, html.EndTagToken:
			switch token.Data {
			case "h1", "h2", "h3":
				if token.Type == html.StartTagToken {
					_, _ = buff.WriteString("<b>")
				}
				if token.Type == html.EndTagToken {
					_, _ = buff.WriteString("</b>")
				}
			case "h4", "h5", "h6":
				if token.Type == html.StartTagToken {
					_, _ = buff.WriteString("<i><b>")
				}
				if token.Type == html.EndTagToken {
					_, _ = buff.WriteString("</b></i>")
				}
			default:
				_, _ = buff.WriteString(token.String())
			}
		default:
			_, _ = buff.WriteString(token.String())
		}
	}
}

// TelegramUpdate contains update information, which is used from whole telegram API response
type TelegramUpdate struct {
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

// GetBotUsername returns bot username
func (t *Telegram) GetBotUsername() string {
	return t.username
}

// AddToken adds token
func (t *Telegram) AddToken(token, user, site string, expires time.Time) {
	t.requests.Lock()
	t.requests.data[token] = tgAuthRequest{
		expires: expires,
		user:    user,
		site:    site,
	}
	t.requests.Unlock()
}

// CheckToken verifies incoming token, returns the user address if it's confirmed and empty string otherwise
func (t *Telegram) CheckToken(token, user string) (telegram, site string, err error) {
	// lookup and consumption are a single transaction, so the one-time token can't be used twice
	t.requests.Lock()
	defer t.requests.Unlock()

	authRequest, ok := t.requests.data[token]
	if !ok {
		return "", "", errors.New("request is not found")
	}

	if time.Now().After(authRequest.expires) {
		delete(t.requests.data, token)
		return "", "", errors.New("request expired")
	}

	if !authRequest.confirmed {
		return "", "", errors.New("request is not verified yet")
	}

	if authRequest.user != user {
		return "", "", errors.New("user does not match original requester")
	}

	delete(t.requests.data, token)

	return authRequest.telegramID, authRequest.site, nil
}

// Run starts processing login requests sent in Telegram, required for user notifications to work
// Blocks caller
func (t *Telegram) Run(ctx context.Context) {
	t.updates.Lock()
	if t.updates.running {
		t.updates.Unlock()
		log.Print("[WARN] telegram updates processing is already running, ignoring the call")
		return
	}
	t.updates.running = true
	t.updates.Unlock()

	defer func() {
		t.updates.Lock()
		t.updates.running = false
		t.updates.Unlock()
	}()

	processUpdatedTicker := time.NewTicker(t.apiPollInterval)
	cleanupTicker := time.NewTicker(t.expiredCleanupInterval)

	for {
		select {
		case <-ctx.Done():
			processUpdatedTicker.Stop()
			cleanupTicker.Stop()
			return
		case <-processUpdatedTicker.C:
			updates, err := t.getUpdates(ctx)
			if err != nil {
				log.Printf("[WARN] Error while getting telegram updates: %v", err)
				continue
			}
			t.processUpdates(ctx, updates)
		case <-cleanupTicker.C:
			now := time.Now()
			t.requests.Lock()
			for key, req := range t.requests.data {
				if now.After(req.expires) {
					delete(t.requests.data, key)
				}
			}
			t.requests.Unlock()
		}
	}
}

// ProcessUpdate is alternative to Run, it processes provided plain text update from Telegram
// so that caller could get updates and send it not only there but to multiple sources
func (t *Telegram) ProcessUpdate(ctx context.Context, textUpdate string) error {
	// read lock is held for the whole call, so Run can't start in the middle of it,
	// while parallel ProcessUpdate calls are still allowed
	t.updates.RLock()
	defer t.updates.RUnlock()

	if t.updates.running {
		return errors.New("the Run goroutine should not be used with ProcessUpdate")
	}
	defer func() {
		// as Run goroutine is not running, clean up old requests on each update
		// even if we hit json decode error
		now := time.Now()
		t.requests.Lock()
		for key, req := range t.requests.data {
			if now.After(req.expires) {
				delete(t.requests.data, key)
			}
		}
		t.requests.Unlock()
	}()
	var updates TelegramUpdate
	if err := json.Unmarshal([]byte(textUpdate), &updates); err != nil {
		return fmt.Errorf("failed to decode provided telegram update: %w", err)
	}
	t.processUpdates(ctx, &updates)
	return nil
}

// Schema returns schema prefix supported by this client
func (t *Telegram) Schema() string {
	return "telegram"
}

func (t *Telegram) String() string {
	return "telegram notifications destination"
}

// parses "telegram:" in a manner "mailto:" URL is parsed url and returns chatID and parseMode.
// if chatID is channel name and not a numerical ID, `@` will be	added to it
func (t *Telegram) parseDestination(destination string) (chatID, parseMode string, err error) {
	// parse URL
	u, err := neturl.Parse(destination)
	if err != nil {
		return "", "", err
	}
	if u.Scheme != "telegram" {
		return "", "", fmt.Errorf("unsupported scheme %s, should be telegram", u.Scheme)
	}

	chatID = u.Opaque
	if _, err := strconv.ParseInt(chatID, 10, 64); err != nil {
		chatID = "@" + chatID // if chatID not a number enforce @ prefix
	}

	parseMode = "Markdown"
	if u.Query().Get("parseMode") != "" {
		parseMode = u.Query().Get("parseMode")
	}

	return chatID, parseMode, nil
}

// getUpdates fetches incoming updates
func (t *Telegram) getUpdates(ctx context.Context) (*TelegramUpdate, error) {
	url := `getUpdates?allowed_updates=["message"]`
	if t.updateOffset != 0 {
		url += fmt.Sprintf("&offset=%d", t.updateOffset)
	}

	var result TelegramUpdate

	err := t.Request(ctx, url, nil, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updates: %w", err)
	}

	for _, u := range result.Result {
		if u.UpdateID >= t.updateOffset {
			t.updateOffset = u.UpdateID + 1
		}
	}

	return &result, nil
}

// processUpdates processes a batch of updates from telegram servers
func (t *Telegram) processUpdates(ctx context.Context, updates *TelegramUpdate) {
	for _, update := range updates.Result {
		if update.Message.Chat.Type != "private" {
			continue
		}

		if !strings.HasPrefix(update.Message.Text, "/start ") {
			continue
		}

		token := strings.TrimPrefix(update.Message.Text, "/start ")

		// confirmation is a single transaction, otherwise a request consumed by CheckToken
		// in the middle of it would be restored from the stale copy
		t.requests.Lock()
		authRequest, ok := t.requests.data[token]
		if ok {
			authRequest.confirmed = true
			authRequest.telegramID = strconv.Itoa(update.Message.Chat.ID)
			t.requests.data[token] = authRequest
		}
		t.requests.Unlock()

		if !ok { // no such token
			if t.ErrorMsg != "" {
				if err := t.sendText(ctx, update.Message.Chat.ID, t.ErrorMsg); err != nil {
					log.Printf("[WARN] failed to notify telegram peer: %v", err)
				}
			}
			continue
		}

		if err := t.sendText(ctx, update.Message.Chat.ID, t.SuccessMsg); err != nil {
			log.Printf("[ERROR] failed to notify telegram peer: %v", err)
		}
	}
}

// sendText sends a plain text message to telegram peer
func (t *Telegram) sendText(ctx context.Context, recipientID int, msg string) error {
	url := fmt.Sprintf("sendMessage?chat_id=%d&text=%s", recipientID, neturl.QueryEscape(msg))
	return t.Request(ctx, url, nil, &struct{}{})
}

// botInfo returns info about configured bot
func (t *Telegram) botInfo(ctx context.Context) (*TelegramBotInfo, error) {
	var resp = struct {
		Result *TelegramBotInfo `json:"result"`
	}{}

	err := t.Request(ctx, "getMe", nil, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Result == nil {
		return nil, errors.New("received empty result")
	}

	return resp.Result, nil
}

// validateTelegramBaseURL checks a caller-supplied API base and returns it without its trailing
// slash. Every request built from it carries the bot token in its path, so a base that resolves
// somewhere unintended ships the token there: "https://api.telegram.org@evil.tld" is a valid URL
// whose host is evil.tld, and a bare scheme or an opaque form silently produces a request to
// somewhere else again. A path prefix is allowed, for a proxy mounted under one.
//
// No rejection echoes the value. It is configuration that can carry credentials in its userinfo, a
// secret in its query or a password where the port belongs, and the error travels to whatever logs
// the constructor failure. The parse error is dropped for the same reason: *url.Error prints the
// URL it was given, and the inner error quotes the input back as well.
func validateTelegramBaseURL(baseURL string) (string, error) {
	baseURL = strings.TrimRight(baseURL, "/")

	u, err := neturl.Parse(baseURL)
	if err != nil {
		return "", errors.New("telegram api url is not a valid url")
	}

	switch {
	case u.Opaque != "":
		return "", errors.New("telegram api url must not be opaque")
	case u.Scheme != "http" && u.Scheme != "https":
		return "", fmt.Errorf("telegram api url must be http or https, got %q", u.Scheme)
	case u.Hostname() == "":
		// u.Host is non-empty for "http://:9000", which resolves to the local machine
		return "", errors.New("telegram api url must have a host")
	case u.User != nil:
		return "", errors.New("telegram api url must not carry userinfo")
	case u.RawQuery != "" || u.ForceQuery:
		return "", errors.New("telegram api url must not carry a query")
	case u.Fragment != "" || strings.HasSuffix(baseURL, "#"):
		return "", errors.New("telegram api url must not carry a fragment")
	}

	// rebuilt from what was checked, so the string validated is the string used
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String(), nil
}

// Request makes a request to the Telegram API and return the result
func (t *Telegram) Request(ctx context.Context, method string, b []byte, data any) error {
	return repeater.NewFixed(3, time.Millisecond*250).Do(ctx, func() error {
		url := fmt.Sprintf("%s%s/%s", t.apiPrefix, t.Token, method)

		var req *http.Request
		var err error
		if b == nil {
			req, err = http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
		} else {
			req, err = http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
		}
		if err != nil {
			return fmt.Errorf("failed to create request: %w", t.redactToken(err))
		}
		if b != nil {
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
		}

		// refusing redirects: Go copies the previous URL into Referer on any hop that is not
		// https-to-http, and every URL here carries the bot token in its path, so following one
		// hands the destination the token, usually into its access log. Telegram does not redirect;
		// something standing in for it can
		client := http.Client{
			Timeout: t.Timeout,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return errors.New("refusing to follow a telegram api redirect: the bot token travels in the URL")
			},
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", t.redactToken(err))
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return t.redactToken(t.parseError(resp.Body, resp.StatusCode))
		}

		if err = json.NewDecoder(resp.Body).Decode(data); err != nil {
			return fmt.Errorf("failed to decode json response: %w", err)
		}

		return nil
	})
}

// redactToken removes the bot token from an error before it reaches a caller's log.
//
// Two routes, and the second only exists once APIURL can point somewhere the operator chose. The
// token is part of every API URL, so a transport failure carries it in *url.Error's URL field. And
// the upstream decides the text of an API error: something standing in for Telegram can echo the
// request URI into its description, in whatever encoding it likes, so scrubbing the token itself is
// what holds rather than matching a URL shape.
func (t *Telegram) redactToken(err error) error {
	if err == nil || t.Token == "" {
		return err
	}

	var urlErr *neturl.Error
	if errors.As(err, &urlErr) && strings.Contains(urlErr.URL, t.Token) {
		return &neturl.Error{
			Op: urlErr.Op, URL: strings.ReplaceAll(urlErr.URL, t.Token, "<redacted>"), Err: urlErr.Err,
		}
	}

	// the text scrub is for something that could actually be a bot token. Telegram issues them as
	// "<bot id>:<secret>", and blanking a short arbitrary string out of a diagnostic corrupts more
	// than it protects: a token of "404" would turn every "status code 404" into "status code
	// <redacted>". The URL-field redaction above stays unconditional, since there the token is
	// whatever the caller configured and the field is nothing else
	if !looksLikeBotToken(t.Token) {
		return err
	}

	msg := err.Error()
	for _, form := range []string{t.Token, neturl.QueryEscape(t.Token), neturl.PathEscape(t.Token)} {
		msg = strings.ReplaceAll(msg, form, "<redacted>")
	}

	// substitution only catches encodings we thought of, and the upstream picks the encoding, so the
	// result is checked once more against a decoded copy. If the token is still recoverable by any
	// of those routes the text goes rather than the token stays
	if tokenRecoverable(msg, t.Token) {
		return errors.New("unexpected telegram API error, text withheld: the bot token could not be ruled out of it")
	}

	if msg == err.Error() {
		return err
	}
	return errors.New(msg)
}

// looksLikeBotToken reports whether token has the shape Telegram issues, "<bot id>:<secret>". Used
// to keep the text scrub off values that cannot be one, where it would damage diagnostics for
// nothing
func looksLikeBotToken(token string) bool {
	id, secret, found := strings.Cut(token, ":")
	if !found || len(secret) < 8 {
		return false
	}
	if _, err := strconv.Atoi(id); err != nil {
		return false
	}
	return true
}

// tokenRecoverable reports whether token can still be read out of msg after undoing the encodings
// an upstream might have applied to it.
//
// Fails closed: after a decode error, or after the cap runs out while the text is still changing,
// absence was never established, and withholding is the only answer that cannot leak. One stray "%"
// in whatever the upstream echoed is enough to reach that.
func tokenRecoverable(msg, token string) bool {
	lowered := strings.ToLower(token)
	seen := msg
	for i := 0; i < 5; i++ {
		if strings.Contains(strings.ToLower(seen), lowered) {
			return true
		}
		next, err := neturl.QueryUnescape(seen)
		if err != nil {
			return true
		}
		if next == seen {
			// decoding has converged and every form has been examined, so the token is absent
			// rather than undecided
			return false
		}
		seen = next
	}
	return true
}

func (t *Telegram) parseError(r io.Reader, statusCode int) error {
	tgErr := struct {
		Description string `json:"description"`
	}{}
	if err := json.NewDecoder(r).Decode(&tgErr); err != nil {
		return fmt.Errorf("unexpected telegram API status code %d", statusCode)
	}
	return fmt.Errorf("unexpected telegram API status code %d, error: %q", statusCode, tgErr.Description)
}
