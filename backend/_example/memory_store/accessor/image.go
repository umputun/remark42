/*
 * Copyright 2020 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package accessor

import (
	"context"
	"path"
	"sync"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
	"github.com/rs/xid"
)

// MemImage implements image.Store with memory backend
type MemImage struct {
	imagesStaging map[string][]byte
	images        map[string][]byte
	insertTime    map[string]time.Time
	sync.RWMutex
}

// NewMemImageStore makes admin Store in memory.
func NewMemImageStore() *MemImage {
	log.Print("[DEBUG] make memory image store")
	return &MemImage{
		imagesStaging: map[string][]byte{},
		images:        map[string][]byte{},
		insertTime:    map[string]time.Time{},
	}
}

func (m *MemImage) Save(userID string, img []byte) (id string, err error) {
	id = path.Join(userID, guid())
	return m.SaveWithID(id, img)
}

func (m *MemImage) SaveWithID(id string, img []byte) (string, error) {
	m.Lock()
	m.imagesStaging[id] = img
	m.insertTime[id] = time.Now()
	m.Unlock()

	return id, nil
}

func (m *MemImage) Load(id string) ([]byte, error) {
	m.RLock()
	img, ok := m.images[id]
	if !ok {
		img, ok = m.imagesStaging[id]
	}
	m.RUnlock()
	if !ok {
		return nil, errors.Errorf("image %s not found", id)
	}
	return img, nil
}

func (m *MemImage) Commit(id string) error {
	m.RLock()
	img, ok := m.imagesStaging[id]
	m.RUnlock()
	if !ok {
		return errors.Errorf("failed to commit %s, not found in staging", id)
	}

	m.Lock()
	m.images[id] = img
	m.Unlock()

	return nil
}

func (m *MemImage) Cleanup(_ context.Context, ttl time.Duration) error {
	var idsToRemove []string

	m.RLock()
	for id, t := range m.insertTime {
		age := time.Since(t)
		if age > ttl {
			log.Printf("[INFO] remove staging image %s, age %v", id, age)
			idsToRemove = append(idsToRemove, id)
		}
	}
	m.RUnlock()

	m.Lock()
	for _, id := range idsToRemove {
		delete(m.insertTime, id)
		delete(m.imagesStaging, id)
	}
	m.Unlock()
	return nil
}

// guid makes a globally unique id
func guid() string {
	return xid.New().String()
}
