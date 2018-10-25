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

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/hashicorp/go-multierror"

	"github.com/umputun/remark/backend/app/rest"
	"github.com/umputun/remark/backend/app/rest/auth"
	"github.com/umputun/remark/backend/app/rest/cache"
	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/service"
)

// POST /comment - adds comment, resets all immutable fields
func (s *Rest) createCommentCtrl(w http.ResponseWriter, r *http.Request) {

	comment := store.Comment{}
	if err := render.DecodeJSON(http.MaxBytesReader(w, r.Body, hardBodyLimit), &comment); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't bind comment")
		return
	}

	user := rest.MustGetUserInfo(r)
	log.Printf("[DEBUG] create comment %+v", comment)

	comment.PrepareUntrusted() // clean all fields user not supposed to set
	comment.User = user
	comment.User.IP = strings.Split(r.RemoteAddr, ":")[0]

	comment.Orig = comment.Text // original comment text, prior to md render
	if err := s.DataService.ValidateComment(&comment); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "invalid comment")
		return
	}
	comment = s.CommentFormatter.Format(comment)

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
	s.Cache.Flush(cache.Flusher(comment.Locator.SiteID).
		Scopes(comment.Locator.URL, lastCommentsScope, comment.User.ID, comment.Locator.SiteID))

	s.NotifyService.Submit(finalComment)

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, &finalComment)
}

// PUT /comment/{id}?site=siteID&url=post-url - update comment
func (s *Rest) updateCommentCtrl(w http.ResponseWriter, r *http.Request) {

	edit := struct {
		Text    string
		Summary string
		Delete  bool
	}{}

	if err := render.DecodeJSON(http.MaxBytesReader(w, r.Body, hardBodyLimit), &edit); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't bind comment")
		return
	}

	user := rest.MustGetUserInfo(r)
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	id := chi.URLParam(r, "id")

	log.Printf("[DEBUG] update comment %s", id)

	var currComment store.Comment
	var err error
	if currComment, err = s.DataService.Get(locator, id); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't find comment")
		return
	}

	if currComment.User.ID != user.ID {
		rest.SendErrorJSON(w, r, http.StatusForbidden, errors.New("rejected"), "can not edit comments for other users")
		return
	}

	editReq := service.EditRequest{
		Text:    s.CommentFormatter.FormatText(edit.Text),
		Orig:    edit.Text,
		Summary: edit.Summary,
		Delete:  edit.Delete,
	}

	res, err := s.DataService.EditComment(locator, id, editReq)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't update comment")
		return
	}

	s.Cache.Flush(cache.Flusher(locator.SiteID).Scopes(locator.URL, lastCommentsScope, user.ID))
	render.JSON(w, r, res)
}

// GET /user?site=siteID - returns user info
func (s *Rest) userInfoCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)
	if siteID := r.URL.Query().Get("site"); siteID != "" {
		user.Verified = s.DataService.IsVerified(siteID, user.ID)
	}

	render.JSON(w, r, user)
}

// PUT /vote/{id}?site=siteID&url=post-url&vote=1 - vote for/against comment
func (s *Rest) voteCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	id := chi.URLParam(r, "id")
	log.Printf("[DEBUG] vote for comment %s", id)

	vote := r.URL.Query().Get("vote") == "1"

	// check if user blocked
	if s.adminService.checkBlocked(locator.SiteID, user) {
		rest.SendErrorJSON(w, r, http.StatusForbidden, errors.New("rejected"), "user blocked")
		return
	}

	comment, err := s.DataService.Vote(locator, id, user.ID, vote)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't vote for comment")
		return
	}
	s.Cache.Flush(cache.Flusher(locator.SiteID).Scopes(locator.URL, comment.User.ID))
	render.JSON(w, r, JSON{"id": comment.ID, "score": comment.Score})
}

// GET /userdata?site=siteID - exports all data about the user as a json with user info and list of all comments
func (s *Rest) userAllDataCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	user := rest.MustGetUserInfo(r)
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

	write := func(val []byte) error {
		_, e := gzWriter.Write(val)
		return e
	}

	var merr error

	merr = multierror.Append(merr, write([]byte(`{"info": `)))     // send user prefix
	merr = multierror.Append(merr, write(userB))                   // send user info
	merr = multierror.Append(merr, write([]byte(`, "comments":`))) // send comments prefix

	// get comments in 100 in each paginated request
	for i := 0; i < 100; i++ {
		comments, err := s.DataService.User(siteID, user.ID, 100, i*100)
		if err != nil {
			rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't get user comments")
			return
		}
		b, err := json.Marshal(comments)
		if err != nil {
			rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't marshal user comments")
			return
		}

		merr = multierror.Append(merr, write(b))
		if len(comments) != 100 {
			break
		}
	}

	merr = multierror.Append(merr, write([]byte(`}`)))
	if merr.(*multierror.Error).ErrorOrNil() != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, merr, "can't write user info")
		return
	}

}

// POST /deleteme?site_id=site - requesting delete of all user info
// makes jwt with user info and sends it back as a part of json response
func (s *Rest) deleteMeCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)
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
	claims.Flags.DeleteMe = true // prevent this token from being used for login

	tokenStr, err := s.Authenticator.JWTService.Token(&claims)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't make token")
		return
	}

	link := fmt.Sprintf("%s/web/deleteme.html?token=%s", s.RemarkURL, tokenStr)
	render.JSON(w, r, JSON{"site": siteID, "user_id": user.ID, "token": tokenStr, "link": link})
}
