// Package admin defines and implements store for admin-level data like secret key, list of admins and so on
package admin

import (
	"errors"
	"log"
)

// Store defines interface returning admins info for given site
type Store interface {
	Key(siteID string) (key string, err error)
	Admins(siteID string) (ids []string)
	Email(siteID string) (email string)
}

// StaticStore implements keys.Store with a single, predefined key
type StaticStore struct {
	admins []string
	email  string
	key    string
}

// Key returns static key for all sites, allows empty site
func (s *StaticStore) Key(siteID string) (key string, err error) {
	if s.key == "" {
		return "", errors.New("empty key for static key store")
	}
	return s.key, nil
}

// NewStaticStore makes StaticStore instance with given key
func NewStaticStore(key string, admins []string, email string) *StaticStore {
	log.Printf("[DEBUG] admin users %+v, email %s", admins, email)
	return &StaticStore{key: key, admins: admins, email: email}
}

// NewStaticKeyStore is a shortcut for making StaticStore for key consumers only
func NewStaticKeyStore(key string) *StaticStore {
	return &StaticStore{key: key, admins: []string{}, email: ""}
}

// Admins returns static list of admin's ids, the same for all sites
func (s *StaticStore) Admins(string) (ids []string) {
	return s.admins
}

// Email gets static email address
func (s *StaticStore) Email(string) (email string) {
	return s.email
}
