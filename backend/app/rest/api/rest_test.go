package api

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coreos/bbolt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/migrator"
	"github.com/umputun/remark/backend/app/rest/auth"
	"github.com/umputun/remark/backend/app/rest/cache"
	"github.com/umputun/remark/backend/app/rest/proxy"
	"github.com/umputun/remark/backend/app/store"
	adminstore "github.com/umputun/remark/backend/app/store/admin"
	"github.com/umputun/remark/backend/app/store/avatar"
	"github.com/umputun/remark/backend/app/store/engine"
	"github.com/umputun/remark/backend/app/store/keys"
	"github.com/umputun/remark/backend/app/store/service"
)

var testDb = "/tmp/test-remark.db"
var testHTML = "/tmp/test-remark.html"
var getStartedHTML = "/tmp/getstarted.html"

func TestRest_FileServer(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts, srv)

	body, code := get(t, ts.URL+"/web/test-remark.html")
	assert.Equal(t, 200, code)
	assert.Equal(t, "some html", body)
}

func TestRest_GetStarted(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts, srv)

	err := ioutil.WriteFile(getStartedHTML, []byte("some html blah"), 0700)
	assert.Nil(t, err)

	body, code := get(t, ts.URL+"/index.html")
	assert.Equal(t, 200, code)
	assert.Equal(t, "some html blah", body)

	os.Remove(getStartedHTML)
	_, code = get(t, ts.URL+"/index.html")
	assert.Equal(t, 404, code)

}

func TestRest_Shutdown(t *testing.T) {
	srv := Rest{Authenticator: auth.Authenticator{}, AvatarProxy: &proxy.Avatar{Store: avatar.NewLocalFS("/tmp", 300),
		RoutePath: "/api/v1/avatar"}, ImageProxy: &proxy.Image{}}

	go func() {
		time.Sleep(100 * time.Millisecond)
		srv.Shutdown()
	}()

	st := time.Now()
	srv.Run(0)
	assert.True(t, time.Since(st).Seconds() < 1, "should take about 100ms")
}

func TestRest_filterComments(t *testing.T) {
	user := store.User{ID: "user1", Name: "user name 1"}
	c1 := store.Comment{User: user, Text: "test test #1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 05, 27, 1, 14, 10, 0, time.Local)}
	c2 := store.Comment{User: user, Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 05, 27, 1, 14, 20, 0, time.Local)}
	c3 := store.Comment{User: user, Text: "test test #3", ParentID: "p1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 05, 27, 1, 14, 25, 0, time.Local)}

	r := filterComments([]store.Comment{c1, c2, c3}, func(c store.Comment) bool {
		return c.Text == "test test #1" || c.Text == "test test #3"
	})
	assert.Equal(t, 2, len(r), "one comment filtered")
}

func prep(t *testing.T) (srv *Rest, ts *httptest.Server) {
	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: testDb, SiteID: "radio-t"})
	require.Nil(t, err)

	adminStore := adminstore.NewStaticStore([]string{"a1", "a2"}, "admin@remark-42.com")

	dataStore := &service.DataStore{
		Interface:      b,
		EditDuration:   5 * time.Minute,
		MaxCommentSize: 4000,
		KeyStore:       keys.NewStaticStore("123456"),
		AdminStore:     adminStore,
	}
	srv = &Rest{
		DataService: dataStore,
		Authenticator: auth.Authenticator{
			DevPasswd:  "password",
			Providers:  nil,
			AdminStore: adminStore,
			JWTService: auth.NewJWT(keys.NewStaticStore("123456"), false, time.Minute, time.Hour),
		},
		Cache:            &cache.Nop{},
		WebRoot:          "/tmp",
		RemarkURL:        "https://demo.remark42.com",
		AvatarProxy:      &proxy.Avatar{Store: avatar.NewLocalFS("/tmp", 300), RoutePath: "/api/v1/avatar"},
		ImageProxy:       &proxy.Image{},
		ReadOnlyAge:      10,
		CommentFormatter: store.NewCommentFormatter(&proxy.Image{}),
		Migrator: &Migrator{
			DisqusImporter:    &migrator.Disqus{DataStore: dataStore},
			WordPressImporter: &migrator.WordPress{DataStore: dataStore},
			NativeImporter:    &migrator.Remark{DataStore: dataStore},
			NativeExported:    &migrator.Remark{DataStore: dataStore},
			Cache:             &cache.Nop{},
			KeyStore:          keys.NewStaticStore("123456"),
		},
	}
	srv.ScoreThresholds.Low, srv.ScoreThresholds.Critical = -5, -10

	err = ioutil.WriteFile(testHTML, []byte("some html"), 0700)
	assert.Nil(t, err)
	ts = httptest.NewServer(srv.routes())
	return srv, ts
}

func get(t *testing.T, url string) (string, int) {
	r, err := http.Get(url)
	require.Nil(t, err)
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	require.Nil(t, err)
	return string(body), r.StatusCode
}

func getWithAuth(t *testing.T, url string) (string, int) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	req.SetBasicAuth("dev", "password")
	r, err := client.Do(req)
	require.Nil(t, err)
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	assert.Nil(t, err)
	return string(body), r.StatusCode
}

func post(t *testing.T, url string, body string) (*http.Response, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	assert.Nil(t, err)
	req.SetBasicAuth("dev", "password")
	return client.Do(req)
}

func addComment(t *testing.T, c store.Comment, ts *httptest.Server) string {

	b, err := json.Marshal(c)
	assert.Nil(t, err, "can't marshal comment %+v", c)

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/comment", bytes.NewBuffer(b))
	assert.Nil(t, err)
	req.SetBasicAuth("dev", "password")
	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	b, err = ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)

	crResp := JSON{}
	err = json.Unmarshal(b, &crResp)
	assert.Nil(t, err)
	time.Sleep(time.Nanosecond * 10)
	return crResp["id"].(string)
}

func cleanup(ts *httptest.Server, srv *Rest) {
	ts.Close()
	srv.DataService.Close()
	os.Remove(testDb)
	os.Remove(testHTML)
}
