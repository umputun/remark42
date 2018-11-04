package api

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
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
