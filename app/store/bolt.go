package store

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"sort"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
)

// BoltDB implements store.Interface. Each instance represents one site.
// Keys are commendID. Each url (post) makes it's own bucket.
// In addition there is a bucket "last" with reference to other buckets+keys to all cross-posts last comment extraction.
// Thread safe.
type BoltDB struct {
	*bolt.DB
}

const (
	lastBucketName     = "last"
	blocksBucketPrefix = "block-"
	lastLimit          = 1000
)

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
func (b *BoltDB) Create(comment Comment) (string, error) {

	if comment.ID == "" {
		comment.ID = makeCommentID()
	}

	comment.Timestamp = time.Now()
	comment.Votes = make(map[string]bool)

	err := b.Update(func(tx *bolt.Tx) error {
		bucket, e := tx.CreateBucketIfNotExists([]byte(comment.Locator.URL))
		if e != nil {
			return errors.Wrapf(e, "can't make bucket", comment.Locator.URL)
		}

		// check if key already in store, reject doubles
		if bucket.Get([]byte(comment.ID)) != nil {
			return errors.Errorf("key %s already in store", comment.ID)
		}

		// serialize comment to json []byte for bolt and save
		jdata, jerr := json.Marshal(&comment)
		if jerr != nil {
			return errors.Wrap(jerr, "can't marshal comment")
		}

		if err := bucket.Put([]byte(comment.ID), jdata); err != nil {
			return errors.Wrapf(err, "failed to put key %s", comment.ID)
		}

		// add reference to comment to "last" bucket
		bucket, e = tx.CreateBucketIfNotExists([]byte(lastBucketName))
		if e != nil {
			return errors.Wrapf(e, "can't make bucket %s", lastBucketName)
		}

		rv := refFromComment(comment)
		e = bucket.Put([]byte(rv.key), []byte(rv.value))
		if e != nil {
			return errors.Wrapf(e, "can't put reference %s to %s", rv.value, lastBucketName)
		}

		return nil
	})

	return comment.ID, err
}

// Delete removes comment by url and comment id from the store
func (b *BoltDB) Delete(locator Locator, commentID string) error {

	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(locator.URL))
		if bucket == nil {
			return errors.Errorf("no bucket %s in store", locator.URL)
		}
		if err := bucket.Delete([]byte(commentID)); err != nil {
			return errors.Wrapf(err, "can't delete key %s from bucket %s", commentID, locator.URL)
		}

		// delete from "last" bucket
		bucket = tx.Bucket([]byte(lastBucketName))
		if bucket == nil {
			return errors.Errorf("no bucket %s in store", lastBucketName)
		}

		if err := bucket.Delete([]byte(commentID)); err != nil {
			return errors.Wrapf(err, "can't delete key %s from bucket %s", commentID, lastBucketName)
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

	sort.Slice(res, func(i, j int) bool { return res[i].Timestamp.Before(res[j].Timestamp) })
	return res, err
}

// Get comment by id
func (b *BoltDB) Get(locator Locator, commentID string) (comment Comment, err error) {
	err = b.View(func(tx *bolt.Tx) error {

		lastBucket := tx.Bucket([]byte(lastBucketName))
		if lastBucket == nil {
			return errors.Errorf("no bucket %s in store", lastBucketName)
		}

		c := lastBucket.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			url, foundID, e := refFromValue(v).parseValue()
			if e != nil {
				return e
			}

			if foundID == commentID && url == locator.URL {
				urlBucket := tx.Bucket([]byte(url))
				if urlBucket == nil {
					return errors.Errorf("no bucket %s in store", url)
				}
				commentVal := urlBucket.Get([]byte(commentID))
				if commentVal == nil {
					return errors.Errorf("no comment for %s in store %s", commentID, url)
				}

				if e := json.Unmarshal(commentVal, &comment); e != nil {
					return errors.Wrap(e, "failed to unmarshal")
				}
				return nil
			}
		}
		return errors.Errorf("no id %s in store %s", commentID, locator.URL)
	})

	return comment, err
}

