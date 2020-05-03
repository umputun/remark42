package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	cache "github.com/go-pkgz/lcw"
	R "github.com/go-pkgz/rest"
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
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "<p>test 123</p>\n", string(b))

	resp, err = post(t, ts.URL+"/api/v1/preview", "bad")
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
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

	resp, err := post(t, ts.URL+"/api/v1/preview", j)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t,
		`<h1>h1</h1>
<pre class="chroma">func TestRest_Preview(t *testing.T) {
srv, ts := prep(t)
  require.NotNil(t, srv)
}
</pre>`,
		string(b))
}

func TestRest_PreviewCode(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	text := `BKTgo
func main(aa string) int {return 0}
BKT
`
	text = strings.Replace(text, "BKT", "```", -1)
	j := fmt.Sprintf(`{"text": "%s", "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`, text)
	j = strings.Replace(j, "\n", "\\n", -1)

	resp, err := post(t, ts.URL+"/api/v1/preview", j)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, `<pre class="chroma"><span class="kd">func</span> <span class="nf">main</span><span class="p">(</span><span class="nx">aa</span> <span class="kt">string</span><span class="p">)</span> <span class="kt">int</span> <span class="p">{</span><span class="k">return</span> <span class="mi">0</span><span class="p">}</span>
</pre>`, string(b))
}

func TestRest_Find(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	res, code := get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah1")
	assert.Equal(t, 200, code)
	comments := commentsWithInfo{}
	err := json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(comments.Comments), "should have 0 comments")

	c1 := store.Comment{Text: "test test #1", ParentID: "",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	id1 := addComment(t, c1, ts)

	c2 := store.Comment{Text: "test test #2", ParentID: id1,
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	id2 := addComment(t, c2, ts)

	assert.NotEqual(t, id1, id2)

	// get sorted by +time
	res, code = get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah1&sort=+time")
	assert.Equal(t, 200, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	require.Equal(t, 2, len(comments.Comments), "should have 2 comments")
	assert.Equal(t, id1, comments.Comments[0].ID)
	assert.Equal(t, id2, comments.Comments[1].ID)
	assert.Equal(t, "<p>test test #1</p>\n", comments.Comments[0].Text)
	assert.Equal(t, "<p>test test #2</p>\n", comments.Comments[1].Text)
	assert.Equal(t, "https://radio-t.com/blah1", comments.Info.URL)
	assert.Equal(t, 2, comments.Info.Count)
	assert.Equal(t, false, comments.Info.ReadOnly)
	assert.True(t, comments.Info.FirstTS.Before(comments.Info.LastTS))

	// get sorted by -time
	res, code = get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah1&sort=-time")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	require.Equal(t, 2, len(comments.Comments), "should have 2 comments")
	assert.Equal(t, id1, comments.Comments[1].ID)
	assert.Equal(t, id2, comments.Comments[0].ID)

	// get in tree mode
	tree := service.Tree{}
	res, code = get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah1&format=tree")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &tree)
	assert.NoError(t, err)
	require.Equal(t, 1, len(tree.Nodes))
	assert.Equal(t, 1, len(tree.Nodes[0].Replies))
	assert.Equal(t, 2, tree.Info.Count)
	assert.Equal(t, "https://radio-t.com/blah1", tree.Info.URL)
	assert.False(t, tree.Info.ReadOnly, "post is fresh")
}

func TestRest_FindAge(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1", ParentID: "", Timestamp: time.Now().AddDate(0, 0, -5),
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}, User: store.User{ID: "u1"}}
	_, err := srv.DataService.Create(c1)
	require.NoError(t, err)

	c2 := store.Comment{Text: "test test #2", ParentID: "", Timestamp: time.Now().AddDate(0, 0, -15),
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah2"}, User: store.User{ID: "u1"}}
	_, err = srv.DataService.Create(c2)
	require.NoError(t, err)

	tree := service.Tree{}

	res, code := get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah1&format=tree")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &tree)
	assert.NoError(t, err)
	assert.Equal(t, "https://radio-t.com/blah1", tree.Info.URL)
	assert.False(t, tree.Info.ReadOnly, "post is fresh")

	res, code = get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah2&format=tree")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &tree)
	assert.NoError(t, err)
	assert.Equal(t, "https://radio-t.com/blah2", tree.Info.URL)
	assert.True(t, tree.Info.ReadOnly, "post is old")
}

func TestRest_FindReadOnly(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1", ParentID: "", Timestamp: time.Now().AddDate(0, 0, -1),
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}, User: store.User{ID: "u1"}}
	_, err := srv.DataService.Create(c1)

	require.NoError(t, err)

	c2 := store.Comment{Text: "test test #2", ParentID: "", Timestamp: time.Now().AddDate(0, 0, -2),
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah2"}, User: store.User{ID: "u1"}}
	_, err = srv.DataService.Create(c2)
	require.NoError(t, err)

	// set post to read-only
	client := http.Client{}
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/readonly?site=remark42&url=https://radio-t.com/blah1&ro=1", ts.URL), nil)
	assert.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	_, err = client.Do(req)
	require.NoError(t, err)

	tree := service.Tree{}
	res, code := get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah1&format=tree")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &tree)
	require.NoError(t, err)
	assert.Equal(t, "https://radio-t.com/blah1", tree.Info.URL)
	assert.True(t, tree.Info.ReadOnly, "post is ro")

	tree = service.Tree{}
	res, code = get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah2&format=tree")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &tree)
	require.NoError(t, err)
	assert.Equal(t, "https://radio-t.com/blah2", tree.Info.URL)
	assert.False(t, tree.Info.ReadOnly, "post is writable")
}

