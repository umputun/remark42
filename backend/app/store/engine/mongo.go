package engine

import (
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/go-pkgz/mongo"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/store"
)

// Mongo implements engine interface
type Mongo struct {
	conn       *mongo.Connection
	postWriter mongo.BufferedWriter
}

const (
	mongoPosts     = "posts"
	mongoMetaPosts = "meta_posts"
	mongoMetaUsers = "meta_users"
)

type metaPost struct {
	ID       string `bson:"_id"` // url
	SiteID   string `bson:"site"`
	ReadOnly bool   `bson:"read_only"`
}

type metaUser struct {
	ID           string    `bson:"_id"` // user_id
	SiteID       string    `bson:"site"`
	Verified     bool      `bson:"verified"`
	Blocked      bool      `bson:"blocked"`
	BlockedUntil time.Time `bson:"blocked_until"`
}

// NewMongo makes mongo engine. bufferSize denies how many records will be buffered, 0 turns buffering off.
// flushDuration triggers automatic flus (write from buffer), 0 disables it and will flush as buffer size reached.
// important! don't use flushDuration=0 for production use as it can leave records in-fly state for long or even unlimited time.
func NewMongo(conn *mongo.Connection, bufferSize int, flushDuration time.Duration) (*Mongo, error) {
	writer := mongo.NewBufferedWriter(bufferSize, conn).WithCollection(mongoPosts).WithAutoFlush(flushDuration)
	result := Mongo{conn: conn, postWriter: writer}
	err := result.prepare()
	return &result, errors.Wrap(err, "failed to prepare mongo")
}

// Create new comment, write can be buffered and delayed.
func (m *Mongo) Create(comment store.Comment) (commentID string, err error) {
	err = m.conn.WithCustomCollection(mongoPosts, func(coll *mgo.Collection) error {
		return coll.Insert(&comment)
	})
	return comment.ID, err
}

// Find returns all comments for post and sorts results
func (m *Mongo) Find(locator store.Locator, sortFld string) (comments []store.Comment, err error) {
	comments = []store.Comment{}
	err = m.conn.WithCustomCollection(mongoPosts, func(coll *mgo.Collection) error {
		query := bson.M{"locator.site": locator.SiteID, "locator.url": locator.URL}
		return coll.Find(query).Sort(sortFld).All(&comments)
	})
	return comments, err
}

// Get returns comment for locator.URL and commentID string
func (m *Mongo) Get(locator store.Locator, commentID string) (comment store.Comment, err error) {
	err = m.conn.WithCustomCollection(mongoPosts, func(coll *mgo.Collection) error {
		query := bson.M{"_id": commentID, "locator.site": locator.SiteID, "locator.url": locator.URL}
		return coll.Find(query).One(&comment)
	})
	return comment, err
}

// Put updates comment for locator.URL with mutable part of comment
func (m *Mongo) Put(locator store.Locator, comment store.Comment) error {
	return m.conn.WithCustomCollection(mongoPosts, func(coll *mgo.Collection) error {
		return coll.Update(bson.M{"_id": comment.ID, "locator.site": locator.SiteID, "locator.url": locator.URL},
			bson.M{"$set": bson.M{
				"text":    comment.Text,
				"orig":    comment.Orig,
				"score":   comment.Score,
				"votes":   comment.Votes,
				"pin":     comment.Pin,
				"deleted": comment.Deleted,
			}})
	})
}

// Last returns up to max last comments for given siteID
func (m *Mongo) Last(siteID string, max int) (comments []store.Comment, err error) {
	comments = []store.Comment{}
	if max > lastLimit || max == 0 {
		max = lastLimit
	}
	err = m.conn.WithCustomCollection(mongoPosts, func(coll *mgo.Collection) error {
		query := bson.M{"locator.site": siteID, "delete": false}
		return coll.Find(query).Sort("-time").Limit(max).All(&comments)
	})
	return comments, err
}

// Count returns number of comments for locator
func (m *Mongo) Count(locator store.Locator) (count int, err error) {

	e := m.conn.WithCustomCollection(mongoPosts, func(coll *mgo.Collection) error {
		query := bson.M{"locator.site": locator.SiteID, "locator.url": locator.URL, "delete": false}
		count, err = coll.Find(query).Count()
		return err
	})
	return count, e
}

// List returns list of all commented posts with counters
func (m *Mongo) List(siteID string, limit, skip int) (list []store.PostInfo, err error) {
	list = []store.PostInfo{}

	if limit <= 0 {
		limit = 1000
	}
	if skip < 0 {
		skip = 0
	}

	err = m.conn.WithCustomCollection(mongoPosts, func(coll *mgo.Collection) error {
		pipeline := coll.Pipe([]bson.M{
			{"$match": bson.M{"locator.site": siteID}},
			{"$project": bson.M{"locator.site": 1, "locator.url": 1, "time": 1}},
			{"$group": bson.M{"_id": "$locator.url", "url": bson.M{"$first": "$locator.url"}, "count": bson.M{"$sum": 1},
				"first_time": bson.M{"$min": "$time"}, "last_time": bson.M{"$max": "$time"}}},
			{"$skip": skip},
			{"$limit": limit},
		})
		return errors.Wrap(pipeline.AllowDiskUse().All(&list), "list pipeline failed")
	})
	return list, errors.Wrap(err, "can't get list")
}

