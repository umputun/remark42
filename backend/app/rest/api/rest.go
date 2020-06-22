package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/didip/tollbooth/v6"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/go-pkgz/auth"
	"github.com/go-pkgz/lcw"
	log "github.com/go-pkgz/lgr"
	R "github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/logger"
	"github.com/pkg/errors"
	"github.com/rakyll/statik/fs"

	"github.com/umputun/remark42/backend/app/notify"
	"github.com/umputun/remark42/backend/app/rest"
	"github.com/umputun/remark42/backend/app/rest/proxy"
	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/image"
	"github.com/umputun/remark42/backend/app/store/service"
	"github.com/umputun/remark42/backend/app/templates"
)

// Rest is a rest access server
type Rest struct {
	Version string

	DataService      *service.DataStore
	Authenticator    *auth.Service
	Cache            LoadingCache
	ImageProxy       *proxy.Image
	CommentFormatter *store.CommentFormatter
	Migrator         *Migrator
	NotifyService    *notify.Service
	ImageService     *image.Service
	Streamer         *Streamer

	AnonVote        bool
	WebRoot         string
	RemarkURL       string
	AdminEmail      string
	ReadOnlyAge     int
	SharedSecret    string
	ScoreThresholds struct {
		Low      int
		Critical int
	}
	UpdateLimiter      float64
	EmailNotifications bool
	EmojiEnabled       bool
	SimpleView         bool

	SSLConfig   SSLConfig
	httpsServer *http.Server
	httpServer  *http.Server
	lock        sync.Mutex

	pubRest   public
	privRest  private
	adminRest admin
	rssRest   rss
}

// LoadingCache defines interface for caching
type LoadingCache interface {
	Get(key lcw.Key, fn func() ([]byte, error)) (data []byte, err error) // load from cache if found or put to cache and return
	Flush(req lcw.FlusherRequest)                                        // evict matched records
	Close() error
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
		// WriteTimeout:      120 * time.Second, // TODO: such a long timeout needed for blocking export (backup) request
		IdleTimeout: 30 * time.Second,
	}
}

