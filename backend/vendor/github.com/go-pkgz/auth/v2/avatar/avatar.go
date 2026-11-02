// Package avatar implements avatart proxy for oauth and
// defines store interface and implements local (fs), gridfs (mongo) and boltdb stores.
package avatar

import (
	"bytes"
	"crypto/md5" //nolint gosec
	"encoding/hex"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-pkgz/rest"
	"github.com/rrivera/identicon"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder so Discord-style .webp avatars validate

	"github.com/go-pkgz/auth/v2/logger"
	"github.com/go-pkgz/auth/v2/token"
)

// http.sniffLen is 512 bytes which is how much we need to read to detect content type
const sniffLen = 512

// maxAvatarFetchSize caps the byte length of any avatar accepted by load,
// PutContent, or resize — i.e. the upper bound on bytes that ever reach Store.Put.
const maxAvatarFetchSize = 10 << 20

// maxAvatarPixels caps cfg.Width*cfg.Height (from image.DecodeConfig) before any
// full image.Decode runs. Defends against decompression-bomb inputs.
const maxAvatarPixels = 16 * 1024 * 1024

// Proxy provides http handler for avatars from avatar.Store
// On user login token will call Put and it will retrieve and save picture locally.
type Proxy struct {
	logger.L
	Store       Store
	RoutePath   string
	URL         string
	ResizeLimit int
}

// Put fetches u.Picture, validates and optionally resizes the body, stores it via
// Store, and returns the proxied avatar URL. If u.Picture is empty, the fetch fails,
// or the upstream bytes are not a recognized image format within the configured
// dimension and size limits, Put silently falls back to a generated identicon and
// returns its proxied URL — the caller is not told upstream was rejected.
func (p *Proxy) Put(u token.User, client *http.Client) (avatarURL string, err error) {

	genIdenticon := func(userID string) (avatarURL string, err error) {
		b, e := GenerateAvatar(userID)
		if e != nil {
			return "", fmt.Errorf("no picture for %s: %w", userID, e)
		}
		// put returns avatar base name, like 123456.image
		avatarID, e := p.Store.Put(userID, p.resize(b, p.ResizeLimit))
		if e != nil {
			return "", e
		}

		p.Logf("[DEBUG] saved identicon avatar to %s, user %q", avatarID, u.Name)
		return p.URL + p.RoutePath + "/" + avatarID, nil
	}

	// no picture for user, try to generate identicon avatar
	if u.Picture == "" {
		return genIdenticon(u.ID)
	}

	body, err := p.load(u.Picture, client)
	if err != nil {
		p.Logf("[DEBUG] failed to fetch avatar from the orig %s, %v", redactAvatarURL(u.Picture), err)
		return genIdenticon(u.ID)
	}

	resized := p.resize(body, p.ResizeLimit)
	if resized == nil {
		// non-image upstream — refuse to store attacker-controlled bytes under
		// the user's avatar id and fall back to a generated identicon instead.
		p.Logf("[WARN] upstream avatar from %s is not a valid image, using identicon", redactAvatarURL(u.Picture))
		return genIdenticon(u.ID)
	}

	avatarID, err := p.Store.Put(u.ID, resized) // put returns avatar base name, like 123456.image
	if err != nil {
		return "", err
	}

	p.Logf("[DEBUG] saved avatar from %s to %s, user %q", redactAvatarURL(u.Picture), avatarID, u.Name)
	return p.URL + p.RoutePath + "/" + avatarID, nil
}

// PutContent stores already-fetched avatar bytes via the underlying Store and returns
// the proxied URL. It exists so providers that authenticate with credentials embedded
// in the upstream URL (e.g. Telegram bot file API: /file/bot{TOKEN}/...) can fetch the
// content themselves and avoid exposing the credential to Put's URL-fetching path —
// where it would land in u.Picture, debug logs, and the user JSON returned to clients.
//
// Bytes are read into memory bounded by maxAvatarFetchSize so an unbounded caller
// (e.g. a streaming HTTP body) cannot exhaust process memory.
func (p *Proxy) PutContent(userID string, content io.Reader) (avatarURL string, err error) {
	body, err := io.ReadAll(io.LimitReader(content, maxAvatarFetchSize+1))
	if err != nil {
		return "", fmt.Errorf("failed to read avatar content for %s: %w", userID, err)
	}
	if int64(len(body)) > maxAvatarFetchSize {
		return "", fmt.Errorf("avatar content for %s exceeds %d bytes", userID, maxAvatarFetchSize)
	}
	resized := p.resize(body, p.ResizeLimit)
	if resized == nil {
		return "", fmt.Errorf("avatar content for %s is not a valid image", userID)
	}
	avatarID, err := p.Store.Put(userID, resized)
	if err != nil {
		return "", err
	}
	p.Logf("[DEBUG] saved avatar bytes to %s, user %q", avatarID, userID)
	return p.URL + p.RoutePath + "/" + avatarID, nil
}

