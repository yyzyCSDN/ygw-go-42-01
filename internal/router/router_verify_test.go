package router

import (
	"testing"

	"svcregistry/internal/lookup"
	"svcregistry/internal/model"
	"svcregistry/internal/registry"
	"svcregistry/internal/version"
)

type firstStrategy struct{}

func (firstStrategy) Pick(_ string, instances []*model.Instance) *model.Instance {
	if len(instances) == 0 {
		return nil
	}
	return instances[0]
}

func TestRouterUsesLatestVersionWeights(t *testing.T) {
	reg := registry.New()
	ls := lookup.New(reg)
	vs := version.New()
	r := New(ls, vs, firstStrategy{})
	vs.Ensure("cart", "v1")
	vs.Ensure("cart", "v2")
	a1 := model.NewInstance("a1", "cart", "v1", "10.0.0.1", 1)
	a1.Status = model.InstanceHealthy
	b1 := model.NewInstance("b1", "cart", "v2", "10.0.0.2", 1)
	b1.Status = model.InstanceHealthy
	reg.Register("cart", a1)
	reg.Register("cart", b1)
	if _, ok := r.Route("cart", "warm"); !ok {
		t.Fatalf("first route failed")
	}
	vs.Deactivate("cart", "v1")
	picked, ok := r.Route("cart", "k")
	if !ok {
		t.Fatalf("route failed after deactivation")
	}
	if picked.Version != "v2" {
		t.Fatalf("routed to deactivated version: %s", picked.Version)
	}
}
