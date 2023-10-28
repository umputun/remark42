# auth - authentication via oauth2, direct and email
[![Build Status](https://github.com/go-pkgz/auth/workflows/build/badge.svg)](https://github.com/go-pkgz/auth/actions) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/auth/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/auth?branch=master) [![godoc](https://godoc.org/github.com/go-pkgz/auth?status.svg)](https://pkg.go.dev/github.com/go-pkgz/auth?tab=doc)

This library provides "social login" with Github, Google, Facebook, Microsoft, Twitter, Yandex, Battle.net, Apple, Patreon and Telegram as well as custom auth providers and email verification.

- Multiple oauth2 providers can be used at the same time
- Special `dev` provider allows local testing and development
- JWT stored in a secure cookie with XSRF protection. Cookies can be session-only
- Minimal scopes with user name, id and picture (avatar) only
- Direct authentication with user's provided credential checker
- Verified authentication with user's provided sender (email, im, etc)
- Custom oauth2 server and ability to use any third party provider
- Integrated avatar proxy with an FS, boltdb and gridfs storage
- Support of user-defined storage for avatars
- Identicon for default avatars
- Black list with user-defined validator
- Multiple aud (audience) supported
- Secure key with customizable `SecretReader`
- Ability to store an extra information to token and retrieve on login
- Pre-auth and post-auth hooks to handle custom use cases.
- Middleware for easy integration into http routers
- Wrappers to extract user info from the request
- Role based access control

## Install

`go get -u github.com/go-pkgz/auth`

## Usage

Example with chi router:

```go

func main() {
	// define options
	options := auth.Opts{
		SecretReader: token.SecretFunc(func(id string) (string, error) { // secret key for JWT
			return "secret", nil
		}),
		TokenDuration:  time.Minute * 5, // token expires in 5 minutes
		CookieDuration: time.Hour * 24,  // cookie expires in 1 day and will enforce re-login
		Issuer:         "my-test-app",
		URL:            "http://127.0.0.1:8080",
		AvatarStore:    avatar.NewLocalFS("/tmp"),
		Validator: token.ValidatorFunc(func(_ string, claims token.Claims) bool {
			// allow only dev_* names
			return claims.User != nil && strings.HasPrefix(claims.User.Name, "dev_")
		}),
	}

	// create auth service with providers
	service := auth.NewService(options)
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
- `middleware.Admin` - requires authenticated admin user
- `middleware.Trace` - doesn't require authenticated user, but adds user info to request
- `middleware.RBAC` - requires authenticated user with passed role(s)

Also, there is a special middleware `middleware.UpdateUser` for population and modifying UserInfo in every request. See "Customization" for more details.

## Details

Generally, adding support of `auth` includes a few relatively simple steps:

1. Setup `auth.Opts` structure with all parameters. Each of them [documented](https://github.com/go-pkgz/auth/blob/master/auth.go#L29) and most of parameters are optional and have sane defaults.
2. [Create](https://github.com/go-pkgz/auth/blob/master/auth.go#L56) the new `auth.Service` with provided options.
3. [Add all](https://github.com/go-pkgz/auth/blob/master/auth.go#L149) desirable authentication providers.
4. Retrieve [middleware](https://github.com/go-pkgz/auth/blob/master/auth.go#L144) and [http handlers](https://github.com/go-pkgz/auth/blob/master/auth.go#L105) from `auth.Service`
5. Wire auth and avatar handlers into http router as sub–routes.

### API

For the example above authentication handlers wired as `/auth` and provides:

- `/auth/<provider>/login?site=<site_id>&from=<redirect_url>` - site_id used as `aud` claim for the token and can be processed by `SecretReader` to load/retrieve/define different secrets. redirect_url is the url to redirect after successful login.
- `/avatar/<avatar_id>` - returns the avatar (image). Links to those pictures added into user info automatically, for details see "Avatar proxy"
- `/auth/<provider>/logout` and `/auth/logout` - invalidate "session" by removing JWT cookie
- `/auth/list` - gives a json list of active providers
- `/auth/user` - returns `token.User` (json)
- `/auth/status` - returns status of logged in user (json)

### User info

Middleware populates `token.User` to request's context. It can be loaded with `token.GetUserInfo(r *http.Request) (user User, err error)` or `token.MustGetUserInfo(r *http.Request) User` functions.

`token.User` object includes all fields retrieved from oauth2 provider:
- `Name` - user name
- `ID` - hash of user id
- `Picture` - full link to proxied avatar (see "Avatar proxy")

It also has placeholders for fields application can populate with custom `token.ClaimsUpdater` (see "Customization")

- `IP`  - hash of user's IP address
- `Email` - user's email
- `Attributes` - map of string:any-value. To simplify management of this map some setters and getters provided, for example `users.StrAttr`, `user.SetBoolAttr` and so on. See [user.go](https://github.com/go-pkgz/auth/blob/master/token/user.go) for more details.

### Avatar proxy

Direct links to avatars won't survive any real-life usage if they linked from a public page. For example, page [like this](https://remark42.com/demo/) may have hundreds of avatars and, most likely, will trigger throttling on provider's side. To eliminate such restriction `auth` library provides an automatic proxy

- On each login the proxy will retrieve user's picture and save it to `AvatarStore`
- Local (proxied) link to avatar included in user's info (jwt token)
- API for avatar removal provided as a part of `AvatarStore`
- User can leverage one of the provided stores:
    - `avatar.LocalFS` - file system, each avatar in a separate file
    - `avatar.BoltDB`  - single [boltdb](https://go.etcd.io/bbolt) file (embedded KV store).
    - `avatar.GridFS` - external [GridFS](https://docs.mongodb.com/manual/core/gridfs/) (mongo db).
- In case of need custom implementations of other stores can be passed in and used by `auth` library. Each store has to implement `avatar.Store` [interface](https://github.com/go-pkgz/auth/blob/master/avatar/store.go#L25).
- All avatar-related setup done as a part of `auth.Opts` and needs:
    - `AvatarStore` - avatar store to use, i.e. `avatar.NewLocalFS("/tmp/avatars")` or more generic `avatar.NewStore(uri)`
        - file system uri - `file:///tmp/location` or just `/tmp/location`
        - boltdb - `bolt://tmp/avatars.bdb`
        - mongo - `"mongodb://127.0.0.1:27017/test?ava_db=db1&ava_coll=coll1`
    - `AvatarRoutePath` - route prefix for direct links to proxied avatar. For example `/api/v1/avatars` will make full links like this - `http://example.com/api/v1/avatars/1234567890123.image`. The url will be stored in user's token and retrieved by middleware (see "User Info")
    - `AvatarResizeLimit` - size (in pixels) used to resize the avatar. Pls note - resize happens once as a part of `Put` call, i.e. on login. 0 size (default) disables resizing.

### Direct authentication

In addition to oauth2 providers `auth.Service` allows to use direct user-defined authentication. This is done by adding direct provider with `auth.AddDirectProvider`.

```go
	service.AddDirectProvider("local", provider.CredCheckerFunc(func(user, password string) (ok bool, err error) {
		ok, err := checkUserSomehow(user, password)
		return ok, err
	}))
```

Such provider acts like any other, i.e. will be registered as `/auth/local/login`.

The API for this provider supports both GET and POST requests:

* POST request could be encoded as application/x-www-form-urlencoded or application/json:
  ```
  POST /auth/<name>/login?session=[1|0]
  body: application/x-www-form-urlencoded
  user=<user>&passwd=<password>&aud=<site_id>
  ```
  ```
  POST /auth/<name>/login?session=[1|0]
  body: application/json
  {
    "user": "name",
    "passwd": "xyz",
    "aud": "bar",
  }
  ```
* GET request with user credentials provided as query params, but be aware that [the https query string is not secure](https://stackoverflow.com/a/323286/633961):
  ```
  GET /auth/<name>/login?user=<user>&passwd=<password>&aud=<site_id>&session=[1|0]
  ```

_note: password parameter doesn't have to be naked/real password and can be any kind of password hash prepared by caller._

### Verified authentication

Another non-oauth2 provider allowing user-confirmed authentication, for example by email or slack or telegram. This is
done by adding confirmed provider with `auth.AddVerifProvider`.

```go
    msgTemplate := "Confirmation email, token: {{.Token}}"
	service.AddVerifProvider("email", msgTemplate, sender)
```

Message template may use the follow elements:

- `{{.Address}}` - user address, for example email
- `{{.User}}` - user name
- `{{.Token}}` - confirmation token
- `{{.Site}}` - site ID

Sender should be provided by end-user and implements a single function interface

```go
type Sender interface {
	Send(address string, text string) error
}
```

For convenience a functional wrapper `SenderFunc` provided. Email sender provided in `provider/sender` package and can be
used as `Sender`.

The API for this provider:

 - `GET /auth/<name>/login?user=<user>&address=<adsress>&aud=<site_id>&from=<url>` - send confirmation request to user
 - `GET /auth/<name>/login?token=<conf.token>&sess=[1|0]` - authorize with confirmation token

The provider acts like any other, i.e. will be registered as `/auth/email/login`.

### Email

For email notify provider, please use `github.com/go-pkgz/auth/provider/sender` package:
```go
    sndr := sender.NewEmailClient(sender.EmailParams{
        Host:         "email.hostname",
        Port:         567,
        SMTPUserName: "username",
        SMTPPassword: "pass",
        StartTLS:     true,
        From:         "notify@email.hostname",
        Subject:      "subject",
        ContentType:  "text/html",
        Charset:      "UTF-8",
    }, log.Default())
    authenticator.AddVerifProvider("email", "template goes here", sndr)
```

See [that documentation](https://github.com/go-pkgz/email#options) for full options list.

### Telegram

Telegram provider allows your users to log in with Telegram account. First, you will need to create your bot.
Contact [@BotFather](https://t.me/botfather) and follow his instructions to create your own bot (call it, for example, "My site auth bot")

Next initialize TelegramHandler with following parameters:
* `ProviderName` - Any unique name to distinguish between providers
* `SuccessMsg` - Message sent to user on successfull authentication
* `ErrorMsg` - Message sent on errors (e.g. login request expired)
* `Telegram` - Telegram API implementation. Use provider.NewTelegramAPI with following arguments
	1. The secret token bot father gave you
	2. An http.Client for accessing Telegram API's

```go
token := os.Getenv("TELEGRAM_TOKEN")

telegram := provider.TelegramHandler{
	ProviderName: "telegram",
	ErrorMsg:     "❌ Invalid auth request. Please try clicking link again.",
	SuccessMsg:   "✅ You have successfully authenticated!",
	Telegram:     provider.NewTelegramAPI(token, http.DefaultClient),

	L:            log.Default(),
	TokenService: service.TokenService(),
	AvatarSaver:  service.AvatarProxy(),
}
```

After that run provider and register it's handlers:
```go
// Run Telegram provider in the background
go func() {
	err := telegram.Run(context.Background())
	if err != nil {
		log.Fatalf("[PANIC] failed to start telegram: %v", err)
	}
}()

// Register Telegram provider
service.AddCustomHandler(&telegram)
```

Now all your users have to do is click one of the following links and press **start**
`tg://resolve?domain=<botname>&start=<token>` or `https://t.me/<botname>/?start=<token>`

Use the following routes to interact with provider:
1. `/auth/<providerName>/login` - Obtain auth token. Returns JSON object with `bot` (bot username) and `token` (token itself) fields.
2. `/auth/<providerName>/login?token=<token>` - Check if auth request has been confirmed (i.e. user pressed start). Sets session cookie and returns user info on success, errors with 404 otherwise.

3. `/auth/<providerName>/logout` - Invalidate user session.

### Custom oauth2

This provider brings two extra functions:

1. Adds ability to use any third-party oauth2 providers in addition to the list of directly supported. Included [example](https://github.com/go-pkgz/auth/blob/master/_example/main.go#L113) demonstrates how to do it for bitbucket.
In order to add a new oauth2 provider following input is required:
	* `Name` - any name is allowed except the names from list of supported providers. It is possible to register more than one client for one given oauth2 provider (for example using different names `bitbucket_dev` and `bitbucket_prod`)
	* `Client` - ID and secret of client
	* `Endpoint` - auth URL and token URL. This information could be obtained from auth2 provider page
	* `InfoURL` - oauth2 provider API method to read information of logged in user. This method could be found in documentation of oauth2 provider (e.g. for bitbucket https://developer.atlassian.com/bitbucket/api/2/reference/resource/user)
	* `MapUserFn` - function to convert the response from `InfoURL` to `token.User` (s. example below)
	* `Scopes` - minimal needed scope to read user information. Client should be authorized to these scopes
	```go
	c := auth.Client{
		Cid:     os.Getenv("AEXMPL_BITBUCKET_CID"),
		Csecret: os.Getenv("AEXMPL_BITBUCKET_CSEC"),
	}

	service.AddCustomProvider("bitbucket", c, provider.CustomHandlerOpt{
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://bitbucket.org/site/oauth2/authorize",
			TokenURL: "https://bitbucket.org/site/oauth2/access_token",
		},
		InfoURL: "https://api.bitbucket.org/2.0/user/",
		MapUserFn: func(data provider.UserData, _ []byte) token.User {
			userInfo := token.User{
				ID: "bitbucket_" + token.HashID(sha1.New(),
					data.Value("username")),
				Name: data.Value("nickname"),
			}
			return userInfo
		},
		Scopes: []string{"account"},
	})
	```
2.  Adds local oauth2 server user can fully customize. It uses [`gopkg.in/oauth2.v3`](https://github.com/go-oauth2/oauth2) library and example shows how [to initialize](https://github.com/go-pkgz/auth/blob/master/_example/main.go#L227) the server and [setup a provider](https://github.com/go-pkgz/auth/blob/master/_example/main.go#L100).
	*  to start local oauth2 server following options are required:
		* `URL` - url of oauth2 server with port
		* `WithLoginPage` - flag to define whether login page should be shown
		* `LoginPageHandler` - function to handle login request. If not specified default login page will be shown
		```go
		sopts := provider.CustomServerOpt{
			URL:           "http://127.0.0.1:9096",
			L:             options.Logger,
			WithLoginPage: true,
		}
		prov := provider.NewCustomServer(srv, sopts)

		// Start server
		go prov.Run(context.Background())
		```
	* to register handler for local oauth2 following option are required:
		* `Name` - any name except the names from list of supported providers
		* `Client` - ID and secret of client
		* `HandlerOpt` - handler options of custom oauth provider
		```go
		service.AddCustomProvider("custom123", auth.Client{Cid: "cid", Csecret: "csecret"}, prov.HandlerOpt)
		```

### Self-implemented auth handler
Additionally it is possible to implement own auth handler. It may be useful if auth provider does not conform to oauth standard. Self-implemented handler has to implement `provider.Provider` interface.
```go
// customHandler implements provider.Provider interface
c := customHandler{}

// add customHandler to stack of auth handlers
service.AddCustomHandler(c)
```

### Customization

There are several ways to adjust functionality of the library:

1. `SecretReader` - interface with a single method `Get(aud string) string` to return the secret used for JWT signing and verification
1. `ClaimsUpdater` - interface with `Update(claims Claims) Claims` method. This is the primary way to alter a token at login time and add any attributes, set ip, email, admin status, roles and so on.
1. `Validator` - interface with `Validate(token string, claims Claims) bool` method. This is post-token hook and will be called on **each request** wrapped with `Auth` middleware. This will be the place for special logic to reject some tokens or users.
1. `UserUpdater` - interface with `Update(claims token.User) token.User` method.  This method will be called on **each request** wrapped with `UpdateUser` middleware. This will be the place for special logic modify User Info in request context. [Example of usage.](https://github.com/go-pkgz/auth/blob/19c1b6d26608494955a4480f8f6165af85b1deab/_example/main.go#L189)

All of the interfaces above have corresponding Func adapters - `SecretFunc`, `ClaimsUpdFunc`, `ValidatorFunc` and `UserUpdFunc`.

### Implementing black list logic or some other filters

Restricting some users or some tokens is two step process:

- `ClaimsUpdater` sets an attribute, like `blocked` (or `allowed`)
- `Validator` checks the attribute and returns true/false

_This technique used in the [example](https://github.com/go-pkgz/auth/blob/master/_example/main.go#L56) code_

The process can be simplified by doing all checks directly in `Validator`, but depends on particular case such solution
can be too expensive because `Validator` runs on each request as a part of auth middleware. In contrast, `ClaimsUpdater` called on token creation/refresh only.

### Multi-tenant services and support for different audiences

For complex systems a single authenticator may serve multiple distinct subsystems or multiple set of independent users. For example some SaaS offerings may need to provide different authentications for different customers and prevent use of tokens/cookies made by another customer.

Such functionality can be implemented in 3 different ways:

- Different instances of `auth.Service` each one with different secret. Doing this way will ensure the highest level of isolation and cookies/tokens won't be even parsable across the instances. Practically such architecture can be too complicated and not always possible.
– Handling "allowed audience" as a part of `ClaimsUpdater` and `Validator` chain. I.e. `ClaimsUpdater` sets a claim indicating expected audience code/id and `Validator` making sure it matches. This way a single `auth.Service` could handle multiple groups of auth tokens and reject some based on the audience.
- Using the standard JWT `aud` claim. This method conceptually very similar to the previous one, but done by library internally and consumer don't need to define special  `ClaimsUpdater` and `Validator` logic.

In order to allow `aud` support the list of allowed audiences should be passed in as `opts.Audiences` parameter. Non-empty value will trigger internal checks for token generation (will reject token creation for alien `aud`) as well as `Auth` middleware.

### Dev provider

Working with oauth2 providers can be a pain, especially during development phase. A special, development-only provider `dev` can make it less painful. This one can be registered directly, i.e. `service.AddProvider("dev", "", "")` or `service.AddDevProvider(port)` and should be activated like this:

```go
	// runs dev oauth2 server on :8084 by default
	go func() {
		devAuthServer, err := service.DevAuth()
		if err != nil {
			log.Fatal(err)
		}
		devAuthServer.Run()
	}()
```

It will run fake aouth2 "server" on port :8084 and user could login with any user name. See [example](https://github.com/go-pkgz/auth/blob/master/_example/main.go) for more details.

_Warning: this is not the real oauth2 server but just a small fake thing for development and testing only. Don't use `dev` provider with any production code._

By default, Dev provider doesn't return `email` claim from `/user` endpoint, to match behaviour of other providers which only request minimal scopes.
However sometimes it is useful to have `email` included into user info. This can be done by configuring `devAuthServer.GetEmailFn` function:

```go
    go func() {
		devAuthServer, err := service.DevAuth()
		devOauth2Srv.GetEmailFn = func(username string) string {
			return username + "@example.com"
		}
		if err != nil {
			log.Fatal(err)
		}
		devAuthServer.Run()
	}()
```

### Other ways to authenticate

In addition to the primary method (i.e. JWT cookie with XSRF header) there are two more ways to authenticate:

1. Send JWT header as `X-JWT`. This shouldn't be used for web application, however can be helpful for service-to-service authentication.
2. Send JWT token as query parameter, i.e. `/something?token=<jwt>`
3. Basic access authentication, for more details see below [Basic authentication](#basic-authentication).

### Basic authentication

In some cases the `middleware.Authenticator` allow use  [Basic access authentication](https://en.wikipedia.org/wiki/Basic_access_authentication), which transmits credentials as user-id/password pairs, encoded using Base64 ([RFC7235](https://tools.ietf.org/html/rfc7617)).
When basic authentication used, client doesn't get auth token in response. It's auth type expect credentials in a header `Authorization` at every client request. It can be helpful, if client side not support cookie/token store (e.g. embedded device or custom apps).
This mode disabled by default and will be enabled with options.

The `auth` package has two options of basic authentication:
- simple basic auth will be enabled if `Opts.AdminPasswd` defined. This will allow access with basic auth admin:<Opts.AdminPasswd> with user [admin](https://github.com/go-pkgz/auth/blob/master/middleware/auth.go#L24). Such method can be used for automation scripts.
- basic auth with custom checker [function](https://github.com/go-pkgz/auth/blob/master/middleware/auth.go#L47), which allow adding user data from store to context of request.  It will be enabled if `Opts.BasicAuthChecker` defined. When `BasicAuthChecker` defined then `Opts.AdminPasswd` option will be ignore.
```go
options := auth.Opts{
   //...
   AdminPasswd:    "admin_secret_password", // will ignore if BasicAuthChecker defined
   BasicAuthChecker: func(user, passwd string) (bool, token.User, error) {
      if user == "basic_user" && passwd == "123456" {
           return true, token.User{Name: user, Role: "test_r"}, nil
      }
      return false, token.User{}, errors.New("basic auth credentials check failed")
   }
   //...
}
```
### Logging

By default, this library doesn't print anything to stdout/stderr, however user can pass a logger implementing `logger.L` interface with a single method `Logf(format string, args ...interface{})`. Functional adapter for this interface included as `logger.Func`. There are two predefined implementations in the `logger` package - `NoOp` (prints nothing, default) and `Std` wrapping `log.Printf` from stdlib.

## Register oauth2 providers

Authentication handled by external providers. You should setup oauth2 for all (or some) of them to allow users to authenticate. It is not mandatory to have all of them, but at least one should be correctly configured.

#### Google Auth Provider

1.  Create a new project: https://console.developers.google.com/project
2.  Choose the new project from the top right project dropdown (only if another project is selected)
3.  In the project Dashboard center pane, choose **"API Manager"**
4.  In the left Nav pane, choose **"Credentials"**
5. In the center pane, choose the **"OAuth consent screen"** tab.
  * Select "**External**" and click "Create"
  * Fill in **"App name"** and select **User support email**
  * Upload a logo, if you want to
  * In the **App Domain** section:
    * **Application home page** - your site URL, e.g., `https://mysite.com`
    * **Application privacy policy link** - `/web/privacy.html` of your Remark42 installation, e.g. `https://remark42.mysite.com/web/privacy.html` (please check that it works)
    * **Terms of service** - leave empty
  * **Authorized domains** - your site domain, e.g., `mysite.com`
  * **Developer contact information** - add your email, and then click **Save and continue**
  * On the **Scopes** tab, just click **Save and continue**
  * On the **Test users**, add your email, then click **Save and continue**
  * Before going to the next step, set the app to "Production" and send it to verification
6. In the center pane, choose the **"Credentials"** tab
   * Open the **"Create credentials"** drop-down
   * Choose **"OAuth client ID"**
   * Choose **"Web application"**
   * Application **Name** is freeform; choose something appropriate, like "Comments on mysite.com"
   * **Authorized JavaScript Origins** should be your domain, e.g., `https://remark42.mysite.com`
   * **Authorized redirect URIs** is the location of OAuth2/callback constructed as domain + `/auth/google/callback`, e.g., `https://remark42.mysite.com/auth/google/callback`
   * Click **"Create"**
7.  Take note of the **Client ID** and **Client Secret**

_instructions for google oauth2 setup borrowed from [oauth2_proxy](https://github.com/bitly/oauth2_proxy)_

#### Microsoft Auth Provider

1. Register a new application [using the Azure portal](https://docs.microsoft.com/en-us/graph/auth-register-app-v2).
2. Under **"Authentication/Platform configurations/Web"** enter the correct url constructed as domain + `/auth/microsoft/callback`. i.e. `https://example.mysite.com/auth/microsoft/callback`
3. In "Overview" take note of the **Application (client) ID**
4. Choose the new project from the top right project dropdown (only if another project is selected)
5.  Select "Certificates & secrets" and click on "+ New Client Secret".


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


#### Apple Auth Provider

To configure this provider, a user requires an Apple developer account (without it setting up a sign in with Apple is impossible). [Sign in with Apple](https://developer.apple.com/documentation/sign_in_with_apple/sign_in_with_apple_rest_api) lets users log in to your app using their two-factor authentication Apple ID.

Follow to next steps for configuring on the Apple side:

1. Log in [to the developer account](https://developer.apple.com/account).
1. If you don't have an App ID yet, [create one](https://developer.apple.com/account/resources/identifiers/add/bundleId). Later on, you'll need **TeamID**, which is an "App ID Prefix" value.
1. Enable the "Sign in with Apple" capability for your App ID in [the Certificates, Identifiers & Profiles](https://developer.apple.com/account/resources/identifiers/list) section.
1. Create [Service ID](https://developer.apple.com/account/resources/identifiers/list/serviceId) and bind with App ID from the previous step. Apple will display the description field value to end-users on sign-in. You'll need that service **Identifier as a ClientID** later on**.**
1. Configure "Sign in with Apple" for created Service ID. Add domain where you will use that auth on to "Domains and subdomains" and its main page URL (like `https://example.com/` to "Return URLs".
1. Register a [New Key](https://developer.apple.com/account/resources/authkeys/list) (private key) for the "Sign in with Apple" feature and download it. Write down the **Key ID**. This key will be used to create [JWT](https://developer.apple.com/documentation/sign_in_with_apple/generate_and_validate_tokens#3262048) Client Secret.
1. Add your domain name and sender email in the Certificates, Identifiers & Profiles >> [More](https://developer.apple.com/account/resources/services/configure) section as a new Email Source.

After completing the previous steps, you can proceed with configuring the Apple auth provider. Here are the parameters for AppleConfig:

- _ClientID_ (**required**) - Service ID identifier which is used for Sign with Apple
- _TeamID_ (**required**) - Identifier a developer account (use as prefix for all App ID)
- _KeyID_ (**required**) - Identifier a generated key for Sign with Apple
- _ResponseMode_  - Response Mode, please see [documentation](https://developer.apple.com/documentation/sign_in_with_apple/request_an_authorization_to_the_sign_in_with_apple_server?changes=_1_2#4066168) for reference, default is `form_post`

```go
    // apple config parameters
	appleCfg := provider.AppleConfig{
		TeamID:   os.Getenv("AEXMPL_APPLE_TID"), // developer account identifier
		ClientID: os.Getenv("AEXMPL_APPLE_CID"), // service identifier
		KeyID:    os.Getenv("AEXMPL_APPLE_KEYID"), // private key identifier
	}
```

Then add an Apple provider that accepts the following parameters:
* `appleConfig (provider.AppleConfig)`  created above
* `privateKeyLoader (PrivateKeyLoaderInterface)`

`PrivateKeyLoaderInterface` [implements](https://github.com/go-pkgz/auth/blob/master/provider/apple.go#L98:L100) a loader for the private key (which you downloaded above) to create a `client_secret`. The user can use a pre-defined function `provider.LoadApplePrivateKeyFromFile(filePath string)` to load the private key from local file.

`AddAppleProvide` tries to load private key at call and return an error if load failed. Always check error when calling this provider.

```go
    if err := service.AddAppleProvider(appleCfg, provider.LoadApplePrivateKeyFromFile("PATH_TO_PRIVATE_KEY_FILE")); err != nil {
		log.Fatalf("[ERROR] failed create to AppleProvider: %v", err)
	}
```

**Limitation:**

* Map a userName (if specific scope defined) is only sent in the upon initial user sign up.
  Subsequent logins to your app using Sign In with Apple with the same account do not share any user info and will only return a user identifier in IDToken claims.
  This behaves correctly until a user delete sign in for you service with Apple ID in own Apple account profile (security section).
  It is recommend that you securely cache the at first login containing the user info for bind it with a user UID at next login.
  Provider always get user `UID` (`sub` claim) in `IDToken`.

* Apple doesn't have an API for fetch avatar and user info.

See [example](https://github.com/go-pkgz/auth/blob/master/_example/main.go#L83:L93) before use.

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

##### Battle.net Auth Provider

1. Log into Battle.net as a developer: https://develop.battle.net/nav/login-redirect
1.  Click "+ CREATE CLIENT" https://develop.battle.net/access/clients/create
1. For "Client name", enter whatever you want
1. For "Redirect URLs", one of the lines must be "http\[s\]://your_remark_installation:port//auth/battlenet/callback", e.g. https://localhost:8443/auth/battlenet/callback or https://remark.mysite.com/auth/battlenet/callback
1. For "Service URL", enter the URL to your site or check "I do not have a service URL for this client." checkbox if you don't have any
1. For "Intended use", describe the application you're developing
1. Click "Save".
1. You can see your client ID and client secret at https://develop.battle.net/access/clients by clicking the client you created

For more details refer to [Complete Guide of Battle.net OAuth API and Login Button](https://hakanu.net/oauth/2017/01/26/complete-guide-of-battle-net-oauth-api-and-login-button/) or [the official Battle.net OAuth2 guide](https://develop.battle.net/documentation/guides/using-oauth)

#### Patreon Auth Provider ####
1.	Create a new Patreon client https://www.patreon.com/portal/registration/register-clients
1.  Fill **"App Name"**, **"Description"**, **"App Category"** and **"Author"** for your site
1.  Under **"Redirect URIs"** enter the correct url constructed as domain + `/auth/patreon/callback`. ie `https://example.mysite.com/auth/patreon/callback`
1.  Take note of the **Client ID** and **Client Secret**

#### Twitter Auth Provider
1.	Create a new twitter application https://developer.twitter.com/en/apps
1.	Fill **App name**  and **Description** and **URL** of your site
1.	In the field **Callback URLs** enter the correct url of your callback handler e.g. https://example.mysite.com/{route}/twitter/callback
1.	Under **Key and tokens** take note of the **Consumer API Key** and **Consumer API Secret key**. Those will be used as `cid` and `csecret`
## Status

The library extracted from [remark42](https://github.com/umputun/remark) project. The original code in production use on multiple sites and seems to work fine.

`go-pkgz/auth` library still in development and until version 1 released some breaking changes possible.
