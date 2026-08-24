package version

import "svcregistry/internal/model"

// ActiveVersions returns the versions that may currently take traffic.
func ActiveVersions(snap model.VersionSnapshot) []string {
	return snap.ActiveVersions()
}

// Weighted returns a weight-sorted list of active versions.
func Weighted(snap model.VersionSnapshot) []string {
	out := make([]string, 0)
	for version, active := range snap.Active {
		if active {
			out = append(out, version)
		}
	}
	return out
}
