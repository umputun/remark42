//go:build e2e

// Package e2e drives the remark42 widget in a real browser through playwright.
//
// The suite talks to the stack in compose-e2e-test.yml at the repository root and brings it
// up itself when nothing is listening. Files:
//
//   - e2e_test.go: TestMain, shared helpers, constants
//   - harness_test.go: the stale-stack guard and the browser failures no assertion covers
//   - auth_test.go: dev, anonymous and email sign-in
//   - comment_test.go: post, reply, edit, delete
//   - vote_test.go: voting and its failure path
//   - thread_test.go: sorting and collapse persistence
//   - iframe_test.go: the iframe's color scheme and reveal handshake
//   - geometry_test.go: the height the widget reports and the parent applies
//   - embed_test.go: the surface the host page holds, placeholder to destroy
//   - config_test.go: the remark_config surface an integrator sets
//   - crossorigin_test.go: a host page on an origin the widget is not served from
//   - https_test.go: the widget over TLS, where the browser's protocol gates apply
//   - deployment_test.go: the instances whose configuration is the thing under test
//   - subscribe_test.go: the email subscription round trip
//   - hostframe_test.go: sender checks on both sides of the iframe boundary
//   - profile_test.go: the reader's own-comment overlay
//   - webfiles_test.go: the published /web surface
//   - telegram_test.go: Telegram authentication through the bot API stub
//   - telegramsub_test.go: the Telegram notification subscription round trip
//   - widgets_test.go: last-comments, counter and the profile iframe
package e2e

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
)

const (
	// what the browser asks for. these have to be names, not 127.0.0.1: the dev oauth2
	// server binds whatever host it reads out of REMARK_URL, and a loopback bind inside the
	// container cannot be published, so the browser resolves the names back with
	// --host-resolver-rules and reaches the published ports
	baseURL      = "http://remark42:8080"
	shortEditURL = "http://remark42-shortedit:8081"
	// the deployment modes that cannot be a setting on an instance shared with anything else,
	// each covering a regression the default configuration cannot reach. see compose-e2e-test.yml
	adminEditURL = "http://remark42-adminedit:8082"
	jwtHeaderURL = "http://remark42-jwtheader:8083"
	noAuthURL    = "http://remark42-noauth:8085"
	anonVoteURL  = "http://remark42-anonvote:8086"

	// telegram auth against a stub standing in for the bot api, see compose-e2e-test.yml
	telegramURL = "http://remark42-telegram:8087"

	// a page on an origin the widget is not served from, see compose-e2e-test.yml
	hostSiteURL = "http://host-site:8090"

	// the host page over TLS, which is the only way to reach what the browser gates on the page
	// protocol. the certificate is self-signed, so every context passes IgnoreHTTPSErrors
	httpsHostSiteURL = "https://host-site-https:8444"

	// what this process asks for, since it is not the browser and has no resolver rules
	probeURL          = "http://127.0.0.1:8080"
	shortEditProbeURL = "http://127.0.0.1:8081"
	adminEditProbeURL = "http://127.0.0.1:8082"
	jwtHeaderProbeURL = "http://127.0.0.1:8083"
	noAuthProbeURL    = "http://127.0.0.1:8085"
	anonVoteProbeURL  = "http://127.0.0.1:8086"
	telegramProbeURL  = "http://127.0.0.1:8087"
	// the stub is driven from this process because the step it stands in for happens inside
	// Telegram, where no page can reach
	telegramStubURL   = "http://127.0.0.1:8091"
	hostSiteProbeURL  = "http://127.0.0.1:8090"
	httpsProbeURL     = "https://127.0.0.1:8443"
	httpsHostProbeURL = "https://127.0.0.1:8444"
	mailpitURL        = "http://127.0.0.1:8025"

	composeFile = "../compose-e2e-test.yml"

	// every comment form carries this label, whatever mode it is in
	commentFormSel = `form[aria-label="New comment"]`

	// traces of failed tests land here; CI uploads the directory
	traceDir = "traces"

	// generous because CI runners are slower and less predictable than a laptop
	waitTimeout = 15 * time.Second

	// how long assertSignedIn waits before nudging the widget into re-reading auth state. short
	// enough that a refused status read costs a fraction of a test instead of a whole timeout,
	// long enough that the ordinary path never takes the nudge
	authRepaintWait = 3 * time.Second

	// what a single locator call inside a poll body gets. playwright's own default is 30s,
	// twice the budget of the loops here, so without this a missing element would block one
	// attempt for longer than the whole loop and then report the wrong thing
	pollTimeout = time.Second
)

