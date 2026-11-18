package migrator

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-pkgz/auth/v2/avatar"
	"github.com/go-pkgz/auth/v2/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/admin"
	"github.com/umputun/remark42/backend/app/store/engine"
	"github.com/umputun/remark42/backend/app/store/service"
)

// fakeDisqus answers the avatar endpoint the way disqus does: a redirect to the CDN image, or to its
// generic avatar for a user without one
type fakeDisqus struct {
	*httptest.Server
	images map[string][]byte // avatar images by username
	status map[string][]int  // statuses to answer with, one per request, before the redirect
	retry  string            // Retry-After sent with a 429

	mu       sync.Mutex
	requests map[string]int // avatar endpoint requests by username
}

func newFakeDisqus(t *testing.T) *fakeDisqus {
	t.Helper()
	f := &fakeDisqus{images: map[string][]byte{}, status: map[string][]int{}, requests: map[string]int{}}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, ok := strings.CutPrefix(r.URL.Path, "/api/users/avatars/"); ok {
			user = strings.TrimSuffix(user, ".jpg")
			f.mu.Lock()
			n := f.requests[user]
			f.requests[user]++
			f.mu.Unlock()
			if statuses := f.status[user]; n < len(statuses) {
				if statuses[n] == http.StatusTooManyRequests && f.retry != "" {
					w.Header().Set("Retry-After", f.retry)
				}
				w.WriteHeader(statuses[n])
				return
			}
			if _, ok := f.images[user]; ok {
				http.Redirect(w, r, "/uploads/users/"+user+"/avatar92.jpg", http.StatusFound)
				return
			}
			http.Redirect(w, r, "/images/noavatar92.png", http.StatusFound)
			return
		}
		if user, ok := strings.CutPrefix(r.URL.Path, "/uploads/users/"); ok {
			_, _ = w.Write(f.images[strings.TrimSuffix(user, "/avatar92.jpg")])
			return
		}
		if r.URL.Path == "/images/noavatar92.png" {
			_, _ = w.Write(testPNG(t, color.White))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeDisqus) requested(user string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests[user]
}

// newAvatarTestStore makes a data store on a fresh bolt db with the site "test"
func newAvatarTestStore(t *testing.T) *service.DataStore {
	t.Helper()
	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: filepath.Join(t.TempDir(), "test.db"), SiteID: "test"})
	require.NoError(t, err)
	ds := &service.DataStore{Engine: b, AdminStore: admin.NewStaticStore("12345", nil, []string{}, "")}
	t.Cleanup(func() { _ = ds.Close() })
	return ds
}

