// Package image handles storing, resizing and retrieval of images
// Provides Interface with Save and Load and one implementation on top of local file system.
package image

import (
	"fmt"
	"hash/crc64"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	log "github.com/go-pkgz/lgr"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// Interface defines Save and Load methods
type Interface interface {
	Save(fileName string, userID string, r io.Reader) (id string, err error) // get name and reader and returns ID of stored image
	Load(id string) (io.ReadCloser, int64, error)                            // load image by ID. Caller has to close the reader.
}

// FileSystem provides image Interface for local files. Saves and loads files from Location, restricts max size
type FileSystem struct {
	Location   string
	MaxSize    int
	Partitions int

	crc struct {
		*crc64.Table
		sync.Once
		mask    string
		divider uint64
	}
}

// Save data from reader for given file name to local FS. Returns id as user/uuid.ext
// Files partitioned across multiple subdirectories and the final path includes part, i.e. /location/user1/03/123-4567.png
func (f *FileSystem) Save(fileName string, userID string, r io.Reader) (id string, err error) {

	uid, err := uuid.NewUUID()
	if err != nil {
		return "", errors.Wrap(err, "can't make image uuid")
	}

	id = path.Join(userID, uid.String()) + filepath.Ext(fileName) // make id as user/uuid.ext
	dst := f.location(id)

	if err = os.MkdirAll(path.Dir(dst), 0700); err != nil {
		return "", errors.Wrap(err, "can't make image directory")
	}

	fh, err := os.Create(dst)
	if err != nil {
		return "", errors.Wrapf(err, "can't make image file %s", dst)
	}
	lr := io.LimitReader(r, int64(f.MaxSize)+1)
	written, err := io.Copy(fh, lr)
	if err != nil {
		return "", errors.Wrapf(err, "can't write image file %s", dst)
	}
	if err = fh.Close(); err != nil {
		return "", errors.Wrapf(err, "can't close image file %s", dst)
	}
	if written > int64(f.MaxSize) {
		if err = os.Remove(dst); err != nil {
			log.Printf("[WARN] can't remove image file %s, %v", dst, err)
		}
		return "", errors.Errorf("file %s is too large", fileName)
	}
	log.Printf("[DEBUG] file %s saved for image %s", fh.Name(), fileName)
	return id, nil
}

// Load image from FS. Uses id to get partition subdirectory.
// returns ReadCloser and caller should call close after processing completed.
func (f *FileSystem) Load(id string) (io.ReadCloser, int64, error) {
	imgFile := f.location(id)

	st, err := os.Stat(imgFile)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "can't get image size for %s", id)
	}

	fh, err := os.Open(imgFile)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "can't load image %s", id)
	}
	return fh, st.Size(), nil
}

// location gets full path for id by adding partition to the final path in order to keep files in different subdirectories
// and avoid too many files in a single place.
// the end result is a full path like this - /tmp/images/user1/92/xxx-yyy.png.
// Number of partitions defined by FileSystem.Partitions
func (f *FileSystem) location(id string) string {

	partition := func(id string) string {
		f.crc.Do(func() {
			f.crc.Table = crc64.MakeTable(crc64.ECMA)
			p := int(math.Round(math.Log10(float64(f.Partitions))))
			f.crc.mask = "%0" + strconv.Itoa(p) + "d"
			f.crc.divider = uint64(math.Pow(10, float64(p)))
		})
		checksum64 := crc64.Checksum([]byte(id), f.crc.Table)
		partition := checksum64 % f.crc.divider
		return fmt.Sprintf(f.crc.mask, partition)
	}

	user, file := "unknown", id // default if no user in id
	if elems := strings.Split(id, "/"); len(elems) == 2 {
		user, file = elems[0], elems[1] // user in id
	}

	if f.Partitions == 0 {
		return path.Join(f.Location, user, file) // avoid partition directory if 0 Partitions
	}

	return path.Join(f.Location, user, partition(id), file)
}
