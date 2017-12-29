package auth

import (
	"fmt"
	"strings"

	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/umputun/remark/app/store"
)

// NewGoogle makes google oauth2 provider
func NewGoogle(p Params) *Provider {
	return initProvider(p, Provider{
		Name:            "google",
		Endpoint:        google.Endpoint,
		RedirectURL:     p.RemarkURL + "/auth/google/callback",
		Scopes:          []string{"https://www.googleapis.com/auth/userinfo.email"},
		InfoURL:         "https://www.googleapis.com/oauth2/v3/userinfo",
		FilesystemStore: p.SessionStore,
		MapUser: func(data map[string]interface{}) store.User {
			userInfo := store.User{
				ID:      value(data, "email"),
				Name:    value(data, "name"),
				Picture: value(data, "picture"),
				Profile: value(data, "profile"),
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
		RedirectURL:     p.RemarkURL + "/auth/github/callback",
		Scopes:          []string{"user:email"},
		InfoURL:         "https://api.github.com/user",
		FilesystemStore: p.SessionStore,
		MapUser: func(data map[string]interface{}) store.User {
			userInfo := store.User{
				ID:      value(data, "login"),
				Name:    value(data, "name"),
				Picture: value(data, "avatar_url"),
				Profile: value(data, "html_url"),
			}
			if userInfo.Name == "" {
				userInfo.Name = userInfo.ID
			}
			userInfo.ID = "github_" + userInfo.ID
			return userInfo
		},
	})
}

// NewFacebook makes facebook oauth2 provider
func NewFacebook(p Params) *Provider {
	return initProvider(p, Provider{
		Name:            "facebook",
		Endpoint:        facebook.Endpoint,
		RedirectURL:     p.RemarkURL + "/auth/facebook/callback",
		Scopes:          []string{"public_profile"},
		InfoURL:         "https://graph.facebook.com/me",
		FilesystemStore: p.SessionStore,
		MapUser: func(data map[string]interface{}) store.User {
			userInfo := store.User{
				ID:   value(data, "id"),
				Name: value(data, "name"),
				// Picture: data["avatar_url"].(string),
				// Profile: data["html_url"].(string),
			}
			if userInfo.Name == "" {
				userInfo.Name = userInfo.ID
			}
			userInfo.ID = "facebook_" + userInfo.ID
			return userInfo
		},
	})
}

func value(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		return fmt.Sprintf("%v", val)
	}
	return ""
}
