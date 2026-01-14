package repeater

import (
	"math/rand"
	"time"
)

// Strategy defines how to calculate delays between retries
type Strategy interface {
	// NextDelay returns delay for the next attempt, attempt starts from 1
	NextDelay(attempt int) time.Duration
}

// FixedDelay implements fixed time delay between attempts
type FixedDelay struct {
	Delay time.Duration
}

// NewFixedDelay creates a new FixedDelay strategy
func NewFixedDelay(delay time.Duration) FixedDelay {
	return FixedDelay{Delay: delay}
}

// NextDelay returns fixed delay
func (s FixedDelay) NextDelay(_ int) time.Duration {
	return s.Delay
}

// BackoffType represents the backoff strategy type
type BackoffType int

const (
	// BackoffConstant keeps delays the same between attempts
	BackoffConstant BackoffType = iota
	// BackoffLinear increases delays linearly between attempts
	BackoffLinear
	// BackoffExponential increases delays exponentially between attempts
	BackoffExponential
)

// backoff implements various backoff strategies with optional jitter
type backoff struct {
	initial  time.Duration
	maxDelay time.Duration
	btype    BackoffType
	jitter   float64
}

type backoffOption func(*backoff)

// WithMaxDelay sets maximum delay for the backoff strategy
func WithMaxDelay(d time.Duration) backoffOption { //nolint:revive // unexported type is used in the same package
	return func(b *backoff) {
		b.maxDelay = d
	}
}

// WithBackoffType sets backoff type for the strategy
func WithBackoffType(t BackoffType) backoffOption { //nolint:revive // unexported type is used in the same package
	return func(b *backoff) {
		b.btype = t
	}
}

// WithJitter sets jitter factor for the backoff strategy
func WithJitter(factor float64) backoffOption { //nolint:revive // unexported type is used in the same package
	return func(b *backoff) {
		b.jitter = factor
	}
}

func newBackoff(initial time.Duration, opts ...backoffOption) *backoff {
	b := &backoff{
		initial:  initial,
		maxDelay: 30 * time.Second,
		btype:    BackoffExponential,
		jitter:   0.1,
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// NextDelay returns delay for the next attempt
func (s backoff) NextDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	var delay time.Duration
	switch s.btype {
	case BackoffConstant:
		delay = s.initial
	case BackoffLinear:
		delay = s.initial * time.Duration(attempt)
	case BackoffExponential:
		delay = s.initial * time.Duration(1<<(attempt-1))
	}

	if s.maxDelay > 0 && delay > s.maxDelay {
		delay = s.maxDelay
	}

	if s.jitter > 0 {
		jitter := float64(delay) * s.jitter
		delay = time.Duration(float64(delay) + (rand.Float64()*jitter - jitter/2)) //nolint:gosec // no need for secure random here
	}

	return delay
}
