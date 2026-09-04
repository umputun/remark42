//go:build e2e

package e2e

import (
	"encoding/base64"
	"fmt"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComment_PostRendersMarkdownAndSurvivesReload(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	original := "before edit " + runID
	postComment(t, frame, original)

	// the countdown only renders while the comment is still editable
	waitVisible(t, actions(frame, original).Locator(`[role="timer"]`))
	require.NoError(t, actions(frame, original).Locator(`button:has-text("Edit")`).Click())
	// the action becomes its own way out; leaving Edit in place strands a reader in the editor
	waitVisible(t, actions(frame, original).Locator(`button:has-text("Cancel")`))
	waitHidden(t, actions(frame, original).Locator(`button:has-text("Edit")`),
		"the Edit action stayed beside Cancel while the comment was being edited")

	edited := "after edit " + runID
	submitForm(t, replyForm(t, frame), edited)

	waitVisible(t, comment(frame, edited))
	// an edit replaces the comment instead of adding one, and counting is the only sound
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
	t.Parallel()

	page := newPage(t)
	url := threadURLOn(t, shortEditURL)
	frame := openURL(t, page, url)
	signInAnon(t, page, frame, "expirytester")

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
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)
	// deliberately not the dev user: ADMIN_SHARED_ID makes that one an admin, and the widget
	// sends admins to the admin endpoint, so signing in there would leave the path every
	// ordinary reader takes untested
	signInAnon(t, page, frame, "deletetester")

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

	// and it stays gone instead of reappearing from cache on the next load. the survivor is
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
	t.Parallel()

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
	t.Parallel()

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
// to say something instead of swallowing itself
func TestComment_PostFailureKeepsTheText(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	require.NoError(t, page.Route("**/api/v1/comment?**", func(route playwright.Route) {
		if err := route.Fulfill(playwright.RouteFulfillOptions{
			Status:      playwright.Int(http.StatusBadRequest),
			ContentType: playwright.String("application/json"),
			Body:        playwright.String(`{"code":19,"details":"comment contains restricted words","error":"rejected"}`),
		}); err != nil {
			t.Errorf("fulfill refused comment response: %v", err)
		}
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
// and not from the moderator's own optimistic render
func TestComment_AdminPinsAndVerifies(t *testing.T) {
	t.Parallel()

	text := "moderated " + runID

	// verification is a property of the user and outlives the run in the stack's database, so a
	// fixed name is only verifiable once: the next run would toggle an already verified author
	// off and wait for a badge that is being taken away
	author := newPage(t)
	authorFrame := openThread(t, author)
	signInAnon(t, author, authorFrame, anonName("moderated"))
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
	// nothing marks an author verified until an admin says so, and the icon is what says it did
	waitHidden(t, comment(adminFrame, text).Locator(`[title="Verified user"]`).First(),
		"the author was shown as verified before anyone verified them")
	require.NoError(t, comment(adminFrame, text).Locator(`[title="Toggle verification"]`).First().Click())
	waitVisible(t, comment(adminFrame, text).Locator(`[title="Verified user"]`).First())

	reader := newPage(t)
	readerFrame := openURL(t, reader, threadURL(t))

	pinned := readerFrame.Locator(`[role="region"][aria-label="Pinned comments"]`)
	waitVisible(t, pinned)
	waitVisible(t, pinned.Locator("article", playwright.LocatorLocatorOptions{HasText: text}))
	waitVisible(t, comment(readerFrame, text).Locator(`[title="Verified user"]`).First())

	// unpinning has to reach every reader too, so the region goes away instead of merely
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
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	// a 1x1 png, written out inline so the case does not depend on a fixture file
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
			if err := route.Fulfill(playwright.RouteFulfillOptions{
				Status:      playwright.Int(http.StatusInternalServerError),
				ContentType: playwright.String("application/json"),
				Body:        playwright.String(`{"code":0,"details":"upload failed","error":"nope"}`),
			}); err != nil {
				t.Errorf("fulfill failed upload response: %v", err)
			}
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
// instead of swallowing it, which is the half no unit test can speak for
func TestComment_BlockedAuthorCannotPost(t *testing.T) {
	t.Parallel()

	text := "before the block " + runID

	// the block is permanent and the stack's database outlives the run, so a fixed name would
	// only be postable once: every later run would find the author already blocked
	author := newPage(t)
	authorFrame := openThread(t, author)
	signInAnon(t, author, authorFrame, anonName("blocked"))
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
// server and not from the admin's own page
func TestComment_ReadOnlyThreadTakesTheFormAway(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	url := threadURL(t)
	frame := openURL(t, page, url)
	signInDev(t, page, frame)

	// a comment to hang the Reply action on. Without one the thread is empty and an assertion that
	// Reply is gone passes whether or not read-only has anything to do with it
	text := "locked thread reply " + runID
	postComment(t, frame, text)
	waitVisible(t, actions(frame, text).Locator(`button:has-text("Reply")`))

	// the admin panel swaps its own button instead of showing the read-only notice, which is
	// what an ordinary reader gets
	require.NoError(t, frame.Locator(`button:has-text("Disable comments")`).Click())
	waitVisible(t, frame.Locator(`button:has-text("Enable comments")`))

	// not openURL: it waits for a comment form, and a read-only thread is exactly the case with
	// no form to wait for
	reader := newPage(t)
	_, err := reader.Goto(url, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded})
	require.NoError(t, err)

	readerFrame := reader.FrameLocator("#remark42 iframe")
	// an exact match on the status: `text=` is a case-insensitive substring, so any comment whose
	// own text contains the phrase would satisfy it too, and two matches is a strict-mode failure
	waitVisible(t, readerFrame.Locator(`text="Read-only"`))
	waitHidden(t, readerFrame.Locator(commentFormSel).First(),
		"the thread is read-only but a reader is still shown a comment form")
	// the form and the per-comment Reply action are separate controls, and a thread that takes one
	// away has to take the other with it
	waitVisible(t, comment(readerFrame, text))
	waitHidden(t, readerFrame.Locator(`button:has-text("Reply")`).First(),
		"the thread is read-only but a reader is still offered Reply on a comment")

	require.NoError(t, frame.Locator(`button:has-text("Enable comments")`).Click())
	waitVisible(t, frame.Locator(commentFormSel).First())
}

// TestComment_AdminActionsKeepTheirOrder covers the order of the moderation actions, which nothing
// else asserts: every other case reaches one of them by name, so any arrangement passes.
//
// The order is what a moderator's hand learns, and Delete sits at the end of it deliberately. A
// reshuffle that moved Delete next to Hide would pass every other case in this file while making
// the destructive action the neighbor of a routine one.
func TestComment_AdminActionsKeepTheirOrder(t *testing.T) {
	t.Parallel()

	text := "admin action order " + runID
	author := newPage(t)
	authorFrame := openThread(t, author)
	signInAnon(t, author, authorFrame, anonName("actionorder"))
	postComment(t, authorFrame, text)

	admin := newPage(t)
	adminFrame := openURL(t, admin, threadURL(t))
	signInDev(t, admin, adminFrame)

	// the class is kept outside the css modules for this: the production bundle hashes the rest
	// and strips the data-testid the unit suite selects by
	additional := comment(adminFrame, text).Locator(".comment-actions-additional").First()
	waitVisible(t, additional)

	labels, err := additional.Locator("> *").AllTextContents()
	require.NoError(t, err)
	require.Len(t, labels, 5, "the moderation menu no longer holds five actions: %v", labels)

	want := []string{"Hide", "Copy", "Pin", "Block", "Delete"}
	for i, expected := range want {
		assert.Contains(t, labels[i], expected,
			"the moderation actions are out of order, wanted %v, got %v", want, labels)
	}

	// the label switches, which is the render branch under test. It says nothing about the
	// clipboard: copyComment sets isCopied outside its own catch, so the label appears whether the
	// write succeeded or threw
	require.NoError(t, additional.Locator(`button:has-text("Copy")`).Click())
	waitVisible(t, additional.Locator(`button:has-text("Copied!")`))
}

// TestComment_EditCountdownCountsDown covers what the countdown says, which nothing else reads.
// TestComment_EditWithinTheDeadline waits for the element and TestComment_EditExpiresAfterTheDeadline
// waits for it to go, so a timer rendering blank, showing the wrong unit, or frozen at its starting
// value passes both of them while telling the reader nothing about how long they have left.
//
// Runs against the short-edit instance so the window is small enough to watch a tick.
func TestComment_EditCountdownCountsDown(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openURL(t, page, threadURLOn(t, shortEditURL))
	signInAnon(t, page, frame, anonName("countdown"))

	text := "countdown " + runID
	postedAt := time.Now()
	postComment(t, frame, text)

	timer := actions(frame, text).Locator(`[role="timer"]`)
	waitVisible(t, timer)

	// the whole shape, not just the digits: trimming a suffix that is not there and parsing what
	// is left accepts a bare number, and the unit is part of what the reader is being told
	countdownShape := regexp.MustCompile(`^\d+s$`)
	seconds := func() int {
		t.Helper()
		// pollText and not TextContent: this runs inside eventually, and a bare read carries
		// playwright's own 30s default, which outlives the loop's budget and reports the wrong
		// failure
		shown, err := pollText(timer)
		require.NoError(t, err)
		trimmed := strings.TrimSpace(shown)
		require.Truef(t, countdownShape.MatchString(trimmed),
			"the countdown reads %q, which is not a number of seconds", shown)
		n, convErr := strconv.Atoi(strings.TrimSuffix(trimmed, "s"))
		require.NoError(t, convErr)
		return n
	}

	first := seconds()
	// bounded from below as well as above, or a countdown starting at 2s would satisfy the shape,
	// stay under the window and still decrease while telling the reader something wrong. The floor
	// comes from the time actually spent since the comment was posted rather than a fixed number,
	// since under -parallel 4 the setup can take seconds the scheduler decides
	spent := int(time.Since(postedAt).Seconds())
	assert.GreaterOrEqual(t, first, int(editWindow.Seconds())-spent-1,
		"the countdown started %ds below the edit window, having spent %ds getting there", int(editWindow.Seconds())-first, spent)
	// the widget rounds the remaining time up, so a fifteen second window reads 16 at the moment
	// the comment lands
	assert.LessOrEqual(t, first, int(editWindow.Seconds())+1,
		"the countdown starts above the edit window the instance was given")

	// it has to move, or a value hard-coded at the window would satisfy everything above
	eventually(t, waitTimeout, "the countdown never decreased", func() bool {
		return seconds() < first
	})
}

// TestComment_ActionsDependOnWhoseCommentItIs covers which moderation actions an ordinary reader is
// offered, which the rest of the suite only ever exercises as an admin: TestThread_HideUserRemovesTheirCommentsOnly
// clicks Hide from a reader signed in with signInDev, and the stack gives that user ADMIN_SHARED_ID,
// so making either action admin-only would pass every other case here.
//
// One reader, two comments, so each half is the other's positive control: a rule that dropped both
// actions everywhere would satisfy an absence assertion on its own.
func TestComment_ActionsDependOnWhoseCommentItIs(t *testing.T) {
	t.Parallel()

	other := newPage(t)
	otherFrame := openThread(t, other)
	signInAnon(t, other, otherFrame, anonName("actorsforeign"))
	foreign := "foreign comment " + runID
	postComment(t, otherFrame, foreign)

	reader := newPage(t)
	readerFrame := openURL(t, reader, threadURL(t))
	signInAnon(t, reader, readerFrame, anonName("actorsown"))
	own := "own comment " + runID
	postComment(t, readerFrame, own)

	waitVisible(t, comment(readerFrame, foreign))

	// on their own comment the reader may delete but has nobody to hide
	waitVisible(t, actions(readerFrame, own).Locator(`button:has-text("Delete")`))
	waitHidden(t, actions(readerFrame, own).Locator(`button:has-text("Hide")`),
		"the widget offered to hide the reader's own author")

	// on another reader's comment the reverse: hideable, not deletable
	waitVisible(t, actions(readerFrame, foreign).Locator(`button:has-text("Hide")`))
	waitHidden(t, actions(readerFrame, foreign).Locator(`button:has-text("Delete")`),
		"the widget offered an ordinary reader the delete action on someone else's comment")
}
