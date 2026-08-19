package provider

// This is implementation need for fetch and parse Apple public key to verify the ID token signature.
// Apple endpoint can return multiple keys, and the count of keys can vary over time.
// From this set of keys, select the key with the matching key identifier (kid) to verify the signature of any JSON Web Token (JWT) issued by Apple.
// For more details go to link https://developer.apple.com/documentation/sign_in_with_apple/fetch_apple_s_public_key_for_verifying_token_signature

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// appleKeysURL is the endpoint URL for fetch Apple’s public key
const appleKeysURL = "https://appleid.apple.com/auth/keys"

// applePublicKey is the Apple public key object
// Apple public key is a data structure that represents a cryptographic key as JSON Web Key (JWK)
// based on RFC-7517 https://datatracker.ietf.org/doc/html/rfc7517
type applePublicKey struct {
	ID        string `json:"id"`
	KeyType   string `json:"kty"`
	Usage     string `json:"use"`
	Algorithm string `json:"alg"`

	publicKey *rsa.PublicKey
}

// appleRawKey is raw json object
type appleRawKey struct {
	KTY string `json:"kty"`
	KID string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// fetchAppleJWK to make web request to Apple service for get Apple public keys (JWK)
func fetchAppleJWK(ctx context.Context, keyURL string) (set appleKeySet, err error) {
	client := http.Client{Timeout: time.Second * 5}

	if keyURL == "" {
		keyURL = appleKeysURL
	}

	req, err := http.NewRequestWithContext(ctx, "GET", keyURL, http.NoBody)

	if err != nil {
		return set, fmt.Errorf("failed to prepare new request for fetch Apple public keys: %w", err)
	}

	req.Header.Add("accept", AcceptJSONHeader)
	req.Header.Add("user-agent", defaultUserAgent) // apple requires a user agent

	res, err := client.Do(req)
	if err != nil {
		return set, fmt.Errorf("failed to fetch Apple public keys: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	// an error body parses as a key set with no keys, caching it would break every login until it expires
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return set, fmt.Errorf("failed to fetch Apple public keys, status %s", res.Status)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return set, fmt.Errorf("failed read data after Apple public key fetched: %w", err)
	}

	set, err = parseAppleJWK(data)
	if err != nil {
		return set, fmt.Errorf("get set of apple public key failed: %w", err)
	}
	if len(set.keys) == 0 {
		return appleKeySet{}, fmt.Errorf("no keys in Apple public key response")
	}

	return set, nil
}

// parseAppleJWK try parse keys data for return set of Apple public keys, if no errors
func parseAppleJWK(keyData []byte) (set appleKeySet, err error) {

	var rawKeys struct {
		Keys []appleRawKey `json:"keys"`
	}

	set = appleKeySet{} // init key sets
	keys := make(map[string]*applePublicKey)

	if err = json.Unmarshal(keyData, &rawKeys); err != nil {
		return set, fmt.Errorf("parse json data with Apple keys failed: %w", err)
	}
	for _, rawKey := range rawKeys.Keys {
		key, err := parseApplePublicKey(rawKey)
		if err != nil {
			return set, err // no idea to continue iterate keys if at least one return error, need will check all public keys
		}
		keys[key.ID] = key
	}

	set.keys = keys
	return set, nil
}

// parseApplePublicKey to  make parse JWK data for create an Apple public key
func parseApplePublicKey(rawKey appleRawKey) (key *applePublicKey, err error) {

	key = &applePublicKey{
		KeyType:   rawKey.KTY,
		ID:        rawKey.KID,
		Usage:     rawKey.Use,
		Algorithm: rawKey.Alg,
	}

	// parse and create public key
	if err := key.createApplePublicKey(rawKey.N, rawKey.E); err != nil {
		return nil, err
	}

	return key, nil
}

// createApplePublicKey need to decodes a base64-encoded larger integer from Apple's key format.
func (apk *applePublicKey) createApplePublicKey(n, e string) error {

	bufferN, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(n) // decode modulus
	if err != nil {
		return fmt.Errorf("failed to decode Apple public key modulus (n): %w", err)
	}

	bufferE, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(e) // decode exponent
	if err != nil {
		return fmt.Errorf("failed to decode Apple public key exponent (e): %w", err)
	}

	// create rsa public key from JWK data
	apk.publicKey = &rsa.PublicKey{
		N: big.NewInt(0).SetBytes(bufferN),
		E: int(big.NewInt(0).SetBytes(bufferE).Int64()),
	}
	return nil
}

// appleKeySet is a set of Apple public keys
type appleKeySet struct {
	keys map[string]*applePublicKey
}

// get return Apple public key with specific KeyID (kid)
func (aks *appleKeySet) get(kid string) (keys *applePublicKey, err error) {
	if len(aks.keys) == 0 {
		return nil, fmt.Errorf("failed to get key in appleKeySet, key set is nil or empty")
	}

	if val, ok := aks.keys[kid]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("key with ID %s not found", kid)
}

// keyFunc use for JWT verify with specific public key
func (aks *appleKeySet) keyFunc(token *jwt.Token) (any, error) {

	keyID, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("get JWT kid header not found")
	}
	key, err := aks.get(keyID)

	if err != nil {
		return nil, err
	}

	return key.publicKey, nil
}

// appleJWKTTL is how long a successfully fetched key set is reused before a refresh
const appleJWKTTL = time.Hour

// appleJWKStaleTTL bounds the use of a cached key set when the key service is unreachable
const appleJWKStaleTTL = 12 * time.Hour

// appleJWKCache keeps the last fetched Apple key set. AppleHandler is copied by value on
// every request, so the cache is held behind a pointer and shared by all copies.
type appleJWKCache struct {
	lock      sync.Mutex
	set       appleKeySet
	fetchedAt time.Time
}

// jwkSet returns a set of Apple public keys able to verify kid. The cached set is reused
// until appleJWKTTL passes; an unknown kid forces a refresh to pick up Apple's key rotation.
// If the refresh fails, a cached set holding the kid is used for up to appleJWKStaleTTL, so
// logins survive a short outage of the key service.
func (ah AppleHandler) jwkSet(ctx context.Context, kid string) (appleKeySet, error) {
	if ah.jwkCache == nil { // handler constructed without NewApple
		return fetchAppleJWK(ctx, ah.conf.jwkURL)
	}

	ah.jwkCache.lock.Lock()
	defer ah.jwkCache.lock.Unlock()

	cached := ah.jwkCache.set
	_, cachedErr := cached.get(kid)
	if cachedErr == nil && time.Since(ah.jwkCache.fetchedAt) < appleJWKTTL {
		return cached, nil
	}

	set, err := fetchAppleJWK(ctx, ah.conf.jwkURL)
	if err != nil {
		if cachedErr == nil && time.Since(ah.jwkCache.fetchedAt) < appleJWKStaleTTL {
			ah.Logf("[WARN] failed to refresh Apple public keys, using cached set: %v", err)
			return cached, nil
		}
		return set, err
	}

	ah.jwkCache.set, ah.jwkCache.fetchedAt = set, time.Now()
	return set, nil
}

// jwkKeyFunc verifies an Apple id_token signature with the key named by the token's kid header
func (ah AppleHandler) jwkKeyFunc(ctx context.Context) jwt.Keyfunc {
	return func(jwtToken *jwt.Token) (any, error) {
		kid, ok := jwtToken.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("get JWT kid header not found")
		}

		set, err := ah.jwkSet(ctx, kid)
		if err != nil {
			ah.Logf("[ERROR] failed to fetch JWK from Apple key service: %v", err)
			return nil, fmt.Errorf("failed to fetch JWK from Apple key service: %w", err)
		}

		key, err := set.get(kid)
		if err != nil {
			return nil, err
		}
		return key.publicKey, nil
	}
}
