// Package provider implements all oauth2, oauth1 as well as custom and direct providers
package provider

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
	"golang.org/x/oauth2/yandex"

	"github.com/dghubble/oauth1"
	"github.com/dghubble/oauth1/twitter"

	"github.com/go-pkgz/auth/token"
)

// NewGoogle makes google oauth2 provider
func NewGoogle(p Params) Oauth2Handler {
	return initOauth2Handler(p, Oauth2Handler{
		name:     "google",
		endpoint: google.Endpoint,
		scopes:   []string{"https://www.googleapis.com/auth/userinfo.profile"},
		infoURL:  "https://www.googleapis.com/oauth2/v3/userinfo",
		mapUser: func(data UserData, _ []byte) token.User {
			userInfo := token.User{
				// encode email with provider name to avoid collision if same id returned by other provider
				ID:      "google_" + token.HashID(sha1.New(), data.Value("sub")),
				Name:    data.Value("name"),
				Picture: data.Value("picture"),
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
		name:     "github",
		endpoint: github.Endpoint,
		scopes:   []string{},
		infoURL:  "https://api.github.com/user",
		mapUser: func(data UserData, _ []byte) token.User {
			userInfo := token.User{
				ID:      "github_" + token.HashID(sha1.New(), data.Value("login")),
				Name:    data.Value("name"),
				Picture: data.Value("avatar_url"),
			}
			// github may have no user name, use login in this case
			if userInfo.Name == "" {
				userInfo.Name = data.Value("login")
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
		name:     "facebook",
		endpoint: facebook.Endpoint,
		scopes:   []string{"public_profile"},
		infoURL:  "https://graph.facebook.com/me?fields=id,name,picture",
		mapUser: func(data UserData, bdata []byte) token.User {
			userInfo := token.User{
				ID:   "facebook_" + token.HashID(sha1.New(), data.Value("id")),
				Name: data.Value("name"),
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
		name:     "yandex",
		endpoint: yandex.Endpoint,
		scopes:   []string{},
		// See https://tech.yandex.com/passport/doc/dg/reference/response-docpage/
		infoURL: "https://login.yandex.ru/info?format=json",
		mapUser: func(data UserData, _ []byte) token.User {
			userInfo := token.User{
				ID:   "yandex_" + token.HashID(sha1.New(), data.Value("id")),
				Name: data.Value("display_name"), // using Display Name by default
			}
			if userInfo.Name == "" {
				userInfo.Name = data.Value("real_name") // using Real Name (== full name) if Display Name is empty
			}
			if userInfo.Name == "" {
				userInfo.Name = data.Value("login") // otherwise using login
			}

			if data.Value("default_avatar_id") != "" {
				userInfo.Picture = fmt.Sprintf("https://avatars.yandex.net/get-yapic/%s/islands-200", data.Value("default_avatar_id"))
			}
			return userInfo
		},
	})
}

// NewTwitter makes twitter oauth2 provider
func NewTwitter(p Params) Oauth1Handler {
	return initOauth1Handler(p, Oauth1Handler{
		name: "twitter",
		conf: oauth1.Config{
			Endpoint: twitter.AuthorizeEndpoint,
		},
		infoURL: "https://api.twitter.com/1.1/account/verify_credentials.json",
		mapUser: func(data UserData, _ []byte) token.User {
			userInfo := token.User{
				ID:      "twitter_" + token.HashID(sha1.New(), data.Value("id_str")),
				Name:    data.Value("screen_name"),
				Picture: data.Value("profile_image_url_https"),
			}
			if userInfo.Name == "" {
				userInfo.Name = data.Value("name")
			}
			return userInfo
		},
	})
}

// NewBattlenet makes Battle.net oauth2 provider
func NewBattlenet(p Params) Oauth2Handler {
	return initOauth2Handler(p, Oauth2Handler{
		name: "battlenet",
		endpoint: oauth2.Endpoint{
			AuthURL:   "https://eu.battle.net/oauth/authorize",
			TokenURL:  "https://eu.battle.net/oauth/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		scopes:  []string{},
		infoURL: "https://eu.battle.net/oauth/userinfo",
		mapUser: func(data UserData, _ []byte) token.User {
			userInfo := token.User{
				ID:   "battlenet_" + token.HashID(sha1.New(), data.Value("id")),
				Name: data.Value("battletag"),
			}

			return userInfo
		},
	})
}

// NewMicrosoft makes microsoft azure oauth2 provider
func NewMicrosoft(p Params) Oauth2Handler {
	return initOauth2Handler(p, Oauth2Handler{
		name:     "microsoft",
		endpoint: microsoft.AzureADEndpoint("consumers"),
		scopes:   []string{"User.Read"},
		infoURL:  "https://graph.microsoft.com/v1.0/me",
		// non-beta doesn't provide photo for consumers yet
		// see https://github.com/microsoftgraph/microsoft-graph-docs/issues/3990
		mapUser: func(data UserData, b []byte) token.User {
			userInfo := token.User{
				ID:      "microsoft_" + token.HashID(sha1.New(), data.Value("id")),
				Name:    data.Value("displayName"),
				Picture: "https://graph.microsoft.com/beta/me/photo/$value",
			}
			return userInfo
		},
	})
}
