package notify

import (
	"context"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
	"github.com/slack-go/slack"
)

// Slack implements notify.Destination for Slack
type Slack struct {
	channelID   string
	channelName string
	client      *slack.Client
}

// NewSlack makes Slack bot for notifications
func NewSlack(token, channelName string, opts ...slack.Option) (*Slack, error) {

	if channelName == "" {
		channelName = "general"
	}

	client := slack.New(token, opts...)
	res := &Slack{client: client, channelName: channelName}

	channelID, err := res.findChannelIDByName(channelName)
	if err != nil {
		return nil, errors.Wrap(err, "can not find slack channel '"+channelName+"'")
	}

	res.channelID = channelID
	log.Printf("[DEBUG] create new slack notifier for chan %s", channelID)

	return res, nil
}

// Send to Slack channel
func (t *Slack) Send(ctx context.Context, req Request) error {

	log.Printf("[DEBUG] send slack notification, comment id %s", req.Comment.ID)

	user := req.Comment.User.Name
	if req.Comment.ParentID != "" {
		user += " → " + req.parent.User.Name
	}

	title := "↦ original comment"
	if req.Comment.PostTitle != "" {
		title = "↦ " + req.Comment.PostTitle
	}

	_, _, err := t.client.PostMessageContext(ctx, t.channelID,
		slack.MsgOptionText("New comment from "+user, false),
		slack.MsgOptionAttachments(
			slack.Attachment{
				TitleLink: req.Comment.Locator.URL + uiNav + req.Comment.ID,
				Title:     title,
				Text:      req.Comment.Orig,
			},
		),
	)

	return err

}

// SendVerification is not implemented for Slack
func (t *Slack) SendVerification(_ context.Context, _ VerificationRequest) error {
	return nil
}

func (t *Slack) String() string {
	return "slack: " + t.channelName + " (" + t.channelID + ")"
}

func (t *Slack) findChannelIDByName(name string) (string, error) {

	params := slack.GetConversationsParameters{}
	for {

		chans, next, err := t.client.GetConversations(&params)
		if err != nil {
			return "", err
		}

		for _, channel := range chans {
			if channel.Name == name {
				return channel.ID, nil
			}
		}

		if next == "" {
			break
		}
		params.Cursor = next

	}
	return "", errors.New("no such channel")
}
