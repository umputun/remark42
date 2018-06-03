package auth

import (
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	"github.com/umputun/remark/app/store"
)

// JWT wraps jwt operations
// supports both header and cookie jwt
type JWT struct {
	secret        string
	secureCookies bool
	exp           time.Duration
}

// CustomClaims stores user info for auth and state & from from login
type CustomClaims struct {
	jwt.StandardClaims
	User *store.User `json:"user,omitempty"`

	// state and from used for oauth handshake
	State       string `json:"state,omitempty"`
	From        string `json:"from,omitempty"`
	SiteID      string `json:"site_id,omitempty"`
	SessionOnly bool   `json:"sess_only,omitempty"`
}

const jwtCookieName = "JWT"
const jwtHeaderKey = "X-JWT"
const xsrfCookieName = "XSRF-TOKEN"
const xsrfHeaderKey = "X-XSRF-TOKEN"

// NewJWT makes JWT service
func NewJWT(secret string, secureCookies bool, exp time.Duration) *JWT {
	res := JWT{
		secret:        secret,
		secureCookies: secureCookies,
		exp:           exp,
	}
	return &res
}

// Token makes jwt with claims
func (j *JWT) Token(claims *CustomClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(j.secret))
	if err != nil {
		return "", errors.Wrap(err, "can't sign jwt token")
	}
	return tokenString, nil
}

// Parse token string and verify
func (j *JWT) Parse(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.secret), nil
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
		claims.ExpiresAt = time.Now().Add(j.exp).Unix()
	}

	tokenString, err := j.Token(claims)
	if err != nil {
		return errors.Wrap(err, "failed to make jwt token")
	}

	cookieExpiration := 0 // session cookie
	if !sessionOnly {
		cookieExpiration = 365 * 24 * 3600 // 1 year
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

// Refresh gets jwt from request, checks if it will be expiring soon (1/2 of expiration) and create the new onw
func (j *JWT) Refresh(w http.ResponseWriter, r *http.Request) (*CustomClaims, error) {
	claims, err := j.Get(r)
	if err != nil {
		return nil, err
	}
	untilExp := claims.ExpiresAt - time.Now().Unix()
	if untilExp <= int64(j.exp.Seconds()/2) {
		claims.ExpiresAt = time.Now().Add(j.exp).Unix()
		e := j.Set(w, claims, claims.SessionOnly)
		return claims, e
	}
	return claims, nil
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
