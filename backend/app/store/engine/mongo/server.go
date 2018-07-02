// Package mongo wraps mgo to provide easier way to construct mongo server (with auth).
// Connection provides With* func warapers to run query with session copy
package mongo

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/globalsign/mgo"
)

// Server represents mongo instance and provides session accessor
type Server struct {
	dial   mgo.DialInfo
	params ServerParams
	sess   *mgo.Session
}

// ServerParams optional set of parameters
type ServerParams struct {
	Delay           int  // initial delay. needs to give mongo server some time to start, in case if mongo part of the same compose
	Debug           bool // turn on mgo debug mode
	Credential      *mgo.Credential
	ConsistencyMode mgo.Mode
}

// NewDefaultServer makes a server with default extra params.
func NewDefaultServer(dial mgo.DialInfo) (res *Server, err error) {
	return NewServer(dial, ServerParams{ConsistencyMode: mgo.Monotonic})
}

// NewServer doing auth if passwd != "" and can delay to make sure local mongo is up
func NewServer(dial mgo.DialInfo, params ServerParams) (res *Server, err error) {
	log.Printf("[INFO] make new mongo server %v with %+v", dial.Addrs, params)
	result := Server{dial: dial, params: params}

	if params.Debug {
		mgo.SetDebug(true)
		mgo.SetLogger(log.New(os.Stdout, "MGO ", log.Ldate|log.Ltime|log.Lmicroseconds))
	}

	if len(dial.Addrs) == 0 {
		err = errors.New("missing mongo address")
		log.Printf("[ERROR] %v", err)
		return nil, err
	}

	if params.Delay > 0 {
		log.Printf("[DEBUG] initial mongo delay=%d", params.Delay)
		time.Sleep(time.Duration(params.Delay) * time.Second)
	}

	log.Printf("[DEBUG] dial mongo %s", dial.Addrs)

	session, err := mgo.DialWithInfo(&dial)
	if err != nil {
		err = fmt.Errorf("can't connect to mongo, %v", err)
		log.Printf("[ERROR] %v", err)
		return nil, err
	}
	session.SetMode(params.ConsistencyMode, true)

	if params.Credential != nil && params.Credential.Username != "" && params.Credential.Password != "" {
		log.Printf("[INFO] login to mongo, user %s", params.Credential.Username)
		if err = session.Login(params.Credential); err != nil {
			log.Printf("[ERROR] can't login to mongo, %v", err)
			return nil, err
		}
	}

	result.sess = session
	return &result, nil
}

// SessionCopy returns copy of main session. Client should close it
func (m Server) SessionCopy() *mgo.Session {
	r := m.sess.Copy()
	r.SetSyncTimeout(30 * time.Second)
	r.SetSocketTimeout(m.dial.Timeout)
	return r
}

func (m Server) String() string {
	return fmt.Sprintf("%v%s", m.dial.Addrs, m.dial.Database)
}

// ParseMode translate reading mode string (case insensitive) to mgo.Mode
func ParseMode(m string) mgo.Mode {
	switch strings.ToLower(m) {
	case "primary", "strong":
		return mgo.Primary
	case "primary_pref":
		return mgo.PrimaryPreferred
	case "secondary":
		return mgo.Secondary
	case "secondary_pref":
		return mgo.SecondaryPreferred
	case "eventual":
		return mgo.Eventual
	case "monotonic":
		return mgo.Monotonic
	}
	return mgo.PrimaryPreferred
}
