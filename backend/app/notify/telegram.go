package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater"
	"github.com/pkg/errors"
)

// TelegramParams contain settings for telegram notifications
type TelegramParams struct {
	AdminChannelID string        // unique identifier for the target chat or username of the target channel (in the format @channelusername)
	Token          string        // token for telegram bot API interactions
	Timeout        time.Duration // http client timeout

	apiPrefix string // changed only in tests
}

// Telegram implements notify.Destination for telegram
type Telegram struct {
	TelegramParams
}

// telegramMsg is used to send message trough Telegram bot API
type telegramMsg struct {
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
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

	err := repeater.NewDefault(5, time.Millisecond*250).Do(ctx, func() error {
		client := http.Client{Timeout: res.Timeout}
		resp, err := client.Get(fmt.Sprintf("%s%s/getMe", res.apiPrefix, res.Token))
		if err != nil {
			return errors.Wrap(err, "can't initialize telegram notifications")
		}
		defer func() {
			if err = resp.Body.Close(); err != nil {
				log.Printf("[WARN] can't close request body, %s", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			tgErr := struct {
				Description string `json:"description"`
			}{}
			if err = json.NewDecoder(resp.Body).Decode(&tgErr); err == nil {
				return errors.Errorf("unexpected telegram API status code %d, error: %q", resp.StatusCode, tgErr.Description)
			}
			return errors.Errorf("unexpected telegram API status code %d", resp.StatusCode)
		}

		tgResp := struct {
			OK     bool `json:"ok"`
			Result struct {
				FirstName string `json:"first_name"`
				ID        uint64 `json:"id"`
				IsBot     bool   `json:"is_bot"`
				UserName  string `json:"username"`
			}
		}{}

		if err = json.NewDecoder(resp.Body).Decode(&tgResp); err != nil {
			return errors.Wrap(err, "can't decode response")
		}

		if !tgResp.OK || !tgResp.Result.IsBot {
			return errors.Errorf("unexpected telegram response %+v", tgResp)
		}
		return nil
	})

	return &res, err
}

// Send to telegram recipients
func (t *Telegram) Send(ctx context.Context, req Request) error {
	log.Printf("[DEBUG] send telegram notification for comment ID %s", req.Comment.ID)

	msg, err := buildTelegramMessage(req)
	if err != nil {
		return errors.Wrapf(err, "failed to make telegram message body for comment ID %s", req.Comment.ID)
	}

	if t.AdminChannelID != "" {
		err := t.sendMessage(ctx, msg, t.AdminChannelID)
		return errors.Wrapf(err,
			"problem sending admin telegram notification about comment ID %s to %s", req.Comment.ID, t.AdminChannelID)
	}
	return nil
}

func (t *Telegram) sendMessage(ctx context.Context, b []byte, chatID string) error {
	if _, err := strconv.ParseInt(chatID, 10, 64); err != nil {
		chatID = "@" + chatID // if chatID not a number enforce @ prefix
	}

	u := fmt.Sprintf("%s%s/sendMessage?chat_id=%s&parse_mode=Markdown&disable_web_page_preview=true",
		t.apiPrefix, t.Token, chatID)
	r, err := http.NewRequest("POST", u, bytes.NewReader(b))
	if err != nil {
		return errors.Wrap(err, "failed to make telegram request")
	}
	r.Header.Set("Content-Type", "application/json; charset=utf-8")

	client := http.Client{Timeout: t.Timeout}
	r = r.WithContext(ctx)
	resp, err := client.Do(r)
	if err != nil {
		return errors.Wrap(err, "failed to get telegram response")
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Printf("[WARN] can't close request body, %s", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		tgErr := struct {
			Description string `json:"description"`
		}{}
		if err = json.NewDecoder(resp.Body).Decode(&tgErr); err == nil {
			return errors.Errorf("unexpected telegram API status code %d, error: %q", resp.StatusCode, tgErr.Description)
		}
		return errors.Errorf("unexpected telegram API status code %d", resp.StatusCode)
	}

	tgResp := struct {
		OK bool `json:"ok"`
	}{}

	if err = json.NewDecoder(resp.Body).Decode(&tgResp); err != nil {
		return errors.Wrap(err, "can't decode telegram response")
	}
	return nil
}

func buildTelegramMessage(req Request) ([]byte, error) {
	commentURLPrefix := req.Comment.Locator.URL + uiNav

	msg := "New reply to comment"
	if req.Comment.PostTitle != "" {
		msg += fmt.Sprintf(" for %q", req.Comment.PostTitle)
	}
	msg += ":"

	if req.Comment.ParentID != "" {
		msg += fmt.Sprintf(
			"\n[Original comment](%s) from %s at %s:\n%s",
			commentURLPrefix+req.parent.ID,
			escapeText(req.parent.User.Name),
			escapeText(req.parent.Timestamp.Format("02.01.2006 at 15:04")),
			escapeText(req.parent.Orig),
		)
	}

	msg += fmt.Sprintf(
		"\n[Reply](%s) from %s at %s:\n%s",
		commentURLPrefix+req.Comment.ID,
		escapeText(req.Comment.User.Name),
		escapeText(req.Comment.Timestamp.Format("02.01.2006 at 15:04")),
		escapeText(req.Comment.Orig),
	)
	msg = html.UnescapeString(msg)
	body := telegramMsg{Text: msg, ParseMode: "MarkdownV2"}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func escapeText(title string) string {
	escSymbols := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", "!"}
	res := title
	for _, esc := range escSymbols {
		res = strings.Replace(res, esc, "\\"+esc, -1)
	}
	return res
}

// SendVerification is not implemented for telegram
func (t *Telegram) SendVerification(_ context.Context, _ VerificationRequest) error {
	return nil
}

func (t *Telegram) String() string {
	result := "telegram"
	if t.AdminChannelID != "" {
		result += " with admin notifications to " + t.AdminChannelID
	}
	return result
}
