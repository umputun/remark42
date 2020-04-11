package proxy

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // not used for cryptography
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater"
	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/rest"
	"github.com/umputun/remark/backend/app/store/image"
)

// Image extracts image src from comment's html and provides proxy for them
// this is needed to keep remark42 running behind of HTTPS serve all images via https
type Image struct {
	RemarkURL     string
	RoutePath     string
	HTTP2HTTPS    bool
	CacheExternal bool
	Timeout       time.Duration
	ImageService  *image.Service
}

// Convert img src links to proxied links depends on enabled options
func (p Image) Convert(commentHTML string) string {
	if p.CacheExternal {
		imgs, err := p.extract(commentHTML, func(img string) bool { return !strings.HasPrefix(img, p.RemarkURL) })
		if err != nil {
			return commentHTML
		}
		commentHTML = p.replace(commentHTML, imgs)
	}

	if p.HTTP2HTTPS && !strings.HasPrefix(p.RemarkURL, "http://") {
		imgs, err := p.extract(commentHTML, func(img string) bool { return strings.HasPrefix(img, "http://") })
		if err != nil {
			return commentHTML
		}
		commentHTML = p.replace(commentHTML, imgs)
	}

	return commentHTML
}

// extract gets all images matching predicate and return list of src
func (p Image) extract(commentHTML string, imgSrcPred func(string) bool) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(commentHTML))
	if err != nil {
		return nil, errors.Wrap(err, "can't create document")
	}
	result := []string{}
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		if im, ok := s.Attr("src"); ok {
			if imgSrcPred(im) {
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

// Handler returns http handler respond to proxied request
func (p Image) Handler(w http.ResponseWriter, r *http.Request) {
	if !p.HTTP2HTTPS && !p.CacheExternal {
		// TODO: we might need to find a better way to handle it. If admin enables caching/proxy and disables it later on
		// all comments that got converted will lose their images. We can't just return a redirect (it will open an ability
		// to redirect anywhere). We can probably continue proxying these images (but need to make sure this behavior is
		// documented) or, better, provide a way to migrate back converted comments.
		http.Error(w, "none of the proxy features are enabled", http.StatusNotImplemented)
		return
	}

	src, err := base64.URLEncoding.DecodeString(r.URL.Query().Get("src"))
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't decode image url", rest.ErrDecode)
		return
	}

	imgURL := string(src)
	var img []byte
	imgID, err := cachedImgID(imgURL)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't parse image url "+imgURL, rest.ErrAssetNotFound)
		return
	}
	if p.CacheExternal {
		img, _ = p.ImageService.Load(imgID)
	}
	if img == nil {
		img, err = p.downloadImage(context.Background(), imgURL)
		if err != nil {
			rest.SendErrorJSON(w, r, http.StatusNotFound, err, "can't get image "+imgURL, rest.ErrAssetNotFound)
			return
		}
		if p.CacheExternal {
			p.cacheImage(bytes.NewReader(img), imgID)
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

	w.Header().Add("Content-Type", p.ImageService.ImgContentType(img))
	_, err = io.Copy(w, bytes.NewReader(img))
	if err != nil {
		log.Printf("[WARN] can't copy image stream, %s", err)
	}
}

// cache image from provided Reader using given ID
func (p Image) cacheImage(r io.Reader, imgID string) {
	id, err := p.ImageService.SaveWithID(imgID, r)
	if err != nil {
		log.Printf("[WARN] unable to save image to the storage: %+v", err)
	}
	p.ImageService.Submit(func() []string { return []string{id} })
}

// download an image.
func (p Image) downloadImage(ctx context.Context, imgURL string) ([]byte, error) {
	log.Printf("[DEBUG] downloading image %s", imgURL)

	timeout := 60 * time.Second // default
	if p.Timeout > 0 {
		timeout = p.Timeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := http.Client{Timeout: 30 * time.Second}
	var resp *http.Response
	err := repeater.NewDefault(5, time.Second).Do(ctx, func() error {
		var e error
		req, e := http.NewRequest("GET", imgURL, nil)
		if e != nil {
			return errors.Wrapf(e, "failed to make request for %s", imgURL)
		}
		resp, e = client.Do(req.WithContext(ctx))
		return e
	})
	if err != nil {
		return nil, errors.Wrapf(err, "can't download image %s", imgURL)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("got unsuccessful response status %d while fetching %s", resp.StatusCode, imgURL)
	}

	imgData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Errorf("unable to read image body")
	}
	return imgData, nil
}

func sha1Str(s string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(s))) //nolint:gosec // not used for cryptography
}

// generates ID for a cached image.
// ID would look like: "cached_images/<sha1-of-image-url-hostname>-<sha1-of-image-entire-url>"
// <sha1-of-image-url-hostname> - would allow us to identify all images from particular site if ever needed
// <sha1-of-image-entire-url> - would allow us to avoid storing duplicates of the same image
//                              (as accurate as deduplication based on potentially mutable url can be)
func cachedImgID(imgURL string) (string, error) {
	parsedURL, err := url.Parse(imgURL)
	if err != nil {
		return "", errors.Wrapf(err, "can parse url %s", imgURL)
	}
	return fmt.Sprintf("cached_images/%s-%s", sha1Str(parsedURL.Hostname()), sha1Str(imgURL)), nil
}
