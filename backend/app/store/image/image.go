// Package image handles storing, resizing and retrieval of images
// Provides Store with Save and Load implementations on top of local file system and bolt db.
// Service object encloses Store and add common methods, this is the one consumer should use.
package image

// NOTE: matryer/moq should be installed globally and works with `go generate ./...`
//go:generate moq --out image_mock.go . Store

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // not used for cryptography
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"  // register gif decoder
	_ "image/jpeg" // register jpeg decoder
	"image/png"
	"io"
	"math"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	log "github.com/go-pkgz/lgr"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register webp decoder so DecodeConfig accepts what readAndValidateImage allows
)

// Service wraps Store with common functions needed for any store implementation
// It also provides async Submit with func param retrieving all submitting IDs.
// Submitted IDs committed (i.e. moved from staging to final) on ServiceParams.EditDuration expiration.
type Service struct {
	ServiceParams

	store     Store
	wg        sync.WaitGroup
	once      sync.Once     // starts the submitter on the first Submit
	mu        sync.Mutex    // guards queue; Close closes done under it, so nothing is queued after that
	queue     []submitReq   // submits waiting for the submitter, oldest first; not bounded, so Submit never blocks
	queued    *sync.Cond    // wakes the submitter when a submit is queued or Close is called
	done      chan struct{} // closed by Close, so queued submits commit without waiting out EditDuration
	closed    chan struct{} // closed once every submitted image is committed after Close
	closeOnce sync.Once
}

// ServiceParams contains externally adjustable parameters of Service
type ServiceParams struct {
	EditDuration time.Duration // edit period for comments
	ImageAPI     string        // image api matching path
	ProxyAPI     string        // proxy api matching path
	MaxSize      int
	MaxHeight    int
	MaxWidth     int
}

// StoreInfo contains image store meta information
type StoreInfo struct {
	FirstStagingImageTS time.Time
}

// Store defines interface for saving and loading pictures.
// Declares two-stage save with Commit. Save stores to staging area and Commit moves to the final location.
// Two-stage commit scheme is used for not storing images which are uploaded but later never used in the comments,
// e.g. when somebody uploaded a picture but did not sent the comment.
type Store interface {
	Info() (StoreInfo, error)         // get meta information about storage
	Save(id string, img []byte) error // store image with passed id to staging
	Load(id string) ([]byte, error)   // load image by ID
	Delete(id string) error           // delete image by ID

	ResetCleanupTimer(id string) error                    // resets cleanup timer for the image, called on comment preview
	Commit(id string) error                               // move image from staging to permanent
	Cleanup(ctx context.Context, ttl time.Duration) error // run removal loop for old images on staging
}

type submitReq struct {
	idsFn func() (ids []string)
	TS    time.Time
}

// NewService returns new Service instance
func NewService(s Store, p ServiceParams) *Service {
	res := &Service{ServiceParams: p, store: s, done: make(chan struct{}), closed: make(chan struct{})}
	res.queued = sync.NewCond(&res.mu)
	return res
}

// Commit multiple ids immediately
func (s *Service) Commit(idsFn func() []string) error {
	var errs []error
	for _, id := range idsFn() {
		err := s.store.Commit(id)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to commit image %s: %w", id, err))
		}
	}
	return errors.Join(errs...)
}

// Submit multiple ids via function for delayed commit
func (s *Service) Submit(idsFn func() []string) {
	if idsFn == nil || s == nil {
		return
	}

	// reset cleanup timer before submitting the images
	// to prevent them from being cleaned up while waiting for EditDuration to expire
	for _, imgID := range idsFn() {
		_ = s.store.ResetCleanupTimer(imgID)
	}

	s.mu.Lock()
	select {
	case <-s.done: // closed, nothing is left to wait out EditDuration, so commit right away
		s.mu.Unlock()
		if err := s.Commit(idsFn); err != nil {
			log.Printf("[WARN] image commit error %v", err)
		}
		return
	default:
	}

	s.once.Do(func() {
		log.Printf("[DEBUG] image submitter activated")
		s.wg.Go(func() {
			for {
				req, ok := s.nextSubmit()
				if !ok {
					break
				}
				// wait for EditDuration expiration, Close cuts the wait short
				select {
				case <-time.After(time.Until(req.TS.Add(s.EditDuration))):
				case <-s.done:
				}
				if err := s.Commit(req.idsFn); err != nil {
					log.Printf("[WARN] image commit error %v", err)
				}
			}
			log.Printf("[INFO] image submitter terminated")
		})
	})
	s.queue = append(s.queue, submitReq{idsFn: idsFn, TS: time.Now()})
	s.mu.Unlock()
	s.queued.Signal()
}