func TestRest_FindUserView(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	res, code := get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah1&view=user")
	assert.Equal(t, 200, code)
	comments := commentsWithInfo{}
	err := json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(comments.Comments), "should have 0 comments")

	c1 := store.Comment{Text: "test test #1", ParentID: "",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	id1 := addComment(t, c1, ts)

	c2 := store.Comment{Text: "test test #2", ParentID: id1,
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	id2 := addComment(t, c2, ts)

	assert.NotEqual(t, id1, id2)

	// get sorted by +time with view=user
	res, code = get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah1&sort=+time&view=user")
	assert.Equal(t, 200, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	require.Equal(t, 2, len(comments.Comments), "should have 2 comments")
	assert.Equal(t, id1, comments.Comments[0].ID)
	assert.Equal(t, id2, comments.Comments[1].ID)
	assert.Equal(t, "dev", comments.Comments[0].User.ID)
	assert.Equal(t, "dev", comments.Comments[1].User.ID)
	assert.Equal(t, "", comments.Comments[0].Text)
	assert.Equal(t, "", comments.Comments[1].Text)
}

func TestRest_Last(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	res, code := get(t, ts.URL+"/api/v1/last/2?site=remark42")
	assert.Equal(t, 200, code)
	assert.Equal(t, "[]\n", res, "empty last should return empty list")

	c1 := store.Comment{Text: "test test #1", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah2"}}

	// add 3 comments
	ts1 := time.Now().UnixNano() / 1000000
	addComment(t, c1, ts)
	id1 := addComment(t, c1, ts)
	time.Sleep(10 * time.Millisecond)
	ts2 := time.Now().UnixNano() / 1000000
	id2 := addComment(t, c2, ts)

	res, code = get(t, ts.URL+"/api/v1/last/2?site=remark42")
	assert.Equal(t, 200, code)
	comments := []store.Comment{}
	err := json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	require.Equal(t, 2, len(comments), "should have 2 comments")
	assert.Equal(t, id1, comments[1].ID)
	assert.Equal(t, id2, comments[0].ID)

	res, code = get(t, fmt.Sprintf("%s/api/v1/last/2?site=remark42&since=%d", ts.URL, ts1))
	assert.Equal(t, 200, code)
	comments = []store.Comment{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	require.Equal(t, 2, len(comments), "should have 2 comments")
	assert.Equal(t, id1, comments[1].ID)
	assert.Equal(t, id2, comments[0].ID)

	res, code = get(t, fmt.Sprintf("%s/api/v1/last/2?site=remark42&since=%d", ts.URL, ts2))
	assert.Equal(t, 200, code)
	comments = []store.Comment{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	require.Equal(t, 1, len(comments), "should have 1 comments")
	assert.Equal(t, id2, comments[0].ID)

	res, code = get(t, ts.URL+"/api/v1/last/5?site=remark42")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(comments), "should have 3 comments")

	res, code = get(t, ts.URL+"/api/v1/last/X?site=remark42")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(comments), "should have 3 comments")

	err = srv.DataService.Delete(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}, id1, store.SoftDelete)
	assert.NoError(t, err)
	srv.Cache.Flush(cache.FlusherRequest{})
	res, code = get(t, ts.URL+"/api/v1/last/5?site=remark42")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(comments), "should have 2 comments")

	_, code = get(t, ts.URL+"/api/v1/last/2?site=remark42-BLAH")
	assert.Equal(t, 500, code)
}

func TestRest_FindUserComments(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #3", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah2"}}

	// add 3 comments
	addComment(t, c1, ts)
	addComment(t, c2, ts)
	addComment(t, c2, ts)

	// add one deleted
	id := addComment(t, c2, ts)
	err := srv.DataService.Delete(c2.Locator, id, store.SoftDelete)
	assert.NoError(t, err)

	_, code := get(t, ts.URL+"/api/v1/comments?site=remark42&user=blah")
	assert.Equal(t, 400, code, "noting for user blah")

	res, code := get(t, ts.URL+"/api/v1/comments?site=remark42&user=dev")
	assert.Equal(t, 200, code)

	resp := struct {
		Comments []store.Comment
		Count    int
	}{}

	err = json.Unmarshal([]byte(res), &resp)
	assert.NoError(t, err)
	require.Equal(t, 3, len(resp.Comments), "should have 3 comments")
	assert.Equal(t, 4, resp.Count, "should have 3 count")

	// user comment sorted with -time
	assert.True(t, resp.Comments[0].Timestamp.After(resp.Comments[1].Timestamp))
	assert.True(t, resp.Comments[1].Timestamp.After(resp.Comments[2].Timestamp))
}

func TestRest_UserInfo(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	body, code := getWithDevAuth(t, ts.URL+"/api/v1/user?site=remark42")
	assert.Equal(t, 200, code)
	user := store.User{}
	err := json.Unmarshal([]byte(body), &user)
	assert.NoError(t, err)
	assert.Equal(t, store.User{Name: "developer one", ID: "dev", Picture: "http://example.com/pic.png",
		IP: "127.0.0.1", SiteID: "remark42"}, user)
}

func TestRest_Count(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah2"}}

	addComment(t, c1, ts)
	addComment(t, c1, ts)
	addComment(t, c1, ts)
	addComment(t, c2, ts)
	addComment(t, c2, ts)

	body, code := get(t, ts.URL+"/api/v1/count?site=remark42&url=https://radio-t.com/blah1")
	assert.Equal(t, 200, code)
	j := R.JSON{}
	err := json.Unmarshal([]byte(body), &j)
	assert.NoError(t, err)
	assert.Equal(t, 3.0, j["count"])

	body, code = get(t, ts.URL+"/api/v1/count?site=remark42&url=https://radio-t.com/blah2")
	assert.Equal(t, 200, code)
	err = json.Unmarshal([]byte(body), &j)
	assert.NoError(t, err)
	assert.Equal(t, 2.0, j["count"])

	_, code = get(t, ts.URL+"/api/v1/count?site=remark42-BLAH&url=https://radio-t.com/blah1XXX")
	assert.Equal(t, 400, code)
}

func TestRest_Counts(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah2"}}

	addComment(t, c1, ts)
	addComment(t, c1, ts)
	addComment(t, c1, ts)
	addComment(t, c2, ts)
	addComment(t, c2, ts)

	resp, err := post(t, ts.URL+"/api/v1/counts?site=remark42", `["https://radio-t.com/blah1","https://radio-t.com/blah2"]`)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	j := []store.PostInfo{}
	err = json.Unmarshal(body, &j)
	assert.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/blah1", Count: 3},
		{URL: "https://radio-t.com/blah2", Count: 2}}, j)

	resp, err = post(t, ts.URL+"/api/v1/counts?site=radio-XXX", `{}`)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestRest_List(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah2"}}

	addComment(t, c1, ts)
	addComment(t, c1, ts)
	addComment(t, c1, ts)
	addComment(t, c2, ts)
	addComment(t, c2, ts)

	body, code := get(t, ts.URL+"/api/v1/list?site=remark42")
	assert.Equal(t, 200, code)
	pi := []store.PostInfo{}
	err := json.Unmarshal([]byte(body), &pi)
	assert.NoError(t, err)
	assert.Equal(t, "https://radio-t.com/blah2", pi[0].URL)
	assert.Equal(t, 2, pi[0].Count)
	assert.Equal(t, "https://radio-t.com/blah1", pi[1].URL)
	assert.Equal(t, 3, pi[1].Count)

	_, code = get(t, ts.URL+"/api/v1/list?site=remark42-BLAH")
	assert.Equal(t, 400, code)
}

