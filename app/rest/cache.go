package rest

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

// LoadingCache defines interface for caching
type LoadingCache interface {
	Get(key string, ttl time.Duration, fn func() ([]byte, error)) (data []byte, err error)
	Flush()
}

// loadingCache implements LoadingCache interface on top of cache.Cache (go-cache)
type loadingCache struct {
	bytesCache        *cache.Cache
	postFlushFn       func()
	defaultExpiration time.Duration
	cleanupInterval   time.Duration
	maxKeys           int
	maxValueSize      int
}

// NewLoadingCache makes loadingCache implementation
func NewLoadingCache(options ...CacheOption) LoadingCache {
	res := loadingCache{
		defaultExpiration: time.Hour,
		cleanupInterval:   5 * time.Minute,
		postFlushFn:       func() {},
		maxKeys:           0,
		maxValueSize:      0,
	}
	for _, opt := range options {
		if err := opt(&res); err != nil {
			log.Printf("[WARN] failed to set cache option, %v", err)
		}
	}
	res.bytesCache = cache.New(res.defaultExpiration, res.cleanupInterval)
	log.Printf("[DEBUG] create cache with cleanupInterval=%s, maxKeys=%d, maxValueSize=%d",
		res.cleanupInterval, res.maxKeys, res.maxValueSize)

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
		lc.bytesCache.Set(key, data, ttl)
	}
	return data, nil
}

// Flush clears cache and calls postFlushFn async
func (lc *loadingCache) Flush() {
	lc.bytesCache.Flush()
	if lc.postFlushFn != nil {
		go lc.postFlushFn()
	}
}

func (lc *loadingCache) allowed(data []byte) bool {
	if lc.maxValueSize > 0 && len(data) >= lc.maxValueSize {
		return false
	}
	if lc.maxKeys > 0 && lc.bytesCache.ItemCount() >= lc.maxKeys {
		return false
	}
	return true
}

// CacheOption func type
type CacheOption func(lc *loadingCache) error

// MaxValueSize functional option defines the largest value's size allowed to be cached
// By default it is 0, which means unlimited.
func MaxValueSize(max int) CacheOption {
	return func(lc *loadingCache) error {
		lc.maxValueSize = max
		return nil
	}
}

// MaxKeys functional option defines how many keys to keep.
// By default it is 0, which means unlimited.
func MaxKeys(max int) CacheOption {
	return func(lc *loadingCache) error {
		lc.maxKeys = max
		return nil
	}
}

// CleanupInterval functional option defines how often cleanup loop activated.
func CleanupInterval(interval time.Duration) CacheOption {
	return func(lc *loadingCache) error {
		lc.cleanupInterval = interval
		return nil
	}
}

// PostFlushFn functional option defines how callback function called after each Flush.
func PostFlushFn(postFlushFn func()) CacheOption {
	return func(lc *loadingCache) error {
		lc.postFlushFn = postFlushFn
		return nil
	}
}

// URLKey gets url from request to use it as cache key
// admins will have different keys in order to prevent leak of admin-only data to regular users
func URLKey(r *http.Request) string {
	adminPrefix := "admin!!"
	key := strings.TrimPrefix(r.URL.String(), adminPrefix)     // prevents attach with fake url to get admin view
	if user, err := GetUserInfo(r); err == nil && user.Admin { // make separate cache key for admins
		key = adminPrefix + key
	}
	return key
}
