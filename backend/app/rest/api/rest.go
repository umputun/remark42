package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-pkgz/auth/v2"
	"github.com/go-pkgz/lcw/v2"
	log "github.com/go-pkgz/lgr"
	R "github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/logger"
	"github.com/go-pkgz/routegroup"

	"github.com/umputun/remark42/backend/app/notify"
	"github.com/umputun/remark42/backend/app/rest"
	"github.com/umputun/remark42/backend/app/rest/proxy"
	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/image"
	"github.com/umputun/remark42/backend/app/store/service"
	"github.com/umputun/remark42/backend/app/webassets"
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
	TelegramService  telegramService
	ImageService     *image.Service

	AnonVote        bool
	WebRoot         string
	WebFS           fs.FS
	RemarkURL       string
	DemoSiteID      string // site the demo pages under /web show, the first configured one
	ReadOnlyAge     int
	SharedSecret    string
	TrustedProxies  []*net.IPNet // reverse-proxy networks whose forwarding headers (X-Real-IP, X-Forwarded-For, ...) are trusted
	ScoreThresholds struct {
		Low      int
		Critical int
	}
	UpdateLimiter              float64
	EmailNotifications         bool
	TelegramNotifications      bool
	EmojiEnabled               bool
	SimpleView                 bool
	ProxyCORS                  bool
	SendJWTHeader              bool
	AllowedAncestors           []string // sets Content-Security-Policy "frame-ancestors ..."
	SubscribersOnly            bool
	DisableSignature           bool // prevent signature from being added to headers
	DisableFancyTextFormatting bool // disables SmartyPants in the comment text rendering of the posted comments
	ExternalImageProxy         bool

	SSLConfig         SSLConfig
	httpsServer       *http.Server
	httpServer        *http.Server
	shutdownRequested bool
	lock              sync.Mutex

	pubRest          public
	privRest         private
	adminRest        admin
	rssRest          rss
	openRouteLimiter float64
}

// LoadingCache defines interface for caching
type LoadingCache interface {
	Get(key lcw.Key, fn func() ([]byte, error)) (data []byte, err error) // load from cache if found or put to cache and return
	Flush(req lcw.FlusherRequest)                                        // evict matched records
	Close() error
}

const hardBodyLimit = 1024 * 64 // limit size of body
const openRouteLimiter = 10     // limit for open routes
const lastCommentsScope = "last"

// lastVotesScope is carried only by the JSON last comments, the one response under that scope with
// per-user vote data, so a vote does not also drop the feeds
const lastVotesScope = "last-votes"

// avatars are served at two routes: AvatarRoutePath, where the avatar proxy makes user pictures, and
// LegacyAvatarRoutePath at the root, where pictures stored by older versions point
const (
	apiRoutePath          = "/api/v1"
	LegacyAvatarRoutePath = "/avatar"
	AvatarRoutePath       = apiRoutePath + LegacyAvatarRoutePath
)

type commentsWithInfo struct {
	Comments []store.Comment `json:"comments"`
	Info     store.PostInfo  `json:"info"`
}

type treeWithInfo struct {
	*service.Tree
	Info store.PostInfo `json:"info"`
	// the pinned comments of the whole post, set on the first page of a limited tree only, so that
	// a pin on a later page is shown before the page holding it is loaded
	Pinned []store.Comment `json:"pinned,omitempty"`
}

