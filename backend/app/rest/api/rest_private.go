package api

import (
	"compress/gzip"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec //not used for security
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-pkgz/auth/v2"
	"github.com/go-pkgz/auth/v2/token"
	cache "github.com/go-pkgz/lcw/v2"
	log "github.com/go-pkgz/lgr"
	R "github.com/go-pkgz/rest"
	"github.com/golang-jwt/jwt/v5"

	"github.com/umputun/remark42/backend/app/notify"
	"github.com/umputun/remark42/backend/app/rest"
	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/engine"
	"github.com/umputun/remark42/backend/app/store/image"
	"github.com/umputun/remark42/backend/app/store/service"
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
	disableFancyTextFormatting bool // disables SmartyPants in the comment text rendering of the posted comments
	premoderator               Premoderator
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
	ReplyDepth(locator store.Locator, comment store.Comment, limit int) (int, error)
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
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, hardBodyLimit)).Decode(&comment); err != nil {
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
			rest.SendErrorJSONAs(w, r, http.StatusBadRequest, err, fmt.Errorf("image not found"), "can't load picture from the comment", rest.ErrImgNotFound)
			return
		}
	}
	rest.HTMLResponse(w, http.StatusOK, comment.Text)
}

// maxReplyDepth is how many replies can stand one under another below a top-level comment. The tree of
// a post goes out as nested JSON, a level of nesting for every reply in such a chain, and the encoder
// refuses nesting past a limit of its own; discussions stay far below either number.
const maxReplyDepth = 500

// POST /comment - adds comment, resets all immutable fields
func (s *private) createCommentCtrl(w http.ResponseWriter, r *http.Request) {
	comment := store.Comment{}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, hardBodyLimit)).Decode(&comment); err != nil {
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
	comment.User.IP = extractIP(r.RemoteAddr)

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
			rest.SendErrorJSONAs(w, r, http.StatusBadRequest, err, fmt.Errorf("image not found"), "can't load picture from the comment", rest.ErrImgNotFound)
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

	// a reply must answer a comment on the same post: the comments tree is built by following parents
	// from the top-level comments, so a reply to anything else would be stored and counted but never shown.
	// A deleted parent stays in the store and can still be answered.
	if comment.ParentID != "" {
		parent, err := s.dataService.Get(comment.Locator, comment.ParentID, store.User{})
		if err != nil {
			rest.SendErrorJSONAs(w, r, http.StatusBadRequest, err, fmt.Errorf("parent comment not found"),
				"can't find parent comment", rest.ErrCommentNotFound)
			return
		}
		depth, err := s.dataService.ReplyDepth(comment.Locator, parent, maxReplyDepth)
		if err != nil {
			rest.SendErrorJSONAs(w, r, http.StatusInternalServerError, err, fmt.Errorf("can't read the thread"),
				"can't check reply depth", rest.ErrInternal)
			return
		}
		if depth >= maxReplyDepth {
			rest.SendErrorJSON(w, r, http.StatusBadRequest,
				fmt.Errorf("a reply can be at most %d replies below a top-level comment", maxReplyDepth),
				"invalid comment", rest.ErrCommentValidation)
			return
		}
	}

	comment, err := s.premoderator.premoderateComment(comment, user)

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "error handling premoderation", rest.ErrPremoderationFailure)
		return
	}

	id, err := s.dataService.Create(comment)
	if errors.Is(err, service.ErrUserBlocked) { // the author got blocked during the title fetch in Create
		rest.SendErrorJSON(w, r, http.StatusForbidden, fmt.Errorf("rejected"), "user blocked", rest.ErrUserBlocked)
		return
	}
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

	_ = R.EncodeJSON(w, http.StatusCreated, &finalComment)
}

// todo should this be part of the admin routes??

