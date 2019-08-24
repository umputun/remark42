package api

import (
	"context"
	"crypto/tls"
	"io/ioutil"
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

	// check http to https redirect response
	resp, err := client.Get(ts.URL + "/blah?param=1")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 307, resp.StatusCode)
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

	// check http to https redirect response
	resp, err := client.Get(ts.URL + "/blah?param=1")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 307, resp.StatusCode)
	assert.Equal(t, "https://localhost:443/blah?param=1", resp.Header.Get("Location"))

	// check acme http challenge
	req, err := http.NewRequest("GET", ts.URL+"/.well-known/acme-challenge/token123", nil)
	require.Nil(t, err)
	req.Host = "localhost" // for passing hostPolicy check
	resp, err = client.Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 404, resp.StatusCode)

	err = m.Cache.Put(context.Background(), "token123+http-01", []byte("token"))
	assert.Nil(t, err)

	resp, err = client.Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)
	assert.Equal(t, "token", string(body))
}
