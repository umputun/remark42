package provider

import (
	"bytes"
	"crypto/sha1"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-pkgz/rest"
	"github.com/pkg/errors"

	"github.com/go-pkgz/auth/avatar"
	"github.com/go-pkgz/auth/logger"
	"github.com/go-pkgz/auth/token"
)

// VerifyHandler implements non-oauth2 provider authorizing users with some confirmation.
// can be email, IM or anything else implementing Sender interface
type VerifyHandler struct {
	logger.L
	ProviderName string
	TokenService VerifTokenService
	Issuer       string
	AvatarSaver  AvatarSaver
	Sender       Sender
	Template     string
	UseGravatar  bool
}

// Sender defines interface to send emails
type Sender interface {
	Send(address, text string) error
}

// SenderFunc type is an adapter to allow the use of ordinary functions as Sender.
type SenderFunc func(address, text string) error

// Send calls f(address,text) to implement Sender interface
func (f SenderFunc) Send(address, text string) error {
	return f(address, text)
}

// VerifTokenService defines interface accessing tokens
type VerifTokenService interface {
	Token(claims token.Claims) (string, error)
	Parse(tokenString string) (claims token.Claims, err error)
	IsExpired(claims token.Claims) bool
	Set(w http.ResponseWriter, claims token.Claims) (token.Claims, error)
	Reset(w http.ResponseWriter)
}

// Name of the handler
func (e VerifyHandler) Name() string { return e.ProviderName }

// LoginHandler gets name and address from query, makes confirmation token and sends it to user.
// In case if confirmation token presented in the query uses it to create auth token
func (e VerifyHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {

	// GET /login?site=site&user=name&address=someone@example.com
	tkn := r.URL.Query().Get("token")
	if tkn == "" { // no token, ask confirmation via email
		e.sendConfirmation(w, r)
		return
	}

	// confirmation token presented
	// GET /login?token=confirmation-jwt&sess=1
	confClaims, err := e.TokenService.Parse(tkn)
	if err != nil {
		rest.SendErrorJSON(w, r, e.L, http.StatusForbidden, err, "failed to verify confirmation token")
		return
	}

	if e.TokenService.IsExpired(confClaims) {
		rest.SendErrorJSON(w, r, e.L, http.StatusForbidden, errors.New("expired"), "failed to verify confirmation token")
		return
	}

	elems := strings.Split(confClaims.Handshake.ID, "::")
	if len(elems) != 2 {
		rest.SendErrorJSON(w, r, e.L, http.StatusBadRequest, errors.New(confClaims.Handshake.ID), "invalid handshake token")
		return
	}
	user, address := elems[0], elems[1]
	sessOnly := r.URL.Query().Get("sess") == "1"

	u := token.User{
		Name: user,
		ID:   e.ProviderName + "_" + token.HashID(sha1.New(), address),
	}
	// try to get gravatar for email
	if e.UseGravatar && strings.Contains(address, "@") { // TODO: better email check to avoid silly hits to gravatar api
		if picURL, e := avatar.GetGravatarURL(address); e == nil {
			u.Picture = picURL
		}
	}

	if u, err = setAvatar(e.AvatarSaver, u, &http.Client{Timeout: 5 * time.Second}); err != nil {
		rest.SendErrorJSON(w, r, e.L, http.StatusInternalServerError, err, "failed to save avatar to proxy")
		return
	}

	cid, err := randToken()
	if err != nil {
		rest.SendErrorJSON(w, r, e.L, http.StatusInternalServerError, err, "can't make token id")
		return
	}

	claims := token.Claims{
		User: &u,
		StandardClaims: jwt.StandardClaims{
			Id:       cid,
			Issuer:   e.Issuer,
			Audience: confClaims.Audience,
		},
		SessionOnly: sessOnly,
	}

	if _, err = e.TokenService.Set(w, claims); err != nil {
		rest.SendErrorJSON(w, r, e.L, http.StatusInternalServerError, err, "failed to set token")
		return
	}
	if confClaims.Handshake != nil && confClaims.Handshake.From != "" {
		http.Redirect(w, r, confClaims.Handshake.From, http.StatusTemporaryRedirect)
		return
	}
	rest.RenderJSON(w, claims.User)
}

// GET /login?site=site&user=name&address=someone@example.com
func (e VerifyHandler) sendConfirmation(w http.ResponseWriter, r *http.Request) {
	user, address := r.URL.Query().Get("user"), r.URL.Query().Get("address")
	if user == "" || address == "" {
		rest.SendErrorJSON(w, r, e.L, http.StatusBadRequest, errors.New("wrong request"), "can't get user and address")
		return
	}
	claims := token.Claims{
		Handshake: &token.Handshake{
			State: "",
			ID:    user + "::" + address,
		},
		SessionOnly: r.URL.Query().Get("session") != "" && r.URL.Query().Get("session") != "0",
		StandardClaims: jwt.StandardClaims{
			Audience:  r.URL.Query().Get("site"),
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
			NotBefore: time.Now().Add(-1 * time.Minute).Unix(),
			Issuer:    e.Issuer,
		},
	}

	tkn, err := e.TokenService.Token(claims)
	if err != nil {
		rest.SendErrorJSON(w, r, e.L, http.StatusForbidden, err, "failed to make login token")
		return
	}

	tmpl := msgTemplate
	if e.Template != "" {
		tmpl = e.Template
	}
	emailTmpl, err := template.New("confirm").Parse(tmpl)
	if err != nil {
		rest.SendErrorJSON(w, r, e.L, http.StatusInternalServerError, err, "can't parse confirmation template")
		return
	}

	tmplData := struct {
		User    string
		Address string
		Token   string
		Site    string
	}{
		User:    user,
		Address: address,
		Token:   tkn,
		Site:    r.URL.Query().Get("site"),
	}
	buf := bytes.Buffer{}
	if err = emailTmpl.Execute(&buf, tmplData); err != nil {
		rest.SendErrorJSON(w, r, e.L, http.StatusInternalServerError, err, "can't execute confirmation template")
		return
	}

	if err := e.Sender.Send(address, buf.String()); err != nil {
		rest.SendErrorJSON(w, r, e.L, http.StatusInternalServerError, err, "failed to send confirmation")
		return
	}

	rest.RenderJSON(w, rest.JSON{"user": user, "address": address})
}

// AuthHandler doesn't do anything for direct login as it has no callbacks
func (e VerifyHandler) AuthHandler(w http.ResponseWriter, r *http.Request) {}

// LogoutHandler - GET /logout
func (e VerifyHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	e.TokenService.Reset(w)
}

var msgTemplate = `
Confirmation for {{.User}} {{.Address}}, site {{.Site}}

Token: {{.Token}}
`
