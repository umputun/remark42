//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Where the counters the instrumented bundle keeps are written, one file per frame that carries
// any. `e2e/coverage.sh` merges the directory and reports it; nothing here reads it back.
const jsCoverageDir = "coverage/frontend"

// The bundle is built inside the image, so every path it records is the path it had there. They
// are rewritten as the counters are read, because the merge and the report run on the host, where
// nothing resolves under that prefix.
const jsCoverageBuildRoot = "/srv/frontend/apps/remark42/"

// Counters live on the document, so a reload or a second navigation throws away everything the
// page had counted. Each snapshot is taken before that happens and numbered, since one page
// contributes several and they must not overwrite each other.
var jsCoverageSeq atomic.Int64

// jsCoverageEnabled reports whether the stack under test was built with an instrumented bundle.
// The counters simply do not exist otherwise, so collection is skipped rather than empty.
func jsCoverageEnabled() bool {
	return os.Getenv("E2E_COVERAGE") == "1"
}

// snapshotJSCoverage writes what the bundle has counted so far and is safe to call at any point.
//
// It is called before every reload and every navigation as well as at cleanup, because
// `window.__coverage__` belongs to the document: the browser discards it on unload, so a test that
// acts and then reloads would otherwise report only what the last document happened to run, and
// the total would look plausible while missing the very steps the test was written for.
//
// Every frame is asked, because both carry counters: the embed script runs in the page the test
// navigated and the widget itself runs in the iframe below it, and a test may open more frames
// still. A frame with no bundle in it answers null, which is not an error.
func snapshotJSCoverage(t *testing.T, page playwright.Page) {
	t.Helper()
	if !jsCoverageEnabled() {
		return
	}

	seq := jsCoverageSeq.Add(1)
	for i, frame := range page.Frames() {
		// a frame that has gone is a frame whose counters went with it, which is the loss this
		// whole file exists to prevent; every navigation and removal is snapshotted ahead of
		// time, so reaching here means one was missed
		raw, err := frame.Evaluate(`() => window.__coverage__ ?? null`)
		require.NoErrorf(t, err, "reading the widget coverage from frame %d: a document was destroyed before it was read", i)
		// no bundle in this frame, which is the ordinary answer for the host page
		if raw == nil {
			continue
		}

		counters, ok := raw.(map[string]any)
		require.Truef(t, ok, "the widget coverage came back as %T, which no reporter can merge", raw)
		if len(counters) == 0 {
			continue
		}

		rewritten, err := rewriteCoveragePaths(counters)
		require.NoError(t, err)

		body, err := json.Marshal(rewritten)
		require.NoError(t, err, "encoding the widget coverage")

		require.NoError(t, os.MkdirAll(jsCoverageDir, 0o750))
		// the pid keeps two processes from overwriting each other, the same way the traces are
		// named; the sequence separates the snapshots and the index the frames of one page
		name := strings.ReplaceAll(t.Name(), "/", "-")
		path := filepath.Join(jsCoverageDir, fmt.Sprintf("%s-%d-%d-%d.json", name, os.Getpid(), seq, i))
		require.NoError(t, os.WriteFile(path, body, 0o600), "writing the widget coverage")
	}
}

// rewriteCoveragePaths moves the counters from the paths the bundle was built at to paths that
// resolve where the report is made. Both the key and the `path` inside each entry are rewritten,
// since the reporter reads the second and the merge keys on the first.
//
// A path that is still absolute afterwards means the image no longer builds where this expects,
// and the reporter would quietly drop or misplace that file, so it is an error rather than a
// value passed through.
func rewriteCoveragePaths(counters map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(counters))
	for file, entry := range counters {
		rewritten, err := underBuildRoot(file)
		if err != nil {
			return nil, err
		}

		fields, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("the entry for %q is %T, which no reporter can merge", file, entry)
		}
		embedded, ok := fields["path"].(string)
		if !ok {
			return nil, fmt.Errorf("the entry for %q carries no path, so the reporter would place it nowhere", file)
		}
		// the reporter reads this one and the merge keys on the other, so a rewrite that moved
		// only the key would file the counters under a name whose contents are read from elsewhere
		if fields["path"], err = underBuildRoot(embedded); err != nil {
			return nil, err
		}

		out[rewritten] = fields
	}
	return out, nil
}

// underBuildRoot strips the root the image builds at, and refuses anything that was not under it.
// TrimPrefix alone accepts a path from a root that has moved, and nyc then omits or misplaces that
// file without the empty-directory guard noticing anything.
func underBuildRoot(path string) (string, error) {
	if !strings.HasPrefix(path, jsCoverageBuildRoot) {
		return "", fmt.Errorf("the bundle recorded %q, which is not under %s: the image build layout has moved",
			path, jsCoverageBuildRoot)
	}
	rest := strings.TrimPrefix(path, jsCoverageBuildRoot)
	if rest == "" {
		return "", fmt.Errorf("the bundle recorded %q, which names no file", path)
	}
	return rest, nil
}

// TestRewriteCoveragePaths pins the rewrite, since everything downstream of it is a report that
// looks the same whether or not the paths are right: nyc silently omits a file it cannot place,
// and the empty-directory guard only notices when nothing at all arrives.
func TestRewriteCoveragePaths(t *testing.T) {
	const root = jsCoverageBuildRoot

	t.Run("moves both the key and the path the reporter reads", func(t *testing.T) {
		out, err := rewriteCoveragePaths(map[string]any{
			root + "app/common/api.ts": map[string]any{"path": root + "app/common/api.ts", "s": map[string]any{}},
		})
		require.NoError(t, err)

		entry, ok := out["app/common/api.ts"].(map[string]any)
		require.True(t, ok, "the counters were not filed under the rewritten name")
		assert.Equal(t, "app/common/api.ts", entry["path"], "the reporter would read this one and place the file by it")
	})

	t.Run("refuses a key from another root", func(t *testing.T) {
		_, err := rewriteCoveragePaths(map[string]any{
			"/srv/elsewhere/app/a.ts": map[string]any{"path": root + "app/a.ts"},
		})
		require.Error(t, err, "a path from a root that moved would be reported against a file nobody has")
	})

	t.Run("refuses an embedded path from another root", func(t *testing.T) {
		_, err := rewriteCoveragePaths(map[string]any{
			root + "app/a.ts": map[string]any{"path": "/srv/elsewhere/app/a.ts"},
		})
		require.Error(t, err, "the key and the path have to name the same file for the report to mean anything")
	})

	t.Run("refuses an entry that is not an object", func(t *testing.T) {
		_, err := rewriteCoveragePaths(map[string]any{root + "app/a.ts": "not an object"})
		require.Error(t, err)
	})

	t.Run("refuses an entry with no path at all", func(t *testing.T) {
		_, err := rewriteCoveragePaths(map[string]any{root + "app/a.ts": map[string]any{"s": map[string]any{}}})
		require.Error(t, err)
	})

	t.Run("refuses the build root naming no file", func(t *testing.T) {
		_, err := rewriteCoveragePaths(map[string]any{root: map[string]any{"path": root}})
		require.Error(t, err)
	})
}
