package store

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHideRemoteAddress(t *testing.T) {
	assert.NoError(t, HideRemoteAddress("store.get", nil))

	storeErr := errors.New("comment not found")
	assert.Equal(t, storeErr, HideRemoteAddress("store.get", storeErr), "an error of the store is kept")

	client := http.Client{Transport: &http.Transport{DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
		return nil, &net.OpError{Op: "dial", Net: network, Addr: &net.TCPAddr{IP: net.IPv4(10, 1, 2, 3), Port: 9191},
			Err: errors.New("connect: connection refused")}
	}}}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://store.internal:9191/rpc", http.NoBody)
	require.NoError(t, err)
	resp, err := client.Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	require.Error(t, err)
	require.Contains(t, err.Error(), "http://store.internal:9191/rpc")
	require.Contains(t, err.Error(), "10.1.2.3:9191")
	assert.EqualError(t, HideRemoteAddress("store.get", err), "remote call failed for store.get")
}
