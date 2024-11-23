package proxy

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater"

	"github.com/umputun/remark42/backend/app/rest"
	"github.com/umputun/remark42/backend/app/store/image"
)

// Image extracts image src from comment's html and provides proxy for them
// this is needed to keep remark42 running behind of HTTPS serve all images via https
type Image struct {
	RemarkURL            string
	RoutePath            string
	Blacklist            []string
	HTTP2HTTPS           bool
	CacheExternal        bool
	Timeout              time.Duration
	ImageService         *image.Service
	AllowPrivateNetworks bool
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
		return nil, fmt.Errorf("can't create document: %w", err)
	}
	result := []string{}
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
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
	src, err := base64.URLEncoding.DecodeString(r.URL.Query().Get("src"))
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't decode image url", rest.ErrDecode)
		return
	}

	imgURL := string(src)

	// Check if the URL is blacklisted
	if p.isBlacklisted(imgURL) {
		rest.SendErrorJSON(w, r, http.StatusForbidden, fmt.Errorf("blacklisted URL"), "URL is blacklisted", rest.ErrAssetNotFound)
		return
	}

	// Check for private network access
	if !p.AllowPrivateNetworks && isPrivateURL(imgURL) {
		rest.SendErrorJSON(w, r, http.StatusForbidden, fmt.Errorf("private network access not allowed"), "private network access not allowed", rest.ErrAssetNotFound)
		return
	}

	var img []byte
	imgID, err := image.CachedImgID(imgURL)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't parse image url "+imgURL, rest.ErrAssetNotFound)
		return
	}
	// try to load from cache for case it was saved when CacheExternal was enabled
	img, _ = p.ImageService.Load(imgID)
	if img == nil {
		img, err = p.downloadImage(context.Background(), imgURL)
		if err != nil {
			if strings.Contains(err.Error(), "invalid content type") {
				rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "invalid content type", rest.ErrImgNotFound)
				return
			}
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
	err := p.ImageService.SaveWithID(imgID, r)
	if err != nil {
		log.Printf("[WARN] unable to save image to the storage: %+v", err)
	}
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
	defer client.CloseIdleConnections()
	var resp *http.Response
	err := repeater.NewDefault(5, time.Second).Do(ctx, func() error {
		var e error
		req, e := http.NewRequest("GET", imgURL, http.NoBody)
		if e != nil {
			return fmt.Errorf("failed to make request for %s: %w", imgURL, e)
		}
		resp, e = client.Do(req.WithContext(ctx)) //nolint:bodyclose // need a refactor to fix that
		return e
	})
	if err != nil {
		return nil, fmt.Errorf("can't download image %s: %w", imgURL, err)
	}
	defer resp.Body.Close() //nolint gosec // we don't care about response body

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got unsuccessful response status %d while fetching %s", resp.StatusCode, imgURL)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return nil, fmt.Errorf("invalid content type %s", contentType)
	}

	imgData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read image body")
	}
	return imgData, nil
}

// isBlacklisted checks if the given URL matches any blacklisted domain, IP, or CIDR
func (p Image) isBlacklisted(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	host := parsedURL.Hostname()
	ip := net.ParseIP(host)

	for _, item := range p.Blacklist {
		// Check for IP address
		if ip != nil && item == ip.String() {
			return true
		}

		// Check for exact domain match or proper subdomain
		if host == item || (strings.HasSuffix(host, "."+item) && len(host) > len(item)+1) {
			return true
		}

		// Check for CIDR
		_, ipNet, err := net.ParseCIDR(item)
		if err == nil && ip != nil && ipNet.Contains(ip) {
			return true
		}
	}

	return false
}
