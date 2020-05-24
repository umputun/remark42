package api

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-pkgz/auth/token"
	cache "github.com/go-pkgz/lcw"
	R "github.com/go-pkgz/rest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/service"
)

func TestAdmin_Delete(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1", User: store.User{ID: "id", Name: "name"},
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}
	c2 := store.Comment{Text: "test test #2", User: store.User{ID: "id", Name: "name"}, ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}

	id1 := addComment(t, c1, ts)
	addComment(t, c2, ts)

	// check last comments
	res, code := get(t, ts.URL+"/api/v1/last/2?site=remark42")
	assert.Equal(t, 200, code)
	comments := []store.Comment{}
	err := json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(comments), "should have 2 comments")

	// check multi count
	resp, err := post(t, ts.URL+"/api/v1/counts?site=remark42", `["https://radio-t.com/blah","https://radio-t.com/blah2"]`)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	bb, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, resp.Body.Close())
	assert.NoError(t, err)
	j := []store.PostInfo{}
	err = json.Unmarshal(bb, &j)
	assert.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/blah", Count: 2},
		{URL: "https://radio-t.com/blah2", Count: 0}}, j)

	// delete a comment
	req, err := http.NewRequest(http.MethodDelete,
		fmt.Sprintf("%s/api/v1/admin/comment/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1), nil)
	require.NoError(t, err)
	requireAdminOnly(t, req)
	resp, err = sendReq(t, req, adminUmputunToken)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, 200, resp.StatusCode)

	body, code := getWithDevAuth(t, fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1))
	assert.Equal(t, 200, code)
	cr := store.Comment{}
	err = json.Unmarshal([]byte(body), &cr)
	assert.NoError(t, err)
	assert.Equal(t, "", cr.Text)
	assert.True(t, cr.Deleted)

	time.Sleep(250 * time.Millisecond)
	// check last comments updated
	res, code = get(t, ts.URL+"/api/v1/last/2?site=remark42")
	assert.Equal(t, 200, code)
	comments = []store.Comment{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(comments), "should have 1 comments")

	// check count updated
	res, code = get(t, ts.URL+"/api/v1/count?site=remark42&url=https://radio-t.com/blah")
	assert.Equal(t, 200, code)
	b := map[string]interface{}{}
	err = json.Unmarshal([]byte(res), &b)
	assert.NoError(t, err)
	t.Logf("%#v", b)
	assert.Equal(t, 1.0, b["count"], "should report 1 comments")

	// check multi count updated
	resp, err = post(t, ts.URL+"/api/v1/counts?site=remark42", `["https://radio-t.com/blah","https://radio-t.com/blah2"]`)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	bb, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, resp.Body.Close())
	assert.NoError(t, err)
	j = []store.PostInfo{}
	err = json.Unmarshal(bb, &j)
	assert.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/blah", Count: 1},
		{URL: "https://radio-t.com/blah2", Count: 0}}, j)
}

func TestAdmin_Title(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	srv.DataService.TitleExtractor = service.NewTitleExtractor(http.Client{Timeout: time.Second})
	tss := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == "/post1" {
			_, err := w.Write([]byte("<html><title>post1 blah 123</title><body> 2222</body></html>"))
			assert.NoError(t, err)
			return
		}
		if r.URL.String() == "/post2" {
			_, err := w.Write([]byte("<html><title>post2 blah 123</title><body> 2222</body></html>"))
			assert.NoError(t, err)
			return
		}
		w.WriteHeader(404)
	}))
	defer tss.Close()

	c1 := store.Comment{Text: "test test #1", User: store.User{ID: "id", Name: "name"},
		Locator: store.Locator{SiteID: "remark42", URL: tss.URL + "/post1"}}
	c2 := store.Comment{Text: "test test #2", User: store.User{ID: "id", Name: "name"}, ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: tss.URL + "/post2"}}

	id1 := addComment(t, c1, ts)
	addComment(t, c2, ts)

	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/title/%s?site=remark42&url=%s/post1", ts.URL, id1, tss.URL), nil)
	assert.NoError(t, err)
	requireAdminOnly(t, req)
	resp, err := sendReq(t, req, adminUmputunToken)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 200, resp.StatusCode)

	body, code := get(t, fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=%s/post1", ts.URL, id1, tss.URL))
	require.Equal(t, 200, code)
	cr := store.Comment{}
	err = json.Unmarshal([]byte(body), &cr)
	assert.NoError(t, err)
	assert.Equal(t, "post1 blah 123", cr.PostTitle)
}

