package registry

import (
	"testing"

	"svcregistry/internal/model"
)

func TestRetryRegisterPreservesOrder(t *testing.T) {
	reg := New()
	seq1 := reg.Register("cart", model.NewInstance("a1", "cart", "v1", "10.0.0.1", 1))
	reg.Register("cart", model.NewInstance("a1", "cart", "v2", "10.0.0.2", 1))
	if reg.RegisterSeq("cart", model.NewInstance("a1", "cart", "v1", "10.0.0.1", 1), seq1) {
		t.Fatalf("stale retry overwrote a newer registration")
	}
	inst, _ := reg.Get("a1")
	if inst.Version != "v2" {
		t.Fatalf("stale retry rolled the registration back to %s", inst.Version)
	}
}
