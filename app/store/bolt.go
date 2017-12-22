package store

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
)

// BoltDB implements store.Interface. Each instance represents one site.
// Keys built as pid-id. Each url (post) makes it's own bucket
// In addition there is a bucket "last" with reference to other buckets+keys to all cross-posts last comment extraction.
// Thread safe.
type BoltDB struct {
	*bolt.DB
}

var lastBucketName = "last"

// NewBoltDB makes persistent boltdb-based store
func NewBoltDB(dbFile string) (*BoltDB, error) {
	log.Printf("[INFO] bolt store, %s", dbFile)
	result := BoltDB{}
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to make boltdb for %s", dbFile)
	}
	result.DB = db
	return &result, err
}

// Create saves new comment to store
func (b *BoltDB) Create(comment Comment) (int64, error) {

	comment.ID = time.Now().UnixNano()
	comment.Timestamp = time.Now()

	err := b.Update(func(tx *bolt.Tx) error {
		bucket, e := tx.CreateBucketIfNotExists([]byte(comment.Locator.URL))
		if e != nil {
			return errors.Wrapf(e, "can't make bucket", comment.Locator.URL)
		}

		// check if key already in store, reject doubles
		key := b.keyFromComment(comment)
		if bucket.Get(key) != nil {
			return errors.Errorf("key %s already in store", string(key))
		}

		// serialise comment to json's []byte for bolt and save
		jdata, jerr := json.Marshal(&comment)
		if jerr != nil {
			return errors.Wrap(jerr, "can't marshal comment")
		}

		if err := bucket.Put(key, jdata); err != nil {
			return errors.Wrapf(err, "failed to put key %s", string(key))
		}

		// add reference to comment to "last" bucket
		bucket, e = tx.CreateBucketIfNotExists([]byte(lastBucketName))
		if e != nil {
			return errors.Wrapf(e, "can't make bucket %s", lastBucketName)
		}

		rv := refFromComment(comment)
		e = bucket.Put([]byte(fmt.Sprintf("%d", time.Now().UnixNano())), []byte(rv.value()))
		if e != nil {
			return errors.Wrapf(e, "can't put reference %s to %s", rv.value(), lastBucketName)
		}

		return nil
	})

	return comment.ID, err
}

// Delete removed comment by url and id from the store
func (b *BoltDB) Delete(url string, id int64) error {

	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(url))
		if bucket == nil {
			return errors.Errorf("no bucket %s in store", url)
		}
		key := b.keyFromValue(id)
		if err := bucket.Delete(key); err != nil {
			return errors.Wrapf(err, "can't delete key %s from bucket %s", key, url)
		}
		return nil
	})
}

// Find comments for post
func (b *BoltDB) Find(request Request) ([]Comment, error) {
	res := []Comment{}

	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(request.Locator.URL))
		if bucket == nil {
			return errors.Errorf("no bucket %s in store", request.Locator.URL)
		}

		return bucket.ForEach(func(k, v []byte) error {
			comment := Comment{}
			if e := json.Unmarshal(v, &comment); e != nil {
				return errors.Wrap(e, "failed to unmarshal")
			}
			res = append(res, comment)
			return nil
		})
	})

	return res, err
}

// Get comment by id
func (b *BoltDB) Get(locator Locator, id int64) (comment Comment, err error) {
	err = b.View(func(tx *bolt.Tx) error {

		lastBucket := tx.Bucket([]byte(lastBucketName))
		if lastBucket == nil {
			return errors.Errorf("no bucket %s in store", lastBucketName)
		}

		c := lastBucket.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			url, foundID, e := refFromValue(v).parse()
			if e != nil {
				return e
			}

			if foundID == id && url == locator.URL {
				urlBucket := tx.Bucket([]byte(url))
				if urlBucket == nil {
					return errors.Errorf("no bucket %s in store", url)
				}
				commentVal := urlBucket.Get(b.keyFromValue(id))
				if commentVal == nil {
					return errors.Errorf("no comment for %d in store %s", id, url)
				}

				if e := json.Unmarshal(commentVal, &comment); e != nil {
					return errors.Wrap(e, "failed to unmarshal")
				}
				return nil
			}
		}
		return errors.Errorf("no id %d in store %s", id, locator.URL)
	})

	return comment, err
}

// Last returns up to max last comments for given locator
func (b *BoltDB) Last(locator Locator, max int) (result []Comment, err error) {

	err = b.View(func(tx *bolt.Tx) error {
		lastBucket := tx.Bucket([]byte(lastBucketName))
		if lastBucket == nil {
			return errors.Errorf("no bucket %s in store", lastBucketName)
		}

		c := lastBucket.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			url, id, e := refFromValue(v).parse()
			if e != nil {
				return e
			}
			urlBucket := tx.Bucket([]byte(url))
			if urlBucket == nil {
				return errors.Errorf("no bucket %s in store", url)
			}
			commentVal := urlBucket.Get(b.keyFromValue(id))
			if commentVal == nil {
				return errors.Errorf("no comment for %d in store %s", id, url)
			}

			comment := Comment{}
			if e := json.Unmarshal(commentVal, &comment); e != nil {
				return errors.Wrap(e, "failed to unmarshal")
			}
			result = append(result, comment)
			if max > 0 && len(result) >= max {
				return nil
			}
		}
		return nil
	})

	return result, err
}

func (b *BoltDB) keyFromComment(comment Comment) []byte {
	return []byte(fmt.Sprintf("%22d", comment.ID))
}

func (b *BoltDB) keyFromValue(id int64) []byte {
	return []byte(fmt.Sprintf("%22d", id))
}

// buckets returns list of buckets, which is list of all commented posts
func (b BoltDB) buckets() (result []string) {

	_ = b.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
			result = append(result, string(name))
			return nil
		})
	})
	return result
}

type ref string

func refFromComment(comment Comment) *ref {
	result := ref(fmt.Sprintf("%s!!%d", comment.Locator.URL, comment.ID))
	return &result
}

func refFromValue(val []byte) *ref {
	result := ref(string(val))
	return &result
}

func (r ref) value() string { return string(r) }

func (r ref) parse() (url string, id int64, err error) {
	elems := strings.Split(string(r), "!!")
	if len(elems) < 2 {
		return "", 0, errors.Errorf("can't parse ref %s", r)
	}
	url = elems[0]
	if id, err = strconv.ParseInt(elems[1], 10, 64); err != nil {
		return "", 0, errors.Wrapf(err, "can't extract id from ref %s", r)
	}
	return url, id, nil
}
