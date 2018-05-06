package auth

import (
	"encoding/json"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/umputun/remark/app/rest"
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
		MapUser: func(data userData, _ []byte) store.User {
			userInfo := store.User{
				// encode email with provider name to avoid collision if same id returned by other provider
				ID:      "google_" + rest.EncodeID("google_"+data.value("email")),
				Name:    data.value("name"),
				Picture: data.value("picture"),
			}
			if userInfo.Name == "" {
				userInfo.Name = strings.Split(data.value("email"), "@")[0]
			}
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
		MapUser: func(data userData, _ []byte) store.User {
			userInfo := store.User{
				ID:      "github_" + rest.EncodeID("github_"+data.value("login")),
				Name:    data.value("name"),
				Picture: data.value("avatar_url"),
			}
			if userInfo.Name == "" {
				userInfo.Name = userInfo.ID
			}
			return userInfo
		},
	})
}

// NewFacebook makes facebook oauth2 provider
func NewFacebook(p Params) Provider {

	// response format for fb /me call
	type uinfo struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Picture struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"picture"`
	}

	return initProvider(p, Provider{
		Name:        "facebook",
		Endpoint:    facebook.Endpoint,
		RedirectURL: p.RemarkURL + "/auth/facebook/callback",
		Scopes:      []string{"public_profile"},
		InfoURL:     "https://graph.facebook.com/me?fields=id,name,picture",
		Store:       p.SessionStore,
		MapUser: func(data userData, bdata []byte) store.User {
			userInfo := store.User{
				ID:   "facebook_" + ("facebook_" + rest.EncodeID(data.value("id"))),
				Name: data.value("name"),
			}
			if userInfo.Name == "" {
				userInfo.Name = userInfo.ID
			}

			uinfoJSON := uinfo{}
			if err := json.Unmarshal(bdata, &uinfoJSON); err == nil {
				userInfo.Picture = uinfoJSON.Picture.Data.URL
			}
			return userInfo
		},
	})
}

// NewDisqus makes disqus oauth2 provider. TODO: WIP - seems to need client_id param
func NewDisqus(p Params) Provider {
	return initProvider(p, Provider{
		Name: "disqus",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://disqus.com/api/oauth/2.0/authorize/",
			TokenURL: "https://disqus.com/api/oauth/2.0/access_token/",
		},
		RedirectURL: p.RemarkURL + "/auth/disqus/callback",
		Scopes:      []string{"read"},
		InfoURL:     "https://disqus.com/api/3.0/users/details.json",
		Store:       p.SessionStore,
		MapUser: func(data userData, _ []byte) store.User {
			userInfo := store.User{
				ID:      "disqus_" + rest.EncodeID("disqus_"+data.value("login")),
				Name:    data.value("name"),
				Picture: data.value("avatar_url"),
			}
			if userInfo.Name == "" {
				userInfo.Name = userInfo.ID
			}
			return userInfo
		},
	})
}
