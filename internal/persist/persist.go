// Package persist owns the write-ahead log that makes instance
// registrations and their leases crash-safe.
//
// A registration is only durable after the instance record and its lease
// record have been fsynced to disk. Append stages an in-memory intent;
// Commit flushes and syncs every staged intent for one instance so a crash
// can no longer lose it. Replay reads the synced records back after a
// restart. Intents that were staged but never committed are discarded on
// replay, which is exactly the property the caller relies on: no success
// confirmation is ever returned for state a crash would lose.
package persist

import (
	"sync"

	"svcregistry/internal/model"
)

// Kind identifies what a record persists.
type Kind string

const (
	// KindRegister persists an instance registration intent.
	KindRegister Kind = "register"
	// KindLease persists an instance lease intent.
	KindLease Kind = "lease"
)

// Record is one durably stored intent. The Kind field selects which payload
// field is meaningful.
type Record struct {
	Kind       Kind            `json:"kind"`
	InstanceID string          `json:"instance_id"`
	Service    string          `json:"service,omitempty"`
	Instance   *model.Instance `json:"instance,omitempty"`
	Lease      model.Lease     `json:"lease,omitempty"`
}

// Persister is the crash-safety barrier in front of instance registrations
// and their leases. It exists so that a registration confirmation can be
// gated on real durability rather than promised before the fact.
type Persister interface {
	// Append stages an uncommitted intent for an instance. It must not be
	// considered durable until Commit returns a nil error for that
	// instance.
	Append(rec Record)
	// Commit fsyncs every staged intent for instanceID to disk. A nil
	// error means the instance and its lease are now crash-safe and may
	// be acknowledged to the caller.
	Commit(instanceID string) error
	// Replay returns the committed records read back from disk, in the
	// order they were committed. Staged-but-uncommitted intents are
	// absent: a crash discards exactly what the caller was never told
	// succeeded.
	Replay() ([]Record, error)
}

// Nop is an in-memory Persister. It keeps staged intents in memory only and
// treats Commit as a no-op that always succeeds. It is used by the
// in-process registry when no on-disk durability is configured, so the
// in-memory demo path needs no filesystem.
type Nop struct {
	mu      sync.Mutex
	records []Record
}

// NewNop creates an in-memory persister.
func NewNop() *Nop { return &Nop{} }

// Append stages a record in memory.
func (n *Nop) Append(rec Record) {
	n.mu.Lock()
	n.records = append(n.records, rec)
	n.mu.Unlock()
}

// Commit is a no-op; in-memory state is never crash-safe across a restart.
func (n *Nop) Commit(instanceID string) error { return nil }

// Replay returns every staged record. There is no commit boundary in
// memory, so nothing is filtered.
func (n *Nop) Replay() ([]Record, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := make([]Record, len(n.records))
	copy(out, n.records)
	return out, nil
}
