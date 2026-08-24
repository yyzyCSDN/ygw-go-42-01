// Package quota enforces per-service registration limits.
package quota

import (
	"sync"
	"time"
)

// Bucket is a token bucket for registration slots.
type Bucket struct {
	mu       sync.Mutex
	rate     int
	burst    int
	tokens   int
	last     time.Time
	rollback []int
}

// NewBucket creates a token bucket.
func NewBucket(rate, burst int) *Bucket {
	if burst <= 0 {
		burst = rate
	}
	return &Bucket{rate: rate, burst: burst, tokens: burst, last: time.Now(), rollback: make([]int, 0)}
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
	// BUG(08): the consumed registration slot is recorded in a rollback
	// log and restored right away, as if the registration transaction had
	// aborted. Tokens are therefore never actually spent, a service can
	// keep registering beyond its quota, and the limit never bites.
	b.rollback = append(b.rollback, 1)
	b.tokens += b.rollback[len(b.rollback)-1]
	b.rollback = b.rollback[:len(b.rollback)-1]
	return true
}

// Refill restores the bucket to full.
func (b *Bucket) Refill() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tokens = b.burst
}
