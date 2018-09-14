package avatar

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"

	"github.com/coreos/bbolt"
	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/store"
)

// BoltDB implements avatar store with bolt
// using separate db (file) with top-level keys by avatarID
type BoltDB struct {
	fileName    string // full path to boltdb
	resizeLimit int
	db          *bolt.DB
}

const bktName = "avatars"

// NewBoltDB makes bolt avatar store
func NewBoltDB(fileName string, options bolt.Options, resizeLimit int) (*BoltDB, error) {
	db, err := bolt.Open(fileName, 0600, &options)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to make boltdb for %s", fileName)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists([]byte(bktName))
		return errors.Wrapf(e, "failed to create top level bucket %s", bktName)
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initialize boltdb db bucket %s for %s", bktName, fileName)
	}
	return &BoltDB{db: db, fileName: fileName, resizeLimit: resizeLimit}, nil
}

// Put avatar to bolt, key by avatarID
func (b *BoltDB) Put(userID string, reader io.Reader) (avatar string, err error) {
	id := encodeID(userID)

	// Trying to resize avatar.
	if reader = resize(reader, b.resizeLimit); reader == nil {
		return "", errors.New("avatar resize reader is nil")
	}

	avatarID := id + imgSfx
	err = b.db.Update(func(tx *bolt.Tx) error {
		buf := &bytes.Buffer{}
		if _, err = io.Copy(buf, reader); err != nil {
			return errors.Wrapf(err, "can't read avatar %s", avatarID)
		}
		return errors.Wrapf(tx.Bucket([]byte(bktName)).Put([]byte(avatarID), buf.Bytes()),
			"can't put to bucket with %s", avatarID)
	})
	return avatarID, err
}

// Get avatar reader for avatar id.image, avatarID used as the direct key
func (b *BoltDB) Get(avatarID string) (reader io.ReadCloser, size int, err error) {
	buf := &bytes.Buffer{}
	err = b.db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket([]byte(bktName)).Get([]byte(avatarID))
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
	reader, size, err := b.Get(avatarID)
	if err != nil {
		log.Printf("[DEBUG] can't get avatar info '%s', %s", avatarID, err)
		return store.EncodeID(avatarID)
	}
	data := make([]byte, size)
	if _, e := reader.Read(data); e != nil {
		log.Printf("[DEBUG] can't read avatar data '%s', %s", avatarID, err)
		return store.EncodeID(avatarID)
	}

	h := sha1.New()
	if _, err = h.Write(data); err != nil {
		log.Printf("[DEBUG] can't apply sha1 for content of '%s', %s", avatarID, err)
		return store.EncodeID(avatarID)
	}

	return hex.EncodeToString(h.Sum(nil))
}

// Remove avatar from bolt
func (b *BoltDB) Remove(avatarID string) (err error) {
	return b.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(bktName))
		if bkt.Get([]byte(avatarID)) == nil {
			return errors.Errorf("avatar key not found, %s", avatarID)
		}
		return errors.Wrapf(tx.Bucket([]byte(bktName)).Delete([]byte(avatarID)), "can't delete object %s", avatarID)
	})
}

// List all avatars (ids) from avatars bucket
// note: id includes .image suffix
func (b *BoltDB) List() (ids []string, err error) {
	err = b.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(bktName)).ForEach(func(k, _ []byte) error {
			ids = append(ids, string(k))
			return nil
		})
	})
	return ids, errors.Wrap(err, "failed to list")
}
