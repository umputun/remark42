package migrator

import (
	"errors"
	"io"
	"io/ioutil"
	"strings"
)

// UrlMapper implements Mapper interface
type UrlMapper struct {
	rules       map[string]string
	prefixRules map[string]string
}

// LoadRules loads url-mapping rules from reader to mapper.
// Rules must be a text consists of rows separated by \n.
// Each row holds from-url and to-url separated by space.
// If urls end with asterisk (*) it means try to match by prefix.
// Example:
// https://www.myblog.com/blog/1/ https://myblog.com/blog/1/
// https://www.myblog.com/* https://myblog.com/*
func (u *UrlMapper) LoadRules(reader io.Reader) error {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	rulesText := strings.TrimSpace(string(data))

	u.rules = make(map[string]string)
	u.prefixRules = make(map[string]string)

	for _, row := range strings.Split(rulesText, "\n") {
		row = strings.TrimSpace(row)
		urls := strings.Split(row, " ")
		if len(urls) != 2 {
			return errors.New("bad row " + row)
		}

		from, to := strings.TrimSpace(urls[0]), strings.TrimSpace(urls[1])

		// determine pattern matching rule
		if strings.HasSuffix(from, "*") {
			from, to = strings.TrimSuffix(from, "*"), strings.TrimSuffix(to, "*")
			u.prefixRules[from] = to
			continue
		}

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
	for prefix, newPrefix := range u.prefixRules {
		if strings.HasPrefix(url, prefix) {
			return newPrefix + strings.TrimPrefix(url, prefix)
		}
	}
	return url
}
