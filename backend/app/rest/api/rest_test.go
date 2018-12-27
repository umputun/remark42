package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coreos/bbolt"
	"github.com/go-pkgz/auth"
	R "github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/migrator"
	"github.com/umputun/remark/backend/app/rest/proxy"
	"github.com/umputun/remark/backend/app/store"
	adminstore "github.com/umputun/remark/backend/app/store/admin"
	"github.com/umputun/remark/backend/app/store/avatar"
	"github.com/umputun/remark/backend/app/store/engine"
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

	_ = os.Remove(getStartedHTML)
	_, code = get(t, ts.URL+"/index.html")
	assert.Equal(t, 404, code)

}

func TestRest_Shutdown(t *testing.T) {
	srv := Rest{Authenticator: auth.Service{}, AvatarProxy: &proxy.Avatar{Store: avatar.NewLocalFS("/tmp", 300),
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

func TestRest_RunStaticSSLMode(t *testing.T) {
	srv := Rest{
		Authenticator: auth.Service{},
		AvatarProxy: &proxy.Avatar{
			Store:     avatar.NewLocalFS("/tmp", 300),
			RoutePath: "/api/v1/avatar",
		},
		ImageProxy: &proxy.Image{},
		SSLConfig: SSLConfig{
			SSLMode: Static,
			Port:    8443,
			Key:     "../../cmd/testdata/key.pem",
			Cert:    "../../cmd/testdata/cert.pem",
		},
		RemarkURL: "https://localhost:8443",
	}

	go func() {
		srv.Run(38080)
	}()

	time.Sleep(100 * time.Millisecond) // let server start

	client := http.Client{
		// prevent http redirect
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},

		// allow self-signed certificate
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Get("http://localhost:38080/blah?param=1")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 307, resp.StatusCode)
	assert.Equal(t, "https://localhost:8443/blah?param=1", resp.Header.Get("Location"))

	resp, err = client.Get("https://localhost:8443/ping")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "pong", string(body))

	srv.Shutdown()
}

func TestRest_RunAutocertModeHTTPOnly(t *testing.T) {
	srv := Rest{
		Authenticator: auth.Service{},
		AvatarProxy: &proxy.Avatar{
			Store:     avatar.NewLocalFS("/tmp", 300),
			RoutePath: "/api/v1/avatar",
		},
		ImageProxy: &proxy.Image{},
		SSLConfig: SSLConfig{
			SSLMode: Auto,
			Port:    8443,
		},
		RemarkURL: "https://localhost:8443",
	}

	go func() {
		// can't check https server locally, just only http server
		srv.Run(38081)
	}()

	time.Sleep(100 * time.Millisecond) // let server start

	client := http.Client{
		// prevent http redirect
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get("http://localhost:38081/blah?param=1")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 307, resp.StatusCode)
	assert.Equal(t, "https://localhost:8443/blah?param=1", resp.Header.Get("Location"))

	srv.Shutdown()
}

func prep(t *testing.T) (srv *Rest, ts *httptest.Server) {
	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: testDb, SiteID: "radio-t"})
	require.Nil(t, err)

	adminStore := adminstore.NewStaticStore("123456", []string{"a1", "a2"}, "admin@remark-42.com")

	dataStore := &service.DataStore{
		Interface:      b,
		EditDuration:   5 * time.Minute,
		MaxCommentSize: 4000,
		AdminStore:     adminStore,
		MaxVotes:       service.UnlimitedVotes,
	}

	//DevPasswd:  "password",
	//	Providers:  nil,
	//		KeyStore:   adminStore,
	//		JWTService: auth.NewJWT(adminStore, false, time.Minute, time.Hour),
	srv = &Rest{
		DataService:      dataStore,
		Authenticator:    *auth.NewService(auth.Opts{}),
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
			NativeImporter:    &migrator.Native{DataStore: dataStore},
			NativeExporter:    &migrator.Native{DataStore: dataStore},
			Cache:             &cache.Nop{},
			KeyStore:          adminStore,
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

	crResp := R.JSON{}
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
