package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	R "github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/service"
)

func TestRest_Ping(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	res, code := get(t, ts.URL+"/api/v1/ping")
	assert.Equal(t, "pong", res)
	assert.Equal(t, 200, code)
}

func TestRest_Preview(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	resp, err := post(t, ts.URL+"/api/v1/preview", `{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "<p>test 123</p>\n", string(b))
}

func TestRest_PreviewWithMD(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	text := `
# h1

BKT
func TestRest_Preview(t *testing.T) {
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
	assert.Equal(t, "<h1>h1</h1>\n\n<pre><code>func TestRest_Preview(t *testing.T) {\nsrv, ts := prep(t)\n  require.NotNil(t, srv)\n}\n</code></pre>\n", string(b))
}

func TestRest_Find(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	res, code := get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/blah1")
	assert.Equal(t, 200, code)
	comments := commentsWithInfo{}
	err := json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(comments.Comments), "should have 0 comments")

	c1 := store.Comment{Text: "test test #1", ParentID: "",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	id1 := addComment(t, c1, ts)

	c2 := store.Comment{Text: "test test #2", ParentID: id1,
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	id2 := addComment(t, c2, ts)

	assert.NotEqual(t, id1, id2)

	// get sorted by +time
	res, code = get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/blah1&sort=+time")
	assert.Equal(t, 200, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments.Comments), "should have 2 comments")
	assert.Equal(t, id1, comments.Comments[0].ID)
	assert.Equal(t, id2, comments.Comments[1].ID)
	assert.Equal(t, "https://radio-t.com/blah1", comments.Info.URL)
	assert.Equal(t, 2, comments.Info.Count)
	assert.Equal(t, false, comments.Info.ReadOnly)
	assert.True(t, comments.Info.FirstTS.Before(comments.Info.LastTS))

	// get sorted by -time
	res, code = get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/blah1&sort=-time")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments.Comments), "should have 2 comments")
	assert.Equal(t, id1, comments.Comments[1].ID)
	assert.Equal(t, id2, comments.Comments[0].ID)

	// get in tree mode
	tree := service.Tree{}
	res, code = get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/blah1&format=tree")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &tree)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(tree.Nodes))
	assert.Equal(t, 1, len(tree.Nodes[0].Replies))
	assert.Equal(t, 2, tree.Info.Count)
	assert.Equal(t, "https://radio-t.com/blah1", tree.Info.URL)
	assert.False(t, tree.Info.ReadOnly, "post is fresh")
}

func TestRest_FindAge(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1", ParentID: "", Timestamp: time.Now().AddDate(0, 0, -5),
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}, User: store.User{ID: "u1"}}
	_, err := srv.DataService.Create(c1)
	require.Nil(t, err)

	c2 := store.Comment{Text: "test test #2", ParentID: "", Timestamp: time.Now().AddDate(0, 0, -15),
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah2"}, User: store.User{ID: "u1"}}
	_, err = srv.DataService.Create(c2)
	require.Nil(t, err)

	tree := service.Tree{}

	res, code := get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/blah1&format=tree")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &tree)
	assert.Nil(t, err)
	assert.Equal(t, "https://radio-t.com/blah1", tree.Info.URL)
	assert.False(t, tree.Info.ReadOnly, "post is fresh")

	res, code = get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/blah2&format=tree")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &tree)
	assert.Nil(t, err)
	assert.Equal(t, "https://radio-t.com/blah2", tree.Info.URL)
	assert.True(t, tree.Info.ReadOnly, "post is old")
}

func TestRest_FindReadOnly(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1", ParentID: "", Timestamp: time.Now().AddDate(0, 0, -1),
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}, User: store.User{ID: "u1"}}
	_, err := srv.DataService.Create(c1)

	require.Nil(t, err)

	c2 := store.Comment{Text: "test test #2", ParentID: "", Timestamp: time.Now().AddDate(0, 0, -2),
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah2"}, User: store.User{ID: "u1"}}
	_, err = srv.DataService.Create(c2)
	require.Nil(t, err)

	// set post to read-only
	client := http.Client{}
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/readonly?site=radio-t&url=https://radio-t.com/blah1&ro=1", ts.URL), nil)
	assert.Nil(t, err)
	req.SetBasicAuth("admin", "password")
	_, err = client.Do(req)
	require.Nil(t, err)

	tree := service.Tree{}
	res, code := get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/blah1&format=tree")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &tree)
	require.Nil(t, err)
	assert.Equal(t, "https://radio-t.com/blah1", tree.Info.URL)
	assert.True(t, tree.Info.ReadOnly, "post is ro")

	tree = service.Tree{}
	res, code = get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/blah2&format=tree")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &tree)
	require.Nil(t, err)
	assert.Equal(t, "https://radio-t.com/blah2", tree.Info.URL)
	assert.False(t, tree.Info.ReadOnly, "post is writable")
}

