package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/go-pkgz/auth/v2"
	"github.com/go-pkgz/auth/v2/avatar"
	"github.com/go-pkgz/auth/v2/provider"
	"github.com/go-pkgz/auth/v2/token"
	cache "github.com/go-pkgz/lcw/v2"
	R "github.com/go-pkgz/rest"
	"github.com/go-pkgz/routegroup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/goleak"

	"github.com/umputun/remark42/backend/app/migrator"
	"github.com/umputun/remark42/backend/app/notify"
	"github.com/umputun/remark42/backend/app/rest"
	"github.com/umputun/remark42/backend/app/rest/proxy"
	"github.com/umputun/remark42/backend/app/store"
	adminstore "github.com/umputun/remark42/backend/app/store/admin"
	"github.com/umputun/remark42/backend/app/store/engine"
	"github.com/umputun/remark42/backend/app/store/image"
	"github.com/umputun/remark42/backend/app/store/service"
	"github.com/umputun/remark42/backend/app/webassets"
)

// To generate a token, enter one of the tokens here into https://jwt.io, change the secret to one you're using in your test
// ("secret" in case of startupT), and alter the fields you want to be changed.

var devToken = `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJyZW1hcms0MiIsImV4cCI6Mzc4OTE5MTgyMiwianRpIjoicmFuZG9tIGlkIiwiaXNzIjoicmVtYXJrNDIiLCJuYmYiOjE1MjE4ODQyMjIsInVzZXIiOnsibmFtZSI6ImRldmVsb3BlciBvbmUiLCJpZCI6InByb3ZpZGVyMV9kZXYiLCJwaWN0dXJlIjoiaHR0cDovL2V4YW1wbGUuY29tL3BpYy5wbmciLCJpcCI6IjEyNy4wLjAuMSIsImVtYWlsIjoibWVAZXhhbXBsZS5jb20ifX0.dirTS_ahSF6375sdO2iodm2K2UmRTzQNQMFiHuTQCVs`

var dev2Token = `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJyZW1hcms0MiIsImV4cCI6Mzc4OTE5MTgyMiwianRpIjoicmFuZG9tIGlkIiwiaXNzIjoicmVtYXJrNDIiLCJuYmYiOjE1MjE4ODQyMjIsInVzZXIiOnsibmFtZSI6ImRldmVsb3BlciBvbmUiLCJpZCI6InByb3ZpZGVyMV9kZXYyIiwicGljdHVyZSI6Imh0dHA6Ly9leGFtcGxlLmNvbS9waWMucG5nIiwiaXAiOiIxMjcuMC4wLjEiLCJlbWFpbCI6Im1lQGV4YW1wbGUuY29tIn19.qsR_PupfjIq7uw0eAuyGV8nsUoMx9v541c9olnRInRQ`

var anonToken = `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJyZW1hcms0MiIsImV4cCI6Mzc4OTE5MTgyMiwianRpIjoicmFuZG9tIGlkIiwiaXNzIjoicmVtYXJrNDIiLCJuYmYiOjE1MjE4ODQyMjIsInVzZXIiOnsibmFtZSI6ImFub255bW91cyB0ZXN0IHVzZXIiLCJpZCI6ImFub255bW91c190ZXN0X3VzZXIiLCJwaWN0dXJlIjoiaHR0cDovL2V4YW1wbGUuY29tL3BpYy5wbmciLCJpcCI6IjEyNy4wLjAuMSIsImVtYWlsIjoiYW5vbkBleGFtcGxlLmNvbSJ9fQ.gAae2WMxZNZE5ebVboptPEyQ7Nk6EQxciNnGJ_mPOuU`

var emailUserToken = `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJyZW1hcms0MiIsImV4cCI6Mzc4OTE5MTgyMiwianRpIjoicmFuZG9tIGlkIiwiaXNzIjoicmVtYXJrNDIiLCJuYmYiOjE1MjE4ODQyMjIsInVzZXIiOnsibmFtZSI6Imdvb2RAZXhhbXBsZS5jb20gdGVzdCB1c2VyIiwiaWQiOiJlbWFpbF9mNWRmZTlkMmU2YmQ3NWZjNzRlYTVmYWJmMjczYjQ1YjViYWViMTk1IiwicGljdHVyZSI6Imh0dHA6Ly9leGFtcGxlLmNvbS9waWMucG5nIiwiaXAiOiIxMjcuMC4wLjEiLCJlbWFpbCI6Imdvb2RAZXhhbXBsZS5jb20ifX0.vH2HN1JpuXL8okTJq1A-zGHQ-l2ILcwxvDDEmu2zwks`

