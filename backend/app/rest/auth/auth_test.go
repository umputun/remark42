package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testJwtUserBlocked = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjI3ODkxOTE4MjIsImp0aSI6InJhbmRvbSBpZCIsImlzcyI6InJlbWFyazQyIiwibmJmIjoxNTI2ODg0MjIyLCJ1c2VyIjp7Im5hbWUiOiJuYW1lMSIsImlkIjoiaWQxIiwicGljdHVyZSI6IiIsImFkbWluIjpmYWxzZSwiYmxvY2siOnRydWV9LCJzdGF0ZSI6IjEyMzQ1NiIsImZyb20iOiJmcm9tIn0.6P_OwGf8CUJRtvNSlW20GmaMb5pFvCNemP94fHCqb5Q"

func TestAuthJWTCookie(t *testing.T) {
	a := Authenticator{DevPasswd: "123456", JWTService: NewJWT("xyz 12345", false, time.Hour, time.Hour),
		UserFlags: &mockUserFlager{}}
	router := chi.NewRouter()
	router.With(a.Auth(true)).Get("/auth", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	})
	server := httptest.NewServer(router)
	defer server.Close()

	expiration := int(time.Duration(365 * 24 * time.Hour).Seconds())
	req, err := http.NewRequest("GET", server.URL+"/auth", nil)
	require.Nil(t, err)
	req.AddCookie(&http.Cookie{Name: "JWT", Value: testJwtValid, HttpOnly: true, Path: "/", MaxAge: expiration, Secure: false})
	req.Header.Add("X-XSRF-TOKEN", "random id")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode, "valid auth user")

	req, err = http.NewRequest("GET", server.URL+"/auth", nil)
	require.Nil(t, err)
	req.AddCookie(&http.Cookie{Name: "JWT", Value: testJwtValid, HttpOnly: true, Path: "/", MaxAge: expiration, Secure: false})
	req.Header.Add("X-XSRF-TOKEN", "wrong id")
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode, "xsrf mismatch")

	req, err = http.NewRequest("GET", server.URL+"/auth", nil)
	require.Nil(t, err)
	req.AddCookie(&http.Cookie{Name: "JWT", Value: testJwtExpired, HttpOnly: true, Path: "/", MaxAge: expiration, Secure: false})
	req.Header.Add("X-XSRF-TOKEN", "random id")
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode, "token expired and refreshed")
}

func TestAuthJWTHeader(t *testing.T) {
	a := Authenticator{DevPasswd: "123456", JWTService: NewJWT("xyz 12345", false, time.Hour, time.Hour)}
	router := chi.NewRouter()
	router.With(a.Auth(true)).Get("/auth", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	})
	server := httptest.NewServer(router)
	defer server.Close()

	jar, err := cookiejar.New(nil)
	require.Nil(t, err)
	client := &http.Client{Jar: jar, Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", server.URL+"/auth", nil)
	require.Nil(t, err)
	req.Header.Add("X-JWT", testJwtValid)
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode, "valid auth user")

	req, err = http.NewRequest("GET", server.URL+"/auth", nil)
	require.Nil(t, err)
	req.Header.Add("X-JWT", testJwtExpired)
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode, "token expired and refreshed")
}

func TestAuthJWtBlocked(t *testing.T) {
	a := Authenticator{DevPasswd: "123456", JWTService: NewJWT("xyz 12345", false, time.Hour, time.Hour)}
	router := chi.NewRouter()
	router.With(a.Auth(true)).Get("/auth", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	})
	server := httptest.NewServer(router)
	defer server.Close()

	jar, err := cookiejar.New(nil)
	require.Nil(t, err)
	client := &http.Client{Jar: jar, Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", server.URL+"/auth", nil)
	require.Nil(t, err)
	req.Header.Add("X-JWT", testJwtUserBlocked)
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode, "blocked user")
}

func TestAuthRequired(t *testing.T) {
	a := Authenticator{DevPasswd: "123456"}
	router := chi.NewRouter()
	router.With(a.Auth(true)).Get("/auth", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	})
	server := httptest.NewServer(router)
	defer server.Close()

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("GET", server.URL+"/auth", nil)
	require.NoError(t, err)
	req = withBasicAuth(req, "dev", "123456")
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode, "valid auth user")

	req, err = http.NewRequest("GET", server.URL+"/auth", nil)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode, "no auth user")

	req, err = http.NewRequest("GET", server.URL+"/auth", nil)
	require.NoError(t, err)
	req = withBasicAuth(req, "dev", "xyz")
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode, "wrong auth creds")
}

func TestAuthNotRequired(t *testing.T) {
	a := Authenticator{DevPasswd: "123456"}
	router := chi.NewRouter()
	router.With(a.Auth(false)).Get("/auth", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	})
	server := httptest.NewServer(router)
	defer server.Close()

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("GET", server.URL+"/auth", nil)
	require.NoError(t, err)
	req = withBasicAuth(req, "dev", "123456")
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode, "valid auth user")

	req, err = http.NewRequest("GET", server.URL+"/auth", nil)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode, "no auth user")

	req, err = http.NewRequest("GET", server.URL+"/auth", nil)
	require.NoError(t, err)
	req = withBasicAuth(req, "dev", "ZZZZ123456")
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode, "wrong auth creds")
}

func TestAdminRequired(t *testing.T) {
	a := Authenticator{DevPasswd: "123456"}
	router := chi.NewRouter()
	router.With(a.Auth(true), a.AdminOnly).Get("/auth", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	})
	server := httptest.NewServer(router)
	defer server.Close()

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("GET", server.URL+"/auth", nil)
	require.NoError(t, err)
	req = withBasicAuth(req, "dev", "123456")
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode, "valid auth user, admin")

	devUser.Admin = false
	req, err = http.NewRequest("GET", server.URL+"/auth", nil)
	require.NoError(t, err)
	req = withBasicAuth(req, "dev", "123456")
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode, "valid auth user, not admin")

}

func withBasicAuth(r *http.Request, username, password string) *http.Request {
	auth := username + ":" + password
	r.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	return r
}
