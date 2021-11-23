package image

import (
	"bytes"
	"context"
	"encoding/binary"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

const imagesStagedBktName = "imagesStaged"
const imagesBktName = "images"
const insertTimeBktName = "insertTimestamps"

// Bolt provides image Store for images keeping data in bolt DB, restricts max size.
// It uses 3 buckets to manage images data.
// Two buckets contains image data (staged and committed images). Third bucket holds insertion timestamps.
type Bolt struct {
	fileName string
	db       *bolt.DB
}

// NewBoltStorage create bolt image store
func NewBoltStorage(fileName string, options bolt.Options) (*Bolt, error) {
	db, err := bolt.Open(fileName, 0o600, &options) //nolint:gocritic //octalLiteral is OK as FileMode
	if err != nil {
		return nil, errors.Wrapf(err, "failed to make boltdb for %s", fileName)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		if _, e := tx.CreateBucketIfNotExists([]byte(imagesBktName)); e != nil {
			return errors.Wrapf(e, "failed to create top level bucket %s", imagesBktName)
		}
		if _, e := tx.CreateBucketIfNotExists([]byte(imagesStagedBktName)); e != nil {
			return errors.Wrapf(e, "failed to create top level bucket %s", imagesStagedBktName)
		}
		if _, e := tx.CreateBucketIfNotExists([]byte(insertTimeBktName)); e != nil {
			return errors.Wrapf(e, "failed to create top level bucket %s", insertTimeBktName)
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initialize boltdb db %q buckets", fileName)
	}
	return &Bolt{
		db:       db,
		fileName: fileName,
	}, nil
}

// Save saves image for given id to staging bucket in DB
func (b *Bolt) Save(id string, img []byte) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		if err := tx.Bucket([]byte(imagesStagedBktName)).Put([]byte(id), img); err != nil {
			return errors.Wrapf(err, "can't put to bucket with %s", id)
		}
		tsBuf := &bytes.Buffer{}
		if err := binary.Write(tsBuf, binary.LittleEndian, time.Now().UnixNano()); err != nil {
			return errors.Wrapf(err, "can't serialize timestamp for %s", id)
		}
		if err := tx.Bucket([]byte(insertTimeBktName)).Put([]byte(id), tsBuf.Bytes()); err != nil {
			return errors.Wrapf(err, "can't put to bucket with %s", id)
		}
		return nil
	})
}

// Commit file stored in staging bucket by copying it to permanent bucket
// Data from staging bucket not removed immediately, but would be removed on Cleanup
func (b *Bolt) Commit(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		data := tx.Bucket([]byte(imagesStagedBktName)).Get([]byte(id))
		if data == nil {
			return errors.Errorf("failed to commit %s, not found in staging", id)
		}
		err := tx.Bucket([]byte(imagesBktName)).Put([]byte(id), data)
		return errors.Wrapf(err, "can't put to bucket with %s", id)
	})
}

// ResetCleanupTimer resets cleanup timer for the image
func (b *Bolt) ResetCleanupTimer(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		tsBuf := &bytes.Buffer{}
		if err := binary.Write(tsBuf, binary.LittleEndian, time.Now().UnixNano()); err != nil {
			return errors.Wrapf(err, "can't serialize timestamp for %s", id)
		}
		if err := tx.Bucket([]byte(insertTimeBktName)).Put([]byte(id), tsBuf.Bytes()); err != nil {
			return errors.Wrapf(err, "can't put to bucket with %s", id)
		}
		return nil
	})
}

// Load image from DB
func (b *Bolt) Load(id string) ([]byte, error) {
	var data []byte
	err := b.db.View(func(tx *bolt.Tx) error {
		data = tx.Bucket([]byte(imagesBktName)).Get([]byte(id))
		if data == nil {
			data = tx.Bucket([]byte(imagesStagedBktName)).Get([]byte(id))
		}
		if data == nil {
			return errors.Errorf("can't load image %s", id)
		}
		return nil
	})
	if err != nil {
		// separate error handler to return nil and not empty []byte
		return nil, err
	}
	return data, nil
}

// Cleanup runs scan of staging and removes old data based on ttl
func (b *Bolt) Cleanup(_ context.Context, ttl time.Duration) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(insertTimeBktName)).Cursor()

		var idsToRemove [][]byte

		for id, tsData := c.First(); id != nil; id, tsData = c.Next() {
			var ts int64
			err := binary.Read(bytes.NewReader(tsData), binary.LittleEndian, &ts)
			if err != nil {
				return errors.Wrapf(err, "failed to deserialize timestamp for %s", id)
			}

			age := time.Since(time.Unix(0, ts))

			if age > ttl {
				log.Printf("[INFO] remove staging image %s, age %v", id, age)
				idsToRemove = append(idsToRemove, id)
				err := c.Delete()
				if err != nil {
					return errors.Wrapf(err, "failed to remove timestamp for %s", id)
				}
			}
		}
		imgBkt := tx.Bucket([]byte(imagesStagedBktName))
		for _, id := range idsToRemove {
			err := imgBkt.Delete(id)
			if err != nil {
				return errors.Wrapf(err, "failed to remove image for %s", id)
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
			var createdRaw int64
			err := binary.Read(bytes.NewReader(tsData), binary.LittleEndian, &createdRaw)
			if err != nil {
				return errors.Wrapf(err, "failed to deserialize timestamp for %s", id)
			}

			created := time.Unix(0, createdRaw)
			if ts.IsZero() || created.Before(ts) {
				ts = created
			}
		}
		return nil
	})
	return StoreInfo{FirstStagingImageTS: ts}, errors.Wrapf(err, "problem retrieving first timestamp from staging images")
}