// PUT /comment/{id}?site=siteID&url=post-url - update comment
func (s *private) updateCommentCtrl(w http.ResponseWriter, r *http.Request) {
	edit := struct {
		Text    string
		Summary string
		Delete  bool
	}{}

	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, hardBodyLimit)).Decode(&edit); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't read comment details from body", rest.ErrDecode)
		return
	}

	user := rest.MustGetUserInfo(r)
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	id := r.PathValue("id")

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

	// a token issued before the block is still accepted until it is refreshed, so check the block here,
	// as creating a comment and voting do. Deleting one's own comment stays allowed.
	if !edit.Delete && s.dataService.IsBlocked(locator.SiteID, user.ID) {
		rest.SendErrorJSON(w, r, http.StatusForbidden, fmt.Errorf("rejected"), "user blocked", rest.ErrUserBlocked)
		return
	}

	// a closed post takes no new text, as it takes no new comments. An admin's edit keeps working,
	// and deleting one's own comment stays allowed.
	if !edit.Delete && !user.Admin && s.isReadOnly(locator) {
		rest.SendErrorJSON(w, r, http.StatusForbidden, fmt.Errorf("rejected"), "old post, read-only", rest.ErrReadOnly)
		return
	}

	// the new text is held to the rules of a new comment; a delete carries no text and is not checked
	if !edit.Delete {
		edited := currComment
		edited.Orig = edit.Text
		edited.User = user // the stored name was sanitized and can be empty where the signed-in one is not
		if err = s.dataService.ValidateComment(&edited); err != nil {
			rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "invalid comment", rest.ErrCommentValidation)
			return
		}
	}

	editReq := service.EditRequest{Orig: edit.Text, Summary: edit.Summary, Delete: edit.Delete, Admin: user.Admin}
	if !edit.Delete { // a delete stores no text, so whatever text it carries is not rendered
		editReq.Text = s.commentFormatter.FormatText(edit.Text, s.disableFancyTextFormatting)
	}

	_, err = s.dataService.EditComment(locator, id, editReq)
	if errors.Is(err, service.ErrRestrictedWordsFound) {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "invalid comment", rest.ErrCommentRestrictWords)
		return
	}

	if err != nil {
		code := parseError(err, rest.ErrCommentRejected)
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't update comment", code)
		return
	}

	s.cache.Flush(cache.Flusher(locator.SiteID).Scopes(locator.SiteID, locator.URL, lastCommentsScope, user.ID))

	// answer with what a read returns: the record EditComment hands back is the stored one
	res, err := s.dataService.Get(locator, id, rest.GetUserOrEmpty(r))
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't load updated comment", rest.ErrInternal)
		return
	}
	R.RenderJSON(w, res)
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
		if email != "" {
			user.EmailSubscription = true
		}
	}

	R.RenderJSON(w, user)
}

