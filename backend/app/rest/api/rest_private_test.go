package api

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-pkgz/auth/v2/token"
	"github.com/go-pkgz/lgr"
	R "github.com/go-pkgz/rest"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"

	"github.com/umputun/remark42/backend/app/notify"
	"github.com/umputun/remark42/backend/app/rest"
	"github.com/umputun/remark42/backend/app/rest/proxy"
	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/engine"
	"github.com/umputun/remark42/backend/app/store/image"
	"github.com/umputun/remark42/backend/app/store/service"
)

// gopher png for test, from https://golang.org/src/image/png/example_test.go
const gopher = `iVBORw0KGgoAAAANSUhEUgAAAEsAAAA8CAAAAAALAhhPAAAFfUlEQVRYw62XeWwUVRzHf2+OPbo9d7tsWyiyaZti6eWGAhISoIGKECEKCAiJJkYTiUgTMYSIosYYBBIUIxoSPIINEBDi2VhwkQrVsj1ESgu9doHWdrul7ba73WNm3vOPtsseM9MdwvvrzTs+8/t95ze/33sI5BqiabU6m9En8oNjduLnAEDLUsQXFF8tQ5oxK3vmnNmDSMtrncks9Hhtt/qeWZapHb1ha3UqYSWVl2ZmpWgaXMXGohQAvmeop3bjTRtv6SgaK/Pb9/bFzUrYslbFAmHPp+3WhAYdr+7GN/YnpN46Opv55VDsJkoEpMrY/vO2BIYQ6LLvm0ThY3MzDzzeSJeeWNyTkgnIE5ePKsvKlcg/0T9QMzXalwXMlj54z4c0rh/mzEfr+FgWEz2w6uk8dkzFAgcARAgNp1ZYef8bH2AgvuStbc2/i6CiWGj98y2tw2l4FAXKkQBIf+exyRnteY83LfEwDQAYCoK+P6bxkZm/0966LxcAAILHB56kgD95PPxltuYcMtFTWw/FKkY/6Opf3GGd9ZF+Qp6mzJxzuRSractOmJrH1u8XTvWFHINNkLQLMR+XHXvfPPHw967raE1xxwtA36IMRfkAAG29/7mLuQcb2WOnsJReZGfpiHsSBX81cvMKywYZHhX5hFPtOqPGWZCXnhWGAu6lX91ElKXSalcLXu3UaOXVay57ZSe5f6Gpx7J2MXAsi7EqSp09b/MirKSyJfnfEEgeDjl8FgDAfvewP03zZ+AJ0m9aFRM8eEHBDRKjfcreDXnZdQuAxXpT2NRJ7xl3UkLBhuVGU16gZiGOgZmrSbRdqkILuL/yYoSXHHkl9KXgqNu3PB8oRg0geC5vFmLjad6mUyTKLmF3OtraWDIfACyXqmephaDABawfpi6tqqBZytfQMqOz6S09iWXhktrRaB8Xz4Yi/8gyABDm5NVe6qq/3VzPrcjELWrebVuyY2T7ar4zQyybUCtsQ5Es1FGaZVrRVQwAgHGW2ZCRZshI5bGQi7HesyE972pOSeMM0dSktlzxRdrlqb3Osa6CCS8IJoQQQgBAbTAa5l5epO34rJszibJI8rxLfGzcp1dRosutGeb2VDNgqYrwTiPNsLxXiPi3dz7LiS1WBRBDBOnqEjyy3aQb+/bLiJzz9dIkscVBBLxMfSEac7kO4Fpkngi0ruNBeSOal+u8jgOuqPz12nryMLCniEjtOOOmpt+KEIqsEdocJjYXwrh9OZqWJQyPCTo67LNS/TdxLAv6R5ZNK9npEjbYdT33gRo4o5oTqR34R+OmaSzDBWsAIPhuRcgyoteNi9gF0KzNYWVItPf2TLoXEg+7isNC7uJkgo1iQWOfRSP9NR11RtbZZ3OMG/VhL6jvx+J1m87+RCfJChAtEBQkSBX2PnSiihc/Twh3j0h7qdYQAoRVsRGmq7HU2QRbaxVGa1D6nIOqaIWRjyRZpHMQKWKpZM5feA+lzC4ZFultV8S6T0mzQGhQohi5I8iw+CsqBSxhFMuwyLgSwbghGb0AiIKkSDmGZVmJSiKihsiyOAUs70UkywooYP0bii9GdH4sfr1UNysd3fUyLLMQN+rsmo3grHl9VNJHbbwxoa47Vw5gupIqrZcjPh9R4Nye3nRDk199V+aetmvVtDRE8/+cbgAAgMIWGb3UA0MGLE9SCbWX670TDy1y98c3D27eppUjsZ6fql3jcd5rUe7+ZIlLNQny3Rd+E5Tct3WVhTM5RBCEdiEK0b6B+/ca2gYU393nFj/n1AygRQxPIUA043M42u85+z2SnssKrPl8Mx76NL3E6eXc3be7OD+H4WHbJkKI8AU8irbITQjZ+0hQcPEgId/Fn/pl9crKH02+5o2b9T/eMx7pKoskYgAAAABJRU5ErkJggg==`

func gopherPNG() io.Reader { return base64.NewDecoder(base64.StdEncoding, strings.NewReader(gopher)) }

func TestRest_Create(t *testing.T) {
	ts, _ := startupT(t)

	b, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42",
		`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, adminPasswd)
	require.Equal(t, http.StatusCreated, code, b)

	c := R.JSON{}
	err := json.Unmarshal([]byte(b), &c)
	assert.NoError(t, err)
	loc := c["locator"].(map[string]any)
	assert.Equal(t, "remark42", loc["site"])
	assert.Equal(t, "https://radio-t.com/blah1", loc["url"])
	assert.True(t, len(c["id"].(string)) > 8)
}

// based on issue https://github.com/umputun/remark42/issues/1292
func TestRest_CreateFilteredCode(t *testing.T) {
	ts, _ := startupT(t)

	b, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42",
		`{"text": "`+"`foo<bar>`"+`", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, adminPasswd)
	require.Equal(t, http.StatusCreated, code, b)

	c := R.JSON{}
	err := json.Unmarshal([]byte(b), &c)
	require.NoError(t, err, b)
	loc := c["locator"].(map[string]any)
	assert.Equal(t, "remark42", loc["site"])
	assert.Equal(t, "https://radio-t.com/blah1", loc["url"])
	assert.Equal(t, "`foo<bar>`", c["orig"])
	assert.Contains(t, c["text"], "foo")
	assert.Contains(t, c["text"], "bar")
	assert.True(t, len(c["id"].(string)) > 8)
}

// based on issue https://github.com/umputun/remark42/issues/1631. With external images cached, the image a posted
// or edited comment shows is stored, while one only previewed or requested from the proxy is not
func TestRest_CreateAndPreviewWithImage(t *testing.T) {
	ts, srv := startupT(t)
	ts.Close()

	srv.ImageService.ProxyAPI = srv.RemarkURL + "/api/v1/img"
	srv.ImageProxy = &proxy.Image{
		HTTP2HTTPS:    true,
		CacheExternal: true,
		RoutePath:     "/api/v1/img",
		RemarkURL:     srv.RemarkURL,
		ImageService:  srv.ImageService,
		Transport:     http.DefaultTransport,
	}
	srv.DataService.ImageService = srv.ImageService
	t.Cleanup(func() { srv.ImageService.Close(context.Background()) })
	srv.CommentFormatter = store.NewCommentFormatter(srv.ImageProxy, store.CommentConverterFunc(srv.ImageService.SetSizes))
	// need to recreate the server with new ImageProxy, otherwise old one will be used
	ts = httptest.NewServer(srv.routes())
	defer ts.Close()

	var mu sync.Mutex
	requested := map[string]bool{}
	wasRequested := func(path string) bool {
		mu.Lock()
		defer mu.Unlock()
		return requested[path]
	}
	// server with the test PNG image
	pngServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, e := io.Copy(w, gopherPNG())
		assert.NoError(t, e)
		mu.Lock()
		requested[r.URL.Path] = true
		mu.Unlock()
	}))
	defer pngServer.Close()
	// images of other hosts, which a subtest posts, are not fetched over the network
	srv.ImageService.FetchExternal = func(ctx context.Context, imgURL string) ([]byte, error) {
		if !strings.HasPrefix(imgURL, pngServer.URL+"/") {
			return nil, errors.New("not the test image server")
		}
		return srv.ImageProxy.Download(ctx, imgURL)
	}
	stored := func(path string) bool {
		imgID, err := image.CachedImgID(pngServer.URL + path)
		require.NoError(t, err)
		_, err = srv.ImageService.Load(imgID)
		return err == nil
	}
	var commentID string

	t.Run("create", func(t *testing.T) {
		b, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42",
			`{"text": "![](`+pngServer.URL+`/gopher.png)", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, adminPasswd)
		require.Equal(t, http.StatusCreated, code, b)

		c := R.JSON{}
		err := json.Unmarshal([]byte(b), &c)
		require.NoError(t, err, b)
		assert.NotContains(t, c["text"], pngServer.URL)
		assert.Contains(t, c["text"], srv.RemarkURL)
		loc := c["locator"].(map[string]any)
		assert.Equal(t, "remark42", loc["site"])
		assert.Equal(t, "https://radio-t.com/blah1", loc["url"])
		commentID = c["id"].(string)
		assert.True(t, len(commentID) > 8)
		require.Eventually(t, func() bool { return stored("/gopher.png") }, waitTimeout, pollInterval,
			"the posted comment's image is stored")
	})

	t.Run("edit", func(t *testing.T) {
		_, code := call(t, http.MethodPut, ts.URL+"/api/v1/comment/"+commentID+"?site=remark42&url=https://radio-t.com/blah1",
			`{"text": "![](`+pngServer.URL+`/gopher.png) ![](`+pngServer.URL+`/edited.png)"}`, adminPasswd)
		require.Equal(t, http.StatusOK, code)
		require.Eventually(t, func() bool { return stored("/edited.png") }, waitTimeout, pollInterval,
			"the image the edit added is stored")
	})

	t.Run("an image requested through the proxy is not stored", func(t *testing.T) {
		src := base64.URLEncoding.EncodeToString([]byte(pngServer.URL + "/requested.png"))
		b, code := get(t, ts.URL+"/api/v1/img?src="+src)
		require.Equal(t, http.StatusOK, code, b)
		assert.Equal(t, gopher, base64.StdEncoding.EncodeToString([]byte(b)))
		assert.True(t, wasRequested("/requested.png"), "the proxy fetched the image to show it")
		assert.False(t, stored("/requested.png"), "and did not store it")
	})

	t.Run("preview, then post", func(t *testing.T) {
		text := `{"text": "![](` + pngServer.URL + `/preview.png)", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`
		b, code := call(t, http.MethodPost, ts.URL+"/api/v1/preview", text, adminPasswd)
		require.Equal(t, http.StatusOK, code, b)
		assert.NotContains(t, b, pngServer.URL)
		assert.Contains(t, b, srv.RemarkURL)
		require.Eventually(t, func() bool { return stored("/preview.png") }, waitTimeout, pollInterval,
			"the previewed comment's image is stored")

		b, code = call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42", text, adminPasswd)
		require.Equal(t, http.StatusCreated, code, b)
		c := R.JSON{}
		require.NoError(t, json.Unmarshal([]byte(b), &c), b)
		assert.Contains(t, c["text"], `width="`, "the posted image, stored by the preview, has its size")
	})

	t.Run("every stored image is proxied", func(t *testing.T) {
		text := `plain <img src=\"http://attacker.example/plain.png\"> amp ![](http://attacker.example/a.png?a=1&b=2) ` +
			`hidden <svg><title><!-- </title><img src=\"http://attacker.example/hidden.png\" --></title></svg> ` +
			strings.Repeat(`<img src>`, 8)
		b, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42",
			`{"text": "`+text+`", "locator":{"url": "https://radio-t.com/blah2", "site": "remark42"}}`, adminPasswd)
		require.Equal(t, http.StatusCreated, code, b)
		c := R.JSON{}
		require.NoError(t, json.Unmarshal([]byte(b), &c), b)
		stored := c["text"].(string)
		assert.NotContains(t, stored, "attacker.example", "an image is stored with its own address")
		assert.Equal(t, 3, strings.Count(stored, srv.RemarkURL+"/api/v1/img?src="), stored)
		assert.Less(t, len(stored), 2048, "empty sources don't multiply the text")
	})
}

