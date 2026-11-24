//go:build e2e

package e2e

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Everything else in this suite speaks http, which hides an entire class of defect: the browser
// gates Secure cookies, SameSite=None, Partitioned and anything reading location.protocol on the
// page protocol, so code taking those paths is never executed. The widget once decorated the
// cookies it wrote with __Host- on https pages, and that survived for exactly this reason.
//
// These run against the TLS pair in compose-e2e-test.yml, which carries a self-signed certificate
// every context here accepts. That instance also runs with AUTH_SEND_JWT_HEADER, which is what
// gives a widget framed by another site cookies it can keep: the backend repeats its own pair in
// the partitioned form for it. Without the setting that does not happen on any https page in the
// stack, and every assertion about those cookies is vacuous.

// An unpartitioned third-party cookie is refused only when storage partitioning is enforced, and
// playwright's own default arguments disable it. Dropping that default and re-supplying it without
// the one feature is the only lever; see "What this suite cannot reach" in the README.
//
// The list is playwright's own and therefore version-specific. It is not trusted on its own: the
// case using it asserts a control first, so a list that stops matching fails as itself.
const (
	playwrightDisabledFeatures = "--disable-features=AvoidUnnecessaryBeforeUnloadCheckSync," +
		"BoundaryEventDispatchTracksNodeRemoval,DestroyProfileOnBrowserClose,DialMediaRouteProvider," +
		"GlobalMediaControls,HttpsUpgrades,LensOverlay,MediaRouter,PaintHolding," +
		"ThirdPartyStoragePartitioning,BlockOriginHeaderModificationOnRedirect,Translate,AutoDeElevate," +
		"OptimizationHints,msForceBrowserSignIn,msEdgeUpdateLaunchServicesPreferredVersion"

	// the one entry that has to go, named once
	partitioningFeature = "ThirdPartyStoragePartitioning,"
)

// withPartitioningEnabled is playwright's own list with the one feature removed, derived rather
// than written out again: the two differ by a single entry in three hundred characters, and a
// second copy is a thing to forget on the next playwright bump
func withPartitioningEnabled(t *testing.T) string {
	t.Helper()

	out := strings.Replace(playwrightDisabledFeatures, partitioningFeature, "", 1)
	require.NotEqual(t, playwrightDisabledFeatures, out,
		"%q is no longer in playwright's default argument list, so removing it enables nothing and "+
			"this browser would block no cookie at all", partitioningFeature)
	return out
}

// The flows worth measuring in a third-party frame. OAuth is absent on purpose: the provider
// callback lands in a popup, which is a top-level context of its own, so the cookie set there is
// keyed to the auth host and never reaches the frame. Telegram is absent because the stack cannot
// answer as Telegram, see #2208.
//
// Email as well as anonymous because the partitioned pair is added to whichever response sets the
// cookies, whatever provider produced it, so the two are expected to behave alike. Expected is not
// measured, and email is the flow an operator running off-domain actually reaches for
var httpsFlows = []struct {
	name   string
	signIn func(t *testing.T, page playwright.Page, frame playwright.FrameLocator)
}{
	{"anonymous", func(t *testing.T, page playwright.Page, frame playwright.FrameLocator) {
		signInAnon(t, page, frame, anonName("httpsreader"))
	}},
	{"email", func(t *testing.T, page playwright.Page, frame playwright.FrameLocator) {
		// the address decides the user id and the mailbox this reads back, so it carries the run
		// id and the case: mailpit keeps everything, and a shared address would let one case read
		// the token another asked for
		signInEmail(t, page, frame, "httpsemail",
			fmt.Sprintf("https-email-%s-%s@example.com", strings.ReplaceAll(t.Name(), "/", "-"), runID))
	}},
}

// httpsThread is a thread on the TLS host page, which is a different site from the widget's own
// origin, so the widget's cookies there are genuinely third-party
func httpsThread(t *testing.T, label string) string {
	t.Helper()
	return fmt.Sprintf("%s/post-https.html?e2e=%s-%s", httpsHostSiteURL, label, runID)
}

