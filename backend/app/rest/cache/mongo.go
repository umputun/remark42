package cache

import (
	"log"
	"time"

	"github.com/go-pkgz/repeater"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/go-pkgz/mongo"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

type mongoCache struct {
	connection   *mongo.Connection
	postFlushFn  func()
	maxKeys      int
	maxValueSize int
	maxCacheSize int64
}

const cacheCollection = "cache"

type mongoDoc struct {
	SiteID string   `bson:"site"`
	Key    string   `bson:"key"`
	Scopes []string `bson:"scopes,omitempty"`
	Data   []byte   `bson:"data"`
}

// NewMongoCache makes mongoCache implementation
func NewMongoCache(connection *mongo.Connection, options ...Option) (LoadingCache, error) {
	res := &mongoCache{
		connection:   connection,
		postFlushFn:  func() {},
		maxKeys:      1000,
		maxValueSize: 0,
	}
	for _, opt := range options {
		if err := opt(res); err != nil {
			return nil, errors.Wrap(err, "failed to set cache option")
		}
	}
	if err := res.prepare(); err != nil {
		return nil, err
	}
	return res, nil
}

// Get is loading cache method to get value by key or load via fn if not found
func (m *mongoCache) Get(key Key, fn func() ([]byte, error)) (data []byte, err error) {

	d := mongoDoc{}

	// repeat find from cache with small delay to avoid mgo random error
	rep := repeater.NewDefault(5, 10*time.Millisecond)
	mgErr := rep.Do(func() error {
		return m.connection.WithCustomCollection(cacheCollection, func(coll *mgo.Collection) error {
			return coll.Find(bson.M{"site": key.siteID, "key": key.id}).One(&d)
		})
	})
	if mgErr == nil { // cached result found
		return d.Data, nil
	}

	if data, err = fn(); err != nil {
		return data, err
	}

	if mgErr != mgo.ErrNotFound { // some other error in mgo query, don't try to update cache
		log.Printf("[WARN] unexpected mgo error %+v", mgErr)
		return data, err
	}

	if !m.allowed(data) {
		return data, nil
	}

	d = mongoDoc{
		SiteID: key.siteID,
		Key:    key.id,
		Data:   data,
		Scopes: key.scopes,
	}
	err = m.connection.WithCustomCollection(cacheCollection, func(coll *mgo.Collection) error {
		_, e := coll.Upsert(bson.M{"site": key.siteID, "key": key.id}, bson.M{"$set": d})
		return e
	})
	if err != nil {
		return nil, errors.Wrapf(err, "can't set cached value for %+v", key)
	}

	if m.maxKeys > 0 {
		err = m.cleanup(key.siteID)
	}

	return data, errors.Wrap(err, "failed to cleanup cached records")
}

func (m *mongoCache) cleanup(siteID string) (err error) {
	ids := []struct {
		ID bson.ObjectId `bson:"_id"`
	}{}

	err = m.connection.WithCustomCollection(cacheCollection, func(coll *mgo.Collection) error {
		n, countErr := coll.Find(bson.M{"site": siteID}).Count()
		if countErr != nil {
			return countErr
		}
		if countErr == nil && n > m.maxKeys {
			if findErr := coll.Find(bson.M{"site": siteID}).Sort("+id").Limit(n - m.maxKeys).All(&ids); findErr == nil {
				bsonIDs := []bson.ObjectId{}
				for _, id := range ids {
					bsonIDs = append(bsonIDs, id.ID)
				}
				_, removalErr := coll.RemoveAll(bson.M{"_id": bson.M{"$in": bsonIDs}})
				return removalErr
			}
		}
		return nil
	})
	return err
}

// Flush clears cache and calls postFlushFn async
func (m *mongoCache) Flush(req FlusherRequest) {
	err := m.connection.WithCustomCollection(cacheCollection, func(coll *mgo.Collection) error {
		q := bson.M{"site": req.siteID}
		if len(req.scopes) > 0 {
			q["scopes"] = bson.M{"$in": req.scopes}
		}
		_, e := coll.RemoveAll(q)
		return e
	})

	if err == nil && m.postFlushFn != nil {
		m.postFlushFn()
	}
}

// prepare collections with all indexes
func (m *mongoCache) prepare() error {
	errs := new(multierror.Error)
	return m.connection.WithCustomCollection(cacheCollection, func(coll *mgo.Collection) error {
		errs = multierror.Append(errs, coll.EnsureIndexKey("site", "key"))
		errs = multierror.Append(errs, coll.EnsureIndexKey("site", "scopes"))
		return errors.Wrapf(errs.ErrorOrNil(), "can't create index for %s", cacheCollection)
	})
}

func (m *mongoCache) allowed(data []byte) bool {
	if m.maxValueSize > 0 && len(data) >= m.maxValueSize {
		return false
	}
	return true
}

func (m *mongoCache) setMaxValSize(max int) error {
	m.maxValueSize = max
	if max <= 0 {
		return errors.Errorf("negative size for MaxValSize, %d", max)
	}
	return nil
}

func (m *mongoCache) setMaxKeys(max int) error {
	m.maxKeys = max
	if max <= 0 {
		return errors.Errorf("negative size for MaxKeys, %d", max)
	}
	return nil
}

func (m *mongoCache) setMaxCacheSize(max int64) error {
	m.maxCacheSize = max
	if max <= 0 {
		return errors.Errorf("negative size or MaxCacheSize, %d", max)
	}
	return nil
}

func (m *mongoCache) setPostFlushFn(postFlushFn func()) error {
	m.postFlushFn = postFlushFn
	return nil
}