// TestRest_PreviewCommitsNothing previews a comment with an external image when no edit window is left to wait
// for: the image is fetched to staging, so a post after the preview can size it, and is not committed, which only
// a posted comment's commit does
func TestRest_PreviewCommitsNothing(t *testing.T) {
	pngServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, e := io.Copy(w, gopherPNG())
		assert.NoError(t, e)
	}))
	defer pngServer.Close()
	location, staging := t.TempDir(), t.TempDir()
	ts, srv := startupT(t, func(srv *Rest) {
		srv.ImageService = image.NewService(&image.FileSystem{Location: location, Staging: staging}, image.ServiceParams{
			ImageAPI: srv.RemarkURL + "/api/v1/picture/", ProxyAPI: srv.RemarkURL + "/api/v1/img", MaxSize: 10000})
		srv.ImageProxy = &proxy.Image{CacheExternal: true, RoutePath: "/api/v1/img", RemarkURL: srv.RemarkURL,
			ImageService: srv.ImageService, Transport: http.DefaultTransport}
		srv.ImageService.FetchExternal = srv.ImageProxy.Download
		srv.DataService.ImageService = srv.ImageService
		srv.CommentFormatter = store.NewCommentFormatter(srv.ImageProxy)
	})
	t.Cleanup(func() { srv.ImageService.Close(context.Background()) })

	b, code := call(t, http.MethodPost, ts.URL+"/api/v1/preview",
		`{"text": "![](`+pngServer.URL+`/a.png)", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, adminPasswd)
	require.Equal(t, http.StatusOK, code, b)
	imgID, err := image.CachedImgID(pngServer.URL + "/a.png")
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		_, loadErr := srv.ImageService.Load(imgID)
		return loadErr == nil
	}, waitTimeout, pollInterval, "the previewed image is stored")
	_, err = os.Stat(filepath.Join(location, imgID))
	assert.ErrorIs(t, err, os.ErrNotExist, "the previewed image is not committed")
	_, err = os.Stat(filepath.Join(staging, imgID))
	assert.NoError(t, err, "the previewed image is in staging")
}

func TestRest_CreateOldPost(t *testing.T) {
	ts, srv := startupT(t)

	// make old, but not too old comment
	old := store.Comment{Text: "test test old", ParentID: "", Timestamp: time.Now().AddDate(0, 0, -5),
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}, User: store.User{ID: "u1"}}
	_, err := srv.DataService.Create(old)
	assert.NoError(t, err)

	comments, err := srv.DataService.Find(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}, "time", store.User{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(comments))

	// try to add new comment to the same old post
	_, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42",
		`{"text": "test 123", "locator":{"site": "remark42","url": "https://radio-t.com/blah1"}}`, adminPasswd)
	assert.Equal(t, http.StatusCreated, code)

	assert.NoError(t, srv.DataService.DeleteAll("remark42"))
	// make too old comment
	old = store.Comment{Text: "test test old", ParentID: "", Timestamp: time.Now().AddDate(0, 0, -15),
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}, User: store.User{ID: "u1"}}
	_, err = srv.DataService.Create(old)
	assert.NoError(t, err)

	_, code = call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42",
		`{"text": "test 123", "locator":{"site": "remark42","url": "https://radio-t.com/blah1"}}`, adminPasswd)
	assert.Equal(t, http.StatusForbidden, code)
}

func TestRest_CreateTooBig(t *testing.T) {
	ts, _ := startupT(t)

	longComment := fmt.Sprintf(`{"text": "%4001s", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, "Щ")

	b, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42", longComment, adminPasswd)
	assert.Equal(t, http.StatusBadRequest, code)
	c := R.JSON{}
	err := json.Unmarshal([]byte(b), &c)
	assert.NoError(t, err)
	assert.Equal(t, "comment text exceeded max allowed size 4000 (4001)", c["error"])
	assert.Equal(t, "invalid comment", c["details"])

	veryLongComment := fmt.Sprintf(`{"text": "%70000s", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, "Щ")
	b, code = call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42", veryLongComment, adminPasswd)
	assert.Equal(t, http.StatusBadRequest, code)
	c = R.JSON{}
	err = json.Unmarshal([]byte(b), &c)
	assert.NoError(t, err)
	assert.Equal(t, "http: request body too large", c["error"])
	assert.Equal(t, "can't bind comment", c["details"])
}

func TestRest_CreateWithRestrictedWord(t *testing.T) {
	ts, _ := startupT(t)

	badComment := `{"text": "What the duck is that?", "locator":{"url": "https://radio-t.com/blah1",
"site": "remark42"}}`

	b, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42", badComment, adminPasswd)
	assert.Equal(t, http.StatusBadRequest, code)
	c := R.JSON{}
	err := json.Unmarshal([]byte(b), &c)
	assert.NoError(t, err)
	assert.Equal(t, "comment contains restricted words", c["error"])
	assert.Equal(t, "invalid comment", c["details"])
	assert.EqualValues(t, rest.ErrCommentRestrictWords, c["code"], "the restricted-words code, the same for a new and an edited comment")
}

func TestRest_CreateRespelledPostURL(t *testing.T) {
	ts, srv := startupT(t, func(srv *Rest) { srv.UpdateLimiter = 100 })
	mockDestination := &fakeDestination{}
	srv.privRest.notifyService = notify.NewService(srv.DataService, 1, mockDestination)
	defer srv.privRest.notifyService.Close()

	oldPost := store.Locator{SiteID: "remark42", URL: "https://radio-t.com/old"}
	_, err := srv.DataService.Create(store.Comment{Text: "old", Timestamp: time.Now().AddDate(0, 0, -15),
		Locator: oldPost, User: store.User{ID: "u1", Name: "u1"}})
	require.NoError(t, err)
	closedPost := store.Locator{SiteID: "remark42", URL: "https://radio-t.com/closed"}
	_, err = srv.DataService.Create(store.Comment{Text: "closed", Locator: closedPost, User: store.User{ID: "u1", Name: "u1"}})
	require.NoError(t, err)
	require.NoError(t, srv.DataService.SetReadOnly(closedPost, true))
	// a moderator closes a post by the url the widget sends, which sanitizing changes
	require.NoError(t, srv.DataService.SetReadOnly(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/a|b"}, true))
	// a flag set by an older version is kept only under the url as it was sent
	_, err = srv.DataService.Engine.Flag(engine.FlagRequest{Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/e|f"},
		Flag: engine.ReadOnly, Update: engine.FlagTrue})
	require.NoError(t, err)
	// comments stored before urls were sanitized sit under the url as it was sent
	_, err = srv.DataService.Engine.Create(store.Comment{ID: "legacy", Text: "old", Timestamp: time.Now().AddDate(0, 0, -15),
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/c|d"}, User: store.User{ID: "u1", Name: "u1"}})
	require.NoError(t, err)

	send := func(t *testing.T, url string) (status int, c store.Comment) {
		t.Helper()
		body, err := json.Marshal(map[string]any{"text": "test 123", "locator": map[string]string{"site": "remark42", "url": url}})
		require.NoError(t, err)
		b, status := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42", string(body), adminPasswd)
		if status == http.StatusCreated {
			require.NoError(t, json.Unmarshal([]byte(b), &c))
		}
		return status, c
	}

	for _, tc := range []struct {
		url, post string
		stored    int
	}{
		{"HTTPS://radio-t.com/old", oldPost.URL, 1},
		{" https://radio-t.com/old ", oldPost.URL, 1},
		{"https://radio-t.com/old#", oldPost.URL, 1},
		{"HTTPS://radio-t.com/closed", closedPost.URL, 1},
		{"https://radio-t.com/a|b", "https://radio-t.com/a%7Cb", 0},
		{"HTTPS://radio-t.com/a|b", "https://radio-t.com/a%7Cb", 0},
		{"https://radio-t.com/a%7Cb", "https://radio-t.com/a%7Cb", 0},
		{"https://radio-t.com/e|f", "https://radio-t.com/e%7Cf", 0},
		{"https://radio-t.com/c|d", "https://radio-t.com/c%7Cd", 0},
	} {
		t.Run("read-only "+tc.url, func(t *testing.T) {
			status, _ := send(t, tc.url)
			assert.Equal(t, http.StatusForbidden, status)
			// a post with no comments has no bucket, and Find returns an error for it
			comments, _ := srv.DataService.Find(store.Locator{SiteID: "remark42", URL: tc.post}, "time", store.User{})
			assert.Len(t, comments, tc.stored, "nothing is added to the closed post")
		})
	}

	t.Run("open post", func(t *testing.T) {
		status, c := send(t, "HTTPS://radio-t.com/open#")
		require.Equal(t, http.StatusCreated, status)
		assert.Equal(t, "https://radio-t.com/open", c.Locator.URL)
		waitForCount(t, 1, func() int { return len(mockDestination.Get()) })
		assert.Equal(t, c.ID, mockDestination.Get()[0].Comment.ID)
	})

	t.Run("url sanitizing drops", func(t *testing.T) {
		status, _ := send(t, "javascript:alert(1)")
		assert.Equal(t, http.StatusBadRequest, status)
	})
}

func TestRest_CreateRelativeURL(t *testing.T) {
	ts, _ := startupT(t)

	// check that it's not possible to click insert URL button and not alter the URL in it (which is `url` by default)
	relativeURLText := `{"text": "here is a link with relative URL: [google.com](url)", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`
	b, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42", relativeURLText, adminPasswd)
	assert.Equal(t, http.StatusBadRequest, code)
	c := R.JSON{}
	err := json.Unmarshal([]byte(b), &c)
	assert.NoError(t, err)
	assert.Equal(t, "links should start with mailto:, http:// or https://", c["error"])
	assert.Equal(t, "invalid comment", c["details"])
}

func TestRest_CreateRejected(t *testing.T) {
	ts, _ := startupT(t)
	body := `{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`

	// try to create without auth
	_, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42", body, "")
	assert.Equal(t, http.StatusUnauthorized, code)

	// try with wrong aud
	_, code = call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42", body, devTokenBadAud)
	require.Equal(t, http.StatusForbidden, code, "reject wrong aud")
}

func TestRest_CreateBlockedDuringTitleFetch(t *testing.T) {
	var pageReads atomic.Int32
	var srvRef *Rest
	// the page blocks the author permanently while its title is fetched, as an admin's block request does
	tss := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		pageReads.Add(1)
		assert.NoError(t, srvRef.DataService.SetBlock("remark42", "provider1_dev", true, 0))
		assert.NoError(t, srvRef.DataService.DeleteUser("remark42", "provider1_dev", store.SoftDelete))
		_, err := w.Write([]byte("<html><title>post title</title><body>...</body></html>"))
		assert.NoError(t, err)
	}))
	t.Cleanup(tss.Close) // after the server of startupT stops, which is registered later

	ts, srv := startupT(t, func(srv *Rest) {
		srvRef = srv
		srv.DataService.TitleExtractor = service.NewTitleExtractor(http.Client{Timeout: 5 * time.Second}, []string{"127.0.0.1"})
	})

	create := func() (code int, body string) {
		reqBody := fmt.Sprintf(`{"text": "test 123", "locator":{"url": "%s/post1", "site": "remark42"}}`, tss.URL)
		body, code = call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42", reqBody, devToken)
		return code, body
	}

	code, body := create()
	assert.Equal(t, int32(1), pageReads.Load(), "the block was set during the title fetch")
	assert.Equal(t, http.StatusForbidden, code, body)
	comments, err := srv.DataService.Last("remark42", 10, time.Time{}, store.User{})
	require.NoError(t, err)
	assert.Empty(t, comments, "the comment of the blocked user is not stored")

	// the answer is the one a user blocked before posting gets
	earlyCode, earlyBody := create()
	assert.Equal(t, earlyCode, code)
	assert.Equal(t, earlyBody, body)
}

func TestRest_CreateWithWrongImage(t *testing.T) {
	ts, srv := startupT(t)
	logs := captureLog(t)

	// create comment
	body, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42",
		fmt.Sprintf(`{"text": "![non-existent.jpg](%s/api/v1/picture/dev_user/bad_picture)", "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`, srv.RemarkURL), adminPasswd)
	e := decodeError(t, body)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, errorResponse{Error: "image not found", Details: "can't load picture from the comment", Code: 20}, e,
		"the store error is replaced in the body")
	assert.NotContains(t, body, serverDir(srv), "no server path in the body")
	assert.Contains(t, logs.String(), serverDir(srv)+"/pics-remark42/staging/dev_user/62/bad_picture: no such file or directory",
		"the store error, with the path, is still logged")
}

func TestRest_CreateWithLazyImage(t *testing.T) {
	ts, _ := startupT(t)
	body := `{"text": "test 123 ![](http://example.com/image.png)", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`
	// create comment
	b, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42", body, adminPasswd)
	require.Equal(t, http.StatusCreated, code)
	c := store.Comment{}
	err := json.Unmarshal([]byte(b), &c)
	assert.NoError(t, err)
	assert.Equal(t, c.Text, "<p>test 123 <img src=\"http://example.com/image.png\" alt=\"\" loading=\"lazy\"/></p>\n")
}

func TestRest_CreateAndGet(t *testing.T) {
	ts, _ := startupT(t)

	// create comment
	b, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42",
		`{"text": "**test** *123*\n\n http://radio-t.com", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, adminPasswd)
	require.Equal(t, http.StatusCreated, code)
	c := R.JSON{}
	err := json.Unmarshal([]byte(b), &c)
	assert.NoError(t, err)

	id := c["id"].(string)

	// get created comment by id as admin
	comment := callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah1", ts.URL, id), "", adminPasswd)
	assert.Equal(t, "<p><strong>test</strong> <em>123</em></p>\n\n<p><a href=\"http://radio-t.com\" rel=\"nofollow ugc\">http://radio-t.com</a></p>\n", comment.Text)
	assert.Equal(t, "**test** *123*\n\n http://radio-t.com", comment.Orig)
	assert.Equal(t, store.User{Name: "admin", ID: "admin", Admin: true, Blocked: false,
		IP: "dbc7c999343f003f189f70aaf52cc04443f90790"},
		comment.User)

	// get created comment by id as non-admin
	comment = callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah1", ts.URL, id), "", devToken)
	assert.Equal(t, store.User{Name: "admin", ID: "admin", Admin: true, Blocked: false, IP: ""}, comment.User, "no ip")
}

func TestRest_CreateReplyWithoutParentRefused(t *testing.T) {
	ts, _ := startupT(t)

	top := addComment(t, store.Comment{Text: "top-level",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}, ts)
	otherPost := addComment(t, store.Comment{Text: "on another post",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah2"}}, ts)

	tbl := []struct {
		name, pid string
	}{
		{name: "unknown id", pid: "no-such-comment"},
		{name: "comment on another post", pid: otherPost},
		{name: "markup", pid: `<img src=x onerror=alert(1)>`},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(store.Comment{Text: "reply", ParentID: tt.pid,
				Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}})
			require.NoError(t, err)
			b, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42", string(body), adminPasswd)
			assert.Equal(t, http.StatusBadRequest, code, b)
			assert.JSONEq(t, `{"code":1,"details":"can't find parent comment","error":"parent comment not found"}`, b)
		})
	}

	// none of the replies was stored on the post they were sent to
	comments := callJSON[commentsWithInfo](t, http.MethodGet,
		ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah1", "", "")
	require.Len(t, comments.Comments, 1)
	assert.Equal(t, top, comments.Comments[0].ID)
	assert.Equal(t, 1, comments.Info.Count)
}

func TestRest_CreateReplyToExistingParent(t *testing.T) {
	ts, _ := startupT(t)

	const postURL = "https://radio-t.com/blah1"
	loc := store.Locator{SiteID: "remark42", URL: postURL}

	top := addComment(t, store.Comment{Text: "top-level", Locator: loc}, ts)
	reply := addComment(t, store.Comment{Text: "reply", ParentID: top, Locator: loc}, ts)

	// a comment deleted by its author can still be answered
	deletedByUser := addComment(t, store.Comment{Text: "deleted by user", Locator: loc}, ts)
	_, code := call(t, http.MethodPut, ts.URL+"/api/v1/comment/"+deletedByUser+"?site=remark42&url="+postURL,
		`{"delete": true}`, devToken)
	require.Equal(t, http.StatusOK, code)
	replyToUserDeleted := addComment(t, store.Comment{Text: "reply to deleted", ParentID: deletedByUser, Locator: loc}, ts)

	// and so can a comment deleted by an admin
	deletedByAdmin := addComment(t, store.Comment{Text: "deleted by admin", Locator: loc}, ts)
	_, code = call(t, http.MethodDelete,
		ts.URL+"/api/v1/admin/comment/"+deletedByAdmin+"?site=remark42&url="+postURL, "", adminUmputunToken)
	require.Equal(t, http.StatusOK, code)
	replyToAdminDeleted := addComment(t, store.Comment{Text: "reply to deleted", ParentID: deletedByAdmin, Locator: loc}, ts)

	// every reply is in the tree under its parent
	tree := callJSON[service.Tree](t, http.MethodGet, ts.URL+"/api/v1/find?site=remark42&format=tree&url="+postURL, "", "")
	replies := map[string][]string{}
	for _, n := range tree.Nodes {
		for _, r := range n.Replies {
			replies[n.Comment.ID] = append(replies[n.Comment.ID], r.Comment.ID)
		}
	}
	assert.Equal(t, map[string][]string{top: {reply}, deletedByUser: {replyToUserDeleted},
		deletedByAdmin: {replyToAdminDeleted}}, replies)
}

func TestRest_CreateReplyDepthLimit(t *testing.T) {
	ts, srv := startupT(t)

	const postURL = "https://radio-t.com/blah1"
	loc := store.Locator{SiteID: "remark42", URL: postURL}

	// a top-level comment with a chain of replies under it, each answering the one before, put
	// straight into the store. The last of them is one reply short of the limit
	deepest := ""
	for depth := range maxReplyDepth {
		id := fmt.Sprintf("depth-%d", depth)
		_, err := srv.DataService.Engine.Create(store.Comment{ID: id, ParentID: deepest, Text: "chain", Locator: loc,
			User: store.User{ID: "u1", Name: "user"}, Timestamp: time.Now()})
		require.NoError(t, err)
		deepest = id
	}

	// a reply to it stands at the limit and is taken
	atLimit := addComment(t, store.Comment{Text: "at the limit", ParentID: deepest, Locator: loc}, ts)

	// a reply to that one would stand past the limit
	body, err := json.Marshal(store.Comment{Text: "past the limit", ParentID: atLimit, Locator: loc})
	require.NoError(t, err)
	b, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42", string(body), adminPasswd)
	assert.Equal(t, http.StatusBadRequest, code, b)
	assert.JSONEq(t, `{"code":4,"details":"invalid comment",
		"error":"a reply can be at most 500 replies below a top-level comment"}`, b)

	// a comment higher up the same chain can still be answered
	branch := addComment(t, store.Comment{Text: "branch", ParentID: "depth-100", Locator: loc}, ts)

	// the post comes back as a tree with both replies that were taken and without the refused one
	res, code := get(t, ts.URL+"/api/v1/find?site=remark42&format=tree&url="+postURL)
	require.Equal(t, http.StatusOK, code)
	tree := service.Tree{}
	require.NoError(t, json.Unmarshal([]byte(res), &tree))
	ids := map[string]bool{}
	nodes := tree.Nodes
	for len(nodes) > 0 {
		n := nodes[len(nodes)-1]
		nodes = append(nodes[:len(nodes)-1], n.Replies...)
		ids[n.Comment.ID] = true
	}
	assert.Equal(t, maxReplyDepth+2, len(ids), "the chain and the two replies that were taken")
	assert.True(t, ids[atLimit], "the reply at the limit")
	assert.True(t, ids[branch], "the reply higher up the chain")
}

func TestRest_CreateReplyUnknownDepth(t *testing.T) {
	ts, srv := startupT(t)

	const postURL = "https://radio-t.com/blah1"
	loc := store.Locator{SiteID: "remark42", URL: postURL}

	// a comment whose parent is not in the store, as an import can leave one
	_, err := srv.DataService.Engine.Create(store.Comment{ID: "orphan", ParentID: "missing", Text: "orphan",
		Locator: loc, User: store.User{ID: "u1", Name: "user"}, Timestamp: time.Now()})
	require.NoError(t, err)

	// a reply to it can't be placed at a known depth and is refused before it is stored
	body, err := json.Marshal(store.Comment{Text: "reply", ParentID: "orphan", Locator: loc})
	require.NoError(t, err)
	b, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42", string(body), adminPasswd)
	assert.Equal(t, http.StatusInternalServerError, code, b)
	assert.JSONEq(t, `{"code":0,"details":"can't check reply depth","error":"can't read the thread"}`, b)

	comments, err := srv.DataService.Engine.Find(engine.FindRequest{Locator: loc, Sort: "time"})
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "orphan", comments[0].ID)
}

func TestRest_CreateWithQuotes(t *testing.T) {
	ts, srv := startupT(t)

	// create comment with quotes with smartypants
	b, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42",
		`{"text": "smartpants \"quoted\" text", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, adminPasswd)
	require.Equal(t, http.StatusCreated, code)
	c := R.JSON{}
	err := json.Unmarshal([]byte(b), &c)
	assert.NoError(t, err)
	id := c["id"].(string)

	// get created comment by id as non-admin
	comment := callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah1", ts.URL, id), "", devToken)
	assert.Equal(t, "<p>smartpants «quoted» text</p>\n", comment.Text)
	assert.Equal(t, "smartpants \"quoted\" text", comment.Orig)

	// create comment with quotes without smartypants
	srv.privRest.disableFancyTextFormatting = true
	b, code = call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42",
		`{"text": "no_smartpants \"quoted\" text", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, adminPasswd)
	require.Equal(t, http.StatusCreated, code)
	c = R.JSON{}
	err = json.Unmarshal([]byte(b), &c)
	assert.NoError(t, err)
	id = c["id"].(string)

	// get created comment by id as non-admin
	comment = callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah1", ts.URL, id), "", devToken)
	assert.Equal(t, "<p>no_smartpants &#34;quoted&#34; text</p>\n", comment.Text)
	assert.Equal(t, "no_smartpants \"quoted\" text", comment.Orig)
}

func TestRest_Update(t *testing.T) {
	ts, _ := startupT(t)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	id := addComment(t, c1, ts)

	beforeUpdate := time.Now()
	c2 := callJSON[store.Comment](t, http.MethodPut, ts.URL+"/api/v1/comment/"+id+"?site=remark42&url=https://radio-t.com/blah1",
		`{"text":"updated text", "summary":"my edit"}`, devToken)

	// comments returned by update
	assert.Equal(t, id, c2.ID)
	assert.Equal(t, "<p>updated text</p>\n", c2.Text)
	assert.Equal(t, "updated text", c2.Orig)
	assert.Equal(t, "my edit", c2.Edit.Summary)
	assert.WithinRange(t, c2.Edit.Timestamp, beforeUpdate, time.Now(), "edit stamped during the update")

	// read updated comment as its author
	c3 := callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah1", ts.URL, id), "", devToken)
	assert.Equal(t, c2, c3, "same as response from update")
}

func TestRest_UpdateSanitizesSummary(t *testing.T) {
	ts, srv := startupT(t)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	id := addComment(t, c1, ts)

	body, code := call(t, http.MethodPut, ts.URL+"/api/v1/comment/"+id+"?site=remark42&url=https://radio-t.com/blah1",
		`{"text":"updated text", "summary":"<script>alert(1)</script><b>why</b> & \"how\""}`, devToken)
	require.Equal(t, http.StatusOK, code, body)

	const want = `why &amp; "how"`
	updated := store.Comment{}
	require.NoError(t, json.Unmarshal([]byte(body), &updated))
	require.NotNil(t, updated.Edit)
	assert.Equal(t, want, updated.Edit.Summary, "update response")

	locator := store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}
	stored, err := srv.DataService.Engine.Get(engine.GetRequest{Locator: locator, CommentID: id})
	require.NoError(t, err)
	require.NotNil(t, stored.Edit)
	assert.Equal(t, want, stored.Edit.Summary, "stored record")

	byID := callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah1", ts.URL, id), "", "")
	require.NotNil(t, byID.Edit)
	assert.Equal(t, want, byID.Edit.Summary, "GET /id")

	found := callJSON[commentsWithInfo](t, http.MethodGet,
		ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah1", "", "")
	require.Len(t, found.Comments, 1)
	require.NotNil(t, found.Comments[0].Edit)
	assert.Equal(t, want, found.Comments[0].Edit.Summary, "GET /find")
}

func TestRest_ReadSanitizesStoredSummary(t *testing.T) {
	ts, srv := startupT(t)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	id := addComment(t, c1, ts)

	// a summary stored before edits cleaned it
	locator := store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}
	c, err := srv.DataService.Engine.Get(engine.GetRequest{Locator: locator, CommentID: id})
	require.NoError(t, err)
	c.Edit = &store.Edit{Timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Summary: "<script>alert(1)</script><b>why</b>"}
	require.NoError(t, srv.DataService.Engine.Update(c))

	byID := callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah1", ts.URL, id), "", "")
	require.NotNil(t, byID.Edit)
	assert.Equal(t, "why", byID.Edit.Summary, "GET /id")

	found := callJSON[commentsWithInfo](t, http.MethodGet,
		ts.URL+"/api/v1/find?site=remark42&url=https://radio-t.com/blah1", "", "")
	require.Len(t, found.Comments, 1)
	require.NotNil(t, found.Comments[0].Edit)
	assert.Equal(t, "why", found.Comments[0].Edit.Summary, "GET /find")
}

func TestRest_UpdateHidesVoters(t *testing.T) {
	ts, _ := startupT(t)

	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	id := addComment(t, c1, ts)

	// another user votes for the comment
	_, code := call(t, http.MethodPut,
		fmt.Sprintf("%s/api/v1/vote/%s?site=remark42&url=https://radio-t.com/blah1&vote=1", ts.URL, id), "", dev2Token)
	require.Equal(t, http.StatusOK, code)

	// the author edits it
	body, code := call(t, http.MethodPut, ts.URL+"/api/v1/comment/"+id+"?site=remark42&url=https://radio-t.com/blah1",
		`{"text":"updated text", "summary":"my edit"}`, devToken)
	require.Equal(t, http.StatusOK, code, body)

	c2 := store.Comment{}
	require.NoError(t, json.Unmarshal([]byte(body), &c2))
	assert.Equal(t, "<p>updated text</p>\n", c2.Text)
	assert.Equal(t, 1, c2.Score)
	assert.Nil(t, c2.Votes, "voters are hidden, as on every read")
	assert.Nil(t, c2.VotedIPs, "voters' ip hashes are hidden, as on every read")
	assert.NotContains(t, body, `"votes"`)
	assert.NotContains(t, body, `"voted_ips"`)

	// the author deletes it through the same endpoint
	body, code = call(t, http.MethodPut, ts.URL+"/api/v1/comment/"+id+"?site=remark42&url=https://radio-t.com/blah1",
		`{"delete": true, "summary":"removed"}`, devToken)
	require.Equal(t, http.StatusOK, code, body)

	c3 := store.Comment{}
	require.NoError(t, json.Unmarshal([]byte(body), &c3))
	assert.True(t, c3.Deleted)
	assert.Empty(t, c3.Text, "the answer to a delete is the deleted comment, not the one before it")
	assert.NotContains(t, body, `"votes"`)
	assert.NotContains(t, body, `"voted_ips"`)
	assert.NotContains(t, body, "updated text")
}

func TestRest_UpdateDelete(t *testing.T) {
	ts, _ := startupT(t)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	id := addComment(t, c1, ts)

	// check multi count updated
	j := callJSON[[]store.PostInfo](t, http.MethodPost, ts.URL+"/api/v1/counts?site=remark42", `["https://radio-t.com/blah1","https://radio-t.com/blah2"]`, adminPasswd)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/blah1", Count: 1},
		{URL: "https://radio-t.com/blah2", Count: 0}}, j)

	// delete a comment
	body, code := call(t, http.MethodPut, ts.URL+"/api/v1/comment/"+id+"?site=remark42&url=https://radio-t.com/blah1",
		`{"delete": true, "summary":"removed by user"}`, devToken)
	assert.Equal(t, http.StatusOK, code, body)

	// comments returned by update
	c2 := store.Comment{}
	err := json.Unmarshal([]byte(body), &c2)
	require.NoError(t, err)
	assert.Equal(t, id, c2.ID)
	assert.True(t, c2.Deleted)

	// read updated comment
	c3 := callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah1", ts.URL, id), "", devToken)
	assert.Equal(t, "", c3.Text)
	assert.Equal(t, "", c3.Orig)
	assert.True(t, c3.Deleted)

	// check multi count updated
	j = callJSON[[]store.PostInfo](t, http.MethodPost, ts.URL+"/api/v1/counts?site=remark42", `["https://radio-t.com/blah1","https://radio-t.com/blah2"]`, adminPasswd)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/blah1", Count: 0},
		{URL: "https://radio-t.com/blah2", Count: 0}}, j)
}

func TestRest_SavePictureSubscribersOnly(t *testing.T) {
	ts, srv := startupT(t, func(srv *Rest) { srv.SubscribersOnly = true })

	t.Run("not a subscriber", func(t *testing.T) {
		resp := uploadPicture(t, ts.URL, gopherPNG())
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		assert.Equal(t, "Access denied\n", string(body))
	})

	t.Run("subscriber", func(t *testing.T) {
		subscriber := token.User{ID: "provider1_subscriber", Name: "subscriber"}
		subscriber.SetPaidSub(true)
		rest.SetPaidSubUntil(&subscriber, time.Now().Add(time.Hour))
		tkn, err := srv.Authenticator.TokenService().Token(token.Claims{Audience: jwt.ClaimStrings{"remark42"}, Issuer: "remark42",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Minute)), User: &subscriber})
		require.NoError(t, err)

		resp := uploadPictureAs(t, ts.URL, tkn, gopherPNG())
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		assert.Equal(t, http.StatusOK, resp.StatusCode, string(body))
		assert.Contains(t, string(body), `"id":"provider1_subscriber/`)
	})
}

// a user deleting their own comment, or editing a picture out of it, removes the pictures they uploaded
// that no other comment on the page shows, and keeps another user's upload the comment showed
func TestRest_UpdatePictures(t *testing.T) {
	tbl := []struct {
		name    string
		body    string
		removed []string
		kept    []string
	}{
		{name: "delete", body: `{"delete": true}`, removed: []string{"provider1_dev/own.png", "provider1_dev/kept.png"},
			kept: []string{"provider1_dev/reused.png", "user2/other.png"}},
		{name: "picture edited out", body: `{"text": "![](https://demo.remark42.com/api/v1/picture/provider1_dev/kept.png)"}`,
			removed: []string{"provider1_dev/own.png"},
			kept:    []string{"provider1_dev/kept.png", "provider1_dev/reused.png", "user2/other.png"}},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			ts, srv := startupT(t)

			fs := &image.FileSystem{Location: t.TempDir(), Staging: t.TempDir()}
			imgSvc := image.NewService(fs, image.ServiceParams{ImageAPI: srv.RemarkURL + "/api/v1/picture/"})
			defer imgSvc.Close(context.Background())
			srv.DataService.ImageService = imgSvc

			pic := func(id string) string { return `<img src="` + srv.RemarkURL + `/api/v1/picture/` + id + `"/>` }
			for _, id := range []string{"provider1_dev/own.png", "provider1_dev/kept.png", "provider1_dev/reused.png", "user2/other.png"} {
				require.NoError(t, fs.Save(id, []byte("image data")))
				require.NoError(t, fs.Commit(id))
			}
			locator := store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}
			now := time.Now()
			for _, c := range []store.Comment{
				{ID: "c1", Text: pic("provider1_dev/own.png") + pic("provider1_dev/kept.png") + pic("provider1_dev/reused.png") +
					pic("user2/other.png"), Locator: locator, User: store.User{ID: "provider1_dev", Name: "developer one"}, Timestamp: now},
				// another user shows the author's upload on the same page
				{ID: "c2", Text: pic("provider1_dev/reused.png"), Locator: locator,
					User: store.User{ID: "user2", Name: "user2"}, Timestamp: now.Add(time.Second)},
			} {
				_, err := srv.DataService.Engine.Create(c) // directly to the engine, the pictures are already committed
				require.NoError(t, err)
			}

			body, code := call(t, http.MethodPut, ts.URL+"/api/v1/comment/c1?site=remark42&url=https://radio-t.com/blah",
				tt.body, devToken)
			require.Equal(t, http.StatusOK, code, body)

			for _, id := range tt.removed {
				_, err := fs.Load(id)
				assert.Error(t, err, "picture %s removed", id)
			}
			for _, id := range tt.kept {
				_, err := fs.Load(id)
				assert.NoError(t, err, "picture %s kept", id)
			}
		})
	}
}

