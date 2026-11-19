package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	log "github.com/go-pkgz/lgr"
)

// RemapCommand set of flags and command for change linkage between comments to
// different urls based on given rules (input file)
type RemapCommand struct {
	InputFile string `short:"f" long:"file" description:"input file name" required:"true"`

	SupportCmdOpts
	CommonOpts
}

// Execute runs (re)mapper with RemapCommand parameters, entry point for "remap" command
func (rc *RemapCommand) Execute(_ []string) error {
	log.Printf("[INFO] start remap, site %s, file with rules %s", rc.Site, rc.InputFile)
	resetEnv("SECRET", "ADMIN_PASSWD")

	rulesReader, err := os.Open(rc.InputFile)
	if err != nil {
		return fmt.Errorf("cant open file %s: %w", rc.InputFile, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), rc.Timeout)
	defer cancel()
	query := url.Values{"site": {rc.Site}}
	endpoint := rc.RemarkURL + "/api/v1/admin/remap"
	return rc.adminRequest(ctx, http.MethodPost, endpoint, query, rulesReader, logResponse)
}