// redactAvatarURL returns the hostname only, dropping scheme, userinfo, path,
// query and fragment. This is enough to keep avatar URLs identifiable in logs
// while ensuring credentials carried in any of those parts (e.g. Telegram bot
// tokens, time-limited signed-URL tokens, basic-auth in userinfo) don't reach
// log destinations. On parse failure a sentinel is returned.
func redactAvatarURL(raw string) string {
	if u, err := url.Parse(raw); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	return "<unparseable>"
}

// load fetches an avatar from a remote url and returns the body bytes, capped at
// maxAvatarFetchSize. The bytes are passed straight to resize without an
// intermediate Reader wrapper so we don't pay for buffering twice.
func (p *Proxy) load(url string, client *http.Client) ([]byte, error) {
	var resp *http.Response
	err := retry(5, time.Second, func() error {
		var e error
		resp, e = client.Get(url)
		return e
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch avatar from the orig: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get avatar from the orig, status %s", resp.Status)
	}

	// buffer the body up to the cap to fail fast on oversized inputs.
	// Reading +1 byte beyond the cap distinguishes "exactly cap" from "too big".
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxAvatarFetchSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read avatar body: %w", err)
	}
	if int64(len(body)) > maxAvatarFetchSize {
		return nil, fmt.Errorf("avatar body exceeds %d bytes", maxAvatarFetchSize)
	}
	return body, nil
}

// Handler serves stored avatar content by avatar id. GET only; rejects invalid ids
// (403) and stored bytes that fail the safeImgContentType sniff (415). Layered
// defense headers are set on every response via setAvatarDefenseHeaders.
func (p *Proxy) Handler(w http.ResponseWriter, r *http.Request) {
	setAvatarDefenseHeaders(w)

	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	elems := strings.Split(r.URL.Path, "/")
	avatarID := elems[len(elems)-1]
	if !reValidAvatarID.MatchString(avatarID) {
		rest.SendErrorJSON(w, r, p.L, http.StatusForbidden, fmt.Errorf("invalid avatar id from %s", r.URL.Path), "can't load avatar")
		return
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

	// ReadFull (not Read) so a Store backend that hands back a short first Read
	// doesn't truncate the sniff window. Short bodies are signaled by
	// ErrUnexpectedEOF and treated like EOF.
	buf := make([]byte, sniffLen)
	n, err := io.ReadFull(avReader, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		p.Logf("[WARN] can't read from avatar reader for %s, %s", avatarID, err)
		rest.SendErrorJSON(w, r, p.L, http.StatusInternalServerError, err, "can't read avatar")
		return
	}

	// sniff buf[:n] against the safeImgContentType allowlist. This is not a full
	// image decode; it catches stores poisoned before Put validated. An empty body
	// (e.g. NoOp store) is treated as benign — nothing to render.
	var contentType string
	if n > 0 {
		var ctErr error
		contentType, ctErr = safeImgContentType(buf[:n])
		if ctErr != nil {
			p.Logf("[WARN] rejecting non-image avatar %s: %v", avatarID, ctErr)
			rest.SendErrorJSON(w, r, p.L, http.StatusUnsupportedMediaType, ctErr, "invalid avatar content")
			return
		}
	} else {
		contentType = "image/*"
	}

	// caching headers only after validation so error responses aren't cached
	etag := `"` + p.Store.ID(avatarID) + `"`
	w.Header().Set("Etag", etag)
	w.Header().Set("Cache-Control", "max-age=604800") // 7 days
	if match := r.Header.Get("If-None-Match"); match != "" && etagMatches(match, etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(size))
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(buf[:n]); err != nil {
		p.Logf("[WARN] can't write response to %s, %s", r.RemoteAddr, err)
		return
	}
	// write the rest of response size if it's bigger than 512 bytes, or nothing as EOF would be sent right away then
	if _, err = io.Copy(w, avReader); err != nil {
		p.Logf("[WARN] can't send response to %s, %s", r.RemoteAddr, err)
	}
}

// resize checks body via image.DecodeConfig (format + dimensions only, no raster
// allocation), then either returns the original bytes verbatim (limit <= 0, or
// dimensions already within limit) or fully decodes and re-encodes to PNG fitting
// within limit px on the larger side. Returns nil for non-image input or for
// dimensions exceeding maxAvatarPixels. Caller must ensure body is within
// maxAvatarFetchSize; a body over the cap is also refused defensively.
func (p *Proxy) resize(body []byte, limit int) io.Reader {
	if len(body) == 0 || int64(len(body)) > maxAvatarFetchSize {
		p.Logf("[WARN] avatar resize(): refusing body of size %d (cap %d)", len(body), maxAvatarFetchSize)
		return nil
	}

	// validate format and dimensions without allocating pixel memory.
	cfg, _, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		// non-image input must never reach the store: refuse and let the caller
		// fall back to an identicon. Returning the raw bytes here previously let
		// an attacker who controlled u.Picture poison the store with HTML/SVG that
		// the Handler would later serve back with text/html content type.
		p.Logf("[WARN] avatar resize(): can't decode avatar image, %s", err)
		return nil
	}
	// multiply in int64 — on 32-bit builds (GOARCH=386, 32-bit arm) the int
	// product of two 16-bit-or-larger dimensions can overflow and wrap below
	// maxAvatarPixels, bypassing the cap. GIF's 16-bit logical screen and
	// JPEG's 16-bit SOF dimensions both hit this if multiplied as int32.
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > int64(maxAvatarPixels) {
		p.Logf("[WARN] avatar resize(): declared dimensions %dx%d exceed safe limit", cfg.Width, cfg.Height)
		return nil
	}

	if limit <= 0 || (cfg.Width <= limit && cfg.Height <= limit) {
		p.Logf("[DEBUG] avatar resize(): no resize needed (dim %dx%d, limit %d)", cfg.Width, cfg.Height, limit)
		return bytes.NewReader(body)
	}

	// dimensions are bounded — full decode is now safe to allocate.
	src, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		p.Logf("[WARN] avatar resize(): decode after dim-check failed, %s", err)
		return nil
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	newW, newH := w*limit/h, limit
	if w > h {
		newW, newH = limit, h*limit/w
	}
	m := image.NewRGBA(image.Rect(0, 0, newW, newH))
	// slower than `draw.ApproxBiLinear.Scale()` but better quality.
	draw.BiLinear.Scale(m, m.Bounds(), src, src.Bounds(), draw.Src, nil)

	var out bytes.Buffer
	if err = png.Encode(&out, m); err != nil {
		p.Logf("[WARN] avatar resize(): can't encode resized avatar to PNG, %s", err)
		return bytes.NewReader(body) // fall back to the validated original
	}
	return &out
}

