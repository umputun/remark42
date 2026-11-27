//go:build e2e

package e2e

import (
	"crypto/sha1" //nolint:gosec // the auth library's id scheme
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The login email carries a sign-in link next to the token to paste. The link opens a page on the
// remark42 host that signs in the browser it is opened in, only once its button is pressed, since
// mail scanners open links on their own. The widget waiting on the token step notices a sign-in in
// its own browser by itself. The link has a token of its own, so using it anywhere leaves the token
// to paste working.

// signInLink pulls the sign-in link out of the message as the message carries it, so a wrong
// REMARK_URL prefix fails here too
func signInLink(t *testing.T, body, base string) string {
	t.Helper()
	link := regexp.MustCompile(regexp.QuoteMeta(base) + `/email/login\.html\?tkn=[A-Za-z0-9._-]+`).FindString(body)
	require.NotEmpty(t, link, "no sign-in link in the message:\n%s", body)
	return link
}

// openAsMailClient opens the link the way a mail client does, a click on a link that opens a new
// tab, in the given browser context. That tab starts with one history entry, which is what lets the
// page close itself
func openAsMailClient(t *testing.T, ctx playwright.BrowserContext, link string) playwright.Page {
	t.Helper()
	mail, err := ctx.NewPage()
	require.NoError(t, err)
	t.Cleanup(func() { _ = mail.Close() })
	require.NoError(t, mail.SetContent(`<a id="link" target="_blank" rel="noopener" href="`+link+`">sign in</a>`))
	reader, err := ctx.ExpectPage(func() error { return mail.Locator("#link").Click() })
	require.NoError(t, err, "the link opened no tab")
	return reader
}

// pressSignIn presses the page's button and waits for the page to close itself, which it does half
// a second after the sign-in. The listener goes on before the click, as a close is not replayed
func pressSignIn(t *testing.T, reader playwright.Page) {
	t.Helper()
	button := reader.Locator(`button:text-is("Sign in")`)
	waitVisible(t, button)
	closed := make(chan struct{})
	reader.OnClose(func(playwright.Page) { close(closed) })
	pauseForAuthLimit()
	require.NoError(t, button.Click())
	select {
	case <-closed:
	case <-time.After(waitTimeout):
		text, _ := reader.Locator("body").InnerText()
		t.Fatalf("the sign-in page did not close itself, it reads: %q", text)
	}
}

// staysOnTokenStep asserts the widget is still waiting for the token a while after a sign-in it
// must not see. A few of the widget's one-second checks, as an absence has no event to wait for
func staysOnTokenStep(t *testing.T, frame playwright.FrameLocator, msg string) {
	t.Helper()
	signOut := frame.Locator(`[title="Sign out"]`)
	require.Never(t, func() bool {
		n, err := signOut.Count()
		return err == nil && n > 0
	}, 3*time.Second, 200*time.Millisecond, msg)
	waitVisible(t, frame.Locator(".auth-token-textarea"))
}

// sameSiteBrowser is the engine's browser able to reach the stack under the names it is configured
// with, which the link in the email carries. Chromium maps them itself; the others go through a proxy
func sameSiteBrowser(t *testing.T, engine string) playwright.Browser {
	t.Helper()
	if engine == "chromium" {
		return browser
	}

	extraBrowsersMu.Lock()
	err := playwright.Install(installOpts(engine))
	extraBrowsersMu.Unlock()
	require.NoError(t, err)

	types := map[string]playwright.BrowserType{"firefox": pw.Firefox, "webkit": pw.WebKit}
	bt, ok := types[engine]
	require.True(t, ok, "unknown browser %q", engine)
	b, err := bt.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: new(os.Getenv("E2E_HEADLESS") != "false"),
		Proxy:    &playwright.Proxy{Server: loopbackProxy(t)},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = b.Close() })
	return b
}

// TestEmailLink_SignsInTheWaitingWidget is the case the link is for: the comments wait on the token
// step, the reader opens the link in the same browser and presses the button, and the comments
// show the reader signed in with no reload and nothing pasted. Run in every engine, as it rests on
// the frame reading a cookie a top-level page has just set.
func TestEmailLink_SignsInTheWaitingWidget(t *testing.T) {
	t.Parallel()

	for _, engine := range engines() {
		t.Run(engine, func(t *testing.T) {
			page := newPageOn(t, sameSiteBrowser(t, engine))
			frame := openThread(t, page)

			name := "link" + strings.ReplaceAll(engine, "-", "")
			address := fmt.Sprintf("link-same-%s-%s-%d@example.com", engine, runID, os.Getpid())
			link := signInLink(t, requestSignInEmail(t, frame, name, address), baseURL)

			reader := openAsMailClient(t, page.Context(), link)
			waitVisible(t, reader.Locator(`text=in this browser`))

			// opening the page alone signed nobody in
			status, body := pageFetch(t, page, http.MethodGet, baseURL+"/api/v1/user?site=remark", nil)
			assert.Equal(t, http.StatusUnauthorized, status, "opening the link signed in before the button: %s", body)

			// a mark on the frame's document, gone if the widget reloads itself to sign in
			_, err := frame.Locator("body").Evaluate(`() => { window.e2eSameDocument = true }`, nil)
			require.NoError(t, err)

			pressSignIn(t, reader)

			// noticed by the widget itself: no reload, no focus handed back, nothing pasted
			waitVisible(t, frame.Locator(`[title="Sign out"]`))
			same, err := frame.Locator("body").Evaluate(`() => window.e2eSameDocument === true`, nil)
			require.NoError(t, err)
			assert.Equal(t, true, same, "the widget reloaded itself to sign in")
			profile, err := frame.Locator(`[title="Open your profile"]`).InnerText()
			require.NoError(t, err)
			assert.Contains(t, profile, name)

			// the link opened again in the signed-in browser says so, and asks for nothing
			again, err := page.Context().NewPage()
			require.NoError(t, err)
			t.Cleanup(func() { _ = again.Close() })
			_, err = again.Goto(link)
			require.NoError(t, err)
			waitVisible(t, again.Locator(`text=Signed in as`))
			n, err := again.Locator("button").Count()
			require.NoError(t, err)
			assert.Zero(t, n, "the signed-in page offers to sign in again")
		})
	}
}

