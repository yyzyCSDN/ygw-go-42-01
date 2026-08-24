// Package lease manages per-instance keep-alive leases.
package lease

import (
	"sync"
	"time"

	"svcregistry/internal/model"
	"svcregistry/internal/persist"
)

// Store holds one lease per registered instance. A lease is staged (not yet
// durable) when Start is called; it becomes crash-safe only once the shared
// persister's Commit barrier flushes it alongside the instance record, which
// happens in registry.ConfirmDurable. That ordering guarantees the
// registration confirmation never outruns the lease's durability.
type Store struct {
	mu        sync.Mutex
	leases    map[string]model.Lease
	now       func() time.Time
	persister persist.Persister
}

// New creates a lease store backed by an in-memory (non-durable) persister.
// Use NewWithPersister when crash-safe durability is required, sharing the
// same persister as the registry so a registration and its lease are
// committed by a single barrier.
func New() *Store {
	return NewWithPersister(persist.NewNop())
}

// NewWithPersister creates a lease store that stages lease intents through
// the given persister. Pass the same persister the registry uses so
// ConfirmDurable commits the instance and its lease together.
func NewWithPersister(p persist.Persister) *Store {
	return &Store{leases: make(map[string]model.Lease), now: time.Now, persister: p}
}

// Start opens a lease for an instance. The lease is placed in memory so
// keep-alives can proceed immediately, and a lease intent is staged in the
// shared persister so it is fsynced together with the instance record by
// registry.ConfirmDurable. It returns false if a lease is already held.
func (s *Store) Start(instanceID string, ttl time.Duration) bool {
	s.mu.Lock()
	if _, ok := s.leases[instanceID]; ok {
		s.mu.Unlock()
		return false
	}
	l := model.Lease{
		InstanceID: instanceID,
		Holder:     instanceID,
		ExpiresAt:  s.now().Add(ttl),
		Generation: 1,
	}
	s.leases[instanceID] = l
	persister := s.persister
	s.mu.Unlock()

	// Stage the lease intent; it is not durable until the shared barrier
	// (registry.ConfirmDurable) commits it alongside the instance record.
	if persister != nil {
		persister.Append(persist.Record{
			Kind:       persist.KindLease,
			InstanceID: instanceID,
			Lease:      l,
		})
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

// Recover rebuilds the in-memory lease table from replayed durability
// records. Only KindLease records are applied; an uncommitted lease intent
// (staged but never fsynced) is absent from the replay and stays gone,
// matching the registration it was never confirmed against. records is
// typically the slice returned by registry.Recover.
func (s *Store) Recover(records []persist.Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rec := range records {
		if rec.Kind != persist.KindLease {
			continue
		}
		s.leases[rec.InstanceID] = rec.Lease
	}
}
