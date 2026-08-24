// Package lookup resolves service names to eligible instances.
package lookup

import (
	"sort"

	"svcregistry/internal/model"
	"svcregistry/internal/registry"
)

// Resolver finds instances for a service.
type Resolver struct {
	reg *registry.Registry
}

// New creates a lookup resolver.
func New(reg *registry.Registry) *Resolver {
	return &Resolver{reg: reg}
}

// Find returns instances of a service that may take traffic.
func (r *Resolver) Find(service string) []*model.Instance {
	out := make([]*model.Instance, 0)
	for _, inst := range r.reg.Instances(service) {
		if inst.TakesTraffic() {
			out = append(out, inst)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// FindVersion returns instances of a service restricted to a version.
func (r *Resolver) FindVersion(service, version string) []*model.Instance {
	out := make([]*model.Instance, 0)
	for _, inst := range r.reg.Instances(service) {
		if inst.Version == version && inst.TakesTraffic() {
			out = append(out, inst)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// FindTag returns instances of a service carrying a tag.
func (r *Resolver) FindTag(service, tag string) []*model.Instance {
	out := make([]*model.Instance, 0)
	svc, ok := r.reg.Service(service)
	if !ok || !svc.HasTag(tag) {
		return out
	}
	for _, inst := range r.reg.Instances(service) {
		if inst.TakesTraffic() {
			out = append(out, inst)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
