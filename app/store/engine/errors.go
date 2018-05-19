package engine

import (
	"errors"
)

var (
	// ErrBucketDoesNotExists reports that bucket not exists
	ErrBucketDoesNotExists = errors.New("bucket does not exists")
	// ErrDBDoesNotExists reports that database not exists
	ErrDBDoesNotExists = errors.New("database does not exists")
)
