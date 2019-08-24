package image

import (
	"context"
	"fmt"
	"hash/crc64"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
)

// FileSystem provides image Store for local files. Saves and loads files from Location, restricts max size.
type FileSystem struct {
	Location   string
	Staging    string
	MaxSize    int
	Partitions int
	MaxHeight  int
	MaxWidth   int

	crc struct {
		*crc64.Table
		sync.Once
		mask    string
		divider uint64
	}
}

// Save data from reader for given file name to local FS, staging directory. Returns id as user/uuid.ext
// Files partitioned across multiple subdirectories and the final path includes part, i.e. /location/user1/03/123-4567.png
func (f *FileSystem) Save(fileName string, userID string, r io.Reader) (id string, err error) {

	lr := io.LimitReader(r, int64(f.MaxSize)+1)
	data, err := ioutil.ReadAll(lr)
	if err != nil {
		return "", errors.Wrapf(err, "can't read source data for image %s", fileName)
	}
	if len(data) > f.MaxSize {
		return "", errors.Errorf("file %s is too large (limit=%d)", fileName, f.MaxSize)
	}

	// read header first, needs it to check if data is valid png/gif/jpeg
	if !isValidImage(data[:512]) {
		return "", errors.Errorf("file %s is not in allowed format", fileName)
	}

	data, resized := resize(data, f.MaxWidth, f.MaxHeight)

	id = path.Join(userID, guid()) + filepath.Ext(fileName) // make id as user/uuid.ext
	dst := f.location(f.Staging, id)
	if resized { // resized also converted to png
		id = strings.TrimSuffix(id, filepath.Ext(id)) + ".png"
		dst = f.location(f.Staging, id)
	}

	if err = os.MkdirAll(path.Dir(dst), 0700); err != nil {
		return "", errors.Wrap(err, "can't make image directory")
	}

	if err = ioutil.WriteFile(dst, data, 0600); err != nil {
		return "", errors.Wrapf(err, "can't write image file %s", dst)
	}

	log.Printf("[DEBUG] file %s saved for image %s, size=%d", dst, fileName, len(data))
	return id, nil
}

// Commit file stored in staging location by moving it to permanent location
func (f *FileSystem) Commit(id string) error {
	log.Printf("[DEBUG] commit image %s", id)
	stagingImage, permImage := f.location(f.Staging, id), f.location(f.Location, id)

	if err := os.MkdirAll(path.Dir(permImage), 0700); err != nil {
		return errors.Wrap(err, "can't make image directory")
	}

	err := os.Rename(stagingImage, permImage)
	return errors.Wrapf(err, "failed to commit image %s", id)
}

// Load image from FS. Uses id to get partition subdirectory.
// returns ReadCloser and caller should call close after processing completed.
func (f *FileSystem) Load(id string) (io.ReadCloser, int64, error) {

	// get image file by id. first try permanent location and if not found - staging
	img := func(id string) (file string, st os.FileInfo, err error) {
		file = f.location(f.Location, id)
		st, err = os.Stat(file)
		if err != nil {
			file = f.location(f.Staging, id)
			st, err = os.Stat(file)
		}
		return file, st, errors.Wrapf(err, "can't get image stats for %s", id)
	}

	imgFile, st, err := img(id)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "can't get image file for %s", id)
	}

	fh, err := os.Open(imgFile)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "can't load image %s", id)
	}
	return fh, st.Size(), nil
}

// Cleanup runs scan of staging and removes old files based on ttl
func (f *FileSystem) Cleanup(ctx context.Context, ttl time.Duration) error {

	if _, err := os.Stat(f.Staging); os.IsNotExist(err) {
		return nil
	}

	err := filepath.Walk(f.Staging, func(fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		age := time.Since(info.ModTime())
		if age > ttl {
			log.Printf("[INFO] remove staging image %s, age %v", fpath, age)
			rmErr := os.Remove(fpath)
			_ = os.Remove(path.Dir(fpath)) // try to remove directory
			return rmErr
		}
		return nil
	})
	return errors.Wrap(err, "failed to cleanup images")
}

// SizeLimit returns max size of allowed image
func (f *FileSystem) SizeLimit() int {
	return f.MaxSize
}

// location gets full path for id by adding partition to the final path in order to keep files in different subdirectories
// and avoid too many files in a single place.
// the end result is a full path like this - /tmp/images/user1/92/xxx-yyy.png.
// Number of partitions defined by FileSystem.Partitions
func (f *FileSystem) location(base string, id string) string {

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
		return path.Join(base, user, file) // avoid partition directory if 0 Partitions
	}

	return path.Join(base, user, partition(id), file)
}
