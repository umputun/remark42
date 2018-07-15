package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/pkg/errors"
	"gopkg.in/russross/blackfriday.v2"

	"github.com/umputun/remark/backend/app/migrator"
	"github.com/umputun/remark/backend/app/rest"
	"github.com/umputun/remark/backend/app/rest/auth"
	"github.com/umputun/remark/backend/app/rest/cache"
	"github.com/umputun/remark/backend/app/rest/proxy"
	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/service"
)

// Rest is a rest access server
type Rest struct {
	Version         string
	DataService     *service.DataStore
	Authenticator   auth.Authenticator
	Exporter        migrator.Exporter
	Cache           cache.LoadingCache
	AvatarProxy     *proxy.Avatar
	ImageProxy      *proxy.Image
	WebRoot         string
	RemarkURL       string
	ReadOnlyAge     int
	ScoreThresholds struct {
		Low      int
		Critical int
	}

	httpServer *http.Server
	lock       sync.Mutex

	adminService admin
}

const hardBodyLimit = 1024 * 64 // limit size of body

var mdExt = blackfriday.NoIntraEmphasis | blackfriday.Tables | blackfriday.FencedCode |
	blackfriday.Strikethrough | blackfriday.SpaceHeadings | blackfriday.HardLineBreak |
	blackfriday.BackslashLineBreak | blackfriday.Autolink

type commentsWithInfo struct {
	Comments []store.Comment `json:"comments"`
	Info     store.PostInfo  `json:"info,omitempty"`
}

// Run the lister and request's router, activate rest server
func (s *Rest) Run(port int) {
	log.Printf("[INFO] activate rest server on port %d", port)

	if s.DataService != nil && len(s.DataService.Admins) > 0 {
		log.Printf("[DEBUG] admins %+v", s.DataService.Admins)
	}

	router := s.routes()

	s.lock.Lock()
	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	s.lock.Unlock()

	err := s.httpServer.ListenAndServe()
	log.Printf("[WARN] http server terminated, %s", err)
}

// Shutdown rest http server
func (s *Rest) Shutdown() {
	log.Print("[WARN] shutdown rest server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	s.lock.Lock()
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("[DEBUG] rest shutdown error, %s", err)
		}
	}
	log.Print("[DEBUG] shutdown rest server completed")
	s.lock.Unlock()
}