func TestAdmin_DeleteUser(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1", Orig: "o test test #1", User: store.User{ID: "id1", Name: "name"},
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}
	c2 := store.Comment{Text: "test test #2", Orig: "o test test #2", User: store.User{ID: "id2", Name: "name"}, ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}
	c3 := store.Comment{Text: "test test #3", Orig: "o test test #3", User: store.User{ID: "id2", Name: "name"}, ParentID: "",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}

	// write comments directly to store to keep user id
	id1, err := srv.DataService.Create(c1)
	assert.NoError(t, err)
	_, err = srv.DataService.Create(c2)
	assert.NoError(t, err)
	_, err = srv.DataService.Create(c3)
	assert.NoError(t, err)

	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/v1/admin/user/%s?site=remark42", ts.URL, "id2"), nil)
	assert.NoError(t, err)
	requireAdminOnly(t, req)
	resp, err := sendReq(t, req, adminUmputunToken)
	assert.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 200, resp.StatusCode)

	// all 3 comments here, but for id2 they deleted
	res, code := get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah&sort=+time")
	assert.Equal(t, 200, code)
	cmntWithInfo := commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &cmntWithInfo)
	assert.NoError(t, err)
	require.Equal(t, 3, len(cmntWithInfo.Comments), "should have 3 comment")

	// id1 comment untouched
	assert.Equal(t, id1, cmntWithInfo.Comments[0].ID)
	assert.Equal(t, "o test test #1", cmntWithInfo.Comments[0].Orig)
	assert.False(t, cmntWithInfo.Comments[0].Deleted)
	t.Logf("%+v", cmntWithInfo.Comments[0].User)

	// id2 comments fully deleted
	assert.Equal(t, "", cmntWithInfo.Comments[1].Text)
	assert.Equal(t, "", cmntWithInfo.Comments[1].Orig)
	assert.Equal(t, store.User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, cmntWithInfo.Comments[1].User)
	assert.True(t, cmntWithInfo.Comments[1].Deleted)

	assert.Equal(t, "", cmntWithInfo.Comments[2].Text)
	assert.Equal(t, "", cmntWithInfo.Comments[2].Orig)
	assert.Equal(t, store.User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, cmntWithInfo.Comments[1].User)
	assert.True(t, cmntWithInfo.Comments[2].Deleted)
}

func TestAdmin_Pin(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}

	id1 := addComment(t, c1, ts)
	addComment(t, c2, ts)

	pin := func(val int) int {
		client := http.Client{}
		req, err := http.NewRequest(http.MethodPut,
			fmt.Sprintf("%s/api/v1/admin/pin/%s?site=remark42&url=https://radio-t.com/blah&pin=%d", ts.URL, id1, val), nil)
		assert.NoError(t, err)
		requireAdminOnly(t, req)
		req.SetBasicAuth("admin", "password")
		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.NoError(t, resp.Body.Close())
		return resp.StatusCode
	}

	code := pin(1)
	assert.Equal(t, 200, code)

	body, code := get(t, fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1))
	assert.Equal(t, 200, code)
	cr := store.Comment{}
	err := json.Unmarshal([]byte(body), &cr)
	assert.NoError(t, err)
	assert.True(t, cr.Pin)

	code = pin(-1)
	assert.Equal(t, 200, code)
	body, code = get(t, fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1))
	assert.Equal(t, 200, code)
	cr = store.Comment{}
	err = json.Unmarshal([]byte(body), &cr)
	assert.NoError(t, err)
	assert.False(t, cr.Pin)
}