// Last returns up to max last comments for given locator
func (b *BoltDB) Last(locator Locator, max int) (result []Comment, err error) {

	if max > lastLimit || max == 0 {
		max = lastLimit
	}

	err = b.View(func(tx *bolt.Tx) error {
		lastBucket := tx.Bucket([]byte(lastBucketName))
		if lastBucket == nil {
			return errors.Errorf("no bucket %s in store", lastBucketName)
		}

		c := lastBucket.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			url, commentID, e := refFromValue(v).parseValue()
			if e != nil {
				return e
			}
			urlBucket := tx.Bucket([]byte(url))
			if urlBucket == nil {
				return errors.Errorf("no bucket %s in store", url)
			}
			commentVal := urlBucket.Get([]byte(commentID))
			if commentVal == nil {
				log.Printf("[WARN] no comment for %s in store %s", commentID, url)
				continue
			}

			comment := Comment{}
			if e := json.Unmarshal(commentVal, &comment); e != nil {
				return errors.Wrap(e, "failed to unmarshal")
			}
			result = append(result, comment)
			if len(result) >= max {
				return nil
			}
		}
		return nil
	})

	return result, err
}

// Vote for comment by id and locator
func (b *BoltDB) Vote(locator Locator, commentID string, userID string, val bool) (comment Comment, err error) {

	err = b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(locator.URL))
		if bucket == nil {
			return errors.Errorf("no bucket %s in store", locator.URL)
		}

		// get and unmarshal comment for the store
		commentVal := bucket.Get([]byte(commentID))
		if commentVal == nil {
			return errors.Errorf("no comment for %s in store %s", commentID, locator.URL)
		}

		if e := json.Unmarshal(commentVal, &comment); e != nil {
			return errors.Wrap(e, "failed to unmarshal")
		}

		// check if user voted already
		for k := range comment.Votes {
			if k == userID {
				return errors.Errorf("user %s already voted for comment %s", userID, commentID)
			}
		}

		// update votes and score
		comment.Votes[userID] = val
		if val {
			comment.Score++
		} else {
			comment.Score--
		}
		data, e := json.Marshal(&comment)
		if e != nil {
			return errors.Wrap(e, "can't marshal comment with updated votes")
		}
		if e = bucket.Put([]byte(commentID), data); e != nil {
			return errors.Wrap(e, "failed to save comment with updated votes")
		}
		return nil
	})

	return comment, err
}

// Count returns number of comments for locator
func (b *BoltDB) Count(locator Locator) (count int, err error) {
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(locator.URL))
		if bucket == nil {
			return errors.Errorf("no bucket %s in store", locator.URL)
		}

		count = bucket.Stats().KeyN
		return nil
	})

	return count, err
}

// SetBlock blocks/unblocks user for given site
func (b *BoltDB) SetBlock(locator Locator, userID string, status bool) error {
	blockBucketName := b.bucketForBlock(locator, userID)
	return b.Update(func(tx *bolt.Tx) error {

		bucket, e := tx.CreateBucketIfNotExists(blockBucketName)
		if e != nil {
			return errors.Errorf("no bucket %s in store", string(blockBucketName))
		}

		switch status {
		case true:
			if e := bucket.Put([]byte(userID), []byte(time.Now().Format(time.RFC3339))); e != nil {
				return errors.Wrapf(e, "failed to put %s to %s", userID, string(blockBucketName))
			}
		case false:
			if e := bucket.Delete([]byte(userID)); e != nil {
				return errors.Wrapf(e, "failed to clean %s from %s", userID, string(blockBucketName))
			}
		}
		return nil
	})
}

// IsBlocked checks if user blocked
func (b *BoltDB) IsBlocked(locator Locator, userID string) (result bool) {
	blockBucketName := b.bucketForBlock(locator, userID)
	_ = b.View(func(tx *bolt.Tx) error {
		result = false
		bucket := tx.Bucket(blockBucketName)
		if bucket != nil && bucket.Get([]byte(userID)) != nil {
			result = true
		}
		return nil
	})
	return result
}

func (b *BoltDB) bucketForBlock(locator Locator, userID string) []byte {
	return []byte(fmt.Sprintf("%s%s", blocksBucketPrefix, locator.SiteID))
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

type ref struct {
	key   string
	value string
}

func refFromComment(comment Comment) *ref {
	result := ref{
		key:   fmt.Sprintf("%s!!%s", comment.Timestamp.Format(time.RFC3339Nano), comment.ID),
		value: fmt.Sprintf("%s!!%s", comment.Locator.URL, comment.ID),
	}
	return &result
}

func refFromValue(val []byte) *ref {
	result := ref{value: string(val)}
	return &result
}

func (r ref) parseValue() (url string, commentID string, err error) {
	elems := strings.Split(r.value, "!!")
	if len(elems) < 2 {
		return "", "", errors.Errorf("can't parse ref %s", r)
	}
	return elems[0], elems[1], nil
}
