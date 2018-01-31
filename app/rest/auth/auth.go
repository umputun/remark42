// Package auth provides oauth2 support as well as related middlewares.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"

	"github.com/umputun/remark/app/rest/common"
	"github.com/umputun/remark/app/store"
)

// Provider represents oauth2 provider
type Provider struct {
	sessions.Store

	Name        string
	RedirectURL string
	InfoURL     string
	Endpoint    oauth2.Endpoint
	Scopes      []string
	MapUser     func(userData) store.User

	conf *oauth2.Config
}

// Params to make initialized and ready to use provider
type Params struct {
	Cid          string
	Csecret      string
	SessionStore sessions.Store
	RemarkURL    string
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
	log.Printf("[INFO] create %s auth, id=%s, redir: %s", provider.Name, p.Cid, provider.RedirectURL)

	conf := oauth2.Config{
		ClientID:     p.Cid,
		ClientSecret: p.Csecret,
		RedirectURL:  provider.RedirectURL,
		Scopes:       provider.Scopes,
		Endpoint:     provider.Endpoint,
	}

	provider.conf = &conf
	provider.Store = p.SessionStore
	return provider
}

// Routes returns auth routes for given provider
func (p Provider) Routes() chi.Router {
	router := chi.NewRouter()
	router.Get("/login", p.LoginHandler)
	router.Get("/callback", p.AuthHandler)
	router.Get("/logout", p.LogoutHandler)
	return router
}

// LoginHandler - GET /login?from=redirect-back-url
func (p Provider) LoginHandler(w http.ResponseWriter, r *http.Request) {

	// make state (random) and store in session
	state := p.randToken()
	session, err := p.Get(r, "remark")
	if err != nil {
		log.Printf("[DEBUG] can't get session, %s", err)
	}

	session.Values["state"] = state

	if from := r.URL.Query().Get("from"); from != "" {
		session.Values["from"] = from
	}

	log.Printf("[DEBUG] login, %+v", session.Values)
	if err := session.Save(r, w); err != nil {
		common.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to save state")
		return
	}

	// return login url
	loginURL := p.conf.AuthCodeURL(state)
	log.Printf("[DEBUG] login url %s", loginURL)
	http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
}

// AuthHandler fills user info and redirects to "from" url. This is callback url redirected locally by browser
// GET /callback
func (p Provider) AuthHandler(w http.ResponseWriter, r *http.Request) {

	session, err := p.Get(r, "remark")
	if err != nil {
		common.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to get session")
		return
	}

	// compare saved state to the one from redirect url
	retrievedState, ok := session.Values["state"]
	if !ok {
		http.Error(w, "missing state in store", http.StatusUnauthorized)
		return
	}

	if retrievedState == "" || retrievedState != r.URL.Query().Get("state") {
		http.Error(w, fmt.Sprintf("unexpected state %v", retrievedState), http.StatusUnauthorized)
		return
	}

	log.Printf("[DEBUG] auth, %+v", session.Values)
	tok, err := p.conf.Exchange(context.Background(), r.URL.Query().Get("code"))
	if err != nil {
		common.SendErrorJSON(w, r, http.StatusInternalServerError, err, "exchange failed")
		return
	}

	client := p.conf.Client(context.Background(), tok)
	uinfo, err := client.Get(p.InfoURL)
	if err != nil {
		common.SendErrorJSON(w, r, http.StatusBadRequest, err, fmt.Sprintf("failed to get client info via %s", p.InfoURL))
		return
	}

	defer func() {
		if e := uinfo.Body.Close(); e != nil {
			log.Printf("[WARN] failed to close response body, %s", e)
		}
	}()

	data, err := ioutil.ReadAll(uinfo.Body)
	if err != nil {
		common.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to read user info")
		return
	}

	jData := map[string]interface{}{}
	if e := json.Unmarshal(data, &jData); e != nil {
		common.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to unmarshal user info")
		return
	}
	log.Printf("[DEBUG] got raw user info %+v", jData)

	session.Values["uinfo"] = p.MapUser(jData)
	if err = session.Save(r, w); err != nil {
		common.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to save user info")
		return
	}

	log.Printf("[DEBUG] %+v", jData)

	// redirect to back url if presented in login query params
	if fromURL, ok := session.Values["from"]; ok {
		http.Redirect(w, r, fromURL.(string), http.StatusTemporaryRedirect)
		return
	}

	render.JSON(w, r, jData)
}

// LogoutHandler - GET /logout
func (p Provider) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := p.Get(r, "remark")
	if err != nil {
		common.SendErrorJSON(w, r, http.StatusBadRequest, err, "failed to get session")
		return
	}

	session.Values["uinfo"] = ""
	session.Values["from"] = ""
	session.Values["state"] = ""

	delete(session.Values, "uinfo")
	delete(session.Values, "from")
	delete(session.Values, "state")

	if err = session.Save(r, w); err != nil {
		common.SendErrorJSON(w, r, http.StatusInternalServerError, err, "failed to reset user info")
		return
	}
	log.Printf("[DEBUG] logout, %+v", session.Values)
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

func init() {
	gob.Register(store.User{})
}
