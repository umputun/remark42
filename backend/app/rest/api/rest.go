package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-pkgz/auth"
	log "github.com/go-pkgz/lgr"
	R "github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/cache"
	"github.com/go-pkgz/rest/logger"
	"github.com/pkg/errors"
	"github.com/rakyll/statik/fs"

	"github.com/umputun/remark/backend/app/notify"
	"github.com/umputun/remark/backend/app/rest"
	"github.com/umputun/remark/backend/app/rest/proxy"
	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/image"
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
	ImageService     *image.Service

	WebRoot         string
	RemarkURL       string
	ReadOnlyAge     int
	SharedSecret    string
	ScoreThresholds struct {
		Low      int
		Critical int
	}
	UpdateLimiter float64

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
		s.httpServer.ErrorLog = log.ToStdLogger(log.Default(), "WARN")
		s.lock.Unlock()

		err := s.httpServer.ListenAndServe()
		log.Printf("[WARN] http server terminated, %s", err)
	case Static:
		log.Printf("[INFO] activate https server in 'static' mode on port %d", s.SSLConfig.Port)

		s.lock.Lock()
		s.httpsServer = s.makeHTTPSServer(s.SSLConfig.Port, s.routes())
		s.httpsServer.ErrorLog = log.ToStdLogger(log.Default(), "WARN")

		s.httpServer = s.makeHTTPServer(port, s.httpToHTTPSRouter())
		s.httpServer.ErrorLog = log.ToStdLogger(log.Default(), "WARN")
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
		s.httpsServer.ErrorLog = log.ToStdLogger(log.Default(), "WARN")

		s.httpServer = s.makeHTTPServer(port, s.httpChallengeRouter(m))
		s.httpServer.ErrorLog = log.ToStdLogger(log.Default(), "WARN")

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
	router.Use(middleware.RealIP, R.Recoverer(log.Default()))
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
		l := logger.New(logger.Log(log.Default()), logger.WithBody, logger.IPfn(ipFn), logger.Prefix("[INFO]"))
		r.Use(l.Handler, tollbooth_chi.LimitHandler(tollbooth.NewLimiter(5, nil)), middleware.NoCache)
		r.Mount("/auth", authHandler)
	})

	router.Group(func(r chi.Router) {
		r.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(100, nil)), middleware.NoCache)
		r.Mount("/avatar", avatarHandler)
	})

	authMiddleware := s.Authenticator.Middleware()

	// api routes
	router.Route("/api/v1", func(rapi chi.Router) {

		rapi.Group(func(rava chi.Router) {
			rava.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(100, nil)))
			rava.Use(middleware.NoCache)
			rava.Mount("/avatar", avatarHandler)
		})

		// open routes
		rapi.Group(func(ropen chi.Router) {
			ropen.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))
			ropen.Use(authMiddleware.Trace)
			ropen.Use(middleware.NoCache)
			ropen.Use(logger.New(logger.Log(log.Default()), logger.WithBody,
				logger.Prefix("[INFO]"), logger.IPfn(ipFn)).Handler)
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
			ropen.Get("/img", s.ImageProxy.Handler)

			ropen.Route("/rss", func(rrss chi.Router) {
				rrss.Get("/post", s.rssPostCommentsCtrl)
				rrss.Get("/site", s.rssSiteCommentsCtrl)
				rrss.Get("/reply", s.rssRepliesCtrl)
			})
		})

		// open routes, cached
		rapi.Group(func(ropen chi.Router) {
			ropen.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))
			ropen.Use(authMiddleware.Trace)
			ropen.Use(logger.New(logger.Log(log.Default()), logger.WithBody,
				logger.Prefix("[INFO]"), logger.IPfn(ipFn)).Handler)
			ropen.Get("/picture/{user}/{id}", s.loadPictureCtrl)
		})

		// protected routes, require auth
		rapi.Group(func(rauth chi.Router) {
			rauth.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))
			rauth.Use(authMiddleware.Auth)
			rauth.Use(middleware.NoCache)
			rauth.Use(logger.New(logger.Log(log.Default()), logger.WithBody,
				logger.Prefix("[INFO]"), logger.IPfn(ipFn)).Handler)
			rauth.Get("/user", s.userInfoCtrl)
			rauth.Get("/userdata", s.userAllDataCtrl)
		})

		// admin routes, require auth and admin users only
		rapi.Route("/admin", func(radmin chi.Router) {
			radmin.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))
			radmin.Use(authMiddleware.Auth, authMiddleware.AdminOnly)
			radmin.Use(middleware.NoCache)
			radmin.Use(logger.New(logger.Log(log.Default()), logger.WithBody,
				logger.Prefix("[INFO]"), logger.IPfn(ipFn)).Handler)

			radmin.Delete("/comment/{id}", s.adminService.deleteCommentCtrl)
			radmin.Put("/user/{userid}", s.adminService.setBlockCtrl)
			radmin.Delete("/user/{userid}", s.adminService.deleteUserCtrl)
			radmin.Get("/user/{userid}", s.adminService.getUserInfoCtrl)
			radmin.Get("/deleteme", s.adminService.deleteMeRequestCtrl)
			radmin.Put("/verify/{userid}", s.adminService.setVerifyCtrl)
			radmin.Put("/pin/{id}", s.adminService.setPinCtrl)
			radmin.Get("/blocked", s.adminService.blockedUsersCtrl)
			radmin.Put("/readonly", s.adminService.setReadOnlyCtrl)
			radmin.Put("/title/{id}", s.adminService.setTitleCtrl)

			// migrator
			radmin.Get("/export", s.adminService.migrator.exportCtrl)
			radmin.Post("/import", s.adminService.migrator.importCtrl)
			radmin.Post("/import/form", s.adminService.migrator.importFormCtrl)
			radmin.Get("/import/wait", s.adminService.migrator.importWaitCtrl)
		})

		// protected routes, throttled to 10/s by default, controlled by external UpdateLimiter param
		rapi.Group(func(rauth chi.Router) {
			lmt := 10.0
			if s.UpdateLimiter > 0 {
				lmt = s.UpdateLimiter
			}
			rauth.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(lmt, nil)))
			rauth.Use(authMiddleware.Auth)
			rauth.Use(middleware.NoCache)
			rauth.Use(logger.New(logger.Log(log.Default()), logger.WithBody,
				logger.Prefix("[DEBUG]"), logger.IPfn(ipFn)).Handler)

			rauth.Put("/comment/{id}", s.updateCommentCtrl)
			rauth.Post("/comment", s.createCommentCtrl)
			rauth.With(rejectAnonUser).Put("/vote/{id}", s.voteCtrl)
			rauth.With(rejectAnonUser).Post("/deleteme", s.deleteMeCtrl)
		})

		rapi.Group(func(rauth chi.Router) {
			lmt := 10.0
			if s.UpdateLimiter > 0 {
				lmt = s.UpdateLimiter
			}
			rauth.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(lmt, nil)))
			rauth.Use(authMiddleware.Auth)
			rauth.Use(logger.New(logger.Log(log.Default()), logger.Prefix("[DEBUG]"), logger.IPfn(ipFn)).Handler)
			rauth.With(rejectAnonUser).Post("/picture", s.savePictureCtrl)
		})

	})

	router.Group(func(rroot chi.Router) {
		tollbooth_chi.LimitHandler(tollbooth.NewLimiter(50, nil))
		rroot.Get("/index.html", s.getStartedCtrl)
		rroot.Get("/robots.txt", s.getRobotsCtrl)
	})

	// file server for static content from /web
	addFileServer(router, "/web", http.Dir(s.WebRoot))
	return router
}