// PUT /vote/{id}?site=siteID&url=post-url&vote=1 - vote for/against comment
func (s *private) voteCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	id := r.PathValue("id")
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
		UserIP:    extractIP(r.RemoteAddr),
		Val:       vote,
	}
	comment, err := s.dataService.Vote(req)
	if err != nil {
		code := parseError(err, rest.ErrVoteRejected)
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't vote for comment", code)
		return
	}
	s.cache.Flush(cache.Flusher(locator.SiteID).Scopes(locator.URL, comment.User.ID, lastVotesScope))
	R.RenderJSON(w, R.JSON{"id": comment.ID, "score": comment.Score})
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

	R.RenderJSON(w, R.JSON{"user": user, "address": address})
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
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, hardBodyLimit)).Decode(&subscribe); err != nil {
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
	// the widget sends the field as it was typed
	subscribe.Address = strings.TrimSpace(subscribe.Address)
	// "?" and "#" can be part of an address, but the notification mails take their recipient as a
	// mailto URL, where either of the two ends it, and are not sent to such an address
	if !isPlainEmail(subscribe.Address) || strings.ContainsAny(subscribe.Address, "?#") {
		rest.SendErrorJSON(w, r, http.StatusBadRequest,
			fmt.Errorf("not an email address"), "address parameter has to be an email address and nothing else", rest.ErrInternal)
		return
	}
	subscribe.Site = verifiedSite(r, subscribe.Site)
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
		Handshake:    &token.Handshake{ID: user.ID + "::" + subscribe.Address},
		AuthProvider: &token.AuthProvider{Name: emailSubscribeProvider},
		Audience:     jwt.ClaimStrings{subscribe.Site},
		ExpiresAt:    jwt.NewNumericDate(time.Now().Add(30 * time.Minute)),
		NotBefore:    jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
		Issuer:       "remark42",
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

	R.RenderJSON(w, R.JSON{"user": user, "address": subscribe.Address, "updated": false})
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
		// generate and send token
		rnd := make([]byte, 20)
		_, _ = rand.Read(rnd) // never fails, crashes the program instead
		tkn := hex.EncodeToString(rnd)
		expires := time.Now().Add(10 * time.Minute)

		s.telegramService.AddToken(tkn, user.ID, siteID, expires)

		R.RenderJSON(w, R.JSON{"token": tkn, "bot": s.telegramService.GetBotUsername()})

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
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't set telegram for user", rest.ErrInternal)
		return
	}

	R.RenderJSON(w, R.JSON{"updated": true, "address": val})
}

// setConfirmedEmailCtrl uses provided token parameter (generated by sendEmailConfirmationCtrl) to set email and add it to user token
// POST /email/confirm with site and token in json body
func (s *private) setConfirmedEmailCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)

	confirm := struct {
		Site  string
		Token string
	}{}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, hardBodyLimit)).Decode(&confirm); err != nil {
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
	claims, userID, address, herr := s.parseHandshake(confirm.Token, emailSubscribeProvider)
	if herr != nil {
		rest.SendErrorJSON(w, r, herr.status, herr.err, herr.details, rest.ErrInternal)
		return
	}
	siteID := verifiedSite(r, confirm.Site)
	if site := tokenSite(claims); site == "" || site != siteID {
		rest.SendErrorJSON(w, r, http.StatusForbidden, fmt.Errorf("token issued for another site"),
			"can't use provided token for this site", rest.ErrInternal)
		return
	}
	if userID != user.ID {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, fmt.Errorf("handshake for %s, request from %s", userID, user.ID),
			"invalid handshake token", rest.ErrInternal)
		return
	}
	s.setEmail(w, r, user.ID, siteID, address)
}

// emailSubscribeProvider is the provider name in a token made by sendEmailConfirmationCtrl. Three tokens
// carry a user::address handshake signed with the same key: this one, which expires in 30 minutes and proves
// the reader holds the address now; the unsubscribe link in every reply email, which has no provider and
// lasts a century; and the email login confirmation, whose user part the requester chooses and whose
// provider is the login's. parseHandshake accepts each token only where it is meant to be used.
const emailSubscribeProvider = "email-subscribe"

