package migrator

import (
	"errors"
	"io"
	"io/ioutil"
	"strings"
)

// UrlMapper implements Mapper interface
type UrlMapper map[string]string

func NewUrlMapper(reader io.Reader) (UrlMapper, error) {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	mapper := make(map[string]string)
	for _, row := range strings.Split(string(data), "\n") {
		urls := strings.Split(row, " ")
		if len(urls) != 2 {
			return nil, errors.New("bad row " + row)
		}
		mapper[urls[0]] = urls[1]
	}
	return mapper, nil
}

// URL maps given url to another url if it found.
// Otherwise returns given url
func (u UrlMapper) URL(url string) string {
	newUrl, ok := u[url]
	if !ok {
		return url
	}
	return newUrl
}
