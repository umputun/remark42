# Remark42 Development Guidelines

## Build/Test/Lint Commands
- **Backend**:
  - Run server: `make rundev`
  - Build: `make backend`
  - Race test: `make race_test`
- **Backend Testing**:
  - Run all tests: `cd backend/app && go test -timeout=300s -count 1 ./...`
  - Run single test: `cd backend/app && go test -run TestName ./path/to/package`
  - **IMPORTANT**: Run example tests: `cd backend/_example/memory_store && go test -race ./... && go build -race ./...`
- **Frontend**:
  - Development: `cd frontend/apps/remark42 && pnpm dev`
  - Tests: `cd frontend/apps/remark42 && pnpm test`
- **End-to-end**: `make e2e` drives the widget in a real browser; see `e2e/README.md`. Build-tagged, so `go test ./...` never runs it.
- **Lint**:
  - Backend: `cd backend && golangci-lint run`
  - **IMPORTANT**: Example lint: `cd backend/_example/memory_store && golangci-lint run --config ../../.golangci.yml`
  - Frontend: `cd frontend/apps/remark42 && pnpm lint`
  - **Before committing**: Always run tests and linter on both main backend AND examples
- **Go module changes**:
  - **Any** change to `backend/go.mod` or `backend/go.sum` requires `go mod tidy` in `backend/_example/memory_store` in the same commit. That covers dependency bumps, adding or removing a dependency, and changing the `go` directive, not only version updates.
  - Only `go mod tidy` there, not `go mod vendor`: the example's vendor directory is gitignored (`.gitignore:26`), so its output is never committed, while a stale local copy silently becomes what the example resolves against.
  - The example module replaces `github.com/umputun/remark42/backend` with `../../`, so it carries the backend's dependencies as indirect entries. Leaving them stale fails the `test examples` CI step with `go: updates to go.mod needed; to update it: go mod tidy`.
  - This applies to Dependabot pull requests too: the bot updates `backend/` only, so its Go module PRs need the example tidied before they can go green.


## Backend Test Determinism

Backend tests must never depend on how fast the machine is. CI runs them under `-race` with coverage on a shared runner, so any test that assumes an operation finishes within some duration eventually fails on a rerun-and-it-passes basis.

- **Wait on a condition, never on a duration.** Use `require.Eventually` / `require.EventuallyWithT` to poll for the state the assertion needs, and `require.Never` when the point is that something did *not* happen. A bare `time.Sleep` before an assertion is a defect; sleeping until a deadline you computed, as `waitPastMillisecond` does, is not.
- **Polling closures must not touch `*testing.T`.** testify runs them on a separate goroutine, where `t.FailNow` is undefined behaviour. Assert on the `*assert.CollectT` that `EventuallyWithT` hands the closure, so the real error also lands in the failure message.
- **Mind the rate limiter when polling over HTTP.** Route groups are capped independently and most of the caps are hard-coded in `rest.go`, out of reach of a test: `/auth/` at 2 req/s and the admin, protected and image routes at 10 req/s. Only the open-route group is settable, via `openRouteLimiter` (100 in `startupT`). Poll with the existing constants rather than a new number, `httpPoll` for anything issuing an HTTP request and `pollInterval` only for in-process or filesystem checks, or the poll manufactures the 429s it then has to interpret.
- **When a test needs time to have passed, pin the clock input rather than waiting for it:** `os.Chtimes` for file ages, an explicit `store.Comment.Timestamp` for anything that formats a timestamp.
- **Prefer a `testing/synctest` bubble** where the code under test has no real I/O. Inside one the clock is fake, so `time.Sleep` is instant and deterministic. `app/notify`, `app/store/service`, `app/store/image`, `app/store/engine`, `app/providers`, `app/migrator` and `_example/memory_store/accessor` already use it, and most surviving `time.Sleep` calls live in them.
- **Helpers fail loudly.** A wait that gives up must call `t.Fatal`/`require` naming what it was waiting for, never return silently and leave the next assertion to fail with something unrelated. Because these packages run `goleak.VerifyTestMain`, a failing helper also exits the test goroutine, so anything that started a server in a goroutine must `defer cancel()` or `defer srv.Shutdown()` right after launching it; otherwise a failed readiness wait is reported as a goroutine leak rather than the failure that caused it.
- **Take ports and paths from outside the test.** Ports come from the kernel with `net.Listen("tcp", ":0")`, files from `t.TempDir()`. `go test ./...` runs package binaries concurrently, so a number out of a fixed range or a fixed name under `/tmp` lets two of them collide.
- **Close idle connections before shutting a test server down.** Clients built as `http.Client{Timeout: x}` share `http.DefaultTransport`, and `Shutdown` waits on their keep-alive connections until its own deadline expires.
- **Keep the test timeout budgets aligned.** `Makefile`, `ci-backend.yml`, `release.yml` and the command above all use `-timeout=300s`; the wait helpers allow 30s per condition, so a shorter per-package budget turns a slow runner into a timeout panic instead of a readable failure.

