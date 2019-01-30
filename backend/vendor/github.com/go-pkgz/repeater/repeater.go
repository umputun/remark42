// Package repeater call fun till it returns no error, up to repeat some number of iterations and delays defined by strategy.
// Repeats number and delays defined by strategy.Interface. Terminates immediately on err from
// provided, optional list of critical errors
package repeater

import (
	"context"
	"time"

	"github.com/go-pkgz/repeater/strategy"
)

// Repeater is the main object, should be made by New or NewDefault, embeds strategy
type Repeater struct {
	strategy.Interface
}

// New repeater with a given strategy. If strategy=nil initializes with FixedDelay 5sec, 10 times.
func New(strtg strategy.Interface) *Repeater {
	if strtg == nil {
		strtg = strategy.NewFixedDelay(10, time.Second*5)
	}
	result := Repeater{Interface: strtg}
	return &result
}

// NewDefault makes repeater with FixedDelay strategy
func NewDefault(repeats int, delay time.Duration) *Repeater {
	return New(strategy.NewFixedDelay(repeats, delay))
}

// Do repeats fun till no error. Predefined (optional) errors terminate immediately
func (r Repeater) Do(ctx context.Context, fun func() error, errors ...error) (err error) {

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc() // ensure strategy's channel termination

	inErrors := func(err error) bool {
		for _, e := range errors {
			if e == err {
				return true
			}
		}
		return false
	}

	ch := r.Start(ctx) // channel of ticks-like events provided by strategy

	// closed channel indicates completion or early termination, set by strategy
	for range ch {

		if err = fun(); err == nil {
			return nil
		}
		if err != nil && inErrors(err) { //terminate on critical error from provided list
			return err
		}
	}
	return err
}
