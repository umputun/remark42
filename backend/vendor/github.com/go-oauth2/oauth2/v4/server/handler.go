package server

import (
	"context"
	"net/http"
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
	PasswordAuthorizationHandler func(ctx context.Context, username, password string) (userID string, err error)

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

	// ResponseTokenHandler response token handing
	ResponseTokenHandler func(w http.ResponseWriter, data map[string]interface{}, header http.Header, statusCode ...int) error
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
