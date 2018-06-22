package api

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/store"
)

func TestAdmin_Delete(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1", User: store.User{ID: "id", Name: "name"},
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}}
	c2 := store.Comment{Text: "test test #2", User: store.User{ID: "id", Name: "name"}, ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}}

	id1 := addComment(t, c1, ts)
	addComment(t, c2, ts)

	client := http.Client{}
	req, err := http.NewRequest(http.MethodDelete,
		fmt.Sprintf("%s/api/v1/admin/comment/%s?site=radio-t&url=https://radio-t.com/blah", ts.URL, id1), nil)
	assert.Nil(t, err)
	req.SetBasicAuth("dev", "password")
	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, code := getWithAuth(t, fmt.Sprintf("%s/api/v1/id/%s?site=radio-t&url=https://radio-t.com/blah", ts.URL, id1))
	assert.Equal(t, 200, code)
	cr := store.Comment{}
	err = json.Unmarshal([]byte(body), &cr)
	assert.Nil(t, err)
	assert.Equal(t, "", cr.Text)
	assert.True(t, cr.Deleted)
}

func TestAdmin_DeleteUser(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1", Orig: "o test test #1", User: store.User{ID: "id1", Name: "name"},
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}}
	c2 := store.Comment{Text: "test test #2", Orig: "o test test #2", User: store.User{ID: "id2", Name: "name"}, ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}}
	c3 := store.Comment{Text: "test test #3", Orig: "o test test #3", User: store.User{ID: "id2", Name: "name"}, ParentID: "",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}}

	// write comments directly to store to keep user id
	id1, err := srv.DataService.Create(c1)
	assert.NoError(t, err)
	_, err = srv.DataService.Create(c2)
	assert.NoError(t, err)
	_, err = srv.DataService.Create(c3)
	assert.NoError(t, err)

	client := http.Client{}
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/v1/admin/user/%s?site=radio-t", ts.URL, "id2"), nil)
	assert.Nil(t, err)
	req.SetBasicAuth("dev", "password")
	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// all 3 comments here, but for id2 they deleted
	res, code := get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/blah&sort=+time")
	assert.Equal(t, 200, code)
	commentsWithInfo := commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &commentsWithInfo)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(commentsWithInfo.Comments), "should have 3 comment")

	// id1 comment untouched
	assert.Equal(t, id1, commentsWithInfo.Comments[0].ID)
	assert.Equal(t, "o test test #1", commentsWithInfo.Comments[0].Orig)
	assert.False(t, commentsWithInfo.Comments[0].Deleted)
	t.Logf("%+v", commentsWithInfo.Comments[0].User)

	// id2 comments fully deleted
	assert.Equal(t, "", commentsWithInfo.Comments[1].Text)
	assert.Equal(t, "", commentsWithInfo.Comments[1].Orig)
	assert.Equal(t, store.User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, commentsWithInfo.Comments[1].User)
	assert.True(t, commentsWithInfo.Comments[1].Deleted)

	assert.Equal(t, "", commentsWithInfo.Comments[2].Text)
	assert.Equal(t, "", commentsWithInfo.Comments[2].Orig)
	assert.Equal(t, store.User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, commentsWithInfo.Comments[1].User)
	assert.True(t, commentsWithInfo.Comments[2].Deleted)
}

func TestAdmin_Pin(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}}

	id1 := addComment(t, c1, ts)
	addComment(t, c2, ts)

	pin := func(val int) int {
		client := http.Client{}
		req, err := http.NewRequest(http.MethodPut,
			fmt.Sprintf("%s/api/v1/admin/pin/%s?site=radio-t&url=https://radio-t.com/blah&pin=%d", ts.URL, id1, val), nil)
		assert.Nil(t, err)
		req.SetBasicAuth("dev", "password")
		resp, err := client.Do(req)
		assert.Nil(t, err)
		return resp.StatusCode
	}

	code := pin(1)
	assert.Equal(t, 200, code)

	body, code := get(t, fmt.Sprintf("%s/api/v1/id/%s?site=radio-t&url=https://radio-t.com/blah", ts.URL, id1))
	assert.Equal(t, 200, code)
	cr := store.Comment{}
	err := json.Unmarshal([]byte(body), &cr)
	assert.Nil(t, err)
	assert.True(t, cr.Pin)

	code = pin(-1)
	assert.Equal(t, 200, code)
	body, code = get(t, fmt.Sprintf("%s/api/v1/id/%s?site=radio-t&url=https://radio-t.com/blah", ts.URL, id1))
	assert.Equal(t, 200, code)
	cr = store.Comment{}
	err = json.Unmarshal([]byte(body), &cr)
	assert.Nil(t, err)
	assert.False(t, cr.Pin)
}

