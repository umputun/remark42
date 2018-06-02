package api

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/pkg/errors"
	"gopkg.in/russross/blackfriday.v2"

	"github.com/umputun/remark/app/migrator"
	"github.com/umputun/remark/app/rest"
	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/rest/cache"
	"github.com/umputun/remark/app/rest/proxy"
	"github.com/umputun/remark/app/store"
	"github.com/umputun/remark/app/store/service"
)

// Rest is a rest access server
type Rest struct {
	Version         string
	DataService     service.DataStore
	Authenticator   auth.Authenticator
	Exporter        migrator.Exporter
	Cache           cache.LoadingCache
	AvatarProxy     *proxy.Avatar
	ImageProxy      *proxy.Image
	WebRoot         string
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

// Run the lister and request's router, activate rest server
func (s *Rest) Run(port int) {
	log.Printf("[INFO] activate rest server on port %d", port)

	if len(s.Authenticator.Admins) > 0 {
		log.Printf("[DEBUG] admins %+v", s.Authenticator.Admins)
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
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("[DEBUG] rest shutdown error, %s", err)
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
		dataService: s.DataService,
		exporter:    s.Exporter,
		cache:       s.Cache,
	}

	ipFn := func(ip string) string { return store.HashValue(ip, s.DataService.Secret)[:12] }

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

			// admin routes, admin users only
			rauth.Mount("/admin", s.adminService.routes(s.Authenticator.AdminOnly, Logger(nil, LogAll)))
		})
	})

	router.With(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil))).
		Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
			render.PlainText(w, r, "User-agent: *\nDisallow: /auth/\nDisallow: /api/\n")
		})

	// file server for static content from /web
	addFileServer(router, "/web", http.Dir(s.WebRoot))
	return router
}

// POST /comment - adds comment, resets all immutable fields
func (s *Rest) createCommentCtrl(w http.ResponseWriter, r *http.Request) {

	comment := store.Comment{}
	if err := render.DecodeJSON(http.MaxBytesReader(w, r.Body, hardBodyLimit), &comment); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't bind comment")
		return
	}

	user, err := rest.GetUserInfo(r)
	if err != nil { // this not suppose to happen (handled by Auth), just dbl-check
		rest.SendErrorJSON(w, r, http.StatusUnauthorized, err, "can't get user info")
		return
	}
	log.Printf("[DEBUG] create comment %+v", comment)

	comment.PrepareUntrusted() // clean all fields user not supposed to set
	comment.User = user
	comment.User.IP = strings.Split(r.RemoteAddr, ":")[0]

	comment.Orig = comment.Text // original comment text, prior to md render
	if err = s.DataService.ValidateComment(&comment); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "invalid comment")
		return
	}
	comment.Text = string(blackfriday.Run([]byte(comment.Text), blackfriday.WithExtensions(mdExt)))
	comment.Text = s.ImageProxy.Convert(comment.Text)
	// check if user blocked
	if s.adminService.checkBlocked(comment.Locator.SiteID, comment.User) {
		rest.SendErrorJSON(w, r, http.StatusForbidden, errors.New("rejected"), "user blocked")
		return
	}

	if s.ReadOnlyAge > 0 {
		if info, e := s.DataService.Info(comment.Locator, s.ReadOnlyAge); e == nil && info.ReadOnly {
			rest.SendErrorJSON(w, r, http.StatusForbidden, errors.New("rejected"), "old post, read-only")
			return
		}
	}

	id, err := s.DataService.Create(comment)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't save comment")
		return
	}

	// DataService modifies comment
	finalComment, err := s.DataService.Get(comment.Locator, id)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't load created comment")
		return
	}
	s.Cache.Flush(comment.Locator.URL, "last", comment.User.ID, comment.Locator.SiteID)

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, &finalComment)
}

