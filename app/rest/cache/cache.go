package cache

import (
	"log"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/umputun/remark/app/rest"
)

// LoadingCache defines interface for caching
type LoadingCache interface {
	Get(key string, fn func() ([]byte, error)) (data []byte, err error)
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
	maxCacheSize int64
	currentSize  int64
}

// NewLoadingCache makes loadingCache implementation
func NewLoadingCache(options ...Option) (LoadingCache, error) {
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
func (lc *loadingCache) Get(key string, fn func() ([]byte, error)) (data []byte, err error) {
	if b, ok := lc.bytesCache.Get(key); ok {
		return b.([]byte), nil
	}

	if data, err = fn(); err != nil {
		return data, err
	}
	if lc.allowed(data) {
		lc.bytesCache.Add(key, data)
		atomic.AddInt64(&lc.currentSize, int64(len(data)))

		if lc.maxCacheSize > 0 && atomic.LoadInt64(&lc.currentSize) > lc.maxCacheSize {
			for atomic.LoadInt64(&lc.currentSize) > lc.maxCacheSize {
				lc.bytesCache.RemoveOldest()
			}
		}
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

// URLKey gets url from request to use it as cache key
// admins will have different keys in order to prevent leak of admin-only data to regular users
func URLKey(r *http.Request) string {
	adminPrefix := "admin!!"
	key := strings.TrimPrefix(r.URL.String(), adminPrefix)          // prevents attach with fake url to get admin view
	if user, err := rest.GetUserInfo(r); err == nil && user.Admin { // make separate cache key for admins
		key = adminPrefix + key
	}
	return key
}
