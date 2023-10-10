package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	cache "github.com/go-pkgz/lcw"
	R "github.com/go-pkgz/rest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/service"
)

func TestRest_Ping(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	resp, err := http.Get(ts.URL + "/api/v1/ping")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, "pong", string(body))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "remark42", resp.Header.Get("App-Name"))
}

func TestRest_PingNoSignature(t *testing.T) {
	ts, _, teardown := startupT(t, func(srv *Rest) {
		srv.DisableSignature = true
	})
	defer teardown()

	resp, err := http.Get(ts.URL + "/api/v1/ping")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, "pong", string(body))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "", resp.Header.Get("App-Name"))
}

func TestRest_Preview(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	resp, err := post(t, ts.URL+"/api/v1/preview", `{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	b, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, "<p>test 123</p>\n", string(b))

	resp, err = post(t, ts.URL+"/api/v1/preview", "bad")
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	resp, err = post(t, ts.URL+"/api/v1/preview", fmt.Sprintf(`{"text": "![non-existent.jpg](%s/api/v1/picture/dev_user/bad_picture)", "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`, srv.RemarkURL))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	b, err = io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Contains(t,
		string(b),
		`{"code":20,"details":"can't load picture from the comment",`+
			`"error":"can't get image stats for dev_user/bad_picture: stat`,
	)
	assert.Contains(t,
		string(b),
		"/pics-remark42/staging/dev_user/62/bad_picture: no such file or directory\"}\n",
	)
}

func TestRest_PreviewWithWrongImage(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	resp, err := post(t, ts.URL+"/api/v1/preview", fmt.Sprintf(`{"text": "![non-existent.jpg](%s/api/v1/picture/dev_user/bad_picture)", "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`, srv.RemarkURL))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	b, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Contains(t,
		string(b),
		`{"code":20,"details":"can't load picture from the comment",`+
			`"error":"can't get image stats for dev_user/bad_picture: stat `,
	)
	assert.Contains(t,
		string(b),
		"/pics-remark42/staging/dev_user/62/bad_picture: no such file or directory\"}\n",
	)
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
	j := fmt.Sprintf(`{"text": %q, "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`, text)
	j = strings.Replace(j, "\n", "\\n", -1)

	resp, err := post(t, ts.URL+"/api/v1/preview", j)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	b, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t,
		`<h1>h1</h1>
<pre class="chroma"><code><span class="line"><span class="cl">func TestRest_Preview(t *testing.T) {
</span></span><span class="line"><span class="cl">srv, ts := prep(t)
</span></span><span class="line"><span class="cl">  require.NotNil(t, srv)
</span></span><span class="line"><span class="cl">}
</span></span></code></pre>`,
		string(b))
	assert.NoError(t, resp.Body.Close())
}

func TestRest_PreviewCode(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	text := `BKTgo
func main(aa string) int {return 0}
BKT
`
	text = strings.Replace(text, "BKT", "```", -1)
	j := fmt.Sprintf(`{"text": %q, "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`, text)
	j = strings.Replace(j, "\n", "\\n", -1)

	resp, err := post(t, ts.URL+"/api/v1/preview", j)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	b, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, `<pre class="chroma"><code><span class="line"><span class="cl"><span class="kd">func</span> <span class="nf">main</span><span class="p">(</span><span class="nx">aa</span> <span class="kt">string</span><span class="p">)</span> <span class="kt">int</span> <span class="p">{</span><span class="k">return</span> <span class="mi">0</span><span class="p">}</span>
</span></span></code></pre>`, string(b))
	assert.NoError(t, resp.Body.Close())
}

func TestRest_Find(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	res, code := get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah1")
	assert.Equal(t, http.StatusOK, code)
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
	assert.Equal(t, http.StatusOK, code)
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
	assert.Equal(t, http.StatusOK, code)
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	require.Equal(t, 2, len(comments.Comments), "should have 2 comments")
	assert.Equal(t, id1, comments.Comments[1].ID)
	assert.Equal(t, id2, comments.Comments[0].ID)

	// get in tree mode
	tree := service.Tree{}
	res, code = get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah1&format=tree")
	assert.Equal(t, http.StatusOK, code)
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
	assert.Equal(t, http.StatusOK, code)
	err = json.Unmarshal([]byte(res), &tree)
	assert.NoError(t, err)
	assert.Equal(t, "https://radio-t.com/blah1", tree.Info.URL)
	assert.False(t, tree.Info.ReadOnly, "post is fresh")

	res, code = get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah2&format=tree")
	assert.Equal(t, http.StatusOK, code)
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
	defer client.CloseIdleConnections()
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/readonly?site=remark42&url=https://radio-t.com/blah1&ro=1", ts.URL), http.NoBody)
	assert.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	tree := service.Tree{}
	res, code := get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah1&format=tree")
	assert.Equal(t, http.StatusOK, code)
	err = json.Unmarshal([]byte(res), &tree)
	require.NoError(t, err)
	assert.Equal(t, "https://radio-t.com/blah1", tree.Info.URL)
	assert.True(t, tree.Info.ReadOnly, "post is ro")

	tree = service.Tree{}
	res, code = get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah2&format=tree")
	assert.Equal(t, http.StatusOK, code)
	err = json.Unmarshal([]byte(res), &tree)
	require.NoError(t, err)
	assert.Equal(t, "https://radio-t.com/blah2", tree.Info.URL)
	assert.False(t, tree.Info.ReadOnly, "post is writable")
}

