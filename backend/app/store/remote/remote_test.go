package remote

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/store"
)

func TestClient_Create(t *testing.T) {
	ts := testServer(t, `{"method":"create","params":[{"id":"123","pid":"","text":"msg","user":{"name":"","id":"","picture":"","admin":false},"locator":{"site":"site","url":"http://example.com/url"},"score":0,"vote":0,"time":"0001-01-01T00:00:00Z"}]}`, `{"result":"12345"}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	res, err := c.Create(store.Comment{ID: "123", Locator: store.Locator{URL: "http://example.com/url", SiteID: "site"},
		Text: "msg"})
	assert.NoError(t, err)
	assert.Equal(t, "12345", res)
	t.Logf("%v %T", res, res)
}

func TestClient_Get(t *testing.T) {
	ts := testServer(t, `{"method":"get","params":[{"url":"http://example.com/url"},"site"]}`,
		`{"result":{"id":"123","pid":"","text":"msg","delete":true}}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	res, err := c.Get(store.Locator{URL: "http://example.com/url"}, "site")
	assert.NoError(t, err)
	assert.Equal(t, store.Comment{ID: "123", Text: "msg", Deleted: true}, res)
	t.Logf("%v %T", res, res)
}

func TestClient_GetWithErrorResult(t *testing.T) {
	ts := testServer(t, `{"method":"get","params":[{"url":"http://example.com/url"},"site"]}`, `{"error":"failed"}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	_, err := c.Get(store.Locator{URL: "http://example.com/url"}, "site")
	assert.EqualError(t, err, "failed")
}

func TestClient_GetWithErrorDecode(t *testing.T) {
	ts := testServer(t, `{"method":"get","params":[{"url":"http://example.com/url"},"site"]}`, ``)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	_, err := c.Get(store.Locator{URL: "http://example.com/url"}, "site")
	assert.EqualError(t, err, "failed to decode response for get: EOF")
}

func TestClient_GetWithErrorRemote(t *testing.T) {
	c := Client{API: "http://127.0.0.2", Client: http.Client{Timeout: 10 * time.Millisecond}}

	_, err := c.Get(store.Locator{URL: "http://example.com/url"}, "site")
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "remote call failed for get:"))
}

func TestClient_FailedStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		t.Logf("req: %s", string(body))
		w.WriteHeader(400)
	}))
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	_, err := c.Get(store.Locator{URL: "http://example.com/url"}, "site")
	assert.EqualError(t, err, "bad status 400 for get")
}

func TestClient_Put(t *testing.T) {
	ts := testServer(t, `{"method":"put","params":[{"url":"http://example.com/url"},{"id":"123","pid":"","text":"msg","user":{"name":"","id":"","picture":"","admin":false},"locator":{"site":"site123","url":"http://example.com/url"},"score":0,"vote":0,"time":"0001-01-01T00:00:00Z"}]}`, `{}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	err := c.Put(store.Locator{URL: "http://example.com/url"}, store.Comment{ID: "123",
		Locator: store.Locator{URL: "http://example.com/url", SiteID: "site123"}, Text: "msg"})
	assert.NoError(t, err)

}

func TestClient_Find(t *testing.T) {
	ts := testServer(t, `{"method":"find","params":[{"url":"http://example.com/url"},""]}`,
		`{"result":[{"text":"1"},{"text":"2"}]}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	res, err := c.Find(store.Locator{URL: "http://example.com/url"}, "")
	assert.NoError(t, err)
	assert.Equal(t, []store.Comment{{Text: "1"}, {Text: "2"}}, res)
}

func TestClient_Last(t *testing.T) {
	ts := testServer(t, `{"method":"last","params":["site1",100,"2019-06-06T19:34:10Z"]}`,
		`{"result":[{"text":"1"},{"text":"2"}]}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	res, err := c.Last("site1", 100, time.Date(2019, 6, 6, 19, 34, 10, 0, time.UTC))
	assert.NoError(t, err)
	assert.Equal(t, []store.Comment{{Text: "1"}, {Text: "2"}}, res)
}

func TestClient_User(t *testing.T) {
	ts := testServer(t, `{"method":"user","params":["site1","u1",100,4]}`, `{"result":[{"text":"1"},{"text":"2"}]}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	res, err := c.User("site1", "u1", 100, 4)
	assert.NoError(t, err)
	assert.Equal(t, []store.Comment{{Text: "1"}, {Text: "2"}}, res)
}

func testServer(t *testing.T, req, resp string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, req, string(body))
		t.Logf("req: %s", string(body))
		fmt.Fprintf(w, resp)
	}))
}