func (s *Rest) routes() chi.Router {
	router := chi.NewRouter()
	router.Use(middleware.Throttle(1000), middleware.RealIP, R.Recoverer(log.Default()))
	router.Use(R.AppInfo("remark42", "umputun", s.Version), R.Ping)

	s.pubRest, s.privRest, s.adminRest, s.rssRest = s.controllerGroups() // assign controllers for groups

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
	logInfoWithBody := logger.New(logger.Log(log.Default()), logger.WithBody, logger.IPfn(ipFn), logger.Prefix("[INFO]")).Handler

	authHandler, avatarHandler := s.Authenticator.Handlers()

	router.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(5 * time.Second))
		r.Use(logInfoWithBody, tollbooth_chi.LimitHandler(tollbooth.NewLimiter(5, nil)), middleware.NoCache)
		r.Mount("/auth", authHandler)
	})

	router.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(5 * time.Second))
		r.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(100, nil)), middleware.NoCache)
		r.Mount("/avatar", avatarHandler)
	})

	authMiddleware := s.Authenticator.Middleware()

	// api routes
	router.Route("/api/v1", func(rapi chi.Router) {

		rapi.Group(func(rava chi.Router) {
			rava.Use(middleware.Timeout(5 * time.Second))
			rava.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(100, nil)))
			rava.Use(middleware.NoCache)
			rava.Mount("/avatar", avatarHandler)
		})

		// open routes
		rapi.Group(func(ropen chi.Router) {
			ropen.Use(middleware.Timeout(30 * time.Second))
			ropen.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))
			ropen.Use(authMiddleware.Trace, middleware.NoCache, logInfoWithBody)
			ropen.Get("/config", s.configCtrl)
			ropen.Get("/find", s.pubRest.findCommentsCtrl)
			ropen.Get("/id/{id}", s.pubRest.commentByIDCtrl)
			ropen.Get("/comments", s.pubRest.findUserCommentsCtrl)
			ropen.Get("/last/{limit}", s.pubRest.lastCommentsCtrl)
			ropen.Get("/count", s.pubRest.countCtrl)
			ropen.Post("/counts", s.pubRest.countMultiCtrl)
			ropen.Get("/list", s.pubRest.listCtrl)
			ropen.Post("/preview", s.pubRest.previewCommentCtrl)
			ropen.Get("/info", s.pubRest.infoCtrl)
			ropen.Get("/img", s.ImageProxy.Handler)

			ropen.Route("/rss", func(rrss chi.Router) {
				rrss.Get("/post", s.rssRest.postCommentsCtrl)
				rrss.Get("/site", s.rssRest.siteCommentsCtrl)
				rrss.Get("/reply", s.rssRest.repliesCtrl)
			})

		})

		// open routes, streams, no send timeout
		rapi.Route("/stream", func(rstream chi.Router) {
			rstream.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))
			rstream.Use(authMiddleware.Trace, middleware.NoCache, logInfoWithBody)
			rstream.Get("/info", s.pubRest.infoStreamCtrl)
			rstream.Get("/last", s.pubRest.lastCommentsStreamCtrl)
		})

		// open routes, cached
		rapi.Group(func(ropen chi.Router) {
			ropen.Use(middleware.Timeout(30 * time.Second))
			ropen.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))
			ropen.Use(authMiddleware.Trace, logInfoWithBody)
			ropen.Get("/picture/{user}/{id}", s.pubRest.loadPictureCtrl)
		})

		// protected routes, require auth
		rapi.Group(func(rauth chi.Router) {
			rauth.Use(middleware.Timeout(30 * time.Second))
			rauth.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))
			rauth.Use(authMiddleware.Auth, matchSiteID, middleware.NoCache, logInfoWithBody)
			rauth.Get("/user", s.privRest.userInfoCtrl)
			rauth.Get("/userdata", s.privRest.userAllDataCtrl)
		})

		// admin routes, require auth and admin users only
		rapi.Route("/admin", func(radmin chi.Router) {
			radmin.Use(middleware.Timeout(30 * time.Second))
			radmin.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))
			radmin.Use(authMiddleware.Auth, authMiddleware.AdminOnly, matchSiteID)
			radmin.Use(middleware.NoCache, logInfoWithBody)

			radmin.Delete("/comment/{id}", s.adminRest.deleteCommentCtrl)
			radmin.Put("/user/{userid}", s.adminRest.setBlockCtrl)
			radmin.Delete("/user/{userid}", s.adminRest.deleteUserCtrl)
			radmin.Get("/user/{userid}", s.adminRest.getUserInfoCtrl)
			radmin.Get("/deleteme", s.adminRest.deleteMeRequestCtrl)
			radmin.Put("/verify/{userid}", s.adminRest.setVerifyCtrl)
			radmin.Put("/pin/{id}", s.adminRest.setPinCtrl)
			radmin.Get("/blocked", s.adminRest.blockedUsersCtrl)
			radmin.Put("/readonly", s.adminRest.setReadOnlyCtrl)
			radmin.Put("/title/{id}", s.adminRest.setTitleCtrl)

			// migrator
			radmin.Get("/export", s.adminRest.migrator.exportCtrl)
			radmin.Post("/import", s.adminRest.migrator.importCtrl)
			radmin.Post("/import/form", s.adminRest.migrator.importFormCtrl)
			radmin.Post("/remap", s.adminRest.migrator.remapCtrl)
			radmin.Get("/wait", s.adminRest.migrator.waitCtrl)
		})

		// protected routes, throttled to 10/s by default, controlled by external UpdateLimiter param
		rapi.Group(func(rauth chi.Router) {
			rauth.Use(middleware.Timeout(10 * time.Second))
			rauth.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(s.updateLimiter(), nil)))
			rauth.Use(authMiddleware.Auth, matchSiteID)
			rauth.Use(middleware.NoCache, logInfoWithBody)

			rauth.Put("/comment/{id}", s.privRest.updateCommentCtrl)
			rauth.Post("/comment", s.privRest.createCommentCtrl)
			rauth.Put("/vote/{id}", s.privRest.voteCtrl)
			rauth.With(rejectAnonUser).Post("/deleteme", s.privRest.deleteMeCtrl)
			rauth.With(rejectAnonUser).Get("/email", s.privRest.getEmailCtrl)
			rauth.With(rejectAnonUser).Post("/email/subscribe", s.privRest.sendEmailConfirmationCtrl)
			rauth.With(rejectAnonUser).Post("/email/confirm", s.privRest.setConfirmedEmailCtrl)
			rauth.With(rejectAnonUser).Delete("/email", s.privRest.deleteEmailCtrl)
		})

		// protected routes, anonymous rejected
		rapi.Group(func(rauth chi.Router) {
			rauth.Use(middleware.Timeout(10 * time.Second))
			rauth.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(s.updateLimiter(), nil)))
			rauth.Use(authMiddleware.Auth, rejectAnonUser, matchSiteID)
			rauth.Use(logger.New(logger.Log(log.Default()), logger.Prefix("[DEBUG]"), logger.IPfn(ipFn)).Handler)
			rauth.Post("/picture", s.privRest.savePictureCtrl)
		})

	})

	// open routes on root level
	router.Group(func(rroot chi.Router) {
		rroot.Use(middleware.Timeout(10 * time.Second))
		rroot.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(50, nil)))
		rroot.Get("/index.html", s.pubRest.getStartedCtrl)
		rroot.Get("/robots.txt", s.pubRest.robotsCtrl)
		rroot.Get("/email/unsubscribe.html", s.privRest.emailUnsubscribeCtrl)
		rroot.Post("/email/unsubscribe.html", s.privRest.emailUnsubscribeCtrl)
	})

	// file server for static content from /web
	addFileServer(router, "/web", http.Dir(s.WebRoot), s.Version)
	return router
}

