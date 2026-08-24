package lease

import "time"

// LeaseLoop renews an instance lease until stop.
func (s *Store) LeaseLoop(instanceID string, ttl, interval time.Duration, stop <-chan struct{}) bool {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return true
		case <-ticker.C:
			if !s.Renew(instanceID, ttl) {
				return false
			}
		}
	}
}
