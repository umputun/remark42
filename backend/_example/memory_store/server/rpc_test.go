/*
 * Copyright 2019 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package server

import (
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/go-pkgz/jrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/admin"
	"github.com/umputun/remark/backend/app/store/engine"
	"github.com/umputun/remark/memory_store/accessor"
)

func TestRPC_createHndl(t *testing.T) {
	_, port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	re := engine.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}
	id, err := re.Create(store.Comment{ID: "123456", Locator: store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		Text: "text 123", User: store.User{ID: "u1", Name: "user1"}})
	assert.NoError(t, err)
	assert.Equal(t, "123456", id)
}

func TestRPC_findHndl(t *testing.T) {
	_, port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	re := engine.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}
	findReq := engine.FindRequest{Locator: store.Locator{SiteID: "test-site", URL: "http://example.com/post1"}}
	comments, err := re.Find(findReq)
	require.NoError(t, err)
	assert.Equal(t, 0, len(comments))

	c := store.Comment{ID: "123456", Locator: store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		Text: "text 123", User: store.User{ID: "u1", Name: "user1"}}
	id, err := re.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, "123456", id)

	comments, err = re.Find(findReq)
	require.NoError(t, err)
	assert.Equal(t, 1, len(comments))
	assert.Equal(t, c, comments[0])
}

func TestRPC_getHndl(t *testing.T) {
	_, port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	re := engine.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}
	req := engine.GetRequest{
		Locator:   store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		CommentID: "123456",
	}

	_, err := re.Get(req)
	assert.EqualError(t, err, "not found")

	c := store.Comment{ID: "123456", Locator: store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		Text: "text 123", User: store.User{ID: "u1", Name: "user1"}}
	_, err = re.Create(c)
	assert.NoError(t, err)

	comment, err := re.Get(req)
	assert.NoError(t, err)
	assert.Equal(t, c, comment)
}

func TestRPC_updateHndl(t *testing.T) {
	_, port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	re := engine.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}

	c := store.Comment{ID: "123456", Locator: store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		Text: "text 123", User: store.User{ID: "u1", Name: "user1"}}
	err := re.Update(c)
	assert.EqualError(t, err, "not found")

	_, err = re.Create(c)
	assert.NoError(t, err)

	c.Text = "updates"
	err = re.Update(c)
	assert.NoError(t, err)

	req := engine.GetRequest{
		Locator:   store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		CommentID: "123456",
	}
	comment, err := re.Get(req)
	assert.NoError(t, err)
	assert.Equal(t, c, comment)
}

func TestRPC_countHndl(t *testing.T) {
	_, port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	re := engine.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}
	findReq := engine.FindRequest{Locator: store.Locator{SiteID: "test-site", URL: "http://example.com/post1"}}
	count, err := re.Count(findReq)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	c := store.Comment{ID: "123456", Locator: store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		Text: "text 123", User: store.User{ID: "u1", Name: "user1"}}
	id, err := re.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, "123456", id)

	count, err = re.Count(findReq)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestRPC_infoHndl(t *testing.T) {
	_, port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	re := engine.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}

	c := store.Comment{ID: "123456", Locator: store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		Text: "text 123", User: store.User{ID: "u1", Name: "user1"}}
	id, err := re.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, "123456", id)

	infoReq := engine.InfoRequest{Locator: store.Locator{SiteID: "test-site", URL: "http://example.com/post1"}}
	info, err := re.Info(infoReq)
	require.NoError(t, err)
	assert.Equal(t, 1, len(info))
	i := info[0]
	assert.Equal(t, store.PostInfo{URL: "http://example.com/post1", Count: 1}, i)
}

func TestRPC_flagHndl(t *testing.T) {
	_, port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	re := engine.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}

	c := store.Comment{ID: "123456", Locator: store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		Text: "text 123", User: store.User{ID: "u1", Name: "user1"}}
	id, err := re.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, "123456", id)

	flagReq := engine.FlagRequest{
		Flag: engine.Verified,
		Locator: store.Locator{
			SiteID: "test-site",
		},
		UserID: "u1",
	}
	status, err := re.Flag(flagReq)
	require.NoError(t, err)
	assert.Equal(t, false, status)

	flagReq.Update = engine.FlagTrue
	status, err = re.Flag(flagReq)
	require.NoError(t, err)
	assert.Equal(t, true, status)

	flagReq.Update = engine.FlagNonSet
	status, err = re.Flag(flagReq)
	require.NoError(t, err)
	assert.Equal(t, true, status)
}

func TestRPC_listFlagsHndl(t *testing.T) {
	_, port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	re := engine.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}

	c := store.Comment{ID: "123456", Locator: store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		Text: "text 123", User: store.User{ID: "u1", Name: "user1"}}
	id, err := re.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, "123456", id)

	flagReq := engine.FlagRequest{
		Flag:   engine.Verified,
		UserID: "u1",
		Locator: store.Locator{
			SiteID: "test-site",
		},
	}
	flags, err := re.ListFlags(flagReq)
	require.NoError(t, err)
	assert.Equal(t, []interface{}{}, flags)

	flagReq.Update = engine.FlagTrue
	status, err := re.Flag(flagReq)
	require.NoError(t, err)
	assert.Equal(t, true, status)

	flags, err = re.ListFlags(flagReq)
	require.NoError(t, err)
	assert.Equal(t, []interface{}{"u1"}, flags)
}

func TestRPC_userDetailHndl(t *testing.T) {
	_, port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	re := engine.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}

	c := store.Comment{ID: "123456", Locator: store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		Text: "text 123", User: store.User{ID: "u1", Name: "user1"}}
	id, err := re.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, "123456", id)

	userDetailRequest := engine.UserDetailRequest{
		Detail: engine.Email,
		Locator: store.Locator{
			SiteID: "test-site",
		},
		UserID: "u1",
	}
	status, err := re.UserDetail(userDetailRequest)
	require.NoError(t, err)
	// TODO should return false
	//assert.Equal(t, false, status)

	userDetailRequest.Update = "test@gmail.com"
	status, err = re.UserDetail(userDetailRequest)
	require.NoError(t, err)
	assert.Equal(t, true, status)

	userDetailRequest.Update = "other_test@gmail.com"
	status, err = re.UserDetail(userDetailRequest)
	require.NoError(t, err)
	assert.Equal(t, true, status)
}

func TestRPC_deleteHndl(t *testing.T) {
	_, port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	re := engine.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}
	req := engine.DeleteRequest{
		Locator:   store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		CommentID: "123456",
	}

	err := re.Delete(req)
	assert.EqualError(t, err, "not found")

	c := store.Comment{ID: "123456", Locator: store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		Text: "text 123", User: store.User{ID: "u1", Name: "user1"}}
	_, err = re.Create(c)
	assert.NoError(t, err)

	err = re.Delete(req)
	assert.NoError(t, err)
}

func TestRPC_closeHndl(t *testing.T) {
	_, port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	re := engine.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}
	err := re.Close()
	assert.NoError(t, err)
}
func TestRPC_admKeyHndl(t *testing.T) {
	_, port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	ra := admin.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}
	key, err := ra.Key()
	assert.NoError(t, err)
	assert.Equal(t, "secret", key)
}

func TestRPC_admAdminsHndl(t *testing.T) {
	_, port, teardown := prepTestStore(t)
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
	_, port, teardown := prepTestStore(t)
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
	_, port, teardown := prepTestStore(t)
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
	_, port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	ra := admin.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}
	err := ra.OnEvent("bad site", admin.EvCreate)
	assert.EqualError(t, err, "site bad site not found")

	err = ra.OnEvent("test-site", admin.EvCreate)
	assert.NoError(t, err)
}

func prepTestStore(t *testing.T) (s *RPC, port int, teardown func()) {
	port = 40000 + int(rand.Int31n(10000))

	mg := accessor.NewMemData()
	adm := accessor.NewMemAdminStore("secret")
	s = NewRPC(mg, adm, &jrpc.Server{API: "/test", Logger: jrpc.NoOpLogger})

	admRec := accessor.AdminRec{
		SiteID:  "test-site",
		IDs:     []string{"id1", "id2"},
		Email:   "admin@example.com",
		Enabled: true,
	}
	adm.Set("test-site", admRec)

	admRecDisabled := admRec
	admRecDisabled.Enabled = false
	adm.Set("test-site-disabled", admRecDisabled)

	go func() {
		t.Log(s.Run(port))
	}()
	time.Sleep(time.Millisecond * 10)

	return s, port, func() {
		require.NoError(t, s.Shutdown())
	}
}
