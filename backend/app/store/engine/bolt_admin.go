package engine

import (
	"encoding/json"
	"time"

	bolt "github.com/coreos/bbolt"
	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/store"
)

// Delete removes comment, by locator from the store.
// Posts collection only sets status to deleted and clear fields in order to prevent breaking trees of replies.
// From last bucket removed for real.
func (b *BoltDB) Delete(locator store.Locator, commentID string, mode store.DeleteMode) error {

	bdb, err := b.db(locator.SiteID)
	if err != nil {
		return err
	}

	return bdb.Update(func(tx *bolt.Tx) error {

		postBkt, e := b.getPostBucket(tx, locator.URL)
		if e != nil {
			return e
		}

		comment := store.Comment{}
		if err = b.load(postBkt, commentID, &comment); err != nil {
			return errors.Wrapf(err, "can't load key %s from bucket %s", commentID, locator.URL)
		}
		// set deleted status and clear fields
		comment.SetDeleted(mode)

		if err = b.save(postBkt, commentID, comment); err != nil {
			return errors.Wrapf(err, "can't save deleted comment for key %s from bucket %s", commentID, locator.URL)
		}

		// delete from "last" bucket
		lastBkt := tx.Bucket([]byte(lastBucketName))
		if err = lastBkt.Delete([]byte(commentID)); err != nil {
			return errors.Wrapf(err, "can't delete key %s from bucket %s", commentID, lastBucketName)
		}

		// decrement comments count for post url
		if _, e = b.count(tx, comment.Locator.URL, -1); e != nil {
			return errors.Wrapf(e, "failed to decrement count for %s", comment.Locator)
		}

		return nil
	})
}

// DeleteAll removes all top-level buckets for given siteID
func (b *BoltDB) DeleteAll(siteID string) error {

	bdb, err := b.db(siteID)
	if err != nil {
		return err
	}

	// delete all buckets except blocked users
	toDelete := []string{postsBucketName, lastBucketName, userBucketName, infoBucketName}

	// delete top-level buckets
	err = bdb.Update(func(tx *bolt.Tx) error {
		for _, bktName := range toDelete {

			if e := tx.DeleteBucket([]byte(bktName)); e != nil {
				return errors.Wrapf(err, "failed to delete top level bucket %s", bktName)
			}
			if _, e := tx.CreateBucketIfNotExists([]byte(bktName)); e != nil {
				return errors.Wrapf(err, "failed to create top level bucket %s", bktName)
			}
		}
		return nil
	})

	return errors.Wrapf(err, "failed to delete top level buckets from site %s", siteID)
}

// DeleteUser removes all comments for given user. Everything will be market as deleted
// and user name and userID will be changed to "deleted". Also removes from last and from user buckets.
func (b *BoltDB) DeleteUser(siteID string, userID string) error {
	bdb, err := b.db(siteID)
	if err != nil {
		return err
	}

	// get list of all comments outside of transaction loop
	posts, err := b.List(siteID, 0, 0)
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
		if e := b.Delete(ci.locator, ci.commentID, store.HardDelete); e != nil {
			return errors.Wrapf(err, "failed to delete comment %+v", ci)
		}
	}

	//  delete  user bucket
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

	if len(comments) == 0 {
		return errors.Errorf("unknown user %s", userID)
	}

	return err
}

// SetBlock blocks/unblocks user for given site. ttl defines for for how long, 0 - permanent
// block uses blocksBucketName with key=userID and val=TTL+now
func (b *BoltDB) SetBlock(siteID string, userID string, status bool, ttl time.Duration) error {

	bdb, err := b.db(siteID)
	if err != nil {
		return err
	}

	return bdb.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucketName))
		switch status {
		case true:
			val := time.Now().AddDate(100, 0, 0).Format(tsNano) // permanent is 100 year
			if ttl > 0 {
				val = time.Now().Add(ttl).Format(tsNano)
			}
			if e := bucket.Put([]byte(userID), []byte(val)); e != nil {
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
		val := bucket.Get([]byte(userID))
		if val == nil {
			blocked = false
			return nil
		}

		until, e := time.Parse(tsNano, string(val))
		if e != nil {
			blocked = false
			return nil
		}
		blocked = time.Now().Before(until)
		return nil
	})
	return blocked
}

// Blocked get lists of blocked users for given site
// bucket uses userID:
func (b *BoltDB) Blocked(siteID string) (users []store.BlockedUser, err error) {
	users = []store.BlockedUser{}
	bdb, err := b.db(siteID)
	if err != nil {
		return nil, err
	}

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
				userComments, errUser := b.User(siteID, string(k), 1, 0)
				if errUser == nil && len(userComments) > 0 {
					userName = userComments[0].User.Name
				}
				users = append(users, store.BlockedUser{ID: string(k), Name: userName, Until: ts})
			}
			return nil
		})
	})

	return users, err
}

// SetReadOnly makes post read-only or reset the ro flag
func (b *BoltDB) SetReadOnly(locator store.Locator, status bool) error {
	bdb, err := b.db(locator.SiteID)
	if err != nil {
		return err
	}

	return bdb.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(readonlyBucketName))
		switch status {
		case true:
			if e := bucket.Put([]byte(locator.URL), []byte(time.Now().Format(tsNano))); e != nil {
				return errors.Wrapf(e, "failed to set ro for %s", locator.URL)
			}
		case false:
			if e := bucket.Delete([]byte(locator.URL)); e != nil {
				return errors.Wrapf(e, "failed to clean ro for %s", locator.URL)
			}
		}
		return nil
	})
}

// IsReadOnly checks if post in RO mode
func (b *BoltDB) IsReadOnly(locator store.Locator) (ro bool) {

	bdb, err := b.db(locator.SiteID)
	if err != nil {
		return false
	}

	_ = bdb.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(readonlyBucketName))
		ro = bucket.Get([]byte(locator.URL)) != nil
		return nil
	})
	return ro
}

// SetVerified makes user verified or reset the flag
func (b *BoltDB) SetVerified(siteID string, userID string, status bool) error {
	bdb, err := b.db(siteID)
	if err != nil {
		return err
	}

	return bdb.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(verifiedBucketName))
		switch status {
		case true:
			if e := bucket.Put([]byte(userID), []byte(time.Now().Format(tsNano))); e != nil {
				return errors.Wrapf(e, "failed to set verified status for %s", userID)
			}
		case false:
			if e := bucket.Delete([]byte(userID)); e != nil {
				return errors.Wrapf(e, "failed to clean verified status for %s", userID)
			}
		}
		return nil
	})
}

// IsVerified checks if user verified
func (b *BoltDB) IsVerified(siteID string, userID string) (verified bool) {

	bdb, err := b.db(siteID)
	if err != nil {
		return false
	}

	_ = bdb.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(verifiedBucketName))
		verified = bucket.Get([]byte(userID)) != nil
		return nil
	})
	return verified
}

// Verified returns list of verified userIDs
func (b *BoltDB) Verified(siteID string) (ids []string, err error) {
	bdb, err := b.db(siteID)
	if err != nil {
		return nil, err
	}
	err = bdb.View(func(tx *bolt.Tx) error {
		usersBkt := tx.Bucket([]byte(verifiedBucketName))
		_ = usersBkt.ForEach(func(k, _ []byte) error {
			ids = append(ids, string(k))
			return nil
		})
		return nil
	})
	return ids, err
}
