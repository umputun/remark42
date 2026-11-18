package safehttp

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
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

// TestTransport_ChecksProxiedTarget runs requests through a forward proxy on a loopback address: the proxy
// itself is reachable even though its address is private, and a private target is refused before the
// proxy sees it, for plain http, for an https CONNECT and after a redirect from a public target
func TestTransport_ChecksProxiedTarget(t *testing.T) {
	proxy := newFakeProxy(t)
	setProxyEnv(t, proxy.URL, proxy.URL)
	client := &http.Client{Transport: Transport(), Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()

	resp, err := client.Get("http://203.0.113.1/page")
	require.NoError(t, err, "a public target is fetched through a proxy on a private address")
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "proxied http://203.0.113.1/page", string(body))

	for _, target := range []string{
		"http://169.254.169.254/latest/meta-data/",
		"http://10.1.2.3/",
		"http://100.64.0.1/",
		"http://[fd00::1]/",
		"https://10.1.2.3/",
		"http://203.0.113.10/redirect", // the proxy answers with a redirect to 169.254.169.254
	} {
		t.Run(target, func(t *testing.T) {
			resp, err := client.Get(target)
			if resp != nil {
				_ = resp.Body.Close()
			}
			require.ErrorIs(t, err, ErrPrivateAddress)
		})
	}
	assert.Equal(t, []string{"GET http://203.0.113.1/page", "GET http://203.0.113.10/redirect"}, proxy.seen(),
		"the proxy gets only the public targets")
}

// TestTransport_ProxyAddressReachedDirectly requests the proxy's own address, which is not proxied because it
// is a loopback address; the dialer's exemption for the proxy must not let such a direct request through
func TestTransport_ProxyAddressReachedDirectly(t *testing.T) {
	proxy := newFakeProxy(t)
	setProxyEnv(t, proxy.URL, "")
	client := &http.Client{Transport: Transport(), Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()

	resp, err := client.Get(proxy.URL + "/internal")
	if resp != nil {
		_ = resp.Body.Close()
	}
	require.ErrorIs(t, err, ErrPrivateAddress)
	assert.Empty(t, proxy.seen())
}

// TestTransport_UnicodeHost checks that an internationalised target name is resolved in its ASCII form, the one
// the pure Go resolver accepts, both when it goes through a proxy and when it is fetched directly
func TestTransport_UnicodeHost(t *testing.T) {
	var mu sync.Mutex
	var looked []string
	orig := lookupIPAddr
	lookupIPAddr = func(_ context.Context, host string) ([]net.IPAddr, error) {
		mu.Lock()
		looked = append(looked, host)
		mu.Unlock()
		for i := range len(host) {
			if host[i] >= 0x80 {
				return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true} // as the pure Go resolver does
			}
		}
		if host == "xn--bcher-kva.example" {
			return []net.IPAddr{{IP: net.ParseIP("203.0.113.1")}}, nil
		}
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}
	t.Cleanup(func() { lookupIPAddr = orig })

	proxy := newFakeProxy(t)
	setProxyEnv(t, proxy.URL, "")
	client := &http.Client{Transport: Transport(), Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
	resp, err := client.Get("http://bücher.example/page")
	require.NoError(t, err, "a unicode host is fetched through the proxy")
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, []string{"GET http://xn--bcher-kva.example/page"}, proxy.seen())

	setProxyEnv(t, "", "")
	direct := &http.Client{Transport: Transport(), Timeout: 5 * time.Second}
	defer direct.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond) // the dial to 203.0.113.1 need not succeed
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://bücher.example/page", http.NoBody)
	require.NoError(t, err)
	resp, err = direct.Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "no such host", "the direct path resolves the ASCII name too")

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"xn--bcher-kva.example", "xn--bcher-kva.example"}, looked, "names given to the resolver")
}

type fakeProxy struct {
	*httptest.Server
	mu   sync.Mutex
	reqs []string
}

// newFakeProxy starts a forward proxy that answers every request itself instead of connecting to the target
func newFakeProxy(t *testing.T) *fakeProxy {
	t.Helper()
	p := &fakeProxy{}
	p.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		p.reqs = append(p.reqs, r.Method+" "+r.URL.String())
		p.mu.Unlock()
		if r.URL.Host == "203.0.113.10" {
			http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
			return
		}
		_, _ = io.WriteString(w, "proxied "+r.URL.String())
	}))
	t.Cleanup(p.Close)
	return p
}

func (p *fakeProxy) seen() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.reqs...)
}

// setProxyEnv sets the proxy variables in both cases and clears NO_PROXY, so values from the developer's shell
// do not leak in
func setProxyEnv(t *testing.T, httpProxy, httpsProxy string) {
	t.Helper()
	for name, val := range map[string]string{"HTTP_PROXY": httpProxy, "HTTPS_PROXY": httpsProxy, "NO_PROXY": "",
		"http_proxy": httpProxy, "https_proxy": httpsProxy, "no_proxy": "", "REQUEST_METHOD": ""} {
		t.Setenv(name, val)
	}
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
