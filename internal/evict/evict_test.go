package evict

import (
	"testing"
	"time"

	"svcregistry/internal/health"
	"svcregistry/internal/lease"
	"svcregistry/internal/model"
	"svcregistry/internal/registry"
	"svcregistry/internal/version"
)

// newEvictor builds an evictor wired to a registry with one healthy
// instance whose lease has already expired.
func newEvictor(t *testing.T) (*Evictor, *registry.Registry, string) {
	t.Helper()
	reg := registry.New()
	leases := lease.New()
	h := health.New()
	versions := version.New()

	inst := model.NewInstance("a1", "cart", "v1", "10.0.0.1:8080", 5)
	reg.Register("cart", inst)
	reg.ConfirmDurable("a1")
	leases.Start("a1", time.Second)

	return New(reg, leases, h, versions), reg, "a1"
}

// An instance with in-flight requests must not be evicted: the running call
// chain would be left pointing at a removed endpoint.
func TestEvict_DeferredWhileInflight(t *testing.T) {
	e, reg, id := newEvictor(t)

	// Simulate a request that is mid-flight when the eviction pass fires.
	reg.BeginRequest(id)
	if e.Evict(id) {
		t.Fatalf("Evict removed instance with %d in-flight requests", reg.Inflight(id))
	}
	if _, ok := reg.Get(id); !ok {
		t.Fatal("instance removed while requests were still in flight")
	}

	// Once the in-flight request drains, the deferred eviction succeeds.
	reg.EndRequest(id)
	if !e.Evict(id) {
		t.Fatal("Evict did not remove instance after in-flight requests drained")
	}
	if _, ok := reg.Get(id); ok {
		t.Fatal("instance still present after drain-and-evict")
	}
}

// EvictExpired must defer (not drop) instances whose lease expired but that
// still have in-flight requests.
func TestEvictExpired_DeferredWhileInflight(t *testing.T) {
	e, reg, id := newEvictor(t)

	reg.BeginRequest(id)
	removed := e.EvictExpired(time.Now().Add(time.Hour))
	if len(removed) != 0 {
		t.Fatalf("EvictExpired removed %v with in-flight requests still active", removed)
	}
	if _, ok := reg.Get(id); !ok {
		t.Fatal("expired instance removed while requests were still in flight")
	}

	reg.EndRequest(id)
	removed = e.EvictExpired(time.Now().Add(time.Hour))
	if len(removed) != 1 || removed[0] != id {
		t.Fatalf("EvictExpired = %v, want [%s] after drain", removed, id)
	}
}

// An instance with no in-flight load is evicted as before.
func TestEvict_ImmediateWhenIdle(t *testing.T) {
	e, reg, id := newEvictor(t)

	if !e.Evict(id) {
		t.Fatal("Evict did not remove idle instance")
	}
	if _, ok := reg.Get(id); ok {
		t.Fatal("idle instance still present after Evict")
	}
}
