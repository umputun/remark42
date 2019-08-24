package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gopkg.in/oauth2.v3"
	"gopkg.in/oauth2.v3/errors"
)

// NewDefaultServer create a default authorization server
func NewDefaultServer(manager oauth2.Manager) *Server {
	return NewServer(NewConfig(), manager)
}

// NewServer create authorization server
func NewServer(cfg *Config, manager oauth2.Manager) *Server {
	srv := &Server{
		Config:  cfg,
		Manager: manager,
	}

	// default handler
	srv.ClientInfoHandler = ClientBasicHandler

	srv.UserAuthorizationHandler = func(w http.ResponseWriter, r *http.Request) (userID string, err error) {
		err = errors.ErrAccessDenied
		return
	}

	srv.PasswordAuthorizationHandler = func(username, password string) (userID string, err error) {
		err = errors.ErrAccessDenied
		return
	}
	return srv
}

// Server Provide authorization server
type Server struct {
	Config                       *Config
	Manager                      oauth2.Manager
	ClientInfoHandler            ClientInfoHandler
	ClientAuthorizedHandler      ClientAuthorizedHandler
	ClientScopeHandler           ClientScopeHandler
	UserAuthorizationHandler     UserAuthorizationHandler
	PasswordAuthorizationHandler PasswordAuthorizationHandler
	RefreshingScopeHandler       RefreshingScopeHandler
	ResponseErrorHandler         ResponseErrorHandler
	InternalErrorHandler         InternalErrorHandler
	ExtensionFieldsHandler       ExtensionFieldsHandler
	AccessTokenExpHandler        AccessTokenExpHandler
	AuthorizeScopeHandler        AuthorizeScopeHandler
}

func (s *Server) redirectError(w http.ResponseWriter, req *AuthorizeRequest, err error) (uerr error) {
	if req == nil {
		uerr = err
		return
	}
	data, _, _ := s.GetErrorData(err)
	err = s.redirect(w, req, data)
	return
}

func (s *Server) redirect(w http.ResponseWriter, req *AuthorizeRequest, data map[string]interface{}) (err error) {
	uri, err := s.GetRedirectURI(req, data)
	if err != nil {
		return
	}
	w.Header().Set("Location", uri)
	w.WriteHeader(302)
	return
}

func (s *Server) tokenError(w http.ResponseWriter, err error) (uerr error) {
	data, statusCode, header := s.GetErrorData(err)

	uerr = s.token(w, data, header, statusCode)
	return
}

func (s *Server) token(w http.ResponseWriter, data map[string]interface{}, header http.Header, statusCode ...int) (err error) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	for key := range header {
		w.Header().Set(key, header.Get(key))
	}

	status := http.StatusOK
	if len(statusCode) > 0 && statusCode[0] > 0 {
		status = statusCode[0]
	}

	w.WriteHeader(status)
	err = json.NewEncoder(w).Encode(data)
	return
}

// GetRedirectURI get redirect uri
func (s *Server) GetRedirectURI(req *AuthorizeRequest, data map[string]interface{}) (uri string, err error) {
	u, err := url.Parse(req.RedirectURI)
	if err != nil {
		return
	}

	q := u.Query()
	if req.State != "" {
		q.Set("state", req.State)
	}

	for k, v := range data {
		q.Set(k, fmt.Sprint(v))
	}

	switch req.ResponseType {
	case oauth2.Code:
		u.RawQuery = q.Encode()
	case oauth2.Token:
		u.RawQuery = ""
		u.Fragment, err = url.QueryUnescape(q.Encode())
		if err != nil {
			return
		}
	}

	uri = u.String()
	return
}

// CheckResponseType check allows response type
func (s *Server) CheckResponseType(rt oauth2.ResponseType) bool {
	for _, art := range s.Config.AllowedResponseTypes {
		if art == rt {
			return true
		}
	}
	return false
}

