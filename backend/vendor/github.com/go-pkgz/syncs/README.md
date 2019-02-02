# Syncs - additional synchronization primitives 

[![Build Status](https://travis-ci.org/go-pkgz/syncs.svg?branch=master)](https://travis-ci.org/go-pkgz/syncs) [![Go Report Card](https://goreportcard.com/badge/github.com/go-pkgz/syncs)](https://goreportcard.com/report/github.com/go-pkgz/syncs) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/syncs/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/syncs?branch=master)

Package syncs provides additional synchronization primitives.

## Install and update

`go get -u github.com/go-pkgz/syncs`

## Details

### Semaphore

Implements `sync.Locker` interface but for given capacity, thread safe. Lock increases count and Unlock - decreases. Unlock on 0 count will be blocked.

```go
    sema := syncs.NewSemaphore(10) // make semaphore with 10 initial capacity
    for i :=0; i<10; i++ {
        sema.Lock() // all 10 locks will pass, i.w. won't lock
    }
    sema.Lock() // this is 11 - will lock for real

    // in some other place/goroutine
    sema.Unlock() // decrease semaphore counter
```

### SizedGroup

Mix semaphore and WaitGroup to provide sized waiting group. The result is a wait group allowing limited number of goroutine to run in parallel.

The locking happens inside of goroutine, i.e. **every call will be non-blocked**, but some goroutines may wait if semaphore locked. It means - technically it doesn't limit number of goroutines, but rather number of running (active) goroutines. 

```go
    swg := syncs.NewSizedGroup(5) // wait group with max size=5
     for i :=0; i<10; i++ {
        swg.Go(fn func(){
            doThings() // only 5 of these will run in parallel
        })
    }
    swg.Wait()
```

### ErrSizedGroup

Sized error group is a SizedGroup with error control. 
Works the same as errgrp.Group, i.e. returns first error.
Can work as regular errgrp.Group or with early termination.
Thread safe.

Supports both in-goroutine-wait via `NewErrSizedGroup` as well as outside of goroutine wait with `Preemptive()` option. Another options are  `TermOnErr` which will skip (won't start) all other goroutines if any error returned, and `Context`.

Important! With `Preemptive` Go call **can block**. In case if maximum size reached the call will wait till number of running goroutines 
dropped under max. This way we not only limiting number of running goroutines but also number of waiting goroutines.


```go
    ewg := syncs.NewErrSizedGroup(5, syncs.Preemptive()) // error wait group with max size=5, don't try to start more if any error happened
     for i :=0; i<10; i++ {
        ewg.Go(fn func() error { // Go here could be blocked if trying to run >5 at the same time 
           err := doThings()     // only 5 of these will run in parallel
           return err
        })
    }
    err := ewg.Wait()
```