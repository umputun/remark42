package api

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	blackfriday "gopkg.in/russross/blackfriday.v2"

	"github.com/umputun/remark/app/rest"
	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/store"
	"github.com/umputun/remark/app/store/service"
)

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

// GET /user - returns user info
func (s *Rest) userInfoCtrl(w http.ResponseWriter, r *http.Request) {
	user, err := rest.GetUserInfo(r)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusUnauthorized, err, "can't get user info")
		return
	}
	render.JSON(w, r, user)
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

// GET /userdata?site=siteID - exports all data about the user as a json with user info and list of all comments
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

	// send prefix
	if _, e := gzWriter.Write([]byte(`{"info": `)); e != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, e, "can't write user info")
		return
	}
	// send user info
	if _, e := gzWriter.Write(userB); e != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, e, "can't write user info")
		return
	}
	if _, e := gzWriter.Write([]byte(`, "comments":`)); e != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, e, "can't write user info")
		return
	}

	// get comments in 100 in each paginated request
	for i := 0; i < 100; i++ {
		comments, err := s.DataService.User(siteID, user.ID, 100, i*100)
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
		if len(comments) != 100 {
			break
		}
	}
	if _, e := gzWriter.Write([]byte(`}`)); e != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, e, "can't write user info")
		return
	}

}

// POST /deleteme?site_id=site - requesting delete of all user info
// makes jwt with user info and sends it back as a part of json response
func (s *Rest) deleteMeCtrl(w http.ResponseWriter, r *http.Request) {
	user, err := rest.GetUserInfo(r)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusUnauthorized, err, "can't get user info")
		return
	}
	siteID := r.URL.Query().Get("site")

	claims := auth.CustomClaims{
		SiteID: siteID,
		StandardClaims: jwt.StandardClaims{
			Issuer:    "remark42",
			ExpiresAt: time.Now().AddDate(0, 3, 0).Unix(),
			NotBefore: time.Now().Add(-1 * time.Minute).Unix(),
		},
		User: &user,
	}

	tokenStr, err := s.Authenticator.JWTService.Token(&claims)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't make token")
		return
	}
	render.JSON(w, r, JSON{"site": siteID, "user_id": user.ID, "token": tokenStr})
}
