package api

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/go-pkgz/routegroup"

	"github.com/umputun/remark42/backend/app/webassets"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebFiles_Open(t *testing.T) {
	frontend := fstest.MapFS{
		"both.html":          {Data: []byte("from the frontend build")},
		"only-frontend.html": {Data: []byte("frontend only")},
	}
	embedded := fstest.MapFS{
		"both.html":          {Data: []byte("from the embedded assets")},
		"only-embedded.html": {Data: []byte("embedded only")},
	}
	w := webFiles{frontend: frontend, embedded: embedded}

	tbl := []struct {
		name    string
		lookup  string
		want    string
		wantErr error
	}{
		{name: "present in both is served from the frontend build", lookup: "both.html", want: "from the frontend build"},
		{name: "frontend only", lookup: "only-frontend.html", want: "frontend only"},
		{name: "embedded only", lookup: "only-embedded.html", want: "embedded only"},
		{name: "missing in both", lookup: "neither.html", wantErr: fs.ErrNotExist},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			f, err := w.Open(tt.lookup)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			defer f.Close()
			b, err := io.ReadAll(f)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(b))
		})
	}
}

func TestWebFiles_OpenJSAlias(t *testing.T) {
	frontend := fstest.MapFS{
		"embed.mjs":   {Data: []byte("module embed")},
		"counter.js":  {Data: []byte("operator's own counter")},
		"counter.mjs": {Data: []byte("module counter")},
		"widget.mjs":  {Data: []byte("module widget")},
	}
	embedded := fstest.MapFS{
		"legacy.mjs": {Data: []byte("module legacy")},
		"widget.js":  {Data: []byte("embedded widget")},
	}
	w := webFiles{frontend: frontend, embedded: embedded}

	tbl := []struct {
		name    string
		lookup  string
		want    string
		wantErr error
	}{
		{name: "missing js served from the mjs sibling", lookup: "embed.js", want: "module embed"},
		{name: "alias reaches the embedded assets too", lookup: "legacy.js", want: "module legacy"},
		{name: "a real js file wins over its sibling", lookup: "counter.js", want: "operator's own counter"},
		{name: "an embedded js wins over a frontend sibling", lookup: "widget.js", want: "embedded widget"},
		{name: "mjs is still served directly", lookup: "embed.mjs", want: "module embed"},
		{name: "neither name present", lookup: "absent.js", wantErr: fs.ErrNotExist},
		{name: "only js aliases, not other extensions", lookup: "embed.html", wantErr: fs.ErrNotExist},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			f, err := w.Open(tt.lookup)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			defer f.Close()
			b, err := io.ReadAll(f)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(b))
		})
	}
}

func TestWebFiles_OpenJSAliasNamesTheRequestedFile(t *testing.T) {
	w := webFiles{frontend: fstest.MapFS{}, embedded: fstest.MapFS{}}

	_, err := w.Open("absent.js")
	require.Error(t, err)
	assert.ErrorIs(t, err, fs.ErrNotExist)
	assert.Contains(t, err.Error(), "absent.js")
	assert.NotContains(t, err.Error(), "absent.mjs")
}

func TestWebFiles_OpenJSAliasUnreadableSibling(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file permissions")
	}

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "embed.mjs"), []byte("module embed"), 0o000))
	w := webFiles{frontend: os.DirFS(dir), embedded: fstest.MapFS{}}

	f, err := w.Open("embed.js")
	require.Error(t, err)
	assert.ErrorIs(t, err, fs.ErrPermission)
	assert.NotErrorIs(t, err, fs.ErrNotExist, "an unreadable sibling must not render as 404")
	if err == nil {
		_ = f.Close()
	}
}

// TestEmptyFS_ServesNothing pins the stand-in used when the frontend source cannot be opened:
// every name must report as missing rather than panicking, since it backs a nil-free fallback.
func TestEmptyFS_ServesNothing(t *testing.T) {
	for _, name := range []string{".", "index.html", "web/index.html"} {
		t.Run(name, func(t *testing.T) {
			f, err := emptyFS{}.Open(name)
			require.Error(t, err)
			assert.ErrorIs(t, err, fs.ErrNotExist)
			assert.Nil(t, f)
		})
	}
}