// POST /preview, body is a comment
func (s *Rest) previewCommentCtrl(w http.ResponseWriter, r *http.Request) {
	comment := store.Comment{}
	if err := render.DecodeJSON(http.MaxBytesReader(w, r.Body, hardBodyLimit), &comment); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't bind comment")
		return
	}

	user, err := rest.GetUserInfo(r)
	if err != nil { // this not suppose to happen (handled by Auth), just dbl-check
		rest.SendErrorJSON(w, r, http.StatusUnauthorized, err, "can't get user info")
		return
	}
	comment.User = user
	comment.Orig = comment.Text
	if err = s.DataService.ValidateComment(&comment); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "invalid comment")
		return
	}

	//comment.Text = string(blackfriday.Run([]byte(comment.Text),
	//	blackfriday.WithRenderer(bfchroma.NewRenderer(bfchroma.WithoutAutodetect()))))
	comment.Text = string(blackfriday.Run([]byte(comment.Text), blackfriday.WithExtensions(mdExt)))
	comment.Text = s.ImageProxy.Convert(comment.Text)
	comment.Sanitize()
	render.HTML(w, r, comment.Text)
}

// GET /info?site=siteID&url=post-url - get info about the post
func (s *Rest) infoCtrl(w http.ResponseWriter, r *http.Request) {
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}

	data, err := s.Cache.Get(cache.Key(cache.URLKey(r), locator.SiteID, locator.URL), func() ([]byte, error) {
		info, e := s.DataService.Info(locator, s.ReadOnlyAge)
		if e != nil {
			return nil, e
		}
		return encodeJSONWithHTML(info)
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get post info")
		return
	}

	renderJSONFromBytes(w, r, data)
}

// PUT /comment/{id}?site=siteID&url=post-url - update comment
func (s *Rest) updateCommentCtrl(w http.ResponseWriter, r *http.Request) {

	edit := struct {
		Text    string
		Summary string
	}{}

	if err := render.DecodeJSON(http.MaxBytesReader(w, r.Body, hardBodyLimit), &edit); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't bind comment")
		return
	}

	user, err := rest.GetUserInfo(r)
	if err != nil { // this not suppose to happen (handled by Auth), just dbl-check
		rest.SendErrorJSON(w, r, http.StatusUnauthorized, err, "can't get user info")
		return
	}
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	id := chi.URLParam(r, "id")

	log.Printf("[DEBUG] update comment %s", id)

	var currComment store.Comment
	if currComment, err = s.DataService.Get(locator, id); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't find comment")
		return
	}

	if currComment.User.ID != user.ID {
		rest.SendErrorJSON(w, r, http.StatusForbidden, errors.New("rejected"), "can not edit comments for other users")
		return
	}

	text := string(blackfriday.Run([]byte(edit.Text), blackfriday.WithExtensions(mdExt))) // render markdown
	text = s.ImageProxy.Convert(text)
	editReq := service.EditRequest{
		Text:    text,
		Orig:    edit.Text,
		Summary: edit.Summary,
	}

	res, err := s.DataService.EditComment(locator, id, editReq)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't update comment")
		return
	}

	s.Cache.Flush(locator.URL, "last", user.ID)
	render.JSON(w, r, res)
}

