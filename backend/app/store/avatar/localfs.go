package avatar

//go:generate sh -c "mockery -inpkg -name AvatarStore -print > /tmp/mock.tmp && mv /tmp/mock.tmp avatar_store_mock.go"

import (
	"fmt"
	"hash/crc64"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/store"
)

// LocalFS implements AvatarStore for local file system
type LocalFS struct {
	storePath   string
	resizeLimit int
	ctcTable    *crc64.Table
	once        sync.Once
}

// NewLocalFS makes file-system avatar store
func NewLocalFS(storePath string, resizeLimit int) *LocalFS {
	return &LocalFS{storePath: storePath, resizeLimit: resizeLimit}
}

// Put avatar for userID to file and return avatar's file name (base), like 12345678.image
func (fs *LocalFS) Put(userID string, reader io.Reader) (avatar string, err error) {
	id := store.EncodeID(userID)
	location := fs.location(id) // location adds partition to path

	if _, err = os.Stat(location); os.IsNotExist(err) {
		if e := os.Mkdir(location, 0700); e != nil {
			return "", errors.Wrapf(e, "failed to mkdir avatar location %s", location)
		}
	}

	avFile := path.Join(location, id+ImgSfx)
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
	return id + ImgSfx, nil
}

// Get avatar reader for avatar id.image
func (fs *LocalFS) Get(avatar string) (reader io.ReadCloser, size int, err error) {
	location := fs.location(strings.TrimSuffix(avatar, ImgSfx))
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
func (fs *LocalFS) ID(avatar string) (id string) {
	location := fs.location(strings.TrimSuffix(avatar, ImgSfx))
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
func (fs *LocalFS) location(id string) string {
	fs.once.Do(func() { fs.ctcTable = crc64.MakeTable(crc64.ECMA) })
	checksum64 := crc64.Checksum([]byte(id), fs.ctcTable)
	partition := checksum64 % 100
	return path.Join(fs.storePath, fmt.Sprintf("%02d", partition))
}
