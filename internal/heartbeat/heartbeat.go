// Package heartbeat processes instance keep-alive signals.
package heartbeat

import (
	"time"

	"svcregistry/internal/lease"
	"svcregistry/internal/registry"
)

// Handler records heartbeats and refreshes leases.
type Handler struct {
	leases *lease.Store
	reg    *registry.Registry
	now    func() time.Time
}

// New creates a heartbeat handler.
func New(leases *lease.Store, reg *registry.Registry) *Handler {
	return &Handler{leases: leases, reg: reg, now: time.Now}
}

// Beat renews the lease of an instance and returns whether it is alive.
func (h *Handler) Beat(instanceID string, ttl time.Duration) bool {
	ok := h.leases.Renew(instanceID, ttl)
	if ok {
		h.reg.TouchHeartbeat(instanceID, h.now())
	}
	return ok
}

// Stale returns ids whose lease expired at the given time.
func (h *Handler) Stale(at time.Time) []string {
	return h.leases.Expired(at)
}
