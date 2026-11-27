package image

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/go-pkgz/lgr"
)

// External is an image a comment shows through the image proxy: ID is where it is stored once fetched from URL
type External struct {
	ID  string
	URL string
}

// limits of fetching the external images of posted comments. One comment has at most maxExternalPerComment
// of its images stored, the rest are proxied without being stored. One user has at most
// maxPendingExternalPerUser images waiting to be fetched, so one account can't fill the queue the others' images
// go through
const (
	maxExternalPerComment     = 10
	maxPendingExternalPerUser = 20
	externalQueueSize         = 1000
	externalWorkers           = 2
)

type externalJob struct {
	External
	userID   string
	commitAt time.Time       // when the comment's delayed commit runs
	idsFn    func() []string // the ids of images the comment shows now, as passed to Submit
}

// ExternalPictures returns the external images the comment html shows through the image proxy, at most
// maxExternalPerComment of them
func (s *Service) ExternalPictures(commentHTML string) (res []External) {
	if s == nil || s.ProxyAPI == "" {
		return nil
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(commentHTML))
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	doc.Find("img").EachWithBreak(func(_ int, sl *goquery.Selection) bool {
		src, ok := sl.Attr("src")
		if !ok || !strings.Contains(src, s.ProxyAPI) {
			return true
		}
		proxied, err := url.Parse(src)
		if err != nil {
			return true
		}
		imgURL, err := base64.URLEncoding.DecodeString(proxied.Query().Get("src"))
		if err != nil {
			return true
		}
		id, err := CachedImgID(string(imgURL))
		if err != nil || seen[id] {
			return true
		}
		seen[id] = true
		res = append(res, External{ID: id, URL: string(imgURL)})
		return len(res) < maxExternalPerComment
	})
	return res
}

// SubmitExternal fetches the external images of a comment posted at posted, or previewed, and stores them in
// staging, so the comment's delayed commit keeps them like its uploads. idsFn is the func given to Submit for the
// comment, one returning nothing for a preview. An
// image fetched, or found already stored, after the commit ran is committed right away if the comment still
// shows it; a zero posted means no delayed commit waits for the comment. Does nothing
// without FetchExternal, which is set only when external images are cached, and after Close. An image the
// queue has no room for is not stored, and the proxy keeps fetching it from its site on each view
func (s *Service) SubmitExternal(userID string, posted time.Time, imgs []External, idsFn func() []string) {
	if s == nil || s.FetchExternal == nil || len(imgs) == 0 {
		return
	}
	s.mu.Lock() // Close closes done under it, so no fetcher is started after Close
	select {
	case <-s.done:
		s.mu.Unlock()
		return
	default:
	}
	s.extOnce.Do(s.startExternalWorkers)
	s.mu.Unlock()

	commitAt := posted.Add(s.EditDuration)
	for _, img := range imgs {
		s.extMu.Lock()
		if s.extPending[userID] >= maxPendingExternalPerUser {
			s.extMu.Unlock()
			log.Printf("[WARN] external image %s of user %s not stored, too many of the user's images are waiting", img.URL, userID)
			continue
		}
		select {
		case <-s.done:
			s.extMu.Unlock()
			return
		case s.extJobs <- externalJob{External: img, userID: userID, commitAt: commitAt, idsFn: idsFn}:
			s.extPending[userID]++
		default:
			log.Printf("[WARN] external image %s of user %s not stored, the queue is full", img.URL, userID)
		}
		s.extMu.Unlock()
	}
}

func (s *Service) startExternalWorkers() {
	s.extPending = map[string]int{}
	s.extJobs = make(chan externalJob, externalQueueSize)
	ctx, cancel := context.WithCancel(context.Background())
	s.wg.Go(func() { // Close stops the fetches in flight
		<-s.done
		cancel()
	})
	for range externalWorkers {
		s.wg.Go(func() {
			for {
				select {
				case <-s.done:
					return
				case job := <-s.extJobs:
					s.fetchExternal(ctx, job)
				}
			}
		})
	}
}

// fetchExternal downloads and stores one external image of a comment
func (s *Service) fetchExternal(ctx context.Context, job externalJob) {
	defer func() {
		s.extMu.Lock()
		s.extPending[job.userID]--
		if s.extPending[job.userID] <= 0 {
			delete(s.extPending, job.userID)
		}
		s.extMu.Unlock()
	}()
	// an image stored for another comment, maybe only staged, is not fetched again, but still committed below. A
	// staged copy can be older than this comment, so its cleanup timer is reset first, as Submit does for the
	// comment's other images; should cleanup remove it before that, Load misses it and it is fetched again
	_ = s.store.ResetCleanupTimer(job.ID)
	if stored, err := s.store.Load(job.ID); err != nil || stored == nil {
		data, err := s.FetchExternal(ctx, job.URL)
		if err != nil {
			log.Printf("[WARN] can't fetch external image %s, %v", job.URL, err)
			return
		}
		if err = s.SaveWithID(job.ID, bytes.NewReader(data)); err != nil {
			log.Printf("[WARN] can't store external image %s, %v", job.URL, err)
			return
		}
	}

	// the delayed commit has run already, or Close is making it run now
	select {
	case <-s.done:
	default:
		if time.Now().Before(job.commitAt) {
			return
		}
	}
	if !slices.Contains(job.idsFn(), job.ID) {
		return // the comment was edited or deleted meanwhile, cleanup removes the image from staging
	}
	if err := s.store.Commit(job.ID); err != nil {
		log.Printf("[WARN] can't commit external image %s, %v", job.URL, err)
	}
}
