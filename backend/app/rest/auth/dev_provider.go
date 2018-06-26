package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/umputun/remark/backend/app/store"
)

const devAuthPort = 8084

// DevAuthServer is a fake oauth server for development
type DevAuthServer struct {
	Provider Provider

	httpServer *http.Server
	lock       sync.Mutex
}

// Run oauth2 dev server on port devAuthPort
func (d *DevAuthServer) Run() {
	log.Printf("[INFO] run local oauth2 dev server on %d", devAuthPort)
	d.lock.Lock()
	d.httpServer = &http.Server{
		Addr: fmt.Sprintf(":%d", devAuthPort),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("[DEBUG] dev oauth request %s %s %+v", r.Method, r.URL, r.Header)
			switch {

			case strings.HasPrefix(r.URL.Path, "/login/oauth/authorize"):
				state := r.URL.Query().Get("state")
				callbackURL := fmt.Sprintf("%s?code=g0ZGZmNjVmOWI&state=%s", d.Provider.RedirectURL, state)
				log.Printf("[DEBUG] callback url=%s", callbackURL)
				w.Header().Add("Location", callbackURL)
				w.WriteHeader(http.StatusFound)

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
				if _, err := w.Write([]byte(res)); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

			case strings.HasPrefix(r.URL.Path, "/user"):
				res := `{
					"id": "ignored",
					"name":"ignored"
					}`
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				if _, err := w.Write([]byte(res)); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

			default:
				w.WriteHeader(http.StatusBadRequest)
			}
		}),
	}
	d.lock.Unlock()

	err := d.httpServer.ListenAndServe()
	log.Printf("[WARN] dev oauth2 server terminated, %s", err)
}

// Shutdown oauth2 dev server
func (d *DevAuthServer) Shutdown() {
	log.Print("[WARN] shutdown oauth2 dev server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	d.lock.Lock()
	if d.httpServer != nil {
		if err := d.httpServer.Shutdown(ctx); err != nil {
			log.Printf("[DEBUG] oauth2 dev shutdown error, %s", err)
		}
	}
	log.Print("[DEBUG] shutdown dev oauth2 server completed")
	d.lock.Unlock()
}

// NewDev makes dev oauth2 provider for admin user
func NewDev(p Params) Provider {
	return initProvider(p, Provider{
		Name: "dev",
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("http://127.0.0.1:%d/login/oauth/authorize", devAuthPort),
			TokenURL: fmt.Sprintf("http://127.0.0.1:%d/login/oauth/access_token", devAuthPort),
		},
		RedirectURL: "http://127.0.0.1:8080/auth/dev/callback",
		Scopes:      []string{"user:email"},
		InfoURL:     fmt.Sprintf("http://127.0.0.1:%d/user", devAuthPort),
		MapUser: func(data userData, _ []byte) store.User {
			userInfo := store.User{
				ID:      "dev_user",
				Name:    "developer",
				Picture: "",
			}
			return userInfo
		},
	})
}
