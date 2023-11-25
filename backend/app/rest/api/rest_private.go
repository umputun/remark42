package api

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec //not used for security
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-pkgz/auth"
	"github.com/go-pkgz/auth/token"
	cache "github.com/go-pkgz/lcw"
	log "github.com/go-pkgz/lgr"
	R "github.com/go-pkgz/rest"
	"github.com/golang-jwt/jwt"
	"github.com/hashicorp/go-multierror"

	"github.com/umputun/remark42/backend/app/notify"
	"github.com/umputun/remark42/backend/app/rest"
	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/engine"
	"github.com/umputun/remark42/backend/app/store/image"
	"github.com/umputun/remark42/backend/app/store/service"
	"github.com/umputun/remark42/backend/app/templates"
)

type private struct {
	dataService                privStore
	cache                      LoadingCache
	readOnlyAge                int
	commentFormatter           *store.CommentFormatter
	imageService               *image.Service
	notifyService              *notify.Service
	authenticator              *auth.Service
	telegramService            telegramService
	remarkURL                  string
	anonVote                   bool
	disableFancyTextFormatting bool // disables SmartyPants in the comment text rendering of the posted comments
}

// telegramService is a subset of Telegram service used for setting up user telegram notifications
type telegramService interface {
	AddToken(token, user, site string, expires time.Time)
	CheckToken(token, userID string) (telegram, site string, err error)
	GetBotUsername() string
}

type privStore interface {
	Create(comment store.Comment) (commentID string, err error)
	EditComment(locator store.Locator, commentID string, req service.EditRequest) (comment store.Comment, err error)
	Vote(req service.VoteReq) (comment store.Comment, err error)
	Get(locator store.Locator, commentID string, user store.User) (store.Comment, error)
	User(siteID, userID string, limit, skip int, user store.User) ([]store.Comment, error)
	GetUserEmail(siteID, userID string) (string, error)
	SetUserEmail(siteID, userID, value string) (string, error)
	GetUserTelegram(siteID, userID string) (string, error)
	SetUserTelegram(siteID, userID, value string) (string, error)
	DeleteUserDetail(siteID, userID string, detail engine.UserDetail) error
	ValidateComment(c *store.Comment) error
	IsVerified(siteID, userID string) bool
	IsReadOnly(locator store.Locator) bool
	IsBlocked(siteID, userID string) bool
	Info(locator store.Locator, readonlyAge int) (store.PostInfo, error)
}

// POST /preview, body is a comment, returns rendered html
func (s *private) previewCommentCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)

	comment := store.Comment{}
	if err := render.DecodeJSON(http.MaxBytesReader(w, r.Body, hardBodyLimit), &comment); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't bind comment", rest.ErrDecode)
		return
	}

	comment.User = user
	comment.Orig = comment.Text
	if err := s.dataService.ValidateComment(&comment); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "invalid comment", rest.ErrCommentValidation)
		return
	}

	comment = s.commentFormatter.Format(comment, s.disableFancyTextFormatting)
	comment.Sanitize()

	// check if images are valid, omit proxied images as they are lazy-loaded
	for _, id := range s.imageService.ExtractNonProxiedPictures(comment.Text) {
		err := s.imageService.ResetCleanupTimer(id)
		if err != nil {
			rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't load picture from the comment", rest.ErrImgNotFound)
			return
		}
	}

	render.HTML(w, r, comment.Text)
}

