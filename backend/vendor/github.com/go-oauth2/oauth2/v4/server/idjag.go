package server

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/go-oauth2/oauth2/v4/errors"
	"github.com/golang-jwt/jwt/v5"
)

type IssuerKeyResolver func(ctx context.Context, issuer, kid string) ([]crypto.PublicKey, error)
type IDJAGAuthorizationHandler func(ctx context.Context, claims IDJAGClaims, req IDJAGRequest) (IDJAGGrant, error)

var allowedIDJAGAlgorithms = []string{
	"RS256", "ES256", "PS256",
}

const IDJAGType = "oauth-id-jag+jwt"

type IDJAGClaims struct {
	Issuer, Subject, ClientID, JTI string
	ExpiresAt, IssuedAt            time.Time
	Scope                          string
	Resource                       []string
	AuthorizationDetails           []map[string]any
}

type IDJAGRequest struct {
	Scope                string
	Resource             []string
	AuthorizationDetails []map[string]any
}

type IDJAGGrant struct {
	Scope                string
	Resource             []string
	AuthorizationDetails []map[string]any
}

type idjagRawClaims struct {
	jwt.RegisteredClaims
	ClientID             string           `json:"client_id"`
	Scope                string           `json:"scope,omitempty"`
	Resource             []string         `json:"resource,omitempty"`
	AuthorizationDetails []map[string]any `json:"authorization_details,omitempty"`
	// Tolerated optional claims (spec §3.1): Tenant, AuthTime, ACR, AMR, AudTenant, AudSub, Email, Act
}

func validateIDJAGAssertion(ctx context.Context, cfg *Config, clientID, assertion string) (*IDJAGClaims, error) {
	unverified, _, err := jwt.NewParser().ParseUnverified(assertion, &idjagRawClaims{})
	if err != nil {
		return nil, errors.ErrInvalidRequest
	}
	typ, _ := unverified.Header["typ"].(string)
	if typ != IDJAGType {
		return nil, errors.ErrInvalidGrant
	}

	alg, _ := unverified.Header["alg"].(string)
	algAllowed := false
	for _, a := range allowedIDJAGAlgorithms {
		if alg == a {
			algAllowed = true
			break
		}
	}
	if !algAllowed {
		return nil, errors.ErrInvalidGrant
	}

	unverifiedClaims := unverified.Claims.(*idjagRawClaims)
	iss := unverifiedClaims.Issuer
	kid, _ := unverified.Header["kid"].(string)
	issuerTrusted := false
	for _, trusted := range cfg.TrustedIDJAGIssuers {
		if iss == trusted {
			issuerTrusted = true
			break
		}
	}
	if !issuerTrusted {
		return nil, errors.ErrInvalidGrant
	}

	keys, err := cfg.IDJAGIssuerKeyResolver(ctx, iss, kid)
	if err != nil || len(keys) == 0 {
		return nil, errors.ErrInvalidGrant
	}

	clockSkew := cfg.IDJAGClockSkew
	if clockSkew == 0 {
		clockSkew = 60 * time.Second
	}

	var verified *idjagRawClaims
	for _, key := range keys {
		k := key
		var c idjagRawClaims
		_, parseErr := jwt.NewParser(
			jwt.WithValidMethods(allowedIDJAGAlgorithms),
			jwt.WithLeeway(clockSkew),
			jwt.WithExpirationRequired(),
		).ParseWithClaims(assertion, &c, func(_ *jwt.Token) (interface{}, error) {
			return k, nil
		})
		if parseErr == nil {
			verified = &c
			break
		}
	}
	if verified == nil {
		return nil, errors.ErrInvalidGrant
	}

	if verified.Issuer == "" || verified.Subject == "" || verified.ID == "" || verified.ClientID == "" {
		return nil, errors.ErrInvalidGrant
	}

	if verified.ExpiresAt == nil || verified.IssuedAt == nil {
		return nil, errors.ErrInvalidGrant
	}
	if verified.IssuedAt.Time.After(time.Now().Add(clockSkew)) {
		return nil, errors.ErrInvalidGrant
	}

	audFound := false
	for _, a := range verified.Audience {
		if a == cfg.IDJAGAudience {
			audFound = true
			break
		}
	}

	if !audFound {
		return nil, errors.ErrInvalidGrant
	}

	if verified.ClientID != clientID {
		return nil, errors.ErrInvalidGrant
	}

	if cfg.IDJAGReplayStore != nil {
		if err := cfg.IDJAGReplayStore.StoreAssertionID(ctx, verified.ID, verified.ExpiresAt.Time); err != nil {
			return nil, errors.ErrInvalidGrant
		}
	}

	return &IDJAGClaims{
		Issuer:               verified.Issuer,
		Subject:              verified.Subject,
		ClientID:             verified.ClientID,
		JTI:                  verified.ID,
		ExpiresAt:            verified.ExpiresAt.Time,
		IssuedAt:             verified.IssuedAt.Time,
		Scope:                verified.Scope,
		Resource:             verified.Resource,
		AuthorizationDetails: verified.AuthorizationDetails,
	}, nil
}