func TestAdmin_Block(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2", ID: "user2"}}

	_, err := srv.DataService.Create(c1)
	assert.Nil(t, err)
	_, err = srv.DataService.Create(c2)
	assert.Nil(t, err)

	block := func(val int) (code int, body []byte) {
		client := http.Client{}
		req, e := http.NewRequest(http.MethodPut,
			fmt.Sprintf("%s/api/v1/admin/user/%s?site=radio-t&block=%d", ts.URL, "user1", val), nil)
		assert.Nil(t, e)
		req.SetBasicAuth("dev", "password")
		resp, e := client.Do(req)
		require.Nil(t, e)
		body, e = ioutil.ReadAll(resp.Body)
		assert.Nil(t, e)
		resp.Body.Close()
		return resp.StatusCode, body
	}

	code, body := block(1)
	require.Equal(t, 200, code)
	j := JSON{}
	err = json.Unmarshal(body, &j)
	assert.Nil(t, err)
	assert.Equal(t, "user1", j["user_id"])
	assert.Equal(t, true, j["block"])
	assert.Equal(t, "radio-t", j["site_id"])

	res, code := get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/blah&sort=+time")
	assert.Equal(t, 200, code)
	comments := commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments.Comments), "should have 2 comments")
	assert.Equal(t, "", comments.Comments[0].Text)
	assert.True(t, comments.Comments[0].Deleted)

	code, body = block(-1)
	require.Equal(t, 200, code)
	err = json.Unmarshal(body, &j)
	assert.Nil(t, err)
	assert.Equal(t, false, j["block"])
}

func TestAdmin_BlockedList(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	client := http.Client{}

	// block user1
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/user/%s?site=radio-t&block=%d", ts.URL, "user1", 1), nil)
	assert.Nil(t, err)
	req.SetBasicAuth("dev", "password")
	_, err = client.Do(req)
	require.Nil(t, err)

	// block user2
	req, err = http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/user/%s?site=radio-t&block=%d", ts.URL, "user2", 1), nil)
	assert.Nil(t, err)
	req.SetBasicAuth("dev", "password")
	_, err = client.Do(req)
	require.Nil(t, err)

	res, code := getWithAuth(t, ts.URL+"/api/v1/admin/blocked?site=radio-t")
	require.Equal(t, 200, code, res)
	users := []store.BlockedUser{}
	err = json.Unmarshal([]byte(res), &users)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(users), "two users blocked")
	assert.Equal(t, "user1", users[0].ID)
	assert.Equal(t, "user2", users[1].ID)
}

func TestAdmin_ReadOnly(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2", ID: "user2"}}

	_, err := srv.DataService.Create(c1)
	assert.Nil(t, err)
	_, err = srv.DataService.Create(c2)
	assert.Nil(t, err)

	info, err := srv.DataService.Info(store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}, 0)
	assert.Nil(t, err)
	assert.False(t, info.ReadOnly)

	client := http.Client{}

	// set post to read-only
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/readonly?site=radio-t&url=https://radio-t.com/blah&ro=1", ts.URL), nil)
	assert.Nil(t, err)
	req.SetBasicAuth("dev", "password")
	resp, err := client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	info, err = srv.DataService.Info(store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}, 0)
	assert.Nil(t, err)
	assert.True(t, info.ReadOnly)

	// reset post's read-only
	req, err = http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/readonly?site=radio-t&url=https://radio-t.com/blah&ro=0", ts.URL), nil)
	assert.Nil(t, err)
	req.SetBasicAuth("dev", "password")
	resp, err = client.Do(req)
	assert.Equal(t, 200, resp.StatusCode)
	require.Nil(t, err)
	info, err = srv.DataService.Info(store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}, 0)
	assert.Nil(t, err)
	assert.False(t, info.ReadOnly)
}

