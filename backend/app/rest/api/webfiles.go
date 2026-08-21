package api

import (
	"errors"
	"io/fs"
	"path/filepath"
)

// webFiles serves /web from two sources: a name present in the frontend build is served from there,
// and any other name from the assets embedded in the binary.
type webFiles struct {
	frontend fs.FS
	embedded fs.FS
}

// Open looks the name up in the frontend build first. Only a missing file falls through to the
// embedded assets; every other error is returned so an unreadable file keeps reporting as one
// rather than being replaced by the embedded copy or reported as missing.
func (w webFiles) Open(name string) (fs.File, error) {
	// fs.ValidPath alone is not enough: it accepts names the operating system rejects, NUL among
	// them, and os.DirFS turns those into fs.ErrInvalid, which renders as 500 rather than 404
	if _, err := filepath.Localize(name); err != nil || !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	f, err := w.frontend.Open(name)
	if err == nil {
		return f, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if name == "." {
		// the embedded set is a flat list of files; only the frontend build answers for the
		// directory itself, so a missing web root reports as missing rather than listing them
		return nil, err
	}
	return w.embedded.Open(name)
}

// emptyFS stands in for a frontend source that could not be opened, so a misconfigured one serves
// nothing instead of panicking or serving the build at paths it does not belong at
type emptyFS struct{}

func (emptyFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}