var (
	pw           *playwright.Playwright
	browser      playwright.Browser
	startedStack bool

	// engines beyond chromium, launched on demand for the rendering tests
	extraBrowsers   = map[string]playwright.Browser{}
	extraBrowsersMu sync.Mutex

	// distinguishes this run's threads from those a previous run left in the database.
	// E2E_RUN_ID pins it, so the urls a CI run works on name that run and not the moment
	// the process started
	runID = firstNonEmpty(os.Getenv("E2E_RUN_ID"), fmt.Sprintf("%d", time.Now().UnixNano()))

	contextSeq atomic.Int64

	// Intercept only traffic going to a Remark42 instance. Routing every resource disables the
	// browser cache and puts unrelated host-page traffic through Playwright's driver pipe, which
	// distorts the iframe timing cases this suite measures.
	readerIPRoutePattern = func() *regexp.Regexp {
		// the dots in an address are escaped, so they cannot stand for any character
		quoted := make([]string, 0, len(readerIPAuthorities))
		for _, authority := range readerIPAuthorities {
			quoted = append(quoted, regexp.QuoteMeta(authority))
		}
		return regexp.MustCompile(`^https?://(?:` + strings.Join(quoted, "|") + `)(?:/|$)`)
	}()

	// the default client has no timeout, so a port that accepts and then stalls would block
	// a probe well past its own deadline and leave TestMain looking hung
	probeClient = &http.Client{
		Timeout: 5 * time.Second,
		// the https services in the stack are self-signed, and this client only ever talks to
		// them, so refusing the certificate would only stop the readiness probe
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, //nolint:gosec // test stack
	}
)

// Everything under /auth/ is a token bucket refilling twice a second, keyed by instance, client IP
// and request path, with a burst equal to the rate. Each browser context has its own forwarded IP,
// so one refill interval is enough between repeat requests by one reader to one path, and two
// paths never compete.
//
// The sleep is unconditional at every call site. A fresh context starts with a full bucket and
// needs none of it, so most of these are paid for nothing; the ones that earn it are the repeated
// requests inside a single context, and the probeClient calls this process makes to /auth/.
func pauseForAuthLimit() {
	time.Sleep(600 * time.Millisecond)
}

func TestMain(m *testing.M) {
	if err := ensureStack(); err != nil {
		log.Printf("[ERROR] stack not ready: %v", err)
		teardown(1)
	}

	if err := playwright.Install(installOpts("chromium")); err != nil {
		log.Printf("[ERROR] failed to install playwright: %v", err)
		teardown(1)
	}

	var err error
	if pw, err = playwright.Run(); err != nil {
		log.Printf("[ERROR] failed to start playwright: %v", err)
		teardown(1)
	}

	headless := os.Getenv("E2E_HEADLESS") != "false"
	var slowMo float64
	if !headless {
		slowMo = 50 // slow the visible browser down enough to follow
	}
	browser, err = pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
		SlowMo:   playwright.Float(slowMo),
		Args: []string{
			"--host-resolver-rules=MAP remark42 127.0.0.1, MAP remark42-shortedit 127.0.0.1, " +
				"MAP remark42-adminedit 127.0.0.1, MAP remark42-jwtheader 127.0.0.1, " +
				"MAP remark42-noauth 127.0.0.1, MAP remark42-anonvote 127.0.0.1, " +
				"MAP remark42-telegram 127.0.0.1, " +
				"MAP host-site 127.0.0.1, " +
				"MAP remark42-https 127.0.0.1, MAP host-site-https 127.0.0.1",
		},
	})
	if err != nil {
		_ = pw.Stop()
		log.Printf("[ERROR] failed to launch browser: %v", err)
		teardown(1)
	}

	code := m.Run()

	for _, b := range extraBrowsers {
		_ = b.Close()
	}
	_ = browser.Close()
	_ = pw.Stop()
	teardown(code)
}

