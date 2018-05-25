package cache

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/umputun/remark/app/rest"
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
	bytesCache        *cache.Cache
	postFlushFn       func()
	defaultExpiration time.Duration
	cleanupInterval   time.Duration
	maxKeys           int
	maxValueSize      int

	activeKeys map[string]struct{} // keep all current cached keys
	lock       sync.Mutex
}

// NewLoadingCache makes loadingCache implementation
func NewLoadingCache(options ...Option) LoadingCache {
	res := loadingCache{
		defaultExpiration: time.Hour,
		cleanupInterval:   5 * time.Minute,
		postFlushFn:       func() {},
		maxKeys:           0,
		maxValueSize:      0,
		activeKeys:        map[string]struct{}{},
	}
	for _, opt := range options {
		if err := opt(&res); err != nil {
			log.Printf("[WARN] failed to set cache option, %v", err)
		}
	}
	res.bytesCache = cache.New(res.defaultExpiration, res.cleanupInterval)

	// OnEvicted called automatically for expired and manually deleted
	res.bytesCache.OnEvicted(func(key string, _ interface{}) {
		res.withLock(func() { delete(res.activeKeys, key) })
	})

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
		lc.withLock(func() { lc.activeKeys[key] = struct{}{} })
	}
	return data, nil
}

func (lc *loadingCache) withLock(fn func()) {
	lc.lock.Lock()
	fn()
	lc.lock.Unlock()
}

// Flush clears cache and calls postFlushFn async
func (lc *loadingCache) Flush(scopes ...string) {

	if len(scopes) == 0 {
		lc.bytesCache.Flush()
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
	lc.withLock(func() {
		for k := range lc.activeKeys {
			if inScope(k) {
				matchedKeys = append(matchedKeys, k)
			}
		}
	})
	for _, mkey := range matchedKeys {
		lc.bytesCache.Delete(mkey)
	}

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

// Option func type
type Option func(lc *loadingCache) error

// MaxValSize functional option defines the largest value's size allowed to be cached
// By default it is 0, which means unlimited.
func MaxValSize(max int) Option {
	return func(lc *loadingCache) error {
		lc.maxValueSize = max
		return nil
	}
}

// MaxKeys functional option defines how many keys to keep.
// By default it is 0, which means unlimited.
func MaxKeys(max int) Option {
	return func(lc *loadingCache) error {
		lc.maxKeys = max
		return nil
	}
}

// CleanupInterval functional option defines how often cleanup loop activated.
func CleanupInterval(interval time.Duration) Option {
	return func(lc *loadingCache) error {
		lc.cleanupInterval = interval
		return nil
	}
}

// PostFlushFn functional option defines how callback function called after each Flush.
func PostFlushFn(postFlushFn func()) Option {
	return func(lc *loadingCache) error {
		lc.postFlushFn = postFlushFn
		return nil
	}
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
