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

const telegramTimeOut = 5000 * time.Millisecond
const telegramAPIPrefix = "https://api.telegram.org/bot"

// NewTelegram makes telegram bot for notifications
func NewTelegram(params TelegramParams) (*Telegram, error) {
	res := Telegram{TelegramParams: params}
	if _, err := strconv.ParseInt(res.AdminChannelID, 10, 64); err != nil {
		res.AdminChannelID = "@" + res.AdminChannelID // if channelID not a number enforce @ prefix
	}

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
			return errors.Errorf("unexpected telegram status code %d", resp.StatusCode)
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
	var err error

	if t.AdminChannelID != "" {
		err = t.sendAdminNotification(ctx, req)
		if err != nil {
			return errors.Wrapf(err, "problem sending admin telegram notification")
		}
	}

	return nil
}

func (t *Telegram) sendAdminNotification(ctx context.Context, req Request) error {
	log.Printf("[DEBUG] send admin telegram notification to %s, comment id %s", t.AdminChannelID, req.Comment.ID)

	msg, err := buildTelegramMessage(req)
	if err != nil {
		return errors.Wrap(err, "failed to make telegram message body")
	}

	err = t.sendMessage(ctx, msg, t.AdminChannelID)
	if err != nil {
		return errors.Wrapf(err, "failed to send admin notification about %s", req.Comment.ID)
	}
	return nil
}

func (t *Telegram) sendMessage(ctx context.Context, b []byte, chatID string) error {
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
		return errors.Errorf("unexpected telegram status code %d for url %q", resp.StatusCode, u)
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
	from := req.Comment.User.Name
	if req.Comment.ParentID != "" {
		from += " → " + req.parent.User.Name
	}
	from = "*" + from + "*"
	link := fmt.Sprintf("↦ [original comment](%s)", req.Comment.Locator.URL+uiNav+req.Comment.ID)
	if req.Comment.PostTitle != "" {
		link = fmt.Sprintf("↦ [%s](%s)", escapeTitle(req.Comment.PostTitle), req.Comment.Locator.URL+uiNav+req.Comment.ID)
	}

	msg := fmt.Sprintf("%s\n\n%s\n\n%s", from, req.Comment.Orig, link)
	msg = html.UnescapeString(msg)
	body := struct {
		Text string `json:"text"`
	}{Text: msg}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func escapeTitle(title string) string {
	escSymbols := []string{"[", "]", "(", ")"}
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
	return "telegram: " + t.AdminChannelID
}
