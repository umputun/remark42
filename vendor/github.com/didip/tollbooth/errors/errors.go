// Package errors provide data structure for errors.
package errors

import "fmt"

// HTTPError is an error struct that returns both message and status code.
type HTTPError struct {
	Message    string
	StatusCode int
}

// Error returns error message.
func (httperror *HTTPError) Error() string {
	return fmt.Sprintf("%v: %v", httperror.StatusCode, httperror.Message)
}
