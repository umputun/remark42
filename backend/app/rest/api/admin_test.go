package api

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-pkgz/auth/v2/token"
	cache "github.com/go-pkgz/lcw/v2"
	R "github.com/go-pkgz/rest"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"

	"github.com/umputun/remark42/backend/app/store"
	adminstore "github.com/umputun/remark42/backend/app/store/admin"
	"github.com/umputun/remark42/backend/app/store/engine"
	"github.com/umputun/remark42/backend/app/store/image"
	"github.com/umputun/remark42/backend/app/store/service"
)

func TestAdmin_Delete(t *testing.T) {
	ts, _ := startupT(t)

	c1 := store.Comment{Text: "test test #1", User: store.User{ID: "id", Name: "name"},
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}
	c2 := store.Comment{Text: "test test #2", User: store.User{ID: "id", Name: "name"},
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}

	id1 := addComment(t, c1, ts)
	addComment(t, c2, ts)

	// check last comments
	comments := callJSON[[]store.Comment](t, http.MethodGet, ts.URL+"/api/v1/last/2?site=remark42", "", "")
	assert.Equal(t, 2, len(comments), "should have 2 comments")

	// check multi count
	j := callJSON[[]store.PostInfo](t, http.MethodPost, ts.URL+"/api/v1/counts?site=remark42", `["https://radio-t.com/blah","https://radio-t.com/blah2"]`, adminPasswd)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/blah", Count: 2},
		{URL: "https://radio-t.com/blah2", Count: 0}}, j)

	// delete a comment
	_, code := callAdminOnly(t, http.MethodDelete, fmt.Sprintf("%s/api/v1/admin/comment/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1),
		adminUmputunToken)
	assert.Equal(t, http.StatusOK, code)

	cr := callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1), "", devToken)
	assert.Equal(t, "", cr.Text)
	assert.True(t, cr.Deleted)

	// the last-comments list refreshes asynchronously after the delete. the polling closure runs
	// off the test goroutine, so it asserts on the CollectT it is handed rather than on t, which
	// also puts the real transport or decode error in the failure message
	pollClient := http.Client{Timeout: waitTimeout}
	defer pollClient.CloseIdleConnections()
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		lastResp, gErr := pollClient.Get(ts.URL + "/api/v1/last/2?site=remark42")
		if !assert.NoError(c, gErr) {
			return
		}
		defer lastResp.Body.Close()
		assert.Equal(c, http.StatusOK, lastResp.StatusCode)
		last := []store.Comment{}
		assert.NoError(c, json.NewDecoder(lastResp.Body).Decode(&last))
		assert.Len(c, last, 1, "should have 1 comments")
	}, waitTimeout, httpPoll)

	// check count updated
	b := callJSON[map[string]any](t, http.MethodGet, ts.URL+"/api/v1/count?site=remark42&url=https://radio-t.com/blah", "", "")
	t.Logf("%#v", b)
	assert.Equal(t, 1.0, b["count"], "should report 1 comments")

	// check multi count updated
	j = callJSON[[]store.PostInfo](t, http.MethodPost, ts.URL+"/api/v1/counts?site=remark42", `["https://radio-t.com/blah","https://radio-t.com/blah2"]`, adminPasswd)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/blah", Count: 1},
		{URL: "https://radio-t.com/blah2", Count: 0}}, j)
}

func TestAdmin_Title(t *testing.T) {
	ts, srv := startupT(t)

	srv.DataService.TitleExtractor = service.NewTitleExtractor(http.Client{Timeout: time.Second}, []string{"127.0.0.1"})
	tss := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == "/post1" {
			_, err := w.Write([]byte("<html><title>post1 blah 123</title><body> 2222</body></html>"))
			assert.NoError(t, err)
			return
		}
		if r.URL.String() == "/post2" {
			_, err := w.Write([]byte("<html><title>post2 blah 123</title><body> 2222</body></html>"))
			assert.NoError(t, err)
			return
		}
		w.WriteHeader(404)
	}))
	defer tss.Close()

	c1 := store.Comment{Text: "test test #1", User: store.User{ID: "id", Name: "name"},
		Locator: store.Locator{SiteID: "remark42", URL: tss.URL + "/post1"}}
	c2 := store.Comment{Text: "test test #2", User: store.User{ID: "id", Name: "name"},
		Locator: store.Locator{SiteID: "remark42", URL: tss.URL + "/post2"}}

	id1 := addComment(t, c1, ts)
	addComment(t, c2, ts)

	_, code := callAdminOnly(t, http.MethodPut, fmt.Sprintf("%s/api/v1/admin/title/%s?site=remark42&url=%s/post1", ts.URL, id1, tss.URL),
		adminUmputunToken)
	assert.Equal(t, http.StatusOK, code)

	cr := callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=%s/post1", ts.URL, id1, tss.URL), "", "")
	assert.Equal(t, "post1 blah 123", cr.PostTitle)
}

// TestAdmin_TitleKeptWhenPageFails runs the title update the way cleanup --title does, one request per
// comment of a post, against a page that can no longer be read: every comment keeps the title it has
func TestAdmin_TitleKeptWhenPageFails(t *testing.T) {
	ts, srv := startupT(t)

	var pageDown atomic.Bool
	tss := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if pageDown.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err := w.Write([]byte("<html><title>post1 blah 123</title><body> 2222</body></html>"))
		assert.NoError(t, err)
	}))
	defer tss.Close()

	srv.DataService.TitleExtractor = service.NewTitleExtractor(http.Client{Timeout: time.Second}, []string{"127.0.0.1"})
	ids := make([]string, 3)
	for i := range ids {
		ids[i] = addComment(t, store.Comment{Text: fmt.Sprintf("test test #%d", i), User: store.User{ID: "id", Name: "name"},
			Locator: store.Locator{SiteID: "remark42", URL: tss.URL + "/post1"}}, ts)
	}

	// the page stops answering, and the title cache no longer holds its title
	pageDown.Store(true)
	require.NoError(t, srv.DataService.TitleExtractor.Close())
	srv.DataService.TitleExtractor = service.NewTitleExtractor(http.Client{Timeout: time.Second}, []string{"127.0.0.1"})

	for _, id := range ids {
		_, code := call(t, http.MethodPut,
			fmt.Sprintf("%s/api/v1/admin/title/%s?site=remark42&url=%s/post1", ts.URL, id, tss.URL), "", adminUmputunToken)
		assert.Equal(t, http.StatusInternalServerError, code, "title of comment %s not updated", id)
	}

	for _, id := range ids {
		cr := callJSON[store.Comment](t, http.MethodGet,
			fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=%s/post1", ts.URL, id, tss.URL), "", "")
		assert.Equal(t, "post1 blah 123", cr.PostTitle, "title of comment %s kept", id)
	}
}