// Run the lister and request's router, activate rest server
func (s *Rest) Run(address string, port int) {
	if address == "*" {
		address = ""
	}

	// without SSL the http server on port serves the api; with SSL the https server does, and the
	// http server only redirects to it (static) or answers the ACME challenge (auto)
	var httpHandler http.Handler
	var httpsServer *http.Server
	var httpName, certFile, keyFile string
	switch s.SSLConfig.SSLMode {
	case None:
		log.Printf("[INFO] activate http rest server on %s:%d", address, port)
		httpHandler = s.routes()
	case Static:
		log.Printf("[INFO] activate https server in 'static' mode on %s:%d", address, s.SSLConfig.Port)
		httpsServer = s.makeHTTPSServer(address, s.SSLConfig.Port, s.routes(), nil)
		httpHandler, httpName = s.httpRouter(nil), "http redirect"
		certFile, keyFile = s.SSLConfig.Cert, s.SSLConfig.Key
	case Auto:
		log.Printf("[INFO] activate https server in 'auto' mode on %s:%d", address, s.SSLConfig.Port)
		m := s.makeAutocertManager()
		httpsServer = s.makeHTTPSServer(address, s.SSLConfig.Port, s.routes(), m)
		httpHandler, httpName = s.httpRouter(m), "http challenge"
	default:
		return
	}

	httpServer := s.makeHTTPServer(address, port, httpHandler)
	httpServer.ErrorLog = log.ToStdLogger(log.Default(), "WARN")
	if httpsServer != nil {
		httpsServer.ErrorLog = log.ToStdLogger(log.Default(), "WARN")
	}

	s.lock.Lock()
	if s.shutdownRequested {
		s.lock.Unlock()
		log.Print("[WARN] rest server start canceled")
		return
	}
	s.httpServer, s.httpsServer = httpServer, httpsServer
	s.lock.Unlock()

	if httpsServer == nil {
		err := httpServer.ListenAndServe()
		log.Printf("[WARN] http server terminated, %s", err)
		return
	}

	go func() {
		log.Printf("[INFO] activate %s server on %s:%d", httpName, address, port)
		err := httpServer.ListenAndServe()
		log.Printf("[WARN] %s server terminated, %s", httpName, err)
	}()

	err := httpsServer.ListenAndServeTLS(certFile, keyFile)
	log.Printf("[WARN] https server terminated, %s", err)
}