// POST /comment - adds comment, resets all immutable fields
func (s *private) createCommentCtrl(w http.ResponseWriter, r *http.Request) {
	comment := store.Comment{}
	if err := render.DecodeJSON(http.MaxBytesReader(w, r.Body, hardBodyLimit), &comment); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't bind comment", rest.ErrDecode)
		return
	}

	user := rest.MustGetUserInfo(r)
	if user.ID != "admin" && user.SiteID != comment.Locator.SiteID {
		rest.SendErrorJSON(w, r, http.StatusForbidden,
			fmt.Errorf("site mismatch, %q not allowed to post to %s", user.SiteID, comment.Locator.SiteID), "invalid site",
			rest.ErrCommentValidation)
		return
	}

	comment.PrepareUntrusted() // clean all fields user not supposed to set
	comment.User = user
	comment.User.IP = strings.Split(r.RemoteAddr, ":")[0]

	comment.Orig = comment.Text // original comment text, prior to md render
	if err := s.dataService.ValidateComment(&comment); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "invalid comment", rest.ErrCommentValidation)
		return
	}
	comment = s.commentFormatter.Format(comment, s.disableFancyTextFormatting)

	// check if images are valid, omit proxied images as they are lazy-loaded
	for _, id := range s.imageService.ExtractNonProxiedPictures(comment.Text) {
		_, err := s.imageService.Load(id)
		if err != nil {
			rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't load picture from the comment", rest.ErrImgNotFound)
			return
		}
	}

	// check if user blocked
	if s.dataService.IsBlocked(comment.Locator.SiteID, comment.User.ID) {
		rest.SendErrorJSON(w, r, http.StatusForbidden, fmt.Errorf("rejected"), "user blocked", rest.ErrUserBlocked)
		return
	}

	if s.isReadOnly(comment.Locator) {
		rest.SendErrorJSON(w, r, http.StatusForbidden, fmt.Errorf("rejected"), "old post, read-only", rest.ErrReadOnly)
		return
	}

	id, err := s.dataService.Create(comment)
	if errors.Is(err, service.ErrRestrictedWordsFound) {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "invalid comment", rest.ErrCommentRestrictWords)
		return
	}
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't save comment", rest.ErrInternal)
		return
	}

	// dataService modifies comment
	finalComment, err := s.dataService.Get(comment.Locator, id, rest.GetUserOrEmpty(r))
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't load created comment", rest.ErrInternal)
		return
	}
	s.cache.Flush(cache.Flusher(comment.Locator.SiteID).
		Scopes(comment.Locator.URL, lastCommentsScope, comment.User.ID, comment.Locator.SiteID))

	if s.notifyService != nil {
		s.notifyService.Submit(notify.Request{Comment: finalComment})
	}

	log.Printf("[DEBUG] created comment %+v", finalComment)

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, &finalComment)
}

// PUT /comment/{id}?site=siteID&url=post-url - update comment
func (s *private) updateCommentCtrl(w http.ResponseWriter, r *http.Request) {
	edit := struct {
		Text    string
		Summary string
		Delete  bool
	}{}

	if err := render.DecodeJSON(http.MaxBytesReader(w, r.Body, hardBodyLimit), &edit); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't read comment details from body", rest.ErrDecode)
		return
	}

	user := rest.MustGetUserInfo(r)
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	id := chi.URLParam(r, "id")

	log.Printf("[DEBUG] update comment %s", id)

	var currComment store.Comment
	var err error
	if currComment, err = s.dataService.Get(locator, id, rest.GetUserOrEmpty(r)); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't find comment", rest.ErrCommentNotFound)
		return
	}

	if currComment.User.ID != user.ID {
		rest.SendErrorJSON(w, r, http.StatusForbidden, fmt.Errorf("rejected"),
			"can not edit comments for other users", rest.ErrNoAccess)
		return
	}

	editReq := service.EditRequest{
		Text:    s.commentFormatter.FormatText(edit.Text, s.disableFancyTextFormatting),
		Orig:    edit.Text,
		Summary: edit.Summary,
		Delete:  edit.Delete,
		Admin:   user.Admin,
	}

	res, err := s.dataService.EditComment(locator, id, editReq)
	if errors.Is(err, service.ErrRestrictedWordsFound) {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "invalid comment", rest.ErrCommentValidation)
		return
	}

	if err != nil {
		code := parseError(err, rest.ErrCommentRejected)
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't update comment", code)
		return
	}

	s.cache.Flush(cache.Flusher(locator.SiteID).Scopes(locator.SiteID, locator.URL, lastCommentsScope, user.ID))
	render.JSON(w, r, res)
}

// GET /user?site=siteID - returns user info
func (s *private) userInfoCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)
	if siteID := r.URL.Query().Get("site"); siteID != "" {
		user.Verified = s.dataService.IsVerified(siteID, user.ID)

		email, err := s.dataService.GetUserEmail(siteID, user.ID)
		if err != nil {
			log.Printf("[WARN] can't read email for %s, %v", user.ID, err)
		}
		if len(email) > 0 {
			user.EmailSubscription = true
		}
	}

	render.JSON(w, r, user)
}

