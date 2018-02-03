package rest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/app/store"
)

func TestAdmin_Delete(t *testing.T) {
	srv, port := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(srv)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}}

	id1 := addComment(t, c1, port)
	addComment(t, c2, port)

	client := http.Client{}
	req, err := http.NewRequest(http.MethodDelete,
		fmt.Sprintf("http://127.0.0.1:%d/api/v1/admin/comment/%s?site=radio-t&url=https://radio-t.com/blah", port, id1), nil)
	assert.Nil(t, err)
	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	_, code := get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/id/%s?site=radio-t&url=https://radio-t.com/blah", port, id1))
	assert.Equal(t, 400, code)
}

func TestAdmin_Pin(t *testing.T) {
	srv, port := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(srv)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah"}}

	id1 := addComment(t, c1, port)
	addComment(t, c2, port)

	pin := func(val int) int {
		client := http.Client{}
		req, err := http.NewRequest(http.MethodPut,
			fmt.Sprintf("http://127.0.0.1:%d/api/v1/admin/pin/%s?site=radio-t&url=https://radio-t.com/blah&pin=%d", port, id1, val),
			nil)
		assert.Nil(t, err)
		resp, err := client.Do(req)
		assert.Nil(t, err)
		return resp.StatusCode
	}

	code := pin(1)
	assert.Equal(t, 200, code)

	body, code := get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/id/%s?site=radio-t&url=https://radio-t.com/blah", port, id1))
	assert.Equal(t, 200, code)
	cr := store.Comment{}
	err := json.Unmarshal([]byte(body), &cr)
	assert.Nil(t, err)
	assert.True(t, cr.Pin)

	code = pin(-1)
	assert.Equal(t, 200, code)
	body, code = get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/id/%s?site=radio-t&url=https://radio-t.com/blah", port, id1))
	assert.Equal(t, 200, code)
	cr = store.Comment{}
	err = json.Unmarshal([]byte(body), &cr)
	assert.Nil(t, err)
	assert.False(t, cr.Pin)
}

func TestAdmin_Block(t *testing.T) {
	srv, port := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(srv)

	block := func(val int) (code int, body []byte) {
		client := http.Client{}
		req, err := http.NewRequest(http.MethodPut,
			fmt.Sprintf("http://127.0.0.1:%d/api/v1/admin/user/%s?site=radio-t&url=https://radio-t.com/blah&block=%d",
				port, "user1", val), nil)
		assert.Nil(t, err)
		resp, err := client.Do(req)
		require.Nil(t, err)
		body, err = ioutil.ReadAll(resp.Body)
		assert.Nil(t, err)
		resp.Body.Close()
		return resp.StatusCode, body
	}

	code, body := block(1)
	require.Equal(t, 200, code)
	j := JSON{}
	err := json.Unmarshal(body, &j)
	assert.Nil(t, err)
	assert.Equal(t, "user1", j["user_id"])
	assert.Equal(t, true, j["block"])
	assert.Equal(t, "radio-t", j["site_id"])

	code, body = block(-1)
	require.Equal(t, 200, code)
	err = json.Unmarshal(body, &j)
	assert.Nil(t, err)
	assert.Equal(t, false, j["block"])
}

func TestAdmin_Export(t *testing.T) {
	srv, port := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(srv)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com/blah2"}}

	addComment(t, c1, port)
	addComment(t, c2, port)

	body, code := get(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/admin/export?site=radio-t&mode=stream", port))
	assert.Equal(t, 200, code)
	assert.Equal(t, 2, strings.Count(body, "\n"))
	assert.Equal(t, 2, strings.Count(body, "\"text\""))
	t.Logf("%s", body)
}
