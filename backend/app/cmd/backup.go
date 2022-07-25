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

// BackupCommand set of flags and command for export
// ExportPath used as a separate element to leverage BACKUP_PATH. If ExportFile has a path (i.e. with /) BACKUP_PATH ignored.
type BackupCommand struct {
	ExportPath string `short:"p" long:"path" env:"BACKUP_PATH" default:"./var/backup" description:"export path"`
	ExportFile string `short:"f" long:"file" default:"userbackup-{{.SITE}}-{{.TS}}.gz" description:"file name"`

	SupportCmdOpts
	CommonOpts
}

// Execute runs export with ExportCommand parameters, entry point for "export" command
func (ec *BackupCommand) Execute(_ []string) error {
	log.Printf("[INFO] export to %s, site %s", ec.ExportPath, ec.Site)
	resetEnv("SECRET", "ADMIN_PASSWD")

	fp := fileParser{site: ec.Site, path: ec.ExportPath, file: ec.ExportFile}
	fname, err := fp.parse(time.Now())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] export file %s", fname)

	// prepare http client and request
	client := http.Client{}
	defer client.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(context.Background(), ec.Timeout)
	defer cancel()
	exportURL := fmt.Sprintf("%s/api/v1/admin/export?mode=file&site=%s", ec.RemarkURL, ec.Site)
	req, err := http.NewRequest(http.MethodGet, exportURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("can't make export request for %s: %w", exportURL, err)
	}
	req.SetBasicAuth("admin", ec.AdminPasswd)

	// get with timeout
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("request failed for %s: %w", exportURL, err)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Printf("[WARN] failed to close response, %s", err)
		}
	}()

	if resp.StatusCode >= 300 {
		return responseError(resp)
	}

	fh, err := os.Create(fname) //nolint:gosec // harmless
	if err != nil {
		return fmt.Errorf("can't create backup file %s: %w", fname, err)
	}
	defer func() { //nolint:gosec // false positive on defer without error check when it's checked here
		if err = fh.Close(); err != nil {
			log.Printf("[WARN] failed to close file %s, %s", fh.Name(), err)
		}
	}()

	if _, err = io.Copy(fh, resp.Body); err != nil {
		return fmt.Errorf("failed to write backup file %s: %w", fname, err)
	}

	log.Printf("[INFO] export completed, file %s", fname)
	return nil
}
