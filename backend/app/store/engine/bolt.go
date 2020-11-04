package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"

	"github.com/umputun/remark42/backend/app/store"
)

// BoltDB implements store.Interface, represents multiple sites with multiplexing to different bolt dbs. Thread safe.
// there are 6 types of top-level buckets:
//  - comments for post in "posts" top-level bucket. Each url (post) makes its own bucket and each k:v pair is commentID:comment
//  - history of all comments. They all in a single "last" bucket (per site) and key is defined by ref struct as ts+commentID
//    value is not full comment but a reference combined from post-url+commentID
//  - user to comment references in "users" bucket. It used to get comments for user. Key is userID and value
//    is a nested bucket named userID with kv as ts:reference
//  - users details in "user_details" bucket. Key is userID, value - UserDetailEntry
//  - blocking info sits in "block" bucket. Key is userID, value - ts
//  - counts per post to keep number of comments. Key is post url, value - count
//  - readonly per post to keep status of manually set RO posts. Key is post url, value - ts
type BoltDB struct {
	dbs map[string]*bolt.DB
}

const (
	// top level buckets
	postsBucketName       = "posts"
	lastBucketName        = "last"
	userBucketName        = "users"
	userDetailsBucketName = "user_details"
	blocksBucketName      = "block"
	infoBucketName        = "info"
	readonlyBucketName    = "readonly"
	verifiedBucketName    = "verified"

	tsNano = "2006-01-02T15:04:05.000000000Z07:00"
)

// BoltSite defines single site param
type BoltSite struct {
	FileName string // full path to boltdb
	SiteID   string // ID of given site
}

// NewBoltDB makes persistent boltdb-based store. For each site new boltdb file created
func NewBoltDB(options bolt.Options, sites ...BoltSite) (*BoltDB, error) {
	log.Printf("[INFO] bolt store for sites %+v, options %+v", sites, options)
	result := BoltDB{dbs: make(map[string]*bolt.DB)}
	for _, site := range sites {
		db, err := bolt.Open(site.FileName, 0600, &options) //nolint:gocritic //octalLiteral is OK as FileMode
		if err != nil {
			return nil, errors.Wrapf(err, "failed to make boltdb for %s", site.FileName)
		}

		// make top-level buckets
		topBuckets := []string{postsBucketName, lastBucketName, userBucketName, userDetailsBucketName,
			blocksBucketName, infoBucketName, readonlyBucketName, verifiedBucketName}
		err = db.Update(func(tx *bolt.Tx) error {
			for _, bktName := range topBuckets {
				if _, e := tx.CreateBucketIfNotExists([]byte(bktName)); e != nil {
					return errors.Wrapf(e, "failed to create top level bucket %s", bktName)
				}
			}
			return nil
		})

		if err != nil {
			return nil, errors.Wrap(err, "failed to create top level bucket)")
		}

		result.dbs[site.SiteID] = db
		log.Printf("[DEBUG] bolt store created for %s", site.SiteID)
	}
	return &result, nil
}

// Create saves new comment to store. Adds to posts bucket, reference to last and user bucket and increments count bucket
func (b *BoltDB) Create(comment store.Comment) (commentID string, err error) {
	bdb, err := b.db(comment.Locator.SiteID)
	if err != nil {
		return "", err
	}

	if b.checkFlag(FlagRequest{Locator: comment.Locator, Flag: ReadOnly}) {
		return "", errors.Errorf("post %s is read-only", comment.Locator.URL)
	}

	err = bdb.Update(func(tx *bolt.Tx) (err error) {
		var postBkt, lastBkt, userBkt *bolt.Bucket

		if postBkt, err = b.makePostBucket(tx, comment.Locator.URL); err != nil {
			return err
		}
		// check if key already in store, reject doubles
		if postBkt.Get([]byte(comment.ID)) != nil {
			return errors.Errorf("key %s already in store", comment.ID)
		}

		// serialize comment to json []byte for bolt and save
		if err = b.save(postBkt, comment.ID, comment); err != nil {
			return errors.Wrapf(err, "failed to put key %s to bucket %s", comment.ID, comment.Locator.URL)
		}

		ref := b.makeRef(comment) // reference combines url and comment id

		// add reference to comment to "last" bucket
		lastBkt = tx.Bucket([]byte(lastBucketName))
		commentTS := []byte(comment.Timestamp.Format(tsNano))
		if err = lastBkt.Put(commentTS, ref); err != nil {
			return errors.Wrapf(err, "can't put reference %s to %s", ref, lastBucketName)
		}

		// add reference to commentID to "users" bucket
		if userBkt, err = b.getUserBucket(tx, comment.User.ID); err != nil {
			return errors.Wrapf(err, "can't get bucket %s", comment.User.ID)
		}
		// put into individual user's bucket with ts as a key
		if err = userBkt.Put(commentTS, ref); err != nil {
			return errors.Wrapf(err, "failed to put user comment %s for %s", comment.ID, comment.User.ID)
		}

		// set info with the count for post url
		if _, err = b.setInfo(tx, comment); err != nil {
			return errors.Wrapf(err, "failed to set info for %s", comment.Locator)
		}
		return nil
	})

	return comment.ID, err
}

