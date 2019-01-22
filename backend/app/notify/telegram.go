package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater"
	"github.com/pkg/errors"
)

// Telegram implements notify.Destination for telegram
type Telegram struct {
	channelID string // unique identifier for the target chat or username of the target channel (in the format @channelusername)
	token     string
	apiPrefix string
	timeout   time.Duration
}

const telegramTimeOut = 5000 * time.Millisecond
const telegramAPIPrefix = "https://api.telegram.org/bot"

// NewTelegram makes telegram bot for notifications
func NewTelegram(token string, channelID string, timeout time.Duration, api string) (*Telegram, error) {

	if _, err := strconv.ParseInt(channelID, 10, 64); err != nil {
		channelID = "@" + channelID // if channelID not a number enforce @ prefix
	}

	res := Telegram{channelID: channelID, token: token, apiPrefix: api, timeout: timeout}
	if res.apiPrefix == "" {
		res.apiPrefix = telegramAPIPrefix
	}
	if res.timeout == 0 {
		res.timeout = telegramTimeOut
	}
	log.Printf("[DEBUG] create new telegram notifier for cham %s, timeout=%s, api=%s", channelID, res.timeout, res.timeout)

	err := repeater.NewDefault(5, time.Millisecond*250).Do(func() error {
		client := http.Client{Timeout: telegramTimeOut}
		resp, err := client.Get(fmt.Sprintf("%s%s/getMe", res.apiPrefix, token))
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

// Send to telegram channel
func (t *Telegram) Send(ctx context.Context, req request) error {
	client := http.Client{Timeout: telegramTimeOut}
	log.Printf("[DEBUG] send telegram notification to %s, comment id %s", t.channelID, req.comment.ID)

	from := req.comment.User.Name
	if req.comment.ParentID != "" {
		from += " → " + req.parent.User.Name
	}
	from = "*" + from + "*"
	link := fmt.Sprintf("↦ [original comment](%s)", req.comment.Locator.URL+uiNav+req.comment.ID)
	if req.comment.PostTitle != "" {
		link = fmt.Sprintf("↦ [%s](%s)", req.comment.PostTitle, req.comment.Locator.URL+uiNav+req.comment.ID)
	}
	u := fmt.Sprintf("%s%s/sendMessage?chat_id=%s&parse_mode=Markdown&disable_web_page_preview=true",
		t.apiPrefix, t.token, t.channelID)

	msg := fmt.Sprintf("%s\n\n%s\n\n%s", from, req.comment.Orig, link)
	msg = html.UnescapeString(msg)
	body := struct {
		Text string `json:"text"`
	}{Text: msg}

	b, err := json.Marshal(body)
	if err != nil {
		return errors.Wrap(err, "failed to make telegram body")
	}

	r, err := http.NewRequest("POST", u, bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		return errors.Wrap(err, "failed to make telegram request")
	}

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

func (t *Telegram) String() string {
	return "telegram: " + t.channelID
}
