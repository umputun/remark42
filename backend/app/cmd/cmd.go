// Package cmd has all top-level commands dispatched by main's flag.Parse
// The entry point of each command is Execute function
package cmd

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"
)

// fileParser used to convert template strings like blah-{{.SITE}}-{{.YYYYMMDD}} the final format
type fileParser struct {
	site string
	file string
	path string
}

// parse apply template and also concat path and file. In case if file contains path separator path will be ignored
func (p *fileParser) parse(now time.Time) (string, error) {

	fileTemplate := struct {
		YYYYMMDD string
		YYYY     string
		YYYYMM   string
		MM       string
		DD       string
		TS       string
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
		TS:       now.Format("20060102T150405"),
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

// resetEnv clears sensitive env vars
func resetEnv(envs ...string) {
	for _, env := range envs {
		if err := os.Unsetenv(env); err != nil {
			log.Printf("[WARN] can't unset env %s, %s", env, err)
		}
	}
}

// responseError returns error with status and response body
func responseError(resp *http.Response) error {
	body, e := ioutil.ReadAll(resp.Body)
	if e != nil {
		body = []byte("")
	}
	return errors.Errorf("error response %q, %s", resp.Status, body)
}
