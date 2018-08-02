package mongo

import (
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/globalsign/mgo"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var conn *Connection
var once sync.Once

// MakeTestConnection connects to MONGO_TEST url or "mongo" host (in no env) and returns new connection.
// collection name randomized on each call
func MakeTestConnection(t *testing.T) (*Connection, error) {
	mongoURL := getMongoURL(t)
	once.Do(func() {
		log.Print("[DEBUG] connect to mongo test instance")
		srv, err := NewServerWithURL(mongoURL, 10*time.Second)
		assert.Nil(t, err, "failed to dial")
		collName := fmt.Sprintf("test_%d", time.Now().Nanosecond())
		conn = NewConnection(srv, "test", collName)
	})
	RemoveTestCollection(t, conn)
	return conn, nil
}

// RemoveTestCollection removes all records and drop collection from connection
func RemoveTestCollection(t *testing.T, c *Connection) {
	log.Printf("[DEBUG] clean test collection %+v", c.collection)
	_ = c.WithCollection(func(coll *mgo.Collection) error {
		_, e := coll.RemoveAll(nil)
		require.Nil(t, e, "failed to remove records, %s", e)
		e = coll.DropCollection()
		if e != nil && e.Error() != "ns not found" {
			require.Nil(t, e, "failed to drop collection, %s", e)
		}
		return e
	})
}

// RemoveTestCollections clears passed collections
func RemoveTestCollections(t *testing.T, c *Connection, collections ...string) {
	log.Printf("[DEBUG] clean test collections %+v", collections)
	for _, collection := range collections {
		_ = c.WithCustomCollection(collection, func(coll *mgo.Collection) error {
			_, e := coll.RemoveAll(nil)
			require.Nil(t, e, "failed to remove records, %s", e)
			e = coll.DropCollection()
			if e != nil && e.Error() != "ns not found" {
				require.Nil(t, e, "failed to drop collection, %s", e)
			}
			return e
		})
	}

}

func getMongoURL(t *testing.T) string {
	mongoURL := os.Getenv("MONGO_TEST")
	if mongoURL == "" {
		mongoURL = "mongodb://mongo:27017"
		t.Logf("no MONGO_TEST in env, defaulted to %s", mongoURL)
	}
	if mongoURL == "skip" {
		t.Skip("skip mongo test")
	}
	return mongoURL
}
