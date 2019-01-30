package proxy

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-chi/chi"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater"
	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/rest"
)

// Image extracts image src from comment's html and provides proxy for them
// this is needed to keep remark42 running behind of HTTPS serve all images via https
type Image struct {
	RemarkURL string
	RoutePath string
	Enabled   bool
	Timeout   time.Duration
}

// Convert all img src links without https to proxied links
func (p Image) Convert(commentHTML string) string {
	if !p.Enabled || strings.HasPrefix(p.RemarkURL, "http://") {
		return commentHTML
	}

	imgs, err := p.extract(commentHTML)
	if err != nil {
		return commentHTML
	}

	return p.replace(commentHTML, imgs)
}

// Routes returns router group to respond to proxied request
func (p Image) Routes() chi.Router {
	router := chi.NewRouter()
	if !p.Enabled {
		return router
	}
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		src, err := base64.URLEncoding.DecodeString(r.URL.Query().Get("src"))
		if err != nil {
			rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't decode image url")
			return
		}

		timeout := 60 * time.Second // default
		if p.Timeout > 0 {
			timeout = p.Timeout
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		client := http.Client{Timeout: 30 * time.Second}
		var resp *http.Response
		err = repeater.NewDefault(5, time.Second).Do(ctx, func() error {
			var e error
			req, e := http.NewRequest("GET", string(src), nil)
			if e != nil {
				return errors.Wrapf(e, "failed to make request for %s", r.URL.Query().Get("src"))
			}
			resp, e = client.Do(req.WithContext(ctx))
			return e
		})
		if err != nil {
			rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get image "+string(src))
			return
		}
		defer func() {
			if e := resp.Body.Close(); e != nil {
				log.Printf("[WARN] can't close body, %s", e)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			w.WriteHeader(resp.StatusCode)
			return
		}

		for k, v := range resp.Header {
			if strings.EqualFold(k, "Content-Type") {
				w.Header().Set(k, v[0])
			}
			if strings.EqualFold(k, "Content-Length") {
				w.Header().Set(k, v[0])
			}
		}
		// enforce client-side caching
		etag := `"` + r.URL.Query().Get("src") + `"`
		w.Header().Set("Etag", etag)
		w.Header().Set("Cache-Control", "max-age=2592000") // 30 days
		if match := r.Header.Get("If-None-Match"); match != "" {
			if strings.Contains(match, etag) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
		if _, e := io.Copy(w, resp.Body); e != nil {
			log.Printf("[WARN] can't copy image stream, %s", e)
		}
	})
	return router
}

// extract gets all non-https images and return list of src
func (p Image) extract(commentHTML string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(commentHTML))
	if err != nil {
		return nil, errors.Wrap(err, "can't create document")
	}
	result := []string{}
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		if im, ok := s.Attr("src"); ok {
			if strings.HasPrefix(im, "http://") {
				result = append(result, im)
			}
		}
	})
	return result, nil
}

// replace img links in commentHTML with route to proxy, base64 encoded original link
func (p Image) replace(commentHTML string, imgs []string) string {

	for _, img := range imgs {
		encodedImgURL := base64.URLEncoding.EncodeToString([]byte(img))
		resImgURL := p.RemarkURL + p.RoutePath + "?src=" + encodedImgURL
		commentHTML = strings.Replace(commentHTML, img, resImgURL, -1)
	}

	return commentHTML
}
