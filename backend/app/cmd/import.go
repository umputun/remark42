package cmd

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

	ctx, cancel := context.WithTimeout(context.Background(), ic.Timeout)
	defer cancel()
	query := url.Values{"site": {ic.Site}, "provider": {ic.Provider}}
	endpoint := ic.RemarkURL + "/api/v1/admin/import"
	return ic.adminRequest(ctx, http.MethodPost, endpoint, query, reader, logResponse)
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
