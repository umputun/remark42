package rest

import (
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"
)

// Profiler is a convenient subrouter used for mounting net/http/pprof. ie.
//
//	func MyService() http.Handler {
//	  r := chi.NewRouter()
//	  // ..middlewares
//	  r.Mount("/debug", middleware.Profiler())
//	  // ..routes
//	  return r
//	}
func Profiler(onlyIps ...string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/pprof/", pprof.Index)
	mux.HandleFunc("/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/pprof/profile", pprof.Profile)
	mux.HandleFunc("/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/pprof/trace", pprof.Trace)
	mux.Handle("/pprof/block", pprof.Handler("block"))
	mux.Handle("/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.HandleFunc("/vars", expVars)

	return Wrap(mux, NoCache, OnlyFrom(onlyIps...))
}

// expVars copied from stdlib expvar.go as is not public.
func expVars(w http.ResponseWriter, _ *http.Request) {
	first := true
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "{\n")
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")
}