// ensureStack waits for a running stack and starts one with compose when there is none
func ensureStack() error {
	// the digest of the sources the image is built from, exported so compose stamps the build
	// with it. read back off the running stack below, since answering on the right ports says
	// nothing about which checkout the code came from
	stamp, err := sourceStamp()
	if err != nil {
		return err
	}
	if err := os.Setenv(stampEnv, stamp); err != nil {
		return fmt.Errorf("exporting %s: %w", stampEnv, err)
	}

	// every service, not just the first: compose-dev-backend.yml and `make rundev` also
	// publish 8080, and adopting one of those would run the suite against a developer's own
	// database and fail later as unexplained locator timeouts
	if stackReady(2 * time.Second) {
		return assertStackMatches(stamp, true)
	}

	// the https services will not start without one, and compose cannot make it itself
	if out, gerr := exec.Command("./tls/generate.sh").CombinedOutput(); gerr != nil {
		return fmt.Errorf("generating the test certificate: %w\n%s", gerr, out)
	}

	log.Printf("[INFO] no complete stack on 127.0.0.1, bringing one up from %s", composeFile)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// set before running, not after: a failed `up` leaves containers behind, and teardown
	// has to know to remove them
	startedStack = true

	// --build matters: the image tag this compose file uses is the same one the dev compose
	// files produce, so without it the suite can quietly test an image from another checkout
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile,
		"up", "-d", "--build", "--quiet-pull", "--wait")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker compose up: %w\n%s", err, out)
	}
	if !stackReady(waitTimeout) {
		return fmt.Errorf("compose reported the stack healthy but it does not answer")
	}
	// checked after our own build too, so a stamp that never reaches the image is a failure
	// here, and not a guard that silently passes everything for the rest of its life
	return assertStackMatches(stamp, false)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// stackReady reports whether every service the suite needs answers
func stackReady(timeout time.Duration) bool {
	for _, url := range []string{
		probeURL + "/ping",
		shortEditProbeURL + "/ping",
		adminEditProbeURL + "/ping",
		jwtHeaderProbeURL + "/ping",
		noAuthProbeURL + "/ping",
		anonVoteProbeURL + "/ping",
		telegramProbeURL + "/ping",
		telegramStubURL + "/botstub-token/getMe",
		hostSiteProbeURL + "/post.html",
		httpsProbeURL + "/ping",
		httpsHostProbeURL + "/post-https.html",
		mailpitURL + "/api/v1/messages",
	} {
		if err := serverReady(url, timeout); err != nil {
			return false
		}
	}
	return true
}

func teardown(code int) {
	if startedStack && os.Getenv("E2E_KEEP") == "" {
		composeDown()
	}
	os.Exit(code)
}

// composeDown is separate so its context is closed before teardown calls os.Exit. it is
// bounded because it runs after m.Run, where `go test -timeout` can no longer rescue a
// wedged daemon and the binary would otherwise hang until the CI job times out
func composeDown() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile, "down", "-v")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[WARN] docker compose down: %v\n%s", err, out)
	}
}

func serverReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := probeClient.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("%s not ready after %v", url, timeout)
}

// newPage opens an isolated context, so cookies and storage never leak between tests. the
// context records a trace, kept only when the test fails, since a CI-only failure otherwise
// leaves nothing to look at from the browser side
func newPage(t *testing.T) playwright.Page {
	t.Helper()
	return newPageOn(t, browser)
}

func newPageOn(t *testing.T, b playwright.Browser) playwright.Page {
	t.Helper()
	return newPageInContext(t, b, playwright.BrowserNewContextOptions{})
}

