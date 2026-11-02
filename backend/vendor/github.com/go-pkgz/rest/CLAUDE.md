# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`github.com/go-pkgz/rest` is a library of HTTP middlewares and small helpers for REST services. It is not an application — there is no `main`, no server, no CLI. It is consumed by other projects (remark42, etc.). Keep the public API stable and minimal.

Dependencies are deliberately tiny: only `stretchr/testify` (tests) and `golang.org/x/crypto` (Argon2/bcrypt for BasicAuth). Do not add a routing framework or logging library dependency — middlewares are plain stdlib and must stay router-agnostic.

## Commands

- Test everything: `go test ./...`
- Single test: `go test -run TestName ./...` (e.g. `go test -run TestPing ./...`)
- Race + coverage (mirrors CI): `TZ="America/Chicago" go test -timeout=60s -race -covermode=atomic ./...`
- Lint (run from repo root): `golangci-lint run --max-issues-per-linter=0 --max-same-issues=0`

Time-parsing tests (`ParseFromTo`) are timezone-sensitive; CI runs with `TZ=America/Chicago`. Set it locally if a from/to test behaves oddly.

## Architecture

Three packages:
- **`rest`** (root) — all middlewares plus JSON/error/file-server helpers.
- **`logger`** — request-logging middleware, split out so it can be wired to any backend via the `logger.Backend` interface (`Logf(format, args...)`). Configured with functional options (`logger.New(logger.Prefix(...), logger.WithBody, ...)`).
- **`realip`** — `realip.Get(r)` extracts the client IP from proxy headers. Used by both `rest.RealIP` and the `logger` package. Only public IPs are accepted from headers.

### Middleware conventions

Every middleware is a `func(http.Handler) http.Handler` (or a `func(...) func(http.Handler) http.Handler` when it takes config). This is the chi/stdlib-compatible shape — match it for any new middleware. Some middlewares short-circuit the chain (`Ping`, `Health`, metrics) by writing a response and returning without calling the next handler.

Configurable middlewares use the **functional-options pattern**, not option structs:
- `CORS(CorsAllowedOrigins(...), CorsAllowCredentials(true), ...)` — options named `CorsXxx`.
- `Secure(SecFrameOptions(...), SecHSTS(...), ...)` — options named `SecXxx`, plus `SecAllHeaders()` convenience.
- `logger.New(logger.Prefix(...), ...)` — options in `logger/options.go`.

When adding an option to one of these, follow the existing prefix/naming and keep defaults sensible so calling the constructor with no options is safe.

### CSRF build-tag split (important)

CSRF protection has **two implementations behind one identical public API**, selected by Go version:
- `csrf_go125.go` (`//go:build go1.25`) — thin wrapper over stdlib `http.CrossOriginProtection`.
- `csrf.go` (`//go:build !go1.25`) — self-contained equivalent for older Go.

`NewCrossOriginProtection`, `AddTrustedOrigin`, `AddBypassPattern`, `SetDenyHandler`, `Check`, `Handler` must exist with the **same signatures and behavior in both files**. When you change the CSRF API or behavior, edit both files and keep `csrf_test.go` passing under both build tags. go.mod targets go 1.24, so by default the `!go1.25` path compiles unless building with a 1.25 toolchain.

### Helpers

`rest.go` holds the JSON render/encode/decode helpers (`RenderJSON`, `EncodeJSON`/`DecodeJSON` generics, `RenderJSONWithHTML`) and `ParseFromTo`. `httperrors.go` has `SendErrorJSON`/`NewErrorLogger`. `file_server.go` has `FileServer` (directory listing disabled by design). `benchmarks.go` keeps an in-memory ring of up to 900 per-second data points (15 min) queried via `Stats(duration)`.

## Conventions

- **One test file per source file**: `foo.go` → `foo_test.go` only. Table-driven with testify. Note `depricattion.go`/`depricattion_test.go` is misspelled but is the real filename — don't "fix" it without intent, it's the established path.
- After changing or adding a middleware/helper, update `README.md` — it documents every middleware and helper and is the primary user-facing doc.
- Lint config (`.golangci.yml`) is strict (`govet enable-all`, revive, gocritic with performance/style/experimental). `modernize` is enabled — prefer `any` over `interface{}`, `slices`/`maps` stdlib, etc.
