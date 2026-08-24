package model

// InstanceMeta is the key/value metadata attached to an instance.
type InstanceMeta struct {
	InstanceID string
	Values     map[string]string
}

// Get returns a metadata value.
func (m InstanceMeta) Get(key string) string {
	return m.Values[key]
}

// Clone returns an independent copy of the metadata map.
func (m InstanceMeta) Clone() map[string]string {
	out := make(map[string]string, len(m.Values))
	for k, v := range m.Values {
		out[k] = v
	}
	return out
}
