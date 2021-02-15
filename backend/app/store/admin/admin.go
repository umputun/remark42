// Package admin defines and implements store for admin-level data like secret key, list of admins and so on
package admin

import (
	"errors"
	"strings"

	log "github.com/go-pkgz/lgr"
)

// Store defines interface returning admins info for given site
type Store interface {
	Key(siteID string) (key string, err error)
	Admins(siteID string) (ids []string, err error)
	Email(siteID string) (email string, err error)
	Enabled(siteID string) (ok bool, err error)
	OnEvent(siteID string, et EventType) error
}

// EventType indicates type of the event
type EventType int

// enum of all event types
const (
	EvCreate EventType = iota
	EvDelete
	EvUpdate
	EvVote
)

// StaticStore implements keys.Store with a single set of admins and email for all sites
type StaticStore struct {
	admins []string
	email  string
	key    string
	sites  []string
}

// NewStaticStore makes StaticStore instance with given key
func NewStaticStore(key string, sites, adminIDs []string, email string) *StaticStore {
	log.Printf("[DEBUG] admin users %+v, email %s", adminIDs, email)
	return &StaticStore{key: key, sites: sites, admins: adminIDs, email: email}
}

// NewStaticKeyStore is a shortcut for making StaticStore for key consumers only
func NewStaticKeyStore(key string) *StaticStore {
	return &StaticStore{key: key, admins: []string{}, email: ""}
}

// Key returns static key, same for all sites
func (s *StaticStore) Key(_ string) (key string, err error) {
	if s.key == "" {
		return "", errors.New("empty key for static key store")
	}
	return s.key, nil
}

// Admins returns static list of admin ids, the same for all sites
func (s *StaticStore) Admins(string) (ids []string, err error) {
	return s.admins, nil
}

// Email gets static email address
func (s *StaticStore) Email(string) (email string, err error) {
	return s.email, nil
}

// Enabled if always true for StaticStore
func (s *StaticStore) Enabled(site string) (ok bool, err error) {
	if len(s.sites) == 0 {
		return true, nil
	}
	for _, allowedSite := range s.sites {
		if strings.EqualFold(allowedSite, site) {
			return true, nil
		}
	}
	return false, nil
}

// OnEvent doesn nothing for StaticStore
func (s *StaticStore) OnEvent(_ string, _ EventType) error { return nil }
