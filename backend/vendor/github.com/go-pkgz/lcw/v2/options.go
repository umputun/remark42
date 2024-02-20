package lcw

import (
	"fmt"
	"time"

	"github.com/go-pkgz/lcw/v2/eventbus"
)

type Workers[V any] struct {
	maxKeys      int
	maxValueSize int
	maxKeySize   int
	maxCacheSize int64
	ttl          time.Duration
	onEvicted    func(key string, value V)
	eventBus     eventbus.PubSub
	strToV       func(string) V
}

// Option func type
type Option[V any] func(o *Workers[V]) error

// WorkerOptions holds the option setting methods
type WorkerOptions[T any] struct{}

// NewOpts creates a new WorkerOptions instance
func NewOpts[T any]() *WorkerOptions[T] {
	return &WorkerOptions[T]{}
}

// MaxValSize functional option defines the largest value's size allowed to be cached
// By default it is 0, which means unlimited.
func (o *WorkerOptions[V]) MaxValSize(max int) Option[V] {
	return func(o *Workers[V]) error {
		if max < 0 {
			return fmt.Errorf("negative max value size")
		}
		o.maxValueSize = max
		return nil
	}
}

// MaxKeySize functional option defines the largest key's size allowed to be used in cache
// By default it is 0, which means unlimited.
func (o *WorkerOptions[V]) MaxKeySize(max int) Option[V] {
	return func(o *Workers[V]) error {
		if max < 0 {
			return fmt.Errorf("negative max key size")
		}
		o.maxKeySize = max
		return nil
	}
}

// MaxKeys functional option defines how many keys to keep.
// By default, it is 0, which means unlimited.
func (o *WorkerOptions[V]) MaxKeys(max int) Option[V] {
	return func(o *Workers[V]) error {
		if max < 0 {
			return fmt.Errorf("negative max keys")
		}
		o.maxKeys = max
		return nil
	}
}

// MaxCacheSize functional option defines the total size of cached data.
// By default, it is 0, which means unlimited.
func (o *WorkerOptions[V]) MaxCacheSize(max int64) Option[V] {
	return func(o *Workers[V]) error {
		if max < 0 {
			return fmt.Errorf("negative max cache size")
		}
		o.maxCacheSize = max
		return nil
	}
}

// TTL functional option defines duration.
// Works for ExpirableCache only
func (o *WorkerOptions[V]) TTL(ttl time.Duration) Option[V] {
	return func(o *Workers[V]) error {
		if ttl < 0 {
			return fmt.Errorf("negative ttl")
		}
		o.ttl = ttl
		return nil
	}
}

// OnEvicted sets callback on invalidation event
func (o *WorkerOptions[V]) OnEvicted(fn func(key string, value V)) Option[V] {
	return func(o *Workers[V]) error {
		o.onEvicted = fn
		return nil
	}
}

// EventBus sets PubSub for distributed cache invalidation
func (o *WorkerOptions[V]) EventBus(pubSub eventbus.PubSub) Option[V] {
	return func(o *Workers[V]) error {
		o.eventBus = pubSub
		return nil
	}
}

// StrToV sets strToV function for RedisCache
func (o *WorkerOptions[V]) StrToV(fn func(string) V) Option[V] {
	return func(o *Workers[V]) error {
		o.strToV = fn
		return nil
	}
}
