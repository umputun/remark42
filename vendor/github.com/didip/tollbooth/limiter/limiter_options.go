package limiter

import (
	"time"
)

type ExpirableOptions struct {
	DefaultExpirationTTL time.Duration

	// How frequently expire job triggers
	ExpireJobInterval time.Duration
}