func (s *Rest) routes() chi.Router {
	router := chi.NewRouter()
	router.Use(middleware.RealIP, Recoverer)
	router.Use(middleware.Throttle(1000), middleware.Timeout(60*time.Second))
	router.Use(AppInfo("remark42", s.Version), Ping)

	s.adminService = admin{
		dataService:   s.DataService,
		exporter:      s.Exporter,
		cache:         s.Cache,
		authenticator: s.Authenticator,
		readOnlyAge:   s.ReadOnlyAge,
	}

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-XSRF-Token", "X-JWT"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})
	router.Use(corsMiddleware.Handler)

	ipFn := func(ip string) string { return store.HashValue(ip, s.DataService.Secret)[:12] } // logger uses it for anonymization

	// auth routes for all providers
	router.Route("/auth", func(r chi.Router) {
		r.Use(Logger(ipFn, LogAll), tollbooth_chi.LimitHandler(tollbooth.NewLimiter(5, nil)))
		for _, provider := range s.Authenticator.Providers {
			r.Mount("/"+provider.Name, provider.Routes()) // mount auth providers as /auth/{name}
		}
		if len(s.Authenticator.Providers) > 0 {
			// shortcut, can be any of providers, all logouts do the same - removes cookie
			r.Get("/logout", s.Authenticator.Providers[0].LogoutHandler)
		}
	})

	avatarMiddlewares := []func(http.Handler) http.Handler{
		Logger(ipFn, LogNone),
		tollbooth_chi.LimitHandler(tollbooth.NewLimiter(100, nil)),
	}
	router.Mount(s.AvatarProxy.Routes(avatarMiddlewares...)) // mount avatars to /api/v1/avatar/{file.img}

	// api routes
	router.Route("/api/v1", func(rapi chi.Router) {
		rapi.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))

		// open routes
		rapi.Group(func(ropen chi.Router) {
			ropen.Use(s.Authenticator.Auth(false))
			ropen.Use(Logger(ipFn, LogAll))
			ropen.Get("/find", s.findCommentsCtrl)
			ropen.Get("/id/{id}", s.commentByIDCtrl)
			ropen.Get("/comments", s.findUserCommentsCtrl)
			ropen.Get("/last/{limit}", s.lastCommentsCtrl)
			ropen.Get("/count", s.countCtrl)
			ropen.Post("/counts", s.countMultiCtrl)
			ropen.Get("/list", s.listCtrl)
			ropen.Get("/config", s.configCtrl)
			ropen.Post("/preview", s.previewCommentCtrl)
			ropen.Get("/info", s.infoCtrl)

			ropen.Mount("/rss", s.rssRoutes())
			ropen.Mount("/img", s.ImageProxy.Routes())
		})

		// protected routes, require auth
		rapi.Group(func(rauth chi.Router) {
			rauth.Use(s.Authenticator.Auth(true))
			rauth.Use(Logger(ipFn, LogAll))
			rauth.Post("/comment", s.createCommentCtrl)
			rauth.Put("/comment/{id}", s.updateCommentCtrl)
			rauth.Get("/user", s.userInfoCtrl)
			rauth.Put("/vote/{id}", s.voteCtrl)
			rauth.Get("/userdata", s.userAllDataCtrl)
			rauth.Post("/deleteme", s.deleteMeCtrl)

			// admin routes, admin users only
			rauth.Mount("/admin", s.adminService.routes(s.Authenticator.AdminOnly, Logger(nil, LogAll)))
		})
	})

	// respond to /robots.tx with the list of allowed paths
	router.With(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(50, nil))).
		Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
			allowed := []string{"/find", "/last", "/id", "/count", "/counts", "/list", "/config", "/img", "/avatar"}
			for i := range allowed {
				allowed[i] = "Allow: /api/v1" + allowed[i]
			}
			render.PlainText(w, r, "User-agent: *\nDisallow: /auth/\nDisallow: /api/\n"+strings.Join(allowed, "\n")+"\n")
		})

	// respond to /index.html with the content of getstarted.html under /web root
	router.With(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(50, nil))).
		Get("/index.html", func(w http.ResponseWriter, r *http.Request) {
			data, err := ioutil.ReadFile(path.Join(s.WebRoot, "getstarted.html"))
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			render.HTML(w, r, string(data))
		})

	// file server for static content from /web
	addFileServer(router, "/web", http.Dir(s.WebRoot))
	return router
}

// serves static files from /web
func addFileServer(r chi.Router, path string, root http.FileSystem) {
	log.Printf("[INFO] run file server for %s, path %s", root, path)
	origPath := path
	fs := http.StripPrefix(path, http.FileServer(root))
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.With(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(20, nil))).
		Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// don't show dirs, just serve files
			if strings.HasSuffix(r.URL.Path, "/") && len(r.URL.Path) > 1 && r.URL.Path != (origPath+"/") {
				http.NotFound(w, r)
				return
			}
			fs.ServeHTTP(w, r)
		}))
}

// renderJSONWithHTML allows html tags and forces charset=utf-8
func renderJSONWithHTML(w http.ResponseWriter, r *http.Request, v interface{}) {
	data, err := encodeJSONWithHTML(v)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't render json response")
		return
	}
	renderJSONFromBytes(w, r, data)
}

func encodeJSONWithHTML(v interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, errors.Wrap(err, "json encoding failed")
	}
	return buf.Bytes(), nil
}

// renderJSONWithHTML allows html tags and forces charset=utf-8
func renderJSONFromBytes(w http.ResponseWriter, r *http.Request, data []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if status, ok := r.Context().Value(render.StatusCtxKey).(int); ok {
		w.WriteHeader(status)
	}
	if _, err := w.Write(data); err != nil {
		log.Printf("[WARN] failed to send response to %s, %s", r.RemoteAddr, err)
	}
}

func filterComments(comments []store.Comment, fn func(c store.Comment) bool) (filtered []store.Comment) {
	for _, c := range comments {
		if fn(c) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}
