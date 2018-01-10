package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/store"
)

var testDb = "/tmp/test-remark.db"

func TestServer_Ping(t *testing.T) {
	srv := prep(t)
	assert.NotNil(t, srv)
	defer func() {
		srv.httpServer.Shutdown(context.Background())
		os.Remove(testDb)
	}()

	res, code := get(t, "http://127.0.0.1:8080/api/v1/ping")
	assert.Equal(t, "pong", res)
	assert.Equal(t, 200, code)
}

func TestServer_Create(t *testing.T) {
	srv := prep(t)
	assert.NotNil(t, srv)
	defer func() {
		srv.httpServer.Shutdown(context.Background())
		os.Remove(testDb)
	}()

	r := strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`)
	resp, err := http.Post("http://127.0.0.1:8080/api/v1/comment", "application/json", r)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	b, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	c := JSON{}
	err = json.Unmarshal(b, &c)
	assert.Nil(t, err)
	loc := c["loc"].(map[string]interface{})
	assert.Equal(t, "radio-t", loc["site"])
	assert.Equal(t, "https://radio-t.com/blah1", loc["url"])
	assert.True(t, len(c["id"].(string)) > 8)
}

func TestServer_CreateAndGet(t *testing.T) {
	srv := prep(t)
	assert.NotNil(t, srv)
	defer func() {
		srv.httpServer.Shutdown(context.Background())
		os.Remove(testDb)
	}()

	// create comment
	r := strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`)
	resp, err := http.Post("http://127.0.0.1:8080/api/v1/comment", "application/json", r)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	c := JSON{}
	err = json.Unmarshal(b, &c)
	assert.Nil(t, err)

	id := c["id"].(string)

	// get created comment by id
	res, code := get(t, fmt.Sprintf("http://127.0.0.1:8080/api/v1/id/%s?site=radio-t&url=https://radio-t.com/blah1", id))
	assert.Equal(t, 200, code)
	comment := store.Comment{}
	err = json.Unmarshal([]byte(res), &comment)
	assert.Nil(t, err)
	assert.Equal(t, "test 123", comment.Text)
	assert.Equal(t, store.User{Name: "developer one", ID: "dev",
		Picture: "https://friends.radio-t.com/resources/images/rt_logo_64.png",
		Profile: "https://radio-t.com/info/", Admin: true, Blocked: false, IP: ""},
		comment.User)
}

func TestServer_Find(t *testing.T) {
	srv := prep(t)
	assert.NotNil(t, srv)
	defer func() {
		srv.httpServer.Close()
		srv.httpServer.Shutdown(context.Background())
		os.Remove(testDb)
	}()
	_, code := get(t, "http://127.0.0.1:8080/api/v1/find?site=radio-t&url=https://radio-t.com/blah1")
	assert.Equal(t, 400, code, "nothing in")

	c1 := store.Comment{Text: "test test #1", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}

	id1 := addComment(t, c1)
	id2 := addComment(t, c2)
	assert.NotEqual(t, id1, id2)

	// get sorted by +time
	res, code := get(t, "http://127.0.0.1:8080/api/v1/find?site=radio-t&url=https://radio-t.com/blah1&sort=+time")
	assert.Equal(t, 200, code)
	comments := []store.Comment{}
	err := json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments), "should have 2 comments")
	assert.Equal(t, id1, comments[0].ID)
	assert.Equal(t, id2, comments[1].ID)

	// get sorted by -time
	res, code = get(t, "http://127.0.0.1:8080/api/v1/find?site=radio-t&url=https://radio-t.com/blah1&sort=-time")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments), "should have 2 comments")
	assert.Equal(t, id1, comments[1].ID)
	assert.Equal(t, id2, comments[0].ID)
}

func prep(t *testing.T) *Server {
	dataStore, err := store.NewBoltDB(store.BoltSite{FileName: testDb, SiteID: "radio-t"})
	assert.Nil(t, err)
	srv := Server{
		DataService:  store.Service{Interface: dataStore, EditDuration: 5 * time.Minute},
		DevMode:      true,
		AuthFacebook: &auth.Provider{},
		AuthGithub:   &auth.Provider{},
		AuthGoogle:   &auth.Provider{},
	}
	go func() {
		srv.Run()
	}()
	time.Sleep(100 * time.Millisecond)
	return &srv
}

func get(t *testing.T, url string) (string, int) {
	r, err := http.Get(url)
	assert.Nil(t, err)
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	assert.Nil(t, err)
	return string(body), r.StatusCode
}

func addComment(t *testing.T, c store.Comment) string {

	b, err := json.Marshal(c)
	assert.Nil(t, err, "can't marshal comment %+v", c)
	resp, err := http.Post("http://127.0.0.1:8080/api/v1/comment", "application/json", bytes.NewBuffer(b))
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	b, err = ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)

	crResp := JSON{}
	err = json.Unmarshal(b, &crResp)
	assert.Nil(t, err)

	return crResp["id"].(string)
}
