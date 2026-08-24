package evict

import (
	"testing"

	"svcregistry/internal/health"
	"svcregistry/internal/lease"
	"svcregistry/internal/model"
	"svcregistry/internal/registry"
	"svcregistry/internal/version"
)

func TestVersionSwitchKeepsUnackedInstance(t *testing.T) {
	reg := registry.New()
	ls := lease.New()
	h := health.New()
	vs := version.New()
	e := New(reg, ls, h, vs)
	vs.Ensure("cart", "v1")
	vs.Ensure("cart", "v2")
	vs.Deactivate("cart", "v1")
	reg.Register("cart", model.NewInstance("a1", "cart", "v1", "10.0.0.1", 1))
	removed := e.EvictInactiveVersion()
	if len(removed) != 0 {
		t.Fatalf("removed unacked instance: %v", removed)
	}
	if _, ok := reg.Get("a1"); !ok {
		t.Fatalf("unacked instance was dropped by version switch")
	}
}
