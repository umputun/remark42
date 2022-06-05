package provider

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-pkgz/auth/token"
)

const (
	urlLoginSuffix    = "/login"
	urlCallbackSuffix = "/callback"
	urlLogoutSuffix   = "/logout"
)

// Service represents oauth2 provider. Adds Handler method multiplexing login, auth and logout requests
type Service struct {
	Provider
}

// NewService makes service for given provider
func NewService(p Provider) Service {
	return Service{Provider: p}
}

// AvatarSaver defines minimal interface to save avatar
type AvatarSaver interface {
	Put(u token.User, client *http.Client) (avatarURL string, err error)
}

// TokenService defines interface accessing tokens
type TokenService interface {
	Parse(tokenString string) (claims token.Claims, err error)
	Set(w http.ResponseWriter, claims token.Claims) (token.Claims, error)
	Get(r *http.Request) (claims token.Claims, token string, err error)
	Reset(w http.ResponseWriter)
}

// Provider defines interface for auth handler
type Provider interface {
	Name() string
	LoginHandler(w http.ResponseWriter, r *http.Request)
	AuthHandler(w http.ResponseWriter, r *http.Request)
	LogoutHandler(w http.ResponseWriter, r *http.Request)
}

// Handler returns auth routes for given provider
func (p Service) Handler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if strings.HasSuffix(r.URL.Path, urlLoginSuffix) {
		p.LoginHandler(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, urlCallbackSuffix) {
		p.AuthHandler(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, urlLogoutSuffix) {
		p.LogoutHandler(w, r)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

// setAvatar saves avatar and puts proxied URL to u.Picture
func setAvatar(ava AvatarSaver, u token.User, client *http.Client) (token.User, error) {
	if ava != nil {
		avatarURL, e := ava.Put(u, client)
		if e != nil {
			return u, fmt.Errorf("failed to save avatar for: %w", e)
		}
		u.Picture = avatarURL
		return u, nil
	}
	return u, nil // empty AvatarSaver ok, just skipped
}

func randToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("can't get random: %w", err)
	}
	s := sha1.New()
	if _, err := s.Write(b); err != nil {
		return "", fmt.Errorf("can't write randoms to sha1: %w", err)
	}
	return fmt.Sprintf("%x", s.Sum(nil)), nil
}
