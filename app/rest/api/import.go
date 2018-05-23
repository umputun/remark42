package api

import (
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

	"github.com/umputun/remark/app/migrator"
	"github.com/umputun/remark/app/rest"
)

// Import rest runs on unexposed port and available for local requests only
type Import struct {
	Version        string
	Cache          rest.LoadingCache
	NativeImporter migrator.Importer
	DisqusImporter migrator.Importer
	SecretKey      string

	httpServer *http.Server
	lock       sync.Mutex
}

// Run the listener and request's router, activate rest server
// this server doesn't have any authentication and SHOULDN'T BE EXPOSED in any way
func (s *Import) Run(port int) {
	log.Printf("[INFO] activate import server on port %d", port)
	router := s.routes()

	s.lock.Lock()
	s.httpServer = &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", port), Handler: router}
	s.lock.Unlock()

	err := s.httpServer.ListenAndServe()
	log.Printf("[WARN] http server terminated, %s", err)
}

// Shutdown import http server
func (s *Import) Shutdown() {
	log.Print("[WARN] shutdown import server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	s.lock.Lock()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("[DEBUG] importer shutdown error, %s", err)
	}
	s.lock.Unlock()

	log.Print("[DEBUG] shutdown import server completed")
}

func (s Import) routes() chi.Router {
	router := chi.NewRouter()
	router.Use(middleware.RealIP, Recoverer)
	router.Use(middleware.Throttle(1000), middleware.Timeout(60*time.Second))
	router.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))
	router.Use(AppInfo("remark42-importer", s.Version), Ping, Logger(LogAll))
	router.Post("/api/v1/admin/import", s.importCtrl)
	return router
}

// POST /import?secret=key&site=site-id&provider=disqus|remark
// imports comments from post body.
func (s *Import) importCtrl(w http.ResponseWriter, r *http.Request) {

	secret := r.URL.Query().Get("secret")
	if strings.TrimSpace(secret) == "" || secret != s.SecretKey {
		render.Status(r, http.StatusForbidden)
		render.JSON(w, r, JSON{"status": "error", "details": "secret key"})
		return
	}

	siteID := r.URL.Query().Get("site")
	importer := s.NativeImporter
	if r.URL.Query().Get("provider") == "disqus" {
		importer = s.DisqusImporter
	}

	size, err := importer.Import(r.Body, siteID)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "import failed")
		return
	}
	s.Cache.Flush()

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, JSON{"status": "ok", "size": size})
}