// emailSubscribeCtrl confirms an email subscription from the link in the confirmation email, with no
// session: the token proves the reader holds the address, and it names the user who asked for it and the
// site. GET only shows the page with the confirm button, as mail scanners fetch links on their own; POST
// from that button stores the address.
// GET/POST /email/subscribe.html?tkn=jwt
func (s *private) emailSubscribeCtrl(w http.ResponseWriter, r *http.Request) {
	// the global policy forbids forms and scripts, and this page is one button posting back to itself,
	// with a script that posts it in place so the tab can close itself afterwards. Nothing may frame
	// it, and the token in its address must not leave in a Referer
	nonce := rand.Text()
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; form-action 'self'; "+
		"script-src 'nonce-"+nonce+"'; connect-src 'self'; style-src 'unsafe-inline'; frame-ancestors 'none';")
	w.Header().Set("Referrer-Policy", "no-referrer")

	tkn := r.URL.Query().Get("tkn")
	if tkn == "" {
		rest.SendErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("missing parameter"), "token parameter is required", rest.ErrInternal)
		return
	}
	claims, userID, address, herr := s.parseHandshake(tkn, emailSubscribeProvider)
	if herr != nil {
		details := herr.details
		if errors.Is(herr.err, errHandshakeExpired) {
			details = "the confirmation link has expired, subscribe again from the comment form"
		}
		rest.SendErrorHTML(w, r, herr.status, herr.err, details, rest.ErrInternal)
		return
	}
	siteID := tokenSite(claims)
	if siteID == "" {
		rest.SendErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("no site in token"),
			"invalid confirmation token", rest.ErrInternal)
		return
	}
	// the session check refuses a blocked user on the paste path, and there is no session here
	if s.dataService.IsBlocked(siteID, userID) {
		rest.SendErrorHTML(w, r, http.StatusForbidden, fmt.Errorf("user %s is blocked", userID), "user is blocked", rest.ErrInternal)
		return
	}
	// the link works for whoever holds it until it expires, so an older link, such as one sent to a
	// mistyped address, must not replace an address confirmed since. Unsubscribing removes the stored
	// address, after which the link works
	current, err := s.dataService.GetUserEmail(siteID, userID)
	if err != nil {
		rest.SendErrorHTML(w, r, http.StatusInternalServerError, err, "can't read email for user", rest.ErrInternal)
		return
	}
	if current != "" && current != address {
		rest.SendErrorHTML(w, r, http.StatusConflict, fmt.Errorf("another address is confirmed for user %s", userID),
			"a different email address is already confirmed, unsubscribe in the comment form and open this link again to use this one",
			rest.ErrInternal)
		return
	}

	confirmed := false
	if r.Method == http.MethodPost {
		log.Printf("[DEBUG] set email for user %s from the confirmation link", userID)
		if _, err = s.dataService.SetUserEmail(siteID, userID, address); err != nil {
			rest.SendErrorHTML(w, r, http.StatusBadRequest, err, "can't set email for user", rest.ErrInternal)
			return
		}
		confirmed = true
	}

	page, err := rest.RenderTemplate("subscribe", "email_subscribe.html.tmpl", struct {
		Site      string
		Confirmed bool
		Nonce     string
	}{Site: siteID, Confirmed: confirmed, Nonce: nonce})
	if err != nil {
		log.Printf("[WARN] can't make subscribe page, %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	rest.HTMLResponse(w, http.StatusOK, page)
}

// handshakeError is a failure to read the handshake from a confirmation token,
// with the response status and details to report it with
type handshakeError struct {
	status  int
	err     error
	details string
}

// errHandshakeExpired is the error parseHandshake reports for an expired token
var errHandshakeExpired = errors.New("expired")

// parseHandshake verifies a confirmation token made by sendEmailConfirmationCtrl or for the email unsubscribe link,
// and returns its claims with the user ID and address from its handshake, which is user.ID + "::" + address.
// The token must have been made by provider: emailSubscribeProvider for a confirmation, empty for an unsubscribe
// link. A token without a handshake, such as a login token, or made for another use is rejected as invalid.
func (s *private) parseHandshake(tkn, provider string) (claims token.Claims, userID, address string, herr *handshakeError) {
	claims, err := s.authenticator.TokenService().Parse(tkn)
	if err != nil {
		return token.Claims{}, "", "", &handshakeError{status: http.StatusForbidden, err: err, details: "failed to verify confirmation token"}
	}
	if s.authenticator.TokenService().IsExpired(claims) {
		return token.Claims{}, "", "", &handshakeError{status: http.StatusForbidden, err: errHandshakeExpired, details: "failed to verify confirmation token"}
	}
	if claims.Handshake == nil {
		return token.Claims{}, "", "", &handshakeError{status: http.StatusBadRequest, err: fmt.Errorf("no handshake in token"), details: "invalid handshake token"}
	}
	elems := strings.Split(claims.Handshake.ID, "::")
	if len(elems) != 2 || elems[0] == "" || elems[1] == "" {
		// the handshake carries the address, so it is not echoed into the error
		return token.Claims{}, "", "", &handshakeError{status: http.StatusBadRequest, err: fmt.Errorf("malformed handshake"), details: "invalid handshake token"}
	}
	tokenProvider := ""
	if claims.AuthProvider != nil {
		tokenProvider = claims.AuthProvider.Name
	}
	if tokenProvider != provider {
		return token.Claims{}, "", "", &handshakeError{status: http.StatusBadRequest, err: fmt.Errorf("token made for another use"), details: "invalid handshake token"}
	}
	return claims, elems[0], elems[1], nil
}

// tokenSite returns the site a confirmation or unsubscribe token was issued for, or an empty string when
// the token names no site or several. Users have the same ID on every site of a server, so a token from
// one site must not act on another
func tokenSite(claims token.Claims) string {
	if len(claims.Audience) != 1 {
		return ""
	}
	return claims.Audience[0]
}

// verifiedSite returns the site from the query, which matchSiteID has checked against the user's site.
// The site in the request body is used only when the query has none, which matchSiteID allows only for
// the basic-auth admin.
func verifiedSite(r *http.Request, bodySite string) string {
	if site := r.URL.Query().Get("site"); site != "" {
		return site
	}
	return bodySite
}

func (s *private) setEmail(w http.ResponseWriter, r *http.Request, userID, siteID, address string) {
	log.Printf("[DEBUG] set email for user %s", userID)

	val, err := s.dataService.SetUserEmail(siteID, userID, address)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't set email for user", rest.ErrInternal)
		return
	}

	// update User.Email field of the session token. The admin password signs in without one, and a token
	// without a user, such as a handshake token sent along with that password, has no email to update
	claims, ok := s.sessionClaims(r)
	if ok && claims.User != nil {
		claims.User.Email = address
		if err = s.setTokenClaims(w, claims); err != nil {
			rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to set token", rest.ErrInternal)
			return
		}
	}
	R.RenderJSON(w, R.JSON{"updated": true, "address": val})
}