// newPageInContext is newPageOn with the context configured, for the cases where what the browser
// says about itself is the thing under test
func newPageInContext(t *testing.T, b playwright.Browser, opts playwright.BrowserNewContextOptions) playwright.Page {
	t.Helper()
	seq := contextSeq.Add(1)

	// the https services carry a self-signed certificate, and a context that refuses it cannot
	// reach them at all. harmless for the http ones
	opts.IgnoreHttpsErrors = playwright.Bool(true)

	ctx, err := b.NewContext(opts)
	require.NoError(t, err)

	// Inject the reader address after the browser has made its CORS decision. Extra HTTP headers
	// would turn cross-origin script loads into preflighted requests, while routing at the transport
	// boundary leaves their browser-visible shape unchanged.
	// RealIP deliberately rejects special-use ranges. These globally routable-form values stay
	// inside the loopback-only stack and give more than enough distinct readers for one process.
	readerIP := forwardedReaderIP(0, seq)
	require.NoError(t, ctx.Route(readerIPRoutePattern, func(route playwright.Route) {
		continueRequest := func(options ...playwright.RouteContinueOptions) {
			if routeErr := route.Continue(options...); routeErr != nil &&
				!strings.Contains(strings.ToLower(routeErr.Error()), "target closed") {
				t.Errorf("continue routed request: %v", routeErr)
			}
		}
		if !requestNeedsReaderIP(route.Request().URL()) {
			continueRequest()
			return
		}
		headers := route.Request().Headers()
		headers["X-Real-IP"] = readerIP
		continueRequest(playwright.RouteContinueOptions{Headers: headers})
	}))

	// the reveal timers start when the iframe element is created, so the tests that bound
	// them have to measure from there and not from anything this process can time
	require.NoError(t, ctx.AddInitScript(playwright.Script{Content: playwright.String(iframeMarkScript)}))

	tracing := ctx.Tracing().Start(playwright.TracingStartOptions{
		Screenshots: playwright.Bool(true),
		Snapshots:   playwright.Bool(true),
	}) == nil
	// a test that opens two contexts would otherwise have them write the same file, and
	// cleanup runs last-in-first-out, so the surviving trace would be of the page that was
	// only setting the scenario up
	t.Cleanup(func() {
		if tracing {
			if !t.Failed() {
				_ = ctx.Tracing().Stop()
				_ = ctx.Close()
				return
			}
			// say so instead of swallowing it: this runs only on a test that already failed,
			// and a silently missing trace is what the reader goes looking for
			// the pid keeps two processes sharing a run id, and so a trace directory, from
			// overwriting each other: the counter restarts with every process
			name := strings.ReplaceAll(t.Name(), "/", "-")
			path := filepath.Join(traceDir, fmt.Sprintf("%s-%d-%d.zip", name, os.Getpid(), seq))
			// the driver creates the parent of the trace path too, but nothing here states or
			// tests that, so the suite makes the directory itself
			if derr := os.MkdirAll(traceDir, 0o750); derr != nil {
				t.Logf("could not create %s: %v", traceDir, derr)
			}
			if serr := ctx.Tracing().Stop(path); serr != nil {
				t.Logf("could not write the trace to %s: %v", path, serr)
			}
		}
		_ = ctx.Close()
	})

	page, err := ctx.NewPage()
	require.NoError(t, err)
	// before the first navigation, so nothing the page reports on its way up is missed
	watchPage(t, page)
	debug := os.Getenv("E2E_DEBUG") != ""
	page.OnResponse(func(r playwright.Response) {
		switch {
		case r.Status() == http.StatusTooManyRequests:
			// the rate limiter answers the widget, not the test, so without this the failure
			// arrives as an unexplained locator timeout. recorded as well as logged: a lost
			// /auth/status renders as signed out, and every sign-in assertion then fails
			// somewhere that does not name the cause
			recordPageIssue(page, "rate limited", r.URL())
			log.Printf("[WARN] rate limited: %s", r.URL())
		case debug && r.Status() >= 400:
			body, _ := r.Text()
			log.Printf("[DEBUG] HTTP %d %s %s", r.Status(), r.URL(), body)
		}
	})
	return page
}

