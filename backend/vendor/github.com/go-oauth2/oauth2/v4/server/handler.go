package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/errors"
)

type (
	// ClientInfoHandler get client info from request
	ClientInfoHandler func(r *http.Request) (clientID, clientSecret string, err error)

	// ClientAuthorizedHandler check the client allows to use this authorization grant type
	ClientAuthorizedHandler func(clientID string, grant oauth2.GrantType) (allowed bool, err error)

	// ClientScopeHandler check the client allows to use scope
	ClientScopeHandler func(tgr *oauth2.TokenGenerateRequest) (allowed bool, err error)

	// UserAuthorizationHandler get user id from request authorization
	UserAuthorizationHandler func(w http.ResponseWriter, r *http.Request) (userID string, err error)

	// PasswordAuthorizationHandler get user id from username and password
	PasswordAuthorizationHandler func(ctx context.Context, clientID, username, password string) (userID string, err error)

	// RefreshingScopeHandler check the scope of the refreshing token
	RefreshingScopeHandler func(tgr *oauth2.TokenGenerateRequest, oldScope string) (allowed bool, err error)

	// RefreshingValidationHandler check if refresh_token is still valid. eg no revocation or other
	RefreshingValidationHandler func(ti oauth2.TokenInfo) (allowed bool, err error)

	// ResponseErrorHandler response error handing
	ResponseErrorHandler func(re *errors.Response)

	// InternalErrorHandler internal error handing
	InternalErrorHandler func(err error) (re *errors.Response)

	// PreRedirectErrorHandler is used to override "redirect-on-error" behavior
	PreRedirectErrorHandler func(w http.ResponseWriter, req *AuthorizeRequest, err error) error

	// AuthorizeScopeHandler set the authorized scope
	AuthorizeScopeHandler func(w http.ResponseWriter, r *http.Request) (scope string, err error)

	// AccessTokenExpHandler set expiration date for the access token
	AccessTokenExpHandler func(w http.ResponseWriter, r *http.Request) (exp time.Duration, err error)

	// ExtensionFieldsHandler in response to the access token with the extension of the field
	ExtensionFieldsHandler func(ti oauth2.TokenInfo) (fieldsValue map[string]interface{})

	// ResponseTokenHandler response token handling
	ResponseTokenHandler func(w http.ResponseWriter, data map[string]interface{}, header http.Header, statusCode ...int) error

	// Handler to fetch the refresh token from the request
	RefreshTokenResolveHandler func(r *http.Request) (string, error)

	// Handler to fetch the access token from the request
	AccessTokenResolveHandler func(r *http.Request) (string, bool)
)

// ClientFormHandler get client data from form
func ClientFormHandler(r *http.Request) (string, string, error) {
	clientID := r.Form.Get("client_id")
	if clientID == "" {
		return "", "", errors.ErrInvalidClient
	}
	clientSecret := r.Form.Get("client_secret")
	return clientID, clientSecret, nil
}

// ClientBasicHandler get client data from basic authorization
func ClientBasicHandler(r *http.Request) (string, string, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return "", "", errors.ErrInvalidClient
	}
	return username, password, nil
}

func RefreshTokenFormResolveHandler(r *http.Request) (string, error) {
	rt := r.FormValue("refresh_token")
	if rt == "" {
		return "", errors.ErrInvalidRequest
	}

	return rt, nil
}

func RefreshTokenCookieResolveHandler(r *http.Request) (string, error) {
	c, err := r.Cookie("refresh_token")
	if err != nil {
		return "", errors.ErrInvalidRequest
	}

	return c.Value, nil
}

func AccessTokenDefaultResolveHandler(r *http.Request) (string, bool) {
	token := ""
	auth := r.Header.Get("Authorization")
	prefix := "Bearer "

	if auth != "" && strings.HasPrefix(auth, prefix) {
		token = auth[len(prefix):]
	} else {
		token = r.FormValue("access_token")
	}

	return token, token != ""
}

func AccessTokenCookieResolveHandler(r *http.Request) (string, bool) {
	c, err := r.Cookie("access_token")
	if err != nil {
		return "", false
	}

	return c.Value, true
}
