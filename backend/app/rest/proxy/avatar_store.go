package proxy

//go:generate sh -c "mockery -inpkg -name AvatarStore -print > /tmp/mock.tmp && mv /tmp/mock.tmp avatar_store_mock.go"

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
	"strconv"
	"strings"
	"sync"

	// Initializing packages for supporting GIF and JPEG formats.
	_ "image/gif"
	_ "image/jpeg"

	"github.com/pkg/errors"
	"golang.org/x/image/draw"

	"github.com/umputun/remark/backend/app/store"
)

// AvatarStore defines interface to store and serve avatars
type AvatarStore interface {
	Put(userID string, reader io.Reader) (avatar string, err error)
	Get(avatar string) (reader io.ReadCloser, size int, err error)
	ID(avatar string) (id string)
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

	// Trying to resize avatar.
	if reader = resize(reader, fs.resizeLimit); reader == nil {
		return "", errors.New("avatar reader is nil")
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
		return nil, 0, errors.Wrapf(err, "can't load avatar %s, id", avatar)
	}
	if fi, e := fh.Stat(); e == nil {
		size = int(fi.Size())
	}
	return fh, size, nil
}

// ID returns a fingerprint of the avatar content.
func (fs *FSAvatarStore) ID(avatar string) (id string) {
	location := fs.location(strings.TrimSuffix(avatar, imgSfx))
	avFile := path.Join(location, avatar)
	fi, err := os.Stat(avFile)
	if err != nil {
		log.Printf("[DEBUG] can't get file info '%s', %s", avFile, err)
		return store.EncodeID(avatar)
	}
	return store.EncodeID(avatar + strconv.FormatInt(fi.ModTime().Unix(), 10))
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
func resize(reader io.Reader, limit int) io.Reader {
	if reader == nil {
		log.Print("[WARN] avatar resize(): reader is nil")
		return nil
	}
	if limit <= 0 {
		log.Print("[DEBUG] avatar resize(): limit should be greater than 0")
		return reader
	}

	var teeBuf bytes.Buffer
	tee := io.TeeReader(reader, &teeBuf)
	src, _, err := image.Decode(tee)
	if err != nil {
		log.Printf("[WARN] avatar resize(): can't decode avatar image, %s", err)
		return &teeBuf
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= limit && h <= limit || w <= 0 || h <= 0 {
		log.Print("[DEBUG] resizing image is smaller that the limit or has 0 size")
		return &teeBuf
	}
	newW, newH := w*limit/h, limit
	if w > h {
		newW, newH = limit, h*limit/w
	}
	m := image.NewRGBA(image.Rect(0, 0, newW, newH))
	// Slower than `draw.ApproxBiLinear.Scale()` but better quality.
	draw.BiLinear.Scale(m, m.Bounds(), src, src.Bounds(), draw.Src, nil)

	var out bytes.Buffer
	if err = png.Encode(&out, m); err != nil {
		log.Printf("[WARN] avatar resize(): can't encode resized avatar to PNG, %s", err)
		return &teeBuf
	}
	return &out
}
