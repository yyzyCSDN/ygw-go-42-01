// Package registry owns the service table and per-service instance sets.
package registry

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"svcregistry/internal/model"
	"svcregistry/internal/persist"
)

// errDeleted is returned when an in-flight registration targets a deleted id.
var errDeleted = errors.New("registry: instance was deleted")

// Registry is the in-process source of truth for services and instances.
// A registration is only confirmed once its instance record (and its lease,
// persisted alongside it) has been fsynced to disk; the confirmation is
// never returned before that durability barrier succeeds.
type Registry struct {
	mu         sync.RWMutex
	services   map[string]*model.Service
	instances  map[string]*model.Instance
	byService  map[string]map[string]*model.Instance
	confirmed  map[string]bool
	inflight   map[string]int
	seqs       map[string]uint64
	deleted    map[string]bool
	persister  persist.Persister
}

// New creates an empty registry backed by an in-memory (non-durable)
// persister. Use NewWithPersister when crash-safe durability is required.
func New() *Registry {
	return NewWithPersister(persist.NewNop())
}

// NewWithPersister creates a registry that gates registration confirmations
// on the given persister's durability barrier.
func NewWithPersister(p persist.Persister) *Registry {
	return &Registry{
		services:  make(map[string]*model.Service),
		instances: make(map[string]*model.Instance),
		byService:  make(map[string]map[string]*model.Instance),
		confirmed: make(map[string]bool),
		inflight:  make(map[string]int),
		seqs:      make(map[string]uint64),
		deleted:   make(map[string]bool),
		persister: p,
	}
}

// Persister returns the durability barrier the registry confirms against.
func (r *Registry) Persister() persist.Persister {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.persister
}

// Register adds an instance to the registry under its service. The instance is
// recorded as an unconfirmed intent: it is staged in the durability log but
// not yet confirmed to the caller. Confirmation only happens once
// ConfirmDurable fsyncs both the instance record and its lease (staged
// separately via the lease store) and reports success — so the success
// acknowledgement never outruns the data a crash would lose. A fresh
// registration clears any previous delete marker for the id. The returned
// sequence number lets stale retries be rejected.
func (r *Registry) Register(svc string, inst *model.Instance) uint64 {
	r.mu.Lock()
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
	// This is now an unconfirmed intent: deliberately NOT confirmed here.
	// Stage the registration intent so ConfirmDurable can fsync it (and the
	// lease staged alongside it) before the caller is told it succeeded.
	r.confirmed[inst.ID] = false
	persister := r.persister
	r.mu.Unlock()

	if persister != nil {
		persister.Append(persist.Record{
			Kind:       persist.KindRegister,
			InstanceID: inst.ID,
			Service:    svc,
			Instance:   inst,
		})
	}
	return r.seqs[inst.ID]
}

// RegisterSeq registers an instance carrying an explicit sequence number. A
// stale retry (older sequence than the current registration) is rejected so a
// late duplicate cannot roll back newer state.
func (r *Registry) RegisterSeq(svc string, inst *model.Instance, seq uint64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if seq < r.seqs[inst.ID] {
		return false
	}
	if r.byService[svc] == nil {
		r.byService[svc] = make(map[string]*model.Instance)
	}
	if r.services[svc] == nil {
		r.services[svc] = model.NewService(svc)
	}
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

// ConfirmDurable is the durability barrier for a registration. It fsyncs the
// instance record together with any lease staged for the same instance, and
// only then marks the registration confirmed. The caller must not
// acknowledge success to its own caller before this returns true, so the
// confirmation can never outrun the data a crash would lose. It returns
// false when the instance is unknown or the durability barrier failed.
func (r *Registry) ConfirmDurable(instanceID string) bool {
	r.mu.Lock()
	if r.instances[instanceID] == nil {
		r.mu.Unlock()
		return false
	}
	persister := r.persister
	r.mu.Unlock()

	// Commit the staged instance + lease intents to stable storage before
	// telling the caller the registration succeeded. A failure here must
	// leave the registration unconfirmed.
	if persister != nil {
		if err := persister.Commit(instanceID); err != nil {
			return false
		}
	}

	r.mu.Lock()
	r.confirmed[instanceID] = true
	r.mu.Unlock()
	return true
}

// Confirmed reports whether an instance registration was durably committed.
func (r *Registry) Confirmed(instanceID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.confirmed[instanceID]
}

// Recover replays the durability log and rebuilds registry state from the
// committed records. Only fsynced (committed) intents are applied; an
// intent that was staged but never committed is discarded, which is the
// property callers rely on: state they were never told succeeded does not
// reappear after a crash. Instance records are restored confirmed, since a
// committed record is by definition one whose confirmation was earned.
// Lease records are returned to the caller so the lease store can rebuild
// its table from the same replay (call lease.Store.Recover with them).
func (r *Registry) Recover() ([]persist.Record, error) {
	if r.persister == nil {
		return nil, nil
	}
	records, err := r.persister.Replay()
	if err != nil {
		return nil, err
	}
	var leases []persist.Record
	r.mu.Lock()
	for _, rec := range records {
		switch rec.Kind {
		case persist.KindRegister:
			if rec.Instance == nil {
				continue
			}
			inst := rec.Instance
			if r.services[rec.Service] == nil {
				r.services[rec.Service] = model.NewService(rec.Service)
			}
			if r.byService[rec.Service] == nil {
				r.byService[rec.Service] = make(map[string]*model.Instance)
			}
			r.instances[inst.ID] = inst
			r.byService[rec.Service][inst.ID] = inst
			r.seqs[inst.ID]++
			// A replayed registration was committed, therefore confirmed.
			r.confirmed[inst.ID] = true
		case persist.KindLease:
			leases = append(leases, rec)
		}
	}
	r.mu.Unlock()
	return leases, nil
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
