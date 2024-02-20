package lcw

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/go-pkgz/lcw/v2/eventbus"
)

// ExpirableCache implements LoadingCache with TTL.
type ExpirableCache[V any] struct {
	Workers[V]
	CacheStat
	currentSize int64
	id          string
	backend     *expirable.LRU[string, V]
}

// NewExpirableCache makes expirable LoadingCache implementation, 1000 max keys by default and 5m TTL
func NewExpirableCache[V any](opts ...Option[V]) (*ExpirableCache[V], error) {
	res := ExpirableCache[V]{
		Workers: Workers[V]{
			maxKeys:      1000,
			maxValueSize: 0,
			ttl:          5 * time.Minute,
			eventBus:     &eventbus.NopPubSub{},
		},
		id: uuid.New().String(),
	}

	for _, opt := range opts {
		if err := opt(&res.Workers); err != nil {
			return nil, fmt.Errorf("failed to set cache option: %w", err)
		}
	}

	if err := res.eventBus.Subscribe(res.onBusEvent); err != nil {
		return nil, fmt.Errorf("can't subscribe to event bus: %w", err)
	}

	res.backend = expirable.NewLRU[string, V](res.maxKeys, func(key string, value V) {
		if res.onEvicted != nil {
			res.onEvicted(key, value)
		}
		if s, ok := any(value).(Sizer); ok {
			size := s.Size()
			atomic.AddInt64(&res.currentSize, -1*int64(size))
		}
		// ignore the error on Publish as we don't have log inside the module and
		// there is no other way to handle it: we publish the cache invalidation
		// and hope for the best
		_ = res.eventBus.Publish(res.id, key)
	}, res.ttl)

	return &res, nil
}

// Get gets value by key or load with fn if not found in cache
func (c *ExpirableCache[V]) Get(key string, fn func() (V, error)) (data V, err error) {
	if v, ok := c.backend.Get(key); ok {
		atomic.AddInt64(&c.Hits, 1)
		return v, nil
	}

	if data, err = fn(); err != nil {
		atomic.AddInt64(&c.Errors, 1)
		return data, err
	}
	atomic.AddInt64(&c.Misses, 1)

	if !c.allowed(key, data) {
		return data, nil
	}

	if s, ok := any(data).(Sizer); ok {
		if c.maxCacheSize > 0 && atomic.LoadInt64(&c.currentSize)+int64(s.Size()) >= c.maxCacheSize {
			return data, nil
		}
		atomic.AddInt64(&c.currentSize, int64(s.Size()))
	}

	c.backend.Add(key, data)

	return data, nil
}

// Invalidate removes keys with passed predicate fn, i.e. fn(key) should be true to get evicted
func (c *ExpirableCache[V]) Invalidate(fn func(key string) bool) {
	for _, key := range c.backend.Keys() {
		if fn(key) {
			c.backend.Remove(key)
		}
	}
}

// Peek returns the key value (or undefined if not found) without updating the "recently used"-ness of the key.
func (c *ExpirableCache[V]) Peek(key string) (V, bool) {
	return c.backend.Peek(key)
}

// Purge clears the cache completely.
func (c *ExpirableCache[V]) Purge() {
	c.backend.Purge()
	atomic.StoreInt64(&c.currentSize, 0)
}

// Delete cache item by key
func (c *ExpirableCache[V]) Delete(key string) {
	c.backend.Remove(key)
}

// Keys returns cache keys
func (c *ExpirableCache[V]) Keys() (res []string) {
	return c.backend.Keys()
}

// Stat returns cache statistics
func (c *ExpirableCache[V]) Stat() CacheStat {
	return CacheStat{
		Hits:   c.Hits,
		Misses: c.Misses,
		Size:   c.size(),
		Keys:   c.keys(),
		Errors: c.Errors,
	}
}

// Close supposed to kill cleanup goroutine,
// but it's not possible before https://github.com/hashicorp/golang-lru/issues/159 is solved
// so for now it just cleans it.
func (c *ExpirableCache[V]) Close() error {
	c.backend.Purge()
	atomic.StoreInt64(&c.currentSize, 0)
	return nil
}

// onBusEvent reacts on invalidation message triggered by event bus from another cache instance
func (c *ExpirableCache[V]) onBusEvent(id, key string) {
	if id != c.id {
		c.backend.Remove(key)
	}
}

func (c *ExpirableCache[V]) size() int64 {
	return atomic.LoadInt64(&c.currentSize)
}

func (c *ExpirableCache[V]) keys() int {
	return c.backend.Len()
}

func (c *ExpirableCache[V]) allowed(key string, data V) bool {
	if c.backend.Len() >= c.maxKeys {
		return false
	}
	if c.maxKeySize > 0 && len(key) > c.maxKeySize {
		return false
	}
	if s, ok := any(data).(Sizer); ok {
		if c.maxValueSize > 0 && s.Size() >= c.maxValueSize {
			return false
		}
	}
	return true
}
