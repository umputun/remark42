package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

	"github.com/umputun/remark/app/migrator"
	"github.com/umputun/remark/app/rest"
	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/rest/proxy"
	"github.com/umputun/remark/app/store"
	"github.com/umputun/remark/app/store/engine"
	"github.com/umputun/remark/app/store/service"
)

var testDb = "/tmp/test-remark.db"
var testHTML = "/tmp/test-remark.html"

func TestServer_Ping(t *testing.T) {
	srv, ts := prep(t)
	require.NotNil(t, srv)
	defer cleanup(ts)

	res, code := get(t, ts.URL+"/api/v1/ping")
	assert.Equal(t, "pong", res)
	assert.Equal(t, 200, code)
}

func TestServer_Create(t *testing.T) {
	srv, ts := prep(t)
	require.NotNil(t, srv)
	defer cleanup(ts)

	resp, err := post(t, ts.URL+"/api/v1/comment",
		`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	b, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	c := JSON{}
	err = json.Unmarshal(b, &c)
	assert.Nil(t, err)
	loc := c["locator"].(map[string]interface{})
	assert.Equal(t, "radio-t", loc["site"])
	assert.Equal(t, "https://radio-t.com/blah1", loc["url"])
	assert.True(t, len(c["id"].(string)) > 8)
}

func TestServer_CreateTooBig(t *testing.T) {
	srv, ts := prep(t)
	require.NotNil(t, srv)
	defer cleanup(ts)

	longComment := fmt.Sprintf(`{"text": "%4001s", "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`, "Щ")

	resp, err := post(t, ts.URL+"/api/v1/comment", longComment)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	c := JSON{}
	err = json.Unmarshal(b, &c)
	assert.Nil(t, err)

	assert.Equal(t, "comment text exceeded max allowed size 4000 (4001)", c["error"])
	assert.Equal(t, "invalid comment", c["details"])
}

func TestServer_Preview(t *testing.T) {
	srv, ts := prep(t)
	require.NotNil(t, srv)
	defer cleanup(ts)

	resp, err := post(t, ts.URL+"/api/v1/preview", `{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "<p>test 123</p>\n", string(b))
}

func TestServer_PreviewWithMD(t *testing.T) {
	srv, ts := prep(t)
	require.NotNil(t, srv)
	defer cleanup(ts)

	text := `
# h1

BKT
func TestServer_Preview(t *testing.T) {
srv, ts := prep(t)
  require.NotNil(t, srv)
}
BKT
`
	text = strings.Replace(text, "BKT", "```", -1)
	j := fmt.Sprintf(`{"text": "%s", "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`, text)
	j = strings.Replace(j, "\n", "\\n", -1)
	t.Log(j)

	resp, err := post(t, ts.URL+"/api/v1/preview", j)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "<h1>h1</h1>\n\n<pre><code>func TestServer_Preview(t *testing.T) {\nsrv, ts := prep(t)\n  require.NotNil(t, srv)\n}\n</code></pre>\n", string(b))
}

func TestServer_CreateAndGet(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	// create comment
	resp, err := post(t, ts.URL+"/api/v1/comment",
		`{"text": "**test** *123* http://radio-t.com", "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`)
	require.Nil(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	c := JSON{}
	err = json.Unmarshal(b, &c)
	assert.Nil(t, err)

	id := c["id"].(string)

	// get created comment by id
	res, code := getWithAuth(t, fmt.Sprintf("%s/api/v1/id/%s?site=radio-t&url=https://radio-t.com/blah1", ts.URL, id))
	assert.Equal(t, 200, code)
	comment := store.Comment{}
	err = json.Unmarshal([]byte(res), &comment)
	assert.Nil(t, err)
	assert.Equal(t, `<p><strong>test</strong> <em>123</em> <a href="http://radio-t.com" rel="nofollow">http://radio-t.com</a></p>`+"\n", comment.Text)
	assert.Equal(t, "**test** *123* http://radio-t.com", comment.Orig)
	assert.Equal(t, store.User{Name: "developer one", ID: "dev",
		Picture: "/api/v1/avatar/remark.image", Admin: true, Blocked: false, IP: "dbc7c999343f003f189f70aaf52cc04443f90790"},
		comment.User)
	t.Logf("%+v", comment)
}

func TestServer_Find(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	_, code := get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/blah1")
	assert.Equal(t, 400, code, "nothing in")

	c1 := store.Comment{Text: "test test #1", ParentID: "",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	id1 := addComment(t, c1, ts)

	c2 := store.Comment{Text: "test test #2", ParentID: id1,
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	id2 := addComment(t, c2, ts)

	assert.NotEqual(t, id1, id2)

	// get sorted by +time
	res, code := get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/blah1&sort=+time")
	assert.Equal(t, 200, code)
	comments := []store.Comment{}
	err := json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments), "should have 2 comments")
	assert.Equal(t, id1, comments[0].ID)
	assert.Equal(t, id2, comments[1].ID)

	// get sorted by -time
	res, code = get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/blah1&sort=-time")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments), "should have 2 comments")
	assert.Equal(t, id1, comments[1].ID)
	assert.Equal(t, id2, comments[0].ID)

	// get in tree mode
	tree := rest.Tree{}
	res, code = get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/blah1&format=tree")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &tree)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(tree.Nodes))
	assert.Equal(t, 1, len(tree.Nodes[0].Replies))
}

func TestServer_Update(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	id := addComment(t, c1, ts)

	client := http.Client{}
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/comment/"+id+"?site=radio-t&url=https://radio-t.com/blah1",
		strings.NewReader(`{"text":"updated text", "summary":"my edit"}`))
	assert.Nil(t, err)
	req = withBasicAuth(req, "dev", "password")
	b, err := client.Do(req)
	assert.Nil(t, err)
	body, err := ioutil.ReadAll(b.Body)
	assert.Nil(t, err)
	assert.Equal(t, 200, b.StatusCode, string(body))

	// comments returned by update
	c2 := store.Comment{}
	err = json.Unmarshal(body, &c2)
	assert.Nil(t, err)
	assert.Equal(t, id, c2.ID)
	assert.Equal(t, "<p>updated text</p>\n", c2.Text)
	assert.Equal(t, "updated text", c2.Orig)
	assert.Equal(t, "my edit", c2.Edit.Summary)
	assert.True(t, time.Since(c2.Edit.Timestamp) < 1*time.Second)

	// read updated comment
	res, code := getWithAuth(t, fmt.Sprintf("%s/api/v1/id/%s?site=radio-t&url=https://radio-t.com/blah1", ts.URL, id))
	assert.Equal(t, 200, code)
	c3 := store.Comment{}
	err = json.Unmarshal([]byte(res), &c3)
	assert.Nil(t, err)
	assert.Equal(t, c2, c3, "same as response from update")
}

func TestServer_Last(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah2"}}

	// add 3 comments
	addComment(t, c1, ts)
	id1 := addComment(t, c1, ts)
	id2 := addComment(t, c2, ts)

	res, code := get(t, ts.URL+"/api/v1/last/2?site=radio-t")
	assert.Equal(t, 200, code)
	comments := []store.Comment{}
	err := json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments), "should have 2 comments")
	assert.Equal(t, id1, comments[1].ID)
	assert.Equal(t, id2, comments[0].ID)

	res, code = get(t, ts.URL+"/api/v1/last/5?site=radio-t")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(comments), "should have 3 comments")
}

