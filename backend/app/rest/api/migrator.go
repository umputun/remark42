package api

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"

	"github.com/umputun/remark/backend/app/migrator"
	"github.com/umputun/remark/backend/app/rest"
	"github.com/umputun/remark/backend/app/rest/cache"
)

// Migrator rest with import and export controllers
type Migrator struct {
	Cache             cache.LoadingCache
	NativeImporter    migrator.Importer
	DisqusImporter    migrator.Importer
	WordPressImporter migrator.Importer
	NativeExported    migrator.Exporter
	KeyStore          KeyStore
}

// KeyStore defines sub-interface for consumers needed just a key
type KeyStore interface {
	Key(siteID string) (key string, err error)
}

func (m *Migrator) withRoutes(router chi.Router) chi.Router {
	router.Get("/export", m.exportCtrl)
	router.Post("/import", m.importCtrl)
	return router
}

// POST /import?secret=key&site=site-id&provider=disqus|remark|wordpress
// imports comments from post body.
func (m *Migrator) importCtrl(w http.ResponseWriter, r *http.Request) {

	siteID := r.URL.Query().Get("site")

	var importer migrator.Importer
	switch r.URL.Query().Get("provider") {
	case "disqus":
		importer = m.DisqusImporter
	case "wordpress":
		importer = m.WordPressImporter
	default:
		importer = m.NativeImporter
	}

	log.Printf("[DEBUG] import request for site=%s, provider=%s", siteID, r.URL.Query().Get("provider"))
	size, err := importer.Import(r.Body, siteID)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "import failed")
		return
	}
	m.Cache.Flush(cache.Flusher(siteID).Scopes(siteID))

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, JSON{"status": "ok", "size": size})
}

// GET /export?site=site-id&secret=12345&?mode=file|stream
// exports all comments for siteID as gz file
func (m *Migrator) exportCtrl(w http.ResponseWriter, r *http.Request) {

	siteID := r.URL.Query().Get("site")

	var writer io.Writer = w
	if r.URL.Query().Get("mode") == "file" {
		exportFile := fmt.Sprintf("%s-%s.json.gz", siteID, time.Now().Format("20060102"))
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", "attachment;filename="+exportFile)
		w.WriteHeader(http.StatusOK)
		gzWriter := gzip.NewWriter(w)
		defer func() {
			if e := gzWriter.Close(); e != nil {
				log.Printf("[WARN] can't close gzip writer, %s", e)
			}
		}()
		writer = gzWriter
	}

	if _, err := m.NativeExported.Export(writer, siteID); err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "export failed")
		return
	}
}
