package quota

import "testing"

func TestQuotaDebitSurvivesRollback(t *testing.T) {
	b := NewBucket(1, 1)
	if !b.Allow() {
		t.Fatalf("first registration slot should be allowed")
	}
	if b.Allow() {
		t.Fatalf("second slot must be denied (no refund)")
	}
}
