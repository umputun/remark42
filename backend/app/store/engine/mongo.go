package engine

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/engine/mongo"
)

// Mongo implements engine interface
type Mongo struct {
	*mongo.Connection
}

// Create new comment
func (m *Mongo) Create(comment store.Comment) (commentID string, err error) {
	err = m.WithCollection(func(coll *mgo.Collection) error {
		return coll.Insert(comment)
	})
	return comment.ID, err
}

// Find returns all comments for post and sorts results
func (m *Mongo) Find(locator store.Locator, sortFld string) (comments []store.Comment, err error) {
	err = m.WithCollection(func(coll *mgo.Collection) error {
		query := bson.M{"locator.site": locator.SiteID, "locator.url": locator.URL}
		return coll.Find(query).Sort(sortFld).All(&comments)
	})
	return comments, err
}

// Get returns comment for locator.URL and commentID string
func (m *Mongo) Get(_ store.Locator, commentID string) (comment store.Comment, err error) {
	err = m.WithCollection(func(coll *mgo.Collection) error {
		query := bson.M{"_id": commentID}
		return coll.Find(query).One(&comment)
	})
	return comment, err
}

// Put updates comment for locator.URL with mutable part of comment
func (m *Mongo) Put(locator store.Locator, comment store.Comment) error {
	return m.WithCollection(func(coll *mgo.Collection) error {
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
	if max > lastLimit || max == 0 {
		max = lastLimit
	}
	err = m.WithCollection(func(coll *mgo.Collection) error {
		query := bson.M{"locator.site": siteID}
		return coll.Find(query).Sort("-time").Limit(max).All(&comments)
	})
	return comments, err
}

// Count returns number of comments for locator
func (m *Mongo) Count(locator store.Locator) (count int, err error) {

	e := m.WithCollection(func(coll *mgo.Collection) error {
		query := bson.M{"locator.site": locator.SiteID, "locator.url": locator.URL}
		count, err = coll.Find(query).Count()
		return err
	})
	return count, e
}

func (m *Mongo) prepare() error {
	errs := new(multierror.Error)
	return m.WithCollection(func(coll *mgo.Collection) error {
		errs = multierror.Append(errs, coll.EnsureIndexKey("user.id", "locator.site", "time"))
		errs = multierror.Append(errs, coll.EnsureIndexKey("locator.url", "locator.site", "time"))
		return errors.Wrap(errs.ErrorOrNil(), "can't create index")
	})
}
