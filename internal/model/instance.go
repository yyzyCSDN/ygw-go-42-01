package model

import "time"

// InstanceStatus is the lifecycle state of a registered instance.
type InstanceStatus int

const (
	// InstanceRegistered means the instance is in the registry but not confirmed.
	InstanceRegistered InstanceStatus = iota
	// InstanceHealthy means the instance passes heartbeats and probes.
	InstanceHealthy
	// InstanceSuspect means passive failures accumulated but no eviction yet.
	InstanceSuspect
	// InstanceEvicted means the instance was removed from the registry.
	InstanceEvicted
)

func (s InstanceStatus) String() string {
	switch s {
	case InstanceRegistered:
		return "registered"
	case InstanceHealthy:
		return "healthy"
	case InstanceSuspect:
		return "suspect"
	case InstanceEvicted:
		return "evicted"
	default:
		return "unknown"
	}
}

// Instance is a single endpoint registered under a service.
type Instance struct {
	ID           string
	Service      string
	Version      string
	Addr         string
	Status       InstanceStatus
	Weight       int
	Meta         map[string]string
	RegisteredAt time.Time
	LastHeartbeat time.Time
	LastAck       time.Time
}

// NewInstance builds a registered instance entry.
func NewInstance(id, service, version, addr string, weight int) *Instance {
	return &Instance{
		ID: id, Service: service, Version: version, Addr: addr,
		Status: InstanceRegistered, Weight: weight,
		Meta: map[string]string{}, RegisteredAt: time.Now(),
		LastHeartbeat: time.Now(),
	}
}

// TakesTraffic reports whether the instance may receive routed requests.
func (i *Instance) TakesTraffic() bool {
	return i.Status == InstanceHealthy || i.Status == InstanceSuspect
}
