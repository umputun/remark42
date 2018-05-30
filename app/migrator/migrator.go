// Package migrator provides import/export functionality. It defines Importer and Exporter interfaces
// amd implements for disqus (importer only) and "native" remark (both importer and exporter).
// Also implements AutoBackup scheduler running exports as backups and saving them locally.
package migrator

import (
	"io"
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/umputun/remark/app/store"
)

// Importer defines interface to convert posts from external sources
type Importer interface {
	Import(r io.Reader, siteID string) (int, error)
}

// Exporter defines interface to export comments from internal store
type Exporter interface {
	Export(w io.Writer, siteID string) (int, error)
}

// Store defines minimal interface needed to export and import comments
type Store interface {
	Create(comment store.Comment) (commentID string, err error)
	Find(locator store.Locator, sort string) ([]store.Comment, error)
	List(siteID string, limit int, skip int) ([]store.PostInfo, error)
	DeleteAll(siteID string) error
}

// ImportParams defines everything needed to run import
type ImportParams struct {
	DataStore Store
	InputFile string
	Provider  string
	SiteID    string
}

// ImportComments imports from given provider format and saves to store
func ImportComments(p ImportParams) (int, error) {
	log.Printf("[INFO] import from %s (%s) to %s", p.InputFile, p.Provider, p.SiteID)

	var importer Importer
	switch p.Provider {
	case "disqus":
		importer = &Disqus{DataStore: p.DataStore}
	case "native":
		importer = &Remark{DataStore: p.DataStore}
	default:
		return 0, errors.Errorf("unsupported import provider %s", p.Provider)
	}

	fh, err := os.Open(p.InputFile)
	if err != nil {
		return 0, errors.Wrapf(err, "can't open import file %s", p.InputFile)
	}

	defer func() {
		if err = fh.Close(); err != nil {
			log.Printf("[WARN] can't close %s, %s", p.InputFile, err)
		}
	}()

	return importer.Import(fh, p.SiteID)
}
