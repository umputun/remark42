package keys

import (
	"log"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/go-pkgz/mongo"
	"github.com/pkg/errors"
)

// Store defines interface returning key for given site
// this key used for JWT and HMAC hashes
type Store interface {
	Get(siteID string) (key string, err error)
}

// StaticStore implements keys.Store with a single, predefined key
type StaticStore struct {
	key string
}

// NewStaticStore makes StaticStore instance with given key
func NewStaticStore(key string) *StaticStore {
	return &StaticStore{key: key}
}

// Get returns static key for all sites, allows empty site
func (s *StaticStore) Get(siteID string) (key string, err error) {
	if s.key == "" {
		return "", errors.New("empty key for static key store")
	}
	return s.key, nil
}

// MongoStore implements keys.Store with mongo backend
type MongoStore struct {
	connection *mongo.Connection
}

// NewMongoStore makes keys Store for mongo's connection
func NewMongoStore(conn *mongo.Connection) *MongoStore {
	log.Printf("[DEBUG] make mongo keys store with %+v", conn)
	return &MongoStore{connection: conn}
}

// Get executes find by siteID and returns substructure with secret key
func (m *MongoStore) Get(siteID string) (key string, err error) {
	resp := struct {
		SiteID    string `bson:"site"`
		SecretKey string `bson:"secret"`
	}{}
	err = m.connection.WithCollection(func(coll *mgo.Collection) error {
		return coll.Find(bson.M{"site": siteID}).One(&resp)
	})
	return resp.SecretKey, errors.Wrapf(err, "can't get secret for site %s", siteID)
}
