package notify

import (
	"context"
	"testing"

	ntf "github.com/go-pkgz/notify"
	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark42/backend/app/store"
)

func TestTelegram_NewError(t *testing.T) {
	tb, err := NewTelegram(TelegramParams{})
	assert.Error(t, err)
	assert.Nil(t, tb)
}

func TestTelegram_Send(t *testing.T) {
	tb := Telegram{
		AdminChannelID:    "remark_test",
		UserNotifications: true,
		Telegram:          &ntf.Telegram{}, // broken sender due to unset API
	}
	assert.Equal(t, "telegram notifications destination with admin notifications to remark_test with user notifications enabled", tb.String())
	c := store.Comment{Text: "some text", ParentID: "1", ID: "999", PostTitle: "[test title]", Locator: store.Locator{URL: "http://example.org/"}}
	c.User.Name = "from"
	cp := store.Comment{Text: `<p>some parent text with a <a href="http://example.org">link</a> and special text:<br>& < > &</p>`}
	cp.User.Name = "to"

	err := tb.Send(context.Background(), Request{Comment: c, parent: cp, Telegrams: []string{"test_user_channel"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "2 errors occurred")
	assert.Contains(t, err.Error(), "problem sending user telegram notification about comment ID 999 to \"test_user_channel\"")
	assert.Contains(t, err.Error(), "problem sending admin telegram notification about comment ID 999 to remark_test")

	// test buildMessage separately for message text
	res := tb.buildMessage(Request{Comment: c, parent: cp})
	assert.Equal(t, `<a href="http://example.org/#remark42__comment-999">from</a> -> <a href="http://example.org/#remark42__comment-">to</a>

some text

"<i>some parent text with a <a href="http://example.org">link</a> and special text:&amp; &lt; &gt; &amp;</i>"

↦  <a href="http://example.org/">[test title]</a>`,
		res)

	// special case for text with h1-h6 header
	ch := store.Comment{Text: "<h1>Hello</h1><h6>World</h6>", ID: "555", Locator: store.Locator{URL: "http://example.org/"}}
	ch.User.Name = "from"
	res = tb.buildMessage(Request{Comment: ch})
	assert.Equal(t, `<a href="http://example.org/#remark42__comment-555">from</a>

<b>Hello</b><i><b>World</b></i>`,
		res)
}

func TestTelegram_SendVerification(t *testing.T) {
	tb := Telegram{}
	// empty VerificationRequest should return no error and do nothing, as well as any other
	assert.NoError(t, tb.SendVerification(context.Background(), VerificationRequest{}))
}