func testPNG(t *testing.T, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for x := range 8 {
		for y := range 8 {
			img.Set(x, y, c)
		}
	}
	buf := bytes.Buffer{}
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// testAvatars returns disqus avatars fetched from f, saved with a real avatar proxy on a local store,
// and the waits it asked for in place of sleeping
func testAvatars(t *testing.T, f *fakeDisqus) (*disqusAvatars, *avatar.Proxy, *[]time.Duration) {
	t.Helper()
	proxy := &avatar.Proxy{L: logger.NoOp, Store: avatar.NewLocalFS(t.TempDir()), URL: "https://remark42.example.com",
		RoutePath: "/api/v1/avatar"}
	a := newDisqusAvatars(proxy)
	a.baseURL = f.URL + "/api/users/avatars/"
	var waits []time.Duration
	a.sleep = func(d time.Duration) { waits = append(waits, d) }
	return a, proxy, &waits
}

// served returns what the avatar proxy serves for a picture URL it returned
func served(t *testing.T, proxy *avatar.Proxy, picture string) []byte {
	t.Helper()
	path, ok := strings.CutPrefix(picture, proxy.URL)
	require.True(t, ok, "picture %q is served by remark42", picture)
	rr := httptest.NewRecorder()
	proxy.Handler(rr, httptest.NewRequest(http.MethodGet, path, http.NoBody))
	require.Equal(t, http.StatusOK, rr.Code)
	return rr.Body.Bytes()
}

func TestDisqus_ImportAvatars(t *testing.T) {
	f := newFakeDisqus(t)
	alice, carol := testPNG(t, color.RGBA{R: 255, A: 255}), testPNG(t, color.RGBA{B: 255, A: 255})
	f.images["alice"], f.images["carol"] = alice, carol
	f.status["carol"] = []int{http.StatusTooManyRequests, http.StatusServiceUnavailable}
	f.retry = "7" // a wait the doubling backoff never produces
	f.status["dave"] = []int{500, 500, 500, 500}
	f.images["dave"] = alice

	post := func(id, username, name, deleted, spam string) string {
		return `<post dsq:id="` + id + `"><message><![CDATA[<p>text ` + id + `</p>]]></message><createdAt>2020-01-01T10:00:00Z</createdAt>` +
			`<isDeleted>` + deleted + `</isDeleted><isSpam>` + spam + `</isSpam><author><name>` + name + `</name>` +
			`<username>` + username + `</username></author><thread dsq:id="t1"/></post>`
	}
	xml := `<disqus xmlns="http://disqus.com" xmlns:dsq="http://disqus.com/disqus-internals">` +
		`<thread dsq:id="t1"><link>https://example.com/p1</link><createdAt>2020-01-01T09:00:00Z</createdAt></thread>` +
		post("c1", "alice", "Alice", "false", "false") +
		post("c2", "bob", "Bob", "false", "false") + // disqus gives him its generic avatar
		post("c3", "carol", "Carol", "false", "false") + // throttled first
		post("c4", "dave", "Dave", "false", "false") + // disqus fails every time
		post("c5", "", "Guest", "false", "false") + // no disqus account
		post("c9", "", "alice", "false", "false") + // no disqus account, named like a user with one
		post("c6", "alice", "Alice", "false", "false") + // second comment, no second request
		post("c7", "erin", "Erin", "true", "false") + // deleted, not imported
		post("c8", "frank", "Frank", "false", "true") + // spam, not imported
		`</disqus>`

	ds := newAvatarTestStore(t)
	avatars, proxy, waits := testAvatars(t, f)
	d := Disqus{DataStore: ds, avatars: avatars}
	size, err := d.Import(io.MultiReader(strings.NewReader(xml)), "test") // not seekable, so it goes through a temp file
	require.NoError(t, err)
	assert.Equal(t, 7, size)

	get := func(id string) store.Comment {
		c, e := ds.Get(store.Locator{SiteID: "test", URL: "https://example.com/p1"}, id, adminUser)
		require.NoError(t, e)
		return c
	}
	assert.Equal(t, alice, served(t, proxy, get("c1").User.Picture), "alice's own image is stored and served")
	assert.Equal(t, get("c1").User.Picture, get("c6").User.Picture)
	assert.Equal(t, carol, served(t, proxy, get("c3").User.Picture), "carol's image is stored once disqus answers")
	assert.Empty(t, get("c2").User.Picture, "disqus' generic avatar is not stored")
	assert.Empty(t, get("c4").User.Picture, "an avatar disqus failed to return is not linked")
	assert.Empty(t, get("c5").User.Picture)
	assert.Empty(t, get("c9").User.Picture, "a guest does not get the avatar of the account named like them")

	assert.Equal(t, 1, f.requested("alice"))
	assert.Equal(t, 3, f.requested("carol"))
	assert.Equal(t, 4, f.requested("dave"), "a failing request is retried")
	assert.Zero(t, f.requested("erin")+f.requested("frank"), "no avatars for comments that are not imported")
	assert.Contains(t, *waits, 7*time.Second, "Retry-After is honored")
	assert.Contains(t, *waits, disqusAvatarPause, "requests are paced")
}

func TestDisqus_ImportAvatarsDisqusDown(t *testing.T) {
	f := newFakeDisqus(t)
	var posts strings.Builder
	for _, user := range []string{"u1", "u2", "u3", "u4", "u5", "u6", "u7"} {
		f.status[user] = []int{502, 502, 502, 502}
		posts.WriteString(`<post dsq:id="` + user + `"><message>m</message><createdAt>2020-01-01T10:00:00Z</createdAt>` +
			`<author><name>` + user + `</name><username>` + user + `</username></author><thread dsq:id="t1"/></post>`)
	}
	xml := `<disqus xmlns="http://disqus.com" xmlns:dsq="http://disqus.com/disqus-internals">` +
		`<thread dsq:id="t1"><link>https://example.com/p1</link><createdAt>2020-01-01T09:00:00Z</createdAt></thread>` +
		posts.String() + `</disqus>`

	ds := newAvatarTestStore(t)
	avatars, _, _ := testAvatars(t, f)
	size, err := (&Disqus{DataStore: ds, avatars: avatars}).Import(strings.NewReader(xml), "test")
	require.NoError(t, err, "comments are imported without avatars")
	assert.Equal(t, 7, size)
	assert.Equal(t, 4, f.requested("u5"))
	assert.Zero(t, f.requested("u6"), "after five users in a row disqus failed for, the rest are not tried")
}

func TestDisqus_ImportWithoutAvatars(t *testing.T) {
	ds := newAvatarTestStore(t)
	fh := strings.NewReader(`<disqus xmlns="http://disqus.com" xmlns:dsq="http://disqus.com/disqus-internals">` +
		`<thread dsq:id="t1"><link>https://example.com/p1</link><createdAt>2020-01-01T09:00:00Z</createdAt></thread>` +
		`<post dsq:id="c1"><message>m</message><createdAt>2020-01-01T10:00:00Z</createdAt><author><name>A</name>` +
		`<username>alice</username></author><thread dsq:id="t1"/></post></disqus>`)
	size, err := (&Disqus{DataStore: ds}).Import(fh, "test")
	require.NoError(t, err)
	assert.Equal(t, 1, size)
	c, err := ds.Get(store.Locator{SiteID: "test", URL: "https://example.com/p1"}, "c1", adminUser)
	require.NoError(t, err)
	assert.Empty(t, c.User.Picture)
}

func TestDisqusAvatars_RedirectHosts(t *testing.T) {
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(testPNG(t, color.Black)) }))
	defer other.Close()
	f := newFakeDisqus(t)
	f.Config.Handler = http.RedirectHandler(other.URL+"/internal.png", http.StatusFound)
	avatars, _, waits := testAvatars(t, f)
	assert.Empty(t, avatars.fetch([]string{"alice", "bob", "carol", "dave", "erin", "frank"}), "a redirect away from disqus is not followed")
	for _, w := range *waits {
		require.Equal(t, disqusAvatarPause, w, "a refused redirect is not retried, the only waits are the pacing")
	}

	f.images["frank"] = testPNG(t, color.Black)
	f.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/users/avatars/frank") {
			http.Redirect(w, r, "/uploads/users/frank/avatar92.jpg", http.StatusFound)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/uploads/") {
			_, _ = w.Write(f.images["frank"])
			return
		}
		http.Redirect(w, r, other.URL+"/internal.png", http.StatusFound)
	})
	assert.Len(t, avatars.fetch([]string{"alice", "bob", "carol", "dave", "erin", "frank"}), 1,
		"refused redirects do not count as disqus failing, so the user after them still gets an avatar")
}

func TestDisqusAvatars_RetryWait(t *testing.T) {
	assert.Equal(t, time.Second, lastWait(retryAfter{}, 1))
	assert.Equal(t, 4*time.Second, lastWait(retryAfter{}, 3))
	assert.Equal(t, 7*time.Second, lastWait(retryAfter{header: "7"}, 1))
	assert.Equal(t, time.Minute, lastWait(retryAfter{header: "3600"}, 1), "a long Retry-After is capped")
	assert.Equal(t, time.Duration(0), lastWait(retryAfter{header: time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)}, 1))
}