func (s *Rest) controllerGroups() (public, private, admin, rss) {

	pubGrp := public{
		dataService:      s.DataService,
		cache:            s.Cache,
		imageService:     s.ImageService,
		commentFormatter: s.CommentFormatter,
		readOnlyAge:      s.ReadOnlyAge,
		webRoot:          s.WebRoot,
		streamer:         s.Streamer,
	}

	privGrp := private{
		dataService:      s.DataService,
		cache:            s.Cache,
		imageService:     s.ImageService,
		commentFormatter: s.CommentFormatter,
		readOnlyAge:      s.ReadOnlyAge,
		authenticator:    s.Authenticator,
		notifyService:    s.NotifyService,
		remarkURL:        s.RemarkURL,
		adminEmail:       s.AdminEmail,
		anonVote:         s.AnonVote,
		templates:        templates.NewFS(),
	}

	admGrp := admin{
		dataService:   s.DataService,
		migrator:      s.Migrator,
		cache:         s.Cache,
		authenticator: s.Authenticator,
		readOnlyAge:   s.ReadOnlyAge,
	}

	rssGrp := rss{
		dataService: s.DataService,
		cache:       s.Cache,
	}

	return pubGrp, privGrp, admGrp, rssGrp
}

// updateLimiter returns UpdateLimiter if set, or 10 if not
func (s *Rest) updateLimiter() float64 {
	lmt := 10.0
	if s.UpdateLimiter > 0 {
		lmt = s.UpdateLimiter
	}
	return lmt
}

