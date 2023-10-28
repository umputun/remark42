// Package auth provides "social login" with Github, Google, Facebook, Microsoft, Yandex and Battle.net as well as custom auth providers.
package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-pkgz/rest"

	"github.com/go-pkgz/auth/avatar"
	"github.com/go-pkgz/auth/logger"
	"github.com/go-pkgz/auth/middleware"
	"github.com/go-pkgz/auth/provider"
	"github.com/go-pkgz/auth/token"
)

// Client is a type of auth client
type Client struct {
	Cid     string
	Csecret string
}

// Service provides higher level wrapper allowing to construct everything and get back token middleware
type Service struct {
	logger         logger.L
	opts           Opts
	jwtService     *token.Service
	providers      []provider.Service
	authMiddleware middleware.Authenticator
	avatarProxy    *avatar.Proxy
	issuer         string
	useGravatar    bool
}

// Opts is a full set of all parameters to initialize Service
type Opts struct {
	SecretReader   token.Secret        // reader returns secret for given site id (aud), required
	ClaimsUpd      token.ClaimsUpdater // updater for jwt to add/modify values stored in the token
	SecureCookies  bool                // makes jwt cookie secure
	TokenDuration  time.Duration       // token's TTL, refreshed automatically
	CookieDuration time.Duration       // cookie's TTL. This cookie stores JWT token

	DisableXSRF bool // disable XSRF protection, useful for testing/debugging
	DisableIAT  bool // disable IssuedAt claim

	// optional (custom) names for cookies and headers
	JWTCookieName   string        // default "JWT"
	JWTCookieDomain string        // default empty
	JWTHeaderKey    string        // default "X-JWT"
	XSRFCookieName  string        // default "XSRF-TOKEN"
	XSRFHeaderKey   string        // default "X-XSRF-TOKEN"
	JWTQuery        string        // default "token"
	SendJWTHeader   bool          // if enabled send JWT as a header instead of cookie
	SameSiteCookie  http.SameSite // limit cross-origin requests with SameSite cookie attribute

	Issuer string // optional value for iss claim, usually the application name, default "go-pkgz/auth"

	URL       string          // root url for the rest service, i.e. http://blah.example.com, required
	Validator token.Validator // validator allows to reject some valid tokens with user-defined logic

	AvatarStore       avatar.Store // store to save/load avatars, required (use avatar.NoOp to disable avatars support)
	AvatarResizeLimit int          // resize avatar's limit in pixels
	AvatarRoutePath   string       // avatar routing prefix, i.e. "/api/v1/avatar", default `/avatar`
	UseGravatar       bool         // for email based auth (verified provider) use gravatar service

	AdminPasswd      string                   // if presented, allows basic auth with user admin and given password
	BasicAuthChecker middleware.BasicAuthFunc // user custom checker for basic auth, if one defined then "AdminPasswd" will ignored
	AudienceReader   token.Audience           // list of allowed aud values, default (empty) allows any
	AudSecrets       bool                     // allow multiple secrets (secret per aud)
	Logger           logger.L                 // logger interface, default is no logging at all
	RefreshCache     middleware.RefreshCache  // optional cache to keep refreshed tokens
}

// NewService initializes everything
func NewService(opts Opts) (res *Service) {

	res = &Service{
		opts:   opts,
		logger: opts.Logger,
		authMiddleware: middleware.Authenticator{
			Validator:        opts.Validator,
			AdminPasswd:      opts.AdminPasswd,
			BasicAuthChecker: opts.BasicAuthChecker,
			RefreshCache:     opts.RefreshCache,
		},
		issuer:      opts.Issuer,
		useGravatar: opts.UseGravatar,
	}

	if opts.Issuer == "" {
		res.issuer = "go-pkgz/auth"
	}

	if opts.Logger == nil {
		res.logger = logger.NoOp
	}

	jwtService := token.NewService(token.Opts{
		SecretReader:    opts.SecretReader,
		ClaimsUpd:       opts.ClaimsUpd,
		SecureCookies:   opts.SecureCookies,
		TokenDuration:   opts.TokenDuration,
		CookieDuration:  opts.CookieDuration,
		DisableXSRF:     opts.DisableXSRF,
		DisableIAT:      opts.DisableIAT,
		JWTCookieName:   opts.JWTCookieName,
		JWTCookieDomain: opts.JWTCookieDomain,
		JWTHeaderKey:    opts.JWTHeaderKey,
		XSRFCookieName:  opts.XSRFCookieName,
		XSRFHeaderKey:   opts.XSRFHeaderKey,
		SendJWTHeader:   opts.SendJWTHeader,
		JWTQuery:        opts.JWTQuery,
		Issuer:          res.issuer,
		AudienceReader:  opts.AudienceReader,
		AudSecrets:      opts.AudSecrets,
		SameSite:        opts.SameSiteCookie,
	})

	if opts.SecretReader == nil {
		jwtService.SecretReader = token.SecretFunc(func(string) (string, error) {
			return "", fmt.Errorf("secrets reader not available")
		})
		res.logger.Logf("[WARN] no secret reader defined")
	}

	res.jwtService = jwtService
	res.authMiddleware.JWTService = jwtService
	res.authMiddleware.L = res.logger

	if opts.AvatarStore != nil {
		res.avatarProxy = &avatar.Proxy{
			Store:       opts.AvatarStore,
			URL:         opts.URL,
			RoutePath:   opts.AvatarRoutePath,
			ResizeLimit: opts.AvatarResizeLimit,
			L:           res.logger,
		}
		if res.avatarProxy.RoutePath == "" {
			res.avatarProxy.RoutePath = "/avatar"
		}
	}

	return res
}

