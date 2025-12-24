package service

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-pkgz/lcw/v2"
	log "github.com/go-pkgz/lgr"
	"golang.org/x/net/html"
)

const (
	teCacheMaxRecs = 1000
	teCacheTTL     = 15 * time.Minute
)

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
	res.cache, err = lcw.NewExpirableCache(o.TTL(teCacheTTL), o.MaxKeySize(teCacheMaxRecs))
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
	allowed := false
	for _, domain := range t.allowedDomains {
		if u.Hostname() == domain ||
			(strings.HasSuffix(u.Hostname(), domain) && // suffix match, e.g. "example.com" matches "www.example.com"
				u.Hostname()[len(u.Hostname())-len(domain)-1] == '.') { // but we should not match "notexample.com"
			allowed = true
			break
		}
	}
	if !allowed {
		return "", fmt.Errorf("domain %s is not allowed", u.Host)
	}
	client := http.Client{Timeout: t.client.Timeout, Transport: t.client.Transport}
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

		title, ok := t.getTitle(resp.Body)
		if !ok {
			return "", fmt.Errorf("can't get title for %s", pageURL)
		}
		return title, nil
	})

	// on error save result (empty string) to cache too and return "" title
	if err != nil {
		_, _ = t.cache.Get(pageURL, func() (string, error) { return "", nil })
		return "", err
	}

	return b, nil
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
		title := n.FirstChild.Data
		title = strings.ReplaceAll(title, "\n", "")
		title = strings.TrimSpace(title)
		return title, true
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result, ok := t.traverse(c)
		if ok {
			return result, ok
		}
	}
	return "", false
}
