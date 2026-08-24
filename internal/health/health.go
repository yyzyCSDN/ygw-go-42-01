// Package health tracks active probes and passive failures per instance.
package health

import (
	"sync"
	"time"

	"svcregistry/internal/model"
)

type record struct {
	lastProbeOK  bool
	passiveCount int
	updatedAt    time.Time
}

// Status is the per-instance health table.
type Status struct {
	mu      sync.Mutex
	records map[string]*record
	now     func() time.Time
}

// New creates a health status table.
func New() *Status {
	return &Status{records: make(map[string]*record), now: time.Now}
}

// RecordProbe writes the latest active probe result.
func (s *Status) RecordProbe(instanceID string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.ensure(instanceID)
	r.lastProbeOK = ok
	r.updatedAt = s.now()
}

// RecordPassiveFailure accumulates one business failure.
func (s *Status) RecordPassiveFailure(instanceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.ensure(instanceID)
	r.passiveCount++
	r.updatedAt = s.now()
}

// Snapshot returns the merged health view of an instance.
func (s *Status) Snapshot(instanceID string) model.HealthSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.records[instanceID]
	if !ok {
		return model.HealthSnapshot{InstanceID: instanceID, OK: true, UpdatedAt: s.now()}
	}
	return mergeHealth(instanceID, r, s.now())
}

func (s *Status) ensure(instanceID string) *record {
	r, ok := s.records[instanceID]
	if !ok {
		r = &record{lastProbeOK: true}
		s.records[instanceID] = r
	}
	return r
}

// mergeHealth combines the active probe signal and the passive failure count.
// Passive failures are sticky within their window: a single successful probe
// must not clear an accumulated failure signal.
func mergeHealth(instanceID string, r *record, now time.Time) model.HealthSnapshot {
	state := model.InstanceHealthy
	if r.passiveCount > 0 {
		state = model.InstanceSuspect
	} else if !r.lastProbeOK {
		state = model.InstanceEvicted
	}
	_ = state
	return model.HealthSnapshot{
		InstanceID:   instanceID,
		OK:           r.lastProbeOK && r.passiveCount == 0,
		PassiveCount: r.passiveCount,
		UpdatedAt:    r.updatedAt,
	}
}