// GET /config?site=siteID - returns configuration
func (s *Rest) configCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")

	admins, _ := s.DataService.AdminStore.Admins(siteID)
	emails, _ := s.DataService.AdminStore.Email(siteID)

	cnf := struct {
		Version            string   `json:"version"`
		EditDuration       int      `json:"edit_duration"`
		MaxCommentSize     int      `json:"max_comment_size"`
		Admins             []string `json:"admins"`
		AdminEmail         string   `json:"admin_email"`
		Auth               []string `json:"auth_providers"`
		AnonVote           bool     `json:"anon_vote"`
		LowScore           int      `json:"low_score"`
		CriticalScore      int      `json:"critical_score"`
		PositiveScore      bool     `json:"positive_score"`
		ReadOnlyAge        int      `json:"readonly_age"`
		MaxImageSize       int      `json:"max_image_size"`
		EmailNotifications bool     `json:"email_notifications"`
		EmojiEnabled       bool     `json:"emoji_enabled"`
		SimpleView         bool     `json:"simple_view"`
	}{
		Version:            s.Version,
		EditDuration:       int(s.DataService.EditDuration.Seconds()),
		MaxCommentSize:     s.DataService.MaxCommentSize,
		Admins:             admins,
		AdminEmail:         emails,
		LowScore:           s.ScoreThresholds.Low,
		CriticalScore:      s.ScoreThresholds.Critical,
		PositiveScore:      s.DataService.PositiveScore,
		ReadOnlyAge:        s.ReadOnlyAge,
		MaxImageSize:       s.ImageService.MaxSize,
		EmailNotifications: s.EmailNotifications,
		EmojiEnabled:       s.EmojiEnabled,
		AnonVote:           s.AnonVote,
		SimpleView:         s.SimpleView,
	}

	cnf.Auth = []string{}
	for _, ap := range s.Authenticator.Providers() {
		cnf.Auth = append(cnf.Auth, ap.Name())
	}

	if cnf.Admins == nil { // prevent json serialization to nil
		cnf.Admins = []string{}
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, cnf)
}

// serves static files from /web or embedded by statik
func addFileServer(r chi.Router, path string, root http.FileSystem, version string) {

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
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.With(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(20, nil)),
		middleware.Timeout(10*time.Second),
		cacheControl(time.Hour, version),
	).Get(path, func(w http.ResponseWriter, r *http.Request) {
		// don't show dirs, just serve files
		if strings.HasSuffix(r.URL.Path, "/") && len(r.URL.Path) > 1 && r.URL.Path != (origPath+"/") {
			http.NotFound(w, r)
			return
		}
		webFS.ServeHTTP(w, r)
	})
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

// matchSiteID is a middleware rejecting users with mismatch between site param and and User.SiteID
func matchSiteID(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user, err := rest.GetUserInfo(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// skip for basic auth user
		if user.Name == "admin" && user.ID == "admin" {
			next.ServeHTTP(w, r)
			return
		}

		siteID := r.URL.Query().Get("site")
		if siteID != "" && user.SiteID != siteID {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// cacheControl is a middleware setting cache expiration. Using url+version as etag
func cacheControl(expiration time.Duration, version string) func(http.Handler) http.Handler {

	etag := func(r *http.Request, version string) string {
		s := version + ":" + r.URL.String()
		return store.EncodeID(s)
	}

	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			e := `"` + etag(r, version) + `"`
			w.Header().Set("Etag", e)
			w.Header().Set("Cache-Control", "max-age="+strconv.Itoa(int(expiration.Seconds())))

			if match := r.Header.Get("If-None-Match"); match != "" {
				if strings.Contains(match, e) {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

func parseError(err error, defaultCode int) (code int) {
	code = defaultCode

	switch {
	// voting errors
	case strings.Contains(err.Error(), "can not vote for his own comment"):
		code = rest.ErrVoteSelf
	case strings.Contains(err.Error(), "already voted for"):
		code = rest.ErrVoteDbl
	case strings.Contains(err.Error(), "maximum number of votes exceeded for comment"):
		code = rest.ErrVoteMax
	case strings.Contains(err.Error(), "minimal score reached for comment"):
		code = rest.ErrVoteMinScore

	// edit errors
	case strings.HasPrefix(err.Error(), "too late to edit"):
		code = rest.ErrCommentEditExpired
	case strings.HasPrefix(err.Error(), "parent comment with reply can't be edited"):
		code = rest.ErrCommentEditChanged

	}

	return code
}
