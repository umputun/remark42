package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/microcosm-cc/bluemonday"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater"
	"github.com/pkg/errors"
)

// TelegramParams contain settings for telegram notifications
type TelegramParams struct {
	AdminChannelID    string        // unique identifier for the target chat or username of the target channel (in the format @channelusername)
	Token             string        // token for telegram bot API interactions
	Timeout           time.Duration // http client timeout
	UserNotifications bool          // flag which enables user notifications

	apiPrefix string // changed only in tests
}

// Telegram implements notify.Destination for telegram
type Telegram struct {
	TelegramParams

	username string // bot username
}

// telegramMsg is used to send message trough Telegram bot API
type telegramMsg struct {
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// TelegramBotInfo structure contains information about telegram bot, which is used from whole telegram API response
type TelegramBotInfo struct {
	Username string `json:"username"`
}

const telegramTimeOut = 5000 * time.Millisecond
const telegramAPIPrefix = "https://api.telegram.org/bot"

// NewTelegram makes telegram bot for notifications
func NewTelegram(params TelegramParams) (*Telegram, error) {
	res := Telegram{TelegramParams: params}

	if res.apiPrefix == "" {
		res.apiPrefix = telegramAPIPrefix
	}
	if res.Timeout == 0 {
		res.Timeout = telegramTimeOut
	}
	log.Printf("[DEBUG] create new telegram notifier for api=%s, timeout=%s", res.apiPrefix, res.Timeout)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	botInfo, err := res.botInfo(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "can't retrieve bot info from Telegram API")
	}
	res.username = botInfo.Username

	return &res, nil
}

// Send to telegram recipients
func (t *Telegram) Send(ctx context.Context, req Request) error {
	log.Printf("[DEBUG] send telegram notification for comment ID %s", req.Comment.ID)
	result := new(multierror.Error)

	msg, err := buildMessage(req)
	if err != nil {
		return errors.Wrapf(err, "failed to make telegram message body for comment ID %s", req.Comment.ID)
	}

	if t.AdminChannelID != "" {
		err := t.sendMessage(ctx, msg, t.AdminChannelID)
		result = multierror.Append(errors.Wrapf(err,
			"problem sending admin telegram notification about comment ID %s to %s", req.Comment.ID, t.AdminChannelID),
		)
	}

	if t.UserNotifications {
		for _, user := range req.Telegrams {
			err := t.sendMessage(ctx, msg, user)
			result = multierror.Append(errors.Wrapf(err,
				"problem sending user telegram notification about comment ID %s to %q", req.Comment.ID, user),
			)
		}
	}
	return result.ErrorOrNil()
}

func (t *Telegram) sendMessage(ctx context.Context, b []byte, chatID string) error {
	if _, err := strconv.ParseInt(chatID, 10, 64); err != nil {
		chatID = "@" + chatID // if chatID not a number enforce @ prefix
	}

	url := fmt.Sprintf("sendMessage?chat_id=%s&disable_web_page_preview=true", chatID)
	return t.request(ctx, url, b, &struct{}{})
}

