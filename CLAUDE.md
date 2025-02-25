# Remark42 Development Guidelines

## Build/Test/Lint Commands
- **Backend**:
  - Run server: `make rundev`
  - Build: `make backend`
  - Race test: `make race_test`
- **Backend Testing**:
  - Run all tests: `cd backend/app && go test -timeout=60s -count 1 ./...`
  - Run single test: `cd backend/app && go test -run TestName ./path/to/package`
- **Frontend**:
  - Development: `cd frontend && pnpm dev:app`
  - Tests: `cd frontend && pnpm test`
- **Lint**:
  - Backend: `golangci-lint run`
  - Frontend: `cd frontend && pnpm lint`

## Code Style
- **Backend**: Formatting with golangci-lint, strict error handling
- **Frontend**: TypeScript with ESLint, Stylelint and Prettier
- **Imports**: Group stdlib, external packages, then internal packages
- **CSS**: CSS Modules for new components (`component.module.css`)

## Key Backend Packages
- **Web/API**: `github.com/go-chi/chi/v5`, `github.com/go-pkgz/rest`
- **Auth**: `github.com/go-pkgz/auth/v2`
- **Logging**: `github.com/go-pkgz/lgr`
- **Testing**: `github.com/stretchr/testify`
- **Notifications**: `github.com/go-pkgz/notify`

## Repository Structure
- Backend: Go server using BoltDB for storage
- Frontend: Preact/Redux-based UI with iframe embedding