func TestAdmin_DeleteUser(t *testing.T) {
	ts, srv := startupT(t)

	c1 := store.Comment{Text: "test test #1", Orig: "o test test #1", User: store.User{ID: "id1", Name: "name"},
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}
	c2 := store.Comment{Text: "test test #2", Orig: "o test test #2", User: store.User{ID: "id2", Name: "name"}, ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}
	c3 := store.Comment{Text: "test test #3", Orig: "o test test #3", User: store.User{ID: "id2", Name: "name"}, ParentID: "",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}

	// write comments directly to store to keep user id
	id1, err := srv.DataService.Create(c1)
	assert.NoError(t, err)
	_, err = srv.DataService.Create(c2)
	assert.NoError(t, err)
	_, err = srv.DataService.Create(c3)
	assert.NoError(t, err)

	_, code := callAdminOnly(t, http.MethodDelete, fmt.Sprintf("%s/api/v1/admin/user/%s?site=remark42", ts.URL, "id2"),
		adminUmputunToken)
	assert.Equal(t, http.StatusOK, code)

	// all 3 comments here, but for id2 they deleted
	cmntWithInfo := callJSON[commentsWithInfo](t, http.MethodGet,
		ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah&sort=+time", "", "")
	require.Equal(t, 3, len(cmntWithInfo.Comments), "should have 3 comment")

	// id1 comment untouched
	assert.Equal(t, id1, cmntWithInfo.Comments[0].ID)
	assert.Equal(t, "o test test #1", cmntWithInfo.Comments[0].Orig)
	assert.False(t, cmntWithInfo.Comments[0].Deleted)
	t.Logf("%+v", cmntWithInfo.Comments[0].User)

	// id2 comments fully deleted
	assert.Equal(t, "", cmntWithInfo.Comments[1].Text)
	assert.Equal(t, "", cmntWithInfo.Comments[1].Orig)
	assert.Equal(t, store.User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, cmntWithInfo.Comments[1].User)
	assert.True(t, cmntWithInfo.Comments[1].Deleted)

	assert.Equal(t, "", cmntWithInfo.Comments[2].Text)
	assert.Equal(t, "", cmntWithInfo.Comments[2].Orig)
	assert.Equal(t, store.User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, cmntWithInfo.Comments[2].User)
	assert.True(t, cmntWithInfo.Comments[2].Deleted)
}

// an admin deleting a comment removes the author's own upload and keeps another user's upload the comment showed
func TestAdmin_DeleteCommentPictures(t *testing.T) {
	ts, srv := startupT(t)

	fs := &image.FileSystem{Location: t.TempDir(), Staging: t.TempDir()}
	imgSvc := image.NewService(fs, image.ServiceParams{ImageAPI: srv.RemarkURL + "/api/v1/picture/"})
	defer imgSvc.Close(context.Background())
	srv.DataService.ImageService = imgSvc

	for _, id := range []string{"user1/own.png", "user2/other.png"} {
		require.NoError(t, fs.Save(id, []byte("image data")))
		require.NoError(t, fs.Commit(id))
	}
	now := time.Now()
	for _, c := range []store.Comment{
		{ID: "c1", Text: `<img src="` + srv.RemarkURL + `/api/v1/picture/user1/own.png"/> <img src="` + srv.RemarkURL +
			`/api/v1/picture/user2/other.png"/>`, Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"},
			User: store.User{ID: "user1", Name: "user1"}, Timestamp: now},
		{ID: "c2", Text: `<img src="` + srv.RemarkURL + `/api/v1/picture/user2/other.png"/>`,
			Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah2"},
			User:    store.User{ID: "user2", Name: "user2"}, Timestamp: now.Add(time.Second)},
	} {
		_, err := srv.DataService.Engine.Create(c) // directly to the engine, the pictures are already committed
		require.NoError(t, err)
	}

	_, code := call(t, http.MethodDelete,
		ts.URL+"/api/v1/admin/comment/c1?site=remark42&url=https://radio-t.com/blah", "", adminPasswd)
	require.Equal(t, http.StatusOK, code)

	_, err := fs.Load("user1/own.png")
	assert.Error(t, err, "the author's own upload removed")
	data, err := fs.Load("user2/other.png")
	require.NoError(t, err, "another user's upload kept, though the deleted comment showed it")
	assert.Equal(t, "image data", string(data))
}

// every route that deletes a user's comments removes the pictures used in them and keeps those of other users
func TestAdmin_DeleteUserPictures(t *testing.T) {
	deleteMeReq := func(t *testing.T, srv *Rest, tsURL string) (method, url string) {
		return http.MethodGet, tsURL + "/api/v1/admin/deleteme?token=" + deleteMeToken(t, srv, "remark42", "user1", "")
	}
	tbl := []struct {
		name  string
		reqFn func(t *testing.T, srv *Rest, tsURL string) (method, url string)
	}{
		{name: "admin delete user", reqFn: func(_ *testing.T, _ *Rest, tsURL string) (method, url string) {
			return http.MethodDelete, tsURL + "/api/v1/admin/user/user1?site=remark42"
		}},
		{name: "delete me", reqFn: deleteMeReq},
		{name: "permanent block", reqFn: func(_ *testing.T, _ *Rest, tsURL string) (method, url string) {
			return http.MethodPut, tsURL + "/api/v1/admin/user/user1?site=remark42&block=1"
		}},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			ts, srv := startupT(t)

			fs := &image.FileSystem{Location: t.TempDir(), Staging: t.TempDir()}
			imgSvc := image.NewService(fs, image.ServiceParams{ImageAPI: srv.RemarkURL + "/api/v1/picture/"})
			defer imgSvc.Close(context.Background())
			srv.DataService.ImageService = imgSvc

			for _, id := range []string{"user1/own.png", "user2/other.png"} {
				require.NoError(t, fs.Save(id, []byte("image data")))
				require.NoError(t, fs.Commit(id))
			}
			// distinct timestamps: bolt keys a user's comment references by timestamp alone
			now := time.Now()
			locator := func(page string) store.Locator {
				return store.Locator{SiteID: "remark42", URL: "https://radio-t.com/" + page}
			}
			for _, c := range []store.Comment{
				{ID: "c1", Text: `<img src="` + srv.RemarkURL + `/api/v1/picture/user1/own.png"/>`, Locator: locator("blah"),
					User: store.User{ID: "user1", Name: "user1"}, Timestamp: now},
				{ID: "c2", Text: `<img src="` + srv.RemarkURL + `/api/v1/picture/user2/other.png"/>`, Locator: locator("blah2"),
					User: store.User{ID: "user2", Name: "user2"}, Timestamp: now.Add(time.Second)},
				// user1 shows user2's upload on a page user2 has not commented on
				{ID: "c3", Text: `<img src="` + srv.RemarkURL + `/api/v1/picture/user2/other.png"/>`, Locator: locator("blah3"),
					User: store.User{ID: "user1", Name: "user1"}, Timestamp: now.Add(2 * time.Second)},
			} {
				_, err := srv.DataService.Engine.Create(c) // directly to the engine, the picture is already committed
				require.NoError(t, err)
			}

			method, url := tt.reqFn(t, srv, ts.URL)
			_, code := call(t, method, url, "", adminPasswd)
			require.Equal(t, http.StatusOK, code)

			_, err := fs.Load("user1/own.png")
			assert.Error(t, err, "picture from the deleted user's comment removed")
			data, err := fs.Load("user2/other.png")
			require.NoError(t, err, "another user's upload kept, though the deleted user's comment showed it")
			assert.Equal(t, "image data", string(data))
		})
	}
}

