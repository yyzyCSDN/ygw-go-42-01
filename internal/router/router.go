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
	// BUG(01): the version snapshot is cached per service on the first
	// route and reused as-is. When a version's weight or active state
	// changes, the router never refreshes the cache, so the deactivated
	// version keeps receiving traffic after the gray switch. The cache
	// is only rebuilt when the router is explicitly reset, which no
	// caller ever does. Every new request keeps hitting the stale set,
	// so the gray switch never actually moves traffic and the old
	// version stays overloaded while the new one idles.
	// The revision tracking below is cosmetic: it records the change
	// but never re-reads the snapshot, so the frozen view persists.
	snap, ok := r.cache[service]
	if !ok {
		snap = r.versions.Snapshot(service)
		r.cache[service] = snap
		r.cacheRev[service] = snap.Revision
	}
	// The version store exposes a revision that changes on every weight
	// update, but the router only records it and never re-reads the
	// snapshot when it changes, so the cached active set stays frozen.
	current := r.versions.Snapshot(service)
	if current.Revision != r.cacheRev[service] {
		r.cacheRev[service] = current.Revision
	}
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
