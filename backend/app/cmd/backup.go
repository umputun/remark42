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
// ExportPath used as a separate element to leverage BACKUP_PATH. If ExportFile has a path (i.e. /) BACKUP_PATH ignored.
type BackupCommand struct {
	ExportPath   string        `short:"p" long:"path" env:"BACKUP_PATH" default:"./var/backup" description:"export path"`
	ExportFile   string        `short:"f" long:"file" default:"userbackup-{{.site}}-{{.YYYYMMDD}}.gz" description:"file name"`
	Site         string        `long:"site" env:"SITE" default:"remark" description:"site name"`
	SharedSecret string        `long:"secret" env:"SECRET" description:"shared secret key" required:"true"`
	Timeout      time.Duration `long:"timeout" default:"15m" description:"import timeout"`
	URL          string        `long:"url" default:"http://127.0.0.1:8081" description:"migrator base url"`
}

// Execute runs export with ExportCommand parameters, entry point for "export" command
func (ec *BackupCommand) Execute(args []string) error {
	log.Printf("[INFO] export to %s, site %s", ec.ExportPath, ec.Site)
	fp := fileParser{site: ec.Site, path: ec.ExportPath, file: ec.ExportFile}
	fname, err := fp.parse(time.Now())
	if err != nil {
		return err
	}

	log.Printf("[INFO] export file %s", fname)

	client := http.Client{}
	exportURL := fmt.Sprintf("%s/api/v1/admin/export?site=%s&secret=%s", ec.URL, ec.Site, ec.SharedSecret)
	req, err := http.NewRequest(http.MethodGet, exportURL, nil)
	if err != nil {
		return errors.Wrapf(err, "can't make export request for %s", exportURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), ec.Timeout)
	defer cancel()
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "request failed for %s", exportURL)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Printf("[WARN] failed to close response, %s", err)
		}
	}()

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

	return nil
}
