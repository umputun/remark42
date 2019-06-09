package remote

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Call(t *testing.T) {
	ts := testServer(t, `{"method":"test","params":[123,"abc"]}`, `{"result":"12345"}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}
	resp, err := c.Call("test", 123, "abc")
	assert.NoError(t, err)
	res := ""
	err = json.Unmarshal(*resp.Result, &res)
	assert.Equal(t, "12345", res)
	t.Logf("%v %T", res, res)
}

func TestClient_CallError(t *testing.T) {
	ts := testServer(t, `{"method":"test","params":[123,"abc"]}`, `{"error":"some error"}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}
	_, err := c.Call("test", 123, "abc")
	assert.EqualError(t, err, "some error")
}

func TestClient_CallBadResponse(t *testing.T) {
	ts := testServer(t, `{"method":"test","params":[123,"abc"]}`, `{"result":"12345 invalid}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}
	_, err := c.Call("test", 123, "abc")
	assert.NotNil(t, err)
}

func TestClient_CallBadRemote(t *testing.T) {
	ts := testServer(t, `{"method":"test","params":[123,"abc"]}`, `{"result":"12345"}`)
	defer ts.Close()
	c := Client{API: "http://127.0.0.2", Client: http.Client{Timeout: 10 * time.Millisecond}}
	_, err := c.Call("test", 123)
	assert.NotNil(t, err)
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
