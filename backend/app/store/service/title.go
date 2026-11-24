package service

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/go-pkgz/lcw/v2"
	log "github.com/go-pkgz/lgr"
	"golang.org/x/net/html"
)

const (
	teCacheMaxRecs = 1000
	teCacheTTL     = 15 * time.Minute
	teMaxPageSize  = 1024 * 1024 // how much of the page is read to find the title, in bytes after decompression
	teMaxTitleLen  = 512         // maximum length of the title, in runes
	teMaxRedirects = 10
)

// reTitleEnd matches the start of a closing title tag, up to the character that ends the tag name
var reTitleEnd = regexp.MustCompile(`(?i)</title[\t\n\f\r />]`)

// TitleExtractor gets html title from remote page, cached
type TitleExtractor struct {
	client         http.Client
	cache          lcw.LoadingCache[string]
	allowedDomains []string
}

// NewTitleExtractor makes extractor with cache. If memory cache failed, switching to no-cache
func NewTitleExtractor(client http.Client, allowedDomains []string) *TitleExtractor {
	log.Printf("[DEBUG] creating extractor, allowed domains %+v", allowedDomains)
	res := TitleExtractor{
		client:         client,
		allowedDomains: allowedDomains,
	}
	var err error
	o := lcw.NewOpts[string]()
	res.cache, err = lcw.NewExpirableCache(o.TTL(teCacheTTL), o.MaxKeys(teCacheMaxRecs))
	if err != nil {
		log.Printf("[WARN] failed to make cache, caching disabled for titles, %v", err)
		res.cache = &lcw.Nop[string]{}
	}
	return &res
}

// Get page for url and return title
func (t *TitleExtractor) Get(pageURL string) (string, error) {
	// parse domain of the URL and check if it's in the allowed list
	u, err := url.Parse(pageURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse url %s: %w", pageURL, err)
	}
	if !t.isAllowed(u) {
		return "", fmt.Errorf("domain %s is not allowed", u.Host)
	}
	client := http.Client{
		Timeout:   t.client.Timeout,
		Transport: t.client.Transport,
		// every redirect must lead to an allowed domain too
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= teMaxRedirects {
				return fmt.Errorf("stopped after %d redirects", teMaxRedirects)
			}
			if !t.isAllowed(req.URL) {
				return fmt.Errorf("redirect to domain %s is not allowed", req.URL.Host)
			}
			return nil
		},
	}
	defer client.CloseIdleConnections()
	b, err := t.cache.Get(pageURL, func() (string, error) {
		resp, e := client.Get(pageURL)
		if e != nil {
			return "", fmt.Errorf("failed to load page %s: %w", pageURL, e)
		}
		defer func() {
			if err = resp.Body.Close(); err != nil {
				log.Printf("[WARN] failed to close title extractor body, %v", err)
			}
		}()
		if resp.StatusCode != 200 {
			return "", fmt.Errorf("can't load page %s, code %d", pageURL, resp.StatusCode)
		}

		page, e := readHead(resp.Body)
		if e != nil {
			return "", fmt.Errorf("can't read page %s: %w", pageURL, e)
		}
		title, ok := t.getTitle(bytes.NewReader(page))
		if !ok {
			return "", fmt.Errorf("can't get title for %s", pageURL)
		}
		return title, nil
	})

	// on error save result (empty string) to cache too, so the page is not requested again until it expires
	if err != nil {
		_, _ = t.cache.Get(pageURL, func() (string, error) { return "", nil })
		return "", err
	}
	// an empty title is a failure cached by an earlier call or a blank <title>, never a title to store
	if b == "" {
		return "", fmt.Errorf("no title for %s, the page failed to load or has a blank title", pageURL)
	}

	return b, nil
}

// isAllowed checks if the host of the url is one of the allowed domains or a subdomain of one of them.
// An IP address has no subdomains and is allowed only by exact match; this also covers an IPv6 address
// with a zone, as the zone is free text and could end with an allowed domain. Names are compared
// without regard to the case of ASCII letters, as DNS compares them; any other letter is compared as it is,
// because a letter that only looks like an ASCII one makes a different name.
func (t *TitleExtractor) isAllowed(u *url.URL) bool {
	host := u.Hostname()
	if _, err := netip.ParseAddr(host); err == nil {
		return slices.Contains(t.allowedDomains, host)
	}
	host = lowerASCII(host)
	for _, domain := range t.allowedDomains {
		domain = lowerASCII(domain)
		if host == domain ||
			(strings.HasSuffix(host, domain) && // suffix match, e.g. "example.com" matches "www.example.com"
				host[len(host)-len(domain)-1] == '.') { // but we should not match "notexample.com"
			return true
		}
	}
	return false
}

// lowerASCII lower-cases the ASCII letters of s and leaves every other character as it is
func lowerASCII(s string) string {
	return strings.Map(func(r rune) rune {
		if 'A' <= r && r <= 'Z' {
			return r + 'a' - 'A'
		}
		return r
	}, s)
}

// readHead reads the start of the page, teMaxPageSize bytes at most: the title is in the head. A longer page is
// cut right after the name of the last closing title tag inside the limit, so a title is found only when all
// of it was read, and never ends in a piece of text or of a tag the limit cut in two.
func readHead(r io.Reader) ([]byte, error) {
	page, err := io.ReadAll(io.LimitReader(r, teMaxPageSize+1))
	if err != nil {
		return nil, err
	}
	if len(page) <= teMaxPageSize {
		return page, nil
	}
	ends := reTitleEnd.FindAllIndex(page[:teMaxPageSize], -1)
	if len(ends) == 0 {
		return nil, nil
	}
	return page[:ends[len(ends)-1][1]], nil
}

// Close title extractor
func (t *TitleExtractor) Close() error {
	t.client.CloseIdleConnections()
	return t.cache.Close()
}

// get title from body reader, traverse recursively
func (t *TitleExtractor) getTitle(r io.Reader) (string, bool) {
	doc, err := html.Parse(r)
	if err != nil {
		log.Printf("[WARN] can't get header, %+v", err)
		return "", false
	}
	return t.traverse(doc)
}

func (t *TitleExtractor) isTitleElement(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "title"
}

func (t *TitleExtractor) traverse(n *html.Node) (string, bool) {
	if t.isTitleElement(n) {
		if n.FirstChild == nil { // empty <title></title>, same as no title
			return "", false
		}
		title := n.FirstChild.Data
		title = strings.ReplaceAll(title, "\n", "")
		title = strings.TrimSpace(title)
		// cut the title to teMaxTitleLen runes, and copy it so it doesn't keep the whole page text in memory
		runes := 0
		for i := range title {
			if runes == teMaxTitleLen {
				title = strings.TrimSpace(title[:i])
				break
			}
			runes++
		}
		return strings.Clone(title), true
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result, ok := t.traverse(c)
		if ok {
			return result, ok
		}
	}
	return "", false
}
