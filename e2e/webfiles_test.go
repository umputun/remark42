//go:build e2e

package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// the e2e stack's container, named in compose-e2e-test.yml. addressed directly and not
	// through `docker compose exec`, which resolves the service through a project name taken from
	// the directory the compose file sits in: run the suite from a git worktree and it looks for a
	// project named after the worktree instead of the stack that is answering
	stackContainer = "remark42-e2e"

	// what the served web root is inside that container
	stackWebRoot = "/srv/web"

	// enough for a healthy docker CLI and short enough to fail as itself. an unbounded call to a
	// wedged daemon hangs until the package timeout panics the binary, which skips TestMain's
	// teardown and leaves the stack running
	dockerTimeout = 30 * time.Second

	localesDir = "../frontend/apps/remark42/app/locales"
)

// the /web paths the documentation publishes, plus the legacy names of the same files. each is
// held by somebody outside this repository: an operator pastes privacy.html into an OAuth
// application, an nginx config proxies index.html, the integration guides start from the embed
// script, and the comment form links markdown-help.html. written out and not derived, because
// the documentation decides the list, so a name joining or leaving is an edit made on purpose
var documentedWebPaths = []struct {
	path string
	// what publishes it, so a failure names who is holding the URL
	where string
	// the content type it has to keep. nosniff is set on every response, so a bundle served as
	// text/plain is as broken as one that 404s while its bytes still compare equal
	mime string
	// something only the right file contains. without it a directory listing, or an error page
	// carrying a 200, passes for the real thing
	marker string
}{
	{"/web", "getting-started/installation, the first URL an operator opens", "text/html", `id="remark42"`},
	{"/web/", "linked from most documentation pages", "text/html", `id="remark42"`},
	{"/web/index.html", "manuals/nginx proxies this exact URL", "text/html", `id="remark42"`},
	{"/web/embed.mjs", "configuration/frontend/spa.md, manuals/subdomain, contributing/frontend", "text/javascript", "remark_config"},
	{"/web/embed.js", "the legacy name integrations still hold", "text/javascript", "remark_config"},
	{"/web/counter.mjs", "built from remark_config.components by the loader in configuration/frontend", "text/javascript", "remark42__counter"},
	{"/web/counter.js", "the legacy name the same loader built", "text/javascript", "remark42__counter"},
	{"/web/last-comments.mjs", "built from remark_config.components by the same loader", "text/javascript", "remark_config"},
	{"/web/last-comments.js", "the legacy name the same loader built", "text/javascript", "remark_config"},
	{"/web/privacy.html", "configuration/authorization, pasted into an OAuth application", "text/html", "Privacy Policy"},
	{"/web/markdown-help.html", "linked by the comment form", "text/html", "Markdown"},
	{"/web/400x400.jpeg", "embedded by markdown-help.html", "image/jpeg", ""},
}

// TestWeb_DocumentedURLsResolve pins the published surface as a compatibility contract. A page or
// an OAuth application set up years ago holds these URLs for good, and the build changing shape is
// not their problem
func TestWeb_DocumentedURLsResolve(t *testing.T) {
	for _, tc := range documentedWebPaths {
		t.Run(strings.TrimPrefix(tc.path, "/"), func(t *testing.T) {
			resp := getWeb(t, tc.path)

			assert.Equal(t, http.StatusOK, resp.status, "%s has to keep resolving: %s", tc.path, tc.where)
			assert.NotEmpty(t, resp.body, "%s resolved but served nothing", tc.path)
			assert.Contains(t, resp.mime, tc.mime, "%s serves the wrong type, and nosniff means the "+
				"browser will refuse it whatever the bytes are", tc.path)
			if tc.marker != "" {
				assert.Contains(t, resp.body, tc.marker, "%s resolved but is not the file it should be", tc.path)
			}
		})
	}
}

