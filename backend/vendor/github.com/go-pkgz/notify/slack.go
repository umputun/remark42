package notify

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/slack-go/slack"
)

// Slack notifications client
type Slack struct {
	client *slack.Client
}

// NewSlack makes Slack client for notifications
func NewSlack(token string, opts ...slack.Option) *Slack {
	return &Slack{client: slack.New(token, opts...)}
}

// Send sends the message over Slack, with "title", "titleLink" and "attachmentText" parsed from destination field
// with "slack:" schema same way "mailto:" schema is constructed.
//
// Example:
//
// - slack:channelName
// - slack:channelID
// - slack:userID
// - slack:channel?title=title&attachmentText=test%20text&titleLink=https://example.org
func (s *Slack) Send(ctx context.Context, destination, text string) error {
	channelID, attachment, err := s.parseDestination(destination)
	if err != nil {
		return fmt.Errorf("problem parsing destination: %w", err)
	}
	options := []slack.MsgOption{slack.MsgOptionText(text, false)}
	if attachment.Title != "" {
		options = append(options, slack.MsgOptionAttachments(attachment))
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, _, err = s.client.PostMessageContext(ctx, channelID, options...)
		return err
	}
}

// Schema returns schema prefix supported by this client
func (s *Slack) Schema() string {
	return "slack"
}

func (s *Slack) String() string {
	return "slack notifications destination"
}

// parses "slack:" in a manner "mailto:" URL is parsed url and returns channelID and attachment.
// if channelID is channel name and not ID (starting with C for channel and with U for user),
// then it will be resolved to ID.
func (s *Slack) parseDestination(destination string) (string, slack.Attachment, error) {
	// parse URL
	u, err := url.Parse(destination)
	if err != nil {
		return "", slack.Attachment{}, err
	}
	if u.Scheme != "slack" {
		return "", slack.Attachment{}, fmt.Errorf("unsupported scheme %s, should be slack", u.Scheme)
	}
	channelID := u.Opaque
	if !strings.HasPrefix(u.Opaque, "C") && !strings.HasPrefix(u.Opaque, "U") {
		channelID, err = s.findChannelIDByName(u.Opaque)
		if err != nil {
			return "", slack.Attachment{}, fmt.Errorf("problem retrieving channel ID for #%s: %w", u.Opaque, err)
		}
	}

	return channelID,
		slack.Attachment{
			Title:     u.Query().Get("title"),
			TitleLink: u.Query().Get("titleLink"),
			Text:      u.Query().Get("attachmentText"),
		}, nil
}

func (s *Slack) findChannelIDByName(name string) (string, error) {
	params := slack.GetConversationsParameters{}
	for {
		channels, next, err := s.client.GetConversations(&params)
		if err != nil {
			return "", err
		}

		for i := range channels {
			if channels[i].Name == name {
				return channels[i].ID, nil
			}
		}

		if next == "" {
			break
		}
		params.Cursor = next
	}
	return "", errors.New("no such channel")
}