func TestAdmin_DeleteUserAvatar(t *testing.T) {
	for _, avatarExists := range []bool{true, false} {
		t.Run(fmt.Sprintf("avatar exists %v", avatarExists), func(t *testing.T) {
			ts, srv := startupT(t)

			avatarFile := testAvatarDir(srv) + "/42/pic.image"
			if avatarExists {
				require.NoError(t, os.MkdirAll(filepath.Dir(avatarFile), 0o700))
				require.NoError(t, os.WriteFile(avatarFile, []byte("some image data"), 0o600))
			}
			c := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"},
				User: store.User{Name: "user1", ID: "user1", Picture: "https://demo.remark42.com/api/v1/avatar/pic.image"}}
			_, err := srv.DataService.Create(c)
			require.NoError(t, err)

			_, code := call(t, http.MethodDelete, ts.URL+"/api/v1/admin/user/user1?site=remark42", "", adminPasswd)
			assert.Equal(t, http.StatusOK, code, "a missing avatar does not fail the deletion")
			assert.NoFileExists(t, avatarFile, "user's avatar removed on admin user deletion")
		})
	}
}

// a failed avatar removal leaves the user in place, so a retry still finds the avatar in the comments,
// and a successful deletion flushes the cached comments
func TestAdmin_DeleteUserAvatarRemoveError(t *testing.T) {
	lru, err := cache.NewLruCache(cache.NewOpts[[]byte]().MaxKeys(100))
	require.NoError(t, err)
	ts, srv := startupT(t, func(srv *Rest) { srv.Cache = cache.NewScache[[]byte](lru) })

	c := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"},
		User: store.User{Name: "user1", ID: "user1", Picture: "https://demo.remark42.com/api/v1/avatar/pic.image"}}
	_, err = srv.DataService.Create(c)
	require.NoError(t, err)

	findText := func() string {
		cmnts := callJSON[commentsWithInfo](t, http.MethodGet,
			ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah", "", "")
		require.Len(t, cmnts.Comments, 1)
		return cmnts.Comments[0].Text
	}
	deleteUser := func() int {
		_, code := call(t, http.MethodDelete, ts.URL+"/api/v1/admin/user/user1?site=remark42", "", adminPasswd)
		return code
	}
	require.Contains(t, findText(), "test test #1") // cached from here on

	// a non-empty directory where the avatar file is expected makes Store.Remove fail with a real error
	avatarFile := testAvatarDir(srv) + "/42/pic.image"
	require.NoError(t, os.MkdirAll(avatarFile, 0o700))
	require.NoError(t, os.WriteFile(avatarFile+"/child", []byte("x"), 0o600))

	assert.Equal(t, http.StatusInternalServerError, deleteUser(), "avatar-store failure surfaces")
	comments, err := srv.DataService.User("remark42", "user1", 0, 0, store.User{})
	require.NoError(t, err, "comments stay when the avatar removal fails")
	require.Len(t, comments, 1)

	// the store works again: the retry removes the avatar and the comments
	require.NoError(t, os.RemoveAll(avatarFile))
	require.NoError(t, os.WriteFile(avatarFile, []byte("some image data"), 0o600))
	assert.Equal(t, http.StatusOK, deleteUser())
	assert.NoFileExists(t, avatarFile, "avatar removed on retry")
	assert.Empty(t, findText(), "cached comments flushed after the deletion")
}

// the comments are deleted before the avatar, so a failed avatar removal must not leave them in the cache
func TestAdmin_DeleteMeRequestAvatarRemoveErrorFlushesCache(t *testing.T) {
	lru, err := cache.NewLruCache(cache.NewOpts[[]byte]().MaxKeys(100))
	require.NoError(t, err)
	ts, srv := startupT(t, func(srv *Rest) { srv.Cache = cache.NewScache[[]byte](lru) })

	c := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"},
		User: store.User{Name: "user1", ID: "user1"}}
	_, err = srv.DataService.Create(c)
	require.NoError(t, err)

	findText := func() string {
		cmnts := callJSON[commentsWithInfo](t, http.MethodGet,
			ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah", "", "")
		require.Len(t, cmnts.Comments, 1)
		return cmnts.Comments[0].Text
	}
	require.Contains(t, findText(), "test test #1") // cached from here on

	// a non-empty directory where the avatar file is expected makes Store.Remove fail with a real error
	avatarFile := testAvatarDir(srv) + "/42/pic.image"
	require.NoError(t, os.MkdirAll(avatarFile, 0o700))
	require.NoError(t, os.WriteFile(avatarFile+"/child", []byte("x"), 0o600))

	tkn := deleteMeToken(t, srv, "remark42", "user1", "https://demo.remark42.com/api/v1/avatar/pic.image")
	_, code := call(t, http.MethodGet, ts.URL+"/api/v1/admin/deleteme?token="+tkn, "", adminPasswd)
	assert.Equal(t, http.StatusInternalServerError, code, "avatar-store failure surfaces")
	assert.Empty(t, findText(), "cached comments flushed after the deletion")
}

func TestAdmin_Pin(t *testing.T) {
	ts, _ := startupT(t)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}
	c2 := store.Comment{Text: "test test #2",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}

	id1 := addComment(t, c1, ts)
	addComment(t, c2, ts)

	pin := func(val int) int {
		_, code := callAdminOnly(t, http.MethodPut, fmt.Sprintf("%s/api/v1/admin/pin/%s?site=remark42&url=https://radio-t.com/blah&pin=%d", ts.URL, id1, val),
			adminPasswd)
		return code
	}

	code := pin(1)
	assert.Equal(t, http.StatusOK, code)

	cr := callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1), "", "")
	assert.True(t, cr.Pin)

	code = pin(-1)
	assert.Equal(t, http.StatusOK, code)
	cr = callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1), "", "")
	assert.False(t, cr.Pin)
}

