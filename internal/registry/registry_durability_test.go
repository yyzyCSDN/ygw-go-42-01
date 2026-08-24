package registry

import (
	"testing"
	"time"

	"svcregistry/internal/lease"
	"svcregistry/internal/model"
	"svcregistry/internal/persist"
)

// This test encodes the bug report directly: a registration confirmation
// must not be returned until the instance and its lease are durably
// persisted. A crash right after the confirmation must leave the instance
// discoverable; a registration that was staged but never confirmed must
// vanish.

// harness wires a registry and a lease store to the same on-disk WAL so a
// single ConfirmDurable commits the instance record and its lease together.
type harness struct {
	reg    *Registry
	leases *lease.Store
	wal    *persist.WAL
	dir    string
}

func newHarness(t *testing.T, dir string) *harness {
	t.Helper()
	wal, err := persist.NewWAL(dir)
	if err != nil {
		t.Fatalf("new wal: %v", err)
	}
	reg := NewWithPersister(wal)
	leases := lease.NewWithPersister(wal)
	return &harness{reg: reg, leases: leases, wal: wal, dir: dir}
}

// reopen simulates a process restart: brand-new registry and lease store
// pointed at the same on-disk log, with in-memory staging buffers empty.
func reopen(t *testing.T, dir string) *harness {
	t.Helper()
	wal, err := persist.NewWAL(dir)
	if err != nil {
		t.Fatalf("reopen wal: %v", err)
	}
	reg := NewWithPersister(wal)
	leases := lease.NewWithPersister(wal)
	leaseRecords, err := reg.Recover()
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	leases.Recover(leaseRecords)
	return &harness{reg: reg, leases: leases, wal: wal, dir: dir}
}

// TestConfirmationSurvivesCrash proves the core invariant: once a
// registration is confirmed, the instance and its lease are durable. A
// crash immediately after confirmation leaves both recoverable, so service
// discovery still finds the endpoint and the lease table still holds the
// keep-alive.
func TestConfirmationSurvivesCrash(t *testing.T) {
	dir := t.TempDir()
	h := newHarness(t, dir)
	defer h.wal.Close()

	inst := model.NewInstance("a1", "cart", "v1", "10.0.0.1:8080", 5)
	if !h.leases.Start("a1", 30*time.Second) {
		t.Fatalf("lease start failed")
	}
	h.reg.Register("cart", inst)

	// Before confirmation: the registration is an unconfirmed intent. A
	// caller MUST NOT have been told it succeeded.
	if h.reg.Confirmed("a1") {
		t.Fatalf("registration was confirmed before the durability barrier ran")
	}

	if !h.reg.ConfirmDurable("a1") {
		t.Fatalf("ConfirmDurable returned false for a staged registration")
	}
	if !h.reg.Confirmed("a1") {
		t.Fatalf("registration not confirmed after ConfirmDurable succeeded")
	}

	// Simulate a crash: drop the process and reopen against the same log.
	h2 := reopen(t, dir)
	defer h2.wal.Close()

	got, ok := h2.reg.Get("a1")
	if !ok {
		t.Fatalf("confirmed instance was lost after crash; discovery cannot find it")
	}
	if got.Addr != "10.0.0.1:8080" {
		t.Fatalf("recovered instance addr = %q, want 10.0.0.1:8080", got.Addr)
	}
	if !h2.reg.Confirmed("a1") {
		t.Fatalf("recovered instance not confirmed; eviction/discovery semantics diverged")
	}
	if _, ok := h2.leases.Get("a1"); !ok {
		t.Fatalf("confirmed lease was lost after crash; keep-alive table diverged")
	}
}

// TestUnconfirmedRegistrationVanishes proves the other half: a registration
// that was staged but never confirmed (the caller was never told it
// succeeded) must not survive a crash. This is the property that makes the
// fix safe to retry — the caller knows nothing succeeded, so re-registering
// cannot create a duplicate that races with stale state.
func TestUnconfirmedRegistrationVanishes(t *testing.T) {
	dir := t.TempDir()
	h := newHarness(t, dir)
	defer h.wal.Close()

	confirmed := model.NewInstance("a1", "cart", "v1", "10.0.0.1:8080", 5)
	staged := model.NewInstance("a2", "cart", "v1", "10.0.0.2:8080", 5)

	// a1: register, start lease, confirm (durably committed).
	h.leases.Start("a1", 30*time.Second)
	h.reg.Register("cart", confirmed)
	h.reg.ConfirmDurable("a1")

	// a2: register and start lease, but CRASH before ConfirmDurable. The
	// caller was never told a2 succeeded.
	h.leases.Start("a2", 30*time.Second)
	h.reg.Register("cart", staged)
	// Intentionally no ConfirmDurable("a2") — simulate crash here.

	h2 := reopen(t, dir)
	defer h2.wal.Close()

	if _, ok := h2.reg.Get("a1"); !ok {
		t.Fatalf("confirmed instance a1 lost after crash")
	}
	if _, ok := h2.reg.Get("a2"); ok {
		t.Fatalf("unconfirmed instance a2 survived crash; caller was told nothing succeeded")
	}
	if _, ok := h2.leases.Get("a2"); ok {
		t.Fatalf("unconfirmed lease a2 survived crash; keep-alive outlived its registration")
	}
}

// TestConfirmDurableFailsLeavesUnconfirmed proves a failed durability
// barrier does not flip the confirmed flag: the caller is never told a
// registration succeeded when its data is not on disk.
func TestConfirmDurableFailsLeavesUnconfirmed(t *testing.T) {
	dir := t.TempDir()
	h := newHarness(t, dir)
	defer h.wal.Close()

	inst := model.NewInstance("a1", "cart", "v1", "10.0.0.1:8080", 5)
	h.leases.Start("a1", 30*time.Second)
	h.reg.Register("cart", inst)

	// Closing the WAL out from under the registry simulates a failing
	// fsync: Commit can no longer reach stable storage.
	h.wal.Close()

	if h.reg.ConfirmDurable("a1") {
		t.Fatalf("ConfirmDurable returned true after the durability barrier failed")
	}
	if h.reg.Confirmed("a1") {
		t.Fatalf("registration marked confirmed despite a failed durability barrier")
	}
}
