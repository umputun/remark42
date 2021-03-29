package provider

import (
	"crypto/sha1"
	"encoding/json"
	"mime"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-pkgz/rest"
	"github.com/pkg/errors"

	"github.com/go-pkgz/auth/logger"
	"github.com/go-pkgz/auth/token"
)

const (
	// MaxHTTPBodySize defines max http body size
	MaxHTTPBodySize = 1024 * 1024
)

// DirectHandler implements non-oauth2 provider authorizing user in traditional way with storage
// with users and hashes
type DirectHandler struct {
	logger.L
	CredChecker  CredChecker
	ProviderName string
	TokenService TokenService
	Issuer       string
	AvatarSaver  AvatarSaver
}

// CredChecker defines interface to check credentials
type CredChecker interface {
	Check(user, password string) (ok bool, err error)
}

// CredCheckerFunc type is an adapter to allow the use of ordinary functions as CredsChecker.
type CredCheckerFunc func(user, password string) (ok bool, err error)

// Check calls f(user,passwd)
func (f CredCheckerFunc) Check(user, password string) (ok bool, err error) {
	return f(user, password)
}

// credentials holds user credentials
type credentials struct {
	User     string `json:"user"`
	Password string `json:"passwd"`
	Audience string `json:"aud"`
}

// Name of the handler
func (p DirectHandler) Name() string { return p.ProviderName }

// LoginHandler checks "user" and "passwd" against data store and makes jwt if all passed.
//
// GET /something?user=name&passwd=xyz&aud=bar&sess=[0|1]
//
// POST /something?sess[0|1]
// Accepts application/x-www-form-urlencoded or application/json encoded requests.
//
// application/x-www-form-urlencoded body example:
// user=name&passwd=xyz&aud=bar
//
// application/json body example:
// {
//   "user": "name",
//   "passwd": "xyz",
//   "aud": "bar",
// }
func (p DirectHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	creds, err := p.getCredentials(w, r)
	if err != nil {
		rest.SendErrorJSON(w, r, p.L, http.StatusBadRequest, err, "failed to parse credentials")
		return
	}
	sessOnly := r.URL.Query().Get("sess") == "1"
	if p.CredChecker == nil {
		rest.SendErrorJSON(w, r, p.L, http.StatusInternalServerError,
			errors.New("no credential checker"), "no credential checker")
		return
	}
	ok, err := p.CredChecker.Check(creds.User, creds.Password)
	if err != nil {
		rest.SendErrorJSON(w, r, p.L, http.StatusInternalServerError, err, "failed to check user credentials")
		return
	}
	if !ok {
		rest.SendErrorJSON(w, r, p.L, http.StatusForbidden, nil, "incorrect user or password")
		return
	}
	u := token.User{
		Name: creds.User,
		ID:   p.ProviderName + "_" + token.HashID(sha1.New(), creds.User),
	}
	u, err = setAvatar(p.AvatarSaver, u, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		rest.SendErrorJSON(w, r, p.L, http.StatusInternalServerError, err, "failed to save avatar to proxy")
		return
	}

	cid, err := randToken()
	if err != nil {
		rest.SendErrorJSON(w, r, p.L, http.StatusInternalServerError, err, "can't make token id")
		return
	}

	claims := token.Claims{
		User: &u,
		StandardClaims: jwt.StandardClaims{
			Id:       cid,
			Issuer:   p.Issuer,
			Audience: creds.Audience,
		},
		SessionOnly: sessOnly,
	}

	if _, err = p.TokenService.Set(w, claims); err != nil {
		rest.SendErrorJSON(w, r, p.L, http.StatusInternalServerError, err, "failed to set token")
		return
	}
	rest.RenderJSON(w, claims.User)
}

// getCredentials extracts user and password from request
func (p DirectHandler) getCredentials(w http.ResponseWriter, r *http.Request) (credentials, error) {

	// GET /something?user=name&passwd=xyz&aud=bar
	if r.Method == "GET" {
		return credentials{
			User:     r.URL.Query().Get("user"),
			Password: r.URL.Query().Get("passwd"),
			Audience: r.URL.Query().Get("aud"),
		}, nil
	}

	if r.Method != "POST" {
		return credentials{}, errors.Errorf("method %s not supported", r.Method)
	}

	if r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, MaxHTTPBodySize)
	}
	contentType := r.Header.Get("Content-Type")
	if contentType != "" {
		mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			return credentials{}, err
		}
		contentType = mt
	}

	// POST with json body
	if contentType == "application/json" {
		var creds credentials
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			return credentials{}, errors.Wrap(err, "failed to parse request body")
		}
		return creds, nil
	}

	// POST with form
	if err := r.ParseForm(); err != nil {
		return credentials{}, errors.Wrap(err, "failed to parse request")
	}

	return credentials{
		User:     r.Form.Get("user"),
		Password: r.Form.Get("passwd"),
		Audience: r.Form.Get("aud"),
	}, nil
}

// AuthHandler doesn't do anything for direct login as it has no callbacks
func (p DirectHandler) AuthHandler(w http.ResponseWriter, r *http.Request) {}

// LogoutHandler - GET /logout
func (p DirectHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	p.TokenService.Reset(w)
}