// Shutdown rest http server
func (s *Rest) Shutdown() {
	log.Print("[WARN] shutdown rest server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	s.lock.Lock()
	s.shutdownRequested = true
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

func (s *Rest) makeHTTPServer(address string, port int, router http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf("%s:%d", address, port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		// WriteTimeout:      120 * time.Second, // TODO: such a long timeout needed for blocking export (backup) request
		IdleTimeout: 30 * time.Second,
	}
}

// authProviderRegistered reports whether the authenticator has a sign-in provider of that name
func (s *Rest) authProviderRegistered(name string) bool {
	_, err := s.Authenticator.Provider(name)
	return err == nil
}

func (s *Rest) routes() http.Handler {
	if s.openRouteLimiter == 0 {
		// set the default open route limiter. Just a safety measure as it should be set by Run method anyway
		s.openRouteLimiter = openRouteLimiter
	}
	router := routegroup.New(http.NewServeMux())
	router.Use(R.Throttle(1000), realIPMiddleware(s.TrustedProxies), recoverer(log.Default()))
	router.Use(ignoreQueryToken) // ahead of everything that authenticates a request
	router.Use(securityHeadersMiddleware(s.ExternalImageProxy, s.AllowedAncestors))
	if !s.DisableSignature {
		router.Use(R.AppInfo("remark42", "umputun", s.Version))
	}
	router.Use(R.Ping)

	s.pubRest, s.privRest, s.adminRest, s.rssRest = s.controllerGroups() // assign controllers for groups

	if s.ProxyCORS {
		log.Printf("[WARN] internal CORS disabled")
	} else {
		router.Use(corsMiddleware())
	}

	ipFn := func(ip string) string { return store.HashValue(ip, s.SharedSecret)[:12] } // logger uses it for anonymization
	// request urls and bodies carry email addresses (email login, subscription) and tokens, the log hides both
	reqLog := logger.Log(rest.MaskedLog{L: log.Default()})
	logInfoWithBody := logger.New(reqLog, logger.WithBody, logger.IPfn(ipFn), logger.Prefix("[INFO]")).Handler

	authHandler, avatarHandler := s.Authenticator.Handlers()

	// on the two groups where the auth library writes its cookies: sign-in and sign-out under
	// /auth, renewal and reset by the auth middleware under /api/v1
	authCookies := func(next http.Handler) http.Handler { return next }
	if strings.HasPrefix(s.RemarkURL, "https://") {
		// a partitioned cookie has to be Secure, so there is nothing to add over http. the names
		// are the auth library's defaults, which the server does not override. the partitioned
		// pair is issued in header mode only, and expired at sign-out in either mode, for a reader
		// who was given one before the mode was switched off
		authCookies = partitionedAuthCookies(s.SendJWTHeader, "JWT", "XSRF-TOKEN")
	}

	router.Route(func(r *routegroup.Bundle) {
		r.Use(authCookies)
		r.Use(R.Timeout(5 * time.Second))
		// sign-in and sign-out are not for an image or another embedded resource. Refused ahead of
		// the limiter, so such requests do not use up what it allows the reader's own. singleAuthPath
		// also goes ahead of the limiter and of validEmailAuth, which both go by the path
		r.Use(logInfoWithBody, rejectEmbedded, singleAuthPath(s.authProviderRegistered), authRateLimiter(), R.NoCache)
		r.Use(validEmailAuth()) // reject suspicious email logins
		r.Handle(authRoutePrefix, authHandler)
	})

	// avatars are served at the root, where pictures stored by older versions point, and under the api
	avatarRoutes := func(r *routegroup.Bundle) {
		r.Use(R.Timeout(5 * time.Second))
		r.Use(rateLimiter(100))
		r.Handle(LegacyAvatarRoutePath+"/", avatarHandler)
	}
	router.Route(avatarRoutes)

	authMiddleware := s.Authenticator.Middleware()

	// api routes
	rapi := router.Mount(apiRoutePath)
	rapi.Use(apiCSPMiddleware, s.rejectCrossSiteBasicAuth, authCookies)

	rapi.Group().Route(avatarRoutes)

	// open routes
	rapi.Group().Route(func(ropen *routegroup.Bundle) {
		ropen.Use(R.Timeout(30 * time.Second))
		ropen.Use(rateLimiter(s.openRouteLimiter))
		ropen.Use(authMiddleware.Trace, R.NoCache, logInfoWithBody)
		ropen.HandleFunc("GET /config", s.configCtrl)
		ropen.HandleFunc("GET /find", s.pubRest.findCommentsCtrl)
		ropen.HandleFunc("GET /id/{id}", s.pubRest.commentByIDCtrl)
		ropen.HandleFunc("GET /comments", s.pubRest.findUserCommentsCtrl)
		ropen.HandleFunc("GET /last/{limit}", s.pubRest.lastCommentsCtrl)
		ropen.HandleFunc("GET /count", s.pubRest.countCtrl)
		ropen.HandleFunc("POST /counts", s.pubRest.countMultiCtrl)
		ropen.HandleFunc("GET /list", s.pubRest.listCtrl)
		ropen.HandleFunc("GET /info", s.pubRest.infoCtrl)

		ropen.Mount("/rss").Route(func(rrss *routegroup.Bundle) {
			rrss.HandleFunc("GET /post", s.rssRest.postCommentsCtrl)
			rrss.HandleFunc("GET /site", s.rssRest.siteCommentsCtrl)
			rrss.HandleFunc("GET /reply", s.rssRest.repliesCtrl)
		})
	})

	// open routes, cached. /img lives here (not in the NoCache group above) because
	// R.NoCache strips If-None-Match from incoming requests, which would
	// defeat the proxy handler's 304 short-circuit. The handler sets a 30-day
	// max-age on validated success responses (with a versioned etag for cache
	// invalidation on revalidation); error responses get Cache-Control: no-store
	// so transient failures aren't pinned in the cache.
	rapi.Group().Route(func(ropen *routegroup.Bundle) {
		ropen.Use(R.Timeout(30 * time.Second))
		ropen.Use(rateLimiter(10))
		ropen.Use(authMiddleware.Trace, logInfoWithBody)
		ropen.HandleFunc("GET /img", s.ImageProxy.Handler)
		ropen.HandleFunc("GET /picture/{user}/{id}", s.pubRest.loadPictureCtrl)
		ropen.HandleFunc("GET /qr/telegram", s.pubRest.telegramQrCtrl)
	})

	// protected routes, require auth
	rapi.Group().Route(func(rauth *routegroup.Bundle) {
		rauth.Use(rateLimiter(10))
		rauth.Use(authMiddleware.Auth, matchSiteID, R.NoCache, logInfoWithBody)

		// GET /userdata streams a gzipped export of the user's data straight to the client, so it
		// deliberately runs without R.Timeout: that middleware buffers the whole response in memory
		// before sending and aborts at the deadline, which would hold a full export in RAM and truncate it.
		rauth.HandleFunc("GET /userdata", s.privRest.userAllDataCtrl)

		rauth.Group().Route(func(r *routegroup.Bundle) {
			r.Use(R.Timeout(30 * time.Second))
			r.HandleFunc("GET /user", s.privRest.userInfoCtrl)
			// a read, which the comment form repeats while it waits for the confirmation link. Under the
			// update limit it would share a bucket with DELETE /email and refuse the reader's Unsubscribe
			r.With(rejectAnonUser).HandleFunc("GET /email", s.privRest.getEmailCtrl)
		})
	})

	// admin routes, require auth and admin users only
	rapi.Mount("/admin").Route(func(radmin *routegroup.Bundle) {
		radmin.Use(rateLimiter(10))
		radmin.Use(authMiddleware.Auth, authMiddleware.AdminOnly, matchSiteID)
		radmin.Use(R.NoCache, logInfoWithBody)

		// bounded admin operations return small responses and get the enforcing request timeout
		radmin.Group().Route(func(r *routegroup.Bundle) {
			r.Use(R.Timeout(30 * time.Second))
			r.HandleFunc("DELETE /comment/{id}", s.adminRest.deleteCommentCtrl)
			r.HandleFunc("PUT /user/{userid}", s.adminRest.setBlockCtrl)
			r.HandleFunc("DELETE /user/{userid}", s.adminRest.deleteUserCtrl)
			r.HandleFunc("GET /user/{userid}", s.adminRest.getUserInfoCtrl)
			r.With(rejectHead("GET")).HandleFunc("GET /deleteme", s.adminRest.deleteMeRequestCtrl)
			r.HandleFunc("PUT /verify/{userid}", s.adminRest.setVerifyCtrl)
			r.HandleFunc("PUT /pin/{id}", s.adminRest.setPinCtrl)
			r.HandleFunc("GET /blocked", s.adminRest.blockedUsersCtrl)
			r.HandleFunc("PUT /readonly", s.adminRest.setReadOnlyCtrl)
			r.HandleFunc("PUT /title/{id}", s.adminRest.setTitleCtrl)
		})

		// migrator routes deliberately run without R.Timeout: GET /export streams a full-site
		// backup, GET /wait long-polls for up to 15m, and import/remap ingest large uploads. The
		// enforcing timeout buffers the whole response and aborts at the deadline, which would
		// truncate backups, break waiting, and reject large imports.
		radmin.HandleFunc("GET /export", s.adminRest.migrator.exportCtrl)
		radmin.HandleFunc("POST /import", s.adminRest.migrator.importCtrl)
		radmin.HandleFunc("POST /import/form", s.adminRest.migrator.importFormCtrl)
		radmin.HandleFunc("POST /remap", s.adminRest.migrator.remapCtrl)
		radmin.HandleFunc("GET /wait", s.adminRest.migrator.waitCtrl)
	})

	// protected routes, throttled per client address to UpdateLimiter requests a second, 10 if it is not set;
	// the server sets it from --update-limit, 0.5 by default
	rapi.Group().Route(func(rauth *routegroup.Bundle) {
		rauth.Use(R.Timeout(10 * time.Second))
		rauth.Use(rateLimiter(s.updateLimiter()))
		rauth.Use(authMiddleware.Auth, matchSiteID, subscribersOnly(s.SubscribersOnly))
		rauth.Use(R.NoCache, logInfoWithBody)

		rauth.HandleFunc("PUT /comment/{id}", s.privRest.updateCommentCtrl)
		rauth.HandleFunc("POST /preview", s.privRest.previewCommentCtrl)
		rauth.HandleFunc("POST /comment", s.privRest.createCommentCtrl)
		if s.AnonVote {
			rauth.HandleFunc("PUT /vote/{id}", s.privRest.voteCtrl)
		} else {
			rauth.With(rejectAnonUser).HandleFunc("PUT /vote/{id}", s.privRest.voteCtrl)
		}
		rauth.With(rejectAnonUser).HandleFunc("POST /deleteme", s.privRest.deleteMeCtrl)
		rauth.With(rejectAnonUser).HandleFunc("POST /email/subscribe", s.privRest.sendEmailConfirmationCtrl)
		rauth.With(rejectAnonUser).HandleFunc("POST /email/confirm", s.privRest.setConfirmedEmailCtrl)
		rauth.With(rejectAnonUser).HandleFunc("DELETE /email", s.privRest.deleteEmailCtrl)
		rauth.With(rejectAnonUser, rejectHead("GET")).HandleFunc("GET /telegram/subscribe", s.privRest.telegramSubscribeCtrl)
		rauth.With(rejectAnonUser).HandleFunc("DELETE /telegram", s.privRest.deleteTelegramCtrl)
	})

	// protected routes, anonymous rejected
	rapi.Group().Route(func(rauth *routegroup.Bundle) {
		rauth.Use(R.Timeout(10 * time.Second))
		rauth.Use(rateLimiter(s.updateLimiter()))
		rauth.Use(authMiddleware.Auth, rejectAnonUser, matchSiteID, subscribersOnly(s.SubscribersOnly))
		rauth.Use(logger.New(reqLog, logger.Prefix("[DEBUG]"), logger.IPfn(ipFn)).Handler)
		rauth.HandleFunc("POST /picture", s.privRest.savePictureCtrl)
	})

	// open routes on root level
	router.Route(func(rroot *routegroup.Bundle) {
		rroot.Use(R.Timeout(10 * time.Second))
		rroot.Use(rateLimiter(50))
		rroot.HandleFunc("GET /robots.txt", s.pubRest.robotsCtrl)
		rroot.With(rejectHead("GET, POST")).HandleFunc("GET /email/unsubscribe.html", s.privRest.emailUnsubscribeCtrl)
		rroot.HandleFunc("POST /email/unsubscribe.html", s.privRest.emailUnsubscribeCtrl)
		rroot.HandleFunc("GET /email/subscribe.html", s.privRest.emailSubscribeCtrl)
		rroot.HandleFunc("POST /email/subscribe.html", s.privRest.emailSubscribeCtrl)
		rroot.HandleFunc("GET /email/login.html", s.privRest.emailLoginCtrl)
		rroot.HandleFunc("POST /email/login.html", s.privRest.emailLoginCtrl)
	})

	// file server for /web: the frontend build first, then the assets embedded in the binary.
	// the build is embedded under web/ by app/cmd, so that prefix is stripped here. fs.Sub only
	// fails for an fs.SubFS that refuses, and a nil result would panic on the first request, so
	// serve nothing from the frontend rather than serving it at the wrong paths
	embeddedFrontend, err := fs.Sub(s.WebFS, "web")
	if err != nil {
		log.Printf("[WARN] no embedded frontend, serving built-in assets only: %v", err)
		embeddedFrontend = emptyFS{}
	}
	addFileServer(router, embeddedFrontend, s.WebRoot, s.Version, s.RemarkURL, s.DemoSiteID)
	return router
}

func (s *Rest) controllerGroups() (public, private, admin, rss) {
	pubGrp := public{
		dataService:      s.DataService,
		cache:            s.Cache,
		imageService:     s.ImageService,
		commentFormatter: s.CommentFormatter,
		readOnlyAge:      s.ReadOnlyAge,
	}

	privGrp := private{
		dataService:                s.DataService,
		cache:                      s.Cache,
		imageService:               s.ImageService,
		commentFormatter:           s.CommentFormatter,
		readOnlyAge:                s.ReadOnlyAge,
		authenticator:              s.Authenticator,
		notifyService:              s.NotifyService,
		telegramService:            s.TelegramService,
		remarkURL:                  s.RemarkURL,
		disableFancyTextFormatting: s.DisableFancyTextFormatting,
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
		Version               string   `json:"version"`
		EditDuration          int      `json:"edit_duration"`
		AdminEdit             bool     `json:"admin_edit"`
		MinCommentSize        int      `json:"min_comment_size"`
		MaxCommentSize        int      `json:"max_comment_size"`
		Admins                []string `json:"admins"`
		AdminEmail            string   `json:"admin_email"`
		Auth                  []string `json:"auth_providers"`
		AnonVote              bool     `json:"anon_vote"`
		LowScore              int      `json:"low_score"`
		CriticalScore         int      `json:"critical_score"`
		PositiveScore         bool     `json:"positive_score"`
		ReadOnlyAge           int      `json:"readonly_age"`
		MaxImageSize          int      `json:"max_image_size"`
		EmailNotifications    bool     `json:"email_notifications"`
		TelegramNotifications bool     `json:"telegram_notifications"`
		EmojiEnabled          bool     `json:"emoji_enabled"`
		SimpleView            bool     `json:"simple_view"`
		SendJWTHeader         bool     `json:"send_jwt_header"`
		SubscribersOnly       bool     `json:"subscribers_only"`
	}{
		Version:               s.Version,
		EditDuration:          int(s.DataService.EditDuration.Seconds()),
		AdminEdit:             s.DataService.AdminEdits,
		MinCommentSize:        s.DataService.MinCommentSize,
		MaxCommentSize:        s.DataService.MaxCommentSize,
		Admins:                admins,
		AdminEmail:            emails,
		LowScore:              s.ScoreThresholds.Low,
		CriticalScore:         s.ScoreThresholds.Critical,
		PositiveScore:         s.DataService.PositiveScore,
		ReadOnlyAge:           s.ReadOnlyAge,
		MaxImageSize:          s.ImageService.MaxSize,
		EmailNotifications:    s.EmailNotifications,
		TelegramNotifications: s.TelegramNotifications,
		EmojiEnabled:          s.EmojiEnabled,
		AnonVote:              s.AnonVote,
		SimpleView:            s.SimpleView,
		SendJWTHeader:         s.SendJWTHeader,
		SubscribersOnly:       s.SubscribersOnly,
	}

	cnf.Auth = []string{}
	for _, ap := range s.Authenticator.Providers() {
		cnf.Auth = append(cnf.Auth, ap.Name())
	}

	if cnf.Admins == nil { // prevent json serialization to nil
		cnf.Admins = []string{}
	}
	R.RenderJSON(w, cnf)
}

// serves /web from the frontend build, falling back to the assets embedded in the binary for
// names the build does not produce. the frontend build is read from webRoot on disk, or from the
// copy embedded at app/cmd/web when that directory is absent.
func addFileServer(r *routegroup.Bundle, embeddedFrontend fs.FS, webRoot, version, remarkURL, siteID string) {
	frontendFS := embeddedFrontend

	if _, err := os.Stat(webRoot); err == nil {
		log.Printf("[INFO] run file server from %s from the disk", webRoot)
		frontendFS = os.DirFS(webRoot)
	} else {
		log.Printf("[INFO] run file server, embedded")
	}

	// wrapped rather than substituted once at startup: the disk root can change under a running
	// server, and the docker image has already substituted its copy, where this is a no-op
	files := webFiles{frontend: frontendFS, embedded: webassets.FS}
	sources := templatedFS{
		fs:        files,
		remarkURL: remarkURL,
		siteID:    siteID,
	}
	webFS := http.StripPrefix("/web", http.FileServer(http.FS(sources)))
	r.HandleFunc("GET /web", http.RedirectHandler("/web/", http.StatusMovedPermanently).ServeHTTP)

	r.With(rateLimiter(20),
		R.Timeout(10*time.Second),
		// the validator covers everything the served body depends on: the file's bytes, and remarkURL
		// and the demo site substituted into them. Without any one of them an operator who edits a file
		// in the web root, or corrects REMARK_URL or SITE, and restarts the same binary keeps getting
		// 304 on revalidation, and the client keeps the old page for good, since no-cache means it
		// revalidates rather than aging out
		cacheControl(time.Hour, version+":"+remarkURL+":"+siteID, webFingerprint(files)),
	).HandleFunc("GET /web/", func(w http.ResponseWriter, r *http.Request) {
		// don't show dirs, just serve files
		if strings.HasSuffix(r.URL.Path, "/") && len(r.URL.Path) > 1 && r.URL.Path != ("/web/") {
			http.NotFound(w, r)
			return
		}
		// the file server keeps a type that is already set, so this one wins over the operating system's table
		if ct := webContentType(r.URL.Path); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		webFS.ServeHTTP(w, r)
	})
}

func encodeJSONWithHTML(v any) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("json encoding failed: %w", err)
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

// urlKey makes a cache key from the request url. Admins get keys of their own, so admin-only data never
// reaches regular users, and with perUser so does every authenticated user, for responses carrying
// per-user data. A request url starts with "/" or a scheme, so it can't pose as an admin key
func urlKey(r *http.Request, perUser bool) string {
	key := r.URL.String()
	user, err := rest.GetUserInfo(r)
	if err != nil {
		return key
	}
	if perUser {
		key = user.ID + "!!" + key
	}
	if user.Admin {
		key = "admin!!" + key
	}
	return key
}

func parseError(err error, defaultCode int) (code int) {
	code = defaultCode

	switch {
	// voting errors
	case strings.Contains(err.Error(), "can not vote for his own comment"):
		code = rest.ErrVoteSelf
	// another user voted from the same address within VOTES_IP_TIME, or at all when it is 0: this user
	// has not voted, so it is a rejected action, not a repeated vote
	case strings.HasPrefix(err.Error(), "the same ip already voted for"):
		code = rest.ErrActionRejected
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
