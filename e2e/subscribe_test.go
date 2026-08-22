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

	// the panel confirming an unsubscribe is not observable: the click changes the step, and the
	// dropdown closes on an element no longer in the rerendered view, which the component notes
	// as a known awkwardness of its own. So assert what the server was asked and what it
	// answered, which is what decides whether the reader still gets mail
	resp, err = page.ExpectResponse("**/api/v1/email**", func() error {
		return unsubscribe.Click()
	}, playwright.PageExpectResponseOptions{Timeout: playwright.Float(float64(waitTimeout.Milliseconds()))})
	require.NoError(t, err, "clicking Unsubscribe asked the server nothing")
	assert.Equal(t, "DELETE", resp.Request().Method())
	assert.Equal(t, http.StatusOK, resp.Status(), "the server refused the unsubscribe")
}
