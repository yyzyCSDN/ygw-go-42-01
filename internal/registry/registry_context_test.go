package registry

import (
	"context"
	"errors"
	"testing"

	"svcregistry/internal/model"
)

// newRegistryInstance builds a fresh registry plus one instance for tests.
func newRegistryInstance(t *testing.T) (*Registry, *model.Instance) {
	t.Helper()
	reg := New()
	inst := model.NewInstance("c1", "cart", "v1", "10.0.0.1:8080", 5)
	return reg, inst
}

// A cancelled context must leave the registry untouched: no instance, no
// service, and discovery returns nothing.
func TestRegisterContextCancelledLeavesNoTrace(t *testing.T) {
	reg, inst := newRegistryInstance(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := reg.RegisterContext(ctx, "cart", inst)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	// No instance record.
	if got, ok := reg.Get(inst.ID); ok {
		t.Fatalf("cancelled registration left an instance behind: %v", got)
	}
	// No service entry.
	if _, ok := reg.Service("cart"); ok {
		t.Fatalf("cancelled registration left a service entry behind")
	}
	// Discovery returns nothing.
	if got := reg.Instances("cart"); len(got) != 0 {
		t.Fatalf("discovery returned instances after a cancelled registration: %v", got)
	}
	if got := reg.Services(); len(got) != 0 {
		t.Fatalf("service list non-empty after a cancelled registration: %v", got)
	}
	// Snapshot does not advertise a partial endpoint.
	for _, st := range reg.Snapshot() {
		if st.Service == "cart" {
			t.Fatalf("snapshot contains cancelled registration: %+v", st)
		}
	}
	// Confirmed reflects the absence.
	if reg.Confirmed(inst.ID) {
		t.Fatalf("cancelled registration was marked confirmed")
	}
}

// A fresh context registers normally and returns a nil error.
func TestRegisterContextUncancelledRegisters(t *testing.T) {
	reg, inst := newRegistryInstance(t)

	if err := reg.RegisterContext(context.Background(), "cart", inst); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	got, ok := reg.Get(inst.ID)
	if !ok {
		t.Fatalf("instance was not registered")
	}
	if got.Addr != inst.Addr {
		t.Fatalf("registered instance addr mismatch: got %q want %q", got.Addr, inst.Addr)
	}
}

// A cancelled registration must not clear a prior delete marker: it never
// ran, so the registry state from the caller's perspective is unchanged.
func TestRegisterContextCancelledDoesNotClearDelete(t *testing.T) {
	reg, inst := newRegistryInstance(t)
	reg.Register("cart", inst)
	reg.Unregister(inst.ID)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := reg.RegisterContext(ctx, "cart", inst); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if _, ok := reg.Get(inst.ID); ok {
		t.Fatalf("cancelled registration resurrected a deleted instance")
	}
}
