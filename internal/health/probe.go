package health

import "svcregistry/internal/model"

// Prober runs simulated active probes against instances.
type Prober struct {
	status *Status
	ok     func(*model.Instance) bool
}

// NewProber creates a prober with a deterministic ok predicate.
func NewProber(status *Status, ok func(*model.Instance) bool) *Prober {
	return &Prober{status: status, ok: ok}
}

// Probe checks one instance and records the result.
func (p *Prober) Probe(inst *model.Instance) {
	p.status.RecordProbe(inst.ID, p.ok(inst))
}

// ProbeAll checks every instance of a service.
func (p *Prober) ProbeAll(instances []*model.Instance) {
	for _, inst := range instances {
		p.Probe(inst)
	}
}
