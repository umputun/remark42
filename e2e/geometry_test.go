//go:build e2e

package e2e

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The widget lives in an iframe the host page owns, so its height is not something CSS can
// settle: the document measures itself and posts the number, and the parent applies it. Every
// way that has broken is invisible to the rest of this suite, which asserts on elements the
// browser lays out inside a frame of whatever size, and reads the same whether the frame is
// right, twice too tall or collapsed to a strip.
const (
	// the collapse in #2126 reported 63px while the preloader was up. anything under this is
	// not a rendered widget
	preloaderCeiling = 150.0

	// layout lands on fractional pixels, and the parent rounds through a css string
	heightTolerance = 2.0

	// enough that a change is the widget resizing, not a font or a scrollbar settling
	growthFloor = 50.0
)

// TestGeometry_FirstReportedHeightIsRenderedContent covers #2126, where the iframe was told the
// preloader's height and shrank to a strip before growing back as content rendered. On pages
// with many comments that reads as the widget blinking several times on every load.
//
// An empty thread, not a populated one: the widget reports once, so a first report describing the
// preloader is the whole difference between the working and the broken version, and not one step
// in a sequence that legitimately grows as comments arrive.
func TestGeometry_FirstReportedHeightIsRenderedContent(t *testing.T) {
	page := newPage(t)
	stubSignedOut(t, page)
	embedConfig(t, page, map[string]any{})
	widget(t, page)
	waitHeightSettled(t, page)

	heights := heightReports(t, page)
	require.NotEmpty(t, heights, "the widget never told the parent how tall it is")

	assert.Greater(t, heights[0], preloaderCeiling,
		"the first height handed to the parent was %v, which is preloader-sized: the iframe "+
			"collapses on load and grows back. full sequence %v", heights[0], heights)
	assert.Equal(t, slices.Min(heights), heights[0],
		"the iframe was told to shrink below the height it started at, which is the blink itself. "+
			"full sequence %v", heights)
}

// TestGeometry_ReportedHeightMatchesTheDocument covers #2151: the body carried 6px of padding
// and the reported height added 12 on top of an offsetHeight that already included it, so every
// embed sat inset with 24px of empty space underneath. Both halves are asserted, since the
// arithmetic and the padding failed independently
func TestGeometry_ReportedHeightMatchesTheDocument(t *testing.T) {
	page := newPage(t)
	stubSignedOut(t, page)
	embedConfig(t, page, map[string]any{})
	widget(t, page)
	waitHeightSettled(t, page)

	document := inWidget(t, page, `() => document.body.offsetHeight`)
	require.Greater(t, document, preloaderCeiling, "the widget document never rendered")

	assert.InDelta(t, document, frameHeight(t, page), heightTolerance,
		"the parent sized the iframe at %v for a document of %v, so the difference is empty space",
		frameHeight(t, page), document)

	padding := inWidget(t, page, `() => ['Top', 'Right', 'Bottom', 'Left']
		.reduce((total, side) => total + parseFloat(getComputedStyle(document.body)['padding' + side]), 0)`)
	assert.Zero(t, padding,
		"the widget body carries %vpx of padding, so the widget cannot sit flush with the host layout", padding)
}

