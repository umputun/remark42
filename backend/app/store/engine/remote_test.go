package engine

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-pkgz/jrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
)

func TestRemote_Create(t *testing.T) {
	ts := testServer(t, `{"method":"store.create","params":{"id":"123","pid":"","text":"msg","user":{"name":"","id":"","picture":"","admin":false},"locator":{"site":"site","url":"http://example.com/url"},"score":0,"vote":0,"time":"0001-01-01T00:00:00Z"},"id":1}`,
		`{"result":"12345","id":1}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	var eng Interface = &c
	_ = eng

	res, err := c.Create(store.Comment{ID: "123", Locator: store.Locator{URL: "http://example.com/url", SiteID: "site"},
		Text: "msg"})
	assert.NoError(t, err)
	assert.Equal(t, "12345", res)
	t.Logf("%v %T", res, res)
}

func TestRemote_Get(t *testing.T) {
	ts := testServer(t, `{"method":"store.get","params":{"locator":{"url":"http://example.com/url"},"comment_id":"site"},"id":1}`, `{"result":{"id":"123","pid":"","text":"msg","delete":true}}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	req := GetRequest{Locator: store.Locator{URL: "http://example.com/url"}, CommentID: "site"}
	res, err := c.Get(req)
	assert.NoError(t, err)
	assert.Equal(t, store.Comment{ID: "123", Text: "msg", Deleted: true}, res)
	t.Logf("%v %T", res, res)
}

func TestRemote_GetWithErrorResult(t *testing.T) {
	ts := testServer(t, `{"method":"store.get","params":{"locator":{"url":"http://example.com/url"},"comment_id":"site"},"id":1}`, `{"error":"failed"}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	req := GetRequest{Locator: store.Locator{URL: "http://example.com/url"}, CommentID: "site"}
	_, err := c.Get(req)
	assert.EqualError(t, err, "failed")
}

func TestRemote_GetWithErrorDecode(t *testing.T) {
	ts := testServer(t, `{"method":"store.get","params":{"locator":{"url":"http://example.com/url"},"comment_id":"site"},"id":1}`, ``)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	req := GetRequest{Locator: store.Locator{URL: "http://example.com/url"}, CommentID: "site"}
	_, err := c.Get(req)
	assert.EqualError(t, err, "failed to decode response for store.get: EOF")
}

func TestRemote_GetWithErrorRemote(t *testing.T) {
	c := RPC{Client: jrpc.Client{API: "http://127.0.0.2", Client: http.Client{Timeout: 10 * time.Millisecond}}}

	req := GetRequest{Locator: store.Locator{URL: "http://example.com/url"}, CommentID: "site"}
	_, err := c.Get(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "remote call failed for store.get:")
}

func TestRemote_FailedStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		t.Logf("req: %s", string(body))
		w.WriteHeader(400)
	}))
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	req := GetRequest{Locator: store.Locator{URL: "http://example.com/url"}, CommentID: "site"}
	_, err := c.Get(req)
	assert.EqualError(t, err, "bad status 400 Bad Request for store.get")
}

func TestRemote_Update(t *testing.T) {
	ts := testServer(t, `{"method":"store.update","params":{"id":"123","pid":"","text":"msg","user":{"name":"","id":"","picture":"","admin":false},"locator":{"site":"site123","url":"http://example.com/url"},"score":0,"vote":0,"time":"0001-01-01T00:00:00Z"},"id":1}`, `{}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	err := c.Update(store.Comment{ID: "123", Locator: store.Locator{URL: "http://example.com/url", SiteID: "site123"},
		Text: "msg"})
	assert.NoError(t, err)

}

func TestRemote_Find(t *testing.T) {
	ts := testServer(t, `{"method":"store.find","params":{"locator":{"url":"http://example.com/url"},"sort":"-time","since":"0001-01-01T00:00:00Z","limit":10},"id":1}`, `{"result":[{"text":"1"},{"text":"2"}]}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	res, err := c.Find(FindRequest{Locator: store.Locator{URL: "http://example.com/url"}, Sort: "-time", Limit: 10})
	assert.NoError(t, err)
	assert.Equal(t, []store.Comment{{Text: "1"}, {Text: "2"}}, res)
}

func TestRemote_Info(t *testing.T) {
	ts := testServer(t, `{"method":"store.info","params":{"locator":{"url":"http://example.com/url"},"limit":10,"skip":5,"ro_age":10},"id":1}`, `{"result":[{"url":"u1","count":22},{"url":"u2","count":33}]}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	res, err := c.Info(InfoRequest{Locator: store.Locator{URL: "http://example.com/url"},
		Limit: 10, Skip: 5, ReadOnlyAge: 10})
	assert.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "u1", Count: 22}, {URL: "u2", Count: 33}}, res)
}

