package avatar

//go:generate sh -c "mockery -inpkg -name Store -print > /tmp/mock.tmp && mv /tmp/mock.tmp store_mock.go"

import (
	"crypto/sha1"
	_ "image/gif"  // initializing packages for supporting GIF
	_ "image/jpeg" // initializing packages for supporting JPEG.
	_ "image/png"  // initializing packages for supporting PNG.
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/go-pkgz/auth/token"
)

// imgSfx for avatars
const imgSfx = ".image"

var reValidAvatarID = regexp.MustCompile(`^[a-fA-F0-9]{40}\.image$`)

// Store defines interface to store and and load avatars
type Store interface {
	Put(userID string, reader io.Reader) (avatarID string, err error) // save avatar data from the reader and return base name
	Get(avatarID string) (reader io.ReadCloser, size int, err error)  // load avatar via reader
	ID(avatarID string) (id string)                                   // unique id of stored avatar's data
	Remove(avatarID string) error                                     // remove avatar data
	List() (ids []string, err error)                                  // list all avatar ids
	Close() error                                                     // close store
}

// Migrate avatars between stores
func Migrate(dst, src Store) (int, error) {
	ids, err := src.List()
	if err != nil {
		return 0, err
	}
	for _, id := range ids {
		srcReader, _, err := src.Get(id)
		if err != nil {
			log.Printf("[WARN] can't get reader for avatar %s", id)
			continue
		}
		if _, err = dst.Put(id, srcReader); err != nil {
			log.Printf("[WARN] can't put avatar %s", id)
		}
		if err = srcReader.Close(); err != nil {
			log.Printf("[WARN] failed to close avatar %s", id)
		}
	}
	return len(ids), nil
}

// encodeID hashes id to sha1. Skip encoding for already processed
func encodeID(id string) string {
	if reValidAvatarID.MatchString(id) {
		return strings.TrimSuffix(id, imgSfx) // already encoded, strip .image
	}
	return token.HashID(sha1.New(), id)
}