func TestRest_UpdateDeletedByAdmin(t *testing.T) {
	ts, _ := startupT(t)

	const postURL = "https://radio-t.com/blah1"
	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42", URL: postURL}}
	id := addComment(t, c1, ts)

	_, code := call(t, http.MethodDelete,
		fmt.Sprintf("%s/api/v1/admin/comment/%s?site=remark42&url=%s", ts.URL, id, postURL), "", adminUmputunToken)
	require.Equal(t, http.StatusOK, code, "admin deletes the comment")

	put := func(body string) (code int, respBody string) {
		respBody, code = call(t, http.MethodPut, fmt.Sprintf("%s/api/v1/comment/%s?site=remark42&url=%s", ts.URL, id, postURL),
			body, devToken)
		return code, respBody
	}

	// the author edits the deleted comment inside the edit window
	code, body := put(`{"text":"back again", "summary":"my edit"}`)
	assert.Equal(t, http.StatusBadRequest, code, body)
	errResp := R.JSON{}
	require.NoError(t, json.Unmarshal([]byte(body), &errResp))
	assert.InDelta(t, float64(rest.ErrCommentRejected), errResp["code"], 0, body)
	assert.Equal(t, "can't update comment", errResp["details"], body)

	// readers still get the comment without text
	cr := callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=%s", ts.URL, id, postURL), "", "")
	assert.True(t, cr.Deleted)
	assert.Empty(t, cr.Text, "text of a deleted comment stays empty")
	assert.Empty(t, cr.Orig, "original text of a deleted comment stays empty")
	assert.Nil(t, cr.Edit, "deleted comment has no edit record")

	res, code := get(t, fmt.Sprintf("%s/api/v1/rss/post?site=remark42&url=%s", ts.URL, postURL))
	require.Equal(t, http.StatusOK, code, res)
	assert.NotContains(t, res, "back again", "post feed carries no text for the deleted comment")

	// a repeated delete request is answered as before
	code, body = put(`{"delete": true}`)
	assert.Equal(t, http.StatusOK, code, body)
	cd := store.Comment{}
	require.NoError(t, json.Unmarshal([]byte(body), &cd))
	assert.Equal(t, id, cd.ID)
	assert.True(t, cd.Deleted)
	assert.Empty(t, cd.Text)
}

func TestRest_UpdateBlockedUser(t *testing.T) {
	ts, srv := startupT(t)

	const postURL = "https://radio-t.com/blah1"
	c1 := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42", URL: postURL}}
	id := addComment(t, c1, ts)

	// a temporary block keeps the comments, and the token issued before the block is still valid
	require.NoError(t, srv.DataService.SetBlock("remark42", "provider1_dev", true, time.Hour))

	put := func(body string) (code int, respBody string) {
		respBody, code = call(t, http.MethodPut, fmt.Sprintf("%s/api/v1/comment/%s?site=remark42&url=%s", ts.URL, id, postURL),
			body, devToken)
		return code, respBody
	}
	locator := store.Locator{SiteID: "remark42", URL: postURL}

	code, body := put(`{"text":"updated text", "summary":"my edit"}`)
	assert.Equal(t, http.StatusForbidden, code, body)
	assert.Equal(t, `{"code":7,"details":"user blocked","error":"rejected"}`+"\n", body)

	cr, err := srv.DataService.Get(locator, id, store.User{})
	require.NoError(t, err)
	assert.Equal(t, "<p>test test #1</p>\n", cr.Text, "comment of a blocked user is not edited")
	assert.Nil(t, cr.Edit)

	// the blocked user can still delete their own comment
	code, body = put(`{"delete": true}`)
	assert.Equal(t, http.StatusOK, code, body)
	cr, err = srv.DataService.Get(locator, id, store.User{})
	require.NoError(t, err)
	assert.True(t, cr.Deleted)
}

func TestRest_UpdateReadOnlyPost(t *testing.T) {
	var rendered atomic.Int32
	ts, srv := startupT(t, func(srv *Rest) {
		srv.CommentFormatter = store.NewCommentFormatter(store.CommentConverterFunc(func(text string) string {
			rendered.Add(1)
			return text
		}))
	})

	put := func(tkn string, locator store.Locator, id, body string) (code int, respBody string) {
		respBody, code = call(t, http.MethodPut, fmt.Sprintf("%s/api/v1/comment/%s?site=remark42&url=%s", ts.URL, id, locator.URL),
			body, tkn)
		return code, respBody
	}
	// comments are stored directly: a post closed by age takes none through the API
	create := func(locator store.Locator, user store.User, ts time.Time) string {
		id, err := srv.DataService.Create(store.Comment{Text: "<p>original text</p>\n", Orig: "original text",
			Locator: locator, User: user, Timestamp: ts})
		require.NoError(t, err)
		return id
	}
	dev := store.User{ID: "provider1_dev", Name: "developer one"}
	admin := store.User{ID: "github_ef0f706a7", Name: "Umputun", Admin: true}

	tbl := []struct {
		name    string
		locator store.Locator
		before  func(t *testing.T, locator store.Locator) // runs before the two comments are stored
		after   func(t *testing.T, locator store.Locator) // runs after it
	}{
		{name: "closed by an admin", locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/closed"},
			after: func(t *testing.T, locator store.Locator) {
				require.NoError(t, srv.DataService.SetReadOnly(locator, true))
			}},
		{name: "closed by age", locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/old"},
			before: func(_ *testing.T, locator store.Locator) {
				// the test server closes a post ten days after its first comment
				create(locator, store.User{ID: "provider1_other", Name: "other"}, time.Now().AddDate(0, 0, -11))
			}},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			if tt.before != nil {
				tt.before(t, tt.locator)
			}
			devID := create(tt.locator, dev, time.Now())
			adminID := create(tt.locator, admin, time.Now())
			if tt.after != nil {
				tt.after(t, tt.locator)
			}
			info, err := srv.DataService.Info(tt.locator, srv.ReadOnlyAge)
			require.NoError(t, err)
			require.True(t, info.ReadOnly, "the post is closed")

			rendered.Store(0)
			code, body := put(devToken, tt.locator, devID, `{"text":"updated text", "summary":"my edit"}`)
			assert.Equal(t, http.StatusForbidden, code, body)
			assert.Equal(t, `{"code":8,"details":"old post, read-only","error":"rejected"}`+"\n", body)
			assert.Zero(t, rendered.Load(), "a refused edit is not rendered")
			c, err := srv.DataService.Get(tt.locator, devID, store.User{})
			require.NoError(t, err)
			assert.Equal(t, "<p>original text</p>\n", c.Text, "a comment on a closed post is not edited")
			assert.Nil(t, c.Edit)

			code, body = put(adminUmputunToken, tt.locator, adminID, `{"text":"updated text", "summary":"my edit"}`)
			assert.Equal(t, http.StatusOK, code, "an admin still edits their comment on a closed post: %s", body)
			c, err = srv.DataService.Get(tt.locator, adminID, store.User{})
			require.NoError(t, err)
			assert.Equal(t, "<p>updated text</p>\n", c.Text)

			code, body = put(devToken, tt.locator, devID, `{"delete": true}`)
			assert.Equal(t, http.StatusOK, code, "the author still deletes their comment on a closed post: %s", body)
			c, err = srv.DataService.Get(tt.locator, devID, store.User{})
			require.NoError(t, err)
			assert.True(t, c.Deleted)
		})
	}
}

