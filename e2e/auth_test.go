//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// signInDev completes the dev oauth2 flow, which opens a popup on the provider's own origin
func signInDev(t *testing.T, page playwright.Page, frame playwright.FrameLocator) {
	t.Helper()

	pauseForAuthLimit()
	require.NoError(t, frame.Locator(".auth-button").Click())
	popup, err := page.ExpectPopup(func() error {
		return frame.Locator(".oauth-button").First().Click()
	})
	require.NoError(t, err)

	require.NoError(t, popup.Locator("text=Authorize").Click())

	// the popup closes itself through the ?selfClose stub. while an oauth sign-in is pending
	// the widget listens for visibilitychange and window focus, so hand focus back to the
	// frame to make it re-read auth state
	pauseForAuthLimit()
	require.NoError(t, page.Locator("#remark42 iframe").Press("Tab"))
	assertSignedIn(t, page, frame)
}

// signInAnon signs in through the anonymous provider, an in-frame form with no popup. it takes
// the page because assertSignedIn may have to nudge it, see there
func signInAnon(t *testing.T, page playwright.Page, frame playwright.FrameLocator, username string) {
	t.Helper()

	pauseForAuthLimit()
	require.NoError(t, frame.Locator(".auth-button").Click())
	waitVisible(t, frame.Locator(".auth-dropdown"))

	// the tabs only render when more than one form provider is enabled; with anonymous alone
	// the form shows it directly. the labels are abbreviated ("anonym"), so match the radio
	tab := frame.Locator(`label[for="form-provider-anonymous"]`)
	if n, err := tab.Count(); err == nil && n > 0 {
		require.NoError(t, tab.Click())
	}
	require.NoError(t, frame.Locator(".auth-input-username").Fill(username))

	// wait for the request the submit is supposed to make, so a form the browser refuses to
	// submit fails here naming that, and not fifteen seconds later on a panel that was never
	// going to change. the input is validated against a pattern, see anonName
	_, err := page.ExpectResponse("**/auth/anonymous/login**", func() error {
		return frame.Locator(".auth-submit").Click()
	}, playwright.PageExpectResponseOptions{Timeout: playwright.Float(float64(waitTimeout.Milliseconds()))})
	require.NoError(t, err, "the anonymous form was never submitted, most likely because %q is not "+
		"a username it accepts", username)

	assertSignedIn(t, page, frame)
}

func TestAuth_DevProviderSignsIn(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)

	signInDev(t, page, frame)
	assertSignedIn(t, page, frame)
}

func TestAuth_AnonymousSignsIn(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)

	signInAnon(t, page, frame, "anontester")
	assertSignedIn(t, page, frame)

	name, err := frame.Locator(`[title="Open My Profile"]`).InnerText()
	require.NoError(t, err)
	assert.Contains(t, name, "anontester")
}

// TestAuth_EmailSignsIn drives the full email flow: request a code, read it back out of the
// mail catcher, and submit it. The token is what the widget sends, so a broken template or a
// broken token round-trip fails here and not silently in production.
func TestAuth_EmailSignsIn(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)

	// the mailbox is per-run: mailpit keeps everything, and with the stack left up between
	// runs a fixed address would let this test read a previous run's token
	signInEmail(t, page, frame, "emailtester", fmt.Sprintf("email-tester-%s@example.com", runID))
}

// signInEmail completes the email flow: ask for a code, read it out of the mail catcher, submit
// it. The address decides the user id, sha1 of it, so a case needing a known id picks the address
func signInEmail(t *testing.T, page playwright.Page, frame playwright.FrameLocator, username, address string) {
	t.Helper()

	pauseForAuthLimit()
	require.NoError(t, frame.Locator(".auth-button").Click())
	waitVisible(t, frame.Locator(".auth-dropdown"))

	// only rendered when more than one form provider is enabled; with email alone the form is
	// shown directly
	tab := frame.Locator(`label[for="form-provider-email"]`)
	if n, err := tab.Count(); err == nil && n > 0 {
		require.NoError(t, tab.Click())
	}
	require.NoError(t, frame.Locator(".auth-input-username").Fill(username))
	require.NoError(t, frame.Locator(".auth-input-email").Fill(address))
	require.NoError(t, frame.Locator(".auth-submit").Click())

	waitVisible(t, frame.Locator(".auth-token-textarea"))

	token := verificationToken(t, mailpitMessage(t, address))
	require.NoError(t, frame.Locator(".auth-token-textarea").Fill(token))
	require.NoError(t, frame.Locator(".auth-submit").Click())

	assertSignedIn(t, page, frame)
}