// ValidationAuthorizeRequest the authorization request validation
func (s *Server) ValidationAuthorizeRequest(r *http.Request) (req *AuthorizeRequest, err error) {
	redirectURI := r.FormValue("redirect_uri")
	clientID := r.FormValue("client_id")
	if !(r.Method == "GET" || r.Method == "POST") ||
		clientID == "" {
		err = errors.ErrInvalidRequest
		return
	}

	resType := oauth2.ResponseType(r.FormValue("response_type"))

	if resType.String() == "" {
		err = errors.ErrUnsupportedResponseType
		return
	} else if allowed := s.CheckResponseType(resType); !allowed {
		err = errors.ErrUnauthorizedClient
		return
	}

	req = &AuthorizeRequest{
		RedirectURI:  redirectURI,
		ResponseType: resType,
		ClientID:     clientID,
		State:        r.FormValue("state"),
		Scope:        r.FormValue("scope"),
		Request:      r,
	}
	return
}

// GetAuthorizeToken get authorization token(code)
func (s *Server) GetAuthorizeToken(req *AuthorizeRequest) (ti oauth2.TokenInfo, err error) {
	// check the client allows the grant type
	if fn := s.ClientAuthorizedHandler; fn != nil {
		gt := oauth2.AuthorizationCode

		if req.ResponseType == oauth2.Token {
			gt = oauth2.Implicit
		}

		allowed, verr := fn(req.ClientID, gt)
		if verr != nil {
			err = verr
			return
		} else if !allowed {
			err = errors.ErrUnauthorizedClient
			return
		}
	}

	// check the client allows the authorized scope
	if fn := s.ClientScopeHandler; fn != nil {

		allowed, verr := fn(req.ClientID, req.Scope)
		if verr != nil {
			err = verr
			return
		} else if !allowed {
			err = errors.ErrInvalidScope
			return
		}
	}

	tgr := &oauth2.TokenGenerateRequest{
		ClientID:       req.ClientID,
		UserID:         req.UserID,
		RedirectURI:    req.RedirectURI,
		Scope:          req.Scope,
		AccessTokenExp: req.AccessTokenExp,
		Request:        req.Request,
	}

	ti, err = s.Manager.GenerateAuthToken(req.ResponseType, tgr)
	return
}

// GetAuthorizeData get authorization response data
func (s *Server) GetAuthorizeData(rt oauth2.ResponseType, ti oauth2.TokenInfo) (data map[string]interface{}) {
	if rt == oauth2.Code {
		data = map[string]interface{}{
			"code": ti.GetCode(),
		}
	} else {
		data = s.GetTokenData(ti)
	}
	return
}

// HandleAuthorizeRequest the authorization request handling
func (s *Server) HandleAuthorizeRequest(w http.ResponseWriter, r *http.Request) (err error) {
	req, verr := s.ValidationAuthorizeRequest(r)
	if verr != nil {
		err = s.redirectError(w, req, verr)
		return
	}

	// user authorization
	userID, verr := s.UserAuthorizationHandler(w, r)

	if verr != nil {
		err = s.redirectError(w, req, verr)
		return
	} else if userID == "" {
		return
	}

	req.UserID = userID

	// specify the scope of authorization
	if fn := s.AuthorizeScopeHandler; fn != nil {

		scope, verr := fn(w, r)
		if verr != nil {
			err = verr
			return
		} else if scope != "" {
			req.Scope = scope
		}
	}

	// specify the expiration time of access token
	if fn := s.AccessTokenExpHandler; fn != nil {

		exp, verr := fn(w, r)
		if verr != nil {
			err = verr
			return
		}
		req.AccessTokenExp = exp
	}

	ti, verr := s.GetAuthorizeToken(req)
	if verr != nil {
		err = s.redirectError(w, req, verr)
		return
	}

	// If the redirect URI is empty, the default domain provided by the client is used.
	if req.RedirectURI == "" {
		client, verr := s.Manager.GetClient(req.ClientID)
		if verr != nil {
			err = verr
			return
		}
		req.RedirectURI = client.GetDomain()
	}

	err = s.redirect(w, req, s.GetAuthorizeData(req.ResponseType, ti))
	return
}