// nextSubmit takes the oldest submit off the queue, waiting for one while the queue is empty.
// Returns false once Close was called and the queue is empty.
func (s *Service) nextSubmit() (req submitReq, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for len(s.queue) == 0 {
		select {
		case <-s.done:
			return submitReq{}, false
		default:
		}
		s.queued.Wait()
	}
	req = s.queue[0]
	s.queue[0] = submitReq{} // the backing array stays in use, so don't let it hold on to idsFn
	s.queue = s.queue[1:]
	return req, true
}

// ExtractPictures gets list of images from the doc html and convert from urls to ids, i.e. user/pic.png
func (s *Service) ExtractPictures(commentHTML string) (ids []string) {
	return s.extractImageIDs(commentHTML, true)
}

// ExtractNonProxiedPictures gets list of non-proxied images from the doc html and convert from urls to ids, i.e. user/pic.png
// This method is used in image check on post preview and load, as proxied images have lazy loading
// and wouldn't be present on disk but still valid as they will be loaded the first time someone requests them.
func (s *Service) ExtractNonProxiedPictures(commentHTML string) (ids []string) {
	return s.extractImageIDs(commentHTML, false)
}

// Cleanup runs periodic cleanup with 1.5*ServiceParams.EditDuration. Blocking loop, should be called inside of goroutine by consumer
func (s *Service) Cleanup(ctx context.Context) {
	if s.EditDuration <= 0 {
		log.Printf("[INFO] pictures cleanup disabled, edit duration is %v", s.EditDuration)
		<-ctx.Done()
		return
	}

	cleanupTTL := s.EditDuration * 15 / 10 // cleanup images older than 1.5 * EditDuration
	log.Printf("[INFO] start pictures cleanup, staging ttl=%v", cleanupTTL)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[INFO] cleanup terminated, %v", ctx.Err())
			return
		case <-time.After(cleanupTTL):
			if err := s.store.Cleanup(ctx, cleanupTTL); err != nil {
				log.Printf("[WARN] failed to cleanup, %v", err)
			}
		}
	}
}

// ResetCleanupTimer resets cleanup timer for the image
func (s *Service) ResetCleanupTimer(id string) error {
	return s.store.ResetCleanupTimer(id)
}

// Info returns meta information about storage
func (s *Service) Info() (StoreInfo, error) {
	return s.store.Info()
}

// Close commits all submitted images without waiting for EditDuration and returns once they are committed,
// or when ctx is done, leaving the rest to finish in the background. Submit after Close commits immediately.
func (s *Service) Close(ctx context.Context) {
	log.Printf("[INFO] close image service ")
	s.closeOnce.Do(func() {
		s.mu.Lock() // a Submit either queued its request before this or sees done closed
		close(s.done)
		s.mu.Unlock()
		s.queued.Broadcast()
		go func() {
			s.wg.Wait()
			close(s.closed)
		}()
	})
	select {
	case <-s.closed:
	case <-ctx.Done():
		log.Printf("[WARN] image service closed before all submitted images were committed, %v", ctx.Err())
	}
}

// Load wraps storage Load function.
func (s *Service) Load(id string) ([]byte, error) {
	return s.store.Load(id)
}

// Delete wraps storage Delete function.
func (s *Service) Delete(id string) error {
	return s.store.Delete(id)
}

// Save wraps storage Save function, validating and resizing the image before calling it.
func (s *Service) Save(userID string, r io.Reader) (id string, err error) {
	id = path.Join(userID, guid())
	return id, s.SaveWithID(id, r)
}

// SaveWithID wraps storage Save function, validating and resizing the image before calling it.
func (s *Service) SaveWithID(id string, r io.Reader) error {
	img, err := s.prepareImage(r)
	if err != nil {
		return err
	}
	return s.store.Save(id, img)
}

