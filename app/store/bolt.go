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

// BoltDB implements store.Interface, represents multiple sites with multiplexing to different bolt dbs. Thread safe.
// there are 4 types of buckets:
//  - comments for post. Each url (post) makes its own bucket and each k:v pair is commentID:comment
//  - history of all comments. They all in a single "last" bucket (per site) and key is defined by ref struct as ts+commentID
//    value is not full comment but a reference combined from post-url+commentID
//  - user to comment references in "users" bucket. It used to get comments for user. Key is userID and value
//    is a nested bucket named userID with kv as ts:reference
//  - blocking info sits in "block" bucket. Key is userID, value - ts
type BoltDB struct {
	dbs map[string]*bolt.DB
}

const (
	// top level buckets
	lastBucketName   = "last"
	userBucketName   = "users"
	blocksBucketName = "block"

	// limits
	lastLimit = 1000
	userLimit = 100
)

// BoltSite defines single site param
type BoltSite struct {
	FileName string
	SiteID   string
}

// NewBoltDB makes persistent boltdb-based store
func NewBoltDB(sites ...BoltSite) (*BoltDB, error) {
	log.Printf("[INFO] bolt store for sites %+v", sites)
	result := BoltDB{dbs: make(map[string]*bolt.DB)}
	for _, site := range sites {
		db, err := bolt.Open(site.FileName, 0600, &bolt.Options{Timeout: 5 * time.Second})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to make boltdb for %s", site.FileName)
		}
		result.dbs[site.SiteID] = db
	}
	return &result, nil
}

// Create saves new comment to store
func (b *BoltDB) Create(comment Comment) (commentID string, err error) {

	// fill ID and time if empty
	if comment.ID == "" {
		comment.ID = makeCommentID()
	}
	if comment.Timestamp.IsZero() {
		comment.Timestamp = time.Now()
	}
	if comment.Votes == nil {
		comment.Votes = make(map[string]bool)
	}

	comment = sanitizeComment(comment) // clear potentially dangerous js from all parts of comment

	bdb, err := b.db(comment.Locator.SiteID)
	if err != nil {
		return "", err
	}
	err = bdb.Update(func(tx *bolt.Tx) error {
		bucket, e := tx.CreateBucketIfNotExists([]byte(comment.Locator.URL)) // bucket per post url
		if e != nil {
			return errors.Wrapf(e, "can't make or open bucket", comment.Locator.URL)
		}

		// check if key already in store, reject doubles
		if bucket.Get([]byte(comment.ID)) != nil {
			return errors.Errorf("key %s already in store", comment.ID)
		}

		// serialize comment to json []byte for bolt and save
		if e = b.save(bucket, []byte(comment.ID), comment); e != nil {
			return errors.Wrapf(e, "failed to put key %s to bucket %s", comment.ID, comment.Locator.URL)
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
		if e = userBkt.Put([]byte(comment.Timestamp.Format(time.RFC3339Nano)), []byte(rv.value)); e != nil {
			return errors.Wrapf(e, "failed to put user comment %s for %s", comment.ID, comment.User.ID)
		}
		return nil
	})

	return comment.ID, err
}

// Delete removes comment, by locator from the store
func (b *BoltDB) Delete(locator Locator, commentID string) error {

	bdb, err := b.db(locator.SiteID)
	if err != nil {
		return err
	}

	return bdb.Update(func(tx *bolt.Tx) error {
		// delete from post bucket
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

// Find returns all comments for post and sorts results
func (b *BoltDB) Find(locator Locator, sortFld string) (comments []Comment, err error) {
	comments = []Comment{}

	bdb, err := b.db(locator.SiteID)
	if err != nil {
		return nil, err
	}

	err = bdb.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(locator.URL))
		if bucket == nil {
			return errors.Errorf("no bucket %s in store", locator.URL)
		}

		return bucket.ForEach(func(k, v []byte) error {
			comment := Comment{}
			if e := json.Unmarshal(v, &comment); e != nil {
				return errors.Wrap(e, "failed to unmarshal")
			}
			comments = append(comments, comment)
			return nil
		})
	})

	// sort result according to sortFld
	sort.Slice(comments, func(i, j int) bool {
		switch sortFld {
		case "+time", "-time", "time":
			if strings.HasPrefix(sortFld, "-") {
				return comments[i].Timestamp.After(comments[j].Timestamp)
			}
			return comments[i].Timestamp.Before(comments[j].Timestamp)

		case "+score", "-score", "score":
			if strings.HasPrefix(sortFld, "-") {
				return comments[i].Score > comments[j].Score
			}
			return comments[i].Score < comments[j].Score

		default:
			return comments[i].Timestamp.Before(comments[j].Timestamp)
		}
	})

	return comments, err
}

// Last returns up to max last comments for given siteID
func (b *BoltDB) Last(siteID string, max int) (comments []Comment, err error) {

	if max > lastLimit || max == 0 {
		max = lastLimit
	}

	bdb, err := b.db(siteID)
	if err != nil {
		return nil, err
	}

	err = bdb.View(func(tx *bolt.Tx) error {
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

			comment, e := b.load(urlBucket, []byte(commentID))
			if e != nil {
				log.Printf("[WARN] can't load comment for %s from store %s", commentID, url)
				continue
			}

			comments = append(comments, comment)
			if len(comments) >= max {
				break
			}
		}
		return nil
	})

	return comments, err
}

