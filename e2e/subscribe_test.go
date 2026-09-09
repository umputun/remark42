//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSubscribe_EmailRoundTrip drives the email subscription: ask for it from the panel, confirm
// it with the token that arrives in the message, and give it up again.
//
// The whole area was unreachable from a browser until the stack enabled the notify module, since
// the widget renders the control from `email_notifications` in the config and hides it otherwise.
// It is also the only place a token out of a real message is exchanged for state the server
// keeps, so a broken template, a broken token round trip or a broken unsubscribe all land here.
//
// The dev user, not an email one: signing in by email leaves the account already subscribed, so
// the panel opens on the subscribed step and there is nothing to ask for. Anonymous will not do
// either, the control being disabled for anonymous users.
//
// The confirmation goes through the page's own session instead of the panel's textarea. The panel
// moves to its subscribed step while that textarea is still on screen, so there is no moment at
// which a control to submit it can be located; the request carries the token that arrived in the
// message either way, which is what the exchange is.
func TestSubscribe_EmailRoundTrip(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	// per-run and per-process, for the reason anonName carries the pid: mailpit keeps every
	// message, so a fixed address would also let this read a token from an earlier run
	address := fmt.Sprintf("subscriber-%s-%d@example.com", runID, os.Getpid())

	// the dev user is shared by the whole suite and a subscription outlives the run in the
	// stack's database, so a stack this has already run against starts on the subscribed step.
	// cleared through the API and not the panel: the precondition is not what is under test.
	// 200 and 400 both leave nothing subscribed, which is all this needs
	status, body := pageFetch(t, page, "DELETE", baseURL+"/api/v1/email?site=remark", nil)
	require.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, status,
		"could not clear a subscription left by an earlier run: %s", body)

	frame = reload(t, page)

	subscribe := frame.Locator(`[title="Subscribe by Email"]`)
	waitVisible(t, subscribe)
	require.NoError(t, subscribe.Click())

	email := frame.Locator(`input[placeholder="Email"]`)
	waitVisible(t, email)
	require.NoError(t, email.Fill(address))

	// the request the panel makes, so a submit going nowhere fails as itself
	resp, err := page.ExpectResponse("**/api/v1/email/subscribe**", func() error {
		return frame.Locator(`button:text-is("Submit")`).Click()
	}, playwright.PageExpectResponseOptions{Timeout: playwright.Float(float64(waitTimeout.Milliseconds()))})
	require.NoError(t, err, "the panel asked the server for nothing")
	require.Equal(t, http.StatusOK, resp.Status(), "the server refused to send a verification")

	// the token out of the message the server actually sent
	token := verificationToken(t, mailpitMessage(t, address))
	status, body = pageFetch(t, page,
		"POST", fmt.Sprintf("%s/api/v1/email/confirm?site=remark&tkn=%s", baseURL, token), nil)
	assert.Equal(t, http.StatusOK, status, "the server refused the token it had just sent: %s", body)

	// the subscription is the server's now, so the panel offers to end it on the next load
	frame = reload(t, page)
	require.NoError(t, frame.Locator(`[title="Subscribe by Email"]`).Click())
	unsubscribe := frame.Locator(`button:text-is("Unsubscribe")`)
	waitVisible(t, unsubscribe)

	resp, err = page.ExpectResponse("**/api/v1/email**", func() error {
		return unsubscribe.Click()
	}, playwright.PageExpectResponseOptions{Timeout: playwright.Float(float64(waitTimeout.Milliseconds()))})
	require.NoError(t, err, "clicking Unsubscribe asked the server nothing")
	assert.Equal(t, "DELETE", resp.Request().Method())
	assert.Equal(t, http.StatusOK, resp.Status(), "the server refused the unsubscribe")

	// the confirmation the panel shows afterwards, which nothing asserted before: the comment that
	// used to stand here said it was unobservable because the dropdown closed over it. Note what
	// this does not pin. It passes with the capture listener reverted, because the unsubscribe is
	// awaited and the step changes a whole task after the click has finished propagating, so the
	// button is still in the tree when a bubble-phase listener asks. The phase only decides a
	// detach that lands inside the click, which is what dropdown.test.tsx drives directly.
	waitVisible(t, frame.Locator(`text=You have been unsubscribed by email to updates`))

	// the panel closes when asked to, which is the half a null render used to fake: the old trick
	// emptied the content and left the listbox itself open on the page
	closeButton := frame.Locator(`button:text-is("Close")`)
	waitVisible(t, closeButton)
	require.NoError(t, closeButton.Click())
	waitHidden(t, frame.Locator(`div[role="listbox"]`))
}

