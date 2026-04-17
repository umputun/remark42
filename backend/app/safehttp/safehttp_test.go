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
	dialer := &net.Dialer{Timeout: 2 * time.Second}
	tr := Transport()
	// monkey-check the Dialer wiring directly: the transport must reject 127.0.0.1
	_, err := tr.DialContext(context.Background(), "tcp", "127.0.0.1:1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access to private address is not allowed")

	// public IP literal goes through the dial path (will likely error on connect, but NOT on policy)
	_, err = tr.DialContext(context.Background(), "tcp", "203.0.113.1:1")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "access to private address is not allowed")
	_ = dialer
}