func NewOIDCIssuerKeyResolver(httpClient *http.Client) IssuerKeyResolver {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return func(ctx context.Context, issuer, kid string) ([]crypto.PublicKey, error) {
		configURL := strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, configURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("fetching openid-configuration: unexpected status %d", resp.StatusCode)
		}

		var oidcConfig struct {
			JWKSURI string `json:"jwks_uri"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&oidcConfig); err != nil {
			return nil, err
		}
		if oidcConfig.JWKSURI == "" {
			return nil, fmt.Errorf("missing jwks_uri in openid-configuration")
		}

		req, err = http.NewRequestWithContext(ctx, http.MethodGet, oidcConfig.JWKSURI, nil)
		if err != nil {
			return nil, err
		}
		resp, err = httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("fetching jwks: unexpected status %d", resp.StatusCode)
		}

		var jwks struct {
			Keys []json.RawMessage `json:"keys"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
			return nil, err
		}

		var keys []crypto.PublicKey
		for _, rawKey := range jwks.Keys {
			var meta struct {
				Kty string `json:"kty"`
				Kid string `json:"kid"`
			}
			if err := json.Unmarshal(rawKey, &meta); err != nil {
				continue
			}
			if kid != "" && meta.Kid != kid {
				continue
			}
			switch meta.Kty {
			case "RSA":
				if k, err := parseRSAPublicKey(rawKey); err == nil {
					keys = append(keys, k)
				}
			case "EC":
				if k, err := parseECPublicKey(rawKey); err == nil {
					keys = append(keys, k)
				}
			}
		}
		return keys, nil
	}
}

func parseRSAPublicKey(raw json.RawMessage) (*rsa.PublicKey, error) {
	var k struct {
		N string `json:"n"`
		E string `json:"e"`
	}
	if err := json.Unmarshal(raw, &k); err != nil {
		return nil, err
	}
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, err
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}, nil
}

func parseECPublicKey(raw json.RawMessage) (*ecdsa.PublicKey, error) {
	var k struct {
		Crv string `json:"crv"`
		X   string `json:"x"`
		Y   string `json:"y"`
	}
	if err := json.Unmarshal(raw, &k); err != nil {
		return nil, err
	}
	xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
	if err != nil {
		return nil, err
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(k.Y)
	if err != nil {
		return nil, err
	}
	var curve elliptic.Curve
	switch k.Crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported EC curve: %s", k.Crv)
	}
	x, y := new(big.Int).SetBytes(xBytes), new(big.Int).SetBytes(yBytes)
	if !curve.IsOnCurve(x, y) {
		return nil, fmt.Errorf("EC point is not on curve %s", k.Crv)
	}
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
}