// TestWebFiles_EmptyFrontendFallsThrough covers the shape routes() builds when fs.Sub refuses:
// the embedded assets must still answer even though the frontend source serves nothing.
func TestWebFiles_EmptyFrontendFallsThrough(t *testing.T) {
	w := webFiles{frontend: emptyFS{}, embedded: webassets.FS}

	want, err := fs.ReadFile(webassets.FS, "privacy.html")
	require.NoError(t, err)

	f, err := w.Open("privacy.html")
	require.NoError(t, err)
	defer f.Close()
	got, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got))
}

// TestWebFiles_OpenRootIsNotListed keeps the embedded assets from being browsable: they answer
// for their own names only, so a web root that has gone missing reports as missing.
func TestWebFiles_OpenRootIsNotListed(t *testing.T) {
	w := webFiles{frontend: os.DirFS(filepath.Join(t.TempDir(), "absent")), embedded: webassets.FS}

	f, err := w.Open(".")
	require.Error(t, err, "the embedded assets must not answer for the directory itself")
	assert.ErrorIs(t, err, fs.ErrNotExist)
	if err == nil {
		_ = f.Close()
	}

	// the assets themselves still serve
	f, err = w.Open("privacy.html")
	require.NoError(t, err)
	require.NoError(t, f.Close())
}

// TestWebFiles_OpenInvalidName pins that a name fs rejects reports as missing rather than invalid.
// os.DirFS returns fs.ErrInvalid for these, which http.FileServer renders as 500, so the check has
// to happen before the lookup. A memory filesystem cannot show this: it reports missing either way.
func TestWebFiles_OpenInvalidName(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "privacy.html"), []byte("frontend"), 0o600))
	w := webFiles{frontend: os.DirFS(dir), embedded: webassets.FS}

	for _, name := range []string{"../escape.html", "/etc/passwd", "a\x00b.html", "./privacy.html"} {
		t.Run(name, func(t *testing.T) {
			f, err := w.Open(name)
			require.Error(t, err)
			assert.ErrorIs(t, err, fs.ErrNotExist)
			assert.NotErrorIs(t, err, fs.ErrInvalid, "an invalid name must not surface as 500")
			if err == nil {
				_ = f.Close()
			}
		})
	}
}

// TestWebFiles_OpenUnreadableFrontendFile pins the rule that only a missing file falls through:
// a frontend file that cannot be read must report that, not be masked by the embedded copy.
func TestWebFiles_OpenUnreadableFrontendFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file permissions")
	}

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "privacy.html"), []byte("operator's own"), 0o000))

	w := webFiles{
		frontend: os.DirFS(dir),
		embedded: fstest.MapFS{"privacy.html": {Data: []byte("built in")}},
	}

	f, err := w.Open("privacy.html")
	require.Error(t, err, "an unreadable frontend file must not be replaced by the embedded copy")
	assert.NotErrorIs(t, err, fs.ErrNotExist, "the error must stay a permission error so it does not render as 404")
	assert.ErrorIs(t, err, fs.ErrPermission)
	if err == nil {
		_ = f.Close()
	}
}