func TestRemote_Flag(t *testing.T) {
	ts := testServer(t, `{"method":"store.flag","params":{"flag":"verified","locator":{"url":"http://example.com/url"}},"id":1}`, `{"result":false}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	res, err := c.Flag(FlagRequest{Locator: store.Locator{URL: "http://example.com/url"}, Flag: Verified})
	assert.NoError(t, err)
	assert.Equal(t, false, res)
}

func TestRemote_ListFlag(t *testing.T) {
	ts := testServer(t, `{"method":"store.list_flags","params":{"flag":"blocked","locator":{"site":"site_id","url":""}},"id":1}`, `{"result":[{"ID":"id1"},{"ID":"id2"}]}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}
	res, err := c.ListFlags(FlagRequest{Locator: store.Locator{SiteID: "site_id"}, Flag: Blocked})
	assert.NoError(t, err)
	assert.Equal(t, []interface{}{map[string]interface{}{"ID": "id1"}, map[string]interface{}{"ID": "id2"}}, res)
}

func TestRemote_UserDetail(t *testing.T) {
	ts := testServer(t, `{"method":"store.user_detail","params":{"detail":"email","locator":{"url":"http://example.com/url"},"user_id":"username"},"id":1}`, `{"result":[{"user_id":"u1","email":"test_email@example.com"}]}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	req := UserDetailRequest{Locator: store.Locator{URL: "http://example.com/url"}, UserID: "username", Detail: UserEmail}
	res, err := c.UserDetail(req)
	assert.NoError(t, err)
	assert.Equal(t, []UserDetailEntry{{UserID: "u1", Email: "test_email@example.com"}}, res)
	t.Logf("%v %T", res, res)
}

func TestRemote_UserDetailWithErrorResult(t *testing.T) {
	ts := testServer(t, `{"method":"store.user_detail","params":{"detail":"email","locator":{"url":"http://example.com/url"},"user_id":"username","update":"new_value@example.com"},"id":1}`, `{"error":"failed"}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	req := UserDetailRequest{Locator: store.Locator{URL: "http://example.com/url"}, UserID: "username", Detail: UserEmail, Update: "new_value@example.com"}
	_, err := c.UserDetail(req)
	assert.EqualError(t, err, "failed")
}

func TestRemote_Count(t *testing.T) {
	ts := testServer(t, `{"method":"store.count","params":{"locator":{"url":"http://example.com/url"},"since":"0001-01-01T00:00:00Z"},"id":1}`, `{"result":11}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	res, err := c.Count(FindRequest{Locator: store.Locator{URL: "http://example.com/url"}})
	assert.NoError(t, err)
	assert.Equal(t, 11, res)
}

func TestRemote_Delete(t *testing.T) {
	ts := testServer(t, `{"method":"store.delete","params":{"locator":{"url":"http://example.com/url"},"del_mode":0},"id":1}`,
		`{}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	err := c.Delete(DeleteRequest{Locator: store.Locator{URL: "http://example.com/url"}})
	assert.NoError(t, err)
}

func TestRemote_Close(t *testing.T) {
	ts := testServer(t, `{"method":"store.close","id":1}`, `{}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}
	err := c.Close()
	assert.NoError(t, err)
}

func testServer(t *testing.T, req, resp string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, req, string(body))
		t.Logf("req: %s", string(body))
		_, _ = fmt.Fprint(w, resp)
	}))
}