func TestAdmin_PinFlushesLastComments(t *testing.T) {
	ts, _ := startupT(t, func(srv *Rest) { // a real cache, the default test one keeps nothing
		lru, err := cache.NewLruCache(cache.NewOpts[[]byte]().MaxKeys(100))
		require.NoError(t, err)
		srv.Cache = cache.NewScache(lru)
	})

	id := addComment(t, store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}, ts)
	res, code := get(t, ts.URL+"/api/v1/last/10?site=remark42")
	require.Equal(t, http.StatusOK, code)
	require.Contains(t, res, "test test #1", "the comment is in the last comments, and they are cached")
	require.NotContains(t, res, `"pin":true`)

	_, code = call(t, http.MethodPut,
		ts.URL+"/api/v1/admin/pin/"+id+"?site=remark42&url=https://radio-t.com/blah&pin=1", "", adminUmputunToken)
	require.Equal(t, http.StatusOK, code)

	res, code = get(t, ts.URL+"/api/v1/last/10?site=remark42")
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, res, `"pin":true`, "the cached last comments are dropped when a comment is pinned")
}

func TestAdmin_Block(t *testing.T) {
	ts, srv := startupT(t)

	makeTwoComments := func() {
		c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
			URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
		c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
			URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2", ID: "user2"}}

		_, err := srv.DataService.Create(c1)
		require.NoError(t, err)
		_, err = srv.DataService.Create(c2)
		require.NoError(t, err)
	}

	block := func(val int, ttl string) (code int, body string) {
		url := fmt.Sprintf("%s/api/v1/admin/user/%s?site=remark42&block=%d", ts.URL, "user1", val)
		if ttl != "" {
			url = url + "&ttl=" + ttl
		}
		body, code = callAdminOnly(t, http.MethodPut, url, adminUmputunToken)
		return code, body
	}

	makeTwoComments()

	// block permanently
	code, body := block(1, "")
	require.Equal(t, http.StatusOK, code)
	j := R.JSON{}
	err := json.Unmarshal([]byte(body), &j)
	assert.NoError(t, err)
	assert.Equal(t, "user1", j["user_id"])
	assert.Equal(t, true, j["block"])
	assert.Equal(t, "remark42", j["site_id"])

	assert.True(t, srv.DataService.IsBlocked("remark42", "user1"))
	assert.False(t, srv.DataService.IsBlocked("remark42", "user2"))

	// get last to confirm one comment deleted
	pi := callJSON[[]store.PostInfo](t, http.MethodGet, ts.URL+"/api/v1/last/10?site=remark42", "", "")
	assert.Equal(t, 1, len(pi), "last status updated, one comment left")

	// check if count call has one comment left
	pi = callJSON[[]store.PostInfo](t, http.MethodPost, ts.URL+"/api/v1/counts?site=remark42", `["https://radio-t.com/blah"]`, adminPasswd)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/blah", Count: 1}}, pi)

	comments := callJSON[commentsWithInfo](t, http.MethodGet,
		ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah&sort=+time", "", "")
	require.Equal(t, 2, len(comments.Comments), "should have 2 comments")
	assert.Equal(t, "", comments.Comments[0].Text, "permanent block clear comment")
	assert.True(t, comments.Comments[0].Deleted, "permanent block set deleted comment's status")

	// unblock
	code, body = block(-1, "")
	require.Equal(t, http.StatusOK, code)
	err = json.Unmarshal([]byte(body), &j)
	assert.NoError(t, err)
	assert.Equal(t, false, j["block"])

	// block with ttl, checked in place rather than through another admin request, which would
	// push this test over the 10 req/s limit on that route
	makeTwoComments()
	code, _ = block(1, "500ms")
	require.Equal(t, http.StatusOK, code)
	require.True(t, srv.DataService.IsBlocked("remark42", "user1"), "user1 blocked with ttl")

	// get as regular user
	comments = callJSON[commentsWithInfo](t, http.MethodGet,
		ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah&sort=+time", "", "")
	require.Equal(t, 4, len(comments.Comments), "should have 4 comments")
	assert.Equal(t, "test test #1", comments.Comments[2].Text, "comment not removed and not cleared")
	assert.False(t, comments.Comments[2].Deleted, "not deleted")

	srv.pubRest.cache = cache.NewScache[[]byte](cache.NewNopCache[[]byte]()) // TODO: with lru cache it won't be refreshed and invalidated for long
	// time

	// the ttl above is wide enough that the checks in between cannot outlast it, so reaching
	// here still inside the block, and the wait below observes it lapse
	require.Eventually(t, func() bool {
		return !srv.DataService.IsBlocked("remark42", "user1")
	}, waitTimeout, pollInterval, "block with ttl did not expire")

	comments = callJSON[commentsWithInfo](t, http.MethodGet,
		ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah&sort=+time", "", "")
	require.Equal(t, 4, len(comments.Comments), "should have 4 comments")
	assert.Equal(t, "test test #1", comments.Comments[2].Text, "restored")
	assert.False(t, comments.Comments[2].Deleted)

	assert.False(t, srv.DataService.IsBlocked("remark42", "user1"))
	assert.False(t, srv.DataService.IsBlocked("remark42", "user2"))
}

// putBlockUser sends PUT /admin/user/{userID} with the given query and returns the status and body.
// Each case below blocks a user of its own, so every request has its own path and its own bucket
// in the 10 req/s limiter of the admin routes.
func putBlockUser(t *testing.T, tsURL, userID, query string) (code int, body string) {
	t.Helper()
	body, code = call(t, http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/user/%s?site=remark42&%s", tsURL, userID, query), "", adminUmputunToken)
	return code, body
}

func TestAdmin_BlockRejectsBadTTL(t *testing.T) {
	ts, srv := startupT(t)

	tbl := []struct {
		name, userID, query string
		blockedBefore       bool
	}{
		{"day unit", "bad-day", "block=1&ttl=7d", false},
		{"not a duration", "bad-word", "block=1&ttl=week", false},
		{"negative", "bad-negative", "block=1&ttl=-1h", false},
		{"unblock with bad ttl", "bad-unblock", "block=0&ttl=7d", true},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			locator := store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}
			id, err := srv.DataService.Create(store.Comment{Text: "kept text", Locator: locator,
				User: store.User{Name: tt.userID, ID: tt.userID}})
			require.NoError(t, err)
			if tt.blockedBefore {
				require.NoError(t, srv.DataService.SetBlock("remark42", tt.userID, true, time.Hour))
			}

			code, body := putBlockUser(t, ts.URL, tt.userID, tt.query)
			assert.Equal(t, http.StatusBadRequest, code, "a bad ttl is rejected, %s", body)
			assert.Contains(t, body, "invalid ttl")
			assert.Equal(t, tt.blockedBefore, srv.DataService.IsBlocked("remark42", tt.userID), "block status unchanged")
			c, err := srv.DataService.Get(locator, id, store.User{})
			require.NoError(t, err)
			assert.Equal(t, "kept text", c.Text, "comment text kept")
			assert.False(t, c.Deleted, "comment not deleted")
		})
	}
}

func TestAdmin_BlockAcceptedTTL(t *testing.T) {
	ts, srv := startupT(t)

	tbl := []struct {
		name, userID, query string
		ttl                 time.Duration // 0 for a permanent block
	}{
		{"widget, a month", "ttl-month", "block=1&ttl=43200m", 30 * 24 * time.Hour},
		{"widget, a week", "ttl-week", "block=1&ttl=10080m", 7 * 24 * time.Hour},
		{"widget, a day", "ttl-day", "block=1&ttl=1440m", 24 * time.Hour},
		{"hours", "ttl-hours", "block=1&ttl=168h", 7 * 24 * time.Hour},
		{"widget, permanently", "ttl-absent", "block=1", 0},
		{"widget before v1.17, permanently", "ttl-undefined", "block=1&ttl=undefined", 0},
		{"zero", "ttl-zero", "block=1&ttl=0", 0},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			locator := store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}
			id, err := srv.DataService.Create(store.Comment{Text: "some text", Locator: locator,
				User: store.User{Name: tt.userID, ID: tt.userID}})
			require.NoError(t, err)

			before := time.Now()
			code, body := putBlockUser(t, ts.URL, tt.userID, tt.query)
			after := time.Now()
			require.Equal(t, http.StatusOK, code, body)
			assert.True(t, srv.DataService.IsBlocked("remark42", tt.userID))

			blocked, err := srv.DataService.BlockedUsers("remark42")
			require.NoError(t, err)
			var until time.Time
			for _, u := range blocked {
				if u.ID == tt.userID {
					until = u.Until
				}
			}
			c, err := srv.DataService.Get(locator, id, store.User{})
			require.NoError(t, err)
			if tt.ttl == 0 {
				assert.True(t, until.After(after.AddDate(99, 0, 0)), "permanent block, until %v", until)
				assert.True(t, c.Deleted, "a permanent block deletes the user's comments")
				return
			}
			assert.False(t, until.Before(before.Add(tt.ttl)), "block until %v, not before %v", until, before.Add(tt.ttl))
			assert.False(t, until.After(after.Add(tt.ttl)), "block until %v, not after %v", until, after.Add(tt.ttl))
			assert.Equal(t, "some text", c.Text, "a block with ttl keeps the user's comments")
			assert.False(t, c.Deleted)
		})
	}
}

