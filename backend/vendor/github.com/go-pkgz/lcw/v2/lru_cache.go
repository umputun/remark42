package lcw

import (
	"fmt"
	"sync/atomic"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/go-pkgz/lcw/v2/eventbus"
)

// LruCache wraps lru.LruCache with loading cache Get and size limits
type LruCache[V any] struct {
	Workers[V]
	CacheStat
	backend     *lru.Cache[string, V]
	currentSize int64
	id          string // uuid identifying cache instance
}

// NewLruCache makes LRU LoadingCache implementation, 1000 max keys by default
func NewLruCache[V any](opts ...Option[V]) (*LruCache[V], error) {
	res := LruCache[V]{
		Workers: Workers[V]{
			maxKeys:      1000,
			maxValueSize: 0,
			eventBus:     &eventbus.NopPubSub{},
		},
		id: uuid.New().String(),
	}
	for _, opt := range opts {
		if err := opt(&res.Workers); err != nil {
			return nil, fmt.Errorf("failed to set cache option: %w", err)
		}
	}

	err := res.init()
	return &res, err
}

func (c *LruCache[V]) init() error {
	if err := c.eventBus.Subscribe(c.onBusEvent); err != nil {
		return fmt.Errorf("can't subscribe to event bus: %w", err)
	}

	onEvicted := func(key string, value V) {
		if c.onEvicted != nil {
			c.onEvicted(key, value)
		}
		if s, ok := any(value).(Sizer); ok {
			size := s.Size()
			atomic.AddInt64(&c.currentSize, -1*int64(size))
		}
		_ = c.eventBus.Publish(c.id, key) // signal invalidation to other nodes
	}

	var err error
	// OnEvicted called automatically for expired and manually deleted
	if c.backend, err = lru.NewWithEvict[string, V](c.maxKeys, onEvicted); err != nil {
		return fmt.Errorf("failed to make lru cache backend: %w", err)
	}

	return nil
}

// Get gets value by key or load with fn if not found in cache
func (c *LruCache[V]) Get(key string, fn func() (V, error)) (data V, err error) {
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

	c.backend.Add(key, data)

	if s, ok := any(data).(Sizer); ok {
		atomic.AddInt64(&c.currentSize, int64(s.Size()))
		if c.maxCacheSize > 0 && atomic.LoadInt64(&c.currentSize) > c.maxCacheSize {
			for atomic.LoadInt64(&c.currentSize) > c.maxCacheSize {
				c.backend.RemoveOldest()
			}
		}
	}

	return data, nil
}

// Peek returns the key value (or undefined if not found) without updating the "recently used"-ness of the key.
func (c *LruCache[V]) Peek(key string) (V, bool) {
	return c.backend.Peek(key)
}

// Purge clears the cache completely.
func (c *LruCache[V]) Purge() {
	c.backend.Purge()
	atomic.StoreInt64(&c.currentSize, 0)
}

// Invalidate removes keys with passed predicate fn, i.e. fn(key) should be true to get evicted
func (c *LruCache[V]) Invalidate(fn func(key string) bool) {
	for _, k := range c.backend.Keys() { // Keys() returns copy of cache's key, safe to remove directly
		if fn(k) {
			c.backend.Remove(k)
		}
	}
}

// Delete cache item by key
func (c *LruCache[V]) Delete(key string) {
	c.backend.Remove(key)
}

// Keys returns cache keys
func (c *LruCache[V]) Keys() (res []string) {
	return c.backend.Keys()
}

// Stat returns cache statistics
func (c *LruCache[V]) Stat() CacheStat {
	return CacheStat{
		Hits:   c.Hits,
		Misses: c.Misses,
		Size:   c.size(),
		Keys:   c.keys(),
		Errors: c.Errors,
	}
}

// Close does nothing for this type of cache
func (c *LruCache[V]) Close() error {
	return nil
}

// onBusEvent reacts on invalidation message triggered by event bus from another cache instance
func (c *LruCache[V]) onBusEvent(id, key string) {
	if id != c.id && c.backend.Contains(key) { // prevent reaction on event from this cache
		c.backend.Remove(key)
	}
}

func (c *LruCache[V]) size() int64 {
	return atomic.LoadInt64(&c.currentSize)
}

func (c *LruCache[V]) keys() int {
	return c.backend.Len()
}

func (c *LruCache[V]) allowed(key string, data V) bool {
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