// TestGeometry_NoFooterKeepsTheContentInsideTheFrame covers #2073, where the negative margin on
// the last thread had no footer margin to collapse against and propagated up instead, leaving
// the document shorter than what it draws. The frame was then sized below the visible bottom of
// the last comment.
//
// Both modes run, because the footer-shown case is the control: it is what says the parameter
// reached the widget at all instead of being quietly ignored.
func TestGeometry_NoFooterKeepsTheContentInsideTheFrame(t *testing.T) {
	thread := threadURL(t)

	poster := newPage(t)
	posted := openURL(t, poster, thread)
	signInAnon(t, poster, posted, "geometrytester")
	postComment(t, posted, "geometry "+runID)

	for _, tc := range []struct {
		name     string
		noFooter bool
	}{
		{"footer shown", false},
		{"no footer", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := newPage(t)
			stubSignedOut(t, page)

			config := map[string]any{"url": thread}
			if tc.noFooter {
				config["no_footer"] = true
			}
			embedConfig(t, page, config)
			frame := widget(t, page)
			waitVisible(t, frame.Locator("article").First())
			waitHeightSettled(t, page)

			// by role: the production build hashes every css-module class name to a short opaque
			// id, so the footer's own class is not a selector any test can hold
			footers, err := frame.Locator(`[role="contentinfo"]`).Count()
			require.NoError(t, err)
			if tc.noFooter {
				require.Zero(t, footers, "no_footer did not reach the widget, so nothing below is about it")
			} else {
				require.NotZero(t, footers, "the footer is missing without no_footer, so the control proves nothing")
			}

			// the bottom of the last comment against the frame it has to fit inside. measured
			// from the document, not from the reported height, which is the number under test and
			// no witness to itself
			bottom := inWidget(t, page, `() => {
				const articles = document.querySelectorAll('article');
				const last = articles[articles.length - 1];
				return last ? last.getBoundingClientRect().bottom + window.scrollY : -1;
			}`)
			require.Positive(t, bottom, "the thread rendered no comment to measure")

			height := frameHeight(t, page)
			assert.LessOrEqual(t, bottom, height+heightTolerance,
				"the last comment ends at %vpx in a frame of %vpx, so it is clipped", bottom, height)
			assert.InDelta(t, inWidget(t, page, `() => document.body.offsetHeight`), height, heightTolerance,
				"the frame and the document it holds disagree on the height")
		})
	}
}

// TestGeometry_HeightFollowsTheAuthPanelAndTheTextarea covers the resize path itself, reverted
// once in #1495 and repaired repeatedly since. The dropdown is positioned absolutely, so the
// document does not grow with it and the widget has to measure it separately; the textarea grows
// the document directly. Both have to come back down again, or the widget leaves a hole in the
// page for as long as the reader stays on it
func TestGeometry_HeightFollowsTheAuthPanelAndTheTextarea(t *testing.T) {
	page := newPage(t)
	stubSignedOut(t, page)
	embedConfig(t, page, map[string]any{})
	frame := widget(t, page)
	waitHeightSettled(t, page)

	baseline := frameHeight(t, page)
	require.Greater(t, baseline, preloaderCeiling, "the widget never rendered, so there is no baseline")

	require.NoError(t, frame.Locator(".auth-button").Click())
	waitVisible(t, frame.Locator(".auth-dropdown"))
	waitHeightAbove(t, page, baseline+growthFloor, "opening the sign-in dropdown did not grow the iframe")

	// a click on the host page, not in the widget: the parent forwards it as
	// clickOutside, which is what closes the dropdown
	require.NoError(t, page.Mouse().Click(5, 5))
	waitHidden(t, frame.Locator(".auth-dropdown"), "the click on the host page did not close the dropdown")
	waitHeightNear(t, page, baseline, "the iframe kept the dropdown's height after it closed")

	textarea := frame.Locator(commentFormSel).First().Locator("textarea")
	require.NoError(t, textarea.Fill(strings.Repeat("a line of a comment\n", 12)))
	waitHeightAbove(t, page, baseline+growthFloor, "the iframe did not grow with the comment being typed")

	require.NoError(t, textarea.Fill(""))
	waitHeightNear(t, page, baseline, "the iframe did not come back down after the text was cleared")
}

