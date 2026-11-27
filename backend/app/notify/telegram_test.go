package notify

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ntf "github.com/go-pkgz/notify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
)

func TestTelegram_NewError(t *testing.T) {
	// a bot API that refuses the token, as the public one does for an empty token
	ts := httptest.NewServer(http.NotFoundHandler())
	defer ts.Close()
	tb, err := NewTelegram(TelegramParams{APIURL: ts.URL})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected telegram API status code 404")
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
	assert.Contains(t, err.Error(), "problem sending user telegram notification about comment ID 999 to \"test_user_channel\"")
	assert.Contains(t, err.Error(), "problem sending admin telegram notification about comment ID 999 to remark_test")

	// test buildMessage separately for message text
	res := tb.buildMessage(Request{Comment: c, parent: cp})
	assert.Equal(t, `<a href="http://example.org/#remark42__comment-999">from</a> -> <a href="http://example.org/#remark42__comment-">to</a>

some text

"<i>some parent text with a link and special text:&amp; &lt; &gt; &amp;</i>"

↦  <a href="http://example.org/">[test title]</a>`,
		res)

	// special case for text with h1-h6 header
	ch := store.Comment{Text: "<h1>Hello</h1><h6>World</h6>", ID: "555", Locator: store.Locator{URL: "http://example.org/"}}
	ch.User.Name = "from"
	res = tb.buildMessage(Request{Comment: ch})
	assert.Equal(t, `<a href="http://example.org/#remark42__comment-555">from</a>

<b>Hello</b><i><b>World</b></i>`,
		res)

	// prune string keeping HTML closing tags
	c = store.Comment{
		Text: "<b>Lorem ipsum <i>dolor sit amet</i>, consectetur adipiscing <code>elit, sed do eiusmod tempor incididunt</code> ut labore et dolore magna aliqua.</b>",
	}
	res = tb.buildMessage(Request{Comment: c})
	assert.Equal(t, `<a href="#remark42__comment-"></a>

<b>Lorem ipsum <i>dolor sit amet</i>, consectetur adipiscing <code>elit, sed do eiusmod tempor incididunt</code> ut...</b>`, res)
}

func TestTelegram_BuildMessageDropsLinks(t *testing.T) {
	tb := Telegram{}
	c := store.Comment{ID: "999", ParentID: "1", Locator: store.Locator{URL: "https://example.org/"},
		Text: `<p>see <b><a href="https://evil.example/login" rel="nofollow ugc">Sign in</a></b> now</p>`}
	c.User.Name = "from"
	cp := store.Comment{ID: "1", Text: `<p>parent <a href="https://evil.example/">https://example.org/help</a></p>`}
	cp.User.Name = "to"

	assert.Equal(t, `<a href="https://example.org/#remark42__comment-999">from</a> -> <a href="https://example.org/#remark42__comment-1">to</a>

see <b>Sign in</b> now

"<i>parent https://example.org/help</i>"`, tb.buildMessage(Request{Comment: c, parent: cp}))
}

// TestTelegram_BuildMessageNestsAndBoundsTags covers comment text Telegram would refuse: tags that do not
// nest, as in comments stored before their tags were balanced, and so many tags that the message passes
// Telegram's limit on its size. The quoted parent goes through the same steps as the comment
func TestTelegram_BuildMessageNestsAndBoundsTags(t *testing.T) {
	tb := Telegram{}

	c := store.Comment{Text: "<p><wbr></i>Buy followers at spam.example</p>\n", ParentID: "1", ID: "999",
		Locator: store.Locator{URL: "http://example.org/"}}
	c.User.Name = "from"
	cp := store.Comment{Text: "<p><del>a <ins>b</del> c</ins> <b/> <b>left open</p>", ID: "1"}
	cp.User.Name = "to"
	res := tb.buildMessage(Request{Comment: c, parent: cp})
	assert.Equal(t, `<a href="http://example.org/#remark42__comment-999">from</a> -> <a href="http://example.org/#remark42__comment-1">to</a>

Buy followers at spam.example

"<i><del>a <ins>b</ins></del> c  <b>left open</b></i>"`, res)
	assert.Empty(t, nestingProblems(res))

	// a comment that opens 140 different elements and leaves them open, followed by 40 short paragraphs.
	// The paragraphs reopen all of them when the comment is stored, multiplying its tags by 40
	var typed strings.Builder
	typed.WriteString("nice post ")
	for i := range 140 {
		fmt.Fprintf(&typed, "<b id=%d>", i)
	}
	typed.WriteString(strings.Repeat("<p>.", 40))
	heavy := store.NewCommentFormatter().Format(store.Comment{Text: typed.String()}, false)
	heavy.Sanitize()
	require.Greater(t, len(heavy.Text), 80_000, "the stored comment did not multiply its tags")

	heavy.ID, heavy.ParentID, heavy.Locator = "1000", "1", c.Locator
	res = tb.buildMessage(Request{Comment: heavy, parent: store.Comment{ID: "1", Text: heavy.Text}})
	assert.Empty(t, nestingProblems(res))
	// Telegram refuses a message text over 32768 bytes; the comment and the quote each keep at most
	// commentTextLengthLimit bytes of text and commentMarkupLengthLimit of tags
	assert.Less(t, len(res), 2*(commentTextLengthLimit+commentMarkupLengthLimit)+200)
	assert.True(t, strings.HasPrefix(res, `<a href="http://example.org/#remark42__comment-1000"></a> -> `), res[:100])
}

func TestTelegram_BuildMessageShortensTitle(t *testing.T) {
	tb := Telegram{}
	title := func(postTitle string) string {
		c := store.Comment{Text: "text", ID: "1", PostTitle: postTitle, Locator: store.Locator{URL: "http://example.org/"}}
		res := tb.buildMessage(Request{Comment: c})
		_, link, found := strings.Cut(res, "↦  ")
		require.True(t, found, res)
		return strings.TrimSuffix(strings.TrimPrefix(link, `<a href="http://example.org/">`), "</a>")
	}

	assert.Equal(t, strings.Repeat("t", 252)+"...", title(strings.Repeat("t", 5000)))
	assert.Equal(t, strings.Repeat("t", 255), title(strings.Repeat("t", 255)), "a title of the limit's length is kept whole")
	assert.Equal(t, strings.Repeat("ж", 252)+"...", title(strings.Repeat("ж", 300)), "cut on characters, not bytes")
	assert.Equal(t, strings.Repeat("&lt;", 252)+"...", title(strings.Repeat("<", 300)), "cut before it is escaped")
}

func TestTelegram_SendVerification(t *testing.T) {
	tb := Telegram{}
	// empty VerificationRequest should return no error and do nothing, as well as any other
	assert.NoError(t, tb.SendVerification(context.Background(), VerificationRequest{}))
}
