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

	"github.com/go-chi/render"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"

	"github.com/umputun/remark/app/store"
)

// Provider represents oauth2 provider
type Provider struct {
	*sessions.FilesystemStore

	Name        string
	RedirectURL string
	InfoURL     string
	Endpoint    oauth2.Endpoint
	Scopes      []string
	MapUser     func(map[string]interface{}) store.User

	conf *oauth2.Config
}

// Params to make initialized and ready to use provider
type Params struct {
	Cid          string
	Csecret      string
	SessionStore *sessions.FilesystemStore
	SiteURL      string
}

// newProvider makes auth for given provider
func initProvider(p Params, provider Provider) *Provider {
	log.Printf("[INFO] create %s auth, id=%s", provider.Name, p.Cid)

	conf := oauth2.Config{
		ClientID:     p.Cid,
		ClientSecret: p.Csecret,
		RedirectURL:  provider.RedirectURL,
		Scopes:       provider.Scopes,
		Endpoint:     provider.Endpoint,
	}

	provider.conf = &conf
	provider.FilesystemStore = p.SessionStore
	return &provider
}

// LoginHandler - GET /login/{provider}?from=redirect-back-url
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
		http.Error(w, fmt.Sprintf("failed to save start, %s", err), http.StatusInternalServerError)
		return
	}

	// return login url
	loginURL := p.conf.AuthCodeURL(state)
	log.Printf("[DEBUG] login url %s", loginURL)
	http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
}

// AuthHandler fills user info and redirects to "from" url. This is callback url redirected locally by browser
func (p Provider) AuthHandler(w http.ResponseWriter, r *http.Request) {

	session, err := p.Get(r, "remark")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get session, %s", err), http.StatusInternalServerError)
		return
	}

	// compare saved state to the one from redirect url
	retrievedState, ok := session.Values["state"]
	if !ok || retrievedState != r.URL.Query().Get("state") {
		http.Error(w, fmt.Sprintf("unexpected state %s", retrievedState.(string)), http.StatusUnauthorized)
		return
	}

	log.Printf("[DEBUG] auth, %+v", session.Values)
	tok, err := p.conf.Exchange(context.Background(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, fmt.Sprintf("exchange failed, %s", err), http.StatusInternalServerError)
		return
	}

	client := p.conf.Client(context.Background(), tok)
	uinfo, err := client.Get(p.InfoURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get client info via %s, %s", p.InfoURL, err), http.StatusBadRequest)
		return
	}

	defer func() {
		if e := uinfo.Body.Close(); e != nil {
			log.Printf("[WARN] failed to close response body, %s", e)
		}
	}()

	data, err := ioutil.ReadAll(uinfo.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read user info, %s", err), http.StatusInternalServerError)
		return
	}

	jData := map[string]interface{}{}
	if e := json.Unmarshal(data, &jData); e != nil {
		http.Error(w, fmt.Sprintf("failed to unmarshal user info, %s", err), http.StatusInternalServerError)
		return
	}
	log.Printf("[DEBUG] got raw user info %+v", jData)

	session.Values["uinfo"] = p.MapUser(jData)
	if err = session.Save(r, w); err != nil {
		http.Error(w, fmt.Sprintf("failed to save user info, %s", err), http.StatusInternalServerError)
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
		http.Error(w, fmt.Sprintf("failed to get session, %s", err), http.StatusInternalServerError)
		return
	}

	session.Values["uinfo"] = ""
	session.Values["from"] = ""
	session.Values["state"] = ""

	delete(session.Values, "uinfo")
	delete(session.Values, "from")
	delete(session.Values, "state")

	if err = session.Save(r, w); err != nil {
		http.Error(w, fmt.Sprintf("failed to reset user info, %s", err), http.StatusInternalServerError)
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