// POST/GET /email/unsubscribe.html?site=siteID&tkn=jwt - unsubscribe the user in token from email notifications
func (s *private) emailUnsubscribeCtrl(w http.ResponseWriter, r *http.Request) {
	tkn := r.URL.Query().Get("tkn")
	if tkn == "" {
		rest.SendErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("missing parameter"), "token parameter is required", rest.ErrInternal)
		return
	}
	siteID := r.URL.Query().Get("site")

	claims, userID, address, herr := s.parseHandshake(tkn, "")
	if herr != nil {
		rest.SendErrorHTML(w, r, herr.status, herr.err, herr.details, rest.ErrInternal)
		return
	}
	// the page has no session to check the site against, and every link carries the site of its token
	if site := tokenSite(claims); site == "" || site != siteID {
		rest.SendErrorHTML(w, r, http.StatusForbidden, fmt.Errorf("token issued for another site"),
			"can't use provided token for this site", rest.ErrInternal)
		return
	}

	existingAddress, err := s.dataService.GetUserEmail(siteID, userID)
	if err != nil {
		// we don't care as much if we can't retrieve the current value of that field for the user,
		// as it's only used to check if we're trying to set to the same value it's already set to
		log.Printf("[WARN] can't read email for %s, %v", userID, err)
	}
	// one answer whether the user has no address or another one: a link mailed to the author of a comment
	// further up a thread named the author of the comment replied to until the recipient was named in it,
	// and such a link must not tell its holder whether that other user is subscribed
	if address != existingAddress {
		rest.SendErrorHTML(w, r, http.StatusConflict, fmt.Errorf("user is not subscribed with this address"),
			"this address is not subscribed", rest.ErrInternal)
		return
	}

	log.Printf("[DEBUG] unsubscribe user %s", userID)

	if err = s.dataService.DeleteUserDetail(siteID, userID, engine.UserEmail); err != nil {
		rest.SendErrorHTML(w, r, http.StatusBadRequest, err, "can't delete email for user", rest.ErrInternal)
		return
	}
	// clean User.Email from the token, if user has the token
	session, _, err := s.authenticator.TokenService().Get(r)
	if err != nil {
		log.Printf("[DEBUG] unsubscribed user doesn't have valid JWT token to update %s, %v", userID, err)
	}
	if session.User != nil && session.User.Email != "" {
		session.User.Email = ""
		if _, err = s.authenticator.TokenService().Set(w, session); err != nil {
			rest.SendErrorHTML(w, r, http.StatusInternalServerError, err, "failed to set token", rest.ErrInternal)
			return
		}
	}

	page, err := rest.RenderTemplate("unsubscribe", "email_unsubscribe.html.tmpl", nil)
	if err != nil {
		log.Printf("[WARN] can't make unsubscribe page, %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	rest.HTMLResponse(w, http.StatusOK, page)
}

// DELETE /email?site=siteID - removes user's email
func (s *private) deleteEmailCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)
	siteID := r.URL.Query().Get("site")
	log.Printf("[DEBUG] remove email for user %s", user.ID)

	if err := s.dataService.DeleteUserDetail(siteID, user.ID, engine.UserEmail); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't delete email for user", rest.ErrInternal)
		return
	}
	// clean User.Email from the session token, if the request has one
	claims, ok := s.sessionClaims(r)
	if ok && claims.User != nil && claims.User.Email != "" {
		claims.User.Email = ""
		if err := s.setTokenClaims(w, claims); err != nil {
			rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to set token", rest.ErrInternal)
			return
		}
	}
	R.RenderJSON(w, R.JSON{"deleted": true})
}

