package proxy

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater/v2"

	"github.com/umputun/remark42/backend/app/rest"
	"github.com/umputun/remark42/backend/app/safehttp"
	"github.com/umputun/remark42/backend/app/store/image"
)

// errInvalidUpstreamContentType is returned by downloadImage when the upstream's
// Content-Type header is not image/*. The handler checks via errors.Is to convert
// it into a 400 (input rejected) instead of the generic 404 (fetch failed).
var errInvalidUpstreamContentType = errors.New("invalid upstream content type")

// Image extracts image src from comment's html and provides proxy for them
// this is needed to keep remark42 running behind of HTTPS serve all images via https
type Image struct {
	RemarkURL     string
	RoutePath     string
	HTTP2HTTPS    bool
	CacheExternal bool
	Timeout       time.Duration
	ImageService  *image.Service
	// Transport, if non-nil, is used as-is for outbound image fetches and is the
	// caller's responsibility to make SSRF-safe. When nil, safehttp.Transport()
	// is installed, which blocks dialing any private/reserved IP and resolves
	// hostnames to defeat DNS rebinding.
	Transport http.RoundTripper
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

// etagVersionPrefix is the security-version tag bumped whenever cached responses for the
// same src need to be invalidated. Pre-fix responses were served as text/html and cached
// by browsers/proxies under ETag `"<base64(src)>"`; the prefix invalidates those validators
// so revalidating clients get a fresh 200 instead of letting the cached HTML 304.
//
// LIMITATION: with the 30-day max-age below, browsers serve pre-fix bytes from their
// local cache without contacting the server until that TTL expires or the cache is
// evicted under memory pressure. The prefix only helps clients that revalidate during
// the cached lifetime (Ctrl+R, intermediaries, post-expiry use). Operators running a
// CDN/edge cache in front of remark42 should purge /api/v1/img after deploy. The
// realistic exposure is narrow: cache carryover only affects users who navigated
// top-level to an attacker URL pre-fix and still have that URL cached — the normal
// <img> embed path cached text/html but never executed it.
const etagVersionPrefix = "v2:"

// Handler returns http handler respond to proxied request
func (p Image) Handler(w http.ResponseWriter, r *http.Request) {
	rest.SetImageDefenseHeaders(w)

	srcParam := r.URL.Query().Get("src")
	src, err := base64.URLEncoding.DecodeString(srcParam)
	if err != nil {
		sendImageProxyError(w, r, http.StatusBadRequest, err, "can't decode image url", rest.ErrDecode)
		return
	}

	imgURL := string(src)
	imgID, err := image.CachedImgID(imgURL)
	if err != nil {
		sendImageProxyError(w, r, http.StatusBadRequest, fmt.Errorf("invalid image url"), "can't parse image url", rest.ErrAssetNotFound)
		return
	}

	// compute the current-version etag once. We don't set it as a response header yet
	// because error paths below must NOT inherit it — otherwise transient failures
	// (4xx) would get cached alongside the 30-day Cache-Control of the success path.
	// The etag (and Cache-Control) are set only on the 304 short-circuit and the
	// validated 200 path.
	etag := `"` + etagVersionPrefix + srcParam + `"`
	// short-circuit revalidation before any cache lookup or upstream fetch: a matching
	// current-version If-None-Match means the client already has bytes from a prior
	// successful (post-fix, validated) 200, so a bodyless 304 is safe and avoids
	// upstream DoS amplification on hot comment pages without CacheExternal.
	if match := r.Header.Get("If-None-Match"); match != "" && rest.EtagMatches(match, etag) {
		w.Header().Set("Etag", etag)
		w.Header().Set("Cache-Control", "max-age=2592000") // 30 days
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// try to load from cache for case it was saved when CacheExternal was enabled
	img, _ := p.ImageService.Load(imgID)
	if img == nil {
		img, err = p.downloadImage(r.Context(), imgURL)
		if err != nil {
			log.Printf("[WARN] failed to download image: %v", err)
			if errors.Is(err, errInvalidUpstreamContentType) {
				sendImageProxyError(w, r, http.StatusBadRequest, fmt.Errorf("invalid content type"), "invalid content type", rest.ErrImgNotFound)
				return
			}
			sendImageProxyError(w, r, http.StatusNotFound, fmt.Errorf("failed to fetch"), "can't get image", rest.ErrAssetNotFound)
			return
		}
		if p.CacheExternal {
			p.cacheImage(bytes.NewReader(img), imgID)
		}
	}

	// validate body bytes are actually an image — never trust upstream Content-Type or cache
	contentType, err := rest.SafeImgContentType(img)
	if err != nil {
		log.Printf("[WARN] rejecting non-image content from %s: %v", imgURL, err)
		sendImageProxyError(w, r, http.StatusUnsupportedMediaType, err, "invalid image content", rest.ErrImgNotFound)
		return
	}

	// success path: long-lived client cache with etag for cheap revalidation. 30-day
	// TTL keeps the proxy efficient for hot pages; when clients DO revalidate
	// (Ctrl+R, intermediaries, post-expiry), the versioned etag ensures pre-fix
	// poisoned validators don't match and a fresh validated 200 is returned. See
	// etagVersionPrefix godoc for the limitation on browser-local caches.
	w.Header().Set("Etag", etag)
	w.Header().Set("Cache-Control", "max-age=2592000") // 30 days
	w.Header().Set("Content-Type", contentType)
	_, err = io.Copy(w, bytes.NewReader(img))
	if err != nil {
		log.Printf("[WARN] can't copy image stream, %s", err)
	}
}

// sendImageProxyError writes a no-store error response so a transient failure (4xx)
// cannot inherit the success path's 30-day Cache-Control or the versioned ETag, which
// would otherwise pin the error in the browser/intermediary cache for that TTL.
// Defense headers from SetImageDefenseHeaders at the top of the handler survive.
func sendImageProxyError(w http.ResponseWriter, r *http.Request, status int, err error, details string, errCode int) {
	w.Header().Set("Cache-Control", "no-store")
	rest.SendErrorJSON(w, r, status, err, details, errCode)
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
		transport = safehttp.Transport()
	}
	client := http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
	defer client.CloseIdleConnections()
	var resp *http.Response
	err := repeater.NewFixed(5, time.Second).Do(ctx, func() error {
		var e error
		// SSRF safety: client.Transport is safehttp.Transport() when p.Transport is nil
		// (see Image.Transport contract above); when caller supplies a transport they
		// own SSRF safety for that path.
		req, e := http.NewRequest("GET", imgURL, http.NoBody) //nolint:gosec // see comment above
		if e != nil {
			return fmt.Errorf("failed to make request for %s: %w", imgURL, e)
		}
		resp, e = client.Do(req.WithContext(ctx)) //nolint:bodyclose,gosec // body closed in defer; transport contract above
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
		return nil, fmt.Errorf("%w: %s", errInvalidUpstreamContentType, contentType)
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
