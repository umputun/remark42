package provider

// Implementation sign in with Apple for allow users to sign in to web services using their Apple ID.
// For correct work this provider user must has Apple developer account and correct configure "sign in with Apple" at in
// See more: https://developer.apple.com/documentation/sign_in_with_apple/sign_in_with_apple_rest_api
// and https://developer.apple.com/documentation/sign_in_with_apple/sign_in_with_apple_js/incorporating_sign_in_with_apple_into_other_platforms

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/go-pkgz/rest"
	"github.com/golang-jwt/jwt"

	"github.com/go-pkgz/auth/logger"
	"github.com/go-pkgz/auth/token"
)

const (
	// appleAuthURL is the base authentication URL for sign in with Apple ID and fetch request code for user validation request.
	appleAuthURL = "https://appleid.apple.com/auth/authorize"

	// appleTokenURL is the endpoint for verifying tokens and get user unique ID and E-mail
	appleTokenURL = "https://appleid.apple.com/auth/token" // #nosec

	// appleRequestContentType is the valid type which apple REST API accept only
	appleRequestContentType = "application/x-www-form-urlencoded"

	// UserAgent required to every request to Apple REST API
	defaultUserAgent = "github.com/go-pkgz/auth"

	// AcceptJSONHeader is the content to accept from response
	AcceptJSONHeader = "application/json"
)

// appleVerificationResponse is based on https://developer.apple.com/documentation/signinwithapplerestapi/tokenresponse
type appleVerificationResponse struct {
	// A token used to access allowed user data, but now not implemented public interface for it.
	AccessToken string `json:"access_token"`

	// Access token type, always equal the "bearer".
	TokenType string `json:"token_type"`

	// Access token expires time in seconds. Always equal 3600 seconds (1 hour)
	ExpiresIn int `json:"expires_in"`

	// The refresh token used to regenerate new access tokens.
	RefreshToken string `json:"refresh_token"`

	// Main JSON Web Token that contains the user’s identity information.
	IDToken string `json:"id_token"`

	// Used to capture any error returned in response. Always check error for empty
	Error string `json:"error"`
}

// AppleConfig is the main oauth2 required parameters for "Sign in with Apple"
type AppleConfig struct {
	ClientID     string // the identifier Services ID for your app created in Apple developer account.
	TeamID       string // developer Team ID (10 characters), required for create JWT. It available, after signed in at developer account, by link: https://developer.apple.com/account/#/membership
	KeyID        string // private key ID  assigned to private key obtain in Apple developer account
	ResponseMode string // changes method of receiving data in callback. Default value "form_post" (https://developer.apple.com/documentation/sign_in_with_apple/request_an_authorization_to_the_sign_in_with_apple_server?changes=_1_2#4066168)

	scopes       []string         // for this package allow only username scope and UID in token claims. Apple service API provide only "email" and "name" scope values (https://developer.apple.com/documentation/sign_in_with_apple/clientconfigi/3230955-scope)
	privateKey   interface{}      // private key from Apple obtained in developer account (the keys section). Required for create the Client Secret (https://developer.apple.com/documentation/sign_in_with_apple/generate_and_validate_tokens#3262048)
	publicKey    crypto.PublicKey // need for validate sign of token
	clientSecret string           // is the JWT client secret will create after first call and then used until expired
	jwkURL       string           // URL for fetch JWK Apple keys, need redefine for tests
}

// AppleHandler implements login via Apple ID
type AppleHandler struct {
	Params

	// all of these fields specific to particular oauth2 provider
	name string
	// infoURL  string not implemented at Apple side
	endpoint oauth2.Endpoint

	mapUser func(jwt.MapClaims) token.User // map info from InfoURL to User
	conf    AppleConfig                    // main config for Apple auth provider

	PrivateKeyLoader PrivateKeyLoaderInterface // custom function interface for load private key

}

// PrivateKeyLoaderInterface interface for implement custom loader for Apple private key from user source
type PrivateKeyLoaderInterface interface {
	LoadPrivateKey() ([]byte, error)
}

// LoadFromFileFunc is the type for use pre-defined private key loader function
// Path field must be set with actual path to private key file
type LoadFromFileFunc struct {
	Path string
}

// LoadApplePrivateKeyFromFile return instance for pre-defined loader function from local file
func LoadApplePrivateKeyFromFile(path string) LoadFromFileFunc {
	return LoadFromFileFunc{
		Path: path,
	}
}

// LoadPrivateKey implement pre-defined (built-in) PrivateKeyLoaderInterface interface method for load private key from local file
func (lf LoadFromFileFunc) LoadPrivateKey() ([]byte, error) {
	if lf.Path == "" {
		return nil, fmt.Errorf("empty private key path not allowed")
	}

	keyFile, err := os.Open(lf.Path)
	if err != nil {
		return nil, err
	}
	keyValue, err := io.ReadAll(keyFile)
	if err != nil {
		return nil, err
	}
	err = keyFile.Close()
	return keyValue, err
}

