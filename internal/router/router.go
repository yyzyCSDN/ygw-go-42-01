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
	cache    map[string]model.VersionSnapshot
	cacheRev map[string]uint64
}

// New creates a router.
func New(l *lookup.Resolver, v *version.Store, strategy Strategy) *Router {
	return &Router{lookup: l, versions: v, strategy: strategy, cache: make(map[string]model.VersionSnapshot), cacheRev: make(map[string]uint64)}
}

// Route picks an instance for a service, considering active versions only.
func (r *Router) Route(service, key string) (*model.Instance, bool) {
	// Read the latest version snapshot on every route and refresh the cache
	// whenever the version store's revision changes. Weight and active-state
	// updates (gray switch, rollout finish/rollback) bump the revision, so
	// following it here keeps routing on the current version weights instead
	// of the frozen view captured on the first route. Without this refresh
	// the deactivated old version keeps receiving traffic after the switch
	// while the new one idles.
	current := r.versions.Snapshot(service)
	if cached, ok := r.cache[service]; !ok || cached.Revision != current.Revision {
		r.cache[service] = current
		r.cacheRev[service] = current.Revision
	}
	snap := r.cache[service]
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