// SafeIDSegment reports whether seg can be one segment of an image id. It rejects empty and dot
// segments, any "..", path separators and control characters, so an id built from untrusted
// input can't resolve outside the image directories.
func SafeIDSegment(seg string) bool {
	if seg == "" || seg == "." || strings.Contains(seg, "..") || strings.ContainsAny(seg, `/\`) {
		return false
	}
	return !strings.ContainsFunc(seg, unicode.IsControl)
}

// returns list of image IDs from the comment html, including proxied images if includeProxied is true
func (s *Service) extractImageIDs(commentHTML string, includeProxied bool) (ids []string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(commentHTML))
	if err != nil {
		log.Printf("[ERROR] can't parse commentHTML to parse images: %q, error: %v", commentHTML, err)
		return nil
	}
	doc.Find("img").Each(func(_ int, sl *goquery.Selection) {
		if im, ok := sl.Attr("src"); ok {
			ids = append(ids, s.srcImageIDs(im, includeProxied)...)
		}
	})

	return ids
}

// srcImageIDs returns the IDs of the stored image an img src points to, including a proxied
// image if includeProxied is true
func (s *Service) srcImageIDs(src string, includeProxied bool) (ids []string) {
	if strings.Contains(src, s.ImageAPI) {
		elems := strings.Split(src, "/")
		if len(elems) >= 2 {
			user, file := elems[len(elems)-2], elems[len(elems)-1]
			if SafeIDSegment(user) && SafeIDSegment(file) {
				ids = append(ids, user+"/"+file)
			}
		}
	}
	if includeProxied && strings.Contains(src, s.ProxyAPI) {
		proxiedURL, err := url.Parse(src)
		if err != nil {
			return ids
		}
		imgURL, err := base64.URLEncoding.DecodeString(proxiedURL.Query().Get("src"))
		if err != nil {
			return ids
		}
		imgID, err := CachedImgID(string(imgURL))
		if err != nil {
			return ids
		}
		ids = append(ids, imgID)
	}
	return ids
}

// SizedClass marks an image SetSizes gave its width and height to, so the widget can tell it
// from one whose commenter set a size
const SizedClass = "sized"

// maxSizedPictures caps how many different pictures SetSizes reads from the store for one comment, so
// a comment pointing at many big uploads can't make it read them all on every preview
const maxSizedPictures = 32

// maxDrawnHeight is the max-height the widget draws a comment image at (raw-content.css)
const maxDrawnHeight = 300

// SetSizes gives every image in the comment html that is in the store the width and height the
// widget draws it at, so a browser reserves its space before loading it and the text below doesn't
// move when it loads. Uploaded images are in the store from the upload on; a proxied one only once
// the proxy has cached it, which needs CacheExternal. An image with a width or height of its own is
// left as it is, and so is any picture past the first maxSizedPictures different ones.
func (s *Service) SetSizes(commentHTML string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(commentHTML))
	if err != nil {
		return commentHTML
	}
	type size struct {
		width, height int
		ok            bool
	}
	sizes := map[string]size{} // a comment can show one picture many times, so each is read once
	doc.Find("img").Each(func(_ int, sl *goquery.Selection) {
		src, ok := sl.Attr("src")
		// a picture url anywhere else in src, such as in the query of an external thumbnail, is not the image loaded
		if !ok || (!strings.HasPrefix(src, s.ImageAPI) && !strings.HasPrefix(src, s.ProxyAPI+"?")) {
			return
		}
		if _, ok = sl.Attr("width"); ok {
			return
		}
		if _, ok = sl.Attr("height"); ok {
			return
		}
		for _, id := range s.srcImageIDs(src, true) {
			sz, seen := sizes[id]
			if !seen && len(sizes) >= maxSizedPictures {
				return
			}
			if !seen {
				sz.width, sz.height, sz.ok = s.drawnSize(id)
				sizes[id] = sz
			}
			if !sz.ok {
				continue
			}
			w, h := sz.width, sz.height
			sl.SetAttr("width", strconv.Itoa(w))
			sl.SetAttr("height", strconv.Itoa(h))
			// the sanitizer keeps no other class on an image, and drops a class attribute holding more than this one
			sl.SetAttr("class", SizedClass)
			return
		}
	})
	res, err := doc.Find("body").Html()
	if err != nil {
		return commentHTML
	}
	return res
}

// drawnSize returns the size the widget draws the stored image with the given id at in a column wide
// enough for it: as the browser shows it, turned by the EXIF orientation of a jpeg, and scaled down
// to maxDrawnHeight. The scaled width is rounded up, so the height that follows from it once the
// picture loads is never below maxDrawnHeight, which caps it. An image some browser may draw at
// another size than the one in its file is not sized: a png or webp whose EXIF orientation turns it
// a quarter, as browsers differ on whether they turn it; a jpeg or png whose EXIF pixel dimensions
// differ from its size, which browsers may take as a density to scale it by; a jpeg whose segments
// can't be walked to its EXIF; and a gif whose first frame doesn't cover its logical screen, which
// some browsers draw at the frame's size.
func (s *Service) drawnSize(id string) (width, height int, ok bool) {
	data, err := s.store.Load(id)
	if err != nil {
		return 0, 0, false
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return 0, 0, false
	}
	width, height = cfg.Width, cfg.Height
	if format == "gif" && !gifFrameCoversScreen(data, width, height) {
		return 0, 0, false
	}
	tiff, readable := pictureExif(data, format)
	if !readable {
		return 0, 0, false
	}
	tags := readExif(tiff)
	if (format == "jpeg" || format == "png") && (tags.pixelWidth > 0 || tags.pixelHeight > 0) &&
		(tags.pixelWidth != width || tags.pixelHeight != height) && (tags.pixelWidth != height || tags.pixelHeight != width) {
		return 0, 0, false
	}
	if tags.orientation >= 5 { // 5 to 8 turn the picture by a quarter
		if format != "jpeg" {
			return 0, 0, false
		}
		width, height = height, width
	}
	if height > maxDrawnHeight {
		width = int(math.Ceil(float64(width) * maxDrawnHeight / float64(height)))
		height = maxDrawnHeight
	}
	return width, height, true
}

// pictureExif returns the EXIF TIFF structure of a jpeg, png or webp, or nil when it has none, and
// false when the file can't be walked to where its EXIF would be
func pictureExif(data []byte, format string) (tiff []byte, readable bool) {
	switch format {
	case "jpeg":
		return jpegExif(data)
	case "png":
		return pngExif(data), true
	case "webp":
		return webpExif(data), true
	}
	return nil, true
}

// gifFrameCoversScreen reports whether the first frame of a gif covers its logical screen of the given
// size, which is what browsers agree to draw it at. It reads the frame's place from the image
// descriptor, stepping over the color table and extensions before it, and decodes no pixels.
func gifFrameCoversScreen(data []byte, width, height int) bool {
	const (
		headerLen      = 6 + 7 // signature and version, logical screen descriptor
		extensionIntro = 0x21
		imageSeparator = 0x2c
	)
	if len(data) < headerLen {
		return false
	}
	i := headerLen
	if flags := data[10]; flags&0x80 != 0 { // global color table
		i += 3 << (flags&0x07 + 1)
	}
	for i < len(data) {
		switch data[i] {
		case imageSeparator:
			if i+9 > len(data) {
				return false
			}
			le := binary.LittleEndian
			left, top := int(le.Uint16(data[i+1:])), int(le.Uint16(data[i+3:]))
			w, h := int(le.Uint16(data[i+5:])), int(le.Uint16(data[i+7:]))
			return left == 0 && top == 0 && w == width && h == height
		case extensionIntro: // introducer, label, then data sub-blocks up to an empty one
			i += 2
			for i < len(data) && data[i] != 0 {
				i += 1 + int(data[i])
			}
			i++
		default:
			return false
		}
	}
	return false
}

// jpegExif returns the TIFF structure of the first Exif APP1 segment of a jpeg, walking the segments
// before the first scan. It returns false at anything between segments other than fill bytes, which
// browsers differ on: some step over it to the EXIF after it, some stop there.
func jpegExif(data []byte) (tiff []byte, readable bool) {
	const (
		app1Marker = 0xe1
		sosMarker  = 0xda
		eoiMarker  = 0xd9
	)
	for i := 2; i+3 < len(data); {
		if data[i] != 0xff {
			return nil, false
		}
		marker := data[i+1]
		if marker == 0xff { // fill byte in front of a marker
			i++
			continue
		}
		if marker == sosMarker || marker == eoiMarker {
			return nil, true
		}
		n := int(data[i+2])<<8 | int(data[i+3])
		if n < 2 || n > len(data)-i-2 {
			return nil, false
		}
		if seg := data[i+4 : i+2+n]; marker == app1Marker && bytes.HasPrefix(seg, []byte("Exif\x00\x00")) {
			return seg[6:], true
		}
		i += 2 + n
	}
	return nil, true
}

// pngExif returns the eXIf chunk of a png, which counts only before IDAT
func pngExif(data []byte) []byte {
	for i := 8; i+8 <= len(data); { // chunks start after the signature
		n := int(binary.BigEndian.Uint32(data[i:]))
		kind := string(data[i+4 : i+8])
		if kind == "IDAT" || kind == "IEND" || n < 0 || n > len(data)-i-12 { // n < 0 when int is 32 bits
			return nil
		}
		if kind == "eXIf" {
			return data[i+8 : i+8+n]
		}
		i += 12 + n // length, type, data, crc
	}
	return nil
}

// webpExif returns the EXIF chunk of an extended webp, without the header some writers put in front
func webpExif(data []byte) []byte {
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return nil
	}
	for i := 12; i+8 <= len(data); {
		n := int(binary.LittleEndian.Uint32(data[i+4:]))
		if n < 0 || n > len(data)-i-8 { // n < 0 when int is 32 bits
			return nil
		}
		if string(data[i:i+4]) == "EXIF" {
			return bytes.TrimPrefix(data[i+8:i+8+n], []byte("Exif\x00\x00"))
		}
		i += 8 + n + n%2 // chunks are padded to an even size
	}
	return nil
}

// exifTags holds the EXIF tags that change how big a browser shows a picture; zero means absent
type exifTags struct {
	orientation             int // 1 to 8
	pixelWidth, pixelHeight int // PixelXDimension and PixelYDimension
}

// readExif reads the orientation from IFD0 of a TIFF structure, and the pixel dimensions from the Exif
// IFD it points to
func readExif(tiff []byte) (tags exifTags) {
	if len(tiff) < 8 {
		return tags
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return tags
	}
	// value returns a single SHORT or LONG held in the value field of an entry, or 0. Browsers read
	// the orientation only as a single SHORT, so shortOnly leaves a LONG out.
	value := func(entry []byte, shortOnly bool) int {
		if order.Uint32(entry[4:]) != 1 {
			return 0
		}
		switch typ := order.Uint16(entry[2:]); {
		case typ == 3: // SHORT
			return int(order.Uint16(entry[8:]))
		case typ == 4 && !shortOnly: // LONG
			return int(order.Uint32(entry[8:]) & 0x7fffffff) // fits int on 32 bits
		}
		return 0
	}
	// entries calls fn for each 12-byte entry of the IFD at the given offset
	entries := func(offset int, fn func(tag uint16, entry []byte)) {
		if offset < 8 || offset > len(tiff)-2 {
			return
		}
		for e := range int(order.Uint16(tiff[offset:])) {
			at := offset + 2 + e*12
			if at+12 > len(tiff) {
				return
			}
			fn(order.Uint16(tiff[at:]), tiff[at:at+12])
		}
	}
	exifIFD := 0
	entries(int(order.Uint32(tiff[4:8])&0x7fffffff), func(tag uint16, entry []byte) {
		switch tag {
		case 0x0112: // orientation
			if v := value(entry, true); v >= 1 && v <= 8 {
				tags.orientation = v
			}
		case 0x8769: // pointer to the Exif IFD
			exifIFD = value(entry, false)
		}
	})
	entries(exifIFD, func(tag uint16, entry []byte) {
		switch tag {
		case 0xa002:
			tags.pixelWidth = value(entry, false)
		case 0xa003:
			tags.pixelHeight = value(entry, false)
		}
	})
	return tags
}

// maxImagePixels caps the declared pixel count of an image before any raster decode
// is allowed. Without this, a tiny compressed "decompression bomb" image declaring
// e.g. 65535x65535 px would force image.Decode to allocate gigabytes of pixel memory
// and OOM the service on a single comment upload. 16 MP covers any realistic image
// (~4096x4096) while keeping peak allocation bounded.
const maxImagePixels = 16 * 1024 * 1024

// maxJPEGScans caps the number of scans of a jpeg before the full decode is allowed.
// The pixel cap bounds memory, not time: a progressive jpeg may carry any number of
// scans, the decoder passes over every block of the picture for each of them, and a
// scan that adds nothing takes a few dozen bytes, so a 5 MB upload declaring 4096x4096
// px would keep a core busy for hours. Encoders write about ten scans.
const maxJPEGScans = 32

// prepareImage calls readAndValidateImage and resize on provided image.
func (s *Service) prepareImage(r io.Reader) ([]byte, error) {
	data, err := readAndValidateImage(r, s.MaxSize)
	if err != nil {
		return nil, fmt.Errorf("can't load image: %w", err)
	}

	resized := resize(data, s.MaxWidth, s.MaxHeight)
	if resized == nil {
		return nil, fmt.Errorf("image rejected: malformed or exceeds %d-pixel safe limit", maxImagePixels)
	}
	return resized, nil
}

// resize validates an image and, if needed, re-encodes it to fit within the given
// pixel limits preserving aspect ratio. Returns nil for malformed input or for
// declared dimensions exceeding maxImagePixels so attacker payloads (decompression
// bombs) never reach the store, and for a jpeg with more than maxJPEGScans scans, which
// would keep the decoder busy instead. With limit <= 0 or when the image already fits,
// the original bytes are returned verbatim so animated GIFs and other multi-frame
// formats round-trip without being flattened to one frame.
//
// Validation uses image.DecodeConfig (cheap — declares dimensions, allocates nothing)
// before any full image.Decode, so a 100 KB compressed image declaring 65535x65535 px
// is rejected without ever materializing the raster.
func resize(data []byte, limitW, limitH int) []byte {
	if len(data) == 0 {
		return nil
	}

	// validate format and dimensions without allocating pixel memory.
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		log.Printf("[WARN] can't decode image config, %s", err)
		return nil
	}
	// multiply in int64 — on 32-bit builds (GOARCH=386, 32-bit arm) the int
	// product of two 16-bit-or-larger dimensions can overflow and wrap below
	// maxImagePixels, bypassing the cap.
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > int64(maxImagePixels) {
		log.Printf("[WARN] image dimensions %dx%d exceed safe limit", cfg.Width, cfg.Height)
		return nil
	}
	if format == "jpeg" {
		if scans := jpegScans(data); scans > maxJPEGScans {
			log.Printf("[WARN] jpeg with %d scans exceeds safe limit", scans)
			return nil
		}
	}

	// dimensions are bounded — full decode is now safe to allocate. Decode also
	// validates the raster body: a header that DecodeConfig accepts but with a
	// corrupt or truncated payload would slip through if we returned early on the
	// no-resize path without ever touching the pixels. Decode unconditionally,
	// then either return the original bytes (no resize needed, multi-frame intact)
	// or the re-encoded result.
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Printf("[WARN] can't decode image after dim-check, %s", err)
		return nil
	}

	if limitW <= 0 || limitH <= 0 || (cfg.Width <= limitW && cfg.Height <= limitH) {
		return data
	}

	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	newW, newH := getProportionalSizes(w, h, limitW, limitH)
	m := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(m, m.Bounds(), src, src.Bounds(), draw.Src, nil)

	var out bytes.Buffer
	if err = png.Encode(&out, m); err != nil {
		log.Printf("[WARN] can't encode resized image to png, %s", err)
		return data // fall back to the validated original
	}
	return out.Bytes()
}

// jpegScans returns the number of scans (SOS segments) image.Decode would process in a
// jpeg, without decoding anything. It follows the segments the way the decoder does: a
// segment is skipped by its declared length, entropy-coded data up to the next marker,
// and nothing after EOI is read, so marker-like bytes in metadata, in an embedded
// thumbnail or past the end of the image are not counted. Every step moves forward by at
// least one byte, so the walk is linear in the size of data. On input the decoder
// refuses, such as a file cut short, it returns the count up to that point: the full
// decode that follows refuses that file anyway.
func jpegScans(data []byte) (scans int) {
	const (
		rst0Marker = 0xd0 // restart markers RST0..RST7 have no length and no payload
		rst7Marker = 0xd7
		eoiMarker  = 0xd9
		sosMarker  = 0xda
	)

	for i := 2; i+1 < len(data); { // starts after SOI
		marker := data[i+1]
		switch {
		case data[i] != 0xff: // entropy-coded data of a scan, or extraneous bytes
			i++
		case marker == 0xff: // fill byte in front of a marker
			i++
		case marker == 0x00 || (marker >= rst0Marker && marker <= rst7Marker): // stuffed ff in a scan, restart
			i += 2
		case marker == eoiMarker:
			return scans
		default:
			if marker == sosMarker {
				scans++
			}
			if i+3 >= len(data) { // the segment length is cut off
				return scans
			}
			// the length counts its own two bytes; the decoder refuses a shorter one or a
			// segment that runs past the end of data
			n := int(data[i+2])<<8 | int(data[i+3])
			if n < 2 || n > len(data)-i-2 {
				return scans
			}
			i += 2 + n
		}
	}
	return scans
}

// getProportionalSizes returns width and height resized by both dimensions proportionally
func getProportionalSizes(srcW, srcH, limitW, limitH int) (resW, resH int) {
	if srcW <= limitW && srcH <= limitH {
		return srcW, srcH
	}

	ratioW := float64(srcW) / float64(limitW)
	propH := float64(srcH) / ratioW

	ratioH := float64(srcH) / float64(limitH)
	propW := float64(srcW) / ratioH

	if int(propH) > limitH {
		return int(propW), limitH
	}

	return limitW, int(propH)
}

// check if file f is a valid image format, i.e. gif, png, jpeg or webp and reads up to maxSize.
func readAndValidateImage(r io.Reader, maxSize int) ([]byte, error) {
	isValidImage := func(b []byte) bool {
		ct := http.DetectContentType(b)
		return ct == "image/gif" || ct == "image/png" || ct == "image/jpeg" || ct == "image/webp"
	}

	lr := io.LimitReader(r, int64(maxSize)+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}

	if len(data) > maxSize {
		return nil, fmt.Errorf("file is too large (limit=%d)", maxSize)
	}

	// read header first to check the format. http.DetectContentType inspects up
	// to the first 512 bytes, but a smaller body is fine — pass the whole slice
	// rather than panicking on a fixed-size sub-slice.
	header := data
	if len(header) > 512 {
		header = header[:512]
	}
	if !isValidImage(header) {
		return nil, fmt.Errorf("file format not allowed")
	}

	return data, nil
}

// guid makes a picture id from 128 random bits, hex-encoded. The id is all that protects a picture
// uploaded but not yet posted, since GET /picture serves it to anyone, so it must reveal nothing about
// other uploads. Nothing parses the id beyond SafeIDSegment, so pictures saved under the 20-character
// ids of earlier versions keep loading
func guid() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b) // crypto/rand.Read never returns an error, it crashes the program instead
	return hex.EncodeToString(b)
}

// Sha1Str converts provided string to sha1
func Sha1Str(s string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(s))) //nolint:gosec // not used for cryptography
}

// CachedImgID generates ID for a cached image.
// ID would look like: "cached_images/<sha1-of-image-url-hostname>-<sha1-of-image-entire-url>"
// <sha1-of-image-url-hostname> - would allow us to identify all images from particular site if ever needed
// <sha1-of-image-entire-url> - would allow us to avoid storing duplicates of the same image
// (as accurate as deduplication based on potentially mutable url can be)
func CachedImgID(imgURL string) (string, error) {
	parsedURL, err := url.Parse(imgURL)
	if err != nil {
		return "", fmt.Errorf("can parse url %s: %w", imgURL, err)
	}
	return fmt.Sprintf("cached_images/%s-%s", Sha1Str(parsedURL.Hostname()), Sha1Str(imgURL)), nil
}
