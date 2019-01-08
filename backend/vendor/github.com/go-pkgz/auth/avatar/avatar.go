// Package avatar implements avatart proxy for oauth and
// defines store interface and implements local (fs), gridfs (mongo) and boltdb stores.
package avatar

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-pkgz/lgr"
	"github.com/go-pkgz/rest"
	"github.com/pkg/errors"
	"golang.org/x/image/draw"

	"github.com/go-pkgz/auth/token"
)

// Proxy provides http handler for avatars from avatar.Store
// On user login token will call Put and it will retrieve and save picture locally.
type Proxy struct {
	lgr.L
	Store       Store
	RoutePath   string
	URL         string
	ResizeLimit int
}

// Put stores retrieved avatar to avatar.Store. Gets image from user info. Returns proxied url
func (p *Proxy) Put(u token.User) (avatarURL string, err error) {

	// no picture for user, try default avatar
	if u.Picture == "" {
		return "", errors.Errorf("no picture for %s", u.ID)
	}

	// load avatar from remote location
	client := http.Client{Timeout: 10 * time.Second}
	var resp *http.Response
	err = retry(5, time.Second, func() error {
		var e error
		resp, e = client.Get(u.Picture)
		return e
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch avatar from the orig")
	}

	defer func() {
		if e := resp.Body.Close(); e != nil {
			p.Logf("[WARN] can't close response body, %s", e)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("failed to get avatar from the orig, status %s", resp.Status)
	}

	avatarID, err := p.Store.Put(u.ID, p.resize(resp.Body, p.ResizeLimit)) // put returns avatar base name, like 123456.image
	if err != nil {
		return "", err
	}

	p.Logf("[DEBUG] saved avatar from %s to %s, user %q", u.Picture, avatarID, u.Name)
	return p.URL + p.RoutePath + "/" + avatarID, nil
}

// Handler returns token routes for given provider
func (p *Proxy) Handler(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
	elems := strings.Split(r.URL.Path, "/")
	avatarID := elems[len(elems)-1]

	// enforce client-side caching
	etag := `"` + p.Store.ID(avatarID) + `"`
	w.Header().Set("Etag", etag)
	w.Header().Set("Cache-Control", "max-age=604800") // 7 days
	if match := r.Header.Get("If-None-Match"); match != "" {
		if strings.Contains(match, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	avReader, size, err := p.Store.Get(avatarID)
	if err != nil {

		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't load avatar")
		return
	}

	defer func() {
		if e := avReader.Close(); e != nil {
			p.Logf("[WARN] can't close avatar reader for %s, %s", avatarID, e)
		}
	}()

	w.Header().Set("Content-Type", "image/*")
	w.Header().Set("Content-Length", strconv.Itoa(size))
	w.WriteHeader(http.StatusOK)
	if _, err = io.Copy(w, avReader); err != nil {
		p.Logf("[WARN] can't send response to %s, %s", r.RemoteAddr, err)
	}
}

// resize an image of supported format (PNG, JPG, GIF) to the size of "limit" px of the biggest side
// (width or height) preserving aspect ratio.
// Returns original reader if resizing is not needed or failed.
func (p *Proxy) resize(reader io.Reader, limit int) io.Reader {
	if reader == nil {
		p.Logf("[WARN] avatar resize(): reader is nil")
		return nil
	}
	if limit <= 0 {
		p.Logf("[DEBUG] avatar resize(): limit should be greater than 0")
		return reader
	}

	var teeBuf bytes.Buffer
	tee := io.TeeReader(reader, &teeBuf)
	src, _, err := image.Decode(tee)
	if err != nil {
		p.Logf("[WARN] avatar resize(): can't decode avatar image, %s", err)
		return &teeBuf
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= limit && h <= limit || w <= 0 || h <= 0 {
		p.Logf("[DEBUG] resizing image is smaller that the limit or has 0 size")
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
		p.Logf("[WARN] avatar resize(): can't encode resized avatar to PNG, %s", err)
		return &teeBuf
	}
	return &out
}
func retry(retries int, delay time.Duration, fn func() error) (err error) {
	for i := 0; i < retries; i++ {
		if err = fn(); err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	return errors.Wrap(err, "retry failed")
}
