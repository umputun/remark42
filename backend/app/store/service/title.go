package service

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-pkgz/lcw"
	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

const (
	teCacheMaxRecs = 1000
	teCacheTTL     = 15 * time.Minute
)

// TitleExtractor gets html title from remote page, cached
type TitleExtractor struct {
	client http.Client
	cache  lcw.LoadingCache
}

// NewTitleExtractor makes extractor with cache. If memory cache failed, switching to no-cache
func NewTitleExtractor(client http.Client) *TitleExtractor {
	res := TitleExtractor{
		client: client,
	}
	var err error
	res.cache, err = lcw.NewExpirableCache(lcw.TTL(teCacheTTL), lcw.MaxKeySize(teCacheMaxRecs))
	if err != nil {
		log.Printf("[WARN] failed to make cache, caching disabled for titles, %v", err)
		res.cache = &lcw.Nop{}
	}
	return &res
}

// Get page for url and return title
func (t *TitleExtractor) Get(url string) (string, error) {
	client := http.Client{Timeout: t.client.Timeout, Transport: t.client.Transport}
	b, err := t.cache.Get(url, func() (lcw.Value, error) {
		resp, err := client.Get(url)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load page %s", url)
		}
		defer func() {
			if err = resp.Body.Close(); err != nil {
				log.Printf("[WARN] failed to close title extractor body, %v", err)
			}
		}()
		if resp.StatusCode != 200 {
			return nil, errors.Errorf("can't load page %s, code %d", url, resp.StatusCode)
		}

		title, ok := t.getTitle(resp.Body)
		if !ok {
			return nil, errors.Errorf("can't get title for %s", url)
		}
		return title, nil
	})

	// on error save result (empty string) to cache too and return "" title
	if err != nil {
		_, _ = t.cache.Get(url, func() (lcw.Value, error) { return "", nil })
		return "", err
	}

	return b.(string), nil
}

// Close title extractor
func (t *TitleExtractor) Close() error {
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
		title = strings.Replace(title, "\n", "", -1)
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
