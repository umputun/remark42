package mongo

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"log"

	"github.com/globalsign/mgo"
)

// sessionFn is a function for all With calls, terminator not supported
type sessionFn func(coll *mgo.Collection) error

// Connection allows to run request in separate session, closing automatically
type Connection struct {
	Server         *Server
	DB, Collection string
}

// WithCollection passes fun with mgo.Collection from session copy, closes it after done,
// uses Connection.DB and Connection.Collection
func (c *Connection) WithCollection(fun sessionFn) (err error) {
	return c.WithCustomCollection(c.Collection, fun)
}

// WithCustomCollection passes fun with mgo.Collection from session copy, closes it after done
// uses Connection.DB or (if not defined) dial.Database, and user-defined collection
func (c *Connection) WithCustomCollection(collection string, fun sessionFn) (err error) {
	db := c.Server.dial.Database
	if c.DB != "" {
		db = c.DB
	}
	return c.WithCustomDbCollection(db, collection, fun)
}

// WithCustomDbCollection passed fun with mgo.Collection from session copy, closes it after done
// uses passed db and collection directly.
func (c *Connection) WithCustomDbCollection(db string, collection string, fun sessionFn) (err error) {
	session := c.Server.SessionCopy()
	defer session.Close()
	return fun(session.DB(db).C(collection))
}

// WithDB passes fun with mgo.Database from session copy, closes it after done
// uses Connection.DB or (if not defined) dial.Database
func (c *Connection) WithDB(fun func(dbase *mgo.Database) error) (err error) {
	db := c.Server.dial.Database
	if c.DB != "" {
		db = c.DB
	}
	return c.WithCustomDB(db, fun)
}

// WithCustomDB passes fun with mgo.Database from session copy, closes it after done
// uses passed db directly
func (c *Connection) WithCustomDB(db string, fun func(dbase *mgo.Database) error) (err error) {
	session := c.Server.SessionCopy()
	defer session.Close()
	return fun(session.DB(db))
}

// MakeRandomID generates sha1(random) string
func MakeRandomID() string {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("[ERROR] can't get randoms, %s", err)
	}
	s := sha1.New()
	if _, err := s.Write(b); err != nil {
		log.Fatalf("[ERROR] can't make sha1 for random, %s", err)
	}
	return fmt.Sprintf("%x", s.Sum(nil))
}

func (c *Connection) String() string {
	return fmt.Sprintf("mongo:%s, db:%s, collection:%s", c.Server, c.DB, c.Collection)
}
