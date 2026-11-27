//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

	// Opening the provider and completing its callback are separate /auth/ requests. The initial
	// status probe may still occupy the other token in this reader's bucket.
	pauseForAuthLimit()
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
	}, playwright.PageExpectResponseOptions{Timeout: new(float64(waitTimeout.Milliseconds()))})
	require.NoError(t, err, "the anonymous form was never submitted, most likely because %q is not "+
		"a username it accepts", username)

	assertSignedIn(t, page, frame)
}

func TestAuth_DevProviderSignsIn(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)

	signInDev(t, page, frame)
	assertSignedIn(t, page, frame)
}

func TestAuth_AnonymousSignsIn(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)

	// a dot and a hyphen, which the form's pattern accepts only when the hyphen is escaped for
	// the v flag browsers compile it with
	signInAnon(t, page, frame, "anon.tester-1")
	assertSignedIn(t, page, frame)

	name, err := frame.Locator(`[title="Open your profile"]`).InnerText()
	require.NoError(t, err)
	assert.Contains(t, name, "anon.tester-1")
}

// TestAuth_NameFieldAppliesItsPattern checks the browser compiles the name field's pattern. A
// browser compiles it with the v flag, under which a bare hyphen in a class is a syntax error, and
// a pattern that does not compile is ignored: every name is then valid, and the ones the server
// refuses come back as a bare 403. jsdom compiles it differently, so only a browser can tell
func TestAuth_NameFieldAppliesItsPattern(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)

	require.NoError(t, frame.Locator(".auth-button").Click())
	waitVisible(t, frame.Locator(".auth-dropdown"))
	tab := frame.Locator(`label[for="form-provider-anonymous"]`)
	if n, err := tab.Count(); err == nil && n > 0 {
		require.NoError(t, tab.Click())
	}
	input := frame.Locator(".auth-input-username")

	cases := map[string]bool{
		"Ann-Marie O. Smith":             false,
		"abc\u00a0":                      false, // trimmed on submit
		strings.Repeat("\U00020BB7", 33): false, // 66 UTF-16 code units, 33 characters as the server counts them
		"Ева\u00a0Ли":                    true,
		"a*b":                            true,
		"ab ":                            true,
		"\U00020BB7田":                    true,
	}
	// typed key by key, so a length cap on the field would cut the name and show here
	for name, mismatch := range cases {
		require.NoError(t, input.Clear())
		require.NoError(t, input.PressSequentially(name))
		value, err := input.InputValue()
		require.NoError(t, err)
		assert.Equal(t, name, value, "the field kept only part of %q", name)
		got, err := input.Evaluate(`(el) => el.validity.patternMismatch`, nil)
		require.NoError(t, err)
		assert.Equal(t, mismatch, got, "patternMismatch for %q", name)
	}

	// the email tab takes no dot, since the name goes into a mail to an unconfirmed address
	require.NoError(t, frame.Locator(`label[for="form-provider-email"]`).Click())
	for name, mismatch := range map[string]bool{"Ann-Marie Smith": false, "evil-site.com": true} {
		require.NoError(t, input.Clear())
		require.NoError(t, input.PressSequentially(name))
		got, err := input.Evaluate(`(el) => el.validity.patternMismatch`, nil)
		require.NoError(t, err)
		assert.Equal(t, mismatch, got, "patternMismatch on the email tab for %q", name)
	}
}

// TestAuth_EmailSignsIn drives the full email flow: request a code, read it back out of the
// mail catcher, and submit it. The token is what the widget sends, so a broken template or a
// broken token round-trip fails here and not silently in production.
func TestAuth_EmailSignsIn(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)

	// the mailbox is per-run: mailpit keeps everything, and with the stack left up between
	// runs a fixed address would let this test read a previous run's token
	signInEmail(t, page, frame, "email-tester-1", fmt.Sprintf("email-tester-%s@example.com", runID))
}

// signInEmail completes the email flow: ask for a code, read it out of the mail catcher, paste
// it. The field signs in by itself once it holds a token, so nothing presses the submit button.
// The address decides the user id, sha1 of it, so a case needing a known id picks the address
func signInEmail(t *testing.T, page playwright.Page, frame playwright.FrameLocator, username, address string) {
	t.Helper()

	token := verificationToken(t, requestSignInEmail(t, frame, username, address))
	require.NoError(t, frame.Locator(".auth-token-textarea").Fill(token))

	assertSignedIn(t, page, frame)
}

// requestSignInEmail sends the email sign-in form, waits for the token step and returns the
// message the reader gets
func requestSignInEmail(t *testing.T, frame playwright.FrameLocator, username, address string) string {
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
	return mailpitMessage(t, address)
}

// assertSignedIn checks the panel has swapped Sign in for the signed-in user's own controls.
// Sign out is an icon button, so its title is the only text it carries.
//
// The first wait is deliberately short. Everything under /auth/ is capped at two requests a
// second per reader, a bare literal at backend/app/rest/api/rest.go:242. The initial status and
// login can spend the bucket before the read that repaints the panel, leaving the widget signed
// out over a session that exists. Waiting longer cannot help because nothing will ask again, so
// on the short wait expiring, hand focus back to the frame: the widget re-probes on
// visibilitychange and window focus while a sign-in is pending. A sign-in that genuinely failed
// still fails here, since the second read finds no state either
func assertSignedIn(t *testing.T, page playwright.Page, frame playwright.FrameLocator) {
	t.Helper()

	signOut := frame.Locator(`[title="Sign out"]`)
	if err := signOut.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: new(float64(authRepaintWait.Milliseconds())),
	}); err != nil {
		pauseForAuthLimit()
		require.NoError(t, page.Locator("#remark42 iframe").Press("Tab"))
	}

	waitVisible(t, signOut)
	waitVisible(t, frame.Locator(`[title="Open your profile"]`))
	waitHidden(t, frame.Locator(".auth-button"), "the panel still offers sign-in after a sign-in")
}

