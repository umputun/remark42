package notify

import (
	"context"
	"fmt"
	"time"

	log "github.com/go-pkgz/lgr"
	ntf "github.com/go-pkgz/notify"
	"github.com/hashicorp/go-multierror"
)

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

// buildMessage generates message for generic notification about new comment
func (t *Telegram) buildMessage(req Request) string {
	commentURLPrefix := req.Comment.Locator.URL + uiNav

	msg := fmt.Sprintf(`<a href=%q>%s</a>`, commentURLPrefix+req.Comment.ID, ntf.EscapeTelegramText(req.Comment.User.Name))

	if req.Comment.ParentID != "" {
		msg += fmt.Sprintf(" -> <a href=%q>%s</a>", commentURLPrefix+req.parent.ID, ntf.EscapeTelegramText(req.parent.User.Name))
	}

	msg += fmt.Sprintf("\n\n%s", ntf.TelegramSupportedHTML(req.Comment.Text))

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