// TestHTTPS_CrossOriginSignInSurvivesAReload is what the http cross-origin case cannot assert.
// The widget keeps its token in memory for the life of a page, so signing in and posting proves
// nothing about persistence: only a reload asks whether the cookie was delivered, stored under a
// name the backend reads, and sent back from a third-party frame. On http that cookie cannot even
// be written, since the third-party form requires Secure.
//
// This runs under the browser's default policy, where third-party cookies are allowed.
// TestHTTPS_SessionSurvivesThirdPartyCookieBlocking is the same question with them blocked.
func TestHTTPS_CrossOriginSignInSurvivesAReload(t *testing.T) {
	t.Parallel()

	for _, flow := range httpsFlows {
		t.Run(flow.name, func(t *testing.T) {
			page := newPage(t)

			_, err := page.Goto(httpsThread(t, "https-signin-"+flow.name))
			require.NoError(t, err)

			frame := widget(t, page)
			flow.signIn(t, page, frame)

			text := "posted over tls by " + flow.name + " " + runID
			postComment(t, frame, text)

			// the assertion the whole file exists for
			pauseForAuthLimit()
			_, err = page.Reload()
			require.NoError(t, err)

			frame = widget(t, page)
			assertSignedIn(t, page, frame)
			waitVisible(t, comment(frame, text))
		})
	}
}

// TestHTTPS_AuthCookiesCarryTheThirdPartyForm reads the cookies the browser actually stored for
// the widget's origin while it is embedded elsewhere. A cookie the browser refused is not in this
// list at all, and one the browser kept but will not send from a frame is worse than useless, so
// the attributes are the assertion and not the sign-in they enable.
//
// Both cookies are checked and not the JWT alone: the fetcher reads XSRF-TOKEN and returns its
// value as a header, and go-pkgz/auth refuses a cookie-borne token whose header does not match, so
// a JWT arriving beside an XSRF cookie the frame cannot receive authenticates nobody.
func TestHTTPS_AuthCookiesCarryTheThirdPartyForm(t *testing.T) {
	t.Parallel()

	page := newPage(t)

	_, err := page.Goto(httpsThread(t, "https-cookies"))
	require.NoError(t, err)

	frame := widget(t, page)
	signInAnon(t, page, frame, anonName("httpscookies"))

	cookies, err := page.Context().Cookies()
	require.NoError(t, err)

	// two pairs are in play here: the one the backend has always set, and the partitioned copy it
	// adds for a widget that says it is framed by another host. Every copy has to be usable from a
	// third-party frame, and the partitioned one has to exist, or the assertions below would be
	// reading the ordinary pair and saying nothing about the one a blocking browser keeps
	for _, name := range []string{"JWT", "XSRF-TOKEN"} {
		var partitioned int
		var seen int
		for _, c := range cookies {
			if c.Name != name {
				continue
			}
			seen++
			assert.True(t, c.Secure, "a %s cookie is not Secure, so a third-party frame can never receive it", name)
			require.NotNil(t, c.SameSite, "a %s cookie carries no SameSite at all", name)
			assert.Equal(t, *playwright.SameSiteAttributeNone, *c.SameSite,
				"a %s cookie is not SameSite=None, and SameSite is judged against the top-level site, so "+
					"anything stricter is never sent from an embedded widget", name)
			if c.PartitionKey != nil {
				partitioned++
			}
		}
		require.NotZero(t, seen, "no %s cookie was stored at all, so the browser refused what was sent: %v",
			name, cookieNames(cookies))
		assert.NotZero(t, partitioned,
			"no partitioned %s cookie was stored, so the backend did not repeat its cookie for the framed "+
				"widget, or chose attributes a blocking browser will drop", name)
	}

	// the partitioned JWT is the backend's and not one written by script. A script-written one is
	// what WebKit refuses to store once an HttpOnly JWT exists for this host outside the frame, see
	// TestHTTPS_WebKitFrameSessionSurvivesAFirstPartyCookie, and it is readable by anything running
	// in the frame. HttpOnly tells the two apart: script cannot set it
	for _, c := range cookies {
		if c.Name == "JWT" && c.PartitionKey != nil {
			assert.True(t, c.HttpOnly, "the partitioned JWT is not HttpOnly, so it was written by the widget's "+
				"script and the backend added no copy of its own")
		}
	}

	// the XSRF value has to be readable by the widget's own script, which is what puts it in the
	// header the backend matches the token against
	for _, c := range cookies {
		if c.Name == "XSRF-TOKEN" {
			assert.False(t, c.HttpOnly, "the XSRF cookie has to be readable by the widget's own script")
		}
	}

	// no name the server does not read. A __Host- prefixed cookie is a perfectly valid cookie the
	// browser stores happily, which is why writing one is a defect nothing rejects: the backend
	// looks for JWT and the fetcher reads XSRF-TOKEN, and neither finds a decorated name. The
	// widget writes no auth cookie any more, so this holds a later change to that
	for _, c := range cookies {
		assert.NotContains(t, c.Name, "__Host-",
			"a __Host- prefixed cookie was stored, and nothing on either side reads that name")
	}
}

