// Package policy centralizes heartbeat and eviction tuning knobs.
package policy

import "time"

// HeartbeatPolicy configures lease and stale detection.
type HeartbeatPolicy struct {
	LeaseTTL  time.Duration
	RenewGap  time.Duration
	StaleAfter time.Duration
}

// DefaultHeartbeatPolicy returns standard heartbeat settings.
func DefaultHeartbeatPolicy() HeartbeatPolicy {
	return HeartbeatPolicy{LeaseTTL: 30 * time.Second, RenewGap: 10 * time.Second, StaleAfter: 45 * time.Second}
}

// Valid reports whether the policy has sane values.
func (p HeartbeatPolicy) Valid() bool {
	return p.LeaseTTL > 0 && p.RenewGap > 0 && p.StaleAfter > p.LeaseTTL
}

// EvictionPolicy configures the eviction pass.
type EvictionPolicy struct {
	MinInflight   int
	MaxPerScan    int
	QuarantineLen int
}

// DefaultEvictionPolicy returns standard eviction settings.
func DefaultEvictionPolicy() EvictionPolicy {
	return EvictionPolicy{MinInflight: 0, MaxPerScan: 100, QuarantineLen: 256}
}

// AllowEvict reports whether an instance may be evicted given its in-flight load.
func (p EvictionPolicy) AllowEvict(inflight int) bool {
	return inflight <= p.MinInflight
}
