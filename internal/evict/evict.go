// Package evict reclaims expired, unhealthy and deactivated instances.
package evict

import (
	"time"

	"svcregistry/internal/health"
	"svcregistry/internal/lease"
	"svcregistry/internal/policy"
	"svcregistry/internal/registry"
	"svcregistry/internal/version"
)

// Evictor removes instances that must leave the registry.
type Evictor struct {
	reg     *registry.Registry
	leases  *lease.Store
	health  *health.Status
	version *version.Store
	policy  policy.EvictionPolicy
	now     func() time.Time
}

// New creates an evictor.
func New(reg *registry.Registry, leases *lease.Store, h *health.Status, v *version.Store) *Evictor {
	return &Evictor{reg: reg, leases: leases, health: h, version: v,
		policy: policy.DefaultEvictionPolicy(), now: time.Now}
}

// WithPolicy configures the eviction policy. Used to raise the in-flight
// threshold above the default of zero.
func (e *Evictor) WithPolicy(p policy.EvictionPolicy) *Evictor {
	e.policy = p
	return e
}

// Evict removes an instance only once it has no in-flight requests. An instance
// whose requests are still being served is kept until the next eviction pass so
// running requests are not left pointing at a removed endpoint.
func (e *Evictor) Evict(instanceID string) bool {
	if !e.policy.AllowEvict(e.reg.Inflight(instanceID)) {
		// In-flight requests are still being served. Defer the removal to a
		// later scan pass instead of orphaning the active call chain.
		return false
	}
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
