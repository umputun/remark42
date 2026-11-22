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
// error instead of failing, because every caller polls it while the list re-renders and a
// momentarily empty list has to be retried, not fail the test
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
	t.Parallel()

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
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	parent := "collapse parent " + runID
	postComment(t, frame, parent)

	require.NoError(t, actions(frame, parent).Locator(`button:has-text("Reply")`).Click())
	reply := "collapse reply " + runID
	submitForm(t, replyForm(t, frame), reply)
	waitVisible(t, comment(frame, reply))

	// anchor on the comment's own id, not its text: collapsing hides the text, which
	// would make a hasText filter stop matching the element under test. the control that
	// collapses the thread is a direct child of its list item and carries aria-expanded
	id, err := comment(frame, parent).GetAttribute("id")
	require.NoError(t, err)
	require.NotEmpty(t, id)
	controlSel := fmt.Sprintf(`[role="listitem"]:has(> article#%s) > [role="button"]`, id)

	control := frame.Locator(controlSel)
	expanded, err := pollAttr(control, "aria-expanded")
	require.NoError(t, err)
	require.Equal(t, "true", expanded)

	require.NoError(t, control.Click())

	eventually(t, waitTimeout, "thread did not collapse", func() bool {
		v, aerr := pollAttr(control, "aria-expanded")
		return aerr == nil && v == "false"
	})
	// counting instead of filtering on the reply's text, which an off-screen comment would
	// satisfy just as well as a collapsed one
	eventually(t, waitTimeout, "the reply was still rendered after collapsing", func() bool {
		n, err := frame.Locator("article").Count()
		return err == nil && n == 1
	})

	// collapse is client-only state, kept in localStorage and not on the server
	frame = reload(t, page)
	control = frame.Locator(controlSel)
	eventually(t, waitTimeout, "collapse did not survive reload", func() bool {
		v, aerr := pollAttr(control, "aria-expanded")
		return aerr == nil && v == "false"
	})

	assert.Equal(t, 1, articleCount(t, frame), "a collapsed thread should not render its replies after reload")
}

// TestThread_HideUserRemovesTheirCommentsOnly covers hiding an author, which is a reader-side
// setting and not a moderator action: it patches every comment already on screen, persists in the
// browser and not on the server, and has to be undone from the settings panel. The second author
// is what makes it a test of hiding one person and not of emptying the thread
func TestThread_HideUserRemovesTheirCommentsOnly(t *testing.T) {
	t.Parallel()

	hidden := "hidden author " + runID
	kept := "kept author " + runID

	author := newPage(t)
	authorFrame := openThread(t, author)
	signInAnon(t, author, authorFrame, "hidetarget")
	postComment(t, authorFrame, hidden)
	postComment(t, authorFrame, hidden+" second")

	other := newPage(t)
	otherFrame := openURL(t, other, threadURL(t))
	signInAnon(t, other, otherFrame, "hidekeeper")
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
	// the restore control is a span and not a button, and "show" appears in the settings
	// toggle as well, so match it exactly and only inside the hidden users region
	hiddenUsers := frame.Locator(`[role="region"][aria-label="Hidden users"]`)
	waitVisible(t, hiddenUsers)
	require.NoError(t, hiddenUsers.GetByText("show", playwright.LocatorGetByTextOptions{
		Exact: new(true),
	}).Click())

	frame = reload(t, reader)
	assert.Equal(t, before, articleCount(t, frame), "restoring the author has to bring their comments back")
}

