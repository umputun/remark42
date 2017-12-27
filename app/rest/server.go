package rest

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/patrickmn/go-cache"

	"github.com/umputun/remark/app/migrator"
	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/rest/format"
	"github.com/umputun/remark/app/store"
)

// Server is a rest access server
type Server struct {
	Version      string
	DataService  store.Service
	Admins       []string
	AuthGoogle   *auth.Provider
	AuthGithub   *auth.Provider
	SessionStore *sessions.FilesystemStore
	Exporter     migrator.Exporter
	DevMode      bool

	mod       admin
	respCache *cache.Cache
}

// Run the lister and request's router, activate rest server
func (s *Server) Run() {
	log.Print("[INFO] activate rest server")

	applyDevMode := func(mode auth.Mode) (modes []auth.Mode) {
		modes = append(modes, mode)
		if s.DevMode {
			modes = append(modes, auth.Developer)
		}
		return modes
	}

	s.respCache = cache.New(time.Hour, 5*time.Minute)

	router := chi.NewRouter()
	router.Use(middleware.RealIP, Recoverer)
	router.Use(middleware.Throttle(1000), middleware.Timeout(60*time.Second))
	router.Use(auth.Auth(s.SessionStore, s.Admins, applyDevMode(auth.Anonymous)))
	router.Use(Limiter(10), AppInfo("remark", s.Version), Ping, Logger(LogAll))

	// If you aren't using gorilla/mux, you need to wrap your handlers with context.ClearHandler
	router.Use(context.ClearHandler)

	router.Get("/login/google", s.AuthGoogle.LoginHandler)
	router.Get("/auth/google", s.AuthGoogle.AuthHandler)
	router.Get("/logout", s.AuthGithub.LogoutHandler) // can hit any provider
	router.Get("/login/github", s.AuthGithub.LoginHandler)
	router.Get("/auth/github", s.AuthGithub.AuthHandler)

	router.Route("/api/v1", func(rapi chi.Router) {
		rapi.Get("/find", s.findCommentsCtrl)
		rapi.Get("/id/{id}", s.commentByIDCtrl)
		rapi.Get("/comments", s.findUserCommentsCtrl)
		rapi.Get("/last/{max}", s.lastCommentsCtrl)
		rapi.Get("/count", s.countCtrl)

		rapi.With(auth.Auth(s.SessionStore, s.Admins, applyDevMode(auth.Full))).Group(func(rauth chi.Router) {
			rauth.Post("/comment", s.createCommentCtrl)
			rauth.Get("/user", s.userInfoCtrl)
			rauth.Put("/vote/{id}", s.voteCtrl)

			s.mod = admin{dataService: s.DataService, exporter: s.Exporter, respCache: s.respCache}
			rauth.Mount("/admin", s.mod.routes())
		})

	})

	s.addFileServer(router, "/web", http.Dir(filepath.Join(".", "web")))

	log.Fatal(http.ListenAndServe(":8080", router))
}

func (s *Server) addFileServer(r chi.Router, path string, root http.FileSystem) {
	fs := http.StripPrefix(path, http.FileServer(root))

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// don't show dirs, just serve files
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		fs.ServeHTTP(w, r)
	}))
}

// POST /comment
func (s *Server) createCommentCtrl(w http.ResponseWriter, r *http.Request) {

	comment := store.Comment{}
	if err := render.DecodeJSON(r.Body, &comment); err != nil {
		log.Printf("[WARN] can't bind request %s", err)
		httpError(w, r, http.StatusBadRequest, err, "can't bind comment")
		return
	}

	user, err := auth.GetUserInfo(r)
	if err != nil { // this not suppose to happen (handled by Auth), just dbl-check
		httpError(w, r, http.StatusUnauthorized, err, "can't get user info")
		return
	}

	comment.ID = ""                 // don't allow user to define ID, force auto-gen
	comment.Timestamp = time.Time{} // reset time, force auto-gen
	comment.User = user
	comment.User.IP = strings.Split(r.RemoteAddr, ":")[0]

	log.Printf("[DEBUG] create comment %+v", comment)

	// check if user blocked
	if s.mod.checkBlocked(store.Locator{}, comment.User) {
		log.Printf("[WARN] user %s rejected (blocked)", err)
		httpError(w, r, http.StatusForbidden, errors.New("rejected"), "user blocked")
		return
	}

	id, err := s.DataService.Create(comment)
	if err != nil {
		log.Printf("[WARN] can't save comment, %s", err)
		httpError(w, r, http.StatusInternalServerError, err, "can't save comment")
		return
	}

	s.respCache.Flush()

	render.Status(r, http.StatusAccepted)
	render.JSON(w, r, JSON{"id": id, "loc": comment.Locator})
}

// DELETE /comment/{id}?site=siteID&url=post-url
func (s *Server) deleteCommentCtrl(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	log.Printf("[DEBUG] delete comment %s", id)

	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	err := s.DataService.Delete(locator, id)
	if err != nil {
		log.Printf("[WARN] can't delete comment %s %+v, %s", id, locator, err)
		httpError(w, r, http.StatusInternalServerError, err, "can't delete comment")
		return
	}

	s.respCache.Flush()

	render.Status(r, http.StatusOK)
	render.JSON(w, r, JSON{"id": id, "loc": locator})
}

