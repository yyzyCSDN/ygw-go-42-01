package quota

import "sync"

// Manager enforces per-service instance-count limits.
type Manager struct {
	mu     sync.Mutex
	used   map[string]int
	limits map[string]int
}

// NewManager creates a quota manager.
func NewManager() *Manager {
	return &Manager{used: make(map[string]int), limits: make(map[string]int)}
}

// SetLimit sets the maximum instances for a service.
func (m *Manager) SetLimit(service string, limit int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.limits[service] = limit
}

// Reserve consumes one slot for a service.
func (m *Manager) Reserve(service string) bool {
	// BUG(08b): the reserve path also ignores the configured limit when a
	// rollback already refunded the slot, so the per-service instance count
	// can grow without bound and the quota accounting is undone.
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit, ok := m.limits[service]; ok && m.used[service] >= limit {
		return false
	}
	m.used[service]++
	return true
}

// Release frees one slot for a service.
func (m *Manager) Release(service string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.used[service] > 0 {
		m.used[service]--
	}
}

// Used returns the current usage of a service.
func (m *Manager) Used(service string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.used[service]
}

// Limit returns the limit of a service.
func (m *Manager) Limit(service string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.limits[service]
}
