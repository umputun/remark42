package config

import (
	"testing"
	"time"
)

func BenchmarkLimitReached(b *testing.B) {
	limiter := NewLimiter(1, time.Second)
	key := "127.0.0.1|/"

	for i := 0; i < b.N; i++ {
		limiter.LimitReached(key)
	}
}