// TestHTTPS_SessionSurvivesThirdPartyCookieBlocking is the reload question under the policy that
// makes it hard. With storage partitioning enforced an ordinary third-party cookie is dropped, so
// the session survives only because of the partitioned pair the backend adds for a framed widget;
// the pair it has always set is not partitioned, and vanishes.
//
// The control comes first and is not decoration: playwright's default arguments disable the policy
// outright, so a run configured wrongly keeps every third-party cookie and this case would pass
// while asserting nothing at all.
func TestHTTPS_SessionSurvivesThirdPartyCookieBlocking(t *testing.T) {
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

	for _, flow := range httpsFlows {
		t.Run(flow.name, func(t *testing.T) {
			// a context of its own, so neither case inherits what the other stored
			page := newPageOn(t, blocking)

			_, err := page.Goto(httpsThread(t, "https-blocked-"+flow.name))
			require.NoError(t, err)
			frame := widget(t, page)

			assertPartitioningEnforced(t, page)

			flow.signIn(t, page, frame)

			// the chain this case rests on, asserted before the reload can fail on the end of it:
			// the widget says it is framed by another host, and the backend answers that with a
			// partitioned copy of its cookies only under AUTH_SEND_JWT_HEADER. drop the flag from
			// the service and the reload below signs nobody in, which reads as a defect in cookie
			// handling instead of a stack that cannot test it
			written, err := page.Context().Cookies()
			require.NoError(t, err)
			require.True(t, slices.ContainsFunc(written, func(c playwright.Cookie) bool {
				return c.Name == "JWT" && c.PartitionKey != nil && c.HttpOnly
			}), "the backend set no partitioned JWT cookie before the reload: remark42-https is most likely no "+
				"longer configured with AUTH_SEND_JWT_HEADER, or the widget no longer sends "+
				"X-Remark42-Partitioned-Cookie. stored: %v", cookieNames(written))

			pauseForAuthLimit()
			_, err = page.Reload()
			require.NoError(t, err)

			frame = widget(t, page)
			assertSignedIn(t, page, frame)

			// and it survived because those cookies are partitioned, which is the only form a
			// blocking browser keeps
			cookies, err := page.Context().Cookies()
			require.NoError(t, err)
			for _, name := range []string{"JWT", "XSRF-TOKEN"} {
				var stored *playwright.Cookie
				for i, c := range cookies {
					if c.Name == name {
						stored = &cookies[i]
						break
					}
				}
				require.NotNil(t, stored, "no %s cookie survived the blocking browser: %v", name, cookieNames(cookies))
				assert.NotNil(t, stored.PartitionKey,
					"the %s cookie is not partitioned, so it only survived because the browser was not blocking", name)
			}
		})
	}
}