func TestRest_UpdateDeleteDoesNotRenderText(t *testing.T) {
	var rendered atomic.Int32
	ts, _ := startupT(t, func(srv *Rest) {
		srv.CommentFormatter = store.NewCommentFormatter(store.CommentConverterFunc(func(text string) string {
			rendered.Add(1)
			return text
		}))
	})

	const postURL = "https://radio-t.com/blah1"
	id := addComment(t, store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42", URL: postURL}}, ts)
	put := func(body string) int {
		_, code := call(t, http.MethodPut, fmt.Sprintf("%s/api/v1/comment/%s?site=remark42&url=%s", ts.URL, id, postURL),
			body, devToken)
		return code
	}

	rendered.Store(0)
	require.Equal(t, http.StatusOK, put(`{"text":"updated text", "summary":"my edit"}`))
	require.Positive(t, rendered.Load(), "an edit renders its text")

	rendered.Store(0)
	require.Equal(t, http.StatusOK, put(`{"delete": true, "text":"text sent with the delete"}`))
	assert.Zero(t, rendered.Load(), "a delete does not render the text sent with it")
}

func TestRest_DeleteChildThenParent(t *testing.T) {
	ts, _ := startupT(t)

	p := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	idP := addComment(t, p, ts)

	c1 := store.Comment{Text: "test test #1", ParentID: idP,
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	idC1 := addComment(t, c1, ts)

	c2 := store.Comment{Text: "test test #1", ParentID: idP,
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	idC2 := addComment(t, c2, ts)

	// check multi count equals two
	j := callJSON[[]store.PostInfo](t, http.MethodPost, ts.URL+"/api/v1/counts?site=remark42", `["https://radio-t.com/blah1"]`, adminPasswd)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/blah1", Count: 3}}, j)

	// update a parent comment fails after child is created
	body, code := call(t, http.MethodPut, ts.URL+"/api/v1/comment/"+idP+"?site=remark42&url=https://radio-t.com/blah1",
		`{"text": "updated text", "summary":"updated by user"}`, devToken)
	assert.Equal(t, http.StatusBadRequest, code, body)

	// delete first child comment
	body, code = call(t, http.MethodPut, ts.URL+"/api/v1/comment/"+idC1+"?site=remark42&url=https://radio-t.com/blah1",
		`{"delete": true, "summary":"removed by user"}`, devToken)
	assert.Equal(t, http.StatusOK, code, body)

	// delete a parent comment, fails as one comment child still present
	body, code = call(t, http.MethodPut, ts.URL+"/api/v1/comment/"+idP+"?site=remark42&url=https://radio-t.com/blah1",
		`{"delete": true, "summary":"removed by user"}`, devToken)
	assert.Equal(t, http.StatusBadRequest, code, body)

	// delete second child comment, as an admin to check both deletion methods
	_, code = callAdminOnly(t, http.MethodDelete, fmt.Sprintf("%s/api/v1/admin/comment/%s?site=remark42&url=https://radio-t.com/blah1", ts.URL, idC2),
		adminUmputunToken)
	assert.Equal(t, http.StatusOK, code)

	// delete a parent comment, shouldn't fail after children are deleted
	body, code = call(t, http.MethodPut, ts.URL+"/api/v1/comment/"+idP+"?site=remark42&url=https://radio-t.com/blah1",
		`{"delete": true, "summary":"removed by user"}`, devToken)
	assert.Equal(t, http.StatusOK, code, body)

	// check multi count decremented to zero
	j = callJSON[[]store.PostInfo](t, http.MethodPost, ts.URL+"/api/v1/counts?site=remark42", `["https://radio-t.com/blah1"]`, adminPasswd)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/blah1", Count: 0}}, j)
}

func TestRest_UpdateNotOwner(t *testing.T) {
	ts, srv := startupT(t)

	c1 := store.Comment{Text: "test test #1", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}, User: store.User{ID: "xyz"}}
	id1, err := srv.DataService.Create(c1)
	assert.NoError(t, err)

	body, code := call(t, http.MethodPut, ts.URL+"/api/v1/comment/"+id1+"?site=remark42&url=https://radio-t.com/blah1",
		`{"text":"updated text", "summary":"my edit"}`, devToken)
	assert.Equal(t, http.StatusForbidden, code, body, "update from non-owner")
	assert.Equal(t, `{"code":3,"details":"can not edit comments for other users","error":"rejected"}`+"\n", body)

	_, code = call(t, http.MethodPut, ts.URL+"/api/v1/comment/"+id1+"?site=remark42&url=https://radio-t.com/blah1",
		`ERRR "text":"updated text", "summary":"my"}`, devToken)
	assert.Equal(t, http.StatusBadRequest, code, body, "update is not json")
}

func TestRest_UpdateWrongAud(t *testing.T) {
	ts, _ := startupT(t)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	id := addComment(t, c1, ts)

	_, code := call(t, http.MethodPut, ts.URL+"/api/v1/comment/"+id+"?site=remark42&url=https://radio-t.com/blah1",
		`{"text":"updated text", "summary":"my edit"}`, devTokenBadAud)
	assert.Equal(t, http.StatusForbidden, code, "reject update with wrong aut in jwt")
}

func TestRest_UpdateWithRestrictedWords(t *testing.T) {
	ts, _ := startupT(t)

	c1 := store.Comment{Text: "What the quack is that?",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	id := addComment(t, c1, ts)

	body, code := call(t, http.MethodPut, ts.URL+"/api/v1/comment/"+id+"?site=remark42&url=https://radio-t.com/blah1",
		`{"text":"What the duck is that?", "summary":"my edit"}`, devToken)
	c := R.JSON{}
	err := json.Unmarshal([]byte(body), &c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, code, body)
	assert.Equal(t, "comment contains restricted words", c["error"])
	assert.Equal(t, "invalid comment", c["details"])
	assert.EqualValues(t, rest.ErrCommentRestrictWords, c["code"], "the restricted-words code, the same for a new and an edited comment")
}

func TestRest_UpdateValidation(t *testing.T) {
	ts, srv := startupT(t, func(srv *Rest) {
		srv.DataService.MinCommentSize = 5
		srv.UpdateLimiter = 1000 // every subtest posts and edits on the protected routes
	})

	const postURL = "https://radio-t.com/blah1"
	locator := store.Locator{SiteID: "remark42", URL: postURL}

	send := func(method, path, tkn string, body any) (code int, respBody string) {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		respBody, code = call(t, method, ts.URL+path, string(b), tkn)
		return code, respBody
	}
	create := func(tkn, text string) (code int, respBody string) {
		return send(http.MethodPost, "/api/v1/comment?site=remark42", tkn,
			map[string]any{"text": text, "locator": map[string]string{"site": "remark42", "url": postURL}})
	}
	edit := func(tkn, id string, body map[string]any) (code int, respBody string) {
		return send(http.MethodPut, "/api/v1/comment/"+id+"?site=remark42&url="+postURL, tkn, body)
	}
	newComment := func(tkn, text string) string {
		code, body := create(tkn, text)
		require.Equal(t, http.StatusCreated, code, body)
		c := store.Comment{}
		require.NoError(t, json.Unmarshal([]byte(body), &c))
		return c.ID
	}

	atLimit := strings.Repeat("Щ", 4000)
	refused := []struct {
		name, tkn, text, errText string
	}{
		{"longer than max", devToken, atLimit + "Щ", "comment text exceeded max allowed size 4000 (4001)"},
		{"empty", devToken, "", "empty comment text"},
		{"shorter than min", devToken, "abcd", "comment text is smaller than min allowed size 5 (4)"},
		{"relative link", devToken, "a link [here](url)", "links should start with mailto:, http:// or https://"},
		{"javascript link", devToken, "a link [here](javascript:alert(1))", "links should start with mailto:, http:// or https://"},
		{"admin, longer than max", adminUmputunToken, atLimit + "Щ", "comment text exceeded max allowed size 4000 (4001)"},
	}
	for _, tt := range refused {
		t.Run("refused, "+tt.name, func(t *testing.T) {
			id := newComment(tt.tkn, "original text")
			c, err := srv.DataService.Get(locator, id, store.User{})
			require.NoError(t, err)
			require.Equal(t, tt.tkn == adminUmputunToken, c.User.Admin, "author is an admin only with the admin token")

			code, body := edit(tt.tkn, id, map[string]any{"text": tt.text, "summary": "my edit"})
			assert.Equal(t, http.StatusBadRequest, code, "edit answered %.200s", body)
			want := fmt.Sprintf(`{"code":%d,"details":"invalid comment","error":%q}`, rest.ErrCommentValidation, tt.errText)
			assert.JSONEq(t, want, body, "edit answered %.200s", body)

			// a new comment with the same text gets the same answer
			createCode, createBody := create(tt.tkn, tt.text)
			assert.Equal(t, http.StatusBadRequest, createCode, "create answered %.200s", createBody)
			assert.JSONEq(t, want, createBody)

			c, err = srv.DataService.Get(locator, id, store.User{})
			require.NoError(t, err)
			assert.Equal(t, "original text", c.Orig, "stored text unchanged")
			assert.Nil(t, c.Edit, "no edit recorded")
		})
	}

	// a request refused today for another reason keeps its answer
	t.Run("refused, another user's comment", func(t *testing.T) {
		id := newComment(dev2Token, "original text")
		code, body := edit(devToken, id, map[string]any{"text": ""})
		assert.Equal(t, http.StatusForbidden, code, body)
		assert.JSONEq(t, fmt.Sprintf(`{"code":%d,"details":"can not edit comments for other users","error":"rejected"}`,
			rest.ErrNoAccess), body)
	})

	accepted := []struct {
		name, orig, text string
	}{
		{"ordinary edit", "original text", "updated text"},
		{"exactly at max", "original text", atLimit},
		{"exactly at min", "original text", "abcde"},
		{"unchanged text", "original text", "original text"},
		{"whitespace only change", "original text", "original  text\n"},
		{"absolute and mailto links", "original text", "[a](https://example.com) and [b](mailto:me@example.com)"},
	}
	for _, tt := range accepted {
		t.Run("accepted, "+tt.name, func(t *testing.T) {
			id := newComment(devToken, tt.orig)
			code, body := edit(devToken, id, map[string]any{"text": tt.text, "summary": "my edit"})
			require.Equal(t, http.StatusOK, code, body)
			c := store.Comment{}
			require.NoError(t, json.Unmarshal([]byte(body), &c))
			assert.Equal(t, tt.text, c.Orig)
			require.NotNil(t, c.Edit)
			assert.Equal(t, "my edit", c.Edit.Summary)
		})
	}

	// a delete goes through the edit request with no text and is not validated as an empty one
	deletes := []struct {
		name string
		body map[string]any
	}{
		{"no text", map[string]any{"delete": true}},
		{"empty text", map[string]any{"delete": true, "text": ""}},
		{"text longer than max", map[string]any{"delete": true, "text": atLimit + "Щ"}},
	}
	for _, tt := range deletes {
		t.Run("delete through edit, "+tt.name, func(t *testing.T) {
			id := newComment(devToken, "original text")
			code, respBody := edit(devToken, id, tt.body)
			require.Equal(t, http.StatusOK, code, respBody)
			c, err := srv.DataService.Get(locator, id, store.User{})
			require.NoError(t, err)
			assert.True(t, c.Deleted)
			assert.Empty(t, c.Orig)
		})
	}
}

func TestRest_RestrictedWordsSplitByMarkup(t *testing.T) {
	ts, _ := startupT(t, func(srv *Rest) {
		srv.UpdateLimiter = 1000 // every row posts and edits on the protected routes
	})

	const postURL = "https://radio-t.com/blah1"
	id := addComment(t, store.Comment{Text: "What the quack is that?", Locator: store.Locator{SiteID: "remark42", URL: postURL}}, ts)

	send := func(t *testing.T, method, path string, body map[string]any) (code int, resp R.JSON) {
		t.Helper()
		b, err := json.Marshal(body)
		require.NoError(t, err)
		raw, code := call(t, method, ts.URL+path, string(b), devToken)
		resp = R.JSON{}
		require.NoError(t, json.Unmarshal([]byte(raw), &resp), "status %d, body %s", code, raw)
		return code, resp
	}

	tbl := []struct {
		name, text string
		refused    bool
	}{
		// markup or a character a reader does not see inside the word: the reader sees "duck"
		{"markdown emphasis", "What the **du**ck is that?", true},
		{"empty tag", "What the du<b></b>ck is that?", true},
		{"html comment", "What the du<!-- -->ck is that?", true},
		{"soft hyphen entity", "What the du&shy;ck is that?", true},
		{"zero-width space", "What the du\u200bck is that?", true},
		// a hand-written line break shows two lines, but tags are dropped without a space, so the words are joined
		{"line break tag", "What the du<br>ck is that?", true},
		// the word in a link address, with one letter percent-escaped
		{"escaped letter in a link address", "see [this](https://example.com/d%75ck) page", true},
		// the reader sees two words, or the markup itself
		{"space", "What the du ck is that?", false},
		{"two lines", "What the du\nck is that?", false},
		{"markdown line break", "What the du  \nck is that?", false},
		{"two paragraphs", "What the du\n\nck is that?", false},
		{"list items", "- du\n- ck", false},
		{"escaped tag", "What the du&lt;b&gt;&lt;/b&gt;ck is that?", false},
		{"tag in code span", "What the du`<b></b>`ck is that?", false},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			code, resp := send(t, http.MethodPost, "/api/v1/comment?site=remark42",
				map[string]any{"text": tt.text, "locator": map[string]string{"site": "remark42", "url": postURL}})
			if !tt.refused {
				assert.Equal(t, http.StatusCreated, code, "new comment %q: %v", tt.text, resp)
			} else {
				assert.Equal(t, http.StatusBadRequest, code, "new comment %q: %v", tt.text, resp)
				assert.Equal(t, "comment contains restricted words", resp["error"], "new comment %q", tt.text)
				assert.EqualValues(t, rest.ErrCommentRestrictWords, resp["code"], "new comment %q", tt.text)
			}

			code, resp = send(t, http.MethodPut, "/api/v1/comment/"+id+"?site=remark42&url="+postURL,
				map[string]any{"text": tt.text, "summary": "my edit"})
			if !tt.refused {
				assert.Equal(t, http.StatusOK, code, "edit to %q: %v", tt.text, resp)
			} else {
				assert.Equal(t, http.StatusBadRequest, code, "edit to %q: %v", tt.text, resp)
				assert.Equal(t, "comment contains restricted words", resp["error"], "edit to %q", tt.text)
				assert.EqualValues(t, rest.ErrCommentRestrictWords, resp["code"], "edit to %q", tt.text)
			}
		})
	}
}

func TestRest_Vote(t *testing.T) {
	ts, _ := startupT(t)

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}
	c2 := store.Comment{Text: "test test #2",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}

	id1 := addComment(t, c1, ts)
	addComment(t, c2, ts)

	vote := func(val int) int {
		_, code := call(t, http.MethodPut,
			fmt.Sprintf("%s/api/v1/vote/%s?site=remark42&url=https://radio-t.com/blah&vote=%d", ts.URL, id1, val), "", dev2Token)
		return code
	}

	assert.Equal(t, http.StatusOK, vote(1), "first vote allowed")
	assert.Equal(t, http.StatusBadRequest, vote(1), "second vote rejected")
	cr := callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1), "", dev2Token)
	assert.Equal(t, 1, cr.Score)
	assert.Equal(t, 1, cr.Vote)
	assert.Equal(t, map[string]bool(nil), cr.Votes, "hidden")
	assert.Equal(t, map[string]store.VotedIPInfo(nil), cr.VotedIPs, "hidden")

	assert.Equal(t, http.StatusOK, vote(-1), "opposite vote allowed")
	cr = callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1), "", dev2Token)
	assert.Equal(t, 0, cr.Score)
	assert.Equal(t, 0, cr.Vote)

	assert.Equal(t, http.StatusOK, vote(-1), "opposite vote allowed one more time")
	cr = callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1), "", dev2Token)
	assert.Equal(t, -1, cr.Score)
	assert.Equal(t, -1, cr.Vote)

	assert.Equal(t, http.StatusBadRequest, vote(-1), "dbl vote not allowed")
	cr = callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1), "", dev2Token)
	assert.Equal(t, -1, cr.Score)
	assert.Equal(t, -1, cr.Vote)

	cr = callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1), "", "")
	assert.Equal(t, -1, cr.Score)
	assert.Equal(t, 0, cr.Vote, "no vote info for not authed user")
	assert.Equal(t, map[string]bool(nil), cr.Votes, "hidden")
	assert.Equal(t, map[string]store.VotedIPInfo(nil), cr.VotedIPs, "hidden")

	cr = callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1), "", adminUmputunToken)
	assert.Equal(t, -1, cr.Score)
	assert.Equal(t, 0, cr.Vote, "no vote info for different user")
	assert.Equal(t, map[string]bool(nil), cr.Votes, "hidden")
	assert.Equal(t, map[string]store.VotedIPInfo(nil), cr.VotedIPs, "hidden")
}

func TestRest_AnonVote(t *testing.T) {
	// startServer starts a server with two comments and returns it with the id of the first one
	startServer := func(anonVote bool) (ts *httptest.Server, id1 string) {
		ts, _ = startupT(t, func(srv *Rest) { srv.AnonVote = anonVote })

		c1 := store.Comment{Text: "test test #1",
			Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}
		c2 := store.Comment{Text: "test test #2",
			Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}

		id1 = addComment(t, c1, ts)
		addComment(t, c2, ts)
		return ts, id1
	}

	vote := func(ts *httptest.Server, id string, val int) (body string, code int) {
		return call(t, http.MethodPut,
			fmt.Sprintf("%s/api/v1/vote/%s?site=remark42&url=https://radio-t.com/blah&vote=%d", ts.URL, id, val), "", anonToken)
	}

	ts, id1 := startServer(false)
	body, code := vote(ts, id1, 1)
	assert.Equal(t, http.StatusForbidden, code, "vote is disallowed with anonVote false")
	assert.Equal(t, "Access denied\n", body)

	ts, id1 = startServer(true)
	_, code = vote(ts, id1, 1)
	assert.Equal(t, http.StatusOK, code, "first vote allowed")
	_, code = vote(ts, id1, 1)
	assert.Equal(t, http.StatusBadRequest, code, "second vote rejected")
	cr := callJSON[store.Comment](t, http.MethodGet,
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1), "", anonToken)
	assert.Equal(t, 1, cr.Score)
	assert.Equal(t, 1, cr.Vote)
	assert.Equal(t, map[string]bool(nil), cr.Votes)
	assert.Equal(t, map[string]store.VotedIPInfo(nil), cr.VotedIPs)
}

