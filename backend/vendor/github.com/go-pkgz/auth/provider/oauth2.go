package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-pkgz/rest"
	"golang.org/x/oauth2"

	"github.com/go-pkgz/auth/logger"
	"github.com/go-pkgz/auth/token"
)

// Oauth2Handler implements /login, /callback and /logout handlers from aouth2 flow
type Oauth2Handler struct {
	Params

	// all of these fields specific to particular oauth2 provider
	name        string
	redirectURL string
	infoURL     string
	endpoint    oauth2.Endpoint
	scopes      []string
	mapUser     func(userData, []byte) token.User // map info from InfoURL to User
	conf        oauth2.Config
}

// Params to make initialized and ready to use provider
type Params struct {
	logger.L
	URL         string
	JwtService  TokenService
	Cid         string
	Csecret     string
	Issuer      string
	AvatarSaver AvatarSaver
}

type userData map[string]interface{}

func (u userData) value(key string) string {
	// json.Unmarshal converts json "null" value to go's "nil", in this case return empty string
	if val, ok := u[key]; ok && val != nil {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// initOauth2Handler makes oauth2 handler for given provider
func initOauth2Handler(p Params, service Oauth2Handler) Oauth2Handler {
	if p.L == nil {
		p.L = logger.Func(func(fmt string, args ...interface{}) {})
	}
	p.Logf("[INFO] init oauth2 service %s", service.name)
	service.Params = p
	service.conf = oauth2.Config{
		ClientID:     service.Cid,
		ClientSecret: service.Csecret,
		RedirectURL:  service.redirectURL,
		Scopes:       service.scopes,
		Endpoint:     service.endpoint,
	}

	p.Logf("[DEBUG] created %s oauth2, id=%s, redir=%s, endpoint=%s",
		service.name, service.Cid, service.endpoint, service.redirectURL)
	return service
}

// Name returns provider name
func (p Oauth2Handler) Name() string { return p.name }

// LoginHandler - GET /login?from=redirect-back-url&site=siteID&session=1
func (p Oauth2Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {

	p.Logf("[DEBUG] login with %s", p.Name())
	// make state (random) and store in session
	state, err := randToken()
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to make oauth2 state")
		return
	}

	cid, err := randToken()
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to make claim's id")
		return
	}

	claims := token.Claims{
		Handshake: &token.Handshake{
			State: state,
			From:  r.URL.Query().Get("from"),
		},
		SessionOnly: r.URL.Query().Get("session") != "" && r.URL.Query().Get("session") != "0",
		StandardClaims: jwt.StandardClaims{
			Id:        cid,
			Audience:  r.URL.Query().Get("site"),
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
			NotBefore: time.Now().Add(-1 * time.Minute).Unix(),
		},
	}

	if err := p.JwtService.Set(w, claims); err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to set token")
		return
	}

	// return login url
	loginURL := p.conf.AuthCodeURL(state)
	p.Logf("[DEBUG] login url %s, claims=%+v", loginURL, claims)

	http.Redirect(w, r, loginURL, http.StatusFound)
}

// AuthHandler fills user info and redirects to "from" url. This is callback url redirected locally by browser
// GET /callback
func (p Oauth2Handler) AuthHandler(w http.ResponseWriter, r *http.Request) {
	oauthClaims, _, err := p.JwtService.Get(r)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to get token")
		return
	}

	if oauthClaims.Handshake == nil {
		rest.SendErrorJSON(w, r, http.StatusForbidden, nil, "invalid handshake token")
		return
	}

	retrievedState := oauthClaims.Handshake.State
	if retrievedState == "" || retrievedState != r.URL.Query().Get("state") {
		rest.SendErrorJSON(w, r, http.StatusForbidden, nil, "unexpected state")
		return
	}

	p.Logf("[DEBUG] token with state %s", retrievedState)
	tok, err := p.conf.Exchange(context.Background(), r.URL.Query().Get("code"))
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "exchange failed")
		return
	}

	client := p.conf.Client(context.Background(), tok)
	uinfo, err := client.Get(p.infoURL)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusServiceUnavailable, err, "failed to get client info")
		return
	}

	defer func() {
		if e := uinfo.Body.Close(); e != nil {
			p.Logf("[WARN] failed to close response body, %s", e)
		}
	}()

	data, err := ioutil.ReadAll(uinfo.Body)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to read user info")
		return
	}

	jData := map[string]interface{}{}
	if e := json.Unmarshal(data, &jData); e != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to unmarshal user info")
		return
	}
	p.Logf("[DEBUG] got raw user info %+v", jData)

	u := p.mapUser(jData, data)
	u, err = setAvatar(p.AvatarSaver, u)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to save avatar to proxy")
		return
	}

	cid, err := randToken()
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to make claim's id")
		return
	}
	claims := token.Claims{
		User: &u,
		StandardClaims: jwt.StandardClaims{
			Issuer:   p.Issuer,
			Id:       cid,
			Audience: oauthClaims.Audience,
		},
		SessionOnly: oauthClaims.SessionOnly,
	}

	if err = p.JwtService.Set(w, claims); err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to set token")
		return
	}

	p.Logf("[DEBUG] user info %+v", u)

	// redirect to back url if presented in login query params
	if oauthClaims.Handshake != nil && oauthClaims.Handshake.From != "" {
		http.Redirect(w, r, oauthClaims.Handshake.From, http.StatusTemporaryRedirect)
		return
	}
	rest.RenderJSON(w, r, &u)
}

// LogoutHandler - GET /logout
func (p Oauth2Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	p.JwtService.Reset(w)
}
