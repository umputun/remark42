# Repeater

[![pipeline status](https://git.tkginternal.com/commons/pkg/repeater/badges/master/pipeline.svg)](https://git.tkginternal.com/commons/pkg/repeater/commits/master) 
[![coverage report](https://git.tkginternal.com/commons/pkg/repeater/badges/master/coverage.svg)](https://git.tkginternal.com/commons/pkg/repeater/commits/master)
[![GoDoc](https://godoc.tkginternal.com/godoc.svg)](https://godoc.tkginternal.com/pkg/git.tkginternal.com/commons/pkg/repeater/)


Package repeater call fun till it returns no error, up to repeat some number of iterations and delays defined by strategy.
Repeats number and delays defined by strategy.Interface. Terminates immediately on err from provided, optional list of critical errors

## Install and update

`go get -u git.tkginternal.com/commons/pkg/repeater`

## How to use

New Repeater created by `New(strtg strategy.Interface)` or shortcut for defaults - `NewDefault(repeats int, delay time.Duration) *Repeater`.

To activate use `Do` method. Do repeats fun till no error. Predefined (optional) errors terminate immediately
                            
`func (r Repeater) Do(fun func() error, errors ...error) (err error)`

### Repeating strategy

User can provide his own strategy implementing this interface:

```go
type Interface interface {
	Start(ctx context.Context) chan struct{}
}
```

Returned channels used as "ticks", i.e. for each repeat (or initial) operation one read from this channel needed. Closing this channel indicates "done with retries". This is pretty much the same idea as `time.Timer` or `time.Tick` implements. Note - the first (technically not-repeated-yet) call won't happen **until something sent to the channel**. This is why typical strategy sends first "tick" prior to first wait/sleep.

Three mist common strategies provided by package and ready to use:
1. **Fixed delay**, up to max number of attempts - `NewFixedDelay(repeats int, delay time.Duration)`. 
This is default strategy used by `repeater.NewDefault` constructor
2. **BackOff** with jitter provides exponential backoff. It starts from 100ms interval and goes in steps with `last * math.Pow(factor, attempt)`. Optional jitter randomizes intervals a little bit. The strategy created by `NewBackoff(repeats int, factor float64, jitter bool)`. _Factor = 1 effectively makes this strategy fixed with 100ms delay._ 

3. **Once** strategy does not do any repeats and mainly useful for tests - `NewOnce()`