// TestSubscribe_PanelStaysOpenWhenAnInnerClickDetachesItsTarget pins the phase of the dropdown's
// outside-click listener. The panel used to close on a click inside itself whenever the click's
// own handler rerendered the clicked node away: by the time a bubble-phase listener on the
// document asked whether the target was inside the panel, the target was no longer anywhere, and
// the answer was no. Deciding in the capture phase asks while the node is still attached.
//
// No flow in the widget detaches its clicked control inside the click's task today, so this case
// drives the detach itself, from a capture listener on the panel that removes the target. That is
// the shape from dropdown.test.tsx, and the reason it lives here as well is that jsdom keeps
// contains(target) true in both phases and so cannot tell the two apart, while a real engine can:
// with the document listener back in the bubble phase, this case closes the panel and fails.
//
// Not parallel: it needs the email step, so it clears the shared dev user's subscription, and the
// round trip above subscribes that same user in the middle of its run.
func TestSubscribe_PanelStaysOpenWhenAnInnerClickDetachesItsTarget(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	status, body := pageFetch(t, page, "DELETE", baseURL+"/api/v1/email?site=remark", nil)
	require.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, status,
		"could not clear a subscription left by an earlier run: %s", body)
	frame = reload(t, page)

	subscribe := frame.Locator(`[title="Subscribe by Email"]`)
	waitVisible(t, subscribe)
	require.NoError(t, subscribe.Click())

	email := frame.Locator(`input[placeholder="Email"]`)
	waitVisible(t, email)

	// the panel's own capture listener runs after the document's, so with the fix the document has
	// already answered "inside" by the time the target goes; without it the document asks in the
	// bubble phase, after this has run, and the target is gone. the handle is kept so the removal
	// itself can be asserted, or a listener that never fired would pass this vacuously
	panel := frame.Locator(`div[role="listbox"]`)
	_, err := panel.Evaluate(`(node) => {
		node.addEventListener('click', (e) => {
			window.__e2eDetached = e.target;
			e.target.remove();
		}, { capture: true, once: true });
	}`, nil)
	require.NoError(t, err)

	// the click's own dispatch removes its target. The playwright documentation says an action
	// throws when its target detaches, but under the pinned 1.62.1 the click completes, because the
	// attachment checks run before the input is dispatched. If a bump makes this line fail, that
	// is playwright and not the phase: the assertion for the phase is the last line of the case
	require.NoError(t, email.Click())

	detached, err := frame.Locator("body").Evaluate(
		`() => window.__e2eDetached instanceof Node && !window.__e2eDetached.isConnected`, nil)
	require.NoError(t, err)
	require.Equal(t, true, detached, "the click did not detach its target, so the phase was never exercised")

	// the assertion the case exists for: a click inside the panel, whatever it did to its own
	// target, is not an outside click
	waitVisible(t, panel)
}

// TestSubscribe_BackFromTheTokenStepKeepsThePanelOpen is the flow in the widget that detaches its
// clicked control inside the click's own task, which is what the capture-phase listener exists
// for. Back on the token step changes the step synchronously, the rerender removes the Back button
// before the click reaches the document, and a bubble-phase listener would find no target inside
// the panel and close it over the email form the reader just asked for. The handler used to defer
// its step change behind a zero timeout to dodge exactly that, which is why nothing pinned it.
//
// Not parallel, for the same reason as the case above: it needs an unsubscribed dev user.
func TestSubscribe_BackFromTheTokenStepKeepsThePanelOpen(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	status, body := pageFetch(t, page, "DELETE", baseURL+"/api/v1/email?site=remark", nil)
	require.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, status,
		"could not clear a subscription left by an earlier run: %s", body)
	frame = reload(t, page)

	subscribe := frame.Locator(`[title="Subscribe by Email"]`)
	waitVisible(t, subscribe)
	require.NoError(t, subscribe.Click())

	email := frame.Locator(`input[placeholder="Email"]`)
	waitVisible(t, email)
	require.NoError(t, email.Fill(fmt.Sprintf("back-%s-%d@example.com", runID, os.Getpid())))

	// the request that moves the panel to the token step, so a submit going nowhere fails as itself
	resp, err := page.ExpectResponse("**/api/v1/email/subscribe**", func() error {
		return frame.Locator(`button:text-is("Submit")`).Click()
	}, playwright.PageExpectResponseOptions{Timeout: playwright.Float(float64(waitTimeout.Milliseconds()))})
	require.NoError(t, err, "the panel asked the server for nothing")
	require.Equal(t, http.StatusOK, resp.Status(), "the server refused to send a verification")

	back := frame.Locator(`button:text-is("Back")`)
	waitVisible(t, back)
	require.NoError(t, back.Click())

	// the assertion the case exists for: the panel is still open, showing the email step again
	waitVisible(t, frame.Locator(`div[role="listbox"]`))
	waitVisible(t, email)
}
