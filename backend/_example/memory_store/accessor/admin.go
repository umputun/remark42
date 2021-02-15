/*
 * Copyright 2019 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package accessor

import (
	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"

	"github.com/umputun/remark42/backend/app/store/admin"
)

// MemAdmin implements admin.Store with memory backend
type MemAdmin struct {
	data map[string]AdminRec // admin info per site
	key  string
}

// AdminRec is a records per site with all admin info in
type AdminRec struct {
	SiteID       string
	IDs          []string // admin ids
	Email        string   // admin email
	Enabled      bool     // site enabled
	CountCreated int64    // number of created posts
}

// NewMemAdminStore makes admin Store in memory
func NewMemAdminStore(key string) *MemAdmin {
	log.Print("[DEBUG] make memory admin store")
	return &MemAdmin{data: map[string]AdminRec{}, key: key}
}

// Key executes find by siteID and returns substructure with secret key
func (m *MemAdmin) Key(_ string) (key string, err error) {
	return m.key, nil
}

// Admins executes find by siteID and returns admins ids
func (m *MemAdmin) Admins(siteID string) (ids []string, err error) {
	resp, ok := m.data[siteID]
	if !ok {
		return nil, errors.Errorf("site %s not found", siteID)
	}
	log.Printf("[DEBUG] admins for %s, %+v", siteID, resp.IDs)
	return resp.IDs, nil
}

// Email executes find by siteID and returns admin's email
func (m *MemAdmin) Email(siteID string) (email string, err error) {
	resp, ok := m.data[siteID]
	if !ok {
		return "", errors.Errorf("site %s not found", siteID)
	}

	return resp.Email, nil
}

// Enabled return
func (m *MemAdmin) Enabled(siteID string) (ok bool, err error) {
	resp, ok := m.data[siteID]
	if !ok {
		return false, errors.Errorf("site %s not found", siteID)
	}
	return resp.Enabled, nil
}

// OnEvent reacts on events from updates, created, delete and vote
func (m *MemAdmin) OnEvent(siteID string, ev admin.EventType) error {
	resp, ok := m.data[siteID]
	if !ok {
		return errors.Errorf("site %s not found", siteID)
	}
	if ev == admin.EvCreate {
		resp.CountCreated++ // not a good idea, just for demo
		m.data[siteID] = resp
	}
	return nil
}

// Set admin data for siteID
func (m *MemAdmin) Set(siteID string, arec AdminRec) {
	m.data[siteID] = arec
}
