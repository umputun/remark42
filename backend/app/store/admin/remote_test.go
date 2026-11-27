/*
 * Copyright 2019 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package admin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
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

func TestRemote_KeyHidesStoreAddress(t *testing.T) {
	c := RPC{Client: jrpc.Client{API: "http://admin-store.internal:9191/rpc", Client: refusingClient()}}
	_, err := c.Key("any")
	assert.EqualError(t, err, "remote call failed for admin.key")
}

// refusingClient returns a client whose every connection fails as a refused one would, with the address it dialed
func refusingClient() http.Client {
	return http.Client{Transport: &http.Transport{DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
		return nil, &net.OpError{Op: "dial", Net: network, Addr: &net.TCPAddr{IP: net.IPv4(10, 1, 2, 3), Port: 9191},
			Err: errors.New("connect: connection refused")}
	}}}
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
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, req, string(body))
		t.Logf("req: %s", string(body))
		_, _ = fmt.Fprint(w, resp)
	}))
}

// a remote admin store written in Go answers with null for a nil slice: the admins list reads that as
// empty, a call that has to return a value fails
func TestRemote_NullResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"result":null,"id":1}`)
	}))
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	admins, err := c.Admins("site")
	require.NoError(t, err)
	assert.Empty(t, admins)

	_, err = c.Key("site")
	require.ErrorIs(t, err, errNoResult, "no key must not become an empty signing key")

	_, err = c.Email("site")
	require.ErrorIs(t, err, errNoResult)

	_, err = c.Enabled("site")
	require.ErrorIs(t, err, errNoResult)
}

func TestRemote_KeyRejectsEmpty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"result":"","id":1}`)
	}))
	defer ts.Close()
	c := RPC{Client: jrpc.Client{API: ts.URL, Client: http.Client{}}}

	_, err := c.Key("site")
	require.EqualError(t, err, "remote admin store returned an empty key")
}
