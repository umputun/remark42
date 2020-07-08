// Package image handles storing, resizing and retrieval of images
// Provides Store with Save and Load implementations on top of local file system and bolt db.
// Service object encloses Store and add common methods, this is the one consumer should use.
package image

//go:generate sh -c "mockery -inpkg -name Store -print > /tmp/mock.tmp && mv /tmp/mock.tmp image_mock.go"

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // not used for cryptography
	"encoding/base64"
	"fmt"
	"image"

	// support gif and jpeg images decoding
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/go-pkgz/lgr"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	"golang.org/x/image/draw"
)

// Service wraps Store with common functions needed for any store implementation
// It also provides async Submit with func param retrieving all submitting ids.
// Submitted ids committed (i.e. moved from staging to final) on commitTTL expiration.
type Service struct {
	ServiceParams

	store       Store
	wg          sync.WaitGroup
	submitCh    chan submitReq
	once        sync.Once
	term        int32 // term value used atomically to detect emergency termination
	submitCount int32 // atomic increment for counting submitted images
}

// ServiceParams contains externally adjustable parameters of Service
type ServiceParams struct {
	EditDuration time.Duration // edit period for comments
	ImageAPI     string        // image api matching path
	ProxyAPI     string        // proxy api matching path
	MaxSize      int
	MaxHeight    int
	MaxWidth     int

	// duration of time after which images are checked and committed if still
	// present in the submitted comment after it's EditDuration is expired
	commitTTL time.Duration
	// duration of time after which images are deleted from staging
	cleanupTTL time.Duration
}

// StoreInfo contains image store meta information
type StoreInfo struct {
	FirstStagingImageTS time.Time
}

// To regenerate mock run from this directory:
// sh -c "mockery -inpkg -name Store -print > /tmp/image-mock.tmp && mv /tmp/image-mock.tmp image_mock.go"

// Store defines interface for saving and loading pictures.
// Declares two-stage save with Commit. Save stores to staging area and Commit moves to the final location.
// Two-stage commit scheme is used for not storing images which are uploaded but later never used in the comments,
// e.g. when somebody uploaded a picture but did not sent the comment.
type Store interface {
	Info() (StoreInfo, error)         // get meta information about storage
	Save(id string, img []byte) error // store image with passed id to staging
	Load(id string) ([]byte, error)   // load image by ID

	Commit(id string) error                               // move image from staging to permanent
	Cleanup(ctx context.Context, ttl time.Duration) error // run removal loop for old images on staging
}

const submitQueueSize = 5000

type submitReq struct {
	idsFn func() (ids []string)
	TS    time.Time
}

// NewService returns new Service instance
func NewService(s Store, p ServiceParams) *Service {
	p.commitTTL = p.EditDuration * 15 / 10  // Commit call on every 1.5 * EditDuration
	p.cleanupTTL = p.EditDuration * 25 / 10 // Cleanup call on every 2.5 * EditDuration
	// In case Cleanup and Submit start at the same time (case of stale staging images check
	// on the program start) these TTL values guarantee that Commit will happen before Cleanup.
	return &Service{ServiceParams: p, store: s}
}

// SubmitAndCommit multiple ids immediately
func (s *Service) SubmitAndCommit(idsFn func() []string) error {
	errs := new(multierror.Error)
	for _, id := range idsFn() {
		err := s.store.Commit(id)
		if err != nil {
			errs = multierror.Append(errs, errors.Wrapf(err, "failed to commit image %s", id))
		}
	}
	return errs.ErrorOrNil()
}

// Submit multiple ids via function for delayed commit
func (s *Service) Submit(idsFn func() []string) {
	if idsFn == nil || s == nil {
		return
	}

	s.once.Do(func() {
		log.Printf("[DEBUG] image submitter activated")
		s.submitCh = make(chan submitReq, submitQueueSize)
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for req := range s.submitCh {
				// wait for commitTTL expiration with emergency pass on term
				for atomic.LoadInt32(&s.term) == 0 && time.Since(req.TS) <= s.commitTTL {
					time.Sleep(time.Millisecond * 10) // small sleep to relive busy wait but keep reactive for term (close)
				}
				err := s.SubmitAndCommit(req.idsFn)
				if err != nil {
					log.Printf("[WARN] image commit error %v", err)
				}

				atomic.AddInt32(&s.submitCount, -1)
			}
			log.Printf("[INFO] image submitter terminated")
		}()
	})

	atomic.AddInt32(&s.submitCount, 1)

	s.submitCh <- submitReq{idsFn: idsFn, TS: time.Now()}
}