func TestAdmin_Block(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	makeTwoComments := func() {
		c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
			URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
		c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
			URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2", ID: "user2"}}

		_, err := srv.DataService.Create(c1)
		require.NoError(t, err)
		_, err = srv.DataService.Create(c2)
		require.NoError(t, err)
	}

	block := func(val int, ttl string) (code int, body []byte) {
		url := fmt.Sprintf("%s/api/v1/admin/user/%s?site=remark42&block=%d", ts.URL, "user1", val)
		if ttl != "" {
			url = url + "&ttl=" + ttl
		}
		req, err := http.NewRequest(http.MethodPut, url, nil)
		assert.NoError(t, err)
		requireAdminOnly(t, req)
		resp, err := sendReq(t, req, adminUmputunToken)
		require.NoError(t, err)
		body, err = ioutil.ReadAll(resp.Body)
		assert.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		return resp.StatusCode, body
	}

	makeTwoComments()

	// block permanently
	code, body := block(1, "")
	require.Equal(t, 200, code)
	j := R.JSON{}
	err := json.Unmarshal(body, &j)
	assert.NoError(t, err)
	assert.Equal(t, "user1", j["user_id"])
	assert.Equal(t, true, j["block"])
	assert.Equal(t, "remark42", j["site_id"])

	assert.True(t, srv.adminRest.dataService.IsBlocked("remark42", "user1"))
	assert.False(t, srv.adminRest.dataService.IsBlocked("remark42", "user2"))

	// get last to confirm one comment deleted
	bodyStr, code := get(t, ts.URL+"/api/v1/last/10?site=remark42")
	assert.Equal(t, 200, code)
	pi := []store.PostInfo{}
	assert.NoError(t, json.Unmarshal([]byte(bodyStr), &pi))
	assert.Equal(t, 1, len(pi), "last status updated, one comment left")

	// check if count call has one comment left
	resp, err := post(t, ts.URL+"/api/v1/counts?site=remark42", `["https://radio-t.com/blah"]`)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	pi = []store.PostInfo{}
	err = json.Unmarshal(body, &pi)
	assert.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/blah", Count: 1}}, pi)

	res, code := get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah&sort=+time")
	assert.Equal(t, 200, code)
	comments := commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	require.Equal(t, 2, len(comments.Comments), "should have 2 comments")
	assert.Equal(t, "", comments.Comments[0].Text, "permanent block clear comment")
	assert.True(t, comments.Comments[0].Deleted, "permanent block set deleted comment's status")

	// unblock
	code, body = block(-1, "")
	require.Equal(t, 200, code)
	err = json.Unmarshal(body, &j)
	assert.NoError(t, err)
	assert.Equal(t, false, j["block"])

	// block with ttl
	makeTwoComments()
	code, _ = block(1, "50ms")
	require.Equal(t, 200, code)

	// get as regular user
	res, code = get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah&sort=+time")
	assert.Equal(t, 200, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	require.Equal(t, 4, len(comments.Comments), "should have 4 comments")
	assert.Equal(t, "test test #1", comments.Comments[2].Text, "comment not removed and not cleared")
	assert.False(t, comments.Comments[2].Deleted, "not deleted")

	srv.pubRest.cache = cache.NewScache(cache.NewNopCache()) // TODO: with lru cache it won't be refreshed and invalidated for long
	// time
	time.Sleep(50 * time.Millisecond)
	res, code = get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah&sort=+time")
	assert.Equal(t, 200, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	require.Equal(t, 4, len(comments.Comments), "should have 4 comments")
	assert.Equal(t, "test test #1", comments.Comments[2].Text, "restored")
	assert.False(t, comments.Comments[2].Deleted)

	assert.False(t, srv.adminRest.dataService.IsBlocked("remark42", "user1"))
	assert.False(t, srv.adminRest.dataService.IsBlocked("remark42", "user2"))
}

func TestAdmin_BlockedList(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2 name", ID: "user2"}}

	// write comments for user1 and user2
	_, err := srv.DataService.Create(c1)
	assert.NoError(t, err)
	_, err = srv.DataService.Create(c2)
	assert.NoError(t, err)

	// block user1
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/user/%s?site=remark42&block=%d", ts.URL, "user1", 1), nil)
	assert.NoError(t, err)
	res, err := sendReq(t, req, adminUmputunToken)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	assert.Equal(t, 200, res.StatusCode)

	// block user2
	req, err = http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/user/%s?site=remark42&block=%d&ttl=150ms", ts.URL, "user2", 1), nil)
	assert.NoError(t, err)
	res, err = sendReq(t, req, adminUmputunToken)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	assert.Equal(t, 200, res.StatusCode)

	req, err = http.NewRequest("GET", ts.URL+"/api/v1/admin/blocked?site=remark42", nil)
	require.NoError(t, err)
	res, err = sendReq(t, req, adminUmputunToken)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
	users := []store.BlockedUser{}
	err = json.NewDecoder(res.Body).Decode(&users)
	assert.NoError(t, err)
	require.NoError(t, res.Body.Close())
	require.Equal(t, 2, len(users), "two users blocked")
	assert.Equal(t, "user1", users[0].ID)
	assert.Equal(t, "user1 name", users[0].Name)
	assert.Equal(t, "user2", users[1].ID)
	assert.Equal(t, "user2 name", users[1].Name)
	t.Logf("%+v", users)
	time.Sleep(150 * time.Millisecond)

	req, err = http.NewRequest("GET", ts.URL+"/api/v1/admin/blocked?site=remark42", nil)
	require.NoError(t, err)
	res, err = sendReq(t, req, adminUmputunToken)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
	users = []store.BlockedUser{}
	err = json.NewDecoder(res.Body).Decode(&users)
	assert.NoError(t, err)
	require.NoError(t, res.Body.Close())
	assert.Equal(t, 1, len(users), "one user left blocked")
}

func TestAdmin_ReadOnly(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2", ID: "user2"}}

	_, err := srv.DataService.Create(c1)
	assert.NoError(t, err)
	_, err = srv.DataService.Create(c2)
	assert.NoError(t, err)

	info, err := srv.DataService.Info(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}, 0)
	assert.NoError(t, err)
	assert.False(t, info.ReadOnly)

	// set post to read-only
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/readonly?site=remark42&url=https://radio-t.com/blah&ro=1", ts.URL), nil)
	assert.NoError(t, err)
	resp, err := sendReq(t, req, "") // non-admin user
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 401, resp.StatusCode)
	resp, err = sendReq(t, req, adminUmputunToken)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 200, resp.StatusCode)
	info, err = srv.DataService.Info(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}, 0)
	assert.NoError(t, err)
	assert.True(t, info.ReadOnly)

	// try to write comment
	c := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}
	b, err := json.Marshal(c)
	assert.NoError(t, err, "can't marshal comment %+v", c)
	req, err = http.NewRequest("POST", ts.URL+"/api/v1/comment", bytes.NewBuffer(b))
	require.NoError(t, err)
	resp, err = sendReq(t, req, adminUmputunToken)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// reset post's read-only
	req, err = http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/readonly?site=remark42&url=https://radio-t.com/blah&ro=0", ts.URL), nil)
	assert.NoError(t, err)
	resp, err = sendReq(t, req, adminUmputunToken)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 200, resp.StatusCode)
	info, err = srv.DataService.Info(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}, 0)
	assert.NoError(t, err)
	assert.False(t, info.ReadOnly)

	// try to write comment
	c = store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}
	b, err = json.Marshal(c)
	assert.NoError(t, err, "can't marshal comment %+v", c)
	req, err = http.NewRequest("POST", ts.URL+"/api/v1/comment", bytes.NewBuffer(b))
	require.NoError(t, err)
	resp, err = sendReq(t, req, adminUmputunToken)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestAdmin_ReadOnlyNoComments(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	// set post to read-only
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/readonly?site=remark42&url=https://radio-t.com/blah&ro=1", ts.URL), nil)
	assert.NoError(t, err)
	requireAdminOnly(t, req)
	resp, err := sendReq(t, req, adminUmputunToken)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 200, resp.StatusCode)
	_, err = srv.DataService.Info(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}, 0)
	assert.Error(t, err)

	res, code := get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah&format=tree")
	assert.Equal(t, 200, code)
	comments := commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(comments.Comments), "should have 0 comments")
	assert.True(t, comments.Info.ReadOnly)
	t.Logf("%+v", comments)
}

