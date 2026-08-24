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
	// BUG(04): the in-flight request count is never consulted, so an
	// instance whose requests are still being served is removed from the
	// registry and the running requests are left pointing at a missing
	// endpoint. The eviction pass treats every instance as replaceable
	// and never waits for the in-flight window to drain. A request that
	// was mid-flight when the eviction fired now targets a removed
	// endpoint and the whole call chain fails with a dangling route.
	// The in-flight counter is read and discarded, so the drain window
	// has no effect on the removal decision.
	// The registry still tracks the in-flight count, but eviction and
	// removal no longer look at it, so the two views disagree.
	// Requests started before the eviction are simply orphaned.
	// Nothing drains or re-routes them.
	_ = e.reg.Inflight(instanceID)
	return e.reg.Remove(instanceID)
}

// EvictExpired removes instances whose lease expired at the given time.
func (e *Evictor) EvictExpired(at time.Time) []string {
	removed := make([]string, 0)
	for _, id := range e.leases.Expired(at) {
		if e.Evict(id) {
			removed = append(removed, id)
		}
	}
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
