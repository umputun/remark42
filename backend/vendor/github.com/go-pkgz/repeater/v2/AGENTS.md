# AGENTS.md

This file provides guidance to AI coding agents working with code in this repository.

## Build and Test Commands

```bash
# run all tests with race detection and coverage
go test -timeout=60s -race -covermode=atomic -coverprofile=profile.cov ./...

# run a single test
go test -run TestFixedDelay

# run tests with verbose output
go test -v ./...

# lint the code (golangci-lint v2 required)
golangci-lint run

# format code
gofmt -s -w .
goimports -w .

# build with race detection
go build -race
```

## Architecture Overview

This is a retry/repeater library with a clean separation of concerns between retry strategies and retry decisions:

### Core Design Pattern
- **Strategy Interface**: Controls **HOW LONG** to wait between retries (delay calculation)
  - `FixedDelay`: Constant delay between attempts
  - `backoff`: Variable delays with constant/linear/exponential algorithms
  - Custom strategies can implement the `Strategy` interface

- **Repeater**: Controls **WHETHER** to retry (retry orchestration)
  - Uses Strategy for delays but owns the retry logic
  - `ErrorClassifier` function determines if errors are retryable
  - Handles context cancellation and critical errors
  - Tracks execution statistics

### Key Architectural Decisions

1. **Separation of Concerns**: Strategy handles timing/delays, Repeater handles retry decisions. This is why `SetErrorClassifier` is a Repeater method, not a strategy option.

2. **Option Types**:
   - `backoffOption` functions (`WithMaxDelay`, `WithBackoffType`, `WithJitter`) configure the backoff strategy
   - These are strategy-specific and only apply to `NewBackoff`
   - `ErrorClassifier` applies to all strategies, hence it's a setter on Repeater

3. **Thread Safety**: Repeater instances are NOT thread-safe by design. Each concurrent operation needs its own Repeater instance.

4. **Statistics Tracking**: The Repeater maintains detailed execution statistics (attempts, durations, errors) accessible via `Stats()` method after execution.

## Module Version

This is v2 of the module (`github.com/go-pkgz/repeater/v2`). The package follows semantic versioning.

## Testing Patterns

- Tests use `t.Run` subtests with testify assertions
- Race condition testing is standard (`-race` flag)
- Tests focus on timing behavior, context cancellation, and error handling edge cases