// Info returns time range and count for locator
func (m *Mongo) Info(locator store.Locator, readOnlyAge int) (info store.PostInfo, err error) {
	list := []store.PostInfo{}
	err = m.conn.WithCustomCollection(mongoPosts, func(coll *mgo.Collection) error {
		pipeline := coll.Pipe([]bson.M{
			{"$match": bson.M{"locator.site": locator.SiteID, "locator.url": locator.URL}},
			{"$project": bson.M{"locator.site": 1, "locator.url": 1, "time": 1}},
			{"$group": bson.M{"_id": "$locator.url", "url": bson.M{"$first": "$locator.url"}, "count": bson.M{"$sum": 1},
				"first_time": bson.M{"$min": "$time"}, "last_time": bson.M{"$max": "$time"}}},
		})
		return errors.Wrap(pipeline.AllowDiskUse().All(&list), "list pipeline failed")
	})
	if err != nil {
		return info, err
	}
	if len(list) == 0 {
		return info, errors.Errorf("can't load info for %s", locator.URL)
	}
	info = list[0]
	// set read-only from age and manual bucket
	info.ReadOnly = readOnlyAge > 0 && !info.FirstTS.IsZero() && info.FirstTS.AddDate(0, 0, readOnlyAge).Before(time.Now())
	if m.IsReadOnly(locator) {
		info.ReadOnly = true
	}
	return info, nil
}

// User extracts all comments for given site and given userID
func (m *Mongo) User(siteID, userID string, limit, skip int) (comments []store.Comment, err error) {
	comments = []store.Comment{}
	err = m.conn.WithCustomCollection(mongoPosts, func(coll *mgo.Collection) error {
		query := bson.M{"locator.site": siteID, "user.id": userID}
		return m.setLimitAndSkip(coll.Find(query).Sort("-time"), limit, skip).All(&comments)
	})
	return comments, errors.Wrapf(err, "can't get comments for user %s", userID)
}

// UserCount returns number of comments for user
func (m *Mongo) UserCount(siteID, userID string) (count int, err error) {
	err = m.conn.WithCustomCollection(mongoPosts, func(coll *mgo.Collection) error {
		var e error
		count, e = coll.Find(bson.M{"locator.site": siteID, "user.id": userID}).Count()
		return e
	})
	return count, errors.Wrapf(err, "can't get comments count for user %s", userID)
}

// SetReadOnly makes post read-only or reset the ro flag
func (m *Mongo) SetReadOnly(locator store.Locator, status bool) (err error) {
	return m.conn.WithCustomCollection(mongoMetaPosts, func(coll *mgo.Collection) error {
		_, e := coll.Upsert(bson.M{"_id": locator.URL, "site": locator.SiteID}, bson.M{"$set": bson.M{"read_only": status}})
		return e
	})
}

// IsReadOnly checks if post in RO
func (m *Mongo) IsReadOnly(locator store.Locator) (ro bool) {
	meta := metaPost{}
	err := m.conn.WithCustomCollection(mongoMetaPosts, func(coll *mgo.Collection) error {
		return coll.Find(bson.M{"_id": locator.URL, "site": locator.SiteID}).One(&meta)
	})
	return err == nil && meta.ReadOnly
}

// SetVerified makes user verified or reset the flag
func (m *Mongo) SetVerified(siteID string, userID string, status bool) error {
	return m.conn.WithCustomCollection(mongoMetaUsers, func(coll *mgo.Collection) error {
		_, e := coll.Upsert(bson.M{"_id": userID, "site": siteID}, bson.M{"$set": bson.M{"verified": status}})
		return e
	})
}

// IsVerified checks if user verified
func (m *Mongo) IsVerified(siteID string, userID string) (verified bool) {
	meta := metaUser{}
	err := m.conn.WithCustomCollection(mongoMetaUsers, func(coll *mgo.Collection) error {
		return coll.Find(bson.M{"_id": userID, "site": siteID}).One(&meta)
	})
	return err == nil && meta.Verified
}

// SetBlock blocks/unblocks user for given site. ttl defines for for how long, 0 - permanent
// block uses blocksBucketName with key=userID and val=TTL+now
func (m *Mongo) SetBlock(siteID string, userID string, status bool, ttl time.Duration) error {

	until := time.Time{}
	if status {
		until = time.Now().AddDate(100, 0, 0) // permanent is 50year
		if ttl > 0 {
			until = time.Now().Add(ttl)
		}
	}
	return m.conn.WithCustomCollection(mongoMetaUsers, func(coll *mgo.Collection) error {
		_, e := coll.Upsert(bson.M{"_id": userID, "site": siteID},
			bson.M{"$set": bson.M{"blocked": status, "blocked_until": until}})
		return errors.Wrapf(e, "failed to set block for %s", userID)
	})
}

