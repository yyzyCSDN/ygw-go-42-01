package heartbeat

import (
	"testing"
	"time"

	"svcregistry/internal/model"
	"svcregistry/internal/registry"
)

func TestHeartbeatUsesHeartbeatTime(t *testing.T) {
	reg := registry.New()
	det := NewTimeoutDetector(reg)
	inst := model.NewInstance("a1", "cart", "v1", "10.0.0.1", 1)
	inst.LastHeartbeat = time.Now()
	reg.Register("cart", inst)
	stale := det.StaleInstances(time.Hour)
	if len(stale) != 0 {
		t.Fatalf("active instance flagged stale: %v", stale)
	}
}