func TestRest_ListWithSkipAndLimit(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah2"}}
	c3 := store.Comment{Text: "test test #3", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah3"}}

	addComment(t, c1, ts)
	addComment(t, c1, ts)
	addComment(t, c1, ts)
	addComment(t, c2, ts)
	addComment(t, c2, ts)
	addComment(t, c3, ts)
	addComment(t, c3, ts)

	body, code := get(t, ts.URL+"/api/v1/list?site=remark42&skip=1&limit=2")
	assert.Equal(t, 200, code)
	pi := []store.PostInfo{}
	err := json.Unmarshal([]byte(body), &pi)
	assert.NoError(t, err)
	require.Equal(t, 2, len(pi))
	assert.Equal(t, "https://radio-t.com/blah2", pi[0].URL)
	assert.Equal(t, 2, pi[0].Count)
	assert.Equal(t, "https://radio-t.com/blah1", pi[1].URL)
	assert.Equal(t, 3, pi[1].Count)
}

func TestRest_Config(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	body, code := get(t, ts.URL+"/api/v1/config?site=remark42")
	assert.Equal(t, 200, code)
	j := R.JSON{}
	err := json.Unmarshal([]byte(body), &j)
	assert.NoError(t, err)
	assert.Equal(t, 300.0, j["edit_duration"])
	assert.EqualValues(t, []interface{}{"a1", "a2"}, j["admins"])
	assert.Equal(t, "admin@remark-42.com", j["admin_email"])
	assert.Equal(t, 4000.0, j["max_comment_size"])
	assert.Equal(t, -5.0, j["low_score"])
	assert.Equal(t, -10.0, j["critical_score"])
	assert.False(t, j["positive_score"].(bool))
	assert.Equal(t, 10.0, j["readonly_age"])
	assert.Equal(t, 10000.0, j["max_image_size"])
	assert.Equal(t, true, j["emoji_enabled"].(bool))
}

func TestRest_Info(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	srv.pubRest.readOnlyAge = 10000000 // make sure we don't hit read-only

	user := store.User{ID: "user1", Name: "user name 1"}
	c1 := store.Comment{User: user, Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 05, 27, 1, 14, 10, 0, time.Local)}
	c2 := store.Comment{User: user, Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 05, 27, 1, 14, 20, 0, time.Local)}
	c3 := store.Comment{User: user, Text: "test test #3", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 05, 27, 1, 14, 25, 0, time.Local)}

	_, err := srv.DataService.Create(c1)
	require.NoError(t, err, "%+v", err)
	_, err = srv.DataService.Create(c2)
	require.NoError(t, err)
	_, err = srv.DataService.Create(c3)
	require.NoError(t, err)

	body, code := get(t, ts.URL+"/api/v1/info?site=remark42&url=https://radio-t.com/blah1")
	assert.Equal(t, 200, code)

	info := store.PostInfo{}
	err = json.Unmarshal([]byte(body), &info)
	assert.NoError(t, err)
	exp := store.PostInfo{URL: "https://radio-t.com/blah1", Count: 3,
		FirstTS: time.Date(2018, 05, 27, 1, 14, 10, 0, time.Local), LastTS: time.Date(2018, 05, 27, 1, 14, 25, 0, time.Local)}
	assert.Equal(t, exp, info)

	_, code = get(t, ts.URL+"/api/v1/info?site=remark42&url=https://radio-t.com/blah-no")
	assert.Equal(t, 400, code)
	_, code = get(t, ts.URL+"/api/v1/info?site=remark42-no&url=https://radio-t.com/blah-no")
	assert.Equal(t, 400, code)
}