// heightReports is every height the widget has asked the parent for, in order. The recorder is
// installed as an init script, so the sequence starts with the one applied during page load
func heightReports(t *testing.T, page playwright.Page) []float64 {
	t.Helper()

	v, err := page.Evaluate(`() => ((window.__r42Marks && window.__r42Marks.heights) || []).map((r) => r.h)`)
	require.NoError(t, err)

	raw, ok := v.([]any)
	require.True(t, ok, "expected a list of heights from the page, got %T (%v)", v, v)

	out := make([]float64, 0, len(raw))
	for _, item := range raw {
		out = append(out, asNumber(t, item))
	}
	return out
}

// frameHeight is the height the parent applied to the iframe element, which is the only number
// the reader ever sees
func frameHeight(t *testing.T, page playwright.Page) float64 {
	t.Helper()
	return evalNumber(t, page, `() => {
		const frame = document.querySelector('#remark42 iframe');
		return frame ? frame.getBoundingClientRect().height : -1;
	}`)
}

// inWidget reads a number out of the widget's own document, which page.Evaluate cannot reach
func inWidget(t *testing.T, page playwright.Page, script string) float64 {
	t.Helper()

	v, err := page.FrameLocator("#remark42 iframe").Locator("body").Evaluate(script, nil,
		playwright.LocatorEvaluateOptions{Timeout: playwright.Float(float64(waitTimeout.Milliseconds()))})
	require.NoError(t, err)
	return asNumber(t, v)
}

// waitHeightSettled waits for the widget to stop re-reporting its height, so an assertion about
// the sequence is made against a complete one, not against however much of it has arrived
func waitHeightSettled(t *testing.T, page playwright.Page) {
	t.Helper()

	seen, stable := 0, 0
	eventually(t, waitTimeout, "the widget never stopped changing the iframe height", func() bool {
		n := len(heightReports(t, page))
		if n > 0 && n == seen {
			stable++
		} else {
			stable = 0
		}
		seen = n
		// five polls of quiet, against a poll interval of 50ms
		return n > 0 && stable >= 5
	})
}

func waitHeightAbove(t *testing.T, page playwright.Page, floor float64, msg string) {
	t.Helper()
	eventually(t, waitTimeout, msg, func() bool { return frameHeight(t, page) > floor })
}

func waitHeightNear(t *testing.T, page playwright.Page, want float64, msg string) {
	t.Helper()
	eventually(t, waitTimeout, msg, func() bool {
		got := frameHeight(t, page)
		return got >= want-heightTolerance && got <= want+heightTolerance
	})
}

// TestGeometry_CollapsingAThreadShrinksTheFrame covers the direction the cases above do not.
// Everything else here asserts the frame growing, and a widget that only ever grew would satisfy
// all of them while leaving a hole in the page under every collapsed thread for as long as the
// reader stays on it
func TestGeometry_CollapsingAThreadShrinksTheFrame(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)
	signInAnon(t, page, frame, anonName("collapsegeometry"))

	parent := "collapse geometry parent " + runID
	postComment(t, frame, parent)

	require.NoError(t, actions(frame, parent).Locator(`button:has-text("Reply")`).Click())
	reply := "collapse geometry reply " + runID
	submitForm(t, replyForm(t, frame), reply)
	waitVisible(t, comment(frame, reply))
	waitHeightSettled(t, page)

	expanded := frameHeight(t, page)
	require.Greater(t, expanded, preloaderCeiling, "the widget never rendered, so there is no baseline")

	id, err := comment(frame, parent).GetAttribute("id")
	require.NoError(t, err)
	thread := frame.Locator(fmt.Sprintf("[aria-expanded]:has(article#%s)", id))
	require.NoError(t, thread.Locator(`:scope > [role="button"]`).Click())

	eventually(t, waitTimeout, "the frame did not shrink when the thread collapsed", func() bool {
		return frameHeight(t, page) < expanded-growthFloor
	})

	// and the document agrees, so the frame is following content and not merely being told a
	// smaller number
	assert.InDelta(t, inWidget(t, page, `() => document.body.offsetHeight`), frameHeight(t, page), heightTolerance,
		"the collapsed frame and the document it holds disagree on the height")
}
