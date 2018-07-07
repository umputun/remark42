package avatar

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"log"

	// Initializing packages for supporting GIF and JPEG formats.
	_ "image/gif"
	_ "image/jpeg"

	"golang.org/x/image/draw"
)

// ImgSfx for avatars
const ImgSfx = ".image"

// Store defines interface to store and and load avatars
type Store interface {
	Put(userID string, reader io.Reader) (avatar string, err error)
	Get(avatar string) (reader io.ReadCloser, size int, err error)
	ID(avatar string) (id string)
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