func TestTemplatedFS_SubstitutesTheInstanceURL(t *testing.T) {
	const placeholder = "host: '" + remarkURLPlaceholder + "'"
	source := fstest.MapFS{
		"iframe.html":  {Data: []byte(placeholder)},
		"embed.mjs":    {Data: []byte(placeholder)},
		"embed.js":     {Data: []byte(placeholder)},
		"remark.css":   {Data: []byte(placeholder)},
		"nothing.html": {Data: []byte("no marker here")},
	}
	tfs := templatedFS{fs: source, remarkURL: "https://remark.example.com"}

	tbl := []struct {
		name string
		want string
	}{
		{"iframe.html", "host: 'https://remark.example.com'"},
		{"embed.mjs", "host: 'https://remark.example.com'"},
		{"embed.js", "host: 'https://remark.example.com'"},
		// the docker image rewrites html, js and mjs and nothing else, and a stylesheet carrying
		// the marker would mean the frontend started templating a file type this does not cover
		{"remark.css", placeholder},
		{"nothing.html", "no marker here"},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			f, err := tfs.Open(tt.name)
			require.NoError(t, err)
			defer func() { assert.NoError(t, f.Close()) }()

			body, err := io.ReadAll(f)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(body))

			info, err := f.Stat()
			require.NoError(t, err)
			assert.Equal(t, int64(len(tt.want)), info.Size(),
				"the size has to be the substituted one, or the response is truncated or left hanging")
			assert.Equal(t, tt.name, info.Name())
		})
	}
}

func TestTemplatedFS_PassesErrorsThrough(t *testing.T) {
	tfs := templatedFS{fs: fstest.MapFS{}, remarkURL: "https://remark.example.com"}

	_, err := tfs.Open("absent.html")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

// TestRest_FileServerFillsInTheInstanceURL covers the reason templatedFS exists: the binary serves
// the frontend build embedded in itself, and nothing else fills the placeholder in for it.
func TestRest_FileServerFillsInTheInstanceURL(t *testing.T) {
	frontend := fstest.MapFS{
		"embed.mjs":  {Data: []byte("host=\"" + remarkURLPlaceholder + "\"")},
		"logo.svg":   {Data: []byte(remarkURLPlaceholder)},
		"plain.html": {Data: []byte("nothing to fill in")},
	}
	router := routegroup.New(http.NewServeMux())
	addFileServer(router, frontend, filepath.Join(t.TempDir(), "absent"), "test-version", "https://remark.example.com")

	ts := httptest.NewServer(router)
	defer ts.Close()

	t.Run("the bundle carries the configured url", func(t *testing.T) {
		body, code := get(t, ts.URL+"/web/embed.mjs")
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, `host="https://remark.example.com"`, body)
	})

	t.Run("the legacy js name carries it too", func(t *testing.T) {
		body, code := get(t, ts.URL+"/web/embed.js")
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, `host="https://remark.example.com"`, body)
	})

	t.Run("other types are served untouched", func(t *testing.T) {
		body, code := get(t, ts.URL+"/web/logo.svg")
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, remarkURLPlaceholder, body)
	})

	t.Run("a file without the marker is unchanged", func(t *testing.T) {
		body, code := get(t, ts.URL+"/web/plain.html")
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, "nothing to fill in", body)
	})
}

// TestRest_FileServerEtagVariesWithTheInstanceURL covers the case this substitution exists for. An
// operator who notices the widget is addressed to the wrong host corrects REMARK_URL and restarts,
// and the binary and so the version is unchanged. If the validator ignores remarkURL the client
// revalidates, gets 304 and keeps the bundle pointing at the old host. Cache-Control is no-cache,
// so it revalidates every time and never ages out of that state.
func TestRest_FileServerEtagVariesWithTheInstanceURL(t *testing.T) {
	frontend := fstest.MapFS{"embed.mjs": {Data: []byte("host=\"" + remarkURLPlaceholder + "\"")}}

	etagFor := func(remarkURL string) string {
		router := routegroup.New(http.NewServeMux())
		addFileServer(router, frontend, filepath.Join(t.TempDir(), "absent"), "test-version", remarkURL)
		ts := httptest.NewServer(router)
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/web/embed.mjs")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		return resp.Header.Get("Etag")
	}

	first := etagFor("https://old.example.com")
	second := etagFor("https://new.example.com")

	require.NotEmpty(t, first, "the file server has to send a validator at all")
	assert.NotEqual(t, first, second,
		"same version and same path, different instance url: the validator has to change or the "+
			"client keeps a bundle addressed to the old host")
}
