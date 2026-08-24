package router

import "svcregistry/internal/model"

// LeastConn picks the instance with the lowest in-flight weight.
type LeastConn struct {
	inflight func(instanceID string) int
}

// NewLeastConn creates a least-connection strategy.
func NewLeastConn(inflight func(instanceID string) int) *LeastConn {
	return &LeastConn{inflight: inflight}
}

// Pick selects the instance with the fewest in-flight requests.
func (l *LeastConn) Pick(_ string, instances []*model.Instance) *model.Instance {
	if len(instances) == 0 {
		return nil
	}
	best := instances[0]
	bestCount := l.inflight(best.ID)
	for _, inst := range instances[1:] {
		count := l.inflight(inst.ID)
		if count < bestCount {
			bestCount = count
			best = inst
		}
	}
	return best
}
