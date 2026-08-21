//go:build e2e

package e2e

import (
	"regexp"
	"testing"
	"time"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The parent page sets color-scheme on the iframe element from the theme param. If the iframe
// document does not carry the same color-scheme before its bundle runs, the canvas is painted
// opaque white instead of staying transparent. Blocking the bundle freezes the document in
// that pre-script state, so these assert the inline head script has already applied the scheme.
func TestIframe_ColorSchemeIsSetBeforeTheBundleRuns(t *testing.T) {
	cases := []struct {
		name     string
		query    string
		expected string
	}{
		{"dark theme", "?site_id=remark&theme=dark", "dark"},
		{"light theme", "?site_id=remark&theme=light", "light"},
		{"no theme falls back to light", "?site_id=remark", "light"},
	}

	for _, engine := range engines() {
		for _, tc := range cases {
			t.Run(engine+"/"+tc.name, func(t *testing.T) {
				page := newPageOn(t, browserFor(t, engine))
				require.NoError(t, page.Route(regexp.MustCompile(`remark\.m?js$`), func(route playwright.Route) {
					_ = route.Abort()
				}))

				pauseForAuthLimit()
				_, err := page.Goto(probeURL + "/web/iframe.html" + tc.query)
				require.NoError(t, err)

				inline, err := page.Evaluate("() => document.documentElement.style.colorScheme")
				require.NoError(t, err)
				assert.Equal(t, tc.expected, inline)

				computed, err := page.Evaluate("() => getComputedStyle(document.documentElement).colorScheme")
				require.NoError(t, err)
				assert.Equal(t, tc.expected, computed)
			})
		}
	}
}

// TestIframe_ParentAndDocumentAgreeOnColorScheme covers the other half of the same defect.
// The tests above load the widget document on its own, so removing the color-scheme the
// parent puts on the iframe element would not disturb them, and it is the disagreement
// between the two that paints the opaque canvas.
func TestIframe_ParentAndDocumentAgreeOnColorScheme(t *testing.T) {
	schemes := map[string]*playwright.ColorScheme{
		"dark":  playwright.ColorSchemeDark,
		"light": playwright.ColorSchemeLight,
	}
	for _, engine := range engines() {
		for theme, scheme := range schemes {
			t.Run(engine+"/"+theme, func(t *testing.T) {
				page := newPageOn(t, browserFor(t, engine))
				// the demo page reads prefers-color-scheme rather than a query parameter
				require.NoError(t, page.EmulateMedia(playwright.PageEmulateMediaOptions{ColorScheme: scheme}))

				pauseForAuthLimit()
				_, err := page.Goto(renderURL(t))
				require.NoError(t, err)
				widget(t, page)

				onElement, err := page.Evaluate(`() => document.querySelector('#remark42 iframe').style.colorScheme`)
				require.NoError(t, err)
				assert.Equal(t, theme, onElement, "the parent has to mark the iframe element")

				inDocument, err := page.FrameLocator("#remark42 iframe").Locator(":root").Evaluate(
					`(el) => getComputedStyle(el).colorScheme`, nil)
				require.NoError(t, err)
				assert.Equal(t, theme, inDocument, "and the document inside it has to agree")
			})
		}
	}
}

// Browsers paint a default surface for an iframe before its document is parsed, and that
// surface is opaque when the element carries a color-scheme the document does not have yet.
// WebKit shows it as a white flash on dark host pages. The parent keeps the iframe hidden
// until the document reports itself inited, so the surface is never presented.
const (
	// REVEAL_TIMEOUT in app/utils/create-iframe.ts. the fallback timer starts when the iframe
	// is created, during page load, so any assertion with a deadline at or past this value can
	// be satisfied by the fallback alone and says nothing about the message path. bound the
	// message-path assertions well under it
	revealTimeout = 5 * time.Second
	// generous enough for a cold navigation on a loaded runner, and still well under the
	// fallback, which is the point of the assertion
	messageRevealBudget = 3 * time.Second
)

// openWithBlockedIframeDoc loads the demo page with the widget document aborted, so the
// iframe element exists but never reports itself inited
func openWithBlockedIframeDoc(t *testing.T, page playwright.Page) time.Time {
	t.Helper()
	require.NoError(t, page.Route(regexp.MustCompile(`/web/iframe\.html`), func(route playwright.Route) {
		_ = route.Abort()
	}))

	// the gate's sleep happens before the iframe exists, so it must not count against the
	// reveal budgets: start the clock with the navigation
	pauseForAuthLimit()
	start := time.Now()
	_, err := page.Goto(renderURL(t))
	require.NoError(t, err)
	require.NoError(t, page.Locator("#remark42 iframe").WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateAttached,
		Timeout: playwright.Float(float64(waitTimeout.Milliseconds())),
	}))
	return start
}

