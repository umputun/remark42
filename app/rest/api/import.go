package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/umputun/remark/app/migrator"
	"github.com/umputun/remark/app/rest"
)

// Import rest runs on unexposed port and available for local requests only
type Import struct {
	Version        string
	Cache          rest.LoadingCache
	NativeImporter migrator.Importer
	DisqusImporter migrator.Importer

	httpServer *http.Server
}

// Run the listener and request's router, activate rest server
// this server doesn't have any authentication and SHOULDN'T BE EXPOSED in any way
func (s *Import) Run(port int) {
	log.Printf("[INFO] activate import server on port %d", port)

	router := chi.NewRouter()
	router.Use(middleware.RealIP, Recoverer)
	router.Use(middleware.Throttle(1000), middleware.Timeout(60*time.Second))
	router.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))
	router.Use(AppInfo("remark42-importer", s.Version), Ping, Logger(LogAll))

	router.Post("/api/v1/admin/import", s.importCtrl)

	s.httpServer = &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", port), Handler: router}
	err := s.httpServer.ListenAndServe()
	log.Printf("[WARN] http server terminated, %s", err)
}

// POST /import?site=site-id&provider=disqus|remark
// imports comments from post body.
func (s *Import) importCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	importer := s.NativeImporter
	if r.URL.Query().Get("provider") == "disqus" {
		importer = s.DisqusImporter
	}

	if err := importer.Import(r.Body, siteID); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "import failed")
		return
	}
	s.Cache.Flush()
}