func TestAdmin_BlockedList(t *testing.T) {
	ts, srv := startupT(t)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2 name", ID: "user2"}}

	// write comments for user1 and user2
	_, err := srv.DataService.Create(c1)
	assert.NoError(t, err)
	_, err = srv.DataService.Create(c2)
	assert.NoError(t, err)

	// block user1
	_, code := call(t, http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/user/%s?site=remark42&block=%d", ts.URL, "user1", 1), "", adminUmputunToken)
	assert.Equal(t, http.StatusOK, code)

	// block user2 for long enough that the "two users blocked" check below cannot race the ttl
	_, code = call(t, http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/user/%s?site=remark42&block=%d&ttl=1h", ts.URL, "user2", 1), "", adminUmputunToken)
	assert.Equal(t, http.StatusOK, code)

	users := callJSON[[]store.BlockedUser](t, http.MethodGet, ts.URL+"/api/v1/admin/blocked?site=remark42", "", adminUmputunToken)
	require.Equal(t, 2, len(users), "two users blocked")
	assert.Equal(t, "user1", users[0].ID)
	assert.Equal(t, "user1 name", users[0].Name)
	assert.Equal(t, "user2", users[1].ID)
	assert.Equal(t, "user2 name", users[1].Name)
	t.Logf("%+v", users)

	// re-block user2 with a short ttl and wait for it to lapse, so the lapse is observed
	// independently of the check above
	_, code = call(t, http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/user/%s?site=remark42&block=%d&ttl=150ms", ts.URL, "user2", 1), "", adminUmputunToken)
	require.Equal(t, http.StatusOK, code)

	// the closure runs off the test goroutine and asserts on the CollectT it is handed, never on t
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		blockedReq, reqErr := http.NewRequest("GET", ts.URL+"/api/v1/admin/blocked?site=remark42", http.NoBody)
		if !assert.NoError(c, reqErr) {
			return
		}
		blockedResp, sendErr := sendReq(blockedReq, adminUmputunToken)
		if !assert.NoError(c, sendErr) {
			return
		}
		defer blockedResp.Body.Close()
		assert.Equal(c, http.StatusOK, blockedResp.StatusCode)
		blocked := []store.BlockedUser{}
		assert.NoError(c, json.NewDecoder(blockedResp.Body).Decode(&blocked))
		assert.Len(c, blocked, 1, "one user left blocked")
	}, waitTimeout, httpPoll)
}

func TestAdmin_ReadOnly(t *testing.T) {
	ts, srv := startupT(t)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2", ID: "user2"}}

	_, err := srv.DataService.Create(c1)
	assert.NoError(t, err)
	_, err = srv.DataService.Create(c2)
	assert.NoError(t, err)

	info, err := srv.DataService.Info(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}, 0)
	assert.NoError(t, err)
	assert.False(t, info.ReadOnly)

	// set post to read-only
	roURL := fmt.Sprintf("%s/api/v1/admin/readonly?site=remark42&url=https://radio-t.com/blah&ro=1", ts.URL)
	_, code := call(t, http.MethodPut, roURL, "", "") // anonymous user
	assert.Equal(t, http.StatusUnauthorized, code)
	_, code = call(t, http.MethodPut, roURL, "", adminUmputunToken)
	assert.Equal(t, http.StatusOK, code)
	info, err = srv.DataService.Info(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}, 0)
	assert.NoError(t, err)
	assert.True(t, info.ReadOnly)

	// try to write comment
	c := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}
	b, err := json.Marshal(c)
	assert.NoError(t, err, "can't marshal comment %+v", c)
	_, code = call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42", string(b), adminUmputunToken)
	assert.Equal(t, http.StatusForbidden, code)

	// reset post's read-only
	_, code = call(t, http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/readonly?site=remark42&url=https://radio-t.com/blah&ro=0", ts.URL), "", adminUmputunToken)
	assert.Equal(t, http.StatusOK, code)
	info, err = srv.DataService.Info(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}, 0)
	assert.NoError(t, err)
	assert.False(t, info.ReadOnly)

	// try to write comment
	c = store.Comment{Text: "test test #2",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}
	b, err = json.Marshal(c)
	assert.NoError(t, err, "can't marshal comment %+v", c)
	_, code = call(t, http.MethodPost, ts.URL+"/api/v1/comment?site="+c.Locator.SiteID, string(b), adminUmputunToken)
	assert.Equal(t, http.StatusCreated, code)
}

