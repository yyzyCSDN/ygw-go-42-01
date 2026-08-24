// Package quota enforces per-service registration limits.
package quota

import (
	"sync"
	"time"
)

// Bucket is a token bucket for registration slots.
type Bucket struct {
	mu     sync.Mutex
	rate   int
	burst  int
	tokens int
	last   time.Time
}

// NewBucket creates a token bucket.
func NewBucket(rate, burst int) *Bucket {
	if burst <= 0 {
		burst = rate
	}
	return &Bucket{rate: rate, burst: burst, tokens: burst, last: time.Now()}
}

// Allow consumes one token if available.
func (b *Bucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += int(elapsed * float64(b.rate))
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	b.last = now
	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

// Refill restores the bucket to full.
func (b *Bucket) Refill() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tokens = b.burst
}