// Get returns comment for locator.URL and commentID string
func (b *BoltDB) Get(req GetRequest) (comment store.Comment, err error) {

	bdb, err := b.db(req.Locator.SiteID)
	if err != nil {
		return comment, err
	}

	err = bdb.View(func(tx *bolt.Tx) error {
		bucket, e := b.getPostBucket(tx, req.Locator.URL)
		if e != nil {
			return e
		}
		return b.load(bucket, req.CommentID, &comment)
	})
	return comment, err
}

// Find returns all comments for given request and sorts results
func (b *BoltDB) Find(req FindRequest) (comments []store.Comment, err error) {
	comments = []store.Comment{}

	bdb, err := b.db(req.Locator.SiteID)
	if err != nil {
		return nil, err
	}

	switch {
	case req.Locator.SiteID != "" && req.Locator.URL != "": // find post comments, i.e. for site and url
		err = bdb.View(func(tx *bolt.Tx) error {

			bucket, e := b.getPostBucket(tx, req.Locator.URL)
			if e != nil {
				return e
			}

			return bucket.ForEach(func(k, v []byte) error {
				comment := store.Comment{}
				if e = json.Unmarshal(v, &comment); e != nil {
					return errors.Wrap(e, "failed to unmarshal")
				}
				if req.Since.IsZero() || comment.Timestamp.After(req.Since) {
					comments = append(comments, comment)
				}
				return nil
			})
		})
	case req.Locator.SiteID != "" && req.Locator.URL == "" && req.UserID == "": // find last comments for site
		comments, err = b.lastComments(req.Locator.SiteID, req.Limit, req.Since)
	case req.Locator.SiteID != "" && req.UserID != "": // find comments for user
		comments, err = b.userComments(req.Locator.SiteID, req.UserID, req.Limit, req.Skip)
	}

	if err != nil {
		return nil, err
	}
	return SortComments(comments, req.Sort), nil
}

// Flag sets and gets flag values
func (b *BoltDB) Flag(req FlagRequest) (val bool, err error) {
	if req.Update == FlagNonSet { // read flag value, no update requested
		return b.checkFlag(req), nil
	}

	// write flag value
	return b.setFlag(req)
}

// UserDetail sets or gets single detail value, or gets all details for requested site.
// UserDetail returns list even for single entry request is a compromise in order to have both single detail getting and setting
// and all site's details listing under the same function (and not to extend interface by two separate functions).
func (b *BoltDB) UserDetail(req UserDetailRequest) ([]UserDetailEntry, error) {
	switch req.Detail {
	case UserEmail:
		if req.UserID == "" {
			return nil, errors.New("userid cannot be empty in request for single detail")
		}

		if req.Update == "" { // read detail value, no update requested
			return b.getUserDetail(req)
		}

		return b.setUserDetail(req)
	case AllUserDetails:
		// list of all details returned in case request is a read request
		// (Update is not set) and does not have UserID
		if req.Update == "" && req.UserID == "" { // read list of all details
			return b.listDetails(req.Locator)
		}
		return nil, errors.New("unsupported request with userdetail all")
	default:
		return nil, errors.Errorf("unsupported detail %q", req.Detail)
	}
}

