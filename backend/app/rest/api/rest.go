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
	"github.com/rakyll/statik/fs"

	"github.com/go-pkgz/auth"
	R "github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/cache"
	"github.com/go-pkgz/rest/logger"

	"github.com/umputun/remark/backend/app/notify"
	"github.com/umputun/remark/backend/app/rest"
	"github.com/umputun/remark/backend/app/rest/proxy"
	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/service"
)

// Rest is a rest access server
type Rest struct {
	Version string

	DataService      *service.DataStore
	Authenticator    *auth.Service
	Cache            cache.LoadingCache
	ImageProxy       *proxy.Image
	CommentFormatter *store.CommentFormatter
	Migrator         *Migrator
	NotifyService    *notify.Service

	WebRoot         string
	RemarkURL       string
	ReadOnlyAge     int
	SharedSecret    string
	ScoreThresholds struct {
		Low      int
		Critical int
	}

	SSLConfig   SSLConfig
	httpsServer *http.Server
	httpServer  *http.Server
	lock        sync.Mutex

	adminService admin
}

const hardBodyLimit = 1024 * 64 // limit size of body

const lastCommentsScope = "last"

type commentsWithInfo struct {
	Comments []store.Comment `json:"comments"`
	Info     store.PostInfo  `json:"info,omitempty"`
}

// Run the lister and request's router, activate rest server
func (s *Rest) Run(port int) {
	switch s.SSLConfig.SSLMode {
	case None:
		log.Printf("[INFO] activate http rest server on port %d", port)

		s.lock.Lock()
		s.httpServer = s.makeHTTPServer(port, s.routes())
		s.lock.Unlock()

		err := s.httpServer.ListenAndServe()
		log.Printf("[WARN] http server terminated, %s", err)
	case Static:
		log.Printf("[INFO] activate https server in 'static' mode on port %d", s.SSLConfig.Port)

		s.lock.Lock()
		s.httpsServer = s.makeHTTPSServer(s.SSLConfig.Port, s.routes())
		s.httpServer = s.makeHTTPServer(port, s.httpToHTTPSRouter())
		s.lock.Unlock()

		go func() {
			log.Printf("[INFO] activate http redirect server on port %d", port)
			err := s.httpServer.ListenAndServe()
			log.Printf("[WARN] http redirect server terminated, %s", err)
		}()

		err := s.httpsServer.ListenAndServeTLS(s.SSLConfig.Cert, s.SSLConfig.Key)
		log.Printf("[WARN] https server terminated, %s", err)
	case Auto:
		log.Printf("[INFO] activate https server in 'auto' mode on port %d", s.SSLConfig.Port)

		m := s.makeAutocertManager()
		s.lock.Lock()
		s.httpsServer = s.makeHTTPSAutocertServer(s.SSLConfig.Port, s.routes(), m)
		s.httpServer = s.makeHTTPServer(port, s.httpChallengeRouter(m))
		s.lock.Unlock()

		go func() {
			log.Printf("[INFO] activate http challenge server on port %d", port)

			err := s.httpServer.ListenAndServe()
			log.Printf("[WARN] http challenge server terminated, %s", err)
		}()

		err := s.httpsServer.ListenAndServeTLS("", "")
		log.Printf("[WARN] https server terminated, %s", err)
	}
}

// Shutdown rest http server
func (s *Rest) Shutdown() {
	log.Print("[WARN] shutdown rest server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	s.lock.Lock()
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("[DEBUG] http shutdown error, %s", err)
		}
		log.Print("[DEBUG] shutdown http server completed")
	}

	if s.httpsServer != nil {
		log.Print("[WARN] shutdown https server")
		if err := s.httpsServer.Shutdown(ctx); err != nil {
			log.Printf("[DEBUG] https shutdown error, %s", err)
		}
		log.Print("[DEBUG] shutdown https server completed")
	}
	s.lock.Unlock()
}

func (s *Rest) makeHTTPServer(port int, router http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      120 * time.Second, // TODO: such a long timeout needed for blocking export (backup) request
		IdleTimeout:       30 * time.Second,
	}
}

