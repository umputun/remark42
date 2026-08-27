//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stampEnv carries the source digest into compose, which sets it on the remark42 service.
// stamp.sh computes it and the Makefile exports it too, so a stack started by hand is stamped
// exactly as one the suite starts itself
const stampEnv = "E2E_STAMP"

// stampVar is what compose names it inside the container
const stampVar = "E2E_SOURCE_STAMP"

var limiterProbeSeq atomic.Int64

func TestHarness_ReaderIPRequestScope(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		rawURL string
		want   bool
	}{
		{name: "named instance", rawURL: baseURL + "/auth/status", want: true},
		{name: "dev oauth provider", rawURL: "http://remark42:8084/login/oauth/authorize", want: true},
		{name: "named deployment mode", rawURL: shortEditURL + "/auth/status", want: true},
		{name: "named tls instance", rawURL: "https://remark42-https:8443/auth/status", want: true},
		{name: "loopback instance", rawURL: probeURL + "/auth/status", want: true},
		{name: "loopback tls instance", rawURL: httpsProbeURL + "/auth/status", want: true},
		{name: "host page", rawURL: hostSiteURL + "/post.html", want: false},
		{name: "tls host page", rawURL: httpsHostSiteURL + "/post-https.html", want: false},
		{name: "mailpit", rawURL: mailpitURL + "/api/v1/messages", want: false},
		{name: "unrelated similar host", rawURL: "https://remark42.example.com/file", want: false},
		{name: "remark url in host-page query", rawURL: hostSiteURL + "/?next=" + baseURL, want: false},
		{name: "non-http scheme", rawURL: "ftp://remark42:8080/file", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, requestNeedsReaderIP(tc.rawURL), tc.rawURL)
			assert.Equal(t, tc.want, readerIPRoutePattern.MatchString(tc.rawURL), "route pattern: "+tc.rawURL)
		})
	}
}

func TestHarness_ForwardedIPSeparatesAuthLimiter(t *testing.T) {
	t.Parallel()

	type result struct {
		status int
		err    error
	}
	status := func(ip string) result {
		req, err := http.NewRequest(http.MethodGet, probeURL+"/auth/status?site=remark", http.NoBody)
		if err != nil {
			return result{err: err}
		}
		req.Header.Set("X-Real-IP", ip)

		resp, err := probeClient.Do(req)
		if err != nil {
			return result{err: err}
		}
		if err = resp.Body.Close(); err != nil {
			return result{err: err}
		}
		return result{status: resp.StatusCode}
	}

	seq := limiterProbeSeq.Add(1) - 1
	clientIP := forwardedReaderIP(255, seq*2+1)
	freshIP := forwardedReaderIP(255, seq*2+2)

	first := status(clientIP)
	require.NoError(t, first.err)
	require.Equal(t, http.StatusOK, first.status)

	const burst = 8
	results := make(chan result, burst)
	for range burst {
		go func() { results <- status(clientIP) }()
	}
	refused := false
	for range burst {
		got := <-results
		require.NoError(t, got.err)
		require.Contains(t, []int{http.StatusOK, http.StatusTooManyRequests}, got.status)
		refused = refused || got.status == http.StatusTooManyRequests
	}
	require.True(t, refused, "one forwarded IP should share one auth-limiter bucket")

	// A new forwarded IP receives its own allowance, proving that the stack configuration and
	// RealIP classification preserve the isolation every parallel browser context relies on.
	fresh := status(freshIP)
	require.NoError(t, fresh.err)
	require.Equal(t, http.StatusOK, fresh.status,
		"a fresh forwarded IP did not get its own bucket; check TRUSTED_PROXY and RealIP classification")
}

func TestHarness_BrowserContextsHaveIndependentAuthLimiters(t *testing.T) {
	t.Parallel()

	pages := []playwright.Page{newPage(t), newPage(t)}
	for _, page := range pages {
		resp, err := page.Goto(probeURL + "/web/privacy.html")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, http.StatusOK, resp.Status())
	}

	type burstResult struct {
		statuses []int
		err      error
	}
	burst := func(page playwright.Page) burstResult {
		value, err := page.Evaluate(`url => Promise.all([fetch(url), fetch(url)]).then(rs => rs.map(r => r.status))`,
			probeURL+"/auth/status?site=remark")
		if err != nil {
			return burstResult{err: err}
		}
		raw, ok := value.([]any)
		if !ok {
			return burstResult{err: fmt.Errorf("auth burst returned %T", value)}
		}
		statuses := make([]int, 0, len(raw))
		for _, item := range raw {
			switch n := item.(type) {
			case int:
				statuses = append(statuses, n)
			case float64:
				statuses = append(statuses, int(n))
			default:
				return burstResult{err: fmt.Errorf("auth burst status is %T", item)}
			}
		}
		return burstResult{statuses: statuses}
	}

	results := make(chan burstResult, len(pages))
	for _, page := range pages {
		go func() { results <- burst(page) }()
	}
	for range pages {
		got := <-results
		require.NoError(t, got.err)
		require.Equal(t, []int{http.StatusOK, http.StatusOK}, got.statuses,
			"two browser contexts shared one auth bucket, so the Playwright reader-IP route is not reaching the backend")
	}
}

