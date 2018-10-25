package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Telegram implements notify.Destination for telegram
type Telegram struct {
	channelName string
	token       string
	apiPrefix   string
}

const telegramTimeOut = 2500 * time.Millisecond
const telegramAPIPrefix = "https://api.telegram.org/bot"

// NewTelegram makes telegram bot for notifications
func NewTelegram(token string, channelName string, api string) (*Telegram, error) {
	res := Telegram{channelName: channelName, token: token, apiPrefix: api}
	res.channelName = strings.TrimPrefix(res.channelName, "@")
	if res.apiPrefix == "" {
		res.apiPrefix = telegramAPIPrefix
	}
	client := http.Client{Timeout: telegramTimeOut}
	resp, err := client.Get(fmt.Sprintf("%s%s/getMe", res.apiPrefix, token))
	if err != nil {
		return nil, errors.Wrap(err, "can't initialize telegram notifications")
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Printf("[WARN] can't close request body, %s", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected telegram status code %d", resp.StatusCode)
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
		return nil, errors.Wrap(err, "can't authorize to telegram")
	}

	if !tgResp.OK || !tgResp.Result.IsBot {
		return nil, errors.Errorf("unexpected telegram response %+v", tgResp)
	}

	return &res, nil
}

// Send to telegram channel
func (t *Telegram) Send(ctx context.Context, req request) error {
	client := http.Client{Timeout: telegramTimeOut}

	from := req.comment.User.Name
	if req.comment.ParentID != "" {
		from += " -> " + req.parent.User.Name
	}
	from = "*" + from + "*"
	link := fmt.Sprintf("[comment](%s)", req.comment.Locator.URL+uiNav+req.comment.ID)
	msg := fmt.Sprintf("%s\n\n%s\n\n%s", from, req.comment.Text, link)

	r, err := http.NewRequest("GET", fmt.Sprintf("%s%s/sendMessage?chat_id=@%s&text=%s&parse_mode=Markdown",
		t.apiPrefix, t.token, t.channelName, url.QueryEscape(msg)), nil)
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
		return errors.Errorf("unexpected telegram status code %d", resp.StatusCode)
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
	return "telegram-" + t.channelName
}