var devTokenBadAud = `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJyZW1hcms0Ml9iYWQiLCJleHAiOjM3ODkxOTE4MjIsImp0aSI6InJhbmRvbSBpZCIsImlzcyI6InJlbWFyazQyIiwibmJmIjoxNTIxODg0MjIyLCJ1c2VyIjp7Im5hbWUiOiJkZXZlbG9wZXIgb25lIiwiaWQiOiJwcm92aWRlcjFfZGV2IiwicGljdHVyZSI6Imh0dHA6Ly9leGFtcGxlLmNvbS9waWMucG5nIiwiaXAiOiIxMjcuMC4wLjEiLCJlbWFpbCI6Im1lQGV4YW1wbGUuY29tIn19.X-lvnHvBz6VfEbVV4f-bjcZuLY5pYtvEansk_TQMrX8`

var adminUmputunToken = `eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOiJyZW1hcms0MiIsImV4cCI6MTk1NDU5Nzk4MCwianRpIjoiOTdhMmUwYWM0ZGM3ZDVmNjkyNmQ1ZTg2MjBhY2VmOWE0MGMwIiwiaWF0IjoxNDU0NTk3NjgwLCJpc3MiOiJyZW1hcms0MiIsInVzZXIiOnsibmFtZSI6IlVtcHV0dW4iLCJpZCI6ImdpdGh1Yl9lZjBmNzA2YTciLCJwaWN0dXJlIjoiaHR0cHM6Ly9yZW1hcms0Mi5yYWRpby10LmNvbS9hcGkvdjEvYXZhdGFyL2NiNDJmZjQ5M2FkZTY5NmQ4OGEzYTU5MGYxMzZhZTllMzRkZTdjMWIuaW1hZ2UiLCJhdHRycyI6eyJhZG1pbiI6dHJ1ZSwiYmxvY2tlZCI6ZmFsc2V9fX0.dZiOjWHguo9f42XCMooMcv4EmYFzifl_-LEvPZHCtks`

func TestRest_FileServer(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	testHTMLName := "test-remark.html"
	testHTMLFile := os.TempDir() + "/" + testHTMLName
	err := os.WriteFile(testHTMLFile, []byte("some html"), 0o700)
	assert.NoError(t, err)

	body, code := get(t, ts.URL+"/web/"+testHTMLName)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "some html", body)
	_ = os.Remove(testHTMLFile)
}

// TestRest_FileServerStaticAssets covers the static file server behaviors that are
// sensitive to the router: the bare /web -> /web/ redirect, cache headers applied to
// served assets, 404 for missing files, and the directory-listing block.
func TestRest_FileServerStaticAssets(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	require.NoError(t, os.WriteFile(srv.WebRoot+"/asset-test.html", []byte("static body"), 0o600))
	require.NoError(t, os.MkdirAll(srv.WebRoot+"/subdir-test", 0o700))
	defer func() {
		_ = os.Remove(srv.WebRoot + "/asset-test.html")
		_ = os.RemoveAll(srv.WebRoot + "/subdir-test")
	}()

	noRedirect := http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer noRedirect.CloseIdleConnections()

	t.Run("bare /web redirects to /web/", func(t *testing.T) {
		resp, err := noRedirect.Get(ts.URL + "/web")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusMovedPermanently, resp.StatusCode)
		assert.Equal(t, "/web/", resp.Header.Get("Location"))
	})

	t.Run("serves an existing asset with cache headers", func(t *testing.T) {
		resp, err := noRedirect.Get(ts.URL + "/web/asset-test.html")
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "static body", string(body))
		assert.NotEmpty(t, resp.Header.Get("Etag"), "cacheControl must set an Etag on served assets")
		assert.Contains(t, resp.Header.Get("Cache-Control"), "max-age", "cacheControl must set max-age on served assets")
	})

	t.Run("missing asset returns 404", func(t *testing.T) {
		_, code := get(t, ts.URL+"/web/does-not-exist.html")
		assert.Equal(t, http.StatusNotFound, code)
	})

	t.Run("directory listing is blocked", func(t *testing.T) {
		resp, err := noRedirect.Get(ts.URL + "/web/subdir-test/")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "directory listings must be blocked")
	})
}

