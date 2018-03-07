package store

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/bbolt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// BoltDB implements store.Interface, represents multiple sites with multiplexing to different bolt dbs. Thread safe.
// there are 5 types of top-level buckets:
//  - comments for post in "posts" top-level bucket. Each url (post) makes its own bucket and each k:v pair is commentID:comment
//  - history of all comments. They all in a single "last" bucket (per site) and key is defined by ref struct as ts+commentID
//    value is not full comment but a reference combined from post-url+commentID
//  - user to comment references in "users" bucket. It used to get comments for user. Key is userID and value
//    is a nested bucket named userID with kv as ts:reference
//  - blocking info sits in "block" bucket. Key is userID, value - ts
//  - counts per post to keep number of comments. Key is post url, value - count
type BoltDB struct {
	dbs map[string]*bolt.DB
}

const (
	// top level buckets
	postsBucketName  = "posts"
	lastBucketName   = "last"
	userBucketName   = "users"
	blocksBucketName = "block"
	countsBucketName = "counts"

	// limits
	lastLimit = 1000
	userLimit = 50
)

const tsNano = "2006-01-02T15:04:05.000000000Z07:00"

// BoltSite defines single site param
type BoltSite struct {
	FileName string // full path to boltdb
	SiteID   string // ID to access given site
}

// NewBoltDB makes persistent boltdb-based store
func NewBoltDB(sites ...BoltSite) (*BoltDB, error) {
	log.Printf("[INFO] bolt store for sites %+v", sites)
	result := BoltDB{dbs: make(map[string]*bolt.DB)}
	for _, site := range sites {

		db, err := bolt.Open(site.FileName, 0600, &bolt.Options{Timeout: 30 * time.Second})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to make boltdb for %s", site.FileName)
		}

		// make top-level buckets
		err = db.Update(func(tx *bolt.Tx) error {
			topBuckets := []string{postsBucketName, lastBucketName, userBucketName, blocksBucketName, countsBucketName}
			for _, bktName := range topBuckets {
				if _, e := tx.CreateBucketIfNotExists([]byte(bktName)); e != nil {
					return errors.Wrapf(err, "failed to create top level bucket %s", bktName)
				}
			}
			return nil
		})

		if err != nil {
			return nil, errors.Wrap(err, "failed to create top level bucket)")
		}

		result.dbs[site.SiteID] = db
	}
	return &result, nil
}

// Create saves new comment to store. Adds to posts bucket, reference to last and user bucket and increments count bucket
func (b *BoltDB) Create(comment Comment) (commentID string, err error) {

	// fill ID and time if empty
	if comment.ID == "" {
		comment.ID = uuid.New().String()
	}
	if comment.Timestamp.IsZero() {
		comment.Timestamp = time.Now()
	}
	// reset votes if nothing
	if comment.Votes == nil {
		comment.Votes = make(map[string]bool)
	}

	comment.Sanitize() // clear potentially dangerous js from all parts of comment

	bdb, err := b.db(comment.Locator.SiteID)
	if err != nil {
		return "", err
	}
	err = bdb.Update(func(tx *bolt.Tx) error {

		postBkt, e := b.makePostBucket(tx, comment.Locator.URL)
		if e != nil {
			return e
		}

		// check if key already in store, reject doubles
		if postBkt.Get([]byte(comment.ID)) != nil {
			return errors.Errorf("key %s already in store", comment.ID)
		}

		// serialize comment to json []byte for bolt and save
		if e = b.save(postBkt, []byte(comment.ID), comment); e != nil {
			return errors.Wrapf(e, "failed to put key %s to bucket %s", comment.ID, comment.Locator.URL)
		}

		// add reference to comment to "last" bucket
		lastBkt := tx.Bucket([]byte(lastBucketName))
		ref := b.makeRef(comment)
		commentTs := []byte(comment.Timestamp.Format(tsNano))
		e = lastBkt.Put(commentTs, ref)
		if e != nil {
			return errors.Wrapf(e, "can't put reference %s to %s", ref, lastBucketName)
		}

		// add reference to commentID to "users" bucket
		usersBkt := tx.Bucket([]byte(userBucketName))
		// get bucket for userID
		userIDBkt, e := usersBkt.CreateBucketIfNotExists([]byte(comment.User.ID))
		if e != nil {
			return errors.Wrapf(e, "can't get bucket %s", comment.User.ID)
		}
		// put into individual user's bucket with ts as a key
		if e = userIDBkt.Put(commentTs, ref); e != nil {
			return errors.Wrapf(e, "failed to put user comment %s for %s", comment.ID, comment.User.ID)
		}

		if _, e = b.count(tx, comment.Locator.URL, 1); e != nil {
			return errors.Wrapf(e, "failed to increment count for %s", comment.Locator)
		}

		return nil
	})

	return comment.ID, err
}

