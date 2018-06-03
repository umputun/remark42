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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/app/store"
)

func TestRest_Create(t *testing.T) {
	srv, ts := prep(t)
	require.NotNil(t, srv)
	defer cleanup(ts)

	resp, err := post(t, ts.URL+"/api/v1/comment",
		`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`)
	assert.Nil(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

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

func TestRest_CreateOldPost(t *testing.T) {
	srv, ts := prep(t)
	require.NotNil(t, srv)
	defer cleanup(ts)

	// make old, but not too old comment
	old := store.Comment{Text: "test test old", ParentID: "", Timestamp: time.Now().AddDate(0, 0, -5),
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}, User: store.User{ID: "u1"}}
	_, err := srv.DataService.Create(old)
	assert.Nil(t, err)

	comments, err := srv.DataService.Find(store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}, "time")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(comments))

	// try to add new comment to the same old post
	resp, err := post(t, ts.URL+"/api/v1/comment",
		`{"text": "test 123", "locator":{"site": "radio-t","url": "https://radio-t.com/blah1"}}`)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	assert.Nil(t, srv.DataService.DeleteAll("radio-t"))
	// make too old comment
	old = store.Comment{Text: "test test old", ParentID: "", Timestamp: time.Now().AddDate(0, 0, -15),
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}, User: store.User{ID: "u1"}}
	_, err = srv.DataService.Create(old)
	assert.Nil(t, err)

	resp, err = post(t, ts.URL+"/api/v1/comment",
		`{"text": "test 123", "locator":{"site": "radio-t","url": "https://radio-t.com/blah1"}}`)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestRest_CreateTooBig(t *testing.T) {
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
func TestRest_CreateAndGet(t *testing.T) {
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

func TestRest_Update(t *testing.T) {
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

func TestRest_UpdateNotOwner(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	c1 := store.Comment{Text: "test test #1", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}, User: store.User{ID: "xyz"}}
	id1, err := srv.DataService.Create(c1)
	assert.Nil(t, err)

	client := http.Client{}
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/comment/"+id1+
		"?site=radio-t&url=https://radio-t.com/blah1", strings.NewReader(`{"text":"updated text", "summary":"my edit"}`))
	assert.Nil(t, err)
	req = withBasicAuth(req, "dev", "password")
	b, err := client.Do(req)
	assert.Nil(t, err)
	body, err := ioutil.ReadAll(b.Body)
	assert.Nil(t, err)
	assert.Equal(t, 403, b.StatusCode, string(body), "update from non-owner")
	assert.Equal(t, `{"details":"can not edit comments for other users","error":"rejected"}`+"\n", string(body))

	client = http.Client{}
	req, err = http.NewRequest(http.MethodPut, ts.URL+"/api/v1/comment/"+id1+
		"?site=radio-t&url=https://radio-t.com/blah1", strings.NewReader(`ERRR "text":"updated text", "summary":"my"}`))
	assert.Nil(t, err)
	req = withBasicAuth(req, "dev", "password")
	b, err = client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, 400, b.StatusCode, string(body), "update is not json")
}

func TestRest_Vote(t *testing.T) {
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

func TestRest_UserAllData(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	// write 3 comments
	user := store.User{ID: "dev", Name: "user name 1"}
	c1 := store.Comment{User: user, Text: "test test #1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 05, 27, 1, 14, 10, 0, time.Local)}
	c2 := store.Comment{User: user, Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 05, 27, 1, 14, 20, 0, time.Local)}
	c3 := store.Comment{User: user, Text: "test test #3", ParentID: "p1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 05, 27, 1, 14, 25, 0, time.Local)}
	_, err := srv.DataService.Create(c1)
	require.Nil(t, err, "%+v", err)
	_, err = srv.DataService.Create(c2)
	require.Nil(t, err)
	_, err = srv.DataService.Create(c3)
	require.Nil(t, err)

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("GET", ts.URL+"/api/v1/userdata?site=radio-t", nil)
	require.Nil(t, err)
	req = withBasicAuth(req, "dev", "password")
	resp, err := client.Do(req)
	require.Nil(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "application/gzip", resp.Header.Get("Content-Type"))

	ungzReader, err := gzip.NewReader(resp.Body)
	assert.NoError(t, err)
	ungzBody, err := ioutil.ReadAll(ungzReader)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(ungzBody),
		`{"name":"developer one","id":"dev","picture":"/api/v1/avatar/remark.image","admin":true}[`))
	assert.Equal(t, 3, strings.Count(string(ungzBody), `"text":`), "3 comments inside")
	t.Logf("%s", string(ungzBody))

	req, err = http.NewRequest("GET", ts.URL+"/api/v1/userdata?site=radio-t", nil)
	require.Nil(t, err)
	resp, err = client.Do(req)
	require.Nil(t, err)
	require.Equal(t, 401, resp.StatusCode)
}
func TestRest_UserAllDataManyComments(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	user := store.User{ID: "dev", Name: "user name 1"}
	c := store.Comment{User: user, Text: "test test #1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 05, 27, 1, 14, 10, 0, time.Local)}

	for i := 0; i < 478; i++ {
		c.ID = fmt.Sprintf("id-%03d", i)
		c.Timestamp = c.Timestamp.Add(time.Second)
		_, err := srv.DataService.Create(c)
		require.Nil(t, err)
	}

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("GET", ts.URL+"/api/v1/userdata?site=radio-t", nil)
	require.Nil(t, err)
	req = withBasicAuth(req, "dev", "password")
	resp, err := client.Do(req)
	require.Nil(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "application/gzip", resp.Header.Get("Content-Type"))

	ungzReader, err := gzip.NewReader(resp.Body)
	assert.NoError(t, err)
	ungzBody, err := ioutil.ReadAll(ungzReader)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(ungzBody),
		`{"name":"developer one","id":"dev","picture":"/api/v1/avatar/remark.image","admin":true}[`))
	assert.Equal(t, 478, strings.Count(string(ungzBody), `"text":`), "478 comments inside")
}

func TestRest_DeleteMe(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	client := http.Client{}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/deleteme?site=radio-t", ts.URL), nil)
	assert.Nil(t, err)
	req = withBasicAuth(req, "dev", "password")
	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)

	m := map[string]string{}
	err = json.Unmarshal(body, &m)
	assert.Nil(t, err)
	assert.Equal(t, "radio-t", m["site"])
	assert.Equal(t, "dev", m["user_id"])

	token := m["token"]
	claims, err := srv.Authenticator.JWTService.Parse(token)
	assert.Nil(t, err)
	assert.Equal(t, "dev", claims.User.ID)

	req, err = http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/deleteme?site=radio-t", ts.URL), nil)
	assert.Nil(t, err)
	resp, err = client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, 401, resp.StatusCode)
}
