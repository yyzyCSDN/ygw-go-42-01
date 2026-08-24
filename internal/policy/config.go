package policy

// Config bundles all registry policies.
type Config struct {
	Heartbeat HeartbeatPolicy
	Eviction  EvictionPolicy
	Quota     QuotaPolicy
}

// QuotaPolicy configures per-service registration limits.
type QuotaPolicy struct {
	DefaultLimit int
	Burst        int
}

// DefaultQuotaPolicy returns the standard quota settings.
func DefaultQuotaPolicy() QuotaPolicy {
	return QuotaPolicy{DefaultLimit: 50, Burst: 10}
}

// DefaultConfig returns the bundled default policies.
func DefaultConfig() Config {
	return Config{
		Heartbeat: DefaultHeartbeatPolicy(),
		Eviction:  DefaultEvictionPolicy(),
		Quota:     DefaultQuotaPolicy(),
	}
}

// Valid reports whether the whole config is sane.
func (c Config) Valid() bool {
	return c.Heartbeat.Valid() && c.Quota.DefaultLimit > 0 && c.Quota.Burst > 0
}