`chooseUnusedPort` and the server-start wait helpers are duplicated in `app`, `app/cmd`, `app/rest/api` and `_example/memory_store/server`. Nothing shares them today; keep the copies in step when changing one.

## Release Procedure

Remark42 uses two tags for each release:
- `vX.Y.Z` - product release tag used by GitHub releases, GoReleaser binary artifacts, and Docker image publishing.
- `backend/vX.Y.Z` - nested Go module tag for `github.com/umputun/remark42/backend`.

Release flow:
1. Create the GitHub release for `vX.Y.Z` with title `Version X.Y.Z`. The GitHub release must exist before the `vX.Y.Z` tag reaches the remote; `gh release create vX.Y.Z` satisfies this because it creates and pushes the tag.
2. The `vX.Y.Z` tag triggers GoReleaser, which builds and uploads binary artifacts to the existing release.
3. Create and push the matching backend module tag pointing at the same commit:

```bash
git fetch origin --tags
git tag backend/vX.Y.Z vX.Y.Z
git push origin backend/vX.Y.Z
```

GoReleaser must ignore `backend/*` tags in `.goreleaser.yml` so release notes and current-tag detection use only product tags. Docker image publishing stays separate and is handled by the existing Docker workflow.

For local artifact runs, install GoReleaser, Go 1.25, Node 24+ and PNPM 10, then use `make release`. The target runs a snapshot/no-publish GoReleaser build, leaves local artifacts and metadata in `dist/`, and cleans generated frontend embed files after GoReleaser exits. Do not run raw `goreleaser release` for local artifacts unless you also run `./scripts/cleanup-release-assets.sh` afterward.

## Milestones and Issue Labels

**Milestones** — one `vX.Y.Z` milestone per release. Assign every merged PR, and every issue closed by a code change, to the milestone of the release it shipped in.
- Decide which release a PR belongs to by whether its merge commit is **contained in a release tag** — not by comparing dates (a tag can be cut from an earlier commit, or moved). `git fetch --tags`, then `git tag --contains <merge_sha> | grep '^v' | sort -V | head -1` is its release. If no release tag contains it yet, it belongs to the next (unreleased) version's milestone — create it if missing (`gh api repos/umputun/remark42/milestones -f title="vX.Y.Z"`).
- An **issue gets a milestone only when it was closed by a code change** (a linked closing PR/commit); take the milestone from that PR/commit (via the commit-in-tag rule). Issues closed as `duplicate`/`invalid`/`wontfix`/answered get no milestone.
- Find unassigned: `gh pr list --state merged --search "no:milestone"`, `gh issue list --state closed --search "no:milestone"`. Assign with `gh pr edit N --milestone "vX.Y.Z"` / `gh issue edit N --milestone "vX.Y.Z"`.

**Issue labels** — classify each issue with a type and an area (add priority when relevant):
- Type: `bug`, `enhancement`, `question`, `documentation`, `discussion`
- Area: `backend`, `frontend`, `site`, `CI`, `design`, `localization`
- Priority: `important`, `minor`, `some day`
- Contribution: `help wanted`, `good-first-issue`
- Resolution (on close, when applicable): `duplicate`, `invalid`, `wontfix`, `no-action-needed`
- PR auto-labels (applied by Dependabot/Actions, not manual PRs): `dependencies`, `go`, `javascript`, `github_actions`

## Code Style
- **Backend**: Formatting with golangci-lint, strict error handling
- **Frontend**: TypeScript with ESLint, Stylelint and Prettier
- **Imports**: Group stdlib, external packages, then internal packages
- **CSS**: All components use CSS Modules (`component.module.css`). Class naming: BEM block = `.root`, elements = camelCase, modifiers = camelCase. Use `clsx` for conditional class composition. `raw-content.css` is the only global CSS file (syntax highlighting utility). Root wrapper keeps bare `.dark`/`.light` theme class — 8+ module CSS files depend on `:global(.dark)` ancestor. `comment_highlighting` uses `:global()` for imperative `classList` usage in root.tsx

## Key Backend Packages
- **Web/API**: `github.com/go-pkgz/routegroup`, `github.com/go-pkgz/rest`
- **Auth**: `github.com/go-pkgz/auth/v2`
- **Logging**: `github.com/go-pkgz/lgr`
- **Testing**: `github.com/stretchr/testify`
- **Notifications**: `github.com/go-pkgz/notify`

## Repository Structure
- Backend: Go server using BoltDB for storage
- Frontend: Preact/Redux-based UI with iframe embedding
- `/web` is served from two sources, in lookup order: the frontend build output
  (`frontend/apps/remark42/public`, embedded at `backend/app/cmd/web` or read from `--web-root`),
  then `backend/app/webassets/assets`, embedded in the binary. A plain page or image the bundler
  does not process belongs in `webassets`; anything needing templating or the widget's CSS/JS goes
  through webpack. A name present in both is served from the frontend build.
