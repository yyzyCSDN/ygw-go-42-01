package evict

import (
	"testing"

	"svcregistry/internal/health"
	"svcregistry/internal/lease"
	"svcregistry/internal/model"
	"svcregistry/internal/registry"
	"svcregistry/internal/version"
)

// TestEvictInactiveVersion_KeepsUnconfirmedDuringSwitch reproduces the gray
// rollout defect: when the old version is deactivated, the version cleanup must
// only remove confirmed instances. An in-flight (not yet durably confirmed)
// registration must survive the switch window so the endpoint set does not lose
// entries mid-rollout.
func TestEvictInactiveVersion_KeepsUnconfirmedDuringSwitch(t *testing.T) {
	reg := registry.New()
	leases := lease.New()
	healthStatus := health.New()
	versions := version.New()
	evictor := New(reg, leases, healthStatus, versions)

	// Old version (v1): one confirmed endpoint, one still in flight.
	oldConfirmed := model.NewInstance("old-confirmed", "cart", "v1", "10.0.0.1:8080", 5)
	oldInflight := model.NewInstance("old-inflight", "cart", "v1", "10.0.0.2:8080", 5)
	reg.Register("cart", oldConfirmed)
	reg.Register("cart", oldInflight)
	// Only oldConfirmed is durably committed; oldInflight is still being registered.
	reg.ConfirmDurable("old-confirmed")

	// New version (v2): the gray target, stays active.
	reg.Register("cart", model.NewInstance("new", "cart", "v2", "10.0.0.3:8080", 1))
	reg.ConfirmDurable("new")

	// Both versions active before the switch.
	versions.Ensure("cart", "v1")
	versions.Ensure("cart", "v2")

	// Run the gray rollout to completion: deactivate the old version.
	versions.Deactivate("cart", "v1")

	removed := evictor.EvictInactiveVersion()

	// The confirmed old endpoint drains; the in-flight one must survive.
	if len(removed) != 1 || removed[0] != "old-confirmed" {
		t.Fatalf("expected only old-confirmed to be removed, got %v", removed)
	}

	if _, ok := reg.Get("old-inflight"); !ok {
		t.Fatal("in-flight (unconfirmed) instance was removed during the version switch")
	}
	// Re-running must keep deleting nothing further for the survivor.
	if again := evictor.EvictInactiveVersion(); len(again) != 0 {
		t.Fatalf("re-run should not remove the unconfirmed instance, got %v", again)
	}
	// Even once healthy, an unconfirmed instance is not auto-removed by the
	// version cleanup until confirmed.
	oldInflight.Status = model.InstanceHealthy
	if again := evictor.EvictInactiveVersion(); len(again) != 0 {
		t.Fatalf("healthy-but-unconfirmed instance must survive, got %v", again)
	}
}

// TestEvictInactiveVersion_RemovesConfirmedAfterSwitch confirms that a
// confirmed instance of a deactivated version is still drained.
func TestEvictInactiveVersion_RemovesConfirmedAfterSwitch(t *testing.T) {
	reg := registry.New()
	versions := version.New()
	evictor := New(reg, lease.New(), health.New(), versions)

	reg.Register("cart", model.NewInstance("c1", "cart", "v1", "10.0.0.1:8080", 5))
	reg.ConfirmDurable("c1")
	versions.Ensure("cart", "v1")
	versions.Deactivate("cart", "v1")

	removed := evictor.EvictInactiveVersion()
	if len(removed) != 1 || removed[0] != "c1" {
		t.Fatalf("expected c1 removed, got %v", removed)
	}
}

// TestEvictInactiveVersion_RespectsInflightRequests ensures the in-flight
// request guard in Evict still applies for confirmed instances.
func TestEvictInactiveVersion_RespectsInflightRequests(t *testing.T) {
	reg := registry.New()
	versions := version.New()
	evictor := New(reg, lease.New(), health.New(), versions)

	reg.Register("cart", model.NewInstance("c1", "cart", "v1", "10.0.0.1:8080", 5))
	reg.ConfirmDurable("c1")
	versions.Ensure("cart", "v1")
	versions.Deactivate("cart", "v1")

	// Pretend a routed request is in flight on the confirmed instance.
	reg.BeginRequest("c1")
	defer reg.EndRequest("c1")

	removed := evictor.EvictInactiveVersion()
	if len(removed) != 0 {
		t.Fatalf("confirmed instance with in-flight requests must not be removed, got %v", removed)
	}
}
