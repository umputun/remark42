package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"
)

type fileParser struct {
	site string
	file string
	path string
}

func (p *fileParser) parse(now time.Time) (string, error) {

	fileTemplate := struct {
		YYYYMMDD string
		YYYY     string
		YYYYMM   string
		MM       string
		DD       string
		UNIX     int64
		SITE     string
	}{
		YYYYMMDD: now.Format("20060102"),
		YYYY:     now.Format("2006"),
		YYYYMM:   now.Format("200601"),
		MM:       now.Format("01"),
		DD:       now.Format("02"),
		UNIX:     now.Unix(),
		SITE:     p.site,
	}

	bb := bytes.Buffer{}
	fname := p.file
	if !strings.Contains(p.file, string(filepath.Separator)) {
		fname = filepath.Join(p.path, p.file)
	}

	if err := template.Must(template.New("bb").Parse(fname)).Execute(&bb, fileTemplate); err != nil {
		return "", errors.Wrapf(err, "failed to parse %q", fname)
	}
	return bb.String(), nil
}
