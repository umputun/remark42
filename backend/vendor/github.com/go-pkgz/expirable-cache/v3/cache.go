// Package cache implements Cache similar to hashicorp/golang-lru
//
// Support LRC, LRU and TTL-based eviction.
// Package is thread-safe and doesn't spawn any goroutines.
// On every Set() call, cache deletes single oldest entry in case it's expired.
// In case MaxSize is set, cache deletes the oldest entry disregarding its expiration date to maintain the size,
// either using LRC or LRU eviction.
// In case of default TTL (10 years) and default MaxSize (0, unlimited) the cache will be truly unlimited
// and will never delete entries from itself automatically.
//
// Important: only reliable way of not having expired entries stuck in a cache is to
// run cache.DeleteExpired periodically using time.Ticker, advisable period is 1/2 of TTL.
package cache

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

// Cache defines cache interface
type Cache[K comparable, V any] interface {
	fmt.Stringer
	options[K, V]
	Add(key K, value V) bool
	Set(key K, value V, ttl time.Duration)
	Get(key K) (V, bool)
	GetExpiration(key K) (time.Time, bool)
	GetOldest() (K, V, bool)
	Contains(key K) (ok bool)
	Peek(key K) (V, bool)
	Values() []V
	Keys() []K
	Len() int
	Remove(key K) bool
	Invalidate(key K)
	InvalidateFn(fn func(key K) bool)
	RemoveOldest() (K, V, bool)
	DeleteExpired()
	Purge()
	Resize(int) int
	Stat() Stats
}

// Stats provides statistics for cache
type Stats struct {
	Hits, Misses   int // cache effectiveness
	Added, Evicted int // number of added and evicted records
}

// cacheImpl provides Cache interface implementation.
type cacheImpl[K comparable, V any] struct {
	ttl       time.Duration
	maxKeys   int
	isLRU     bool
	onEvicted func(key K, value V)

	sync.Mutex
	stat      Stats
	items     map[K]*list.Element
	evictList *list.List
}

// noEvictionTTL - very long ttl to prevent eviction
const noEvictionTTL = time.Hour * 24 * 365 * 10

// NewCache returns a new Cache.
// Default MaxKeys is unlimited (0).
// Default TTL is 10 years, sane value for expirable cache is 5 minutes.
// Default eviction mode is LRC, appropriate option allow to change it to LRU.
func NewCache[K comparable, V any]() Cache[K, V] {
	return &cacheImpl[K, V]{
		items:     map[K]*list.Element{},
		evictList: list.New(),
		ttl:       noEvictionTTL,
		maxKeys:   0,
	}
}

// Add adds a value to the cache. Returns true if an eviction occurred.
// Returns false if there was no eviction: the item was already in the cache,
// or the size was not exceeded.
func (c *cacheImpl[K, V]) Add(key K, value V) (evicted bool) {
	return c.addWithTTL(key, value, c.ttl)
}

// Set key, ttl of 0 would use cache-wide TTL
func (c *cacheImpl[K, V]) Set(key K, value V, ttl time.Duration) {
	c.addWithTTL(key, value, ttl)
}

// Returns true if an eviction occurred.
// Returns false if there was no eviction: the item was already in the cache,
// or the size was not exceeded.
func (c *cacheImpl[K, V]) addWithTTL(key K, value V, ttl time.Duration) (evicted bool) {
	c.Lock()
	defer c.Unlock()
	now := time.Now()
	if ttl == 0 {
		ttl = c.ttl
	}

	// Check for existing item
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		ent.Value.(*cacheItem[K, V]).value = value
		ent.Value.(*cacheItem[K, V]).expiresAt = now.Add(ttl)
		return false
	}

	// Add new item
	ent := &cacheItem[K, V]{key: key, value: value, expiresAt: now.Add(ttl)}
	entry := c.evictList.PushFront(ent)
	c.items[key] = entry
	c.stat.Added++

	// Remove the oldest entry if it is expired, only in case of non-default TTL.
	if c.ttl != noEvictionTTL || ttl != noEvictionTTL {
		c.removeOldestIfExpired()
	}

	evict := c.maxKeys > 0 && len(c.items) > c.maxKeys
	// Verify size not exceeded
	if evict {
		c.removeOldest()
	}
	return evict
}

