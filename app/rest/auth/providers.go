package auth

import (
	"log"
	"strings"

	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/umputun/remark/app/store"
)

// NewGoogle makes google oauth2 provider
func NewGoogle(p Params) Provider {
	return initProvider(p, Provider{
		Name:        "google",
		Endpoint:    google.Endpoint,
		RedirectURL: p.RemarkURL + "/auth/google/callback",
		Scopes:      []string{"https://www.googleapis.com/auth/userinfo.email"},
		InfoURL:     "https://www.googleapis.com/oauth2/v3/userinfo",
		Store:       p.SessionStore,
		MapUser: func(data userData) store.User {
			userInfo := store.User{
				ID:      data.value("email"),
				Name:    data.value("name"),
				Picture: data.value("picture"),
				Profile: data.value("profile"),
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
func NewGithub(p Params) Provider {
	return initProvider(p, Provider{
		Name:        "github",
		Endpoint:    github.Endpoint,
		RedirectURL: p.RemarkURL + "/auth/github/callback",
		Scopes:      []string{"user:email"},
		InfoURL:     "https://api.github.com/user",
		Store:       p.SessionStore,
		MapUser: func(data userData) store.User {
			userInfo := store.User{
				ID:      data.value("login"),
				Name:    data.value("name"),
				Picture: data.value("avatar_url"),
				Profile: data.value("html_url"),
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
func NewFacebook(p Params) Provider {
	return initProvider(p, Provider{
		Name:        "facebook",
		Endpoint:    facebook.Endpoint,
		RedirectURL: p.RemarkURL + "/auth/facebook/callback",
		Scopes:      []string{"public_profile"},
		InfoURL:     "https://graph.facebook.com/me?fields=id,name,picture",
		Store:       p.SessionStore,
		MapUser: func(data userData) store.User {
			userInfo := store.User{
				ID:   data.value("id"),
				Name: data.value("name"),
			}
			if userInfo.Name == "" {
				userInfo.Name = userInfo.ID
			}
			userInfo.ID = "facebook_" + userInfo.ID

			log.Printf("%T", data["picture"])
			// picture under picture[data[url]]
			if p, ok := data["picture"]; ok {
				if d, ok := p.(map[string]interface{}); ok {
					if picURL, ok := d["url"]; ok {
						userInfo.Picture = picURL.(string)
					}
				}
			}
			return userInfo
		},
	})
}
