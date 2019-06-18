package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	log "github.com/go-pkgz/lgr"
	R "github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/logger"
	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/rest"
)

// Server is json-rpc server with an optional basic auth
type Server struct {
	API        string
	AuthUser   string
	AuthPasswd string
	Version    string
	AppName    string

	funcs struct {
		m    map[string]ServerFn
		once sync.Once
	}

	httpServer struct {
		*http.Server
		sync.Mutex
	}
}

// ServerFn handler registered for each method with Add
// Implementations provided by consumer and define response logic.
type ServerFn func(id uint64, params json.RawMessage) Response

// Run http server on given port
func (s *Server) Run(port int) error {
	if s.AuthUser == "" || s.AuthPasswd == "" {
		log.Print("[WARN] extension server runs without auth")
	}
	if s.funcs.m == nil && len(s.funcs.m) == 0 {
		return errors.Errorf("nothing mapped for dispatch, Add has to be called prior to Run")
	}

	router := chi.NewRouter()
	router.Use(middleware.Throttle(1000), middleware.RealIP, R.Recoverer(log.Default()))
	router.Use(R.AppInfo(s.AppName, "umputun", s.Version), R.Ping)
	logInfoWithBody := logger.New(logger.Log(log.Default()), logger.WithBody, logger.Prefix("[INFO]")).Handler
	router.Use(middleware.Timeout(5 * time.Second))
	router.Use(logInfoWithBody, tollbooth_chi.LimitHandler(tollbooth.NewLimiter(5, nil)), middleware.NoCache)
	router.Use(s.basicAuth)

	router.Post(s.API, s.handler)

	s.httpServer.Lock()
	s.httpServer.Server = &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	s.httpServer.Unlock()

	log.Printf("[INFO] listen on %d", port)
	return s.httpServer.ListenAndServe()
}

// EncodeResponse convert anything to Response
func (s *Server) EncodeResponse(id uint64, resp interface{}, e error) (Response, error) {
	v, err := json.Marshal(&resp)
	if err != nil {
		return Response{}, err
	}
	if e != nil {
		return Response{ID: id, Result: nil, Error: e.Error()}, nil
	}
	raw := json.RawMessage{}
	if err = raw.UnmarshalJSON(v); err != nil {
		return Response{}, err
	}
	return Response{ID: id, Result: &raw}, nil
}

// Shutdown http server
func (s *Server) Shutdown() error {
	s.httpServer.Lock()
	defer s.httpServer.Unlock()
	if s.httpServer.Server == nil {
		return errors.Errorf("http server is not running")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

// Add method handler
func (s *Server) Add(method string, fn ServerFn) {
	s.funcs.once.Do(func() {
		s.funcs.m = map[string]ServerFn{}
	})
	s.funcs.m[method] = fn
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	req := struct {
		ID     uint64           `json:"id"`
		Method string           `json:"method"`
		Params *json.RawMessage `json:"params"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, req.Method, 0)
		return
	}
	fn, ok := s.funcs.m[req.Method]
	if !ok {
		rest.SendErrorJSON(w, r, http.StatusNotImplemented, errors.New("unsupported method"), req.Method, 0)
		return
	}
	render.JSON(w, r, fn(req.ID, *req.Params))
}

func (s *Server) basicAuth(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if s.AuthUser == "" || s.AuthPasswd == "" {
			h.ServeHTTP(w, r)
			return
		}

		user, pass, ok := r.BasicAuth()
		if user != s.AuthUser || pass != s.AuthPasswd || !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}
