package rest

import (
	"log"
	"time"

	cache "github.com/patrickmn/go-cache"
)

type loadingCache struct {
	bytesCache *cache.Cache
}

func newLoadingCache(defaultExpiration, cleanupInterval time.Duration) *loadingCache {
	return &loadingCache{bytesCache: cache.New(defaultExpiration, cleanupInterval)}
}

func (lc *loadingCache) get(key string, ttl time.Duration, fn func() ([]byte, error)) (data []byte, err error) {
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

func (lc *loadingCache) flush() {
	lc.bytesCache.Flush()
}