// verificationToken pulls the token to paste out of the confirmation mail. The mail carries the
// link's own token too, inside the link, and that one is skipped: the login endpoint takes either,
// so pasting the link's token would pass while the token meant for pasting went untested
func verificationToken(t *testing.T, body string) string {
	t.Helper()
	re := regexp.MustCompile(`(tkn=)?([A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,})`)
	for _, m := range re.FindAllStringSubmatch(body, -1) {
		if m[1] == "" {
			return m[2]
		}
	}
	require.FailNow(t, "no token to paste in message body", body)
	return ""
}

// TestAuth_SignOutEndsTheSession covers the one half of authentication nothing tested at any
// level: that signing out actually ends the session instead of only repainting the panel. The
// assertion after the reload is the point, since a cleared store with a live cookie looks
// identical until the page comes back
func TestAuth_SignOutEndsTheSession(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)

	signInDev(t, page, frame)

	pauseForAuthLimit()
	require.NoError(t, frame.Locator(`[title="Sign out"]`).Click())
	waitVisible(t, frame.Locator(".auth-button"))
	waitHidden(t, frame.Locator(`[title="Sign out"]`))

	frame = reload(t, page)
	waitVisible(t, frame.Locator(".auth-button"))
	waitHidden(t, frame.Locator(`[title="Sign out"]`),
		"the panel signed out but the session survived the reload, so the cookie was never cleared")
}

// TestAuth_SignOutTakesTheEditControlsAway covers the reader changing under a mounted thread.
// Signing out loads the comments again in place, so the comment the reader has just posted is the
// same component before and after, and it has to stop offering an edit it would be refused. No
// reload in between: a reload renders the comment anew and shows nothing of this
func TestAuth_SignOutTakesTheEditControlsAway(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)
	signInAnon(t, page, frame, "editthensignout")

	text := "mine until I sign out " + runID
	postComment(t, frame, text)
	edit := actions(frame, text).Locator(`button:has-text("Edit")`)
	waitVisible(t, edit)

	pauseForAuthLimit()
	require.NoError(t, frame.Locator(`[title="Sign out"]`).Click())
	waitVisible(t, frame.Locator(".auth-button"))

	waitHidden(t, edit, "the comment still offers Edit to a reader who has signed out")
	waitHidden(t, actions(frame, text).Locator(`[role="timer"]`),
		"the comment still counts an edit window down for a reader who has signed out")
	// the comment itself is still there, so the controls are gone and not the thread
	waitVisible(t, comment(frame, text))
}

// TestAuth_CommentPostedRightAfterSigningInStays covers the reload the widget makes for a reader
// who has just signed in. The thread stays usable while that request is out, so a comment can be
// posted before its response arrives, and the response shows the thread as the server had it
// before the post. The widget has to leave such a response out and ask again, or the comment goes
// off the list until the page is reloaded.
//
// The response is fetched when the widget asks for it and handed over only after the post. Holding
// the request itself would let it reach the server after the comment did, and then the response
// would have the comment in it.
func TestAuth_CommentPostedRightAfterSigningInStays(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)

	// only the first load made after signing in is held: the one before it has to render the thread,
	// and the one after it is the widget asking again
	var armed atomic.Bool
	var loads atomic.Int32
	var fetchErr error
	fetched := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseHeld := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseHeld)
	require.NoError(t, page.Route("**/api/v1/find?**", func(route playwright.Route) {
		if !armed.Load() || loads.Add(1) > 1 {
			// fails only once the page is gone, which the assertions report on their own
			_ = route.Fallback()
			return
		}
		// off the handler, so the held response does not hold up the other requests
		go func() {
			// the fetch is made outside the browser, which is the only place the stack's name resolves
			resp, err := route.Fetch(playwright.RouteFetchOptions{
				URL: new(strings.Replace(route.Request().URL(), baseURL, probeURL, 1)),
			})
			fetchErr = err
			close(fetched)
			<-release
			if err != nil {
				_ = route.Abort()
				return
			}
			_ = route.Fulfill(playwright.RouteFulfillOptions{Response: resp})
		}()
	}))

	armed.Store(true)
	signInAnon(t, page, frame, "postsatonce")
	select {
	case <-fetched:
	case <-time.After(waitTimeout):
		t.Fatal("signing in did not load the comments again")
	}
	require.NoError(t, fetchErr, "the load made after signing in could not be fetched")

	text := "posted while the thread loads again " + runID
	postComment(t, frame, text)
	releaseHeld()

	// the held response has been dealt with once the widget either asks again or takes it, and
	// taking it is what removes the comment
	posted := comment(frame, text)
	eventually(t, waitTimeout, "the widget neither asked for the comments again nor took the held response", func() bool {
		n, err := posted.Count()
		return loads.Load() > 1 || (err == nil && n == 0)
	})
	n, err := posted.Count()
	require.NoError(t, err)
	require.Equal(t, 1, n, "the comment posted while the thread was loading again went off the list "+
		"when the older response arrived")
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
	t.Parallel()

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
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	// one failure, then out of the way. Unroute and not a counter, so the restored state is the
	// real endpoint and not a stub standing in for it
	require.NoError(t, page.Route("**/auth/status**", func(route playwright.Route) {
		fulfillJSON(t, route, http.StatusInternalServerError, `{"error":"failed"}`)
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
