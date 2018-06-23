package api

import (
	"compress/gzip"
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"

	"github.com/umputun/remark/backend/app/migrator"
	"github.com/umputun/remark/backend/app/rest"
	"github.com/umputun/remark/backend/app/rest/cache"
)

// Migrator rest runs on unexposed port and available for local requests only
type Migrator struct {
	Version        string
	Cache          cache.LoadingCache
	NativeImporter migrator.Importer
	DisqusImporter migrator.Importer
	NativeExported migrator.Exporter
	SecretKey      string

	httpServer *http.Server
	lock       sync.Mutex
}

// Run the listener and request's router, activate rest server
// this server doesn't have any authentication and SHOULDN'T BE EXPOSED in any way
func (m *Migrator) Run(port int) {
	log.Printf("[INFO] activate import server on port %d", port)
	router := m.routes()

	m.lock.Lock()
	m.httpServer = &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", port), Handler: router}
	m.lock.Unlock()

	err := m.httpServer.ListenAndServe()
	log.Printf("[WARN] http server terminated, %s", err)
}

// Shutdown import http server
func (m *Migrator) Shutdown() {
	log.Print("[WARN] shutdown import server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	m.lock.Lock()
	if m.httpServer != nil {
		if err := m.httpServer.Shutdown(ctx); err != nil {
			log.Printf("[DEBUG] importer shutdown error, %s", err)
		}
	}
	m.lock.Unlock()

	log.Print("[DEBUG] shutdown import server completed")
}

func (m *Migrator) routes() chi.Router {
	router := chi.NewRouter()
	router.Use(middleware.RealIP, Recoverer)
	router.Use(middleware.Throttle(1000), middleware.Timeout(15*time.Minute))
	router.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))
	router.Use(AppInfo("remark42-migrator", m.Version), Ping, Logger(nil, LogAll))
	router.Post("/api/v1/admin/import", m.importCtrl)
	router.Get("/api/v1/admin/export", m.exportCtrl)
	return router
}

// POST /import?secret=key&site=site-id&provider=disqus|remark
// imports comments from post body.
func (m *Migrator) importCtrl(w http.ResponseWriter, r *http.Request) {

	secret := r.URL.Query().Get("secret")
	if strings.TrimSpace(secret) == "" || secret != m.SecretKey {
		render.Status(r, http.StatusForbidden)
		render.JSON(w, r, JSON{"status": "error", "details": "secret key"})
		return
	}

	siteID := r.URL.Query().Get("site")
	importer := m.NativeImporter
	if r.URL.Query().Get("provider") == "disqus" {
		importer = m.DisqusImporter
	}

	size, err := importer.Import(r.Body, siteID)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "import failed")
		return
	}
	m.Cache.Flush(siteID)

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, JSON{"status": "ok", "size": size})
}

// GET /export?site=site-id&secret=12345
// exports all comments for siteID as gz file
func (m *Migrator) exportCtrl(w http.ResponseWriter, r *http.Request) {

	secret := r.URL.Query().Get("secret")
	if strings.TrimSpace(secret) == "" || secret != m.SecretKey {
		render.Status(r, http.StatusForbidden)
		render.JSON(w, r, JSON{"status": "error", "details": "secret key"})
		return
	}

	siteID := r.URL.Query().Get("site")

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

	if _, err := m.NativeExported.Export(gzWriter, siteID); err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "export failed")
		return
	}
}