// safeImgContentType returns the sniffed Content-Type for img if it is one of the
// allow-listed raster image formats (PNG, JPEG, GIF, WebP, BMP, ICO), else an error.
// SVG is excluded.
func safeImgContentType(img []byte) (string, error) {
	ct := http.DetectContentType(img)
	base := ct
	if idx := strings.Index(base, ";"); idx >= 0 {
		base = strings.TrimSpace(base[:idx])
	}
	switch base {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/bmp",
		"image/x-icon", "image/vnd.microsoft.icon":
		return base, nil
	}
	return "", fmt.Errorf("non-image content type %q", ct)
}

// etagMatches reports whether the If-None-Match header value matches the response
// ETag per RFC 7232: the header is a comma-separated list of opaque-tags (each in
// double quotes), optionally weak-prefixed with W/. The wildcard "*" matches anything.
// We deliberately ignore weak/strong distinction because avatar responses are static
// per id — both forms identify the same resource.
func etagMatches(header, etag string) bool {
	header = strings.TrimSpace(header)
	if header == "*" {
		return true
	}
	for tag := range strings.SplitSeq(header, ",") {
		tag = strings.TrimSpace(tag)
		tag = strings.TrimPrefix(tag, "W/")
		if tag == etag {
			return true
		}
	}
	return false
}

// setAvatarDefenseHeaders applies layered defense headers on every avatar response
// (success, 304, or error). Each header survives content-type validation regressions,
// browser sniffing, and top-level navigation:
//   - Content-Security-Policy: strict, with sandbox — blocks inline scripts/handlers
//   - X-Content-Type-Options: nosniff — prevents MIME-overriding the declared type
//   - Content-Disposition: inline; filename="avatar" — frames the response as a file
func setAvatarDefenseHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox; frame-ancestors 'none'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", `inline; filename="avatar"`)
}

// GenerateAvatar for give user with identicon
func GenerateAvatar(user string) ([]byte, error) {

	iconGen, err := identicon.New("pkgz/auth", 5, 5)
	if err != nil {
		return nil, fmt.Errorf("can't create identicon service: %w", err)
	}

	ii, err := iconGen.Draw(user) // generate an IdentIcon
	if err != nil {
		return nil, fmt.Errorf("failed to draw avatar for %s: %w", user, err)
	}

	buf := &bytes.Buffer{}
	err = ii.Png(300, buf)
	return buf.Bytes(), err
}

// GetGravatarURL returns url to gravatar picture for given email
func GetGravatarURL(email string) (res string, err error) {

	hash := md5.Sum([]byte(strings.ToLower(strings.TrimSpace(email))))
	hexHash := hex.EncodeToString(hash[:])

	client := http.Client{Timeout: 5 * time.Second}
	res = "https://www.gravatar.com/avatar/" + hexHash
	resp, err := client.Get(res + "?d=404&s=80")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close() //nolint gosec // we don't care about response body
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("%s", resp.Status)
	}
	return res, nil
}

func retry(retries int, delay time.Duration, fn func() error) (err error) {
	for range retries {
		if err = fn(); err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	if err != nil {
		return fmt.Errorf("retry failed: %w", err)
	}
	return nil
}
