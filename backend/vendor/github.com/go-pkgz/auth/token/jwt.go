// Package token wraps jwt-go library and provides higher level abstraction to work with JWT.
package token

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

// Service wraps jwt operations
// supports both header and cookie tokens
type Service struct {
	Opts
}

// Claims stores user info for token and state & from from login
type Claims struct {
	jwt.StandardClaims
	User        *User      `json:"user,omitempty"` // user info
	SessionOnly bool       `json:"sess_only,omitempty"`
	Handshake   *Handshake `json:"handshake,omitempty"` // used for oauth handshake
}

// Handshake used for oauth handshake
type Handshake struct {
	State string `json:"state,omitempty"`
	From  string `json:"from,omitempty"`
	ID    string `json:"id,omitempty"`
}

const (
	// default names for cookies and headers
	defaultJWTCookieName  = "JWT"
	defaultJWTHeaderKey   = "X-JWT"
	defaultXSRFCookieName = "XSRF-TOKEN"
	defaultXSRFHeaderKey  = "X-XSRF-TOKEN"

	defaultIssuer = "go-pkgz/auth"

	defaultTokenDuration  = time.Minute * 15
	defaultCookieDuration = time.Hour * 24 * 31

	defaultTokenQuery = "token"
)

// Opts holds constructor params
type Opts struct {
	SecretReader   Secret
	ClaimsUpd      ClaimsUpdater
	SecureCookies  bool
	TokenDuration  time.Duration
	CookieDuration time.Duration
	DisableXSRF    bool
	DisableIAT     bool // disable IssuedAt claim
	// optional (custom) names for cookies and headers
	JWTCookieName  string
	JWTHeaderKey   string
	XSRFCookieName string
	XSRFHeaderKey  string
	JWTQuery       string
	AudienceReader Audience      // allowed aud values
	Issuer         string        // optional value for iss claim, usually application name
	AudSecrets     bool          // uses different secret for differed auds. important: adds pre-parsing of unverified token
	SendJWTHeader  bool          // if enabled send JWT as a header instead of cookie
	SameSite       http.SameSite // define a cookie attribute making it impossible for the browser to send this cookie cross-site
}

// NewService makes JWT service
func NewService(opts Opts) *Service {
	res := Service{Opts: opts}

	setDefault := func(fld *string, def string) {
		if *fld == "" {
			*fld = def
		}
	}

	setDefault(&res.JWTCookieName, defaultJWTCookieName)
	setDefault(&res.JWTHeaderKey, defaultJWTHeaderKey)
	setDefault(&res.XSRFCookieName, defaultXSRFCookieName)
	setDefault(&res.XSRFHeaderKey, defaultXSRFHeaderKey)
	setDefault(&res.JWTQuery, defaultTokenQuery)
	setDefault(&res.Issuer, defaultIssuer)

	if opts.TokenDuration == 0 {
		res.TokenDuration = defaultTokenDuration
	}

	if opts.CookieDuration == 0 {
		res.CookieDuration = defaultCookieDuration
	}

	return &res
}

// Token makes token with claims
func (j *Service) Token(claims Claims) (string, error) {

	// make token for allowed aud values only, rejects others

	// update claims with ClaimsUpdFunc defined by consumer
	if j.ClaimsUpd != nil {
		claims = j.ClaimsUpd.Update(claims)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	if j.SecretReader == nil {
		return "", errors.New("secret reader not defined")
	}

	if err := j.checkAuds(&claims, j.AudienceReader); err != nil {
		return "", errors.Wrap(err, "aud rejected")
	}

	secret, err := j.SecretReader.Get(claims.Audience) // get secret via consumer defined SecretReader
	if err != nil {
		return "", errors.Wrap(err, "can't get secret")
	}

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", errors.Wrap(err, "can't sign token")
	}
	return tokenString, nil
}

