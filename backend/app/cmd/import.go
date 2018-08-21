package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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

	inpFile, err := os.Open(ic.InputFile)
	if err != nil {
		return errors.Wrapf(err, "import failed, can't open %s", ic.InputFile)
	}
	defer func() { _ = inpFile.Close() }()

	client := http.Client{}
	importURL := fmt.Sprintf("%s/api/v1/admin/import?site=%s&provider=%s&secret=%s",
		ic.URL, ic.Site, ic.Provider, ic.SharedSecret)
	req, err := http.NewRequest(http.MethodPost, importURL, inpFile)
	if err != nil {
		return errors.Wrapf(err, "can't make import request for %s", importURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), ic.Timeout)
	defer cancel()
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "request failed for %s", importURL)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		return errors.Wrapf(err, "error response %s (%d)", resp.Status, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "can't get response from importer")
	}

	log.Printf("[INFO] import completed, status=%d, %s", resp.StatusCode, string(body))
	return nil
}
