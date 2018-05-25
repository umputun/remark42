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

	"github.com/umputun/remark/app/rest"
	"github.com/umputun/remark/app/rest/proxy"
	"github.com/umputun/remark/app/store"
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
	RemarkURL   string
	AvatarProxy *proxy.Avatar
	JwtService  *JWT
	SecretKey   string
	Admins      []string
	Cid         string
	Csecret     string
}

type userData map[string]interface{}

func (u userData) value(key string) string {
	if val, ok := u[key]; ok {
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

// loginHandler - GET /login?from=redirect-back-url
func (p Provider) loginHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("[DEBUG] login with %s", p.Name)
	// make state (random) and store in session
	state := p.randToken()

	claims := CustomClaims{
		State: state,
		From:  r.URL.Query().Get("from"),
		StandardClaims: jwt.StandardClaims{
			Id:        p.randToken(),
			Issuer:    "remark42",
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
			NotBefore: time.Now().Add(-1 * time.Minute).Unix(),
		},
	}

	if err := p.JwtService.Set(w, &claims); err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to set jwt")
		return
	}

	// return login url
	loginURL := p.conf.AuthCodeURL(state)
	log.Printf("[DEBUG] login url %s", loginURL)

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
	if p.AvatarProxy != nil {
		if avatarURL, e := p.AvatarProxy.Put(u); e == nil {
			u.Picture = avatarURL
		} else {
			log.Printf("[WARN] failed to proxy avatar, %s", e)
		}
	}
	u.Admin = isAdmin(u.ID, p.Admins)

	authClaims := &CustomClaims{
		User: &u,
		StandardClaims: jwt.StandardClaims{
			Issuer: "remark42",
			Id:     p.randToken(),
		},
	}

	if err = p.JwtService.Set(w, authClaims); err != nil {
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
