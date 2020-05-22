// Package cache implements LoadingCache.
//
// Support LRC TTL-based eviction.
package cache

import (
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// LoadingCache provides expirable loading cache with LRC eviction.
type LoadingCache struct {
	purgeEvery time.Duration
	ttl        time.Duration
	maxKeys    int64
	done       chan struct{}
	onEvicted  func(key string, value interface{})

	sync.Mutex
	data map[string]*cacheItem
}

// noEvictionTTL - very long ttl to prevent eviction
const noEvictionTTL = time.Hour * 24 * 365 * 10

// NewLoadingCache returns a new expirable LRC cache, activates purge with purgeEvery (0 to never purge).
// Default MaxKeys is unlimited (0).
func NewLoadingCache(options ...Option) (*LoadingCache, error) {
	res := LoadingCache{
		data:       map[string]*cacheItem{},
		ttl:        noEvictionTTL,
		purgeEvery: 0,
		maxKeys:    0,
		done:       make(chan struct{}),
	}

	for _, opt := range options {
		if err := opt(&res); err != nil {
			return nil, errors.Wrap(err, "failed to set cache option")
		}
	}

	if res.maxKeys > 0 || res.purgeEvery > 0 {
		if res.purgeEvery == 0 {
			res.purgeEvery = time.Minute * 5 // non-zero purge enforced because maxKeys defined
		}
		go func(done <-chan struct{}) {
			ticker := time.NewTicker(res.purgeEvery)
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					res.Lock()
					res.purge(res.maxKeys)
					res.Unlock()
				}
			}
		}(res.done)
	}
	return &res, nil
}

// Set key
func (c *LoadingCache) Set(key string, value interface{}) {
	c.Lock()
	defer c.Unlock()

	now := time.Now()
	if _, ok := c.data[key]; !ok {
		c.data[key] = &cacheItem{}
	}
	c.data[key].data = value
	c.data[key].expiresAt = now.Add(c.ttl)

	// Enforced purge call in addition the one from the ticker
	// to limit the worst-case scenario with a lot of sets in the
	// short period of time (between two timed purge calls)
	if c.maxKeys > 0 && int64(len(c.data)) >= c.maxKeys*2 {
		c.purge(c.maxKeys)
	}
}

// Get returns the key value
func (c *LoadingCache) Get(key string) (interface{}, bool) {
	c.Lock()
	defer c.Unlock()
	value, ok := c.getValue(key)
	if !ok {
		return nil, false
	}
	return value, ok
}

// Peek returns the key value (or undefined if not found) without updating the "recently used"-ness of the key.
func (c *LoadingCache) Peek(key string) (interface{}, bool) {
	c.Lock()
	defer c.Unlock()
	value, ok := c.getValue(key)
	if !ok {
		return nil, false
	}
	return value, ok
}

// Invalidate key (item) from the cache
func (c *LoadingCache) Invalidate(key string) {
	c.Lock()
	if value, ok := c.data[key]; ok {
		delete(c.data, key)
		if c.onEvicted != nil {
			c.onEvicted(key, value.data)
		}
	}
	c.Unlock()
}

// InvalidateFn deletes multiple keys if predicate is true
func (c *LoadingCache) InvalidateFn(fn func(key string) bool) {
	c.Lock()
	for key, value := range c.data {
		if fn(key) {
			delete(c.data, key)
			if c.onEvicted != nil {
				c.onEvicted(key, value.data)
			}
		}
	}
	c.Unlock()
}

// Keys return slice of current keys in the cache
func (c *LoadingCache) Keys() []string {
	c.Lock()
	defer c.Unlock()
	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

// get value respecting the expiration, should be called with lock
func (c *LoadingCache) getValue(key string) (interface{}, bool) {
	value, ok := c.data[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(c.data[key].expiresAt) {
		return nil, false
	}
	return value.data, ok
}

// Purge clears the cache completely.
func (c *LoadingCache) Purge() {
	c.Lock()
	defer c.Unlock()
	for k, v := range c.data {
		delete(c.data, k)
		if c.onEvicted != nil {
			c.onEvicted(k, v.data)
		}
	}
}

// DeleteExpired clears cache of expired items
func (c *LoadingCache) DeleteExpired() {
	c.Lock()
	defer c.Unlock()
	c.purge(0)
}

// ItemCount return count of items in cache
func (c *LoadingCache) ItemCount() int {
	c.Lock()
	n := len(c.data)
	c.Unlock()
	return n
}

// Close cleans the cache and destroys running goroutines
func (c *LoadingCache) Close() {
	c.Lock()
	defer c.Unlock()
	close(c.done)
}

// keysWithTs includes list of keys with ts. This is for sorting keys
// in order to provide least recently added sorting for size-based eviction
type keysWithTs []struct {
	key string
	ts  time.Time
}

// purge records > maxKeys. Has to be called with lock!
// call with maxKeys 0 will only clear expired entries.
func (c *LoadingCache) purge(maxKeys int64) {
	kts := keysWithTs{}

	for key, value := range c.data {
		// ttl eviction
		if time.Now().After(c.data[key].expiresAt) {
			delete(c.data, key)
			if c.onEvicted != nil {
				c.onEvicted(key, value.data)
			}
		}

		// prepare list of keysWithTs for size eviction
		if maxKeys > 0 && int64(len(c.data)) > maxKeys {
			ts := c.data[key].expiresAt

			kts = append(kts, struct {
				key string
				ts  time.Time
			}{key, ts})
		}
	}

	// size eviction
	size := int64(len(c.data))
	if len(kts) > 0 {
		sort.Slice(kts, func(i int, j int) bool { return kts[i].ts.Before(kts[j].ts) })
		for d := 0; int64(d) < size-maxKeys; d++ {
			key := kts[d].key
			value := c.data[key].data
			delete(c.data, key)
			if c.onEvicted != nil {
				c.onEvicted(key, value)
			}
		}
	}
}

type cacheItem struct {
	expiresAt time.Time
	data      interface{}
}