func TestRest_EmailAndTelegram(t *testing.T) {
	ts, srv := startupT(t)

	srv.privRest.telegramService = &mockTelegram{site: "remark42"}

	// issue a good confirmation token, and the token of the unsubscribe link, which has the same
	// handshake and no provider
	goodToken := handshakeToken(t, srv, "provider1_dev::good@example.com", emailSubscribeProvider)
	unsubscribeToken := handshakeToken(t, srv, "provider1_dev::good@example.com", "")

	var testData = []struct {
		description  string
		url          string
		method       string
		responseCode int
		noAuth       bool
		cookieEmail  string
		body         string
	}{
		{description: "issue delete request without auth", url: "/api/v1/email", method: http.MethodDelete, responseCode: http.StatusUnauthorized, noAuth: true},
		{description: "issue delete request without site_id", url: "/api/v1/email", method: http.MethodDelete, responseCode: http.StatusForbidden},
		{description: "delete non-existent user email", url: "/api/v1/email?site=remark42", method: http.MethodDelete, responseCode: http.StatusOK},
		{description: "set user email, token not set", url: "/api/v1/email/confirm?site=remark42", method: http.MethodPost, responseCode: http.StatusBadRequest, body: `{"site":"remark42"}`},
		{description: "set user email, token not set, old query param", url: "/api/v1/email/confirm?site=remark42", method: http.MethodPost, responseCode: http.StatusBadRequest},
		{description: "send email confirmation without address", url: "/api/v1/email/subscribe?site=remark42", method: http.MethodPost, responseCode: http.StatusBadRequest, body: `{"site":"remark42"}`},
		{description: "send email confirmation without address, old query param", url: "/api/v1/email/subscribe?site=remark42", method: http.MethodPost, responseCode: http.StatusBadRequest},
		{description: "send email confirmation", url: "/api/v1/email/subscribe?site=remark42", method: http.MethodPost, responseCode: http.StatusOK, body: `{"site":"remark42","address":"good@example.com"}`},
		{description: "send email confirmation, old query param", url: "/api/v1/email/subscribe?site=remark42&address=good@example.com", method: http.MethodPost, responseCode: http.StatusOK},
		{description: "set user email, token is good", url: "/api/v1/email/confirm?site=remark42", method: http.MethodPost, responseCode: http.StatusOK, cookieEmail: "good@example.com", body: fmt.Sprintf(`{"site":"remark42","token":%q}`, goodToken)},
		{description: "set user email, token is good, old query param", url: fmt.Sprintf("/api/v1/email/confirm?site=remark42&tkn=%s", goodToken), method: http.MethodPost, responseCode: http.StatusOK, cookieEmail: "good@example.com"},
		{description: "send confirmation with same address", url: "/api/v1/email/subscribe?site=remark42&address=good@example.com", method: http.MethodPost, responseCode: http.StatusConflict},
		{description: "get user email", url: "/api/v1/email?site=remark42", method: http.MethodGet, responseCode: http.StatusOK},
		{description: "delete user email", url: "/api/v1/email?site=remark42", method: http.MethodDelete, responseCode: http.StatusOK},
		{description: "send another confirmation", url: "/api/v1/email/subscribe?site=remark42&address=good@example.com", method: http.MethodPost, responseCode: http.StatusOK},
		{description: "set user email, token is good", url: "/api/v1/email/confirm?site=remark42", method: http.MethodPost, responseCode: http.StatusOK, cookieEmail: "good@example.com", body: fmt.Sprintf(`{"site":"remark42","token":%q}`, goodToken)},
		{description: "set user email, token is good, old query param", url: fmt.Sprintf("/api/v1/email/confirm?site=remark42&tkn=%s", goodToken), method: http.MethodPost, responseCode: http.StatusOK, cookieEmail: "good@example.com"},
		{description: "unsubscribe user, no token", url: "/email/unsubscribe.html?site=remark42", method: http.MethodPost, responseCode: http.StatusBadRequest},
		{description: "unsubscribe user, wrong token", url: "/email/unsubscribe.html?site=remark42&tkn=jwt", method: http.MethodGet, responseCode: http.StatusForbidden},
		{description: "set user email, unsubscribe token", url: "/api/v1/email/confirm?site=remark42", method: http.MethodPost, responseCode: http.StatusBadRequest, body: fmt.Sprintf(`{"site":"remark42","token":%q}`, unsubscribeToken)},
		{description: "set user email, unsubscribe token, old query param", url: fmt.Sprintf("/api/v1/email/confirm?site=remark42&tkn=%s", unsubscribeToken), method: http.MethodPost, responseCode: http.StatusBadRequest},
		{description: "unsubscribe user, confirmation token", url: fmt.Sprintf("/email/unsubscribe.html?site=remark42&tkn=%s", goodToken), method: http.MethodPost, responseCode: http.StatusBadRequest},
		{description: "unsubscribe user, good token", url: fmt.Sprintf("/email/unsubscribe.html?site=remark42&tkn=%s", unsubscribeToken), method: http.MethodPost, responseCode: http.StatusOK},
		{description: "unsubscribe user second time, good token", url: fmt.Sprintf("/email/unsubscribe.html?site=remark42&tkn=%s", unsubscribeToken), method: http.MethodPost, responseCode: http.StatusConflict},
		{description: "issue delete request without auth", url: "/api/v1/telegram", method: http.MethodDelete, responseCode: http.StatusUnauthorized, noAuth: true},
		{description: "issue delete request without site_id", url: "/api/v1/telegram", method: http.MethodDelete, responseCode: http.StatusForbidden},
		{description: "delete non-existent user telegram", url: "/api/v1/telegram?site=remark42", method: http.MethodDelete, responseCode: http.StatusOK},
		{description: "send telegram confirmation, no siteID", url: "/api/v1/telegram/subscribe", method: http.MethodGet, responseCode: http.StatusForbidden},
		{description: "send telegram confirmation", url: "/api/v1/telegram/subscribe?site=remark42", method: http.MethodGet, responseCode: http.StatusOK},
		{description: "set user telegram, token is good", url: "/api/v1/telegram/subscribe?site=remark42&tkn=good_token", method: http.MethodGet, responseCode: http.StatusOK},
		{description: "send confirmation with same address", url: "/api/v1/telegram/subscribe?site=remark42", method: http.MethodGet, responseCode: http.StatusConflict},
		{description: "delete user telegram", url: "/api/v1/telegram?site=remark42", method: http.MethodDelete, responseCode: http.StatusOK},
		{description: "send another confirmation", url: "/api/v1/telegram/subscribe?site=remark42", method: http.MethodGet, responseCode: http.StatusOK},
		{description: "set user telegram, token is good", url: "/api/v1/telegram/subscribe?site=remark42&tkn=good_token", method: http.MethodGet, responseCode: http.StatusOK},
		// telegramSubscribeCtrl mutates state, so HEAD (which stdlib ServeMux would route to the
		// GET handler) must be rejected by rejectHead before it runs
		{description: "HEAD is rejected on telegram subscribe", url: "/api/v1/telegram/subscribe?site=remark42", method: http.MethodHead, responseCode: http.StatusMethodNotAllowed},
	}
	client := http.Client{}
	defer client.CloseIdleConnections()
	for _, x := range testData {
		t.Run(x.description, func(t *testing.T) {
			reqBody := io.NopCloser(strings.NewReader(x.body))
			if x.body == "" {
				reqBody = http.NoBody
			}
			req, err := http.NewRequest(x.method, ts.URL+x.url, reqBody)
			require.NoError(t, err)
			if !x.noAuth {
				req.Header.Add("X-JWT", devToken)
			}
			resp, err := client.Do(req)
			require.NoError(t, err)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.NoError(t, resp.Body.Close())
			// read User.Email from the token in the cookie
			for _, c := range resp.Cookies() {
				if c.Name == "JWT" {
					claims, err := srv.Authenticator.TokenService().Parse(c.Value)
					require.NoError(t, err)
					assert.Equal(t, x.cookieEmail, claims.User.Email, "cookie email check failed")
				}
			}
			assert.Equal(t, x.responseCode, resp.StatusCode, string(body))
		})
	}
}

// fakeDestination is a notify.Destination recording what it was asked to send
type fakeDestination struct {
	mu            sync.Mutex
	requests      []notify.Request
	verifications []notify.VerificationRequest
}

func (f *fakeDestination) Send(_ context.Context, r notify.Request) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, r)
	return nil
}

func (f *fakeDestination) SendVerification(_ context.Context, v notify.VerificationRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.verifications = append(f.verifications, v)
	return nil
}

func (f *fakeDestination) String() string { return "fake destination" }

func (f *fakeDestination) Get() []notify.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.requests)
}

func (f *fakeDestination) GetVerify() []notify.VerificationRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.verifications)
}

// TestRest_EmailSubscribeTakesOnlyBareAddress checks what the subscription accepts as an address.
// The value ends up in the confirmation mail and in the mailto destination the mailer reads it from
func TestRest_EmailSubscribeTakesOnlyBareAddress(t *testing.T) {
	ts, _ := startupT(t)

	tbl := []struct {
		name    string
		address string
		status  int
	}{
		{name: "display name", address: `"unlock it at https://evil.example.com" <good@example.com>`, status: http.StatusBadRequest},
		{name: "comment", address: "good@example.com (comment)", status: http.StatusBadRequest},
		{name: "two addresses", address: "good@example.com,other@example.com", status: http.StatusBadRequest},
		{name: "sender and subject of its own", address: "good@example.com?from=support%40bank.example.com&subject=Your+account+is+locked&x=",
			status: http.StatusBadRequest},
		{name: "address the mailer can't take", address: "first#last@example.com", status: http.StatusBadRequest},
		{name: "as typed, with spaces around it", address: " good@example.com ", status: http.StatusOK},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(map[string]string{"site": "remark42", "address": tt.address})
			require.NoError(t, err)
			respBody, code := call(t, http.MethodPost, ts.URL+"/api/v1/email/subscribe?site=remark42", string(body), devToken)
			assert.Equal(t, tt.status, code, respBody)
			if tt.status == http.StatusOK {
				assert.Contains(t, respBody, `"address":"good@example.com"`, "the address without the spaces")
			}
		})
	}
}