// TestEmailLink_InAnotherBrowser opens the link in a browser other than the one with the comments,
// as a reader reading the mail on another device does. That browser is signed in; the comments
// still waiting are not, and the token to paste, which the link did not use up, signs them in.
func TestEmailLink_InAnotherBrowser(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)
	address := fmt.Sprintf("link-other-%s-%d@example.com", runID, os.Getpid())
	body := requestSignInEmail(t, frame, "linkother", address)
	link, token := signInLink(t, body, baseURL), verificationToken(t, body)

	other := newPage(t)
	pressSignIn(t, openAsMailClient(t, other.Context(), link))

	// the other browser has the session
	otherFrame := openThread(t, other)
	assertSignedIn(t, other, otherFrame)

	// the comments that were waiting have none: the session is in another browser's cookies
	staysOnTokenStep(t, frame, "the widget signed in from a session in another browser")

	require.NoError(t, frame.Locator(".auth-token-textarea").Fill(token))
	assertSignedIn(t, page, frame)
}

// TestEmailLink_CrossSiteWithThirdPartyCookiesBlocked is the case the link cannot serve: remark42
// on another site than the page, in a browser that blocks third-party cookies. The link signs in
// the remark42 host, which the frame on the page cannot see, so the widget stays on the token step
// and the token to paste still signs it in. The control comes first, as in
// TestHTTPS_SessionSurvivesThirdPartyCookieBlocking: without it a browser that blocks nothing
// would let the widget see the session and fail this for the wrong reason.
func TestEmailLink_CrossSiteWithThirdPartyCookiesBlocked(t *testing.T) {
	t.Parallel()

	blocking, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless:          new(os.Getenv("E2E_HEADLESS") != "false"),
		IgnoreDefaultArgs: []string{playwrightDisabledFeatures},
		Args: []string{
			withPartitioningEnabled(t),
			"--test-third-party-cookie-phaseout",
			"--host-resolver-rules=MAP remark42-https 127.0.0.1, MAP host-site-https 127.0.0.1",
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = blocking.Close() })

	page := newPageOn(t, blocking)
	_, err = page.Goto(httpsThread(t, "link-blocked"))
	require.NoError(t, err)
	frame := widget(t, page)
	assertPartitioningEnforced(t, page)

	address := fmt.Sprintf("link-blocked-%s-%d@example.com", runID, os.Getpid())
	body := requestSignInEmail(t, frame, "linkblocked", address)
	link, token := signInLink(t, body, httpsURL), verificationToken(t, body)

	pressSignIn(t, openAsMailClient(t, page.Context(), link))

	// the link did sign in this browser, on the remark42 host itself
	cookies, err := page.Context().Cookies(httpsURL)
	require.NoError(t, err)
	var firstParty bool
	for _, c := range cookies {
		if c.Name == "JWT" && c.PartitionKey == nil {
			firstParty = true
		}
	}
	require.True(t, firstParty, "the link page set no session cookie on the remark42 host: %v", cookieNames(cookies))

	staysOnTokenStep(t, frame, "the widget saw a session it has no cookie for, so the browser is not blocking")

	require.NoError(t, frame.Locator(".auth-token-textarea").Fill(token))
	assertSignedIn(t, page, frame)
}

// TestEmailLink_BlockedReaderIsToldSo presses the link of a reader blocked after asking for the email.
// The server spends the token and refuses the session, and the page has to show that answer: sending
// the form again, as it once did for any refusal, would end on "already used"
func TestEmailLink_BlockedReaderIsToldSo(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)
	address := fmt.Sprintf("link-blocked-reader-%s-%d@example.com", runID, os.Getpid())
	link := signInLink(t, requestSignInEmail(t, frame, "linkblockedreader", address), baseURL)

	userID := fmt.Sprintf("email_%x", sha1.Sum([]byte(address))) //nolint:gosec // the auth library's id scheme
	req, err := http.NewRequest(http.MethodPut, probeURL+"/api/v1/admin/user/"+userID+"?site=remark&block=1", http.NoBody)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err := probeClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode, "could not block the reader")

	reader := openAsMailClient(t, page.Context(), link)
	button := reader.Locator(`button:text-is("Sign in")`)
	waitVisible(t, button)
	closed := make(chan struct{})
	reader.OnClose(func(playwright.Page) { close(closed) })
	pauseForAuthLimit()
	require.NoError(t, button.Click())
	waitVisible(t, reader.Locator(`text=user is blocked`))
	// a sign-in closes the page half a second after its answer, so a refusal has to outlast that
	select {
	case <-closed:
		t.Fatal("the page closed over a refusal")
	case <-time.After(2 * time.Second):
	}
}
