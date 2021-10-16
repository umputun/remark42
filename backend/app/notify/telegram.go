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

	"github.com/hashicorp/go-multierror"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater"
	"github.com/pkg/errors"
)

// TelegramParams contain settings for telegram notifications
type TelegramParams struct {
	AdminChannelID    string        // unique identifier for the target chat or username of the target channel (in the format @channelusername)
	Token             string        // token for telegram bot API interactions
	Timeout           time.Duration // http client timeout
	BotUsername       string        // filled with bot username after Telegram creation, used in frontend
	UserNotifications bool          // flag which enables user notifications

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

		res.BotUsername = tgResp.Result.UserName
		return nil
	})

	return &res, err
}

// Send to telegram recipients
func (t *Telegram) Send(ctx context.Context, req Request) error {
	log.Printf("[DEBUG] send telegram notification for comment ID %s", req.Comment.ID)
	result := new(multierror.Error)

	msg, err := buildTelegramMessage(req)
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

	u := fmt.Sprintf("%s%s/sendMessage?chat_id=%s&disable_web_page_preview=true",
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

	msg := fmt.Sprintf("[%s](%s)", escapeText(req.Comment.User.Name), commentURLPrefix+req.Comment.ID)

	if req.Comment.ParentID != "" {
		msg += fmt.Sprintf(" -> [%s](%s)", escapeText(req.parent.User.Name), commentURLPrefix+req.parent.ID)
	}

	msg += fmt.Sprintf("\n\n%s", escapeText(req.Comment.Orig))

	if req.Comment.ParentID != "" {
		msg += fmt.Sprintf("\n\n> \"_%s_\"", escapeText(req.parent.Orig))
	}

	if req.Comment.PostTitle != "" {
		msg += fmt.Sprintf("\n\n↦  [%s](%s)", escapeText(req.Comment.PostTitle), req.Comment.Locator.URL)
	}

	body := telegramMsg{Text: msg, ParseMode: "MarkdownV2"}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func escapeText(text string) string {
	escSymbols := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	res := html.UnescapeString(text)
	for _, esc := range escSymbols {
		res = strings.Replace(res, esc, "\\"+esc, -1)
	}
	return res
}

func escapeCode(text string) string {
	escSymbols := []string{"`", `\`}
	res := text
	for _, esc := range escSymbols {
		res = strings.Replace(res, esc, "\\"+esc, -1)
	}
	return res
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
	result := fmt.Sprintf("Confirmation for *%s* on site %s\n"+
		"Please copy and paste this text into “token” field on comments page to confirm subscription:\n\n\n"+
		"```%s```",
		escapeText(user), escapeText(site), escapeCode(token))
	body := telegramMsg{Text: result, ParseMode: "MarkdownV2"}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return b, nil
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
