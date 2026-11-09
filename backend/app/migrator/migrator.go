// Package migrator provides import/export functionality. It defines Importer and Exporter interfaces
// amd implements for disqus (importer only) and "native" remark (both importer and exporter).
// Also implements AutoBackup scheduler running exports as backups and saving them locally.
package migrator

import (
	"fmt"
	"io"
	"os"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/service"
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
	FindRaw(locator store.Locator) ([]store.Comment, error)
	List(siteID string, limit, skip int) ([]store.PostInfo, error)
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

// converter reads an export of another comment system from r and passes each comment to emit.
// It returns the first error of emit, or an error when r can't be read to the end.
type converter func(r io.Reader, siteID string, emit func(store.Comment) error) error

// importComments replaces the comments of siteID with the ones convert reads from r.
// The site is cleared when the first comment arrives, or once a conversion without comments
// completes, so an input that can't be read at all leaves the site as it was.
func importComments(ds Store, r io.Reader, siteID string, convert converter) (passed int, err error) {
	cleared, failed := false, 0
	clearSite := func() error {
		if cleared {
			return nil
		}
		cleared = true
		return ds.DeleteAll(siteID)
	}

	err = convert(r, siteID, func(c store.Comment) error {
		if e := clearSite(); e != nil {
			return e
		}
		if _, e := ds.Create(c); e != nil {
			failed++
			return nil
		}
		passed++
		return nil
	})
	if err == nil {
		err = clearSite()
	}
	if err != nil {
		return passed, err
	}

	if failed > 0 {
		err = fmt.Errorf("failed to save %d comments", failed)
		if passed == 0 {
			err = fmt.Errorf("import failed")
		}
	}

	log.Printf("[DEBUG] imported %d comments to site %s", passed, siteID)

	return passed, err
}

// ImportComments imports from given provider format and saves to store
func ImportComments(p ImportParams) (int, error) {
	log.Printf("[INFO] import from %s (%s) to %s", p.InputFile, p.Provider, p.SiteID)

	var importer Importer
	switch p.Provider {
	case "disqus":
		importer = &Disqus{DataStore: p.DataStore}
	case "wordpress":
		importer = &WordPress{DataStore: p.DataStore}
	case "commento":
		importer = &Commento{DataStore: p.DataStore}
	case "native":
		importer = &Native{DataStore: p.DataStore}
	default:
		return 0, fmt.Errorf("unsupported import provider %s", p.Provider)
	}

	fh, err := os.Open(p.InputFile)
	if err != nil {
		return 0, fmt.Errorf("can't open import file %s: %w", p.InputFile, err)
	}

	defer func() { //nolint:gosec // false positive on defer without error check when it's checked here
		if err = fh.Close(); err != nil {
			log.Printf("[WARN] can't close %s, %s", p.InputFile, err)
		}
	}()

	return importer.Import(fh, p.SiteID)
}