// TestWeb_UnknownNameIs404 is the negative control for the case above: without it a fallback
// serving one page for everything under /web would keep every assertion there green
func TestWeb_UnknownNameIs404(t *testing.T) {
	resp := getWeb(t, "/web/no-such-file.html")
	assert.Equal(t, http.StatusNotFound, resp.status, "an unknown name has to 404, or the cases "+
		"above cannot tell a served file from a fallback")
}

// TestWeb_EveryBundleServesUnderBothSuffixes covers the names the list above does not, and takes
// its input from the build and not from a list somebody maintains, so a locale joining or
// leaving needs no edit here. Whatever the bundler emitted has to answer under its legacy .js name
// with the same bytes and the same type, and has to parse as a classic script, which is the premise
// serving one under the other rests on
func TestWeb_EveryBundleServesUnderBothSuffixes(t *testing.T) {
	names := emittedBundles(t)
	// a listing from the wrong place, or one that lost most of its entries to a chunk directory,
	// would leave a handful of cases running and report green. every locale is a chunk of its own,
	// so the catalog count on disk is a floor the build cannot drop below
	require.Contains(t, names, "remark.mjs", "listing is not the served web root: %v", names)
	require.GreaterOrEqual(t, len(names), localeCount(t),
		"%d bundles is fewer than there are locales, so the listing is missing chunks: %v", len(names), names)
	t.Logf("checking %d bundles", len(names))

	page := parsePage(t)

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			legacy := strings.TrimSuffix(name, ".mjs") + ".js"

			modern, alias := getWeb(t, "/web/"+name), getWeb(t, "/web/"+legacy)
			require.NotEmpty(t, modern.body, "%s serves nothing, so everything below compares emptiness", name)

			assert.Equal(t, digest(modern.body), digest(alias.body),
				"%s and %s serve different content (%d and %d bytes), so the legacy name is not the same file",
				name, legacy, len(modern.body), len(alias.body))
			assert.Contains(t, alias.mime, "javascript",
				"%s serves the wrong type, and nosniff means the browser refuses it however right the bytes are", legacy)

			require.Empty(t, classicScriptError(t, page, modern.body),
				"%s does not parse as a classic script, so serving it as %s hands a syntax error to the "+
					"oldest integrations", name, legacy)
		})
	}
}

// webResponse is what the cases assert on: everything read from one request, since the body has to
// be consumed and closed before the next one anyway
type webResponse struct {
	status int
	mime   string
	body   string
}

// getWeb requests a path from the stack. Redirects are followed, because whoever holds a
// documented URL cares whether the page arrives, not how many hops it took: /web and
// /web/index.html both answer 301 towards the directory
func getWeb(t *testing.T, path string) webResponse {
	t.Helper()

	pauseForWebLimit()
	resp, err := probeClient.Get(probeURL + path)
	require.NoError(t, err)
	defer func() { assert.NoError(t, resp.Body.Close()) }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotEqual(t, http.StatusTooManyRequests, resp.StatusCode,
		"%s was rate limited, which is this file pacing itself wrong and not a broken URL", path)

	return webResponse{status: resp.StatusCode, mime: resp.Header.Get("Content-Type"), body: string(body)}
}

func digest(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

// emittedBundles lists the .mjs files the running stack serves, recursively, so chunks landing in
// a subdirectory stay in the set. Read from the container because the bundler runs inside the
// image build: the host has no build output, and a list from anywhere else describes another build
func emittedBundles(t *testing.T) []string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), dockerTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "docker", "exec", stackContainer,
		"find", stackWebRoot, "-name", "*.mjs").Output()
	require.NoError(t, err, "listing %s in %s: %s", stackWebRoot, stackContainer, exitStderr(err))

	var names []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if name := strings.TrimPrefix(strings.TrimSpace(line), stackWebRoot+"/"); name != "" {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// localeCount is how many message catalogs the app carries, each of which the bundler emits as
// its own chunk
func localeCount(t *testing.T) int {
	t.Helper()

	entries, err := os.ReadDir(localesDir)
	require.NoError(t, err, "reading %s", localesDir)

	count := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			count++
		}
	}
	require.NotZero(t, count, "no catalogs in %s, so the floor below would be meaningless", localesDir)
	return count
}

