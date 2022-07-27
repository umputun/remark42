package templates

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed static
var templateFS embed.FS

// Read reads either template from disk if it exists, or from embedded template
func Read(path string) ([]byte, error) {
	if _, err := os.Stat(filepath.Clean(path)); err == nil {
		return os.ReadFile(filepath.Clean(path))
	}
	// remove "static/" prefix from path
	var contentFS, err = fs.Sub(templateFS, "static")
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(contentFS, filepath.Clean(path))
}
