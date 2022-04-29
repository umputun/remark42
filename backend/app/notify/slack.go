package notify

import (
	"context"
	"fmt"
	"net/url"

	log "github.com/go-pkgz/lgr"
	ntf "github.com/go-pkgz/notify"
)

// Slack implements notify.Destination for Slack
type Slack struct {
	*ntf.Slack

	channelName string
}

// NewSlack makes Slack bot for notifications
func NewSlack(token, channelName string) *Slack {
	log.Printf("[DEBUG] create new slack notifier for chan %s", channelName)
	if channelName == "" {
		channelName = "general"
	}

	return &Slack{Slack: ntf.NewSlack(token), channelName: channelName}
}

// Send to Slack channel
func (s *Slack) Send(ctx context.Context, req Request) error {
	log.Printf("[DEBUG] send slack notification, comment id %s", req.Comment.ID)

	user := req.Comment.User.Name
	if req.Comment.ParentID != "" {
		user += " → " + req.parent.User.Name
	}

	title := "↦ original comment"
	if req.Comment.PostTitle != "" {
		title = "↦ " + req.Comment.PostTitle
	}

	destination := fmt.Sprintf(
		"slack:%s?title=%s&attachmentText=%s&titleLink=%s",
		s.channelName,
		url.QueryEscape(title),
		url.QueryEscape(req.Comment.Orig),
		url.QueryEscape(req.Comment.Locator.URL+uiNav+req.Comment.ID),
	)

	return s.Slack.Send(ctx, destination, "New comment from "+user)
}

// SendVerification is not implemented for Slack
func (s *Slack) SendVerification(_ context.Context, _ VerificationRequest) error {
	return nil
}

func (s *Slack) String() string {
	return s.Slack.String() + " for channel " + s.channelName + ""
}
