//go:build e2e

package e2e

import (
	"regexp"
	"sync"
	"testing"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEmbed_PlaceholderGivesWayToOneOwnedIframe covers #2192 together with the placeholder
// support #2002 added. createInstance took root.firstElementChild as its iframe, so anything the
// integrator left in the root was adopted instead: a <noscript> fallback became "the iframe",
// createIframe never ran, and the height messages went to an element that cannot show comments.
// The same defect defeated the placeholder promise, since an element placeholder is mistaken for
// the iframe and inited never arrives to clear it.
//
// A text node and an element, because only the element was ever adopted and a test carrying text
// alone passes against the defect.
func TestEmbed_PlaceholderGivesWayToOneOwnedIframe(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	stubSignedOut(t, page)
	embedConfig(t, page, map[string]any{},
		`loading the comments`, `<div id="element-placeholder">a spinner of the integrator's own</div>`)

	widget(t, page)

	// the placeholder goes only when the document reports itself inited, so reaching this means
	// the message arrived from a real widget document and not from an adopted div
	waitHidden(t, page.Locator("#element-placeholder"),
		"the element placeholder survived, so the widget never reported itself inited")

	frames, err := page.Locator("#remark42 iframe").Count()
	require.NoError(t, err)
	assert.Equal(t, 1, frames, "the root should hold exactly the one iframe the embed created")

	owned, err := page.Locator("#remark42 iframe[data-remark42-iframe]").Count()
	require.NoError(t, err)
	assert.Equal(t, 1, owned, "the iframe is not marked as the embed's own, so the next "+
		"createInstance cannot tell it from whatever else is in the root")

	text, err := page.Locator("#remark42").InnerText()
	require.NoError(t, err)
	assert.NotContains(t, text, "loading the comments", "the text placeholder was never cleared")

	// a second instance has to reuse the marked iframe, not add another. this is what a
	// single-page app does on every navigation
	_, err = page.Evaluate(`() => window.REMARK42.createInstance(window.remark_config)`)
	require.NoError(t, err)

	frames, err = page.Locator("#remark42 iframe").Count()
	require.NoError(t, err)
	assert.Equal(t, 1, frames, "a second createInstance added an iframe instead of reusing the one there")
}

// TestEmbed_DestroyRemovesTheWidgetAndCreateInstanceBringsItBack covers the rest of the surface a
// single-page app holds, documented in configuration/frontend/spa.md and exercised by nothing.
// Neither half is observable from inside the widget, which is where the rest of this suite looks
func TestEmbed_DestroyRemovesTheWidgetAndCreateInstanceBringsItBack(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	stubSignedOut(t, page)
	embedConfig(t, page, map[string]any{})
	widget(t, page)

	// destroy takes the iframe out of the page, and the widget document goes with it
	snapshotJSCoverage(t, page)
	_, err := page.Evaluate(`() => window.REMARK42.destroy()`)
	require.NoError(t, err)

	waitHidden(t, page.Locator("#remark42 iframe"), "destroy left the iframe in the page")

	// and destroy drops the two functions it documents dropping, so a caller can tell a live
	// instance from a destroyed one
	live, err := page.Evaluate(`() => typeof window.REMARK42.destroy`)
	require.NoError(t, err)
	assert.Equal(t, "undefined", live, "destroy left its own handle behind")

	_, err = page.Evaluate(`() => window.REMARK42.createInstance(window.remark_config)`)
	require.NoError(t, err)
	widget(t, page)
}

// TestEmbed_ThemeChangesReachTheWidgetAfterLoad covers the runtime half of the color-scheme
// contract. The iframe cases assert what a load with a theme parameter produces; this is the
// path the demo page's own toggle takes, and the one an integrator calls when a reader switches
// themes on the host page
func TestEmbed_ThemeChangesReachTheWidgetAfterLoad(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	stubSignedOut(t, page)
	embedConfig(t, page, map[string]any{"theme": "light"})
	widget(t, page)

	require.Equal(t, "light", widgetColorScheme(t, page), "the widget did not start in the theme it was given")

	_, err := page.Evaluate(`() => window.REMARK42.changeTheme('dark')`)
	require.NoError(t, err)

	// both sides, since it is the disagreement between them that paints the opaque canvas the
	// iframe cases are about
	eventually(t, waitTimeout, "the widget document never took the new theme", func() bool {
		return widgetColorScheme(t, page) == "dark"
	})

	onElement, err := page.Evaluate(`() => document.querySelector('#remark42 iframe').style.colorScheme`)
	require.NoError(t, err)
	assert.Equal(t, "dark", onElement, "the parent left the old scheme on the iframe element")
}

// TestEmbed_ThemeChangedWhileLoadingReachesTheWidget covers a changeTheme made before the widget
// can hear it: while its document is still loading, and after the document has reported inited
// but before the bundle has run. Neither has a listener for the theme, so the change used to be
// lost and the widget stayed in the theme its URL carried. The request is held until the change
// is made, so the case does not depend on how fast the stack answers
func TestEmbed_ThemeChangedWhileLoadingReachesTheWidget(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		hold *regexp.Regexp
		// whether to wait for the iframe to be revealed, which happens on inited, before the change
		afterInited bool
	}{
		{"before the document loads", regexp.MustCompile(`/web/iframe\.html`), false},
		{"before the bundle runs", regexp.MustCompile(`remark\.m?js$`), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page := newPage(t)
			stubSignedOut(t, page)

			release := make(chan struct{})
			var releaseOnce sync.Once
			releaseAll := func() { releaseOnce.Do(func() { close(release) }) }
			t.Cleanup(releaseAll)
			require.NoError(t, page.Route(tc.hold, func(route playwright.Route) {
				// off the handler, so a held request does not hold up the others
				go func() {
					<-release
					// fails only once the page is gone, which the assertions report on their own
					_ = route.Continue()
				}()
			}))

			embedConfig(t, page, map[string]any{"theme": "light"})

			if tc.afterInited {
				eventually(t, waitTimeout, "the iframe was never revealed, so inited never arrived", func() bool {
					v, err := page.Evaluate(`() => document.querySelector('#remark42 iframe')?.style.visibility`)
					return err == nil && v == "visible"
				})
			}

			_, err := page.Evaluate(`() => window.REMARK42.changeTheme('dark')`)
			require.NoError(t, err)
			releaseAll()

			widget(t, page)

			eventually(t, waitTimeout, "the widget document never took the theme changed while it loaded", func() bool {
				return widgetColorScheme(t, page) == "dark"
			})

			// the body class drives the stylesheets and the bare class on Root comes from the store,
			// which take the theme by different paths
			frame := page.FrameLocator("#remark42 iframe")
			isDark, err := frame.Locator("body").Evaluate(`(el) => el.classList.contains('dark')`, nil)
			require.NoError(t, err)
			assert.Equal(t, true, isDark, "the widget body was left without the dark class")

			count, err := frame.Locator("#remark42 .dark").Count()
			require.NoError(t, err)
			assert.Positive(t, count, "Root rendered from the theme in the URL")
		})
	}
}

// widgetColorScheme is the color-scheme in force inside the widget document
func widgetColorScheme(t *testing.T, page playwright.Page) string {
	t.Helper()

	v, err := page.FrameLocator("#remark42 iframe").Locator(":root").Evaluate(
		`(el) => getComputedStyle(el).colorScheme`, nil,
		playwright.LocatorEvaluateOptions{Timeout: playwright.Float(float64(pollTimeout.Milliseconds()))})
	if err != nil {
		return ""
	}
	scheme, _ := v.(string)
	return scheme
}
