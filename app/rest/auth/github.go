package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/go-chi/render"
	"github.com/gorilla/sessions"
	"github.com/umputun/remark/app/store"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// Github provides oauth2 and session store
type Github struct {
	*sessions.FilesystemStore
	conf *oauth2.Config
}

// NewGithub makes auth with github
func NewGithub(p Params) *Github {
	log.Printf("[INFO] create gihub auth, id=%s", p.Cid)

	conf := oauth2.Config{
		ClientID:     p.Cid,
		ClientSecret: p.Csecret,
		RedirectURL:  "http://remark.umputun.com:8080/auth/github",
		Scopes: []string{
			"user:email",
		},
		Endpoint: github.Endpoint,
	}

	return &Github{conf: &conf, FilesystemStore: p.SessionStore}
}

// LoginHandler - GET /login/github
func (a Github) LoginHandler(w http.ResponseWriter, r *http.Request) {

	// make state (random) and store in session
	state := randToken()
	session, err := a.Get(r, "remark")
	if err != nil {
		log.Printf("[WARN] %s", err)
	}
	session.Values["state-github"] = state
	session.Save(r, w)

	// return login url
	log.Printf("[DEBUG] login url %s", a.conf.AuthCodeURL(state))
	http.Redirect(w, r, a.conf.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

// AuthHandler is redirect url. Should check state to prevent CSRF.
func (a Github) AuthHandler(w http.ResponseWriter, r *http.Request) {
	session, err := a.Get(r, "remark")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get session, %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// compare saved state to the one from redirect url
	retrievedState, ok := session.Values["state-github"]
	if !ok || retrievedState != r.URL.Query().Get("state") {
		http.Error(w, fmt.Sprintf("unexpected state %s", retrievedState.(string)), http.StatusUnauthorized)
		return
	}

	tok, err := a.conf.Exchange(context.Background(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, fmt.Sprintf("exchange failed, %s", err), http.StatusInternalServerError)
		return
	}

	client := a.conf.Client(context.Background(), tok)
	uinfo, err := client.Get("https://api.github.com/user")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get client info, %s", err), http.StatusBadRequest)
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

	session.Values["uinfo"] = a.makeUserInfo(jData)
	if err = session.Save(r, w); err != nil {
		http.Error(w, fmt.Sprintf("failed to save user info, %s", err), http.StatusInternalServerError)
		return
	}

	log.Printf("[DEBUG] %+v", jData)
	render.JSON(w, r, jData)
}

func (a Github) makeUserInfo(jData map[string]interface{}) store.User {
	userInfo := store.User{
		ID:      jData["login"].(string),
		Name:    jData["name"].(string),
		Picture: jData["avatar_url"].(string),
		Profile: jData["html_url"].(string),
	}
	if userInfo.Name == "" {
		userInfo.Name = userInfo.ID
	}
	return userInfo
}
