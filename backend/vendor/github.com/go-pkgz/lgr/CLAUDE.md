# Go-PKGZ/LGR Development Guidelines

## Build & Test Commands
- Build: `go build -race`
- Test all: `go test -timeout=60s -race -covermode=atomic -coverprofile=profile.cov`
- Test single file: `go test -run TestName`
- Benchmark: `go test -bench=. -run=Bench`
- Lint: `golangci-lint run`

## Code Style Guidelines
- Go 1.21 compatibility required
- Maximum line length: 140 characters
- No package names with underscores
- Use early returns (enforced by prealloc linter)
- Test files use testify for assertions: `require` for fatal assertions, `assert` for non-fatal ones
- Indent with tabs, not spaces

## Error Handling
- FATAL logs to stderr and calls os.Exit(1)
- ERROR logs to both stdout and stderr
- PANIC logs stack trace and runtime info to stderr
- Stack traces for ERROR level can be enabled with StackTraceOnError option

## Project Conventions
- Public API follows interface-based design (`lgr.L` interface)
- Avoid global loggers, prefer dependency injection
- Functional options pattern for logger configuration
- Secret logging sanitization with `lgr.Secret` option