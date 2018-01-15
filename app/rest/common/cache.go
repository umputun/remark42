package common

import (
	"log"
	"time"

	cache "github.com/patrickmn/go-cache"
)

// LoadingCache defines interface for caching
type LoadingCache interface {
	Get(key string, ttl time.Duration, fn func() ([]byte, error)) (data []byte, err error)
	Flush()
}

// loadingCache implements LoadingCache interface on top of cache.Cache
type loadingCache struct {
	bytesCache *cache.Cache
}

// NewLoadingCache makes loadingCache implementation
func NewLoadingCache(defaultExpiration, cleanupInterval time.Duration) LoadingCache {
	return &loadingCache{bytesCache: cache.New(defaultExpiration, cleanupInterval)}
}

func (lc *loadingCache) Get(key string, ttl time.Duration, fn func() ([]byte, error)) (data []byte, err error) {
	if b, ok := lc.bytesCache.Get(key); ok {
		log.Printf("[DEBUG] cache hit %s", key)
		return b.([]byte), nil
	}

	log.Printf("[DEBUG] cache miss %s", key)
	if data, err = fn(); err != nil {
		return data, err
	}
	lc.bytesCache.Set(key, data, ttl)
	return data, nil
}

func (lc *loadingCache) Flush() {
	lc.bytesCache.Flush()
}