// TestRest_FileServerBackendAssets covers the assets embedded in the binary and the rule that a
// name the frontend build provides is served from there instead. WebRoot is a fresh empty
// directory so the frontend side is known, rather than the shared temp dir startupT defaults to.
func TestRest_FileServerBackendAssets(t *testing.T) {
	ts, srv, teardown := startupT(t, func(srv *Rest) { srv.WebRoot = t.TempDir() })
	defer teardown()

	t.Run("serves every embedded asset byte for byte", func(t *testing.T) {
		for _, name := range []string{"privacy.html", "markdown-help.html", "400x400.jpeg"} {
			t.Run(name, func(t *testing.T) {
				want, err := fs.ReadFile(webassets.FS, name)
				require.NoError(t, err)

				body, code := get(t, ts.URL+"/web/"+name)
				assert.Equal(t, http.StatusOK, code)
				assert.Equal(t, string(want), body, "the bytes must come from the embedded assets")
			})
		}
	})

	t.Run("serves the image with its own content type", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/web/400x400.jpeg")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "image/jpeg", resp.Header.Get("Content-Type"))
	})

	t.Run("head is served", func(t *testing.T) {
		resp, err := http.Head(ts.URL + "/web/privacy.html")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("frontend output wins over the embedded copy", func(t *testing.T) {
		require.NoError(t, os.WriteFile(srv.WebRoot+"/privacy.html", []byte("operator's own policy"), 0o600))
		t.Cleanup(func() { _ = os.Remove(srv.WebRoot + "/privacy.html") })

		body, code := get(t, ts.URL+"/web/privacy.html")
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, "operator's own policy", body)
	})

	t.Run("traversal out of the asset root is refused", func(t *testing.T) {
		for _, p := range []string{"/web/../../etc/passwd", "/web/..%2f..%2fetc%2fpasswd", "/web/%2e%2e/%2e%2e/etc/passwd"} {
			t.Run(p, func(t *testing.T) {
				body, code := get(t, ts.URL+p)
				assert.NotContains(t, body, "root:", "must never serve a file outside the served roots")
				assert.NotEqual(t, http.StatusInternalServerError, code, "a rejected name must not surface as 500")
			})
		}
	})

	t.Run("missing in both still returns 404", func(t *testing.T) {
		_, code := get(t, ts.URL+"/web/neither-source-has-this.html")
		assert.Equal(t, http.StatusNotFound, code)
	})
}

// TestRest_FileServerEmbeddedFrontend covers the branch taken when no web root exists on disk,
// which is how the released binary runs. The frontend stands in for the copy embedded at
// app/cmd/web, so a name it provides and a name only the assets provide are both exercised.
func TestRest_FileServerEmbeddedFrontend(t *testing.T) {
	frontend := fstest.MapFS{"index.html": {Data: []byte("embedded frontend index")}}
	router := routegroup.New(http.NewServeMux())
	addFileServer(router, frontend, filepath.Join(t.TempDir(), "absent"), "test-version")

	ts := httptest.NewServer(router)
	defer ts.Close()

	t.Run("serves the embedded frontend", func(t *testing.T) {
		body, code := get(t, ts.URL+"/web/index.html")
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, "embedded frontend index", body)
	})

	for _, name := range []string{"privacy.html", "markdown-help.html", "400x400.jpeg"} {
		t.Run("falls back to "+name, func(t *testing.T) {
			want, err := fs.ReadFile(webassets.FS, name)
			require.NoError(t, err)

			body, code := get(t, ts.URL+"/web/"+name)
			assert.Equal(t, http.StatusOK, code)
			assert.Equal(t, string(want), body)
		})
	}

	t.Run("a name neither source has is missing", func(t *testing.T) {
		_, code := get(t, ts.URL+"/web/nothing-here.html")
		assert.Equal(t, http.StatusNotFound, code)
	})

	t.Run("a name the operating system rejects is missing, not an error", func(t *testing.T) {
		_, code := get(t, ts.URL+"/web/a%00b.html")
		assert.Equal(t, http.StatusNotFound, code)
	})
}