func TestRest_FindUserView(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	res, code := get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah1&view=user")
	assert.Equal(t, http.StatusOK, code)
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
	assert.Equal(t, http.StatusOK, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	require.Equal(t, 2, len(comments.Comments), "should have 2 comments")
	assert.Equal(t, id1, comments.Comments[0].ID)
	assert.Equal(t, id2, comments.Comments[1].ID)
	assert.Equal(t, "provider1_dev", comments.Comments[0].User.ID)
	assert.Equal(t, "provider1_dev", comments.Comments[1].User.ID)
	assert.Equal(t, "", comments.Comments[0].Text)
	assert.Equal(t, "", comments.Comments[1].Text)

	err = srv.DataService.Delete(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}, id1, store.SoftDelete)
	assert.NoError(t, err)
	srv.Cache.Flush(cache.FlusherRequest{})

	res, code = get(t, ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah1&sort=+time&view=user")
	assert.Equal(t, http.StatusOK, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	require.Equal(t, 1, len(comments.Comments), "1 comment left")
	assert.Equal(t, id2, comments.Comments[0].ID)
}

func TestRest_Last(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	res, code := get(t, ts.URL+"/api/v1/last/2?site=remark42")
	assert.Equal(t, http.StatusOK, code)
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
	assert.Equal(t, http.StatusOK, code)
	comments := []store.Comment{}
	err := json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	require.Equal(t, 2, len(comments), "should have 2 comments")
	assert.Equal(t, id1, comments[1].ID)
	assert.Equal(t, id2, comments[0].ID)

	res, code = get(t, fmt.Sprintf("%s/api/v1/last/2?site=remark42&since=%d", ts.URL, ts1))
	assert.Equal(t, http.StatusOK, code)
	comments = []store.Comment{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	require.Equal(t, 2, len(comments), "should have 2 comments")
	assert.Equal(t, id1, comments[1].ID)
	assert.Equal(t, id2, comments[0].ID)

	res, code = get(t, fmt.Sprintf("%s/api/v1/last/2?site=remark42&since=%d", ts.URL, ts2))
	assert.Equal(t, http.StatusOK, code)
	comments = []store.Comment{}
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	require.Equal(t, 1, len(comments), "should have 1 comments")
	assert.Equal(t, id2, comments[0].ID)

	res, code = get(t, ts.URL+"/api/v1/last/5?site=remark42")
	assert.Equal(t, http.StatusOK, code)
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(comments), "should have 3 comments")

	res, code = get(t, ts.URL+"/api/v1/last/X?site=remark42")
	assert.Equal(t, http.StatusOK, code)
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(comments), "should have 3 comments")

	err = srv.DataService.Delete(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}, id1, store.SoftDelete)
	assert.NoError(t, err)
	srv.Cache.Flush(cache.FlusherRequest{})
	res, code = get(t, ts.URL+"/api/v1/last/5?site=remark42")
	assert.Equal(t, http.StatusOK, code)
	err = json.Unmarshal([]byte(res), &comments)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(comments), "should have 2 comments")

	_, code = get(t, ts.URL+"/api/v1/last/2?site=remark42-BLAH")
	assert.Equal(t, http.StatusInternalServerError, code)
}

