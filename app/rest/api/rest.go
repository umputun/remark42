package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/gorilla/context"
	"github.com/pkg/errors"
	"gopkg.in/russross/blackfriday.v2"

	"github.com/umputun/remark/app/migrator"
	"github.com/umputun/remark/app/notifier"
	"github.com/umputun/remark/app/rest"
	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/store"
)

// Rest is a rest access server
type Rest struct {
	Version string

	DataService   store.Service
	Authenticator auth.Authenticator
	Exporter      migrator.Exporter
	Cache         rest.LoadingCache
	Notifier      notifier.Interface

	httpServer *http.Server
	mod        admin
}

// Run the lister and request's router, activate rest server
func (s *Rest) Run(port int) {
	log.Print("[INFO] activate rest server")

	if len(s.Authenticator.Admins) > 0 {
		log.Printf("[DEBUG] admins %+v", s.Authenticator.Admins)
	}

	router := chi.NewRouter()
	router.Use(middleware.RealIP, Recoverer)
	router.Use(middleware.Throttle(1000), middleware.Timeout(60*time.Second))
	router.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))

	// all request by default allow anonymous access
	router.Use(s.Authenticator.Auth(false))

	router.Use(AppInfo("remark42", s.Version), Ping, Logger(LogAll))
	router.Use(context.ClearHandler) // if you aren't using gorilla/mux, you need to wrap your handlers with context.ClearHandler

	// auth routes for all providers
	router.Route("/auth", func(r chi.Router) {
		for _, provider := range s.Authenticator.Providers {
			r.Mount("/"+provider.Name, provider.Routes()) // mount auth providers as /auth/{name}
		}
		if len(s.Authenticator.Providers) > 0 {
			// shortcut, can be any of providers, all logouts do the same - removes cookie
			r.Get("/logout", s.Authenticator.Providers[0].LogoutHandler)
		}
	})

	router.Mount(s.Authenticator.AvatarProxy.Routes())

	// api routes
	router.Route("/api/v1", func(rapi chi.Router) {

		// open routes
		rapi.Get("/find", s.findCommentsCtrl)
		rapi.Get("/id/{id}", s.commentByIDCtrl)
		rapi.Get("/comments", s.findUserCommentsCtrl)
		rapi.Get("/last/{max}", s.lastCommentsCtrl)
		rapi.Get("/count", s.countCtrl)
		rapi.Get("/list", s.listCtrl)
		rapi.Get("/config", s.configCtrl)
		rapi.Post("/preview", s.previewCommentCtrl)

		// protected routes, require auth
		rapi.With(s.Authenticator.Auth(true)).Group(func(rauth chi.Router) {
			rauth.Post("/comment", s.createCommentCtrl)
			rauth.Put("/comment/{id}", s.updateCommentCtrl)
			rauth.Get("/user", s.userInfoCtrl)
			rauth.Put("/vote/{id}", s.voteCtrl)
			rauth.Put("/notify", s.notifyActionCtrl)
			rauth.Get("/notify", s.notifyStatusCtrl)
			// admin routes, admin users only
			s.mod = admin{
				dataService:  s.DataService,
				exporter:     s.Exporter,
				cache:        s.Cache,
				defAvatarURL: s.Authenticator.AvatarProxy.Default(),
			}
			rauth.Mount("/admin", s.mod.routes(s.Authenticator.AdminOnly))
		})
	})

	// add robots and file server for static content from /web
	router.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		render.PlainText(w, r, "User-agent: *\nDisallow: /auth/\nDisallow: /api/\n")
	})
	s.addFileServer(router, "/web", http.Dir(filepath.Join(".", "web")))

	s.httpServer = &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: router}
	err := s.httpServer.ListenAndServe()
	log.Printf("[WARN] http server terminated, %s", err)
}

