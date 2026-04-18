package safehttp

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsPrivateIP(t *testing.T) {
	tbl := []struct {
		ip      string
		private bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"192.168.0.1", true},
		{"192.168.255.255", true},
		{"169.254.1.1", true},
		{"100.64.0.1", true},
		{"100.127.255.255", true},
		{"::1", true},
		{"fc00::1", true},
		{"fe80::1", true},
		{"0.0.0.0", true},
		{"::", true},
		{"8.8.8.8", false},
		{"203.0.113.1", false},
		{"1.1.1.1", false},
		{"2001:db8::1", false},
	}

	for _, tt := range tbl {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			require.NotNil(t, ip)
			assert.Equal(t, tt.private, IsPrivateIP(ip))
		})
	}
}

func TestTransport_BlocksPrivate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{Transport: Transport(), Timeout: 2 * time.Second}
	resp, err := client.Get(srv.URL) // httptest.NewServer binds 127.0.0.1
	if resp != nil {
		_ = resp.Body.Close()
	}
	require.Error(t, err, "private address must be refused")
	assert.Contains(t, err.Error(), "access to private address is not allowed")
}

func TestTransport_AllowsPublic(t *testing.T) {
	tr := Transport()
	// the policy check must reject the loopback literal
	_, err := tr.DialContext(context.Background(), "tcp", "127.0.0.1:1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access to private address is not allowed")

	// public IP literal passes the policy check; bound the dial with a tight context
	// so the test does not depend on real-world routing of TEST-NET-3 (203.0.113.0/24).
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err = tr.DialContext(ctx, "tcp", "203.0.113.1:1")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "access to private address is not allowed")
}

func TestTransport_PreservesDefaultTransportSettings(t *testing.T) {
	def := http.DefaultTransport.(*http.Transport)
	tr := Transport()
	assert.NotNil(t, tr.Proxy, "Proxy must be inherited from http.DefaultTransport")
	assert.Equal(t, def.ForceAttemptHTTP2, tr.ForceAttemptHTTP2, "ForceAttemptHTTP2")
	assert.Equal(t, def.MaxIdleConns, tr.MaxIdleConns, "MaxIdleConns")
	assert.Equal(t, def.IdleConnTimeout, tr.IdleConnTimeout, "IdleConnTimeout")
	assert.Equal(t, def.TLSHandshakeTimeout, tr.TLSHandshakeTimeout, "TLSHandshakeTimeout")
	assert.Equal(t, def.ExpectContinueTimeout, tr.ExpectContinueTimeout, "ExpectContinueTimeout")
}
