package registry

import (
	"context"
	"testing"

	"svcregistry/internal/model"
)

func TestCancelledRegisterNotVisible(t *testing.T) {
	reg := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := reg.RegisterContext(ctx, "cart", model.NewInstance("a1", "cart", "v1", "10.0.0.1", 1))
	if err == nil {
		t.Fatalf("cancelled register should fail")
	}
	if _, ok := reg.Get("a1"); ok {
		t.Fatalf("cancelled register partially visible")
	}
}