func TestAdmin_ReadOnlyWithAge(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"},
		Timestamp: time.Date(2001, 1, 1, 1, 1, 1, 0, time.Local)}
	_, err := srv.DataService.Create(c1)
	assert.NoError(t, err)

	info, err := srv.DataService.Info(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}, 10)
	assert.NoError(t, err)
	assert.True(t, info.ReadOnly, "ro by age")

	// set post to read-only
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/readonly?site=remark42&url=https://radio-t.com/blah&ro=1", ts.URL), nil)
	assert.NoError(t, err)
	requireAdminOnly(t, req)
	resp, err := sendReq(t, req, adminUmputunToken)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 200, resp.StatusCode)
	info, err = srv.DataService.Info(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}, 0)
	assert.NoError(t, err)
	assert.True(t, info.ReadOnly)

	// reset post's read-only
	req, err = http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/readonly?site=remark42&url=https://radio-t.com/blah&ro=0", ts.URL), nil)
	assert.NoError(t, err)
	resp, err = sendReq(t, req, adminUmputunToken)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 403, resp.StatusCode)
	info, err = srv.DataService.Info(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}, 0)
	assert.NoError(t, err)
	assert.True(t, info.ReadOnly)

}
func TestAdmin_Verify(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2", ID: "user2"}}

	_, err := srv.DataService.Create(c1)
	assert.NoError(t, err)
	_, err = srv.DataService.Create(c2)
	assert.NoError(t, err)

	verified := srv.DataService.IsVerified("remark42", "user1")
	assert.False(t, verified)

	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/verify/user1?site=remark42&verified=1", ts.URL), nil)
	assert.NoError(t, err)
	requireAdminOnly(t, req)
	resp, err := sendReq(t, req, adminUmputunToken)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 200, resp.StatusCode)
	verified = srv.DataService.IsVerified("remark42", "user1")
	assert.True(t, verified)

	res, code := get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah&sort=+time")
	assert.Equal(t, 200, code)
	comments := commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	require.Equal(t, 2, len(comments.Comments), "should have 2 comments")
	assert.Equal(t, "test test #1", comments.Comments[0].Text)
	assert.True(t, comments.Comments[0].User.Verified)

	req, err = http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/verify/user1?site=remark42&verified=0", ts.URL), nil)
	assert.NoError(t, err)
	resp, err = sendReq(t, req, adminUmputunToken)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 200, resp.StatusCode)
	verified = srv.DataService.IsVerified("remark42", "user1")
	assert.False(t, verified)

	res, code = get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah&sort=+time")
	assert.Equal(t, 200, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	require.Equal(t, 2, len(comments.Comments), "should have 2 comments")
	assert.Equal(t, "test test #1", comments.Comments[0].Text)
	assert.False(t, comments.Comments[0].User.Verified)
}

