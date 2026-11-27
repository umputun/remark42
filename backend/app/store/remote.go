package store

import (
	"errors"
	"fmt"
	"net"

	log "github.com/go-pkgz/lgr"
)

// HideRemoteAddress returns the error of a call to a remote store without the location of that store. A request
// that fails on the network reports the store's url and the address it resolved to, and errors of the stores
// reach the users in error responses. Such an error is logged in full and replaced by one that names the call
// only; errors the store itself returned are kept.
func HideRemoteAddress(method string, err error) error {
	var netErr net.Error // the *url.Error of http.Client and errors of reading the response
	if !errors.As(err, &netErr) {
		return err
	}
	log.Printf("[WARN] remote store call %s failed, %v", method, err)
	return fmt.Errorf("remote call failed for %s", method)
}
