package model

// QuotaState is the per-service registration quota usage.
type QuotaState struct {
	Service   string
	Limit     int
	Used      int
	Exceeded  bool
}

// Available reports whether more instances may register.
func (q QuotaState) Available() bool {
	return !q.Exceeded && q.Used < q.Limit
}
