package api

import (
	"crypto/tls"
	"net"

	"golang.org/x/sys/unix"
)

// unsentLimit is the most of a response the kernel holds unsent for a connection
const unsentLimit = 128 * 1024

// limitUnsent caps how much of a response the kernel queues unsent on the connection. Without the cap Linux grows
// the send buffer to megabytes and wakes a write blocked on a slow client only once a large part of it has drained,
// which at a few kilobytes a second takes longer than writeStallTimeout although the client keeps reading
func limitUnsent(c net.Conn) {
	if tc, ok := c.(*tls.Conn); ok {
		c = tc.NetConn()
	}
	tcp, ok := c.(*net.TCPConn)
	if !ok {
		return
	}
	raw, err := tcp.SyscallConn()
	if err != nil {
		return
	}
	_ = raw.Control(func(fd uintptr) {
		_ = unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_NOTSENT_LOWAT, unsentLimit)
	})
}
