package cmd

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-pkgz/auth/v2/avatar"
	"github.com/go-pkgz/auth/v2/token"
	log "github.com/go-pkgz/lgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomAvatarSaver_Put(t *testing.T) {
	picture := testPNG(t)

	// pictures is where the users' pictures live, on a loopback address
	var requests atomic.Int32
	var authorization atomic.Value
	mux := http.NewServeMux()
	mux.HandleFunc("GET /picture.png", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		authorization.Store(r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(picture)
	})
	mux.HandleFunc("GET /moved.png", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/picture.png", http.StatusFound)
	})
	mux.HandleFunc("GET /page.html", func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = io.WriteString(w, "<html><body>not a picture</body></html>")
	})
	pictures := httptest.NewServer(mux)
	defer pictures.Close()

	// the client the auth library hands to Put adds the user's access token to what it requests
	withToken := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		r = r.Clone(r.Context())
		r.Header.Set("Authorization", "Bearer access-token")
		return http.DefaultTransport.RoundTrip(r)
	})}
	defer http.DefaultClient.CloseIdleConnections()

	// direct reaches the loopback address safehttp refuses, to test what happens to a picture that arrives
	direct := http.DefaultTransport.(*http.Transport).Clone()
	defer direct.CloseIdleConnections()

	tbl := []struct {
		name         string
		picture      string
		transport    http.RoundTripper
		wantPicture  bool // the stored avatar is the picture, an identicon otherwise
		wantRequests int32
	}{
		{name: "picture is stored", picture: pictures.URL + "/picture.png", transport: direct, wantPicture: true, wantRequests: 1},
		{name: "redirect is followed", picture: pictures.URL + "/moved.png", transport: direct, wantPicture: true, wantRequests: 1},
		{name: "private address is not requested", picture: pictures.URL + "/picture.png"},
		{name: "not an image", picture: pictures.URL + "/page.html", transport: direct, wantRequests: 1},
		{name: "not found", picture: pictures.URL + "/missing.png", transport: direct},
		{name: "not http", picture: "file:///etc/passwd", transport: direct},
		{name: "no picture", picture: ""},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			requests.Store(0)
			authorization.Store("")
			proxy := &avatar.Proxy{L: log.Default(), Store: avatar.NewLocalFS(t.TempDir()), RoutePath: "/avatar", URL: "http://remark42.example.com"}
			saver := customAvatarSaver{proxy: proxy, transport: tt.transport}

			avatarURL, err := saver.Put(token.User{ID: "oidc_user1", Name: "user", Picture: tt.picture}, withToken)
			require.NoError(t, err)
			assert.Contains(t, avatarURL, "http://remark42.example.com/avatar/")
			assert.Equal(t, tt.wantRequests, requests.Load(), "requests for the picture")
			assert.Empty(t, authorization.Load(), "the access token is not sent with the request for the picture")

			rd, _, err := proxy.Store.Get(path.Base(avatarURL))
			require.NoError(t, err)
			stored, err := io.ReadAll(rd)
			require.NoError(t, err)
			require.NoError(t, rd.Close())
			if tt.wantPicture {
				assert.Equal(t, picture, stored)
				return
			}
			assert.NotEqual(t, picture, stored)
			_, err = png.Decode(bytes.NewReader(stored))
			assert.NoError(t, err, "identicon is stored in place of the picture")
		})
	}
}

// a Gravatar address asks for 404 when the email has no picture there, the usual answer, which leaves
// the warnings to the pictures that fail
func TestCustomAvatarSaver_PutLogsMissingGravatarQuietly(t *testing.T) {
	logs := &bytes.Buffer{}
	log.Setup(log.Out(logs), log.Err(logs))
	t.Cleanup(func() { log.Setup() })
	notFound := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: http.NoBody, Request: r}, nil
	})

	tbl := []struct {
		name, picture string
		warn          bool
	}{
		{name: "gravatar without a picture", picture: gravatarURL("a@example.com")},
		{name: "provider picture not found", picture: "https://example.com/avatar.png", warn: true},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			logs.Reset()
			proxy := &avatar.Proxy{L: log.Default(), Store: avatar.NewLocalFS(t.TempDir()), RoutePath: "/avatar", URL: "http://remark42.example.com"}
			saver := customAvatarSaver{proxy: proxy, transport: notFound}

			avatarURL, err := saver.Put(token.User{ID: "oidc_user1", Name: "user", Picture: tt.picture}, nil)
			require.NoError(t, err)
			assert.Contains(t, avatarURL, "http://remark42.example.com/avatar/", "signed in with an identicon")
			assert.Equal(t, tt.warn, strings.Contains(logs.String(), " WARN "), logs.String())
		})
	}
}

func TestCustomAvatarSaver_PutRefusesUserWithoutID(t *testing.T) {
	store := avatar.NewLocalFS(t.TempDir())
	saver := customAvatarSaver{proxy: &avatar.Proxy{L: log.Default(), Store: store, RoutePath: "/avatar", URL: "http://remark42.example.com"}}

	avatarURL, err := saver.Put(token.User{Name: "user"}, nil)
	require.EqualError(t, err, "user info has no user id")
	assert.Empty(t, avatarURL)
	ids, err := store.List()
	require.NoError(t, err)
	assert.Empty(t, ids, "stored avatars")
}

func TestPictureHost(t *testing.T) {
	assert.Equal(t, "example.com", pictureHost("https://user:pass@example.com:8443/a.png?token=secret"))
	assert.Equal(t, "an address without a host", pictureHost("/a.png?token=secret"))
	assert.Equal(t, "an address without a host", pictureHost("://"))
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// testPNG returns a small picture, of a size the avatar proxy stores as it is
func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for i := range 8 {
		img.Set(i, i, color.RGBA{R: 200, G: uint8(i * 30), B: 50, A: 255})
	}
	buf := bytes.Buffer{}
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}
