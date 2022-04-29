package notify

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark42/backend/app/store"
)

func TestSlack_New(t *testing.T) {
	ts := NewSlack("", "")
	assert.NotNil(t, ts)
	assert.Equal(t, "general", ts.channelName)
}

func TestSlack_Send(t *testing.T) {
	ts := NewSlack("", "")

	c := store.Comment{PostTitle: "test title", Text: "some text", ParentID: "1", ID: "999"}
	c.User.Name = "from"
	cp := store.Comment{Text: "some parent text"}
	cp.User.Name = "to"

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ts.Send(ctx, Request{Comment: c, parent: cp})
	assert.Error(t, err)
}

func TestSlack_Name(t *testing.T) {
	tb := NewSlack("", "test-channel")
	assert.Equal(t, "slack notifications destination for channel test-channel", tb.String())
}

func TestSlack_SendVerification(t *testing.T) {
	ts := NewSlack("", "")
	assert.NoError(t, ts.SendVerification(context.Background(), VerificationRequest{}))
}