// ExtractPictures gets list of images from the doc html and convert from urls to ids, i.e. user/pic.png
func (s *Service) ExtractPictures(commentHTML string) (ids []string, err error) {

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(commentHTML))
	if err != nil {
		return nil, errors.Wrap(err, "can't create document")
	}
	doc.Find("img").Each(func(i int, sl *goquery.Selection) {
		if im, ok := sl.Attr("src"); ok {
			if strings.Contains(im, s.ImageAPI) {
				elems := strings.Split(im, "/")
				if len(elems) >= 2 {
					id := elems[len(elems)-2] + "/" + elems[len(elems)-1]
					ids = append(ids, id)
				}
			}
			if strings.Contains(im, s.ProxyAPI) {
				proxiedURL, err := url.Parse(im)
				if err != nil {
					return
				}
				imgURL, err := base64.URLEncoding.DecodeString(proxiedURL.Query().Get("src"))
				if err != nil {
					return
				}
				imgID, err := CachedImgID(string(imgURL))
				if err != nil {
					return
				}
				ids = append(ids, imgID)
			}
		}
	})

	return ids, nil
}

// Cleanup runs periodic cleanup with cleanupTTL. Blocking loop, should be called inside of goroutine by consumer
func (s *Service) Cleanup(ctx context.Context) {
	log.Printf("[INFO] start pictures cleanup, staging ttl=%v", s.cleanupTTL)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[INFO] cleanup terminated, %v", ctx.Err())
			return
		case <-time.After(s.cleanupTTL):
			if err := s.store.Cleanup(ctx, s.cleanupTTL); err != nil {
				log.Printf("[WARN] failed to cleanup, %v", err)
			}
		}
	}
}

// Info returns meta information about storage
func (s *Service) Info() (StoreInfo, error) {
	return s.store.Info()
}

// Close flushes all in-progress submits and enforces waiting commits
func (s *Service) Close(ctx context.Context) {
	log.Printf("[INFO] close image service ")

	waitForTerm := func(ctx context.Context) {
		ticker := time.NewTicker(10 * time.Millisecond)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if atomic.LoadInt32(&s.submitCount) == 0 {
					return
				}
			}
		}
	}

	atomic.StoreInt32(&s.term, 1) // enforce non-delayed commits for all ids left in submitCh
	waitForTerm(ctx)

	if s.submitCh != nil {
		close(s.submitCh)
	}
	s.wg.Wait()
}

// Load wraps storage Load function.
func (s *Service) Load(id string) ([]byte, error) {
	return s.store.Load(id)
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

// ImgContentType returns content type for provided image
func (s *Service) ImgContentType(img []byte) string {
	contentType := http.DetectContentType(img)
	if contentType == "application/octet-stream" {
		// replace generic fallback with one which make sense in our scenario
		return "image/*"
	}
	return contentType
}

// prepareImage calls readAndValidateImage and resize on provided image.
func (s *Service) prepareImage(r io.Reader) ([]byte, error) {
	data, err := readAndValidateImage(r, s.MaxSize)
	if err != nil {
		return nil, errors.Wrapf(err, "can't load image")
	}

	data = resize(data, s.MaxWidth, s.MaxHeight)
	return data, nil
}

// resize an image of supported format (PNG, JPG, GIF) to the size of "limit" px of
// the biggest side (width or height) preserving aspect ratio.
// Returns original data if resizing is not needed or failed.
// If resized the result will be for png format
func resize(data []byte, limitW, limitH int) []byte {
	if data == nil || limitW <= 0 || limitH <= 0 {
		return data
	}

	src, _, err := image.Decode(bytes.NewBuffer(data))
	if err != nil {
		log.Printf("[WARN] can't decode image, %s", err)
		return data
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= limitW && h <= limitH || w <= 0 || h <= 0 {
		log.Printf("[DEBUG] resizing image is smaller that the limit or has 0 size")
		return data
	}

	newW, newH := getProportionalSizes(w, h, limitW, limitH)
	m := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(m, m.Bounds(), src, src.Bounds(), draw.Src, nil)

	var out bytes.Buffer
	if err = png.Encode(&out, m); err != nil {
		log.Printf("[WARN] can't encode resized image to png, %s", err)
		return data
	}
	return out.Bytes()
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
	data, err := ioutil.ReadAll(lr)
	if err != nil {
		return nil, err
	}

	if len(data) > maxSize {
		return nil, errors.Errorf("file is too large (limit=%d)", maxSize)
	}

	// read header first, needs it to check if data is valid png/gif/jpeg
	if !isValidImage(data[:512]) {
		return nil, errors.Errorf("file format not allowed")
	}

	return data, nil
}

// guid makes a globally unique id
func guid() string {
	return xid.New().String()
}

// Sha1Str converts provided string to sha1
func Sha1Str(s string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(s))) //nolint:gosec // not used for cryptography
}

// CachedImgID generates ID for a cached image.
// ID would look like: "cached_images/<sha1-of-image-url-hostname>-<sha1-of-image-entire-url>"
// <sha1-of-image-url-hostname> - would allow us to identify all images from particular site if ever needed
// <sha1-of-image-entire-url> - would allow us to avoid storing duplicates of the same image
//                              (as accurate as deduplication based on potentially mutable url can be)
func CachedImgID(imgURL string) (string, error) {
	parsedURL, err := url.Parse(imgURL)
	if err != nil {
		return "", errors.Wrapf(err, "can parse url %s", imgURL)
	}
	return fmt.Sprintf("cached_images/%s-%s", Sha1Str(parsedURL.Hostname()), Sha1Str(imgURL)), nil
}
