package cmd

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
)

// ImportCommand set of flags and command for import
type ImportCommand struct {
	InputFile   string        `short:"f" long:"file" description:"input file name" required:"true"`
	Provider    string        `short:"p" long:"provider" default:"disqus" choice:"disqus" choice:"wordpress" choice:"commento" description:"import format"` //nolint
	Site        string        `short:"s" long:"site" env:"SITE" default:"remark" description:"site name"`
	Timeout     time.Duration `long:"timeout" default:"60m" description:"import timeout"`
	AdminPasswd string        `long:"admin-passwd" env:"ADMIN_PASSWD" required:"true" description:"admin basic auth password"`
	CommonOpts
}

// Execute runs import with ImportCommand parameters, entry point for "import" command
func (ic *ImportCommand) Execute(_ []string) error {
	log.Printf("[INFO] import %s (%s), site %s", ic.InputFile, ic.Provider, ic.Site)
	resetEnv("SECRET", "ADMIN_PASSWD")

	reader, err := ic.reader(ic.InputFile)
	if err != nil {
		return errors.Wrapf(err, "can't open import file %s", ic.InputFile)
	}

	client := http.Client{}
	ctx, cancel := context.WithTimeout(context.Background(), ic.Timeout)
	defer cancel()
	importURL := fmt.Sprintf("%s/api/v1/admin/import?site=%s&provider=%s", ic.RemarkURL, ic.Site, ic.Provider)
	req, err := http.NewRequest(http.MethodPost, importURL, reader)
	if err != nil {
		return errors.Wrapf(err, "can't make import request for %s", importURL)
	}
	req.SetBasicAuth("admin", ic.AdminPasswd)

	resp, err := client.Do(req.WithContext(ctx)) // closes request's reader
	if err != nil {
		return errors.Wrapf(err, "request failed for %s", importURL)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Printf("[WARN] failed to close response, %s", err)
		}
	}()
	if resp.StatusCode >= 300 {
		return responseError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "can't get response from importer")
	}

	log.Printf("[INFO] completed, status=%d, %s", resp.StatusCode, string(body))
	return nil
}

// reader returns reader for file. For .gz file wraps with gunzip
func (ic *ImportCommand) reader(inp string) (reader io.Reader, err error) {
	inpFile, err := os.Open(inp) // nolint
	if err != nil {
		return nil, errors.Wrapf(err, "import failed, can't open %s", inp)
	}

	reader = inpFile
	if strings.HasSuffix(ic.InputFile, ".gz") {
		if reader, err = gzip.NewReader(inpFile); err != nil {
			return nil, errors.Wrap(err, "can't make gz reader")
		}
	}
	return reader, nil
}
