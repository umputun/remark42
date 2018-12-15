package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
)

// BackupCommand set of flags and command for export
// ExportPath used as a separate element to leverage BACKUP_PATH. If ExportFile has a path (i.e. with /) BACKUP_PATH ignored.
type BackupCommand struct {
	ExportPath string        `short:"p" long:"path" env:"BACKUP_PATH" default:"./var/backup" description:"export path"`
	ExportFile string        `short:"f" long:"file" default:"userbackup-{{.SITE}}-{{.TS}}.gz" description:"file name"`
	Site       string        `short:"s" long:"site" env:"SITE" default:"remark" description:"site name"`
	Timeout    time.Duration `long:"timeout" default:"15m" description:"import timeout"`
	CommonOpts
}

// Execute runs export with ExportCommand parameters, entry point for "export" command
func (ec *BackupCommand) Execute(args []string) error {
	log.Printf("[INFO] export to %s, site %s", ec.ExportPath, ec.Site)
	resetEnv("SECRET")

	fp := fileParser{site: ec.Site, path: ec.ExportPath, file: ec.ExportFile}
	fname, err := fp.parse(time.Now())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] export file %s", fname)

	// prepare http client and request
	client := http.Client{}
	ctx, cancel := context.WithTimeout(context.Background(), ec.Timeout)
	defer cancel()
	exportURL := fmt.Sprintf("%s/api/v1/admin/export?mode=file&site=%s&secret=%s", ec.RemarkURL, ec.Site, ec.SharedSecret)
	req, err := http.NewRequest(http.MethodGet, exportURL, nil)
	if err != nil {
		return errors.Wrapf(err, "can't make export request for %s", exportURL)
	}

	// get with timeout
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return errors.Wrapf(err, "request failed for %s", exportURL)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Printf("[WARN] failed to close response, %s", err)
		}
	}()

	if resp.StatusCode >= 300 {
		return responseError(resp)
	}

	fh, err := os.Create(fname)
	if err != nil {
		return errors.Wrapf(err, "can't create backup file %s", fname)
	}
	defer func() {
		if err = fh.Close(); err != nil {
			log.Printf("[WARN] failed to close file %s, %s", fh.Name(), err)
		}
	}()

	if _, err = io.Copy(fh, resp.Body); err != nil {
		return errors.Wrapf(err, "failed to write backup file %s", fname)
	}

	log.Printf("[INFO] export completed, file %s", fname)
	return nil
}
