package migrator

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-pkgz/auth/v2/avatar"
	"github.com/go-pkgz/auth/v2/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/safehttp"
	"github.com/umputun/remark42/backend/app/store"
)

// photoHosts is a forward proxy standing in for the public hosts of commenters' photos. The photos are
// requested by public addresses through the real safehttp transport, which takes the proxy from the
// environment, so private addresses are refused as they are in production, before the proxy sees them.
type photoHosts struct {
	*httptest.Server
	mu       sync.Mutex
	handlers map[string]http.HandlerFunc // by the requested url
	seen     []string                    // requested urls, in order
}

func newPhotoHosts(t *testing.T) *photoHosts {
	t.Helper()
	h := &photoHosts{handlers: map[string]http.HandlerFunc{}}
	h.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		h.seen = append(h.seen, r.URL.String())
		handler, ok := h.handlers[r.URL.String()]
		h.mu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		handler(w, r)
	}))
	t.Cleanup(h.Close)
	// the proxy variables in both cases, and NO_PROXY cleared, so values from the developer's shell do not leak in
	for name, val := range map[string]string{"HTTP_PROXY": h.URL, "HTTPS_PROXY": h.URL, "NO_PROXY": "",
		"http_proxy": h.URL, "https_proxy": h.URL, "no_proxy": "", "REQUEST_METHOD": ""} {
		t.Setenv(name, val)
	}
	return h
}

func (h *photoHosts) handle(photoURL string, handler http.HandlerFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handlers[photoURL] = handler
}

func (h *photoHosts) serve(photoURL string, body []byte) {
	h.handle(photoURL, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) })
}

func (h *photoHosts) requested() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.seen...)
}

// testCommentoAvatars returns commento avatars stored with a real avatar proxy on a local store, and the
// waits it asked for in place of sleeping
func testCommentoAvatars(t *testing.T) (*commentoAvatars, *avatar.Proxy, *[]time.Duration) {
	t.Helper()
	proxy := &avatar.Proxy{L: logger.NoOp, Store: avatar.NewLocalFS(t.TempDir()), URL: "https://remark42.example.com",
		RoutePath: "/api/v1/avatar"}
	a := newCommentoAvatars(proxy)
	var waits []time.Duration
	a.sleep = func(d time.Duration) { waits = append(waits, d) }
	return a, proxy, &waits
}

func TestCommentoAvatars_Fetch(t *testing.T) {
	hosts := newPhotoHosts(t)
	alice := testPNG(t, color.RGBA{R: 255, A: 255})
	hosts.serve("http://203.0.113.1/alice.png", alice)
	hosts.handle("http://203.0.113.1/moved.png", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://203.0.113.2/bob.png", http.StatusFound)
	})
	bob := testPNG(t, color.RGBA{G: 255, A: 255})
	var bobReferer atomic.Value
	hosts.handle("http://203.0.113.2/bob.png", func(w http.ResponseWriter, r *http.Request) {
		bobReferer.Store(r.Header.Get("Referer"))
		_, _ = w.Write(bob)
	})
	hosts.serve("http://203.0.113.1/page.html", []byte("<html><body>not a photo</body></html>"))
	hosts.serve("http://203.0.113.1/big.png", append(testPNG(t, color.Black), make([]byte, commentoPhotoMaxSize)...))
	hosts.handle("http://203.0.113.10/to-metadata.png", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
	})
	hosts.handle("http://203.0.113.10/to-file.png", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "file:///etc/passwd", http.StatusFound)
	})
	hosts.handle("http://203.0.113.20/loop.png", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://203.0.113.20/loop.png", http.StatusFound)
	})

	// a server on a loopback address, reached directly: requests to loopback addresses skip the proxy
	var loopbackRequests atomic.Int32
	loopback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		loopbackRequests.Add(1)
		_, _ = w.Write(alice)
	}))
	defer loopback.Close()
	hosts.handle("http://203.0.113.10/to-loopback.png", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, loopback.URL+"/alice.png", http.StatusFound)
	})

	a, proxy, waits := testCommentoAvatars(t)
	res := a.fetch([]commentoPhoto{
		{userID: "commento_alice", url: "http://203.0.113.1/alice.png"},
		{userID: "commento_bob", url: "http://203.0.113.1/moved.png"},
		{userID: "commento_page", url: "http://203.0.113.1/page.html"},
		{userID: "commento_big", url: "http://203.0.113.1/big.png"},
		{userID: "commento_gone", url: "http://203.0.113.1/gone.png"},
		{userID: "commento_private", url: "http://10.1.2.3/photo.png"},
		{userID: "commento_metadata", url: "http://203.0.113.10/to-metadata.png"},
		{userID: "commento_loopback", url: loopback.URL + "/alice.png"},
		{userID: "commento_to_loopback", url: "http://203.0.113.10/to-loopback.png"},
		{userID: "commento_file", url: "http://203.0.113.10/to-file.png"},
		{userID: "commento_loop", url: "http://203.0.113.20/loop.png"},
	})

	require.Len(t, res, 2, "only the photos that are images on public addresses are stored: %v", res)
	assert.Equal(t, alice, served(t, proxy, res["commento_alice"]))
	assert.Equal(t, bob, served(t, proxy, res["commento_bob"]), "a redirect between public hosts is followed")
	assert.Equal(t, "", bobReferer.Load(), "the address redirected from is not sent to the next host")
	assert.Zero(t, loopbackRequests.Load(), "a loopback address is not requested, directly or by redirect")

	seen := hosts.requested()
	assert.NotContains(t, seen, "http://10.1.2.3/photo.png", "a private address is refused before it is requested")
	assert.NotContains(t, seen, "http://169.254.169.254/latest/meta-data/", "a redirect to a private address is refused")
	loops := 0
	for _, u := range seen {
		if u == "http://203.0.113.20/loop.png" {
			loops++
		}
	}
	assert.Equal(t, commentoPhotoMaxRedirects+1, loops, "redirects are followed up to the limit")

	require.Len(t, *waits, 10, "one pause between each two photos")
	for _, w := range *waits {
		assert.Equal(t, commentoPhotoPause, w)
	}
}