func TestRest_FindUserComments(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah2"}}
	c3 := store.Comment{Text: "test test #3", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah3"}}

	// add 3 comments
	addComment(t, c1, ts)
	addComment(t, c2, ts)
	addComment(t, c3, ts)

	// add one deleted
	id := addComment(t, c2, ts)
	err := srv.DataService.Delete(c2.Locator, id, store.SoftDelete)
	assert.NoError(t, err)

	comments, code := get(t, ts.URL+"/api/v1/comments?site=remark42&user=blah")
	assert.Equal(t, http.StatusOK, code, "noting for user blah")
	assert.Equal(t, `{"comments":[],"count":0}`+"\n", comments)
	{
		res, code := get(t, ts.URL+"/api/v1/comments?site=remark42&user=provider1_dev")
		assert.Equal(t, http.StatusOK, code)

		resp := struct {
			Comments []store.Comment
			Count    int
		}{}

		err = json.Unmarshal([]byte(res), &resp)
		assert.NoError(t, err)
		require.Equal(t, 3, len(resp.Comments), "should have 3 comments")
		assert.Equal(t, 4, resp.Count, "should have 3+1 count") // TODO: fix as we start to skip deleted

		// user comment sorted with -time
		assert.True(t, resp.Comments[0].Timestamp.After(resp.Comments[1].Timestamp))
		assert.True(t, resp.Comments[1].Timestamp.After(resp.Comments[2].Timestamp))
	}

	{
		res, code := get(t, ts.URL+"/api/v1/comments?site=remark42&user=provider1_dev&skip=1&limit=2")
		assert.Equal(t, http.StatusOK, code)

		resp := struct {
			Comments []store.Comment
			Count    int
		}{}

		err = json.Unmarshal([]byte(res), &resp)
		assert.NoError(t, err)
		require.Equal(t, 2, len(resp.Comments), "should have 2 comments due to the limit")
		assert.Equal(t, 4, resp.Count, "should have 4 count")

		assert.Equal(t, "https://radio-t.com/blah3", resp.Comments[0].Locator.URL)
		assert.Equal(t, "https://radio-t.com/blah2", resp.Comments[1].Locator.URL)
	}
}

func TestRest_FindUserComments_CWE_918(t *testing.T) {
	ts, srv, teardown := startupT(t)
	srv.DataService.TitleExtractor = service.NewTitleExtractor(http.Client{Timeout: time.Second}, []string{"radio-t.com"}) // required for extracting the title, bad URL test
	defer srv.DataService.TitleExtractor.Close()
	defer teardown()

	backendRequestedArbitraryServer := false
	arbitraryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("request received: %+v", r)
		backendRequestedArbitraryServer = true
	}))
	defer arbitraryServer.Close()

	arbitraryURLComment := store.Comment{Text: "arbitrary URL request test",
		Locator: store.Locator{SiteID: "remark42", URL: arbitraryServer.URL}}

	assert.False(t, backendRequestedArbitraryServer)
	addComment(t, arbitraryURLComment, ts)
	assert.False(t, backendRequestedArbitraryServer,
		"no request is expected to the test server as it's not in the list of the allowed domains for the title extractor")

	res, code := get(t, ts.URL+"/api/v1/comments?site=remark42&user=provider1_dev")
	assert.Equal(t, http.StatusOK, code)

	resp := struct {
		Comments []store.Comment
		Count    int
	}{}

	err := json.Unmarshal([]byte(res), &resp)
	assert.NoError(t, err)
	require.Equal(t, 1, len(resp.Comments), "should have 2 comments")

	assert.Equal(t, "", resp.Comments[0].PostTitle, "empty from the first post")
	assert.Equal(t, arbitraryServer.URL, resp.Comments[0].Locator.URL, "arbitrary URL provided by the request")
}

func TestRest_UserInfo(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	body, code := getWithDevAuth(t, ts.URL+"/api/v1/user?site=remark42")
	assert.Equal(t, http.StatusOK, code)
	user := store.User{}
	err := json.Unmarshal([]byte(body), &user)
	assert.NoError(t, err)
	assert.Equal(t, store.User{Name: "developer one", ID: "provider1_dev", Picture: "http://example.com/pic.png",
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
	assert.Equal(t, http.StatusOK, code)
	j := R.JSON{}
	err := json.Unmarshal([]byte(body), &j)
	assert.NoError(t, err)
	assert.Equal(t, 3.0, j["count"])

	body, code = get(t, ts.URL+"/api/v1/count?site=remark42&url=https://radio-t.com/blah2")
	assert.Equal(t, http.StatusOK, code)
	err = json.Unmarshal([]byte(body), &j)
	assert.NoError(t, err)
	assert.Equal(t, 2.0, j["count"])

	_, code = get(t, ts.URL+"/api/v1/count?site=remark42-BLAH&url=https://radio-t.com/blah1XXX")
	assert.Equal(t, http.StatusBadRequest, code)
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

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())

	j := []store.PostInfo{}
	err = json.Unmarshal(body, &j)
	assert.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/blah1", Count: 3},
		{URL: "https://radio-t.com/blah2", Count: 2}}, j)

	resp, err = post(t, ts.URL+"/api/v1/counts?site=radio-XXX", `{}`)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, resp.Body.Close())
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
	assert.Equal(t, http.StatusOK, code)
	pi := []store.PostInfo{}
	err := json.Unmarshal([]byte(body), &pi)
	assert.NoError(t, err)
	assert.Equal(t, "https://radio-t.com/blah2", pi[0].URL)
	assert.Equal(t, 2, pi[0].Count)
	assert.Equal(t, "https://radio-t.com/blah1", pi[1].URL)
	assert.Equal(t, 3, pi[1].Count)

	_, code = get(t, ts.URL+"/api/v1/list?site=remark42-BLAH")
	assert.Equal(t, http.StatusBadRequest, code)
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
	assert.Equal(t, http.StatusOK, code)
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
	assert.Equal(t, http.StatusOK, code)
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
	assert.Equal(t, false, j["admin_edit"].(bool))
}