// PUT /vote/{id}?site=siteID&url=post-url&vote=1 - vote for/against comment
func (s *private) voteCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)
	if !s.anonVote && strings.HasPrefix(user.ID, "anonymous_") {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	id := chi.URLParam(r, "id")
	log.Printf("[DEBUG] vote for comment %s", id)

	vote := r.URL.Query().Get("vote") == "1"

	if s.isReadOnly(locator) {
		rest.SendErrorJSON(w, r, http.StatusForbidden, fmt.Errorf("rejected"), "old post, read-only", rest.ErrReadOnly)
		return
	}

	// check if user blocked
	if s.dataService.IsBlocked(locator.SiteID, user.ID) {
		rest.SendErrorJSON(w, r, http.StatusForbidden, fmt.Errorf("rejected"), "user blocked", rest.ErrUserBlocked)
		return
	}

	req := service.VoteReq{
		Locator:   locator,
		CommentID: id,
		UserID:    user.ID,
		UserIP:    strings.Split(r.RemoteAddr, ":")[0],
		Val:       vote,
	}
	comment, err := s.dataService.Vote(req)
	if err != nil {
		code := parseError(err, rest.ErrVoteRejected)
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't vote for comment", code)
		return
	}
	s.cache.Flush(cache.Flusher(locator.SiteID).Scopes(locator.URL, comment.User.ID))
	render.JSON(w, r, R.JSON{"id": comment.ID, "score": comment.Score})
}

// getEmailCtrl gets email address for authenticated user.
// GET /email?site=siteID
func (s *private) getEmailCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)
	siteID := r.URL.Query().Get("site")
	address, err := s.dataService.GetUserEmail(siteID, user.ID)
	if err != nil {
		log.Printf("[WARN] can't read email for %s, %v", user.ID, err)
	}

	render.JSON(w, r, R.JSON{"user": user, "address": address})
}

// sendEmailConfirmationCtrl gets address and siteID from query, makes confirmation token and sends it to user.
// In case user is logged in with the same email, and auto_confirm is true, confirm it right away.
// In case of quick confirmation, "updated" is set to true, otherwise - to false.
// POST /email/subscribe with site and address in json body
//
//nolint:dupl // too hard to deduplicate that logic, as then it's tricky to use SendErrorJSON
func (s *private) sendEmailConfirmationCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)

	subscribe := struct {
		Site        string
		Address     string
		autoConfirm bool
	}{autoConfirm: true}
	if err := render.DecodeJSON(http.MaxBytesReader(w, r.Body, hardBodyLimit), &subscribe); err != nil {
		if err != io.EOF {
			rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't parse request body", rest.ErrDecode)
			return
		}
		// old behavior fallback, reading from the query params. Auto confirm is false in this case.
		subscribe.Address = r.URL.Query().Get("address")
		subscribe.Site = r.URL.Query().Get("site")
		subscribe.autoConfirm = false
	}

	if subscribe.Address == "" {
		rest.SendErrorJSON(w, r, http.StatusBadRequest,
			fmt.Errorf("missing parameter"), "address parameter is required", rest.ErrInternal)
		return
	}
	existingAddress, getErr := s.dataService.GetUserEmail(subscribe.Site, user.ID)
	if getErr != nil {
		log.Printf("[WARN] can't read email for %s, %v", user.ID, getErr)
	}
	if subscribe.Address == existingAddress {
		rest.SendErrorJSON(w, r, http.StatusConflict,
			fmt.Errorf("already verified"), "email address is already verified for this user", rest.ErrInternal)
		return
	}

	// in case the user logged in with the same email as they try to subscribe with, confirm it right away
	// this behavior is different from the previous one and is hidden behind the autoConfirm flag,
	// which is true for the new API, and false for the old one
	//
	// nolint:gosec // this is not used for security purposes
	if subscribe.autoConfirm &&
		strings.HasPrefix(user.ID, "email_") &&
		strings.TrimPrefix(user.ID, "email_") == token.HashID(sha1.New(), subscribe.Address) {
		s.setEmail(w, r, user.ID, subscribe.Site, subscribe.Address)
		return
	}

	claims := token.Claims{
		Handshake: &token.Handshake{ID: user.ID + "::" + subscribe.Address},
		StandardClaims: jwt.StandardClaims{
			Audience:  r.URL.Query().Get("site"),
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
			NotBefore: time.Now().Add(-1 * time.Minute).Unix(),
			Issuer:    "remark42",
		},
	}

	tkn, err := s.authenticator.TokenService().Token(claims)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusForbidden, err, "failed to make verification token", rest.ErrInternal)
		return
	}

	s.notifyService.SubmitVerification(
		notify.VerificationRequest{
			SiteID: subscribe.Site,
			User:   user.Name,
			Email:  subscribe.Address,
			Token:  tkn,
		},
	)

	render.JSON(w, r, R.JSON{"user": user, "address": subscribe.Address, "updated": false})
}