// sourceStamp digests the sources that end up in the image. Shelling out keeps one definition of
// what the digest covers, since the Makefile needs the same value and cannot call into this package
func sourceStamp() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dockerTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "./stamp.sh").Output()
	if err != nil {
		return "", fmt.Errorf("computing the source stamp: %w: %s", err, exitStderr(err))
	}

	stamp := strings.TrimSpace(string(out))
	if stamp == "" {
		return "", fmt.Errorf("stamp.sh produced nothing")
	}
	return stamp, nil
}

// stackStamp is what the running stack was brought up from, read out of the container
func stackStamp() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dockerTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "docker", "exec", stackContainer, "printenv", stampVar).Output()
	if err != nil {
		return "", fmt.Errorf("reading %s out of %s: %w: %s", stampVar, stackContainer, err, exitStderr(err))
	}
	return strings.TrimSpace(string(out)), nil
}

// assertStackMatches refuses a stack built from other sources than the ones under test.
//
// Every checkout shares the image tag the compose file names, so a stack brought up from
// another worktree, or from this one before an edit, answers on the same ports and passes every
// readiness probe while serving code nobody is looking at. adopted says whether the suite found
// the stack instead of starting it, which is the only case a mismatch is expected in:
// after our own build it means the stamp never reached the image
func assertStackMatches(want string, adopted bool) error {
	got, err := stackStamp()
	if err != nil {
		return err
	}
	if got == want {
		return nil
	}
	if !adopted {
		return fmt.Errorf("the stack was just started from %s but reports %q, so %s is not reaching the container",
			want, got, stampEnv)
	}
	return fmt.Errorf("the stack answering on 127.0.0.1 was brought up from %q, not from the sources here (%s). "+
		"it belongs to another checkout or predates an edit: `make e2e-down` and run again", got, want)
}

// pageIssue is a failure the browser reported that no assertion in the test looks at
type pageIssue struct {
	kind string
	text string
}

func (i pageIssue) String() string { return i.kind + ": " + i.text }

// pageWatch collects those failures for one page. Without it a widget throwing on every load,
// or a run quietly eating rate-limit responses, shows up only as whichever assertion happens to
// depend on the damage, and plenty of them depend on none
type pageWatch struct {
	mu     sync.Mutex
	issues []pageIssue
	// console errors, which are context and not a verdict: the browser logs one for every
	// request that fails, so on the cases driving an error path they restate a status the test
	// has already asserted on, and the wording differs between engines. logged when the test
	// fails, never the reason it failed
	noted []pageIssue
}

func (w *pageWatch) record(kind, text string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.issues = append(w.issues, pageIssue{kind: kind, text: text})
}

func (w *pageWatch) note(kind, text string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.noted = append(w.noted, pageIssue{kind: kind, text: text})
}

func (w *pageWatch) notes() []pageIssue {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.noted)
}

// unexpected is everything recorded that has to fail the test
func (w *pageWatch) unexpected() []pageIssue {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.issues)
}

var (
	watchesMu sync.Mutex
	watches   = map[playwright.Page]*pageWatch{}
)

// watchPage starts collecting the page's own failure reports and fails the test at cleanup for
// any the test did not declare.
//
// Uncaught exceptions are what this is for: a widget that throws while rendering still leaves
// most of this suite green, since the assertions are about elements the browser lays out either
// way, and #1979 and #851 were both exactly that. Console errors are collected but never fail a
// test: the browser logs one for every failed request, so the cases that drive an error path
// would all have to declare a status they already assert on, and the wording is engine-specific
func watchPage(t *testing.T, page playwright.Page) {
	t.Helper()

	w := &pageWatch{}
	watchesMu.Lock()
	watches[page] = w
	watchesMu.Unlock()

	page.OnPageError(func(err error) { w.record("uncaught exception", err.Error()) })
	page.OnConsole(func(msg playwright.ConsoleMessage) {
		if msg.Type() == "error" {
			w.note("console error", msg.Text())
		}
	})

	t.Cleanup(func() {
		watchesMu.Lock()
		delete(watches, page)
		watchesMu.Unlock()

		for _, issue := range w.unexpected() {
			assert.Fail(t, "the browser reported a failure no assertion covers", issue.String())
		}
		if t.Failed() {
			for _, issue := range w.notes() {
				t.Logf("from the browser: %s", issue)
			}
		}
	})
}

// recordPageIssue files something seen outside the browser's own event handlers, such as a
// rate-limit response, against the page it happened on
func recordPageIssue(page playwright.Page, kind, text string) {
	watchesMu.Lock()
	w, ok := watches[page]
	watchesMu.Unlock()
	if ok {
		w.record(kind, text)
	}
}