func TestCommentoAvatars_FailuresStopFetching(t *testing.T) {
	hosts := newPhotoHosts(t)
	unavailable := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }
	toFile := func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "file:///etc/passwd", http.StatusFound)
	}
	// a missing photo and a redirect to a file are answers about one photo, each starts the count again
	names := []string{"u1", "u2", "u3", "u4", "gone", "u5", "u6", "u7", "u8", "file", "u9", "u10", "u11", "u12", "u13", "u14"}
	photos := make([]commentoPhoto, 0, len(names))
	want := make([]string, 0, len(names))
	for _, name := range names {
		photoURL := "http://203.0.113.1/" + name + ".png"
		switch name {
		case "gone":
		case "file":
			hosts.handle(photoURL, toFile)
		default:
			hosts.handle(photoURL, unavailable)
		}
		photos = append(photos, commentoPhoto{userID: "commento_" + name, url: photoURL})
		if name != "u14" {
			want = append(want, photoURL)
		}
	}
	a, _, _ := testCommentoAvatars(t)
	assert.Empty(t, a.fetch(photos))
	assert.Equal(t, want, hosts.requested(), "after five photos in a row without an answer, the rest are not requested")
}

func TestCommentoAvatars_StatusBeforeBody(t *testing.T) {
	hosts := newPhotoHosts(t)
	release := make(chan struct{})
	defer close(release) // lets the handler return before the server closes
	hosts.handle("http://203.0.113.1/gone.png", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.(http.Flusher).Flush()
		select { // the error page never arrives
		case <-r.Context().Done():
		case <-release:
		}
	})
	a, _, _ := testCommentoAvatars(t)
	a.timeout = time.Hour // only the status can end the request in time
	done := make(chan map[string]string, 1)
	go func() {
		done <- a.fetch([]commentoPhoto{{userID: "commento_gone", url: "http://203.0.113.1/gone.png"}})
	}()
	select {
	case res := <-done:
		assert.Empty(t, res)
	case <-time.After(30 * time.Second):
		t.Fatal("the body of a missing photo is read before its status is checked")
	}
}

func TestCommentoAvatars_PhotoUnanswered(t *testing.T) {
	tbl := []struct {
		name string
		err  error
		want bool
	}{
		{name: "private address", err: fmt.Errorf("dial: %w", safehttp.ErrPrivateAddress)},
		{name: "redirect refused", err: fmt.Errorf("%w: more than 3 redirects", errRedirectRefused)},
		{name: "no such host", err: &net.DNSError{Err: "no such host", Name: "gone.example", IsNotFound: true}},
		{name: "dns timeout", err: &net.DNSError{Err: "i/o timeout", Name: "slow.example", IsTimeout: true}, want: true},
		{name: "deadline", err: context.DeadlineExceeded, want: true},
		{name: "connection refused", err: errors.New("connect: connection refused"), want: true},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, photoUnanswered(tt.err))
		})
	}
}

func TestCommentoAvatars_Timeout(t *testing.T) {
	hosts := newPhotoHosts(t)
	release := make(chan struct{})
	defer close(release) // lets the handler return before the server closes
	hosts.handle("http://203.0.113.1/slow.png", func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})
	a, _, _ := testCommentoAvatars(t)
	assert.Equal(t, commentoPhotoTimeout, a.timeout, "an import has a deadline for each photo")
	a.timeout = 100 * time.Millisecond // the handler never answers, so the timeout always runs out
	start := time.Now()
	assert.Empty(t, a.fetch([]commentoPhoto{{userID: "commento_slow", url: "http://203.0.113.1/slow.png"}}))
	assert.Less(t, time.Since(start), 30*time.Second, "the request is cut off by its timeout")
}