// assertPartitioningEnforced proves the browser refuses an ordinary third-party cookie before any
// case leans on it. Both cookies are written and read back inside a single frame evaluate, and the
// read is `document.cookie` rather than the context's cookie list, because the two are different
// channels with no ordering between them.
//
// Pinned to a build rather than stated as a rule: playwright 1.62.1 ships chromium 151.0.7922.34,
// where Blink's CookieJar::SetCookie queues the write and returns without a completion callback,
// and the renderer's own document.cookie getter is the barrier that forces the pending write to
// settle. Playwright's Context().Cookies() is a browser-session Storage.getCookies that never
// touches that frame's jar, so reading the control through it can find the name absent because the
// write has not landed yet, which is precisely the outcome the control exists to rule out. An
// implementation is free to serialize both calls in the browser process, so this is what to
// re-check on a playwright bump.
//
// The sentinel is the positive half and it has to come first: a valid third-party cookie proves the
// write path and the partitioned path are both live, and only then does the control's absence mean
// the browser turned it down rather than that nothing was written at all
func assertPartitioningEnforced(t *testing.T, page playwright.Page) {
	t.Helper()

	const (
		sentinel = "e2esentinel"
		control  = "e2econtrol"
	)

	// one evaluate: both writes, then the getter that settles them, returned as separate answers so
	// a missing sentinel and a surviving control stay distinguishable
	raw, err := page.FrameLocator("#remark42 iframe").Locator("body").Evaluate(`() => {
		document.cookie = 'e2econtrol=1; Path=/; Secure; SameSite=None';
		document.cookie = 'e2esentinel=1; Path=/; Secure; SameSite=None; Partitioned';
		const names = document.cookie.split(';').map((c) => c.trim().split('=')[0]);
		return { sentinel: names.includes('e2esentinel'), control: names.includes('e2econtrol'), all: names };
	}`, nil)
	require.NoError(t, err)

	got, ok := raw.(map[string]any)
	require.True(t, ok, "expected an object from the frame, got %T (%v)", raw, raw)

	require.Equal(t, true, got["sentinel"],
		"a %s cookie in the third-party form the browser still accepts was not stored, so nothing was "+
			"written at all and the absence of %s below would mean nothing. cookies in the frame: %v",
		sentinel, control, got["all"])

	require.Equal(t, false, got["control"],
		"an ordinary third-party %s cookie survived alongside the partitioned %s, so this browser is "+
			"not partitioning storage and nothing here is being tested. playwright's default argument "+
			"list has most likely changed: see playwrightDisabledFeatures. cookies in the frame: %v",
		control, sentinel, got["all"])
}

func cookieNames(cookies []playwright.Cookie) []string {
	out := make([]string, 0, len(cookies))
	for _, c := range cookies {
		out = append(out, c.Name)
	}
	return out
}

