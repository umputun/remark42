package notify

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	log "github.com/go-pkgz/lgr"
	ntf "github.com/go-pkgz/notify"
)

// slackEscaper escapes the characters Slack reads as markup in message text, as Slack asks for text written by
// users: unescaped, <url|label> becomes a link that shows only its label and <!channel> or <@user> a mention
var slackEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// Slack implements notify.Destination for Slack
type Slack struct {
	*ntf.Slack
	noVerification

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

	return s.Slack.Send(ctx, s.destination(req), "New comment from "+user)
}

// destination builds the slack: address carrying the attachment of the message. The attachment text is the
// comment as its author typed it, escaped for Slack. The post title is left as it is: a comment is read back
// sanitized before it is sent, so its title holds no < or >, and escaping it again would show its entities
func (s *Slack) destination(req Request) string {
	title := "↦ original comment"
	if req.Comment.PostTitle != "" {
		title = "↦ " + req.Comment.PostTitle
	}
	return fmt.Sprintf(
		"slack:%s?title=%s&attachmentText=%s&titleLink=%s",
		s.channelName,
		url.QueryEscape(title),
		url.QueryEscape(slackEscaper.Replace(req.Comment.Orig)),
		url.QueryEscape(req.Comment.Locator.URL+uiNav+req.Comment.ID),
	)
}

func (s *Slack) String() string {
	return s.Slack.String() + " for channel " + s.channelName + ""
}
