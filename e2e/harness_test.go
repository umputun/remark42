//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stampEnv carries the source digest into compose, which passes it to the image build as
// GITHUB_SHA and so into the org.opencontainers.image.revision label. stamp.sh computes it and
// the Makefile exports it too, so a stack started by hand is stamped exactly as one the suite
// starts itself
const stampEnv = "E2E_STAMP"

// revisionLabel is where the Dockerfile puts the stamp
const revisionLabel = "org.opencontainers.image.revision"

// sourceStamp digests the sources that end up in the image. Shelling out rather than
// reimplementing it keeps one definition of what the digest covers, since the Makefile needs
// the same value and cannot call into this package
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

// stackStamp is what the running stack was built from, read off the image the container runs
func stackStamp() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dockerTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "docker", "inspect",
		"--format", "{{index .Config.Labels "+fmt.Sprintf("%q", revisionLabel)+"}}", stackContainer).Output()
	if err != nil {
		return "", fmt.Errorf("reading the stamp off %s: %w: %s", stackContainer, err, exitStderr(err))
	}
	return strings.TrimSpace(string(out)), nil
}

// assertStackMatches refuses a stack built from other sources than the ones under test.
//
// Every checkout shares the image tag the compose file names, so a stack brought up from
// another worktree, or from this one before an edit, answers on the same ports and passes every
// readiness probe while serving code nobody is looking at. adopted says whether the suite found
// the stack rather than starting it, which is the only case a mismatch is expected in:
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
		return fmt.Errorf("the stack was just built from %s but carries %q, so %s is not reaching the image build",
			want, got, stampEnv)
	}
	return fmt.Errorf("the stack answering on 127.0.0.1 was built from %q, not from the sources here (%s). "+
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
	mu      sync.Mutex
	issues  []pageIssue
	allowed []string
}

func (w *pageWatch) record(kind, text string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.issues = append(w.issues, pageIssue{kind: kind, text: text})
}

func (w *pageWatch) allow(subs ...string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.allowed = append(w.allowed, subs...)
}

// unexpected is everything recorded that no allowance covers
func (w *pageWatch) unexpected() []pageIssue {
	w.mu.Lock()
	defer w.mu.Unlock()

	var out []pageIssue
	for _, issue := range w.issues {
		if !anySubstring(w.allowed, issue.text) {
			out = append(out, issue)
		}
	}
	return out
}

func anySubstring(subs []string, text string) bool {
	for _, sub := range subs {
		if strings.Contains(text, sub) {
			return true
		}
	}
	return false
}

var (
	watchesMu sync.Mutex
	watches   = map[playwright.Page]*pageWatch{}
)

// watchPage starts collecting the page's own failure reports and fails the test at cleanup for
// any the test did not declare. Uncaught exceptions are the ones worth the machinery: a widget
// that throws while rendering still leaves most of this suite green
func watchPage(t *testing.T, page playwright.Page) {
	t.Helper()

	w := &pageWatch{}
	watchesMu.Lock()
	watches[page] = w
	watchesMu.Unlock()

	page.OnPageError(func(err error) { w.record("uncaught exception", err.Error()) })
	page.OnConsole(func(msg playwright.ConsoleMessage) {
		if msg.Type() == "error" {
			w.record("console error", msg.Text())
		}
	})

	t.Cleanup(func() {
		watchesMu.Lock()
		delete(watches, page)
		watchesMu.Unlock()

		for _, issue := range w.unexpected() {
			assert.Fail(t, "the browser reported a failure no assertion covers", issue.String())
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

// expectPageIssues declares failures the test is deliberately causing, by substring. Anything
// not declared fails the test, so a case that drives an error path says which one
func expectPageIssues(t *testing.T, page playwright.Page, subs ...string) {
	t.Helper()

	watchesMu.Lock()
	w, ok := watches[page]
	watchesMu.Unlock()
	require.True(t, ok, "the page has no watch, so an allowance on it would be silently useless")
	w.allow(subs...)
}
