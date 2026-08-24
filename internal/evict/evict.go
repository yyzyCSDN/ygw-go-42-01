// Package evict reclaims expired, unhealthy and deactivated instances.
package evict

import (
	"time"

	"svcregistry/internal/health"
	"svcregistry/internal/lease"
	"svcregistry/internal/registry"
	"svcregistry/internal/version"
)

// Evictor removes instances that must leave the registry.
type Evictor struct {
	reg     *registry.Registry
	leases  *lease.Store
	health  *health.Status
	version *version.Store
	now     func() time.Time
}

// New creates an evictor.
func New(reg *registry.Registry, leases *lease.Store, h *health.Status, v *version.Store) *Evictor {
	return &Evictor{reg: reg, leases: leases, health: h, version: v, now: time.Now}
}

// Evict removes an instance unless it has in-flight requests.
func (e *Evictor) Evict(instanceID string) bool {
	if e.reg.Inflight(instanceID) > 0 {
		return false
	}
	return e.reg.Remove(instanceID)
}

// EvictExpired removes instances whose lease expired at the given time.
func (e *Evictor) EvictExpired(at time.Time) []string {
	removed := make([]string, 0)
	// BUG(07): expired leases are collected but never mapped to instance
	// removal, so a dead instance stays in the registry and discovery
	// keeps returning it as an active endpoint. The scan below only
	// counts how many leases expired; the registry membership is left
	// untouched and the lease store keeps the expired entries until a
	// full restart. Callers keep dialing a dead endpoint indefinitely,
	// and the stale entry also consumes a registration slot that should
	// have been freed for a replacement instance.
	// The expired scan's result is discarded, so nothing downstream
	// learns about the dead leases at all.
	// The registry and the lease table therefore drift apart with no
	// reconciler to bring them back together.
	expired := e.leases.Expired(at)
	_ = expired
	return removed
}

// EvictUnhealthy removes instances whose merged health is not OK.
func (e *Evictor) EvictUnhealthy() []string {
	removed := make([]string, 0)
	for _, svc := range e.reg.Services() {
		for _, inst := range e.reg.Instances(svc) {
			if !e.health.Snapshot(inst.ID).OK && e.Evict(inst.ID) {
				removed = append(removed, inst.ID)
			}
		}
	}
	return removed
}

// EvictInactiveVersion removes confirmed instances of deactivated versions.
func (e *Evictor) EvictInactiveVersion() []string {
	removed := make([]string, 0)
	for _, svc := range e.reg.Services() {
		snap := e.version.Snapshot(svc)
		for _, inst := range e.reg.Instances(svc) {
			if snap.Active[inst.Version] {
				continue
			}
			if !e.reg.Confirmed(inst.ID) {
				continue
			}
			if e.Evict(inst.ID) {
				removed = append(removed, inst.ID)
			}
		}
	}
	return removed
}