// ValidationTokenRequest the token request validation
func (s *Server) ValidationTokenRequest(r *http.Request) (gt oauth2.GrantType, tgr *oauth2.TokenGenerateRequest, err error) {
	if v := r.Method; !(v == "POST" ||
		(s.Config.AllowGetAccessRequest && v == "GET")) {
		err = errors.ErrInvalidRequest
		return
	}

	gt = oauth2.GrantType(r.FormValue("grant_type"))

	if gt.String() == "" {
		err = errors.ErrUnsupportedGrantType
		return
	}

	clientID, clientSecret, err := s.ClientInfoHandler(r)
	if err != nil {
		return
	}

	tgr = &oauth2.TokenGenerateRequest{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Request:      r,
	}

	switch gt {
	case oauth2.AuthorizationCode:
		tgr.RedirectURI = r.FormValue("redirect_uri")
		tgr.Code = r.FormValue("code")

		if tgr.RedirectURI == "" ||
			tgr.Code == "" {
			err = errors.ErrInvalidRequest
			return
		}
	case oauth2.PasswordCredentials:
		tgr.Scope = r.FormValue("scope")
		username, password := r.FormValue("username"), r.FormValue("password")

		if username == "" || password == "" {
			err = errors.ErrInvalidRequest
			return
		}

		userID, verr := s.PasswordAuthorizationHandler(username, password)
		if verr != nil {
			err = verr
			return
		} else if userID == "" {
			err = errors.ErrInvalidGrant
			return
		}

		tgr.UserID = userID
	case oauth2.ClientCredentials:
		tgr.Scope = r.FormValue("scope")
	case oauth2.Refreshing:
		tgr.Refresh = r.FormValue("refresh_token")
		tgr.Scope = r.FormValue("scope")

		if tgr.Refresh == "" {
			err = errors.ErrInvalidRequest
		}
	}
	return
}

// CheckGrantType check allows grant type
func (s *Server) CheckGrantType(gt oauth2.GrantType) bool {
	for _, agt := range s.Config.AllowedGrantTypes {
		if agt == gt {
			return true
		}
	}
	return false
}

// GetAccessToken access token
func (s *Server) GetAccessToken(gt oauth2.GrantType, tgr *oauth2.TokenGenerateRequest) (ti oauth2.TokenInfo, err error) {
	if allowed := s.CheckGrantType(gt); !allowed {
		err = errors.ErrUnauthorizedClient
		return
	}

	if fn := s.ClientAuthorizedHandler; fn != nil {
		allowed, verr := fn(tgr.ClientID, gt)
		if verr != nil {
			err = verr
			return
		} else if !allowed {
			err = errors.ErrUnauthorizedClient
			return
		}
	}

	switch gt {
	case oauth2.AuthorizationCode:
		ati, verr := s.Manager.GenerateAccessToken(gt, tgr)
		if verr != nil {

			if verr == errors.ErrInvalidAuthorizeCode {
				err = errors.ErrInvalidGrant
			} else if verr == errors.ErrInvalidClient {
				err = errors.ErrInvalidClient
			} else {
				err = verr
			}
			return
		}
		ti = ati
	case oauth2.PasswordCredentials, oauth2.ClientCredentials:
		if fn := s.ClientScopeHandler; fn != nil {

			allowed, verr := fn(tgr.ClientID, tgr.Scope)
			if verr != nil {
				err = verr
				return
			} else if !allowed {
				err = errors.ErrInvalidScope
				return
			}
		}
		ti, err = s.Manager.GenerateAccessToken(gt, tgr)
	case oauth2.Refreshing:
		// check scope
		if scope, scopeFn := tgr.Scope, s.RefreshingScopeHandler; scope != "" && scopeFn != nil {

			rti, verr := s.Manager.LoadRefreshToken(tgr.Refresh)
			if verr != nil {
				if verr == errors.ErrInvalidRefreshToken || verr == errors.ErrExpiredRefreshToken {
					err = errors.ErrInvalidGrant
					return
				}
				err = verr
				return
			}

			allowed, verr := scopeFn(scope, rti.GetScope())
			if verr != nil {
				err = verr
				return
			} else if !allowed {
				err = errors.ErrInvalidScope
				return
			}
		}

		rti, verr := s.Manager.RefreshAccessToken(tgr)
		if verr != nil {
			if verr == errors.ErrInvalidRefreshToken || verr == errors.ErrExpiredRefreshToken {
				err = errors.ErrInvalidGrant
			} else {
				err = verr
			}
			return
		}
		ti = rti
	}

	return
}

