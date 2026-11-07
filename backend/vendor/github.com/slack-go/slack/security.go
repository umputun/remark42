package slack

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Signature headers
const (
	hSignature = "X-Slack-Signature"
	hTimestamp = "X-Slack-Request-Timestamp"
)

// SecretsVerifier contains the information needed to verify that the request comes from Slack
type SecretsVerifier struct {
	d         Debug
	signature []byte
	hmac      hash.Hash
}

func signatureVerifierWithoutTimestampCheck(header http.Header, secret string) (_ SecretsVerifier, err error) {
	var (
		bsignature []byte
	)

	if secret == "" {
		return SecretsVerifier{}, ErrInvalidConfiguration
	}

	signature := header.Get(hSignature)
	stimestamp := header.Get(hTimestamp)

	if signature == "" || stimestamp == "" {
		return SecretsVerifier{}, ErrMissingHeaders
	}

	signature, ok := strings.CutPrefix(signature, "v0=")
	if !ok {
		return SecretsVerifier{}, ErrInvalidSignature
	}
	if bsignature, err = hex.DecodeString(signature); err != nil {
		return SecretsVerifier{}, err
	}
	if len(bsignature) != sha256.Size {
		return SecretsVerifier{}, ErrInvalidSignature
	}

	hash := hmac.New(sha256.New, []byte(secret))
	if _, err = fmt.Fprintf(hash, "v0:%s:", stimestamp); err != nil {
		return SecretsVerifier{}, err
	}

	return SecretsVerifier{
		signature: bsignature,
		hmac:      hash,
	}, nil
}

// NewSecretsVerifier creates a verifier for Slack's v0 request signatures and
// rejects timestamps more than five minutes from local time. Write the raw request
// body to the verifier, then call Ensure to check the signature.
//
// Verification does not deduplicate requests: the same valid request can pass
// again within the timestamp window. Events API consumers should deduplicate by
// event_id before performing side effects. X-Slack-Retry-Num describes a retry
// attempt; it is not an event identifier.
//
// Slack API docs: https://docs.slack.dev/authentication/verifying-requests-from-slack/
func NewSecretsVerifier(header http.Header, secret string) (sv SecretsVerifier, err error) {
	var (
		timestamp int64
	)

	stimestamp := header.Get(hTimestamp)

	if sv, err = signatureVerifierWithoutTimestampCheck(header, secret); err != nil {
		return SecretsVerifier{}, err
	}

	if timestamp, err = strconv.ParseInt(stimestamp, 10, 64); err != nil {
		return SecretsVerifier{}, err
	}

	diff := time.Since(time.Unix(timestamp, 0)).Abs()
	if diff > 5*time.Minute {
		return SecretsVerifier{}, ErrExpiredTimestamp
	}

	return sv, err
}

func (v *SecretsVerifier) WithDebug(d Debug) *SecretsVerifier {
	v.d = d
	return v
}

func (v *SecretsVerifier) Write(body []byte) (n int, err error) {
	return v.hmac.Write(body)
}

// Ensure compares the signature sent from Slack with the actual computed hash to judge validity.
// It returns ErrInvalidSignature if the signatures do not match.
func (v SecretsVerifier) Ensure() error {
	computed := v.hmac.Sum(nil)
	// use hmac.Equal prevent leaking timing information.
	if hmac.Equal(computed, v.signature) {
		return nil
	}
	if v.d != nil && v.d.Debug() {
		// A full computed signature would allow the rejected request to be replayed.
		v.d.Debugf("Invalid signature: received prefix=%x (%d bytes), computed prefix=%x (%d bytes)",
			v.signature[:4], len(v.signature), computed[:4], len(computed))
	}
	return ErrInvalidSignature
}