func TestRest_QR(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	// missing parameter
	body, code := get(t, ts.URL+"/api/v1/qr/telegram")
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, "{\"code\":0,\"details\":\"text parameter is required\",\"error\":\"missing parameter\"}\n", body)

	// too long request to build the qr
	body, code = get(t, ts.URL+"/api/v1/qr/telegram?url=https://t.me/"+strings.Repeat("string", 1000))
	assert.Equal(t, http.StatusInternalServerError, code)
	assert.Equal(t, "{\"code\":0,\"details\":\"can't generate QR\",\"error\":\"content too long to encode\"}\n", body)

	// wrong request
	body, code = get(t, ts.URL+"/api/v1/qr/telegram?url=nonsense")
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, "{\"code\":0,\"details\":\"text parameter should start with https://t.me/\",\"error\":\"wrong parameter\"}\n", body)

	// correct request
	r, err := http.Get(ts.URL + "/api/v1/qr/telegram?url=https://t.me/BotFather")
	require.NoError(t, err)
	bdy, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	require.NoError(t, r.Body.Close())
	require.NotEmpty(t, bdy)
	assert.Equal(t, "image/png", r.Header.Get("Content-Type"))
	assert.Equal(t, http.StatusOK, r.StatusCode)

	// compare the image
	fh, err := os.Open("testdata/qr_test.png")
	defer func() { assert.NoError(t, fh.Close()) }()
	assert.NoError(t, err)
	img, err := io.ReadAll(fh)
	assert.NoError(t, err)
	assert.Equal(t, img, bdy)
}

func TestRest_Info(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	srv.pubRest.readOnlyAge = 10000000 // make sure we don't hit read-only

	user := store.User{ID: "user1", Name: "user name 1"}
	c1 := store.Comment{User: user, Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 5, 27, 1, 14, 10, 0, time.Local)}
	c2 := store.Comment{User: user, Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 5, 27, 1, 14, 20, 0, time.Local)}
	c3 := store.Comment{User: user, Text: "test test #3", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 5, 27, 1, 14, 25, 0, time.Local)}

	_, err := srv.DataService.Create(c1)
	require.NoError(t, err, "%+v", err)
	_, err = srv.DataService.Create(c2)
	require.NoError(t, err)
	_, err = srv.DataService.Create(c3)
	require.NoError(t, err)

	body, code := get(t, ts.URL+"/api/v1/info?site=remark42&url=https://radio-t.com/blah1")
	assert.Equal(t, http.StatusOK, code)

	info := store.PostInfo{}
	err = json.Unmarshal([]byte(body), &info)
	assert.NoError(t, err)
	exp := store.PostInfo{URL: "https://radio-t.com/blah1", Count: 3,
		FirstTS: time.Date(2018, 5, 27, 1, 14, 10, 0, time.Local), LastTS: time.Date(2018, 5, 27, 1, 14, 25, 0, time.Local)}
	assert.Equal(t, exp, info)

	_, code = get(t, ts.URL+"/api/v1/info?site=remark42&url=https://radio-t.com/blah-no")
	assert.Equal(t, http.StatusBadRequest, code)
	_, code = get(t, ts.URL+"/api/v1/info?site=remark42-no&url=https://radio-t.com/blah-no")
	assert.Equal(t, http.StatusBadRequest, code)
}

func TestRest_Robots(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	body, code := get(t, ts.URL+"/robots.txt")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "User-agent: *\nDisallow: /auth/\nDisallow: /api/\nAllow: /api/v1/find\n"+
		"Allow: /api/v1/last\nAllow: /api/v1/id\nAllow: /api/v1/count\nAllow: /api/v1/counts\n"+
		"Allow: /api/v1/list\nAllow: /api/v1/config\nAllow: /api/v1/user\nAllow: /api/v1/img\n"+
		"Allow: /api/v1/avatar\nAllow: /api/v1/picture\n", body)
}