// GetTokenData token data
func (s *Server) GetTokenData(ti oauth2.TokenInfo) (data map[string]interface{}) {
	data = map[string]interface{}{
		"access_token": ti.GetAccess(),
		"token_type":   s.Config.TokenType,
		"expires_in":   int64(ti.GetAccessExpiresIn() / time.Second),
	}

	if scope := ti.GetScope(); scope != "" {
		data["scope"] = scope
	}

	if refresh := ti.GetRefresh(); refresh != "" {
		data["refresh_token"] = refresh
	}

	if fn := s.ExtensionFieldsHandler; fn != nil {
		ext := fn(ti)
		for k, v := range ext {
			if _, ok := data[k]; ok {
				continue
			}
			data[k] = v
		}
	}
	return
}

// HandleTokenRequest token request handling
func (s *Server) HandleTokenRequest(w http.ResponseWriter, r *http.Request) (err error) {
	gt, tgr, verr := s.ValidationTokenRequest(r)
	if verr != nil {
		err = s.tokenError(w, verr)
		return
	}

	ti, verr := s.GetAccessToken(gt, tgr)
	if verr != nil {
		err = s.tokenError(w, verr)
		return
	}

	err = s.token(w, s.GetTokenData(ti), nil)
	return
}

// GetErrorData get error response data
func (s *Server) GetErrorData(err error) (data map[string]interface{}, statusCode int, header http.Header) {
	re := new(errors.Response)

	if v, ok := errors.Descriptions[err]; ok {
		re.Error = err
		re.Description = v
		re.StatusCode = errors.StatusCodes[err]
	} else {
		if fn := s.InternalErrorHandler; fn != nil {
			if vre := fn(err); vre != nil {
				re = vre
			}
		}

		if re.Error == nil {
			re.Error = errors.ErrServerError
			re.Description = errors.Descriptions[errors.ErrServerError]
			re.StatusCode = errors.StatusCodes[errors.ErrServerError]
		}
	}

	if fn := s.ResponseErrorHandler; fn != nil {
		fn(re)

		if re == nil {
			re = new(errors.Response)
		}
	}

	data = make(map[string]interface{})

	if err := re.Error; err != nil {
		data["error"] = err.Error()
	}

	if v := re.ErrorCode; v != 0 {
		data["error_code"] = v
	}

	if v := re.Description; v != "" {
		data["error_description"] = v
	}

	if v := re.URI; v != "" {
		data["error_uri"] = v
	}

	header = re.Header

	statusCode = http.StatusInternalServerError
	if v := re.StatusCode; v > 0 {
		statusCode = v
	}

	return
}

// BearerAuth parse bearer token
func (s *Server) BearerAuth(r *http.Request) (accessToken string, ok bool) {
	auth := r.Header.Get("Authorization")
	prefix := "Bearer "

	if auth != "" && strings.HasPrefix(auth, prefix) {
		accessToken = auth[len(prefix):]
	} else {
		accessToken = r.FormValue("access_token")
	}

	if accessToken != "" {
		ok = true
	}

	return
}

// ValidationBearerToken validation the bearer tokens
// https://tools.ietf.org/html/rfc6750
func (s *Server) ValidationBearerToken(r *http.Request) (ti oauth2.TokenInfo, err error) {
	accessToken, ok := s.BearerAuth(r)
	if !ok {
		err = errors.ErrInvalidAccessToken
		return
	}

	ti, err = s.Manager.LoadAccessToken(accessToken)

	return
}
