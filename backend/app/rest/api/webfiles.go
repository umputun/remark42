package api

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
)

// webFiles serves /web from two sources: a name present in the frontend build is served from there,
// and any other name from the assets embedded in the binary.
type webFiles struct {
	frontend fs.FS
	embedded fs.FS
}

// Open resolves the name against both sources, and answers a missing .js with the .mjs sibling.
// The build stopped emitting .js while integrations still request it; the bundles carry no module
// syntax, so the same bytes serve both names.
func (w webFiles) Open(name string) (fs.File, error) {
	// fs.ValidPath alone is not enough: it accepts names the operating system rejects, NUL among
	// them, and os.DirFS turns those into fs.ErrInvalid, which renders as 500 rather than 404
	if _, err := filepath.Localize(name); err != nil || !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	f, err := w.open(name)
	if err == nil {
		return f, nil
	}
	if !errors.Is(err, fs.ErrNotExist) || !strings.HasSuffix(name, ".js") {
		return nil, err
	}

	alias, aliasErr := w.open(strings.TrimSuffix(name, ".js") + ".mjs")
	if aliasErr == nil {
		return alias, nil
	}
	if !errors.Is(aliasErr, fs.ErrNotExist) {
		return nil, aliasErr
	}
	return nil, err
}

// open looks the name up in the frontend build first. Only a missing file falls through to the
// embedded assets; every other error is returned so an unreadable file keeps reporting as one
// rather than being replaced by the embedded copy or reported as missing.
func (w webFiles) open(name string) (fs.File, error) {
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

// remarkURLPlaceholder is what the frontend build carries wherever the instance URL belongs. The
// bundler cannot know that URL, so it emits this marker and every distribution fills it in: the
// docker image rewrites the files under the web root at container start, and the binary, which
// serves the build embedded in itself and has nothing to rewrite, does it here.
const remarkURLPlaceholder = "{% REMARK_URL %}"

// templatedFS fills the instance URL into the files carrying the placeholder. Without it the
// binary serves whatever the build baked in, which is a host no visitor can reach, and the widget
// falls back to it whenever a page omits remark_config.host.
type templatedFS struct {
	fs        fs.FS
	remarkURL string
}

// Open substitutes in the file types the frontend templates, and hands everything else through
// untouched so images and stylesheets keep streaming from their original source
func (t templatedFS) Open(name string) (fs.File, error) {
	f, err := t.fs.Open(name)
	if err != nil || !templatedName(name) {
		return f, err
	}

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return f, err
	}

	body, err := io.ReadAll(f)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return nil, err
	}

	body = bytes.ReplaceAll(body, []byte(remarkURLPlaceholder), []byte(t.remarkURL))
	return &memFile{Reader: bytes.NewReader(body), info: sizedInfo{FileInfo: info, size: int64(len(body))}}, nil
}

// templatedName reports whether the frontend templates this file type. It mirrors the set the
// docker image rewrites, so both distributions substitute in the same files
func templatedName(name string) bool {
	switch filepath.Ext(name) {
	case ".html", ".js", ".mjs":
		return true
	}
	return false
}

// memFile is a substituted file held in memory. The file server needs a seeker to answer range
// requests and to sniff a content type, which a substituted body no longer has on disk
type memFile struct {
	*bytes.Reader
	info fs.FileInfo
}

func (f *memFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *memFile) Close() error               { return nil }

// sizedInfo reports the length after substitution. The file server writes Content-Length from it,
// so reporting the length on disk would truncate the response or leave the client waiting
type sizedInfo struct {
	fs.FileInfo
	size int64
}

func (i sizedInfo) Size() int64 { return i.size }