func TestRest_Last(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	res, code := get(t, ts.URL+"/api/v1/last/2?site=radio-t")
	assert.Equal(t, 200, code)
	assert.Equal(t, "[]\n", res, "empty last should return empty list")

	c1 := store.Comment{Text: "test test #1", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah2"}}

	// add 3 comments
	ts1 := time.Now().UnixNano() / 1000000
	addComment(t, c1, ts)
	id1 := addComment(t, c1, ts)
	time.Sleep(10 * time.Millisecond)
	ts2 := time.Now().UnixNano() / 1000000
	id2 := addComment(t, c2, ts)

	res, code = get(t, ts.URL+"/api/v1/last/2?site=radio-t")
	assert.Equal(t, 200, code)
	comments := []store.Comment{}
	err := json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments), "should have 2 comments")
	assert.Equal(t, id1, comments[1].ID)
	assert.Equal(t, id2, comments[0].ID)

	res, code = get(t, fmt.Sprintf("%s/api/v1/last/2?site=radio-t&since=%d", ts.URL, ts1))
	assert.Equal(t, 200, code)
	comments = []store.Comment{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments), "should have 2 comments")
	assert.Equal(t, id1, comments[1].ID)
	assert.Equal(t, id2, comments[0].ID)

	res, code = get(t, fmt.Sprintf("%s/api/v1/last/2?site=radio-t&since=%d", ts.URL, ts2))
	assert.Equal(t, 200, code)
	comments = []store.Comment{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(comments), "should have 1 comments")
	assert.Equal(t, id2, comments[0].ID)

	res, code = get(t, ts.URL+"/api/v1/last/5?site=radio-t")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(comments), "should have 3 comments")

	res, code = get(t, ts.URL+"/api/v1/last/X?site=radio-t")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(comments), "should have 3 comments")

	err = srv.DataService.Delete(store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}, id1, store.SoftDelete)
	assert.Nil(t, err)
	srv.Cache.Flush(cache.FlusherRequest{})
	res, code = get(t, ts.URL+"/api/v1/last/5?site=radio-t")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &comments)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments), "should have 2 comments")
	t.Logf("%+v", comments)
}

func TestRest_FindUserComments(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #3", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah2"}}

	// add 3 comments
	addComment(t, c1, ts)
	addComment(t, c2, ts)
	addComment(t, c2, ts)

	// add one deleted
	id := addComment(t, c2, ts)
	err := srv.DataService.Delete(c2.Locator, id, store.SoftDelete)
	assert.NoError(t, err)

	_, code := get(t, ts.URL+"/api/v1/comments?site=radio-t&user=blah")
	assert.Equal(t, 400, code, "noting for user blah")

	res, code := get(t, ts.URL+"/api/v1/comments?site=radio-t&user=dev")
	assert.Equal(t, 200, code)

	resp := struct {
		Comments []store.Comment
		Count    int
	}{}

	err = json.Unmarshal([]byte(res), &resp)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(resp.Comments), "should have 3 comments")
	assert.Equal(t, 4, resp.Count, "should have 3 count")
}

func TestRest_UserInfo(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	body, code := getWithDevAuth(t, ts.URL+"/api/v1/user?site=radio-t")
	assert.Equal(t, 200, code)
	user := store.User{}
	err := json.Unmarshal([]byte(body), &user)
	assert.Nil(t, err)
	assert.Equal(t, store.User{Name: "developer one", ID: "dev", Picture: "http://example.com/pic.png", IP: "127.0.0.1"}, user)
}

