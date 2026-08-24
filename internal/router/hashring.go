package router

import (
	"sort"

	"github.com/cespare/xxhash/v2"

	"svcregistry/internal/model"
)

// HashRing maps a key to an instance by hashing the key over instance ids.
type HashRing struct{}

// NewHashRing creates a consistent-hash strategy.
func NewHashRing() *HashRing {
	return &HashRing{}
}

// Pick selects the instance whose id hashes closest to the key hash.
func (h *HashRing) Pick(key string, instances []*model.Instance) *model.Instance {
	if len(instances) == 0 {
		return nil
	}
	keyHash := xxhash.Sum64String(key)
	sorted := make([]*model.Instance, len(instances))
	copy(sorted, instances)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	best := sorted[0]
	bestDist := uint64(1<<64 - 1)
	for _, inst := range sorted {
		dist := keyHash ^ xxhash.Sum64String(inst.ID)
		if dist < bestDist {
			bestDist = dist
			best = inst
		}
	}
	return best
}
