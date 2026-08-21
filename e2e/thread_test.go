//go:build e2e

package e2e

import (
	"fmt"
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
