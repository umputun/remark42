/*
 * Copyright 2019 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package admin

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-pkgz/jrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemote_Key(t *testing.T) {
	ts := testServer(t, `{"method":"admin.key","params":"any","id":1}`,
		`{"result":"12345","params":"any","id":1}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	var a Store = &c
	_ = a

	res, err := c.Key("any")
	assert.NoError(t, err)
	assert.Equal(t, "12345", res)
	t.Logf("%v %T", res, res)
}

func TestRemote_Admins(t *testing.T) {
	ts := testServer(t, `{"method":"admin.admins","params":"site-1","id":1}`,
		`{"result":["id1","id2"],"id":1}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	var a Store = &c
	_ = a

	res, err := c.Admins("site-1")
	assert.NoError(t, err)
	assert.Equal(t, []string{"id1", "id2"}, res)
	t.Logf("%v %T", res, res)
}

func TestRemote_Email(t *testing.T) {
	ts := testServer(t, `{"method":"admin.email","params":"site-1","id":1}`,
		`{"result":"bbb@example.com","id":1}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	var a Store = &c
	_ = a

	res, err := c.Email("site-1")
	assert.NoError(t, err)
	assert.Equal(t, "bbb@example.com", res)
	t.Logf("%v %T", res, res)
}

func TestRemote_Enables(t *testing.T) {
	ts := testServer(t, `{"method":"admin.enabled","params":"site-1","id":1}`,
		`{"result":true,"id":1}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	var a Store = &c
	_ = a

	res, err := c.Enabled("site-1")
	assert.NoError(t, err)
	assert.True(t, res)
	t.Logf("%v %T", res, res)
}

func TestRemote_OnEvent(t *testing.T) {
	ts := testServer(t, `{"method":"admin.event","params":["site-1",2],"id":1}`, `{"id":1}`)
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	var a Store = &c
	_ = a

	err := c.OnEvent("site-1", EvUpdate)
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
