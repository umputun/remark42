package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

// sslMode defines ssl mode for rest server
type sslMode int8

const (
	// None defines to run http server only
	None sslMode = iota

	// Static defines to run both https and http server. Redirect http to https
	Static

	// Auto defines to run both https and http server. Redirect http to https. Https server with autocert support
	Auto
)

// SSLConfig holds all ssl params for rest server
type SSLConfig struct {
	SSLMode sslMode
	Cert    string
	Key     string
	Port    int
}

// httpToHTTPSRouter creates new router which does redirect from http to https server
func (s *Rest) httpToHTTPSRouter() chi.Router {
	router := chi.NewRouter()
	router.Use(middleware.RealIP, Recoverer)
	router.Use(middleware.Throttle(1000), middleware.Timeout(60*time.Second))

	router.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		newURL := s.RemarkURL + r.URL.Path
		if r.URL.RawQuery != "" {
			newURL += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, newURL, http.StatusTemporaryRedirect)
	})
	return router
}
