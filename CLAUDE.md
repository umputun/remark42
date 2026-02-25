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

## Code Style
- **Backend**: Formatting with golangci-lint, strict error handling
- **Frontend**: TypeScript with ESLint, Stylelint and Prettier
- **Imports**: Group stdlib, external packages, then internal packages
- **CSS**: All components use CSS Modules (`component.module.css`). Class naming: BEM block = `.root`, elements = camelCase, modifiers = camelCase. Use `clsx` for conditional class composition. `raw-content.css` is the only global CSS file (syntax highlighting utility). Root wrapper keeps bare `.dark`/`.light` theme class — 8+ module CSS files depend on `:global(.dark)` ancestor. `comment_highlighting` uses `:global()` for imperative `classList` usage in root.tsx

## Key Backend Packages
- **Web/API**: `github.com/go-chi/chi/v5`, `github.com/go-pkgz/rest`
- **Auth**: `github.com/go-pkgz/auth/v2`
- **Logging**: `github.com/go-pkgz/lgr`
- **Testing**: `github.com/stretchr/testify`
- **Notifications**: `github.com/go-pkgz/notify`

## Repository Structure
- Backend: Go server using BoltDB for storage
- Frontend: Preact/Redux-based UI with iframe embedding