// POST /comment - adds comment, resets all immutable fields
func (s *Rest) createCommentCtrl(w http.ResponseWriter, r *http.Request) {

	comment := store.Comment{}
	if err := render.DecodeJSON(r.Body, &comment); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't bind comment")
		return
	}

	user, err := rest.GetUserInfo(r)
	if err != nil { // this not suppose to happen (handled by Auth), just dbl-check
		rest.SendErrorJSON(w, r, http.StatusUnauthorized, err, "can't get user info")
		return
	}

	// reset comment to initial state
	func() {
		comment.ID = ""                 // don't allow user to define ID, force auto-gen
		comment.Timestamp = time.Time{} // reset time, force auto-gen
		comment.Votes = make(map[string]bool)
		comment.Score = 0
		comment.Edit = nil
		comment.Pin = false
	}()

	comment.User = user
	comment.User.IP = strings.Split(r.RemoteAddr, ":")[0]

	// render markdown
	comment.Text = string(blackfriday.Run([]byte(comment.Text), blackfriday.WithNoExtensions()))

	log.Printf("[DEBUG] create comment %+v", comment)

	// check if user blocked
	if s.mod.checkBlocked(comment.Locator.SiteID, comment.User) {
		rest.SendErrorJSON(w, r, http.StatusForbidden, errors.New("rejected"), "user blocked")
		return
	}

	id, err := s.DataService.Create(comment)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't save comment")
		return
	}

	if err = s.Notifier.OnUpdate(comment); err != nil {
		log.Printf("[WARN] can't send notify event for %+v, %s", comment.Locator, err)
	}

	s.Cache.Flush() // reset all caches
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, JSON{"id": id, "locator": comment.Locator})
}

func (s *Rest) previewCommentCtrl(w http.ResponseWriter, r *http.Request) {
	comment := store.Comment{}
	if err := render.DecodeJSON(r.Body, &comment); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't bind comment")
		return
	}
	comment.Text = string(blackfriday.Run([]byte(comment.Text), blackfriday.WithNoExtensions()))
	render.HTML(w, r, comment.Text)
}

// PUT /comment/{id}?site=siteID&url=post-url - update comment
func (s *Rest) updateCommentCtrl(w http.ResponseWriter, r *http.Request) {

	edit := struct {
		Text    string
		Summary string
	}{}

	if err := render.DecodeJSON(r.Body, &edit); err != nil {
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

	// render markdown
	edit.Text = string(blackfriday.Run([]byte(edit.Text), blackfriday.WithNoExtensions()))

	log.Printf("[DEBUG] update comment %s, %+v", id, edit)

	var currComment store.Comment
	if currComment, err = s.DataService.Get(locator, id); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't find comment")
		return
	}

	if currComment.User.ID != user.ID {
		rest.SendErrorJSON(w, r, http.StatusForbidden, errors.New("rejected"), "can not edit comments for other users")
		return
	}

	res, err := s.DataService.EditComment(locator, id, edit.Text, store.Edit{Summary: edit.Summary})
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't update comment")
		return
	}

	s.Cache.Flush() // reset all caches
	render.JSON(w, r, res)
}