// sessionClaims returns the claims of the token the request carries, and false when it carries none. The auth
// middleware also accepts the admin password, which needs no token, and the change the caller made is already
// stored by then, so the caller has only the token left to update and skips it
func (s *private) sessionClaims(r *http.Request) (token.Claims, bool) {
	claims, _, err := s.authenticator.TokenService().Get(r)
	if err != nil {
		log.Printf("[DEBUG] no session token to update: %v", err)
		return token.Claims{}, false
	}
	return claims, true
}

// setTokenClaims writes changed claims back to the client, for a handler behind the auth middleware.
// The claims are read off the request, and the token there may be an expired one the middleware renewed
// a moment ago. Set keeps an ExpiresAt it is given, so writing such claims as they are would replace the
// renewed token, in the cookie and in the X-JWT header, with one that has expired already. An expired
// token therefore gets a new lifetime here, as the middleware gave it; one still valid keeps its own.
func (s *private) setTokenClaims(w http.ResponseWriter, claims token.Claims) error {
	if s.authenticator.TokenService().IsExpired(claims) {
		claims.ExpiresAt = nil
	}
	_, err := s.authenticator.TokenService().Set(w, claims)
	return err
}

// DELETE /telegram?site=siteID - removes user's telegram
func (s *private) deleteTelegramCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)
	siteID := r.URL.Query().Get("site")
	log.Printf("[DEBUG] remove telegram for user %s", user.ID)

	if err := s.dataService.DeleteUserDetail(siteID, user.ID, engine.UserTelegram); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't delete telegram for user", rest.ErrInternal)
		return
	}
	R.RenderJSON(w, R.JSON{"deleted": true})
}

const (
	// userDataPageSize is how many comments userAllDataCtrl requests from the store at a time
	userDataPageSize = 100
	// maxUserDataComments bounds the user data export, so a store that keeps returning full pages
	// (one ignoring skip, say) cannot make the export run forever; reaching it is logged
	maxUserDataComments = 10000
)

