/*
 * Copyright 2019 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package plugin

import (
	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
)

// MemAdminStore implements admin.Store with mongo backend
type MemAdminStore struct {
	data map[string]AdminRec
	key  string
}

// AdminRec is a records per site with all admin info in
type AdminRec struct {
	SiteID string   `bson:"site"`
	IDs    []string `bson:"ids"`
	Email  string   `bson:"email"`
}

const mongoAdmin = "admin"

// NewMemAdminStore makes admin Store in memory
func NewMemAdminStore(key string) *MemAdminStore {
	log.Print("[DEBUG] make memory admin store")
	return &MemAdminStore{data: map[string]AdminRec{}, key: key}
}

// Key executes find by siteID and returns substructure with secret key
func (m *MemAdminStore) Key() (key string, err error) {
	return m.key, nil
}

// Admins executes find by siteID and returns admins ids
func (m *MemAdminStore) Admins(siteID string) (ids []string, err error) {
	resp, ok := m.data[siteID]
	if !ok {
		return nil, errors.Errorf("site %s not found", siteID)
	}

	return resp.IDs, nil
}

// Email executes find by siteID and returns admin's email
func (m *MemAdminStore) Email(siteID string) (email string, err error) {
	resp, ok := m.data[siteID]
	if !ok {
		return "", errors.Errorf("site %s not found", siteID)
	}

	return resp.Email, nil
}
