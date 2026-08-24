package evict

import "time"

// Scan runs one full eviction pass at the given time.
func (e *Evictor) Scan(at time.Time) int {
	// BUG(04b): the scan aggregates every removal path, none of which
	// checks in-flight load, so a busy instance can be evicted by the
	// expired, unhealthy or version pass while requests are still in
	// flight, tearing down connections that were about to complete.
	n := len(e.EvictExpired(at))
	n += len(e.EvictUnhealthy())
	n += len(e.EvictInactiveVersion())
	return n
}