// GET /find?site=siteID&url=post-url&format=[tree|plain]&sort=[+/-time|+/-score]
// find comments for given post. Returns in tree or plain formats, sorted
func (s *Rest) findCommentsCtrl(w http.ResponseWriter, r *http.Request) {
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	log.Printf("[DEBUG] get comments for %+v", locator)

	data, err := s.Cache.Get(rest.URLKey(r), time.Hour, func() ([]byte, error) {
		comments, e := s.DataService.Find(locator, r.URL.Query().Get("sort"))
		if e != nil {
			return nil, e
		}
		maskedComments := s.mod.alterComments(comments, r)
		var b []byte
		switch r.URL.Query().Get("format") {
		case "tree":
			b, e = encodeJSONWithHTML(rest.MakeTree(maskedComments, r.URL.Query().Get("sort")))
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

// GET /last/{max}?site=siteID - last comments for the siteID, across all posts, sorted by time
func (s *Rest) lastCommentsCtrl(w http.ResponseWriter, r *http.Request) {

	log.Printf("[DEBUG] get last comments for %s", r.URL.Query().Get("site"))

	max, err := strconv.Atoi(chi.URLParam(r, "max"))
	if err != nil {
		max = 0
	}

	data, err := s.Cache.Get(rest.URLKey(r), time.Hour, func() ([]byte, error) {
		comments, e := s.DataService.Last(r.URL.Query().Get("site"), max)
		if e != nil {
			return nil, e
		}
		comments = s.mod.alterComments(comments, r)
		return encodeJSONWithHTML(comments)
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
	comment = s.mod.alterComments([]store.Comment{comment}, r)[0]
	render.Status(r, http.StatusOK)
	renderJSONWithHTML(w, r, comment)
}

// GET /comments?site=siteID&user=id - returns comments for given userID
func (s *Rest) findUserCommentsCtrl(w http.ResponseWriter, r *http.Request) {

	userID := r.URL.Query().Get("user")
	siteID := r.URL.Query().Get("site")

	resp := struct {
		Comments []store.Comment
		Count    int
	}{}

	log.Printf("[DEBUG] get comments for userID %s, %s", userID, siteID)

	data, err := s.Cache.Get(rest.URLKey(r), time.Hour, func() ([]byte, error) {
		comments, count, e := s.DataService.User(siteID, userID)
		if e != nil {
			return nil, e
		}
		comments = s.mod.alterComments(comments, r)
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
		Version      string   `json:"version"`
		EditDuration int      `json:"edit_duration"`
		Admins       []string `json:"admins"`
		Auth         []string `json:"auth_providers"`
	}

	cnf := config{
		Version:      s.Version,
		EditDuration: int(s.DataService.EditDuration.Seconds()),
		Admins:       s.Authenticator.Admins,
	}
	authNames := []string{}
	for _, ap := range s.Authenticator.Providers {
		authNames = append(authNames, ap.Name)
	}
	cnf.Auth = authNames
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

// GET /list?site=siteID&limit=50 - list posts with comments
func (s *Rest) listCtrl(w http.ResponseWriter, r *http.Request) {

	siteID := r.URL.Query().Get("site")
	limit := 0
	limitStr := r.URL.Query().Get("limit")
	if v, err := strconv.Atoi(limitStr); err == nil {
		limit = v
	}

	data, err := s.Cache.Get(rest.URLKey(r), 8*time.Hour, func() ([]byte, error) {
		posts, e := s.DataService.List(siteID, limit)
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
	s.Cache.Flush()
	render.JSON(w, r, JSON{"id": comment.ID, "score": comment.Score})
}

// PUT /notify?site=siteID&url=post-url&action=1 - subscribe/unsubscribe to notification
func (s *Rest) notifyActionCtrl(w http.ResponseWriter, r *http.Request) {
	user, err := rest.GetUserInfo(r)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusUnauthorized, err, "can't get user info")
		return
	}
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	action := "unknown"
	switch r.URL.Query().Get("action") {
	case "1":
		err = s.Notifier.Subscribe(locator, user)
		action = "subscribe"
	case "0":
		err = s.Notifier.UnSubscribe(locator, user)
		action = "unsubscribe"
	}
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't subscribe/unsubscribe for notifications")
		return
	}
	render.JSON(w, r, JSON{"locator": locator, "user": user.ID, "action": action})
}

// GET /notify?site=siteID&url=post-url - get notification status
func (s *Rest) notifyStatusCtrl(w http.ResponseWriter, r *http.Request) {
	user, err := rest.GetUserInfo(r)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusUnauthorized, err, "can't get user info")
		return
	}
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	status := "not subscribed"
	if st, err := s.Notifier.Status(locator, user); err == nil && st {
		status = "subscribed"
	}
	render.JSON(w, r, JSON{"locator": locator, "user": user.ID, "status": status})
}

// serves static files from /web
func (s *Rest) addFileServer(r chi.Router, path string, root http.FileSystem) {
	log.Printf("[INFO] run file server for %s", root)
	fs := http.StripPrefix(path, http.FileServer(root))
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// don't show dirs, just serve files
		if strings.HasSuffix(r.URL.Path, "/") && len(r.URL.Path) > 1 && r.URL.Path != "/web/" {
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
		return nil, errors.Wrap(err, "can't encode to json")
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
		log.Printf("[WARN] can't send response to %s, %s", r.RemoteAddr, err)
	}
}
