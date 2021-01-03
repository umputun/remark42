// Package lcw adds a thin layer on top of lru and expirable cache providing more limits and common interface.
// The primary method to get (and set) data to/from the cache is LoadingCache.Get returning stored data for a given key or
// call provided func to retrieve and store, similar to Guava loading cache.
// Limits allow max values for key size, number of keys, value size and total size of values in the cache.
// CacheStat gives general stats on cache performance.
// 3 flavors of cache provided - NoP (do-nothing cache), ExpirableCache (TTL based), and LruCache
package lcw

//go:generate sh -c "mockery -inpkg -name LoadingCache -print > /tmp/cache-mock.tmp && mv /tmp/cache-mock.tmp cache_mock.go"

import (
	"fmt"
)

// Sizer allows to perform size-based restrictions, optional.
// If not defined both maxValueSize and maxCacheSize checks will be ignored
type Sizer interface {
	Size() int
}

// LoadingCache defines guava-like cache with Get method returning cached value ao retrieving it if not in cache
type LoadingCache interface {
	Get(key string, fn func() (interface{}, error)) (val interface{}, err error) // load or get from cache
	Peek(key string) (interface{}, bool)                                         // get from cache by key
	Invalidate(fn func(key string) bool)                                         // invalidate items for func(key) == true
	Delete(key string)                                                           // delete by key
	Purge()                                                                      // clear cache
	Stat() CacheStat                                                             // cache stats
	Keys() []string                                                              // list of all keys
	Close() error                                                                // close open connections
}

// CacheStat represent stats values
type CacheStat struct {
	Hits   int64
	Misses int64
	Keys   int
	Size   int64
	Errors int64
}

// String formats cache stats
func (s CacheStat) String() string {
	ratio := 0.0
	if s.Hits+s.Misses > 0 {
		ratio = float64(s.Hits) / float64(s.Hits+s.Misses)
	}
	return fmt.Sprintf("{hits:%d, misses:%d, ratio:%.2f, keys:%d, size:%d, errors:%d}",
		s.Hits, s.Misses, ratio, s.Keys, s.Size, s.Errors)
}

// Nop is do-nothing implementation of LoadingCache
type Nop struct{}

// NewNopCache makes new do-nothing cache
func NewNopCache() *Nop {
	return &Nop{}
}

// Get calls fn without any caching
func (n *Nop) Get(key string, fn func() (interface{}, error)) (interface{}, error) { return fn() }

// Peek does nothing and always returns false
func (n *Nop) Peek(key string) (interface{}, bool) { return nil, false }

// Invalidate does nothing for nop cache
func (n *Nop) Invalidate(fn func(key string) bool) {}

// Purge does nothing for nop cache
func (n *Nop) Purge() {}

// Delete does nothing for nop cache
func (n *Nop) Delete(key string) {}

// Keys does nothing for nop cache
func (n *Nop) Keys() []string { return nil }

// Stat always 0s for nop cache
func (n *Nop) Stat() CacheStat {
	return CacheStat{}
}

// Close does nothing for nop cache
func (n *Nop) Close() error {
	return nil
}
