package api

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/go-pkgz/auth/token"
	log "github.com/go-pkgz/lgr"
	R "github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/cache"
	multierror "github.com/hashicorp/go-multierror"

	"github.com/umputun/remark/backend/app/rest"
	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/service"
)

// POST /comment - adds comment, resets all immutable fields
func (s *Rest) createCommentCtrl(w http.ResponseWriter, r *http.Request) {

	comment := store.Comment{}
	if err := render.DecodeJSON(http.MaxBytesReader(w, r.Body, hardBodyLimit), &comment); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't bind comment", rest.ErrDecode)
		return
	}

	user := rest.MustGetUserInfo(r)

	comment.PrepareUntrusted() // clean all fields user not supposed to set
	comment.User = user
	comment.User.IP = strings.Split(r.RemoteAddr, ":")[0]

	comment.Orig = comment.Text // original comment text, prior to md render
	if err := s.DataService.ValidateComment(&comment); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "invalid comment", rest.ErrCommentValidation)
		return
	}
	comment = s.CommentFormatter.Format(comment)

	// check if user blocked
	if s.adminService.checkBlocked(comment.Locator.SiteID, comment.User) {
		rest.SendErrorJSON(w, r, http.StatusForbidden, errors.New("rejected"), "user blocked", rest.ErrUserBlocked)
		return
	}

	if s.isReadOnly(comment.Locator) {
		rest.SendErrorJSON(w, r, http.StatusForbidden, errors.New("rejected"), "old post, read-only", rest.ErrReadOnly)
		return
	}

	id, err := s.DataService.Create(comment)
	if err == service.ErrRestrictedWordsFound {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "invalid comment", rest.ErrCommentValidation)
		return
	}
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't save comment", rest.ErrInternal)
		return
	}

	// DataService modifies comment
	finalComment, err := s.DataService.Get(comment.Locator, id)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't load created comment", rest.ErrInternal)
		return
	}
	s.Cache.Flush(cache.Flusher(comment.Locator.SiteID).
		Scopes(comment.Locator.URL, lastCommentsScope, comment.User.ID, comment.Locator.SiteID))

	if s.NotifyService != nil {
		s.NotifyService.Submit(finalComment)
	}

	log.Printf("[DEBUG] created commend %+v", finalComment)

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
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't bind comment", rest.ErrDecode)
		return
	}

	user := rest.MustGetUserInfo(r)
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	id := chi.URLParam(r, "id")

	log.Printf("[DEBUG] update comment %s", id)

	var currComment store.Comment
	var err error
	if currComment, err = s.DataService.Get(locator, id); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't find comment", rest.ErrCommentNotFound)
		return
	}

	if currComment.User.ID != user.ID {
		rest.SendErrorJSON(w, r, http.StatusForbidden, errors.New("rejected"),
			"can not edit comments for other users", rest.ErrNoAccess)
		return
	}

	editReq := service.EditRequest{
		Text:    s.CommentFormatter.FormatText(edit.Text),
		Orig:    edit.Text,
		Summary: edit.Summary,
		Delete:  edit.Delete,
	}

	res, err := s.DataService.EditComment(locator, id, editReq)
	if err == service.ErrRestrictedWordsFound {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "invalid comment", rest.ErrCommentValidation)
		return
	}
	if err != nil {
		code := rest.ErrCommentRejected
		switch {
		case strings.HasPrefix(err.Error(), "too late to edit"):
			code = rest.ErrCommentEditExpired
		case strings.HasPrefix(err.Error(), "parent comment with reply can't be edited"):
			code = rest.ErrCommentEditChanged
		}
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't update comment", code)
		return
	}

	s.Cache.Flush(cache.Flusher(locator.SiteID).Scopes(locator.SiteID, locator.URL, lastCommentsScope, user.ID))
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

	if s.isReadOnly(locator) {
		rest.SendErrorJSON(w, r, http.StatusForbidden, errors.New("rejected"), "old post, read-only", rest.ErrReadOnly)
		return
	}

	// check if user blocked
	if s.adminService.checkBlocked(locator.SiteID, user) {
		rest.SendErrorJSON(w, r, http.StatusForbidden, errors.New("rejected"), "user blocked", rest.ErrUserBlocked)
		return
	}

	comment, err := s.DataService.Vote(locator, id, user.ID, vote)
	if err != nil {
		code := rest.ErrVoteRejected
		switch {
		case strings.Contains(err.Error(), "can not vote for his own comment"):
			code = rest.ErrVoteSelf
		case strings.Contains(err.Error(), "already voted for"):
			code = rest.ErrVoteDbl
		case strings.Contains(err.Error(), "maximum number of votes exceeded for comment"):
			code = rest.ErrVoteMax
		case strings.Contains(err.Error(), "minimal score reached for comment"):
			code = rest.ErrVoteMinScore
		}
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't vote for comment", code)
		return
	}
	s.Cache.Flush(cache.Flusher(locator.SiteID).Scopes(locator.URL, comment.User.ID))
	render.JSON(w, r, R.JSON{"id": comment.ID, "score": comment.Score})
}

