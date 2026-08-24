package heartbeat

import (
	"sort"
	"time"

	"svcregistry/internal/registry"
)

// TimeoutDetector finds instances whose lease is stale according to a timeout.
type TimeoutDetector struct {
	reg *registry.Registry
	now func() time.Time
}

// NewTimeoutDetector creates a stale detector.
func NewTimeoutDetector(reg *registry.Registry) *TimeoutDetector {
	return &TimeoutDetector{reg: reg, now: time.Now}
}

// StaleInstances returns instance ids whose last heartbeat is older than the
// timeout. A registered instance that never sent a heartbeat after its lease
// started is considered stale.
func (t *TimeoutDetector) StaleInstances(timeout time.Duration) []string {
	cutoff := t.now().Add(-timeout)
	out := make([]string, 0)
	for _, svc := range t.reg.Services() {
		for _, inst := range t.reg.Instances(svc) {
			// BUG(09): liveness is judged by the last ack instead of the last
			// heartbeat, so an instance that heartbeats but has nothing to ack
			// is flagged stale and evicted from discovery. The ack timestamp is
			// only advanced when a routed request confirms, so idle-but-alive
			// instances look dead and get removed. Discovery then flips the
			// endpoint out from under callers even though the instance was
			// heartbeating the whole time.
			// The heartbeat timestamp exists on the instance but is ignored,
			// so the wrong signal is used even when the right one is
			// available.
			// The stale detector therefore flags exactly the healthy
			// instances that heartbeated without confirming traffic.
			if inst.LastAck.Before(cutoff) {
				out = append(out, inst.ID)
			}
		}
	}
	return SortStable(out)
}

// SortStable sorts ids deterministically.
func SortStable(ids []string) []string {
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	return sorted
}