// TestRest_FileServerRoutesEmbedded drives the whole router the released binary runs: no web root
// on disk, and the frontend read from WebFS. It is what pins the web/ prefix routes() strips, which
// a test calling addFileServer directly cannot see.
func TestRest_FileServerRoutesEmbedded(t *testing.T) {
	frontend := fstest.MapFS{
		"web/index.html":  {Data: []byte("embedded index")},
		"web/iframe.html": {Data: []byte("embedded iframe")},
		"web/remark.mjs":  {Data: []byte("embedded bundle")},
	}
	ts, _, teardown := startupT(t, func(srv *Rest) {
		srv.WebRoot = filepath.Join(t.TempDir(), "absent")
		srv.WebFS = frontend
	})
	defer teardown()

	t.Run("serves the frontend from under the web prefix", func(t *testing.T) {
		for name, want := range map[string]string{
			"index.html":  "embedded index",
			"iframe.html": "embedded iframe",
			"remark.mjs":  "embedded bundle",
		} {
			t.Run(name, func(t *testing.T) {
				body, code := get(t, ts.URL+"/web/"+name)
				assert.Equal(t, http.StatusOK, code)
				assert.Equal(t, want, body)
			})
		}
	})

	t.Run("the prefix is stripped rather than exposed", func(t *testing.T) {
		_, code := get(t, ts.URL+"/web/web/index.html")
		assert.Equal(t, http.StatusNotFound, code, "the web/ prefix must not be reachable as a path")
	})

	t.Run("the embedded assets still answer alongside it", func(t *testing.T) {
		want, err := fs.ReadFile(webassets.FS, "privacy.html")
		require.NoError(t, err)

		body, code := get(t, ts.URL+"/web/privacy.html")
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, string(want), body)
	})
}

// refusingSubFS is an fs.FS whose Sub refuses, which is the only way fs.Sub returns a nil
// filesystem. routes() has to survive it, since a nil frontend would panic on the first request.
type refusingSubFS struct{}

func (refusingSubFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}
func (refusingSubFS) Sub(string) (fs.FS, error) { return nil, errors.New("refused") }

// TestRest_FileServerFrontendSourceRefused covers the branch where the frontend source cannot be
// sub-rooted: /web must keep serving the embedded assets rather than panicking.
func TestRest_FileServerFrontendSourceRefused(t *testing.T) {
	ts, _, teardown := startupT(t, func(srv *Rest) {
		srv.WebRoot = filepath.Join(t.TempDir(), "absent")
		srv.WebFS = refusingSubFS{}
	})
	defer teardown()

	t.Run("the embedded assets still serve", func(t *testing.T) {
		want, err := fs.ReadFile(webassets.FS, "privacy.html")
		require.NoError(t, err)

		body, code := get(t, ts.URL+"/web/privacy.html")
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, string(want), body)
	})

	t.Run("a frontend name is missing rather than fatal", func(t *testing.T) {
		_, code := get(t, ts.URL+"/web/iframe.html")
		assert.Equal(t, http.StatusNotFound, code)
	})
}

// TestRest_RejectHeadOnDestructiveGET verifies that HEAD is blocked on the state-mutating
// GET routes (which stdlib http.ServeMux would otherwise route to the GET handler) while
// still being served for safe, read-only routes.
func TestRest_RejectHeadOnDestructiveGET(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	client := http.Client{}
	defer client.CloseIdleConnections()

	t.Run("HEAD is rejected on a destructive GET route", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodHead, ts.URL+"/api/v1/admin/deleteme?site=remark42", http.NoBody)
		require.NoError(t, err)
		req.SetBasicAuth("admin", "password")
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode, "HEAD must not reach a state-mutating GET handler")
		assert.Equal(t, "GET", resp.Header.Get("Allow"), "405 must carry an Allow header")
	})

	t.Run("HEAD is rejected on the email unsubscribe route", func(t *testing.T) {
		// emailUnsubscribeCtrl deletes the user's email subscription on GET, so HEAD (which
		// ServeMux would route to the GET handler) must be rejected before it runs
		resp, err := client.Head(ts.URL + "/email/unsubscribe.html?site=remark42")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode, "HEAD must not reach the email-unsubscribe handler")
		assert.Equal(t, "GET, POST", resp.Header.Get("Allow"), "Allow must list every method the resource supports")
	})

	t.Run("HEAD still works on a safe read-only route", func(t *testing.T) {
		resp, err := client.Head(ts.URL + "/api/v1/config?site=remark42")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode, "HEAD must still be served for safe read-only routes")
	})

	t.Run("wrong method on a known route returns 405 with Allow", func(t *testing.T) {
		// method-in-pattern is new under ServeMux; a wrong method on a known route must
		// still yield 405 with the allowed methods advertised
		resp, err := client.Post(ts.URL+"/api/v1/config?site=remark42", "application/json", http.NoBody)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Allow"), "GET", "405 must advertise the allowed methods")
	})
}