// Handlers gets http.Handler for all providers and avatars
func (s *Service) Handlers() (authHandler, avatarHandler http.Handler) {

	ah := func(w http.ResponseWriter, r *http.Request) {
		elems := strings.Split(r.URL.Path, "/")
		if len(elems) < 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// list all providers
		if elems[len(elems)-1] == "list" {
			list := []string{}
			for _, p := range s.providers {
				list = append(list, p.Name())
			}
			rest.RenderJSON(w, list)
			return
		}

		// allow logout without specifying provider
		if elems[len(elems)-1] == "logout" {
			if len(s.providers) == 0 {
				w.WriteHeader(http.StatusBadRequest)
				rest.RenderJSON(w, rest.JSON{"error": "providers not defined"})
				return
			}
			s.providers[0].Handler(w, r)
			return
		}

		// show user info
		if elems[len(elems)-1] == "user" {
			claims, _, err := s.jwtService.Get(r)
			if err != nil || claims.User == nil {
				w.WriteHeader(http.StatusUnauthorized)
				msg := "user is nil"
				if err != nil {
					msg = err.Error()
				}
				rest.RenderJSON(w, rest.JSON{"error": msg})
				return
			}
			rest.RenderJSON(w, claims.User)
			return
		}

		// status of logged-in user
		if elems[len(elems)-1] == "status" {
			claims, _, err := s.jwtService.Get(r)
			if err != nil || claims.User == nil {
				rest.RenderJSON(w, rest.JSON{"status": "not logged in"})
				return
			}
			rest.RenderJSON(w, rest.JSON{"status": "logged in", "user": claims.User.Name})
			return
		}

		// regular auth handlers
		provName := elems[len(elems)-2]
		p, err := s.Provider(provName)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			rest.RenderJSON(w, rest.JSON{"error": fmt.Sprintf("provider %s not supported", provName)})
			return
		}
		p.Handler(w, r)
	}

	return http.HandlerFunc(ah), http.HandlerFunc(s.avatarProxy.Handler)
}

// Middleware returns auth middleware
func (s *Service) Middleware() middleware.Authenticator {
	return s.authMiddleware
}

// AddProvider adds provider for given name
func (s *Service) AddProvider(name, cid, csecret string) {

	p := provider.Params{
		URL:         s.opts.URL,
		JwtService:  s.jwtService,
		Issuer:      s.issuer,
		AvatarSaver: s.avatarProxy,
		Cid:         cid,
		Csecret:     csecret,
		L:           s.logger,
	}

	switch strings.ToLower(name) {
	case "github":
		s.providers = append(s.providers, provider.NewService(provider.NewGithub(p)))
	case "google":
		s.providers = append(s.providers, provider.NewService(provider.NewGoogle(p)))
	case "facebook":
		s.providers = append(s.providers, provider.NewService(provider.NewFacebook(p)))
	case "yandex":
		s.providers = append(s.providers, provider.NewService(provider.NewYandex(p)))
	case "battlenet":
		s.providers = append(s.providers, provider.NewService(provider.NewBattlenet(p)))
	case "microsoft":
		s.providers = append(s.providers, provider.NewService(provider.NewMicrosoft(p)))
	case "twitter":
		s.providers = append(s.providers, provider.NewService(provider.NewTwitter(p)))
	case "patreon":
		s.providers = append(s.providers, provider.NewService(provider.NewPatreon(p)))
	case "dev":
		s.providers = append(s.providers, provider.NewService(provider.NewDev(p)))
	default:
		return
	}

	s.authMiddleware.Providers = s.providers
}

// AddDevProvider with a custom host and port
func (s *Service) AddDevProvider(host string, port int) {
	p := provider.Params{
		URL:         s.opts.URL,
		JwtService:  s.jwtService,
		Issuer:      s.issuer,
		AvatarSaver: s.avatarProxy,
		L:           s.logger,
		Port:        port,
		Host:        host,
	}
	s.providers = append(s.providers, provider.NewService(provider.NewDev(p)))
}

