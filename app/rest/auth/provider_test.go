package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/umputun/remark/app/store"
)

func TestLogin(t *testing.T) {

	p, ts, ots := mockProvider(t, 8981, 8982)
	defer func() {
		ts.Close()
		ots.Close()
	}()

	jar, err := cookiejar.New(nil)
	require.Nil(t, err)
	client := &http.Client{Jar: jar, Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:8981/login")
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	t.Logf("resp %s", string(body))
	t.Logf("headers: %+v", resp.Header)

	assert.Equal(t, 2, len(resp.Cookies()))
	assert.Equal(t, "JWT", resp.Cookies()[0].Name)
	assert.NotEqual(t, "", resp.Cookies()[0].Value, "jwt set")
	assert.Equal(t, "XSRF-TOKEN", resp.Cookies()[1].Name)
	assert.NotEqual(t, "", resp.Cookies()[1].Value, "xsrf cookie set")

	u := store.User{}
	err = json.Unmarshal(body, &u)
	assert.Nil(t, err)
	assert.Equal(t, store.User{Name: "blah", ID: "mock_myuser", Picture: "http://exmple.com/pic1.png",
		Admin: false, Blocked: false, IP: ""}, u)

	// check admin user
	p.Admins = []string{"mock_myuser"}
	resp, err = client.Get("http://localhost:8981/login")
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	err = json.Unmarshal(body, &u)
	assert.Nil(t, err)
	assert.Equal(t, store.User{Name: "blah", ID: "mock_myuser", Picture: "http://exmple.com/pic1.png",
		Admin: true, Blocked: false, IP: ""}, u)
}

func TestLogout(t *testing.T) {

	_, ts, ots := mockProvider(t, 8691, 8692)
	defer func() {
		ts.Close()
		ots.Close()
	}()

	jar, err := cookiejar.New(nil)
	require.Nil(t, err)
	client := &http.Client{Jar: jar, Timeout: 5 * time.Second}

	resp, err := client.Get("http://localhost:8691/login")
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 2, len(resp.Cookies()))
	resp, err = client.Get("http://localhost:8691/logout")
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	assert.Equal(t, 2, len(resp.Cookies()))
	assert.Equal(t, "JWT", resp.Cookies()[0].Name, "jwt cookie cleared")
	assert.Equal(t, "", resp.Cookies()[0].Value)
	assert.Equal(t, "XSRF-TOKEN", resp.Cookies()[1].Name, "xsrf cookie cleared")
	assert.Equal(t, "", resp.Cookies()[1].Value)
}

func TestInitProvider(t *testing.T) {
	params := Params{RemarkURL: "url", SecretKey: "123456", Cid: "cid", Csecret: "csecret"}
	provider := Provider{Name: "test", RedirectURL: "redir"}
	res := initProvider(params, provider)
	assert.Equal(t, "cid", res.conf.ClientID)
	assert.Equal(t, "csecret", res.conf.ClientSecret)
	assert.Equal(t, "redir", res.RedirectURL)
	assert.Equal(t, "123456", res.SecretKey)
	assert.Equal(t, "test", res.Name)
}

func mockProvider(t *testing.T, loginPort, authPort int) (*Provider, *http.Server, *http.Server) {

	provider := Provider{
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
				ID:      "mock_" + data.value("id"),
				Name:    data.value("name"),
				Picture: data.value("picture"),
			}
			return userInfo
		},
	}
	params := Params{RemarkURL: "url", SecretKey: "123456", Cid: "cid", Csecret: "csecret",
		JwtService: NewJWT("12345", false, time.Hour), Admins: []string{""}}
	provider = initProvider(params, provider)

	ts := &http.Server{Addr: fmt.Sprintf(":%d", loginPort), Handler: provider.Routes()}

	oauth := &http.Server{
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
					"picture":"http://exmple.com/pic1.png"
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

	time.Sleep(time.Millisecond * 100) // let the start
	return &provider, ts, oauth
}
