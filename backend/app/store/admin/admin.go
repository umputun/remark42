package admin

import (
	"log"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/go-pkgz/mongo"
)

// Store defines interface returning admins info for given site
type Store interface {
	Admins(siteID string) (ids []string)
	Email(siteID string) (email string)
}

// StaticStore implements keys.Store with a single, predefined key
type StaticStore struct {
	admins []string
	email  string
}

// NewStaticStore makes StaticStore instance with given key
func NewStaticStore(admins []string, email string) *StaticStore {
	log.Printf("[DEBUG] admin users %+v, email %s", admins, email)
	return &StaticStore{admins: admins, email: email}
}

// Admins returns static list of admin's ids, the same for all sites
func (s *StaticStore) Admins(string) (ids []string) {
	return s.admins
}

// Email gets static email address
func (s *StaticStore) Email(string) (email string) {
	return s.email
}

// MongoStore implements admin.Store with mongo backend
type MongoStore struct {
	connection *mongo.Connection
}

// NewMongoStore makes admin Store for mongo's connection
func NewMongoStore(conn *mongo.Connection) *MongoStore {
	return &MongoStore{connection: conn}
}

// Admins executes find by siteID and returns admins ids
func (m *MongoStore) Admins(siteID string) (ids []string) {
	resp := struct {
		SiteID string   `bson:"site"`
		IDs    []string `bson:"admin_ids"`
		Email  string   `bson:"admin_email"`
	}{}
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
	resp := struct {
		SiteID string   `bson:"site"`
		IDs    []string `bson:"admin_ids"`
		Email  string   `bson:"admin_email"`
	}{}
	err := m.connection.WithCollection(func(coll *mgo.Collection) error {
		return coll.Find(bson.M{"site": siteID}).One(&resp)
	})
	if err != nil {
		return ""
	}
	return resp.Email
}
