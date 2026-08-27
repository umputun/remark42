//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every frame on a page can reach window.parent, and the host page acts on what arrives: it resizes
// the widget, scrolls the document and opens the profile overlay. The embed accepts messages only
// from the frame it created, and the widget accepts messages only from its parent.
//
// These cases exercise both guards in a browser. jsdom dispatches events itself, so the sender
// identity a real browser sets is whatever the test assigns, and the layout effects the parent
// applies are not observable there.

// TestHostFrame_ForeignFrameCannotDriveTheHostPage puts a second frame on the host page and posts
// the three messages the embed script acts on. None may take effect.
//
// The height is read from the iframe element the parent sizes, the scroll from the document, and the
// profile from the overlay the parent appends to the body, so each assertion is on the effect rather
// than on a listener's internals
func TestHostFrame_ForeignFrameCannotDriveTheHostPage(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)
	signInAnon(t, page, frame, anonName("hostframeguard"))
	waitHeightSettled(t, page)

	before := frameHeight(t, page)
	require.Greater(t, before, preloaderCeiling, "the widget never rendered, so there is nothing to protect")

	// tall enough that the document can actually scroll, or the scrollTo assertion would pass on a
	// page that simply has nowhere to go
	_, err := page.Evaluate(`() => {
		const filler = document.createElement('div');
		filler.style.height = '4000px';
		document.body.appendChild(filler);
		window.scrollTo(0, 0);
		return new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));
	}`)
	require.NoError(t, err)

	scrollBefore := evalNumber(t, page, `() => window.scrollY`)
	require.Less(t, scrollBefore, float64(100), "the host page did not settle at the top before the foreign message")

	// posted from inside a frame the embed script did not create, so the event's source really is
	// that frame. Calling parent.postMessage from this script instead would run in the host page's
	// own realm and set source to the top window, which is a different sender and a different test.
	//
	// The last message is the barrier. postMessage delivery is ordered, so a probe posted after the
	// three and echoed back means all four were dispatched and the three
	// were ignored. A fixed wait would turn a slow runner into a pass
	_, err = page.Evaluate("() => new Promise((resolve) => {\n" +
		"  window.addEventListener('message', function once(e) {\n" +
		"    if (e.data && e.data.e2eProbe) { window.removeEventListener('message', once); resolve(); }\n" +
		"  });\n" +
		"  const foreign = document.createElement('iframe');\n" +
		"  foreign.srcdoc = `<script>\n" +
		"    parent.postMessage({ height: 9999 }, '*');\n" +
		"    parent.postMessage({ scrollTo: 1500 }, '*');\n" +
		"    parent.postMessage({ profile: { name: 'intruder', id: 'x' } }, '*');\n" +
		"    parent.postMessage({ e2eProbe: true }, '*');\n" +
		"  <\\/script>`;\n" +
		"  document.body.appendChild(foreign);\n" +
		"})")
	require.NoError(t, err)

	assert.InDelta(t, before, frameHeight(t, page), heightTolerance,
		"a frame the embed did not create resized the widget")

	after := evalNumber(t, page, `() => window.scrollY`)
	assert.InDelta(t, scrollBefore, after, 100,
		"a frame the embed did not create scrolled the host page from %v to %v, having asked for 1500",
		scrollBefore, after)

	overlay := page.Locator(`iframe[src*="page=profile"]`)
	n, err := overlay.Count()
	require.NoError(t, err)
	assert.Zero(t, n, "a frame the embed did not create opened the profile overlay")

	// the control for that last one: an overlay that could not open on this page for some unrelated
	// reason would report the guard working. The widget's own request has to open it
	require.NoError(t, frame.Locator(`[title="Open My Profile"]`).Click())
	waitVisible(t, overlay.First())
}

// TestHostFrame_WidgetIgnoresAMessageFromAnythingButItsParent is the other direction. The widget
// document acts on signout and theme, so anything holding a reference to the frame could sign a
// reader out or repaint the widget. The guard is `event.source !== window.parent`, and the origin
// cannot stand in for it: the host page is whatever site embeds the widget.
//
// Signout is the half worth driving, since its effect survives the assertion: a reader signed out by
// a stranger stays signed out
func TestHostFrame_WidgetIgnoresAMessageFromAnythingButItsParent(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)
	signInAnon(t, page, frame, anonName("hostframe"))

	// posted from inside a frame of the host page's own making, so the event's source really is that
	// frame. srcdoc inherits the embedder's origin, which is what lets
	// it reach the widget at all: the guard has to refuse it on the sender, since the origin here
	// is the host page's own.
	//
	// The frame reports back before the test believes anything. Without that, a script that never
	// ran (a future CSP tightening on the widget page would do it, and a console error is only noted)
	// leaves the reader signed in for the trivial reason that nothing was
	// ever sent, and the case passes while testing nothing
	sent, err := page.Evaluate("() => new Promise((resolve) => {\n" +
		"  window.addEventListener('message', function once(e) {\n" +
		"    if (e.data && e.data.e2eSent) { window.removeEventListener('message', once); resolve(true); }\n" +
		"  });\n" +
		"  const foreign = document.createElement('iframe');\n" +
		"  foreign.srcdoc = `<script>\n" +
		"    parent.document.querySelector('#remark42 iframe').contentWindow" +
		".postMessage({ signout: true }, '*');\n" +
		"    parent.postMessage({ e2eSent: true }, '*');\n" +
		"  <\\/script>`;\n" +
		"  document.body.appendChild(foreign);\n" +
		"  setTimeout(() => resolve(false), 5000);\n" +
		"})")
	require.NoError(t, err)
	require.Equal(t, true, sent, "the foreign frame's script never ran, so nothing was posted at the widget")

	// the signout is ordered before the report above, so by here the widget has had it and either
	// acted on it or refused it
	assertSignedIn(t, page, frame)

	// and the reader is still signed in after a reload, so the assertion above is not reading a
	// panel that simply had not repainted yet
	frame = reload(t, page)
	assertSignedIn(t, page, frame)
}