// Count returns number of comments for locator
func (b *BoltDB) Count(locator Locator) (count int, err error) {

	bdb, err := b.db(locator.SiteID)
	if err != nil {
		return 0, err
	}

	err = bdb.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(locator.URL))
		if bucket == nil {
			return errors.Errorf("no bucket %s in store %s", locator.URL, locator.SiteID)
		}
		count = bucket.Stats().KeyN
		return nil
	})

	return count, err
}

// SetBlock blocks/unblocks user for given site
func (b *BoltDB) SetBlock(siteID string, userID string, status bool) error {

	bdb, err := b.db(siteID)
	if err != nil {
		return err
	}

	return bdb.Update(func(tx *bolt.Tx) error {

		bucket, e := tx.CreateBucketIfNotExists([]byte(blocksBucketName))
		if e != nil {
			return errors.Errorf("no bucket %s in store", blocksBucketName)
		}

		switch status {
		case true:
			if e := bucket.Put([]byte(userID), []byte(time.Now().Format(time.RFC3339Nano))); e != nil {
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
func (b *BoltDB) IsBlocked(siteID string, userID string) (blocked bool) {

	bdb, err := b.db(siteID)
	if err != nil {
		return false
	}

	_ = bdb.View(func(tx *bolt.Tx) error {
		blocked = false
		bucket := tx.Bucket([]byte(blocksBucketName))
		if bucket != nil && bucket.Get([]byte(userID)) != nil {
			blocked = true
		}
		return nil
	})
	return blocked
}

// List returns list of buckets, which is list of all commented posts
func (b BoltDB) List(siteID string) (list []PostInfo, err error) {

	bdb, err := b.db(siteID)
	if err != nil {
		return nil, err
	}

	err = bdb.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, bkt *bolt.Bucket) error {
			postURL := string(name)
			if postURL != lastBucketName && postURL != userBucketName {
				list = append(list, PostInfo{URL: postURL, Count: bkt.Stats().KeyN})
			}
			return nil
		})
	})

	return list, err
}

// User extracts all comments for given site and given userID
// "users" bucket has sub-bucket for each userID, and keeps it as ts:ref
func (b *BoltDB) User(siteID string, userID string) (comments []Comment, err error) {

	comments = []Comment{}
	commentRefs := []string{}

	bdb, err := b.db(siteID)
	if err != nil {
		return nil, err
	}
	// get list of references to comments
	err = bdb.View(func(tx *bolt.Tx) error {
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

	// retrieve comments for refs
	for _, v := range commentRefs {
		url, commentID, e := ref{value: v}.parseValue()
		if e != nil {
			return comments, errors.Wrapf(e, "can't parse reference %s", v)
		}
		if c, e := b.Get(Locator{SiteID: siteID, URL: url}, commentID); e == nil {
			comments = append(comments, c)
		}
	}

	return comments, err
}

// Get for locator.URL and commentID string
func (b *BoltDB) Get(locator Locator, commentID string) (comment Comment, err error) {

	bdb, err := b.db(locator.SiteID)
	if err != nil {
		return comment, err
	}

	err = bdb.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(locator.URL))
		if bucket == nil {
			return errors.Errorf("no bucket %s in store", locator.URL)
		}
		var e error
		comment, e = b.load(bucket, []byte(commentID))
		return e
	})
	return comment, err
}

// Put updates comment for locator.URL with mutable part of comment
func (b *BoltDB) Put(locator Locator, comment Comment) error {

	if curComment, err := b.Get(locator, comment.ID); err == nil {
		// preserve immutable fields
		comment.ParentID = curComment.ParentID
		comment.Locator = curComment.Locator
		comment.Timestamp = curComment.Timestamp
		comment.User = curComment.User
	}

	bdb, err := b.db(locator.SiteID)
	if err != nil {
		return err
	}

	return bdb.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(locator.URL))
		if bucket == nil {
			return errors.Errorf("no bucket %s in store", locator.URL)
		}
		return b.save(bucket, []byte(comment.ID), comment)
	})
}

// save comment to key for bucket. Should run in update tx
func (b *BoltDB) save(bkt *bolt.Bucket, key []byte, comment Comment) (err error) {
	jdata, jerr := json.Marshal(&comment)
	if jerr != nil {
		return errors.Wrap(jerr, "can't marshal comment")
	}
	if err = bkt.Put([]byte(comment.ID), jdata); err != nil {
		return errors.Wrapf(err, "failed to save key %s", key)
	}
	return nil
}

// load comment by key from bucket. Should run in view tx
func (b *BoltDB) load(bkt *bolt.Bucket, key []byte) (comment Comment, err error) {
	commentVal := bkt.Get(key)
	if commentVal == nil {
		return comment, errors.Errorf("no comment for %s", key)
	}

	if err = json.Unmarshal(commentVal, &comment); err != nil {
		return comment, errors.Wrap(err, "failed to unmarshal")
	}
	return comment, nil
}

func (b *BoltDB) db(siteID string) (*bolt.DB, error) {
	if res, ok := b.dbs[siteID]; ok {
		return res, nil
	}
	return nil, errors.Errorf("site %q not found", siteID)
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