func TestAdmin_ReadOnlyWithAge(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"},
		Timestamp: time.Date(2001, 1, 1, 1, 1, 1, 0, time.Local)}
	_, err := srv.DataService.Create(c1)
	assert.Nil(t, err)

	info, err := srv.DataService.Info(store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}, 10)
	assert.Nil(t, err)
	assert.True(t, info.ReadOnly, "ro by age")

	client := http.Client{}

	// set post to read-only
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/readonly?site=radio-t&url=https://radio-t.com/blah&ro=1", ts.URL), nil)
	assert.Nil(t, err)
	req.SetBasicAuth("dev", "password")
	resp, err := client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	info, err = srv.DataService.Info(store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}, 0)
	assert.Nil(t, err)
	assert.True(t, info.ReadOnly)

	// reset post's read-only
	req, err = http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/readonly?site=radio-t&url=https://radio-t.com/blah&ro=0", ts.URL), nil)
	assert.Nil(t, err)
	req.SetBasicAuth("dev", "password")
	resp, err = client.Do(req)
	assert.Equal(t, 403, resp.StatusCode)
	require.Nil(t, err)
	info, err = srv.DataService.Info(store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}, 0)
	assert.Nil(t, err)
	assert.True(t, info.ReadOnly)

}
func TestAdmin_Verify(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2", ID: "user2"}}

	_, err := srv.DataService.Create(c1)
	assert.Nil(t, err)
	_, err = srv.DataService.Create(c2)
	assert.Nil(t, err)

	verified := srv.DataService.IsVerified("radio-t", "user1")
	assert.False(t, verified)

	client := http.Client{}
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/verify/user1?site=radio-t&verified=1", ts.URL), nil)
	assert.Nil(t, err)
	req.SetBasicAuth("dev", "password")
	_, err = client.Do(req)
	require.Nil(t, err)
	verified = srv.DataService.IsVerified("radio-t", "user1")
	assert.True(t, verified)

	res, code := get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/blah&sort=+time")
	assert.Equal(t, 200, code)
	comments := commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments.Comments), "should have 2 comments")
	assert.Equal(t, "test test #1", comments.Comments[0].Text)
	assert.True(t, comments.Comments[0].User.Verified)

	req, err = http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/verify/user1?site=radio-t&verified=0", ts.URL), nil)
	assert.Nil(t, err)
	req.SetBasicAuth("dev", "password")
	_, err = client.Do(req)
	require.Nil(t, err)
	verified = srv.DataService.IsVerified("radio-t", "user1")
	assert.False(t, verified)

	res, code = get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/blah&sort=+time")
	assert.Equal(t, 200, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments.Comments), "should have 2 comments")
	assert.Equal(t, "test test #1", comments.Comments[0].Text)
	assert.False(t, comments.Comments[0].User.Verified)

}

func TestAdmin_ExportStream(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah2"}}

	addComment(t, c1, ts)
	addComment(t, c2, ts)

	body, code := getWithAuth(t, ts.URL+"/api/v1/admin/export?site=radio-t&mode=stream")
	assert.Equal(t, 200, code)
	assert.Equal(t, 2, strings.Count(body, "\n"))
	assert.Equal(t, 2, strings.Count(body, "\"text\""))
	t.Logf("%s", body)
}