// Parse token string and verify. Not checking for expiration
func (j *Service) Parse(tokenString string) (Claims, error) {
	parser := jwt.Parser{SkipClaimsValidation: true} // allow parsing of expired tokens

	if j.SecretReader == nil {
		return Claims{}, errors.New("secret reader not defined")
	}

	aud := "ignore"
	if j.AudSecrets {
		var err error
		aud, err = j.aud(tokenString)
		if err != nil {
			return Claims{}, errors.New("can't retrieve audience from the token")
		}
	}

	secret, err := j.SecretReader.Get(aud)
	if err != nil {
		return Claims{}, errors.Wrap(err, "can't get secret")
	}

	token, err := parser.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return Claims{}, errors.Wrap(err, "can't parse token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return Claims{}, errors.New("invalid token")
	}

	if err = j.checkAuds(claims, j.AudienceReader); err != nil {
		return Claims{}, errors.Wrap(err, "aud rejected")
	}
	return *claims, j.validate(claims)
}

// aud pre-parse token and extracts aud from the claim
// important! this step ignores token verification, should not be used for any validations
func (j *Service) aud(tokenString string) (string, error) {
	parser := jwt.Parser{}
	token, _, err := parser.ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return "", errors.Wrap(err, "can't pre-parse token")
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return "", errors.New("invalid token")
	}
	if strings.TrimSpace(claims.Audience) == "" {
		return "", errors.New("empty aud")
	}
	return claims.Audience, nil
}

func (j *Service) validate(claims *Claims) error {
	cerr := claims.Valid()

	if cerr == nil {
		return nil
	}

	if e, ok := cerr.(*jwt.ValidationError); ok {
		if e.Errors == jwt.ValidationErrorExpired {
			return nil // allow expired tokens
		}
	}

	return cerr
}

// Set creates token cookie with xsrf cookie and put it to ResponseWriter
// accepts claims and sets expiration if none defined. permanent flag means long-living cookie,
// false makes it session only.
func (j *Service) Set(w http.ResponseWriter, claims Claims) (Claims, error) {
	if claims.ExpiresAt == 0 {
		claims.ExpiresAt = time.Now().Add(j.TokenDuration).Unix()
	}

	if claims.Issuer == "" {
		claims.Issuer = j.Issuer
	}

	if !j.DisableIAT {
		claims.IssuedAt = time.Now().Unix()
	}

	tokenString, err := j.Token(claims)
	if err != nil {
		return Claims{}, errors.Wrap(err, "failed to make token token")
	}

	if j.SendJWTHeader {
		w.Header().Set(j.JWTHeaderKey, tokenString)
		return claims, nil
	}

	cookieExpiration := 0 // session cookie
	if !claims.SessionOnly && claims.Handshake == nil {
		cookieExpiration = int(j.CookieDuration.Seconds())
	}

	jwtCookie := http.Cookie{Name: j.JWTCookieName, Value: tokenString, HttpOnly: true, Path: "/",
		MaxAge: cookieExpiration, Secure: j.SecureCookies, SameSite: j.SameSite}
	http.SetCookie(w, &jwtCookie)

	xsrfCookie := http.Cookie{Name: j.XSRFCookieName, Value: claims.Id, HttpOnly: false, Path: "/",
		MaxAge: cookieExpiration, Secure: j.SecureCookies, SameSite: j.SameSite}
	http.SetCookie(w, &xsrfCookie)

	return claims, nil
}

// Get token from url, header or cookie
// if cookie used, verify xsrf token to match
func (j *Service) Get(r *http.Request) (Claims, string, error) {

	fromCookie := false
	tokenString := ""

	// try to get from "token" query param
	if tkQuery := r.URL.Query().Get(j.JWTQuery); tkQuery != "" {
		tokenString = tkQuery
	}

	// try to get from JWT header
	if tokenHeader := r.Header.Get(j.JWTHeaderKey); tokenHeader != "" && tokenString == "" {
		tokenString = tokenHeader
	}

	// try to get from JWT cookie
	if tokenString == "" {
		fromCookie = true
		jc, err := r.Cookie(j.JWTCookieName)
		if err != nil {
			return Claims{}, "", errors.Wrap(err, "token cookie was not presented")
		}
		tokenString = jc.Value
	}

	claims, err := j.Parse(tokenString)
	if err != nil {
		return Claims{}, "", errors.Wrap(err, "failed to get token")
	}

	// promote claim's aud to User.Audience
	if claims.User != nil {
		claims.User.Audience = claims.Audience
	}

	if !fromCookie && j.IsExpired(claims) {
		return Claims{}, "", errors.New("token expired")
	}

	if j.DisableXSRF {
		return claims, tokenString, nil
	}

	if fromCookie && claims.User != nil {
		xsrf := r.Header.Get(j.XSRFHeaderKey)
		if claims.Id != xsrf {
			return Claims{}, "", errors.New("xsrf mismatch")
		}
	}

	return claims, tokenString, nil
}