// telegramSubscribeCtrl generates and verifies telegram notification request
// GET /telegram/subscribe?site=siteID<&tkn=token>
func (s *private) telegramSubscribeCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)

	if s.telegramService == nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError,
			fmt.Errorf("not enabled"), "telegram notifications are not enabled", rest.ErrActionRejected)
		return
	}

	queryToken := r.URL.Query().Get("tkn")
	if queryToken == "" {
		// GET /telegram/subscribe?site=siteID (No token supplied)
		siteID := r.URL.Query().Get("site")
		if siteID == "" {
			rest.SendErrorJSON(w, r, http.StatusBadRequest, fmt.Errorf("missing parameter"), "site parameter is required", rest.ErrInternal)
			return
		}
		// we don't care as much if we can't retrieve the current value of that field for the user, so ignore the error
		if existingAddress, _ := s.dataService.GetUserTelegram(siteID, user.ID); existingAddress != "" {
			rest.SendErrorJSON(w, r, http.StatusConflict,
				fmt.Errorf("already subscribed"), "telegram subscription is already set for this user, delete if first to re-subscribe", rest.ErrActionRejected)
			return
		}
		// Generate and send token
		tkn, err := randToken()
		if err != nil {
			rest.SendErrorJSON(w, r, http.StatusForbidden, err, "failed to generate verification token", rest.ErrInternal)
			return
		}
		expires := time.Now().Add(10 * time.Minute)

		s.telegramService.AddToken(tkn, user.ID, siteID, expires)

		render.JSON(w, r, R.JSON{"token": tkn, "bot": s.telegramService.GetBotUsername()})

		return
	}

	// GET /telegram/subscribe?tkn=token (verify token)
	var address, siteID string
	address, siteID, err := s.telegramService.CheckToken(queryToken, user.ID)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusNotFound, err, "request is not verified yet", rest.ErrInternal)
		return
	}

	log.Printf("[DEBUG] set telegram notifications for user %s", user.ID)

	val, err := s.dataService.SetUserTelegram(siteID, user.ID, address)
	if err != nil {
		code := parseError(err, rest.ErrInternal)
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't set telegram for user", code)
		return
	}

	render.JSON(w, r, R.JSON{"updated": true, "address": val})
}

// setConfirmedEmailCtrl uses provided token parameter (generated by sendEmailConfirmationCtrl) to set email and add it to user token
// POST /email/confirm with site and token in json body
func (s *private) setConfirmedEmailCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)

	confirm := struct {
		Site  string
		Token string
	}{}
	if err := render.DecodeJSON(http.MaxBytesReader(w, r.Body, hardBodyLimit), &confirm); err != nil {
		if err != io.EOF {
			rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't parse request body", rest.ErrDecode)
			return
		}
		// old behavior fallback, reading from the query params
		confirm.Token = r.URL.Query().Get("tkn")
		confirm.Site = r.URL.Query().Get("site")
	}

	if confirm.Token == "" {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, fmt.Errorf("missing parameter"), "token parameter is required", rest.ErrInternal)
		return
	}
	confClaims, err := s.authenticator.TokenService().Parse(confirm.Token)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusForbidden, err, "failed to verify confirmation token", rest.ErrInternal)
		return
	}

	if s.authenticator.TokenService().IsExpired(confClaims) {
		rest.SendErrorJSON(w, r, http.StatusForbidden, fmt.Errorf("expired"), "failed to verify confirmation token", rest.ErrInternal)
		return
	}

	// Handshake.ID is user.ID + "::" + address
	elems := strings.Split(confClaims.Handshake.ID, "::")
	if len(elems) != 2 || elems[0] != user.ID {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, fmt.Errorf("%s", confClaims.Handshake.ID), "invalid handshake token", rest.ErrInternal)
		return
	}
	address := elems[1]
	s.setEmail(w, r, user.ID, confirm.Site, address)
}

