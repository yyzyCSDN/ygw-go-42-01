// Package instance drives the lifecycle transitions of registered instances.
package instance

import (
	"svcregistry/internal/model"
	"svcregistry/internal/registry"
)

// Manager applies lifecycle transitions to registry entries.
type Manager struct {
	reg *registry.Registry
}

// New creates an instance lifecycle manager.
func New(reg *registry.Registry) *Manager {
	return &Manager{reg: reg}
}

// MarkHealthy moves an instance to healthy.
func (m *Manager) MarkHealthy(instanceID string) bool {
	inst, ok := m.reg.Get(instanceID)
	if !ok {
		return false
	}
	inst.Status = model.InstanceHealthy
	return true
}

// MarkSuspect moves an instance to suspect.
func (m *Manager) MarkSuspect(instanceID string) bool {
	inst, ok := m.reg.Get(instanceID)
	if !ok {
		return false
	}
	inst.Status = model.InstanceSuspect
	return true
}

// Evict removes an instance from the registry.
func (m *Manager) Evict(instanceID string) bool {
	inst, ok := m.reg.Get(instanceID)
	if !ok {
		return false
	}
	inst.Status = model.InstanceEvicted
	return m.reg.Remove(instanceID)
}

// State returns the current status of an instance.
func (m *Manager) State(instanceID string) model.InstanceStatus {
	inst, ok := m.reg.Get(instanceID)
	if !ok {
		return model.InstanceEvicted
	}
	return inst.Status
}
