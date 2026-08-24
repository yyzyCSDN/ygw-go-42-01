package evict

import "svcregistry/internal/model"

// Quarantine is a bounded list of recently evicted instance ids.
type Quarantine struct {
	ids   []string
	limit int
}

// NewQuarantine creates a quarantine list.
func NewQuarantine(limit int) *Quarantine {
	return &Quarantine{limit: limit}
}

// Add records an evicted instance.
func (q *Quarantine) Add(inst *model.Instance) {
	q.ids = append(q.ids, inst.ID)
	if len(q.ids) > q.limit {
		q.ids = q.ids[len(q.ids)-q.limit:]
	}
}

// Contains reports whether an id is quarantined.
func (q *Quarantine) Contains(id string) bool {
	for _, v := range q.ids {
		if v == id {
			return true
		}
	}
	return false
}

// Size returns the quarantine length.
func (q *Quarantine) Size() int {
	return len(q.ids)
}
