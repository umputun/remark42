/*
 * Copyright 2020 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package accessor

import (
	"context"
	"sync"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"

	"github.com/umputun/remark42/backend/app/store/image"
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

// Save stores image with passed id to staging
func (m *MemImage) Save(id string, img []byte) error {
	m.Lock()
	m.imagesStaging[id] = img
	m.insertTime[id] = time.Now()
	m.Unlock()

	return nil
}

// ResetCleanupTimer resets cleanup timer for the image
func (m *MemImage) ResetCleanupTimer(id string) error {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.insertTime[id]; ok {
		m.insertTime[id] = time.Now()
		return nil
	}
	return errors.Errorf("image %s not found", id)
}

// Load image by ID
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

// Commit moves image from staging to permanent
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

// Cleanup runs removal loop for old images on staging
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

// Info returns meta information about storage
func (m *MemImage) Info() (image.StoreInfo, error) {
	var ts time.Time
	m.RLock()
	for _, t := range m.insertTime {
		if ts.IsZero() || t.Before(ts) {
			ts = t
		}
	}
	m.RUnlock()

	return image.StoreInfo{FirstStagingImageTS: ts}, nil
}
