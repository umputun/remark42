package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-pkgz/rest"
	"github.com/pkg/errors"

	"github.com/go-pkgz/auth/avatar"
	"github.com/go-pkgz/auth/logger"
	"github.com/go-pkgz/auth/middleware"
	"github.com/go-pkgz/auth/provider"
	"github.com/go-pkgz/auth/token"
)

// Service provides higher level wrapper allowing to construct everything and get back token middleware
type Service struct {
	logger         logger.L
	opts           Opts
	jwtService     *token.Service
	providers      []provider.Service
	authMiddleware middleware.Authenticator
	avatarProxy    *avatar.Proxy
	issuer         string
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
	JWTCookieName  string // default "JWT"
	JWTHeaderKey   string // default "X-JWT"
	XSRFCookieName string // default "XSRF-TOKEN"
	XSRFHeaderKey  string // default "X-XSRF-TOKEN"

	Issuer string // optional value for iss claim, usually the application name, default "go-pkgz/auth"

	URL       string          // root url for the rest service, i.e. http://blah.example.com, required
	Validator token.Validator // validator allows to reject some valid tokens with user-defined logic

	AvatarStore       avatar.Store // store to save/load avatars, required
	AvatarResizeLimit int          // resize avatar's limit in pixels
	AvatarRoutePath   string       // avatar routing prefix, i.e. "/api/v1/avatar", default `/avatar`

	AdminPasswd string   // if presented, allows basic auth with user admin and given password
	Logger      logger.L // logger interface, default is no logging at all
}

// NewService initializes everything
func NewService(opts Opts) (res *Service) {

	res = &Service{
		opts:   opts,
		logger: opts.Logger,
		authMiddleware: middleware.Authenticator{
			Validator:   opts.Validator,
			AdminPasswd: opts.AdminPasswd,
		},
		issuer: opts.Issuer,
	}

	if opts.Issuer == "" {
		res.issuer = "go-pkgz/auth"
	}

	if opts.Logger == nil {
		res.logger = logger.Func(func(fmt string, args ...interface{}) {}) // do-nothing logger
	}

	jwtService := token.NewService(token.Opts{
		SecretReader:   opts.SecretReader,
		ClaimsUpd:      opts.ClaimsUpd,
		SecureCookies:  opts.SecureCookies,
		TokenDuration:  opts.TokenDuration,
		CookieDuration: opts.CookieDuration,
		DisableXSRF:    opts.DisableXSRF,
		DisableIAT:     opts.DisableIAT,
		JWTCookieName:  opts.JWTCookieName,
		JWTHeaderKey:   opts.JWTHeaderKey,
		XSRFCookieName: opts.XSRFCookieName,
		XSRFHeaderKey:  opts.XSRFHeaderKey,
		Issuer:         res.issuer,
	})

	if opts.SecretReader == nil {
		jwtService.SecretReader = token.SecretFunc(func(id string) (string, error) {
			return "", errors.New("secrets reader not available")
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
func (s *Service) Handlers() (authHandler http.Handler, avatarHandler http.Handler) {

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
			rest.RenderJSON(w, r, list)
			return
		}

		// allow logout without specifying provider
		if elems[len(elems)-1] == "logout" {
			s.providers[0].Handler(w, r)
			return
		}

		// show user info
		if elems[len(elems)-1] == "user" {
			claims, _, err := s.jwtService.Get(r)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				rest.RenderJSON(w, r, rest.JSON{"error": err.Error()})
				return
			}
			rest.RenderJSON(w, r, claims.User)
			return
		}

		// regular auth handlers
		provName := elems[len(elems)-2]
		p, err := s.Provider(provName)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			rest.RenderJSON(w, r, rest.JSON{"error": fmt.Sprintf("provider %s not supported", provName)})
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
func (s *Service) AddProvider(name string, cid string, csecret string) {

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
		s.providers = append(s.providers, provider.NewService(provider.NewFacebook(p)))
	case "dev":
		s.providers = append(s.providers, provider.NewService(provider.NewDev(p)))
	default:
		return
	}

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
	}
	s.providers = append(s.providers, provider.NewService(dh))
	s.authMiddleware.Providers = s.providers
}

// DevAuth makes dev oauth2 server, for testing and development only!
func (s *Service) DevAuth() (*provider.DevAuthServer, error) {
	p, err := s.Provider("dev") // peak dev provider
	if err != nil {
		return nil, errors.Wrap(err, "dev provider not registered")
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
	return provider.Service{}, errors.Errorf("provider %s not found", name)
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
