package cache

import "time"

type options[K comparable, V any] interface {
	WithTTL(ttl time.Duration) Cache[K, V]
	WithMaxKeys(maxKeys int) Cache[K, V]
	WithLRU() Cache[K, V]
	WithOnEvicted(fn func(key K, value V)) Cache[K, V]
}

// WithTTL functional option defines TTL for all cache entries.
// By default, it is set to 10 years, sane option for expirable cache might be 5 minutes.
func (c *cacheImpl[K, V]) WithTTL(ttl time.Duration) Cache[K, V] {
	c.ttl = ttl
	return c
}

// WithMaxKeys functional option defines how many keys to keep.
// By default, it is 0, which means unlimited.
func (c *cacheImpl[K, V]) WithMaxKeys(maxKeys int) Cache[K, V] {
	c.maxKeys = maxKeys
	return c
}

// WithLRU sets cache to LRU (Least Recently Used) eviction mode.
func (c *cacheImpl[K, V]) WithLRU() Cache[K, V] {
	c.isLRU = true
	return c
}

// WithOnEvicted defined function which would be called automatically for automatically and manually deleted entries
func (c *cacheImpl[K, V]) WithOnEvicted(fn func(key K, value V)) Cache[K, V] {
	c.onEvicted = fn
	return c
}
