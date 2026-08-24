package evict

import "time"

// Scan runs one full eviction pass at the given time.
func (e *Evictor) Scan(at time.Time) int {
	n := len(e.EvictExpired(at))
	n += len(e.EvictUnhealthy())
	n += len(e.EvictInactiveVersion())
	return n
}