// GET /userdata?site=siteID - exports all data about the user as a json with user info, the stored
// notification details (email and telegram, if any) and list of all comments
func (s *private) userAllDataCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	user := rest.MustGetUserInfo(r)
	userB, err := json.Marshal(&user)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't marshal user info", rest.ErrInternal)
		return
	}

	// the first page is read before anything is written, so a store error still gets a normal JSON error
	page, err := s.dataService.User(siteID, user.ID, userDataPageSize, 0, rest.GetUserOrEmpty(r))
	if err != nil {
		if !isNoUserComments(err) {
			rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't get user comments", rest.ErrInternal)
			return
		}
		page = nil // a user without comments still gets an export with their info
	}

	// details are read before anything is written as well, for the same reason
	detailsB, err := s.userDetailsJSON(siteID, user.ID)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't get user details", rest.ErrInternal)
		return
	}

	exportFile := fmt.Sprintf("%s-%s-%s.json.gz", siteID, user.ID, time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", "attachment;filename="+exportFile)

	// stream page by page, so memory use stays at one page whatever the user's history.
	// After the first byte an error can only be logged; the gzip stream is then left unterminated,
	// so the client sees a broken download instead of a complete-looking but truncated export.
	gzWriter := gzip.NewWriter(w)
	write := func(b []byte) bool {
		if _, e := gzWriter.Write(b); e != nil {
			log.Printf("[WARN] can't write user data export for %s, %v", user.ID, e)
			return false
		}
		return true
	}
	if !write([]byte(`{"info": `)) || !write(userB) {
		return
	}
	if detailsB != nil && (!write([]byte(`, "details": `)) || !write(detailsB)) {
		return
	}
	if !write([]byte(`, "comments":[`)) {
		return
	}
	for exported, skip := 0, 0; len(page) > 0; {
		pageB, e := json.Marshal(page)
		if e != nil {
			log.Printf("[WARN] can't marshal user data export for %s, %v", user.ID, e)
			return
		}
		pageB = pageB[1 : len(pageB)-1] // array elements only, the brackets are written once around all pages
		if (exported > 0 && !write([]byte(","))) || !write(pageB) {
			return
		}
		exported += len(page)
		if len(page) < userDataPageSize {
			break
		}
		if exported >= maxUserDataComments {
			log.Printf("[WARN] user data export for %s stopped at %d comments", user.ID, exported)
			break
		}
		skip += userDataPageSize
		if page, e = s.dataService.User(siteID, user.ID, userDataPageSize, skip, rest.GetUserOrEmpty(r)); e != nil {
			log.Printf("[WARN] can't get user comments for data export of %s, %v", user.ID, e)
			return
		}
	}
	if !write([]byte(`]}`)) {
		return
	}
	if err = gzWriter.Close(); err != nil {
		log.Printf("[WARN] can't close gzip writer, %v", err)
	}
}

// userDetailsJSON returns the notification details stored for the user, email and telegram, as a json object.
// The result is nil for a user with none of them, so such an export has no "details" key.
func (s *private) userDetailsJSON(siteID, userID string) ([]byte, error) {
	var details struct {
		Email    string `json:"email,omitempty"`
		Telegram string `json:"telegram,omitempty"`
	}
	var err error
	if details.Email, err = s.dataService.GetUserEmail(siteID, userID); err != nil {
		return nil, fmt.Errorf("can't read email: %w", err)
	}
	if details.Telegram, err = s.dataService.GetUserTelegram(siteID, userID); err != nil {
		return nil, fmt.Errorf("can't read telegram: %w", err)
	}
	if details.Email == "" && details.Telegram == "" {
		return nil, nil
	}
	return json.Marshal(details)
}

// isNoUserComments reports whether err means the user has no comments. The bolt store returns the
// engine.ErrNoUserComments sentinel; the rpc store loses typed errors over jrpc, so its message is matched too.
func isNoUserComments(err error) bool {
	return errors.Is(err, engine.ErrNoUserComments) || strings.Contains(err.Error(), engine.ErrNoUserComments.Error())
}

