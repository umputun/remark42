# auth - authentication via oauth2 [![Build Status](https://travis-ci.org/go-pkgz/auth.svg?branch=master)](https://travis-ci.org/go-pkgz/auth) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/auth/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/auth?branch=master)

This library provides "social login" with Github, Google, Facebook and Yandex.  

- Multiple oauth2 providers can be used at the same time
- Special `dev` provider allows local testing and development
- JWT stored in a secure cookie and with XSRF protection. Cookies can be session-only
- Minimal scopes with user name, id and picture (avatar) only
- Integrated avatar proxy with FS, boltdb or gridfs storage
- Support of user-defined storages
- Black list with user-defined validator
- Multiple aud (audience) supported
- Secure key with customizable `SecretReader`
- Ability to store extra information to token and retrieve on login
- Middleware for easy integration into http routers

## Install

`go install github.com/go-pkgz/auth`

## Usage

Example with chi router:

```go
func main() {
	/// define options
	options := auth.Opts{
		SecretReader:   token.SecretFunc(func(id string) (string, error) { return "secret", nil }), // secret key for JWT
		TokenDuration:  time.Hour,
		CookieDuration: time.Hour * 24,
		Issuer:         "my-test-app",
		URL:            "http://127.0.0.1:8080",
		AvatarStore:    avatar.NewLocalFS("/tmp", 120),
		Validator: middleware.ValidatorFunc(func(_ string, claims token.Claims) bool {
			return claims.User != nil && strings.HasPrefix(claims.User.Name, "dev_") // allow only dev_ names
		}),		
	}

	// create auth service
	service, err := auth.NewService(options)
	if err != nil {
		log.Fatal(err)
	}
	service.AddProvider("github", "<Client ID>", "<Client Secret>")   // add github provider
	service.AddProvider("facebook", "<Client ID>", "<Client Secret>") // add facebook provider

	// retrieve auth middleware
	m := service.Middleware()

	// setup http server
	router := chi.NewRouter()
	router.Get("/open", openRouteHandler)                      // open api
	router.With(m.Auth).Get("/private", protectedRouteHandler) // protected api

	// setup auth routes
	authRoutes, avaRoutes := service.Handlers()
	router.Mount("/auth", authRoutes)  // add auth handlers
	router.Mount("/avatar", avaRoutes) // add avatar handler

	log.Fatal(http.ListenAndServe(":8080", router))
}
```

## Middleware

`github.com/go-pkgz/auth/middleware` provides ready-to-use middleware.

- `middleware.Auth` - requires authenticated user
- `middleware.Admin` - requires authenticated and admin user
- `middleware.Trace` - doesn't require authenticated user, but adds user info to request
  
## Register oauth2 providers

Authentication handled by external providers. You should setup oauth2 for all (or some) of them to allow users to authenticate. It is not mandatory to have all of them, but at least one should be correctly configured.

#### Google Auth Provider

1.  Create a new project: https://console.developers.google.com/project
1.  Choose the new project from the top right project dropdown (only if another project is selected)
1.  In the project Dashboard center pane, choose **"API Manager"**
1.  In the left Nav pane, choose **"Credentials"**
1.  In the center pane, choose **"OAuth consent screen"** tab. Fill in **"Product name shown to users"** and hit save.
1.  In the center pane, choose **"Credentials"** tab.
    * Open the **"New credentials"** drop down
    * Choose **"OAuth client ID"**
    * Choose **"Web application"**
    * Application name is freeform, choose something appropriate
    * Authorized origins is your domain ex: `https://example.mysite.com`
    * Authorized redirect URIs is the location of oauth2/callback constructed as domain + `/auth/google/callback`, ex: `https://example.mysite.com/auth/google/callback`
    * Choose **"Create"**
2.  Take note of the **Client ID** and **Client Secret**

_instructions for google oauth2 setup borrowed from [oauth2_proxy](https://github.com/bitly/oauth2_proxy)_

#### GitHub Auth Provider

1.  Create a new **"OAuth App"**: https://github.com/settings/developers
1.  Fill **"Application Name"** and **"Homepage URL"** for your site
1.  Under **"Authorization callback URL"** enter the correct url constructed as domain + `/auth/github/callback`. ie `https://example.mysite.com/auth/github/callback`
1.  Take note of the **Client ID** and **Client Secret**

#### Facebook Auth Provider

1.  From https://developers.facebook.com select **"My Apps"** / **"Add a new App"**
1.  Set **"Display Name"** and **"Contact email"**
1.  Choose **"Facebook Login"** and then **"Web"**
1.  Set "Site URL" to your domain, ex: `https://example.mysite.com`
1.  Under **"Facebook login"** / **"Settings"** fill "Valid OAuth redirect URIs" with your callback url constructed as domain + `/auth/facebook/callback`
1.  Select **"App Review"** and turn public flag on. This step may ask you to provide a link to your privacy policy.

#### Yandex Auth Provider

1.  Create a new **"OAuth App"**: https://oauth.yandex.com/client/new
1.  Fill **"App name"** for your site
1.  Under **Platforms** select **"Web services"** and enter **"Callback URI #1"** constructed as domain + `/auth/yandex/callback`. ie `https://example.mysite.com/auth/yandex/callback`
1.  Select **Permissions**. You need following permissions only from the **"Yandex.Passport API"** section:
    * Access to user avatar
    * Access to username, first name and surname, gender
1.  Fill out the rest of fields if needed
1.  Take note of the **ID** and **Password**

For more details refer to [Yandex OAuth](https://tech.yandex.com/oauth/doc/dg/concepts/about-docpage/) and [Yandex.Passport](https://tech.yandex.com/passport/doc/dg/index-docpage/) API documentation.


## Status 

The library extracted from [remark42](https://github.com/umputun/remark) project. The code in production use on multiple sites and seems to work fine.