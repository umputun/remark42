# Repeater

[![Build Status](https://github.com/go-pkgz/repeater/workflows/build/badge.svg)](https://github.com/go-pkgz/repeater/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/go-pkgz/repeater)](https://goreportcard.com/report/github.com/go-pkgz/repeater) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/repeater/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/repeater?branch=master)

Package repeater implements a functional mechanism to repeat operations with different retry strategies.

## Install and update

`go get -u github.com/go-pkgz/repeater/v2`

## Usage

### Basic Example with Exponential Backoff

```go
// create repeater with exponential backoff
r := repeater.NewBackoff(5, time.Second) // 5 attempts starting with 1s delay

err := r.Do(ctx, func() error {
// do something that may fail
return nil
})
```

### Fixed Delay with Critical Error

```go
// create repeater with fixed delay
r := repeater.NewFixed(3, 100*time.Millisecond)

criticalErr := errors.New("critical error")

err := r.Do(ctx, func() error {
// do something that may fail
return fmt.Errorf("temp error")
}, criticalErr) // will stop immediately if criticalErr returned
```

### Custom Backoff Strategy

```go
r := repeater.NewBackoff(5, time.Second,
repeater.WithMaxDelay(10*time.Second),
repeater.WithBackoffType(repeater.BackoffLinear),
repeater.WithJitter(0.1),
)

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := r.Do(ctx, func() error {
// do something that may fail
return nil
})
```

### Stop on Any Error

```go
r := repeater.NewFixed(3, time.Millisecond)

err := r.Do(ctx, func() error {
return errors.New("some error")
}, repeater.ErrAny)  // will stop on any error
```

## Strategies

The package provides several retry strategies:

1. **Fixed Delay** - each retry happens after a fixed time interval
2. **Backoff** - delay between retries increases according to the chosen algorithm:
   - Constant - same delay between attempts
   - Linear - delay increases linearly
   - Exponential - delay doubles with each attempt, up to the maximum delay

Backoff strategy can be customized with:
- Maximum delay cap
- Jitter to prevent thundering herd
- Different backoff types (constant/linear/exponential)

### Custom Strategies

You can implement your own retry strategy by implementing the Strategy interface:

```go
type Strategy interface {
    // NextDelay returns the delay after the given failed attempt
    // attempt starts from 1
    NextDelay(attempt int) time.Duration
}
```

The first attempt runs at once. `NextDelay(n)` is the wait after the n-th failed attempt, and it is never asked for after the last one, so `NewWithStrategy(5, strategy)` calls it with 1 to 4.

Example of a custom strategy that increases delay by a custom factor:

```go
// CustomStrategy implements Strategy with custom factor-based delays
type CustomStrategy struct {
    Initial time.Duration
    Factor  float64
}

func (s CustomStrategy) NextDelay(attempt int) time.Duration {
    if attempt <= 0 {
        return 0
    }
    delay := time.Duration(float64(s.Initial) * math.Pow(s.Factor, float64(attempt-1)))
    return delay
}

// Usage
strategy := &CustomStrategy{Initial: time.Second, Factor: 1.5}
r := repeater.NewWithStrategy(5, strategy)
err := r.Do(ctx, func() error {
    // the four retries wait 1s, 1.5s, 2.25s and 3.375s
    return nil
})
```

## Options

For backoff strategy, several options are available:

```go
WithMaxDelay(time.Duration)   // set maximum delay between retries, jitter included
WithBackoffType(BackoffType)  // set backoff type (constant/linear/exponential)
WithJitter(float64)           // add randomness to delays (0-1.0, values outside are clamped)
```

Without options, `NewBackoff` uses exponential backoff, a 30s maximum delay and jitter 0.1. A jitter factor f spreads each delay uniformly over plus or minus f/2 of it, so 0.1 means plus or minus 5%. The part of that range above the maximum delay is cut off, so a delay at or near the maximum spreads evenly below it. `WithMaxDelay(0)` removes the maximum.

For every constructor, an attempts count below 1 means a single attempt, and `NewWithStrategy` with a nil strategy uses a fixed 1s delay.

## Error Handling

- Stops on context cancellation, checked before each attempt and during the delay between attempts; `Do` then returns the context error, and `Stats().LastError` keeps the last error returned by the function
- Does not interrupt a running function: it has to watch `ctx` itself to stop early
- Can stop on specific errors (pass them as additional parameters to Do)
- Special `ErrAny` to stop on any error
- Returns last error if all attempts fail
- Custom error classification via `SetErrorClassifier`

### Error Classification

You can provide a custom error classifier function to dynamically determine if an error should trigger a retry or stop immediately. This is particularly useful for API clients where different error types require different handling:

```go
// Define what errors are retryable
isRetryable := func(err error) bool {
    if err == nil {
        return false
    }
    
    errStr := strings.ToLower(err.Error())
    
    // Retryable patterns
    if strings.Contains(errStr, "429") ||
       strings.Contains(errStr, "rate limit") ||
       strings.Contains(errStr, "timeout") ||
       strings.Contains(errStr, "503") {
        return true
    }
    
    // Non-retryable patterns
    if strings.Contains(errStr, "401") ||
       strings.Contains(errStr, "authentication") ||
       strings.Contains(errStr, "token limit") {
        return false
    }
    
    return true // default to retry
}

// Use with any repeater strategy
r := repeater.NewBackoff(5, time.Second)
r.SetErrorClassifier(isRetryable)

err := r.Do(ctx, func() error {
    // API call that might fail
    return apiClient.Call()
})
```

When an error classifier is set:
- After each error, the classifier function is called
- If it returns `false`, the operation stops immediately
- If it returns `true`, the retry logic continues
- When a classifier is set, the errors passed to `Do`, `ErrAny` included, are ignored

This feature works with all repeater strategies (NewFixed, NewBackoff, NewWithStrategy).

## Execution Statistics

The repeater tracks execution statistics that can be accessed after calling `Do()`:

```go
r := repeater.NewFixed(5, 100*time.Millisecond)

err := r.Do(ctx, func() error {
    // operation that might fail
    return someOperation()
})

// Get execution statistics
stats := r.Stats()

fmt.Printf("Attempts: %d\n", stats.Attempts)
fmt.Printf("Success: %v\n", stats.Success)
fmt.Printf("Total Duration: %v\n", stats.TotalDuration)
fmt.Printf("Work Duration: %v\n", stats.WorkDuration)
fmt.Printf("Delay Duration: %v\n", stats.DelayDuration)
if stats.LastError != nil {
    fmt.Printf("Last Error: %v\n", stats.LastError)
}
```

### Available Statistics

The `Stats` struct provides the following information:

- `Attempts` - Number of attempts made (including successful ones)
- `Success` - Whether the operation eventually succeeded
- `TotalDuration` - Total elapsed time from start to finish
- `WorkDuration` - Time spent executing the function (excluding delays)
- `DelayDuration` - Time spent in delays between attempts
- `LastError` - Last error returned by the function (nil if succeeded); the context error only when the context ended before the first attempt
- `StartedAt` - When the repeater started
- `FinishedAt` - When the repeater finished

### Usage Example

```go
r := repeater.NewBackoff(3, time.Second)

err := r.Do(ctx, func() error {
    // Simulate work that takes time
    time.Sleep(200 * time.Millisecond)
    
    // Randomly fail
    if rand.Float32() < 0.7 {
        return errors.New("temporary error")
    }
    return nil
})

stats := r.Stats()

// Log detailed statistics
log.Printf("Operation completed in %v with %d attempts", 
    stats.TotalDuration, stats.Attempts)
log.Printf("Time spent working: %v", stats.WorkDuration)
log.Printf("Time spent waiting: %v", stats.DelayDuration)

if err != nil {
    log.Printf("Failed after %d attempts: %v", stats.Attempts, err)
} else {
    log.Printf("Succeeded after %d attempts", stats.Attempts)
}
```

### Thread Safety

Note that the `Repeater` is not thread-safe: `Do` records statistics on every call. A `Repeater` must not be used from several goroutines at once, even with the same function; create one per concurrent call.

## Migrating from v1

v2 is a separate module, `github.com/go-pkgz/repeater/v2`, and replaces the whole v1 API, including the `strategy` package:

| v1 | v2 |
|---|---|
| `repeater.NewDefault(n, d)` | `repeater.NewFixed(n, d)` |
| `repeater.New(&strategy.FixedDelay{Repeats: n, Delay: d})` | `repeater.NewFixed(n, d)` |
| `repeater.New(&strategy.Backoff{Duration: d, Repeats: n, Factor: 2})` | `repeater.NewBackoff(n, d)` |
| `strategy.Backoff` with `Factor` 1, 0 or below, or unset | `repeater.NewBackoff(n, d, repeater.WithBackoffType(repeater.BackoffConstant))` |
| `strategy.Backoff` with any other `Factor` | a custom `Strategy`, as in [Custom Strategies](#custom-strategies) |
| `repeater.New(&strategy.Once{})` | `repeater.NewFixed(1, 0)` |
| `repeater.New(nil)` (10 attempts, 5s apart) | `repeater.NewFixed(10, 5*time.Second)` |
| a custom strategy with `Start(ctx) <-chan struct{}` | a type with `NextDelay(attempt int) time.Duration`, passed to `repeater.NewWithStrategy` |

Behavior that changed:

- v1 `Backoff` had no maximum delay, and its `Jitter` option was off by default and spread each delay by plus or minus the initial `Duration`. `NewBackoff` caps delays at 30s and adds proportional jitter of 0.1 by default; `WithMaxDelay(0)` and `WithJitter(0)` turn both off.
- v1 `Backoff` started from 100ms when `Duration` was unset; v2 uses the initial delay as given.
- `NewWithStrategy(n, nil)` uses a fixed 1s delay with the n attempts passed, where v1 `New(nil)` meant 10 attempts 5s apart.
- In v1 each delay started with the attempt and ran while the function ran, and when the last attempt failed with a retryable error, `Do` waited out what was left of its delay before returning. In v2 each delay starts after a failed attempt returns, and `Do` returns right after the last attempt.
- A v1 custom strategy decided when the first attempt ran, since `Do` waited for its first tick, and it stopped retries by closing the channel. In v2 the first attempt runs at once and `NextDelay` only returns a delay, so a v2 strategy can do neither: hold back the first call before calling `Do`, and stop retries with the attempts count, `ctx`, the errors passed to `Do` or an error classifier.
- A v1 `Repeater` kept no state of its own and could be shared between goroutines. A v2 `Repeater` records statistics on every `Do`, so it must not be shared; create one per concurrent call.