func TestAdmin_ExportStream(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah2"}}

	addComment(t, c1, ts)
	addComment(t, c2, ts)

	body, code := getWithAdminAuth(t, ts.URL+"/api/v1/admin/export?site=remark42&mode=stream")
	assert.Equal(t, 200, code)
	assert.Equal(t, 3, strings.Count(body, "\n"))
	assert.Equal(t, 2, strings.Count(body, "\"text\""))
	t.Logf("%s", body)
}

func TestAdmin_ExportFile(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah2"}}

	addComment(t, c1, ts)
	addComment(t, c2, ts)

	req, err := http.NewRequest("GET", ts.URL+"/api/v1/admin/export?site=remark42&mode=file", nil)
	require.NoError(t, err)
	requireAdminOnly(t, req)
	resp, err := sendReq(t, req, adminUmputunToken)
	require.NoError(t, err)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "application/gzip", resp.Header.Get("Content-Type"))

	ungzReader, err := gzip.NewReader(resp.Body)
	assert.NoError(t, err)
	ungzBody, err := ioutil.ReadAll(ungzReader)
	assert.NoError(t, err)
	assert.Equal(t, 3, strings.Count(string(ungzBody), "\n"))
	assert.Equal(t, 2, strings.Count(string(ungzBody), "\"text\""))
	t.Logf("%s", string(ungzBody))
}

