package lcw

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisValueSizeLimit is maximum allowed value size in Redis
const RedisValueSizeLimit = 512 * 1024 * 1024

// RedisCache implements LoadingCache for Redis.
type RedisCache[V any] struct {
	Workers[V]
	CacheStat
	backend *redis.Client
}

// NewRedisCache makes Redis LoadingCache implementation.
// Supports only string and string-based types and will return error otherwise.
func NewRedisCache[V any](backend *redis.Client, opts ...Option[V]) (*RedisCache[V], error) {
	// check if V is string, not underlying type but directly, and otherwise return error if strToV is nil as it should be defined

	res := RedisCache[V]{
		Workers: Workers[V]{
			ttl: 5 * time.Minute,
		},
	}
	for _, opt := range opts {
		if err := opt(&res.Workers); err != nil {
			return nil, fmt.Errorf("failed to set cache option: %w", err)
		}
	}

	// check if underlying type is string, so we can safely store it in Redis
	var v V
	if reflect.TypeOf(v).Kind() != reflect.String {
		return nil, fmt.Errorf("can't store non-string types in Redis cache")
	}
	switch any(v).(type) {
	case string:
	// check strToV option only for string-like but non string types
	default:
		if res.strToV == nil {
			return nil, fmt.Errorf("StrToV option should be set for string-like type")
		}
	}

	if res.maxValueSize <= 0 || res.maxValueSize > RedisValueSizeLimit {
		res.maxValueSize = RedisValueSizeLimit
	}

	res.backend = backend

	return &res, nil
}

// Get gets value by key or load with fn if not found in cache
func (c *RedisCache[V]) Get(key string, fn func() (V, error)) (data V, err error) {
	v, getErr := c.backend.Get(context.Background(), key).Result()
	switch {
	// RedisClient returns nil when find a key in DB
	case getErr == nil:
		atomic.AddInt64(&c.Hits, 1)
		switch any(data).(type) {
		case string:
			return any(v).(V), nil
		default:
			return c.strToV(v), nil
		}
	// RedisClient returns redis.Nil when doesn't find a key in DB
	case errors.Is(getErr, redis.Nil):
		if data, err = fn(); err != nil {
			atomic.AddInt64(&c.Errors, 1)
			return data, err
		}
		// RedisClient returns !nil when something goes wrong while get data
	default:
		atomic.AddInt64(&c.Errors, 1)
		switch any(data).(type) {
		case string:
			return any(v).(V), getErr
		default:
			return c.strToV(v), getErr
		}
	}
	atomic.AddInt64(&c.Misses, 1)

	if !c.allowed(key, data) {
		return data, nil
	}

	_, setErr := c.backend.Set(context.Background(), key, data, c.ttl).Result()
	if setErr != nil {
		atomic.AddInt64(&c.Errors, 1)
		return data, setErr
	}

	return data, nil
}

// Invalidate removes keys with passed predicate fn, i.e. fn(key) should be true to get evicted
func (c *RedisCache[V]) Invalidate(fn func(key string) bool) {
	for _, key := range c.backend.Keys(context.Background(), "*").Val() { // Keys() returns copy of cache's key, safe to remove directly
		if fn(key) {
			c.backend.Del(context.Background(), key)
		}
	}
}

// Peek returns the key value (or undefined if not found) without updating the "recently used"-ness of the key.
func (c *RedisCache[V]) Peek(key string) (data V, found bool) {
	ret, err := c.backend.Get(context.Background(), key).Result()
	if err != nil {
		var emptyValue V
		return emptyValue, false
	}
	switch any(data).(type) {
	case string:
		return any(ret).(V), true
	default:
		return any(ret).(V), true
	}
}

// Purge clears the cache completely.
func (c *RedisCache[V]) Purge() {
	c.backend.FlushDB(context.Background())

}

// Delete cache item by key
func (c *RedisCache[V]) Delete(key string) {
	c.backend.Del(context.Background(), key)
}

// Keys gets all keys for the cache
func (c *RedisCache[V]) Keys() (res []string) {
	return c.backend.Keys(context.Background(), "*").Val()
}

// Stat returns cache statistics
func (c *RedisCache[V]) Stat() CacheStat {
	return CacheStat{
		Hits:   c.Hits,
		Misses: c.Misses,
		Size:   c.size(),
		Keys:   c.keys(),
		Errors: c.Errors,
	}
}

// Close closes underlying connections
func (c *RedisCache[V]) Close() error {
	return c.backend.Close()
}

func (c *RedisCache[V]) size() int64 {
	return 0
}

func (c *RedisCache[V]) keys() int {
	return int(c.backend.DBSize(context.Background()).Val())
}

func (c *RedisCache[V]) allowed(key string, data V) bool {
	if c.maxKeys > 0 && c.backend.DBSize(context.Background()).Val() >= int64(c.maxKeys) {
		return false
	}
	if c.maxKeySize > 0 && len(key) > c.maxKeySize {
		return false
	}
	if s, ok := any(data).(Sizer); ok {
		if c.maxValueSize > 0 && (s.Size() >= c.maxValueSize) {
			return false
		}
	}
	return true
}
