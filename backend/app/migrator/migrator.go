// Package migrator provides import/export functionality. It defines Importer and Exporter interfaces
// amd implements for disqus (importer only) and "native" remark (both importer and exporter).
// Also implements AutoBackup scheduler running exports as backups and saving them locally.
package migrator

import (
	"io"
	"os"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/service"
)

// Importer defines interface to convert posts from external sources
type Importer interface {
	Import(r io.Reader, siteID string) (int, error)
}

// Exporter defines interface to export comments from internal store
type Exporter interface {
	Export(w io.Writer, siteID string) (int, error)
}

// Mapper defines interface to convert data in import procedure
type Mapper interface {
	URL(url string) string
}

// MapperMaker defines function that reads rules from reader and
// returns new Mapper with loaded rules. If rules are not valid
// it returns error.
type MapperMaker func(reader io.Reader) (Mapper, error)

// Store defines minimal interface needed to export and import comments
type Store interface {
	Create(comment store.Comment) (commentID string, err error)
	Find(locator store.Locator, sort string, user store.User) ([]store.Comment, error)
	List(siteID string, limit int, skip int) ([]store.PostInfo, error)
	DeleteAll(siteID string) error
	Metas(siteID string) (umetas []service.UserMetaData, pmetas []service.PostMetaData, err error)
	SetMetas(siteID string, umetas []service.UserMetaData, pmetas []service.PostMetaData) error
}

// ImportParams defines everything needed to run import
type ImportParams struct {
	DataStore Store
	InputFile string
	Provider  string
	SiteID    string
}

var adminUser = store.User{Admin: true}

// ImportComments imports from given provider format and saves to store
func ImportComments(p ImportParams) (int, error) {
	log.Printf("[INFO] import from %s (%s) to %s", p.InputFile, p.Provider, p.SiteID)

	var importer Importer
	switch p.Provider {
	case "disqus":
		importer = &Disqus{DataStore: p.DataStore}
	case "wordpress":
		importer = &WordPress{DataStore: p.DataStore}
	case "native":
		importer = &Native{DataStore: p.DataStore}
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
