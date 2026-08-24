package registry

import (
	"testing"

	"svcregistry/internal/model"
)

func TestRegisterNotConfirmedBeforeDurable(t *testing.T) {
	reg := New()
	reg.Register("cart", model.NewInstance("a1", "cart", "v1", "10.0.0.1", 1))
	if reg.Confirmed("a1") {
		t.Fatalf("registration confirmed before durable")
	}
	reg.ConfirmDurable("a1")
	if !reg.Confirmed("a1") {
		t.Fatalf("confirmation lost after durable")
	}
}
