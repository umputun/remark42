package admin

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	log "github.com/go-pkgz/lgr"

	"github.com/go-pkgz/mongo"
)

// MongoStore implements admin.Store with mongo backend
type MongoStore struct {
	connection *mongo.Connection
	key        string
}

type mongoRec struct {
	SiteID string   `bson:"site"`
	IDs    []string `bson:"admin_ids"`
	Email  string   `bson:"admin_email"`
}

// NewMongoStore makes admin Store for mongo's connection
func NewMongoStore(conn *mongo.Connection, key string) *MongoStore {
	log.Printf("[DEBUG] make mongo admin store with %+v", conn)
	return &MongoStore{connection: conn, key: key}
}

// Key executes find by siteID and returns substructure with secret key
func (m *MongoStore) Key() (key string, err error) {
	return m.key, nil
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