func TestRest_InfoStream(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()
	srv.pubRest.readOnlyAge = 10000000 // make sure we don't hit read-only
	srv.pubRest.streamer.Refresh = 1 * time.Millisecond
	srv.pubRest.streamer.TimeOut = 800 * time.Millisecond
	srv.pubRest.streamer.MaxActive = 100

	postComment(t, ts.URL)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 10; i++ {
			time.Sleep(10 * time.Millisecond)
			postComment(t, ts.URL)
		}
	}()

	body, code := get(t, ts.URL+"/api/v1/stream/info?site=remark42&url=https://radio-t.com/blah1")
	assert.Equal(t, 200, code)
	<-done

	recs := strings.Split(strings.TrimSuffix(body, "\n"), "\n")
	require.Equal(t, 10*3, len(recs), "10 records. each 2 lines +1 emty line")
	assert.True(t, strings.Contains(recs[0+1], `"count":2`), recs[0])
	assert.True(t, strings.Contains(recs[9*3+1], `"count":11`), recs[9])

	_, code = get(t, ts.URL+"/api/v1/stream/info?site=remark42&url=https://radio-t.com/blah123")
	assert.Equal(t, 500, code)
}

func TestRest_InfoStreamTooMany(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()
	srv.pubRest.readOnlyAge = 10000000 // make sure we don't hit read-only
	srv.pubRest.streamer.Refresh = 1 * time.Millisecond
	srv.pubRest.streamer.TimeOut = 300 * time.Millisecond
	srv.pubRest.streamer.MaxActive = 10

	postComment(t, ts.URL)

	var errsCount int32
	wg := sync.WaitGroup{}
	wg.Add(20)
	for i := 0; i < 20; i++ {
		go func() {
			_, code := get(t, ts.URL+"/api/v1/stream/info?site=remark42&url=https://radio-t.com/blah1")
			if code == 429 {
				atomic.AddInt32(&errsCount, 1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(10), atomic.LoadInt32(&errsCount), "10 streams rejected")
}

func TestRest_InfoStreamTimeout(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()
	srv.pubRest.readOnlyAge = 10000000 // make sure we don't hit read-only
	srv.pubRest.streamer.Refresh = 10 * time.Millisecond
	srv.pubRest.streamer.TimeOut = 450 * time.Millisecond
	srv.pubRest.streamer.MaxActive = 100

	postComment(t, ts.URL)

	st := time.Now()
	_, code := get(t, ts.URL+"/api/v1/stream/info?site=remark42&url=https://radio-t.com/blah1")
	assert.Equal(t, 200, code)
	assert.True(t, time.Since(st) > time.Millisecond*450 && time.Since(st) < time.Millisecond*500, time.Since(st))
}

func TestRest_InfoStreamCancel(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()
	srv.pubRest.readOnlyAge = 10000000 // make sure we don't hit read-only
	srv.pubRest.streamer.Refresh = 10 * time.Millisecond
	srv.pubRest.streamer.TimeOut = 1500 * time.Millisecond
	srv.pubRest.streamer.MaxActive = 100

	postComment(t, ts.URL)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 5; i++ {
			time.Sleep(300 * time.Millisecond)
			postComment(t, ts.URL)
		}
	}()

	client := http.Client{}
	req, err := http.NewRequest("GET", ts.URL+"/api/v1/stream/info?site=remark42&url=https://radio-t.com/blah1", nil)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)
	r, err := client.Do(req)
	require.NoError(t, err)
	defer r.Body.Close()
	<-ctx.Done()
	<-done
	body, err := ioutil.ReadAll(r.Body)
	require.EqualError(t, err, "context deadline exceeded")
	assert.Equal(t, 200, r.StatusCode)

	recs := strings.Count(string(body), "data:")
	require.Equal(t, 1, recs, "should have 1 event:\n", string(body))
	assert.Contains(t, string(body), `"count":2`)
}

func TestRest_InfoStreamSince(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()
	srv.pubRest.readOnlyAge = 10000000 // make sure we don't hit read-only
	srv.pubRest.streamer.Refresh = 10 * time.Millisecond
	srv.pubRest.streamer.TimeOut = 900 * time.Millisecond
	srv.pubRest.streamer.MaxActive = 100

	postComment(t, ts.URL)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 10; i++ {
			time.Sleep(15 * time.Millisecond)
			postComment(t, ts.URL)
		}
	}()

	body, code := get(t, ts.URL+"/api/v1/stream/info?site=remark42&url=https://radio-t.com/blah1&since=12345678")
	assert.Equal(t, 200, code)
	<-done
	recs := strings.Split(strings.TrimSuffix(body, "\n"), "\n")
	require.Equal(t, 11*3, len(recs), "include first record, total 11 records. each 2 lines +1 empty line")
}