// exitStderr is what the command wrote to stderr, which Output keeps out of the parsed result
func exitStderr(err error) string {
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return string(exit.Stderr)
	}
	return ""
}

// parsePage is a browser page for the parse probe alone. Not newPage: that one records a trace on
// failure, and a trace of a page which never navigates sends the reader to an empty recording
func parsePage(t *testing.T) playwright.Page {
	t.Helper()

	page, err := browser.NewPage()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, page.Close()) })
	return page
}

// classicScriptError returns the parse error a classic script parser gives for src, and an empty
// string when there is none. new Function parses its argument as a function body, which rejects a
// top level import, export or import.meta exactly as a classic script does, and runs nothing. The
// grammars are not identical, a function body also accepting return and new.target, but only in
// the direction that lets more through, so the probe never fails a bundle a browser would take
func classicScriptError(t *testing.T, page playwright.Page, src string) string {
	t.Helper()

	v, err := page.Evaluate(`src => {
		try { new Function(src); return ""; }
		catch (e) { return e.name === "SyntaxError" ? String(e) : ""; }
	}`, src)
	require.NoError(t, err)

	msg, ok := v.(string)
	require.True(t, ok, "the parse probe returned %T instead of a string", v)
	return msg
}

var (
	webGate    sync.Mutex
	lastWebGet time.Time
)

// everything under /web/ is rate limited to 20 requests a second, hard coded in rest.go rather
// than settable, and the bundle case asks for two files per bundle. Without pacing this file
// manufactures the 429s it would then have to tell apart from a missing URL
func pauseForWebLimit() {
	const spacing = 55 * time.Millisecond

	webGate.Lock()
	defer webGate.Unlock()
	if wait := spacing - time.Since(lastWebGet); wait > 0 {
		time.Sleep(wait)
	}
	lastWebGet = time.Now()
}

// TestWeb_NoBundleHardcodesTheWebRoot pins the fix for #2203, which shipped with no test at any
// level. The bundler used to bake a fixed public path into every entry, so an instance mounted
// under a prefix, which manuals/subdomain documents, asked the domain root for its provider icons
// and got nothing. The path is derived from the URL the bundle was loaded from now, which is
// correct for both arrangements.
//
// What separates the two builds is the assignment webpack emits for its runtime public path: a
// literal when the path is fixed, and a computed value when it is derived. Asset filenames appear
// bare either way, so a case looking for a rooted "/web/name.svg" string finds nothing in either
// build and proves nothing; this asserts the assignment instead.
//
// Every emitted bundle is checked rather than the obvious one: the original defect put fifteen
// icons in remark.mjs and one in last-comments.mjs, so a case reading a single entry would have
// gone green with half of it still live.
func TestWeb_NoBundleHardcodesTheWebRoot(t *testing.T) {
	names := emittedBundles(t)
	require.Contains(t, names, "remark.mjs", "listing is not the served web root: %v", names)
	require.Contains(t, names, "last-comments.mjs",
		"the entry carrying the second half of #2203 is missing from the listing: %v", names)

	// webpack writes its public path to the `p` property of the runtime object. A baked-in path
	// is a string literal there; a derived one is an expression
	baked := regexp.MustCompile("\\.p\\s*=\\s*[\"'`]/web/[\"'`]")

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			body := getWeb(t, "/web/"+name).body
			require.NotEmpty(t, body, "%s serves nothing, so this asserts nothing", name)

			assert.NotRegexp(t, baked, string(body),
				"%s bakes the public path in rather than deriving it, so an instance mounted under "+
					"a prefix fetches its assets from the domain root", name)
		})
	}
}
