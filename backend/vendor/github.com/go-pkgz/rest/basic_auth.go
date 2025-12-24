package rest

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

const baContextKey = "authorizedWithBasicAuth"

// BasicAuth middleware requires basic auth and matches user & passwd with client-provided checker
func BasicAuth(checker func(user, passwd string) bool) func(http.Handler) http.Handler {

	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			u, p, ok := r.BasicAuth()
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if !checker(u, p) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey(baContextKey), true)))
		}
		return http.HandlerFunc(fn)
	}
}

// BasicAuthWithUserPasswd middleware requires basic auth and matches user & passwd with client-provided values
func BasicAuthWithUserPasswd(user, passwd string) func(http.Handler) http.Handler {
	checkFn := func(reqUser, reqPasswd string) bool {
		matchUser := subtle.ConstantTimeCompare([]byte(user), []byte(reqUser))
		matchPass := subtle.ConstantTimeCompare([]byte(passwd), []byte(reqPasswd))
		return matchUser == 1 && matchPass == 1
	}
	return BasicAuth(checkFn)
}

// BasicAuthWithBcryptHash middleware requires basic auth and matches user & bcrypt hashed password
func BasicAuthWithBcryptHash(user, hashedPassword string) func(http.Handler) http.Handler {
	checkFn := func(reqUser, reqPasswd string) bool {
		if reqUser != user {
			return false
		}
		err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(reqPasswd))
		return err == nil
	}
	return BasicAuth(checkFn)
}

// BasicAuthWithArgon2Hash middleware requires basic auth and matches user & argon2 hashed password
// both hashedPassword and salt must be base64 encoded strings
// Uses Argon2id with parameters: t=1, m=64*1024 KB, p=4 threads
func BasicAuthWithArgon2Hash(user, hashedPassword, salt string) func(http.Handler) http.Handler {
	checkFn := func(reqUser, reqPasswd string) bool {
		if reqUser != user {
			return false
		}

		saltBytes, err := base64.StdEncoding.DecodeString(salt)
		if err != nil {
			return false
		}
		storedHashBytes, err := base64.StdEncoding.DecodeString(hashedPassword)
		if err != nil {
			return false
		}

		hash := argon2.IDKey([]byte(reqPasswd), saltBytes, 1, 64*1024, 4, 32)
		return subtle.ConstantTimeCompare(hash, storedHashBytes) == 1
	}
	return BasicAuth(checkFn)
}

// IsAuthorized returns true is user authorized.
// it can be used in handlers to check if BasicAuth middleware was applied
func IsAuthorized(ctx context.Context) bool {
	v := ctx.Value(contextKey(baContextKey))
	return v != nil && v.(bool)
}

// BasicAuthWithPrompt middleware requires basic auth and matches user & passwd with client-provided values
// If the user is not authorized, it will prompt for basic auth
func BasicAuthWithPrompt(user, passwd string) func(http.Handler) http.Handler {
	checkFn := func(reqUser, reqPasswd string) bool {
		matchUser := subtle.ConstantTimeCompare([]byte(user), []byte(reqUser))
		matchPass := subtle.ConstantTimeCompare([]byte(passwd), []byte(reqPasswd))
		return matchUser == 1 && matchPass == 1
	}

	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			// extract basic auth from request
			u, p, ok := r.BasicAuth()
			if ok && checkFn(u, p) {
				h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey(baContextKey), true)))
				return
			}
			// not authorized, prompt for basic auth
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
		return http.HandlerFunc(fn)
	}
}

// BasicAuthWithBcryptHashAndPrompt middleware requires basic auth and matches user & bcrypt hashed password
// If the user is not authorized, it will prompt for basic auth
func BasicAuthWithBcryptHashAndPrompt(user, hashedPassword string) func(http.Handler) http.Handler {
	checkFn := func(reqUser, reqPasswd string) bool {
		if reqUser != user {
			return false
		}
		err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(reqPasswd))
		return err == nil
	}

	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// extract basic auth from request
			u, p, ok := r.BasicAuth()
			if ok && checkFn(u, p) {
				h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey(baContextKey), true)))
				return
			}
			// not authorized, prompt for basic auth
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
		return http.HandlerFunc(fn)
	}
}

// GenerateBcryptHash generates a bcrypt hash from a password
func GenerateBcryptHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// GenerateArgon2Hash generates an argon2 hash and salt from a password
func GenerateArgon2Hash(password string) (hash, salt string, err error) {
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", "", err
	}

	// using recommended parameters: time=1, memory=64*1024, threads=4, keyLen=32
	hashBytes := argon2.IDKey([]byte(password), saltBytes, 1, 64*1024, 4, 32)

	return base64.StdEncoding.EncodeToString(hashBytes), base64.StdEncoding.EncodeToString(saltBytes), nil
}
