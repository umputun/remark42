package notify

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark/backend/app/store"
)

func TestEmail(t *testing.T) {
	// Test failed start of the server.
	email, err := NewEmail(context.Background(), EmailParams{})
	assert.Error(t, err, "No connection established with empty address and port zero")
	assert.NotNil(t, email, "despite the error we got object reference")
	assert.Equal(t, email.server, "")
	assert.Equal(t, email.port, 0)
	assert.Equal(t, email.username, "")
	assert.Equal(t, email.password, "")
	assert.Equal(t, email.keepAlive, 30*time.Second, "default value if keepAlive is not defined is set to 30s")
	assert.Equal(t, "email: ''@'':0", email.String())

	c := store.Comment{Text: "some text", ParentID: "1"}
	c.User.Name = "from"
	c.PostTitle = "post title"
	c.Locator.URL = "//example.org"
	c.Orig = "orig"
	cp := store.Comment{Text: "some parent text"}
	cp.User.Name = "to"
	req := request{comment: c, parent: cp}

	ctx, cancel := context.WithCancel(context.Background())
	go func() { cancel(); <-ctx.Done() }()
	assert.EqualError(t, email.Send(ctx, req), "context canceled")

	go func() { _ = email.Send(context.Background(), req) }()
	message := <-email.sendChan
	assert.Equal(t, message.GetHeader("Subject"), []string{"New comment for \"post title\""})

	assert.Equal(t, prepareBody(req), "from → to\n\norig\n\n↦ [post title](//example.org#remark42__comment-)")
}
