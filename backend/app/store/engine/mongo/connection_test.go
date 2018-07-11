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

func TestConnection_WithCollection(t *testing.T) {
	c := write(t)
	defer RemoveTestCollection(t, c)

	var res []testRecord
	err := c.WithCollection(func(coll *mgo.Collection) error {
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
	c := write(t)
	defer RemoveTestCollection(t, c)

	var res []testRecord
	err := c.WithCollection(func(coll *mgo.Collection) error {
		return coll.Find(nil).All(&res)
	})
	assert.Nil(t, err)
	assert.Equal(t, 100, len(res))
}

func TestConnection_WithDB(t *testing.T) {
	c := write(t)
	defer RemoveTestCollection(t, c)

	var res []testRecord
	err := c.WithCustomDB("test", func(dbase *mgo.Database) error {
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
	c := write(t)

	var res []testRecord
	err := c.WithCustomDB("test", func(dbase *mgo.Database) error {
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

func write(t *testing.T) *Connection {
	c := MakeTestConnection(t)
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
	return c
}
