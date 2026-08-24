package metric

import (
	"fmt"
	"sort"
	"strings"
)

// Report renders collector counters as a stable text table.
func (c *Collector) Report() string {
	snap := c.Snapshot()
	keys := make([]string, 0, len(snap))
	for k := range snap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%d ", k, snap[k])
	}
	return strings.TrimSpace(b.String())
}
