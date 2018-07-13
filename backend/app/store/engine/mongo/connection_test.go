package mongo

import (
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var conn *Connection
var once sync.Once

// MakeTestConnection connects to MONGO_REMARK_TEST url or "mongo" host (in no env) and returns new connection.
// collection name randomized on each call
func MakeTestConnection(t *testing.T) (*Connection, error) {
	mongoURL := os.Getenv("MONGO_REMARK_TEST")
	if mongoURL == "" {
		mongoURL = "mongodb://mongo:27017"
		log.Printf("[WARN] no MONGO_REMARK_TEST in env")
	}
	if mongoURL == "skip" {
		log.Print("skip mongo test")
		return nil, errors.New("skip")
	}

	once.Do(func() {
		log.Print("[DEBUG] connect to mongo test instance")
		srv, err := NewServerWithURL(mongoURL, 10*time.Second)
		assert.Nil(t, err, "failed to dial")
		collName := fmt.Sprintf("remark42_test_%d", time.Now().Nanosecond())
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

type testRecord struct {
	Symbol string
	Num    int
}

func TestConnection_WithCollection(t *testing.T) {
	c, err := write(t)
	if err != nil {
		return
	}
	defer RemoveTestCollection(t, c)

	var res []testRecord
	err = c.WithCollection(func(coll *mgo.Collection) error {
		return coll.Find(nil).All(&res)
	})
	assert.Nil(t, err)
	assert.Equal(t, 100, len(res))

	err = c.WithCollection(func(coll *mgo.Collection) error {
		return coll.Find(bson.M{"symbol": "blah"}).All(&res)
	})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(res))

	r1 := testRecord{}
	err = c.WithCollection(func(coll *mgo.Collection) error {
		return coll.Find(bson.M{"symbol": "blah"}).One(&r1)
	})
	assert.Equal(t, mgo.ErrNotFound, err)

	c = NewConnection(c.server, "test", "bbbbbbbaaad")
	err = c.WithCollection(func(coll *mgo.Collection) error {
		return coll.Find(bson.M{"symbol": "blah"}).One(&r1)
	})
	assert.Equal(t, mgo.ErrNotFound, err)
}

func TestConnection_WithCollectionNoDB(t *testing.T) {
	c, err := write(t)
	if err != nil {
		return
	}
	defer RemoveTestCollection(t, c)

	var res []testRecord
	err = c.WithCollection(func(coll *mgo.Collection) error {
		return coll.Find(nil).All(&res)
	})
	assert.Nil(t, err)
	assert.Equal(t, 100, len(res))
}

func TestConnection_WithDB(t *testing.T) {
	c, err := write(t)
	if err != nil {
		return
	}
	defer RemoveTestCollection(t, c)

	var res []testRecord
	err = c.WithCustomDB("test", func(dbase *mgo.Database) error {
		return dbase.C(c.collection).Find(nil).All(&res)
	})
	assert.Nil(t, err)
	assert.Equal(t, 100, len(res))

	err = c.WithDB(func(dbase *mgo.Database) error {
		return dbase.C(c.collection).Find(nil).All(&res)
	})
	assert.Nil(t, err)
	assert.Equal(t, 100, len(res))
}

func TestCleanup(t *testing.T) {
	c, err := write(t)
	if err != nil {
		return
	}
	var res []testRecord
	err = c.WithCustomDB("test", func(dbase *mgo.Database) error {
		return dbase.C(c.collection).Find(nil).All(&res)
	})
	assert.Nil(t, err)
	assert.Equal(t, 100, len(res))

	RemoveTestCollections(t, c, c.collection)
	err = c.WithCustomDB("test", func(dbase *mgo.Database) error {
		return dbase.C(c.collection).Find(nil).All(&res)
	})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(res))
}

func write(t *testing.T) (*Connection, error) {
	c, err := MakeTestConnection(t)
	if err != nil {
		return nil, err
	}
	c.WithCollection(func(coll *mgo.Collection) error {
		for i := 0; i < 100; i++ {
			r := testRecord{
				Symbol: fmt.Sprintf("symb-%02d", i%5),
				Num:    i,
			}
			assert.Nil(t, coll.Insert(r), fmt.Sprintf("insert %+v", r))
		}
		return nil
	})
	return c, nil
}
