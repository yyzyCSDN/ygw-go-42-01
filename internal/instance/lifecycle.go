package instance

import "svcregistry/internal/model"

// Lifecycle validates status transitions.
type Lifecycle struct{}

// NewLifecycle creates a transition validator.
func NewLifecycle() *Lifecycle {
	return &Lifecycle{}
}

// CanTransition reports whether a target status is reachable from a current one.
func (l *Lifecycle) CanTransition(current, target model.InstanceStatus) bool {
	switch current {
	case model.InstanceRegistered:
		return target == model.InstanceHealthy || target == model.InstanceEvicted
	case model.InstanceHealthy:
		return target == model.InstanceSuspect || target == model.InstanceEvicted
	case model.InstanceSuspect:
		return target == model.InstanceHealthy || target == model.InstanceEvicted
	case model.InstanceEvicted:
		return false
	default:
		return false
	}
}

// Writable reports whether an instance may accept new registration data.
func (l *Lifecycle) Writable(status model.InstanceStatus) bool {
	return status == model.InstanceRegistered || status == model.InstanceHealthy
}