// assertSignedIn checks the panel has swapped Sign In for the signed-in user's own controls.
// Sign Out is an icon button, so its title is the only text it carries.
//
// The first wait is deliberately short. Everything under /auth/ is capped at two requests a
// second for the whole suite, a bare literal at backend/app/rest/api/rest.go:242, and a case
// that signs in on two pages spends that budget twice over. When the read that repaints the
// panel is the request the limiter refuses, the widget shows signed out over a session that
// exists, and waiting longer cannot help because nothing will ask again. So on the short wait
// expiring, hand focus back to the frame: the widget re-probes on visibilitychange and window
// focus while a sign-in is pending, and by then the cookie is long since set. A sign-in that
// genuinely failed still fails here, since the second read finds no state either
func assertSignedIn(t *testing.T, page playwright.Page, frame playwright.FrameLocator) {
	t.Helper()

	signOut := frame.Locator(`[title="Sign Out"]`)
	if err := signOut.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(authRepaintWait.Milliseconds())),
	}); err != nil {
		pauseForAuthLimit()
		require.NoError(t, page.Locator("#remark42 iframe").Press("Tab"))
	}

	waitVisible(t, signOut)
	waitVisible(t, frame.Locator(`[title="Open My Profile"]`))
	waitHidden(t, frame.Locator(".auth-button"), "the panel still offers sign-in after a sign-in")
}

// verificationToken pulls the JWT out of the confirmation mail
func verificationToken(t *testing.T, body string) string {
	t.Helper()
	m := regexp.MustCompile(`[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}`).FindString(body)
	require.NotEmpty(t, m, "no token in message body:\n%s", body)
	return m
}

// TestAuth_SignOutEndsTheSession covers the one half of authentication nothing tested at any
// level: that signing out actually ends the session instead of only repainting the panel. The
// assertion after the reload is the point, since a cleared store with a live cookie looks
// identical until the page comes back
func TestAuth_SignOutEndsTheSession(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)

	signInDev(t, page, frame)

	pauseForAuthLimit()
	require.NoError(t, frame.Locator(`[title="Sign Out"]`).Click())
	waitVisible(t, frame.Locator(".auth-button"))
	waitHidden(t, frame.Locator(`[title="Sign Out"]`))

	frame = reload(t, page)
	waitVisible(t, frame.Locator(".auth-button"))
	waitHidden(t, frame.Locator(`[title="Sign Out"]`),
		"the panel signed out but the session survived the reload, so the cookie was never cleared")
}

// TestAuth_HostPageMessageDoesNotCloseTheDropdown covers #2139. Every message reaching the widget
// closed the sign-in dropdown, because the handler returned early only for a clickOutside payload
// and fell through to closing in every other case. embed.ts watches the host page's title element
// and posts on every mutation, so a page that updates its own title discarded whatever the reader
// had typed into the login form. Browser extensions posting into the page did the same, which is
// what #1761 reports.
//
// The theme change at the end is the synchronization, not a second assertion: postMessage is
// delivered in order, so a widget that has visibly acted on the later message has already had the
// title message. Without it this would assert on a dropdown that simply has not closed yet.
func TestAuth_HostPageMessageDoesNotCloseTheDropdown(t *testing.T) {
	page := newPage(t)
	stubSignedOut(t, page)
	embedConfig(t, page, map[string]any{"theme": "light"})
	frame := widget(t, page)

	require.NoError(t, frame.Locator(".auth-button").Click())
	waitVisible(t, frame.Locator(".auth-dropdown"))

	require.NoError(t, frame.Locator(`label[for="form-provider-email"]`).Click())
	const typed = "half-written name"
	require.NoError(t, frame.Locator(".auth-input-username").Fill(typed))

	// an ordinary thing for a host page to do, and all it took
	_, err := page.Evaluate(`() => { document.title = 'the host page renamed itself'; }`)
	require.NoError(t, err)

	_, err = page.Evaluate(`() => window.REMARK42.changeTheme('dark')`)
	require.NoError(t, err)
	eventually(t, waitTimeout, "the widget never acted on the message sent after the title change", func() bool {
		return widgetColorScheme(t, page) == "dark"
	})

	waitVisible(t, frame.Locator(".auth-dropdown"))
	value, err := frame.Locator(".auth-input-username").InputValue()
	require.NoError(t, err)
	assert.Equal(t, typed, value, "the host page's own message emptied the login form")

	// and the message that is supposed to close it still does, or the fix above would be a
	// dropdown that never closes
	require.NoError(t, page.Mouse().Click(5, 5))
	waitHidden(t, frame.Locator(".auth-dropdown"), "a genuine click outside the widget no longer closes the dropdown")
}

// TestAuth_SessionSurvivesATransientStatusFailure covers the probe #1763 introduced. The widget
// asks /auth/status on every load, and a single failed answer must not be taken for a signed-out
// reader in any lasting way: the session lives in a cookie the server issued, and one bad
// response says nothing about it. A reader on a flaky connection otherwise appears to be logged
// out and cannot get back without signing in again
func TestAuth_SessionSurvivesATransientStatusFailure(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	// one failure, then out of the way. Unroute and not a counter, so the restored state is the
	// real endpoint and not a stub standing in for it
	require.NoError(t, page.Route("**/auth/status**", func(route playwright.Route) {
		_ = route.Fulfill(playwright.RouteFulfillOptions{
			Status:      playwright.Int(http.StatusInternalServerError),
			ContentType: playwright.String("application/json"),
			Body:        playwright.String(`{"error":"failed"}`),
		})
	}))

	pauseForAuthLimit()
	_, err := page.Reload()
	require.NoError(t, err)
	widget(t, page)

	require.NoError(t, page.Unroute("**/auth/status**"))

	// the session has to be there again once the endpoint is, which is the whole claim: a
	// transient failure cost the reader nothing
	frame = reload(t, page)
	assertSignedIn(t, page, frame)
}