func TestServer_FindUserComments(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #3", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah2"}}

	// add 3 comments
	addComment(t, c1, ts)
	addComment(t, c2, ts)
	addComment(t, c2, ts)

	_, code := get(t, ts.URL+"/api/v1/comments?site=radio-t&user=blah")
	assert.Equal(t, 400, code, "noting for user blah")

	res, code := get(t, ts.URL+"/api/v1/comments?site=radio-t&user=dev")
	assert.Equal(t, 200, code)

	resp := struct {
		Comments []store.Comment
		Count    int
	}{}

	err := json.Unmarshal([]byte(res), &resp)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(resp.Comments), "should have 3 comments")
	assert.Equal(t, 3, resp.Count, "should have 3 count")
}

func TestServer_UserInfo(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	body, code := getWithAuth(t, ts.URL+"/api/v1/user?site=radio-t")
	assert.Equal(t, 200, code)
	user := store.User{}
	err := json.Unmarshal([]byte(body), &user)
	assert.Nil(t, err)
	assert.Equal(t, store.User{Name: "developer one", ID: "dev",
		Picture: "/api/v1/avatar/remark.image", Admin: true, Blocked: false, IP: ""}, user)
}

func TestServer_Vote(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}}

	id1 := addComment(t, c1, ts)
	addComment(t, c2, ts)

	vote := func(val int) int {
		client := http.Client{}
		req, err := http.NewRequest(http.MethodPut,
			fmt.Sprintf("%s/api/v1/vote/%s?site=radio-t&url=https://radio-t.com/blah&vote=%d", ts.URL, id1, val), nil)
		assert.Nil(t, err)
		req = withBasicAuth(req, "dev", "password")
		resp, err := client.Do(req)
		assert.Nil(t, err)
		return resp.StatusCode
	}

	assert.Equal(t, 200, vote(1), "first vote allowed")
	assert.Equal(t, 400, vote(1), "second vote rejected")
	body, code := get(t, fmt.Sprintf("%s/api/v1/id/%s?site=radio-t&url=https://radio-t.com/blah", ts.URL, id1))
	assert.Equal(t, 200, code)
	cr := store.Comment{}
	err := json.Unmarshal([]byte(body), &cr)
	assert.Nil(t, err)
	assert.Equal(t, 1, cr.Score)
	assert.Equal(t, map[string]bool{"dev": true}, cr.Votes)

	assert.Equal(t, 200, vote(-1), "opposite vote allowed")
	body, code = get(t, fmt.Sprintf("%s/api/v1/id/%s?site=radio-t&url=https://radio-t.com/blah", ts.URL, id1))
	assert.Equal(t, 200, code)
	cr = store.Comment{}
	err = json.Unmarshal([]byte(body), &cr)
	assert.Nil(t, err)
	assert.Equal(t, 0, cr.Score)
	assert.Equal(t, map[string]bool{}, cr.Votes)
}