func TestAdmin_ReadOnlyNoComments(t *testing.T) {
	ts, srv := startupT(t)

	// set post to read-only
	_, code := callAdminOnly(t, http.MethodPut, fmt.Sprintf("%s/api/v1/admin/readonly?site=remark42&url=https://radio-t.com/blah&ro=1", ts.URL),
		adminUmputunToken)
	assert.Equal(t, http.StatusOK, code)
	_, err := srv.DataService.Info(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}, 0)
	assert.Error(t, err)

	// test format "tree"
	comments := callJSON[commentsWithInfo](t, http.MethodGet,
		ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah&format=tree", "", "")
	assert.Equal(t, 0, len(comments.Comments), "should have 0 comments")
	assert.True(t, comments.Info.ReadOnly)
	t.Logf("%+v", comments)

	// test format "plain"
	comments = callJSON[commentsWithInfo](t, http.MethodGet,
		ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah", "", "")
	assert.Equal(t, 0, len(comments.Comments), "should have 0 comments")
	assert.True(t, comments.Info.ReadOnly)
	t.Logf("%+v", comments)
}

func TestAdmin_ReadOnlyWithAge(t *testing.T) {
	ts, srv := startupT(t)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"},
		Timestamp: time.Date(2001, 1, 1, 1, 1, 1, 0, time.Local)}
	_, err := srv.DataService.Create(c1)
	assert.NoError(t, err)

	info, err := srv.DataService.Info(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}, 10)
	assert.NoError(t, err)
	assert.True(t, info.ReadOnly, "ro by age")

	// set post to read-only
	_, code := callAdminOnly(t, http.MethodPut, fmt.Sprintf("%s/api/v1/admin/readonly?site=remark42&url=https://radio-t.com/blah&ro=1", ts.URL),
		adminUmputunToken)
	assert.Equal(t, http.StatusOK, code)
	info, err = srv.DataService.Info(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}, 0)
	assert.NoError(t, err)
	assert.True(t, info.ReadOnly)

	// reset post's read-only
	_, code = call(t, http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/readonly?site=remark42&url=https://radio-t.com/blah&ro=0", ts.URL), "", adminUmputunToken)
	assert.Equal(t, http.StatusForbidden, code)
	info, err = srv.DataService.Info(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}, 0)
	assert.NoError(t, err)
	assert.True(t, info.ReadOnly)
}
func TestAdmin_Verify(t *testing.T) {
	ts, srv := startupT(t)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2", ID: "user2"}}

	_, err := srv.DataService.Create(c1)
	assert.NoError(t, err)
	_, err = srv.DataService.Create(c2)
	assert.NoError(t, err)

	verified := srv.DataService.IsVerified("remark42", "user1")
	assert.False(t, verified)

	_, code := callAdminOnly(t, http.MethodPut, fmt.Sprintf("%s/api/v1/admin/verify/user1?site=remark42&verified=1", ts.URL),
		adminUmputunToken)
	assert.Equal(t, http.StatusOK, code)
	verified = srv.DataService.IsVerified("remark42", "user1")
	assert.True(t, verified)

	comments := callJSON[commentsWithInfo](t, http.MethodGet,
		ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah&sort=+time", "", "")
	require.Equal(t, 2, len(comments.Comments), "should have 2 comments")
	assert.Equal(t, "test test #1", comments.Comments[0].Text)
	assert.True(t, comments.Comments[0].User.Verified)

	_, code = call(t, http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/verify/user1?site=remark42&verified=0", ts.URL), "", adminUmputunToken)
	assert.Equal(t, http.StatusOK, code)
	verified = srv.DataService.IsVerified("remark42", "user1")
	assert.False(t, verified)

	comments = callJSON[commentsWithInfo](t, http.MethodGet,
		ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah&sort=+time", "", "")
	require.Equal(t, 2, len(comments.Comments), "should have 2 comments")
	assert.Equal(t, "test test #1", comments.Comments[0].Text)
	assert.False(t, comments.Comments[0].User.Verified)
}

func TestAdmin_VerifyFlushesLastComments(t *testing.T) {
	ts, srv := startupT(t, func(srv *Rest) { // a real cache, the default test one keeps nothing
		lru, err := cache.NewLruCache(cache.NewOpts[[]byte]().MaxKeys(100))
		require.NoError(t, err)
		srv.Cache = cache.NewScache(lru)
	})

	_, err := srv.DataService.Create(store.Comment{Text: "test test #1", User: store.User{Name: "user1 name", ID: "user1"},
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}})
	require.NoError(t, err)
	require.NoError(t, srv.DataService.SetVerified("remark42", "user1", true))

	res, code := get(t, ts.URL+"/api/v1/last/10?site=remark42")
	require.Equal(t, http.StatusOK, code)
	require.Contains(t, res, `"verified":true`, "the user is verified in the last comments, and they are cached")

	_, code = call(t, http.MethodPut, ts.URL+"/api/v1/admin/verify/user1?site=remark42&verified=0", "", adminUmputunToken)
	require.Equal(t, http.StatusOK, code)

	res, code = get(t, ts.URL+"/api/v1/last/10?site=remark42")
	require.Equal(t, http.StatusOK, code)
	assert.NotContains(t, res, `"verified":true`, "the cached last comments are dropped when the status changes")
}

func TestAdmin_ExportStream(t *testing.T) {
	ts, _ := startupT(t)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah2"}}

	addComment(t, c1, ts)
	addComment(t, c2, ts)

	body, code := call(t, http.MethodGet, ts.URL+"/api/v1/admin/export?site=remark42&mode=stream", "", adminPasswd)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 3, strings.Count(body, "\n"))
	assert.Equal(t, 2, strings.Count(body, "\"text\""))
	t.Logf("%s", body)
}

func TestAdmin_ExportFile(t *testing.T) {
	ts, _ := startupT(t)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	c2 := store.Comment{Text: "test test #2",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah2"}}

	addComment(t, c1, ts)
	addComment(t, c2, ts)

	// the admin answer is read here for its headers, so the refusals are checked one by one
	exportURL := ts.URL + "/api/v1/admin/export?site=remark42&mode=file"
	_, code := call(t, http.MethodGet, exportURL, "", "")
	assert.Equal(t, http.StatusUnauthorized, code, "anonymous user refused")
	_, code = call(t, http.MethodGet, exportURL, "", devToken)
	assert.Equal(t, http.StatusForbidden, code, "regular user refused")
	req, err := http.NewRequest(http.MethodGet, exportURL, http.NoBody)
	require.NoError(t, err)
	resp, err := sendReq(req, adminUmputunToken)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/gzip", resp.Header.Get("Content-Type"))

	ungzReader, err := gzip.NewReader(resp.Body)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	ungzBody, err := io.ReadAll(ungzReader)
	assert.NoError(t, err)
	assert.Equal(t, 3, strings.Count(string(ungzBody), "\n"))
	assert.Equal(t, 2, strings.Count(string(ungzBody), "\"text\""))
	t.Logf("%s", string(ungzBody))
}

func TestAdmin_DeleteMeRequest(t *testing.T) {
	ts, srv := startupT(t)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2", ID: "user2"}}

	_, err := srv.DataService.Create(c1)
	assert.NoError(t, err)
	_, err = srv.DataService.Create(c2)
	assert.NoError(t, err)

	comments, err := srv.DataService.User("remark42", "user1", 0, 0, store.User{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(comments), "a comment for user1")

	email, err := srv.DataService.SetUserEmail("remark42", "user1", "test@example.org")
	assert.NoError(t, err)
	assert.Equal(t, "test@example.org", email, "new email for user1")

	email, err = srv.DataService.GetUserEmail("remark42", "user1")
	assert.NoError(t, err)
	assert.Equal(t, "test@example.org", email, "new email for user1 is readable")

	require.NoError(t, os.MkdirAll(testAvatarDir(srv)+"/42", 0o700))
	require.NoError(t, os.WriteFile(testAvatarDir(srv)+"/42/pic.image", []byte("some image data"), 0o600))

	// production-shaped picture URL: removal must path.Base it to the avatar id
	tkn := deleteMeToken(t, srv, "remark42", "user1", "https://demo.remark42.com/api/v1/avatar/pic.image")

	_, code := call(t, http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, tkn), "", adminPasswd)
	assert.Equal(t, http.StatusOK, code)

	_, err = srv.DataService.User("remark42", "user1", 0, 0, store.User{})
	assert.EqualError(t, err, "no comments for user user1 in store")

	email, err = srv.DataService.GetUserEmail("remark42", "user1")
	assert.NoError(t, err)
	assert.Empty(t, email, "user1 email was deleted")

	assert.NoFileExists(t, testAvatarDir(srv)+"/42/pic.image", "user's avatar should be removed on deleteme")
}

func TestAdmin_DeleteMeRequestRejectsExpiredToken(t *testing.T) {
	ts, srv := startupT(t)

	c := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"},
		User: store.User{Name: "user1 name", ID: "user1"}}
	_, err := srv.DataService.Create(c)
	require.NoError(t, err)

	tkn, err := srv.Authenticator.TokenService().Token(token.Claims{
		Audience:  jwt.ClaimStrings{"remark42"},
		ID:        "4567890",
		Issuer:    "remark42",
		NotBefore: jwt.NewNumericDate(time.Now().AddDate(0, -4, 0)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
		User:      &token.User{ID: "user1", Attributes: map[string]any{"delete_me": true}},
	})
	require.NoError(t, err)

	_, code := call(t, http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, tkn), "", adminPasswd)
	assert.Equal(t, http.StatusForbidden, code)

	comments, err := srv.DataService.User("remark42", "user1", 0, 0, store.User{})
	require.NoError(t, err)
	assert.Len(t, comments, 1, "an expired link deletes nothing")
}

// an admin is an admin of the site the session belongs to, so a deletion token for another site
// served by the same instance must not be processed under that session
func TestAdmin_DeleteMeRequestRejectsOtherSite(t *testing.T) {
	ts, srv := startupT(t, func(srv *Rest) {
		require.NoError(t, srv.DataService.Engine.Close())
		b, err := engine.NewBoltDB(bolt.Options{},
			engine.BoltSite{FileName: filepath.Join(t.TempDir(), "remark42.db"), SiteID: "remark42"},
			engine.BoltSite{FileName: filepath.Join(t.TempDir(), "other.db"), SiteID: "other"})
		require.NoError(t, err)
		srv.DataService.Engine = b
		srv.DataService.AdminStore = adminstore.NewStaticStore("123456", []string{"remark42", "other"}, []string{"a1", "a2"}, "admin@remark-42.com")
	})

	c := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "other", URL: "https://radio-t.com/blah"},
		User: store.User{Name: "user1 name", ID: "user1"}}
	_, err := srv.DataService.Create(c)
	require.NoError(t, err)

	tkn := deleteMeToken(t, srv, "other", "user1", "")

	// adminUmputunToken is a session for remark42
	_, code := call(t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/admin/deleteme?site=remark42&token=%s", ts.URL, tkn), "", adminUmputunToken)
	assert.Equal(t, http.StatusForbidden, code)
	comments, err := srv.DataService.User("other", "user1", 0, 0, store.User{})
	require.NoError(t, err)
	assert.Len(t, comments, 1, "the other site's data is untouched")

	// the basic-auth admin administers every site and processes it when naming that site
	_, code = call(t, http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?site=other&token=%s", ts.URL, tkn), "", adminPasswd)
	assert.Equal(t, http.StatusOK, code)
	_, err = srv.DataService.User("other", "user1", 0, 0, store.User{})
	assert.EqualError(t, err, "no comments for user user1 in store")
}

// a delete_me request whose token carries a picture must still succeed when the avatar is
// already gone from the store: the user data is deleted and a missing avatar is tolerated
func TestAdmin_DeleteMeRequestMissingAvatar(t *testing.T) {
	ts, srv := startupT(t)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user3 name", ID: "user3"}}
	_, err := srv.DataService.Create(c1)
	require.NoError(t, err)

	tkn := deleteMeToken(t, srv, "remark42", "user3", "missing.image") // no avatar file exists for this picture in the store

	_, code := call(t, http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, tkn), "", adminPasswd)
	assert.Equal(t, http.StatusOK, code, "a missing avatar must not fail the deletion")

	_, err = srv.DataService.User("remark42", "user3", 0, 0, store.User{})
	assert.EqualError(t, err, "no comments for user user3 in store", "user3 comments should be deleted")
}

