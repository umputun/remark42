package migrator

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/umputun/remark/app/store"
)

// Importer defines interface to convert posts from external sources
type Importer interface {
	Import(r io.Reader, siteID string) error
}

// Exporter defines interface to export comments from internal store
type Exporter interface {
	Export(w io.Writer, siteID string) error
}

// ImportParams defines everyting needed to run import
type ImportParams struct {
	DataStore store.Interface
	InputFile string
	Provider  string
	SiteID    string
}

// ImportComments imports from given provider format and saves to store
func ImportComments(p ImportParams) error {
	log.Printf("[INFO] import from %s (%s) to %s", p.InputFile, p.Provider, p.SiteID)

	var importer Importer
	switch p.Provider {
	case "disqus":
		importer = &Disqus{DataStore: p.DataStore}
	case "native":
		importer = &Remark{DataStore: p.DataStore}
	default:
		return errors.Errorf("unsupported import provider %s", p.Provider)
	}

	fh, err := os.Open(p.InputFile)
	if err != nil {
		return errors.Wrapf(err, "can't open import file %s", p.InputFile)
	}

	defer func() {
		if err = fh.Close(); err != nil {
			log.Printf("[WARN] can't close %s, %s", p.InputFile, err)
		}
	}()

	return importer.Import(fh, p.SiteID)
}

// AutoBackup runs daily export to local files
func AutoBackup(exporter Exporter, backupLocation string) {
	log.Print("[INFO] activate auto-backup")
	tick := time.NewTicker(24 * time.Hour)
	for _ = range tick.C {
		log.Print("[DEBUG] make backup")
		fh, err := os.Create(fmt.Sprintf("%s/backup-%s.gz", backupLocation, time.Now().Format("20060102")))
		if err != nil {
			log.Printf("[WARN] can't create backup file, %s", err)
			continue
		}
		gz := gzip.NewWriter(fh)

		if err = exporter.Export(gz, ""); err != nil {
			log.Printf("[WARN] export failed, %+v", err)
		}
		_ = gz.Close()
		_ = fh.Close()
	}
}
