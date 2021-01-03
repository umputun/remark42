package lcw

import (
	"strings"

	"github.com/pkg/errors"
)

// Scache wraps LoadingCache with partitions (sub-system), and scopes.
// Simplified interface with just 4 funcs - Get, Flush, Stats and Close
type Scache struct {
	lc LoadingCache
}

// NewScache creates Scache on top of LoadingCache
func NewScache(lc LoadingCache) *Scache {
	return &Scache{lc: lc}
}

// Get retrieves a key from underlying backend
func (m *Scache) Get(key Key, fn func() ([]byte, error)) (data []byte, err error) {
	keyStr := key.String()
	val, err := m.lc.Get(keyStr, func() (value interface{}, e error) {
		return fn()
	})
	return val.([]byte), err
}

// Stat delegates the call to the underlying cache backend
func (m *Scache) Stat() CacheStat {
	return m.lc.Stat()
}

// Close calls Close function of the underlying cache
func (m *Scache) Close() error {
	return m.lc.Close()
}

// Flush clears cache and calls postFlushFn async
func (m *Scache) Flush(req FlusherRequest) {
	if len(req.scopes) == 0 {
		m.lc.Purge()
		return
	}

	// check if fullKey has matching scopes
	inScope := func(fullKey string) bool {
		key, err := parseKey(fullKey)
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

	for _, k := range m.lc.Keys() {
		if inScope(k) {
			m.lc.Delete(k) // Keys() returns copy of cache's key, safe to remove directly
		}
	}
}

// Key for scoped cache. Created foe given partition (can be empty) and set with ID and Scopes.
// example: k := NewKey("sys1").ID(postID).Scopes("last_posts", customer_id)
type Key struct {
	id        string   // the primary part of the key, i.e. usual cache's key
	partition string   // optional id for a subsystem or cache partition
	scopes    []string // list of scopes to use in invalidation
}

// NewKey makes base key for given partition. Partition can be omitted.
func NewKey(partition ...string) Key {
	if len(partition) == 0 {
		return Key{partition: ""}
	}
	return Key{partition: partition[0]}
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

// String makes full string key from primary key, partition and scopes
// key string made as <partition>@@<id>@@<scope1>$$<scope2>....
func (k Key) String() string {
	bld := strings.Builder{}
	_, _ = bld.WriteString(k.partition)
	_, _ = bld.WriteString("@@")
	_, _ = bld.WriteString(k.id)
	_, _ = bld.WriteString("@@")
	_, _ = bld.WriteString(strings.Join(k.scopes, "$$"))
	return bld.String()
}

// parseKey gets compound key string created by Key func and split it to the actual key, partition and scopes
// key string made as <partition>@@<id>@@<scope1>$$<scope2>....
func parseKey(keyStr string) (Key, error) {
	elems := strings.Split(keyStr, "@@")
	if len(elems) != 3 {
		return Key{}, errors.Errorf("can't parse cache key %s, invalid number of segments %d", keyStr, len(elems))
	}

	scopes := strings.Split(elems[2], "$$")
	if len(scopes) == 1 && scopes[0] == "" {
		scopes = []string{}
	}
	key := Key{
		partition: elems[0],
		id:        elems[1],
		scopes:    scopes,
	}

	return key, nil
}

// FlusherRequest used as input for cache.Flush
type FlusherRequest struct {
	partition string
	scopes    []string
}

// Flusher makes new FlusherRequest with empty scopes
func Flusher(partition string) FlusherRequest {
	res := FlusherRequest{partition: partition}
	return res
}

// Scopes adds scopes to FlusherRequest
func (f FlusherRequest) Scopes(scopes ...string) FlusherRequest {
	f.scopes = scopes
	return f
}
