package instance

import "svcregistry/internal/model"

// StateView is a snapshot of one instance's lifecycle.
type StateView struct {
	InstanceID string
	Status     model.InstanceStatus
	Weight     int
	Addr       string
}

// View renders an instance as a state snapshot.
func View(inst *model.Instance) StateView {
	return StateView{InstanceID: inst.ID, Status: inst.Status, Weight: inst.Weight, Addr: inst.Addr}
}
