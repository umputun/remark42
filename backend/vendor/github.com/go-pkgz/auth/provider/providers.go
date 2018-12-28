package provider

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/yandex"

	"github.com/go-pkgz/auth/token"
)

// NewGoogle makes google oauth2 provider
func NewGoogle(p Params) Service {
	return initService(p, Service{
		Name:        "google",
		Endpoint:    google.Endpoint,
		RedirectURL: p.URL + "/token/google/callback",
		Scopes:      []string{"https://www.googleapis.com/token/userinfo.profile"},
		InfoURL:     "https://www.googleapis.com/oauth2/v3/userinfo",
		MapUser: func(data userData, _ []byte) token.User {
			userInfo := token.User{
				// encode email with provider name to avoid collision if same id returned by other provider
				ID:      "google_" + token.HashID(sha1.New(), data.value("sub")),
				Name:    data.value("name"),
				Picture: data.value("picture"),
			}
			if userInfo.Name == "" {
				userInfo.Name = "noname_" + userInfo.ID[8:12]
			}
			return userInfo
		},
	})
}

// NewGithub makes github oauth2 provider
func NewGithub(p Params) Service {
	return initService(p, Service{
		Name:        "github",
		Endpoint:    github.Endpoint,
		RedirectURL: p.URL + "/token/github/callback",
		Scopes:      []string{},
		InfoURL:     "https://api.github.com/user",
		MapUser: func(data userData, _ []byte) token.User {
			userInfo := token.User{
				ID:      "github_" + token.HashID(sha1.New(), data.value("login")),
				Name:    data.value("name"),
				Picture: data.value("avatar_url"),
			}
			// github may have no user name, use login in this case
			if userInfo.Name == "" {
				userInfo.Name = data.value("login")
			}
			return userInfo
		},
	})
}

// NewFacebook makes facebook oauth2 provider
func NewFacebook(p Params) Service {

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

	return initService(p, Service{
		Name:        "facebook",
		Endpoint:    facebook.Endpoint,
		RedirectURL: p.URL + "/token/facebook/callback",
		Scopes:      []string{"public_profile"},
		InfoURL:     "https://graph.facebook.com/me?fields=id,name,picture",
		MapUser: func(data userData, bdata []byte) token.User {
			userInfo := token.User{
				ID:   "facebook_" + token.HashID(sha1.New(), data.value("id")),
				Name: data.value("name"),
			}
			if userInfo.Name == "" {
				userInfo.Name = userInfo.ID[0:16]
			}

			uinfoJSON := uinfo{}
			if err := json.Unmarshal(bdata, &uinfoJSON); err == nil {
				userInfo.Picture = uinfoJSON.Picture.Data.URL
			}
			return userInfo
		},
	})
}

// NewYandex makes yandex oauth2 provider
func NewYandex(p Params) Service {
	return initService(p, Service{
		Name:        "yandex",
		Endpoint:    yandex.Endpoint,
		RedirectURL: p.URL + "/token/yandex/callback",
		Scopes:      []string{},
		// See https://tech.yandex.com/passport/doc/dg/reference/response-docpage/
		InfoURL: "https://login.yandex.ru/info?format=json",
		MapUser: func(data userData, _ []byte) token.User {
			userInfo := token.User{
				ID:   "yandex_" + token.HashID(sha1.New(), data.value("id")),
				Name: data.value("display_name"), // using Display Name by default
			}
			if userInfo.Name == "" {
				userInfo.Name = data.value("real_name") // using Real Name (== full name) if Display Name is empty
			}
			if userInfo.Name == "" {
				userInfo.Name = data.value("login") // otherwise using login
			}

			if data.value("default_avatar_id") != "" {
				userInfo.Picture = fmt.Sprintf("https://avatars.yandex.net/get-yapic/%s/islands-200", data.value("default_avatar_id"))
			}
			return userInfo
		},
	})
}
