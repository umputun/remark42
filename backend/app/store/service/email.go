package service

import (
	"context"

	"github.com/umputun/remark/backend/app/store"
)

// VerificationDestination defines interface for destination service which supports verification sending
type VerificationDestination interface {
	SendVerification(ctx context.Context, req VerificationRequest) error
}

type VerificationRequest struct {
	Locator store.Locator
	User    string
	Email   string
	Token   string
}