// IsExpired returns true if claims expired
func (j *Service) IsExpired(claims Claims) bool {
	return !claims.VerifyExpiresAt(time.Now().Unix(), true)
}

// Reset token's cookies
func (j *Service) Reset(w http.ResponseWriter) {
	jwtCookie := http.Cookie{Name: j.JWTCookieName, Value: "", HttpOnly: false, Path: "/",
		MaxAge: -1, Expires: time.Unix(0, 0), Secure: j.SecureCookies, SameSite: j.SameSite}
	http.SetCookie(w, &jwtCookie)

	xsrfCookie := http.Cookie{Name: j.XSRFCookieName, Value: "", HttpOnly: false, Path: "/",
		MaxAge: -1, Expires: time.Unix(0, 0), Secure: j.SecureCookies, SameSite: j.SameSite}
	http.SetCookie(w, &xsrfCookie)
}

// checkAuds verifies if claims.Audience in the list of allowed by audReader
func (j *Service) checkAuds(claims *Claims, audReader Audience) error {
	if audReader == nil { // lack of any allowed means any
		return nil
	}
	auds, err := audReader.Get()
	if err != nil {
		return errors.Wrap(err, "failed to get auds")
	}
	for _, a := range auds {
		if strings.EqualFold(a, claims.Audience) {
			return nil
		}
	}
	return errors.Errorf("aud %q not allowed", claims.Audience)
}

func (c Claims) String() string {
	b, err := json.Marshal(c)
	if err != nil {
		return fmt.Sprintf("%+v %+v", c.StandardClaims, c.User)
	}
	return string(b)
}

// Secret defines interface returning secret key for given id (aud)
type Secret interface {
	Get(aud string) (string, error) // aud matching is optional. Implementation may decide if supported or ignored
}

// SecretFunc type is an adapter to allow the use of ordinary functions as Secret. If f is a function
// with the appropriate signature, SecretFunc(f) is a Handler that calls f.
type SecretFunc func(aud string) (string, error)

// Get calls f()
func (f SecretFunc) Get(aud string) (string, error) {
	return f(aud)
}

// ClaimsUpdater defines interface adding extras to claims
type ClaimsUpdater interface {
	Update(claims Claims) Claims
}

// ClaimsUpdFunc type is an adapter to allow the use of ordinary functions as ClaimsUpdater. If f is a function
// with the appropriate signature, ClaimsUpdFunc(f) is a Handler that calls f.
type ClaimsUpdFunc func(claims Claims) Claims

// Update calls f(id)
func (f ClaimsUpdFunc) Update(claims Claims) Claims {
	return f(claims)
}

// Validator defines interface to accept o reject claims with consumer defined logic
// It works with valid token and allows to reject some, based on token match or user's fields
type Validator interface {
	Validate(token string, claims Claims) bool
}

// ValidatorFunc type is an adapter to allow the use of ordinary functions as Validator. If f is a function
// with the appropriate signature, ValidatorFunc(f) is a Validator that calls f.
type ValidatorFunc func(token string, claims Claims) bool

// Validate calls f(id)
func (f ValidatorFunc) Validate(token string, claims Claims) bool {
	return f(token, claims)
}

// Audience defines interface returning list of allowed audiences
type Audience interface {
	Get() ([]string, error)
}

// AudienceFunc type is an adapter to allow the use of ordinary functions as Audience.
type AudienceFunc func() ([]string, error)

// Get calls f()
func (f AudienceFunc) Get() ([]string, error) {
	return f()
}
