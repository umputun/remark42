package notify

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
	ntf "github.com/go-pkgz/notify"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/net/html"
)

const comment_text_length_limit = 100

// TelegramParams contain settings for telegram notifications
type TelegramParams struct {
	AdminChannelID       string        // unique identifier for the target chat or username of the target channel (in the format @channelusername)
	Token                string        // token for telegram bot API interactions
	Timeout              time.Duration // http client timeout
	UserNotifications    bool          // flag which enables user notifications
	ErrorMsg, SuccessMsg string        // messages for successful and unsuccessful subscription requests to bot
}

// Telegram implements notify.Destination for telegram
type Telegram struct {
	*ntf.Telegram

	AdminChannelID    string // unique identifier for the target chat or username of the target channel (in the format @channelusername)
	UserNotifications bool   // flag which enables user notifications
}

// NewTelegram makes telegram bot for notifications
func NewTelegram(params TelegramParams) (*Telegram, error) {
	client, err := ntf.NewTelegram(ntf.TelegramParams{
		Token:      params.Token,
		Timeout:    params.Timeout,
		ErrorMsg:   params.ErrorMsg,
		SuccessMsg: params.SuccessMsg,
	})
	if err != nil {
		return nil, err
	}

	return &Telegram{Telegram: client, AdminChannelID: params.AdminChannelID, UserNotifications: params.UserNotifications}, nil
}

// Send to telegram recipients
func (t *Telegram) Send(ctx context.Context, req Request) error {
	log.Printf("[DEBUG] send telegram notification for comment ID %s", req.Comment.ID)
	result := new(multierror.Error)

	msg := t.buildMessage(req)

	if t.AdminChannelID != "" {
		err := t.Telegram.Send(ctx, fmt.Sprintf("telegram:%s?parseMode=HTML", t.AdminChannelID), msg)
		if err != nil {
			result = multierror.Append(result,
				fmt.Errorf("problem sending admin telegram notification about comment ID %s to %s: %w",
					req.Comment.ID, t.AdminChannelID, err,
				),
			)
		}
	}

	if t.UserNotifications {
		for _, user := range req.Telegrams {
			err := t.Telegram.Send(ctx, fmt.Sprintf("telegram:%s?parseMode=HTML", user), msg)
			if err != nil {
				result = multierror.Append(result,
					fmt.Errorf("problem sending user telegram notification about comment ID %s to %q: %w",
						req.Comment.ID, user, err,
					),
				)
			}
		}
	}
	return result.ErrorOrNil()
}

type stringArr struct {
	data []string
	len  int
}

// Push adds element to the end
func (s *stringArr) Push(v string) {
	s.data = append(s.data, v)
	s.len += len(v)
}

// Pop removes element from end and returns it
func (s *stringArr) Pop() string {
	l := len(s.data)
	newData, v := s.data[:l-1], s.data[l-1]
	s.data = newData
	s.len -= len(v)
	return v
}

// Unshift adds element to the start
func (s *stringArr) Unshift(v string) {
	s.data = append([]string{v}, s.data...)
	s.len += len(v)
}

// Shift removes element from start and returns it
func (s *stringArr) Shift() string {
	v, newData := s.data[0], s.data[1:]
	s.data = newData
	s.len -= len(v)
	return v
}

// String returns all strings concatenated
func (s stringArr) String() string {
	return strings.Join(s.data, "")
}

// Len returns total length of all strings concatenated
func (s stringArr) Len() int {
	return s.len
}

// pruneHTML prunes string keeping HTML closing tags
func pruneHTML(htmlText string, maxLength int) string {
	result := stringArr{}
	endTokens := stringArr{}

	suffix := "..."
	suffixLen := len(suffix)

	tokenizer := html.NewTokenizer(strings.NewReader(htmlText))
	for {
		if tokenizer.Next() == html.ErrorToken {
			return result.String()
		}
		token := tokenizer.Token()

		switch token.Type {
		case html.CommentToken, html.DoctypeToken:
			// skip tokens without content
			continue

		case html.StartTagToken:
			// <token></token>
			// len(token) * 2 + len("<></>")
			totalLenToAppend := len(token.Data)*2 + 5
			if result.Len()+totalLenToAppend+endTokens.Len()+suffixLen > maxLength {
				return result.String() + suffix + endTokens.String()
			}
			endTokens.Unshift(fmt.Sprintf("</%s>", token.Data))

		case html.EndTagToken:
			endTokens.Shift()

		case html.TextToken, html.SelfClosingTagToken:
			if result.Len()+len(token.String())+endTokens.Len()+suffixLen > maxLength {
				text := pruneStringToWord(token.String(), maxLength-result.Len()-endTokens.Len()-suffixLen)
				return result.String() + text + suffix + endTokens.String()
			}
		}

		result.Push((token.String()))
	}
}

// pruneStringToWord prunes string to specified length respecting words
func pruneStringToWord(text string, maxLength int) string {
	if maxLength <= 0 {
		return ""
	}

	result := ""

	arr := strings.Split(text, " ")
	for _, s := range arr {
		if len(result)+len(s) >= maxLength {
			return strings.TrimRight(result, " ")
		}
		// keep last space, it's ok
		result += s + " "
	}

	return text
}

// buildMessage generates message for generic notification about new comment
func (t *Telegram) buildMessage(req Request) string {
	commentURLPrefix := req.Comment.Locator.URL + uiNav

	msg := fmt.Sprintf(`<a href=%q>%s</a>`, commentURLPrefix+req.Comment.ID, ntf.EscapeTelegramText(req.Comment.User.Name))

	if req.Comment.ParentID != "" {
		msg += fmt.Sprintf(" -> <a href=%q>%s</a>", commentURLPrefix+req.parent.ID, ntf.EscapeTelegramText(req.parent.User.Name))
	}

	msg += fmt.Sprintf("\n\n%s", pruneHTML(ntf.TelegramSupportedHTML(req.Comment.Text), comment_text_length_limit))

	if req.Comment.ParentID != "" {
		msg += fmt.Sprintf("\n\n\"<i>%s</i>\"", ntf.TelegramSupportedHTML(req.parent.Text))
	}

	if req.Comment.PostTitle != "" {
		msg += fmt.Sprintf("\n\n↦  <a href=%q>%s</a>", req.Comment.Locator.URL, ntf.EscapeTelegramText(req.Comment.PostTitle))
	}

	return msg
}

// SendVerification is not needed for telegram
func (t *Telegram) SendVerification(_ context.Context, _ VerificationRequest) error {
	return nil
}

func (t *Telegram) String() string {
	result := t.Telegram.String()
	if t.AdminChannelID != "" {
		result += " with admin notifications to " + t.AdminChannelID
	}
	if t.UserNotifications {
		result += " with user notifications enabled"
	}
	return result
}
