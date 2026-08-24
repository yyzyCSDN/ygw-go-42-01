package metric

import "sync"

// Histogram records latency samples.
type Histogram struct {
	mu  sync.Mutex
	all []int64
	sum int64
}

// NewHistogram creates an empty histogram.
func NewHistogram() *Histogram {
	return &Histogram{}
}

// Record adds a sample.
func (h *Histogram) Record(ms int64) {
	h.mu.Lock()
	h.all = append(h.all, ms)
	h.sum += ms
	h.mu.Unlock()
}

// Average returns the mean sample.
func (h *Histogram) Average() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.all) == 0 {
		return 0
	}
	return float64(h.sum) / float64(len(h.all))
}
