/*
 * Copyright 2020 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package server

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-pkgz/jrpc"
	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark42/backend/app/store/admin"
)

func TestRPC_admKeyHndl(t *testing.T) {
	port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	ra := admin.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}
	key, err := ra.Key("any")
	assert.NoError(t, err)
	assert.Equal(t, "secret", key)
}

func TestRPC_admAdminsHndl(t *testing.T) {
	port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	ra := admin.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}
	_, err := ra.Admins("bad site")
	assert.EqualError(t, err, "site bad site not found")

	admins, err := ra.Admins("test-site")
	assert.NoError(t, err)
	assert.Equal(t, []string{"id1", "id2"}, admins)
}

func TestRPC_admEmailHndl(t *testing.T) {
	port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	ra := admin.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}
	_, err := ra.Admins("bad site")
	assert.EqualError(t, err, "site bad site not found")

	email, err := ra.Email("test-site")
	assert.NoError(t, err)
	assert.Equal(t, "admin@example.com", email)
}

func TestRPC_admEnabledHndl(t *testing.T) {
	port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	ra := admin.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}
	_, err := ra.Enabled("bad site")
	assert.EqualError(t, err, "site bad site not found")

	ok, err := ra.Enabled("test-site")
	assert.NoError(t, err)
	assert.Equal(t, true, ok)

	ok, err = ra.Enabled("test-site-disabled")
	assert.NoError(t, err)
	assert.Equal(t, false, ok)
}

func TestRPC_admEventHndl(t *testing.T) {
	port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	ra := admin.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}
	err := ra.OnEvent("bad site", admin.EvCreate)
	assert.EqualError(t, err, "site bad site not found")

	err = ra.OnEvent("test-site", admin.EvCreate)
	assert.NoError(t, err)
}