func (s *Rest) routes() chi.Router {
	router := chi.NewRouter()
	router.Use(middleware.RealIP, R.Recoverer)
	router.Use(middleware.Throttle(1000), middleware.Timeout(60*time.Second))
	router.Use(R.AppInfo("remark42", "umputun", s.Version), R.Ping)

	s.adminService = admin{
		dataService:   s.DataService,
		migrator:      s.Migrator,
		cache:         s.Cache,
		authenticator: s.Authenticator,
		readOnlyAge:   s.ReadOnlyAge,
	}

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-XSRF-Token", "X-JWT"},
		ExposedHeaders:   []string{"Authorization"},
		AllowCredentials: true,
		MaxAge:           300,
	})
	router.Use(corsMiddleware.Handler)

	ipFn := func(ip string) string { return store.HashValue(ip, s.SharedSecret)[:12] } // logger uses it for anonymization

	authHandler, avatarHandler := s.Authenticator.Handlers()

	router.Group(func(r chi.Router) {
		l := logger.New(logger.Flags(logger.All), logger.IPfn(ipFn))
		r.Use(l.Handler, tollbooth_chi.LimitHandler(tollbooth.NewLimiter(5, nil)))
		r.Mount("/auth", authHandler)
	})

	router.Group(func(r chi.Router) {
		r.Use(logger.New(logger.Flags(logger.None)).Handler, tollbooth_chi.LimitHandler(tollbooth.NewLimiter(100, nil)))
		r.Mount("/avatar", avatarHandler)
	})

	authMiddleware := s.Authenticator.Middleware()

	//// auth routes for all providers
	//router.Route("/auth", func(r chi.Router) {
	//	l := logger.New(logger.Flags(logger.All), logger.IPfn(ipFn))
	//	r.Use(l.Handler, tollbooth_chi.LimitHandler(tollbooth.NewLimiter(5, nil)))
	//
	//	for _, provider := range s.Authenticator.Providers {
	//		r.Mount("/"+provider.Name, provider.Routes()) // mount auth providers as /auth/{name}
	//	}
	//	if len(s.Authenticator.Providers) > 0 {
	//		// shortcut, can be any of providers, all logouts do the same - removes cookie
	//		r.Get("/logout", s.Authenticator.Providers[0].LogoutHandler)
	//	}
	//})

	//avatarMiddlewares := []func(http.Handler) http.Handler{
	//	logger.New(logger.Flags(logger.None)).Handler,
	//	tollbooth_chi.LimitHandler(tollbooth.NewLimiter(100, nil)),
	//}
	//router.Mount(s.AvatarProxy.Routes(avatarMiddlewares...)) // mount avatars to /api/v1/avatar/{file.img}

	// api routes
	router.Route("/api/v1", func(rapi chi.Router) {

		rapi.Group(func(rava chi.Router) {
			rava.Use(logger.New(logger.Flags(logger.None)).Handler, tollbooth_chi.LimitHandler(tollbooth.NewLimiter(100, nil)))
			rava.Mount("/avatar", avatarHandler)
		})

		// open routes
		rapi.Group(func(ropen chi.Router) {
			ropen.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))
			ropen.Use(authMiddleware.Trace)
			ropen.Use(logger.New(logger.Flags(logger.All), logger.IPfn(ipFn)).Handler)
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
			rauth.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))
			rauth.Use(authMiddleware.Auth)
			rauth.Use(logger.New(logger.Flags(logger.All), logger.IPfn(ipFn)).Handler)
			rauth.Post("/comment", s.createCommentCtrl)
			rauth.Put("/comment/{id}", s.updateCommentCtrl)
			rauth.Get("/user", s.userInfoCtrl)
			rauth.Put("/vote/{id}", s.voteCtrl)
			rauth.Get("/userdata", s.userAllDataCtrl)
			rauth.Post("/deleteme", s.deleteMeCtrl)

			// admin routes, admin users only
			rauth.Mount("/admin", s.adminService.routes(authMiddleware.AdminOnly))
		})
	})

	// respond to /robots.txt with the list of allowed paths
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

// serves static files from /web or embedded by statik
func addFileServer(r chi.Router, path string, root http.FileSystem) {

	var webFS http.Handler

	statikFS, err := fs.New()
	if err == nil {
		log.Printf("[INFO] run file server for %s, embedded", root)
		webFS = http.FileServer(statikFS)
	}
	if err != nil {
		log.Printf("[DEBUG] no embedded assets loaded, %s", err)
		log.Printf("[INFO] run file server for %s, path %s", root, path)
		webFS = http.FileServer(root)
	}

	origPath := path
	webFS = http.StripPrefix(path, webFS)
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
			webFS.ServeHTTP(w, r)
		}))
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

func filterComments(comments []store.Comment, fn func(c store.Comment) bool) (filtered []store.Comment) {
	for _, c := range comments {
		if fn(c) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// URLKey gets url from request to use it as cache key
// admins will have different keys in order to prevent leak of admin-only data to regular users
func URLKey(r *http.Request) string {
	adminPrefix := "admin!!"
	key := strings.TrimPrefix(r.URL.String(), adminPrefix)          // prevents attach with fake url to get admin view
	if user, err := rest.GetUserInfo(r); err == nil && user.Admin { // make separate cache key for admins
		key = adminPrefix + key
	}
	return key
}
