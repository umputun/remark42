package image

import (
	"context"
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
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
)

// FileSystem provides image Store for local files. Saves and loads files from Location, restricts max size.
type FileSystem struct {
	Location   string
	Staging    string
	Partitions int

	crc struct {
		*crc64.Table
		sync.Once
		mask    string
		divider uint64
	}
}

// Save saves image with given id to local FS, staging directory.
// Files partitioned across multiple subdirectories, and the final path includes part, i.e. /location/user1/03/123-4567
func (f *FileSystem) Save(id string, img []byte) error {
	dst := f.location(f.Staging, id)

	if err := os.MkdirAll(path.Dir(dst), 0o700); err != nil {
		return errors.Wrap(err, "can't make image directory")
	}

	if err := os.WriteFile(dst, img, 0o600); err != nil {
		return errors.Wrapf(err, "can't write image file with id %s", id)
	}

	log.Printf("[DEBUG] file %s saved for image %s, size=%d", dst, id, len(img))
	return nil
}

// Commit file stored in staging location by moving it to permanent location
func (f *FileSystem) Commit(id string) error {
	log.Printf("[DEBUG] Commit image %s", id)
	stagingImage, permImage := f.location(f.Staging, id), f.location(f.Location, id)

	if err := os.MkdirAll(path.Dir(permImage), 0o700); err != nil {
		return errors.Wrap(err, "can't make image directory")
	}

	err := os.Rename(stagingImage, permImage)
	return errors.Wrapf(err, "failed to commit image %s", id)
}

// ResetCleanupTimer resets cleanup timer for the image
func (f *FileSystem) ResetCleanupTimer(id string) error {
	file := f.location(f.Staging, id)
	_, err := os.Stat(file)
	if err != nil {
		return errors.Wrapf(err, "can't get image stats for %s", id)
	}
	// we don't need to update access time (second arg),
	// but reading it is platform-dependent and looks different on darwin and linux,
	// so it's easier to update it as well
	err = os.Chtimes(file, time.Now(), time.Now())
	return errors.Wrapf(err, "problem updating %s modification time", file)
}

// Load image from FS. Uses id to get partition subdirectory.
func (f *FileSystem) Load(id string) ([]byte, error) {

	// get image file by id. first try permanent location and if not found - staging
	img := func(id string) (file string, err error) {
		file = f.location(f.Location, id)
		_, err = os.Stat(file)
		if err != nil {
			file = f.location(f.Staging, id)
			_, err = os.Stat(file)
		}
		return file, errors.Wrapf(err, "can't get image stats for %s", id)
	}

	imgFile, err := img(id)
	if err != nil {
		return nil, errors.Wrapf(err, "can't get image file for %s", id)
	}

	fh, err := os.Open(imgFile) //nolint:gosec // we open file from known location
	if err != nil {
		return nil, errors.Wrapf(err, "can't load image %s", id)
	}
	return io.ReadAll(fh)
}

// Cleanup runs scan of staging and removes old files based on ttl
func (f *FileSystem) Cleanup(_ context.Context, ttl time.Duration) error {

	if _, err := os.Stat(f.Staging); os.IsNotExist(err) {
		return nil
	}

	// we can ignore context as on local FS remove is relatively fast operation
	err := filepath.Walk(f.Staging, func(fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		age := time.Since(info.ModTime())
		if age > (ttl + 100*time.Millisecond) { // delay cleanup triggering to allow commit
			log.Printf("[INFO] remove staging image %s, age %v", fpath, age)
			rmErr := os.Remove(fpath)
			_ = os.Remove(path.Dir(fpath)) // try to remove directory
			return rmErr
		}
		return nil
	})
	return errors.Wrap(err, "failed to cleanup images")
}

// Info returns meta information about storage
func (f *FileSystem) Info() (StoreInfo, error) {
	if _, err := os.Stat(f.Staging); os.IsNotExist(err) {
		return StoreInfo{}, nil
	}

	var ts time.Time
	err := filepath.Walk(f.Staging, func(fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		created := info.ModTime()
		if ts.IsZero() || created.Before(ts) {
			ts = created
		}
		return nil
	})
	if err != nil {
		return StoreInfo{}, errors.Wrapf(err, "problem retrieving first timestamp from staging images on fs")
	}
	return StoreInfo{FirstStagingImageTS: ts}, nil
}

// location gets full path for id by adding partition to the final path in order to keep files in different subdirectories
// and avoid too many files in a single place.
// the end result is a full path like this - /tmp/images/user1/92/xxx-yyy.png.
// Number of partitions defined by FileSystem.Partitions
func (f *FileSystem) location(base, id string) string {

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
