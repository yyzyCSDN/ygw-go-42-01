package registry

import (
	"testing"

	"svcregistry/internal/model"
)

func TestUnregisterNotOverwrittenByInflightRegister(t *testing.T) {
	reg := New()
	reg.Register("cart", model.NewInstance("a1", "cart", "v1", "10.0.0.1", 1))
	reg.Unregister("a1")
	err := reg.RegisterInflight("cart", model.NewInstance("a1", "cart", "v2", "10.0.0.2", 1))
	if err == nil {
		t.Fatalf("inflight register resurrected a deleted instance")
	}
	if _, ok := reg.Get("a1"); ok {
		t.Fatalf("deleted instance was resurrected")
	}
}
