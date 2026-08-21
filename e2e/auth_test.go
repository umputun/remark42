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
	assertSignedIn(t, frame)
}

// signInAnon signs in through the anonymous provider, an in-frame form with no popup
func signInAnon(t *testing.T, frame playwright.FrameLocator, username string) {
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

	assertSignedIn(t, frame)
}

func TestAuth_DevProviderSignsIn(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)

	signInDev(t, page, frame)
	assertSignedIn(t, frame)
}

func TestAuth_AnonymousSignsIn(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)

	signInAnon(t, frame, "anontester")
	assertSignedIn(t, frame)

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

	assertSignedIn(t, frame)
}

// assertSignedIn checks the panel has swapped Sign In for the signed-in user's own controls.
// Sign Out is an icon button, so its title is the only text it carries
func assertSignedIn(t *testing.T, frame playwright.FrameLocator) {
	t.Helper()
	waitVisible(t, frame.Locator(`[title="Sign Out"]`))
	waitVisible(t, frame.Locator(`[title="Open My Profile"]`))
	waitHidden(t, frame.Locator(".auth-button"))
}

// verificationToken pulls the JWT out of the confirmation mail
func verificationToken(t *testing.T, body string) string {
	t.Helper()
	m := regexp.MustCompile(`[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}`).FindString(body)
	require.NotEmpty(t, m, "no token in message body:\n%s", body)
	return m
}
