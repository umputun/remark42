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
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/engine"
)

func TestRPC_createHndl(t *testing.T) {
	port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	re := engine.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}
	id, err := re.Create(store.Comment{ID: "123456", Locator: store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		Text: "text 123", User: store.User{ID: "u1", Name: "user1"}})
	assert.NoError(t, err)
	assert.Equal(t, "123456", id)
}

func TestRPC_findHndl(t *testing.T) {
	port, teardown := prepTestStore(t)
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
	port, teardown := prepTestStore(t)
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
	port, teardown := prepTestStore(t)
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
	port, teardown := prepTestStore(t)
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
	port, teardown := prepTestStore(t)
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
	port, teardown := prepTestStore(t)
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
	port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	re := engine.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}

	c := store.Comment{ID: "123456", Locator: store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		Text: "text 123", User: store.User{ID: "u1", Name: "user1"}}
	id, err := re.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, "123456", id)

	// verify user
	verifyFlagReq := engine.FlagRequest{
		Flag:   engine.Verified,
		UserID: "u1",
		Locator: store.Locator{
			SiteID: "test-site",
		},
	}
	flags, err := re.ListFlags(verifyFlagReq)
	require.NoError(t, err)
	assert.Empty(t, flags)

	verifyFlagReq.Update = engine.FlagTrue
	status, err := re.Flag(verifyFlagReq)
	require.NoError(t, err)
	assert.Equal(t, true, status)

	flags, err = re.ListFlags(verifyFlagReq)
	require.NoError(t, err)
	assert.Equal(t, []interface{}{"u1"}, flags)
	verifiedUsers := make([]string, 0, len(flags))
	for _, v := range flags {
		verifiedUsers = append(verifiedUsers, v.(string))
	}
	assert.Equal(t, []string{"u1"}, verifiedUsers)

	// block user
	blockFlagReq := engine.FlagRequest{
		Flag:   engine.Blocked,
		UserID: "u1",
		Locator: store.Locator{
			SiteID: "test-site",
		},
		TTL: time.Hour,
	}
	flags, err = re.ListFlags(blockFlagReq)
	require.NoError(t, err)
	assert.Empty(t, flags)

	blockFlagReq.Update = engine.FlagTrue
	status, err = re.Flag(blockFlagReq)
	require.NoError(t, err)
	assert.Equal(t, true, status)

	flags, err = re.ListFlags(blockFlagReq)
	require.NoError(t, err)
	assert.NotEmpty(t, flags)
	blockedUsers := make([]store.BlockedUser, 0, len(flags))
	for _, v := range flags {
		blockedUsers = append(blockedUsers, v.(store.BlockedUser))
	}
	require.Equal(t, 1, len(blockedUsers))
	blockedUserInfo := blockedUsers[0]
	assert.Equal(t, "u1", blockedUserInfo.ID)
	assert.True(t, blockedUserInfo.Until.After(time.Now().Add(time.Minute*59)), "blocked duration is more than 59m away")
	assert.True(t, blockedUserInfo.Until.Before(time.Now().Add(time.Minute*61)), "blocked duration is less than 61m away")
}

func TestRPC_userDetailHndl(t *testing.T) {
	port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	re := engine.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}

	// add to entries to DB before we start
	result, err := re.UserDetail(engine.UserDetailRequest{Locator: store.Locator{SiteID: "test-site"}, UserID: "u1", Detail: engine.UserEmail, Update: "test@example.com"})
	assert.NoError(t, err, "No error inserting entry expected")
	assert.ElementsMatch(t, []engine.UserDetailEntry{{UserID: "u1", Email: "test@example.com"}}, result)
	result, err = re.UserDetail(engine.UserDetailRequest{Locator: store.Locator{SiteID: "test-site"}, UserID: "u2", Detail: engine.UserEmail, Update: "other@example.com"})
	assert.NoError(t, err, "No error inserting entry expected")
	assert.ElementsMatch(t, []engine.UserDetailEntry{{UserID: "u2", Email: "other@example.com"}}, result)

	// try to change existing entry with wrong SiteID
	result, err = re.UserDetail(engine.UserDetailRequest{Locator: store.Locator{SiteID: "bad"}, UserID: "u2", Detail: engine.UserEmail, Update: "not_relevant"})
	assert.NoError(t, err, "Updating existing entry with wrong SiteID doesn't produce error")
	assert.ElementsMatch(t, []engine.UserDetailEntry{}, result, "Updating existing entry with wrong SiteID doesn't change anything")

	// stateless tests without changing the state we set up before
	var testData = []struct {
		req      engine.UserDetailRequest
		error    string
		expected []engine.UserDetailEntry
	}{
		{req: engine.UserDetailRequest{Locator: store.Locator{SiteID: "test-site"}, UserID: "u1", Detail: engine.UserEmail},
			expected: []engine.UserDetailEntry{{UserID: "u1", Email: "test@example.com"}}},
		{req: engine.UserDetailRequest{Locator: store.Locator{SiteID: "bad"}, UserID: "u1", Detail: engine.UserEmail},
			expected: []engine.UserDetailEntry{}},
		{req: engine.UserDetailRequest{Locator: store.Locator{SiteID: "test-site"}, UserID: "u1xyz", Detail: engine.UserEmail},
			expected: []engine.UserDetailEntry{}},
		{req: engine.UserDetailRequest{Detail: engine.UserEmail, Update: "new_value"},
			error: `userid cannot be empty in request for single detail`},
		{req: engine.UserDetailRequest{Detail: engine.UserDetail("bad")},
			error: `unsupported detail "bad"`},
		{req: engine.UserDetailRequest{Update: "not_relevant", Detail: engine.AllUserDetails},
			error: `unsupported request with userdetail all`},
		{req: engine.UserDetailRequest{Locator: store.Locator{SiteID: "test-site"}, Detail: engine.AllUserDetails},
			expected: []engine.UserDetailEntry{{UserID: "u1", Email: "test@example.com"}, {UserID: "u2", Email: "other@example.com"}}},
	}

	for i, x := range testData {
		result, err := re.UserDetail(x.req)
		if x.error != "" {
			assert.EqualError(t, err, x.error, "Error should match expected for case %d", i)
		} else {
			assert.NoError(t, err, "Error is not expected expected for case %d", i)
		}
		assert.ElementsMatch(t, x.expected, result, "Result should match expected for case %d", i)
	}
}

func TestRPC_deleteHndl(t *testing.T) {
	port, teardown := prepTestStore(t)
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
	port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	re := engine.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}
	assert.NoError(t, re.Close())
	assert.NoError(t, re.Close(), "second call should not result in panic or errors")
}
