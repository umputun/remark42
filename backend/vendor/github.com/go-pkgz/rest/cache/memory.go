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
	log.Print("[INFO] make memory cache")

	res := memoryCache{
		postFlushFn:  func() {},
		maxKeys:      1000,
		maxValueSize: 0,
	}
	for _, opt := range options {
		if err := opt(&res); err != nil {
			return nil, errors.Wrap(err, "failed to set cache option")
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

	log.Printf("[DEBUG] create lru cache, maxKeys=%d, maxValueSize=%d, maxCacheSize=%d",
		res.maxKeys, res.maxValueSize, res.maxCacheSize)
	return &res, nil
}

// Get is loading cache method to get value by key or load via fn if not found
func (m *memoryCache) Get(key Key, fn func() ([]byte, error)) (data []byte, err error) {
	mkey := key.Merge()
	if b, ok := m.bytesCache.Get(mkey); ok {
		return b.([]byte), nil
	}

	if data, err = fn(); err != nil {
		return data, err
	}
	if m.allowed(data) {
		m.bytesCache.Add(mkey, data)
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
func (m *memoryCache) Flush(req FlusherRequest) {

	if len(req.scopes) == 0 {
		m.bytesCache.Purge()
		go m.postFlushFn()
		return
	}

	// check if fullKey has matching scopes
	inScope := func(fullKey string) bool {
		key, err := ParseKey(fullKey)
		if err != nil {
			return false
		}
		for _, s := range req.scopes {
			for _, ks := range key.scopes {
				if ks == s {
					return true
				}
			}
		}
		return false
	}

	for _, k := range m.bytesCache.Keys() {
		key := k.(string)
		if inScope(key) {
			m.bytesCache.Remove(key) // Keys() returns copy of cache's key, safe to remove directly
		}
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

func (m *memoryCache) setMaxValSize(max int) error {
	m.maxValueSize = max
	if max <= 0 {
		return errors.Errorf("negative size for MaxValSize, %d", max)
	}
	return nil
}

func (m *memoryCache) setMaxKeys(max int) error {
	m.maxKeys = max
	if max <= 0 {
		return errors.Errorf("negative size for MaxKeys, %d", max)
	}
	return nil
}

func (m *memoryCache) setMaxCacheSize(max int64) error {
	m.maxCacheSize = max
	if max <= 0 {
		return errors.Errorf("negative size or MaxCacheSize, %d", max)
	}
	return nil
}

func (m *memoryCache) setPostFlushFn(postFlushFn func()) error {
	m.postFlushFn = postFlushFn
	return nil
}