// Delete removes comment, by locator from the store.
// Posts collection only sets status to deleted and clear fileds in order to prevent breaking trees of replies.
// From last bucket removed for real.
func (b *BoltDB) Delete(locator Locator, commentID string) error {

	bdb, err := b.db(locator.SiteID)
	if err != nil {
		return err
	}

	return bdb.Update(func(tx *bolt.Tx) error {

		postBkt, e := b.getPostBucket(tx, locator.URL)
		if e != nil {
			return e
		}

		comment, err := b.load(postBkt, []byte(commentID))
		if err != nil {
			return errors.Wrapf(err, "can't load key %s from bucket %s", commentID, locator.URL)
		}
		// set deleted status and clear fields
		comment.SetDeleted()

		if err := b.save(postBkt, []byte(commentID), comment); err != nil {
			return errors.Wrapf(err, "can't save deleted comment for key %s from bucket %s", commentID, locator.URL)
		}

		// delete from "last" bucket
		lastBkt := tx.Bucket([]byte(lastBucketName))
		if err := lastBkt.Delete([]byte(commentID)); err != nil {
			return errors.Wrapf(err, "can't delete key %s from bucket %s", commentID, lastBucketName)
		}

		if _, e = b.count(tx, comment.Locator.URL, -1); e != nil {
			return errors.Wrapf(e, "failed to decrement count for %s", comment.Locator)
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

		bucket, e := b.getPostBucket(tx, locator.URL)
		if e != nil {
			return e
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

	comments = sortComments(comments, sortFld)
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
		lastBkt := tx.Bucket([]byte(lastBucketName))
		c := lastBkt.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			url, commentID, e := b.parseRef(v)
			if e != nil {
				return e
			}
			postBkt, e := b.getPostBucket(tx, url)
			if e != nil {
				return e
			}

			comment, e := b.load(postBkt, []byte(commentID))
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
		var e error
		count, e = b.count(tx, locator.URL, 0)
		return e
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
		bucket := tx.Bucket([]byte(blocksBucketName))
		switch status {
		case true:
			if e := bucket.Put([]byte(userID), []byte(time.Now().Format(tsNano))); e != nil {
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
		bucket := tx.Bucket([]byte(blocksBucketName))
		blocked = bucket.Get([]byte(userID)) != nil
		return nil
	})
	return blocked
}

// Blocked get lists of blocked users for given site
// bucket uses userID:
func (b *BoltDB) Blocked(siteID string) (users []BlockedUser, err error) {
	users = []BlockedUser{}
	bdb, err := b.db(siteID)
	if err != nil {
		return nil, err
	}

	err = bdb.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucketName))
		return bucket.ForEach(func(k []byte, v []byte) error {
			ts, e := time.ParseInLocation(tsNano, string(v), time.Local)
			if e != nil {
				return errors.Wrap(e, "can't parse block ts")
			}
			users = append(users, BlockedUser{ID: string(k), Timestamp: ts})
			return nil
		})
	})

	return users, err
}

// List returns list of all commented posts with counters
// uses count bucket to get number of comments
func (b BoltDB) List(siteID string, limit, skip int) (list []PostInfo, err error) {

	bdb, err := b.db(siteID)
	if err != nil {
		return nil, err
	}

	err = bdb.View(func(tx *bolt.Tx) error {
		postsBkt := tx.Bucket([]byte(postsBucketName))

		c := postsBkt.Cursor()
		n := 0
		for k, _ := c.Last(); k != nil; k, _ = c.Prev() {
			n++
			if skip > 0 && n <= skip {
				continue
			}
			postURL := string(k)
			count, e := b.count(tx, postURL, 0)
			if e != nil {
				return e
			}
			list = append(list, PostInfo{URL: postURL, Count: count})
			if limit > 0 && len(list) >= limit {
				break
			}

		}
		return nil
	})

	return list, err
}

