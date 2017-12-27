package auth

import (
	"strings"

	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/umputun/remark/app/store"
)

// NewGoogle makes google oauth2 provider
func NewGoogle(p Params) *Provider {
	return initProvider(p, Provider{
		Name:            "google",
		Endpoint:        google.Endpoint,
		RedirectURL:     p.RemarkURL + "/auth/google",
		Scopes:          []string{"https://www.googleapis.com/auth/userinfo.email"},
		InfoURL:         "https://www.googleapis.com/oauth2/v3/userinfo",
		FilesystemStore: p.SessionStore,
		MapUser: func(data map[string]interface{}) store.User {
			userInfo := store.User{
				ID:      data["email"].(string),
				Name:    data["name"].(string),
				Picture: data["picture"].(string),
				Profile: data["profile"].(string),
			}
			if userInfo.Name == "" {
				userInfo.Name = strings.Split(userInfo.ID, "@")[0]
			}
			userInfo.ID = "google_" + userInfo.ID
			return userInfo
		},
	})
}

// NewGithub makes github oauth2 provider
func NewGithub(p Params) *Provider {
	return initProvider(p, Provider{
		Name:            "github",
		Endpoint:        github.Endpoint,
		RedirectURL:     p.RemarkURL + "/auth/github",
		Scopes:          []string{"user:email"},
		InfoURL:         "https://api.github.com/user",
		FilesystemStore: p.SessionStore,
		MapUser: func(data map[string]interface{}) store.User {
			userInfo := store.User{
				ID:      data["login"].(string),
				Name:    data["name"].(string),
				Picture: data["avatar_url"].(string),
				Profile: data["html_url"].(string),
			}
			if userInfo.Name == "" {
				userInfo.Name = userInfo.ID
			}
			userInfo.ID = "github_" + userInfo.ID
			return userInfo
		},
	})
}
