// Package mongo wraps mgo to provide easier way to construct mongo server (with auth).
// Connection provides With* func warapers to run query with session copy
package mongo

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/pkg/errors"
)

// Server represents mongo instance and provides session accessor
type Server struct {
	dial   mgo.DialInfo
	params ServerParams
	sess   *mgo.Session
}

// ServerParams optional set of parameters
type ServerParams struct {
	Delay           int  // initial delay to give mongo server some time to start, in case if mongo part of the same compose
	Debug           bool // turn on mgo debug mode
	ConsistencyMode mgo.Mode
	SSL             bool
}

// NewDefaultServer makes a server with default extra params.
func NewDefaultServer(dial mgo.DialInfo) (res *Server, err error) {
	return NewServer(dial, ServerParams{ConsistencyMode: mgo.Monotonic})
}

// NewServerWithURL makes mongo server from url like
// mongodb://remark42:password@127.0.0.1t:27017/test?ssl=true&replicaSet=Cluster0-shard-0&authSource=admin
func NewServerWithURL(url string, timeout time.Duration) (res *Server, err error) {
	dial, params, err := parseURL(url, timeout)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create mongo server from url %s", url)
	}
	return NewServer(dial, params)
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
		return nil, errors.New("missing mongo address")
	}

	if params.Delay > 0 {
		log.Printf("[DEBUG] initial mongo delay=%d", params.Delay)
		time.Sleep(time.Duration(params.Delay) * time.Second)
	}

	log.Printf("[DEBUG] dial mongo %s, ssl=%v", dial.Addrs, params.SSL)

	if params.SSL {
		tlsConfig := &tls.Config{}
		dial.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			conn, e := tls.Dial("tcp", addr.String(), tlsConfig)
			return conn, e
		}
	}

	session, err := mgo.DialWithInfo(&dial)
	if err != nil {
		err = fmt.Errorf("can't connect to mongo, %v", err)
		log.Printf("[ERROR] %v", err)
		return nil, err
	}
	session.SetMode(params.ConsistencyMode, true)
	session.SetSyncTimeout(30 * time.Second)
	session.SetSocketTimeout(dial.Timeout)

	if dial.Username != "" && dial.Password != "" {
		creds := &mgo.Credential{Username: dial.Username, Password: dial.Password, Source: dial.Source}
		log.Printf("[DEBUG] login to mongo, user=%s, db=%s", creds.Username, creds.Source)
		if err = session.Login(creds); err != nil {
			log.Printf("[ERROR] can't login to mongo, %v", err)
			return nil, err
		}
	}

	result.sess = session
	return &result, nil
}

// SessionCopy returns copy of main session. Client should close it
func (m Server) SessionCopy() *mgo.Session {
	return m.sess.Copy()
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

// parseURL extends mgo with debug option and extracts ssl flag to make ServerParams
func parseURL(mongoURL string, connectTimeout time.Duration) (mgo.DialInfo, ServerParams, error) {
	params := ServerParams{
		ConsistencyMode: mgo.Monotonic,
		SSL:             strings.Contains(mongoURL, "ssl=true"),
		Debug:           strings.Contains(mongoURL, "debug=true"),
	}
	mongoURL = strings.Replace(mongoURL, "&debug=true", "", 1)
	dial, err := mgo.ParseURL(mongoURL)
	if err != nil {
		return mgo.DialInfo{}, ServerParams{}, errors.Wrapf(err, "failed to pars mongo url %s", mongoURL)
	}
	dial.Timeout = connectTimeout
	return *dial, params, nil
}