// forwardedReaderIP returns a globally routable-form address that stays inside the loopback-only
// stack. The second octet separates browser traffic from harness probes, while seq separates the
// limiter, vote-deduplication and anonymous-reader state of each browser context.
func forwardedReaderIP(second byte, seq int64) string {
	const addressSpace = int64(256 * 254)
	// Different test processes may share a kept stack. Spacing each process's first slot widely
	// apart keeps their short context sequences from reusing one limiter and voter identity.
	slot := (int64(os.Getpid())*7919 + seq - 1) % addressSpace
	return fmt.Sprintf("8.%d.%d.%d", second, slot/254, slot%254+1)
}

// readerIPAuthorities lists every host and port through which a browser can reach a remark42
// instance, by name inside the compose network and over the loopback the demo pages are built on.
// Host pages and Mailpit are absent on purpose: they must keep their ordinary transport identity.
var readerIPAuthorities = []string{
	"remark42:8080",
	"remark42:8084", // the dev oauth2 provider, whose port the provider fixes
	"remark42-shortedit:8081",
	"remark42-adminedit:8082",
	"remark42-jwtheader:8083",
	"remark42-noauth:8085",
	"remark42-anonvote:8086",
	"remark42-telegram:8087",
	"remark42-https:8443",
	"127.0.0.1:8080",
	"127.0.0.1:8081",
	"127.0.0.1:8082",
	"127.0.0.1:8083",
	"127.0.0.1:8085",
	"127.0.0.1:8086",
	"127.0.0.1:8087",
	"127.0.0.1:8443",
}

// requestNeedsReaderIP answers whether a request is going to a remark42 instance. The port is part
// of the answer: an address is matched whole, so a host reached on a port the stack does not serve
// is not one of ours.
func requestNeedsReaderIP(rawURL string) bool {
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return slices.Contains(readerIPAuthorities, u.Host)
}

// installOpts asks for the browser system libraries on CI only: install-deps shells out to
// apt with sudo, which is right for a runner and wrong for someone's laptop
func installOpts(browsers ...string) *playwright.RunOptions {
	return &playwright.RunOptions{
		Browsers: browsers,
		WithDeps: os.Getenv("CI") != "",
	}
}

// engines lists the browsers the rendering tests run in. hiding the frame until its
// document reports itself inited guards against a flash that was a WebKit one, so chromium
// alone would test the engine that never had the bug. E2E_BROWSERS narrows the set
func engines() []string {
	if v := os.Getenv("E2E_BROWSERS"); v != "" {
		return strings.Split(v, ",")
	}
	return []string{"chromium", "firefox", "webkit"}
}

// browserFor launches an engine once and keeps it for the rest of the run
func browserFor(t *testing.T, name string) playwright.Browser {
	t.Helper()
	if name == "chromium" {
		return browser
	}

	extraBrowsersMu.Lock()
	defer extraBrowsersMu.Unlock()
	if b, ok := extraBrowsers[name]; ok {
		return b
	}

	require.NoError(t, playwright.Install(installOpts(name)))

	var bt playwright.BrowserType
	switch name {
	case "firefox":
		bt = pw.Firefox
	case "webkit":
		bt = pw.WebKit
	default:
		t.Fatalf("unknown browser %q", name)
	}

	b, err := bt.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(os.Getenv("E2E_HEADLESS") != "false"),
	})
	require.NoError(t, err)
	extraBrowsers[name] = b
	return b
}

// renderURL is the thread url for the rendering tests. it uses the address directly rather
// than the mapped hostname, because --host-resolver-rules is a chromium flag and these tests
// need no dev oauth2, which is the only reason the hostname exists
func renderURL(t *testing.T) string {
	t.Helper()
	return threadURLOn(t, probeURL)
}

// threadURL gives each test its own comment thread. the demo page passes
// window.location.href as remark_config.url, and remark42 keys comments by it, so a unique
// query string isolates a test without resetting the database between runs
func threadURL(t *testing.T) string {
	t.Helper()
	return threadURLOn(t, baseURL)
}

