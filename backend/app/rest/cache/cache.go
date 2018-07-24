package cache

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/rest"
)

// LoadingCache defines interface for caching
type LoadingCache interface {
	Get(key Key, fn func() ([]byte, error)) (data []byte, err error)
	Flush(req FlusherRequest)
}

type cacheWithOpts interface {
	LoadingCache
	setMaxValSize(max int) error
	setMaxKeys(max int) error
	setMaxCacheSize(max int64) error
	setPostFlushFn(postFlushFn func()) error
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

// Nop does nothing for caching, passing fn call only
type Nop struct{}

// Get calls fn, no actual caching
func (n *Nop) Get(key *Key, fn func() ([]byte, error)) (data []byte, err error) {
	return fn()
}

// Flush does nothing for NoopCache
func (n *Nop) Flush(req *FlusherRequest) {}
