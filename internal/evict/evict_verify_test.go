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

func TestExpiredLeaseEvictsInstance(t *testing.T) {
	reg := registry.New()
	ls := lease.New()
	h := health.New()
	vs := version.New()
	e := New(reg, ls, h, vs)
	reg.Register("cart", model.NewInstance("a1", "cart", "v1", "10.0.0.1", 1))
	ls.Start("a1", time.Minute)
	removed := e.EvictExpired(time.Now().Add(time.Hour))
	if len(removed) == 0 {
		t.Fatalf("expired lease instance was not evicted")
	}
	if _, ok := reg.Get("a1"); ok {
		t.Fatalf("expired instance still in registry")
	}
}