func (s *private) setEmail(w http.ResponseWriter, r *http.Request, userID, siteID, address string) {
	log.Printf("[DEBUG] set email for user %s", userID)

	val, err := s.dataService.SetUserEmail(siteID, userID, address)
	if err != nil {
		code := parseError(err, rest.ErrInternal)
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't set email for user", code)
		return
	}

	// update User.Email field
	claims, _, err := s.authenticator.TokenService().Get(r)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusForbidden, err, "failed to verify confirmation token", rest.ErrInternal)
		return
	}
	claims.User.Email = address
	if _, err = s.authenticator.TokenService().Set(w, claims); err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to set token", rest.ErrInternal)
		return
	}
	render.JSON(w, r, R.JSON{"updated": true, "address": val})
}

// POST/GET /email/unsubscribe.html?site=siteID&tkn=jwt - unsubscribe the user in token from email notifications
func (s *private) emailUnsubscribeCtrl(w http.ResponseWriter, r *http.Request) {
	tkn := r.URL.Query().Get("tkn")
	if tkn == "" {
		rest.SendErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("missing parameter"), "token parameter is required", rest.ErrInternal)
		return
	}
	siteID := r.URL.Query().Get("site")

	confClaims, err := s.authenticator.TokenService().Parse(tkn)
	if err != nil {
		rest.SendErrorHTML(w, r, http.StatusForbidden, err, "failed to verify confirmation token", rest.ErrInternal)
		return
	}

	if s.authenticator.TokenService().IsExpired(confClaims) {
		rest.SendErrorHTML(w, r, http.StatusForbidden, fmt.Errorf("expired"), "failed to verify confirmation token", rest.ErrInternal)
		return
	}

	// Handshake.ID is user.ID + "::" + address
	elems := strings.Split(confClaims.Handshake.ID, "::")
	if len(elems) != 2 {
		rest.SendErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("%s", confClaims.Handshake.ID), "invalid handshake token", rest.ErrInternal)
		return
	}
	userID := elems[0]
	address := elems[1]

	existingAddress, err := s.dataService.GetUserEmail(siteID, userID)
	if err != nil {
		// we don't care as much if we can't retrieve the current value of that field for the user,
		// as it's only used to check if we're trying to set to the same value it's already set to
		log.Printf("[WARN] can't read email for %s, %v", userID, err)
	}
	if existingAddress == "" {
		rest.SendErrorHTML(w, r, http.StatusConflict, fmt.Errorf("user is not subscribed"), "user does not have active email subscription", rest.ErrInternal)
		return
	}
	if address != existingAddress {
		rest.SendErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("wrong email unsubscription"), "email address in request does not match known for this user", rest.ErrInternal)
		return
	}

	log.Printf("[DEBUG] unsubscribe user %s", userID)

	if err = s.dataService.DeleteUserDetail(siteID, userID, engine.UserEmail); err != nil {
		code := parseError(err, rest.ErrInternal)
		rest.SendErrorHTML(w, r, http.StatusBadRequest, err, "can't delete email for user", code)
		return
	}
	// clean User.Email from the token, if user has the token
	claims, _, err := s.authenticator.TokenService().Get(r)
	if err != nil {
		log.Printf("[DEBUG] unsubscribed user doesn't have valid JWT token to update %s, %v", userID, err)
	}
	if claims.User != nil && claims.User.Email != "" {
		claims.User.Email = ""
		if _, err = s.authenticator.TokenService().Set(w, claims); err != nil {
			rest.SendErrorHTML(w, r, http.StatusInternalServerError, err, "failed to set token", rest.ErrInternal)
			return
		}
	}

	// MustExecute behaves like template.Execute, but panics if an error occurs.
	MustExecute := func(tmpl *template.Template, wr io.Writer, data interface{}) {
		if err := tmpl.Execute(wr, data); err != nil {
			panic(err)
		}
	}
	MustRead := func(path string) string {
		file, err := templates.Read(path)
		if err != nil {
			panic(err)
		}
		return string(file)
	}
	tmplstr := MustRead("email_unsubscribe.html.tmpl")
	tmpl := template.Must(template.New("unsubscribe").Parse(tmplstr))
	msg := bytes.Buffer{}
	MustExecute(tmpl, &msg, nil)
	render.HTML(w, r, msg.String())
}

