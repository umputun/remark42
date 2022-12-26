package avatar

import (
	"fmt"
	"hash/crc64"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// LocalFS implements Store for local file system
type LocalFS struct {
	storePath string
	ctcTable  *crc64.Table
	once      sync.Once
}

// NewLocalFS makes file-system avatar store
func NewLocalFS(storePath string) *LocalFS {
	return &LocalFS{storePath: storePath}
}

// Put avatar for userID to file and return avatar's file name (base), like 12345678.image
// userID can be avatarID as well, in this case encoding just strip .image prefix
func (fs *LocalFS) Put(userID string, reader io.Reader) (avatar string, err error) {
	if reader == nil {
		return "", fmt.Errorf("empty reader")
	}
	id := encodeID(userID)
	location := fs.location(id) // location adds partition to path

	if e := os.MkdirAll(location, 0o750); e != nil {
		return "", fmt.Errorf("failed to mkdir avatar location %s: %w", location, e)
	}

	avFile := path.Join(location, id+imgSfx)
	fh, err := os.Create(avFile) //nolint
	if err != nil {
		return "", fmt.Errorf("can't create file %s: %w", avFile, err)
	}
	defer func() { //nolint
		if e := fh.Close(); e != nil {
			err = fmt.Errorf("can't close avatar file %s: %w", avFile, e)
		}
	}()

	if _, err = io.Copy(fh, reader); err != nil {
		return "", fmt.Errorf("can't save file %s: %w", avFile, err)
	}
	return id + imgSfx, nil
}

// Get avatar reader for avatar id.image
func (fs *LocalFS) Get(avatar string) (reader io.ReadCloser, size int, err error) {
	location := fs.location(strings.TrimSuffix(avatar, imgSfx))
	fh, err := os.Open(path.Join(location, avatar)) //nolint
	if err != nil {
		return nil, 0, fmt.Errorf("can't load avatar %s, id: %w", avatar, err)
	}
	if fi, e := fh.Stat(); e == nil {
		size = int(fi.Size())
	}
	return fh, size, nil
}

// ID returns a fingerprint of the avatar content.
func (fs *LocalFS) ID(avatar string) (id string) {
	location := fs.location(strings.TrimSuffix(avatar, imgSfx))
	avFile := path.Join(location, avatar)
	fi, err := os.Stat(avFile)
	if err != nil {
		return encodeID(avatar)
	}
	return encodeID(avatar + strconv.FormatInt(fi.ModTime().Unix(), 10))
}

// Remove avatar file
func (fs *LocalFS) Remove(avatar string) error {
	location := fs.location(strings.TrimSuffix(avatar, imgSfx))
	avFile := path.Join(location, avatar)
	return os.Remove(avFile)
}

// List all avatars (ids) on local file system
// note: id includes .image suffix
func (fs *LocalFS) List() (ids []string, err error) {
	err = filepath.Walk(fs.storePath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), imgSfx) {
				ids = append(ids, info.Name())
			}
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("can't list avatars: %w", err)
	}
	return ids, nil
}

// Close LocalFS does nothing but satisfies interface
func (fs *LocalFS) Close() error {
	return nil
}

func (fs *LocalFS) String() string {
	return fmt.Sprintf("localfs, path=%s", fs.storePath)
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