// GET /find?site=siteID&url=post-url&format=[tree|plain]&sort=[+/-time|+/-score]
// find comments for given post. Returns in tree or plain formats, sorted
func (s *Rest) findCommentsCtrl(w http.ResponseWriter, r *http.Request) {
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	sort := r.URL.Query().Get("sort")
	if strings.HasPrefix(sort, " ") { // restore + replaced by " "
		sort = "+" + sort[1:]
	}
	log.Printf("[DEBUG] get comments for %+v, sort %s, format %s", locator, sort, r.URL.Query().Get("format"))

	data, err := s.Cache.Get(cache.Key(cache.URLKey(r), locator.SiteID, locator.URL), func() ([]byte, error) {
		comments, e := s.DataService.Find(locator, sort)
		if e != nil {
			return nil, e
		}
		maskedComments := s.adminService.alterComments(comments, r)
		var b []byte
		switch r.URL.Query().Get("format") {
		case "tree":
			tree := rest.MakeTree(maskedComments, sort, s.ReadOnlyAge)
			if s.DataService.IsReadOnly(locator) {
				tree.Info.ReadOnly = true
			}
			b, e = encodeJSONWithHTML(tree)
		default:
			b, e = encodeJSONWithHTML(maskedComments)
		}
		return b, e
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't find comments")
		return
	}
	renderJSONFromBytes(w, r, data)
}

// GET /last/{limit}?site=siteID - last comments for the siteID, across all posts, sorted by time
func (s *Rest) lastCommentsCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	log.Printf("[DEBUG] get last comments for %s", siteID)

	limit, err := strconv.Atoi(chi.URLParam(r, "limit"))
	if err != nil {
		limit = 0
	}

	data, err := s.Cache.Get(cache.Key(cache.URLKey(r), "last", siteID), func() ([]byte, error) {
		comments, e := s.DataService.Last(siteID, limit)
		if e != nil {
			return nil, e
		}
		comments = s.adminService.alterComments(comments, r)

		// filter deleted from last comments view. Blocked marked as deleted and will sneak in without
		filterDeleted := []store.Comment{}
		for _, c := range comments {
			if c.Deleted {
				continue
			}
			filterDeleted = append(filterDeleted, c)
		}

		return encodeJSONWithHTML(filterDeleted)
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't get last comments")
		return
	}
	renderJSONFromBytes(w, r, data)
}

// GET /id/{id}?site=siteID&url=post-url - gets a comment by id
func (s *Rest) commentByIDCtrl(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	siteID := r.URL.Query().Get("site")
	url := r.URL.Query().Get("url")

	log.Printf("[DEBUG] get comments by id %s, %s %s", id, siteID, url)

	comment, err := s.DataService.Get(store.Locator{SiteID: siteID, URL: url}, id)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get comment by id")
		return
	}
	comment = s.adminService.alterComments([]store.Comment{comment}, r)[0]
	render.Status(r, http.StatusOK)
	renderJSONWithHTML(w, r, comment)
}

// GET /comments?site=siteID&user=id - returns comments for given userID
func (s *Rest) findUserCommentsCtrl(w http.ResponseWriter, r *http.Request) {

	userID := r.URL.Query().Get("user")
	siteID := r.URL.Query().Get("site")

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 0
	}

	resp := struct {
		Comments []store.Comment
		Count    int
	}{}

	log.Printf("[DEBUG] get comments for userID %s, %s", userID, siteID)

	data, err := s.Cache.Get(cache.Key(cache.URLKey(r), userID, siteID), func() ([]byte, error) {
		comments, e := s.DataService.User(siteID, userID, limit, 0)
		if e != nil {
			return nil, e
		}
		comments = s.adminService.alterComments(comments, r)

		count, e := s.DataService.UserCount(siteID, userID)
		if e != nil {
			return nil, e
		}
		resp.Comments, resp.Count = comments, count
		return encodeJSONWithHTML(resp)
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get comment by user id")
		return
	}
	renderJSONFromBytes(w, r, data)
}

// GET /config?site=siteID - returns configuration
func (s *Rest) configCtrl(w http.ResponseWriter, r *http.Request) {
	type config struct {
		Version        string   `json:"version"`
		EditDuration   int      `json:"edit_duration"`
		MaxCommentSize int      `json:"max_comment_size"`
		Admins         []string `json:"admins"`
		Auth           []string `json:"auth_providers"`
		LowScore       int      `json:"low_score"`
		CriticalScore  int      `json:"critical_score"`
		ReadOnlyAge    int      `json:"readonly_age"`
	}

	cnf := config{
		Version:        s.Version,
		EditDuration:   int(s.DataService.EditDuration.Seconds()),
		MaxCommentSize: s.DataService.MaxCommentSize,
		Admins:         s.Authenticator.Admins,
		LowScore:       s.ScoreThresholds.Low,
		CriticalScore:  s.ScoreThresholds.Critical,
		ReadOnlyAge:    s.ReadOnlyAge,
	}

	cnf.Auth = []string{}
	for _, ap := range s.Authenticator.Providers {
		cnf.Auth = append(cnf.Auth, ap.Name)
	}

	if cnf.Admins == nil { // prevent json serialization to nil
		cnf.Admins = []string{}
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, cnf)
}

