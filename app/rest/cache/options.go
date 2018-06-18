package cache

import "github.com/pkg/errors"

// Option func type
type Option func(lc *memoryCache) error

// MaxValSize functional option defines the largest value's size allowed to be cached
// By default it is 0, which means unlimited.
func MaxValSize(max int) Option {
	return func(lc *memoryCache) error {
		lc.maxValueSize = max
		if max <= 0 {
			return errors.Errorf("negative size for MaxValSize, %d", max)
		}
		return nil
	}
}

// MaxKeys functional option defines how many keys to keep.
// By default it is 0, which means unlimited.
func MaxKeys(max int) Option {
	return func(lc *memoryCache) error {
		lc.maxKeys = max
		if max <= 0 {
			return errors.Errorf("negative size for MaxKeys, %d", max)
		}
		return nil
	}
}

// MaxCacheSize functional option defines the total size of cached data.
// By default it is 0, which means unlimited.
func MaxCacheSize(max int64) Option {
	return func(lc *memoryCache) error {
		lc.maxCacheSize = max
		if max <= 0 {
			return errors.Errorf("negative size or MaxCacheSize, %d", max)
		}
		return nil
	}
}

// PostFlushFn functional option defines how callback function called after each Flush.
func PostFlushFn(postFlushFn func()) Option {
	return func(lc *memoryCache) error {
		lc.postFlushFn = postFlushFn
		return nil
	}
}
