package model

import "time"

// HealthSnapshot is the merged health view of one instance.
type HealthSnapshot struct {
	InstanceID   string
	OK           bool
	PassiveCount int
	UpdatedAt    time.Time
}

// Healthy reports whether the snapshot says the instance is fine.
func (h HealthSnapshot) Healthy() bool {
	return h.OK && h.PassiveCount == 0
}