// buildMessage generates message for generic notification about new comment
func buildMessage(req Request) ([]byte, error) {
	commentURLPrefix := req.Comment.Locator.URL + uiNav

	msg := fmt.Sprintf(`<a href="%s">%s</a>`, commentURLPrefix+req.Comment.ID, escapeTelegramText(req.Comment.User.Name))

	if req.Comment.ParentID != "" {
		msg += fmt.Sprintf(" -> <a href=\"%s\">%s</a>", commentURLPrefix+req.parent.ID, escapeTelegramText(req.parent.User.Name))
	}

	msg += fmt.Sprintf("\n\n%s", telegramSupportedHTML(req.Comment.Text))

	if req.Comment.ParentID != "" {
		msg += fmt.Sprintf("\n\n \"_%s_\"", telegramSupportedHTML(req.parent.Text))
	}

	if req.Comment.PostTitle != "" {
		msg += fmt.Sprintf("\n\n↦  <a href=\"%s\">%s</a>", req.Comment.Locator.URL, escapeTelegramText(req.Comment.PostTitle))
	}

	body := telegramMsg{Text: msg, ParseMode: "HTML"}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// returns HTML with only tags allowed in Telegram HTML message payload
// https://core.telegram.org/bots/api#html-style
func telegramSupportedHTML(htmlText string) string {
	p := bluemonday.NewPolicy()
	p.AllowElements("b", "strong", "i", "em", "u", "ins", "s", "strike", "del", "a", "code", "pre")
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("class").OnElements("code")
	return p.Sanitize(htmlText)
}

// returns text sanitized of symbols not allowed inside other HTML tags in Telegram HTML message payload
// https://core.telegram.org/bots/api#html-style
func escapeTelegramText(text string) string {
	// order is important
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	return text
}

// SendVerification sends user verification message to the specified user
func (t *Telegram) SendVerification(ctx context.Context, req VerificationRequest) error {
	if req.Telegram == "" {
		// this means we can't send this request via Telegram
		return nil
	}
	select {
	case <-ctx.Done():
		return errors.Errorf("sending message to %q aborted due to canceled context", req.User)
	default:
	}

	log.Printf("[DEBUG] send verification via %s, user %s", t, req.User)
	msg, err := t.buildVerificationMessage(req.User, req.Token, req.SiteID)
	if err != nil {
		return err
	}

	return t.sendMessage(ctx, msg, req.Telegram)
}

// buildVerificationMessage generates verification telegram message based on given input
func (t *Telegram) buildVerificationMessage(user, token, site string) ([]byte, error) {
	result := fmt.Sprintf("Confirmation for <i>%s</i> on site %s\n"+
		"Please copy and paste this text into “token” field on comments page to confirm subscription:\n\n\n"+
		"<pre>%s</pre>",
		escapeTelegramText(user), escapeTelegramText(site), escapeTelegramText(token))
	body := telegramMsg{Text: result, ParseMode: "HTML"}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// GetBotUsername returns bot username
func (t *Telegram) GetBotUsername() string {
	return t.username
}

func (t *Telegram) String() string {
	result := "telegram"
	if t.AdminChannelID != "" {
		result += " with admin notifications to " + t.AdminChannelID
	}
	if t.UserNotifications {
		result += " with user notifications enabled"
	}
	return result
}

// botInfo returns info about configured bot
func (t *Telegram) botInfo(ctx context.Context) (*TelegramBotInfo, error) {
	var resp = struct {
		Result *TelegramBotInfo `json:"result"`
	}{}

	err := t.request(ctx, "getMe", nil, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Result == nil {
		return nil, errors.New("received empty result")
	}

	return resp.Result, nil
}

func (t *Telegram) request(ctx context.Context, method string, b []byte, data interface{}) error {
	return repeater.NewDefault(3, time.Millisecond*250).Do(ctx, func() error {
		url := fmt.Sprintf("%s%s/%s", t.apiPrefix, t.Token, method)

		var req *http.Request
		var err error
		if b == nil {
			req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
		} else {
			req, err = http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
		}
		if err != nil {
			return errors.Wrap(err, "failed to create request")
		}

		client := http.Client{Timeout: t.Timeout}
		resp, err := client.Do(req)
		if err != nil {
			return errors.Wrap(err, "failed to send request")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return t.parseError(resp.Body, resp.StatusCode)
		}

		if err = json.NewDecoder(resp.Body).Decode(data); err != nil {
			return errors.Wrap(err, "failed to decode json response")
		}

		return nil
	})
}

func (t *Telegram) parseError(r io.Reader, statusCode int) error {
	tgErr := struct {
		Description string `json:"description"`
	}{}
	if err := json.NewDecoder(r).Decode(&tgErr); err != nil {
		return errors.Errorf("unexpected telegram API status code %d", statusCode)
	}
	return errors.Errorf("unexpected telegram API status code %d, error: %q", statusCode, tgErr.Description)
}
