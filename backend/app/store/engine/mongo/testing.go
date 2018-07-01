package mongo

import (
	"log"

	"github.com/globalsign/mgo"
)

// Testing represents basic operations needed in usual mongo-related tests. Enforces test db, "mongo" host
type Testing struct {
	collection string
}

// NewTesting makes new testingMongo for given collection
func NewTesting(collection string) *Testing {
	return &Testing{collection: collection}
}

// WriteRecords makes fresh collection and write records
func (t Testing) WriteRecords(records ...interface{}) {

	session, err := mgo.Dial("mongo")
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()
	_ = session.DB("test").C(t.collection).DropCollection()
	c := session.DB("test").C(t.collection)
	for i, r := range records {
		if err := c.Insert(r); err != nil {
			log.Fatalf("failed to insert #%d, %v", i, err)
		}
	}
}

// DropCollection removed collection from "test" db
func (t Testing) DropCollection() error {
	session, err := mgo.Dial("mongo")
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()
	return session.DB("test").C(t.collection).DropCollection()
}

// Get testing connection
func (t Testing) Get() (*Connection, error) {
	srv, err := NewDefaultServer(mgo.DialInfo{Addrs: []string{"mongo"}, Database: "test"})
	if err != nil {
		return nil, err
	}
	return &Connection{Server: srv, Collection: t.collection, DB: "test"}, nil
}