// GET /find?site=siteID&url=post-url&format=tree&sort=-time
func (s *Server) findCommentsCtrl(w http.ResponseWriter, r *http.Request) {
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	log.Printf("[DEBUG] get comments for %+v", locator)

	cacheKey := r.URL.String()
	if comments, ok := s.respCache.Get(cacheKey); ok {
		log.Printf("[DEBUG] cash hit for %s", cacheKey)
		renderJSONWithHTML(w, r, comments)
		return
	}

	comments, err := s.DataService.Find(store.Request{Locator: locator, Sort: r.URL.Query().Get("sort")})
	if err != nil {
		log.Printf("[WARN] can't get comments for %+v, %s", locator, err)
		httpError(w, r, http.StatusInternalServerError, err, "can't load comments comment")
		return
	}
	if r.URL.Query().Get("format") == "tree" {
		s.respCache.Set(cacheKey, comments, time.Hour)
		renderJSONWithHTML(w, r, format.MakeTree(comments, r.URL.Query().Get("sort")))
		return
	}

	s.respCache.Set(cacheKey, comments, time.Hour)
	renderJSONWithHTML(w, r, comments)
}

// GET /last/{max}?site=siteID
func (s *Server) lastCommentsCtrl(w http.ResponseWriter, r *http.Request) {

	max, err := strconv.Atoi(chi.URLParam(r, "max"))
	if err != nil {
		max = 0
	}

	cacheKey := r.URL.String()
	if comments, ok := s.respCache.Get(cacheKey); ok {
		log.Printf("[DEBUG] cash hit for %s", cacheKey)
		renderJSONWithHTML(w, r, comments)
		return
	}

	comments, err := s.DataService.Last(store.Locator{SiteID: r.URL.Query().Get("site")}, max)
	if err != nil {
		log.Printf("[WARN] can't get last comments, %s", err)
		httpError(w, r, http.StatusInternalServerError, err, "can't get last comments")
		return
	}
	s.respCache.Set(cacheKey, comments, time.Hour)

	render.Status(r, http.StatusOK)
	renderJSONWithHTML(w, r, comments)
}

// GET /id/{id}?site=siteID&url=post-url
func (s *Server) commentByIDCtrl(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}

	log.Printf("[DEBUG] get comments by id %s, %+v", id, locator)

	comment, err := s.DataService.GetByID(locator, id)
	if err != nil {
		log.Printf("[WARN] can't get comment, %s", err)
		httpError(w, r, http.StatusInternalServerError, err, "can't get comment by id")
		return
	}
	render.Status(r, http.StatusOK)
	renderJSONWithHTML(w, r, comment)
}

// GET /comments?site=siteID&user=id
func (s *Server) findUserCommentsCtrl(w http.ResponseWriter, r *http.Request) {

	userID := r.URL.Query().Get("user")

	log.Printf("[DEBUG] get comments by userID %s", userID)

	cacheKey := r.URL.String()
	if comments, ok := s.respCache.Get(cacheKey); ok {
		log.Printf("[DEBUG] cash hit for %s", cacheKey)
		renderJSONWithHTML(w, r, comments)
		return
	}

	comments, err := s.DataService.GetByUser(store.Locator{SiteID: r.URL.Query().Get("site")}, userID)
	if err != nil {
		log.Printf("[WARN] can't get comment, %s", err)
		httpError(w, r, http.StatusBadRequest, err, "can't get comment by user id")
		return
	}

	s.respCache.Set(cacheKey, comments, time.Hour)
	render.Status(r, http.StatusOK)
	renderJSONWithHTML(w, r, comments)
}

// GET /user
func (s *Server) userInfoCtrl(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetUserInfo(r)
	if err != nil {
		httpError(w, r, http.StatusUnauthorized, err, "can't get user info")
		return
	}
	render.JSON(w, r, user)
}

// GET /count?site=siteID&url=post-url
func (s *Server) countCtrl(w http.ResponseWriter, r *http.Request) {
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	count, err := s.DataService.Count(locator)
	if err != nil {
		httpError(w, r, http.StatusBadRequest, err, "can't get count")
		return
	}
	render.JSON(w, r, JSON{"count": count, "loc": locator})
}

// PUT /vote/{id}?site=siteID&url=post-url&vote=1
func (s *Server) voteCtrl(w http.ResponseWriter, r *http.Request) {

	user, err := auth.GetUserInfo(r)
	if err != nil {
		httpError(w, r, http.StatusUnauthorized, err, "can't get user info")
		return
	}
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	id := chi.URLParam(r, "id")
	log.Printf("[DEBUG] vote for comment %s", id)

	vote := r.URL.Query().Get("vote") == "1"

	comment, err := s.DataService.Vote(locator, id, user.ID, vote)
	if err != nil {
		log.Printf("[WARN] vote rejected for %s - %s, %s", user.ID, id, err)
		httpError(w, r, http.StatusBadRequest, err, "can't vote for comment")
		return
	}
	s.respCache.Flush()
	render.JSON(w, r, JSON{"id": comment.ID, "score": comment.Score})
}

func httpError(w http.ResponseWriter, r *http.Request, code int, err error, details string) {
	render.Status(r, code)
	render.JSON(w, r, JSON{"error": err.Error(), "details": details})
}

// renderJSONWithHTML allows html tags
func renderJSONWithHTML(w http.ResponseWriter, r *http.Request, v interface{}) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if status, ok := r.Context().Value(render.StatusCtxKey).(int); ok {
		w.WriteHeader(status)
	}
	_, _ = w.Write(buf.Bytes())
}
