package lookup

import "svcregistry/internal/model"

// Filter narrows a discovery result set.
type Filter struct {
	Version string
	Tag     string
}

// Apply filters instances by version and tag.
func (f Filter) Apply(instances []*model.Instance) []*model.Instance {
	out := make([]*model.Instance, 0)
	for _, inst := range instances {
		if f.Version != "" && inst.Version != f.Version {
			continue
		}
		out = append(out, inst)
	}
	return out
}