// NewApple create new AppleProvider instance with a user parameters
// Private key must be set, when instance create call, for create `client_secret`
func NewApple(p Params, appleCfg AppleConfig, privateKeyLoader PrivateKeyLoaderInterface) (*AppleHandler, error) {

	if p.L == nil {
		p.L = logger.NoOp
	}
	var emptyParams []string

	// check required parameters filled
	if appleCfg.ClientID == "" {
		emptyParams = append(emptyParams, "ClientID")
	}
	if appleCfg.TeamID == "" {
		emptyParams = append(emptyParams, "TeamID")
	}
	if appleCfg.KeyID == "" {
		emptyParams = append(emptyParams, "KeyID")
	}
	if len(emptyParams) > 0 {
		return nil, fmt.Errorf("required params missed: %s", strings.Join(emptyParams, ", "))
	}

	responseMode := "form_post"
	if appleCfg.ResponseMode != "" {
		responseMode = appleCfg.ResponseMode
	}

	ah := AppleHandler{
		Params: p,
		name:   "apple", // static name for an Apple provider

		conf: AppleConfig{
			ClientID:     appleCfg.ClientID,
			TeamID:       appleCfg.TeamID,
			KeyID:        appleCfg.KeyID,
			scopes:       []string{"name"},
			jwkURL:       appleKeysURL,
			ResponseMode: responseMode,
		},

		endpoint: oauth2.Endpoint{
			AuthURL:  appleAuthURL,
			TokenURL: appleTokenURL,
		},

		mapUser: func(claims jwt.MapClaims) token.User {
			var usr token.User
			if uid, ok := claims["sub"]; ok {
				usr.ID = "apple_" + token.HashID(sha1.New(), uid.(string))
			}
			return usr
		},
	}

	if privateKeyLoader == nil {
		return nil, fmt.Errorf("private key loader undefined")
	}

	ah.PrivateKeyLoader = privateKeyLoader

	err := ah.initPrivateKey()
	return &ah, err
}

// initPrivateKey parse Apple private key and assign to AppleHandler
func (ah *AppleHandler) initPrivateKey() error {

	sKey, err := ah.PrivateKeyLoader.LoadPrivateKey()
	if err != nil {
		return fmt.Errorf("problem with private key loading: %w", err)
	}

	block, _ := pem.Decode(sKey)
	if block == nil {
		return fmt.Errorf("empty block after decoding")
	}
	ah.conf.privateKey, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return err
	}
	publicKey, ok := ah.conf.privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return fmt.Errorf("provided private key is not ECDSA")
	}
	ah.conf.publicKey = publicKey.Public()
	ah.conf.clientSecret, err = ah.createClientSecret()
	if err != nil {
		return err
	}
	return nil
}

// tokenKeyFunc use for verify JWT sign, it receives the parsed token and should return the key for validating.
func (ah *AppleHandler) tokenKeyFunc(jwtToken *jwt.Token) (interface{}, error) {
	if jwtToken == nil {
		return nil, fmt.Errorf("failed to call token keyFunc, because token is nil")
	}
	return ah.conf.publicKey, nil // extract public key from private key
}

// Name of the provider
func (ah *AppleHandler) Name() string { return ah.name }

