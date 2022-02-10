package oauth2

import "context"

type (
	// ClientStore the client information storage interface
	ClientStore interface {
		// according to the ID for the client information
		GetByID(ctx context.Context, id string) (ClientInfo, error)
	}

	// TokenStore the token information storage interface
	TokenStore interface {
		// create and store the new token information
		Create(ctx context.Context, info TokenInfo) error

		// delete the authorization code
		RemoveByCode(ctx context.Context, code string) error

		// use the access token to delete the token information
		RemoveByAccess(ctx context.Context, access string) error

		// use the refresh token to delete the token information
		RemoveByRefresh(ctx context.Context, refresh string) error

		// use the authorization code for token information data
		GetByCode(ctx context.Context, code string) (TokenInfo, error)

		// use the access token for token information data
		GetByAccess(ctx context.Context, access string) (TokenInfo, error)

		// use the refresh token for token information data
		GetByRefresh(ctx context.Context, refresh string) (TokenInfo, error)
	}
)
