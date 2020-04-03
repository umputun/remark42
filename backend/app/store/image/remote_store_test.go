package image

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-pkgz/jrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemote_Save(t *testing.T) {
	ts := testServer(t, fmt.Sprintf(`{"method":"image.save","params":["admin","%s"],"id":1}`, gopher),
		`{"result":"12345","id":1}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	var a Store = &c
	_ = a

	res, err := c.Save("admin", gopherPNGBytes())
	assert.NoError(t, err)
	assert.Equal(t, "12345", res)
}

func TestRemote_SaveWithID(t *testing.T) {
	ts := testServer(t, fmt.Sprintf(`{"method":"image.save_with_id","params":["54321","%s"],"id":1}`, gopher),
		`{"result":"12345","id":1}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	var a Store = &c
	_ = a

	res, err := c.SaveWithID("54321", gopherPNGBytes())
	assert.NoError(t, err)
	assert.Equal(t, "12345", res)
}

func TestRemote_Load(t *testing.T) {
	ts := testServer(t, `{"method":"image.load","params":"54321","id":1}`,
		fmt.Sprintf(`{"result":"%v","id":1}`, gopher))
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	var a Store = &c
	_ = a

	res, err := c.Load("54321")
	assert.NoError(t, err)
	assert.Equal(t, gopherPNGBytes(), res)
}

func TestRemote_Commit(t *testing.T) {
	ts := testServer(t, `{"method":"image.commit","params":"gopher_id","id":1}`, `{"id":1}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	var a Store = &c
	_ = a

	err := c.Commit("gopher_id")
	assert.NoError(t, err)
}

func TestRemote_Cleanup(t *testing.T) {
	ts := testServer(t, `{"method":"image.cleanup","params":60000000000,"id":1}`, `{"id":1}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	var a Store = &c
	_ = a

	err := c.Cleanup(context.TODO(), time.Minute)
	assert.NoError(t, err)
}

func testServer(t *testing.T, req, resp string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, req, string(body))
		_, _ = fmt.Fprint(w, resp)
	}))
}
