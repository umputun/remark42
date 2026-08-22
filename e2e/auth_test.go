//go:build e2e

package e2e

import (
	"fmt"
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
	require.NoError(t, frame.Locator(".auth-submit").Click())

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
// broken token round-trip fails here rather than silently in production.
func TestAuth_EmailSignsIn(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)

	// the mailbox is per-run: mailpit keeps everything, and with the stack left up between
	// runs a fixed address would let this test read a previous run's token
	address := fmt.Sprintf("email-tester-%s@example.com", runID)

	pauseForAuthLimit()
	require.NoError(t, frame.Locator(".auth-button").Click())
	waitVisible(t, frame.Locator(".auth-dropdown"))

	require.NoError(t, frame.Locator(`label[for="form-provider-email"]`).Click())
	require.NoError(t, frame.Locator(".auth-input-username").Fill("emailtester"))
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
// level: that signing out actually ends the session rather than only repainting the panel. The
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
