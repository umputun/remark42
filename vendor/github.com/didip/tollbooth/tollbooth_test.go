package tollbooth

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestLimitByKeys(t *testing.T) {
	limiter := NewLimiter(1, time.Second) // Only 1 request per second is allowed.

	httperror := LimitByKeys(limiter, []string{"127.0.0.1", "/"})
	if httperror != nil {
		t.Errorf("First time count should not return error. Error: %v", httperror.Error())
	}

	httperror = LimitByKeys(limiter, []string{"127.0.0.1", "/"})
	if httperror == nil {
		t.Errorf("Second time count should return error because it exceeds 1 request per second.")
	}

	<-time.After(1 * time.Second)
	httperror = LimitByKeys(limiter, []string{"127.0.0.1", "/"})
	if httperror != nil {
		t.Errorf("Third time count should not return error because the 1 second window has passed.")
	}
}

func TestDefaultBuildKeys(t *testing.T) {
	limiter := NewLimiter(1, time.Second)
	limiter.IPLookups = []string{"X-Forwarded-For", "X-Real-IP", "RemoteAddr"}

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		t.Errorf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")

	sliceKeys := BuildKeys(limiter, request)
	if len(sliceKeys) == 0 {
		t.Error("Length of sliceKeys should never be empty.")
	}

	for _, keys := range sliceKeys {
		for i, keyChunk := range keys {
			if i == 0 && keyChunk != request.Header.Get("X-Real-IP") {
				t.Errorf("The first chunk should be remote IP. KeyChunk: %v", keyChunk)
			}
			if i == 1 && keyChunk != request.URL.Path {
				t.Errorf("The second chunk should be request path. KeyChunk: %v", keyChunk)
			}
		}
	}
}

