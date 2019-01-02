package token

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
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

// default names for cookies and headers
const (
	jwtCookieName  = "JWT"
	jwtHeaderKey   = "X-JWT"
	xsrfCookieName = "XSRF-TOKEN"
	xsrfHeaderKey  = "X-XSRF-TOKEN"
	issuer         = "go-pkgz/auth"
	tokenDuration  = time.Minute * 15
	cookieDuration = time.Hour * 24 * 31
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

	Issuer string // optional value for iss claim, usually application name
}

// NewService makes JWT service
func NewService(opts Opts) *Service {
	res := Service{Opts: opts}

	setDefault := func(fld *string, def string) {
		if *fld == "" {
			*fld = def
		}
	}

	setDefault(&res.JWTCookieName, jwtCookieName)
	setDefault(&res.JWTHeaderKey, jwtHeaderKey)
	setDefault(&res.XSRFCookieName, xsrfCookieName)
	setDefault(&res.XSRFHeaderKey, xsrfHeaderKey)
	setDefault(&res.Issuer, issuer)

	if opts.TokenDuration == 0 {
		res.TokenDuration = tokenDuration
	}

	if opts.CookieDuration == 0 {
		res.CookieDuration = cookieDuration
	}

	return &res
}

// Token makes token with claims
func (j *Service) Token(claims Claims) (string, error) {

	// update claims with ClaimsUpdFunc defined by consumer
	if j.ClaimsUpd != nil {
		claims = j.ClaimsUpd.Update(claims)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	if j.SecretReader == nil {
		return "", errors.New("secretreader not defined")
	}

	secret, err := j.SecretReader.Get(claims.Audience) // get secret via consumer defined SecretReader
	if err != nil {
		return "", errors.Wrap(err, "can't get secret")
	}

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", errors.Wrap(err, "can't sign token token")
	}
	return tokenString, nil
}

// Parse token string and verify. Not checking for expiration
func (j *Service) Parse(tokenString string) (Claims, error) {
	parser := jwt.Parser{SkipClaimsValidation: true} // allow parsing of expired tokens

	getAud := func() (aud string, err error) { // parse token without signature check to get id (aud)
		preToken, _, err := parser.ParseUnverified(tokenString, &Claims{})
		if err != nil {
			return "", errors.Wrap(err, "can't pre-parse token")
		}
		preClaims, ok := preToken.Claims.(*Claims)
		if !ok {
			return "", errors.New("invalid token")
		}
		return preClaims.Audience, nil
	}

	aud, err := getAud()
	if err != nil {
		return Claims{}, errors.Wrap(err, "failed to get aud from token token")
	}

	if j.SecretReader == nil {
		return Claims{}, errors.New("secretreader not defined")
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
	if !ok || !token.Valid {
		return Claims{}, errors.New("invalid token")
	}

	return *claims, nil
}

// Set creates token cookie with xsrf cookie and put it to ResponseWriter
// accepts claims and sets expiration if none defined. permanent flag means long-living cookie,
// false makes it session only.
func (j *Service) Set(w http.ResponseWriter, claims Claims) error {
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
		return errors.Wrap(err, "failed to make token token")
	}

	cookieExpiration := 0 // session cookie
	if !claims.SessionOnly && claims.Handshake == nil {
		cookieExpiration = int(j.CookieDuration.Seconds())
	}

	jwtCookie := http.Cookie{Name: jwtCookieName, Value: tokenString, HttpOnly: true, Path: "/",
		MaxAge: cookieExpiration, Secure: j.SecureCookies}
	http.SetCookie(w, &jwtCookie)

	xsrfCookie := http.Cookie{Name: xsrfCookieName, Value: claims.Id, HttpOnly: false, Path: "/",
		MaxAge: cookieExpiration, Secure: j.SecureCookies}
	http.SetCookie(w, &xsrfCookie)

	return nil
}

// Get token from header or cookie
// if cookie used, verify xsrf token to match
func (j *Service) Get(r *http.Request) (Claims, string, error) {

	fromCookie := false
	tokenString := ""

	// try to get from X-JWT header
	if tokenHeader := r.Header.Get(jwtHeaderKey); tokenHeader != "" {
		tokenString = tokenHeader
	}

	// try to get from JWT cookie
	if tokenString == "" {
		fromCookie = true
		jc, err := r.Cookie(jwtCookieName)
		if err != nil {
			return Claims{}, "", errors.Wrap(err, "token cookie was not presented")
		}
		tokenString = jc.Value
	}

	claims, err := j.Parse(tokenString)
	if err != nil {
		return Claims{}, "", errors.Wrap(err, "failed to get token")
	}

	if j.DisableXSRF {
		return claims, tokenString, nil
	}

	if fromCookie && claims.User != nil {
		xsrf := r.Header.Get(xsrfHeaderKey)
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
	jwtCookie := http.Cookie{Name: jwtCookieName, Value: "", HttpOnly: false, Path: "/",
		MaxAge: -1, Expires: time.Unix(0, 0), Secure: j.SecureCookies}
	http.SetCookie(w, &jwtCookie)

	xsrfCookie := http.Cookie{Name: xsrfCookieName, Value: "", HttpOnly: false, Path: "/",
		MaxAge: -1, Expires: time.Unix(0, 0), Secure: j.SecureCookies}
	http.SetCookie(w, &xsrfCookie)
}

// Secret defines interface returning secret key for given id (aud)
type Secret interface {
	Get(id string) (string, error)
}

// SecretFunc type is an adapter to allow the use of ordinary functions as Secret. If f is a function
// with the appropriate signature, SecretFunc(f) is a Handler that calls f.
type SecretFunc func(id string) (string, error)

// Get calls f(id)
func (f SecretFunc) Get(id string) (string, error) {
	return f(id)
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

func (c Claims) String() string {
	b, err := json.Marshal(c)
	if err != nil {
		return fmt.Sprintf("%+v %+v", c.StandardClaims, c.User)
	}
	return string(b)
}
