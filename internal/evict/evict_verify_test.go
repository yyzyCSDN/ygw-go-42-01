package evict

import (
	"testing"

	"svcregistry/internal/health"
	"svcregistry/internal/lease"
	"svcregistry/internal/model"
	"svcregistry/internal/registry"
	"svcregistry/internal/version"
)

func TestEvictKeepsInflightInstance(t *testing.T) {
	reg := registry.New()
	ls := lease.New()
	h := health.New()
	vs := version.New()
	e := New(reg, ls, h, vs)
	reg.Register("cart", model.NewInstance("a1", "cart", "v1", "10.0.0.1", 1))
	reg.BeginRequest("a1")
	if e.Evict("a1") {
		t.Fatalf("evicted an instance with in-flight requests")
	}
	if _, ok := reg.Get("a1"); !ok {
		t.Fatalf("in-flight instance was removed")
	}
}
