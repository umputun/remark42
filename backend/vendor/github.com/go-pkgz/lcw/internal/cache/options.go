package cache

import "time"

// Option func type
type Option func(lc *LoadingCache) error

// OnEvicted called automatically for expired and manually deleted entries
func OnEvicted(fn func(key string, value interface{})) Option {
	return func(lc *LoadingCache) error {
		lc.onEvicted = fn
		return nil
	}
}

// PurgeEvery functional option defines purge interval
// by default it is 0, i.e. never. If MaxKeys set to any non-zero this default will be 5minutes
func PurgeEvery(interval time.Duration) Option {
	return func(lc *LoadingCache) error {
		lc.purgeEvery = interval
		return nil
	}
}

// MaxKeys functional option defines how many keys to keep.
// By default it is 0, which means unlimited.
// If any non-zero MaxKeys set, default PurgeEvery will be set to 5 minutes
func MaxKeys(max int) Option {
	return func(lc *LoadingCache) error {
		lc.maxKeys = int64(max)
		return nil
	}
}

// TTL functional option defines TTL for all cache entries.
// By default it is set to 10 years, sane option for expirable cache might be 5 minutes.
func TTL(ttl time.Duration) Option {
	return func(lc *LoadingCache) error {
		lc.ttl = ttl
		return nil
	}
}
