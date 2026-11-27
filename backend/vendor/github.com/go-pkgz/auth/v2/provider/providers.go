// Package provider implements all oauth2, oauth1 as well as custom and direct providers
package provider

import (
	"crypto/sha1" //nolint
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dghubble/oauth1"
	"github.com/dghubble/oauth1/twitter"
	"github.com/go-pkgz/auth/v2/logger"
	"github.com/go-pkgz/auth/v2/token"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
	"golang.org/x/oauth2/yandex"
)

// UserAttributes is the type that will be used to map user data from provider to token.User
type UserAttributes map[string]string

// NewGoogle makes google oauth2 provider
func NewGoogle(p Params) Oauth2Handler {
	return initOauth2Handler(p, Oauth2Handler{
		name:     "google",
		endpoint: google.Endpoint,
		scopes:   []string{"https://www.googleapis.com/auth/userinfo.profile"},
		infoURL:  "https://www.googleapis.com/oauth2/v3/userinfo",
		mapUser: func(data UserData, _ []byte) token.User {
			id := data.Value("sub")
			if id == "" {
				return token.User{} // the callback refuses a user without an id
			}
			userInfo := token.User{
				// encode email with provider name to avoid collision if same id returned by other provider
				ID:      "google_" + token.HashID(sha1.New(), id),
				Name:    data.Value("name"),
				Picture: data.Value("picture"),
			}
			if userInfo.Name == "" {
				userInfo.Name = "noname_" + userInfo.ID[8:12]
			}
			for k, v := range p.UserAttributes {
				userInfo.SetStrAttr(v, data.Value(k))
			}
			return userInfo
		},
	})
}

