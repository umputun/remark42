package mongo

import (
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/globalsign/mgo"
	"github.com/stretchr/testify/assert"
)

func TestServer_NewServerGood(t *testing.T) {
	mongoURL := os.Getenv("MONGO_REMARK_TEST")
	if mongoURL == "" {
		mongoURL = "mongodb://mongo:27017"
		log.Printf("[WARN] no MONGO_REMARK_TEST in env")
	}
	m, err := NewServerWithURL(mongoURL, 3*time.Second)
	assert.Nil(t, err)
	assert.NotNil(t, m)
	assert.True(t, strings.HasSuffix(m.String(), "test"))
}

func TestServer_NewServerBad(t *testing.T) {
	_, err := NewServerWithURL("mongodb://127.0.0.3:27017/test", 100*time.Millisecond)
	assert.NotNil(t, err)
	t.Log(err)

	_, err = NewServer(mgo.DialInfo{Addrs: []string{"127.0.0.2"}, Timeout: 100 * time.Millisecond}, ServerParams{})
	assert.NotNil(t, err)

	_, err = NewServer(mgo.DialInfo{}, ServerParams{})
	assert.NotNil(t, err)
}

func TestServer_parse(t *testing.T) {
	tbl := []struct {
		mongoURL string
		timeout  time.Duration
		params   ServerParams
		dial     mgo.DialInfo
		isErr    bool
	}{
		{
			"mongodb://127.0.0.3:27017/test", time.Millisecond,
			ServerParams{ConsistencyMode: 1},
			mgo.DialInfo{Addrs: []string{"127.0.0.3:27017"}, Timeout: 1000000, Database: "test",
				ReadPreference: &mgo.ReadPreference{Mode: 2}},
			false,
		},
		{
			"mongodb://user:passwd@127.0.0.3:27017/test?ssl=true&authSource=admin", time.Millisecond,
			ServerParams{ConsistencyMode: 1, SSL: true},
			mgo.DialInfo{Addrs: []string{"127.0.0.3:27017"}, Timeout: 1000000, Database: "test", Source: "admin",
				Username: "user", Password: "passwd", ReadPreference: &mgo.ReadPreference{Mode: 2}},
			false,
		},
		{
			"mongodb://127.0.0.3", time.Millisecond,
			ServerParams{ConsistencyMode: 1, SSL: false},
			mgo.DialInfo{Addrs: []string{"127.0.0.3"}, Timeout: 1000000, ReadPreference: &mgo.ReadPreference{Mode: 2}},
			false,
		},
		{
			"127.0.0.3", time.Millisecond,
			ServerParams{ConsistencyMode: 1, SSL: false},
			mgo.DialInfo{Addrs: []string{"127.0.0.3"}, Timeout: 1000000, ReadPreference: &mgo.ReadPreference{Mode: 2}},
			false,
		},
		{
			"127.0.0.3?xxx=yyy", time.Millisecond,
			ServerParams{}, mgo.DialInfo{},
			true,
		},
	}

	for i, tt := range tbl {
		dial, params, err := parseURL(tt.mongoURL, tt.timeout)
		dial.DialServer = nil
		if tt.isErr {
			assert.NotNil(t, err, "expect error #%d", i)
			t.Logf("dial %+v, params %+v", dial, params)
			continue
		}
		assert.Equal(t, tt.dial, dial, "test #%d", i)
		assert.Equal(t, tt.params, params, "test #%d", i)
	}
}
