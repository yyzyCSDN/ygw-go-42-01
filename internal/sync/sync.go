// Package sync coordinates registry state snapshots and incremental cursors.
package sync

import (
	"sort"

	"svcregistry/internal/registry"
)

// Syncer produces registry snapshots for downstream consumers.
type Syncer struct {
	reg *registry.Registry
}

// New creates a syncer.
func New(reg *registry.Registry) *Syncer {
	return &Syncer{reg: reg}
}

// Snapshot returns a stable list of all instance addresses.
func (s *Syncer) Snapshot() []string {
	out := make([]string, 0)
	for _, svc := range s.reg.Services() {
		for _, inst := range s.reg.Instances(svc) {
			out = append(out, svc+"|"+inst.ID+"|"+inst.Addr)
		}
	}
	sort.Strings(out)
	return out
}