// a genuine (non not-found) avatar-store failure must now surface, not be silently swallowed:
// avatar.ErrNotFound lets deleteMeRequestCtrl tell an already-gone avatar from a real error
func TestAdmin_DeleteMeRequestAvatarRemoveError(t *testing.T) {
	ts, srv := startupT(t)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user5 name", ID: "user5"}}
	_, err := srv.DataService.Create(c1)
	require.NoError(t, err)

	// put a non-empty directory where the avatar file is expected, so Store.Remove fails with a real
	// error (directory not empty), not os.ErrNotExist - "pic" hashes to partition 42
	require.NoError(t, os.MkdirAll(testAvatarDir(srv)+"/42/pic.image", 0o700))
	require.NoError(t, os.WriteFile(testAvatarDir(srv)+"/42/pic.image/child", []byte("x"), 0o600))

	tkn := deleteMeToken(t, srv, "remark42", "user5", "https://demo.remark42.com/api/v1/avatar/pic.image")

	_, code := call(t, http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, tkn), "", adminPasswd)
	assert.Equal(t, http.StatusInternalServerError, code, "a real avatar-store failure must surface, not be swallowed")
}

// deleteMeToken signs the token of the link deleteMeCtrl sends to delete the data of userID on site
func deleteMeToken(t *testing.T, srv *Rest, site, userID, picture string) string {
	t.Helper()
	tkn, err := srv.Authenticator.TokenService().Token(token.Claims{
		Audience:  jwt.ClaimStrings{site},
		Issuer:    "remark42",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Minute)),
		NotBefore: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
		User:      &token.User{ID: userID, Picture: picture, Attributes: map[string]any{"delete_me": true}},
	})
	require.NoError(t, err)
	return tkn
}