// GET /userdata?site=siteID - exports all data about the user as a json with user info and list of all comments
func (s *Rest) userAllDataCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	user := rest.MustGetUserInfo(r)
	userB, err := json.Marshal(&user)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't marshal user info", rest.ErrInternal)
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
			rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't get user comments", rest.ErrInternal)
			return
		}
		b, err := json.Marshal(comments)
		if err != nil {
			rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't marshal user comments", rest.ErrInternal)
			return
		}

		merr = multierror.Append(merr, write(b))
		if len(comments) != 100 {
			break
		}
	}

	merr = multierror.Append(merr, write([]byte(`}`)))
	if merr.(*multierror.Error).ErrorOrNil() != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, merr, "can't write user info", rest.ErrInternal)
		return
	}

}

// POST /deleteme?site_id=site - requesting delete of all user info
// makes jwt with user info and sends it back as a part of json response
func (s *Rest) deleteMeCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)
	siteID := r.URL.Query().Get("site")

	claims := token.Claims{
		StandardClaims: jwt.StandardClaims{
			Audience:  siteID,
			Issuer:    "remark42",
			ExpiresAt: time.Now().AddDate(0, 3, 0).Unix(),
			NotBefore: time.Now().Add(-1 * time.Minute).Unix(),
		},
		User: &token.User{
			ID:   user.ID,
			Name: user.Name,
			Attributes: map[string]interface{}{
				"delete_me": true, // prevents this token from being used for login
			},
		},
	}

	tokenStr, err := s.Authenticator.TokenService().Token(claims)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't make token", rest.ErrInternal)
		return
	}

	link := fmt.Sprintf("%s/web/deleteme.html?token=%s", s.RemarkURL, tokenStr)
	render.JSON(w, r, R.JSON{"site": siteID, "user_id": user.ID, "token": tokenStr, "link": link})
}

// POST /image - save image with form request
func (s *Rest) savePictureCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)

	if err := r.ParseMultipartForm(5 * 1024 * 1024); err != nil { // 5M max memory, if bigger will make a file
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't parse multipart form", rest.ErrDecode)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't get image file from the request", rest.ErrInternal)
		return
	}
	defer func() { _ = file.Close() }()

	picName := fmt.Sprintf("%s_%d_%s", user.ID, time.Now().Nanosecond(), header.Filename)
	id, err := s.ImageService.Save(picName, file)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't save image", rest.ErrInternal)
		return
	}

	render.JSON(w, r, R.JSON{"location": id})
}

func (s *Rest) isReadOnly(locator store.Locator) bool {
	if s.ReadOnlyAge > 0 {
		// check RO by age
		if info, e := s.DataService.Info(locator, s.ReadOnlyAge); e == nil && info.ReadOnly {
			return true
		}
	}
	return s.DataService.IsReadOnly(locator) // ro manually
}
