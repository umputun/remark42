package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/evanw/esbuild/pkg/api"
)

// built holds the latest development build that compiled. Nothing is written to disk, so a
// production build made meanwhile and the server's files never meet, and a change that does not
// compile leaves the last good build being served.
type built struct {
	mu    sync.RWMutex
	files map[string][]byte

	first sync.Once
	ready chan struct{} // closed once the first build has finished, whichever way
}

func newBuilt() *built { return &built{ready: make(chan struct{})} }

// take replaces the build with a newer one; failed says the build produced nothing to take
func (b *built) take(files map[string][]byte, failed bool) {
	if !failed {
		b.mu.Lock()
		b.files = files
		b.mu.Unlock()
	}
	b.first.Do(func() { close(b.ready) })
}

// file returns a built file once there has been a build to look in
func (b *built) file(name string) ([]byte, bool) {
	<-b.ready
	b.mu.RLock()
	defer b.mu.RUnlock()
	content, ok := b.files[name]
	return content, ok
}

// serve keeps a development build current and serves it the way the backend serves a production
// one: under /web/, with the hand-written pages and images of assetsDir behind it, and with /api
// and /auth answered by backend. A page has to be reloaded to pick a rebuild up.
func serve(appDir, assetsDir, addr string, backend *url.URL) error {
	latest := newBuilt()

	opts := options(appDir, true, nil)
	opts.LogLevel = api.LogLevelInfo
	opts.Plugins = append(opts.Plugins, api.Plugin{
		Name: "serve",
		Setup: func(build api.PluginBuild) {
			build.OnEnd(func(result *api.BuildResult) (api.OnEndResult, error) {
				if len(result.Errors) > 0 {
					latest.take(nil, true)
					return api.OnEndResult{}, nil
				}
				files, err := outputs(opts.Outdir, result.OutputFiles)
				latest.take(files, err != nil)
				return api.OnEndResult{}, err
			})
		},
	})

	ctx, ctxErr := api.Context(opts)
	if ctxErr != nil {
		return fmt.Errorf("%d error(s) in the build options", len(ctxErr.Errors))
	}
	defer ctx.Dispose()

	// the first build starts here, and it and every rebuild print their own errors
	if err := ctx.Watch(api.WatchOptions{}); err != nil {
		return err
	}

	log.Printf("widget on http://%s/web/, API from %s", addr, backend)
	srv := &http.Server{Addr: addr, Handler: devHandler(latest, assetsDir, backend), ReadHeaderTimeout: 10 * time.Second}
	return srv.ListenAndServe()
}

// devHandler routes the development server's requests
func devHandler(latest *built, assetsDir string, backend *url.URL) http.Handler {
	proxy := &httputil.ReverseProxy{Rewrite: func(r *httputil.ProxyRequest) {
		// SetURL also sends the backend its own host name, which a backend behind a name-based
		// proxy needs to be found at all
		r.SetURL(backend)
	}}

	// the build comes first, as it does in the backend's file server, and the assets the bundler
	// does not build are read straight from where they live
	assets := http.FileServer(http.Dir(assetsDir))
	web := http.StripPrefix("/web", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// a page on another origin may load the widget from here, and a module script is a
		// cross-origin read
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// revalidate every time, so a reload always runs the latest build
		w.Header().Set("Cache-Control", "no-cache")

		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if content, ok := latest.file(name); ok {
			// with no modification time there is nothing for a reload to be told has not changed:
			// a time would be whole seconds, and two builds fit in one
			http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(content))
			return
		}
		assets.ServeHTTP(w, r)
	}))

	mux := http.NewServeMux()
	mux.Handle("/api/", proxy)
	mux.Handle("/auth/", proxy)
	mux.Handle("/web/", web)
	mux.Handle("/{$}", http.RedirectHandler("/web/", http.StatusFound))
	return mux
}

// readEnvFile reads KEY=VALUE lines the way node's own reader does: a value may be quoted, with
// either quote or with backticks, and what follows a # after an unquoted value is a comment. A missing file is the usual case and
// reads as empty.
func readEnvFile(file string) (map[string]string, error) {
	f, err := os.Open(file) //nolint:gosec // the path is the app directory given on the command line
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // read-only

	env := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(strings.TrimPrefix(line, "export "), "=")
		if !ok {
			continue
		}
		env[strings.TrimSpace(key)] = envValue(strings.TrimSpace(value))
	}
	return env, scanner.Err()
}

func envValue(value string) string {
	if value != "" && strings.ContainsRune("\"'`", rune(value[0])) {
		if end := strings.IndexByte(value[1:], value[0]); end >= 0 {
			return value[1 : end+1]
		}
	}
	if comment := strings.IndexByte(value, '#'); comment >= 0 {
		value = value[:comment]
	}
	return strings.TrimSpace(value)
}
