package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	log "github.com/go-pkgz/lgr"
)

// RemapCommand set of flags and command for change linkage between comments to
// different urls based on given rules (input file)
type RemapCommand struct {
	Site        string        `short:"s" long:"site" env:"SITE" default:"remark" description:"site name"`
	InputFile   string        `short:"f" long:"file" description:"input file name" required:"true"`
	AdminPasswd string        `long:"admin-passwd" env:"ADMIN_PASSWD" required:"true" description:"admin basic auth password"`
	Timeout     time.Duration `long:"timeout" default:"60m" description:"remap timeout"`
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

	client := http.Client{}
	ctx, cancel := context.WithTimeout(context.Background(), rc.Timeout)
	defer cancel()
	remapURL := fmt.Sprintf("%s/api/v1/admin/remap?site=%s", rc.RemarkURL, rc.Site)
	req, err := http.NewRequest(http.MethodPost, remapURL, rulesReader)
	if err != nil {
		return fmt.Errorf("can't make remap request for %s: %w", remapURL, err)
	}
	req.SetBasicAuth("admin", rc.AdminPasswd)

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("request failed for %s: %w", remapURL, err)
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
		return fmt.Errorf("can't get response: %w", err)
	}

	log.Printf("[INFO] completed, status=%d, %s", resp.StatusCode, string(body))
	return nil
}
