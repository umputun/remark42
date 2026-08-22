/*
 * Copyright 2020 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package server

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/go-pkgz/jrpc"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/memory_store/accessor"
)

// chooseUnusedPort asks the kernel for a free port from the ephemeral range, so concurrently
// running package test binaries never land on the same number
func chooseUnusedPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "no free port available")
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())
	return port
}

// waitForHTTPServerStart blocks until the server on port answers, failing the test naming the
// port if it never does
func waitForHTTPServerStart(t *testing.T, port int) {
	t.Helper()
	client := http.Client{Timeout: time.Second}
	defer client.CloseIdleConnections()
	require.Eventually(t, func() bool {
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d", port))
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return true
	}, 30*time.Second, 10*time.Millisecond, "http server on port %d didn't start", port)
}

func prepTestStore(t *testing.T) (port int, teardown func()) {
	mg := accessor.NewMemData()
	adm := accessor.NewMemAdminStore("secret")
	img := accessor.NewMemImageStore()
	s := NewRPC(mg, adm, img, jrpc.NewServer("/test"))

	admRec := accessor.AdminRec{
		SiteID:  "test-site",
		IDs:     []string{"id1", "id2"},
		Email:   "admin@example.com",
		Enabled: true,
	}
	adm.Set("test-site", admRec)

	admRecDisabled := admRec
	admRecDisabled.Enabled = false
	adm.Set("test-site-disabled", admRecDisabled)

	port = chooseUnusedPort(t)
	go func() {
		_ = s.Run(port)
	}()

	waitForHTTPServerStart(t, port)

	return port, func() {
		// every test client here uses http.DefaultTransport, so their keep-alive connections
		// sit in one shared pool; Shutdown waits on them and hits its own 5s deadline otherwise
		http.DefaultTransport.(*http.Transport).CloseIdleConnections()
		require.NoError(t, s.Shutdown())
	}
}
