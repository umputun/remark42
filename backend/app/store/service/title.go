package service

import (
	"io"
	"log"
	"net/http"

	"github.com/go-pkgz/rest/cache"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

// TitleExtractor gets html title from remote page, cached
type TitleExtractor struct {
	client http.Client
	cache  cache.LoadingCache
}

// NewTitleExtractor makes extractor with cache. If memory cache failed, switching to no-cache
func NewTitleExtractor(client http.Client) *TitleExtractor {
	res := TitleExtractor{
		client: client,
	}
	var err error
	res.cache, err = cache.NewMemoryCache(cache.MaxKeys(1000))
	if err != nil {
		log.Printf("[WARN] failed to make cache, %v", err)
		res.cache = &cache.Nop{}
	}
	return &res
}

// Get page for url and return title
func (t *TitleExtractor) Get(url string) (string, error) {

	b, err := t.cache.Get(cache.NewKey("site").ID(url), func() ([]byte, error) {
		resp, err := t.client.Get(url)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load page %s", url)
		}
		defer resp.Body.Close() //nolint
		if resp.StatusCode != 200 {
			return nil, errors.Errorf("can't load page %s, code %d", url, resp.StatusCode)
		}

		title, ok := t.getTitle(resp.Body)
		if !ok {
			return nil, errors.Errorf("can't get title for %s", url)
		}
		return []byte(title), nil
	})

	if err != nil {
		return "", err
	}

	return string(b), nil
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
		return n.FirstChild.Data, true
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result, ok := t.traverse(c)
		if ok {
			return result, ok
		}
	}
	return "", false
}
