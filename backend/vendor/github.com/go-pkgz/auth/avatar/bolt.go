package avatar

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

// BoltDB implements avatar store with bolt
// using separate db (file) with "avatars" bucket to keep image bin and "metas" bucket
// to keep sha1 of picture. avatarID (base file name) used as a key for both.
type BoltDB struct {
	fileName string // full path to boltdb
	db       *bolt.DB
}

const avatarsBktName = "avatars"
const metasBktName = "metas"

// NewBoltDB makes bolt avatar store
func NewBoltDB(fileName string, options bolt.Options) (*BoltDB, error) {
	db, err := bolt.Open(fileName, 0600, &options) //nolint
	if err != nil {
		return nil, errors.Wrapf(err, "failed to make boltdb for %s", fileName)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		if _, e := tx.CreateBucketIfNotExists([]byte(avatarsBktName)); e != nil {
			return errors.Wrapf(e, "failed to create top level bucket %s", avatarsBktName)
		}
		_, e := tx.CreateBucketIfNotExists([]byte(metasBktName))
		return errors.Wrapf(e, "failed to create top metas bucket %s", metasBktName)
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initialize boltdb db %q buckets", fileName)
	}
	return &BoltDB{db: db, fileName: fileName}, nil
}

// Put avatar to bolt, key by avatarID. Trying to resize image and lso calculates sha1 of the file for ID func
func (b *BoltDB) Put(userID string, reader io.Reader) (avatar string, err error) {
	id := encodeID(userID)

	avatarID := id + imgSfx
	err = b.db.Update(func(tx *bolt.Tx) error {
		buf := &bytes.Buffer{}
		if _, err = io.Copy(buf, reader); err != nil {
			return errors.Wrapf(err, "can't read avatar %s", avatarID)
		}

		if err = tx.Bucket([]byte(avatarsBktName)).Put([]byte(avatarID), buf.Bytes()); err != nil {
			return errors.Wrapf(err, "can't put to bucket with %s", avatarID)
		}
		// store sha1 of the image
		return tx.Bucket([]byte(metasBktName)).Put([]byte(avatarID), []byte(hash(buf.Bytes(), avatarID)))
	})
	return avatarID, err
}

// Get avatar reader for avatar id.image, avatarID used as the direct key
func (b *BoltDB) Get(avatarID string) (reader io.ReadCloser, size int, err error) {
	buf := &bytes.Buffer{}
	err = b.db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket([]byte(avatarsBktName)).Get([]byte(avatarID))
		if data == nil {
			return errors.Errorf("can't load avatar %s", avatarID)
		}
		size, err = buf.Write(data)
		return errors.Wrapf(err, "failed to write for %s", avatarID)
	})
	return ioutil.NopCloser(buf), size, err
}

// ID returns a fingerprint of the avatar content.
func (b *BoltDB) ID(avatarID string) (id string) {
	data := []byte{}
	err := b.db.View(func(tx *bolt.Tx) error {
		if data = tx.Bucket([]byte(metasBktName)).Get([]byte(avatarID)); data == nil {
			return errors.Errorf("can't load avatar's id for %s", avatarID)
		}
		return nil
	})

	if err != nil { // failed to get ID, use encoded avatarID
		log.Printf("[DEBUG] can't get avatar info '%s', %s", avatarID, err)
		return encodeID(avatarID)
	}

	return string(data)
}

// Remove avatar from bolt
func (b *BoltDB) Remove(avatarID string) (err error) {
	return b.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(avatarsBktName))
		if bkt.Get([]byte(avatarID)) == nil {
			return errors.Errorf("avatar key not found, %s", avatarID)
		}
		if err = tx.Bucket([]byte(avatarsBktName)).Delete([]byte(avatarID)); err != nil {
			return errors.Wrapf(err, "can't delete avatar object %s", avatarID)
		}
		return errors.Wrapf(tx.Bucket([]byte(metasBktName)).Delete([]byte(avatarID)),
			"can't delete meta object %s", avatarID)
	})
}

// List all avatars (ids) from metas bucket
// note: id includes .image suffix
func (b *BoltDB) List() (ids []string, err error) {
	err = b.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(metasBktName)).ForEach(func(k, _ []byte) error {
			ids = append(ids, string(k))
			return nil
		})
	})
	return ids, errors.Wrap(err, "failed to list")
}

// Close bolt store
func (b *BoltDB) Close() error {
	return errors.Wrapf(b.db.Close(), "failed to close %s", b.fileName)
}

func (b *BoltDB) String() string {
	return fmt.Sprintf("boltdb, path=%s", b.fileName)
}
