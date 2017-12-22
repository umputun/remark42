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

type Provider struct {
	Name        string
	RedirectURL string
	InfoURL     string
	Endpoint    oauth2.Endpoint
	Scopes      []string
	MapUser     func(map[string]interface{}) store.User

	*sessions.FilesystemStore
	conf *oauth2.Config
}

type Params struct {
	Cid          string
	Csecret      string
	SessionStore *sessions.FilesystemStore
	Admins       []string
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

// LoginHandler - GET /login/github
func (p Provider) LoginHandler(w http.ResponseWriter, r *http.Request) {

	// make state (random) and store in session
	state := randToken()
	session, err := p.Get(r, "remark")
	if err != nil {
		log.Printf("[WARN] %s", err)
	}
	session.Values["state-"+p.Name] = state
	session.Save(r, w)

	// return login url
	log.Printf("[DEBUG] login url %s", p.conf.AuthCodeURL(state))
	http.Redirect(w, r, p.conf.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

// AuthHandler is redirect url. Should check state to prevent CSRF.
func (p Provider) AuthHandler(w http.ResponseWriter, r *http.Request) {
	session, err := p.Get(r, "remark")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get session, %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// compare saved state to the one from redirect url
	retrievedState, ok := session.Values["state-"+p.Name]
	if !ok || retrievedState != r.URL.Query().Get("state") {
		http.Error(w, fmt.Sprintf("unexpected state %s", retrievedState.(string)), http.StatusUnauthorized)
		return
	}

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

	defer uinfo.Body.Close()
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
	render.JSON(w, r, jData)
}

func randToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	s := sha1.New()
	s.Write(b)
	return fmt.Sprintf("%x", s.Sum(nil))
}

func init() {
	gob.Register(store.User{})
}
