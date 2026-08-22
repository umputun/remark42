//go:build e2e

package e2e

import (
	"math"
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
func openWithBlockedIframeDoc(t *testing.T, page playwright.Page) {
	t.Helper()
	require.NoError(t, page.Route(regexp.MustCompile(`/web/iframe\.html`), func(route playwright.Route) {
		_ = route.Abort()
	}))

	pauseForAuthLimit()
	_, err := page.Goto(renderURL(t))
	require.NoError(t, err)
	require.NoError(t, page.Locator("#remark42 iframe").WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateAttached,
		Timeout: playwright.Float(float64(waitTimeout.Milliseconds())),
	}))
}

// iframeMarkScript records, in the page, when the widget's iframe element is created and when
// it is first revealed. create-iframe.ts starts its fallback timer at creation, and neither
// moment can be timed from outside the page: navigation alone can outlast the budgets, which
// would let a bound pass without the assertion it guards ever running
const iframeMarkScript = `(() => {
  // top document only. playwright runs an init script in every frame, and inside the widget
  // document the selector below never matches, so the observer would run for the life of the
  // busiest DOM in the suite without ever finding a reason to disconnect
  if (window.top !== window) { return; }
  if (window.__r42Marks) { return; }
  window.__r42Marks = {};
  const watch = (frame) => {
    if (window.__r42Marks.created !== undefined) { return; }
    window.__r42Marks.created = performance.now();
    const style = new MutationObserver(() => { seen(); });
    const seen = () => {
      if (window.__r42Marks.revealed === undefined && frame.style.visibility === 'visible') {
        window.__r42Marks.revealed = performance.now();
        style.disconnect();
      }
    };
    style.observe(frame, {attributes: true, attributeFilter: ['style']});
    seen();
  };
  // document rather than documentElement: an init script runs before the root element
  // exists, and observing null would throw before any of this could take effect
  const tree = new MutationObserver(() => { scan(); });
  const scan = () => {
    const frame = document.querySelector('#remark42 iframe');
    if (frame) { watch(frame); tree.disconnect(); }
  };
  tree.observe(document, {childList: true, subtree: true});
  document.addEventListener('DOMContentLoaded', scan);
  // and stop looking once the page has loaded. embed.ts creates the frame no later than
  // DOMContentLoaded, so a page still without one never will have one, and the suite opens
  // several: the widget document itself, the counter page and the last-comments page, whose
  // own rendering would otherwise keep this observer busy for the life of the page
  window.addEventListener('load', () => { tree.disconnect(); });
  scan();
})()`

// evalMillis reads a number out of the page. it takes int as well as float64, because the
// driver hands back whichever the value happens to be and a bare float64 assertion turns an
// integral sentinel into a silent zero
func evalMillis(t *testing.T, page playwright.Page, script string) float64 {
	t.Helper()
	v, err := page.Evaluate(script)
	require.NoError(t, err)

	switch n := v.(type) {
	case float64:
		require.False(t, math.IsNaN(n) || math.IsInf(n, 0), "got a non-finite number from the page")
		return n
	case int:
		return float64(n)
	default:
		t.Fatalf("expected a number from the page, got %T (%v)", v, v)
		return 0
	}
}

// iframeAge is how long the iframe element has existed, measured in the page.
//
// the mark is taken when the element is inserted, while create-iframe.ts starts its fallback
// a moment earlier, when the detached element is built. the gap is one task, so every age
// here reads slightly short: bounds below a budget are conservative, bounds above it are not
func iframeAge(t *testing.T, page playwright.Page) time.Duration {
	t.Helper()
	ms := evalMillis(t, page, `() => window.__r42Marks && window.__r42Marks.created !== undefined
		? performance.now() - window.__r42Marks.created : -1`)
	require.GreaterOrEqual(t, ms, float64(0), "the iframe element has not been created yet")
	return time.Duration(ms) * time.Millisecond
}

// revealDelay is how long after creation the iframe was revealed, measured in the page. it
// reports false while the frame is still hidden
func revealDelay(t *testing.T, page playwright.Page) (time.Duration, bool) {
	t.Helper()
	ms := evalMillis(t, page, `() => {
		const m = window.__r42Marks;
		if (!m || m.created === undefined) { return -2; }
		return m.revealed !== undefined ? m.revealed - m.created : -1;
	}`)
	// -2 is the harness, -1 is the widget. without the distinction a broken selector reads as
	// "the frame was never revealed" and the failure names the wrong thing
	require.NotEqual(t, float64(-2), ms, "the iframe element was never seen by the page marks")
	if ms < 0 {
		return 0, false
	}
	return time.Duration(ms) * time.Millisecond, true
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
		openWithBlockedIframeDoc(t, page)

		// unconditional: the loop below is bounded by the frame's own age, and on a slow
		// enough load that bound can already be spent, which would leave the test asserting
		// nothing at all about visibility
		require.Equal(t, "hidden", iframeVisibility(t, page))

		// then hold it for almost the whole fallback window. stopping halfway would only prove
		// the fallback is not shorter than that, and a widget that revealed on anything other
		// than `inited` would still pass. measured from the element's creation, since that is
		// when the fallback it must not have used starts counting
		for iframeAge(t, page) < revealTimeout-500*time.Millisecond {
			require.Equal(t, "hidden", iframeVisibility(t, page))
			time.Sleep(100 * time.Millisecond)
		}
		_, revealed := revealDelay(t, page)
		assert.False(t, revealed, "the frame was revealed before its document reported inited")
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
		_, err := page.Goto(renderURL(t))
		require.NoError(t, err)

		eventually(t, waitTimeout, "iframe was never revealed", func() bool {
			_, ok := revealDelay(t, page)
			return ok
		})

		// the reveal has to have come from the message rather than the fallback, and the two
		// are only distinguishable against the frame's own clock: navigation can outlast the
		// whole 5s window without the widget being at fault
		delay, ok := revealDelay(t, page)
		require.True(t, ok, "the frame reported no reveal at all")
		assert.Less(t, delay, messageRevealBudget,
			"the reveal was slow enough to have come from the fallback rather than the message")
		waitVisible(t, page.Locator("#remark42 iframe"))
	})
}

// The aborted document never reports its height, so the iframe box stays empty and a
// visibility assertion on geometry would fail. Assert the property the fallback actually sets.
func TestIframe_IsRevealedByTheTimeoutWhenInitedNeverArrives(t *testing.T) {
	forEachEngine(t, func(t *testing.T, page playwright.Page) {
		openWithBlockedIframeDoc(t, page)

		eventually(t, revealTimeout*2, "fallback never revealed the iframe", func() bool {
			_, ok := revealDelay(t, page)
			return ok
		})

		// and not before it: without a lower bound, shortening the fallback to a value that
		// defeats its purpose would still pass. against the frame's own clock, so that a slow
		// navigation cannot be mistaken for the timer having run
		// close to the fallback rather than three quarters of it: measured in the page there is
		// no navigation to make room for, and a wider floor tolerates a fallback shortened
		// enough to defeat its purpose
		delay, ok := revealDelay(t, page)
		require.True(t, ok, "the frame reported no reveal at all")
		assert.Greater(t, delay, revealTimeout-500*time.Millisecond,
			"the reveal came too early to have been the fallback timer")
	})
}
