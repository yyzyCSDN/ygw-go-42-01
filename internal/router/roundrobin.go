package router

import "svcregistry/internal/model"

// RoundRobin picks instances in rotation.
type RoundRobin struct {
	next int
}

// NewRoundRobin creates a round-robin strategy.
func NewRoundRobin() *RoundRobin {
	return &RoundRobin{}
}

// Pick selects the next instance in the list.
func (r *RoundRobin) Pick(_ string, instances []*model.Instance) *model.Instance {
	if len(instances) == 0 {
		return nil
	}
	idx := r.next % len(instances)
	r.next = (r.next + 1) % len(instances)
	return instances[idx]
}
