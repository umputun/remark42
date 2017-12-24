package migrator

import "io"

// Importer defines interface to convert posts from external sources
type Importer interface {
	Import(r io.Reader, siteID string) error
}

// Exporter defines interface to export comments from internal store
type Exporter interface {
	Export(w io.Writer, siteID string) error
}
