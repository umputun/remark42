package provider

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/dghubble/oauth1"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-pkgz/rest"

	"github.com/go-pkgz/auth/logger"
	"github.com/go-pkgz/auth/token"
)

// Oauth1Handler implements /login, /callback and /logout handlers for oauth1 flow
type Oauth1Handler struct {
	Params
	name    string
	infoURL string
	conf    oauth1.Config
	mapUser func(UserData, []byte) token.User // map info from InfoURL to User
}

// Name returns provider name
func (h Oauth1Handler) Name() string { return h.name }

// LoginHandler - GET /login?from=redirect-back-url&site=siteID&session=1
func (h Oauth1Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	h.Logf("[DEBUG] login with %s", h.Name())

	// setting RedirectURL to {rootURL}/{routingPath}/{provider}/callback
	// e.g. http://localhost:8080/auth/twitter/callback
	h.conf.CallbackURL = h.makeRedirURL(r.URL.Path)

	requestToken, requestSecret, err := h.conf.RequestToken()
	if err != nil {
		rest.SendErrorJSON(w, r, h.L, http.StatusInternalServerError, err, "failed to get request token")
		return
	}

	// use requestSecret as a state in oauth2
	cid, err := randToken()
	if err != nil {
		rest.SendErrorJSON(w, r, h.L, http.StatusInternalServerError, err, "failed to make claim's id")
		return
	}

	claims := token.Claims{
		Handshake: &token.Handshake{
			State: requestSecret,
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

	if _, err = h.JwtService.Set(w, claims); err != nil {
		rest.SendErrorJSON(w, r, h.L, http.StatusInternalServerError, err, "failed to set token")
		return
	}

	authURL, err := h.conf.AuthorizationURL(requestToken)
	if err != nil {
		rest.SendErrorJSON(w, r, h.L, http.StatusInternalServerError, err, "failed to obtain oauth1 URL")
		return
	}

	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

// AuthHandler fills user info and redirects to "from" url. This is callback url redirected locally by browser
// GET /callback
func (h Oauth1Handler) AuthHandler(w http.ResponseWriter, r *http.Request) {
	oauthClaims, _, err := h.JwtService.Get(r)
	if err != nil {
		rest.SendErrorJSON(w, r, h.L, http.StatusInternalServerError, err, "failed to get token")
		return
	}

	requestToken, verifier, err := oauth1.ParseAuthorizationCallback(r)
	if err != nil {
		rest.SendErrorJSON(w, r, h.L, http.StatusInternalServerError, err, "failed to parse response from oauth1 server")
		return
	}

	accessToken, accessSecret, err := h.conf.AccessToken(requestToken, oauthClaims.Handshake.State, verifier)
	if err != nil {
		rest.SendErrorJSON(w, r, h.L, http.StatusInternalServerError, err, "failed to get accessToken and accessSecret")
		return
	}

	tok := oauth1.NewToken(accessToken, accessSecret)
	client := h.conf.Client(context.Background(), tok)

	uinfo, err := client.Get(h.infoURL)
	if err != nil {
		rest.SendErrorJSON(w, r, h.L, http.StatusServiceUnavailable, err, "failed to get client info")
		return
	}

	defer func() {
		if e := uinfo.Body.Close(); e != nil {
			h.Logf("[WARN] failed to close response body, %s", e)
		}
	}()

	data, err := ioutil.ReadAll(uinfo.Body)
	if err != nil {
		rest.SendErrorJSON(w, r, h.L, http.StatusInternalServerError, err, "failed to read user info")
		return
	}

	jData := map[string]interface{}{}
	if e := json.Unmarshal(data, &jData); e != nil {
		rest.SendErrorJSON(w, r, h.L, http.StatusInternalServerError, err, "failed to unmarshal user info")
		return
	}
	h.Logf("[DEBUG] got raw user info %+v", jData)

	u := h.mapUser(jData, data)
	u, err = setAvatar(h.AvatarSaver, u, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		rest.SendErrorJSON(w, r, h.L, http.StatusInternalServerError, err, "failed to save avatar to proxy")
		return
	}

	cid, err := randToken()
	if err != nil {
		rest.SendErrorJSON(w, r, h.L, http.StatusInternalServerError, err, "failed to make claim's id")
		return
	}
	claims := token.Claims{
		User: &u,
		StandardClaims: jwt.StandardClaims{
			Issuer:   h.Issuer,
			Id:       cid,
			Audience: oauthClaims.Audience,
		},
		SessionOnly: oauthClaims.SessionOnly,
	}

	if _, err = h.JwtService.Set(w, claims); err != nil {
		rest.SendErrorJSON(w, r, h.L, http.StatusInternalServerError, err, "failed to set token")
		return
	}

	h.Logf("[DEBUG] user info %+v", u)

	// redirect to back url if presented in login query params
	if oauthClaims.Handshake != nil && oauthClaims.Handshake.From != "" {
		http.Redirect(w, r, oauthClaims.Handshake.From, http.StatusTemporaryRedirect)
		return
	}
	rest.RenderJSON(w, &u)
}

// LogoutHandler - GET /logout
func (h Oauth1Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if _, _, err := h.JwtService.Get(r); err != nil {
		rest.SendErrorJSON(w, r, h.L, http.StatusForbidden, err, "logout not allowed")
		return
	}
	h.JwtService.Reset(w)
}

func (h Oauth1Handler) makeRedirURL(path string) string {
	elems := strings.Split(path, "/")
	newPath := strings.Join(elems[:len(elems)-1], "/")

	return strings.TrimRight(h.URL, "/") + strings.TrimRight(newPath, "/") + urlCallbackSuffix
}

// initOauth2Handler makes oauth1 handler for given provider
func initOauth1Handler(p Params, service Oauth1Handler) Oauth1Handler {
	if p.L == nil {
		p.L = logger.NoOp
	}
	p.Logf("[INFO] init oauth1 service %s", service.name)
	service.Params = p
	service.conf.ConsumerKey = p.Cid
	service.conf.ConsumerSecret = p.Csecret

	p.Logf("[DEBUG] created %s oauth2, id=%s, redir=%s, endpoint=%s",
		service.name, service.Cid, service.makeRedirURL("/{route}/"+service.name+"/"), service.conf.Endpoint)
	return service
}
