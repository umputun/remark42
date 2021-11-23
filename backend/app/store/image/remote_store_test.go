package image

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-pkgz/jrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemote_SaveWithID(t *testing.T) {
	ts := testServer(t, fmt.Sprintf(`{"method":"image.save_with_id","params":["54321",%q],"id":1}`, gopher),
		`{"id":1}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	var a Store = &c
	_ = a

	err := c.Save("54321", gopherPNGBytes())
	assert.NoError(t, err)
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

func TestRemote_ResetCleanupTimer(t *testing.T) {
	ts := testServer(t, `{"method":"image.reset_cleanup_timer","params":"gopher_id","id":1}`, `{"id":1}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	var a Store = &c
	_ = a

	err := c.ResetCleanupTimer("gopher_id")
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

func TestRemote_Info(t *testing.T) {
	ts := testServer(t, `{"method":"image.info","id":1}`,
		`{"result":{"FirstStagingImageTS":"0001-01-01T00:00:01Z"},"id":1}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	var a Store = &c
	_ = a

	info, err := c.Info()
	assert.NoError(t, err)
	assert.False(t, info.FirstStagingImageTS.IsZero())
}

func testServer(t *testing.T, req, resp string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, req, string(body))
		_, _ = fmt.Fprint(w, resp)
	}))
}
