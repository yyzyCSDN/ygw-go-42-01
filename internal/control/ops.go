package control

import "time"

// RunEviction triggers one eviction pass and returns the count removed.
func (c *Controller) RunEviction(at time.Time) int {
	return c.evictor.Scan(at)
}
