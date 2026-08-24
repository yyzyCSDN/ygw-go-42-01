// Package model defines the core domain types of the service registry:
// services, instances, leases, health and version state.
package model

// Service is a named group of instances that share a discovery key.
type Service struct {
	Name     string
	Versions []string
	Tags     []string
}

// NewService creates a service with the given name.
func NewService(name string) *Service {
	return &Service{Name: name}
}

// HasTag reports whether the service carries a tag.
func (s *Service) HasTag(tag string) bool {
	for _, t := range s.Tags {
		if t == tag {
			return true
		}
	}
	return false
}
