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
	assert.ErrorIs(t, err, ErrPrivateAddress, "the refusal is found in the client's error chain")
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

// TestIsPrivateIP_RangeBoundaries pins the refused set: the first and last address of every refused
// range are refused, the addresses just outside them are not, and neither are ranges deliberately allowed
func TestIsPrivateIP_RangeBoundaries(t *testing.T) {
	refused := []string{
		"0.0.0.0", "::", "::ffff:0.0.0.0",
		"10.0.0.0", "10.255.255.255", "::ffff:10.0.0.1",
		"172.16.0.0", "172.31.255.255", "::ffff:172.16.0.1",
		"192.168.0.0", "192.168.255.255", "::ffff:192.168.1.1",
		"100.64.0.0", "100.127.255.255", "::ffff:100.64.0.1",
		"127.0.0.0", "127.255.255.255", "::ffff:127.0.0.1",
		"169.254.0.0", "169.254.255.255", "::ffff:169.254.1.1",
		"::1",
		"fc00::", "fdff:ffff:ffff:ffff:ffff:ffff:ffff:ffff",
		"fe80::", "febf:ffff:ffff:ffff:ffff:ffff:ffff:ffff",
	}
	allowed := []string{
		"0.0.0.1", "::2", "::ffff:8.8.8.8",
		"9.255.255.255", "11.0.0.0",
		"172.15.255.255", "172.32.0.0",
		"192.167.255.255", "192.169.0.0",
		"100.63.255.255", "100.128.0.0",
		"126.255.255.255", "128.0.0.0",
		"169.253.255.255", "169.255.0.0",
		"fbff:ffff:ffff:ffff:ffff:ffff:ffff:ffff", "fe00::",
		"fe7f:ffff:ffff:ffff:ffff:ffff:ffff:ffff", "fec0::",
		"::127.0.0.1", "::10.0.0.1", // ipv4-compatible, not ipv4-mapped
		"224.0.0.1", "255.255.255.255", "198.18.0.1", "240.0.0.1", // multicast, broadcast, benchmark, reserved
		"ff02::1", "ff01::1", "2001:db8::1", "64:ff9b::a00:1",
	}
	for _, s := range refused {
		ip := net.ParseIP(s)
		require.NotNil(t, ip, s)
		assert.True(t, IsPrivateIP(ip), "%s must be refused", s)
		if v4 := ip.To4(); v4 != nil {
			assert.True(t, IsPrivateIP(v4), "4-byte %s must be refused", s)
		}
	}
	for _, s := range allowed {
		ip := net.ParseIP(s)
		require.NotNil(t, ip, s)
		assert.False(t, IsPrivateIP(ip), "%s must be allowed", s)
		if v4 := ip.To4(); v4 != nil {
			assert.False(t, IsPrivateIP(v4), "4-byte %s must be allowed", s)
		}
	}
	assert.False(t, IsPrivateIP(nil), "nil")
	assert.False(t, IsPrivateIP(net.IP{1, 2, 3}), "malformed length")
}
