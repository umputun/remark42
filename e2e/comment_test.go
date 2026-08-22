//go:build e2e

package e2e

import (
	"encoding/base64"
	"fmt"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
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

// TestComment_EditKeepsTheOriginalSource covers what the widget puts back in the textarea when a
// comment is edited. The thread shows rendered html, so the form has to hold the source it was
// posted with: #2040 shipped a version that handed back the rendered text, and everything the
// author had written in entities or markup was lost on the next save
func TestComment_EditKeepsTheOriginalSource(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	// entities, markup and a character outside latin1, each of which a render-and-read-back
	// round trip mangles differently
	source := "5 &lt; 10 &amp; **bold** <b>tag</b> ю " + runID
	postCommentMatching(t, frame, source, runID)

	require.NoError(t, actions(frame, runID).Locator(`button:has-text("Edit")`).Click())
	form := replyForm(t, frame)

	got, err := form.Locator("textarea").InputValue()
	require.NoError(t, err)
	assert.Equal(t, source, got, "the edit form has to hold the source that was posted, not the rendered comment")

	submitForm(t, form, source+" edited")
	waitVisible(t, comment(frame, "edited"))

	frame = reload(t, page)
	require.NoError(t, actions(frame, runID).Locator(`button:has-text("Edit")`).Click())
	got, err = replyForm(t, frame).Locator("textarea").InputValue()
	require.NoError(t, err)
	assert.Equal(t, source+" edited", got, "the stored source has to survive the round trip through the backend")
}

// TestComment_DraftSurvivesReloadAndClearsAfterPost covers the local draft. A reader who reloads
// mid-sentence keeps what they typed, and a reader who posts does not get it handed back
func TestComment_DraftSurvivesReloadAndClearsAfterPost(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	draft := "half written " + runID
	require.NoError(t, frame.Locator(commentFormSel).First().Locator("textarea").Fill(draft))

	frame = reload(t, page)
	textarea := frame.Locator(commentFormSel).First().Locator("textarea")
	eventually(t, waitTimeout, "the draft was not restored after the reload", func() bool {
		v, err := textarea.InputValue()
		return err == nil && v == draft
	})

	postCommentMatching(t, frame, draft, draft)

	frame = reload(t, page)
	got, err := frame.Locator(commentFormSel).First().Locator("textarea").InputValue()
	require.NoError(t, err)
	assert.Empty(t, got, "a posted draft has to be cleared, or the reader is handed their own comment back")
}

// TestComment_PostFailureKeepsTheText covers the path a reader hits when the server refuses the
// comment. The text is the only copy they have, so it has to stay in the form, and the failure has
// to say something rather than swallowing itself
func TestComment_PostFailureKeepsTheText(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	require.NoError(t, page.Route("**/api/v1/comment?**", func(route playwright.Route) {
		require.NoError(t, route.Fulfill(playwright.RouteFulfillOptions{
			Status:      playwright.Int(http.StatusBadRequest),
			ContentType: playwright.String("application/json"),
			Body:        playwright.String(`{"code":19,"details":"comment contains restricted words","error":"rejected"}`),
		}))
	}))

	text := "rejected " + runID
	form := frame.Locator(commentFormSel).First()
	submitForm(t, form, text)

	waitVisible(t, form.Locator(`p[role="alert"]`))

	got, err := form.Locator("textarea").InputValue()
	require.NoError(t, err)
	assert.Equal(t, text, got, "a refused comment has to stay in the form, it is the only copy the reader has")

	require.NoError(t, page.Unroute("**/api/v1/comment?**"))
	require.NoError(t, form.Locator(`button[type="submit"]`).Click())
	waitVisible(t, comment(frame, text))
}

// TestComment_AdminPinsAndVerifies covers two moderator actions that change what every reader
// sees. Both are server-side, so the assertions come after a reload on a second reader's page
// rather than from the moderator's own optimistic render
func TestComment_AdminPinsAndVerifies(t *testing.T) {
	text := "moderated " + runID

	// verification is a property of the user and outlives the run in the stack's database, so a
	// fixed name is only verifiable once: the next run would toggle an already verified author
	// off and wait for a badge that is being taken away
	author := newPage(t)
	authorFrame := openThread(t, author)
	signInAnon(t, authorFrame, "moderated"+runID[len(runID)-6:])
	postComment(t, authorFrame, text)

	admin := newPage(t)
	adminFrame := openURL(t, admin, threadURL(t))
	signInDev(t, admin, adminFrame)

	admin.OnDialog(func(d playwright.Dialog) { _ = d.Accept() })
	require.NoError(t, actions(adminFrame, text).Locator(`button:has-text("Pin")`).Click())
	// pinning re-renders the thread, and a click that lands during that render is lost, so wait
	// for the pinned region to exist before touching the same comment again
	waitVisible(t, adminFrame.Locator(`[role="region"][aria-label="Pinned comments"]`))

	// the verification toggle sits in the comment header beside the author, not in the action bar
	require.NoError(t, comment(adminFrame, text).Locator(`[title="Toggle verification"]`).First().Click())
	waitVisible(t, comment(adminFrame, text).Locator(`[title="Verified user"]`).First())

	reader := newPage(t)
	readerFrame := openURL(t, reader, threadURL(t))

	pinned := readerFrame.Locator(`[role="region"][aria-label="Pinned comments"]`)
	waitVisible(t, pinned)
	waitVisible(t, pinned.Locator("article", playwright.LocatorLocatorOptions{HasText: text}))
	waitVisible(t, comment(readerFrame, text).Locator(`[title="Verified user"]`).First())

	// unpinning has to reach every reader too, so the region goes away rather than merely
	// emptying on the moderator's own page
	require.NoError(t, actions(adminFrame, text).Locator(`button:has-text("Unpin")`).Click())

	readerFrame = reload(t, reader)
	waitHidden(t, readerFrame.Locator(`[role="region"][aria-label="Pinned comments"]`),
		"the comment was unpinned but readers still see the pinned region")
}

// TestComment_ImageUploadRendersAndRecovers covers the upload path end to end, which nothing
// exercised in a browser: the file input, the temporary markdown the form writes while the request
// is in flight, the final picture URL, and the image actually loading in the posted comment.
// The second half is the part a reader notices most, since a failed upload that leaves the
// placeholder behind corrupts what they were writing
func TestComment_ImageUploadRendersAndRecovers(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	// a 1x1 png, written out rather than fetched so the case does not depend on a fixture file
	png, err := base64.StdEncoding.DecodeString(
		"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==")
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "pixel.png")
	require.NoError(t, os.WriteFile(path, png, 0o600))

	form := frame.Locator(commentFormSel).First()
	textarea := form.Locator("textarea")

	t.Run("a failed upload leaves the text as it was", func(t *testing.T) {
		require.NoError(t, page.Route("**/api/v1/picture**", func(route playwright.Route) {
			// held briefly so the in-flight state is observable: without it the placeholder
			// comes and goes inside one frame, and "the text is unchanged" would hold just as
			// well for an upload that never started
			time.Sleep(300 * time.Millisecond)
			require.NoError(t, route.Fulfill(playwright.RouteFulfillOptions{
				Status:      playwright.Int(http.StatusInternalServerError),
				ContentType: playwright.String("application/json"),
				Body:        playwright.String(`{"code":0,"details":"upload failed","error":"nope"}`),
			}))
		}))
		defer func() { require.NoError(t, page.Unroute("**/api/v1/picture**")) }()

		written := "before the upload " + runID
		require.NoError(t, textarea.Fill(written))
		require.NoError(t, form.Locator(`input[type="file"]`).SetInputFiles(path))

		eventually(t, waitTimeout, "the form never showed the upload in progress", func() bool {
			v, verr := textarea.InputValue()
			return verr == nil && v != written
		})

		waitVisible(t, form.Locator(`p[role="alert"]`))
		eventually(t, waitTimeout, "the upload placeholder was left in the text after the failure", func() bool {
			v, verr := textarea.InputValue()
			return verr == nil && v == written
		})
	})

	t.Run("an uploaded image is posted and renders", func(t *testing.T) {
		require.NoError(t, textarea.Fill("with an image "+runID+" "))
		require.NoError(t, form.Locator(`input[type="file"]`).SetInputFiles(path))

		eventually(t, waitTimeout, "the upload never produced a picture url", func() bool {
			v, verr := textarea.InputValue()
			return verr == nil && strings.Contains(v, "/api/v1/picture/")
		})

		require.NoError(t, form.Locator(`button[type="submit"]`).Click())
		posted := comment(frame, "with an image "+runID)
		waitVisible(t, posted)

		img := posted.Locator(`img[src*="/api/v1/picture/"]`).First()
		waitVisible(t, img)

		// visible is not loaded: a broken src renders as an empty box, and naturalWidth is the
		// only thing that says the bytes came back
		eventually(t, waitTimeout, "the posted image never loaded", func() bool {
			w, jerr := img.Evaluate("el => el.naturalWidth", nil)
			n, ok := w.(int)
			return jerr == nil && ok && n > 0
		})
	})
}

