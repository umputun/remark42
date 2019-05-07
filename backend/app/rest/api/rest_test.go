package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	bolt "github.com/coreos/bbolt"
	"github.com/go-pkgz/auth"
	"github.com/go-pkgz/auth/avatar"
	"github.com/go-pkgz/auth/token"
	R "github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/migrator"
	"github.com/umputun/remark/backend/app/rest"
	"github.com/umputun/remark/backend/app/rest/proxy"
	"github.com/umputun/remark/backend/app/store"
	adminstore "github.com/umputun/remark/backend/app/store/admin"
	"github.com/umputun/remark/backend/app/store/engine"
	"github.com/umputun/remark/backend/app/store/image"
	"github.com/umputun/remark/backend/app/store/service"
)

var testHTML = "/tmp/test-remark.html"
var getStartedHTML = "/tmp/getstarted.html"

var devToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJyZW1hcms0MiIsImV4cCI6Mzc4OTE5MTgyMiwianRpIjoicmFuZG9tIGlkIiwiaXNzIjoicmVtYXJrNDIiLCJuYmYiOjE1MjE4ODQyMjIsInVzZXIiOnsibmFtZSI6ImRldmVsb3BlciBvbmUiLCJpZCI6ImRldiIsInBpY3R1cmUiOiJodHRwOi8vZXhhbXBsZS5jb20vcGljLnBuZyIsImlwIjoiMTI3LjAuMC4xIiwiZW1haWwiOiJtZUBleGFtcGxlLmNvbSJ9fQ.aKUAXiZxXypgV7m1wEOgUcyPOvUDXHDi3A06YWKbcLg"

var adminUmputunToken = "eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOiJyYWRpb3QiLCJleHAiOjE5NTQ1OTc5ODAsImp0aSI6Ijk3YTJlMGFjNGRjN2Q1ZjY5MjZkNWU4NjIwYWNlZjlhNDBjMCIsImlhdCI6MTQ1NDU5NzY4MCwiaXNzIjoicmVtYXJrNDIiLCJ1c2VyIjp7Im5hbWUiOiJVbXB1dHVuIiwiaWQiOiJnaXRodWJfZWYwZjcwNmE3IiwicGljdHVyZSI6Imh0dHBzOi8vcmVtYXJrNDIucmFkaW8tdC5jb20vYXBpL3YxL2F2YXRhci9jYjQyZmY0OTNhZGU2OTZkODhhM2E1OTBmMTM2YWU5ZTM0ZGU3YzFiLmltYWdlIiwiYXR0cnMiOnsiYWRtaW4iOnRydWUsImJsb2NrZWQiOmZhbHNlfX19.gAR_sZT7hTx7CNHByyrJQWMB5tAtoiISAiG8kes1IjA"

func TestRest_FileServer(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	body, code := get(t, ts.URL+"/web/test-remark.html")
	assert.Equal(t, 200, code)
	assert.Equal(t, "some html", body)
}

func TestRest_GetStarted(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

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
	srv := Rest{Authenticator: &auth.Service{}, ImageProxy: &proxy.Image{}}

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
		Authenticator: auth.NewService(auth.Opts{
			AvatarStore:       avatar.NewLocalFS("/tmp"),
			AvatarResizeLimit: 300,
		}),

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
		Authenticator: &auth.Service{},
		ImageProxy:    &proxy.Image{},
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

func TestRest_rejectAnonUser(t *testing.T) {

	ts := httptest.NewServer(fakeAuth(rejectAnonUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello")
	}))))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "use not logged in")

	resp, err = http.Get(ts.URL + "?fake_id=anonymous_user123&fake_name=test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "anon rejected")

	resp, err = http.Get(ts.URL + "?fake_id=real_user123&fake_name=test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "real user")
}

func Test_URLKey(t *testing.T) {
	tbl := []struct {
		url  string
		user store.User
		key  string
	}{
		{"http://example.com/1", store.User{}, "http://example.com/1"},
		{"http://example.com/1", store.User{ID: "user"}, "http://example.com/1"},
		{"http://example.com/1", store.User{ID: "user", Admin: true}, "admin!!http://example.com/1"},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r, err := http.NewRequest("GET", tt.url, nil)
			require.NoError(t, err)
			if tt.user.ID != "" {
				r = rest.SetUserInfo(r, tt.user)
			}
			assert.Equal(t, tt.key, URLKey(r))
		})
	}

}

