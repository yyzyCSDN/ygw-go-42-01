package persist

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// WAL is a write-ahead log that persists instance registration and lease
// intents to disk. Append stages an uncommitted intent; Commit fsyncs the
// staged records for one instance so a crash can no longer lose them.
// Replay reads the committed records back, skipping any
// staged-but-uncommitted tail, which is how an unacknowledged
// registration is correctly dropped after a crash.
type WAL struct {
	mu       sync.Mutex
	dir      string
	path     string
	pending  map[string][]Record // keyed by instanceID
	file     *os.File
	encoder  *json.Encoder
}

// NewWAL opens (or creates) a write-ahead log rooted at dir. The log file
// is opened for append; existing committed records are preserved so Replay
// can recover them after a restart.
func NewWAL(dir string) (*WAL, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "registry.wal")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &WAL{
		dir:      dir,
		path:     path,
		pending:  make(map[string][]Record),
		file:     f,
		encoder:  json.NewEncoder(f),
	}, nil
}

// Append stages an uncommitted intent for an instance. It is not durable
// until Commit returns nil for that instance.
func (w *WAL) Append(rec Record) {
	w.mu.Lock()
	w.pending[rec.InstanceID] = append(w.pending[rec.InstanceID], rec)
	w.mu.Unlock()
}

// Commit fsyncs every staged intent for instanceID to disk. On success the
// instance registration and its lease are crash-safe and may be
// acknowledged to the caller; on failure nothing is acknowledged. The
// staged intents are cleared once they are on stable storage.
func (w *WAL) Commit(instanceID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	recs := w.pending[instanceID]
	if len(recs) == 0 {
		// Already committed in a prior call, or nothing staged. The
		// instance is as durable as a previous Commit left it.
		return nil
	}

	// 1. Write each staged record to the log.
	for _, rec := range recs {
		if err := w.encoder.Encode(rec); err != nil {
			return err
		}
	}
	// 2. Flush the kernel page cache into the file.
	if err := w.file.Sync(); err != nil {
		return err
	}
	// 3. fsync the directory entry so the file's existence is durable
	//    across a full-machine crash, not just a process crash.
	if d, err := os.Open(w.dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}

	// The intents are now on stable storage; drop them from staging so a
	// later Replay cannot double-apply them and a later Commit is a no-op.
	delete(w.pending, instanceID)
	return nil
}

// Close flushes any buffered data and releases the log file handle. It must
// be called before the WAL is discarded so the underlying file is not left
// open (which would prevent the log directory from being removed on
// Windows). A crashed process skips Close; the committed records already
// fsynced to disk survive regardless.
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		if err := w.file.Sync(); err != nil {
			return err
		}
		err := w.file.Close()
		w.file = nil
		return err
	}
	return nil
}

// Replay reads the committed records back from disk, in commit order.
// Staged-but-uncommitted intents (those in the pending buffer that were
// never fsynced) are absent: a crash discards exactly what the caller was
// never told succeeded.
func (w *WAL) Replay() ([]Record, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	f, err := os.Open(w.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	out := make([]Record, 0)
	for {
		var rec Record
		if err := dec.Decode(&rec); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}
