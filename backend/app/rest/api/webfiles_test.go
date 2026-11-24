package api

import (
	"io"
	"io/fs"
	"mime"
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

// TestTemplatedFS_FillsTheEmbeddedPages runs the pages the binary embeds through the substitution,
// as the file server does. The demo, counter and last-comments pages name their site as one exact
// literal, and comments.html carries the instance placeholder: a page that loses either keeps a
// site the instance may not serve, or a host no reader can reach, and nothing else would notice.
func TestTemplatedFS_FillsTheEmbeddedPages(t *testing.T) {
	tfs := templatedFS{fs: webassets.FS, remarkURL: "https://remark.example.com", siteID: "myblog"}

	read := func(t *testing.T, name string) string {
		t.Helper()
		f, err := tfs.Open(name)
		require.NoError(t, err)
		defer func() { assert.NoError(t, f.Close()) }()
		body, err := io.ReadAll(f)
		require.NoError(t, err)
		return string(body)
	}

	for _, name := range []string{"index.html", "counter.html", "last-comments.html"} {
		t.Run(name+" gets the first configured site", func(t *testing.T) {
			body := read(t, name)
			assert.Contains(t, body, `site_id:"myblog"`)
			assert.NotContains(t, body, demoSiteLiteral)
		})
	}

	t.Run("comments.html gets the instance url", func(t *testing.T) {
		body := read(t, "comments.html")
		assert.Contains(t, body, `var host = 'https://remark.example.com';`)
		assert.NotContains(t, body, remarkURLPlaceholder)
	})
}

// TestTemplatedFS_FillsTheDemoSite covers the demo, counter and last-comments pages, which name
// their site as "remark". The docker image rewrites that literal in html files at container start,
// and the binary has to do the same, or an instance serving another site answers those pages with
// "site not found" and their widget asks the API about a site the reader is not signed in to.
func TestTemplatedFS_FillsTheDemoSite(t *testing.T) {
	const built = `var remark_config={site_id:"remark",host:"` + remarkURLPlaceholder + `"}`
	source := fstest.MapFS{
		"index.html":  {Data: []byte(built)},
		"counter.mjs": {Data: []byte(built)},
		"custom.html": {Data: []byte(`var remark_config={site_id:"mine"}`)},
	}

	tbl := []struct {
		name, file, siteID, want string
	}{
		{"html gets the site", "index.html", "myblog",
			`var remark_config={site_id:"myblog",host:"https://remark.example.com"}`},
		{"no site leaves the build's", "index.html", "",
			`var remark_config={site_id:"remark",host:"https://remark.example.com"}`},
		{"the site cannot end the string or the script", "index.html", `x"</script><script>alert(1)//`,
			`var remark_config={site_id:"x\"\u003c/script\u003e\u003cscript\u003ealert(1)//",host:"https://remark.example.com"}`},
		// the docker image rewrites the site in html only, so a script keeps the literal here too
		{"scripts are left alone", "counter.mjs", "myblog",
			`var remark_config={site_id:"remark",host:"https://remark.example.com"}`},
		// only the build's own default is replaced, never a site an operator wrote into a page
		{"another site is left alone", "custom.html", "myblog", `var remark_config={site_id:"mine"}`},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			tfs := templatedFS{fs: source, remarkURL: "https://remark.example.com", siteID: tt.siteID}
			f, err := tfs.Open(tt.file)
			require.NoError(t, err)
			defer func() { assert.NoError(t, f.Close()) }()

			body, err := io.ReadAll(f)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(body))

			info, err := f.Stat()
			require.NoError(t, err)
			assert.Equal(t, int64(len(tt.want)), info.Size())
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
	addFileServer(router, frontend, filepath.Join(t.TempDir(), "absent"), "test-version", "https://remark.example.com", "")

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
		addFileServer(router, frontend, filepath.Join(t.TempDir(), "absent"), "test-version", remarkURL, "")
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

// TestRest_FileServerEtagVariesWithTheDemoSite is the same case for SITE: the demo pages carry the
// first configured site, so an operator who changes it must not keep getting the old page on 304
func TestRest_FileServerEtagVariesWithTheDemoSite(t *testing.T) {
	frontend := fstest.MapFS{"index.html": {Data: []byte(`site_id:"remark"`)}}

	etagFor := func(siteID string) string {
		router := routegroup.New(http.NewServeMux())
		addFileServer(router, frontend, filepath.Join(t.TempDir(), "absent"), "test-version", "https://remark.example.com", siteID)
		ts := httptest.NewServer(router)
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/web/index.html")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		return resp.Header.Get("Etag")
	}

	first := etagFor("old")
	require.NotEmpty(t, first)
	assert.NotEqual(t, first, etagFor("new"), "a different demo site has to change the validator")
}

// revalidate asks for path with the validator a browser holds and reports the status, the body and
// the validator the server answers with
func revalidate(t *testing.T, url, etag string) (code int, body, newEtag string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	require.NoError(t, err)
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(b), resp.Header.Get("Etag")
}

// TestRest_FileServerEtagFollowsTheWebRootFile covers the customisation the documentation offers:
// an operator puts a privacy.html of their own into the web root, or edits it later, under the
// same binary. Cache-Control is no-cache, so a browser that has the page revalidates every time; if
// the validator ignores the file, it gets 304 and keeps showing the old page until the version
// changes. An unchanged file has to keep answering 304, or the validator does nothing at all.
func TestRest_FileServerEtagFollowsTheWebRootFile(t *testing.T) {
	webRoot := t.TempDir()
	page := filepath.Join(webRoot, "privacy.html")
	require.NoError(t, os.WriteFile(page, []byte("old policy"), 0o600))

	router := routegroup.New(http.NewServeMux())
	addFileServer(router, fstest.MapFS{}, webRoot, "test-version", "https://remark.example.com", "")
	ts := httptest.NewServer(router)
	defer ts.Close()
	url := ts.URL + "/web/privacy.html"

	code, body, etag := revalidate(t, url, "")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "old policy", body)
	require.NotEmpty(t, etag)

	code, _, _ = revalidate(t, url, etag)
	assert.Equal(t, http.StatusNotModified, code, "an unchanged file keeps its validator")

	// same length, so nothing but the bytes tells the two apart
	require.NoError(t, os.WriteFile(page, []byte("new policy"), 0o600))

	code, body, newEtag := revalidate(t, url, etag)
	assert.Equal(t, http.StatusOK, code, "the browser holds the old page and has to get the new one")
	assert.Equal(t, "new policy", body)
	assert.NotEqual(t, etag, newEtag)
}

// TestWebFingerprint covers the name resolution: the fingerprint has to come from the file the
// file server answers with, which for the directory is its index page and for a legacy .js name
// the .mjs sibling, and there is none where no file answers
func TestWebFingerprint(t *testing.T) {
	files := webFiles{
		frontend: fstest.MapFS{
			"index.html":    {Data: []byte("index")},
			"remark.mjs":    {Data: []byte("bundle")},
			"sub/page.html": {Data: []byte("nested")},
		},
		embedded: fstest.MapFS{"privacy.html": {Data: []byte("policy")}},
	}
	fingerprint := webFingerprint(files)
	of := func(p string) string {
		return fingerprint(httptest.NewRequest(http.MethodGet, p, http.NoBody))
	}

	assert.Equal(t, of("/web/index.html"), of("/web/"), "the directory is answered with its index page")
	assert.Equal(t, of("/web/remark.mjs"), of("/web/remark.js"), "a legacy .js name is answered with the .mjs")
	assert.NotEmpty(t, of("/web/privacy.html"), "the embedded assets are fingerprinted too")
	assert.NotEqual(t, of("/web/index.html"), of("/web/remark.mjs"))
	assert.Empty(t, of("/web/missing.html"))
	assert.Empty(t, of("/web/sub/"), "a directory other than the root has no file to fingerprint")
	assert.Empty(t, of("/web/remark.mjs/"), "the handler answers a trailing slash with 404, whatever the name")
	assert.Equal(t, of("/web/index.html"), of("/web/sub/../index.html"), "the path is cleaned the way the file server cleans it")
}

// TestRest_FileServerEtagFollowsTheEmbeddedBuild is the same case for the frontend embedded in the
// binary: two builds stamped with the same version, as a local or a dev build is, serve different
// bytes under one name. Embedded files carry no modification time and builds of one commit have
// equal sizes, so only the bytes can tell them apart.
func TestRest_FileServerEtagFollowsTheEmbeddedBuild(t *testing.T) {
	etagFor := func(bundle string) string {
		router := routegroup.New(http.NewServeMux())
		addFileServer(router, fstest.MapFS{"remark.mjs": {Data: []byte(bundle)}}, filepath.Join(t.TempDir(), "absent"),
			"test-version", "https://remark.example.com", "")
		ts := httptest.NewServer(router)
		defer ts.Close()
		code, _, etag := revalidate(t, ts.URL+"/web/remark.mjs", "")
		require.Equal(t, http.StatusOK, code)
		require.NotEmpty(t, etag)
		return etag
	}

	assert.NotEqual(t, etagFor("build one"), etagFor("build two"))
}

// TestRest_FileServerContentTypeIgnoresTheOSTable covers an operating system whose table maps the
// served extensions to text/plain, as a Windows registry entry or a line in /etc/mime.types can. The
// file server asks that table, and a widget bundle served as text/plain does not run. The table is
// process-global, so it is poisoned for this test only and restored after; nothing in this package
// runs in parallel.
func TestRest_FileServerContentTypeIgnoresTheOSTable(t *testing.T) {
	for _, ext := range []string{".html", ".js", ".mjs", ".css", ".svg"} {
		orig := mime.TypeByExtension(ext)
		require.NotEmpty(t, orig, "go's own table has %s, so there is a type to restore", ext)
		require.NoError(t, mime.AddExtensionType(ext, "text/plain"))
		t.Cleanup(func() { require.NoError(t, mime.AddExtensionType(ext, orig)) })
		require.Equal(t, "text/plain; charset=utf-8", mime.TypeByExtension(ext), "the table has to be poisoned for %s, or the test proves nothing", ext)
	}

	frontend := fstest.MapFS{
		"remark.mjs":  {Data: []byte("export {}")},
		"embed.mjs":   {Data: []byte("(function(){})()")},
		"remark.css":  {Data: []byte("body{}")},
		"iframe.html": {Data: []byte("<!doctype html>")},
	}
	router := routegroup.New(http.NewServeMux())
	addFileServer(router, frontend, filepath.Join(t.TempDir(), "absent"), "test-version", "https://remark.example.com", "")

	tbl := []struct {
		name, path, want string
	}{
		{name: "module bundle from the frontend build", path: "/web/remark.mjs", want: "text/javascript; charset=utf-8"},
		{name: "legacy js name served from the mjs file", path: "/web/embed.js", want: "text/javascript; charset=utf-8"},
		{name: "hand-written script from the embedded assets", path: "/web/counter.mjs", want: "text/javascript; charset=utf-8"},
		{name: "stylesheet", path: "/web/remark.css", want: "text/css; charset=utf-8"},
		{name: "widget document", path: "/web/iframe.html", want: "text/html; charset=utf-8"},
		{name: "embedded page", path: "/web/privacy.html", want: "text/html; charset=utf-8"},
		{name: "demo page at the root", path: "/web/", want: "text/html; charset=utf-8"},
		{name: "embedded svg icon", path: "/web/github-dark.svg", want: "image/svg+xml"},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, http.NoBody))
			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.want, w.Header().Get("Content-Type"))
		})
	}

	t.Run("a missing script is answered as a plain error", func(t *testing.T) {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/web/absent.mjs", http.NoBody))
		require.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	})
}
