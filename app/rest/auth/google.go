package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/render"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/umputun/remark/app/store"
)

// Google provides oauth2 and session store
type Google struct {
	*sessions.FilesystemStore
	conf *oauth2.Config
}

// NewGoogle makes auth with google
func NewGoogle(p Params) *Google {
	log.Printf("[INFO] create google auth, id=%s", p.Cid)

	conf := oauth2.Config{
		ClientID:     p.Cid,
		ClientSecret: p.Csecret,
		RedirectURL:  "http://remark.umputun.com:8080/auth/google",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
		},
		Endpoint: google.Endpoint,
	}

	return &Google{conf: &conf, FilesystemStore: p.SessionStore}
}

// LoginHandler - GET /login/google
func (a Google) LoginHandler(w http.ResponseWriter, r *http.Request) {

	// make state (random) and store in session
	state := randToken()
	session, err := a.Get(r, "remark")
	if err != nil {
		log.Printf("[WARN] %s", err)
	}
	session.Values["state-google"] = state
	session.Save(r, w)

	// return login url
	log.Printf("[DEBUG] login url %s", a.conf.AuthCodeURL(state))
	http.Redirect(w, r, a.conf.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

// AuthHandler is redirect url. Should check state to prevent CSRF.
func (a Google) AuthHandler(w http.ResponseWriter, r *http.Request) {
	session, err := a.Get(r, "remark")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get session, %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// compare saved state to the one from redirect url
	retrievedState, ok := session.Values["state-google"]
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
	uinfo, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
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

func (a Google) makeUserInfo(jData map[string]interface{}) store.User {
	userInfo := store.User{
		Name:    jData["name"].(string),
		ID:      jData["email"].(string),
		Picture: jData["picture"].(string),
		Profile: jData["profile"].(string),
	}
	if userInfo.Name == "" {
		userInfo.Name = strings.Split(userInfo.ID, "@")[0]
	}
	return userInfo
}