// Get returns the key value if it's not expired
func (c *cacheImpl[K, V]) Get(key K) (V, bool) {
	def := *new(V)
	c.Lock()
	defer c.Unlock()
	if ent, ok := c.items[key]; ok {
		// Expired item check
		if time.Now().After(ent.Value.(*cacheItem[K, V]).expiresAt) {
			c.stat.Misses++
			return ent.Value.(*cacheItem[K, V]).value, false
		}
		if c.isLRU {
			c.evictList.MoveToFront(ent)
		}
		c.stat.Hits++
		return ent.Value.(*cacheItem[K, V]).value, true
	}
	c.stat.Misses++
	return def, false
}

// Contains checks if a key is in the cache, without updating the recent-ness
// or deleting it for being stale.
func (c *cacheImpl[K, V]) Contains(key K) (ok bool) {
	c.Lock()
	defer c.Unlock()
	_, ok = c.items[key]
	return ok
}

// Peek returns the key value (or undefined if not found) without updating the "recently used"-ness of the key.
// Works exactly the same as Get in case of LRC mode (default one).
func (c *cacheImpl[K, V]) Peek(key K) (V, bool) {
	def := *new(V)
	c.Lock()
	defer c.Unlock()
	if ent, ok := c.items[key]; ok {
		// Expired item check
		if time.Now().After(ent.Value.(*cacheItem[K, V]).expiresAt) {
			c.stat.Misses++
			return ent.Value.(*cacheItem[K, V]).value, false
		}
		c.stat.Hits++
		return ent.Value.(*cacheItem[K, V]).value, true
	}
	c.stat.Misses++
	return def, false
}

// GetExpiration returns the expiration time of the key. Non-existing key returns zero time.
func (c *cacheImpl[K, V]) GetExpiration(key K) (time.Time, bool) {
	c.Lock()
	defer c.Unlock()
	if ent, ok := c.items[key]; ok {
		return ent.Value.(*cacheItem[K, V]).expiresAt, true
	}
	return time.Time{}, false
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *cacheImpl[K, V]) Keys() []K {
	c.Lock()
	defer c.Unlock()
	return c.keys()
}

// Values returns a slice of the values in the cache, from oldest to newest.
// Expired entries are filtered out.
func (c *cacheImpl[K, V]) Values() []V {
	c.Lock()
	defer c.Unlock()
	values := make([]V, 0, len(c.items))
	now := time.Now()
	for ent := c.evictList.Back(); ent != nil; ent = ent.Prev() {
		if now.After(ent.Value.(*cacheItem[K, V]).expiresAt) {
			continue
		}
		values = append(values, ent.Value.(*cacheItem[K, V]).value)
	}
	return values
}

// Len return count of items in cache, including expired
func (c *cacheImpl[K, V]) Len() int {
	c.Lock()
	defer c.Unlock()
	return c.evictList.Len()
}

// Resize changes the cache size. Size of 0 means unlimited.
func (c *cacheImpl[K, V]) Resize(size int) int {
	c.Lock()
	defer c.Unlock()
	if size <= 0 {
		c.maxKeys = 0
		return 0
	}
	diff := c.evictList.Len() - size
	if diff < 0 {
		diff = 0
	}
	for i := 0; i < diff; i++ {
		c.removeOldest()
	}
	c.maxKeys = size
	return diff
}

// Invalidate key (item) from the cache
func (c *cacheImpl[K, V]) Invalidate(key K) {
	c.Lock()
	defer c.Unlock()
	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
	}
}

