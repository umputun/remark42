// Package avatar implements avatart proxy for oauth and
// defines store interface and implements local (fs), gridfs (mongo) and boltdb stores.
package avatar

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"image"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-pkgz/rest"
	"github.com/nullrocks/identicon"
	"github.com/pkg/errors"
	"golang.org/x/image/draw"

	"github.com/go-pkgz/auth/logger"
	"github.com/go-pkgz/auth/token"
)

// Proxy provides http handler for avatars from avatar.Store
// On user login token will call Put and it will retrieve and save picture locally.
type Proxy struct {
	logger.L
	Store       Store
	RoutePath   string
	URL         string
	ResizeLimit int
}

// Put stores retrieved avatar to avatar.Store. Gets image from user info. Returns proxied url
func (p *Proxy) Put(u token.User, client *http.Client) (avatarURL string, err error) {

	// no picture for user, try to generate identicon avatar
	if u.Picture == "" {
		b, e := GenerateAvatar(u.ID)
		if e != nil {
			return "", errors.Wrapf(e, "no picture for %s", u.ID)
		}
		// put returns avatar base name, like 123456.image
		avatarID, e := p.Store.Put(u.ID, p.resize(bytes.NewBuffer(b), p.ResizeLimit))
		if e != nil {
			return "", err
		}

		p.Logf("[DEBUG] saved identicon avatar to %s, user %q", avatarID, u.Name)
		return p.URL + p.RoutePath + "/" + avatarID, nil
	}

	body, err := p.load(u.Picture, client)
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch avatar from the orig")
	}

	defer func() {
		if e := body.Close(); e != nil {
			p.Logf("[WARN] can't close response body, %s", e)
		}
	}()

	avatarID, err := p.Store.Put(u.ID, p.resize(body, p.ResizeLimit)) // put returns avatar base name, like 123456.image
	if err != nil {
		return "", err
	}

	p.Logf("[DEBUG] saved avatar from %s to %s, user %q", u.Picture, avatarID, u.Name)
	return p.URL + p.RoutePath + "/" + avatarID, nil
}

// load avatar from remote url and return body. Caller has to close the reader
func (p *Proxy) load(url string, client *http.Client) (rc io.ReadCloser, err error) {
	// load avatar from remote location
	var resp *http.Response
	err = retry(5, time.Second, func() error {
		var e error
		resp, e = client.Get(url)
		return e
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch avatar from the orig")
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close() // caller won't close on error
		return nil, errors.Errorf("failed to get avatar from the orig, status %s", resp.Status)
	}

	return resp.Body, nil
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

		rest.SendErrorJSON(w, r, p.L, http.StatusBadRequest, err, "can't load avatar")
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

// GenerateAvatar for give user with identicon
func GenerateAvatar(user string) ([]byte, error) {

	iconGen, err := identicon.New("pkgz/auth", 5, 5)
	if err != nil {
		return nil, errors.Wrap(err, "can't create identicon service")
	}

	ii, err := iconGen.Draw(user) // generate an IdentIcon
	if err != nil {
		return nil, errors.Wrapf(err, "failed to draw avatar for %s", user)
	}

	buf := &bytes.Buffer{}
	err = ii.Png(300, buf)
	return buf.Bytes(), err
}

// GetGravatarURL returns url to gravatar picture for given email
func GetGravatarURL(email string) (res string, err error) {

	hash := md5.Sum([]byte(email))
	hexHash := hex.EncodeToString(hash[:])

	client := http.Client{Timeout: 1 * time.Second}
	res = "https://www.gravatar.com/avatar/" + hexHash + ".jpg"
	resp, err := client.Get(res + "?d=404&s=80")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.New(resp.Status)
	}
	return res, nil
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