func TestAdmin_DeleteMeRequest(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2", ID: "user2"}}

	_, err := srv.DataService.Create(c1)
	assert.NoError(t, err)
	_, err = srv.DataService.Create(c2)
	assert.NoError(t, err)

	comments, err := srv.DataService.User("remark42", "user1", 0, 0, store.User{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(comments), "a comment for user1")

	email, err := srv.DataService.SetUserEmail("remark42", "user1", "test@example.org")
	assert.NoError(t, err)
	assert.Equal(t, "test@example.org", email, "new email for user1")

	email, err = srv.DataService.GetUserEmail("remark42", "user1")
	assert.NoError(t, err)
	assert.Equal(t, "test@example.org", email, "new email for user1 is readable")

	claims := token.Claims{
		SessionOnly: true,
		StandardClaims: jwt.StandardClaims{
			Audience:  "remark42",
			Id:        "1234567",
			Issuer:    "remark42",
			NotBefore: time.Now().Add(-1 * time.Minute).Unix(),
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
		},
		User: &token.User{
			ID:      "user1",
			Picture: "pic.image",
			Attributes: map[string]interface{}{
				"delete_me": true,
			},
		},
	}

	require.NoError(t, os.MkdirAll(os.TempDir()+"/ava-remark42/42", 0700))
	require.NoError(t, ioutil.WriteFile(os.TempDir()+"/ava-remark42/42/pic.image", []byte("some image data"), 0600))

	tkn, err := srv.Authenticator.TokenService().Token(claims)
	assert.NoError(t, err)

	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, tkn), nil)
	assert.NoError(t, err)

	req.SetBasicAuth("admin", "password")
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 200, resp.StatusCode)

	_, err = srv.DataService.User("remark42", "user1", 0, 0, store.User{})
	assert.EqualError(t, err, "no comments for user user1 in store")

	email, err = srv.DataService.GetUserEmail("remark42", "user1")
	assert.NoError(t, err)
	assert.Empty(t, email, "user1 email was deleted")

}

func TestAdmin_DeleteMeRequestFailed(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2", ID: "user2"}}

	_, err := srv.DataService.Create(c1)
	assert.NoError(t, err)
	_, err = srv.DataService.Create(c2)
	assert.NoError(t, err)

	// try with bad token
	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, "bad token"), nil)
	assert.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, 400, resp.StatusCode)

	// try with bad auth
	claims := token.Claims{
		SessionOnly: true,
		StandardClaims: jwt.StandardClaims{
			Audience:  "remark42",
			Id:        "1234567",
			Issuer:    "remark42",
			NotBefore: time.Now().Add(-1 * time.Minute).Unix(),
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
		},
		User: &token.User{
			ID: "user1",
			Attributes: map[string]interface{}{
				"delete_me": true,
			},
		},
	}

	tkn, err := srv.Authenticator.TokenService().Token(claims)
	assert.NoError(t, err)
	req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, tkn), nil)
	assert.NoError(t, err)
	req.SetBasicAuth("admin", "bad-password")
	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, 403, resp.StatusCode)

	// try bad user
	badClaims := claims
	badClaims.User.ID = "no-such-id"
	tkn, err = srv.Authenticator.TokenService().Token(badClaims)
	assert.NoError(t, err)
	req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, tkn), nil)
	assert.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, 400, resp.StatusCode, resp.Status)

	// try without deleteme flag
	badClaims2 := claims
	badClaims2.User.SetBoolAttr("delete_me", false)
	tkn, err = srv.Authenticator.TokenService().Token(badClaims2)
	assert.NoError(t, err)
	req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, tkn), nil)
	assert.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.True(t, strings.Contains(string(b), "can't use provided token"))
}

func TestAdmin_GetUserInfo(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2", ID: "user2"}}

	_, err := srv.DataService.Create(c1)
	assert.NoError(t, err)
	_, err = srv.DataService.Create(c2)
	assert.NoError(t, err)

	body, code := getWithAdminAuth(t, fmt.Sprintf("%s/api/v1/admin/user/user1?site=remark42&url=https://radio-t.com/blah",
		ts.URL))
	assert.Equal(t, 200, code)
	u := store.User{}
	err = json.Unmarshal([]byte(body), &u)
	assert.NoError(t, err)
	assert.Equal(t, store.User{Name: "user1 name", ID: "user1", Picture: "", IP: "823688dafca7393d24c871a2da98a84d8732e927",
		Admin: false, Blocked: false, Verified: false}, u)

	_, code = get(t, fmt.Sprintf("%s/api/v1/admin/user/user1?site=remark42&url=https://radio-t.com/blah", ts.URL))
	assert.Equal(t, 401, code, "no auth")

	_, code = getWithAdminAuth(t, fmt.Sprintf("%s/api/v1/admin/user/userX?site=remark42&url=https://radio-t.com/blah", ts.URL))
	assert.Equal(t, 400, code, "no info about user")
}
