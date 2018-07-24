# Repeater [![Build Status](https://travis-ci.org/go-pkgz/repeater.svg?branch=master)](https://travis-ci.org/go-pkgz/repeater) [![Go Report Card](https://goreportcard.com/badge/github.com/go-pkgz/repeater)](https://goreportcard.com/report/github.com/go-pkgz/repeater) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/repeater/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/repeater?branch=master)

Repeater calls a function until it returns no error, up to some number of iterations and delays defined by strategy. It terminates immediately on err from the provided (optional) list of critical errors.

## Install and update

`go get -u github.com/go-pkgz/repeater`

## How to use

New Repeater created by `New(strtg strategy.Interface)` or shortcut for defaults - `NewDefault(repeats int, delay time.Duration) *Repeater`.

To activate invoke `Do` method. `Do` repeats func until no error returned. Predefined (optional) errors terminates the loop immediately.
                            
`func (r Repeater) Do(fun func() error, errors ...error) (err error)`

### Repeating strategy

User can provide his own strategy implementing the interface:

```go
type Interface interface {
	Start(ctx context.Context) chan struct{}
}
```

Returned channels used as "ticks," i.e., for each repeat or initial operation one read from this channel needed. Closing this channel indicates "done with retries." It is pretty much the same idea as `time.Timer` or `time.Tick` implements. Note - the first (technically not-repeated-yet) call won't happen **until something sent to the channel**. For this reason, the typical strategy sends first "tick" before the first wait/sleep.

Three most common strategies provided by package and ready to use:
1. **Fixed delay**, up to max number of attempts - `NewFixedDelay(repeats int, delay time.Duration)`. 
It is the default strategy used by `repeater.NewDefault` constructor
2. **BackOff** with jitter provides exponential backoff. It starts from 100ms interval and goes in steps with `last * math.Pow(factor, attempt)`. Optional jitter randomizes intervals a little bit. The strategy created by `NewBackoff(repeats int, factor float64, jitter bool)`. _Factor = 1 effectively makes this strategy fixed with 100ms delay._ 
3. **Once** strategy does not do any repeats and mainly used for tests/mocks - `NewOnce()`