// TestHTTPS_WebKitFrameSessionSurvivesAFirstPartyCookie covers a refusal only WebKit makes. Once
// an HttpOnly cookie named JWT exists for the remark42 host as a site of its own, script inside a
// frame on another site can no longer store a Partitioned cookie of that name there. An OAuth
// popup leaves such a cookie, and so does signing in on any page that shares remark42's
// registrable domain, and it lives for 200 hours. The widget used to keep its session in exactly
// that script-written cookie, so the reader signed in, saw their name, and was signed out by the
// next reload. A cookie the server sets is not refused, which is what the backend now does for a
// widget that says it is framed by another host.
//
// WebKit only, and macOS only in practice. Chromium keys the two cookies apart and never had the
// defect. The WebKit build playwright ships for Linux keeps no cookie at all in a third-party
// frame, Partitioned included, so there the reload signs the reader out with or without the fix and
// the case would assert nothing: it skips, saying so. CI runs on Linux, so this case is not part of
// what CI proves; TestHTTPS_AuthCookiesCarryTheThirdPartyForm holds the server's half there.
//
// The names in this stack resolve nowhere, and --host-resolver-rules is a Chromium flag, so this
// browser reaches them through a proxy in the test process.
func TestHTTPS_WebKitFrameSessionSurvivesAFirstPartyCookie(t *testing.T) {
	t.Parallel()

	if !slices.Contains(engines(), "webkit") {
		t.Skip("webkit is not among E2E_BROWSERS")
	}

	extraBrowsersMu.Lock()
	err := playwright.Install(installOpts("webkit"))
	extraBrowsersMu.Unlock()
	require.NoError(t, err)

	wk, err := pw.WebKit.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: new(os.Getenv("E2E_HEADLESS") != "false"),
		Proxy:    &playwright.Proxy{Server: loopbackProxy(t)},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = wk.Close() })

	page := newPageOn(t, wk)

	// the cookie in the way: a session on remark42's own site, made the ordinary way
	frame := openURL(t, page, fmt.Sprintf("%s/web/?e2e=webkit-own-%s", httpsURL, runID))
	signInAnon(t, page, frame, anonName("wkown"))

	cookies, err := page.Context().Cookies(httpsURL)
	require.NoError(t, err)
	require.True(t, slices.ContainsFunc(cookies, func(c playwright.Cookie) bool {
		return c.Name == "JWT" && c.HttpOnly && c.PartitionKey == nil
	}), "signing in on remark42's own site left no unpartitioned HttpOnly JWT cookie, so the cookie this "+
		"case is about is not there and the reload below would pass on any build. stored: %v", cookieNames(cookies))

	frame = openURL(t, page, httpsThread(t, "webkit-frame"))

	if !frameKeepsPartitionedCookies(t, page) {
		// never on a mac, where this WebKit is known to keep them: a skip there would be the case
		// rotting quietly on the one platform it can run on
		require.NotEqual(t, "darwin", runtime.GOOS, "this WebKit kept no Partitioned cookie written inside the "+
			"third-party frame, which the build for macOS does, so something else is wrong with the frame")
		t.Skip("this WebKit build keeps no cookie in a third-party frame, Partitioned included, so no session " +
			"can survive a reload there whatever the server sends")
	}

	// the session above is not visible in here: WebKit sends no unpartitioned cookie from a frame on
	// another site. if it were, the sign-in below would have nothing to do
	waitVisible(t, frame.Locator(".auth-button"))

	reader := anonName("wkframe")
	signInAnon(t, page, frame, reader)

	frame = reload(t, page)
	assertSignedIn(t, page, frame)

	// as the reader who signed in here, and not the one from remark42's own site
	name, err := frame.Locator(`[title="Open your profile"]`).InnerText()
	require.NoError(t, err)
	assert.Contains(t, name, reader)
}

// frameKeepsPartitionedCookies reports whether the widget frame can store a cookie in the only
// form a third-party frame is still allowed one. Written and read back inside one evaluate, for the
// reason assertPartitioningEnforced gives
func frameKeepsPartitionedCookies(t *testing.T, page playwright.Page) bool {
	t.Helper()

	kept, err := page.FrameLocator("#remark42 iframe").Locator("body").Evaluate(`() => {
		document.cookie = 'e2eprobe=1; Path=/; Secure; SameSite=None; Partitioned';
		return document.cookie.split(';').some((c) => c.trim().split('=')[0] === 'e2eprobe');
	}`, nil)
	require.NoError(t, err)
	return kept == true
}

// loopbackProxy starts an http proxy that answers CONNECT for any name by connecting to the same port on
// 127.0.0.1, and returns its address. It is how an engine without --host-resolver-rules reaches the
// services of this stack under the names they are configured with
func loopbackProxy(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodConnect {
				http.Error(w, "this proxy only tunnels", http.StatusMethodNotAllowed)
				return
			}
			_, port, err := net.SplitHostPort(r.Host)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			upstream, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", port), 5*time.Second)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			defer upstream.Close()

			client, _, err := http.NewResponseController(w).Hijack()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer client.Close()

			if _, err := client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
				return
			}
			// either side closing ends the tunnel: the first copy to return closes both through the
			// defers, which unblocks the other
			done := make(chan struct{}, 2)
			go func() { _, _ = io.Copy(upstream, client); done <- struct{}{} }()
			go func() { _, _ = io.Copy(client, upstream); done <- struct{}{} }()
			<-done
		}),
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	return "http://" + ln.Addr().String()
}
