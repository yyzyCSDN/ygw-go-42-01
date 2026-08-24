package health

// Unhealthy returns the instance ids that are currently not OK.
func (s *Status) Unhealthy(ids []string) []string {
	out := make([]string, 0)
	for _, id := range ids {
		if !s.Snapshot(id).OK {
			out = append(out, id)
		}
	}
	return out
}
