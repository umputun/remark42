package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"golang.org/x/oauth2"

	"github.com/umputun/remark/backend/app/rest"
	"github.com/umputun/remark/backend/app/rest/proxy"
	"github.com/umputun/remark/backend/app/store"
)

// Provider represents oauth2 provider
type Provider struct {
	Params
	Name        string
	RedirectURL string
	InfoURL     string
	Endpoint    oauth2.Endpoint
	Scopes      []string
	MapUser     func(userData, []byte) store.User // map info from InfoURL to User
	conf        oauth2.Config
}

// Params to make initialized and ready to use provider
type Params struct {
	RemarkURL         string
	AvatarProxy       *proxy.Avatar
	JwtService        *JWT
	PermissionChecker PermissionChecker
	SecretKey         string
	Cid               string
	Csecret           string
}

type userData map[string]interface{}

func (u userData) value(key string) string {
	// json.Unmarshal converts json "null" value to go's "nil", in this case return empty string
	if val, ok := u[key]; ok && val != nil {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// newProvider makes auth for given provider
func initProvider(p Params, provider Provider) Provider {
	log.Printf("[INFO] init auth provider %s", provider.Name)
	provider.Params = p
	provider.conf = oauth2.Config{
		ClientID:     provider.Cid,
		ClientSecret: provider.Csecret,
		RedirectURL:  provider.RedirectURL,
		Scopes:       provider.Scopes,
		Endpoint:     provider.Endpoint,
	}

	log.Printf("[DEBUG] created %s auth, id=%s, redir=%s, endpoint=%s",
		provider.Name, provider.Cid, provider.Endpoint, provider.RedirectURL)
	return provider
}

// Routes returns auth routes for given provider
func (p Provider) Routes() chi.Router {
	router := chi.NewRouter()
	router.Get("/login", p.loginHandler)
	router.Get("/callback", p.authHandler)
	router.Get("/logout", p.LogoutHandler)
	return router
}

// loginHandler - GET /login?from=redirect-back-url&site=siteID&session=1
func (p Provider) loginHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("[DEBUG] login with %s", p.Name)
	// make state (random) and store in session
	state := p.randToken()

	claims := CustomClaims{
		State:       state,
		From:        r.URL.Query().Get("from"),
		SiteID:      r.URL.Query().Get("site"),
		SessionOnly: r.URL.Query().Get("session") != "" && r.URL.Query().Get("session") != "0",
		StandardClaims: jwt.StandardClaims{
			Id:        p.randToken(),
			Issuer:    "remark42",
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
			NotBefore: time.Now().Add(-1 * time.Minute).Unix(),
		},
	}
	claims.Flags.Login = true

	if err := p.JwtService.Set(w, &claims, false); err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to set jwt")
		return
	}

	// return login url
	loginURL := p.conf.AuthCodeURL(state)
	log.Printf("[DEBUG] login url %s, claims=%+v", loginURL, claims)

	http.Redirect(w, r, loginURL, http.StatusFound)
}

// authHandler fills user info and redirects to "from" url. This is callback url redirected locally by browser
// GET /callback
func (p Provider) authHandler(w http.ResponseWriter, r *http.Request) {
	oauthClaims, err := p.JwtService.Get(r)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to get jwt")
		return
	}

	retrievedState := oauthClaims.State
	if retrievedState == "" || retrievedState != r.URL.Query().Get("state") {
		http.Error(w, fmt.Sprintf("unexpected state %v", retrievedState), http.StatusUnauthorized)
		return
	}

	log.Printf("[DEBUG] auth with state %s", retrievedState)
	tok, err := p.conf.Exchange(context.Background(), r.URL.Query().Get("code"))
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "exchange failed")
		return
	}

	client := p.conf.Client(context.Background(), tok)
	uinfo, err := client.Get(p.InfoURL)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, fmt.Sprintf("failed to get client info via %s", p.InfoURL))
		return
	}

	defer func() {
		if e := uinfo.Body.Close(); e != nil {
			log.Printf("[WARN] failed to close response body, %s", e)
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
	log.Printf("[DEBUG] got raw user info %+v", jData)

	u := p.MapUser(jData, data)
	u = p.setPermissions(u, oauthClaims.SiteID)
	u = p.setAvatar(u)

	claims := &CustomClaims{
		User: &u,
		StandardClaims: jwt.StandardClaims{
			Issuer: "remark42",
			Id:     p.randToken(),
		},
		SessionOnly: oauthClaims.SessionOnly,
	}

	if err = p.JwtService.Set(w, claims, oauthClaims.SessionOnly); err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to save user info")
		return
	}

	log.Printf("[DEBUG] user info %+v", u)

	// redirect to back url if presented in login query params
	if oauthClaims.From != "" {
		http.Redirect(w, r, oauthClaims.From, http.StatusTemporaryRedirect)
		return
	}
	render.JSON(w, r, &u)
}

// setAvatar saves avatar and puts proxied URL to u.Picture
func (p Provider) setAvatar(u store.User) store.User {
	if p.AvatarProxy != nil {
		if avatarURL, e := p.AvatarProxy.Put(u); e == nil {
			u.Picture = avatarURL
		} else {
			log.Printf("[WARN] failed to proxy avatar, %s", e)
		}
	}
	return u
}

// setPermissions sets permission fields not handled by provider's MapUser, things like admin, verified and blocked
func (p Provider) setPermissions(u store.User, siteID string) store.User {
	u.Admin = p.PermissionChecker.IsAdmin(u.ID)
	u.Verified = p.PermissionChecker.IsVerified(siteID, u.ID)
	u.Blocked = p.PermissionChecker.IsBlocked(siteID, u.ID)
	log.Printf("[DEBUG] set permissions for user %s, site %s - %+v", u.ID, siteID, u)
	return u
}

// LogoutHandler - GET /logout
func (p Provider) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	p.JwtService.Reset(w)
	log.Printf("[DEBUG] logout")
}

func (p Provider) randToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("[ERROR] can't get randoms, %s", err)
	}
	s := sha1.New()
	if _, err := s.Write(b); err != nil {
		log.Printf("[WARN] can't write randoms, %s", err)
	}
	return fmt.Sprintf("%x", s.Sum(nil))
}
