package model

// VersionState is the rollout/weight state of one service version.
type VersionState struct {
	Service string
	Version string
	Weight  int
	Active  bool
}

// VersionSnapshot is an immutable copy of a service's version weights.
type VersionSnapshot struct {
	Service  string
	Weights  map[string]int
	Active   map[string]bool
	Revision uint64
}

// ActiveVersions returns the versions currently taking traffic.
func (v VersionSnapshot) ActiveVersions() []string {
	out := make([]string, 0)
	for version, active := range v.Active {
		if active {
			out = append(out, version)
		}
	}
	return out
}