func TestServer_Count(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah2"}}

	addComment(t, c1, ts)
	addComment(t, c1, ts)
	addComment(t, c1, ts)
	addComment(t, c2, ts)
	addComment(t, c2, ts)

	body, code := get(t, ts.URL+"/api/v1/count?site=radio-t&url=https://radio-t.com/blah1")
	assert.Equal(t, 200, code)
	j := JSON{}
	err := json.Unmarshal([]byte(body), &j)
	assert.Nil(t, err)
	assert.Equal(t, 3.0, j["count"])

	body, code = get(t, ts.URL+"/api/v1/count?site=radio-t&url=https://radio-t.com/blah2")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(body), &j)
	assert.Nil(t, err)
	assert.Equal(t, 2.0, j["count"])
}

func TestServer_Counts(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah2"}}

	addComment(t, c1, ts)
	addComment(t, c1, ts)
	addComment(t, c1, ts)
	addComment(t, c2, ts)
	addComment(t, c2, ts)

	resp, err := post(t, ts.URL+"/api/v1/counts?site=radio-t", `["https://radio-t.com/blah1","https://radio-t.com/blah2"]`)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)

	j := []store.PostInfo{}
	err = json.Unmarshal(body, &j)
	assert.Nil(t, err)
	assert.Equal(t, []store.PostInfo([]store.PostInfo{{URL: "https://radio-t.com/blah1", Count: 3},
		{URL: "https://radio-t.com/blah2", Count: 2}}), j)
}

func TestServer_List(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah2"}}

	addComment(t, c1, ts)
	addComment(t, c1, ts)
	addComment(t, c1, ts)
	addComment(t, c2, ts)
	addComment(t, c2, ts)

	body, code := get(t, ts.URL+"/api/v1/list?site=radio-t")
	assert.Equal(t, 200, code)
	pi := []store.PostInfo{}
	err := json.Unmarshal([]byte(body), &pi)
	assert.Nil(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/blah2", Count: 2}, {URL: "https://radio-t.com/blah1", Count: 3}}, pi)
}

func TestServer_Config(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	body, code := get(t, ts.URL+"/api/v1/config?site=radio-t")
	assert.Equal(t, 200, code)
	j := JSON{}
	err := json.Unmarshal([]byte(body), &j)
	assert.Nil(t, err)
	assert.Equal(t, 300., j["edit_duration"])
	assert.EqualValues(t, []interface{}([]interface{}{"a1", "a2"}), j["admins"])
	assert.Equal(t, 4000., j["max_comment_size"])
	assert.Equal(t, -5., j["low_score"])
	assert.Equal(t, -10., j["critical_score"])
	t.Logf("%+v", j)
}

func TestServer_FileServer(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	body, code := get(t, ts.URL+"/web/test-remark.html")
	assert.Equal(t, 200, code)
	assert.Equal(t, "some html", body)
}

func TestServer_Shutdown(t *testing.T) {
	srv := Rest{Authenticator: auth.Authenticator{},
		AvatarProxy: &proxy.Avatar{StorePath: "/tmp", RoutePath: "/api/v1/avatar"}, ImageProxy: &proxy.Image{}}
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
	dataStore := service.DataStore{Interface: b, EditDuration: 5 * time.Minute, MaxCommentSize: 4000, Secret: "123456"}
	srv = &Rest{
		DataService: dataStore,
		Authenticator: auth.Authenticator{
			DevPasswd: "password",
			Providers: nil,
			Admins:    []string{"a1", "a2"},
		},
		Exporter:    &migrator.Remark{DataStore: &dataStore},
		Cache:       &mockCache{},
		WebRoot:     "/tmp",
		AvatarProxy: &proxy.Avatar{StorePath: "/tmp", RoutePath: "/api/v1/avatar"},
		ImageProxy:  &proxy.Image{},
	}
	srv.ScoreThresholds.Low, srv.ScoreThresholds.Critical = -5, -10

	ioutil.WriteFile(testHTML, []byte("some html"), 0700)
	ts = httptest.NewServer(srv.routes())
	return srv, ts
}

func withBasicAuth(r *http.Request, username, password string) *http.Request {
	creds := username + ":" + password
	r.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(creds)))
	return r
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
	withBasicAuth(req, "dev", "password")
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
	withBasicAuth(req, "dev", "password")
	return client.Do(req)
}

func addComment(t *testing.T, c store.Comment, ts *httptest.Server) string {

	b, err := json.Marshal(c)
	assert.Nil(t, err, "can't marshal comment %+v", c)

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/comment", bytes.NewBuffer(b))
	assert.Nil(t, err)
	withBasicAuth(req, "dev", "password")
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

func (mc *mockCache) Get(key string, ttl time.Duration, fn func() ([]byte, error)) (data []byte, err error) {
	return fn()
}

func (mc *mockCache) Flush(scopes ...string) {}
