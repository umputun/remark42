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
	"github.com/umputun/remark/backend/app/rest/proxy"
	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/engine"
	"github.com/umputun/remark/backend/app/store/service"
)

var testDb = "/tmp/test-remark.db"
var testHTML = "/tmp/test-remark.html"

func TestRest_FileServer(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	body, code := get(t, ts.URL+"/web/test-remark.html")
	assert.Equal(t, 200, code)
	assert.Equal(t, "some html", body)
}

func TestRest_Shutdown(t *testing.T) {
	srv := Rest{Authenticator: auth.Authenticator{}, AvatarProxy: &proxy.Avatar{Store: proxy.NewFSAvatarStore("/tmp", 300),
		RoutePath: "/api/v1/avatar"}, ImageProxy: &proxy.Image{}}

	go func() {
		time.Sleep(100 * time.Millisecond)
		srv.Shutdown()
	}()

	st := time.Now()
	srv.Run(0)
	assert.True(t, time.Since(st).Seconds() < 1, "should take about 100ms")
}

func prep(t *testing.T) (srv *Rest, ts *httptest.Server) {
	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: testDb, SiteID: "radio-t"})
	require.Nil(t, err)
	dataStore := &service.DataStore{Interface: b, EditDuration: 5 * time.Minute, MaxCommentSize: 4000, Secret: "123456"}
	srv = &Rest{
		DataService: dataStore,
		Authenticator: auth.Authenticator{
			DevPasswd:  "password",
			Providers:  nil,
			Admins:     []string{"a1", "a2"},
			AdminEmail: "admin@remark-42.com",
			JWTService: auth.NewJWT("12345", false, time.Minute, time.Hour),
		},
		Exporter:    &migrator.Remark{DataStore: dataStore},
		Cache:       &mockCache{},
		WebRoot:     "/tmp",
		RemarkURL:   "https://demo.remark42.com",
		AvatarProxy: &proxy.Avatar{Store: proxy.NewFSAvatarStore("/tmp", 300), RoutePath: "/api/v1/avatar"},
		ImageProxy:  &proxy.Image{},
		ReadOnlyAge: 10,
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

func cleanup(ts *httptest.Server) {
	ts.Close()
	os.Remove(testDb)
	os.Remove(testHTML)
}

type mockCache struct{}

func (mc *mockCache) Get(key string, fn func() ([]byte, error)) (data []byte, err error) {
	return fn()
}

func (mc *mockCache) Flush(scopes ...string) {}
