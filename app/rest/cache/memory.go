package cache

import (
	"log"
	"sync/atomic"

	"github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
)

// memoryCache implements LoadingCache interface on top of cache.Cache (go-cache)
type memoryCache struct {
	bytesCache   *lru.Cache
	postFlushFn  func()
	maxKeys      int
	maxValueSize int
	maxCacheSize int64
	currentSize  int64
}

// NewMemoryCache makes memoryCache implementation
func NewMemoryCache(options ...Option) (LoadingCache, error) {
	res := memoryCache{
		postFlushFn:  func() {},
		maxKeys:      1000,
		maxValueSize: 0,
	}
	for _, opt := range options {
		if err := opt(&res); err != nil {
			log.Printf("[WARN] failed to set cache option, %v", err)
		}
	}

	onEvicted := func(key interface{}, value interface{}) {
		size := len(value.([]byte))
		atomic.AddInt64(&res.currentSize, -1*int64(size))
	}

	var err error
	// OnEvicted called automatically for expired and manually deleted
	if res.bytesCache, err = lru.NewWithEvict(res.maxKeys, onEvicted); err != nil {
		return nil, errors.Wrap(err, "failed to make cache")
	}

	log.Printf("[DEBUG] create lru cache, maxKeys=%d, maxValueSize=%d", res.maxKeys, res.maxValueSize)
	return &res, nil
}

// Get is loading cache method to get value by key or load via fn if not found
func (m *memoryCache) Get(key string, fn func() ([]byte, error)) (data []byte, err error) {
	if b, ok := m.bytesCache.Get(key); ok {
		return b.([]byte), nil
	}

	if data, err = fn(); err != nil {
		return data, err
	}
	if m.allowed(data) {
		m.bytesCache.Add(key, data)
		atomic.AddInt64(&m.currentSize, int64(len(data)))

		if m.maxCacheSize > 0 && atomic.LoadInt64(&m.currentSize) > m.maxCacheSize {
			for atomic.LoadInt64(&m.currentSize) > m.maxCacheSize {
				m.bytesCache.RemoveOldest()
			}
		}
	}
	return data, nil
}

// Flush clears cache and calls postFlushFn async
func (m *memoryCache) Flush(scopes ...string) {

	if len(scopes) == 0 {
		m.bytesCache.Purge()
		go m.postFlushFn()
		return
	}

	// check if fullKey has matching scopes
	inScope := func(fullKey string) bool {
		for _, s := range scopes {
			_, keyScopes, err := ParseKey(fullKey)
			if err != nil {
				return false
			}
			for _, ks := range keyScopes {
				if ks == s {
					return true
				}
			}
		}
		return false
	}

	// all matchedKeys should be collected first
	// we can't delete it from locked section, it will lock on eviction callback
	matchedKeys := []string{}
	for _, k := range m.bytesCache.Keys() {
		key := k.(string)
		if inScope(key) {
			matchedKeys = append(matchedKeys, key)
		}
	}
	for _, mkey := range matchedKeys {
		m.bytesCache.Remove(mkey)
	}

	if m.postFlushFn != nil {
		go m.postFlushFn()
	}
}

func (m *memoryCache) allowed(data []byte) bool {
	if m.maxValueSize > 0 && len(data) >= m.maxValueSize {
		return false
	}
	return true
}
