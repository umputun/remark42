package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/umputun/remark/app/store"
)

func TestLogin(t *testing.T) {

	sessionStore := &mockStore{values: make(map[interface{}]interface{})}

	_, ts, ots := mockProvider(t, sessionStore, 8981, 8982)
	defer func() {
		ts.Close()
		ots.Close()
	}()

	resp, err := http.Get("http://localhost:8981/login")
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	t.Logf("resp %s", string(body))
	t.Logf("headers: %+v", resp.Header)
	u := store.User{}
	err = json.Unmarshal(body, &u)
	assert.Nil(t, err)
	assert.Equal(t, store.User{Name: "blah", ID: "myuser", Picture: "",
		Admin: false, Blocked: false, IP: ""}, u)
}

func TestLogout(t *testing.T) {
	sessionStore := &mockStore{values: make(map[interface{}]interface{})}

	_, ts, ots := mockProvider(t, sessionStore, 8691, 8692)
	defer func() {
		ts.Close()
		ots.Close()
	}()

	resp, err := http.Get("http://localhost:8691/login")
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	_, err = http.Get("http://localhost:8691/logout")
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	s, err := sessionStore.Get(nil, "remark")
	assert.Nil(t, err)
	t.Log(s.Values)
	assert.Equal(t, 0, len(s.Values))
}

func mockProvider(t *testing.T, sessStore sessions.Store, loginPort, authPort int) (provider Provider, ts *http.Server, oauth *http.Server) {

	provider = Provider{
		Name: "mock",
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("http://localhost:%d/login/oauth/authorize", authPort),
			TokenURL: fmt.Sprintf("http://localhost:%d/login/oauth/access_token", authPort),
		},
		RedirectURL: fmt.Sprintf("http://localhost:%d/callback", loginPort),
		Scopes:      []string{"user:email"},
		InfoURL:     fmt.Sprintf("http://localhost:%d/user", authPort),
		MapUser: func(data userData, _ []byte) store.User {
			userInfo := store.User{
				ID:      data.value("id"),
				Name:    data.value("name"),
				Picture: data.value("picture"),
			}
			return userInfo
		},
	}

	provider = initProvider(Params{SessionStore: sessStore, Cid: "cid", Csecret: "csecret"}, provider)

	ts = &http.Server{Addr: fmt.Sprintf(":%d", loginPort), Handler: provider.Routes()}

	oauth = &http.Server{
		Addr: fmt.Sprintf(":%d", authPort),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("[MOCK OAUTH] request %s %s %+v", r.Method, r.URL, r.Header)
			switch {
			case strings.HasPrefix(r.URL.Path, "/login/oauth/authorize"):
				state := r.URL.Query().Get("state")
				w.Header().Add("Location", fmt.Sprintf("http://localhost:%d/callback?code=g0ZGZmNjVmOWI&state=%s", loginPort, state))
				w.WriteHeader(302)
			case strings.HasPrefix(r.URL.Path, "/login/oauth/access_token"):
				res := `{
					"access_token":"MTQ0NjJkZmQ5OTM2NDE1ZTZjNGZmZjI3",
					"token_type":"bearer",
					"expires_in":3600,
					"refresh_token":"IwOGYzYTlmM2YxOTQ5MGE3YmNmMDFkNTVk",
					"scope":"create",
					"state":"12345678"
					}`
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(200)
				w.Write([]byte(res))
			case strings.HasPrefix(r.URL.Path, "/user"):
				res := `{
					"id":"myuser",
					"name":"blah",
					"profile": "http://blah.com/p.html"
					}`
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(200)
				w.Write([]byte(res))
			default:
				t.Fatalf("unexpected oauth request %s %s", r.Method, r.URL)
			}
		}),
	}

	go oauth.ListenAndServe()
	go ts.ListenAndServe()

	return provider, ts, oauth
}

type mockStore struct {
	values map[interface{}]interface{}
}

func (ms *mockStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	if ms.values == nil {
		ms.values = make(map[interface{}]interface{})
	}
	s := sessions.NewSession(ms, name)
	s.Values = ms.values
	return s, nil
}

func (ms *mockStore) New(r *http.Request, name string) (*sessions.Session, error) {
	ms.values = make(map[interface{}]interface{})
	return &sessions.Session{Values: ms.values}, nil
}

func (ms *mockStore) Save(r *http.Request, w http.ResponseWriter, s *sessions.Session) error {
	if ms.values == nil {
		ms.values = make(map[interface{}]interface{})
	}
	ms.values = s.Values
	return nil
}
