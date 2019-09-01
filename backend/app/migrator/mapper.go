package migrator

import (
	"errors"
	"io"
	"io/ioutil"
	"strings"
)

// UrlMapper implements mapper interface
type UrlMapper map[string]string

func NewUrlMapper(reader io.Reader) (UrlMapper, error) {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	mapper := make(map[string]string)
	for _, row := range strings.Split(string(data), "\n") {
		urls := strings.Split(row, ":")
		if len(urls) != 2 {
			return nil, errors.New("bad input")
		}
		mapper[urls[0]] = urls[1]
	}
	return mapper, nil
}

func (u UrlMapper) URL(url string) string {
	newUrl, ok := u[url]
	if !ok {
		return url
	}
	return newUrl
}