// threadURLOn is threadURL against a given instance, for the tests that do not use the
// default one. every thread url goes through here so the rewriting below cannot be missed
func threadURLOn(t *testing.T, base string) string {
	t.Helper()
	// underscores are left in on purpose: collapse persistence keys off the page url, and a
	// url carrying the separator its storage used to join on is the case that broke it. only
	// the "/" of a subtest name is dropped, having no business in a query value
	name := strings.ReplaceAll(strings.ToLower(t.Name()), "/", "-")
	return fmt.Sprintf("%s/web/?e2e=%s-%s", base, name, runID)
}

// anonName builds a username no earlier run has used. Blocking a user and verifying one are
// properties of the user, and the stack's database outlives the run, so a name that comes back
// arrives already blocked or already verified: the case then toggles the state off and waits for
// something that is being taken away.
//
// The run id alone is not enough, since E2E_RUN_ID pins it and a second run against a surviving
// stack repeats it, which is exactly what `make e2e-up` invites. The pid distinguishes those
// while leaving CI, where every job is one process, with the run id it wants in the name.
//
// Everything outside [\p{L}\d\s_] is dropped, because the form validates its input against that
// pattern (auth.tsx:332) and the browser refuses to submit anything else: no request is made, and
// the test waits out its timeout on a panel that was never going to change. E2E_RUN_ID is
// "<run id>-<attempt>" on CI, whose hyphen is exactly such a character
func anonName(prefix string) string {
	var b strings.Builder
	b.WriteString(prefix)
	for _, r := range runID {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		}
	}
	fmt.Fprintf(&b, "_%d", os.Getpid())
	return b.String()
}

// openThread loads the demo page on this test's own thread and waits for the widget
func openThread(t *testing.T, page playwright.Page) playwright.FrameLocator {
	t.Helper()
	return openURL(t, page, threadURL(t))
}

func openURL(t *testing.T, page playwright.Page, url string) playwright.FrameLocator {
	t.Helper()
	_, err := page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	require.NoError(t, err)
	return widget(t, page)
}

// embedConfig loads a page carrying a remark_config of the test's choosing and runs the embed
// script against it.
//
// The demo page cannot be used for this: it writes its own remark_config, so anything a test
// needs to vary has to go into a page that carries no widget of its own. privacy.html is served
// by the instance and has none, which also keeps the widget on its own origin, where its CSP
// allows the chunks it loads. host, site_id and url are filled in when the caller leaves them out
// placeholder, when given, is markup left inside the root before the widget runs, which the
// documentation promises is cleared once the iframe reports itself inited
func embedConfig(t *testing.T, page playwright.Page, config map[string]any, placeholder ...string) {
	t.Helper()
	embedConfigOn(t, page, "/web/privacy.html", config, placeholder...)
}

// embedConfigOn is embedConfig on a chosen host page, for the cases where the page's own address
// is part of what is under test
func embedConfigOn(t *testing.T, page playwright.Page, hostPage string, config map[string]any, placeholder ...string) {
	t.Helper()

	for key, value := range map[string]any{"host": baseURL, "site_id": "remark", "url": threadURL(t)} {
		if _, ok := config[key]; !ok {
			config[key] = value
		}
	}

	_, err := page.Goto(baseURL + hostPage)
	require.NoError(t, err)

	_, err = page.Evaluate(`([config, placeholder]) => {
		window.remark_config = config;
		const node = document.createElement('div');
		node.id = 'remark42';
		node.innerHTML = placeholder;
		document.body.appendChild(node);
	}`, []any{config, strings.Join(placeholder, "")})
	require.NoError(t, err)

	_, err = page.AddScriptTag(playwright.PageAddScriptTagOptions{URL: playwright.String(baseURL + "/web/embed.mjs")})
	require.NoError(t, err)
}

// stubSignedOut keeps a case whose subject is outside authentication independent of the auth
// deployment. Cases that exercise identity use the endpoint itself.
func stubSignedOut(t *testing.T, page playwright.Page) {
	t.Helper()
	require.NoError(t, page.Route("**/auth/status**", func(route playwright.Route) {
		if err := route.Fulfill(playwright.RouteFulfillOptions{
			Status:      playwright.Int(http.StatusOK),
			ContentType: playwright.String("application/json"),
			Body:        playwright.String(`{"status":"not logged in"}`),
		}); err != nil {
			t.Errorf("fulfill signed-out auth status: %v", err)
		}
	}))
}

