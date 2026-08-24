// Package control exposes operator-facing registry operations.
package control

import (
	"sort"

	"svcregistry/internal/evict"
	"svcregistry/internal/meta"
	"svcregistry/internal/registry"
	"svcregistry/internal/version"
)

// Controller is the operator surface of the registry.
type Controller struct {
	reg      *registry.Registry
	versions *version.Store
	meta     *meta.Store
	evictor  *evict.Evictor
}

// New creates a controller.
func New(reg *registry.Registry, versions *version.Store, m *meta.Store, e *evict.Evictor) *Controller {
	return &Controller{reg: reg, versions: versions, meta: m, evictor: e}
}

// ListServices returns the service names.
func (c *Controller) ListServices() []string {
	return c.reg.Services()
}

// ListInstances returns the instance ids of a service.
func (c *Controller) ListInstances(service string) []string {
	ids := make([]string, 0)
	for _, inst := range c.reg.Instances(service) {
		ids = append(ids, inst.ID)
	}
	sort.Strings(ids)
	return ids
}

// DeleteService removes every instance of a service.
func (c *Controller) DeleteService(service string) int {
	removed := 0
	for _, inst := range c.reg.Instances(service) {
		if c.reg.Remove(inst.ID) {
			c.meta.Remove(inst.ID)
			removed++
		}
	}
	return removed
}

// ResetService re-registers the version weights of a service.
func (c *Controller) ResetService(service string, versions []string) {
	for _, v := range versions {
		c.versions.Ensure(service, v)
		c.versions.Activate(service, v)
	}
}
