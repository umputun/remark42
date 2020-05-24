package cache

import "time"

// Option func type
type Option func(lc *cacheImpl) error

// OnEvicted called automatically for automatically and manually deleted entries
func OnEvicted(fn func(key string, value interface{})) Option {
	return func(lc *cacheImpl) error {
		lc.onEvicted = fn
		return nil
	}
}

// MaxKeys functional option defines how many keys to keep.
// By default it is 0, which means unlimited.
func MaxKeys(max int) Option {
	return func(lc *cacheImpl) error {
		lc.maxKeys = max
		return nil
	}
}

// TTL functional option defines TTL for all cache entries.
// By default it is set to 10 years, sane option for expirable cache might be 5 minutes.
func TTL(ttl time.Duration) Option {
	return func(lc *cacheImpl) error {
		lc.ttl = ttl
		return nil
	}
}

// LRU sets cache to LRU (Least Recently Used) eviction mode.
func LRU() Option {
	return func(lc *cacheImpl) error {
		lc.isLRU = true
		return nil
	}
}