// GET /user - returns user info
func (s *Rest) userInfoCtrl(w http.ResponseWriter, r *http.Request) {
	user, err := rest.GetUserInfo(r)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusUnauthorized, err, "can't get user info")
		return
	}
	render.JSON(w, r, user)
}

// GET /count?site=siteID&url=post-url - get number of comments for given post
func (s *Rest) countCtrl(w http.ResponseWriter, r *http.Request) {
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	count, err := s.DataService.Count(locator)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get count")
		return
	}
	render.JSON(w, r, JSON{"count": count, "locator": locator})
}

// POST /count?site=siteID - get number of comments for posts from post body
func (s *Rest) countMultiCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	posts := []string{}
	if err := render.DecodeJSON(http.MaxBytesReader(w, r.Body, hardBodyLimit), &posts); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get list of posts from request")
		return
	}

	// key could be long for multiple posts, make it sha1
	key := cache.URLKey(r) + strings.Join(posts, ",")
	hasher := sha1.New()
	if _, err := hasher.Write([]byte(key)); err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't make sha1 for list of urls")
		return
	}
	sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	data, err := s.Cache.Get(cache.Key(sha, siteID), func() ([]byte, error) {
		counts, e := s.DataService.Counts(siteID, posts)
		if e != nil {
			return nil, e
		}
		return encodeJSONWithHTML(counts)
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get counts for "+siteID)
		return
	}
	renderJSONFromBytes(w, r, data)
}

// GET /list?site=siteID&limit=50&skip=10 - list posts with comments
func (s *Rest) listCtrl(w http.ResponseWriter, r *http.Request) {

	siteID := r.URL.Query().Get("site")
	limit, skip := 0, 0

	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil {
		limit = v
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("skip")); err == nil {
		skip = v
	}

	data, err := s.Cache.Get(cache.Key(cache.URLKey(r), siteID), func() ([]byte, error) {
		posts, e := s.DataService.List(siteID, limit, skip)
		if e != nil {
			return nil, e
		}
		return encodeJSONWithHTML(posts)
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get list of comments for "+siteID)
		return
	}
	renderJSONFromBytes(w, r, data)
}

// PUT /vote/{id}?site=siteID&url=post-url&vote=1 - vote for/against comment
func (s *Rest) voteCtrl(w http.ResponseWriter, r *http.Request) {

	user, err := rest.GetUserInfo(r)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusUnauthorized, err, "can't get user info")
		return
	}
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	id := chi.URLParam(r, "id")
	log.Printf("[DEBUG] vote for comment %s", id)

	vote := r.URL.Query().Get("vote") == "1"

	comment, err := s.DataService.Vote(locator, id, user.ID, vote)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't vote for comment")
		return
	}
	s.Cache.Flush(locator.URL)
	render.JSON(w, r, JSON{"id": comment.ID, "score": comment.Score})
}

// GET /userdata?site=siteID - exports all data about the user as a json fragments
func (s *Rest) userAllDataCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	user, err := rest.GetUserInfo(r)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusUnauthorized, err, "can't get user info")
		return
	}
	userB, err := json.Marshal(&user)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't marshal user info")
		return
	}

	exportFile := fmt.Sprintf("%s-%s-%s.json.gz", siteID, user.ID, time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", "attachment;filename="+exportFile)
	gzWriter := gzip.NewWriter(w)
	defer func() {
		if e := gzWriter.Close(); e != nil {
			log.Printf("[WARN] can't close gzip writer, %s", e)
		}
	}()

	if _, e := gzWriter.Write(userB); e != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, e, "can't write user info")
		return
	}

	for i := 0; i < 1000; i++ {
		comments, err := s.DataService.User(siteID, user.ID, 1000, i*1000)
		if err != nil {
			rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't write user comments")
			return
		}
		b, err := json.Marshal(comments)
		if err != nil {
			rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't marshal user comments")
			return
		}
		if _, e := gzWriter.Write(b); e != nil {
			rest.SendErrorJSON(w, r, http.StatusInternalServerError, e, "can't write user comment")
			return
		}
		if len(comments) != 1000 {
			break
		}
	}
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