// Update for locator.URL with mutable part of comment
func (b *BoltDB) Update(comment store.Comment) error {

	getReq := GetRequest{Locator: comment.Locator, CommentID: comment.ID}
	if curComment, err := b.Get(getReq); err == nil {
		// preserve immutable fields
		comment.ParentID = curComment.ParentID
		comment.Locator = curComment.Locator
		comment.Timestamp = curComment.Timestamp
		comment.User = curComment.User
	}

	bdb, err := b.db(comment.Locator.SiteID)
	if err != nil {
		return err
	}

	return bdb.Update(func(tx *bolt.Tx) error {
		bucket, e := b.getPostBucket(tx, comment.Locator.URL)
		if e != nil {
			return e
		}
		return b.save(bucket, comment.ID, comment)
	})
}

// Count returns number of comments for post or user
func (b *BoltDB) Count(req FindRequest) (count int, err error) {

	bdb, err := b.db(req.Locator.SiteID)
	if err != nil {
		return 0, err
	}

	if req.Locator.URL != "" { // comment's count for post
		err = bdb.View(func(tx *bolt.Tx) error {
			var e error
			count, e = b.count(tx, req.Locator.URL, 0)
			return e
		})
		return count, err
	}

	if req.UserID != "" { // comment's count for user
		err = bdb.View(func(tx *bolt.Tx) error {
			usersBkt := tx.Bucket([]byte(userBucketName))
			userIDBkt := usersBkt.Bucket([]byte(req.UserID))
			if userIDBkt == nil {
				return errors.Errorf("no comments for user %s in store for %s site", req.UserID, req.Locator.SiteID)
			}
			stats := userIDBkt.Stats()
			count = stats.KeyN
			return nil
		})
		return count, err
	}

	return 0, errors.Errorf("invalid count request %+v", req)
}

// Info get post(s) meta info
func (b *BoltDB) Info(req InfoRequest) ([]store.PostInfo, error) {

	bdb, err := b.db(req.Locator.SiteID)
	if err != nil {
		return []store.PostInfo{}, err
	}

	if req.Locator.URL != "" { // post info
		info := store.PostInfo{}
		err = bdb.View(func(tx *bolt.Tx) error {
			infoBkt := tx.Bucket([]byte(infoBucketName))
			if e := b.load(infoBkt, req.Locator.URL, &info); e != nil {
				return errors.Wrapf(e, "can't load info for %s", req.Locator.URL)
			}
			return nil
		})

		// set read-only from age and manual bucket
		readOnlyAge := req.ReadOnlyAge
		info.ReadOnly = readOnlyAge > 0 && !info.FirstTS.IsZero() && info.FirstTS.AddDate(0, 0, readOnlyAge).Before(time.Now())
		if b.checkFlag(FlagRequest{Locator: req.Locator, Flag: ReadOnly}) {
			info.ReadOnly = true
		}
		return []store.PostInfo{info}, err
	}

	if req.Locator.URL == "" && req.Locator.SiteID != "" { // site info (list)
		list := []store.PostInfo{}
		err = bdb.View(func(tx *bolt.Tx) error {
			postsBkt := tx.Bucket([]byte(postsBucketName))

			c := postsBkt.Cursor()
			n := 0
			for k, _ := c.Last(); k != nil; k, _ = c.Prev() {
				n++
				if req.Skip > 0 && n <= req.Skip {
					continue
				}
				postURL := string(k)
				infoBkt := tx.Bucket([]byte(infoBucketName))
				info := store.PostInfo{}
				if e := b.load(infoBkt, postURL, &info); e != nil {
					return errors.Wrapf(e, "can't load info for %s", postURL)
				}
				list = append(list, info)
				if req.Limit > 0 && len(list) >= req.Limit {
					break
				}
			}
			return nil
		})
		return list, err
	}

	return nil, errors.Errorf("invalid info request %+v", req)
}

