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

// MakeTestConnection connects to MONGO_REMARK_TEST url and returns new connection
// collection name randomized on each call
func MakeTestConnection(t *testing.T) *Connection {
	once.Do(func() {
		log.Print("[DEBUG] connect to mongo test instance")
		mongoURL := os.Getenv("MONGO_REMARK_TEST")
		if mongoURL == "" {
			mongoURL = "mongodb://mongo:27017"
			log.Printf("[WARN] no MONGO_REMARK_TEST in env")
		}
		srv, err := NewServerWithURL(mongoURL, 10*time.Second)
		assert.Nil(t, err, "failed to dial")
		collName := fmt.Sprintf("remark42_test_%d", time.Now().Nanosecond())
		conn = NewConnection(srv, "test", collName)
	})
	RemoveTestCollection(t, conn)
	return conn
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

// RemoveTestCollections clears colls
func RemoveTestCollections(t *testing.T, c *Connection, colls ...string) {
	log.Printf("[DEBUG] clean test collections %+v", colls)
	for _, collection := range colls {
		c.WithCustomCollection(collection, func(coll *mgo.Collection) error {
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
