# Remark42 Development Guidelines

## Build/Test/Lint Commands
- **Backend**:
  - Run server: `make rundev`
  - Build: `make backend`
  - Race test: `make race_test`
- **Backend Testing**:
  - Run all tests: `cd backend/app && go test -timeout=60s -count 1 ./...`
  - Run single test: `cd backend/app && go test -run TestName ./path/to/package`
  - **IMPORTANT**: Run example tests: `cd backend/_example/memory_store && go test -race ./... && go build -race ./...`
- **Frontend**:
  - Development: `cd frontend && pnpm dev:app`
  - Tests: `cd frontend && pnpm test`
- **Lint**:
  - Backend: `cd backend && golangci-lint run`
  - **IMPORTANT**: Example lint: `cd backend/_example/memory_store && golangci-lint run --config ../../.golangci.yml`
  - Frontend: `cd frontend && pnpm lint`
  - **Before committing**: Always run tests and linter on both main backend AND examples
- **Dependency Updates**:
  - When updating Go modules in `backend/`, also run `go mod tidy` (and `go mod vendor`) in `backend/_example/memory_store` to keep indirect deps in sync. The example module replaces `github.com/umputun/remark42/backend` with `../../` so stale indirect deps there will break the example build.

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

For local artifact runs, install GoReleaser, Go 1.25, Node 16+, PNPM 8, and Perl, then use `make release`. The target runs a snapshot/no-publish GoReleaser build, leaves local artifacts and metadata in `dist/`, and cleans generated frontend embed files after GoReleaser exits. Do not run raw `goreleaser release` for local artifacts unless you also run `./scripts/cleanup-release-assets.sh` afterward.

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
