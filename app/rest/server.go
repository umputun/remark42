package rest

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

// Server is a rest access server
type Server struct {
	Version string
}

// Run the lister and request's router, activate rest server
func (s *Server) Run() {
	log.Print("[INFO] activate rest server")
	router := chi.NewRouter()
	router.Use(middleware.RealIP, Recoverer)
	router.Use(middleware.Throttle(100), middleware.Timeout(60*time.Second))
	router.Use(Limiter(10), AppInfo("remark", s.Version), Ping)

	router.Route("/blah", func(r chi.Router) {
		r.Get("/{id}", s.getBlahCtrl)
	})

	log.Fatal(http.ListenAndServe(":8080", router))
}

// GET /blah/:id?foo=bar
func (s *Server) getBlahCtrl(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	foo := r.URL.Query().Get("foo")
	log.Printf("[INFO] request for id=%s, foo=%s", id, foo)

	render.Status(r, http.StatusAccepted)
	render.JSON(w, r, JSON{"data": "something"})
}
