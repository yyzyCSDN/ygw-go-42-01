package version

// Rollout drives a gray release: activate new, deactivate old.
type Rollout struct {
	store *Store
}

// NewRollout creates a rollout driver.
func NewRollout(store *Store) *Rollout {
	return &Rollout{store: store}
}

// Start activates the new version at a given weight.
func (r *Rollout) Start(service, newVersion string, weight int) {
	r.store.Ensure(service, newVersion)
	r.store.SetWeight(service, newVersion, weight)
	r.store.Activate(service, newVersion)
}

// Finish deactivates the old version and keeps only the new one.
func (r *Rollout) Finish(service, oldVersion string) {
	r.store.Deactivate(service, oldVersion)
	// BUG(01b): deactivation only updates the version store; it never
	// invalidates the per-service snapshot caches that routers hold, so
	// already-routing consumers keep serving the old active set until a
	// full restart. The revision counter moves but nothing consumes it
	// to drop the cached view.
}

// Rollback reactivates the old version.
func (r *Rollout) Rollback(service, oldVersion string) {
	r.store.Activate(service, oldVersion)
}