// TestComment_BlockedAuthorCannotPost covers the refusal a blocked author meets. The backend
// answers with its own code, and the widget has to turn that into something the reader can read
// rather than swallowing it, which is the half no unit test can speak for
func TestComment_BlockedAuthorCannotPost(t *testing.T) {
	text := "before the block " + runID

	// the block is permanent and the stack's database outlives the run, so a fixed name would
	// only be postable once: every later run would find the author already blocked
	author := newPage(t)
	authorFrame := openThread(t, author)
	signInAnon(t, authorFrame, "blocked"+runID[len(runID)-6:])
	postComment(t, authorFrame, text)

	admin := newPage(t)
	adminFrame := openURL(t, admin, threadURL(t))
	signInDev(t, admin, adminFrame)

	admin.OnDialog(func(d playwright.Dialog) { _ = d.Accept() })
	_, err := actions(adminFrame, text).Locator("select").SelectOption(playwright.SelectOptionValues{
		Values: &[]string{"permanently"},
	})
	require.NoError(t, err)

	// the author's own page still believes it can post, which is the point: the refusal has to
	// come back from the server and be shown
	form := authorFrame.Locator(commentFormSel).First()
	submitForm(t, form, "after the block "+runID)

	// not scoped to the form: the widget re-renders the whole panel once the server reports the
	// author as blocked, so where the message lands is not the point, only that it is said
	waitVisible(t, authorFrame.Locator("text=blocked").First())
}

