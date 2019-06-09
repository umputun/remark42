package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/pkg/errors"
)

type Server struct {
	CommandURL string
	AuthUser   string
	AuthPasswd string

	funcs struct {
		m    map[string]ServerFn
		once sync.Once
	}

	httpServer struct {
		*http.Server
		sync.Mutex
	}
}

type ServerFn func(params *json.RawMessage) Response

func (s *Server) Run(port int) error {
	if s.funcs.m == nil && len(s.funcs.m) == 0 {
		return errors.Errorf("nothing mapped for dispatch, Add has to be called prior to Run")
	}

	router := chi.NewRouter()

	type request struct {
		Method string           `json:"method"`
		Params *json.RawMessage `json:"params"`
	}

	router.Post(s.CommandURL, func(w http.ResponseWriter, r *http.Request) {
		req := request{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		fn, ok := s.funcs.m[req.Method]
		if !ok {
			w.WriteHeader(http.StatusNotImplemented)
			return
		}
		render.JSON(w, r, fn(req.Params))
	})

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

func (s *Server) EncodeResponse(resp interface{}) (Response, error) {
	v, err := json.Marshal(&resp)
	if err != nil {
		return Response{}, err
	}
	raw := json.RawMessage{}
	if err = raw.UnmarshalJSON(v); err != nil {
		return Response{}, err
	}
	return Response{Result: &raw}, nil
}

func (s *Server) Shutdown() error {
	s.httpServer.Lock()
	defer s.httpServer.Unlock()
	if s.httpServer.Server == nil {
		return errors.Errorf("http server is not running")
	}
	return s.httpServer.Shutdown(context.TODO())
}

func (s *Server) Add(method string, fn ServerFn) {
	s.funcs.once.Do(func() {
		s.funcs.m = map[string]ServerFn{}
	})
	s.funcs.m[method] = fn
}
