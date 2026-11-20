package api

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
	R "github.com/go-pkgz/rest"
	"github.com/go-pkgz/routegroup"
	"golang.org/x/crypto/acme/autocert"
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
	SSLMode      sslMode
	Cert         string
	Key          string
	Port         int
	ACMELocation string
	ACMEEmail    string
}

// httpRouter makes the router of the http port when https serves the api. In 'static' ssl mode it
// redirects to the https server. In 'auto' mode m answers the ACME "http-01" challenge, which is
// necessary to obtain a certificate from LE, and everything else is redirected.
func (s *Rest) httpRouter(m *autocert.Manager) http.Handler {
	router := routegroup.New(http.NewServeMux())
	router.Use(recoverer(log.Default()))
	router.Use(R.Throttle(1000), R.Timeout(60*time.Second))
	handler := s.redirectHandler()
	if m != nil {
		handler = m.HTTPHandler(handler)
	}
	router.Handle("/", handler)
	return router
}

func (s *Rest) redirectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		newURL, err := s.redirectURL(r)
		if err != nil {
			log.Printf("[WARN] failed to build redirect URL, %s", err)
			http.Error(w, "invalid redirect URL", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, newURL, http.StatusTemporaryRedirect)
	})
}

func (s *Rest) redirectURL(r *http.Request) (string, error) {
	baseURL, err := url.Parse(s.RemarkURL)
	if err != nil {
		return "", fmt.Errorf("parse remark URL: %w", err)
	}
	if baseURL.Scheme != "http" && baseURL.Scheme != "https" || baseURL.Host == "" {
		return "", fmt.Errorf("remark URL must be absolute HTTP(S) URL")
	}

	basePath := strings.TrimRight(baseURL.Path, "/")
	requestPath := "/" + strings.TrimLeft(r.URL.Path, "/")
	baseURL.Path = basePath + requestPath
	baseURL.RawQuery = r.URL.RawQuery
	baseURL.Fragment = ""
	return baseURL.String(), nil
}

func (s *Rest) makeAutocertManager() *autocert.Manager {
	return &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache(s.SSLConfig.ACMELocation),
		HostPolicy: autocert.HostWhitelist(s.getRemarkHost()),
		Email:      s.SSLConfig.ACMEEmail,
	}
}

// makeHTTPSServer makes https server, with autocert (LE support) when m is given
func (s *Rest) makeHTTPSServer(address string, port int, router http.Handler, m *autocert.Manager) *http.Server {
	server := s.makeHTTPServer(address, port, router)
	server.TLSConfig = s.makeTLSConfig()
	if m != nil {
		server.TLSConfig.GetCertificate = m.GetCertificate
	}
	return server
}

// getRemarkHost returns hostname for remark server.
// For example for remarkURL https://remark.com:443 it should return remark.com
func (s *Rest) getRemarkHost() string {
	u, err := url.Parse(s.RemarkURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func (s *Rest) makeTLSConfig() *tls.Config {
	return &tls.Config{
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		MinVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
			tls.CurveP384,
		},
	}
}
