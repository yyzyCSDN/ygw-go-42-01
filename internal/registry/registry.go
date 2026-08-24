// Package registry owns the service table and per-service instance sets.
package registry

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"svcregistry/internal/model"
)

// errDeleted is returned when an in-flight registration targets a deleted id.
var errDeleted = errors.New("registry: instance was deleted")

// Registry is the in-process source of truth for services and instances.
type Registry struct {
	mu        sync.RWMutex
	services  map[string]*model.Service
	instances map[string]*model.Instance
	byService map[string]map[string]*model.Instance
	confirmed map[string]bool
	inflight  map[string]int
	seqs      map[string]uint64
	deleted   map[string]bool
}

// New creates an empty registry.
func New() *Registry {
	return &Registry{
		services:  make(map[string]*model.Service),
		instances: make(map[string]*model.Instance),
		byService: make(map[string]map[string]*model.Instance),
		confirmed: make(map[string]bool),
		inflight:  make(map[string]int),
		seqs:      make(map[string]uint64),
		deleted:   make(map[string]bool),
	}
}

// Register adds an instance to the registry under its service. The instance is
// recorded as an unconfirmed intent until ConfirmDurable is called. A fresh
// registration clears any previous delete marker for the id. The returned
// sequence number lets stale retries be rejected.
func (r *Registry) Register(svc string, inst *model.Instance) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.deleted, inst.ID)
	s := r.services[svc]
	if s == nil {
		s = model.NewService(svc)
		r.services[svc] = s
	}
	if r.byService[svc] == nil {
		r.byService[svc] = make(map[string]*model.Instance)
	}
	r.instances[inst.ID] = inst
	r.byService[svc][inst.ID] = inst
	r.seqs[inst.ID]++
	return r.seqs[inst.ID]
}

// RegisterSeq registers an instance carrying an explicit sequence number. A
// stale retry (older sequence than the current registration) is rejected so a
// late duplicate cannot roll back newer state.
//
// Registration advances monotonically by sequence number: only a write whose
// seq is strictly greater than the currently recorded sequence is applied. A
// retried registration that timed out and arrives after a newer registration
// carries an older seq, so it is rejected instead of overwriting the newer
// version/address. The recorded sequence is never rolled back, so the
// registry keeps a single authoritative notion of which write is the latest.
func (r *Registry) RegisterSeq(svc string, inst *model.Instance, seq uint64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Reject a stale retry: a sequence that is not strictly newer than the
	// current one belongs to an earlier registration that already lost the
	// race to a newer write. Applying it would roll the version and address
	// back to the old values and revert the recorded sequence, so a late
	// duplicate would silently win over the original.
	if cur, ok := r.seqs[inst.ID]; ok && seq <= cur {
		return false
	}
	if r.byService[svc] == nil {
		r.byService[svc] = make(map[string]*model.Instance)
	}
	if r.services[svc] == nil {
		r.services[svc] = model.NewService(svc)
	}
	delete(r.deleted, inst.ID)
	r.instances[inst.ID] = inst
	r.byService[svc][inst.ID] = inst
	r.seqs[inst.ID] = seq
	return true
}

// RegisterContext registers an instance unless the context was cancelled.
func (r *Registry) RegisterContext(ctx context.Context, svc string, inst *model.Instance) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.Register(svc, inst)
	return nil
}

// RegisterInflight completes an in-flight registration that started before a
// concurrent unregister. It fails when the id was deleted in the meantime.
func (r *Registry) RegisterInflight(svc string, inst *model.Instance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.deleted[inst.ID] {
		return errDeleted
	}
	if r.byService[svc] == nil {
		r.byService[svc] = make(map[string]*model.Instance)
	}
	if r.services[svc] == nil {
		r.services[svc] = model.NewService(svc)
	}
	r.instances[inst.ID] = inst
	r.byService[svc][inst.ID] = inst
	return nil
}

// ConfirmDurable marks an instance registration as durably committed.
func (r *Registry) ConfirmDurable(instanceID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.instances[instanceID] == nil {
		return false
	}
	r.confirmed[instanceID] = true
	return true
}

// Confirmed reports whether an instance registration was durably committed.
func (r *Registry) Confirmed(instanceID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.confirmed[instanceID]
}

// Ack records an acknowledgement time for an instance.
func (r *Registry) Ack(instanceID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	inst := r.instances[instanceID]
	if inst == nil {
		return false
	}
	inst.LastAck = time.Now()
	return true
}

// TouchHeartbeat refreshes the last heartbeat time of an instance.
func (r *Registry) TouchHeartbeat(instanceID string, at time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	inst := r.instances[instanceID]
	if inst == nil {
		return false
	}
	inst.LastHeartbeat = at
	return true
}

// Unregister removes an instance and marks it unregistered.
func (r *Registry) Unregister(instanceID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	inst := r.instances[instanceID]
	if inst == nil {
		return false
	}
	r.deleted[instanceID] = true
	delete(r.instances, instanceID)
	delete(r.byService[inst.Service], instanceID)
	delete(r.confirmed, instanceID)
	inst.Status = model.InstanceEvicted
	return true
}

// Remove drops an instance without returning it (used by eviction).
func (r *Registry) Remove(instanceID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	inst := r.instances[instanceID]
	if inst == nil {
		return false
	}
	delete(r.instances, instanceID)
	delete(r.byService[inst.Service], instanceID)
	delete(r.confirmed, instanceID)
	inst.Status = model.InstanceEvicted
	return true
}

// Get returns an instance by id.
func (r *Registry) Get(instanceID string) (*model.Instance, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	inst, ok := r.instances[instanceID]
	return inst, ok
}

// Instances returns a copy of the instances of a service.
func (r *Registry) Instances(svc string) []*model.Instance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*model.Instance, 0, len(r.byService[svc]))
	for _, inst := range r.byService[svc] {
		out = append(out, inst)
	}
	return out
}

// Services returns the sorted service names.
func (r *Registry) Services() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.services))
	for name := range r.services {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// BeginRequest tracks an in-flight routed request on an instance.
func (r *Registry) BeginRequest(instanceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inflight[instanceID]++
}

// EndRequest releases an in-flight routed request.
func (r *Registry) EndRequest(instanceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.inflight[instanceID] > 0 {
		r.inflight[instanceID]--
	}
}

// Inflight returns the in-flight request count of an instance.
func (r *Registry) Inflight(instanceID string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.inflight[instanceID]
}

// Snapshot returns per-service stats.
func (r *Registry) Snapshot() []model.Stats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]model.Stats, 0, len(r.services))
	for name := range r.services {
		insts := r.byService[name]
		st := model.Stats{Service: name, Instances: len(insts)}
		for _, inst := range insts {
			switch inst.Status {
			case model.InstanceHealthy:
				st.Healthy++
			case model.InstanceSuspect:
				st.Suspect++
			case model.InstanceEvicted:
				st.Evicted++
			}
		}
		out = append(out, st)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Service < out[j].Service })
	return out
}

// Service returns a service by name.
func (r *Registry) Service(name string) (*model.Service, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.services[name]
	return s, ok
}

// AddTag attaches a tag to a service.
func (r *Registry) AddTag(svc, tag string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.services[svc]
	if s == nil {
		s = model.NewService(svc)
		r.services[svc] = s
	}
	for _, t := range s.Tags {
		if t == tag {
			return
		}
	}
	s.Tags = append(s.Tags, tag)
}
