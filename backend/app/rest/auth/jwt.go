package auth

import (
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/store"
)

// JWT wraps jwt operations
// supports both header and cookie jwt
type JWT struct {
	keyStore       KeyStore
	secureCookies  bool
	tokenDuration  time.Duration
	cookieDuration time.Duration
}

// CustomClaims stores user info for auth and state & from from login
type CustomClaims struct {
	jwt.StandardClaims
	User *store.User `json:"user,omitempty"`

	// used for oauth handshake
	State       string `json:"state,omitempty"`
	From        string `json:"from,omitempty"`
	SiteID      string `json:"site_id,omitempty"`
	SessionOnly bool   `json:"sess_only,omitempty"`

	// flags indicate different uses
	Flags struct {
		Login    bool `json:"login,omitempty"`
		DeleteMe bool `json:"deleteme,omitempty"`
	} `json:"flags,omitempty"`
}

const jwtCookieName = "JWT"
const jwtHeaderKey = "X-JWT"
const xsrfCookieName = "XSRF-TOKEN"
const xsrfHeaderKey = "X-XSRF-TOKEN"

// NewJWT makes JWT service
func NewJWT(keyStore KeyStore, secureCookies bool, tokenDuration time.Duration, cookieDuration time.Duration) *JWT {
	res := JWT{
		keyStore:       keyStore,
		secureCookies:  secureCookies,
		tokenDuration:  tokenDuration,
		cookieDuration: cookieDuration,
	}
	return &res
}

// Token makes jwt with claims
func (j *JWT) Token(claims *CustomClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	secret, err := j.keyStore.Key(claims.SiteID)
	if err != nil {
		return "", errors.Wrap(err, "can't get secret")
	}

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", errors.Wrap(err, "can't sign jwt token")
	}
	return tokenString, nil
}

// HasFlags indicates presence of special flags
func (j *JWT) HasFlags(claims *CustomClaims) bool {
	return claims.Flags.DeleteMe || claims.Flags.Login
}

// Parse token string and verify. Not checking for expiration
func (j *JWT) Parse(tokenString string) (*CustomClaims, error) {
	parser := jwt.Parser{SkipClaimsValidation: true} // allow parsing of expired tokens

	getSiteID := func() (siteID string, err error) { // parse token without signature check to get siteID
		preToken, _, err := parser.ParseUnverified(tokenString, &CustomClaims{})
		if err != nil {
			return "", errors.Wrap(err, "can't pre-parse jwt")
		}
		preClaims, ok := preToken.Claims.(*CustomClaims)
		if !ok {
			return "", errors.New("invalid jwt")
		}
		return preClaims.SiteID, nil
	}

	siteID, err := getSiteID()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get siteID from jwt token")
	}

	secret, err := j.keyStore.Key(siteID)
	if err != nil {
		return nil, errors.Wrap(err, "can't get secret")
	}

	token, err := parser.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "can't parse jwt")
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid jwt")
	}

	return claims, nil
}

// Set creates jwt cookie with xsrf cookie and put it to ResponseWriter
// accepts claims and sets expiration if none defined. permanent flag means long-living cookie, false makes it session only.
func (j *JWT) Set(w http.ResponseWriter, claims *CustomClaims, sessionOnly bool) error {
	if claims.ExpiresAt == 0 {
		claims.ExpiresAt = time.Now().Add(j.tokenDuration).Unix()
	}

	tokenString, err := j.Token(claims)
	if err != nil {
		return errors.Wrap(err, "failed to make jwt token")
	}

	cookieExpiration := 0 // session cookie
	if !sessionOnly {
		cookieExpiration = int(j.cookieDuration.Seconds())
	}

	jwtCookie := http.Cookie{Name: jwtCookieName, Value: tokenString, HttpOnly: true, Path: "/",
		MaxAge: cookieExpiration, Secure: j.secureCookies}
	http.SetCookie(w, &jwtCookie)

	xsrfCookie := http.Cookie{Name: xsrfCookieName, Value: claims.Id, HttpOnly: false, Path: "/",
		MaxAge: cookieExpiration, Secure: j.secureCookies}
	http.SetCookie(w, &xsrfCookie)

	return nil
}

// Get jwt from header or cookie
// if cookie used, verify xsrf token to match
func (j *JWT) Get(r *http.Request) (*CustomClaims, error) {

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
			return nil, errors.Wrap(err, "jwt cookie was not presented")
		}
		tokenString = jc.Value
	}

	claims, err := j.Parse(tokenString)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get jwt")
	}

	if fromCookie && claims.User != nil {
		xsrf := r.Header.Get(xsrfHeaderKey)
		if claims.Id != xsrf {
			return nil, errors.New("xsrf mismatch")
		}
	}

	return claims, nil
}

// IsExpired returns true if claims expired
func (j *JWT) IsExpired(claims *CustomClaims) bool {
	return !claims.VerifyExpiresAt(time.Now().Unix(), true)
}

// Reset token's cookies
func (j *JWT) Reset(w http.ResponseWriter) {
	jwtCookie := http.Cookie{Name: jwtCookieName, Value: "", HttpOnly: false, Path: "/",
		MaxAge: -1, Expires: time.Unix(0, 0), Secure: j.secureCookies}
	http.SetCookie(w, &jwtCookie)

	xsrfCookie := http.Cookie{Name: xsrfCookieName, Value: "", HttpOnly: false, Path: "/",
		MaxAge: -1, Expires: time.Unix(0, 0), Secure: j.secureCookies}
	http.SetCookie(w, &xsrfCookie)
}
