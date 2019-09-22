package migrator

import (
	"errors"
	"io"
	"io/ioutil"
	"strings"
)

// UrlMapper implements Mapper interface
type UrlMapper struct {
	rules map[string]string
}

// NewUrlMapper reads rules from given reader and returns initialised UrlMapper
// if given rules are valid.
func NewUrlMapper(reader io.Reader) (Mapper, error) {
	u := &UrlMapper{}
	if err := u.loadRules(reader); err != nil {
		return u, err
	}
	return u, nil
}

// loadRules loads url-mapping rules from reader to mapper.
// Rules must be a text consists of rows separated by \n.
// Each row holds from-url and to-url separated by space.
// If urls end with asterisk (*) it means try to match by prefix.
// Example:
// https://www.myblog.com/blog/1/ https://myblog.com/blog/1/
// https://www.myblog.com/* https://myblog.com/*
func (u *UrlMapper) loadRules(reader io.Reader) error {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	rulesText := strings.TrimSpace(string(data))

	u.rules = make(map[string]string)

	for _, row := range strings.Split(rulesText, "\n") {
		row = strings.TrimSpace(row)
		urls := strings.Split(row, " ")
		if len(urls) != 2 {
			return errors.New("bad row " + row)
		}

		from, to := strings.TrimSpace(urls[0]), strings.TrimSpace(urls[1])
		u.rules[from] = to
	}
	return nil
}

// URL maps given url to another url according loaded url-rules.
// If not matched returns given url.
func (u *UrlMapper) URL(url string) string {
	if newUrl, ok := u.rules[url]; ok {
		return newUrl
	}
	// try to match by prefix
	for oldUrl, newUrl := range u.rules {
		if !strings.HasSuffix(oldUrl, "*") {
			continue
		}
		oldUrl = strings.TrimSuffix(oldUrl, "*")
		newUrl = strings.TrimSuffix(newUrl, "*")
		if strings.HasPrefix(url, oldUrl) {
			return newUrl + strings.TrimPrefix(url, oldUrl)
		}
	}
	// search failed, return given url
	return url
}
