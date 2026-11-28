//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"testing"
	"time"

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

	frame, email := openEmailStep(t, page)
	require.NoError(t, email.Fill(address))
	submitEmail(t, page, frame)

	// the token out of the message the server actually sent
	token := verificationToken(t, mailpitMessage(t, address))
	status, body := pageFetch(t, page,
		"POST", fmt.Sprintf("%s/api/v1/email/confirm?site=remark&tkn=%s", baseURL, token), nil)
	assert.Equal(t, http.StatusOK, status, "the server refused the token it had just sent: %s", body)

	// the subscription is the server's now, so the panel offers to end it on the next load
	frame = reload(t, page)
	require.NoError(t, frame.Locator(`[title="Subscribe by email"]`).Click())
	unsubscribe := frame.Locator(`button:text-is("Unsubscribe")`)
	waitVisible(t, unsubscribe)

	resp, err := page.ExpectResponse("**/api/v1/email**", func() error {
		return unsubscribe.Click()
	}, playwright.PageExpectResponseOptions{Timeout: new(float64(waitTimeout.Milliseconds()))})
	require.NoError(t, err, "clicking Unsubscribe asked the server nothing")
	assert.Equal(t, "DELETE", resp.Request().Method())
	assert.Equal(t, http.StatusOK, resp.Status(), "the server refused the unsubscribe")

	// the confirmation the panel shows afterwards, which nothing asserted before: the comment that
	// used to stand here said it was unobservable because the dropdown closed over it. Note what
	// this does not pin. It passes with the capture listener reverted, because the unsubscribe is
	// awaited and the step changes a whole task after the click has finished propagating, so the
	// button is still in the tree when a bubble-phase listener asks. The phase only decides a
	// detach that lands inside the click, which is what dropdown.test.tsx drives directly.
	waitVisible(t, frame.Locator(`text=You have unsubscribed from email updates`))

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

	frame, email := openEmailStep(t, page)

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

	frame, email := openEmailStep(t, page)
	require.NoError(t, email.Fill(fmt.Sprintf("back-%s-%d@example.com", runID, os.Getpid())))

	// moves the panel to the token step
	submitEmail(t, page, frame)

	back := frame.Locator(`button:text-is("Back")`)
	waitVisible(t, back)
	require.NoError(t, back.Click())

	// the assertion the case exists for: the panel is still open, showing the email step again
	waitVisible(t, frame.Locator(`div[role="listbox"]`))
	waitVisible(t, email)
}

// TestSubscribe_ConfirmByLink confirms the subscription from the link in the message, opened in a
// browser context with no session, the way a reader opening the mail on another device would. The
// page must ask before it subscribes, since mail scanners fetch links on their own.
//
// Not parallel: the dev user's subscription is shared with the other cases here, and a parallel one
// clearing it between the click and the reload would fail this for nothing it did.
func TestSubscribe_ConfirmByLink(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	address := fmt.Sprintf("linked-%s-%d@example.com", runID, os.Getpid())
	frame, email := openEmailStep(t, page)
	require.NoError(t, email.Fill(address))
	submitEmail(t, page, frame)
	// the token step tells the reader about the link too, and its field must be there now for its
	// absence after the link to mean anything
	tokenField := frame.Locator(`textarea[placeholder="Paste the token from the email, or click the link in it"]`)
	waitVisible(t, tokenField)

	// the reader goes to the mailbox, often by clicking somewhere on the page first. The host page
	// forwards that click as clickOutside, and the panel waiting for the token stays open: if it
	// closed, the switch to the subscribed state checked below would never show
	require.NoError(t, page.Mouse().Click(5, 5))

	// the link as the message carries it, so a wrong REMARK_URL prefix fails here too
	link := regexp.MustCompile(regexp.QuoteMeta(baseURL) + `/email/subscribe\.html\?tkn=[A-Za-z0-9._-]+`).
		FindString(mailpitMessage(t, address))
	require.NotEmpty(t, link, "no confirmation link in the message")

	// opened the way a mail client opens it: a click on a link that opens a new tab, in a browser
	// with no session. That tab starts with one history entry, which is what lets it close itself
	mail := newPage(t)
	require.NoError(t, mail.SetContent(`<a id="link" target="_blank" rel="noopener" href="`+link+`">confirm</a>`))
	reader, err := mail.Context().ExpectPage(func() error { return mail.Locator("#link").Click() })
	require.NoError(t, err, "the link opened no tab")
	confirm := reader.Locator(`button:text-is("Confirm subscription")`)
	waitVisible(t, confirm)

	// opening the page alone subscribed nobody
	status, body := pageFetch(t, page, http.MethodGet, baseURL+"/api/v1/email?site=remark", nil)
	require.Equal(t, http.StatusOK, status, body)
	assert.NotContains(t, body, address, "opening the link subscribed before the button was pressed")

	// posted in place, the page is still the tab's only history entry, so the browser lets it close
	// itself. A form post would add a second entry and leave it open. The listener goes on before the
	// click: the page closes half a second after the answer, and a close is not replayed to a later one
	closed := make(chan struct{})
	reader.OnClose(func(playwright.Page) { close(closed) })
	require.NoError(t, confirm.Click())
	select {
	case <-closed:
	case <-time.After(waitTimeout):
		t.Fatal("the confirmation page did not close itself")
	}

	status, body = pageFetch(t, page, http.MethodGet, baseURL+"/api/v1/email?site=remark", nil)
	require.Equal(t, http.StatusOK, status, body)
	assert.Contains(t, body, address, "the confirm button did not subscribe")

	// the panel still waiting for a token notices the subscription on its own, with no reload
	waitVisible(t, frame.Locator(`text=You have subscribed to email updates`))
	waitVisible(t, frame.Locator(`button:text-is("Unsubscribe")`))
	tokenFields, err := tokenField.Count()
	require.NoError(t, err)
	assert.Zero(t, tokenFields, "the token field is still open")

	// done waiting, the panel closes on a click outside again, which also shows the click above
	// reached the widget
	require.NoError(t, page.Mouse().Click(5, 5))
	waitHidden(t, frame.Locator(`button:text-is("Unsubscribe")`), "a click outside no longer closes the panel")

	status, body = pageFetch(t, page, http.MethodDelete, baseURL+"/api/v1/email?site=remark", nil)
	assert.Equal(t, http.StatusOK, status, "could not clear the subscription: %s", body)
}