// IsBlocked checks if user blocked
func (m *Mongo) IsBlocked(siteID string, userID string) (blocked bool) {
	meta := metaUser{}
	err := m.conn.WithCustomCollection(mongoMetaUsers, func(coll *mgo.Collection) error {
		return coll.Find(bson.M{"_id": userID, "site": siteID}).One(&meta)
	})
	return err == nil && meta.Blocked && meta.BlockedUntil.After(time.Now())
}

// Blocked get lists of blocked users for given site
func (m *Mongo) Blocked(siteID string) (users []store.BlockedUser, err error) {
	users = []store.BlockedUser{}
	metas := []metaUser{}
	err = m.conn.WithCustomCollection(mongoMetaUsers, func(coll *mgo.Collection) error {
		return coll.Find(bson.M{"site": siteID,
			"blocked": true, "blocked_until": bson.M{"$gt": time.Now()}}).All(&metas)
	})
	if err != nil {
		return users, errors.Wrapf(err, "can't get blocked users for site for %s", siteID)
	}

	for _, mu := range metas {
		blockedUser := store.BlockedUser{ID: mu.ID, Until: mu.BlockedUntil}
		if ucc, e := m.User(siteID, mu.ID, 1, 0); e == nil && len(ucc) > 0 {
			blockedUser.Name = ucc[0].User.Name
		}
		users = append(users, blockedUser)
	}
	return users, nil
}

// Delete removes comment, by locator from the store.
// Posts collection only sets status to deleted and clear fields in order to prevent breaking trees of replies.
func (m *Mongo) Delete(locator store.Locator, commentID string, mode store.DeleteMode) error {
	comment := store.Comment{}
	err := m.conn.WithCustomCollection(mongoPosts, func(coll *mgo.Collection) error {
		e := coll.Find(bson.M{"locator.site": locator.SiteID, "locator.url": locator.URL, "_id": commentID}).One(&comment)
		if e != nil {
			return e
		}
		comment.SetDeleted(mode)
		return coll.Update(bson.M{"locator.site": locator.SiteID, "locator.url": locator.URL, "_id": commentID}, comment)
	})
	return errors.Wrapf(err, "can't delete %s", commentID)
}

// DeleteAll removes all info about siteID
func (m *Mongo) DeleteAll(siteID string) error {
	err := m.conn.WithCustomCollection(mongoPosts, func(coll *mgo.Collection) error {
		_, e := coll.RemoveAll(bson.M{"locator.site": siteID})
		return e
	})
	return errors.Wrapf(err, "can't delete site %s", siteID)
}

// DeleteUser removes all comments for given user. Everything will be market as deleted
// and user name and userID will be changed to "deleted".
func (m *Mongo) DeleteUser(siteID string, userID string) error {
	comments := []store.Comment{}
	return m.conn.WithCustomCollection(mongoPosts, func(coll *mgo.Collection) error {
		e := coll.Find(bson.M{"locator.site": siteID, "user.id": userID}).All(&comments)
		if e != nil {
			return e
		}
		for _, c := range comments {
			if e = m.Delete(c.Locator, c.ID, store.HardDelete); e != nil {
				return e
			}
		}
		return nil
	})
}

// Close boltdb store
func (m *Mongo) Close() error {
	if m.postWriter != nil {
		return m.postWriter.Close()
	}
	return nil
}

// prepare collections with all indexes
func (m *Mongo) prepare() error {
	errs := new(multierror.Error)
	e := m.conn.WithCustomCollection(mongoPosts, func(coll *mgo.Collection) error {
		errs = multierror.Append(errs, coll.EnsureIndexKey("user.id", "locator.site", "time"))
		errs = multierror.Append(errs, coll.EnsureIndexKey("locator.url", "locator.site", "time"))
		errs = multierror.Append(errs, coll.EnsureIndexKey("locator.site", "time"))
		errs = multierror.Append(errs, coll.EnsureIndexKey("locator.url", "locator.site", "score"))
		return errors.Wrapf(errs.ErrorOrNil(), "can't create index for %s", mongoPosts)
	})
	if e != nil {
		return e
	}

	e = m.conn.WithCustomCollection(mongoMetaPosts, func(coll *mgo.Collection) error {
		errs = multierror.Append(errs, coll.EnsureIndexKey("_id", "site"))
		errs = multierror.Append(errs, coll.EnsureIndexKey("site", "read_only"))
		return errors.Wrapf(errs.ErrorOrNil(), "can't create index for %s", mongoMetaPosts)
	})
	if e != nil {
		return e
	}

	return m.conn.WithCustomCollection(mongoMetaUsers, func(coll *mgo.Collection) error {
		errs = multierror.Append(errs, coll.EnsureIndexKey("_id", "site"))
		errs = multierror.Append(errs, coll.EnsureIndexKey("site", "blocked"))
		errs = multierror.Append(errs, coll.EnsureIndexKey("site", "verified"))
		return errors.Wrapf(errs.ErrorOrNil(), "can't create index for %s", mongoMetaUsers)
	})
}

func (m *Mongo) setLimitAndSkip(q *mgo.Query, limit, skip int) *mgo.Query {
	if limit <= 0 {
		limit = 1000
	}
	if skip < 0 {
		skip = 0
	}
	return q.Skip(skip).Limit(limit)
}
