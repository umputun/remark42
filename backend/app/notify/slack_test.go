package notify

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestSlack_DestinationEscapesCommentText(t *testing.T) {
	ts := NewSlack("", "C024BE91L")

	// the title as a sanitized comment carries it, Orig as the author typed it
	c := store.Comment{PostTitle: "title &lt;1&gt; &amp; more", ID: "999", Locator: store.Locator{URL: "https://example.com/post"},
		Orig: "see <https://example.org/login|https://example.com/admin> <!channel> <@U024BE7LH> a > b && c &lt; d\n" +
			"[link](https://example.com) *bold*"}
	dest := ts.destination(Request{Comment: c})

	channel, query, found := strings.Cut(dest, "?")
	require.True(t, found, dest)
	assert.Equal(t, "slack:C024BE91L", channel)
	params, err := url.ParseQuery(query)
	require.NoError(t, err)
	assert.Equal(t, "see &lt;https://example.org/login|https://example.com/admin&gt; &lt;!channel&gt; &lt;@U024BE7LH&gt; "+
		"a &gt; b &amp;&amp; c &amp;lt; d\n[link](https://example.com) *bold*", params.Get("attachmentText"),
		"Slack markup in the comment arrives as text")
	assert.Equal(t, "↦ title &lt;1&gt; &amp; more", params.Get("title"), "a sanitized title is not escaped twice")
	assert.Equal(t, "https://example.com/post#remark42__comment-999", params.Get("titleLink"))

}

func TestSlack_Name(t *testing.T) {
	tb := NewSlack("", "test-channel")
	assert.Equal(t, "slack notifications destination for channel test-channel", tb.String())
}

func TestSlack_SendVerification(t *testing.T) {
	ts := NewSlack("", "")
	assert.NoError(t, ts.SendVerification(context.Background(), VerificationRequest{}))
}
