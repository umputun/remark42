//go:build e2e

package e2e

import (
	"fmt"
	neturl "net/url"
	"strings"
	"testing"
	"time"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComment_PostRendersMarkdownAndSurvivesReload(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	// the marker has to be free of markdown syntax: the backend renders the comment, so a
	// filter on the raw source would never match the rendered text
	text := "hello from " + runID
	posted := postCommentMatching(t, frame, text+" with **bold**", text)

	// the backend renders the markdown, so a plain-text match would pass even if it stopped
	bold, err := posted.Locator(".raw-content strong").InnerText()
	require.NoError(t, err)
	assert.Equal(t, "bold", bold)

	frame = reload(t, page)
	posted = comment(frame, text)
	waitVisible(t, posted)
	// the rendered markup has to survive the round trip too, not just the words
	bold, err = posted.Locator(".raw-content strong").InnerText()
	require.NoError(t, err)
	assert.Equal(t, "bold", bold)
}

func TestComment_ReplyNestsUnderItsParent(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	parent := "parent " + runID
	postComment(t, frame, parent)

	require.NoError(t, actions(frame, parent).Locator(`button:has-text("Reply")`).Click())

	reply := "reply " + runID
	submitForm(t, replyForm(t, frame), reply)

	// nesting is the point: the reply has to live inside the parent's thread, not beside it
	parentThread := frame.Locator("[aria-expanded]", playwright.FrameLocatorLocatorOptions{HasText: parent}).First()
	waitVisible(t, parentThread.Locator("article", playwright.LocatorLocatorOptions{HasText: reply}))
}

func TestComment_EditWithinTheDeadline(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	original := "before edit " + runID
	postComment(t, frame, original)

	// the countdown only renders while the comment is still editable
	waitVisible(t, actions(frame, original).Locator(`[role="timer"]`))
	require.NoError(t, actions(frame, original).Locator(`button:has-text("Edit")`).Click())

	edited := "after edit " + runID
	submitForm(t, replyForm(t, frame), edited)

	waitVisible(t, comment(frame, edited))
	// an edit replaces the comment rather than adding one, and counting is the only sound
	// way to say the old text is gone: a text filter cannot tell absent from off screen
	assert.Equal(t, 1, articleCount(t, frame), "editing should not add a comment")
	txt, err := frame.Locator("article").First().InnerText()
	require.NoError(t, err)
	assert.NotContains(t, txt, original)

	// the DOM update comes from the response, so without a reload a handler that returned the
	// edited comment without storing it would pass
	frame = reload(t, page)
	waitVisible(t, comment(frame, edited))
	txt, err = frame.Locator("article").First().InnerText()
	require.NoError(t, err)
	assert.NotContains(t, txt, original, "the edit should have been stored, not just rendered")
}

// TestComment_EditExpiresAfterTheDeadline runs against the second instance, whose edit window
// is short enough to wait out and long enough that the setup fits inside it. That instance
// offers anonymous auth only, see compose-e2e-test.yml.
// editWindow mirrors EDIT_TIME on the short-edit instance in compose-e2e-test.yml
const editWindow = 15 * time.Second

func TestComment_EditExpiresAfterTheDeadline(t *testing.T) {
	page := newPage(t)
	url := threadURLOn(t, shortEditURL)
	frame := openURL(t, page, url)
	signInAnon(t, frame, "expirytester")

	text := "expires " + runID
	postComment(t, frame, text)

	editButton := actions(frame, text).Locator(`button:has-text("Edit")`)
	timer := actions(frame, text).Locator(`[role="timer"]`)
	waitVisible(t, editButton)
	waitVisible(t, timer)

	id, err := comment(frame, text).GetAttribute("id")
	require.NoError(t, err)
	commentID := strings.TrimPrefix(id, "remark42__comment-")

	// the countdown fires onTimePassed, which drops the edit affordance entirely. the wait has
	// to outlast the window itself, which started when the comment was posted
	expiry := playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateHidden,
		Timeout: playwright.Float(float64((waitTimeout + editWindow).Milliseconds())),
	}
	require.NoError(t, editButton.WaitFor(expiry))
	require.NoError(t, timer.WaitFor(expiry))

	// the button going away is only the widget being polite. the deadline is enforced by the
	// backend, and without asking it directly this test would still pass with that guard
	// removed, so put the request in from the signed-in page itself
	edit := fmt.Sprintf("%s/api/v1/comment/%s?site=remark&url=%s", shortEditURL, commentID, neturl.QueryEscape(url))
	status, body := pageFetch(t, page, "PUT", edit, map[string]string{"text": "edited after the deadline"})
	assert.Equal(t, 400, status, "the backend should refuse an edit past the deadline")
	assert.Contains(t, body, `"code":10`, "and say so with ErrCommentEditExpired")
}

func TestComment_DeleteRemovesTheText(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)
	// deliberately not the dev user: ADMIN_SHARED_ID makes that one an admin, and the widget
	// sends admins to the admin endpoint, so signing in there would leave the path every
	// ordinary reader takes untested
	signInAnon(t, frame, "deletetester")

	text := "doomed " + runID
	survivor := "survivor " + runID
	postComment(t, frame, text)
	postComment(t, frame, survivor)

	// delete is gated by window.confirm; without a handler playwright dismisses it and the
	// comment quietly survives
	page.OnDialog(func(d playwright.Dialog) { _ = d.Accept() })

	require.NoError(t, actions(frame, text).Locator(`button:has-text("Delete")`).Click())

	// the widget does not remove the node, it swaps the text for a tombstone. asserting the
	// tombstone is present says more than asserting the old text is gone, which a comment
	// scrolled out of view would also satisfy
	waitVisible(t, comment(frame, "This comment was deleted"))

	// and it stays gone rather than reappearing from cache on the next load. the survivor is
	// what makes this assertion mean anything: without it a thread that had not rendered yet
	// would satisfy "the deleted text is absent" just as well
	frame = reload(t, page)
	waitVisible(t, comment(frame, survivor))
	assert.Equal(t, 1, articleCount(t, frame), "the deleted comment should be gone from the thread")
}
