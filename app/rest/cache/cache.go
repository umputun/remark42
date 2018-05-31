package cache

import (
	"log"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
)

// LoadingCache defines interface for caching
type LoadingCache interface {
	Get(key string, ttl time.Duration, fn func() ([]byte, error)) (data []byte, err error)
	Flush(scopes ...string)
}

// Key makes full key from primary key and scopes
func Key(key string, scopes ...string) string {
	return strings.Join(scopes, "$$") + "@@" + key
}

func parseKey(fullKey string) (key string, scopes []string, err error) {
	elems := strings.Split(fullKey, "@@")
	if len(elems) != 2 {
		return "", nil, errors.Errorf("can't parse cache key %s", key)
	}
	scopes = strings.Split(elems[0], "$$")
	if len(scopes) == 1 && scopes[0] == "" {
		scopes = []string{}
	}
	key = elems[1]
	return key, scopes, nil
}

// loadingCache implements LoadingCache interface on top of cache.Cache (go-cache)
type loadingCache struct {
	bytesCache   *lru.Cache
	postFlushFn  func()
	maxKeys      int
	maxValueSize int
}

// NewLoadingCache makes loadingCache implementation
func NewLoadingCache(options ...Option) LoadingCache {
	res := loadingCache{
		postFlushFn:  func() {},
		maxKeys:      1000,
		maxValueSize: 0,
	}
	for _, opt := range options {
		if err := opt(&res); err != nil {
			log.Printf("[WARN] failed to set cache option, %v", err)
		}
	}

	// OnEvicted called automatically for expired and manually deleted
	res.bytesCache, _ = lru.New(res.maxKeys)

	log.Printf("[DEBUG] create lru cache, maxKeys=%d, maxValueSize=%d", res.maxKeys, res.maxValueSize)
	return &res
}

// Get is loading cache method to get value by key or load via fn if not found
func (lc *loadingCache) Get(key string, ttl time.Duration, fn func() ([]byte, error)) (data []byte, err error) {
	if b, ok := lc.bytesCache.Get(key); ok {
		return b.([]byte), nil
	}

	if data, err = fn(); err != nil {
		return data, err
	}
	if lc.allowed(data) {
		lc.bytesCache.Add(key, data)
	}
	return data, nil
}

// Flush clears cache and calls postFlushFn async
func (lc *loadingCache) Flush(scopes ...string) {

	if len(scopes) == 0 {
		lc.bytesCache.Purge()
		go lc.postFlushFn()
		return
	}

	// check if fullKey has matching scopes
	inScope := func(fullKey string) bool {
		for _, s := range scopes {
			_, keyScopes, err := parseKey(fullKey)
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
	for _, k := range lc.bytesCache.Keys() {
		key := k.(string)
		if inScope(key) {
			matchedKeys = append(matchedKeys, key)
		}
	}
	for _, mkey := range matchedKeys {
		lc.bytesCache.Remove(mkey)
	}

	if lc.postFlushFn != nil {
		go lc.postFlushFn()
	}
}

func (lc *loadingCache) allowed(data []byte) bool {
	if lc.maxValueSize > 0 && len(data) >= lc.maxValueSize {
		return false
	}
	return true
}
