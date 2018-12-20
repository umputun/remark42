package engine

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	bolt "github.com/coreos/bbolt"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/store"
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
//  - readonly per post to keep status of manually set RO posts. Key is post url, value - ts
type BoltDB struct {
	dbs map[string]*bolt.DB
}

const (
	// top level buckets
	postsBucketName    = "posts"
	lastBucketName     = "last"
	userBucketName     = "users"
	blocksBucketName   = "block"
	infoBucketName     = "info"
	readonlyBucketName = "readonly"
	verifiedBucketName = "verified"

	tsNano = "2006-01-02T15:04:05.000000000Z07:00"
)

// BoltSite defines single site param
type BoltSite struct {
	FileName string // full path to boltdb
	SiteID   string // ID to access given site
}

// NewBoltDB makes persistent boltdb-based store
func NewBoltDB(options bolt.Options, sites ...BoltSite) (*BoltDB, error) {
	log.Printf("[INFO] bolt store for sites %+v", sites)
	result := BoltDB{dbs: make(map[string]*bolt.DB)}
	for _, site := range sites {
		db, err := bolt.Open(site.FileName, 0600, &options) // bolt.Options{Timeout: 30 * time.Second}
		if err != nil {
			return nil, errors.Wrapf(err, "failed to make boltdb for %s", site.FileName)
		}

		// make top-level buckets
		topBuckets := []string{postsBucketName, lastBucketName, userBucketName, blocksBucketName,
			infoBucketName, readonlyBucketName, verifiedBucketName}
		err = db.Update(func(tx *bolt.Tx) error {
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
func (b *BoltDB) Create(comment store.Comment) (commentID string, err error) {

	bdb, err := b.db(comment.Locator.SiteID)
	if err != nil {
		return "", err
	}

	if b.IsReadOnly(comment.Locator) {
		return "", errors.Errorf("post %s is read-only", comment.Locator.URL)
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

		ref := b.makeRef(comment)

		// add reference to comment to "last" bucket
		lastBkt := tx.Bucket([]byte(lastBucketName))
		commentTs := []byte(comment.Timestamp.Format(tsNano))
		e = lastBkt.Put(commentTs, ref)
		if e != nil {
			return errors.Wrapf(e, "can't put reference %s to %s", ref, lastBucketName)
		}

		// add reference to commentID to "users" bucket
		userBkt, e := b.getUserBucket(tx, comment.User.ID)
		if e != nil {
			return errors.Wrapf(e, "can't get bucket %s", comment.User.ID)
		}
		// put into individual user's bucket with ts as a key
		if e = userBkt.Put(commentTs, ref); e != nil {
			return errors.Wrapf(e, "failed to put user comment %s for %s", comment.ID, comment.User.ID)
		}

		// set info with countfor post url
		if _, e = b.setInfo(tx, comment); e != nil {
			return errors.Wrapf(e, "failed to set info for %s", comment.Locator)
		}
		return nil
	})

	return comment.ID, err
}

// Find returns all comments for post and sorts results
func (b *BoltDB) Find(locator store.Locator, sortFld string) (comments []store.Comment, err error) {
	comments = []store.Comment{}

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
			comment := store.Comment{}
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
func (b *BoltDB) Last(siteID string, max int) (comments []store.Comment, err error) {

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

			comment := store.Comment{}
			if e := b.load(postBkt, []byte(commentID), &comment); e != nil {
				log.Printf("[WARN] can't load comment for %s from store %s", commentID, url)
				continue
			}
			if comment.Deleted {
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
func (b *BoltDB) Count(locator store.Locator) (count int, err error) {

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

// List returns list of all commented posts with counters
// uses count bucket to get number of comments
func (b BoltDB) List(siteID string, limit, skip int) (list []store.PostInfo, err error) {

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
			infoBkt := tx.Bucket([]byte(infoBucketName))
			info := store.PostInfo{}
			if e := b.load(infoBkt, []byte(postURL), &info); e != nil {
				return errors.Wrapf(e, "can't load info for %s", postURL)
			}
			list = append(list, info)
			if limit > 0 && len(list) >= limit {
				break
			}
		}
		return nil
	})

	return list, err
}

// Info returns time range and count for locator
func (b *BoltDB) Info(locator store.Locator, readOnlyAge int) (store.PostInfo, error) {
	bdb, err := b.db(locator.SiteID)
	if err != nil {
		return store.PostInfo{}, err
	}

	info := store.PostInfo{}
	err = bdb.View(func(tx *bolt.Tx) error {
		infoBkt := tx.Bucket([]byte(infoBucketName))
		if e := b.load(infoBkt, []byte(locator.URL), &info); e != nil {
			return errors.Wrapf(e, "can't load info for %s", locator.URL)
		}
		return nil
	})

	// set read-only from age and manual bucket
	info.ReadOnly = readOnlyAge > 0 && !info.FirstTS.IsZero() && info.FirstTS.AddDate(0, 0, readOnlyAge).Before(time.Now())
	if b.IsReadOnly(locator) {
		info.ReadOnly = true
	}
	return info, err
}

// User extracts all comments for given site and given userID
// "users" bucket has sub-bucket for each userID, and keeps it as ts:ref
func (b *BoltDB) User(siteID, userID string, limit, skip int) (comments []store.Comment, err error) {

	comments = []store.Comment{}
	commentRefs := []string{}

	bdb, err := b.db(siteID)
	if err != nil {
		return nil, err
	}

	if limit == 0 || limit > userLimit {
		limit = userLimit
	}

	// get list of references to comments
	err = bdb.View(func(tx *bolt.Tx) error {
		usersBkt := tx.Bucket([]byte(userBucketName))
		userIDBkt := usersBkt.Bucket([]byte(userID))
		if userIDBkt == nil {
			return errors.Errorf("no comments for user %s in store", userID)
		}

		c := userIDBkt.Cursor()
		skipComments := 0
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			if len(commentRefs) >= limit {
				break
			}
			if skip > 0 && skipComments < skip {
				skipComments++
				continue
			}
			commentRefs = append(commentRefs, string(v))
		}
		return nil
	})

	if err != nil {
		return comments, err
	}

	// retrieve comments for refs
	for _, v := range commentRefs {
		url, commentID, e := b.parseRef([]byte(v))
		if e != nil {
			return comments, errors.Wrapf(e, "can't parse reference %s", v)
		}
		if c, e := b.Get(store.Locator{SiteID: siteID, URL: url}, commentID); e == nil {
			comments = append(comments, c)
		}
	}

	return comments, err
}

// UserCount returns number of comments for user
func (b *BoltDB) UserCount(siteID, userID string) (int, error) {
	bdb, err := b.db(siteID)
	if err != nil {
		return 0, err
	}
	count := 0
	err = bdb.View(func(tx *bolt.Tx) error {
		usersBkt := tx.Bucket([]byte(userBucketName))
		userIDBkt := usersBkt.Bucket([]byte(userID))
		if userIDBkt == nil {
			return errors.Errorf("no comments for user %s in store", userID)
		}
		stats := userIDBkt.Stats()
		count = stats.KeyN
		return nil
	})
	return count, err
}

// Get returns comment for locator.URL and commentID string
func (b *BoltDB) Get(locator store.Locator, commentID string) (comment store.Comment, err error) {

	bdb, err := b.db(locator.SiteID)
	if err != nil {
		return comment, err
	}

	err = bdb.View(func(tx *bolt.Tx) error {
		bucket, e := b.getPostBucket(tx, locator.URL)
		if e != nil {
			return e
		}
		return b.load(bucket, []byte(commentID), &comment)
	})
	return comment, err
}

// Put updates comment for locator.URL with mutable part of comment
func (b *BoltDB) Put(locator store.Locator, comment store.Comment) error {

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

// Close boltdb store
func (b *BoltDB) Close() error {
	errs := new(multierror.Error)
	for site, db := range b.dbs {
		err := errors.Wrapf(db.Close(), "can't close site %s", site)
		errs = multierror.Append(errs, err)
	}
	return errs.ErrorOrNil()
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

func (b *BoltDB) getUserBucket(tx *bolt.Tx, userID string) (*bolt.Bucket, error) {
	usersBkt := tx.Bucket([]byte(userBucketName))
	userIDBkt, e := usersBkt.CreateBucketIfNotExists([]byte(userID)) // get bucket for userID
	if e != nil {
		return nil, errors.Wrapf(e, "can't get bucket %s", userID)
	}
	return userIDBkt, nil
}

// save marshaled value to key for bucket. Should run in update tx
func (b *BoltDB) save(bkt *bolt.Bucket, key []byte, value interface{}) (err error) {
	if value == nil {
		return errors.Errorf("can't save nil value for %s", key)
	}
	jdata, jerr := json.Marshal(value)
	if jerr != nil {
		return errors.Wrap(jerr, "can't marshal comment")
	}
	if err = bkt.Put(key, jdata); err != nil {
		return errors.Wrapf(err, "failed to save key %s", key)
	}
	return nil
}

// load and unmarshal json value by key from bucket. Should run in view tx
func (b *BoltDB) load(bkt *bolt.Bucket, key []byte, res interface{}) error {
	value := bkt.Get(key)
	if value == nil {
		return errors.Errorf("no value for %s", key)
	}

	if err := json.Unmarshal(value, &res); err != nil {
		return errors.Wrap(err, "failed to unmarshal")
	}
	return nil
}

// count adds val to counts key postURL. val can be negative to subtract. if val 0 can be used as accessor
// it uses separate counts bucket because boltdb Stat call is very slow
func (b *BoltDB) count(tx *bolt.Tx, postURL string, val int) (int, error) {

	infoBkt := tx.Bucket([]byte(infoBucketName))

	info := store.PostInfo{}
	if err := b.load(infoBkt, []byte(postURL), &info); err != nil {
		info = store.PostInfo{}
	}
	if val == 0 { // get current count, don't update
		return info.Count, nil
	}
	info.Count += val

	return info.Count, b.save(infoBkt, []byte(postURL), &info)
}

func (b *BoltDB) setInfo(tx *bolt.Tx, comment store.Comment) (store.PostInfo, error) {
	infoBkt := tx.Bucket([]byte(infoBucketName))
	info := store.PostInfo{}
	if err := b.load(infoBkt, []byte(comment.Locator.URL), &info); err != nil {
		info = store.PostInfo{
			Count:   0,
			URL:     comment.Locator.URL,
			FirstTS: comment.Timestamp,
			LastTS:  comment.Timestamp,
		}
	}
	info.Count++
	info.LastTS = comment.Timestamp
	return info, b.save(infoBkt, []byte(comment.Locator.URL), &info)
}

func (b *BoltDB) db(siteID string) (*bolt.DB, error) {
	if res, ok := b.dbs[siteID]; ok {
		return res, nil
	}
	return nil, errors.Errorf("site %q not found", siteID)
}

// makeRef creates reference combining url and comment id
func (b *BoltDB) makeRef(comment store.Comment) []byte {
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
