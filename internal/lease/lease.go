// Package lease manages per-instance keep-alive leases.
package lease

import (
	"sync"
	"time"

	"svcregistry/internal/model"
)

// Store holds one lease per registered instance.
type Store struct {
	mu     sync.Mutex
	leases map[string]model.Lease
	now    func() time.Time
}

// New creates a lease store.
func New() *Store {
	return &Store{leases: make(map[string]model.Lease), now: time.Now}
}

// Start opens a lease for an instance.
func (s *Store) Start(instanceID string, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.leases[instanceID]; ok {
		return false
	}
	s.leases[instanceID] = model.Lease{
		InstanceID: instanceID,
		Holder:     instanceID,
		ExpiresAt:  s.now().Add(ttl),
		Generation: 1,
		// BUG(02b): the lease is written straight into the store with no
		// durability barrier, so the confirmed registration and its keep-alive
		// can diverge after a crash: the endpoint is gone but the confirmation
		// was already returned. The lease table is also never replayed from a
		// log, so a restart starts with an empty table.
	}
	return true
}

// Renew extends a lease held by the instance.
func (s *Store) Renew(instanceID string, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.leases[instanceID]
	if !ok {
		return false
	}
	l.ExpiresAt = s.now().Add(ttl)
	l.Generation++
	s.leases[instanceID] = l
	return true
}

// Get returns the lease of an instance.
func (s *Store) Get(instanceID string) (model.Lease, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.leases[instanceID]
	return l, ok
}

// Release drops a lease.
func (s *Store) Release(instanceID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.leases[instanceID]; !ok {
		return false
	}
	delete(s.leases, instanceID)
	return true
}
