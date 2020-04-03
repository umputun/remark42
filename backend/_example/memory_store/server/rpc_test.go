/*
 * Copyright 2020 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package server

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/go-pkgz/jrpc"
	log "github.com/go-pkgz/lgr"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/memory_store/accessor"
)

func chooseRandomUnusedPort() (port int) {
	for i := 0; i < 10; i++ {
		port = 40000 + int(rand.Int31n(10000))
		if ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port)); err == nil {
			_ = ln.Close()
			break
		}
	}
	return port
}

func waitForHTTPServerStart(port int) {
	// wait for up to 3 seconds for server to start before returning it
	client := http.Client{Timeout: time.Second}
	for i := 0; i < 300; i++ {
		time.Sleep(time.Millisecond * 10)
		if resp, err := client.Get(fmt.Sprintf("http://localhost:%d", port)); err == nil {
			_ = resp.Body.Close()
			return
		}
	}
}

func prepTestStore(t *testing.T) (s *RPC, port int, teardown func()) {
	mg := accessor.NewMemData()
	adm := accessor.NewMemAdminStore("secret")
	img := accessor.NewMemImageStore()
	s = NewRPC(mg, adm, img, &jrpc.Server{API: "/test", Logger: jrpc.NoOpLogger})

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

	port = chooseRandomUnusedPort()
	go func() {
		log.Printf("%v", s.Run(port))
	}()

	waitForHTTPServerStart(port)

	return s, port, func() {
		require.NoError(t, s.Shutdown())
	}
}
