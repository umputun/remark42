package keys

import "github.com/pkg/errors"

// Store defines interface returning key for given site
// this key used for JWT and HMAC hashes
type Store interface {
	Get(siteID string) (key string, err error)
}

// StaticStore implements keys.Store with a single, predefined key
type StaticStore struct {
	key string
}

// NewStaticStore makes StaticStore instance with given key
func NewStaticStore(key string) *StaticStore {
	return &StaticStore{key: key}
}

// Get returns static key for all sites, allows empty site
func (s *StaticStore) Get(siteID string) (key string, err error) {
	if s.key == "" {
		return "", errors.New("empty key for static key store")
	}
	return s.key, nil
}
