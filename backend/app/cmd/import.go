package cmd

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	log "github.com/go-pkgz/lgr"
)

// ImportCommand set of flags and command for import
type ImportCommand struct {
	InputFile string `short:"f" long:"file" description:"input file name" required:"true"`
	Provider  string `short:"p" long:"provider" default:"disqus" choice:"disqus" choice:"wordpress" choice:"commento" description:"import format"` //nolint

	SupportCmdOpts
	CommonOpts
}

// Execute runs import with ImportCommand parameters, entry point for "import" command
func (ic *ImportCommand) Execute(_ []string) error {
	log.Printf("[INFO] import %s (%s), site %s", ic.InputFile, ic.Provider, ic.Site)
	resetEnv("SECRET", "ADMIN_PASSWD")

	reader, err := ic.reader(ic.InputFile)
	if err != nil {
		return fmt.Errorf("can't open import file %s: %w", ic.InputFile, err)
	}

	client := http.Client{}
	defer client.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(context.Background(), ic.Timeout)
	defer cancel()
	importURL := fmt.Sprintf("%s/api/v1/admin/import?site=%s&provider=%s", ic.RemarkURL, ic.Site, ic.Provider)
	req, err := http.NewRequest(http.MethodPost, importURL, reader)
	if err != nil {
		return fmt.Errorf("can't make import request for %s: %w", importURL, err)
	}
	req.SetBasicAuth("admin", ic.AdminPasswd)

	resp, err := client.Do(req.WithContext(ctx)) // closes request's reader
	if err != nil {
		return fmt.Errorf("request failed for %s: %w", importURL, err)
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
		return fmt.Errorf("can't get response from importer: %w", err)
	}

	log.Printf("[INFO] completed, status=%d, %s", resp.StatusCode, string(body))
	return nil
}

// reader returns reader for file. For .gz file wraps with gunzip
func (ic *ImportCommand) reader(inp string) (reader io.Reader, err error) {
	inpFile, err := os.Open(inp) // nolint
	if err != nil {
		return nil, fmt.Errorf("import failed, can't open %s: %w", inp, err)
	}

	reader = inpFile
	if strings.HasSuffix(ic.InputFile, ".gz") {
		if reader, err = gzip.NewReader(inpFile); err != nil {
			return nil, fmt.Errorf("can't make gz reader: %w", err)
		}
	}
	return reader, nil
}
