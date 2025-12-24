package syncs

import "context"

type options struct {
	ctx           context.Context
	preLock       bool
	termOnError   bool
	discardIfFull bool
}

// GroupOption functional option type
type GroupOption func(o *options)

// Context passes ctx to group, goroutines will be canceled if ctx is canceled
func Context(ctx context.Context) GroupOption {
	return func(o *options) {
		o.ctx = ctx
	}
}

// Preemptive sets locking mode preventing spawning waiting goroutine. May cause Go call to block!
func Preemptive(o *options) {
	o.preLock = true
}

// TermOnErr prevents new goroutines to start after first error
func TermOnErr(o *options) {
	o.termOnError = true
}

// Discard will discard new goroutines if semaphore is full, i.e. no more goroutines allowed
func Discard(o *options) {
	o.discardIfFull = true
	o.preLock = true // discard implies preemptive
}