// ListFlags get list of flagged keys, like blocked & verified user
// works for full locator (post flags) or with userID
func (b *BoltDB) ListFlags(req FlagRequest) (res []interface{}, err error) {

	bdb, e := b.db(req.Locator.SiteID)
	if e != nil {
		return nil, e
	}

	res = []interface{}{}
	switch req.Flag {
	case Verified:
		err = bdb.View(func(tx *bolt.Tx) error {
			usersBkt := tx.Bucket([]byte(verifiedBucketName))
			_ = usersBkt.ForEach(func(k, _ []byte) error {
				res = append(res, string(k))
				return nil
			})
			return nil
		})
		return res, err
	case Blocked:
		err = bdb.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(blocksBucketName))
			return bucket.ForEach(func(k []byte, v []byte) error {
				ts, errParse := time.ParseInLocation(tsNano, string(v), time.Local)
				if errParse != nil {
					return errors.Wrap(errParse, "can't parse block ts")
				}
				if time.Now().Before(ts) {
					// get user name from comment user section
					userName := ""
					findReq := FindRequest{Locator: store.Locator{SiteID: req.Locator.SiteID}, UserID: string(k), Limit: 1}
					userComments, errUser := b.Find(findReq)
					if errUser == nil && len(userComments) > 0 {
						userName = userComments[0].User.Name
					}
					res = append(res, store.BlockedUser{ID: string(k), Name: userName, Until: ts})
				}
				return nil
			})
		})
		return res, err
	}
	return nil, errors.Errorf("flag %s not listable", req.Flag)
}

// Delete post(s), user, comment, user details, or everything
func (b *BoltDB) Delete(req DeleteRequest) error {

	bdb, e := b.db(req.Locator.SiteID)
	if e != nil {
		return e
	}

	switch {
	case req.UserDetail != "": // delete user detail
		return b.deleteUserDetail(bdb, req.UserID, req.UserDetail)
	case req.Locator.URL != "" && req.CommentID != "" && req.UserDetail == "": // delete comment
		return b.deleteComment(bdb, req.Locator, req.CommentID, req.DeleteMode)
	case req.Locator.SiteID != "" && req.UserID != "" && req.CommentID == "" && req.UserDetail == "": // delete user
		return b.deleteUser(bdb, req.Locator.SiteID, req.UserID, req.DeleteMode)
	case req.Locator.SiteID != "" && req.Locator.URL == "" && req.CommentID == "" && req.UserID == "" && req.UserDetail == "": // delete site
		return b.deleteAll(bdb, req.Locator.SiteID)
	}

	return errors.Errorf("invalid delete request %+v", req)
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

// Last returns up to max last comments for given siteID
func (b *BoltDB) lastComments(siteID string, max int, since time.Time) (comments []store.Comment, err error) {

	comments = []store.Comment{}

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

			if !since.IsZero() {
				// stop if reached "since" ts
				tsSince := []byte(since.Format(tsNano))
				if bytes.Compare(k, tsSince) <= 0 {
					break
				}
			}
			url, commentID, e := b.parseRef(v)
			if e != nil {
				return e
			}
			postBkt, e := b.getPostBucket(tx, url)
			if e != nil {
				return e
			}

			comment := store.Comment{}
			if e = b.load(postBkt, commentID, &comment); e != nil {
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

// userComments extracts all comments for given site and given userID
// "users" bucket has sub-bucket for each userID, and keeps it as ts:ref
func (b *BoltDB) userComments(siteID, userID string, limit, skip int) (comments []store.Comment, err error) {

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
		url, commentID, errParse := b.parseRef([]byte(v))
		if errParse != nil {
			return comments, errors.Wrapf(errParse, "can't parse reference %s", v)
		}
		getReq := GetRequest{Locator: store.Locator{SiteID: siteID, URL: url}, CommentID: commentID}
		if c, errRef := b.Get(getReq); errRef == nil {
			comments = append(comments, c)
		}
	}

	return comments, err
}

func (b *BoltDB) checkFlag(req FlagRequest) (val bool) {

	bdb, err := b.db(req.Locator.SiteID)
	if err != nil {
		return false
	}

	key := req.Locator.URL
	if req.UserID != "" {
		key = req.UserID
	}

	if req.Flag == Blocked {
		var blocked bool
		_ = bdb.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(blocksBucketName))
			v := bucket.Get([]byte(key))
			if v == nil {
				blocked = false
				return nil
			}

			until, e := time.Parse(tsNano, string(v))
			if e != nil {
				blocked = false
				return nil
			}
			blocked = time.Now().Before(until)
			return nil
		})
		return blocked
	}

	_ = bdb.View(func(tx *bolt.Tx) error {
		var bucket *bolt.Bucket
		if bucket, err = b.flagBucket(tx, req.Flag); err != nil {
			return err
		}
		val = bucket.Get([]byte(key)) != nil
		return nil
	})
	return val
}

