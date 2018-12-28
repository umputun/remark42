package admin

import (
	"log"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"

	"github.com/go-pkgz/mongo"
)

// MongoStore implements admin.Store with mongo backend
type MongoStore struct {
	connection *mongo.Connection
}

type mongoRec struct {
	SiteID    string   `bson:"site"`
	SecretKey string   `bson:"secret"`
	IDs       []string `bson:"admin_ids"`
	Email     string   `bson:"admin_email"`
}

// NewMongoStore makes admin Store for mongo's connection
func NewMongoStore(conn *mongo.Connection) *MongoStore {
	log.Printf("[DEBUG] make mongo admin store with %+v", conn)
	return &MongoStore{connection: conn}
}

// Key executes find by siteID and returns substructure with secret key
func (m *MongoStore) Key(siteID string) (key string, err error) {
	resp := mongoRec{}
	err = m.connection.WithCollection(func(coll *mgo.Collection) error {
		return coll.Find(bson.M{"site": siteID}).One(&resp)
	})
	return resp.SecretKey, errors.Wrapf(err, "can't get secret for site %s", siteID)
}

// Admins executes find by siteID and returns admins ids
func (m *MongoStore) Admins(siteID string) (ids []string) {
	resp := mongoRec{}
	err := m.connection.WithCollection(func(coll *mgo.Collection) error {
		return coll.Find(bson.M{"site": siteID}).One(&resp)
	})
	if err != nil {
		return []string{}
	}
	return resp.IDs
}

// Email executes find by siteID and returns admin's email
func (m *MongoStore) Email(siteID string) (email string) {
	resp := mongoRec{}
	err := m.connection.WithCollection(func(coll *mgo.Collection) error {
		return coll.Find(bson.M{"site": siteID}).One(&resp)
	})
	if err != nil {
		return ""
	}
	return resp.Email
}