// openEmailStep clears the dev user's email subscription, reloads, opens the subscription panel and
// returns the frame with the panel's email field.
//
// The dev user is shared by the whole suite and a subscription outlives the run in the stack's
// database, so a stack this has already run against starts on the subscribed step. Cleared through
// the API and not the panel: the precondition is not what is under test. 200 and 400 both leave
// nothing subscribed, which is all this needs
func openEmailStep(t *testing.T, page playwright.Page) (playwright.FrameLocator, playwright.Locator) {
	t.Helper()
	status, body := pageFetch(t, page, http.MethodDelete, baseURL+"/api/v1/email?site=remark", nil)
	require.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, status,
		"could not clear a subscription left by an earlier run: %s", body)
	frame := reload(t, page)

	subscribe := frame.Locator(`[title="Subscribe by email"]`)
	waitVisible(t, subscribe)
	require.NoError(t, subscribe.Click())

	email := frame.Locator(`input[placeholder="Email"]`)
	waitVisible(t, email)
	return frame, email
}

// submitEmail submits the email step and waits for the request it makes, so a submit going nowhere
// fails as itself
func submitEmail(t *testing.T, page playwright.Page, frame playwright.FrameLocator) {
	t.Helper()
	resp, err := page.ExpectResponse("**/api/v1/email/subscribe**", func() error {
		return frame.Locator(`button:text-is("Submit")`).Click()
	}, playwright.PageExpectResponseOptions{Timeout: new(float64(waitTimeout.Milliseconds()))})
	require.NoError(t, err, "the panel asked the server for nothing")
	require.Equal(t, http.StatusOK, resp.Status(), "the server refused to send a verification")
}

// TestSubscribe_RSSPanelOffersTheFeedsAndClosesOnAClickOutside covers the RSS panel, which no other
// case opens: the three feeds it links to, this page and the reader's own id in them, and a click
// elsewhere in the widget closing it.
func TestSubscribe_RSSPanelOffersTheFeedsAndClosesOnAClickOutside(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	rss := frame.Locator(`[title="Subscribe by RSS"]`)
	panel := frame.Locator(`[data-dropdown]:has([title="Subscribe by RSS"]) div[role="listbox"]`)
	require.NoError(t, rss.Click())
	waitVisible(t, panel)

	links := panel.Locator("a")
	hrefs, err := links.EvaluateAll(`(nodes) => nodes.map((a) => a.getAttribute('href'))`)
	require.NoError(t, err)
	got, ok := hrefs.([]any)
	require.True(t, ok, "the panel's links came back as %T", hrefs)
	require.Len(t, got, 3, "the RSS panel links to the page, the site and the replies")
	// the feeds are the instance's own, which serves the widget's document
	sameOrigin, err := links.EvaluateAll(
		`(nodes) => nodes.every((a) => new URL(a.href).origin === a.ownerDocument.location.origin)`)
	require.NoError(t, err)
	assert.Equal(t, true, sameOrigin, "a feed links to a server other than the instance's own")
	status, body := pageFetch(t, page, http.MethodGet, baseURL+"/api/v1/user?site=remark", nil)
	require.Equal(t, http.StatusOK, status, "the signed-in reader could not be read: %s", body)
	var reader struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &reader))
	require.NotEmpty(t, reader.ID)

	// each link taken apart: its path is compared whole, and so is each parameter that picks the feed
	feeds := make([]*url.URL, len(got))
	for i, href := range got {
		text, isText := href.(string)
		require.True(t, isText, "link %d has no address", i)
		feed, perr := url.Parse(text)
		require.NoError(t, perr)
		feeds[i] = feed
	}
	for i, want := range []struct {
		path, key, value, what string
	}{
		{"/api/v1/rss/post", "url", threadURL(t), "the feed of this page"},
		{"/api/v1/rss/site", "", "", "the feed of the whole site"},
		{"/api/v1/rss/reply", "user", reader.ID, "the feed of the replies to the signed-in reader, by id"},
	} {
		assert.Equal(t, want.path, feeds[i].Path, want.what)
		assert.Equal(t, "remark", feeds[i].Query().Get("site"), want.what)
		if want.key != "" {
			// the page address is the key the server finds the comments by, so it has to be the whole of it
			assert.Equal(t, want.value, feeds[i].Query().Get(want.key), want.what)
		}
	}

	// a click in the widget outside the panel, on the comment box
	require.NoError(t, frame.Locator("textarea").First().Click())
	waitHidden(t, panel, "a click outside left the RSS panel open")
}
