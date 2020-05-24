# expirable-cache

[![Build Status](https://github.com/go-pkgz/expirable-cache/workflows/build/badge.svg)](https://github.com/go-pkgz/expirable-cache/actions)
[![Coverage Status](https://coveralls.io/repos/github/go-pkgz/expirable-cache/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/expirable-cache?branch=master)
[![godoc](https://godoc.org/github.com/go-pkgz/expirable-cache?status.svg)](https://pkg.go.dev/github.com/go-pkgz/expirable-cache?tab=doc)

Package cache implements expirable cache.

- Support LRC, LRU and TTL-based eviction.
- Package is thread-safe and doesn't spawn any goroutines.
- On every Set() call, cache deletes single oldest entry in case it's expired.
- In case MaxSize is set, cache deletes the oldest entry disregarding its expiration date to maintain the size,
either using LRC or LRU eviction.
- In case of default TTL (10 years) and default MaxSize (0, unlimited) the cache will be truly unlimited
 and will never delete entries from itself automatically.

**Important**: only reliable way of not having expired entries stuck in a cache is to
run cache.DeleteExpired periodically using [time.Ticker](https://golang.org/pkg/time/#Ticker),
advisable period is 1/2 of TTL.

This cache is heavily inspired by [hashicorp/golang-lru](https://github.com/hashicorp/golang-lru) _simplelru_ implementation.

### Usage example

```go
package main

import (
	"fmt"
	"time"

	"github.com/go-pkgz/expirable-cache"
)

func main() {
	// make cache with short TTL and 3 max keys
	c, _ := cache.NewCache(cache.MaxKeys(3), cache.TTL(time.Millisecond*10))

	// set value under key1.
	// with 0 ttl (last parameter) will use cache-wide setting instead (10ms).
	c.Set("key1", "val1", 0)

	// get value under key1
	r, ok := c.Get("key1")

	// check for OK value, because otherwise return would be nil and
	// type conversion will panic
	if ok {
		rstr := r.(string) // convert cached value from interface{} to real type
		fmt.Printf("value before expiration is found: %v, value: %v\n", ok, rstr)
	}

	time.Sleep(time.Millisecond * 11)

	// get value under key1 after key expiration
	r, ok = c.Get("key1")
	// don't convert to string as with ok == false value would be nil
	fmt.Printf("value after expiration is found: %v, value: %v\n", ok, r)

	// set value under key2, would evict old entry because it is already expired.
	// ttl (last parameter) overrides cache-wide ttl.
	c.Set("key2", "val2", time.Minute*5)

	fmt.Printf("%+v\n", c)
	// Output:
	// value before expiration is found: true, value: val1
	// value after expiration is found: false, value: <nil>
	// Size: 1, Stats: {Hits:1 Misses:1 Added:2 Evicted:1} (50.0%)
}
```