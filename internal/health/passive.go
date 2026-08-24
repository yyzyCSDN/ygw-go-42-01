package health

// PassiveWindow is the default accumulation window.
const PassiveWindow = 60

// RecordPassiveFailureDefault records one failure using the default window.
func (s *Status) RecordPassiveFailureDefault(instanceID string) {
	s.RecordPassiveFailure(instanceID)
}

// SuspectThreshold is the passive failure count that triggers suspect.
const SuspectThreshold = 3

// Suspect returns ids whose passive failures exceed the threshold.
func (s *Status) Suspect(ids []string) []string {
	out := make([]string, 0)
	for _, id := range ids {
		if s.Snapshot(id).PassiveCount >= SuspectThreshold {
			out = append(out, id)
		}
	}
	return out
}