// reload re-navigates and returns the widget once the thread itself has loaded.
//
// waiting for the comment form is not enough here: root.tsx renders it as soon as the user
// fetch resolves, while the thread keeps its own preloader until /find answers. an assertion
// that a comment is absent would otherwise pass against a thread that has not rendered yet
func reload(t *testing.T, page playwright.Page) playwright.FrameLocator {
	t.Helper()
	pauseForAuthLimit()
	_, err := page.ExpectResponse("**/api/v1/find**", func() error {
		_, rerr := page.Reload()
		return rerr
	}, playwright.PageExpectResponseOptions{Timeout: playwright.Float(float64(waitTimeout.Milliseconds()))})
	require.NoError(t, err)

	frame := widget(t, page)
	// the response has arrived; give preact the frame it needs to swap the preloader out
	waitHidden(t, frame.Locator(`[role="list"] .preloader`))
	return frame
}

// postComment types into the form the widget is currently showing and waits for the comment
// to appear in the thread
func postComment(t *testing.T, frame playwright.FrameLocator, text string) {
	t.Helper()
	postCommentMatching(t, frame, text, text)
}

// postCommentMatching posts source and waits for a comment carrying marker. the two differ
// whenever the source has markdown in it, since the thread shows the rendered result
func postCommentMatching(t *testing.T, frame playwright.FrameLocator, source, marker string) playwright.Locator {
	t.Helper()
	submitForm(t, frame.Locator(commentFormSel).First(), source)

	posted := comment(frame, marker)
	waitVisible(t, posted)
	return posted
}

// submitForm fills a comment form and submits it. the submit label is Send, Reply or Save
// depending on the form's mode, so match on the type and not the text
func submitForm(t *testing.T, form playwright.Locator, text string) {
	t.Helper()
	require.NoError(t, form.Locator("textarea").Fill(text))
	require.NoError(t, form.Locator(`button[type="submit"]`).Click())
}

// comment locates the comment carrying the text.
//
// only sound for asserting that something IS there. comments render through an
// IntersectionObserver (components/root/in-view), so one below the fold is an empty
// placeholder article and no text filter matches it: an absence assertion written this way
// passes whether the comment is gone or merely off screen. use articleCount for absence
func comment(frame playwright.FrameLocator, text string) playwright.Locator {
	return frame.Locator("article", playwright.FrameLocatorLocatorOptions{HasText: text})
}

// articleCount counts the comments in the thread, including any the viewport has not reached
// yet, which still occupy an empty article element.
//
// it is an absence oracle only after a reload. before one the widget keeps a deleted
// comment's node and only swaps its text, and the backend prunes a deleted comment from the
// tree only when it has no replies (store/service/tree.go)
func articleCount(t *testing.T, frame playwright.FrameLocator) int {
	t.Helper()
	n, err := frame.Locator("article").Count()
	require.NoError(t, err)
	return n
}

// replyForm returns the form the widget opened last, which is the reply or edit form.
//
// it waits for a second form to exist first: playwright auto-waits for the element a locator
// resolves to, not for a better one to appear, so filling .Last() too early would type into
// the top-level form and post a root comment instead of a reply
func replyForm(t *testing.T, frame playwright.FrameLocator) playwright.Locator {
	t.Helper()
	eventually(t, waitTimeout, "the reply or edit form did not open", func() bool {
		n, err := frame.Locator(commentFormSel).Count()
		return err == nil && n > 1
	})
	return frame.Locator(commentFormSel).Last()
}

// actions returns the action bar of the comment carrying the text
func actions(frame playwright.FrameLocator, text string) playwright.Locator {
	return comment(frame, text).Locator(".comment-actions").First()
}

