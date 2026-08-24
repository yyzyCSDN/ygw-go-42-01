package persist

import (
	"os"
	"path/filepath"
	"testing"

	"svcregistry/internal/model"
)

func TestNopAppendAndReplay(t *testing.T) {
	n := NewNop()
	n.Append(Record{Kind: KindRegister, InstanceID: "a1"})
	if err := n.Commit("a1"); err != nil {
		t.Fatalf("commit: %v", err)
	}
	recs, err := n.Replay()
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}
}

func TestWALCommitIsReplayed(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWAL(dir)
	if err != nil {
		t.Fatalf("new wal: %v", err)
	}
	// Stage an instance + its lease for one instance.
	w.Append(Record{Kind: KindRegister, InstanceID: "a1"})
	w.Append(Record{Kind: KindLease, InstanceID: "a1"})
	// Nothing is durable yet: a fresh reader must see no committed records
	// before Commit, because the in-memory staging buffer is not on disk.
	preReplay, err := NewWAL(dir)
	if err != nil {
		t.Fatalf("reopen wal: %v", err)
	}
	pre, err := preReplay.Replay()
	if err != nil {
		t.Fatalf("pre replay: %v", err)
	}
	if err := preReplay.Close(); err != nil {
		t.Fatalf("close preReplay: %v", err)
	}
	if len(pre) != 0 {
		t.Fatalf("expected 0 committed records before commit, got %d", len(pre))
	}

	// Commit the staged intents for a1.
	if err := w.Commit("a1"); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// A new process opens the same log and replays only committed records.
	r, err := NewWAL(dir)
	if err != nil {
		t.Fatalf("reopen wal: %v", err)
	}
	defer r.Close()
	recs, err := r.Replay()
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 committed records, got %d", len(recs))
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestWALUncommittedIntentIsDroppedOnCrash(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWAL(dir)
	if err != nil {
		t.Fatalf("new wal: %v", err)
	}
	// a1 is committed (crash-safe); a2 is only staged (a crash loses it).
	w.Append(Record{Kind: KindRegister, InstanceID: "a1"})
	if err := w.Commit("a1"); err != nil {
		t.Fatalf("commit a1: %v", err)
	}
	w.Append(Record{Kind: KindRegister, InstanceID: "a2"})
	// Simulate a crash: never call Commit("a2"). Reopen the log (the
	// in-memory staging buffer is gone with the process).
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	crash, err := NewWAL(dir)
	if err != nil {
		t.Fatalf("reopen wal: %v", err)
	}
	defer crash.Close()
	recs, err := crash.Replay()
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected only the committed record to survive, got %d", len(recs))
	}
	if recs[0].InstanceID != "a1" {
		t.Fatalf("surviving record = %q, want a1", recs[0].InstanceID)
	}
}

func TestWALRecordsPreserveInstance(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWAL(dir)
	if err != nil {
		t.Fatalf("new wal: %v", err)
	}
	inst := model.NewInstance("a1", "cart", "v1", "10.0.0.1:8080", 5)
	w.Append(Record{Kind: KindRegister, InstanceID: "a1", Service: "cart", Instance: inst})
	if err := w.Commit("a1"); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	crash, err := NewWAL(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer crash.Close()
	recs, err := crash.Replay()
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(recs) != 1 || recs[0].Instance == nil || recs[0].Instance.ID != "a1" {
		t.Fatalf("replayed record lost instance payload: %+v", recs)
	}
	// The log file must exist on disk.
	if _, err := os.Stat(filepath.Join(dir, "registry.wal")); err != nil {
		t.Fatalf("log file missing: %v", err)
	}
}