func TestRest_Robots(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	body, code := get(t, ts.URL+"/robots.txt")
	assert.Equal(t, 200, code)
	assert.Equal(t, "User-agent: *\nDisallow: /auth/\nDisallow: /api/\nAllow: /api/v1/find\n"+
		"Allow: /api/v1/last\nAllow: /api/v1/id\nAllow: /api/v1/count\nAllow: /api/v1/counts\n"+
		"Allow: /api/v1/list\nAllow: /api/v1/config\nAllow: /api/v1/user\nAllow: /api/v1/img\n"+
		"Allow: /api/v1/avatar\nAllow: /api/v1/picture\n", body)
}

func TestRest_LastCommentsStream(t *testing.T) {
	t.Skip() // TODO: enable after cache is migrated to https://github.com/dgraph-io/ristretto
	ts, srv, teardown := startupT(t)
	srv.pubRest.readOnlyAge = 10000000 // make sure we don't hit read-only
	srv.pubRest.streamer.Refresh = 50 * time.Millisecond
	srv.pubRest.streamer.TimeOut = 500 * time.Millisecond
	srv.pubRest.streamer.MaxActive = 100

	// stream endpoint currently relies on real cache being present
	cacheBackend, err := cache.NewExpirableCache()
	require.NoError(t, err)
	memCache := cache.NewScache(cacheBackend)
	srv.privRest.cache = memCache
	srv.pubRest.cache = memCache

	postComment(t, ts.URL)

	defer teardown()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 1; i < 10; i++ {
			postComment(t, ts.URL)
			time.Sleep(100 * time.Millisecond)
		}
		t.Log("wrote 10 records")
	}()

	client := http.Client{}
	req, err := http.NewRequest("GET", ts.URL+"/api/v1/stream/last?site=remark42", nil)
	require.NoError(t, err)
	r, err := client.Do(req)
	require.NoError(t, err)
	defer r.Body.Close()
	<-done
	body, err := ioutil.ReadAll(r.Body)
	require.NoError(t, err)
	assert.Equal(t, 200, r.StatusCode)

	assert.Equal(t, "text/event-stream", r.Header.Get("content-type"))
	assert.Equal(t, "keep-alive", r.Header.Get("connection"))

	recs := strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
	require.Equal(t, 9*3, len(recs), "9 events")
	assert.True(t, strings.Contains(recs[1], `test 123`), recs[1])
}