// widget returns the widget's iframe once its own content has rendered. it waits on the
// comment form and not the auth panel, because .auth is only present while signed out
func widget(t *testing.T, page playwright.Page) playwright.FrameLocator {
	t.Helper()
	frame := page.FrameLocator("#remark42 iframe")
	waitVisible(t, frame.Locator(commentFormSel).First())
	return frame
}

func waitVisible(t *testing.T, loc playwright.Locator) {
	t.Helper()
	require.NoError(t, loc.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(waitTimeout.Milliseconds())),
	}))
}

// waitHidden waits for the element to go away. msgAndArgs says what its staying would mean,
// since the failure otherwise names a selector and not the behavior that broke
func waitHidden(t *testing.T, loc playwright.Locator, msgAndArgs ...any) {
	t.Helper()
	err := loc.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateHidden,
		Timeout: playwright.Float(float64(waitTimeout.Milliseconds())),
	})
	require.NoError(t, err, msgAndArgs...)
}

// pollText reads an element's text with a timeout short enough to be used inside eventually
func pollText(loc playwright.Locator) (string, error) {
	return loc.InnerText(playwright.LocatorInnerTextOptions{
		Timeout: playwright.Float(float64(pollTimeout.Milliseconds())),
	})
}

// pollAttr reads an attribute with the same short timeout
func pollAttr(loc playwright.Locator, name string) (string, error) {
	return loc.GetAttribute(name, playwright.LocatorGetAttributeOptions{
		Timeout: playwright.Float(float64(pollTimeout.Milliseconds())),
	})
}

// eventually polls fn until it returns true, for assertions playwright's own auto-waiting
// does not cover, such as a value read out of the page with Evaluate
func eventually(t *testing.T, timeout time.Duration, msg string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("condition not met within %v: %s", timeout, msg)
}

// pageFetch issues a request from the page's own session, so it carries the browser's cookies
// and the XSRF header the widget's own calls carry. body may be nil for a bodyless request
func pageFetch(t *testing.T, page playwright.Page, method, url string, body any) (status int, respBody string) {
	t.Helper()

	payload := ""
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		payload = string(raw)
	}

	res, err := page.Evaluate(`async ({method, url, payload}) => {
		const xsrf = document.cookie.split('; ').find((c) => c.startsWith('XSRF-TOKEN='));
		const resp = await fetch(url, {
			method,
			credentials: 'include',
			headers: {
				'Content-Type': 'application/json',
				'X-XSRF-TOKEN': xsrf ? xsrf.slice(xsrf.indexOf('=') + 1) : '',
			},
			body: payload === '' ? undefined : payload,
		});
		return {status: resp.status, body: await resp.text()};
	}`, map[string]any{"method": method, "url": url, "payload": payload})
	require.NoError(t, err)

	out, ok := res.(map[string]any)
	require.True(t, ok, "unexpected shape from the page: %#v", res)
	switch v := out["status"].(type) {
	case int:
		status = v
	case float64:
		status = int(v)
	default:
		t.Fatalf("unexpected status type %T in %#v", v, out)
	}
	respBody, _ = out["body"].(string)
	return status, respBody
}

// mailpitMessage returns the newest message sent to the address, with its body
func mailpitMessage(t *testing.T, to string) string {
	t.Helper()

	type item struct {
		ID string `json:"ID"`
		To []struct {
			Address string `json:"Address"`
		} `json:"To"`
	}
	var found string
	eventually(t, waitTimeout, "no message for "+to, func() bool {
		resp, err := probeClient.Get(mailpitURL + "/api/v1/messages?limit=50")
		if err != nil {
			return false
		}
		defer func() { _ = resp.Body.Close() }()

		var list struct {
			Messages []item `json:"messages"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
			return false
		}
		for _, msg := range list.Messages {
			for _, addr := range msg.To {
				if strings.EqualFold(addr.Address, to) {
					found = msg.ID
					return true
				}
			}
		}
		return false
	})

	resp, err := probeClient.Get(fmt.Sprintf("%s/api/v1/message/%s", mailpitURL, found))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var body struct {
		Text string `json:"Text"`
		HTML string `json:"HTML"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return body.Text + body.HTML
}
