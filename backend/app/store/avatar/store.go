// Package avatar defines store interface and implements local (fs) and gridfs (mongo) stores.
package avatar

//go:generate sh -c "mockery -inpkg -name Store -print > /tmp/mock.tmp && mv /tmp/mock.tmp store_mock.go"

import (
	"bytes"
	"image"
	"strings"

	// Initializing packages for supporting GIF and JPEG formats.
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"log"
	"regexp"

	"github.com/umputun/remark/backend/app/store"
	"golang.org/x/image/draw"
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
	Close() error
}

// Migrate avatars between stores
func Migrate(dst Store, src Store) (int, error) {
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

// resize an image of supported format (PNG, JPG, GIF) to the size of "limit" px of the biggest side
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

// encodeID converts string to encoded id unless already encoded and valid avatar id (with .image) passed
func encodeID(val string) string {
	if reValidAvatarID.MatchString(val) {
		return strings.TrimSuffix(val, imgSfx) // already encoded, strip .image
	}
	return store.EncodeID(val)
}