func TestCommento_ImportAvatars(t *testing.T) {
	hosts := newPhotoHosts(t)
	ds := newAvatarTestStore(t)
	_, err := ds.Create(store.Comment{ID: "old", Text: "old comment", User: store.User{ID: "old_user", Name: "Old"},
		Locator: store.Locator{SiteID: "test", URL: "https://example.com/old"}, Timestamp: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)

	alice := testPNG(t, color.RGBA{R: 255, A: 255})
	var oldWhileFetching atomic.Int64
	hosts.serve("http://203.0.113.1/u7.png", alice) // the photo of the first row, which the last row drops
	hosts.handle("http://203.0.113.1/alice.png", func(w http.ResponseWriter, _ *http.Request) {
		n, e := ds.Count(store.Locator{SiteID: "test", URL: "https://example.com/old"})
		if e == nil {
			oldWhileFetching.Store(int64(n))
		}
		_, _ = w.Write(alice)
	})

	export := `{"version":1,"comments":[
{"commentHex":"c1","domain":"example.com","url":"/p1","commenterHex":"u1","markdown":"first","parentHex":"root","state":"approved","creationDate":"2021-03-17T12:00:01Z"},
{"commentHex":"c2","domain":"example.com","url":"/p1","commenterHex":"u1","markdown":"second","parentHex":"root","state":"approved","creationDate":"2021-03-17T12:00:02Z"},
{"commentHex":"c3","domain":"example.com","url":"/p1","commenterHex":"u2","markdown":"waits for moderation","parentHex":"root","state":"unapproved","creationDate":"2021-03-17T12:00:03Z"},
{"commentHex":"c4","domain":"example.com","url":"/p1","commenterHex":"u3","markdown":"no photo","parentHex":"root","creationDate":"2021-03-17T12:00:04Z"},
{"commentHex":"c5","domain":"example.com","url":"/p1","commenterHex":"u4","markdown":"private photo","parentHex":"root","creationDate":"2021-03-17T12:00:05Z"},
{"commentHex":"c6","domain":"example.com","url":"/p1","commenterHex":"u5","markdown":"photo gone","parentHex":"root","creationDate":"2021-03-17T12:00:06Z"},
{"commentHex":"c7","domain":"example.com","url":"/p1","commenterHex":"u7","markdown":"listed twice","parentHex":"root","creationDate":"2021-03-17T12:00:07Z"},
{"commentHex":"c8","domain":"example.com","url":"/p1","commenterHex":"anonymous","markdown":"anonymous","parentHex":"root","creationDate":"2021-03-17T12:00:08Z"}
],"commenters":[
{"commenterHex":"u1","name":"Alice","photo":"http://203.0.113.1/alice.png"},
{"commenterHex":"u2","name":"Not imported","photo":"http://203.0.113.1/u2.png"},
{"commenterHex":"u3","name":"No photo","photo":"undefined"},
{"commenterHex":"u4","name":"Private","photo":"http://192.168.1.1/u4.png"},
{"commenterHex":"u5","name":"Gone","photo":"http://203.0.113.1/gone.png"},
{"commenterHex":"u6","name":"No comments","photo":"http://203.0.113.1/u6.png"},
{"commenterHex":"u7","name":"Before","photo":"http://203.0.113.1/u7.png"},
{"commenterHex":"u7","name":"After","photo":"undefined"},
{"commenterHex":"anonymous","name":"Anonymous","photo":"http://203.0.113.1/anonymous.png"}
]}`
	_, proxy, _ := testCommentoAvatars(t)
	size, err := (&Commento{DataStore: ds, Avatars: proxy}).Import(strings.NewReader(export), "test") // as the server sets it up
	require.NoError(t, err)
	assert.Equal(t, 7, size)

	get := func(id string) store.Comment {
		c, e := ds.Get(store.Locator{SiteID: "test", URL: "https://example.com/p1"}, id, adminUser)
		require.NoError(t, e)
		return c
	}
	assert.Equal(t, alice, served(t, proxy, get("c1").User.Picture), "alice's photo is stored and served by remark42")
	assert.Equal(t, get("c1").User.Picture, get("c2").User.Picture)
	assert.Empty(t, get("c4").User.Picture)
	assert.Empty(t, get("c5").User.Picture, "a photo on a private address is not linked")
	assert.Empty(t, get("c6").User.Picture, "a photo that can't be fetched is not linked")
	assert.Equal(t, "After", get("c7").User.Name)
	assert.Empty(t, get("c7").User.Picture, "a commenter listed twice takes the photo from the row the name comes from")
	assert.Equal(t, []string{"http://203.0.113.1/alice.png", "http://203.0.113.1/gone.png"}, hosts.requested(),
		"photos of commenters without imported comments, and of the anonymous commenter, are not requested")
	assert.Equal(t, int64(1), oldWhileFetching.Load(), "the site keeps its comments while photos are fetched")
	count, err := ds.Count(store.Locator{SiteID: "test", URL: "https://example.com/old"})
	require.NoError(t, err)
	assert.Zero(t, count, "the import replaces the site's comments")
}
