package image

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"io/ioutil"
	"path"
	"time"

	bolt "github.com/coreos/bbolt"
	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
)

type Bolt struct {
	fileName  string
	db        *bolt.DB
	MaxSize   int
	MaxHeight int
	MaxWidth  int
}

const imagesBktName = "images"
const insertTimeBktName = "insert_times"
const commitedFlagBktName = "commited_flags"

func NewBoltStorage(fileName string, maxSize int, maxHeight int, maxWidth int, options bolt.Options) (*Bolt, error) {
	db, err := bolt.Open(fileName, 0600, &options)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to make boltdb for %s", fileName)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		if _, e := tx.CreateBucketIfNotExists([]byte(imagesBktName)); e != nil {
			return errors.Wrapf(e, "failed to create top level bucket %s", imagesBktName)
		}
		if _, e := tx.CreateBucketIfNotExists([]byte(insertTimeBktName)); e != nil {
			return errors.Wrapf(e, "failed to create top level bucket %s", insertTimeBktName)
		}
		if _, e := tx.CreateBucketIfNotExists([]byte(commitedFlagBktName)); e != nil {
			return errors.Wrapf(e, "failed to create top level bucket %s", commitedFlagBktName)
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initialize boltdb db %q buckets", fileName)
	}
	return &Bolt{
		db:        db,
		fileName:  fileName,
		MaxSize:   maxSize,
		MaxHeight: maxHeight,
		MaxWidth:  maxWidth,
	}, nil
}

func (b *Bolt) Save(fileName string, userID string, r io.Reader) (id string, err error) {
	data, err := readAndValidateImage(r, b.MaxSize)
	if err != nil {
		return "", errors.Wrapf(err, "can't load image %s", fileName)
	}

	data, _ = resize(data, b.MaxWidth, b.MaxHeight)

	id = path.Join(userID, guid())

	err = b.db.Update(func(tx *bolt.Tx) error {
		if err = tx.Bucket([]byte(imagesBktName)).Put([]byte(id), data); err != nil {
			return errors.Wrapf(err, "can't put to bucket with %s", id)
		}
		tsBuf := &bytes.Buffer{}
		binary.Write(tsBuf, binary.LittleEndian, time.Now().UnixNano())
		if err = tx.Bucket([]byte(insertTimeBktName)).Put([]byte(id), tsBuf.Bytes()); err != nil {
			return errors.Wrapf(err, "can't put to bucket with %s", id)
		}
		return err
	})

	return id, err
}

func (b *Bolt) Commit(id string) error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte(commitedFlagBktName)).Put([]byte(id), []byte{1})
		if err != nil {
			return errors.Wrapf(err, "failed to set commited flag for %s", id)
		}
		return nil
	})
	return err
}

func (b *Bolt) Load(id string) (io.ReadCloser, int64, error) {
	buf := &bytes.Buffer{}
	var size int = 0
	err := b.db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket([]byte(imagesBktName)).Get([]byte(id))
		if data == nil {
			return errors.Errorf("can't load image %s", id)
		}
		var err error
		size, err = buf.Write(data)
		return errors.Wrapf(err, "failed to write for %s", id)
	})
	return ioutil.NopCloser(buf), int64(size), err
}

func (b *Bolt) Cleanup(ctx context.Context, ttl time.Duration) error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(insertTimeBktName)).Cursor()

		idsToRemove := [][]byte{}
		flagBkt := tx.Bucket([]byte(commitedFlagBktName))

		for id, tsData := c.First(); id != nil; id, tsData = c.Next() {
			if isCommited := flagBkt.Get([]byte(id)); isCommited != nil {
				continue
			}
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
		imgBkt := tx.Bucket([]byte(imagesBktName))
		for _, id := range idsToRemove {
			err := imgBkt.Delete([]byte(id))
			if err != nil {
				return errors.Wrapf(err, "failed to remove image for %s", id)
			}
		}
		return nil
	})
	return err
}

func (b *Bolt) SizeLimit() int {
	return b.MaxSize
}