func TestRest_LastCommentsStreamTimeout(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()
	srv.pubRest.readOnlyAge = 10000000 // make sure we don't hit read-only
	srv.pubRest.streamer.Refresh = 10 * time.Millisecond
	srv.pubRest.streamer.TimeOut = 450 * time.Millisecond
	srv.pubRest.streamer.MaxActive = 100

	postComment(t, ts.URL)

	st := time.Now()
	_, code := get(t, ts.URL+"/api/v1/stream/last?site=remark42")
	assert.Equal(t, 200, code)
	assert.True(t, time.Since(st) > time.Millisecond*450 && time.Since(st) < time.Millisecond*500, time.Since(st))
}

func TestRest_LastCommentsStreamCancel(t *testing.T) {
	ts, srv, teardown := startupT(t)
	srv.pubRest.readOnlyAge = 10000000 // make sure we don't hit read-only
	srv.pubRest.streamer.Refresh = 10 * time.Millisecond
	srv.pubRest.streamer.TimeOut = 500 * time.Millisecond
	srv.pubRest.streamer.MaxActive = 100

	// stream endpoint currently relies on real cache being present
	cacheBackend, err := cache.NewExpirableCache()
	require.NoError(t, err)
	memCache := cache.NewScache(cacheBackend)
	srv.privRest.cache = memCache
	srv.pubRest.cache = memCache

	postComment(t, ts.URL)

	defer teardown()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 1; i < 10; i++ {
			time.Sleep(100 * time.Millisecond)
			postComment(t, ts.URL)
		}
	}()

	client := http.Client{}
	req, err := http.NewRequest("GET", ts.URL+"/api/v1/stream/last?site=remark42", nil)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 290*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)
	r, err := client.Do(req)
	require.NoError(t, err)
	<-done
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	require.EqualError(t, err, "context deadline exceeded")
	assert.Equal(t, 200, r.StatusCode)

	recs := strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
	assert.True(t, len(recs) < 30, "less 10 events")
}

