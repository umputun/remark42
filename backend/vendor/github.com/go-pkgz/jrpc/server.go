package jrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/logger"
)

// Server is json-rpc server with an optional basic auth
type Server struct {
	api string // url path, i.e. "/command" or "/rpc" etc., required

	authUser          string      // basic auth user name, should match Client.AuthUser, optional
	authPasswd        string      // basic auth password, should match Client.AuthPasswd, optional
	customMiddlewares middlewares // list of custom middlewares, should match array of http.Handler func, optional

	signature signaturePayload // add server signature to server response headers appName, author, version), disable by default

	timeouts Timeouts // values and timeouts for the server
	limits   limits   // values and limits for the server
	logger   L        // logger, if nil will default to NoOpLogger

	funcs struct {
		m    map[string]ServerFn
		once sync.Once
	}

	httpServer struct {
		*http.Server
		sync.Mutex
	}
}

// Timeouts includes values and timeouts for the server
type Timeouts struct {
	ReadHeaderTimeout time.Duration // amount of time allowed to read request headers
	WriteTimeout      time.Duration // max duration before timing out writes of the response
	IdleTimeout       time.Duration // max amount of time to wait for the next request when keep-alive enabled
	CallTimeout       time.Duration // max time allowed to finish the call, optional
}

// limits includes limits values for a server
type limits struct {
	serverThrottle int     // max number of parallel calls for the server
	clientLimit    float64 // max number of call/sec per client
}

// signaturePayload is the server application info which add to server response headers
type signaturePayload struct {
	appName string // server version, injected from main and used for informational headers only
	author  string // plugin name, injected from main and used for informational headers only
	version string // custom application server number
}

// ServerFn handler registered for each method with Add or Group.
// Implementations provided by consumer and defines response logic.
type ServerFn func(id uint64, params json.RawMessage) Response

// middlewares contains list of custom middlewares which user can attach to server
type middlewares []func(http.Handler) http.Handler

// NewServer the main constructor of server instance which pass API url and another options values
func NewServer(api string, options ...Option) *Server {

	srv := &Server{
		api:      api,
		timeouts: getDefaultTimeouts(),
		logger:   NoOpLogger,
	}

	for _, opt := range options {
		opt(srv)
	}
	return srv
}

// Run http server on given port
func (s *Server) Run(port int) error {

	if s.authUser == "" || s.authPasswd == "" {
		s.logger.Logf("[WARN] extension server runs without auth")
	}

	if s.funcs.m == nil && len(s.funcs.m) == 0 {
		return fmt.Errorf("nothing mapped for dispatch, Add has to be called prior to Run")
	}

	router := chi.NewRouter()

	if s.limits.serverThrottle > 0 {
		router.Use(middleware.Throttle(s.limits.serverThrottle))
	}

	router.Use(middleware.RealIP, rest.Ping, rest.Recoverer(s.logger))

	if s.signature.version != "" || s.signature.author != "" || s.signature.appName != "" {
		router.Use(rest.AppInfo(s.signature.appName, s.signature.author, s.signature.version))
	}

	if s.timeouts.CallTimeout > 0 {
		router.Use(middleware.Timeout(s.timeouts.CallTimeout))
	}

	logInfoWithBody := logger.New(logger.Log(s.logger), logger.WithBody, logger.Prefix("[DEBUG]")).Handler
	router.Use(logInfoWithBody)

	if s.limits.clientLimit > 0 {
		router.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(s.limits.clientLimit, nil)))
	}

	router.Use(middleware.NoCache)
	router.Use(s.basicAuth)
	router.Use(s.customMiddlewares...)
	router.Post(s.api, s.handler)

	s.httpServer.Lock()
	s.httpServer.Server = &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           router,
		ReadHeaderTimeout: s.timeouts.ReadHeaderTimeout,
		WriteTimeout:      s.timeouts.WriteTimeout,
		IdleTimeout:       s.timeouts.IdleTimeout,
	}
	s.httpServer.Unlock()

	s.logger.Logf("[INFO] listen on %d", port)
	return s.httpServer.ListenAndServe()
}

// Shutdown http server
func (s *Server) Shutdown() error {
	s.httpServer.Lock()
	defer s.httpServer.Unlock()
	if s.httpServer.Server == nil {
		return fmt.Errorf("http server is not running")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

// Add method handler. Handler will be called on matching method (Request.Method)
func (s *Server) Add(method string, fn ServerFn) {
	s.httpServer.Lock()
	defer s.httpServer.Unlock()
	if s.httpServer.Server != nil {
		s.logger.Logf("[WARN] ignored method %s, can't be added to activated server", method)
		return
	}

	s.funcs.once.Do(func() {
		s.funcs.m = map[string]ServerFn{}
	})

	s.funcs.m[method] = fn
	s.logger.Logf("[INFO] add handler for %s", method)
}

// HandlersGroup alias for map of handlers
type HandlersGroup map[string]ServerFn

// Group of handlers with common prefix, match on group.method
func (s *Server) Group(prefix string, m HandlersGroup) {
	for k, v := range m {
		s.Add(prefix+"."+k, v)
	}
}

// handler is http handler multiplexing calls by req.Method
func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	req := struct {
		ID     uint64           `json:"id"`
		Method string           `json:"method"`
		Params *json.RawMessage `json:"params"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rest.SendErrorJSON(w, r, s.logger, http.StatusBadRequest, err, req.Method)
		return
	}
	fn, ok := s.funcs.m[req.Method]
	if !ok {
		rest.SendErrorJSON(w, r, s.logger, http.StatusNotImplemented, fmt.Errorf("unsupported method"), req.Method)
		return

	}

	params := json.RawMessage{}
	if req.Params != nil {
		params = *req.Params
	}

	render.JSON(w, r, fn(req.ID, params))
}

// basicAuth middleware. enabled only if both AuthUser and AuthPasswd defined.
func (s *Server) basicAuth(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if s.authUser == "" || s.authPasswd == "" {
			h.ServeHTTP(w, r)
			return
		}

		user, pass, ok := r.BasicAuth()
		if user != s.authUser || pass != s.authPasswd || !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func getDefaultTimeouts() Timeouts {
	return Timeouts{
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       5 * time.Second,
	}
}

// L defined logger interface used for an optional rest logging
type L interface {
	Logf(format string, args ...interface{})
}

// LoggerFunc type is an adapter to allow the use of ordinary functions as Logger.
type LoggerFunc func(format string, args ...interface{})

// Logf calls f(id)
func (f LoggerFunc) Logf(format string, args ...interface{}) { f(format, args...) }

// NoOpLogger logger does nothing
var NoOpLogger = LoggerFunc(func(format string, args ...interface{}) {})
