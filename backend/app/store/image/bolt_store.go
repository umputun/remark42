package image

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	log "github.com/go-pkgz/lgr"
	bolt "go.etcd.io/bbolt"
)

const imagesStagedBktName = "imagesStaged"
const imagesBktName = "images"
const insertTimeBktName = "insertTimestamps"

// Bolt provides image Store for images keeping data in bolt DB, restricts max size.
// It uses 3 buckets to manage images data.
// Two buckets contains image data (staged and committed images). Third bucket holds insertion timestamps.
type Bolt struct {
	db *bolt.DB
}

// NewBoltStorage create bolt image store
func NewBoltStorage(fileName string, options bolt.Options) (*Bolt, error) {
	db, err := bolt.Open(fileName, 0o600, &options) //nolint:gocritic //octalLiteral is OK as FileMode
	if err != nil {
		return nil, fmt.Errorf("failed to make boltdb for %s: %w", fileName, err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		for _, name := range []string{imagesBktName, imagesStagedBktName, insertTimeBktName} {
			if _, e := tx.CreateBucketIfNotExists([]byte(name)); e != nil {
				return fmt.Errorf("failed to create top level bucket %s: %w", name, e)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize boltdb db %q buckets: %w", fileName, err)
	}
	return &Bolt{db: db}, nil
}

// Save saves image for given id to staging bucket in DB
func (b *Bolt) Save(id string, img []byte) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		if err := tx.Bucket([]byte(imagesStagedBktName)).Put([]byte(id), img); err != nil {
			return fmt.Errorf("can't put to bucket with %s: %w", id, err)
		}
		return putTS(tx, id, time.Now())
	})
}

// Commit file stored in staging bucket by copying it to permanent bucket
// Data from staging bucket not removed immediately, but would be removed on Cleanup
func (b *Bolt) Commit(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		data := tx.Bucket([]byte(imagesStagedBktName)).Get([]byte(id))
		if data == nil {
			return fmt.Errorf("failed to commit %s, not found in staging", id)
		}
		err := tx.Bucket([]byte(imagesBktName)).Put([]byte(id), data)
		if err != nil {
			return fmt.Errorf("can't put to bucket with %s: %w", id, err)
		}
		return nil
	})
}

// ResetCleanupTimer resets cleanup timer for the image
func (b *Bolt) ResetCleanupTimer(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error { return putTS(tx, id, time.Now()) })
}

// Load image from DB
func (b *Bolt) Load(id string) ([]byte, error) {
	var data []byte
	err := b.db.View(func(tx *bolt.Tx) error {
		value := tx.Bucket([]byte(imagesBktName)).Get([]byte(id))
		if value == nil {
			value = tx.Bucket([]byte(imagesStagedBktName)).Get([]byte(id))
		}
		if value == nil {
			return fmt.Errorf("can't load image %s", id)
		}
		// the value points into bolt's memory map and is only valid until the transaction ends
		data = bytes.Clone(value)
		return nil
	})
	if err != nil {
		// separate error handler to return nil and not empty []byte
		return nil, err
	}
	return data, nil
}

// Delete image from storage
func (b *Bolt) Delete(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		// deleting a non-existing key doesn't return an error, so joining errors from deleting an image
		// from both buckets is safe and will return nil if there are no errors on the real delete
		// or image is absent in both buckets
		err := tx.Bucket([]byte(imagesBktName)).Delete([]byte(id))
		err = errors.Join(err, tx.Bucket([]byte(imagesStagedBktName)).Delete([]byte(id)))
		return err
	})
}

// Cleanup runs scan of staging and removes old data based on ttl
func (b *Bolt) Cleanup(_ context.Context, ttl time.Duration) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(insertTimeBktName)).Cursor()

		var idsToRemove [][]byte

		for id, tsData := c.First(); id != nil; id, tsData = c.Next() {
			ts, err := readTS(id, tsData)
			if err != nil {
				return err
			}

			age := time.Since(ts)

			if age > ttl {
				log.Printf("[INFO] remove staging image %s, age %v", id, age)
				idsToRemove = append(idsToRemove, id)
				err := c.Delete()
				if err != nil {
					return fmt.Errorf("failed to remove timestamp for %s: %w", id, err)
				}
			}
		}
		imgBkt := tx.Bucket([]byte(imagesStagedBktName))
		for _, id := range idsToRemove {
			err := imgBkt.Delete(id)
			if err != nil {
				return fmt.Errorf("failed to remove image for %s: %w", id, err)
			}
		}
		return nil
	})
}

// Info returns meta information about storage
func (b *Bolt) Info() (StoreInfo, error) {
	var ts time.Time
	err := b.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(insertTimeBktName)).Cursor()

		for id, tsData := c.First(); id != nil; id, tsData = c.Next() {
			created, err := readTS(id, tsData)
			if err != nil {
				return err
			}
			if ts.IsZero() || created.Before(ts) {
				ts = created
			}
		}
		return nil
	})
	if err != nil {
		return StoreInfo{}, fmt.Errorf("problem retrieving first timestamp from staging images: %w", err)
	}
	return StoreInfo{FirstStagingImageTS: ts}, nil
}

// putTS records ts as the insertion time of id, eight bytes of little-endian Unix nanoseconds
func putTS(tx *bolt.Tx, id string, ts time.Time) error {
	data, err := binary.Append(nil, binary.LittleEndian, ts.UnixNano())
	if err != nil {
		return fmt.Errorf("can't serialize timestamp for %s: %w", id, err)
	}
	if err := tx.Bucket([]byte(insertTimeBktName)).Put([]byte(id), data); err != nil {
		return fmt.Errorf("can't put to bucket with %s: %w", id, err)
	}
	return nil
}

// readTS decodes the insertion time of id written by putTS
func readTS(id, data []byte) (time.Time, error) {
	var ns int64
	if _, err := binary.Decode(data, binary.LittleEndian, &ns); err != nil {
		return time.Time{}, fmt.Errorf("failed to deserialize timestamp for %s: %w", id, err)
	}
	return time.Unix(0, ns), nil
}