// POST /deleteme?site=siteID - deletes nothing itself: makes a deletion request token for the user, valid for
// three months and not usable for login, and returns it with the link to the page where an admin carries it out
func (s *private) deleteMeCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)
	siteID := r.URL.Query().Get("site")

	claims := token.Claims{
		Audience:  jwt.ClaimStrings{siteID},
		Issuer:    "remark42",
		ExpiresAt: jwt.NewNumericDate(time.Now().AddDate(0, 3, 0)),
		NotBefore: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
		User: &token.User{
			ID:      user.ID,
			Name:    user.Name,
			Picture: user.Picture, // carried so the avatar can be removed when the request is processed
			Attributes: map[string]any{
				"delete_me": true, // prevents this token from being used for login
			},
		},
	}

	tokenStr, err := s.authenticator.TokenService().Token(claims)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't make token", rest.ErrInternal)
		return
	}

	// the page signs the admin's requests for the site it is given, so the link names the user's site
	link := fmt.Sprintf("%s/web/deleteme.html?site_id=%s&token=%s", s.remarkURL, url.QueryEscape(siteID), tokenStr)
	R.RenderJSON(w, R.JSON{"site": siteID, "user_id": user.ID, "token": tokenStr, "link": link})
}

// POST /picture - saves the image from the "file" field of a multipart form and returns its id
func (s *private) savePictureCtrl(w http.ResponseWriter, r *http.Request) {
	user := rest.MustGetUserInfo(r)

	r.Body = http.MaxBytesReader(w, r.Body, 32*1024*1024) // hard cap on upload to prevent memory exhaustion
	// gosec G120: r.Body is already bounded by MaxBytesReader on the line above (32 MB),
	// so ParseMultipartForm cannot read more than that regardless of the in-memory threshold.
	// The 5 MB argument is the soft threshold above which the form is spilled to disk.
	if err := r.ParseMultipartForm(5 * 1024 * 1024); err != nil { //nolint:gosec // bounded by MaxBytesReader above
		rest.SendErrorJSONAs(w, r, http.StatusInternalServerError, err, withoutPath(err, fmt.Errorf("upload storage error")),
			"can't parse multipart form", rest.ErrDecode)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		// a part over the in-memory limit is read back from a temp file, whose error names its path
		rest.SendErrorJSONAs(w, r, http.StatusInternalServerError, err, withoutPath(err, fmt.Errorf("upload storage error")),
			"can't get image file from the request", rest.ErrInternal)
		return
	}
	defer func() { _ = file.Close() }()

	id, err := s.imageService.Save(user.ID, file)
	if err != nil {
		rest.SendErrorJSONAs(w, r, http.StatusBadRequest, err, imageRejection(err), "can't save image", rest.ErrInternal)
		return
	}

	R.RenderJSON(w, R.JSON{"id": id})
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

// extractIP returns the IP portion of the remote address, handling both IPv4 and IPv6 formats.
// supports "ip:port", "[ip]:port", and bare "ip" formats.
func extractIP(remoteAddr string) string {
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr // already a bare IP (no port)
	}
	return ip
}

// imageRejection returns err when the image service rejected the upload itself, as too large, not an image or
// malformed, since API users tell those cases apart by the text. Any other error comes from the image store and
// can name a file on the server, so a fixed error is returned in its place
func imageRejection(err error) error {
	for _, prefix := range []string{"can't load image: file is too large", "can't load image: file format not allowed",
		"image rejected: "} {
		if strings.HasPrefix(err.Error(), prefix) {
			return err
		}
	}
	return fmt.Errorf("image store error")
}

// withoutPath returns err, or replacement when err carries a file system path, as the errors of temp files do
func withoutPath(err, replacement error) error {
	if _, ok := errors.AsType[*fs.PathError](err); ok {
		return replacement
	}
	return err
}
