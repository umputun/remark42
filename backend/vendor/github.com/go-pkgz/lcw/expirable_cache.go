package lcw

import (
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/go-pkgz/lcw/eventbus"
	"github.com/go-pkgz/lcw/internal/cache"
)

// ExpirableCache implements LoadingCache with TTL.
type ExpirableCache struct {
	options
	CacheStat
	currentSize int64
	id          string
	backend     *cache.LoadingCache
}

// NewExpirableCache makes expirable LoadingCache implementation, 1000 max keys by default and 5m TTL
func NewExpirableCache(opts ...Option) (*ExpirableCache, error) {
	res := ExpirableCache{
		options: options{
			maxKeys:      1000,
			maxValueSize: 0,
			ttl:          5 * time.Minute,
			eventBus:     &eventbus.NopPubSub{},
		},
		id: uuid.New().String(),
	}

	for _, opt := range opts {
		if err := opt(&res.options); err != nil {
			return nil, errors.Wrap(err, "failed to set cache option")
		}
	}

	if err := res.eventBus.Subscribe(res.onBusEvent); err != nil {
		return nil, errors.Wrapf(err, "can't subscribe to event bus")
	}

	backend, err := cache.NewLoadingCache(
		cache.MaxKeys(res.maxKeys),
		cache.TTL(res.ttl),
		cache.PurgeEvery(res.ttl/2),
		cache.OnEvicted(func(key string, value interface{}) {
			if res.onEvicted != nil {
				res.onEvicted(key, value)
			}
			if s, ok := value.(Sizer); ok {
				size := s.Size()
				atomic.AddInt64(&res.currentSize, -1*int64(size))
			}
			// ignore the error on Publish as we don't have log inside the module and
			// there is no other way to handle it: we publish the cache invalidation
			// and hope for the best
			_ = res.eventBus.Publish(res.id, key)
		}),
	)
	if err != nil {
		return nil, errors.Wrap(err, "error creating backend")
	}
	res.backend = backend

	return &res, nil
}

// Get gets value by key or load with fn if not found in cache
func (c *ExpirableCache) Get(key string, fn func() (interface{}, error)) (data interface{}, err error) {
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

	if s, ok := data.(Sizer); ok {
		if c.maxCacheSize > 0 && atomic.LoadInt64(&c.currentSize)+int64(s.Size()) >= c.maxCacheSize {
			c.backend.DeleteExpired()
			return data, nil
		}
		atomic.AddInt64(&c.currentSize, int64(s.Size()))
	}

	c.backend.Set(key, data)

	return data, nil
}

// Invalidate removes keys with passed predicate fn, i.e. fn(key) should be true to get evicted
func (c *ExpirableCache) Invalidate(fn func(key string) bool) {
	c.backend.InvalidateFn(fn)
}

// Peek returns the key value (or undefined if not found) without updating the "recently used"-ness of the key.
func (c *ExpirableCache) Peek(key string) (interface{}, bool) {
	return c.backend.Peek(key)
}

// Purge clears the cache completely.
func (c *ExpirableCache) Purge() {
	c.backend.Purge()
	atomic.StoreInt64(&c.currentSize, 0)
}

// Delete cache item by key
func (c *ExpirableCache) Delete(key string) {
	c.backend.Invalidate(key)
}

// Keys returns cache keys
func (c *ExpirableCache) Keys() (res []string) {
	return c.backend.Keys()
}

// Stat returns cache statistics
func (c *ExpirableCache) Stat() CacheStat {
	return CacheStat{
		Hits:   c.Hits,
		Misses: c.Misses,
		Size:   c.size(),
		Keys:   c.keys(),
		Errors: c.Errors,
	}
}

// Close kills cleanup goroutine
func (c *ExpirableCache) Close() error {
	c.backend.Close()
	return nil
}

// onBusEvent reacts on invalidation message triggered by event bus from another cache instance
func (c *ExpirableCache) onBusEvent(id, key string) {
	if id != c.id {
		c.backend.Invalidate(key)
	}
}

func (c *ExpirableCache) size() int64 {
	return atomic.LoadInt64(&c.currentSize)
}

func (c *ExpirableCache) keys() int {
	return c.backend.ItemCount()
}

func (c *ExpirableCache) allowed(key string, data interface{}) bool {
	if c.backend.ItemCount() >= c.maxKeys {
		return false
	}
	if c.maxKeySize > 0 && len(key) > c.maxKeySize {
		return false
	}
	if s, ok := data.(Sizer); ok {
		if c.maxValueSize > 0 && s.Size() >= c.maxValueSize {
			return false
		}
	}
	return true
}