// LoginHandler - GET */{provider-name}/login
func (ah *AppleHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {

	ah.Logf("[DEBUG] login with %s", ah.Name())
	// make state (random) and store in session
	state, err := randToken()
	if err != nil {
		rest.SendErrorJSON(w, r, ah.L, http.StatusInternalServerError, err, "failed to make oauth2 state")
		return
	}

	cid, err := randToken()
	if err != nil {
		rest.SendErrorJSON(w, r, ah.L, http.StatusInternalServerError, err, "failed to make claim's id")
		return
	}

	claims := token.Claims{
		Handshake: &token.Handshake{
			State: state,
			From:  r.URL.Query().Get("from"),
		},
		SessionOnly: r.URL.Query().Get("session") != "" && r.URL.Query().Get("session") != "0",
		StandardClaims: jwt.StandardClaims{
			Id:        cid,
			Audience:  r.URL.Query().Get("site"),
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
			NotBefore: time.Now().Add(-1 * time.Minute).Unix(),
		},
	}

	if _, err = ah.JwtService.Set(w, claims); err != nil {
		rest.SendErrorJSON(w, r, ah.L, http.StatusInternalServerError, err, "failed to set token")
		return
	}

	// return login url
	loginURL, err := ah.prepareLoginURL(state, r.URL.Path)
	if err != nil {
		errMsg := fmt.Sprintf("prepare login url for [%s] provider failed", ah.name)
		ah.Logf("[ERROR] %s", errMsg)
		rest.SendErrorJSON(w, r, ah.L, http.StatusInternalServerError, err, errMsg)
		return
	}
	ah.Logf("[DEBUG] login url %s, claims=%+v", loginURL, claims)

	http.Redirect(w, r, loginURL, http.StatusFound)
}

// AuthHandler fills user info and redirects to "from" url. This is callback url redirected locally by browser
// GET /callback
func (ah AppleHandler) AuthHandler(w http.ResponseWriter, r *http.Request) {

	// read response form data
	if err := r.ParseForm(); err != nil {
		rest.SendErrorJSON(w, r, ah.L, http.StatusInternalServerError, err, "read callback response from data failed")
		return
	}

	state := r.FormValue("state") // state value which sent with auth request
	code := r.FormValue("code")   //  client code for validation

	// response with user name filed return only one time at first login, next login  field user doesn't exist
	// until user delete sign with Apple ID in account profile (security section)
	// example response: {"name":{"firstName":"Chan","lastName":"Lu"},"email":"user@email.com"}
	jUser := r.FormValue("user") // json string with user name

	oauthClaims, _, err := ah.JwtService.Get(r)
	if err != nil {
		rest.SendErrorJSON(w, r, ah.L, http.StatusInternalServerError, err, "failed to get token")
		return
	}

	if oauthClaims.Handshake == nil {
		rest.SendErrorJSON(w, r, ah.L, http.StatusForbidden, nil, "invalid handshake token")
		return
	}

	retrievedState := oauthClaims.Handshake.State
	if retrievedState == "" || retrievedState != state {
		rest.SendErrorJSON(w, r, ah.L, http.StatusForbidden, nil, "unexpected state")
		return
	}

	var resp appleVerificationResponse
	err = ah.exchange(context.Background(), code, ah.makeRedirURL(r.URL.Path), &resp)
	if err != nil {
		rest.SendErrorJSON(w, r, ah.L, http.StatusInternalServerError, err, "exchange failed")
		return
	}
	ah.Logf("[DEBUG] response data %+v", resp)
	if resp.Error != "" {
		rest.SendErrorJSON(w, r, ah.L, http.StatusInternalServerError, nil, fmt.Sprintf("fetch IDtoken response error: %s", resp.Error))
		return
	}

	// trying to fetch Apple public key (JWK) for verify token signature, it need for verify IDToken received from Apple
	keySet, err := fetchAppleJWK(r.Context(), ah.conf.jwkURL)
	if err != nil {
		ah.L.Logf("[ERROR] failed to fetch JWK from Apple key service: " + err.Error())
		rest.SendErrorJSON(w, r, ah.L, http.StatusInternalServerError, nil, fmt.Sprintf("failed to fetch JWK from Apple key service: %s", resp.Error))
		return
	}

	// get token claims for extract uid (and email or name if they exist in scope)
	tokenClaims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(resp.IDToken, tokenClaims, keySet.keyFunc)
	if err != nil {
		ah.L.Logf("[ERROR] failed to get claims: " + err.Error())
		rest.SendErrorJSON(w, r, ah.L, http.StatusInternalServerError, nil, fmt.Sprintf("failed to token validation, key is invalid: %s", resp.Error))
		return
	}

	u := ah.mapUser(tokenClaims)

	u, err = setAvatar(ah.AvatarSaver, u, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		rest.SendErrorJSON(w, r, ah.L, http.StatusInternalServerError, err, "failed to save avatar to proxy")
		return
	}

	// try parse username if one exist at response or noname assign
	ah.parseUserData(&u, jUser)

	cid, err := randToken()
	if err != nil {
		rest.SendErrorJSON(w, r, ah.L, http.StatusInternalServerError, err, "failed to make claim's id")
		return
	}

	claims := token.Claims{
		User: &u,
		StandardClaims: jwt.StandardClaims{
			Issuer:   ah.Issuer,
			Id:       cid,
			Audience: oauthClaims.Audience,
		},
		SessionOnly: false,
	}

	if _, err = ah.JwtService.Set(w, claims); err != nil {
		rest.SendErrorJSON(w, r, ah.L, http.StatusInternalServerError, err, "failed to set token")
		return
	}

	ah.Logf("[DEBUG] user info %+v", u)

	// redirect to back url if presented in login query params
	if oauthClaims.Handshake != nil && oauthClaims.Handshake.From != "" {
		http.Redirect(w, r, oauthClaims.Handshake.From, http.StatusTemporaryRedirect)
		return
	}
	rest.RenderJSON(w, &u)

}

// LogoutHandler - GET /logout
func (ah AppleHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if _, _, err := ah.JwtService.Get(r); err != nil {
		rest.SendErrorJSON(w, r, ah.L, http.StatusForbidden, err, "logout not allowed")
		return
	}
	ah.JwtService.Reset(w)
}

// exchange sends the validation token request and gets access token and user claims
// (e.g. https://developer.apple.com/documentation/sign_in_with_apple/generate_and_validate_tokens)
func (ah *AppleHandler) exchange(ctx context.Context, code, redirectURI string, result *appleVerificationResponse) error {

	// check client_secret for valid and recreate new (client_secret JWT) if required
	if tkn, err := jwt.Parse(ah.conf.clientSecret, ah.tokenKeyFunc); err != nil || tkn == nil {
		ah.conf.clientSecret, err = ah.createClientSecret()
		if err != nil {
			return fmt.Errorf("client secret create failed: %w", err)
		}
	}

	data := url.Values{}
	data.Set("client_id", ah.conf.ClientID)
	data.Set("client_secret", ah.conf.clientSecret) // JWT signed with Apple private key
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI) // redirect URL can't refer to localhost and must have trusted certificate and https protocol
	data.Set("grant_type", "authorization_code")

	client := http.Client{Timeout: time.Second * 5}
	req, err := http.NewRequestWithContext(ctx, "POST", ah.endpoint.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Add("content-type", appleRequestContentType)
	req.Header.Add("accept", AcceptJSONHeader)
	req.Header.Add("user-agent", defaultUserAgent) // apple requires a user agent

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	// Trying to decode (unmarshal json) data of response
	err = json.NewDecoder(res.Body).Decode(result)
	if err != nil {
		return fmt.Errorf("unmarshalling data from apple service response failed: %w", err)
	}

	defer func() {
		if err = res.Body.Close(); err != nil {
			ah.L.Logf("[ERROR] close request body failed when get access token: %v", err)
		}
	}()

	// If above operation done successfully checking a response code and error descriptions, if one exist.
	// Apple service will response either 200 (OK) or 400 (any error).
	if res.StatusCode != http.StatusOK || result.Error != "" {
		return fmt.Errorf("apple token service error: %s", result.Error)
	}

	return err
}

// createClientSecret use for create the JWT client secret required to make requests to the Apple validation server.
// for more details go to link: https://developer.apple.com/documentation/sign_in_with_apple/generate_and_validate_tokens#3262048
func (ah *AppleHandler) createClientSecret() (string, error) {

	if ah.conf.privateKey == nil {
		return "", fmt.Errorf("private key can't be empty")
	}
	// Create a claims
	now := time.Now()
	exp := now.Add(time.Minute * 30).Unix() // default value

	claims := &jwt.StandardClaims{
		Issuer:    ah.conf.TeamID,
		IssuedAt:  now.Unix(),
		ExpiresAt: exp,
		Audience:  "https://appleid.apple.com",
		Subject:   ah.conf.ClientID,
	}

	tkn := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tkn.Header["alg"] = "ES256"
	tkn.Header["kid"] = ah.conf.KeyID

	return tkn.SignedString(ah.conf.privateKey)
}

func (ah *AppleHandler) parseUserData(user *token.User, jUser string) {

	type UserData struct {
		Name struct {
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
		} `json:"name"`
		Email string `json:"email"`
	}

	var userData UserData

	// Catch error for log only. No need break flow if user name doesn't exist
	if err := json.Unmarshal([]byte(jUser), &userData); err != nil {
		ah.L.Logf("[DEBUG] failed to parse user data %s: %v", user, err)
		user.Name = "noname_" + user.ID[6:12] // paste noname if user name failed to parse
		return
	}

	user.Name = fmt.Sprintf("%s %s", userData.Name.FirstName, userData.Name.LastName)
}

func (ah *AppleHandler) prepareLoginURL(state, path string) (string, error) {

	scopesList := strings.Join(ah.conf.scopes, " ")

	if scopesList != "" && ah.conf.ResponseMode != "form_post" {
		return "", fmt.Errorf("response_mode must be form_post if scope is not empty")
	}

	authURL, err := url.Parse(ah.endpoint.AuthURL)
	if err != nil {
		return "", err
	}

	query := authURL.Query()
	query.Set("state", state)
	query.Set("response_type", "code")
	query.Set("response_mode", ah.conf.ResponseMode)
	query.Set("client_id", ah.conf.ClientID)
	query.Set("scope", scopesList)
	query.Set("redirect_uri", ah.makeRedirURL(path))
	authURL.RawQuery = query.Encode()

	return authURL.String(), nil

}

func (ah AppleHandler) makeRedirURL(path string) string {
	elems := strings.Split(path, "/")
	newPath := strings.Join(elems[:len(elems)-1], "/")

	return strings.TrimRight(ah.URL, "/") + strings.TrimSuffix(newPath, "/") + urlCallbackSuffix
}
