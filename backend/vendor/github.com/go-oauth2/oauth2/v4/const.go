package oauth2

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

// ResponseType the type of authorization request
type ResponseType string

// define the type of authorization request
const (
	Code  ResponseType = "code"
	Token ResponseType = "token"
)

func (rt ResponseType) String() string {
	return string(rt)
}

// GrantType authorization model
type GrantType string

// define authorization model
const (
	AuthorizationCode   GrantType = "authorization_code"
	PasswordCredentials GrantType = "password"
	ClientCredentials   GrantType = "client_credentials"
	Refreshing          GrantType = "refresh_token"
	Implicit            GrantType = "__implicit"
)

func (gt GrantType) String() string {
	if gt == AuthorizationCode ||
		gt == PasswordCredentials ||
		gt == ClientCredentials ||
		gt == Refreshing {
		return string(gt)
	}
	return ""
}

// CodeChallengeMethod PCKE method
type CodeChallengeMethod string

const (
	// CodeChallengePlain PCKE Method
	CodeChallengePlain CodeChallengeMethod = "plain"
	// CodeChallengeS256 PCKE Method
	CodeChallengeS256 CodeChallengeMethod = "S256"
)

func (ccm CodeChallengeMethod) String() string {
	if ccm == CodeChallengePlain ||
		ccm == CodeChallengeS256 {
		return string(ccm)
	}
	return ""
}

// Validate code challenge
func (ccm CodeChallengeMethod) Validate(cc, ver string) bool {
	switch ccm {
	case CodeChallengePlain:
		return cc == ver
	case CodeChallengeS256:
		s256 := sha256.Sum256([]byte(ver))
		// trim padding
		a := strings.TrimRight(base64.URLEncoding.EncodeToString(s256[:]), "=")
		b := strings.TrimRight(cc, "=")
		return a == b
	default:
		return false
	}
}