func Test_URLKeyWithUser(t *testing.T) {
	tbl := []struct {
		url  string
		user store.User
		key  string
	}{
		{"http://example.com/1", store.User{}, "http://example.com/1"},
		{"http://example.com/1", store.User{ID: "user"}, "user!!http://example.com/1"},
		{"http://example.com/2", store.User{ID: "user2"}, "user2!!http://example.com/2"},
		{"http://example.com/1", store.User{ID: "user", Admin: true}, "admin!!user!!http://example.com/1"},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r, err := http.NewRequest("GET", tt.url, nil)
			require.NoError(t, err)
			if tt.user.ID != "" {
				r = rest.SetUserInfo(r, tt.user)
			}
			assert.Equal(t, tt.key, URLKeyWithUser(r))
		})
	}

}
func startupT(t *testing.T) (ts *httptest.Server, srv *Rest, teardown func()) {

	testDb := fmt.Sprintf("/tmp/test-remark-%d.db", rand.Int31())
	os.Remove(testDb)
	os.Remove(testHTML)
	os.RemoveAll("/tmp/ava-remark42")
	os.RemoveAll("/tmp/pics-remark42")

	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: testDb, SiteID: "radio-t"})
	require.Nil(t, err)

	memCache, err := cache.NewMemoryCache()
	assert.NoError(t, err)

	adminStore := adminstore.NewStaticStore("123456", []string{"a1", "a2"}, "admin@remark-42.com")
	restrictedWordsMatcher := service.NewRestrictedWordsMatcher(service.StaticRestrictedWordsLister{Words: []string{"duck"}})

	dataStore := &service.DataStore{
		Interface:              b,
		EditDuration:           5 * time.Minute,
		MaxCommentSize:         4000,
		AdminStore:             adminStore,
		MaxVotes:               service.UnlimitedVotes,
		RestrictedWordsMatcher: restrictedWordsMatcher,
	}

	srv = &Rest{
		DataService: dataStore,
		Authenticator: auth.NewService(auth.Opts{
			AdminPasswd:  "password",
			SecretReader: token.SecretFunc(func() (string, error) { return "secret", nil }),
			AvatarStore:  avatar.NewLocalFS("/tmp/ava-remark42"),
		}),
		Cache:     memCache,
		WebRoot:   "/tmp",
		RemarkURL: "https://demo.remark42.com",
		ImageService: &image.Service{
			Store: &image.FileSystem{
				Location:   "/tmp/pics-remark42",
				Partitions: 100,
				MaxSize:    10000,
				Staging:    "/tmp/pics-remark42/staging",
			},
			TTL: time.Millisecond * 100,
		},
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

	teardown = func() {
		ts.Close()
		require.NoError(t, srv.DataService.Close())
		os.Remove(testDb)
		os.Remove(testHTML)
		os.RemoveAll("/tmp/ava-remark42")
		os.RemoveAll("/tmp/pics-remark42")
	}

	return ts, srv, teardown
}

// fake auth middleware make user authed and uses query's fake_id for ID and fake_name for Name
func fakeAuth(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("fake_id") != "" {
			r = rest.SetUserInfo(r, store.User{
				ID:   r.URL.Query().Get("fake_id"),
				Name: r.URL.Query().Get("fake_name"),
			})
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func get(t *testing.T, url string) (string, int) {
	r, err := http.Get(url)
	require.Nil(t, err)
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	require.Nil(t, err)
	return string(body), r.StatusCode
}

func sendReq(t *testing.T, r *http.Request, token string) (*http.Response, error) {
	client := http.Client{Timeout: 5 * time.Second}
	if token != "" {
		r.Header.Set("X-JWT", token)
	}
	return client.Do(r)
}

func getWithDevAuth(t *testing.T, url string) (body string, code int) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	req.Header.Add("X-JWT", devToken)
	r, err := client.Do(req)
	require.Nil(t, err)
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	assert.Nil(t, err)
	return string(b), r.StatusCode
}

func getWithAdminAuth(t *testing.T, url string) (string, int) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	req.SetBasicAuth("admin", "password")
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
	req.SetBasicAuth("admin", "password")
	return client.Do(req)
}

func addComment(t *testing.T, c store.Comment, ts *httptest.Server) string {
	b, err := json.Marshal(c)
	require.Nil(t, err, "can't marshal comment %+v", c)

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/comment", bytes.NewBuffer(b))
	require.Nil(t, err)
	req.Header.Add("X-JWT", devToken)
	resp, err := client.Do(req)
	require.Nil(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	b, err = ioutil.ReadAll(resp.Body)
	require.Nil(t, err)

	crResp := R.JSON{}
	err = json.Unmarshal(b, &crResp)
	require.Nil(t, err)
	time.Sleep(time.Nanosecond * 10)
	return crResp["id"].(string)
}

func requireAdminOnly(t *testing.T, req *http.Request) {
	resp, err := sendReq(t, req, "") // no-auth user
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)

	resp, err = sendReq(t, req, devToken) // non-admin user
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)
}