// TestRest_AvatarMounts verifies both avatar mounts (root /avatar/ and /api/v1/avatar/)
// still route to the avatar handler after the chi Mount -> ServeMux Handle rewiring,
// rather than falling through to a router 404.
func TestRest_AvatarMounts(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	for _, path := range []string{"/api/v1/avatar/nonexistent.image", "/avatar/nonexistent.image"} {
		t.Run(path, func(t *testing.T) {
			body, code := get(t, ts.URL+path)
			// the avatar handler responds (403 "can't load avatar"), not a router 404
			assert.Equal(t, http.StatusForbidden, code, "avatar mount must reach the avatar handler")
			assert.Contains(t, body, "can't load avatar", "request must reach the avatar handler, not a routing 404")
		})
	}
}

func TestRest_Shutdown(t *testing.T) {
	srv := Rest{Authenticator: &auth.Service{}, ImageProxy: &proxy.Image{}}
	port := chooseUnusedPort(t)
	done := make(chan bool)

	// without waiting for channel close at the end goroutine will stay alive after test finish
	// which would create data race with next test
	go func() {
		srv.Run("127.0.0.1", port)
		close(done)
	}()

	defer srv.Shutdown() // a failed readiness wait must not leave srv.Run behind for goleak
	waitForServerStart(t, port)
	srv.Shutdown()

	select {
	case <-done:
	case <-time.After(serverStopTimeout):
		t.Fatal("rest server did not stop after Shutdown")
	}
}

func TestRest_filterComments(t *testing.T) {
	user := store.User{ID: "user1", Name: "user name 1"}
	c1 := store.Comment{User: user, Text: "test test #1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 5, 27, 1, 14, 10, 0, time.Local)}
	c2 := store.Comment{User: user, Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 5, 27, 1, 14, 20, 0, time.Local)}
	c3 := store.Comment{User: user, Text: "test test #3", ParentID: "p1", Locator: store.Locator{SiteID: "radio-t",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 5, 27, 1, 14, 25, 0, time.Local)}

	r := filterComments([]store.Comment{c1, c2, c3}, func(c store.Comment) bool {
		return c.Text == "test test #1" || c.Text == "test test #3"
	})
	assert.Equal(t, 2, len(r), "one comment filtered")
}

func TestRest_RunStaticSSLMode(t *testing.T) {
	sslPort := chooseUnusedPort(t)
	srv := Rest{
		Authenticator: auth.NewService(auth.Opts{
			AvatarStore:       avatar.NewLocalFS("/tmp"),
			AvatarResizeLimit: 300,
		}),

		ImageProxy: &proxy.Image{},
		SSLConfig: SSLConfig{
			SSLMode: Static,
			Port:    sslPort,
			Key:     "../../cmd/testdata/key.pem",
			Cert:    "../../cmd/testdata/cert.pem",
		},
		RemarkURL: fmt.Sprintf("https://localhost:%d", sslPort),
	}

	port := chooseUnusedPort(t)
	go func() {
		srv.Run("", port)
	}()

	waitForServerStart(t, sslPort, port)

	client := http.Client{
		// prevent http redirect
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},

		// allow self-signed certificate
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	defer client.CloseIdleConnections()

	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/blah?param=1", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	assert.Equal(t, fmt.Sprintf("https://localhost:%d/blah?param=1", sslPort), resp.Header.Get("Location"))

	resp, err = client.Get(fmt.Sprintf("https://localhost:%d/ping", sslPort))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "pong", string(body))

	srv.Shutdown()
}

func TestRest_RunAutocertModeHTTPOnly(t *testing.T) {
	sslPort := chooseUnusedPort(t)
	srv := Rest{
		Authenticator: &auth.Service{},
		ImageProxy:    &proxy.Image{},
		SSLConfig: SSLConfig{
			SSLMode: Auto,
			Port:    sslPort,
		},
		RemarkURL: fmt.Sprintf("https://localhost:%d", sslPort),
	}

	port := chooseUnusedPort(t)
	go func() {
		// can't check https server locally, just only http server
		srv.Run("", port)
	}()

	waitForServerStart(t, sslPort, port)

	client := http.Client{
		// prevent http redirect
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	defer client.CloseIdleConnections()

	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/blah?param=1", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	assert.Equal(t, fmt.Sprintf("https://localhost:%d/blah?param=1", sslPort), resp.Header.Get("Location"))

	srv.Shutdown()
}

func Test_URLKey(t *testing.T) {
	tbl := []struct {
		url  string
		user store.User
		key  string
	}{
		{"http://example.com/1", store.User{}, "http://example.com/1"},
		{"http://example.com/1", store.User{ID: "user"}, "http://example.com/1"},
		{"http://example.com/1", store.User{ID: "user", Admin: true}, "admin!!http://example.com/1"},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r, err := http.NewRequest("GET", tt.url, http.NoBody)
			require.NoError(t, err)
			if tt.user.ID != "" {
				r = rest.SetUserInfo(r, tt.user)
			}
			assert.Equal(t, tt.key, URLKey(r))
		})
	}
}

