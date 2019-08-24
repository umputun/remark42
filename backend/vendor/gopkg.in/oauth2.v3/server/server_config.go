package server

import (
	"gopkg.in/oauth2.v3"
)

// SetTokenType token type
func (s *Server) SetTokenType(tokenType string) {
	s.Config.TokenType = tokenType
}

// SetAllowGetAccessRequest to allow GET requests for the token
func (s *Server) SetAllowGetAccessRequest(allow bool) {
	s.Config.AllowGetAccessRequest = allow
}

// SetAllowedResponseType allow the authorization types
func (s *Server) SetAllowedResponseType(types ...oauth2.ResponseType) {
	s.Config.AllowedResponseTypes = types
}

// SetAllowedGrantType allow the grant types
func (s *Server) SetAllowedGrantType(types ...oauth2.GrantType) {
	s.Config.AllowedGrantTypes = types
}

// SetClientInfoHandler get client info from request
func (s *Server) SetClientInfoHandler(handler ClientInfoHandler) {
	s.ClientInfoHandler = handler
}

// SetClientAuthorizedHandler check the client allows to use this authorization grant type
func (s *Server) SetClientAuthorizedHandler(handler ClientAuthorizedHandler) {
	s.ClientAuthorizedHandler = handler
}

// SetClientScopeHandler check the client allows to use scope
func (s *Server) SetClientScopeHandler(handler ClientScopeHandler) {
	s.ClientScopeHandler = handler
}

// SetUserAuthorizationHandler get user id from request authorization
func (s *Server) SetUserAuthorizationHandler(handler UserAuthorizationHandler) {
	s.UserAuthorizationHandler = handler
}

// SetPasswordAuthorizationHandler get user id from username and password
func (s *Server) SetPasswordAuthorizationHandler(handler PasswordAuthorizationHandler) {
	s.PasswordAuthorizationHandler = handler
}

// SetRefreshingScopeHandler check the scope of the refreshing token
func (s *Server) SetRefreshingScopeHandler(handler RefreshingScopeHandler) {
	s.RefreshingScopeHandler = handler
}

// SetResponseErrorHandler response error handling
func (s *Server) SetResponseErrorHandler(handler ResponseErrorHandler) {
	s.ResponseErrorHandler = handler
}

// SetInternalErrorHandler internal error handling
func (s *Server) SetInternalErrorHandler(handler InternalErrorHandler) {
	s.InternalErrorHandler = handler
}

// SetExtensionFieldsHandler in response to the access token with the extension of the field
func (s *Server) SetExtensionFieldsHandler(handler ExtensionFieldsHandler) {
	s.ExtensionFieldsHandler = handler
}

// SetAccessTokenExpHandler set expiration date for the access token
func (s *Server) SetAccessTokenExpHandler(handler AccessTokenExpHandler) {
	s.AccessTokenExpHandler = handler
}

// SetAuthorizeScopeHandler set scope for the access token
func (s *Server) SetAuthorizeScopeHandler(handler AuthorizeScopeHandler) {
	s.AuthorizeScopeHandler = handler
}
