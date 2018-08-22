package cmd

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// ImportCommand set of flags and command for import
type ImportCommand struct {
	InputFile    string        `short:"f" long:"file" description:"input file name" required:"true"`
	Provider     string        `short:"p" long:"provider" default:"disqus" choice:"disqus" choice:"wordpress" description:"import format"`
	Site         string        `long:"site" env:"SITE" default:"remark" description:"site name"`
	SharedSecret string        `long:"secret" env:"SECRET" description:"shared secret key" required:"true"`
	Timeout      time.Duration `long:"timeout" default:"15m" description:"import timeout"`
	URL          string        `long:"url" default:"http://127.0.0.1:8081" description:"migrator base url"`
}

// Execute runs import with ImportCommand parameters, entry point for "import" command
func (ic *ImportCommand) Execute(args []string) error {
	log.Printf("[INFO] import %s (%s), site %s", ic.InputFile, ic.Provider, ic.Site)

	reader, err := ic.openReader(ic.InputFile)
	if err != nil {
		return errors.Wrapf(err, "can't open import file %s", ic.InputFile)
	}

	client := http.Client{}
	importURL := fmt.Sprintf("%s/api/v1/admin/import?site=%s&provider=%s&secret=%s",
		ic.URL, ic.Site, ic.Provider, ic.SharedSecret)
	req, err := http.NewRequest(http.MethodPost, importURL, reader)
	if err != nil {
		return errors.Wrapf(err, "can't make import request for %s", importURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), ic.Timeout)
	defer cancel()
	req = req.WithContext(ctx)
	resp, err := client.Do(req) // closes reader
	if err != nil {
		return errors.Wrapf(err, "request failed for %s", importURL)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Printf("[WARN] failed to close response, %s", err)
		}
	}()
	if resp.StatusCode != 200 {
		return errors.Errorf("error response %s (%d)", resp.Status, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "can't get response from importer")
	}

	log.Printf("[INFO] import completed, status=%d, %s", resp.StatusCode, string(body))
	return nil
}

// openReader returns reader and close func. For .gz files wraps with gzipper
func (ic *ImportCommand) openReader(inp string) (reader io.Reader, err error) {
	inpFile, err := os.Open(inp)
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
