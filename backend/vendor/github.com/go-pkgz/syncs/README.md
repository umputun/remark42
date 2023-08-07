# Syncs - additional synchronization primitives 

[![Build Status](https://github.com/go-pkgz/syncs/workflows/build/badge.svg)](https://github.com/go-pkgz/syncs/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/go-pkgz/syncs)](https://goreportcard.com/report/github.com/go-pkgz/syncs) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/syncs/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/syncs?branch=master)

The `syncs` package offers extra synchronization primitives, such as `Semaphore`, `SizedGroup`, and `ErrSizedGroup`, to help manage concurrency in Go programs. With `syncs` package, you can efficiently manage concurrency in your Go programs using additional synchronization primitives. Use them according to your specific use-case requirements to control and limit concurrent goroutines while handling errors and early termination effectively.

## Install and update

`go get -u github.com/go-pkgz/syncs`

## Details

### Semaphore

`Semaphore` implements the `sync.Locker` interface with an additional `TryLock` function and a specified capacity. 
It is thread-safe. The `Lock` function increases the count, while Unlock decreases it. When the count is 0, `Unlock` will block, and `Lock` will block until the count is greater than 0. The `TryLock` function will return false if locking failed (i.e. semaphore is locked) and true otherwise.

```go
	sema := syncs.NewSemaphore(10) // make semaphore with 10 initial capacity
	for i :=0; i<10; i++ {
		sema.Lock() // all 10 locks will pass, i.w. won't lock
	}
	sema.Lock() // this is 11 - will lock for real

	// in some other place/goroutine
	sema.Unlock() // decrease semaphore counter
	ok := sema.TryLock() // try to lock, will return false if semaphore is locked 
```

### SizedGroup

`SizedGroup` combines `Semaphore` and `WaitGroup` to provide a wait group that allows a limited number of goroutines to run in parallel.

By default, locking happens inside the goroutine. This means every call will be non-blocking, but some goroutines may wait if the semaphore is locked. Technically, it doesn't limit the number of goroutines but rather the number of running (active) goroutines.

To block goroutines from starting, use the `Preemptive` option. Important: With `Preemptive`, the `Go` call can block. If the maximum size is reached, the call will wait until the number of running goroutines drops below the maximum. This not only limits the number of running goroutines but also the number of waiting goroutines.


```go
	swg := syncs.NewSizedGroup(5) // wait group with max size=5
	for i :=0; i<10; i++ {
		swg.Go(func(ctx context.Context){
			doThings(ctx) // only 5 of these will run in parallel
	    })
	}
	swg.Wait()
```

Another option is `Discard`, which will skip (won't start) goroutines if the semaphore is locked. In other words, if a defined number of goroutines are already running, the call will be discarded. `Discard` is useful when you don't care about the results of extra goroutines; i.e., you just want to run some tasks in parallel but can allow some number of them to be ignored. This flag sets `Preemptive` as well, because otherwise, it doesn't make sense.


```go
	swg := syncs.NewSizedGroup(5, Discard) // wait group with max size=5 and discarding extra goroutines
	for i :=0; i<10; i++ {
		swg.Go(func(ctx context.Context){
			doThings(ctx) // only 5 of these will run in parallel and 5 other can be discarded
		})
	}
	swg.Wait()
```


### ErrSizedGroup

`ErrSizedGroup` is a `SizedGroup` with error control. It works the same as `errgrp.Group`, i.e., it returns the first error. 
It can work as a regular errgrp.Group or with early termination. It is thread-safe.


`ErrSizedGroup` supports both in-goroutine-wait as well as outside of goroutine wait with `Preemptive` and `Discard` options (see above). Other options include `TermOnErr`, which skips (won't start) all other goroutines if any error is returned, and `Context` for early termination/timeouts.


```go
	ewg := syncs.NewErrSizedGroup(5, syncs.Preemptive) // error wait group with max size=5, don't try to start more if any error happened
	for i :=0; i<10; i++ {
		ewg.Go(func(ctx context.Context) error { // Go here could be blocked if trying to run >5 at the same time 
			err := doThings(ctx)     // only 5 of these will run in parallel
			return err
		})
	}
	err := ewg.Wait()
```

