// Package cache implements a wrapper on top of hashicorp/golang-lru with guava-style loader func.
// In addition for Get (i.e. load from cache if found and put to cache if absent) it adds a Key struct
// with record ID, site ID and invalidation scopes. Scopes used to evict matching records.
// Additional limits added for total cache size, mac number of keys and max size of the value. If exceeded it won't
// put to cache but will call loader func.
// Usually the cache involved on []byte response level, i.e. post-marshaling right before response's send.
package cache

import (
	"strings"

	"github.com/pkg/errors"
)

// LoadingCache defines interface for caching
type LoadingCache interface {
	Get(key Key, fn func() ([]byte, error)) (data []byte, err error) // load from cache if found or put to cache and return
	Flush(req FlusherRequest)                                        // evict matched records
}

type cacheWithOpts interface {
	LoadingCache
	setMaxValSize(max int) error             // max value size, in bytes
	setMaxKeys(max int) error                // max number of keys
	setMaxCacheSize(max int64) error         // max cache size (total values) in bytes
	setPostFlushFn(postFlushFn func()) error // optional callback after flush
}

// Key for cache
type Key struct {
	id     string
	siteID string
	scopes []string
}

// NewKey makes keys for site
func NewKey(site string) Key {
	res := Key{siteID: site}
	return res
}

// ID sets key id
func (k Key) ID(id string) Key {
	k.id = id
	return k
}

// Scopes of the key
func (k Key) Scopes(scopes ...string) Key {
	k.scopes = scopes
	return k
}

// Merge makes full string key from primary key and scopes
func (k Key) Merge() string {
	return strings.Join(k.scopes, "$$") + "@@" + k.id + "@@" + k.siteID
}

// ParseKey gets compound key created by Key func and split it to the actual key and scopes
func ParseKey(fullKey string) (Key, error) {
	elems := strings.Split(fullKey, "@@")
	if len(elems) != 3 {
		return Key{}, errors.Errorf("can't parse cache key %s", fullKey)
	}
	scopes := strings.Split(elems[0], "$$")
	if len(scopes) == 1 && scopes[0] == "" {
		scopes = []string{}
	}
	key := Key{
		scopes: scopes,
		id:     elems[1],
		siteID: elems[2],
	}
	return key, nil
}

// FlusherRequest used as input for cache.Flush
type FlusherRequest struct {
	siteID string
	scopes []string
}

// Flusher makes new FlusherRequest with empty scopes
func Flusher(siteID string) FlusherRequest {
	res := FlusherRequest{siteID: siteID}
	return res
}

// Scopes adds scopes to FlusherRequest
func (f FlusherRequest) Scopes(scopes ...string) FlusherRequest {
	f.scopes = scopes
	return f
}

// Nop does nothing for caching, passing fn call only
type Nop struct{}

// Get calls fn, no actual caching
func (n *Nop) Get(key Key, fn func() ([]byte, error)) (data []byte, err error) {
	return fn()
}

// Flush does nothing for NoopCache
func (n *Nop) Flush(req FlusherRequest) {}