// AddAppleProvider allow SignIn with Apple ID
func (s *Service) AddAppleProvider(appleConfig provider.AppleConfig, privKeyLoader provider.PrivateKeyLoaderInterface) error {
	p := provider.Params{
		URL:         s.opts.URL,
		JwtService:  s.jwtService,
		Issuer:      s.issuer,
		AvatarSaver: s.avatarProxy,
		L:           s.logger,
	}

	// Error checking at create need for catch one when apple private key init
	appleProvider, err := provider.NewApple(p, appleConfig, privKeyLoader)
	if err != nil {
		return fmt.Errorf("an AppleProvider creating failed: %w", err)
	}

	s.providers = append(s.providers, provider.NewService(appleProvider))
	return nil
}

// AddCustomProvider adds custom provider (e.g. https://gopkg.in/oauth2.v3)
func (s *Service) AddCustomProvider(name string, client Client, copts provider.CustomHandlerOpt) {
	p := provider.Params{
		URL:         s.opts.URL,
		JwtService:  s.jwtService,
		Issuer:      s.issuer,
		AvatarSaver: s.avatarProxy,
		Cid:         client.Cid,
		Csecret:     client.Csecret,
		L:           s.logger,
	}

	s.providers = append(s.providers, provider.NewService(provider.NewCustom(name, p, copts)))
	s.authMiddleware.Providers = s.providers
}

// AddDirectProvider adds provider with direct check against data store
// it doesn't do any handshake and uses provided credChecker to verify user and password from the request
func (s *Service) AddDirectProvider(name string, credChecker provider.CredChecker) {
	dh := provider.DirectHandler{
		L:            s.logger,
		ProviderName: name,
		Issuer:       s.issuer,
		TokenService: s.jwtService,
		CredChecker:  credChecker,
		AvatarSaver:  s.avatarProxy,
	}
	s.providers = append(s.providers, provider.NewService(dh))
	s.authMiddleware.Providers = s.providers
}

// AddDirectProviderWithUserIDFunc adds provider with direct check against data store and sets custom UserIDFunc allows
// to modify user's ID on the client side.
// it doesn't do any handshake and uses provided credChecker to verify user and password from the request
func (s *Service) AddDirectProviderWithUserIDFunc(name string, credChecker provider.CredChecker, ufn provider.UserIDFunc) {
	dh := provider.DirectHandler{
		L:            s.logger,
		ProviderName: name,
		Issuer:       s.issuer,
		TokenService: s.jwtService,
		CredChecker:  credChecker,
		AvatarSaver:  s.avatarProxy,
		UserIDFunc:   ufn,
	}
	s.providers = append(s.providers, provider.NewService(dh))
	s.authMiddleware.Providers = s.providers
}

// AddVerifProvider adds provider user's verification sent by sender
func (s *Service) AddVerifProvider(name, msgTmpl string, sender provider.Sender) {
	dh := provider.VerifyHandler{
		L:            s.logger,
		ProviderName: name,
		Issuer:       s.issuer,
		TokenService: s.jwtService,
		AvatarSaver:  s.avatarProxy,
		Sender:       sender,
		Template:     msgTmpl,
		UseGravatar:  s.useGravatar,
	}
	s.providers = append(s.providers, provider.NewService(dh))
	s.authMiddleware.Providers = s.providers
}

// AddCustomHandler adds user-defined self-implemented handler of auth provider
func (s *Service) AddCustomHandler(handler provider.Provider) {
	s.providers = append(s.providers, provider.NewService(handler))
	s.authMiddleware.Providers = s.providers
}

// DevAuth makes dev oauth2 server, for testing and development only!
func (s *Service) DevAuth() (*provider.DevAuthServer, error) {
	p, err := s.Provider("dev") // peak dev provider
	if err != nil {
		return nil, fmt.Errorf("dev provider not registered: %w", err)
	}
	// make and start dev auth server
	return &provider.DevAuthServer{Provider: p.Provider.(provider.Oauth2Handler), L: s.logger}, nil
}

// Provider gets provider by name
func (s *Service) Provider(name string) (provider.Service, error) {
	for _, p := range s.providers {
		if p.Name() == name {
			return p, nil
		}
	}
	return provider.Service{}, fmt.Errorf("provider %s not found", name)
}

// Providers gets all registered providers
func (s *Service) Providers() []provider.Service {
	return s.providers
}

// TokenService returns token.Service
func (s *Service) TokenService() *token.Service {
	return s.jwtService
}

// AvatarProxy returns stored in service
func (s *Service) AvatarProxy() *avatar.Proxy {
	return s.avatarProxy
}
