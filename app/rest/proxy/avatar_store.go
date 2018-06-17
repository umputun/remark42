package proxy

import (
	"bytes"
	"fmt"
	"hash/crc64"
	"image"
	"image/png"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	// Initializing packages for supporting GIF and JPEG formats.
	_ "image/gif"
	_ "image/jpeg"

	"github.com/pkg/errors"
	"golang.org/x/image/draw"

	"github.com/umputun/remark/app/store"
)

// AvatarStore defines interface to store and serve avatars
type AvatarStore interface {
	Put(userID string, reader io.Reader) (avatar string, err error)
	Get(avatar string) (reader io.ReadCloser, size int, err error)
}

// FSAvatarStore implements AvatarStore for local file system
type FSAvatarStore struct {
	storePath   string
	resizeLimit int
	ctcTable    *crc64.Table
	once        sync.Once
}

// NewFSAvatarStore makes file-system avatar store
func NewFSAvatarStore(storePath string, resizeLimit int) *FSAvatarStore {
	return &FSAvatarStore{storePath: storePath, resizeLimit: resizeLimit}
}

// Put avatar for userID to file and return avatar's file name (base), like 12345678.image
func (fs *FSAvatarStore) Put(userID string, reader io.Reader) (avatar string, err error) {
	id := store.EncodeID(userID)
	location := fs.location(id) // location adds partition to path

	if _, err = os.Stat(location); os.IsNotExist(err) {
		if e := os.Mkdir(location, 0700); e != nil {
			return "", errors.Wrapf(e, "failed to mkdir avatar location %s", location)
		}
	}

	avFile := path.Join(location, id+imgSfx)
	fh, err := os.Create(avFile)
	if err != nil {
		return "", errors.Wrapf(err, "can't create file %s", avFile)
	}
	defer func() {
		if e := fh.Close(); e != nil {
			log.Printf("[WARN] can't close avatar file %s, %s", avFile, e)
		}
	}()

	// Trying to resize avatar; using original image if failing.
	if fs.resizeLimit > 0 {
		reader, err = resize(reader, fs.resizeLimit)
		if err != nil {
			log.Printf("[WARN] eor on resize avatar for user %s, %s", userID, err)
		}
	}

	if _, err = io.Copy(fh, reader); err != nil {
		return "", errors.Wrapf(err, "can't save file %s", avFile)
	}
	return id + imgSfx, nil
}

// Get avatar reader for avatar id.image
func (fs *FSAvatarStore) Get(avatar string) (reader io.ReadCloser, size int, err error) {
	location := fs.location(strings.TrimSuffix(avatar, imgSfx))
	avFile := path.Join(location, avatar)
	fh, err := os.Open(avFile)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "can't load avatar %s, id")
	}
	if fi, e := fh.Stat(); e == nil {
		size = int(fi.Size())
	}
	return fh, size, nil
}

// get location (directory) for user id by adding partition to final path in order to keep files
// in different subdirectories and avoid too many files in a single place.
// the end result is a full path like this - /tmp/avatars.test/92
func (fs *FSAvatarStore) location(id string) string {
	fs.once.Do(func() { fs.ctcTable = crc64.MakeTable(crc64.ECMA) })
	checksum64 := crc64.Checksum([]byte(id), fs.ctcTable)
	partition := checksum64 % 100
	return path.Join(fs.storePath, fmt.Sprintf("%02d", partition))
}

// Resizes an image of supported format (PNG, JPG, GIF) to the size of "limit" px of the biggest side
// (width or height) preserving aspect ratio.
// Returns original reader if resizing is not needed or failed.
func resize(reader io.Reader, limit int) (io.Reader, error) {
	if limit <= 0 {
		return reader, errors.New("limit should be greater than 0")
	}

	var teeBuf bytes.Buffer
	tee := io.TeeReader(reader, &teeBuf)
	src, _, err := image.Decode(tee)
	if err != nil {
		return &teeBuf, errors.Wrap(err, "can't decode avatar image")
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= limit && h <= limit || w <= 0 || h <= 0 {
		return &teeBuf, errors.New("resizing image is smaller that the limit or has 0 size")
	}
	var newW, newH int
	if w > h {
		newW, newH = limit, h*limit/w
	} else {
		newW, newH = w*limit/h, limit
	}
	m := image.NewRGBA(image.Rect(0, 0, newW, newH))
	// Slower than `draw.ApproxBiLinear.Scale()` but better quality.
	draw.BiLinear.Scale(m, m.Bounds(), src, src.Bounds(), draw.Src, nil)

	var out bytes.Buffer
	if err = png.Encode(&out, m); err != nil {
		return &teeBuf, errors.Wrapf(err, "can't encode resized avatar to PNG")
	}

	return &out, nil
}
