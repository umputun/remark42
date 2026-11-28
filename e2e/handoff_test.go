//go:build e2e

package e2e

import (
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// OAuth sign-in for a widget framed by a page on another site. The provider's callback lands in a
// popup, which is a top-level page of remark42's own site, so the session cookie it sets is that
// site's and a frame on another site does not see it, unless the browser still sends unpartitioned
// cookies to frames and the cookie is SameSite=None. The popup hands the sign-in over to the frame
// instead, and the frame ends with a partitioned session of its own.
//
// remark42-oauth runs with the defaults a new installation has: SameSite left to the browser,
// which is Lax, and no JWT header. Without the handoff, every cross-site case below ends signed out.

// oauthThread is a thread on the TLS host page embedding remark42-oauth, a site apart from it
func oauthThread(t *testing.T, label string) string {
	t.Helper()
	return fmt.Sprintf("%s/post-oauth.html?e2e=%s-%s", httpsHostSiteURL, label, runID)
}

// blockingChromium is Chromium with third-party cookies blocked, as in
// TestHTTPS_SessionSurvivesThirdPartyCookieBlocking
func blockingChromium(t *testing.T) playwright.Browser {
	t.Helper()
	b, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless:          new(os.Getenv("E2E_HEADLESS") != "false"),
		IgnoreDefaultArgs: []string{playwrightDisabledFeatures},
		Args:              []string{withPartitioningEnabled(t), "--test-third-party-cookie-phaseout", hostResolverRules()},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = b.Close() })
	return b
}

// asStubUser sets what the stub provider signs the next sign-in in as, and whether it cuts the
// popup off from its opener with Cross-Origin-Opener-Policy, and returns the account
func asStubUser(t *testing.T, page playwright.Page, user string, coop bool) string {
	t.Helper()
	cookies := []playwright.OptionalCookie{{Name: "as", Value: user, URL: new(oauthStubURL)}}
	if coop {
		cookies = append(cookies, playwright.OptionalCookie{Name: "coop", Value: "1", URL: new(oauthStubURL)})
	}
	require.NoError(t, page.Context().AddCookies(cookies))
	return user
}

// signInStub signs in through the stub provider in a popup, as a reader clicks through a provider,
// as user, and checks the frame shows that account. coop says whether the provider was asked to
// cut the popup off from the frame, which its page reports by whether it still has an opener
func signInStub(t *testing.T, page playwright.Page, frame playwright.FrameLocator, user string, coop bool) {
	t.Helper()

	pauseForAuthLimit()
	require.NoError(t, frame.Locator(".auth-button").Click())
	popup, err := page.ExpectPopup(func() error {
		return frame.Locator(".oauth-button").First().Click()
	})
	require.NoError(t, err)

	want := "opener"
	if coop {
		want = "no opener"
	}
	waitText(t, popup.Locator("#opener"), want, "the provider page's opener")

	// the start page, the provider's login and the callback are separate /auth/ requests
	pauseForAuthLimit()
	require.NoError(t, popup.Locator("#signin").Click())
	assertSignedIn(t, page, frame)
	waitText(t, frame.Locator(`[title="Open your profile"]`), user, "the account the provider signed in")
}

func storedCookie(t *testing.T, page playwright.Page, name string) *playwright.Cookie {
	t.Helper()
	cookies, err := page.Context().Cookies()
	require.NoError(t, err)
	i := slices.IndexFunc(cookies, func(c playwright.Cookie) bool { return c.Name == name })
	if i < 0 {
		return nil
	}
	return &cookies[i]
}

// TestHandoff_CrossSiteOAuthSignsTheFrameIn signs in through the popup from a frame on another
// site, and checks the session is the frame's own: it survives a reload and carries a comment
func TestHandoff_CrossSiteOAuthSignsTheFrameIn(t *testing.T) {
	t.Parallel()

	blocking := blockingChromium(t)
	browsers := []struct {
		name string
		b    playwright.Browser
	}{{"default", browser}, {"third-party cookies blocked", blocking}}

	for _, br := range browsers {
		for _, coop := range []bool{false, true} {
			name := br.name
			if coop {
				name += ", provider cuts the opener"
			}
			t.Run(name, func(t *testing.T) {
				page := newPageOn(t, br.b)
				_, err := page.Goto(oauthThread(t, anonName("handoff")))
				require.NoError(t, err)
				frame := widget(t, page)
				if br.b == blocking {
					assertPartitioningEnforced(t, page)
				}

				user := asStubUser(t, page, anonName("handoff"), coop)
				signInStub(t, page, frame, user, coop)

				// the frame's own session, and the marker that keeps the frame on it
				for _, name := range []string{"__Host-JWT-EMBEDDED", "__Host-XSRF-TOKEN-EMBEDDED", "__Host-REMARK42-EMBEDDED"} {
					c := storedCookie(t, page, name)
					require.NotNil(t, c, "no %s cookie after the sign-in", name)
					assert.NotNil(t, c.PartitionKey, "%s is not partitioned", name)
				}

				text := "posted after an oauth handoff " + anonName(name)
				postComment(t, frame, text)

				pauseForAuthLimit()
				frame = reload(t, page)
				assertSignedIn(t, page, frame)
				waitVisible(t, comment(frame, text))
			})
		}
	}
}

