package api

import (
	"bytes"
	"context"
	"embed"
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
	WebFS           embed.FS
	RemarkURL       string
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
	NameCharacters             string
	NameMinLength              int
	NameMaxLength              int
	SimpleView                 bool
	HideVoting                 bool
	HideHide                   bool
	HideAvatars                bool
	HideUserID                 bool
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

type commentsWithInfo struct {
	Comments []store.Comment `json:"comments"`
	Info     store.PostInfo  `json:"info"`
}

type treeWithInfo struct {
	*service.Tree
	Info store.PostInfo `json:"info"`
}

// Run the lister and request's router, activate rest server
func (s *Rest) Run(address string, port int) {
	if address == "*" {
		address = ""
	}

	switch s.SSLConfig.SSLMode {
	case None:
		log.Printf("[INFO] activate http rest server on %s:%d", address, port)

		s.lock.Lock()
		s.httpServer = s.makeHTTPServer(address, port, s.routes())
		s.httpServer.ErrorLog = log.ToStdLogger(log.Default(), "WARN")
		if s.shutdownRequested {
			s.lock.Unlock()
			log.Print("[WARN] rest server start canceled")
			return
		}
		s.lock.Unlock()

		err := s.httpServer.ListenAndServe()
		log.Printf("[WARN] http server terminated, %s", err)
	case Static:
		log.Printf("[INFO] activate https server in 'static' mode on %s:%d", address, s.SSLConfig.Port)

		s.lock.Lock()
		s.httpsServer = s.makeHTTPSServer(address, s.SSLConfig.Port, s.routes())
		s.httpsServer.ErrorLog = log.ToStdLogger(log.Default(), "WARN")

		s.httpServer = s.makeHTTPServer(address, port, s.httpToHTTPSRouter())
		s.httpServer.ErrorLog = log.ToStdLogger(log.Default(), "WARN")
		if s.shutdownRequested {
			s.lock.Unlock()
			log.Print("[WARN] rest server start canceled")
			return
		}
		s.lock.Unlock()

		go func() {
			log.Printf("[INFO] activate http redirect server on %s:%d", address, port)
			err := s.httpServer.ListenAndServe()
			log.Printf("[WARN] http redirect server terminated, %s", err)
		}()

		err := s.httpsServer.ListenAndServeTLS(s.SSLConfig.Cert, s.SSLConfig.Key)
		log.Printf("[WARN] https server terminated, %s", err)
	case Auto:
		log.Printf("[INFO] activate https server in 'auto' mode on %s:%d", address, s.SSLConfig.Port)

		m := s.makeAutocertManager()
		s.lock.Lock()
		s.httpsServer = s.makeHTTPSAutocertServer(address, s.SSLConfig.Port, s.routes(), m)
		s.httpsServer.ErrorLog = log.ToStdLogger(log.Default(), "WARN")

		s.httpServer = s.makeHTTPServer(address, port, s.httpChallengeRouter(m))
		s.httpServer.ErrorLog = log.ToStdLogger(log.Default(), "WARN")
		if s.shutdownRequested {
			s.lock.Unlock()
			log.Print("[WARN] rest server start canceled")
			return
		}

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

func (s *Rest) routes() http.Handler {
	if s.openRouteLimiter == 0 {
		// set the default open route limiter. Just a safety measure as it should be set by Run method anyway
		s.openRouteLimiter = openRouteLimiter
	}
	router := routegroup.New(http.NewServeMux())
	router.Use(R.Throttle(1000), realIPMiddleware(s.TrustedProxies), R.Recoverer(log.Default()))
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
	logInfoWithBody := logger.New(logger.Log(log.Default()), logger.WithBody, logger.IPfn(ipFn), logger.Prefix("[INFO]")).Handler

	authHandler, avatarHandler := s.Authenticator.Handlers()

	router.Route(func(r *routegroup.Bundle) {
		r.Use(R.Timeout(5 * time.Second))
		r.Use(logInfoWithBody, rateLimiter(2), R.NoCache)
		r.Use(validEmailAuth()) // reject suspicious email logins
		r.Handle("/auth/", authHandler)
	})

	router.Route(func(r *routegroup.Bundle) {
		r.Use(R.Timeout(5 * time.Second))
		r.Use(rateLimiter(100))
		r.Handle("/avatar/", avatarHandler)
	})

	authMiddleware := s.Authenticator.Middleware()

	// api routes
	rapi := router.Mount("/api/v1")
	rapi.Use(apiCSPMiddleware)

	rapi.Group().Route(func(rava *routegroup.Bundle) {
		rava.Use(R.Timeout(5 * time.Second))
		rava.Use(rateLimiter(100))
		rava.Handle("/avatar/", avatarHandler)
	})

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

	// protected routes, throttled to 10/s by default, controlled by external UpdateLimiter param
	rapi.Group().Route(func(rauth *routegroup.Bundle) {
		rauth.Use(R.Timeout(10 * time.Second))
		rauth.Use(rateLimiter(s.updateLimiter()))
		rauth.Use(authMiddleware.Auth, matchSiteID, subscribersOnly(s.SubscribersOnly))
		rauth.Use(R.NoCache, logInfoWithBody)

		rauth.HandleFunc("PUT /comment/{id}", s.privRest.updateCommentCtrl)
		rauth.HandleFunc("POST /preview", s.privRest.previewCommentCtrl)
		rauth.HandleFunc("POST /comment", s.privRest.createCommentCtrl)
		rauth.HandleFunc("PUT /vote/{id}", s.privRest.voteCtrl)
		rauth.With(rejectAnonUser).HandleFunc("POST /deleteme", s.privRest.deleteMeCtrl)
		rauth.With(rejectAnonUser).HandleFunc("GET /email", s.privRest.getEmailCtrl)
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
		rauth.Use(authMiddleware.Auth, rejectAnonUser, matchSiteID)
		rauth.Use(logger.New(logger.Log(log.Default()), logger.Prefix("[DEBUG]"), logger.IPfn(ipFn)).Handler)
		rauth.HandleFunc("POST /picture", s.privRest.savePictureCtrl)
	})

	// open routes on root level
	router.Route(func(rroot *routegroup.Bundle) {
		rroot.Use(R.Timeout(10 * time.Second))
		rroot.Use(rateLimiter(50))
		rroot.HandleFunc("GET /robots.txt", s.pubRest.robotsCtrl)
		rroot.With(rejectHead("GET, POST")).HandleFunc("GET /email/unsubscribe.html", s.privRest.emailUnsubscribeCtrl)
		rroot.HandleFunc("POST /email/unsubscribe.html", s.privRest.emailUnsubscribeCtrl)
	})

	// file server for static content from s.WebRoot on path /web
	addFileServer(router, s.WebFS, s.WebRoot, s.Version)
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
		anonVote:                   s.AnonVote,
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
		NameCharacters        string   `json:"name_characters"`
		NameMinLength         int      `json:"name_minlength"`
		NameMaxLength         int      `json:"name_maxlength"`
		EmojiEnabled          bool     `json:"emoji_enabled"`
		SimpleView            bool     `json:"simple_view"`
		HideVoting            bool     `json:"hide_voting"`
		HideHide              bool     `json:"hide_hide"`
		HideAvatars           bool     `json:"hide_avatars"`
		HideUserID            bool     `json:"hide_userid"`
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
		NameCharacters:        s.NameCharacters,
		NameMinLength:         s.NameMinLength,
		NameMaxLength:         s.NameMaxLength,
		AnonVote:              s.AnonVote,
		SimpleView:            s.SimpleView,
		HideVoting:            s.HideVoting,
		HideHide:              s.HideHide,
		HideAvatars:           s.HideAvatars,
		HideUserID:            s.HideUserID,
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

// serves static files from the webRoot directory or files embedded into the compiled binary if that directory is absent
func addFileServer(r *routegroup.Bundle, embedFS embed.FS, webRoot, version string) {
	var webFS http.Handler

	if _, err := os.Stat(webRoot); err == nil {
		log.Printf("[INFO] run file server from %s from the disk", webRoot)
		webFS = http.FileServer(http.Dir(webRoot))
	} else {
		log.Printf("[INFO] run file server, embedded")
		var contentFS, _ = fs.Sub(embedFS, "web")
		webFS = http.FileServer(http.FS(contentFS))
	}

	webFS = http.StripPrefix("/web", webFS)
	r.HandleFunc("GET /web", http.RedirectHandler("/web/", http.StatusMovedPermanently).ServeHTTP)

	r.With(rateLimiter(20),
		R.Timeout(10*time.Second),
		cacheControl(time.Hour, version),
	).HandleFunc("GET /web/", func(w http.ResponseWriter, r *http.Request) {
		// don't show dirs, just serve files
		if strings.HasSuffix(r.URL.Path, "/") && len(r.URL.Path) > 1 && r.URL.Path != ("/web/") {
			http.NotFound(w, r)
			return
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