func TestRest_LastCommentsStreamTooMany(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()
	srv.pubRest.readOnlyAge = 10000000 // make sure we don't hit read-only
	srv.pubRest.streamer.Refresh = 1 * time.Millisecond
	srv.pubRest.streamer.TimeOut = 300 * time.Millisecond
	srv.pubRest.streamer.MaxActive = 10

	postComment(t, ts.URL)

	var errsCount int32
	wg := sync.WaitGroup{}
	wg.Add(20)
	for i := 0; i < 20; i++ {
		go func() {
			_, code := get(t, ts.URL+"/api/v1/stream/last?site=remark42")
			if code == 429 {
				atomic.AddInt32(&errsCount, 1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(10), atomic.LoadInt32(&errsCount), "10 streams rejected")

	_, code := get(t, ts.URL+"/api/v1/stream/last?site=remark42")
	assert.Equal(t, 200, code, "all streams closed, good to go again")
}

func TestRest_LastCommentsStreamSince(t *testing.T) {
	ts, srv, teardown := startupT(t)
	srv.pubRest.readOnlyAge = 10000000 // make sure we don't hit read-only
	srv.pubRest.streamer.Refresh = 10 * time.Millisecond
	srv.pubRest.streamer.TimeOut = 500 * time.Millisecond
	srv.pubRest.streamer.MaxActive = 100

	// stream endpoint currently relies on real cache being present
	cacheBackend, err := cache.NewExpirableCache()
	require.NoError(t, err)
	memCache := cache.NewScache(cacheBackend)
	srv.privRest.cache = memCache
	srv.pubRest.cache = memCache

	postComment(t, ts.URL)

	defer teardown()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 1; i < 10; i++ {
			time.Sleep(50 * time.Millisecond)
			postComment(t, ts.URL)
		}
	}()

	client := http.Client{}
	req, err := http.NewRequest("GET", ts.URL+"/api/v1/stream/last?site=remark42&since=123456", nil)
	require.NoError(t, err)
	r, err := client.Do(req)
	require.NoError(t, err)
	<-done
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	require.NoError(t, err)
	assert.Equal(t, 200, r.StatusCode)

	assert.Equal(t, "text/event-stream", r.Header.Get("content-type"))

	recs := strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
	require.Equal(t, 10*3, len(recs), "should be 10 events, including first record:\n", recs)
}

func postComment(t *testing.T, url string) {
	resp, err := post(t, url+"/api/v1/comment",
		`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`)
	require.NoError(t, err)
	b, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode, string(b))
}
