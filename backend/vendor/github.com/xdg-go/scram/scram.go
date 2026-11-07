// Copyright 2018 by David A. Golden. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package scram

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"hash"

	"github.com/xdg-go/stringprep"
)

// HashGeneratorFcn abstracts a factory function that returns a hash.Hash
// value to be used for SCRAM operations.  Generally, one would use the
// provided package variables, `scram.SHA1` and `scram.SHA256`, for the most
// common forms of SCRAM.
type HashGeneratorFcn func() hash.Hash

// SHA1 is a function that returns a crypto/sha1 hasher and should be used to
// create Client objects configured for SHA-1 hashing.
var SHA1 HashGeneratorFcn = func() hash.Hash { return sha1.New() }

// SHA256 is a function that returns a crypto/sha256 hasher and should be used
// to create Client objects configured for SHA-256 hashing.
var SHA256 HashGeneratorFcn = func() hash.Hash { return sha256.New() }

// SHA512 is a function that returns a crypto/sha512 hasher and should be used
// to create Client objects configured for SHA-512 hashing.
var SHA512 HashGeneratorFcn = func() hash.Hash { return sha512.New() }

// NewClient constructs a SCRAM client component based on a given hash.Hash
// factory receiver.  This constructor will normalize the username, password
// and authzID via the SASLprep algorithm, as recommended by RFC-5802.  If
// SASLprep fails, the method returns an error.
//
// SASLprep rejects code points unassigned in Unicode 3.2, which includes
// all emoji, and rejects strings that mix right-to-left and left-to-right
// text. Passwords set on systems that skip SASLprep may therefore fail
// here; use NewClientUnprepped for those.
func (f HashGeneratorFcn) NewClient(username, password, authzID string) (*Client, error) {
	var userprep, passprep, authprep string
	var err error

	if userprep, err = stringprep.SASLprep.Prepare(username); err != nil {
		return nil, fmt.Errorf("Error SASLprepping username '%s': %v", username, err)
	}
	if passprep, err = stringprep.SASLprep.Prepare(password); err != nil {
		return nil, passwordPrepError(err)
	}
	if authprep, err = stringprep.SASLprep.Prepare(authzID); err != nil {
		return nil, fmt.Errorf("Error SASLprepping authzID '%s': %v", authzID, err)
	}

	return newClient(userprep, passprep, authprep, f), nil
}

// passwordPrepError reports why SASLprep failed without echoing any part of
// the password; stringprep.Error.Error() includes the offending rune.
func passwordPrepError(err error) error {
	var prepErr stringprep.Error
	if errors.As(err, &prepErr) {
		return fmt.Errorf("Error SASLprepping password: %s", prepErr.Msg)
	}
	return errors.New("Error SASLprepping password")
}

// NewClientUnprepped acts like NewClient, except none of the arguments will
// be normalized via SASLprep.  This is not generally recommended, but is
// provided for users that may have custom normalization needs.
func (f HashGeneratorFcn) NewClientUnprepped(username, password, authzID string) (*Client, error) {
	return newClient(username, password, authzID, f), nil
}

// NewServer constructs a SCRAM server component based on a given hash.Hash
// factory receiver.  To be maximally generic, it uses dependency injection to
// handle credential lookup, which is the process of turning a username string
// into a struct with stored credentials for authentication.
func (f HashGeneratorFcn) NewServer(cl CredentialLookup) (*Server, error) {
	return newServer(cl, f)
}