func TestAdmin_ExportFile(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah2"}}

	addComment(t, c1, ts)
	addComment(t, c2, ts)

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", ts.URL+"/api/v1/admin/export?site=radio-t&mode=file", nil)
	require.Nil(t, err)
	req.SetBasicAuth("dev", "password")
	resp, err := client.Do(req)
	require.Nil(t, err)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "application/gzip", resp.Header.Get("Content-Type"))

	ungzReader, err := gzip.NewReader(resp.Body)
	assert.NoError(t, err)
	ungzBody, err := ioutil.ReadAll(ungzReader)
	assert.NoError(t, err)
	assert.Equal(t, 2, strings.Count(string(ungzBody), "\n"))
	assert.Equal(t, 2, strings.Count(string(ungzBody), "\"text\""))
	t.Logf("%s", string(ungzBody))
}

func TestAdmin_DeleteMeRequest(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2", ID: "user2"}}

	_, err := srv.DataService.Create(c1)
	assert.Nil(t, err)
	_, err = srv.DataService.Create(c2)
	assert.Nil(t, err)

	comments, err := srv.DataService.User("radio-t", "user1", 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(comments), "a comment for user1")

	claims := auth.CustomClaims{
		SiteID:      "radio-t",
		SessionOnly: true,
		StandardClaims: jwt.StandardClaims{
			Id:        "1234567",
			Issuer:    "remark42",
			NotBefore: time.Now().Add(-1 * time.Minute).Unix(),
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
		},
		User: &store.User{
			ID: "user1",
		},
	}

	token, err := srv.Authenticator.JWTService.Token(&claims)
	assert.Nil(t, err)

	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, token), nil)
	assert.Nil(t, err)
	req.SetBasicAuth("dev", "password")
	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	_, err = srv.DataService.User("radio-t", "user1", 0, 0)
	assert.EqualError(t, err, "no comments for user user1 in store")
}

func TestAdmin_DeleteMeRequestFailed(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2", ID: "user2"}}

	_, err := srv.DataService.Create(c1)
	assert.Nil(t, err)
	_, err = srv.DataService.Create(c2)
	assert.Nil(t, err)

	// try with bad token
	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, "bad token"), nil)
	assert.Nil(t, err)
	req.SetBasicAuth("dev", "password")
	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	// try with bad auth
	claims := auth.CustomClaims{
		SiteID:      "radio-t",
		SessionOnly: true,
		StandardClaims: jwt.StandardClaims{
			Id:        "1234567",
			Issuer:    "remark42",
			NotBefore: time.Now().Add(-1 * time.Minute).Unix(),
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
		},
		User: &store.User{
			ID: "user1",
		},
	}

	token, err := srv.Authenticator.JWTService.Token(&claims)
	assert.Nil(t, err)
	req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, token), nil)
	assert.Nil(t, err)
	req.SetBasicAuth("dev", "bad-password")
	resp, err = client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, 401, resp.StatusCode)

	// try bad user
	badClaims := claims
	badClaims.User.ID = "no-such-id"
	token, err = srv.Authenticator.JWTService.Token(&badClaims)
	assert.Nil(t, err)
	req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, token), nil)
	assert.Nil(t, err)
	req.SetBasicAuth("dev", "password")
	resp, err = client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, 400, resp.StatusCode, resp.Status)
}

func TestAdmin_GetUserInfo(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2", ID: "user2"}}

	_, err := srv.DataService.Create(c1)
	assert.Nil(t, err)
	_, err = srv.DataService.Create(c2)
	assert.Nil(t, err)

	body, code := getWithAuth(t, fmt.Sprintf("%s/api/v1/admin/user/user1?site=radio-t&url=https://radio-t.com/blah", ts.URL))
	assert.Equal(t, 200, code)
	u := store.User{}
	err = json.Unmarshal([]byte(body), &u)
	assert.Nil(t, err)
	assert.Equal(t, store.User{Name: "user1 name", ID: "user1", Picture: "", IP: "", Admin: false, Blocked: false, Verified: false}, u)

	_, code = get(t, fmt.Sprintf("%s/api/v1/admin/user/user1?site=radio-t&url=https://radio-t.com/blah", ts.URL))
	assert.Equal(t, 401, code, "no auth")

	_, code = getWithAuth(t, fmt.Sprintf("%s/api/v1/admin/user/userX?site=radio-t&url=https://radio-t.com/blah", ts.URL))
	assert.Equal(t, 400, code, "no info about user")
}
