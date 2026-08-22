//go:build e2e

package e2e

import (
	"fmt"
	neturl "net/url"
	"strings"
	"testing"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// firstCommentText returns the text of the topmost comment in the thread. it returns the
// error rather than failing, because every caller polls it while the list re-renders and a
// momentarily empty list has to be retried rather than fail the test
func firstCommentText(frame playwright.FrameLocator) (string, error) {
	return pollText(frame.Locator("article").First())
}

const (
	sortOldestFirst = "+time"
	sortNewestFirst = "-time"
)

func setSort(t *testing.T, frame playwright.FrameLocator, value string) {
	t.Helper()
	_, err := frame.Locator(".sort-picker select").SelectOption(playwright.SelectOptionValues{
		Values: &[]string{value},
	})
	require.NoError(t, err)
}

func TestThread_SortChangeReordersComments(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	first := "sort first " + runID
	second := "sort second " + runID
	postComment(t, frame, first)
	postComment(t, frame, second)

	setSort(t, frame, sortOldestFirst)
	eventually(t, waitTimeout, "oldest-first did not put the first comment on top", func() bool {
		txt, err := firstCommentText(frame)
		return err == nil && strings.Contains(txt, first)
	})

	setSort(t, frame, sortNewestFirst)
	eventually(t, waitTimeout, "newest-first did not put the second comment on top", func() bool {
		txt, err := firstCommentText(frame)
		return err == nil && strings.Contains(txt, second)
	})

	// the choice is kept in localStorage and re-applied on the next load. it has to be the
	// oldest-first one: the default is -active, which orders two reply-free comments exactly
	// as -time does, so persisting newest-first would be indistinguishable from not
	// persisting anything at all
	setSort(t, frame, sortOldestFirst)
	eventually(t, waitTimeout, "oldest-first did not take effect before reload", func() bool {
		txt, err := firstCommentText(frame)
		return err == nil && strings.Contains(txt, first)
	})

	frame = reload(t, page)
	eventually(t, waitTimeout, "sort choice did not survive reload", func() bool {
		txt, err := firstCommentText(frame)
		return err == nil && strings.Contains(txt, first)
	})
}

func TestThread_CollapsePersistsAcrossReload(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	parent := "collapse parent " + runID
	postComment(t, frame, parent)

	require.NoError(t, actions(frame, parent).Locator(`button:has-text("Reply")`).Click())
	reply := "collapse reply " + runID
	submitForm(t, replyForm(t, frame), reply)
	waitVisible(t, comment(frame, reply))

	// anchor on the comment's own id rather than its text: collapsing hides the text, which
	// would make a hasText filter stop matching the element under test. the id also excludes
	// the RSS dropdown, the other thing on the page carrying aria-expanded
	id, err := comment(frame, parent).GetAttribute("id")
	require.NoError(t, err)
	require.NotEmpty(t, id)
	threadSel := fmt.Sprintf("[aria-expanded]:has(article#%s)", id)

	thread := frame.Locator(threadSel)
	expanded, err := pollAttr(thread, "aria-expanded")
	require.NoError(t, err)
	require.Equal(t, "true", expanded)

	require.NoError(t, thread.Locator(`:scope > [role="button"]`).Click())

	eventually(t, waitTimeout, "thread did not collapse", func() bool {
		v, aerr := pollAttr(thread, "aria-expanded")
		return aerr == nil && v == "false"
	})
	// counting rather than filtering on the reply's text, which an off-screen comment would
	// satisfy just as well as a collapsed one
	eventually(t, waitTimeout, "the reply was still rendered after collapsing", func() bool {
		n, err := frame.Locator("article").Count()
		return err == nil && n == 1
	})

	// collapse is client-only state, kept in localStorage rather than on the server
	frame = reload(t, page)
	thread = frame.Locator(threadSel)
	eventually(t, waitTimeout, "collapse did not survive reload", func() bool {
		v, aerr := pollAttr(thread, "aria-expanded")
		return aerr == nil && v == "false"
	})

	assert.Equal(t, 1, articleCount(t, frame), "a collapsed thread should not render its replies after reload")
}

// TestThread_HideUserRemovesTheirCommentsOnly covers hiding an author, which is a reader-side
// setting rather than a moderator action: it patches every comment already on screen, persists in
// the browser rather than on the server, and has to be undone from the settings panel. The second
// author is what makes it a test of hiding one person rather than of emptying the thread
func TestThread_HideUserRemovesTheirCommentsOnly(t *testing.T) {
	hidden := "hidden author " + runID
	kept := "kept author " + runID

	author := newPage(t)
	authorFrame := openThread(t, author)
	signInAnon(t, authorFrame, "hidetarget")
	postComment(t, authorFrame, hidden)
	postComment(t, authorFrame, hidden+" second")

	other := newPage(t)
	otherFrame := openURL(t, other, threadURL(t))
	signInAnon(t, otherFrame, "hidekeeper")
	postComment(t, otherFrame, kept)

	reader := newPage(t)
	frame := openURL(t, reader, threadURL(t))
	signInDev(t, reader, frame)
	before := articleCount(t, frame)
	require.Equal(t, 3, before, "the three seeded comments have to be on screen before hiding one author")

	reader.OnDialog(func(d playwright.Dialog) { _ = d.Accept() })
	require.NoError(t, actions(frame, hidden).Locator(`button:has-text("Hide")`).Click())

	eventually(t, waitTimeout, "hiding the author did not remove both of their comments", func() bool {
		return articleCount(t, frame) == before-2
	})
	waitVisible(t, comment(frame, kept))

	frame = reload(t, reader)
	assert.Equal(t, before-2, articleCount(t, frame), "hiding has to survive a reload, it is stored in the browser")
	waitVisible(t, comment(frame, kept))

	require.NoError(t, frame.Locator(`button:has-text("Show settings")`).Click())
	// the restore control is a span rather than a button, and "show" appears in the settings
	// toggle as well, so match it exactly and only inside the hidden users region
	hiddenUsers := frame.Locator(`[role="region"][aria-label="Hidden users"]`)
	waitVisible(t, hiddenUsers)
	require.NoError(t, hiddenUsers.GetByText("show", playwright.LocatorGetByTextOptions{
		Exact: playwright.Bool(true),
	}).Click())

	frame = reload(t, reader)
	assert.Equal(t, before, articleCount(t, frame), "restoring the author has to bring their comments back")
}

// TestThread_LocaleLoadsItsTranslationChunk covers the locale bundles, which nothing else asks
// for. loadLocale imports the chunk dynamically and falls back to the English defaults on any
// error, so a missing or broken chunk renders a perfectly good English widget and says nothing.
// The widget document is opened directly, since the demo page has no way to pass a locale.
func TestThread_LocaleLoadsItsTranslationChunk(t *testing.T) {
	page := newPage(t)

	pauseForAuthLimit()
	// the widget document, not the demo page, since nothing there can pass a locale. it has to
	// be opened on the instance's own origin: the bundle addresses the host the build was
	// substituted with, and its CSP is `self`, so a mismatched origin blocks its own chunk
	_, err := page.Goto(fmt.Sprintf("%s/web/iframe.html?site_id=remark&locale=ru&url=%s",
		baseURL, neturl.QueryEscape(threadURL(t))))
	require.NoError(t, err)

	// two strings rather than one, and both from the chunk rather than from the code's own
	// defaults: the sort label is always present, and the sign-in button only while signed out
	waitVisible(t, page.Locator("text=Сортировать по"))
	waitVisible(t, page.Locator(`text=Войти`))
}
