// Package router selects a target instance for a discovery request.
package router

import (
	"svcregistry/internal/lookup"
	"svcregistry/internal/model"
	"svcregistry/internal/version"
)

// Strategy selects one instance from a candidate set.
type Strategy interface {
	Pick(key string, instances []*model.Instance) *model.Instance
}

// Router routes discovery requests to eligible instances of active versions.
type Router struct {
	lookup   *lookup.Resolver
	versions *version.Store
	strategy Strategy
}

// New creates a router.
func New(l *lookup.Resolver, v *version.Store, strategy Strategy) *Router {
	return &Router{lookup: l, versions: v, strategy: strategy}
}

// Route picks an instance for a service, considering active versions only.
func (r *Router) Route(service, key string) (*model.Instance, bool) {
	snap := r.versions.Snapshot(service)
	active := snap.ActiveVersions()
	activeSet := make(map[string]bool, len(active))
	for _, v := range active {
		activeSet[v] = true
	}
	candidates := make([]*model.Instance, 0)
	for _, inst := range r.lookup.Find(service) {
		if activeSet[inst.Version] {
			candidates = append(candidates, inst)
		}
	}
	if len(candidates) == 0 {
		return nil, false
	}
	return r.strategy.Pick(key, candidates), true
}

// Routes returns one pick per provided key.
func (r *Router) Routes(service string, keys []string) []*model.Instance {
	out := make([]*model.Instance, 0, len(keys))
	for _, key := range keys {
		if inst, ok := r.Route(service, key); ok {
			out = append(out, inst)
		}
	}
	return out
}
