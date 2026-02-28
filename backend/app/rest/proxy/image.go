package proxy

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater/v2"

	"github.com/umputun/remark42/backend/app/rest"
	"github.com/umputun/remark42/backend/app/store/image"
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
	Transport     http.RoundTripper // if nil, uses SSRF-safe transport blocking private IPs
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
		commentHTML = strings.ReplaceAll(commentHTML, img, resImgURL)
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
	var img []byte
	imgID, err := image.CachedImgID(imgURL)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, fmt.Errorf("invalid image url"), "can't parse image url", rest.ErrAssetNotFound)
		return
	}
	// try to load from cache for case it was saved when CacheExternal was enabled
	img, _ = p.ImageService.Load(imgID)
	if img == nil {
		img, err = p.downloadImage(context.Background(), imgURL)
		if err != nil {
			log.Printf("[WARN] failed to download image: %v", err)
			if strings.Contains(err.Error(), "invalid content type") {
				rest.SendErrorJSON(w, r, http.StatusBadRequest, fmt.Errorf("invalid content type"), "invalid content type", rest.ErrImgNotFound)
				return
			}
			rest.SendErrorJSON(w, r, http.StatusNotFound, fmt.Errorf("failed to fetch"), "can't get image", rest.ErrAssetNotFound)
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

	transport := p.Transport
	if transport == nil {
		transport = ssrfSafeTransport()
	}
	client := http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
	defer client.CloseIdleConnections()
	var resp *http.Response
	err := repeater.NewFixed(5, time.Second).Do(ctx, func() error {
		var e error
		req, e := http.NewRequest("GET", imgURL, http.NoBody)
		if e != nil {
			return fmt.Errorf("failed to make request for %s: %w", imgURL, e)
		}
		resp, e = client.Do(req.WithContext(ctx)) //nolint:bodyclose,gosec // body closed in defer; SSRF mitigated by ssrfSafeTransport
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

	maxSize := 5 * 1024 * 1024 // 5MB default
	if p.ImageService != nil && p.ImageService.MaxSize > 0 {
		maxSize = p.ImageService.MaxSize
	}
	lr := io.LimitReader(resp.Body, int64(maxSize)+1)
	imgData, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("unable to read image body: %w", err)
	}
	if len(imgData) > maxSize {
		return nil, fmt.Errorf("image is too large")
	}
	return imgData, nil
}

// ssrfSafeTransport returns an http.Transport with a dialer that blocks connections to private IP addresses.
// it resolves the host, validates all IPs, then dials using the resolved IP to prevent DNS rebinding attacks.
// tries each resolved IP in order to handle dual-stack hosts where the first IP may be unreachable.
func ssrfSafeTransport() *http.Transport {
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("invalid address %s: %w", addr, err)
			}

			// resolve the host to IP addresses
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("can't resolve host %s: %w", host, err)
			}
			if len(ips) == 0 {
				return nil, fmt.Errorf("no IP addresses resolved for host %s", host)
			}

			for _, ip := range ips {
				if isPrivateIP(ip.IP) {
					return nil, fmt.Errorf("access to private address is not allowed")
				}
			}

			// try each resolved IP to handle dual-stack hosts where some IPs may be unreachable
			var lastErr error
			for _, ip := range ips {
				conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
				if dialErr == nil {
					return conn, nil
				}
				lastErr = dialErr
			}
			return nil, fmt.Errorf("can't connect to %s: %w", host, lastErr)
		},
	}
}

// privateCIDRs holds pre-parsed private/reserved CIDR blocks for SSRF protection.
var privateCIDRs = func() []*net.IPNet {
	cidrs := []string{
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
		"100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16",
		"::1/128", "fc00::/7", "fe80::/10",
	}
	blocks := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, block, _ := net.ParseCIDR(cidr)
		blocks = append(blocks, block)
	}
	return blocks
}()

// isPrivateIP checks if the given IP belongs to a private/reserved range.
func isPrivateIP(ip net.IP) bool {
	if ip.IsUnspecified() {
		return true
	}
	for _, block := range privateCIDRs {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}
