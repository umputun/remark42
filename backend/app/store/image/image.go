// Package image handles storing, resizing and retrieval of images
// Provides Store with Save and Load and one implementation on top of local file system.
// Service object encloses Store and add common methods, this is the one consumer should use
package image

//go:generate sh -c "mockgen -source=image.go -package=image > image_mock.go"

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
	"github.com/rs/xid"
)

// Store defines interface for saving and loading pictures.
// Declares two-stage save with commit
type Store interface {
	Save(fileName string, userID string, r io.Reader) (id string, err error) // get name and reader and returns ID of stored image
	Commit(id string) error                                                  // move image from staging to permanent
	Load(id string) (io.ReadCloser, int64, error)                            // load image by ID. Caller has to close the reader.
	Cleanup(ctx context.Context, ttl time.Duration) error                    // run removal loop for old images on staging
	SizeLimit() int                                                          // max image size
}

// Service extends Store with common functions needed for any store implementation
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
					if err := s.Commit(id); err != nil {
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
			if err := s.Store.Cleanup(ctx, s.TTL); err != nil {
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

// check if file f is a valid image format, i.e. gif, png or jpeg
func isValidImage(b []byte) bool {
	ct := http.DetectContentType(b)
	return ct == "image/gif" || ct == "image/png" || ct == "image/jpeg" || ct == "image/webp"
}

// guid makes a globally unique id
func guid() string {
	return xid.New().String()
}