// InvalidateFn deletes multiple keys if predicate is true
func (c *cacheImpl[K, V]) InvalidateFn(fn func(key K) bool) {
	c.Lock()
	defer c.Unlock()
	for key, ent := range c.items {
		if fn(key) {
			c.removeElement(ent)
		}
	}
}

// Remove removes the provided key from the cache, returning if the
// key was contained.
func (c *cacheImpl[K, V]) Remove(key K) bool {
	c.Lock()
	defer c.Unlock()
	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
		return true
	}
	return false
}

// RemoveOldest remove the oldest element in the cache
func (c *cacheImpl[K, V]) RemoveOldest() (key K, value V, ok bool) {
	c.Lock()
	defer c.Unlock()
	if ent := c.evictList.Back(); ent != nil {
		c.removeElement(ent)
		return ent.Value.(*cacheItem[K, V]).key, ent.Value.(*cacheItem[K, V]).value, true
	}
	return
}

// GetOldest returns the oldest entry
func (c *cacheImpl[K, V]) GetOldest() (key K, value V, ok bool) {
	c.Lock()
	defer c.Unlock()
	if ent := c.evictList.Back(); ent != nil {
		return ent.Value.(*cacheItem[K, V]).key, ent.Value.(*cacheItem[K, V]).value, true
	}
	return
}

// DeleteExpired clears cache of expired items
func (c *cacheImpl[K, V]) DeleteExpired() {
	c.Lock()
	defer c.Unlock()
	for _, key := range c.keys() {
		if time.Now().After(c.items[key].Value.(*cacheItem[K, V]).expiresAt) {
			c.removeElement(c.items[key])
		}
	}
}

// Purge clears the cache completely.
func (c *cacheImpl[K, V]) Purge() {
	c.Lock()
	defer c.Unlock()
	for k, v := range c.items {
		delete(c.items, k)
		c.stat.Evicted++
		if c.onEvicted != nil {
			c.onEvicted(k, v.Value.(*cacheItem[K, V]).value)
		}
	}
	c.evictList.Init()
}

// Stat gets the current stats for cache
func (c *cacheImpl[K, V]) Stat() Stats {
	c.Lock()
	defer c.Unlock()
	return c.stat
}

func (c *cacheImpl[K, V]) String() string {
	stats := c.Stat()
	size := c.Len()
	return fmt.Sprintf("Size: %d, Stats: %+v (%0.1f%%)", size, stats, 100*float64(stats.Hits)/float64(stats.Hits+stats.Misses))
}

// Keys returns a slice of the keys in the cache, from oldest to newest. Has to be called with lock!
func (c *cacheImpl[K, V]) keys() []K {
	keys := make([]K, 0, len(c.items))
	for ent := c.evictList.Back(); ent != nil; ent = ent.Prev() {
		keys = append(keys, ent.Value.(*cacheItem[K, V]).key)
	}
	return keys
}

// removeOldest removes the oldest item from the cache. Has to be called with lock!
func (c *cacheImpl[K, V]) removeOldest() {
	ent := c.evictList.Back()
	if ent != nil {
		c.removeElement(ent)
	}
}

// removeOldest removes the oldest item from the cache in case it's already expired. Has to be called with lock!
func (c *cacheImpl[K, V]) removeOldestIfExpired() {
	ent := c.evictList.Back()
	if ent != nil && time.Now().After(ent.Value.(*cacheItem[K, V]).expiresAt) {
		c.removeElement(ent)
	}
}

// removeElement is used to remove a given list element from the cache. Has to be called with lock!
func (c *cacheImpl[K, V]) removeElement(e *list.Element) {
	c.evictList.Remove(e)
	kv := e.Value.(*cacheItem[K, V])
	delete(c.items, kv.key)
	c.stat.Evicted++
	if c.onEvicted != nil {
		c.onEvicted(kv.key, kv.value)
	}
}

// cacheItem is used to hold a value in the evictList
type cacheItem[K comparable, V any] struct {
	expiresAt time.Time
	key       K
	value     V
}
