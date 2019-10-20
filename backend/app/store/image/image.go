// Package image handles storing, resizing and retrieval of images
// Provides Store with Save and Load and one implementation on top of local file system.
// Service object encloses Store and add common methods, this is the one consumer should use
package image

//go:generate sh -c "mockery -inpkg -name Store -print > /tmp/mock.tmp && mv /tmp/mock.tmp image_mock.go"

import (
	"bytes"
	"context"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	"golang.org/x/image/draw"
)

// Store defines interface for saving and loading pictures.
// Declares two-stage save with commit
type Store interface {
	Save(fileName string, userID string, data []byte) (id string, err error) // get name, userID and data and returns ID of stored image
	Commit(id string) error                                                  // move image from staging to permanent
	Load(id string) (io.ReadCloser, int64, error)                            // load image by ID. Caller has to close the reader.
	Cleanup(ctx context.Context, ttl time.Duration) error                    // run removal loop for old images on staging
}

// Service extends Store with common functions needed for any store implementation
type Service struct {
	TTL       time.Duration // for how long file allowed on staging
	ImageAPI  string        // image api matching path
	MaxSize   int
	MaxWidth  int
	MaxHeight int

	store    Store
	wg       sync.WaitGroup
	submitCh chan submitReq
	once     sync.Once
	term     int32
}

const submitQueueSize = 5000

type submitReq struct {
	idsFn func() (ids []string)
	TS    time.Time
}

// NewImageService creates a new Image Service
func NewImageService(store Store, ttl time.Duration, imageAPI string, maxSize int, maxWidth int, maxHeight int) *Service {
	return &Service{
		store:     store,
		ImageAPI:  imageAPI,
		TTL:       ttl,
		MaxSize:   maxSize,
		MaxHeight: maxHeight,
		MaxWidth:  maxWidth,
	}
}

// Save preprocess image and sends it to a Store
func (s *Service) Save(fileName string, userID string, r io.Reader) (string, error) {
	data, resized, err := preprocessImage(r, s.MaxSize, s.MaxWidth, s.MaxHeight)
	if err != nil {
		return "", errors.Wrapf(err, "image file %s preprocessing failed", fileName)
	}
	if resized {
		ext := filepath.Ext(fileName)
		fileName = strings.TrimSuffix(fileName, ext) + ".png"
	}

	return s.store.Save(fileName, userID, data)
}

// Load image from configured store.
// returns ReadCloser and caller should call close after processing completed.
func (s *Service) Load(id string) (io.ReadCloser, int64, error) {
	return s.store.Load(id)
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
				// wait for TTL expiration with emergency pass on term
				for atomic.LoadInt32(&s.term) == 0 && time.Since(req.TS) <= s.TTL {
					time.Sleep(time.Millisecond * 10) // small sleep to relive busy wait but keep reactive for term (close)
				}
				for _, id := range req.idsFn() {
					if err := s.store.Commit(id); err != nil {
						log.Printf("[WARN] failed to commit image %s", id)
					}
				}
			}
			log.Printf("[INFO] image submitter terminated")
		}()
	})

	s.submitCh <- submitReq{idsFn: idsFn, TS: time.Now()}
}

// ExtractPictures gets list of images from the doc html and convert from urls to ids, i.e. user/pic.png
func (s *Service) ExtractPictures(commentHTML string) (ids []string, err error) {

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(commentHTML))
	if err != nil {
		return nil, errors.Wrap(err, "can't create document")
	}
	result := []string{}
	doc.Find("img").Each(func(i int, sl *goquery.Selection) {
		if im, ok := sl.Attr("src"); ok {
			if strings.Contains(im, s.ImageAPI) {
				elems := strings.Split(im, "/")
				if len(elems) >= 2 {
					id := elems[len(elems)-2] + "/" + elems[len(elems)-1]
					result = append(result, id)
				}
			}
		}
	})

	return result, nil
}

// Cleanup runs periodic cleanup with TTL. Blocking loop, should be called inside of goroutine by consumer
func (s *Service) Cleanup(ctx context.Context) {
	log.Printf("[INFO] start pictures cleanup, staging ttl=%v", s.TTL)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[INFO] cleanup terminated, %v", ctx.Err())
			return
		case <-time.After(s.TTL / 2):
			if err := s.store.Cleanup(ctx, s.TTL); err != nil {
				log.Printf("[WARN] failed to cleanup, %v", err)
			}
		}
	}
}

// Close flushes all in-progress submits and enforces waiting commits
func (s *Service) Close() {
	log.Printf("[INFO] close image service ")
	atomic.AddInt32(&s.term, 1) // enforce non-delayed commits for all ids left in submitCh
	if s.submitCh != nil {
		close(s.submitCh)
	}
	s.wg.Wait()
}

// resize an image of supported format (PNG, JPG, GIF) to the size of "limit" px of the
// biggest side (width or height) preserving aspect ratio.
// Returns original data if resizing is not needed or failed.
// If resized the result will be for png format and ok flag will be true.
func resize(data []byte, limitW, limitH int) ([]byte, bool) {
	if data == nil || limitW <= 0 || limitH <= 0 {
		return data, false
	}

	src, _, err := image.Decode(bytes.NewBuffer(data))
	if err != nil {
		log.Printf("[WARN] can't decode image, %s", err)
		return data, false
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= limitW && h <= limitH || w <= 0 || h <= 0 {
		log.Printf("[DEBUG] resizing image is smaller that the limit or has 0 size")
		return data, false
	}

	newW, newH := getProportionalSizes(w, h, limitW, limitH)
	m := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.BiLinear.Scale(m, m.Bounds(), src, src.Bounds(), draw.Src, nil)

	var out bytes.Buffer
	if err = png.Encode(&out, m); err != nil {
		log.Printf("[WARN] can't encode resized image to png, %s", err)
		return data, false
	}
	return out.Bytes(), true
}

// getProportionalSizes returns width and height resized by both dimensions proportionally
func getProportionalSizes(srcW, srcH int, limitW, limitH int) (resW, resH int) {

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

// check if file f is a valid image format, i.e. gif, png, jpeg or webp
func isValidImage(b []byte) bool {
	ct := http.DetectContentType(b)
	return ct == "image/gif" || ct == "image/png" || ct == "image/jpeg" || ct == "image/webp"
}

func preprocessImage(r io.Reader, maxSize int, maxWidth int, maxHeight int) ([]byte, bool, error) {
	lr := io.LimitReader(r, int64(maxSize)+1)
	data, err := ioutil.ReadAll(lr)
	if err != nil {
		return nil, false, errors.Wrap(err, "can't read source data")
	}
	if len(data) > maxSize {
		return nil, false, errors.Errorf("file is too large (limit=%d)", maxSize)
	}

	// read header first, needs it to check if data is valid png/gif/jpeg
	if !isValidImage(data[:512]) {
		return nil, false, errors.Errorf("file is not in allowed format")
	}

	data, resized := resize(data, maxWidth, maxHeight)
	return data, resized, nil
}

// guid makes a globally unique id
func guid() string {
	return xid.New().String()
}