func TestRest_EmailNotification(t *testing.T) {
	ts, srv := startupT(t)

	mockDestination := &fakeDestination{}
	srv.privRest.notifyService = notify.NewService(srv.DataService, 1, mockDestination)
	defer srv.privRest.notifyService.Close()

	// create new comment from dev user
	body, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42",
		`{"text": "test 123", "user": {"name": "provider1_dev::good@example.com"}, "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, devToken)
	require.Equal(t, http.StatusCreated, code, body)
	parentComment := store.Comment{}
	require.NoError(t, json.Unmarshal([]byte(body), &parentComment))
	waitForCount(t, 1, func() int { return len(mockDestination.Get()) })
	assert.Empty(t, mockDestination.Get()[0].Emails)

	// create child comment from another user, email notification only to admin expected
	body, code = call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42",
		fmt.Sprintf(`{"text": "test 456", "pid": %q, "user": {"name": "other_user"}, "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, parentComment.ID), anonToken)
	require.Equal(t, http.StatusCreated, code, body)
	waitForCount(t, 2, func() int { return len(mockDestination.Get()) })
	assert.Empty(t, mockDestination.Get()[1].Emails)

	// send confirmation token for email
	body, code = call(t, http.MethodPost, ts.URL+"/api/v1/email/subscribe?site=remark42",
		`{"site": "remark42", "address": "good@example.com"}`, devToken)
	require.Equal(t, http.StatusOK, code, body)
	waitForCount(t, 1, func() int { return len(mockDestination.GetVerify()) })
	assert.Equal(t, "good@example.com", mockDestination.GetVerify()[0].Email)
	verificationToken := mockDestination.GetVerify()[0].Token

	// get user information to verify lack of the subscription
	clearUser := callJSON[store.User](t, http.MethodGet, ts.URL+"/api/v1/user?site=remark42", "", devToken)
	assert.Equal(t, store.User{Name: "developer one", ID: "provider1_dev", EmailSubscription: false,
		Picture: "http://example.com/pic.png", IP: "127.0.0.1", SiteID: "remark42"}, clearUser)

	// verify email
	body, code = call(t, http.MethodPost, ts.URL+"/api/v1/email/confirm?site=remark42",
		fmt.Sprintf(`{"site": "remark42", "token": %q}`, verificationToken), devToken)
	require.Equal(t, http.StatusOK, code, body)

	// get user information to verify the subscription
	subscribedUser := callJSON[store.User](t, http.MethodGet, ts.URL+"/api/v1/user?site=remark42", "", devToken)
	assert.Equal(t, store.User{Name: "developer one", ID: "provider1_dev", EmailSubscription: true,
		Picture: "http://example.com/pic.png", IP: "127.0.0.1", SiteID: "remark42"}, subscribedUser)

	// create child comment from another user, email notification expected
	body, code = call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42",
		fmt.Sprintf(`{"text": "test 789", "pid": %q, "user": {"name": "other_user"}, "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, parentComment.ID), anonToken)
	require.Equal(t, http.StatusCreated, code, body)
	waitForCount(t, 3, func() int { return len(mockDestination.Get()) })
	assert.Equal(t, []string{"good@example.com"}, mockDestination.Get()[2].Emails)

	// delete user's email
	body, code = call(t, http.MethodDelete, ts.URL+"/api/v1/email?site=remark42", "", devToken)
	assert.Equal(t, http.StatusOK, code, body)

	// create child comment from another user, no email notification
	body, code = call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42",
		`{"text": "test 321", "user": {"name": "other_user"}, "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, devToken)
	require.Equal(t, http.StatusCreated, code, body)
	waitForCountSettled(t, 4, func() int { return len(mockDestination.Get()) })
	assert.Empty(t, mockDestination.Get()[3].Emails)

	// confirm email via subscribe call with query params, old behavior, email notification is expected
	body, code = call(t, http.MethodPost,
		ts.URL+"/api/v1/email/subscribe?site=remark42&address=good@example.com", "", emailUserToken)
	require.Equal(t, http.StatusOK, code, body)
	waitForCount(t, 2, func() int { return len(mockDestination.GetVerify()) }, "verification email was sent")

	// get email user information to verify there is no subscription yet
	unsubscribedEmailUser := callJSON[store.User](t, http.MethodGet, ts.URL+"/api/v1/user?site=remark42", "", emailUserToken)
	assert.Equal(t, store.User{Name: "good@example.com test user", ID: "email_f5dfe9d2e6bd75fc74ea5fabf273b45b5baeb195", EmailSubscription: false,
		Picture: "http://example.com/pic.png", IP: "127.0.0.1", SiteID: "remark42"}, unsubscribedEmailUser)

	// confirm email via subscribe call, no email notification is expected
	body, code = call(t, http.MethodPost, ts.URL+"/api/v1/email/subscribe?site=remark42",
		`{"site": "remark42", "address": "good@example.com"}`, emailUserToken)
	require.Equal(t, http.StatusOK, code, body)
	waitForCountSettled(t, 2, func() int { return len(mockDestination.GetVerify()) }, "no new verification email was sent")

	// get email user information to verify the subscription happened without the confirmation call
	subscribedEmailUser := callJSON[store.User](t, http.MethodGet, ts.URL+"/api/v1/user?site=remark42", "", emailUserToken)
	assert.Equal(t, store.User{Name: "good@example.com test user", ID: "email_f5dfe9d2e6bd75fc74ea5fabf273b45b5baeb195", EmailSubscription: true,
		Picture: "http://example.com/pic.png", IP: "127.0.0.1", SiteID: "remark42"}, subscribedEmailUser)
}

func TestRest_TelegramNotification(t *testing.T) {
	ts, srv := startupT(t)

	mockDestination := &fakeDestination{}
	srv.privRest.notifyService = notify.NewService(srv.DataService, 1, mockDestination)
	defer srv.privRest.notifyService.Close()

	// create new comment from dev user
	body, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42",
		`{"text": "test 123", "user": {"name": "provider1_dev::good@example.com"}, "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, devToken)
	require.Equal(t, http.StatusCreated, code, body)
	parentComment := store.Comment{}
	require.NoError(t, json.Unmarshal([]byte(body), &parentComment))
	waitForCount(t, 1, func() int { return len(mockDestination.Get()) })
	assert.Empty(t, mockDestination.Get()[0].Telegrams)

	// create child comment from another user, telegram notification only to admin expected
	body, code = call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42",
		fmt.Sprintf(`{"text": "test 456", "pid": %q, "user": {"name": "other_user"}, "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, parentComment.ID), anonToken)
	require.Equal(t, http.StatusCreated, code, body)
	waitForCount(t, 2, func() int { return len(mockDestination.Get()) })
	assert.Empty(t, mockDestination.Get()[1].Telegrams)

	// subscribe to telegram while the telegram destination is absent
	body, code = call(t, http.MethodGet, ts.URL+"/api/v1/telegram/subscribe?site=remark42", "", devToken)
	require.Equal(t, http.StatusInternalServerError, code, body)
	assert.Equal(t, `{"code":17,"details":"telegram notifications are not enabled","error":"not enabled"}`+"\n", body)

	mockTlgrm := &mockTelegram{notVerified: true, site: "unknown_site"}
	srv.privRest.telegramService = mockTlgrm
	// send confirmation token for telegram
	body, code = call(t, http.MethodGet, ts.URL+"/api/v1/telegram/subscribe?site=remark42", "", devToken)
	require.Equal(t, http.StatusOK, code, body)
	var subscribeRequest struct {
		Bot   string `json:"bot"`
		Token string `json:"token"`
	}
	err := json.Unmarshal([]byte(body), &subscribeRequest)
	assert.NoError(t, err)
	assert.Equal(t, "botUsername", subscribeRequest.Bot)
	assert.Regexp(t, "^[0-9a-f]{40}$", subscribeRequest.Token, "the bot is handed a 40 hex character token")

	// verify telegram, unsuccessfully because of not verified
	body, code = call(t, http.MethodGet,
		ts.URL+fmt.Sprintf("/api/v1/telegram/subscribe?site=remark42&tkn=%s", subscribeRequest.Token), "", devToken)
	require.Equal(t, http.StatusNotFound, code, body)
	require.Equal(t, `{"code":0,"details":"request is not verified yet","error":"not verified"}`+"\n", body)

	mockTlgrm.notVerified = false

	// verify telegram, unsuccessfully because of unknown site
	body, code = call(t, http.MethodGet,
		ts.URL+fmt.Sprintf("/api/v1/telegram/subscribe?site=remark42&tkn=%s", subscribeRequest.Token), "", devToken)
	require.Equal(t, http.StatusBadRequest, code, body)
	require.Equal(t, `{"code":0,"details":"can't set telegram for user","error":"site \"unknown_site\" not found"}`+"\n", body)

	mockTlgrm.site = "remark42"
	// verify telegram, successfully
	body, code = call(t, http.MethodGet,
		ts.URL+fmt.Sprintf("/api/v1/telegram/subscribe?site=remark42&tkn=%s", subscribeRequest.Token), "", devToken)
	require.Equal(t, http.StatusOK, code, body)
	assert.Equal(t, `{"updated":true}`+"\n", body, "the answer does not name the chat the bot was started from")

	// get user information to verify the subscription
	user := callJSON[store.User](t, http.MethodGet, ts.URL+"/api/v1/user?site=remark42", "", devToken)
	assert.Equal(t, store.User{Name: "developer one", ID: "provider1_dev",
		Picture: "http://example.com/pic.png", IP: "127.0.0.1", SiteID: "remark42"}, user)

	// create child comment from another user, telegram notification expected
	body, code = call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42",
		fmt.Sprintf(`{"text": "test 789", "pid": %q, "user": {"name": "other_user"}, "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, parentComment.ID), anonToken)
	require.Equal(t, http.StatusCreated, code, body)
	waitForCount(t, 3, func() int { return len(mockDestination.Get()) })
	assert.Equal(t, []string{"good_telegram"}, mockDestination.Get()[2].Telegrams)

	// delete user's telegram
	body, code = call(t, http.MethodDelete, ts.URL+"/api/v1/telegram?site=remark42", "", devToken)
	assert.Equal(t, http.StatusOK, code, body)

	// create child comment from another user, no telegram notification
	body, code = call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42",
		`{"text": "test 321", "user": {"name": "other_user"}, "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, devToken)
	require.Equal(t, http.StatusCreated, code, body)
	waitForCountSettled(t, 4, func() int { return len(mockDestination.Get()) })
	assert.Empty(t, mockDestination.Get()[3].Telegrams)
}

func TestRest_UserAllData(t *testing.T) {
	ts, srv := startupT(t)

	// write 3 comments
	user := store.User{ID: "provider1_dev", Name: "user name 1"}
	c1 := store.Comment{User: user, Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 5, 27, 1, 14, 10, 0, time.Local)}
	c2 := store.Comment{User: user, Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 5, 27, 1, 14, 20, 0, time.Local)}
	c3 := store.Comment{User: user, Text: "test test #3", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 5, 27, 1, 14, 25, 0, time.Local)}
	_, err := srv.DataService.Create(c1)
	require.NoError(t, err, "%+v", err)
	_, err = srv.DataService.Create(c2)
	require.NoError(t, err)
	_, err = srv.DataService.Create(c3)
	require.NoError(t, err)

	client := &http.Client{Timeout: waitTimeout}
	defer client.CloseIdleConnections()
	req, err := http.NewRequest("GET", ts.URL+"/api/v1/userdata?site=remark42", http.NoBody)
	require.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/gzip", resp.Header.Get("Content-Type"))

	ungzReader, err := gzip.NewReader(resp.Body)
	require.NoError(t, err)
	defer ungzReader.Close()
	require.NoError(t, resp.Body.Close())
	ungzBody, err := io.ReadAll(ungzReader)
	require.NoError(t, err)
	strUungzBody := string(ungzBody)
	assert.True(t, strings.HasPrefix(strUungzBody,
		`{"info": {"name":"developer one","id":"provider1_dev","picture":"http://example.com/pic.png","ip":"127.0.0.1","admin":false,"site_id":"remark42"}, "comments":[{`))
	assert.Equal(t, 3, strings.Count(strUungzBody, `"text":`), "3 comments inside")

	parsed := struct {
		Info     store.User      `json:"info"`
		Comments []store.Comment `json:"comments"`
	}{}

	err = json.Unmarshal(ungzBody, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, store.User{Name: "developer one", ID: "provider1_dev",
		Picture: "http://example.com/pic.png", IP: "127.0.0.1", SiteID: "remark42"}, parsed.Info)
	assert.Equal(t, 3, len(parsed.Comments))

	_, code := call(t, http.MethodGet, ts.URL+"/api/v1/userdata?site=remark42", "", "")
	require.Equal(t, http.StatusUnauthorized, code)
}

func TestRest_UserAllDataManyComments(t *testing.T) {
	ts, srv := startupT(t)

	user := store.User{ID: "provider1_dev", Name: "user name 1"}
	c := store.Comment{User: user, Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 5, 27, 1, 14, 10, 0, time.Local)}

	for i := range 51 {
		c.ID = fmt.Sprintf("id-%03d", i)
		c.Timestamp = c.Timestamp.Add(time.Second)
		_, err := srv.DataService.Create(c)
		require.NoError(t, err)
	}
	client := &http.Client{Timeout: waitTimeout}
	defer client.CloseIdleConnections()
	req, err := http.NewRequest("GET", ts.URL+"/api/v1/userdata?site=remark42", http.NoBody)
	require.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/gzip", resp.Header.Get("Content-Type"))

	ungzReader, err := gzip.NewReader(resp.Body)
	assert.NoError(t, err)
	defer ungzReader.Close()
	assert.NoError(t, resp.Body.Close())
	ungzBody, err := io.ReadAll(ungzReader)
	assert.NoError(t, err)
	strUngzBody := string(ungzBody)
	assert.True(t, strings.HasPrefix(strUngzBody,
		`{"info": {"name":"developer one","id":"provider1_dev","picture":"http://example.com/pic.png","ip":"127.0.0.1","admin":false,"site_id":"remark42"}, "comments":[{`))
	assert.Equal(t, 51, strings.Count(strUngzBody, `"text":`), "51 comments inside")
}

func TestRest_UserAllDataPageBoundaries(t *testing.T) {
	for _, n := range []int{100, 101, 250} {
		t.Run(fmt.Sprintf("%d comments", n), func(t *testing.T) {
			ts, srv := startupT(t)

			user := store.User{ID: "provider1_dev", Name: "user name 1"}
			c := store.Comment{User: user, Text: "test", Locator: store.Locator{SiteID: "remark42",
				URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 5, 27, 1, 14, 10, 0, time.UTC)}
			for i := range n {
				c.ID = fmt.Sprintf("id-%03d", i)
				c.Timestamp = c.Timestamp.Add(time.Second)
				_, err := srv.DataService.Create(c)
				require.NoError(t, err)
			}

			parsed := getUserAllData(t, ts.URL)
			assert.Equal(t, n, len(parsed.Comments), "every comment is exported in one JSON array")
		})
	}
}

func TestRest_UserAllDataNoComments(t *testing.T) {
	ts, _ := startupT(t)

	parsed := getUserAllData(t, ts.URL)
	assert.Equal(t, "provider1_dev", parsed.Info.ID)
	assert.Empty(t, parsed.Comments)
}

// getUserAllData requests /userdata as the dev user and decodes the gzipped JSON export
func getUserAllData(t *testing.T, url string) (res struct {
	Info     store.User      `json:"info"`
	Comments []store.Comment `json:"comments"`
}) {
	t.Helper()
	client := &http.Client{Timeout: waitTimeout}
	defer client.CloseIdleConnections()
	req, err := http.NewRequest("GET", url+"/api/v1/userdata?site=remark42", http.NoBody)
	require.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/gzip", resp.Header.Get("Content-Type"))

	ungzReader, err := gzip.NewReader(resp.Body)
	require.NoError(t, err)
	defer ungzReader.Close()
	body, err := io.ReadAll(ungzReader)
	require.NoError(t, err, "gzip body decodes")
	require.NoError(t, json.Unmarshal(body, &res), "export is valid JSON")
	return res
}

func TestRest_UserAllDataDetails(t *testing.T) {
	const info = `{"info": {"name":"developer one","id":"provider1_dev","picture":"http://example.com/pic.png","ip":"127.0.0.1","admin":false,"site_id":"remark42"}`
	type details struct {
		Email    string `json:"email"`
		Telegram string `json:"telegram"`
	}

	tbl := []struct {
		name            string
		email, telegram string
		comments        int
		body            string // exact export, checked for the exports without comments
		details         *details
	}{
		{name: "email", email: "good@example.com", details: &details{Email: "good@example.com"},
			body: info + `, "details": {"email":"good@example.com"}, "comments":[]}`},
		{name: "telegram", telegram: "good_telegram", details: &details{Telegram: "good_telegram"},
			body: info + `, "details": {"telegram":"good_telegram"}, "comments":[]}`},
		{name: "email and telegram", email: "good@example.com", telegram: "good_telegram",
			details: &details{Email: "good@example.com", Telegram: "good_telegram"},
			body:    info + `, "details": {"email":"good@example.com","telegram":"good_telegram"}, "comments":[]}`},
		{name: "email with comments", email: "good@example.com", comments: 2, details: &details{Email: "good@example.com"}},
		{name: "nothing stored", body: info + `, "comments":[]}`},
		{name: "nothing stored with comments", comments: 2},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			ts, srv := startupT(t)

			if tt.email != "" {
				_, err := srv.DataService.SetUserEmail("remark42", "provider1_dev", tt.email)
				require.NoError(t, err)
			}
			if tt.telegram != "" {
				_, err := srv.DataService.SetUserTelegram("remark42", "provider1_dev", tt.telegram)
				require.NoError(t, err)
			}
			// details of another user never get into the export
			_, err := srv.DataService.SetUserEmail("remark42", "provider1_other", "other@example.com")
			require.NoError(t, err)
			c := store.Comment{User: store.User{ID: "provider1_dev", Name: "user name 1"}, Text: "test",
				Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
			for i := range tt.comments {
				c.ID = fmt.Sprintf("id-%03d", i)
				_, err = srv.DataService.Create(c)
				require.NoError(t, err)
			}

			code, body := getUserAllDataBody(t, ts.URL)
			require.Equal(t, http.StatusOK, code)
			if tt.body != "" {
				assert.Equal(t, tt.body, string(body))
			}
			assert.NotContains(t, string(body), "other@example.com")

			parsed := struct {
				Info     store.User      `json:"info"`
				Details  *details        `json:"details"`
				Comments []store.Comment `json:"comments"`
			}{}
			require.NoError(t, json.Unmarshal(body, &parsed), "export is valid JSON")
			assert.Equal(t, "provider1_dev", parsed.Info.ID)
			assert.Equal(t, tt.details, parsed.Details, "stored notification details are exported, the key is absent without them")
			assert.Len(t, parsed.Comments, tt.comments)
		})
	}
}

func TestRest_UserAllDataDetailsError(t *testing.T) {
	ts, srv := startupT(t)
	srv.privRest.dataService = failingDetailsStore{privStore: srv.privRest.dataService}

	code, body := getUserAllDataBody(t, ts.URL)
	assert.Equal(t, http.StatusInternalServerError, code)
	assert.JSONEq(t, `{"code":0,"details":"can't get user details","error":"can't read telegram: no telegram today"}`, string(body),
		"a failed read of the details is a JSON error, not an export without them")
}

// failingDetailsStore is a store unable to read the user's telegram
type failingDetailsStore struct{ privStore }

func (failingDetailsStore) GetUserTelegram(string, string) (string, error) {
	return "", fmt.Errorf("no telegram today")
}

// getUserAllDataBody requests /userdata as the dev user and returns the status with the body, ungzipped for an export
func getUserAllDataBody(t *testing.T, url string) (code int, body []byte) {
	t.Helper()
	client := &http.Client{Timeout: waitTimeout}
	defer client.CloseIdleConnections()
	req, err := http.NewRequest("GET", url+"/api/v1/userdata?site=remark42", http.NoBody)
	require.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Type") == "application/gzip" {
		ungzReader, e := gzip.NewReader(resp.Body)
		require.NoError(t, e)
		defer ungzReader.Close()
		reader = ungzReader
	}
	body, err = io.ReadAll(reader)
	require.NoError(t, err, "body is readable to the end")
	return resp.StatusCode, body
}

func TestRest_DeleteMe(t *testing.T) {
	ts, srv := startupT(t)

	m := callJSON[map[string]string](t, http.MethodPost, fmt.Sprintf("%s/api/v1/deleteme?site=remark42", ts.URL), "", devToken)
	assert.Equal(t, "remark42", m["site"])
	assert.Equal(t, "provider1_dev", m["user_id"])

	tkn := m["token"]
	claims, err := srv.Authenticator.TokenService().Parse(tkn)
	assert.NoError(t, err)
	assert.Equal(t, "provider1_dev", claims.User.ID)
	assert.Equal(t, "http://example.com/pic.png", claims.User.Picture,
		"delete_me token must carry the user's picture so the avatar can be removed when the request is processed")
	assert.Equal(t, "https://demo.remark42.com/web/deleteme.html?site_id=remark42&token="+tkn, m["link"])

	_, code := call(t, http.MethodPost, fmt.Sprintf("%s/api/v1/deleteme?site=remark42", ts.URL), "", "")
	assert.Equal(t, http.StatusUnauthorized, code)
}

func TestRest_SavePictureCtrl(t *testing.T) {
	ts, srv := startupT(t)

	// save picture
	savePic := func(name string) (id string) {
		bodyBuf, contentType := multipartFile(t, name, gopherPNG())
		client := http.Client{}
		defer client.CloseIdleConnections()
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/picture?site=remark42", ts.URL), bodyBuf)
		require.NoError(t, err)
		req.Header.Add("Content-Type", contentType)
		req.Header.Add("X-JWT", devToken)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		m := map[string]string{}
		err = json.Unmarshal(body, &m)
		assert.NoError(t, err)
		assert.Regexp(t, `^provider1_dev/[0-9a-f]{32}$`, m["id"], "the uploader's id and 128 random bits")
		return m["id"]
	}

	id := savePic("picture.png")
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/picture/%s", ts.URL, id))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 1462, len(body))
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))

	id = savePic("picture.gif")
	resp, err = http.Get(fmt.Sprintf("%s/api/v1/picture/%s", ts.URL, id))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))

	id = savePic("picture.jpg")
	resp, err = http.Get(fmt.Sprintf("%s/api/v1/picture/%s", ts.URL, id))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))

	id = savePic("picture.blah")
	resp, err = http.Get(fmt.Sprintf("%s/api/v1/picture/%s", ts.URL, id))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))

	resp, err = http.Get(fmt.Sprintf("%s/api/v1/picture/blah/pic.blah", ts.URL))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// a picture saved under the 20-character id of earlier versions is still served
	require.NoError(t, srv.ImageService.SaveWithID("provider1_dev/dc0ctt0j3vck07oc07s0", gopherPNG()))
	resp, err = http.Get(fmt.Sprintf("%s/api/v1/picture/provider1_dev/dc0ctt0j3vck07oc07s0", ts.URL))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestRest_SavePictureRemovesTempFiles uploads forms too big to be kept in memory, so their parts are written
// to temp files, and checks that none is left, whether the picture is rejected, accepted, or the form is
// stored and then reported as an error
func TestRest_SavePictureRemovesTempFiles(t *testing.T) {
	ts, _ := startupT(t)

	big := make([]byte, 6*1024*1024) // over the 5 MiB ParseMultipartForm keeps in memory
	upload := func(t *testing.T, query string, parts map[string][]byte) int {
		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)
		for name, data := range parts {
			fileWriter, err := bodyWriter.CreateFormFile(name, name+".png")
			require.NoError(t, err)
			_, err = fileWriter.Write(data)
			require.NoError(t, err)
		}
		require.NoError(t, bodyWriter.Close())
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/picture?site=remark42"+query, bodyBuf)
		require.NoError(t, err)
		req.Header.Add("Content-Type", bodyWriter.FormDataContentType())
		resp, err := sendReq(req, devToken)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		return resp.StatusCode
	}
	gopher, err := io.ReadAll(gopherPNG())
	require.NoError(t, err)

	tbl := []struct {
		name   string
		query  string
		parts  map[string][]byte
		status int
	}{
		{"picture too large", "", map[string][]byte{"file": big}, http.StatusBadRequest},
		{"picture with an extra large part", "", map[string][]byte{"file": gopher, "extra": big}, http.StatusOK},
		{"no picture part", "", map[string][]byte{"extra": big}, http.StatusInternalServerError},
		{"form stored, then a query error", "&x=%zz", map[string][]byte{"file": big}, http.StatusInternalServerError},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("TMPDIR", tmp) // os.CreateTemp, which writes the parts, reads it on each call
			assert.Equal(t, tt.status, upload(t, tt.query, tt.parts))
			files, err := os.ReadDir(tmp)
			require.NoError(t, err)
			names := make([]string, 0, len(files))
			for _, f := range files {
				names = append(names, f.Name())
			}
			assert.Empty(t, names, "temp files left by the upload")
		})
	}
}

