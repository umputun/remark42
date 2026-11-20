package migrator

import (
	"fmt"
	"io"
	"strings"
)

// URLMapper implements Mapper interface
type URLMapper struct {
	rules map[string]string
}

// NewURLMapper reads rules from given reader and returns initialized URLMapper
// if given rules are valid.
func NewURLMapper(reader io.Reader) (Mapper, error) {
	u := &URLMapper{}
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
func (u *URLMapper) loadRules(reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	rulesText := strings.TrimSpace(string(data))

	u.rules = make(map[string]string)

	for row := range strings.SplitSeq(rulesText, "\n") {
		row = strings.TrimSpace(row)
		urls := strings.Split(row, " ")
		if len(urls) != 2 {
			return fmt.Errorf("bad row %s", row)
		}

		from, to := strings.TrimSpace(urls[0]), strings.TrimSpace(urls[1])
		u.rules[from] = to
	}
	return nil
}

// URL maps given url to another url according loaded url-rules.
// An exact rule wins, then the prefix rule with the longest prefix.
// If not matched returns given url.
func (u *URLMapper) URL(url string) string {
	if newURL, ok := u.rules[url]; ok {
		return newURL
	}
	// try to match by prefix
	matched, matchedPrefix, mappedURL := false, "", url
	for oldURL, newURL := range u.rules {
		prefix, ok := strings.CutSuffix(oldURL, "*")
		if !ok || (matched && len(prefix) <= len(matchedPrefix)) {
			continue
		}
		if after, ok := strings.CutPrefix(url, prefix); ok {
			matched, matchedPrefix, mappedURL = true, prefix, strings.TrimSuffix(newURL, "*")+after
		}
	}
	return mappedURL
}
