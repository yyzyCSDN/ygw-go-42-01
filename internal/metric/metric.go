// Package metric accumulates registry counters.
package metric

import "sync"

// Collector tracks registry counters.
type Collector struct {
	mu          sync.Mutex
	registrations int
	discoveries   int
	heartbeats    int
	evictions     int
}

// New creates a collector.
func New() *Collector {
	return &Collector{}
}

// RecordRegistration increments the registration counter.
func (c *Collector) RecordRegistration() {
	c.mu.Lock()
	c.registrations++
	c.mu.Unlock()
}

// RecordDiscovery increments the discovery counter.
func (c *Collector) RecordDiscovery() {
	c.mu.Lock()
	c.discoveries++
	c.mu.Unlock()
}

// RecordHeartbeat increments the heartbeat counter.
func (c *Collector) RecordHeartbeat() {
	c.mu.Lock()
	c.heartbeats++
	c.mu.Unlock()
}

// RecordEviction increments the eviction counter.
func (c *Collector) RecordEviction() {
	c.mu.Lock()
	c.evictions++
	c.mu.Unlock()
}

// Snapshot returns current counters.
func (c *Collector) Snapshot() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return map[string]int{
		"registrations": c.registrations,
		"discoveries":   c.discoveries,
		"heartbeats":    c.heartbeats,
		"evictions":     c.evictions,
	}
}