// TestComment_ReadOnlyThreadTakesTheFormAway covers the admin switch that closes a thread. A
// reader arriving afterwards has to find no way to post, and the state has to come from the
// server rather than from the admin's own page
func TestComment_ReadOnlyThreadTakesTheFormAway(t *testing.T) {
	page := newPage(t)
	url := threadURL(t)
	frame := openURL(t, page, url)
	signInDev(t, page, frame)

	// the admin panel swaps its own button rather than showing the read-only notice, which is
	// what an ordinary reader gets
	require.NoError(t, frame.Locator(`button:has-text("Disable comments")`).Click())
	waitVisible(t, frame.Locator(`button:has-text("Enable comments")`))

	// not openURL: it waits for a comment form, and a read-only thread is exactly the case with
	// no form to wait for
	reader := newPage(t)
	pauseForAuthLimit()
	_, err := reader.Goto(url, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded})
	require.NoError(t, err)

	readerFrame := reader.FrameLocator("#remark42 iframe")
	waitVisible(t, readerFrame.Locator(`text=Read-only`))
	waitHidden(t, readerFrame.Locator(commentFormSel).First(),
		"the thread is read-only but a reader is still shown a comment form")

	require.NoError(t, frame.Locator(`button:has-text("Enable comments")`).Click())
	waitVisible(t, frame.Locator(commentFormSel).First())
}
