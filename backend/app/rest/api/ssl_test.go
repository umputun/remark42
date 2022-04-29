package api

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSL_Redirect(t *testing.T) {
	rest := Rest{RemarkURL: "https://localhost:443"}

	ts := httptest.NewServer(rest.httpToHTTPSRouter())
	defer ts.Close()

	client := http.Client{
		// prevent http redirect
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},

		// allow self-signed certificate
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	defer client.CloseIdleConnections()

	// check http to https redirect response
	resp, err := client.Get(ts.URL + "/blah?param=1")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	assert.Equal(t, "https://localhost:443/blah?param=1", resp.Header.Get("Location"))
}

func TestSSL_ACME_HTTPChallengeRouter(t *testing.T) {
	rest := Rest{
		RemarkURL: "https://localhost:443",
		SSLConfig: SSLConfig{
			ACMELocation: "acme",
		},
	}

	m := rest.makeAutocertManager()
	defer os.RemoveAll(rest.SSLConfig.ACMELocation)

	ts := httptest.NewServer(rest.httpChallengeRouter(m))
	defer ts.Close()

	client := http.Client{
		// prevent http redirect
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	defer client.CloseIdleConnections()

	// check http to https redirect response
	resp, err := client.Get(ts.URL + "/blah?param=1")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	assert.Equal(t, "https://localhost:443/blah?param=1", resp.Header.Get("Location"))

	// check acme http challenge
	req, err := http.NewRequest("GET", ts.URL+"/.well-known/acme-challenge/token123", http.NoBody)
	require.NoError(t, err)
	req.Host = "localhost" // for passing hostPolicy check
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	err = m.Cache.Put(context.Background(), "token123+http-01", []byte("token"))
	assert.NoError(t, err)

	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "token", string(body))
}
