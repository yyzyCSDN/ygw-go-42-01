package model

import "time"

// Lease is the exclusive keep-alive a registered instance holds.
type Lease struct {
	InstanceID string
	Holder     string
	ExpiresAt  time.Time
	Generation uint64
}

// Expired reports whether the lease has passed its deadline.
func (l Lease) Expired(now time.Time) bool {
	return !now.Before(l.ExpiresAt)
}
