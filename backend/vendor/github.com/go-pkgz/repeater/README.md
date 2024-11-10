# Repeater [![Build Status](https://github.com/go-pkgz/repeater/workflows/build/badge.svg)](https://github.com/go-pkgz/repeater/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/go-pkgz/repeater)](https://goreportcard.com/report/github.com/go-pkgz/repeater) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/repeater/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/repeater?branch=master)

Repeater calls a function until it returns no error, up to some number of iterations and delays defined by strategy. It terminates immediately on err from the provided (optional) list of critical errors.

## Install and update

`go get -u github.com/go-pkgz/repeater`

## How to use

New Repeater created by `New(strtg strategy.Interface)` or shortcut for default - `NewDefault(repeats int, delay time.Duration) *Repeater`.

To activate invoke `Do` method. `Do` repeats func until no error returned. Predefined (optional) errors terminate the loop immediately.
                            
`func (r Repeater) Do(ctx context.Context, fun func() error, errors ...error) (err error)`

### Repeating strategy

User can provide his own strategy implementing the interface:

```go
type Interface interface {
	Start(ctx context.Context) chan struct{}
}
```

Returned channels used as "ticks," i.e., for each repeat or initial operation one read from this channel needed. Closing the channel indicates "done with retries." It is pretty much the same idea as `time.Timer` or `time.Tick` implements. Note - the first (technically not-repeated-yet) call won't happen **until something sent to the channel**. For this reason, the typical strategy sends the first "tick" before the first wait/sleep.

Three strategies provided byt the package:

1. **Fixed delay**, up to max number of attempts. It is the default strategy used by `repeater.NewDefault` constructor.
2. **BackOff** with jitter provides an exponential backoff. It starts from `Duration` interval and goes in steps with `last * math.Pow(factor, attempt)`. Optional jitter randomizes intervals a little. _Factor = 1 effectively makes this strategy fixed with `Duration` delay._ 
3. **Once** strategy does not do any repeats and mainly used for tests/mocks`.
