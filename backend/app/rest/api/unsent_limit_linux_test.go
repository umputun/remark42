package api

import (
	"context"
	"crypto/tls"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

// the server Run starts caps the unsent queue on every connection it accepts, TLS ones included
func TestRest_makeHTTPServerLimitsUnsent(t *testing.T) {
	srv := (&Rest{}).makeHTTPServer("127.0.0.1", 0, nil)
	for name, wrap := range map[string]func(net.Conn) net.Conn{
		"tcp": func(c net.Conn) net.Conn { return c },
		"tls": func(c net.Conn) net.Conn { return tls.Server(c, &tls.Config{}) },
	} {
		t.Run(name, func(t *testing.T) {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer ln.Close()
			client, err := net.Dial("tcp", ln.Addr().String())
			require.NoError(t, err)
			defer client.Close()
			conn, err := ln.Accept()
			require.NoError(t, err)
			defer conn.Close()

			srv.ConnContext(context.Background(), wrap(conn))

			raw, err := conn.(*net.TCPConn).SyscallConn()
			require.NoError(t, err)
			var got int
			var optErr error
			require.NoError(t, raw.Control(func(fd uintptr) {
				got, optErr = unix.GetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_NOTSENT_LOWAT)
			}))
			require.NoError(t, optErr)
			assert.Equal(t, unsentLimit, got)
		})
	}
}
