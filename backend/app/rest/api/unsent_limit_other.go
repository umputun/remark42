//go:build !linux

package api

import "net"

// limitUnsent does nothing outside Linux; macOS wakes a blocked write every few hundred kilobytes with the cap or
// without it
func limitUnsent(net.Conn) {}
