package store

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
)

// BoltDB implements store.Interface. Each instance represents one site. Thread safe.
// there are 4 types of buckets:
//  - comments for post. Each url (post) makes it's own bucket and each k:v pair is commentID:comment
//  - history of all comments. They all in a single "last" bucket and key is defined by ref struct as ts+commentID
//    value is not full comment but a reference combined from post-url+commentID
//  - user to comment references in "users" bucket. It used to get comments for user. Key is userID and value
//    is a bucket with ts:reference
//  - blocking info sits in "block" bucket. Key is userID, value - ts
type BoltDB struct {
	*bolt.DB
}

const (
	lastBucketName   = "last"
	userBucketName   = "users"
	blocksBucketName = "block"
	lastLimit        = 1000
	userLimit        = 100
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

	// fille ID and time if empty
	if comment.ID == "" {
		comment.ID = makeCommentID()
	}
	if comment.Timestamp.IsZero() {
		comment.Timestamp = time.Now()
	}
	comment.Votes = make(map[string]bool)
	comment = sanitizeComment(comment) // clear potentially dangerous js from all parts of comment

	err := b.Update(func(tx *bolt.Tx) error {
		bucket, e := tx.CreateBucketIfNotExists([]byte(comment.Locator.URL)) // bucket per post url
		if e != nil {
			return errors.Wrapf(e, "can't make or open bucket", comment.Locator.URL)
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

		// add reference to commentID to "users" bucket
		bucket, e = tx.CreateBucketIfNotExists([]byte(userBucketName))
		if e != nil {
			return errors.Wrapf(e, "can't make bucket %s", userBucketName)
		}
		// get bucket for userID
		userBkt, e := bucket.CreateBucketIfNotExists([]byte(comment.User.ID))
		if e != nil {
			return errors.Wrapf(e, "can't get bucket %s", comment.User.ID)
		}
		// put into individual user's bucket with ts as a key
		if e = userBkt.Put([]byte(comment.Timestamp.Format(time.RFC3339)), []byte(rv.value)); e != nil {
			return errors.Wrapf(e, "failed to put user comment %s for %s", comment.ID, comment.User.ID)
		}
		return nil
	})

	return comment.ID, err
}

// Delete removes comment locator from the store
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

	// sort result according to request.Sort
	sort.Slice(res, func(i, j int) bool {
		switch request.Sort {
		case "+time", "-time", "time":
			if strings.HasPrefix(request.Sort, "-") {
				return res[i].Timestamp.After(res[j].Timestamp)
			}
			return res[i].Timestamp.Before(res[j].Timestamp)

		case "+score", "-score", "score":
			if strings.HasPrefix(request.Sort, "-") {
				return res[i].Score > res[j].Score
			}
			return res[i].Score < res[j].Score

		default:
			return res[i].Timestamp.Before(res[j].Timestamp)
		}
	})

	return res, err
}

// GetByID returns comment by id across posts
func (b *BoltDB) GetByID(locator Locator, commentID string) (comment Comment, err error) {

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
		return errors.Errorf("no comment id %s", commentID)
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
				break
			}
		}
		return nil
	})

	return result, err
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
	return b.Update(func(tx *bolt.Tx) error {

		bucket, e := tx.CreateBucketIfNotExists([]byte(blocksBucketName))
		if e != nil {
			return errors.Errorf("no bucket %s in store", blocksBucketName)
		}

		switch status {
		case true:
			if e := bucket.Put([]byte(userID), []byte(time.Now().Format(time.RFC3339))); e != nil {
				return errors.Wrapf(e, "failed to put %s to %s", userID, blocksBucketName)
			}
		case false:
			if e := bucket.Delete([]byte(userID)); e != nil {
				return errors.Wrapf(e, "failed to clean %s from %s", userID, blocksBucketName)
			}
		}
		return nil
	})
}

// IsBlocked checks if user blocked
func (b *BoltDB) IsBlocked(locator Locator, userID string) (result bool) {
	_ = b.View(func(tx *bolt.Tx) error {
		result = false
		bucket := tx.Bucket([]byte(blocksBucketName))
		if bucket != nil && bucket.Get([]byte(userID)) != nil {
			result = true
		}
		return nil
	})
	return result
}

// List returns list of buckets, which is list of all commented posts
func (b BoltDB) List(locator Locator) (result []string, err error) {
	err = b.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
			if string(name) != lastBucketName && string(name) != userBucketName {
				result = append(result, string(name))
			}
			return nil
		})
	})
	return result, err
}

// GetByUser extracts all comments for given site and given userID
// "users" bucket has sub-bucket for each userID, and keeps it as ts:ref
func (b *BoltDB) GetByUser(locator Locator, userID string) (comments []Comment, err error) {

	comments = []Comment{}
	commentRefs := []string{}

	// get list of references to comments
	err = b.View(func(tx *bolt.Tx) error {
		userBucket := tx.Bucket([]byte(userBucketName))
		if userBucket == nil {
			return errors.Errorf("no bucket %s in store", userBucketName)
		}

		userBkt := userBucket.Bucket([]byte(userID))
		if userBkt == nil {
			return errors.Errorf("no comments for user %s in store", userID)
		}

		c := userBkt.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			commentRefs = append(commentRefs, string(v))
			if len(commentRefs) > userLimit {
				break
			}
		}
		return nil
	})

	if err != nil {
		return comments, err
	}

	// retrive comments for refs
	for _, v := range commentRefs {
		url, commentID, e := ref{value: v}.parseValue()
		if e != nil {
			return comments, errors.Wrapf(e, "can't parse reference %s", v)
		}
		if c, e := b.GetByID(Locator{URL: url, SiteID: locator.SiteID}, commentID); e == nil {
			comments = append(comments, c)
		}
	}

	return comments, err
}

// GetComment for locator.URL and commentID string
func (b *BoltDB) GetComment(locator Locator, commentID string) (comment Comment, err error) {

	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(locator.URL))
		if bucket == nil {
			return errors.Errorf("no bucket %s in store", locator.URL)
		}

		// get and unmarshal comment
		commentVal := bucket.Get([]byte(commentID))
		if commentVal == nil {
			return errors.Errorf("no comment for %s in store %s", commentID, locator.URL)
		}

		if e := json.Unmarshal(commentVal, &comment); e != nil {
			return errors.Wrap(e, "failed to unmarshal")
		}
		return nil
	})
	return comment, err
}

// PutComment updates comment for locator.URL with mutable part of comment
func (b *BoltDB) PutComment(locator Locator, comment Comment) error {

	if curComment, err := b.GetComment(locator, comment.ID); err == nil {
		// preserve immutable fields
		comment.ParentID = curComment.ParentID
		comment.Locator = curComment.Locator
		comment.Timestamp = curComment.Timestamp
		comment.User = curComment.User
	}

	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(locator.URL))
		if bucket == nil {
			return errors.Errorf("no bucket %s in store", locator.URL)
		}

		// serialize comment to json []byte for bolt and save
		jdata, jerr := json.Marshal(&comment)
		if jerr != nil {
			return errors.Wrap(jerr, "can't marshal comment")
		}
		if err := bucket.Put([]byte(comment.ID), jdata); err != nil {
			return errors.Wrapf(err, "failed to put key %s to bucket %s", comment.ID, locator.URL)
		}
		return nil
	})
}

// ref represents key:value pair for extra, index-only buckets
type ref struct {
	key   string
	value string
}

// refFromComment makes reference record used for related buckets referencing prim data set
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