func TestBasicAuthBuildKeys(t *testing.T) {
	limiter := NewLimiter(1, time.Second)
	limiter.BasicAuthUsers = []string{"bro"}

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		t.Errorf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")

	request.SetBasicAuth("bro", "tato")

	for _, keys := range BuildKeys(limiter, request) {
		if len(keys) != 3 {
			t.Error("Keys should be made of 3 parts.")
		}
		for i, keyChunk := range keys {
			if i == 0 && keyChunk != request.Header.Get("X-Real-IP") {
				t.Errorf("The (%v) chunk should be remote IP. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 1 && keyChunk != request.URL.Path {
				t.Errorf("The (%v) chunk should be request path. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 2 && keyChunk != "bro" {
				t.Errorf("The (%v) chunk should be request username. KeyChunk: %v", i+1, keyChunk)
			}
		}
	}
}

func TestCustomHeadersBuildKeys(t *testing.T) {
	limiter := NewLimiter(1, time.Second)
	limiter.Headers = make(map[string][]string)
	limiter.Headers["X-Auth-Token"] = []string{"totally-top-secret", "another-secret"}

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		t.Errorf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")
	request.Header.Set("X-Auth-Token", "totally-top-secret")

	for _, keys := range BuildKeys(limiter, request) {
		if len(keys) != 4 {
			t.Errorf("Keys should be made of 4 parts. Keys: %v", keys)
		}
		for i, keyChunk := range keys {
			if i == 0 && keyChunk != request.Header.Get("X-Real-IP") {
				t.Errorf("The (%v) chunk should be remote IP. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 1 && keyChunk != request.URL.Path {
				t.Errorf("The (%v) chunk should be request path. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 2 && keyChunk != "X-Auth-Token" {
				t.Errorf("The (%v) chunk should be request header. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 3 && (keyChunk != "totally-top-secret" && keyChunk != "another-secret") {
				t.Errorf("The (%v) chunk should be request path. KeyChunk: %v", i+1, keyChunk)
			}
		}
	}
}

func TestRequestMethodBuildKeys(t *testing.T) {
	limiter := NewLimiter(1, time.Second)
	limiter.Methods = []string{"GET"}

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		t.Errorf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")

	for _, keys := range BuildKeys(limiter, request) {
		if len(keys) != 3 {
			t.Errorf("Keys should be made of 3 parts. Keys: %v", keys)
		}
		for i, keyChunk := range keys {
			if i == 0 && keyChunk != request.Header.Get("X-Real-IP") {
				t.Errorf("The (%v) chunk should be remote IP. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 1 && keyChunk != request.URL.Path {
				t.Errorf("The (%v) chunk should be request path. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 2 && keyChunk != "GET" {
				t.Errorf("The (%v) chunk should be request method. KeyChunk: %v", i+1, keyChunk)
			}
		}
	}
}

func TestRequestMethodAndCustomHeadersBuildKeys(t *testing.T) {
	limiter := NewLimiter(1, time.Second)
	limiter.Methods = []string{"GET"}
	limiter.Headers = make(map[string][]string)
	limiter.Headers["X-Auth-Token"] = []string{"totally-top-secret", "another-secret"}

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		t.Errorf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")
	request.Header.Set("X-Auth-Token", "totally-top-secret")

	for _, keys := range BuildKeys(limiter, request) {
		if len(keys) != 5 {
			t.Errorf("Keys should be made of 4 parts. Keys: %v", keys)
		}
		for i, keyChunk := range keys {
			if i == 0 && keyChunk != request.Header.Get("X-Real-IP") {
				t.Errorf("The (%v) chunk should be remote IP. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 1 && keyChunk != request.URL.Path {
				t.Errorf("The (%v) chunk should be request path. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 2 && keyChunk != "GET" {
				t.Errorf("The (%v) chunk should be request method. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 3 && keyChunk != "X-Auth-Token" {
				t.Errorf("The (%v) chunk should be request header. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 4 && (keyChunk != "totally-top-secret" && keyChunk != "another-secret") {
				t.Errorf("The (%v) chunk should be request path. KeyChunk: %v", i+1, keyChunk)
			}
		}
	}
}

func TestRequestMethodAndBasicAuthUsersBuildKeys(t *testing.T) {
	limiter := NewLimiter(1, time.Second)
	limiter.Methods = []string{"GET"}
	limiter.BasicAuthUsers = []string{"bro"}

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		t.Errorf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")
	request.SetBasicAuth("bro", "tato")

	for _, keys := range BuildKeys(limiter, request) {
		if len(keys) != 4 {
			t.Errorf("Keys should be made of 4 parts. Keys: %v", keys)
		}
		for i, keyChunk := range keys {
			if i == 0 && keyChunk != request.Header.Get("X-Real-IP") {
				t.Errorf("The (%v) chunk should be remote IP. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 1 && keyChunk != request.URL.Path {
				t.Errorf("The (%v) chunk should be request path. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 2 && keyChunk != "GET" {
				t.Errorf("The (%v) chunk should be request method. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 3 && keyChunk != "bro" {
				t.Errorf("The (%v) chunk should be basic auth user. KeyChunk: %v", i+1, keyChunk)
			}
		}
	}
}

func TestRequestMethodCustomHeadersAndBasicAuthUsersBuildKeys(t *testing.T) {
	limiter := NewLimiter(1, time.Second)
	limiter.Methods = []string{"GET"}
	limiter.Headers = make(map[string][]string)
	limiter.Headers["X-Auth-Token"] = []string{"totally-top-secret", "another-secret"}
	limiter.BasicAuthUsers = []string{"bro"}

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		t.Errorf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")
	request.Header.Set("X-Auth-Token", "totally-top-secret")
	request.SetBasicAuth("bro", "tato")

	for _, keys := range BuildKeys(limiter, request) {
		if len(keys) != 6 {
			t.Errorf("Keys should be made of 4 parts. Keys: %v", keys)
		}
		for i, keyChunk := range keys {
			if i == 0 && keyChunk != request.Header.Get("X-Real-IP") {
				t.Errorf("The (%v) chunk should be remote IP. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 1 && keyChunk != request.URL.Path {
				t.Errorf("The (%v) chunk should be request path. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 2 && keyChunk != "GET" {
				t.Errorf("The (%v) chunk should be request method. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 3 && keyChunk != "X-Auth-Token" {
				t.Errorf("The (%v) chunk should be request header. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 4 && (keyChunk != "totally-top-secret" && keyChunk != "another-secret") {
				t.Errorf("The (%v) chunk should be request path. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 5 && keyChunk != "bro" {
				t.Errorf("The (%v) chunk should be basic auth user. KeyChunk: %v", i+1, keyChunk)
			}
		}
	}

}