func iframeVisibility(t *testing.T, page playwright.Page) string {
	t.Helper()
	v, err := page.Evaluate(`() => {
		const iframe = document.querySelector('#remark42 iframe');
		return iframe ? iframe.style.visibility : 'no-iframe';
	}`)
	require.NoError(t, err)
	s, _ := v.(string)
	return s
}

func TestIframe_StaysHiddenUntilTheDocumentReportsInited(t *testing.T) {
	forEachEngine(t, func(t *testing.T, page playwright.Page) {
		start := openWithBlockedIframeDoc(t, page)

		// sampling once would pass against a widget that revealed a frame moments later, so
		// hold the assertion for a stretch of the window in which it must stay hidden
		for time.Since(start) < revealTimeout/2 {
			require.Equal(t, "hidden", iframeVisibility(t, page))
			time.Sleep(100 * time.Millisecond)
		}
		// a slow run could have let the fallback fire, which would make the assertion above
		// pass or fail for the wrong reason. fail loudly instead of flaking
		assert.Less(t, time.Since(start), revealTimeout)
	})
}

// forEachEngine runs body once per configured browser, on a fresh page each time
func forEachEngine(t *testing.T, body func(t *testing.T, page playwright.Page)) {
	t.Helper()
	for _, engine := range engines() {
		t.Run(engine, func(t *testing.T) {
			body(t, newPageOn(t, browserFor(t, engine)))
		})
	}
}

// The reveal has to come from the inited message, not the fallback: a broken message listener
// would leave the widget invisible for five seconds on every load. The fallback timer starts
// when the iframe is created, partway through the navigation, so bounding only the poll leaves
// the navigation window unmeasured. Time the whole thing.
func TestIframe_IsRevealedByTheInitedMessage(t *testing.T) {
	forEachEngine(t, func(t *testing.T, page playwright.Page) {
		pauseForAuthLimit()
		start := time.Now()
		_, err := page.Goto(renderURL(t))
		require.NoError(t, err)

		eventually(t, messageRevealBudget, "iframe was not revealed by the inited message", func() bool {
			return iframeVisibility(t, page) == "visible"
		})
		assert.Less(t, time.Since(start), revealTimeout)
		waitVisible(t, page.Locator("#remark42 iframe"))
	})
}

// The aborted document never reports its height, so the iframe box stays empty and a
// visibility assertion on geometry would fail. Assert the property the fallback actually sets.
func TestIframe_IsRevealedByTheTimeoutWhenInitedNeverArrives(t *testing.T) {
	forEachEngine(t, func(t *testing.T, page playwright.Page) {
		start := openWithBlockedIframeDoc(t, page)

		// and not before it: without a lower bound, shortening the fallback to a value that
		// defeats its purpose would still pass
		eventually(t, revealTimeout*2, "fallback never revealed the iframe", func() bool {
			return iframeVisibility(t, page) == "visible"
		})
		assert.Greater(t, time.Since(start), revealTimeout*3/4,
			"the reveal came too early to have been the fallback timer")
	})
}