func TestRest_Count(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

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
	j := R.JSON{}
	err := json.Unmarshal([]byte(body), &j)
	assert.Nil(t, err)
	assert.Equal(t, 3.0, j["count"])

	body, code = get(t, ts.URL+"/api/v1/count?site=radio-t&url=https://radio-t.com/blah2")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(body), &j)
	assert.Nil(t, err)
	assert.Equal(t, 2.0, j["count"])
}

func TestRest_Counts(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

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

func TestRest_List(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

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
	assert.Equal(t, "https://radio-t.com/blah2", pi[0].URL)
	assert.Equal(t, 2, pi[0].Count)
	assert.Equal(t, "https://radio-t.com/blah1", pi[1].URL)
	assert.Equal(t, 3, pi[1].Count)
}

func TestRest_ListWithSkipAndLimit(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah2"}}
	c3 := store.Comment{Text: "test test #3", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah3"}}

	addComment(t, c1, ts)
	addComment(t, c1, ts)
	addComment(t, c1, ts)
	addComment(t, c2, ts)
	addComment(t, c2, ts)
	addComment(t, c3, ts)
	addComment(t, c3, ts)

	body, code := get(t, ts.URL+"/api/v1/list?site=radio-t&skip=1&limit=2")
	assert.Equal(t, 200, code)
	pi := []store.PostInfo{}
	err := json.Unmarshal([]byte(body), &pi)
	assert.Nil(t, err)
	require.Equal(t, 2, len(pi))
	assert.Equal(t, "https://radio-t.com/blah2", pi[0].URL)
	assert.Equal(t, 2, pi[0].Count)
	assert.Equal(t, "https://radio-t.com/blah1", pi[1].URL)
	assert.Equal(t, 3, pi[1].Count)
}

func TestRest_Config(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	body, code := get(t, ts.URL+"/api/v1/config?site=radio-t")
	assert.Equal(t, 200, code)
	j := R.JSON{}
	err := json.Unmarshal([]byte(body), &j)
	assert.Nil(t, err)
	assert.Equal(t, 300., j["edit_duration"])
	assert.EqualValues(t, []interface{}([]interface{}{"a1", "a2"}), j["admins"])
	assert.Equal(t, "admin@remark-42.com", j["admin_email"])
	assert.Equal(t, 4000., j["max_comment_size"])
	assert.Equal(t, -5., j["low_score"])
	assert.Equal(t, -10., j["critical_score"])
	assert.False(t, j["positive_score"].(bool))
	assert.Equal(t, 10., j["readonly_age"])
	assert.Equal(t, 10000., j["max_image_size"])
	t.Logf("%+v", j)
}

func TestRest_Info(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	srv.pubRest.readOnlyAge = 10000000 // make sure we don't hit read-only

	user := store.User{ID: "user1", Name: "user name 1"}
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

	body, code := get(t, ts.URL+"/api/v1/info?site=radio-t&url=https://radio-t.com/blah1")
	assert.Equal(t, 200, code)

	info := store.PostInfo{}
	err = json.Unmarshal([]byte(body), &info)
	assert.Nil(t, err)
	exp := store.PostInfo{URL: "https://radio-t.com/blah1", Count: 3,
		FirstTS: time.Date(2018, 05, 27, 1, 14, 10, 0, time.Local), LastTS: time.Date(2018, 05, 27, 1, 14, 25, 0, time.Local)}
	assert.Equal(t, exp, info)

	_, code = get(t, ts.URL+"/api/v1/info?site=radio-t&url=https://radio-t.com/blah-no")
	assert.Equal(t, 400, code)
	_, code = get(t, ts.URL+"/api/v1/info?site=radio-t-no&url=https://radio-t.com/blah-no")
	assert.Equal(t, 400, code)
}

func TestRest_Robots(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	body, code := get(t, ts.URL+"/robots.txt")
	assert.Equal(t, 200, code)
	assert.Equal(t, "User-agent: *\nDisallow: /auth/\nDisallow: /api/\nAllow: /api/v1/find\n"+
		"Allow: /api/v1/last\nAllow: /api/v1/id\nAllow: /api/v1/count\nAllow: /api/v1/counts\n"+
		"Allow: /api/v1/list\nAllow: /api/v1/config\nAllow: /api/v1/img\nAllow: /api/v1/avatar\nAllow: /api/v1/picture\n", string(body))
}