func Test_URLKeyWithUser(t *testing.T) {
	tbl := []struct {
		url  string
		user store.User
		key  string
	}{
		{"http://example.com/1", store.User{}, "http://example.com/1"},
		{"http://example.com/1", store.User{ID: "user"}, "user!!http://example.com/1"},
		{"http://example.com/2", store.User{ID: "user2"}, "user2!!http://example.com/2"},
		{"http://example.com/1", store.User{ID: "user", Admin: true}, "admin!!user!!http://example.com/1"},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r, err := http.NewRequest("GET", tt.url, http.NoBody)
			require.NoError(t, err)
			if tt.user.ID != "" {
				r = rest.SetUserInfo(r, tt.user)
			}
			assert.Equal(t, tt.key, URLKeyWithUser(r))
		})
	}
}

func TestRest_parseError(t *testing.T) {
	tbl := []struct {
		err error
		res int
	}{
		{fmt.Errorf("can not vote for his own comment"), rest.ErrVoteSelf},
		{fmt.Errorf("already voted for"), rest.ErrVoteDbl},
		{fmt.Errorf("maximum number of votes exceeded for comment"), rest.ErrVoteMax},
		{fmt.Errorf("minimal score reached for comment"), rest.ErrVoteMinScore},
		{fmt.Errorf("too late to edit"), rest.ErrCommentEditExpired},
		{fmt.Errorf("parent comment with reply can't be edited"), rest.ErrCommentEditChanged},
		{fmt.Errorf("blah blah"), rest.ErrInternal},
	}

	for n, tt := range tbl {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			res := parseError(tt.err, rest.ErrInternal)
			assert.Equal(t, tt.res, res)
		})
	}
}

func TestRest_frameAncestors(t *testing.T) {
	ts, _, teardown := startupT(t, func(o *Rest) {
		o.AllowedAncestors = []string{"'self'", "https://example.com"}
	})

	// test case with frame-ancestors
	client := http.Client{}
	resp, err := client.Get(ts.URL + "/web/index.html")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "frame-ancestors 'self' https://example.com;")
	// httptest.Server.Close waits on connections still in use, and a deferred close does not run
	// until the test ends, so the body has to be released before the server is torn down here
	require.NoError(t, resp.Body.Close())
	client.CloseIdleConnections()
	teardown()

	// test case without frame-ancestors
	ts, _, teardown = startupT(t, func(srv *Rest) {
		srv.AllowedAncestors = []string{}
	})
	defer teardown()
	resp, err = client.Get(ts.URL + "/web/index.html")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "frame-ancestors *;")
}