// TestThread_WidgetDocumentServesItsOwnChunks covers the widget document loaded directly on the
// instance's origin, which is how a reader's browser loads it and how nothing else here does:
// TestWidgets_EveryLocaleLoadsAndRenders injects a config of its own into a plain page, so it
// exercises the translations and not the delivery.
//
// The distinction is the point. The bundle addresses the host the build was substituted with and
// its CSP is `self`, so a mismatch between the two blocks the widget's own chunks with nothing in
// the page to say why. A locale is the payload because it is the one thing loaded as a separate
// chunk after boot, so it fails when the origin is wrong and renders English instead of throwing.
func TestThread_WidgetDocumentServesItsOwnChunks(t *testing.T) {
	t.Parallel()

	page := newPage(t)

	// the widget document, not the demo page, since nothing there can pass a locale. it has to
	// be opened on the instance's own origin: the bundle addresses the host the build was
	// substituted with, and its CSP is `self`, so a mismatched origin blocks its own chunk
	_, err := page.Goto(fmt.Sprintf("%s/web/iframe.html?site_id=remark&locale=ru&url=%s",
		baseURL, neturl.QueryEscape(threadURL(t))))
	require.NoError(t, err)

	// two strings, not one, and both from the chunk and not from the code's own
	// defaults: the sort label is always present, and the sign-in button only while signed out
	waitVisible(t, page.Locator("text=Сортировать по"))
	waitVisible(t, page.Locator(`text=Войти`))
}

// TestThread_LoadsInPages covers comments_page_size and the paging of /find behind it (#782).
// The thread gets five top-level comments, one with a reply. A page holds whole threads and stops
// before one that would overflow it, so with the size at two and newest first the pages are
// paged 4; paged 3 with its reply; paged 2 and 1; paged 0. Each Show more brings the next one.
// A page with no setting is the control: the default page of 100 takes the whole thread and
// offers no button. A link to the oldest comment, which is on the last page,
// has to bring it on screen in one go
func TestThread_LoadsInPages(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	embedConfig(t, page, map[string]any{})
	frame := widget(t, page)
	signInAnon(t, page, frame, anonName("pages"))
	text := func(i int) string { return fmt.Sprintf("paged %d %s", i, runID) }
	for i := range 5 {
		postComment(t, frame, text(i))
	}
	// a reply makes its thread two comments, a whole page
	require.NoError(t, actions(frame, text(3)).GetByText("Reply").Click())
	submitForm(t, replyForm(t, frame), "reply to paged 3 "+runID)
	waitVisible(t, comment(frame, "reply to paged 3 "+runID))
	showMore := func(f playwright.FrameLocator) playwright.Locator {
		return f.GetByRole(*playwright.AriaRoleButton, playwright.FrameLocatorGetByRoleOptions{Name: "Show more"})
	}
	// the control: a reader on a page with no setting gets the whole thread and no button
	whole := newPage(t)
	stubSignedOut(t, whole)
	embedConfig(t, whole, map[string]any{})
	wholeFrame := widget(t, whole)
	eventually(t, waitTimeout, "the default page size did not take the whole thread", func() bool {
		n, err := wholeFrame.Locator("article").Count()
		return err == nil && n == 6
	})
	waitHidden(t, showMore(wholeFrame), "Show more was offered for a thread loaded whole")

	paged := newPage(t)
	stubSignedOut(t, paged)
	embedConfig(t, paged, map[string]any{"comments_page_size": 2, "default_sort": sortNewestFirst})
	pagedFrame := widget(t, paged)
	more := showMore(pagedFrame)
	waitVisible(t, more)
	assert.Equal(t, 1, articleCount(t, pagedFrame), "comments_page_size=2 did not limit the first page")

	for _, want := range []int{3, 5, 6} {
		require.NoError(t, more.Click())
		eventually(t, waitTimeout, fmt.Sprintf("Show more did not bring the thread to %d comments", want), func() bool {
			n, err := pagedFrame.Locator("article").Count()
			return err == nil && n == want
		})
	}
	waitHidden(t, more, "Show more stayed with the whole thread loaded")

	// the oldest comment's id, for a link to it
	oldest, err := comment(pagedFrame, text(0)).GetAttribute("id")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(oldest, "remark42__comment-"), "unexpected comment id %q", oldest)

	linked := newPage(t)
	stubSignedOut(t, linked)
	embedConfigOn(t, linked, "/web/privacy.html#"+oldest, map[string]any{"comments_page_size": 2, "default_sort": sortNewestFirst})
	linkedFrame := widget(t, linked)
	waitVisible(t, comment(linkedFrame, text(0)))
	waitHidden(t, showMore(linkedFrame), "the link loaded the rest of the thread and still offered more")
}