func (b *BoltDB) setFlag(req FlagRequest) (res bool, err error) {
	bdb, e := b.db(req.Locator.SiteID)
	if e != nil {
		return false, e
	}

	key := req.Locator.URL
	if req.UserID != "" {
		key = req.UserID
	}

	err = bdb.Update(func(tx *bolt.Tx) error {
		var bucket *bolt.Bucket
		if bucket, err = b.flagBucket(tx, req.Flag); err != nil {
			return err
		}
		switch req.Update {
		case FlagTrue:
			if req.Flag == Blocked {
				val := time.Now().AddDate(100, 0, 0).Format(tsNano) // permanent is 100 year
				if req.TTL > 0 {
					val = time.Now().Add(req.TTL).Format(tsNano)
				}
				if e = bucket.Put([]byte(key), []byte(val)); e != nil {
					return errors.Wrapf(e, "failed to put blocked to %s", key)
				}
				res = true
				return nil
			}

			if e = bucket.Put([]byte(key), []byte(time.Now().Format(tsNano))); e != nil {
				return errors.Wrapf(e, "failed to set flag %s for %s", req.Flag, req.Locator.URL)
			}
			res = true
			return nil
		case FlagFalse:
			if e = bucket.Delete([]byte(key)); e != nil {
				return errors.Wrapf(e, "failed to clean flag %s for %s", req.Flag, req.Locator.URL)
			}
			res = false
		}
		return nil
	})

	return res, err
}

func (b *BoltDB) flagBucket(tx *bolt.Tx, flag Flag) (bkt *bolt.Bucket, err error) {
	switch flag {
	case ReadOnly:
		bkt = tx.Bucket([]byte(readonlyBucketName))
	case Blocked:
		bkt = tx.Bucket([]byte(blocksBucketName))
	case Verified:
		bkt = tx.Bucket([]byte(verifiedBucketName))
	default:
		return nil, errors.Errorf("unsupported flag %v", flag)
	}
	return bkt, nil
}

// getUserDetail returns UserDetailEntry with requested userDetail (omitting other details)
// as an only element of the slice.
func (b *BoltDB) getUserDetail(req UserDetailRequest) (result []UserDetailEntry, err error) {
	bdb, e := b.db(req.Locator.SiteID)
	if e != nil {
		return result, e
	}

	err = bdb.View(func(tx *bolt.Tx) error {
		var entry UserDetailEntry
		bucket := tx.Bucket([]byte(userDetailsBucketName))
		value := bucket.Get([]byte(req.UserID))
		// return no error in case of absent entry
		if value != nil {
			if err = json.Unmarshal(value, &entry); err != nil {
				return errors.Wrap(e, "failed to unmarshal entry")
			}
			switch req.Detail {
			case UserEmail:
				result = []UserDetailEntry{{UserID: req.UserID, Email: entry.Email}}
			}
		}
		return nil
	})

	return result, err
}

// setUserDetail sets requested userDetail, returning complete updated UserDetailEntry as an onlyIps
// element of the slice in case of success
func (b *BoltDB) setUserDetail(req UserDetailRequest) (result []UserDetailEntry, err error) {
	bdb, e := b.db(req.Locator.SiteID)
	if e != nil {
		return result, e
	}

	var entry UserDetailEntry
	err = bdb.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(userDetailsBucketName))
		value := bucket.Get([]byte(req.UserID))
		// return no error in case of absent entry
		if value != nil {
			if err = json.Unmarshal(value, &entry); err != nil {
				return errors.Wrap(e, "failed to unmarshal entry")
			}
		}
		return nil
	})
	if err != nil {
		return result, err
	}

	if entry.UserID == "" {
		// new entry to be created, need to set UserID for it
		entry.UserID = req.UserID
	}

	switch req.Detail {
	case UserEmail:
		entry.Email = req.Update
	}

	err = bdb.Update(func(tx *bolt.Tx) error {
		err = b.save(tx.Bucket([]byte(userDetailsBucketName)), req.UserID, entry)
		return errors.Wrapf(err, "failed to update detail %s for %s in %s", req.Detail, req.UserID, req.Locator.SiteID)
	})

	return []UserDetailEntry{entry}, err
}

