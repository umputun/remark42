package mongo

import (
	"fmt"
	"testing"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/stretchr/testify/assert"
)

type testRecord struct {
	Symbol string
	Num    int
}

func TestWithCollection(t *testing.T) {
	write(t)

	var res []testRecord
	srv, err := NewServer(mgo.DialInfo{Addrs: []string{"mongo"}}, ServerParams{})
	assert.Nil(t, err)

	c := Connection{Server: srv, DB: "test", Collection: "connection"}
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

	c = Connection{Server: srv, DB: "test", Collection: "bbbbbbbaaad"}
	err = c.WithCollection(func(coll *mgo.Collection) error {
		return coll.Find(bson.M{"symbol": "blah"}).One(&r1)
	})
	assert.Equal(t, mgo.ErrNotFound, err)
}

func TestWithCollectionNoDB(t *testing.T) {
	write(t)

	var res []testRecord
	srv, err := NewServer(mgo.DialInfo{Addrs: []string{"mongo"}, Database: "test"}, ServerParams{})
	assert.Nil(t, err)
	c := Connection{Server: srv, Collection: "connection"}
	err = c.WithCollection(func(coll *mgo.Collection) error {
		return coll.Find(nil).All(&res)
	})
	assert.Nil(t, err)
	assert.Equal(t, 100, len(res))
}

func TestWithDB(t *testing.T) {
	write(t)

	var res []testRecord
	srv, err := NewServer(mgo.DialInfo{Addrs: []string{"mongo"}, Database: "test"}, ServerParams{})
	assert.Nil(t, err)
	c := Connection{Server: srv}
	err = c.WithCustomDB("test", func(dbase *mgo.Database) error {
		return dbase.C("connection").Find(nil).All(&res)
	})
	assert.Nil(t, err)
	assert.Equal(t, 100, len(res))
}

func write(t *testing.T) {
	mongo, err := mgo.Dial("mongo")
	assert.Nil(t, err, "connect to mongo")
	coll := mongo.DB("test").C("connection")
	_ = coll.DropCollection()

	for i := 0; i < 100; i++ {
		r := testRecord{
			Symbol: fmt.Sprintf("symb-%02d", i%5),
			Num:    i,
		}
		assert.Nil(t, coll.Insert(r), fmt.Sprintf("insert %+v", r))
	}
}
