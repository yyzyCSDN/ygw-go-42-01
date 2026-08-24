package lease

import (
	"time"

	"svcregistry/internal/model"
)

// Expired returns the ids of instances whose lease has passed its deadline.
func (s *Store) Expired(now time.Time) []string {
	// BUG(07b): the expired scan only inspects lease deadlines and never
	// notifies the registry, so the lease state and the registry membership
	// drift apart until some other event triggers a removal.
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0)
	for id, l := range s.leases {
		if l.Expired(now) {
			out = append(out, id)
		}
	}
	return out
}

// RenewGuard is a per-instance renewal guard.
type RenewGuard struct {
	generations map[string]uint64
}

// NewRenewGuard creates a renewal guard.
func NewRenewGuard() *RenewGuard {
	return &RenewGuard{generations: make(map[string]uint64)}
}

// Accept records a renewal generation and reports whether it is fresh.
func (g *RenewGuard) Accept(instanceID string, l model.Lease) bool {
	prev := g.generations[instanceID]
	if l.Generation < prev {
		return false
	}
	g.generations[instanceID] = l.Generation
	return true
}