// startupT runs fully configured testing server
// srvHook is an optional func to set some Rest param after the creation but prior to Run
func startupT(t *testing.T, srvHook ...func(srv *Rest)) (ts *httptest.Server, srv *Rest, teardown func()) {
	tmp := os.TempDir()
	testDB := filepath.Join(t.TempDir(), "test-remark.db") // per-test dir, removed when the test ends

	_ = os.RemoveAll(tmp + "/ava-remark42")
	_ = os.RemoveAll(tmp + "/pics-remark42")

	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: testDB, SiteID: "remark42"})
	require.NoError(t, err)

	memCache := cache.NewScache[[]byte](cache.NewNopCache[[]byte]())

	astore := adminstore.NewStaticStore("123456", []string{"remark42"}, []string{"a1", "a2"}, "admin@remark-42.com")
	restrictedWordsMatcher := service.NewRestrictedWordsMatcher(service.StaticRestrictedWordsLister{Words: []string{"duck"}})

	dataStore := &service.DataStore{
		Engine:                 b,
		EditDuration:           5 * time.Minute,
		MaxCommentSize:         4000,
		AdminStore:             astore,
		MaxVotes:               service.UnlimitedVotes,
		RestrictedWordsMatcher: restrictedWordsMatcher,
	}

	remarkURL := "https://demo.remark42.com"

	srv = &Rest{
		DataService: dataStore,
		Authenticator: auth.NewService(auth.Opts{
			AdminPasswd:  "password",
			SecretReader: token.SecretFunc(func(string) (string, error) { return "secret", nil }),
			AvatarStore:  avatar.NewLocalFS(tmp + "/ava-remark42"),
		}),
		Cache:     memCache,
		WebRoot:   tmp,
		RemarkURL: remarkURL,
		ImageService: image.NewService(&image.FileSystem{
			Location:   tmp + "/pics-remark42",
			Partitions: 100,
			Staging:    tmp + "/pics-remark42/staging",
		}, image.ServiceParams{
			EditDuration: 100 * time.Millisecond,
			ImageAPI:     remarkURL + "/api/v1/picture/",
			ProxyAPI:     remarkURL + "/api/v1/img",
			MaxSize:      10000,
		}),
		ImageProxy:       &proxy.Image{},
		ReadOnlyAge:      10,
		CommentFormatter: store.NewCommentFormatter(&proxy.Image{}),
		Migrator: &Migrator{
			DisqusImporter:    &migrator.Disqus{DataStore: dataStore},
			WordPressImporter: &migrator.WordPress{DataStore: dataStore},
			CommentoImporter:  &migrator.Commento{DataStore: dataStore},
			NativeImporter:    &migrator.Native{DataStore: dataStore},
			NativeExporter:    &migrator.Native{DataStore: dataStore},
			URLMapperMaker:    migrator.NewURLMapper,
			Cache:             memCache,
			KeyStore:          astore,
		},
		NotifyService:    notify.NopService,
		EmojiEnabled:     true,
		openRouteLimiter: 100,
	}
	srv.ScoreThresholds.Low, srv.ScoreThresholds.Critical = -5, -10

	// add some providers. Needed because we don't allow users with unlisted providers to authenticate
	providers := []string{"provider1", "anonymous", "github", "email"}
	for _, p := range providers {
		srv.Authenticator.AddDirectProvider(p, provider.CredCheckerFunc(func(_, _ string) (ok bool, err error) {
			return true, nil
		}))
	}

	for _, h := range srvHook {
		h(srv)
	}

	routes := srv.routes()
	ts = httptest.NewServer(routes)

	teardown = func() {
		ts.Close()
		require.NoError(t, srv.DataService.Close())
		_ = os.RemoveAll(tmp + "/ava-remark42")
		_ = os.RemoveAll(tmp + "/pics-remark42")
	}

	return ts, srv, teardown
}

const (
	// outer bound before a wait is called a hang, generous enough for a loaded CI runner
	waitTimeout  = 30 * time.Second
	pollInterval = 10 * time.Millisecond

	// budget for a server to stop once asked, tight enough to catch a shutdown that hangs
	serverStopTimeout = 10 * time.Second

	// connect budget for a single probe, kept off the poll interval so a slow loopback connect
	// on a loaded runner does not look like a server that is not listening
	probeDialTimeout = time.Second

	// window to prove something did not happen
	notifySettle = 300 * time.Millisecond

	// poll interval for waits that issue an HTTP request. the admin routes allow 10 req/s and
	// the open ones 100 in tests, so this stays below the tighter of the two and the poll
	// cannot manufacture the 429s it would then have to interpret
	httpPoll = 150 * time.Millisecond
)

// waitForCount blocks until got reaches want, failing the test with the last value it saw.
// for work that is delivered asynchronously, such as notifications reaching a mock destination
func waitForCount(t *testing.T, want int, got func() int, msgAndArgs ...any) {
	t.Helper()
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.Equal(c, want, got(), msgAndArgs...)
	}, waitTimeout, pollInterval)
}

// waitForCountSettled waits for got to reach want and then holds it there, so a delivery
// arriving late is caught rather than passing because the count was read the instant it matched
func waitForCountSettled(t *testing.T, want int, got func() int, msgAndArgs ...any) {
	t.Helper()
	waitForCount(t, want, got, msgAndArgs...)
	require.Never(t, func() bool { return got() != want }, notifySettle, pollInterval, msgAndArgs...)
}