// User extracts all comments for given site and given userID
// "users" bucket has sub-bucket for each userID, and keeps it as ts:ref
func (b *BoltDB) User(siteID string, userID string) (comments []Comment, totalComments int, err error) {

	comments = []Comment{}
	commentRefs := []string{}

	bdb, err := b.db(siteID)
	if err != nil {
		return nil, 0, err
	}
	// get list of references to comments
	err = bdb.View(func(tx *bolt.Tx) error {
		usersBkt := tx.Bucket([]byte(userBucketName))
		userIDBkt := usersBkt.Bucket([]byte(userID))
		if userIDBkt == nil {
			return errors.Errorf("no comments for user %s in store", userID)
		}

		c := userIDBkt.Cursor()
		totalComments = 0
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			totalComments++
			if len(commentRefs) <= userLimit {
				commentRefs = append(commentRefs, string(v))
			}
		}
		return nil
	})

	if err != nil {
		return comments, totalComments, err
	}

	// retrieve comments for refs
	for _, v := range commentRefs {
		url, commentID, e := b.parseRef([]byte(v))
		if e != nil {
			return comments, totalComments, errors.Wrapf(e, "can't parse reference %s", v)
		}
		if c, e := b.Get(Locator{SiteID: siteID, URL: url}, commentID); e == nil {
			comments = append(comments, c)
		}
	}

	return comments, totalComments, err
}

// Get for locator.URL and commentID string
func (b *BoltDB) Get(locator Locator, commentID string) (comment Comment, err error) {

	bdb, err := b.db(locator.SiteID)
	if err != nil {
		return comment, err
	}

	err = bdb.View(func(tx *bolt.Tx) error {
		bucket, e := b.getPostBucket(tx, locator.URL)
		if e != nil {
			return e
		}
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
		bucket, e := b.getPostBucket(tx, locator.URL)
		if e != nil {
			return e
		}
		return b.save(bucket, []byte(comment.ID), comment)
	})
}

// getPostBucket return bucket with all comments for postURL
func (b *BoltDB) getPostBucket(tx *bolt.Tx, postURL string) (*bolt.Bucket, error) {
	postsBkt := tx.Bucket([]byte(postsBucketName))
	if postsBkt == nil {
		return nil, errors.Errorf("no bucket %s", postsBucketName)
	}
	res := postsBkt.Bucket([]byte(postURL))
	if res == nil {
		return nil, errors.Errorf("no bucket %s in store", postURL)
	}
	return res, nil
}

// makePostBucket create new bucket for postURL as a key. This bucket holds all comments for the post.
func (b *BoltDB) makePostBucket(tx *bolt.Tx, postURL string) (*bolt.Bucket, error) {
	postsBkt := tx.Bucket([]byte(postsBucketName))
	if postsBkt == nil {
		return nil, errors.Errorf("no bucket %s", postsBucketName)
	}
	res, err := postsBkt.CreateBucketIfNotExists([]byte(postURL))
	if err != nil {
		return nil, errors.Wrapf(err, "no bucket %s in store", postURL)
	}
	return res, nil
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

// count adds val to counts key postURL. val can be negative to subtract. if val 0 can be used as accessor
// it uses separate counts bucket because boltdb Stat call is very slow
func (b *BoltDB) count(tx *bolt.Tx, postURL string, val int) (int, error) {

	btoi := func(v []byte) int {
		res, _ := strconv.Atoi(string(v))
		return res
	}

	itob := func(v int) []byte {
		return []byte(strconv.Itoa(v))
	}

	countBkt := tx.Bucket([]byte(countsBucketName))
	countVal := countBkt.Get([]byte(postURL))
	if countVal == nil {
		countVal = itob(0)
	}
	if val == 0 {
		return btoi(countVal), nil
	}
	updatedCount := btoi(countVal) + val
	return updatedCount, countBkt.Put([]byte(postURL), itob(updatedCount))
}

func (b *BoltDB) db(siteID string) (*bolt.DB, error) {
	if res, ok := b.dbs[siteID]; ok {
		return res, nil
	}
	return nil, errors.Errorf("site %q not found", siteID)
}

// makeRef creates reference combining url and comment id
func (b *BoltDB) makeRef(comment Comment) []byte {
	return []byte(fmt.Sprintf("%s!!%s", comment.Locator.URL, comment.ID))
}

// parseRef gets parts of reference
func (b *BoltDB) parseRef(val []byte) (url string, id string, err error) {
	elems := strings.Split(string(val), "!!")
	if len(elems) != 2 {
		return "", "", errors.Errorf("invalid reference value %s", string(val))
	}
	return elems[0], elems[1], nil
}
