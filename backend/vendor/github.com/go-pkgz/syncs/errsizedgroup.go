package syncs

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
)

// ErrSizedGroup is a SizedGroup with error control. Works the same as errgrp.Group, i.e. returns first error.
// Can work as regular errgrp.Group or with early termination. Thread safe.
// ErrSizedGroup interface enforces constructor usage and doesn't allow direct creation of errSizedGroup
type ErrSizedGroup interface {
	Go(fn func() error)
	Wait() error
}

type errSizedGroup struct {
	wg          sync.WaitGroup
	sema        sync.Locker
	ctx         context.Context
	cancel      func()
	termOnError bool
	preLock     bool

	err     *multierror
	errLock sync.RWMutex
	errOnce sync.Once
}

// NewErrSizedGroup makes wait group with limited size alive goroutines.
// By default all goroutines will be started but will wait inside. For limited number of goroutines use Preemptive() options.
// TermOnErr will skip (won't start) all other goroutines if any error returned.
func NewErrSizedGroup(size int, options ...ESGOption) ErrSizedGroup {
	res := errSizedGroup{
		sema: NewSemaphore(size),
		err:  new(multierror),
	}

	for _, opt := range options {
		if err := opt(&res); err != nil {
			log.Printf("[WARN] failed to set cache option, %v", err)
		}
	}

	return &res
}

// Go calls the given function in a new goroutine.
// The first call to return a non-nil error cancels the group if termOnError; its error will be
// returned by Wait. If no termOnError all errors will be collected in multierror.
func (g *errSizedGroup) Go(f func() error) {

	g.wg.Add(1)

	if g.preLock {
		g.sema.Lock()
	}

	go func() {
		defer g.wg.Done()

		// terminated will be true if any error happened before and g.termOnError
		terminated := func() bool {
			if !g.termOnError {
				return false
			}
			g.errLock.RLock()
			defer g.errLock.RUnlock()
			return g.err.errorOrNil() != nil
		}

		if terminated() {
			return // terminated due prev error, don't run anything in this group anymore
		}

		if !g.preLock {
			g.sema.Lock()
		}

		if err := f(); err != nil {

			g.errLock.Lock()
			g.err = g.err.append(err)
			g.errLock.Unlock()

			g.errOnce.Do(func() { // call context cancel once
				if g.cancel != nil {
					g.cancel()
				}
			})
		}
		g.sema.Unlock()
	}()
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the first all errors (if any) wrapped with multierror from them.
func (g *errSizedGroup) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel()
	}
	return g.err.errorOrNil()
}

// ESGOption functional option type
type ESGOption func(esg *errSizedGroup) error

// Context passes ctx and makes it cancelable
func Context(ctx context.Context) ESGOption {
	return func(esg *errSizedGroup) error {
		ctxWithCancel, cancel := context.WithCancel(ctx)
		esg.cancel = cancel
		esg.ctx = ctxWithCancel
		return nil
	}
}

// Preemptive sets locking mode preventing spawning waiting goroutine. May cause Go call to block!
func Preemptive() ESGOption {
	return func(esg *errSizedGroup) error {
		esg.preLock = true
		return nil
	}
}

// TermOnErr prevents new goroutines to start after first error
func TermOnErr() ESGOption {
	return func(esg *errSizedGroup) error {
		esg.termOnError = true
		return nil
	}
}

type multierror struct {
	errors []error
	lock   sync.Mutex
}

func (m *multierror) append(err error) *multierror {
	m.lock.Lock()
	m.errors = append(m.errors, err)
	m.lock.Unlock()
	return m
}

func (m *multierror) errorOrNil() *multierror {
	m.lock.Lock()
	defer m.lock.Unlock()
	if len(m.errors) == 0 {
		return nil
	}
	return m
}

// Error returns multierror string
func (m *multierror) Error() string {
	m.lock.Lock()
	defer m.lock.Unlock()
	if len(m.errors) == 0 {
		return ""
	}

	errs := []string{}

	for n, e := range m.errors {
		errs = append(errs, fmt.Sprintf("[%d] {%s}", n, e.Error()))
	}
	return fmt.Sprintf("%d error(s) occurred: %s", len(m.errors), strings.Join(errs, ", "))
}