func (s *Rest) alterComments(comments []store.Comment, r *http.Request) (res []store.Comment) {

	res = s.adminService.alterComments(comments, r) // apply admin's alteration

	// prepare vote info for client view
	vote := func(c store.Comment, r *http.Request) store.Comment {

		c.Vote = 0 // default is "none" (not voted)

		user, err := rest.GetUserInfo(r)
		if err != nil {
			c.Votes = nil // hide voters list and don't set Vote for non-authed user
			return c
		}

		if v, ok := c.Votes[user.ID]; ok {
			if v {
				c.Vote = 1
			} else {
				c.Vote = -1
			}
		}

		c.Votes = nil // hide voters list
		return c
	}

	for i, c := range res {
		c = vote(c, r)
		res[i] = c
	}

	return res
}

// serves static files from /web or embedded by statik
func addFileServer(r chi.Router, path string, root http.FileSystem) {

	var webFS http.Handler

	statikFS, err := fs.New()
	if err != nil {
		log.Printf("[DEBUG] no embedded assets loaded, %s", err)
		log.Printf("[INFO] run file server for %s, path %s", root, path)
		webFS = http.FileServer(root)
	} else {
		log.Printf("[INFO] run file server for %s, embedded", root)
		webFS = http.FileServer(statikFS)
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

func filterComments(comments []store.Comment, fn func(c store.Comment) bool) []store.Comment {
	filtered := []store.Comment{}
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
	key := strings.TrimPrefix(r.URL.String(), adminPrefix) // prevents attach with fake url to get admin view
	if user, err := rest.GetUserInfo(r); err == nil && user.Admin {
		key = adminPrefix + key // make separate cache key for admins
	}
	return key
}

// URLKeyWithUser gets url from request to use it as cache key and attaching user ID
// admins will have different keys in order to prevent leak of admin-only data to regular users
func URLKeyWithUser(r *http.Request) string {
	adminPrefix := "admin!!"
	key := strings.TrimPrefix(r.URL.String(), adminPrefix) // prevents attach with fake url to get admin view
	if user, err := rest.GetUserInfo(r); err == nil {
		if user.Admin {
			key = adminPrefix + user.ID + "!!" + key // make separate cache key for admins
		} else {
			key = user.ID + "!!" + key // make separate cache key for authed users
		}
	}
	return key
}

// rejectAnonUser is a middleware rejecting anonymous users
func rejectAnonUser(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user, err := rest.GetUserInfo(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if strings.HasPrefix(user.ID, "anonymous_") {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
