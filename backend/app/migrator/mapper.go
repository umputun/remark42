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

func (u *UrlMapper) LoadRules(reader io.Reader) error {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	rules := make(map[string]string)
	for _, row := range strings.Split(string(data), "\n") {
		urls := strings.Split(row, " ")
		if len(urls) != 2 {
			return errors.New("bad row " + row)
		}
		rules[urls[0]] = urls[1]
	}
	u.rules = rules
	return nil
}

// URL maps given url to another url according loaded url-rules.
// If match failed returns given url.
func (u *UrlMapper) URL(url string) string {
	newUrl, ok := u.rules[url]
	if ok {
		return newUrl
	}
	// try to match
	return url
}
