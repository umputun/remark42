package templates

import (
	"io/ioutil"
	"net/http"
	"path/filepath"

	log "github.com/go-pkgz/lgr"
	"github.com/rakyll/statik/fs"
)

// FS stores link to statikFS if it exists
type FS struct {
	statik http.FileSystem
}

// FileReader describes methods of filesystem
type FileReader interface {
	ReadFile(path string) ([]byte, error)
}

// NewFS returns new FS instance, which will read from statik if it's available and from fs otherwise
func NewFS() *FS {
	f := &FS{}
	if statikFS, err := fs.NewWithNamespace("templates"); err == nil {
		log.Printf("[INFO] templates will be read from statik")
		f.statik = statikFS
	}
	return f
}

// ReadFile depends on statik achieve exists
func (f *FS) ReadFile(path string) ([]byte, error) {
	if f.statik != nil {
		return fs.ReadFile(f.statik, filepath.Join("/", path)) //nolint:gocritic // root folder is a requirement for statik
	}
	return ioutil.ReadFile(filepath.Clean(path))
}
