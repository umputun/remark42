package proxy

import (
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	"github.com/umputun/remark/app/rest"
)

// Image extracts image src from comment's html and provides proxy for them
// this is needed to keep remark42 running behind of HTTPS serve all images via https
type Image struct {
	RemarkURL string
	RoutePath string
	Enabled   bool
}

// Convert all img src links without https to proxied links
func (p Image) Convert(commentHTML string) string {
	if !p.Enabled {
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
		client := http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(string(src))
		if err != nil {
			rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get image "+string(src))
			return
		}
		defer func() {
			if e := resp.Body.Close(); e != nil {
				log.Printf("[WARN] can't close body, %s", e)
			}
		}()

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

// replace img links in commentHTML with route to proxy with base64 encoded original link
func (p Image) replace(commentHTML string, imgs []string) string {
	for _, img := range imgs {
		encodedImgURL := base64.URLEncoding.EncodeToString([]byte(img))
		resImgURL := p.RemarkURL + p.RoutePath + "?src=" + encodedImgURL
		commentHTML = strings.Replace(commentHTML, img, resImgURL, -1)
	}
	return commentHTML
}
