package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/app/migrator"
	"github.com/umputun/remark/app/notifier"
	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/rest/avatar"
	"github.com/umputun/remark/app/store"
)

var testDb = "/tmp/test-remark.db"

func TestServer_Ping(t *testing.T) {
	srv, port := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(srv)

	res, code := get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/ping", port))
	assert.Equal(t, "pong", res)
	assert.Equal(t, 200, code)
}

func TestServer_Create(t *testing.T) {
	srv, port := prep(t)
	require.NotNil(t, srv)
	defer cleanup(srv)

	r := strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/v1/comment", port), "application/json", r)
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

func TestServer_CreateAndGet(t *testing.T) {
	srv, port := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(srv)

	// create comment
	r := strings.NewReader(`{"text": "**test** *123* http://radio-t.com", "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/v1/comment", port), "application/json", r)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	c := JSON{}
	err = json.Unmarshal(b, &c)
	assert.Nil(t, err)

	id := c["id"].(string)

	// get created comment by id
	res, code := get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/id/%s?site=radio-t&url=https://radio-t.com/blah1", port, id))
	assert.Equal(t, 200, code)
	comment := store.Comment{}
	err = json.Unmarshal([]byte(res), &comment)
	assert.Nil(t, err)
	assert.Equal(t, "<p><strong>test</strong> <em>123</em> http://radio-t.com</p>", comment.Text)
	assert.Equal(t, store.User{Name: "developer one", ID: "dev",
		Picture: "https://friends.radio-t.com/resources/images/rt_logo_64.png",
		Profile: "https://radio-t.com/info/", Admin: true, Blocked: false, IP: ""},
		comment.User)
	t.Logf("%+v", comment)
}

func TestServer_Find(t *testing.T) {
	srv, port := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(srv)

	_, code := get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/find?site=radio-t&url=https://radio-t.com/blah1", port))
	assert.Equal(t, 400, code, "nothing in")

	c1 := store.Comment{Text: "test test #1", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}

	id1 := addComment(t, c1, port)
	id2 := addComment(t, c2, port)
	assert.NotEqual(t, id1, id2)

	// get sorted by +time
	res, code := get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/find?site=radio-t&url=https://radio-t.com/blah1&sort=+time", port))
	assert.Equal(t, 200, code)
	comments := []store.Comment{}
	err := json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments), "should have 2 comments")
	assert.Equal(t, id1, comments[0].ID)
	assert.Equal(t, id2, comments[1].ID)

	// get sorted by -time
	res, code = get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/find?site=radio-t&url=https://radio-t.com/blah1&sort=-time", port))
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments), "should have 2 comments")
	assert.Equal(t, id1, comments[1].ID)
	assert.Equal(t, id2, comments[0].ID)
}

func TestServer_Update(t *testing.T) {
	srv, port := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(srv)

	c1 := store.Comment{Text: "test test #1", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	id := addComment(t, c1, port)

	client := http.Client{}
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("http://127.0.0.1:%d/api/v1/comment/"+id+"?site=radio-t&url=https://radio-t.com/blah1", port),
		strings.NewReader(`{"text":"updated text", "summary":"my edit"}`))
	assert.Nil(t, err)
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
	assert.Equal(t, "<p>updated text</p>", c2.Text)
	assert.Equal(t, "my edit", c2.Edit.Summary)
	assert.True(t, time.Since(c2.Edit.Timestamp) < 1*time.Second)

	// read updated comment
	res, code := get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/id/%s?site=radio-t&url=https://radio-t.com/blah1", port, id))
	assert.Equal(t, 200, code)
	c3 := store.Comment{}
	err = json.Unmarshal([]byte(res), &c3)
	assert.Nil(t, err)
	assert.Equal(t, c2, c3, "same as response from update")
}

func TestServer_Last(t *testing.T) {
	srv, port := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(srv)

	c1 := store.Comment{Text: "test test #1", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah2"}}

	// add 3 comments
	addComment(t, c1, port)
	id1 := addComment(t, c1, port)
	id2 := addComment(t, c2, port)

	res, code := get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/last/2?site=radio-t", port))
	assert.Equal(t, 200, code)
	comments := []store.Comment{}
	err := json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments), "should have 2 comments")
	assert.Equal(t, id1, comments[1].ID)
	assert.Equal(t, id2, comments[0].ID)

	res, code = get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/last/5?site=radio-t", port))
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(comments), "should have 3 comments")
}

func TestServer_FindUserComments(t *testing.T) {
	srv, port := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(srv)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #3", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah2"}}

	// add 3 comments
	addComment(t, c1, port)
	addComment(t, c2, port)
	addComment(t, c2, port)

	_, code := get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/comments?site=radio-t&user=blah", port))
	assert.Equal(t, 400, code, "noting for user blah")

	res, code := get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/comments?site=radio-t&user=dev", port))
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
	srv, port := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(srv)

	body, code := get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/user?site=radio-t", port))
	assert.Equal(t, 200, code)
	user := store.User{}
	err := json.Unmarshal([]byte(body), &user)
	assert.Nil(t, err)
	assert.Equal(t, store.User{Name: "developer one", ID: "dev",
		Picture: "https://friends.radio-t.com/resources/images/rt_logo_64.png", Profile: "https://radio-t.com/info/",
		Admin: true, Blocked: false, IP: ""}, user)
}

func TestServer_Vote(t *testing.T) {
	srv, port := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(srv)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}}

	id1 := addComment(t, c1, port)
	addComment(t, c2, port)

	vote := func(val int) int {
		client := http.Client{}
		req, err := http.NewRequest(http.MethodPut,
			fmt.Sprintf("http://127.0.0.1:%d/api/v1/vote/%s?site=radio-t&url=https://radio-t.com/blah&vote=%d", port, id1, val),
			nil)
		assert.Nil(t, err)
		resp, err := client.Do(req)
		assert.Nil(t, err)
		return resp.StatusCode
	}

	assert.Equal(t, 200, vote(1), "first vote allowed")
	assert.Equal(t, 400, vote(1), "second vote rejected")
	body, code := get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/id/%s?site=radio-t&url=https://radio-t.com/blah", port, id1))
	assert.Equal(t, 200, code)
	cr := store.Comment{}
	err := json.Unmarshal([]byte(body), &cr)
	assert.Nil(t, err)
	assert.Equal(t, 1, cr.Score)
	assert.Equal(t, map[string]bool{"dev": true}, cr.Votes)

	assert.Equal(t, 200, vote(-1), "opposite vote allowed")
	body, code = get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/id/%s?site=radio-t&url=https://radio-t.com/blah", port, id1))
	assert.Equal(t, 200, code)
	cr = store.Comment{}
	err = json.Unmarshal([]byte(body), &cr)
	assert.Nil(t, err)
	assert.Equal(t, 0, cr.Score)
	assert.Equal(t, map[string]bool{}, cr.Votes)

}

func TestServer_Count(t *testing.T) {
	srv, port := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(srv)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah2"}}

	addComment(t, c1, port)
	addComment(t, c1, port)
	addComment(t, c1, port)
	addComment(t, c2, port)
	addComment(t, c2, port)

	body, code := get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/count?site=radio-t&url=https://radio-t.com/blah1", port))
	assert.Equal(t, 200, code)
	j := JSON{}
	err := json.Unmarshal([]byte(body), &j)
	assert.Nil(t, err)
	assert.Equal(t, 3.0, j["count"])

	body, code = get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/count?site=radio-t&url=https://radio-t.com/blah2", port))
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(body), &j)
	assert.Nil(t, err)
	assert.Equal(t, 2.0, j["count"])
}

func TestServer_List(t *testing.T) {
	srv, port := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(srv)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah2"}}

	addComment(t, c1, port)
	addComment(t, c1, port)
	addComment(t, c1, port)
	addComment(t, c2, port)
	addComment(t, c2, port)

	body, code := get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/list?site=radio-t", port))
	assert.Equal(t, 200, code)
	pi := []store.PostInfo{}
	err := json.Unmarshal([]byte(body), &pi)
	assert.Nil(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/blah1", Count: 3}, {URL: "https://radio-t.com/blah2", Count: 2}}, pi)
}

func prep(t *testing.T) (srv *Server, port int) {
	dataStore, err := store.NewBoltDB(store.BoltSite{FileName: testDb, SiteID: "radio-t"})
	require.Nil(t, err)
	srv = &Server{
		DataService: store.Service{Interface: dataStore, EditDuration: 5 * time.Minute},
		DevMode:     true,
		Authenticator: auth.Authenticator{
			Providers:   nil,
			AvatarProxy: &avatar.Proxy{StorePath: "/tmp", RoutePath: "/api/v1/avatar"},
		},
		Exporter: &migrator.Remark{DataStore: dataStore},
		Cache:    &mockCache{},
		Notifier: notifier.NewNoperation(),
	}
	go func() {
		port = rand.Intn(50000) + 1025
		srv.Run(port)
	}()
	time.Sleep(100 * time.Millisecond)
	return srv, port
}

func get(t *testing.T, url string) (string, int) {
	r, err := http.Get(url)
	assert.Nil(t, err)
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	assert.Nil(t, err)
	return string(body), r.StatusCode
}

func addComment(t *testing.T, c store.Comment, port int) string {

	b, err := json.Marshal(c)
	assert.Nil(t, err, "can't marshal comment %+v", c)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/v1/comment", port), "application/json", bytes.NewBuffer(b))
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

func cleanup(srv *Server) {
	srv.httpServer.Close()
	srv.httpServer.Shutdown(context.Background())
	os.Remove(testDb)
}

type mockCache struct{}

func (mc *mockCache) Get(key string, ttl time.Duration, fn func() ([]byte, error)) (data []byte, err error) {
	return fn()
}

func (mc *mockCache) Flush() {}
