//go:build e2e

package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCrossOrigin_WidgetRendersOnAnotherOrigin covers the separate-domain setup the manuals
// describe, which is the configuration readers actually hit problems with and which nothing else
// here reaches: every other host page in this suite is served by remark42 itself, so the widget
// and its embedder always share an origin and the cross-site path is never taken.
//
// The host page is served by its own nginx on a different name and port, and the widget it embeds
// addresses remark42 by remark_config.host. What has to hold is that the frame is revealed at all,
// which means its document loaded and reported itself inited through postMessage across origins,
// and that the thread it renders is the one the page's own address names.
//
// Signing in is deliberately not asserted here. An embedded cookie needs SameSite=None, which
// browsers only accept as Secure, and this page is served over http, so the form cannot be
// delivered at all. That half is covered over TLS in https_test.go, where the assertion that
// matters is the reload: the widget holds its token in memory for the life of a page, so a
// sign-in that never reloads passes while persistence is broken
func TestCrossOrigin_WidgetRendersOnAnotherOrigin(t *testing.T) {
	t.Parallel()

	thread := fmt.Sprintf("%s/post.html?e2e=%s-%s", hostSiteURL, "crossorigin", runID)
	text := "cross origin " + runID

	// seeded from a page on remark42's own origin, since posting needs a session and this case is
	// about rendering, not about carrying a cookie across origins
	seeder := newPage(t)
	seederFrame := openThread(t, seeder)
	signInAnon(t, seeder, seederFrame, anonName("crossorigin"))
	status, body := pageFetch(t, seeder, "POST", baseURL+"/api/v1/comment?site=remark", map[string]any{
		"text":    text,
		"locator": map[string]string{"site": "remark", "url": thread},
	})
	require.Equal(t, 201, status, "could not seed the cross-origin thread: %s", body)

	page := newPage(t)
	stubSignedOut(t, page)
	_, err := page.Goto(thread)
	require.NoError(t, err)

	frame := widget(t, page)

	// revealed, so the document loaded and its inited message crossed the origin boundary. a
	// frame that never reported would still be here, hidden, until the fallback timer
	waitVisible(t, page.Locator("#remark42 iframe"))
	waitVisible(t, comment(frame, text))
}

// TestCrossOrigin_DisallowedHostNeverReportsInited is the other half of ALLOWED_HOSTS. An
// operator sets it so their comments cannot be framed by anyone else, and what the reader on such
// a page gets is decided entirely by the widget's own fallback: the browser refuses to load the
// document, nothing ever posts inited, and the frame is revealed five seconds later by the timer.
//
// Worth pinning in both directions. Without the fallback an integrator who mistyped the host
// would be left with a permanently invisible widget and nothing in the page to say why, and
// without the refusal ALLOWED_HOSTS would be doing nothing at all
func TestCrossOrigin_DisallowedHostNeverReportsInited(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	stubSignedOut(t, page)

	_, err := page.Goto(fmt.Sprintf("%s/restricted.html?e2e=%s-%s", hostSiteURL, "crossorigin-blocked", runID))
	require.NoError(t, err)

	require.NoError(t, page.Locator("#remark42 iframe").WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateAttached,
		Timeout: playwright.Float(float64(waitTimeout.Milliseconds())),
	}))

	// hidden for as long as the fallback runs, since the document the browser refused cannot
	// report anything
	require.Equal(t, "hidden", iframeVisibility(t, page))

	eventually(t, revealTimeout*2, "the frame was never revealed, so a mistyped host would stay invisible", func() bool {
		_, ok := revealDelay(t, page)
		return ok
	})

	delay, ok := revealDelay(t, page)
	require.True(t, ok)
	assert.Greater(t, delay, revealTimeout-500*time.Millisecond,
		"the frame was revealed too early to have been the fallback, so something reported inited "+
			"from a document the browser should not have loaded")

	// and the widget really is not there, which is what makes the reveal above the fallback's
	forms, err := page.FrameLocator("#remark42 iframe").Locator(commentFormSel).Count()
	require.NoError(t, err)
	assert.Zero(t, forms, "the widget rendered on a host the instance does not allow")
}
