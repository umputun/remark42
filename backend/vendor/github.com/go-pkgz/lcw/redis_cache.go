package lcw

import (
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/pkg/errors"
)

// RedisValueSizeLimit is maximum allowed value size in Redis
const RedisValueSizeLimit = 512 * 1024 * 1024

// RedisCache implements LoadingCache for Redis.
type RedisCache struct {
	options
	CacheStat
	backend *redis.Client
}

// NewRedisCache makes Redis LoadingCache implementation.
func NewRedisCache(backend *redis.Client, opts ...Option) (*RedisCache, error) {
	res := RedisCache{
		options: options{
			ttl: 5 * time.Minute,
		},
	}
	for _, opt := range opts {
		if err := opt(&res.options); err != nil {
			return nil, errors.Wrap(err, "failed to set cache option")
		}
	}

	if res.maxValueSize <= 0 || res.maxValueSize > RedisValueSizeLimit {
		res.maxValueSize = RedisValueSizeLimit
	}

	res.backend = backend

	return &res, nil
}

// Get gets value by key or load with fn if not found in cache
func (c *RedisCache) Get(key string, fn func() (interface{}, error)) (data interface{}, err error) {
	v, getErr := c.backend.Get(key).Result()
	switch getErr {
	// RedisClient returns nil when find a key in DB
	case nil:
		atomic.AddInt64(&c.Hits, 1)
		return v, nil
	// RedisClient returns redis.Nil when doesn't find a key in DB
	case redis.Nil:
		if data, err = fn(); err != nil {
			atomic.AddInt64(&c.Errors, 1)
			return data, err
		}
	// RedisClient returns !nil when something goes wrong while get data
	default:
		atomic.AddInt64(&c.Errors, 1)
		return v, getErr
	}
	atomic.AddInt64(&c.Misses, 1)

	if !c.allowed(key, data) {
		return data, nil
	}

	_, setErr := c.backend.Set(key, data, c.ttl).Result()
	if setErr != nil {
		atomic.AddInt64(&c.Errors, 1)
		return data, setErr
	}

	return data, nil
}

// Invalidate removes keys with passed predicate fn, i.e. fn(key) should be true to get evicted
func (c *RedisCache) Invalidate(fn func(key string) bool) {
	for _, key := range c.backend.Keys("*").Val() { // Keys() returns copy of cache's key, safe to remove directly
		if fn(key) {
			c.backend.Del(key)
		}
	}
}

// Peek returns the key value (or undefined if not found) without updating the "recently used"-ness of the key.
func (c *RedisCache) Peek(key string) (interface{}, bool) {
	ret, err := c.backend.Get(key).Result()
	if err != nil {
		return nil, false
	}
	return ret, true
}

// Purge clears the cache completely.
func (c *RedisCache) Purge() {
	c.backend.FlushDB()

}

// Delete cache item by key
func (c *RedisCache) Delete(key string) {
	c.backend.Del(key)
}

// Keys gets all keys for the cache
func (c *RedisCache) Keys() (res []string) {
	return c.backend.Keys("*").Val()
}

// Stat returns cache statistics
func (c *RedisCache) Stat() CacheStat {
	return CacheStat{
		Hits:   c.Hits,
		Misses: c.Misses,
		Size:   c.size(),
		Keys:   c.keys(),
		Errors: c.Errors,
	}
}

// Close closes underlying connections
func (c *RedisCache) Close() error {
	return c.backend.Close()
}

func (c *RedisCache) size() int64 {
	return 0
}

func (c *RedisCache) keys() int {
	return int(c.backend.DBSize().Val())
}

func (c *RedisCache) allowed(key string, data interface{}) bool {
	if c.maxKeys > 0 && c.backend.DBSize().Val() >= int64(c.maxKeys) {
		return false
	}
	if c.maxKeySize > 0 && len(key) > c.maxKeySize {
		return false
	}
	if s, ok := data.(Sizer); ok {
		if c.maxValueSize > 0 && (s.Size() >= c.maxValueSize) {
			return false
		}
	}
	return true
}
