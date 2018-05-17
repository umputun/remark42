package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthRequired(t *testing.T) {
	store := mockStore{}
	a := Authenticator{SessionStore: &store, DevPasswd: "123456"}
	router := chi.NewRouter()
	router.With(a.Auth(true)).Get("/auth", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	})
	server := httptest.NewServer(router)
	defer server.Close()

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("GET", server.URL+"/auth", nil)
	req.Header.Add("Authorization", "Basic "+basicAuth("dev", "123456"))
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode, "valid auth user")

	req, err = http.NewRequest("GET", server.URL+"/auth", nil)
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode, "no auth user")

	req, err = http.NewRequest("GET", server.URL+"/auth", nil)
	req.Header.Add("Authorization", "Basic "+basicAuth("dev", "ZZZZ123456"))
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode, "wrong auth creds")
}

func TestAuthNotRequired(t *testing.T) {
	store := mockStore{}
	a := Authenticator{SessionStore: &store, DevPasswd: "123456"}
	router := chi.NewRouter()
	router.With(a.Auth(false)).Get("/auth", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	})
	server := httptest.NewServer(router)
	defer server.Close()

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("GET", server.URL+"/auth", nil)
	req.Header.Add("Authorization", "Basic "+basicAuth("dev", "123456"))
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode, "valid auth user")

	req, err = http.NewRequest("GET", server.URL+"/auth", nil)
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode, "no auth user")

	req, err = http.NewRequest("GET", server.URL+"/auth", nil)
	req.Header.Add("Authorization", "Basic "+basicAuth("dev", "ZZZZ123456"))
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode, "wrong auth creds")
}

func TestAdminRequired(t *testing.T) {
	store := mockStore{}
	a := Authenticator{SessionStore: &store, DevPasswd: "123456"}
	router := chi.NewRouter()
	router.With(a.Auth(true), a.AdminOnly).Get("/auth", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	})
	server := httptest.NewServer(router)
	defer server.Close()

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("GET", server.URL+"/auth", nil)
	req.Header.Add("Authorization", "Basic "+basicAuth("dev", "123456"))
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode, "valid auth user, admin")

	devUser.Admin = false
	req, err = http.NewRequest("GET", server.URL+"/auth", nil)
	req.Header.Add("Authorization", "Basic "+basicAuth("dev", "123456"))
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode, "valid auth user, not admin")

}
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
