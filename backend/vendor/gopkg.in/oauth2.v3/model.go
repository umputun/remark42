package oauth2

import (
	"time"
)

type (
	// ClientInfo the client information model interface
	ClientInfo interface {
		GetID() string
		GetSecret() string
		GetDomain() string
		GetUserID() string
	}

	// TokenInfo the token information model interface
	TokenInfo interface {
		New() TokenInfo

		GetClientID() string
		SetClientID(string)
		GetUserID() string
		SetUserID(string)
		GetRedirectURI() string
		SetRedirectURI(string)
		GetScope() string
		SetScope(string)

		GetCode() string
		SetCode(string)
		GetCodeCreateAt() time.Time
		SetCodeCreateAt(time.Time)
		GetCodeExpiresIn() time.Duration
		SetCodeExpiresIn(time.Duration)

		GetAccess() string
		SetAccess(string)
		GetAccessCreateAt() time.Time
		SetAccessCreateAt(time.Time)
		GetAccessExpiresIn() time.Duration
		SetAccessExpiresIn(time.Duration)

		GetRefresh() string
		SetRefresh(string)
		GetRefreshCreateAt() time.Time
		SetRefreshCreateAt(time.Time)
		GetRefreshExpiresIn() time.Duration
		SetRefreshExpiresIn(time.Duration)
	}
)
