package image

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	bolt "github.com/coreos/bbolt"
	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
)

// BoltStore defines parameters for BoltDB Image Store
type BoltStore struct {
	FileName string
}

// BoltDB provides Image Store for BoltDB. Saves and loads files from Location, restricts max size.
type BoltDB struct {
	db     *bolt.DB
	config BoltStore
}

const (
	committedBucketName = "committed"
	stagingBucketName   = "staging"
	metasBucketName     = "metas"
)

// NewBoltDB ensures bolt DB exists and has required top level buckets
func NewBoltDB(storeConfig BoltStore, extraBoltOptions bolt.Options) (*BoltDB, error) {
	log.Printf("[INFO] ensuring bolt DB for storing images: %+v, extra options: %+v", storeConfig, extraBoltOptions)
	db, err := bolt.Open(storeConfig.FileName, 0600, &extraBoltOptions)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to make boltdb for %s", storeConfig.FileName)
	}

	buckets := []string{committedBucketName, stagingBucketName, metasBucketName}
	err = db.Update(func(tx *bolt.Tx) error {
		for _, bktName := range buckets {
			if _, e := tx.CreateBucketIfNotExists([]byte(bktName)); e != nil {
				return errors.Wrapf(e, "failed to create top level bucket %s", bktName)
			}
		}
		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to create top level bucket)")
	}
	return &BoltDB{db: db, config: storeConfig}, nil
}

func (b *BoltDB) addMeta(id string, tx *bolt.Tx) error {
	bucket := tx.Bucket([]byte(metasBucketName))
	key := []byte(time.Now().Format(time.RFC3339))
	value := id
	if res := bucket.Get(key); res != nil {
		value = string(res) + "," + value
	}
	return bucket.Put(key, []byte(value))
}

// Save gets name and reader and returns ID of stored image
func (b *BoltDB) Save(fileName string, userID string, data []byte) (id string, err error) {
	id = userID + "/" + guid() + filepath.Ext(fileName)

	log.Printf("[DEBUG] save image %s to staging", id)
	err = b.db.Update(func(tx *bolt.Tx) error {
		userBucket := tx.Bucket([]byte(stagingBucketName))
		if err = b.addMeta(id, tx); err != nil {
			return errors.Wrapf(err, "unable to save meta for image %s", id)
		}
		if err = userBucket.Put([]byte(id), data); err != nil {
			return errors.Wrapf(err, "can't put to bucket with %s", userID)
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	return id, nil
}

// Commit moves image from staging to permanent
func (b *BoltDB) Commit(id string) error {
	log.Printf("[DEBUG] commit image %s", id)
	return b.db.Update(func(tx *bolt.Tx) error {
		stagingBkt := tx.Bucket([]byte(stagingBucketName))
		data := stagingBkt.Get([]byte(id))
		if data == nil {
			// There's nothing to commit, maybe picture got committed already or cleaned up
			return nil
		}
		err := tx.Bucket([]byte(committedBucketName)).Put([]byte(id), data)
		if err != nil {
			return errors.Wrapf(err, "unable to put image %s into committed bucket", id)
		}
		// We don't delete meta for committed image, next cleanup will take care of it
		return stagingBkt.Delete([]byte(id))
	})
}

// Load returns image by ID. Caller has to close the reader
func (b *BoltDB) Load(id string) (reader io.ReadCloser, size int64, err error) {
	log.Printf("[DEBUG] load image %s", id)
	buf := &bytes.Buffer{}
	err = b.db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket([]byte(committedBucketName)).Get([]byte(id))
		if data == nil {
			data = tx.Bucket([]byte(stagingBucketName)).Get([]byte(id))
			if data == nil {
				return errors.Errorf("can't load image %s", id)
			}
		}
		_, err = buf.Write(data)
		return errors.Wrapf(err, "failed to write for %s", id)
	})
	return ioutil.NopCloser(buf), int64(buf.Len()), err
}

// Cleanup runs removal loop for old images on staging
func (b *BoltDB) Cleanup(ctx context.Context, ttl time.Duration) error {
	log.Printf("[DEBUG] cleaning up staged images")
	return b.db.Update(func(tx *bolt.Tx) error {
		metasBkt := tx.Bucket([]byte(metasBucketName))
		stagingBkt := tx.Bucket([]byte(stagingBucketName))

		// date-time of writing this code, assuming there can't be any staged images before that
		min := []byte("2018-10-19T19:43:00Z")
		max := []byte((time.Now().Add(-ttl)).Format(time.RFC3339))

		c := metasBkt.Cursor()
		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			for _, img := range strings.Split(string(v), ",") {
				log.Printf("[DEBUG] removing staged image %s", img)
				if stagingBkt.Delete([]byte(img)) != nil {
					log.Printf("[WARN] unable to delete image %s", img)
				}
			}
			if err := metasBkt.Delete(k); err != nil {
				log.Printf("[WARN] failed to delete meta %s", string(k))
			}
		}
		return nil
	})
}