func TestAvatarIDFromPicture(t *testing.T) {
	tbl := []struct {
		name    string
		picture string
		want    string
	}{
		{"local avatar url", "https://demo.remark42.com/api/v1/avatar/cb42ff493ade696d88a3a590f136ae9e34de7c1b.image", "cb42ff493ade696d88a3a590f136ae9e34de7c1b.image"},
		{"bare avatar id", "pic.image", "pic.image"},
		{"parent sentinel", "https://demo.remark42.com/api/v1/avatar/..", ""},
		{"trailing slash", "https://demo.remark42.com/api/v1/avatar/", ""},
		{"root", "/", ""},
		{"dotdot", "..", ""},
		{"empty", "", ""},
		{"provider url without image suffix", "https://example.com/pic.png", ""},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, avatarIDFromPicture(tc.picture))
		})
	}
}

func TestAdmin_DeleteMeRequestFailed(t *testing.T) {
	ts, srv := startupT(t)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "provider1_user1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2", ID: "provider1_user2"}}

	_, err := srv.DataService.Create(c1)
	assert.NoError(t, err)
	_, err = srv.DataService.Create(c2)
	assert.NoError(t, err)

	// try with bad token
	_, code := call(t, http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, "bad token"), "", adminPasswd)
	assert.Equal(t, http.StatusBadRequest, code)

	// try with bad auth
	claims := token.Claims{
		SessionOnly: true,
		Audience:    jwt.ClaimStrings{"remark42"},
		ID:          "provider1_1234567",
		Issuer:      "remark42",
		NotBefore:   jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
		ExpiresAt:   jwt.NewNumericDate(time.Now().Add(30 * time.Minute)),
		User: &token.User{
			ID: "provider1_user1",
			Attributes: map[string]any{
				"delete_me": true,
			},
		},
	}

	tkn, err := srv.Authenticator.TokenService().Token(claims)
	assert.NoError(t, err)
	_, code = call(t, http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, tkn), "", "basic:bad-password")
	assert.Equal(t, http.StatusUnauthorized, code, "the delete-me token in the query is not a session")

	// unknown user: deletion is idempotent, so a valid (signed) delete_me token for a user with
	// no stored data is a no-op success rather than an error
	badClaimsUser := claims
	badClaimsUser.User.ID = "no-such-id"
	tkn, err = srv.Authenticator.TokenService().Token(badClaimsUser)
	assert.NoError(t, err)
	b, code := call(t, http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, tkn), "", adminPasswd)
	assert.Equal(t, http.StatusOK, code, b)
	badClaimsUser.User.ID = "provider1_user1"

	// try without deleteme flag
	badClaimsWithoutDeleteMe := claims
	badClaimsWithoutDeleteMe.User.SetBoolAttr("delete_me", false)
	tkn, err = srv.Authenticator.TokenService().Token(badClaimsWithoutDeleteMe)
	assert.NoError(t, err)
	b, code = call(t, http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, tkn), "", adminPasswd)
	assert.Equal(t, http.StatusForbidden, code)
	assert.Contains(t, b, "can't use provided token")
	badClaimsWithoutDeleteMe.User.SetBoolAttr("delete_me", true)

	// try with wrong audience
	badClaimsMultipleAudience := claims
	badClaimsMultipleAudience.Audience = jwt.ClaimStrings{"remark42", "something else"}
	tkn, err = srv.Authenticator.TokenService().Token(badClaimsMultipleAudience)
	assert.NoError(t, err)
	b, code = call(t, http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, tkn), "", adminPasswd)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Contains(t, b, "can't process token, claims.Audience expected to be a single element but it's not")
	badClaimsMultipleAudience.Audience = jwt.ClaimStrings{"remark42"}
}

func TestAdmin_GetUserInfo(t *testing.T) {
	ts, srv := startupT(t)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user1 name", ID: "user1"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah"}, User: store.User{Name: "user2", ID: "user2"}}

	_, err := srv.DataService.Create(c1)
	assert.NoError(t, err)
	_, err = srv.DataService.Create(c2)
	assert.NoError(t, err)

	u := callJSON[store.User](t, http.MethodGet, fmt.Sprintf("%s/api/v1/admin/user/user1?site=remark42&url=https://radio-t.com/blah",
		ts.URL), "", adminPasswd)
	assert.Equal(t, store.User{Name: "user1 name", ID: "user1", Picture: "", IP: "823688dafca7393d24c871a2da98a84d8732e927",
		Admin: false, Blocked: false, Verified: false}, u)

	_, code := get(t, fmt.Sprintf("%s/api/v1/admin/user/user1?site=remark42&url=https://radio-t.com/blah", ts.URL))
	assert.Equal(t, http.StatusUnauthorized, code, "no auth")

	_, code = call(t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/admin/user/userX?site=remark42&url=https://radio-t.com/blah", ts.URL), "", adminPasswd)
	assert.Equal(t, http.StatusBadRequest, code, "no info about user")
}

// a token without a user, such as an email confirmation token, is refused, not dereferenced
func TestAdmin_DeleteMeRequestTokenWithoutUser(t *testing.T) {
	ts, srv := startupT(t)

	tkn := handshakeToken(t, srv, "provider1_dev::me@example.com", "")

	body, code := call(t, http.MethodGet, fmt.Sprintf("%s/api/v1/admin/deleteme?token=%s", ts.URL, tkn), "", adminPasswd)
	assert.Equal(t, http.StatusForbidden, code, body)
}

// emptyUserFinder answers a user's comments with an empty list and no error, as a remote engine such as
// the memory_store example does for a user it has never seen
type emptyUserFinder struct{ engine.Interface }

func (e emptyUserFinder) Find(req engine.FindRequest) ([]store.Comment, error) {
	if req.UserID != "" {
		return []store.Comment{}, nil
	}
	return e.Interface.Find(req)
}

// an empty user list without an error is reported as a client error, not passed on as a nil error
func TestAdmin_GetUserInfoEmptyWithoutError(t *testing.T) {
	ts, _ := startupT(t, func(srv *Rest) {
		srv.DataService.Engine = emptyUserFinder{Interface: srv.DataService.Engine}
	})

	body, code := call(t, http.MethodGet, ts.URL+"/api/v1/admin/user/nobody?site=remark42", "", adminPasswd)
	assert.Equal(t, http.StatusBadRequest, code, body)
	assert.Contains(t, body, "no comments for user nobody")
}