// NewGithub makes github oauth2 provider
func NewGithub(p Params) Oauth2Handler {
	if p.L == nil {
		p.L = logger.NoOp // mapUser below captures p, initOauth2Handler defaults its own copy only
	}
	return initOauth2Handler(p, Oauth2Handler{
		name:     "github",
		endpoint: github.Endpoint,
		scopes:   []string{},
		infoURL:  "https://api.github.com/user",
		mapUser: func(data UserData, bdata []byte) token.User {
			userInfo := token.User{
				Name:    data.Value("name"),
				Picture: data.Value("avatar_url"),
			}
			if login := data.Value("login"); login != "" {
				userInfo.ID = "github_" + token.HashID(sha1.New(), login)
			}
			if p.GithubNumericID {
				// data.Value is not usable here, json numbers decode to float64 and format as "1.345027e+06".
				// the "gid:" prefix keeps numeric ids out of the login hash space, logins may be all-digit
				var uinfoJSON struct {
					ID int64 `json:"id"`
				}
				if err := json.Unmarshal(bdata, &uinfoJSON); err == nil && uinfoJSON.ID != 0 {
					userInfo.ID = "github_" + token.HashID(sha1.New(), "gid:"+strconv.FormatInt(uinfoJSON.ID, 10))
				} else {
					// keep the login-based value, matching the default derivation and its recycling caveat
					p.Logf("[WARN] github numeric id not available, keeping login-based id")
				}
			}
			if userInfo.ID == "" {
				return token.User{} // neither login nor numeric id, the callback refuses a user without an id
			}
			// github may have no user name, use login in this case
			if userInfo.Name == "" {
				userInfo.Name = data.Value("login")
			}
			for k, v := range p.UserAttributes {
				userInfo.SetStrAttr(v, data.Value(k))
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
			id := data.Value("id")
			if id == "" {
				return token.User{} // the callback refuses a user without an id
			}
			userInfo := token.User{
				ID:   "facebook_" + token.HashID(sha1.New(), id),
				Name: data.Value("name"),
			}
			if userInfo.Name == "" {
				userInfo.Name = userInfo.ID[0:16]
			}

			uinfoJSON := uinfo{}
			if err := json.Unmarshal(bdata, &uinfoJSON); err == nil {
				userInfo.Picture = uinfoJSON.Picture.Data.URL
			}
			for k, v := range p.UserAttributes {
				userInfo.SetStrAttr(v, data.Value(k))
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
			id := data.Value("id")
			if id == "" {
				return token.User{} // the callback refuses a user without an id
			}
			userInfo := token.User{
				ID:   "yandex_" + token.HashID(sha1.New(), id),
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
			for k, v := range p.UserAttributes {
				userInfo.SetStrAttr(v, data.Value(k))
			}
			return userInfo
		},
	})
}

// NewTwitter makes twitter (X) oauth1 provider
func NewTwitter(p Params) Oauth1Handler {
	return initOauth1Handler(p, Oauth1Handler{
		name: "twitter",
		conf: oauth1.Config{
			Endpoint: twitter.AuthorizeEndpoint,
		},
		infoURL: "https://api.twitter.com/1.1/account/verify_credentials.json",
		mapUser: func(data UserData, _ []byte) token.User {
			id := data.Value("id_str")
			if id == "" {
				return token.User{} // the callback refuses a user without an id
			}
			userInfo := token.User{
				ID:      "twitter_" + token.HashID(sha1.New(), id),
				Name:    data.Value("screen_name"),
				Picture: data.Value("profile_image_url_https"),
			}
			if userInfo.Name == "" {
				userInfo.Name = data.Value("name")
			}
			for k, v := range p.UserAttributes {
				userInfo.SetStrAttr(v, data.Value(k))
			}
			return userInfo
		},
	})
}

// NewTwitterV2 makes twitter (X) oauth2 provider, it uses X API v2 where NewTwitter uses oauth1 and API v1.1.
// The provider name and the user id derivation are the same as in NewTwitter, register only one of the two.
func NewTwitterV2(p Params) Oauth2Handler {
	return initOauth2Handler(p, Oauth2Handler{
		name: "twitter",
		// see https://docs.x.com/fundamentals/authentication/oauth-2-0/authorization-code
		endpoint: oauth2.Endpoint{ //nolint:gosec // G101 false positive, oauth endpoint urls are not credentials
			AuthURL:  "https://x.com/i/oauth2/authorize",
			TokenURL: "https://api.x.com/2/oauth2/token",
			// a confidential client authenticates with the basic header, without this oauth2 retries
			// a rejected exchange with the secret in the request body
			AuthStyle: oauth2.AuthStyleInHeader,
		},
		scopes:  []string{"users.read", "tweet.read"},
		infoURL: "https://api.x.com/2/users/me?user.fields=profile_image_url",
		mapUser: func(data UserData, _ []byte) token.User {
			// the user object is nested under "data". A response without it, or with an id which is not
			// a string, maps to a user with no id and the callback refuses it
			nested, _ := data["data"].(map[string]any)
			user := UserData(nested)
			id, _ := nested["id"].(string)
			if id == "" {
				return token.User{}
			}
			userInfo := token.User{
				ID:      "twitter_" + token.HashID(sha1.New(), id),
				Name:    user.Value("username"),
				Picture: user.Value("profile_image_url"),
			}
			if userInfo.Name == "" {
				userInfo.Name = user.Value("name")
			}
			for k, v := range p.UserAttributes {
				userInfo.SetStrAttr(v, user.Value(k))
			}
			return userInfo
		},
	})
}

// NewBattlenet makes Battle.net oauth2 provider
func NewBattlenet(p Params) Oauth2Handler {
	return initOauth2Handler(p, Oauth2Handler{
		name: "battlenet",
		endpoint: oauth2.Endpoint{ //nolint:gosec // G101 false positive, oauth endpoint urls are not credentials
			AuthURL:   "https://eu.battle.net/oauth/authorize",
			TokenURL:  "https://eu.battle.net/oauth/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		scopes:  []string{},
		infoURL: "https://eu.battle.net/oauth/userinfo",
		mapUser: func(data UserData, _ []byte) token.User {
			id := data.Value("id")
			if id == "" {
				return token.User{} // the callback refuses a user without an id
			}
			userInfo := token.User{
				ID:   "battlenet_" + token.HashID(sha1.New(), id),
				Name: data.Value("battletag"),
			}
			for k, v := range p.UserAttributes {
				userInfo.SetStrAttr(v, data.Value(k))
			}
			return userInfo
		},
	})
}

// NewMicrosoft makes microsoft azure oauth2 provider
func NewMicrosoft(p Params) Oauth2Handler {
	tenant := p.MicrosoftTenant
	if tenant != "" && (strings.ContainsAny(tenant, "/?# \t\n\r") || strings.Contains(tenant, "..")) {
		if p.L != nil {
			p.Logf("[WARN] invalid microsoft tenant %q, using the common endpoint", tenant)
		}
		tenant = ""
	}
	if tenant == "" {
		tenant = "common"
	}
	return initOauth2Handler(p, Oauth2Handler{
		name:     "microsoft",
		endpoint: microsoft.AzureADEndpoint(tenant),
		scopes:   []string{"User.Read"},
		infoURL:  "https://graph.microsoft.com/v1.0/me",
		// non-beta doesn't provide photo for consumers yet
		// see https://github.com/microsoftgraph/microsoft-graph-docs/issues/3990
		mapUser: func(data UserData, _ []byte) token.User {
			id := data.Value("id")
			if id == "" {
				return token.User{} // the callback refuses a user without an id
			}
			userInfo := token.User{
				ID:      "microsoft_" + token.HashID(sha1.New(), id),
				Name:    data.Value("displayName"),
				Picture: "https://graph.microsoft.com/beta/me/photo/$value",
			}
			for k, v := range p.UserAttributes {
				userInfo.SetStrAttr(v, data.Value(k))
			}
			return userInfo
		},
	})
}

// NewPatreon makes patreon oauth2 provider
func NewPatreon(p Params) Oauth2Handler {
	type uinfo struct {
		Data struct {
			Attributes struct {
				FullName string `json:"full_name"`
				ImageURL string `json:"image_url"`
			} `json:"attributes"`
			ID            string `json:"id"`
			Relationships struct {
				Pledges struct {
					Data []struct {
						ID   string `json:"id"`
						Type string `json:"type"`
					} `json:"data"`
				} `json:"pledges"`
			} `json:"relationships"`
		} `json:"data"`
	}

	return initOauth2Handler(p, Oauth2Handler{
		name: "patreon",
		// see https://docs.patreon.com/?shell#oauth
		endpoint: oauth2.Endpoint{ //nolint:gosec // G101 false positive, oauth endpoint urls are not credentials
			AuthURL:   "https://www.patreon.com/oauth2/authorize",
			TokenURL:  "https://api.patreon.com/oauth2/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		scopes:  []string{},
		infoURL: "https://www.patreon.com/api/oauth2/api/current_user",

		mapUser: func(data UserData, bdata []byte) token.User {
			uinfoJSON := uinfo{}
			if err := json.Unmarshal(bdata, &uinfoJSON); err != nil || uinfoJSON.Data.ID == "" {
				return token.User{} // the callback refuses a user without an id
			}
			userInfo := token.User{
				ID:      "patreon_" + token.HashID(sha1.New(), uinfoJSON.Data.ID),
				Name:    uinfoJSON.Data.Attributes.FullName,
				Picture: uinfoJSON.Data.Attributes.ImageURL,
			}

			// check if the user is your subscriber
			if len(uinfoJSON.Data.Relationships.Pledges.Data) > 0 {
				userInfo.SetPaidSub(true)
			}
			// the user object is nested as data.attributes, with the id in data; a key is looked up
			// in attributes first, then in data
			nested, _ := data["data"].(map[string]any)
			attrs, _ := nested["attributes"].(map[string]any)
			for k, v := range p.UserAttributes {
				if _, ok := attrs[k]; ok {
					userInfo.SetStrAttr(v, UserData(attrs).Value(k))
					continue
				}
				userInfo.SetStrAttr(v, UserData(nested).Value(k))
			}
			return userInfo
		},
	})
}

// NewDiscord makes discord oauth2 provider
func NewDiscord(p Params) Oauth2Handler {
	return initOauth2Handler(p, Oauth2Handler{
		name: "discord",
		// see https://discord.com/developers/docs/topics/oauth2
		endpoint: oauth2.Endpoint{ //nolint:gosec // G101 false positive, oauth endpoint urls are not credentials
			AuthURL:  "https://discord.com/oauth2/authorize",
			TokenURL: "https://discord.com/api/oauth2/token",
		},
		infoURL: "https://discord.com/api/v10/users/@me",
		scopes:  []string{"identify"},
		mapUser: func(data UserData, _ []byte) token.User {
			id := data.Value("id")
			if id == "" {
				return token.User{} // the callback refuses a user without an id
			}
			userInfo := token.User{
				ID:   "discord_" + token.HashID(sha1.New(), id),
				Name: data.Value("username"),
			}
			if data.Value("avatar") != "" { // a user without an avatar has none to link to
				userInfo.Picture = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.webp", id, data.Value("avatar"))
			}

			for k, v := range p.UserAttributes {
				userInfo.SetStrAttr(v, data.Value(k))
			}
			return userInfo
		},
	})
}
