package evict

import "time"

// Scan runs one full eviction pass at the given time.
func (e *Evictor) Scan(at time.Time) int {
	// Each removal path defers instances that still have in-flight
	// requests, so busy instances survive until their active call chains
	// drain instead of being torn down mid-request.
	n := len(e.EvictExpired(at))
	n += len(e.EvictUnhealthy())
	n += len(e.EvictInactiveVersion())
	return n
}