// fake auth middleware make user authenticated and uses query's fake_id for ID and fake_name for Name
func fakeAuth(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("fake_id") != "" {
			r = rest.SetUserInfo(r, store.User{
				ID:   r.URL.Query().Get("fake_id"),
				Name: r.URL.Query().Get("fake_name"),
			})
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func get(t *testing.T, url string) (response string, statusCode int) {
	r, err := http.Get(url)
	require.NoError(t, err)
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	require.NoError(t, r.Body.Close())
	return string(body), r.StatusCode
}

func sendReq(r *http.Request, tkn string) (*http.Response, error) {
	client := http.Client{Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
	if tkn != "" {
		r.Header.Set("X-JWT", tkn)
	}
	return client.Do(r)
}

func getWithDevAuth(t *testing.T, url string) (body string, code int) {
	client := &http.Client{Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
	req, err := http.NewRequest("GET", url, http.NoBody)
	require.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	r, err := client.Do(req)
	require.NoError(t, err)
	b, err := io.ReadAll(r.Body)
	assert.NoError(t, err)
	require.NoError(t, r.Body.Close())
	return string(b), r.StatusCode
}

func getWithDev2Auth(t *testing.T, url string) (body string, code int) {
	client := &http.Client{Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
	req, err := http.NewRequest("GET", url, http.NoBody)
	require.NoError(t, err)
	req.Header.Add("X-JWT", dev2Token)
	r, err := client.Do(req)
	require.NoError(t, err)
	b, err := io.ReadAll(r.Body)
	assert.NoError(t, err)
	require.NoError(t, r.Body.Close())
	return string(b), r.StatusCode
}

func getWithAdminAuth(t *testing.T, url string) (response string, statusCode int) {
	client := &http.Client{Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
	req, err := http.NewRequest("GET", url, http.NoBody)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	r, err := client.Do(req)
	require.NoError(t, err)
	body, err := io.ReadAll(r.Body)
	assert.NoError(t, err)
	require.NoError(t, r.Body.Close())
	return string(body), r.StatusCode
}
func post(t *testing.T, url, body string) (*http.Response, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	assert.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	return client.Do(req)
}

func addCommentGetCreatedTime(t *testing.T, c store.Comment, ts *httptest.Server) (id string, created time.Time) {
	b, err := json.Marshal(c)
	require.NoError(t, err, "can't marshal comment %+v", c)

	client := &http.Client{Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
	postURL := ts.URL + "/api/v1/comment"
	if c.Locator.SiteID != "" {
		postURL += "?site=" + c.Locator.SiteID
	}
	req, err := http.NewRequest("POST", postURL, bytes.NewBuffer(b))
	require.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	b, err = io.ReadAll(resp.Body)
	require.NoError(t, resp.Body.Close())
	require.NoError(t, err)

	crResp := R.JSON{}
	err = json.Unmarshal(b, &crResp)
	require.NoError(t, err)
	created, err = time.Parse(time.RFC3339, crResp["time"].(string))
	require.NoError(t, err)
	return crResp["id"].(string), created
}

func addComment(t *testing.T, c store.Comment, ts *httptest.Server) string {
	id, _ := addCommentGetCreatedTime(t, c, ts)
	return id
}

func requireAdminOnly(t *testing.T, req *http.Request) {
	resp, err := sendReq(req, "") // no-auth user
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	resp, err = sendReq(req, devToken) // non-admin user
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// chooseUnusedPort asks the kernel for a free port from the ephemeral range, which makes a
// collision between concurrently running package test binaries very unlikely
func chooseUnusedPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "no free port available")
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())
	return port
}

// waitForServerStart blocks until something accepts on every listed port, failing the test
// naming the port that never came up
func waitForServerStart(t *testing.T, ports ...int) {
	t.Helper()
	for _, port := range ports {
		require.Eventually(t, func() bool {
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), probeDialTimeout)
			if err != nil {
				return false
			}
			_ = conn.Close()
			return true
		}, waitTimeout, pollInterval, "server on port %d didn't start", port)
	}
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(
		m,
		// this will be fixed in https://github.com/hashicorp/golang-lru/issues/159
		goleak.IgnoreTopFunction("github.com/hashicorp/golang-lru/v2/expirable.NewLRU[...].func1"),
		// regexp2, pulled in by chroma for syntax highlighting, keeps one shared clock goroutine
		// alive for up to a second after the last match with a timeout, sleeping in 100ms ticks.
		// it ends on its own, but a binary that finishes inside that window is reported as leaking
		goleak.IgnoreAnyFunction("github.com/dlclark/regexp2/v2.runClock"),
	)
}