// listDetails lists all available users details for given site
func (b *BoltDB) listDetails(loc store.Locator) (result []UserDetailEntry, err error) {
	bdb, e := b.db(loc.SiteID)
	if e != nil {
		return result, e
	}

	err = bdb.View(func(tx *bolt.Tx) error {
		var entry UserDetailEntry
		bucket := tx.Bucket([]byte(userDetailsBucketName))
		return bucket.ForEach(func(userID, value []byte) error {
			if err = json.Unmarshal(value, &entry); err != nil {
				return errors.Wrap(e, "failed to unmarshal entry")
			}
			result = append(result, entry)
			return nil
		})
	})
	return result, err
}

// deleteUserDetail deletes requested UserDetail or whole UserDetailEntry
func (b *BoltDB) deleteUserDetail(bdb *bolt.DB, userID string, userDetail UserDetail) error {
	var entry UserDetailEntry
	err := bdb.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(userDetailsBucketName))
		value := bucket.Get([]byte(userID))
		// return no error in case of absent entry
		if value != nil {
			if err := json.Unmarshal(value, &entry); err != nil {
				return errors.Wrap(err, "failed to unmarshal entry")
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	if entry == (UserDetailEntry{}) {
		// absent entry means that we should not do anything
		return nil
	}

	switch userDetail {
	case UserEmail:
		entry.Email = ""
	case AllUserDetails:
		entry = UserDetailEntry{UserID: userID}
	}

	if entry == (UserDetailEntry{UserID: userID}) {
		// if entry doesn't have non-empty details, we should delete it
		return bdb.Update(func(tx *bolt.Tx) error {
			err := tx.Bucket([]byte(userDetailsBucketName)).Delete([]byte(userID))
			return errors.Wrapf(err, "failed to delete user detail %s for %s", userDetail, userID)
		})
	}

	return bdb.Update(func(tx *bolt.Tx) error {
		// updated entry is not empty and we need to store it's updated copy
		err := b.save(tx.Bucket([]byte(userDetailsBucketName)), userID, entry)
		return errors.Wrapf(err, "failed to update detail %s for %s", userDetail, userID)
	})
}

func (b *BoltDB) deleteComment(bdb *bolt.DB, locator store.Locator, commentID string, mode store.DeleteMode) error {

	return bdb.Update(func(tx *bolt.Tx) error {

		postBkt, e := b.getPostBucket(tx, locator.URL)
		if e != nil {
			return e
		}

		comment := store.Comment{}
		if e = b.load(postBkt, commentID, &comment); e != nil {
			return errors.Wrapf(e, "can't load key %s from bucket %s", commentID, locator.URL)
		}

		if !comment.Deleted {
			// decrement comments count for post url
			if _, e = b.count(tx, comment.Locator.URL, -1); e != nil {
				return errors.Wrapf(e, "failed to decrement count for %s", comment.Locator)
			}
		}

		// set deleted status and clear fields
		comment.SetDeleted(mode)

		if e = b.save(postBkt, commentID, comment); e != nil {
			return errors.Wrapf(e, "can't save deleted comment for key %s from bucket %s", commentID, locator.URL)
		}

		// delete from "last" bucket
		lastBkt := tx.Bucket([]byte(lastBucketName))
		if e = lastBkt.Delete([]byte(commentID)); e != nil {
			return errors.Wrapf(e, "can't delete key %s from bucket %s", commentID, lastBucketName)
		}

		return nil
	})
}

// deleteAll removes all top-level buckets for given siteID
func (b *BoltDB) deleteAll(bdb *bolt.DB, siteID string) error {

	// delete all buckets except blocked users
	toDelete := []string{postsBucketName, lastBucketName, userBucketName, userDetailsBucketName, infoBucketName}

	// delete top-level buckets
	err := bdb.Update(func(tx *bolt.Tx) error {
		for _, bktName := range toDelete {

			if e := tx.DeleteBucket([]byte(bktName)); e != nil {
				return errors.Wrapf(e, "failed to delete top level bucket %s", bktName)
			}
			if _, e := tx.CreateBucketIfNotExists([]byte(bktName)); e != nil {
				return errors.Wrapf(e, "failed to create top level bucket %s", bktName)
			}
		}
		return nil
	})

	return errors.Wrapf(err, "failed to delete top level buckets from site %s", siteID)
}

// deleteUser removes all comments and details for given user. Everything will be market as deleted
// and user name and userID will be changed to "deleted". Also removes from last and from user buckets.
func (b *BoltDB) deleteUser(bdb *bolt.DB, siteID, userID string, mode store.DeleteMode) error {

	// get list of all comments outside of transaction loop
	posts, err := b.Info(InfoRequest{Locator: store.Locator{SiteID: siteID}})
	if err != nil {
		return err
	}

	type commentInfo struct {
		locator   store.Locator
		commentID string
	}

	// get list of commentID for all user's comment
	comments := []commentInfo{}
	for _, postInfo := range posts {
		postInfo := postInfo
		err = bdb.View(func(tx *bolt.Tx) error {
			postsBkt := tx.Bucket([]byte(postsBucketName))
			postBkt := postsBkt.Bucket([]byte(postInfo.URL))
			err = postBkt.ForEach(func(postURL []byte, commentVal []byte) error {
				comment := store.Comment{}
				if err = json.Unmarshal(commentVal, &comment); err != nil {
					return errors.Wrap(err, "failed to unmarshal")
				}
				if comment.User.ID == userID {
					comments = append(comments, commentInfo{locator: comment.Locator, commentID: comment.ID})
				}
				return nil
			})
			return errors.Wrapf(err, "failed to collect list of comments for deletion from %s", postInfo.URL)
		})
		if err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] comments for removal=%d", len(comments))

	// delete collected comments
	for _, ci := range comments {
		if e := b.deleteComment(bdb, ci.locator, ci.commentID, mode); e != nil {
			return errors.Wrapf(err, "failed to delete comment %+v", ci)
		}
	}

	// delete user bucket in hard mode
	if mode == store.HardDelete {
		err = bdb.Update(func(tx *bolt.Tx) error {
			usersBkt := tx.Bucket([]byte(userBucketName))
			if usersBkt != nil {
				if e := usersBkt.DeleteBucket([]byte(userID)); e != nil {
					return errors.Wrapf(err, "failed to delete user bucket for %s", userID)
				}
			}
			return nil
		})

		if err != nil {
			return errors.Wrap(err, "can't delete user meta")
		}
	}

	if len(comments) == 0 {
		return errors.Errorf("unknown user %s", userID)
	}

	return b.deleteUserDetail(bdb, userID, AllUserDetails)
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
func (b *BoltDB) save(bkt *bolt.Bucket, key string, value interface{}) (err error) {
	if value == nil {
		return errors.Errorf("can't save nil value for %s", key)
	}
	jdata, jerr := json.Marshal(value)
	if jerr != nil {
		return errors.Wrap(jerr, "can't marshal comment")
	}
	if err = bkt.Put([]byte(key), jdata); err != nil {
		return errors.Wrapf(err, "failed to save key %s", key)
	}
	return nil
}

// load and unmarshal json value by key from bucket. Should run in view tx
func (b *BoltDB) load(bkt *bolt.Bucket, key string, res interface{}) error {
	value := bkt.Get([]byte(key))
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
	if err := b.load(infoBkt, postURL, &info); err != nil {
		info = store.PostInfo{}
	}
	if val == 0 { // get current count, don't update
		return info.Count, nil
	}
	info.Count += val

	return info.Count, b.save(infoBkt, postURL, &info)
}

func (b *BoltDB) setInfo(tx *bolt.Tx, comment store.Comment) (store.PostInfo, error) {
	infoBkt := tx.Bucket([]byte(infoBucketName))
	info := store.PostInfo{}
	if err := b.load(infoBkt, comment.Locator.URL, &info); err != nil {
		info = store.PostInfo{
			Count:   0,
			URL:     comment.Locator.URL,
			FirstTS: comment.Timestamp,
			LastTS:  comment.Timestamp,
		}
	}
	info.Count++
	info.LastTS = comment.Timestamp
	err := b.save(infoBkt, comment.Locator.URL, &info)
	return info, err
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
func (b *BoltDB) parseRef(val []byte) (url, id string, err error) {
	elems := strings.Split(string(val), "!!")
	if len(elems) != 2 {
		return "", "", errors.Errorf("invalid reference value %s", string(val))
	}
	return elems[0], elems[1], nil
}
