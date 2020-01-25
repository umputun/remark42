// Package image handles storing, resizing and retrieval of images
// Provides Store with Save and Load and implementations on top of local file system and bolt db.
// Service object encloses Store and add common methods, this is the one consumer should use.
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
// Declares two-stage save with commit. Save stores to staging area and Commit moves to the final location
type Store interface {
	Save(fileName string, userID string, r io.Reader) (id string, err error) // get name and reader and returns ID of stored (staging) image
	SaveWithID(id string, r io.Reader) (string, error)                       // store image for passed id to staging
	Load(id string) (io.ReadCloser, int64, error)                            // load image by ID. Caller has to close the reader.
	SizeLimit() int                                                          // max image size

	commit(id string) error                               // move image from staging to permanent
	cleanup(ctx context.Context, ttl time.Duration) error // run removal loop for old images on staging
}

// Service wrap Store with common functions needed for any store implementation
// It also provides async Submit with func param retrieving all submitting ids.
// Submitted ids committed (i.e. moved from staging to final) on TTL expiration.
type Service struct {
	Store
	TTL      time.Duration // for how long file allowed on staging
	ImageAPI string        // image api matching path

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
					if err := s.commit(id); err != nil {
						log.Printf("[WARN] failed to commit image %s", id)
					}
				}
				atomic.StoreInt32(&s.term, 0) // indicates completion of ids commits
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
			if err := s.Store.cleanup(ctx, s.TTL); err != nil {
				log.Printf("[WARN] failed to cleanup, %v", err)
			}
		}
	}
}

// Close flushes all in-progress submits and enforces waiting commits
func (s *Service) Close() {
	log.Printf("[INFO] close image service ")
	atomic.StoreInt32(&s.term, 1) // enforce non-delayed commits for all ids left in submitCh
	for {
		// set to 0 by commit goroutine after everything waited on TTL sent
		if atomic.LoadInt32(&s.term) == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if s.submitCh != nil {
		close(s.submitCh)
	}
	s.wg.Wait()
}

// resize an image of supported format (PNG, JPG, GIF) to the size of "limit" px of the
// biggest side (width or height) preserving aspect ratio.
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
	draw.BiLinear.Scale(m, m.Bounds(), src, src.Bounds(), draw.Src, nil)

	var out bytes.Buffer
	if err = png.Encode(&out, m); err != nil {
		log.Printf("[WARN] can't encode resized image to png, %s", err)
		return data
	}
	return out.Bytes()
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
