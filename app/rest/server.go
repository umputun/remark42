package rest

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/gorilla/sessions"
	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/store"
)

// Server is a rest access server
type Server struct {
	Version      string
	Store        store.Interface
	AuthGoogle   *auth.Google
	AuthGithub   *auth.Github
	SessionStore *sessions.FilesystemStore
}

// Run the lister and request's router, activate rest server
func (s *Server) Run() {
	log.Print("[INFO] activate rest server")
	router := chi.NewRouter()
	router.Use(middleware.RealIP, Recoverer)
	router.Use(middleware.Throttle(100), middleware.Timeout(60*time.Second))
	router.Use(Limiter(10), AppInfo("remark", s.Version), Ping)

	router.Get("/login/google", s.AuthGoogle.LoginHandler)
	router.Get("/auth/google", s.AuthGoogle.AuthHandler)

	router.Get("/login/github", s.AuthGithub.LoginHandler)
	router.Get("/auth/github", s.AuthGithub.AuthHandler)

	router.Post("/comment", s.createCommentCtrl)
	router.Delete("/comment/{id}", s.deleteCommentCtrl)
	router.Get("/find", s.getURLComments)
	router.With(Auth(s.SessionStore)).Get("/last/{max}", s.getLastComments)
	router.Get("/id/{id}", s.getByID)

	log.Fatal(http.ListenAndServe(":8080", router))
}

// POST /comment
func (s *Server) createCommentCtrl(w http.ResponseWriter, r *http.Request) {

	comment := store.Comment{}
	if err := render.DecodeJSON(r.Body, &comment); err != nil {
		log.Printf("[WARN] can't bind request %s", comment)
		httpError(w, r, http.StatusBadRequest, err, "can't bind comment")
		return
	}

	comment.User.IP = strings.Split(r.RemoteAddr, ":")[0]
	comment.User.Name = template.HTMLEscapeString(comment.User.Name)
	comment.Text = template.HTMLEscapeString(comment.Text)

	log.Printf("[INFO] create comment %+v", comment)

	id, err := s.Store.Create(comment)
	if err != nil {
		log.Printf("[WARN] can't save comment, %s", err)
		httpError(w, r, http.StatusInternalServerError, err, "can't save comment")
		return
	}

	render.Status(r, http.StatusAccepted)
	render.JSON(w, r, JSON{"id": id, "url": comment.Locator.URL})
}

// DELETE /comment/{id}?url=post-url
func (s *Server) deleteCommentCtrl(w http.ResponseWriter, r *http.Request) {

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		log.Printf("[WARN] bad id %s", chi.URLParam(r, "id"))
		httpError(w, r, http.StatusBadRequest, err, "can't parse id")
	}

	log.Printf("[INFO] delete comment %d", id)

	url := r.URL.Query().Get("url")
	err = s.Store.Delete(url, id)
	if err != nil {
		log.Printf("[WARN] can't delete comment, %s", err)
		httpError(w, r, http.StatusInternalServerError, err, "can't delete comment")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, JSON{"id": id, "url": url})
}

// GET /find?url=post-url
func (s *Server) getURLComments(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	log.Printf("[INFO] get comments for %s", url)

	comments, err := s.Store.Find(store.Request{Locator: store.Locator{URL: url}})
	if err != nil {
		log.Printf("[WARN] can't get comments for %s, %s", url, err)
		httpError(w, r, http.StatusInternalServerError, err, "can't load comments comment")
		return
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, comments)
}

// GET /last/{max}?url=abc
func (s *Server) getLastComments(w http.ResponseWriter, r *http.Request) {

	max, err := strconv.Atoi(chi.URLParam(r, "max"))
	if err != nil {
		max = 0
	}

	uinfoData, err := GetUserInfo(r)
	if err != nil {
		http.Error(w, "login required", http.StatusUnauthorized)
		return
	}
	log.Printf("[DEBUG] user: %+v", uinfoData)

	comments, err := s.Store.Last(store.Locator{}, max)
	if err != nil {
		log.Printf("[WARN] can't get last comments, %s", err)
		httpError(w, r, http.StatusInternalServerError, err, "can't get last comments")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, comments)
}

// GET /id/{id}?url=post-url
func (s *Server) getByID(w http.ResponseWriter, r *http.Request) {

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		log.Printf("[WARN] bad id %s", chi.URLParam(r, "id"))
		httpError(w, r, http.StatusBadRequest, err, "can't parse id")
	}
	url := r.URL.Query().Get("url")

	log.Printf("[INFO] get comments by id %d, %s", id, url)

	comment, err := s.Store.Get(store.Locator{URL: url}, id)
	if err != nil {
		log.Printf("[WARN] can't get comment, %s", err)
		httpError(w, r, http.StatusInternalServerError, err, "can't get comment by id")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, comment)
}

func httpError(w http.ResponseWriter, r *http.Request, code int, err error, details string) {
	render.Status(r, code)
	render.JSON(w, r, JSON{"error": err.Error(), "details": details})
}
