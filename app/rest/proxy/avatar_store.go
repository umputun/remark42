package proxy

import (
	"fmt"
	"hash/crc64"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/umputun/remark/app/store"
)

// AvatarStore defines interface to store and serve avatars
type AvatarStore interface {
	Put(userID string, reader io.Reader) (avatar string, err error)
	Get(avatar string) (reader io.ReadCloser, size int, err error)
}

// FSAvatarStore implements AvatarStore for local file system
type FSAvatarStore struct {
	storePath string
	ctcTable  *crc64.Table
	once      sync.Once
}

// NewFSAvatarStore makes file-system avatar store
func NewFSAvatarStore(storePath string) *FSAvatarStore {
	return &FSAvatarStore{storePath: storePath}
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
