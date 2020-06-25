package lcw

import (
	"errors"
	"time"

	"github.com/go-pkgz/lcw/eventbus"
)

type options struct {
	maxKeys      int
	maxValueSize int
	maxKeySize   int
	maxCacheSize int64
	ttl          time.Duration
	onEvicted    func(key string, value Value)
	eventBus     eventbus.PubSub
}

// Option func type
type Option func(o *options) error

// MaxValSize functional option defines the largest value's size allowed to be cached
// By default it is 0, which means unlimited.
func MaxValSize(max int) Option {
	return func(o *options) error {
		if max < 0 {
			return errors.New("negative max value size")
		}
		o.maxValueSize = max
		return nil
	}
}

// MaxKeySize functional option defines the largest key's size allowed to be used in cache
// By default it is 0, which means unlimited.
func MaxKeySize(max int) Option {
	return func(o *options) error {
		if max < 0 {
			return errors.New("negative max key size")
		}
		o.maxKeySize = max
		return nil
	}
}

// MaxKeys functional option defines how many keys to keep.
// By default it is 0, which means unlimited.
func MaxKeys(max int) Option {
	return func(o *options) error {
		if max < 0 {
			return errors.New("negative max keys")
		}
		o.maxKeys = max
		return nil
	}
}

// MaxCacheSize functional option defines the total size of cached data.
// By default it is 0, which means unlimited.
func MaxCacheSize(max int64) Option {
	return func(o *options) error {
		if max < 0 {
			return errors.New("negative max cache size")
		}
		o.maxCacheSize = max
		return nil
	}
}

// TTL functional option defines duration.
// Works for ExpirableCache only
func TTL(ttl time.Duration) Option {
	return func(o *options) error {
		if ttl < 0 {
			return errors.New("negative ttl")
		}
		o.ttl = ttl
		return nil
	}
}

// OnEvicted sets callback on invalidation event
func OnEvicted(fn func(key string, value Value)) Option {
	return func(o *options) error {
		o.onEvicted = fn
		return nil
	}
}

// EventBus sets PubSub for distributed cache invalidation
func EventBus(pubSub eventbus.PubSub) Option {
	return func(o *options) error {
		o.eventBus = pubSub
		return nil
	}
}
