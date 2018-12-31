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
func NewGoogle(p Params) Oauth2Handler {
	return initOauth2Handler(p, Oauth2Handler{
		name:        "google",
		endpoint:    google.Endpoint,
		redirectURL: p.URL + "/auth/google/callback",
		scopes:      []string{"https://www.googleapis.com/auth/userinfo.profile"},
		infoURL:     "https://www.googleapis.com/oauth2/v3/userinfo",
		mapUser: func(data userData, _ []byte) token.User {
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
func NewGithub(p Params) Oauth2Handler {
	return initOauth2Handler(p, Oauth2Handler{
		name:        "github",
		endpoint:    github.Endpoint,
		redirectURL: p.URL + "/auth/github/callback",
		scopes:      []string{},
		infoURL:     "https://api.github.com/user",
		mapUser: func(data userData, _ []byte) token.User {
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
func NewFacebook(p Params) Oauth2Handler {

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

	return initOauth2Handler(p, Oauth2Handler{
		name:        "facebook",
		endpoint:    facebook.Endpoint,
		redirectURL: p.URL + "/auth/facebook/callback",
		scopes:      []string{"public_profile"},
		infoURL:     "https://graph.facebook.com/me?fields=id,name,picture",
		mapUser: func(data userData, bdata []byte) token.User {
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
func NewYandex(p Params) Oauth2Handler {
	return initOauth2Handler(p, Oauth2Handler{
		name:        "yandex",
		endpoint:    yandex.Endpoint,
		redirectURL: p.URL + "/auth/yandex/callback",
		scopes:      []string{},
		// See https://tech.yandex.com/passport/doc/dg/reference/response-docpage/
		infoURL: "https://login.yandex.ru/info?format=json",
		mapUser: func(data userData, _ []byte) token.User {
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
