// Package version manages per-service version weights and gray rollout state.
package version

import (
	"sort"
	"sync"

	"svcregistry/internal/model"
)

// Store keeps the active/weight state of every service version.
type Store struct {
	mu       sync.RWMutex
	versions map[string]map[string]*model.VersionState
	revision uint64
}

// New creates a version store.
func New() *Store {
	return &Store{versions: make(map[string]map[string]*model.VersionState)}
}

// Ensure registers a version for a service.
func (s *Store) Ensure(service, version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.versions[service] == nil {
		s.versions[service] = make(map[string]*model.VersionState)
	}
	if s.versions[service][version] == nil {
		s.versions[service][version] = &model.VersionState{Service: service, Version: version, Weight: 1, Active: true}
		s.revision++
	}
}

// SetWeight changes the weight of a version.
func (s *Store) SetWeight(service, version string, weight int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.versions[service] == nil {
		s.versions[service] = make(map[string]*model.VersionState)
	}
	st := s.versions[service][version]
	if st == nil {
		st = &model.VersionState{Service: service, Version: version, Active: true}
		s.versions[service][version] = st
	}
	st.Weight = weight
	s.revision++
}

// Activate marks a version eligible for traffic.
func (s *Store) Activate(service, version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.versions[service] == nil {
		s.versions[service] = make(map[string]*model.VersionState)
	}
	st := s.versions[service][version]
	if st == nil {
		st = &model.VersionState{Service: service, Version: version, Weight: 1}
		s.versions[service][version] = st
	}
	st.Active = true
	s.revision++
}

// Deactivate stops a version from taking traffic.
func (s *Store) Deactivate(service, version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st := s.versions[service][version]; st != nil {
		st.Active = false
		s.revision++
	}
}

// Snapshot returns a consistent copy of a service's version state.
func (s *Store) Snapshot(service string) model.VersionSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap := model.VersionSnapshot{
		Service: service,
		Weights: make(map[string]int),
		Active:  make(map[string]bool),
		Revision: s.revision,
	}
	for version, st := range s.versions[service] {
		snap.Weights[version] = st.Weight
		snap.Active[version] = st.Active
	}
	return snap
}

// Versions returns the sorted version names of a service.
func (s *Store) Versions(service string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.versions[service]))
	for version := range s.versions[service] {
		out = append(out, version)
	}
	sort.Strings(out)
	return out
}