// TestHandoff_SameOriginKeepsTheOrdinarySession is the control: on remark42's own pages the popup's
// session is the one the widget sees, so nothing is handed over and no embedded cookie is set
func TestHandoff_SameOriginKeepsTheOrdinarySession(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openURL(t, page, threadURLOn(t, oauthURL))
	user := asStubUser(t, page, anonName("sameorigin"), false)
	signInStub(t, page, frame, user, false)

	require.NotNil(t, storedCookie(t, page, "JWT"), "the popup's own session")
	for _, name := range []string{"__Host-JWT-EMBEDDED", "__Host-XSRF-TOKEN-EMBEDDED", "__Host-REMARK42-EMBEDDED"} {
		assert.Nil(t, storedCookie(t, page, name), "%s set for a widget on remark42's own site", name)
	}

	frame = reload(t, page)
	assertSignedIn(t, page, frame)
}

// TestHandoff_FrameSignOutLeavesTheSite signs the frame out of its own session and checks it stays
// out after a reload, with the marker kept. remark42's own site, which the popup signed in, is
// still signed in afterwards; under this blocking policy the browser would not let the frame's
// response touch the site's cookies anyway, so the server's half of that is
// TestHandoff_SessionsStayApart in the backend
func TestHandoff_FrameSignOutLeavesTheSite(t *testing.T) {
	t.Parallel()

	page := newPageOn(t, blockingChromium(t))
	_, err := page.Goto(oauthThread(t, "handoff-signout"))
	require.NoError(t, err)
	frame := widget(t, page)
	user := asStubUser(t, page, anonName("signout"), false)
	signInStub(t, page, frame, user, false)

	pauseForAuthLimit()
	require.NoError(t, frame.Locator(`[title="Sign out"]`).Click())
	waitVisible(t, frame.Locator(".auth-button"))

	frame = reload(t, page)
	waitVisible(t, frame.Locator(".auth-button"))
	waitHidden(t, frame.Locator(`[title="Sign out"]`), "the frame is signed in again after its sign-out")
	assert.NotNil(t, storedCookie(t, page, "__Host-REMARK42-EMBEDDED"), "the marker outlives the sign-out")

	site, err := page.Context().NewPage()
	require.NoError(t, err)
	siteFrame := openURL(t, site, threadURLOn(t, oauthURL))
	assertSignedIn(t, site, siteFrame)
}

// TestHandoff_StartPageWithoutAnOpener opens the start page as a page of its own, as a reader who
// opens the sign-in link in a new tab does, so there is no frame to hand the sign-in to. The page
// gives the handoff up and goes on to the ordinary login, which signs the reader in as before
func TestHandoff_StartPageWithoutAnOpener(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	asStubUser(t, page, anonName("noopener"), false)
	from := oauthURL + "/web/"
	start := oauthURL + "/auth/handoff/start?" + url.Values{"provider": {"stubauth"}, "site": {"remark"}, "from": {from},
		"challenge": {strings.Repeat("A", 43)}}.Encode()
	_, err := page.Goto(start)
	require.NoError(t, err)

	pauseForAuthLimit()
	require.NoError(t, page.Locator("#signin").Click())
	// the callback goes back to the from of the ordinary login, not to the handoff's done page
	require.NoError(t, page.WaitForURL(from+"*"))
	assert.Nil(t, storedCookie(t, page, "__Host-REMARK42-EMBEDDED"))

	frame := openURL(t, page, threadURLOn(t, oauthURL))
	assertSignedIn(t, page, frame)
}

// TestHandoff_BlockedPopupSaysSo has the browser refuse the sign-in window, and the panel says
// what to do about it.
//
// The refusal is stood in for: no engine Playwright ships blocks a popup, Chromium and Firefox
// open one without a user gesture whatever their flags and preferences, so the frame's
// window.open answers with null, which is what a blocking browser returns. Release Chrome 158 and
// Firefox 160 were checked outside this suite and answer a window.open without user activation
// that way.
func TestHandoff_BlockedPopupSaysSo(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	require.NoError(t, page.Context().AddInitScript(playwright.Script{Content: new(`window.open = () => null;`)}))
	_, err := page.Goto(oauthThread(t, "handoff-blocked"))
	require.NoError(t, err)
	frame := widget(t, page)

	pauseForAuthLimit()
	require.NoError(t, frame.Locator(".auth-button").Click())
	require.NoError(t, frame.Locator(".oauth-button").First().Click())
	waitVisible(t, frame.GetByText("Your browser blocked the sign-in window"))
	assert.Len(t, page.Context().Pages(), 1, "no window was opened")
}
