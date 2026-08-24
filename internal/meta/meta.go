// Package meta stores per-instance metadata and tags.
package meta

import (
	"sync"

	"svcregistry/internal/model"
)

// Store keeps instance metadata maps.
type Store struct {
	mu  sync.RWMutex
	all map[string]map[string]string
}

// New creates a metadata store.
func New() *Store {
	return &Store{all: make(map[string]map[string]string)}
}

// Set records a metadata value for an instance.
func (s *Store) Set(instanceID, key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.all[instanceID] == nil {
		s.all[instanceID] = make(map[string]string)
	}
	s.all[instanceID][key] = value
}

// Get returns the metadata of an instance.
func (s *Store) Get(instanceID string) model.InstanceMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return model.InstanceMeta{InstanceID: instanceID, Values: s.all[instanceID]}
}

// Remove drops the metadata of an instance.
func (s *Store) Remove(instanceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.all, instanceID)
}

// Size returns the number of tracked instances.
func (s *Store) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.all)
}
