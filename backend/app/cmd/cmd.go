// Package cmd has all top-level commands dispatched by main's flag.Parse
// The entry point of each command is Execute function
package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	log "github.com/go-pkgz/lgr"
)

// CommonOptionsCommander extends flags.Commander with SetCommon
// All commands should implement this interfaces
type CommonOptionsCommander interface {
	SetCommon(commonOpts CommonOpts)
	Execute(args []string) error
	HandleDeprecatedFlags() []DeprecatedFlag
}

// CommonOpts sets externally from main, shared across all commands
type CommonOpts struct {
	RemarkURL    string
	SharedSecret string
	Revision     string
}

// SupportCmdOpts is set of commands shared among similar commands like backup/restore and such.
// Order of fields defines the help command output order.
type SupportCmdOpts struct {
	Site        string        `short:"s" long:"site" env:"SITE" default:"remark" description:"site name"`
	AdminPasswd string        `long:"admin-passwd" env:"ADMIN_PASSWD" default:"" description:"admin basic auth password"`
	Timeout     time.Duration `long:"timeout" default:"60m" description:"timeout for the command run"`
}

// DeprecatedFlag contains information about deprecated option
type DeprecatedFlag struct {
	Old       string
	New       string
	Version   string
	Collision bool
}

// SetCommon satisfies CommonOptionsCommander interface and sets common option fields
// The method called by main for each command
func (c *CommonOpts) SetCommon(commonOpts CommonOpts) {
	c.RemarkURL = strings.TrimSuffix(commonOpts.RemarkURL, "/") // allow RemarkURL with trailing /
	c.SharedSecret = commonOpts.SharedSecret
	c.Revision = commonOpts.Revision
}

// HandleDeprecatedFlags sets new flags from deprecated and returns their list
func (c *CommonOpts) HandleDeprecatedFlags() []DeprecatedFlag { return nil }

// fileParser used to convert template strings like blah-{{.SITE}}-{{.YYYYMMDD}} the final format
type fileParser struct {
	site string
	file string
	path string
}

// parse apply template and also concat path and file. In case if file contains path separator path will be ignored
func (p *fileParser) parse(now time.Time) (string, error) {
	// file/location parameters my have template masks
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
		return "", fmt.Errorf("failed to parse %q: %w", fname, err)
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
	body, e := io.ReadAll(resp.Body)
	if e != nil {
		body = []byte("")
	}
	return fmt.Errorf("error response %q, %s", resp.Status, body)
}

// mkdir -p for all dirs
func makeDirs(dirs ...string) error {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o700); err != nil { // If path is already a directory, MkdirAll does nothing
			return fmt.Errorf("can't make directory %s: %w", dir, err)
		}
	}
	return nil
}