// DELETE /email?site=siteID - removes user's email
func (s *private) deleteEmailCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)
	siteID := r.URL.Query().Get("site")
	log.Printf("[DEBUG] remove email for user %s", user.ID)

	if err := s.dataService.DeleteUserDetail(siteID, user.ID, engine.UserEmail); err != nil {
		code := parseError(err, rest.ErrInternal)
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't delete email for user", code)
		return
	}
	// clean User.Email from the token
	claims, _, err := s.authenticator.TokenService().Get(r)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusForbidden, err, "failed to verify confirmation token", rest.ErrInternal)
		return
	}
	if claims.User.Email != "" {
		claims.User.Email = ""
		if _, err = s.authenticator.TokenService().Set(w, claims); err != nil {
			rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to set token", rest.ErrInternal)
			return
		}
	}
	render.JSON(w, r, R.JSON{"deleted": true})
}

// DELETE /telegram?site=siteID - removes user's telegram
func (s *private) deleteTelegramCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)
	siteID := r.URL.Query().Get("site")
	log.Printf("[DEBUG] remove telegram for user %s", user.ID)

	if err := s.dataService.DeleteUserDetail(siteID, user.ID, engine.UserTelegram); err != nil {
		code := parseError(err, rest.ErrInternal)
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't delete telegram for user", code)
		return
	}
	render.JSON(w, r, R.JSON{"deleted": true})
}

// GET /userdata?site=siteID - exports all data about the user as a json with user info and list of all comments
func (s *private) userAllDataCtrl(w http.ResponseWriter, r *http.Request) {
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
		comments, errUser := s.dataService.User(siteID, user.ID, 100, i*100, rest.GetUserOrEmpty(r))
		if errUser != nil {
			rest.SendErrorJSON(w, r, http.StatusInternalServerError, errUser, "can't get user comments", rest.ErrInternal)
			return
		}
		b, errUser := json.Marshal(comments)
		if errUser != nil {
			rest.SendErrorJSON(w, r, http.StatusInternalServerError, errUser, "can't marshal user comments", rest.ErrInternal)
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
func (s *private) deleteMeCtrl(w http.ResponseWriter, r *http.Request) {
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

	tokenStr, err := s.authenticator.TokenService().Token(claims)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't make token", rest.ErrInternal)
		return
	}

	link := fmt.Sprintf("%s/web/deleteme.html?token=%s", s.remarkURL, tokenStr)
	render.JSON(w, r, R.JSON{"site": siteID, "user_id": user.ID, "token": tokenStr, "link": link})
}

// POST /image - save image with form request
func (s *private) savePictureCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)

	if err := r.ParseMultipartForm(5 * 1024 * 1024); err != nil { // 5M max memory, if bigger will make a file
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't parse multipart form", rest.ErrDecode)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't get image file from the request", rest.ErrInternal)
		return
	}
	defer func() { _ = file.Close() }()

	id, err := s.imageService.Save(user.ID, file)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't save image", rest.ErrInternal)
		return
	}

	render.JSON(w, r, R.JSON{"id": id})
}

func (s *private) isReadOnly(locator store.Locator) bool {
	if s.readOnlyAge > 0 {
		// check RO by age
		if info, e := s.dataService.Info(locator, s.readOnlyAge); e == nil && info.ReadOnly {
			return true
		}
	}
	return s.dataService.IsReadOnly(locator) // ro manually
}

func randToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("can't get random: %w", err)
	}
	s := sha1.New() //nolint:gosec // not used for security
	if _, err := s.Write(b); err != nil {
		return "", fmt.Errorf("can't write randoms to sha1: %w", err)
	}
	return fmt.Sprintf("%x", s.Sum(nil)), nil
}
