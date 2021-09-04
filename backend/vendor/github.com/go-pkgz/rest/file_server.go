package rest

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// FileServer returns http.FileServer handler to serve static files from a http.FileSystem,
// prevents directory listing.
// - public defines base path of the url, i.e. for http://example.com/static/* it should be /static
// - local for the local path to the root of the served directory
// - notFound is the reader for the custom 404 html, can be nil for default
func FileServer(public, local string, notFound io.Reader) (http.Handler, error) {

	root, err := filepath.Abs(local)
	if err != nil {
		return nil, fmt.Errorf("can't get absolute path for %s: %w", local, err)
	}
	if _, err = os.Stat(root); os.IsNotExist(err) {
		return nil, fmt.Errorf("local path %s doesn't exist: %w", root, err)
	}

	fs := http.StripPrefix(public, http.FileServer(noDirListingFS{http.Dir(root), false}))
	return custom404Handler(fs, notFound)
}

// FileServerSPA returns FileServer as above, but instead of no-found returns /local/index.html
func FileServerSPA(public, local string, notFound io.Reader) (http.Handler, error) {

	root, err := filepath.Abs(local)
	if err != nil {
		return nil, fmt.Errorf("can't get absolute path for %s: %w", local, err)
	}
	if _, err = os.Stat(root); os.IsNotExist(err) {
		return nil, fmt.Errorf("local path %s doesn't exist: %w", root, err)
	}

	fs := http.StripPrefix(public, http.FileServer(noDirListingFS{http.Dir(root), true}))
	return custom404Handler(fs, notFound)
}

type noDirListingFS struct {
	fs  http.FileSystem
	spa bool
}

// Open file on FS, for directory enforce index.html and fail on a missing index
func (fs noDirListingFS) Open(name string) (http.File, error) {

	f, err := fs.fs.Open(name)
	if err != nil {
		if fs.spa {
			return fs.fs.Open("/index.html")
		}
		return nil, err
	}

	s, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if s.IsDir() {
		index := strings.TrimSuffix(name, "/") + "/index.html"
		if _, err := fs.fs.Open(index); err != nil {
			return nil, err
		}
	}
	return f, nil
}

// respWriter404 intercept Write to provide custom 404 response
type respWriter404 struct {
	http.ResponseWriter
	status int
	msg    []byte
}

func (w *respWriter404) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *respWriter404) Write(p []byte) (n int, err error) {
	if w.status != http.StatusNotFound || w.msg == nil {
		return w.ResponseWriter.Write(p)
	}
	_, err = w.ResponseWriter.Write(w.msg)
	return len(p), err
}

func custom404Handler(next http.Handler, notFound io.Reader) (http.Handler, error) {
	if notFound == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { next.ServeHTTP(w, r) }), nil
	}

	body, err := ioutil.ReadAll(notFound)
	if err != nil {
		return nil, err
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&respWriter404{ResponseWriter: w, msg: body}, r)
	}), nil
}
