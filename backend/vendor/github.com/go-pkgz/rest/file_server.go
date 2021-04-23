package rest

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// FileServer returns http.FileServer handler to serve static files from a http.FileSystem,
// prevents directory listing.
// - public defines base path of the url, i.e. for http://example.com/static/* it should be /static
// - local for the local path to the root of the served directory
func FileServer(public, local string) (http.Handler, error) {

	root, err := filepath.Abs(local)
	if err != nil {
		return nil, fmt.Errorf("can't get absolute path for %s: %w", local, err)
	}
	if _, err = os.Stat(root); os.IsNotExist(err) {
		return nil, fmt.Errorf("local path %s doesn't exist: %w", root, err)
	}

	return http.StripPrefix(public, http.FileServer(noDirListingFS{http.Dir(root)})), nil
}

type noDirListingFS struct{ fs http.FileSystem }

// Open file on FS, for directory enforce index.html and fail on a missing index
func (fs noDirListingFS) Open(name string) (http.File, error) {
	f, err := fs.fs.Open(name)
	if err != nil {
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