func TestRest_CreateWithPictures(t *testing.T) {
	ts, svc := startupT(t)
	lgr.Setup(lgr.Debug, lgr.CallerFile, lgr.CallerFunc)

	images := filepath.Join(t.TempDir(), "images")
	imageService := image.NewService(&image.FileSystem{
		Staging:  images + ".staging",
		Location: images,
	}, image.ServiceParams{
		// the "not moved yet" checks below run right after the comment POST returns, so the
		// commit window has to be wide enough that a stalled runner cannot close it first
		EditDuration: 3 * time.Second,
		MaxSize:      2000,
		ImageAPI:     svc.RemarkURL + "/api/v1/picture/",
		ProxyAPI:     svc.RemarkURL + "/api/v1/img",
	})
	defer imageService.Close(context.Background())

	svc.privRest.imageService = imageService
	svc.ImageService = imageService

	dataService := svc.DataService
	dataService.ImageService = svc.ImageService
	svc.privRest.dataService = dataService

	uploadPicture := func(file string) (id string) {
		bodyBuf, contentType := multipartFile(t, file, gopherPNG())
		client := http.Client{}
		defer client.CloseIdleConnections()
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/picture?site=remark42", ts.URL), bodyBuf)
		require.NoError(t, err)
		req.Header.Add("Content-Type", contentType)
		req.Header.Add("X-JWT", devToken)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.NoError(t, resp.Body.Close())
		m := map[string]string{}
		err = json.Unmarshal(body, &m)
		assert.NoError(t, err)
		return m["id"]
	}

	var ids [3]string

	for i := range ids {
		ids[i] = uploadPicture(fmt.Sprintf("pic%d.png", i))
	}

	text := fmt.Sprintf(`text 123  ![](%s/api/v1/picture/%s) *xxx* ![](%s/api/v1/picture/%s) ![](%s/api/v1/picture/%s)`, svc.RemarkURL, ids[0], svc.RemarkURL, ids[1], svc.RemarkURL, ids[2])
	body := fmt.Sprintf(`{"text": %q, "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, text)

	b, code := call(t, http.MethodPost, ts.URL+"/api/v1/comment?site=remark42", body, adminPasswd)
	require.Equal(t, http.StatusCreated, code, b)

	for i := range ids {
		_, err := os.Stat(filepath.Join(images, ids[i]))
		assert.Error(t, err, "picture %d not moved from staging yet", i)
	}

	// the commit runs once EditDuration expires
	for i := range ids {
		require.Eventually(t, func() bool {
			_, e := os.Stat(filepath.Join(images, ids[i]))
			return e == nil
		}, waitTimeout, pollInterval, "picture %d moved from staging and available in permanent location", i)
	}
}

type mockTelegram struct {
	notVerified bool
	site        string
}

func (m *mockTelegram) AddToken(string, string, string, time.Time) {}

func (m *mockTelegram) GetBotUsername() string {
	return "botUsername"
}

func (m *mockTelegram) CheckToken(string, string) (telegram, site string, err error) {
	if m.notVerified {
		return "", "", fmt.Errorf("not verified")
	}
	return "good_telegram", m.site, nil
}

func TestExtractIP(t *testing.T) {
	tbl := []struct {
		addr string
		exp  string
	}{
		{"127.0.0.1:8080", "127.0.0.1"},
		{"127.0.0.1", "127.0.0.1"},
		{"192.168.1.1:443", "192.168.1.1"},
		{"[::1]:8080", "::1"},
		{"::1", "::1"},
		{"[2001:db8::1]:8080", "2001:db8::1"},
		{"2001:db8::1", "2001:db8::1"},
		{"[fe80::1%25eth0]:80", "fe80::1%25eth0"},
		{"", ""},
	}

	for _, tt := range tbl {
		t.Run(tt.addr, func(t *testing.T) {
			assert.Equal(t, tt.exp, extractIP(tt.addr))
		})
	}
}

// a token without a handshake, such as the login token, used as a confirmation token is rejected as invalid,
// not dereferenced
func TestRest_EmailTokensWithoutHandshake(t *testing.T) {
	ts, srv := startupT(t)

	otherUserTkn := handshakeToken(t, srv, "someone_else::good@example.com", emailSubscribeProvider)

	tbl := []struct {
		name, method, url, body string
		code                    int
	}{
		{name: "confirm with a login token", method: http.MethodPost, url: "/api/v1/email/confirm?site=remark42",
			body: fmt.Sprintf(`{"site":"remark42","token":%q}`, devToken), code: http.StatusBadRequest},
		{name: "confirm with another user's handshake", method: http.MethodPost, url: "/api/v1/email/confirm?site=remark42",
			body: fmt.Sprintf(`{"site":"remark42","token":%q}`, otherUserTkn), code: http.StatusBadRequest},
		{name: "unsubscribe with a login token", method: http.MethodGet, url: "/email/unsubscribe.html?site=remark42&tkn=" + devToken,
			code: http.StatusBadRequest},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			body, code := call(t, tt.method, ts.URL+tt.url, tt.body, devToken)
			assert.Equal(t, tt.code, code, body)
		})
	}
}

// subscribe and confirm act on the site matchSiteID verified in the query, not on a different site named in the body
func TestRest_EmailSubscribeUsesVerifiedSite(t *testing.T) {
	ts, srv := startupT(t, func(srv *Rest) {
		require.NoError(t, srv.DataService.Engine.Close())
		dir := t.TempDir()
		b, err := engine.NewBoltDB(bolt.Options{},
			engine.BoltSite{FileName: filepath.Join(dir, "remark42.db"), SiteID: "remark42"},
			engine.BoltSite{FileName: filepath.Join(dir, "other.db"), SiteID: "other"})
		require.NoError(t, err)
		srv.DataService.Engine = b
	})

	tkn := handshakeToken(t, srv, "provider1_dev::good@example.com", emailSubscribeProvider)

	send := func(url, body string) int {
		_, code := call(t, http.MethodPost, ts.URL+url, body, devToken)
		return code
	}

	// the session belongs to remark42; the body names the other site.
	// The address is already confirmed on the verified site, so subscribing to it again conflicts
	_, err := srv.DataService.SetUserEmail("remark42", "provider1_dev", "good@example.com")
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, send("/api/v1/email/subscribe?site=remark42", `{"site":"other","address":"good@example.com"}`),
		"subscribe checks the verified site")
	require.NoError(t, srv.DataService.DeleteUserDetail("remark42", "provider1_dev", engine.UserEmail))

	assert.Equal(t, http.StatusOK, send("/api/v1/email/confirm?site=remark42", fmt.Sprintf(`{"site":"other","token":%q}`, tkn)))
	email, err := srv.DataService.GetUserEmail("remark42", "provider1_dev")
	require.NoError(t, err)
	assert.Equal(t, "good@example.com", email, "email set on the verified site")
	email, err = srv.DataService.GetUserEmail("other", "provider1_dev")
	require.NoError(t, err)
	assert.Empty(t, email, "email not set on the site named only in the body")
}

// A request that arrives with an expired token in the cookie is renewed by the auth middleware. Setting or
// removing the email then writes the token a second time, to carry the new address, and used to write it
// with the expiry of the token the request came with: the response replaced the renewed token, in the
// cookie and in the X-JWT header, with one that had expired already.
func TestRest_EmailChangeKeepsTheRenewedToken(t *testing.T) {
	ts, srv := startupT(t)

	userToken := func(email string, expires time.Time) string {
		tkn, err := srv.Authenticator.TokenService().Token(token.Claims{
			ID: "xsrf-id", Audience: jwt.ClaimStrings{"remark42"},
			Issuer: "remark42", ExpiresAt: jwt.NewNumericDate(expires),
			User: &token.User{ID: "provider1_dev", Name: "developer one", Email: email},
		})
		require.NoError(t, err)
		return tkn
	}
	expired := func(email string) string { return userToken(email, time.Now().Add(-time.Minute)) }
	confirmation := handshakeToken(t, srv, "provider1_dev::new@example.com", emailSubscribeProvider)

	client := http.Client{}
	defer client.CloseIdleConnections()
	// the token the client is left with: of the JWT cookies in the answer, the last one set
	send := func(method, url, body, tkn string) (status int, left token.Claims) {
		var reqBody io.Reader = http.NoBody
		if body != "" {
			reqBody = strings.NewReader(body)
		}
		req, reqErr := http.NewRequest(method, ts.URL+url, reqBody)
		require.NoError(t, reqErr)
		req.AddCookie(&http.Cookie{Name: "JWT", Value: tkn})
		req.Header.Set("X-XSRF-TOKEN", "xsrf-id")
		resp, reqErr := client.Do(req)
		require.NoError(t, reqErr)
		require.NoError(t, resp.Body.Close())

		var last string
		for _, c := range resp.Cookies() {
			if c.Name == "JWT" {
				last = c.Value
			}
		}
		require.NotEmpty(t, last, "the answer set no JWT cookie")
		left, reqErr = srv.Authenticator.TokenService().Parse(last)
		require.NoError(t, reqErr)
		return resp.StatusCode, left
	}

	status, left := send(http.MethodPost, "/api/v1/email/confirm?site=remark42",
		fmt.Sprintf(`{"site":"remark42","token":%q}`, confirmation), expired(""))
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, left.User)
	assert.Equal(t, "new@example.com", left.User.Email, "the token carries the confirmed address")
	assert.False(t, srv.Authenticator.TokenService().IsExpired(left),
		"confirming the email left the client with a token that has expired already")

	status, left = send(http.MethodDelete, "/api/v1/email?site=remark42", "", expired("new@example.com"))
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, left.User)
	assert.Empty(t, left.User.Email, "the token no longer carries the address")
	assert.False(t, srv.Authenticator.TokenService().IsExpired(left),
		"removing the email left the client with a token that has expired already")

	// a token that is still valid is not renewed on the way: it keeps the expiry it came with. a day
	// ahead, so no stall of the test process can turn it into an expired one before it is sent
	expires := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	status, left = send(http.MethodDelete, "/api/v1/email?site=remark42", "", userToken("new@example.com", expires))
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, left.ExpiresAt)
	assert.True(t, expires.Equal(left.ExpiresAt.Time),
		"a valid token's expiry moved from %s to %s", expires, left.ExpiresAt.Time)
}

// the admin password sent along with a token that has no user, such as a confirmation token, passes
// authentication; setting and deleting the email then has no token user to update, which is not a panic
func TestRest_EmailWithAdminPasswordAndTokenWithoutUser(t *testing.T) {
	ts, srv := startupT(t)

	tkn := handshakeToken(t, srv, "admin::admin@example.com", emailSubscribeProvider)

	client := http.Client{}
	defer client.CloseIdleConnections()
	send := func(method, url, body string) (int, string) {
		var reqBody io.Reader = http.NoBody
		if body != "" {
			reqBody = strings.NewReader(body)
		}
		req, reqErr := http.NewRequest(method, ts.URL+url, reqBody)
		require.NoError(t, reqErr)
		req.SetBasicAuth("admin", "password")
		req.Header.Add("X-JWT", tkn)
		resp, reqErr := client.Do(req)
		require.NoError(t, reqErr)
		b, reqErr := io.ReadAll(resp.Body)
		require.NoError(t, reqErr)
		require.NoError(t, resp.Body.Close())
		return resp.StatusCode, string(b)
	}

	code, body := send(http.MethodPost, "/api/v1/email/confirm?site=remark42", fmt.Sprintf(`{"site":"remark42","token":%q}`, tkn))
	assert.Equal(t, http.StatusOK, code, body)
	email, err := srv.DataService.GetUserEmail("remark42", "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin@example.com", email)

	code, body = send(http.MethodDelete, "/api/v1/email?site=remark42", "")
	assert.Equal(t, http.StatusOK, code, body)
	email, err = srv.DataService.GetUserEmail("remark42", "admin")
	require.NoError(t, err)
	assert.Empty(t, email)
}

// a malformed handshake carries the address, which stays out of the error sent back and logged
func TestRest_EmailMalformedHandshakeHidesAddress(t *testing.T) {
	ts, srv := startupT(t)

	tkn := handshakeToken(t, srv, "provider1_dev::secret@example.com::extra", "")

	b, code := call(t, http.MethodPost, ts.URL+"/api/v1/email/confirm?site=remark42",
		fmt.Sprintf(`{"site":"remark42","token":%q}`, tkn), devToken)
	assert.Equal(t, http.StatusBadRequest, code, b)
	assert.NotContains(t, b, "secret@example.com", "confirm error echoes the address")

	body, code := get(t, ts.URL+"/email/unsubscribe.html?site=remark42&tkn="+tkn)
	assert.Equal(t, http.StatusBadRequest, code, body)
	assert.NotContains(t, body, "secret@example.com", "unsubscribe error echoes the address")
}

// a handshake with an empty user or address is rejected, not acted on with the empty part
func TestRest_EmailHandshakeWithEmptyPart(t *testing.T) {
	ts, srv := startupT(t)

	for _, hs := range []string{"provider1_dev::", "::good@example.com"} {
		t.Run(hs, func(t *testing.T) {
			tkn := handshakeToken(t, srv, hs, "")

			b, code := call(t, http.MethodPost, ts.URL+"/api/v1/email/confirm?site=remark42",
				fmt.Sprintf(`{"site":"remark42","token":%q}`, tkn), devToken)
			assert.Equal(t, http.StatusBadRequest, code, "confirm rejects the handshake: %s", b)

			body, code := get(t, ts.URL+"/email/unsubscribe.html?site=remark42&tkn="+tkn)
			assert.Equal(t, http.StatusBadRequest, code, "unsubscribe rejects the handshake: %s", body)
		})
	}
}

// subscribedUserToken stores an email for provider1_dev and returns the token of its unsubscribe link
func subscribedUserToken(t *testing.T, srv *Rest) string {
	t.Helper()
	_, err := srv.DataService.SetUserEmail("remark42", "provider1_dev", "good@example.com")
	require.NoError(t, err)
	return handshakeToken(t, srv, "provider1_dev::good@example.com", "")
}

// handshakeToken signs a token for site remark42 with the user::address handshake id. With the
// provider emailSubscribeProvider it is the confirmation token sendEmailConfirmationCtrl sends, with
// no provider the token of the unsubscribe link
func handshakeToken(t *testing.T, srv *Rest, id, provider string) string {
	t.Helper()
	claims := token.Claims{
		Handshake: &token.Handshake{ID: id},
		Audience:  jwt.ClaimStrings{"remark42"},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
		NotBefore: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
		Issuer:    "remark42",
	}
	if provider != "" {
		claims.AuthProvider = &token.AuthProvider{Name: provider}
	}
	tkn, err := srv.Authenticator.TokenService().Token(claims)
	require.NoError(t, err)
	return tkn
}

// a template on disk takes precedence over the embedded one; when it is broken, the html pages
// fail with a plain 500 instead of a half-rendered page
func TestRest_EmailUnsubscribeBrokenTemplate(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"error_response.html.tmpl", "email_unsubscribe.html.tmpl"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("{{ .Broken"), 0o600))
	}
	t.Chdir(dir)

	ts, srv := startupT(t)
	tkn := subscribedUserToken(t, srv)

	for name, url := range map[string]string{
		"error page":       "/email/unsubscribe.html?site=remark42",
		"unsubscribe page": "/email/unsubscribe.html?site=remark42&tkn=" + tkn,
	} {
		t.Run(name, func(t *testing.T) {
			body, code := get(t, ts.URL+url)
			assert.Equal(t, http.StatusInternalServerError, code)
			assert.Equal(t, "Internal Server Error\n", body)
		})
	}
	email, err := srv.DataService.GetUserEmail("remark42", "provider1_dev")
	require.NoError(t, err)
	assert.Empty(t, email, "the unsubscribe itself went through before the page failed")
}

// a template on disk may wrap its content in {{define "error"}} or {{define "unsubscribe"}}, the names
// the pages are executed under, and still renders
func TestRest_EmailUnsubscribeNamedTemplateOverride(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"error_response.html.tmpl":    `{{define "error"}}custom error page: {{.Error}}{{end}}`,
		"email_unsubscribe.html.tmpl": `{{define "unsubscribe"}}custom unsubscribe page{{end}}`,
	} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}
	t.Chdir(dir)

	ts, srv := startupT(t)
	tkn := subscribedUserToken(t, srv)

	for name, tc := range map[string]struct {
		url  string
		code int
		body string
	}{
		"error page":       {url: "/email/unsubscribe.html?site=remark42", code: http.StatusBadRequest, body: "custom error page: "},
		"unsubscribe page": {url: "/email/unsubscribe.html?site=remark42&tkn=" + tkn, code: http.StatusOK, body: "custom unsubscribe page"},
	} {
		t.Run(name, func(t *testing.T) {
			body, code := get(t, ts.URL+tc.url)
			assert.Equal(t, tc.code, code)
			assert.True(t, strings.HasPrefix(body, tc.body), "the override's defined template renders: %q", body)
		})
	}
}

// the link in the confirmation email subscribes with no session: GET only shows the button, POST from it
// stores the address. The token comes from the subscribe request itself, so the marker and the site it
// carries are the ones sendEmailConfirmationCtrl writes
func TestRest_EmailSubscribeLink(t *testing.T) {
	ts, srv := startupT(t)

	mockDestination := &fakeDestination{}
	srv.privRest.notifyService = notify.NewService(srv.DataService, 1, mockDestination)
	defer srv.privRest.notifyService.Close()

	client := http.Client{}
	defer client.CloseIdleConnections()
	do := func(method, url string, auth bool) (status int, body string, hdr http.Header) {
		req, err := http.NewRequest(method, ts.URL+url, http.NoBody)
		require.NoError(t, err)
		if auth {
			req.Header.Add("X-JWT", devToken)
		}
		resp, err := client.Do(req)
		require.NoError(t, err)
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		return resp.StatusCode, string(b), resp.Header
	}
	storedEmail := func() string {
		email, err := srv.DataService.GetUserEmail("remark42", "provider1_dev")
		require.NoError(t, err)
		return email
	}

	status, body, _ := do(http.MethodPost, "/api/v1/email/subscribe?site=remark42&address=good@example.com", true)
	require.Equal(t, http.StatusOK, status, body)
	waitForCount(t, 1, func() int { return len(mockDestination.GetVerify()) })
	link := "/email/subscribe.html?tkn=" + mockDestination.GetVerify()[0].Token

	status, body, hdr := do(http.MethodGet, link, false)
	require.Equal(t, http.StatusOK, status, body)
	assert.Contains(t, body, "Confirm subscription")
	assert.Contains(t, body, "<b>remark42</b>")
	assert.Contains(t, hdr.Get("Content-Security-Policy"), "form-action 'self'", "the global policy would block the form")
	assert.Contains(t, hdr.Get("Content-Security-Policy"), "frame-ancestors 'none'")
	assert.Equal(t, "no-referrer", hdr.Get("Referrer-Policy"))
	assert.Empty(t, storedEmail(), "GET must not subscribe, mail scanners fetch links")

	// the script that posts in place and closes the tab runs only under the nonce the policy names,
	// and the nonce is new on every response
	nonceRe := regexp.MustCompile(`script-src 'nonce-([A-Z2-7]{26})'`)
	m := nonceRe.FindStringSubmatch(hdr.Get("Content-Security-Policy"))
	require.Len(t, m, 2, hdr.Get("Content-Security-Policy"))
	assert.Contains(t, body, `<script nonce="`+m[1]+`">`)
	assert.Contains(t, hdr.Get("Content-Security-Policy"), "connect-src 'self'", "the script posts the page to itself")
	_, _, hdr2 := do(http.MethodGet, link, false)
	m2 := nonceRe.FindStringSubmatch(hdr2.Get("Content-Security-Policy"))
	require.Len(t, m2, 2)
	assert.NotEqual(t, m[1], m2[1])

	status, body, _ = do(http.MethodHead, link, false)
	assert.Equal(t, http.StatusOK, status, body)
	assert.Empty(t, storedEmail(), "HEAD must not subscribe")

	status, body, _ = do(http.MethodPost, link, false)
	require.Equal(t, http.StatusOK, status, body)
	assert.Contains(t, body, "Successfully subscribed")
	assert.Equal(t, "good@example.com", storedEmail())

	// a second click on the same link changes nothing and says the same
	status, body, _ = do(http.MethodPost, link, false)
	assert.Equal(t, http.StatusOK, status, body)
	assert.Contains(t, body, "Successfully subscribed")
	assert.Equal(t, "good@example.com", storedEmail())

	// a link for another address, here sent before or after the confirmed one, does not replace it: it
	// may have gone to a mistyped address whose owner would then get this user's notifications
	status, body, _ = do(http.MethodPost, "/api/v1/email/subscribe?site=remark42&address=typo@example.com", true)
	require.Equal(t, http.StatusOK, status, body)
	waitForCount(t, 2, func() int { return len(mockDestination.GetVerify()) })
	staleLink := "/email/subscribe.html?tkn=" + mockDestination.GetVerify()[1].Token
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		status, body, _ = do(method, staleLink, false)
		assert.Equal(t, http.StatusConflict, status, method)
		assert.Contains(t, body, "a different email address is already confirmed", method)
		// the widget shows only Unsubscribe once an address is stored, so that is the advice that works
		assert.Contains(t, body, "unsubscribe in the comment form and open this link again", method)
		assert.Equal(t, "good@example.com", storedEmail(), method)
	}

	// once unsubscribed, the user has no address to protect, and the link sets the new one
	require.NoError(t, srv.DataService.DeleteUserDetail("remark42", "provider1_dev", engine.UserEmail))
	status, body, _ = do(http.MethodPost, staleLink, false)
	require.Equal(t, http.StatusOK, status, body)
	assert.Equal(t, "typo@example.com", storedEmail())

	// the paste path still takes the marked token
	require.NoError(t, srv.DataService.DeleteUserDetail("remark42", "provider1_dev", engine.UserEmail))
	status, body, _ = do(http.MethodPost, "/api/v1/email/confirm?site=remark42&tkn="+mockDestination.GetVerify()[0].Token, true)
	assert.Equal(t, http.StatusOK, status, body)
	assert.Equal(t, "good@example.com", storedEmail())
}

// the comment form reads GET /email over and over while it waits for the confirmation link. At the
// default update limit, half a request a second per path, that read must not use up the reader's
// DELETE /email: the limiter keys on the path and not the method
func TestRest_GetEmailLeavesUnsubscribeAvailable(t *testing.T) {
	ts, _ := startupT(t, func(srv *Rest) { srv.UpdateLimiter = 0.5 })

	do := func(method string) int {
		_, code := call(t, method, ts.URL+"/api/v1/email?site=remark42", "", devToken)
		return code
	}

	for range 3 {
		assert.Equal(t, http.StatusOK, do(http.MethodGet))
	}
	assert.Equal(t, http.StatusOK, do(http.MethodDelete), "the reads used up the unsubscribe")
}

// the link takes only a token made for subscription confirmation: the unsubscribe and email login tokens
// carry the same user::address handshake, and with no session to compare the user with, accepting them
// would let anyone point another user's notifications at their own address
func TestRest_EmailSubscribeLinkRejects(t *testing.T) {
	ts, srv := startupT(t)

	mint := func(c token.Claims) string {
		if c.ExpiresAt == nil {
			c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(10 * time.Minute))
		}
		c.NotBefore = jwt.NewNumericDate(time.Now().Add(-1 * time.Minute))
		c.Issuer = "remark42"
		tkn, err := srv.Authenticator.TokenService().Token(c)
		require.NoError(t, err)
		return tkn
	}
	// the claims of a confirmation token, as sendEmailConfirmationCtrl makes it
	subscribeClaims := func(id string, aud ...string) token.Claims {
		return token.Claims{Handshake: &token.Handshake{ID: id}, AuthProvider: &token.AuthProvider{Name: emailSubscribeProvider}, Audience: aud}
	}
	expired := subscribeClaims("provider1_dev::bad@example.com", "remark42")
	expired.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-1 * time.Minute))

	tbl := []struct {
		name, tkn, details string
		code               int
	}{
		{name: "no token", code: http.StatusBadRequest, details: "token parameter is required"},
		{name: "not a token", tkn: "jwt", code: http.StatusForbidden, details: "failed to verify confirmation token"},
		{name: "expired", code: http.StatusForbidden, details: "the confirmation link has expired",
			tkn: mint(expired)},
		{name: "unsubscribe token", code: http.StatusBadRequest, details: "invalid handshake token",
			tkn: mint(token.Claims{Handshake: &token.Handshake{ID: "provider1_dev::bad@example.com"}, Audience: jwt.ClaimStrings{"remark42"},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(100 * 365 * 24 * time.Hour))})},
		{name: "email login token", code: http.StatusBadRequest, details: "invalid handshake token",
			tkn: mint(token.Claims{Handshake: &token.Handshake{ID: "provider1_dev::bad@example.com"}, Audience: jwt.ClaimStrings{"remark42"},
				SessionOnly: true, AuthProvider: &token.AuthProvider{Name: "email"}})},
		{name: "session token", tkn: devToken, code: http.StatusBadRequest, details: "invalid handshake token"},
		{name: "two audiences", code: http.StatusBadRequest, details: "invalid confirmation token",
			tkn: mint(subscribeClaims("provider1_dev::bad@example.com", "remark42", "other"))},
		{name: "malformed handshake", code: http.StatusBadRequest, details: "invalid handshake token",
			tkn: mint(subscribeClaims("provider1_dev", "remark42"))},
		{name: "blocked user", code: http.StatusForbidden, details: "user is blocked",
			tkn: mint(subscribeClaims("blocked_user::bad@example.com", "remark42"))},
	}
	require.NoError(t, srv.DataService.SetBlock("remark42", "blocked_user", true, 0))

	client := http.Client{}
	defer client.CloseIdleConnections()
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			for _, method := range []string{http.MethodGet, http.MethodPost} {
				req, err := http.NewRequest(method, ts.URL+"/email/subscribe.html?tkn="+tt.tkn, http.NoBody)
				require.NoError(t, err)
				resp, err := client.Do(req)
				require.NoError(t, err)
				b, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.NoError(t, resp.Body.Close())
				assert.Equal(t, tt.code, resp.StatusCode, "%s: %s", method, string(b))
				assert.Contains(t, string(b), tt.details, method)
				assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "frame-ancestors 'none'", method)
			}
			for _, user := range []string{"provider1_dev", "blocked_user"} {
				email, err := srv.DataService.GetUserEmail("remark42", user)
				require.NoError(t, err)
				assert.Empty(t, email, "nothing stored for %s", user)
			}
		})
	}
}

// the token of the unsubscribe link is mailed with every reply and lasts a century, so it does not confirm
// the address it names, while the confirmation token sent on subscribe does, and does not unsubscribe
func TestRest_EmailConfirmWithUnsubscribeToken(t *testing.T) {
	ts, srv := startupT(t)

	mockDestination := &fakeDestination{}
	srv.privRest.notifyService = notify.NewService(srv.DataService, 1, mockDestination)
	defer srv.privRest.notifyService.Close()

	confirm := func(tkn string) (int, string) {
		body, code := call(t, http.MethodPost, ts.URL+"/api/v1/email/confirm?site=remark42",
			fmt.Sprintf(`{"site":"remark42","token":%q}`, tkn), devToken)
		return code, body
	}
	userEmail := func() string {
		email, err := srv.DataService.GetUserEmail("remark42", "provider1_dev")
		require.NoError(t, err)
		return email
	}

	unsubscribeTkn := subscribedUserToken(t, srv)
	body, code := get(t, ts.URL+"/email/unsubscribe.html?site=remark42&tkn="+unsubscribeTkn)
	require.Equal(t, http.StatusOK, code, body)

	code, body = confirm(unsubscribeTkn)
	assert.Equal(t, http.StatusBadRequest, code, body)
	assert.Empty(t, userEmail(), "the unsubscribe token subscribed the address back")

	_, code = call(t, http.MethodPost, ts.URL+"/api/v1/email/subscribe?site=remark42",
		`{"site":"remark42","address":"good@example.com"}`, devToken)
	require.Equal(t, http.StatusOK, code)
	waitForCount(t, 1, func() int { return len(mockDestination.GetVerify()) })
	confirmTkn := mockDestination.GetVerify()[0].Token

	code, body = confirm(confirmTkn)
	assert.Equal(t, http.StatusOK, code, body)
	assert.Equal(t, "good@example.com", userEmail())

	body, code = get(t, ts.URL+"/email/unsubscribe.html?site=remark42&tkn="+confirmTkn)
	assert.Equal(t, http.StatusBadRequest, code, body)
	assert.Equal(t, "good@example.com", userEmail(), "the confirmation token unsubscribed the address")
}

// the unsubscribe page gives one answer to a link that does not match the stored address, whether the user
// has no address or another one, and takes no email login token, whose user part the requester chooses
func TestRest_EmailUnsubscribeTellsNothingAboutAnotherUser(t *testing.T) {
	ts, srv := startupT(t)

	mint := func(c token.Claims) string {
		c.Audience = jwt.ClaimStrings{"remark42"}
		c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(10 * time.Minute))
		c.NotBefore = jwt.NewNumericDate(time.Now().Add(-1 * time.Minute))
		c.Issuer = "remark42"
		tkn, err := srv.Authenticator.TokenService().Token(c)
		require.NoError(t, err)
		return tkn
	}
	// a link naming provider1_dev with an address that is not theirs, and an email login token asked for with
	// provider1_dev as the user name and the requester's own address
	link := mint(token.Claims{Handshake: &token.Handshake{ID: "provider1_dev::other@example.com"}})
	login := mint(token.Claims{Handshake: &token.Handshake{ID: "provider1_dev::other@example.com"},
		AuthProvider: &token.AuthProvider{Name: "email"}})

	answers := func() (linkBody, loginBody string) {
		body, code := get(t, ts.URL+"/email/unsubscribe.html?site=remark42&tkn="+link)
		assert.Equal(t, http.StatusConflict, code, body)
		linkBody = body
		body, code = get(t, ts.URL+"/email/unsubscribe.html?site=remark42&tkn="+login)
		assert.Equal(t, http.StatusBadRequest, code, body)
		return linkBody, body
	}

	notSubscribedLink, notSubscribedLogin := answers()
	_, err := srv.DataService.SetUserEmail("remark42", "provider1_dev", "good@example.com")
	require.NoError(t, err)
	subscribedLink, subscribedLogin := answers()
	assert.Equal(t, notSubscribedLink, subscribedLink, "the link tells whether the user is subscribed")
	assert.Equal(t, notSubscribedLogin, subscribedLogin, "the login token tells whether the user is subscribed")

	email, err := srv.DataService.GetUserEmail("remark42", "provider1_dev")
	require.NoError(t, err)
	assert.Equal(t, "good@example.com", email, "the address of the user is kept")
}

// confirmation and unsubscribe tokens are issued for one site, kept in the token's audience, and are refused
// for another site: the unsubscribe page needs no session, so nothing else ties the link to its site, and a
// user has the same ID on every site of a server
func TestRest_EmailTokenOfAnotherSite(t *testing.T) {
	ts, srv := startupT(t, func(srv *Rest) {
		require.NoError(t, srv.DataService.Engine.Close())
		dir := t.TempDir()
		b, err := engine.NewBoltDB(bolt.Options{},
			engine.BoltSite{FileName: filepath.Join(dir, "remark42.db"), SiteID: "remark42"},
			engine.BoltSite{FileName: filepath.Join(dir, "other.db"), SiteID: "other"})
		require.NoError(t, err)
		srv.DataService.Engine = b
	})

	siteToken := func(provider *token.AuthProvider, site ...string) string {
		tkn, err := srv.Authenticator.TokenService().Token(token.Claims{
			Handshake:    &token.Handshake{ID: "provider1_dev::good@example.com"},
			AuthProvider: provider,
			Audience:     site,
			ExpiresAt:    jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			NotBefore:    jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
			Issuer:       "remark42",
		})
		require.NoError(t, err)
		return tkn
	}
	userEmail := func(site string) string {
		email, err := srv.DataService.GetUserEmail(site, "provider1_dev")
		require.NoError(t, err)
		return email
	}
	confirm := func(tkn string) int {
		_, code := call(t, http.MethodPost, ts.URL+"/api/v1/email/confirm?site=remark42",
			fmt.Sprintf(`{"site":"remark42","token":%q}`, tkn), devToken)
		return code
	}

	// the session belongs to remark42, the confirmation token was issued on the other site or on both
	subscribe := &token.AuthProvider{Name: emailSubscribeProvider}
	assert.Equal(t, http.StatusForbidden, confirm(siteToken(subscribe, "other")), "confirm with the token of another site")
	assert.Equal(t, http.StatusForbidden, confirm(siteToken(subscribe, "remark42", "other")), "confirm with a token of two sites")
	assert.Empty(t, userEmail("remark42"), "email set by a token of another site")

	// the user is subscribed on both sites, the unsubscribe link comes from a mail of remark42
	for _, site := range []string{"remark42", "other"} {
		_, err := srv.DataService.SetUserEmail(site, "provider1_dev", "good@example.com")
		require.NoError(t, err)
	}
	tkn := siteToken(nil, "remark42")
	for _, site := range []string{"other", ""} {
		body, code := get(t, ts.URL+"/email/unsubscribe.html?site="+site+"&tkn="+tkn)
		assert.Equal(t, http.StatusForbidden, code, "unsubscribe on site %q with the link of remark42: %s", site, body)
	}
	assert.Equal(t, "good@example.com", userEmail("other"), "subscription on the other site is kept")
	assert.Equal(t, "good@example.com", userEmail("remark42"), "subscription is kept")

	body, code := get(t, ts.URL+"/email/unsubscribe.html?site=remark42&tkn="+tkn)
	assert.Equal(t, http.StatusOK, code, body)
	assert.Empty(t, userEmail("remark42"), "unsubscribed on the site of the link")
	assert.Equal(t, "good@example.com", userEmail("other"), "subscription on the other site is kept")
}

// the basic-auth admin may name the site in the body only; the confirmation token is issued for that site,
// and the confirmation for the same site accepts it
func TestRest_EmailConfirmationTokenForSiteInBody(t *testing.T) {
	ts, srv := startupT(t)

	mockDestination := &fakeDestination{}
	srv.privRest.notifyService = notify.NewService(srv.DataService, 1, mockDestination)
	defer srv.privRest.notifyService.Close()

	client := http.Client{}
	defer client.CloseIdleConnections()
	send := func(url, body, tkn string) (int, string) {
		req, err := http.NewRequest(http.MethodPost, ts.URL+url, strings.NewReader(body))
		require.NoError(t, err)
		req.SetBasicAuth("admin", "password")
		if tkn != "" {
			req.Header.Add("X-JWT", tkn) // setEmail reads the token of the request to update it
		}
		resp, err := client.Do(req)
		require.NoError(t, err)
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		return resp.StatusCode, string(b)
	}

	code, body := send("/api/v1/email/subscribe", `{"site":"remark42","address":"admin@example.com"}`, "")
	require.Equal(t, http.StatusOK, code, body)
	waitForCount(t, 1, func() int { return len(mockDestination.GetVerify()) })
	tkn := mockDestination.GetVerify()[0].Token

	code, body = send("/api/v1/email/confirm", fmt.Sprintf(`{"site":"remark42","token":%q}`, tkn), tkn)
	assert.Equal(t, http.StatusOK, code, body)
	email, err := srv.DataService.GetUserEmail("remark42", "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin@example.com", email)
}

// the admin password authenticates a request with no session token, and the address must be stored, and removed,
// with a 200 for both, as there is no token to update
func TestRest_EmailWithAdminPasswordAndNoSessionToken(t *testing.T) {
	ts, srv := startupT(t)

	mockDestination := &fakeDestination{}
	srv.privRest.notifyService = notify.NewService(srv.DataService, 1, mockDestination)
	defer srv.privRest.notifyService.Close()

	client := http.Client{}
	defer client.CloseIdleConnections()
	send := func(method, url, body string) (int, string) {
		req, err := http.NewRequest(method, ts.URL+url, strings.NewReader(body))
		require.NoError(t, err)
		req.SetBasicAuth("admin", "password")
		resp, err := client.Do(req)
		require.NoError(t, err)
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		return resp.StatusCode, string(b)
	}

	code, body := send(http.MethodPost, "/api/v1/email/subscribe", `{"site":"remark42","address":"admin@example.com"}`)
	require.Equal(t, http.StatusOK, code, body)
	waitForCount(t, 1, func() int { return len(mockDestination.GetVerify()) })
	tkn := mockDestination.GetVerify()[0].Token

	code, body = send(http.MethodPost, "/api/v1/email/confirm", fmt.Sprintf(`{"site":"remark42","token":%q}`, tkn))
	assert.Equal(t, http.StatusOK, code, body)
	assert.JSONEq(t, `{"updated":true,"address":"admin@example.com"}`, body)
	email, err := srv.DataService.GetUserEmail("remark42", "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin@example.com", email)

	code, body = send(http.MethodDelete, "/api/v1/email?site=remark42", "")
	assert.Equal(t, http.StatusOK, code, body)
	assert.JSONEq(t, `{"deleted":true}`, body)
	email, err = srv.DataService.GetUserEmail("remark42", "admin")
	require.NoError(t, err)
	assert.Empty(t, email)
}
