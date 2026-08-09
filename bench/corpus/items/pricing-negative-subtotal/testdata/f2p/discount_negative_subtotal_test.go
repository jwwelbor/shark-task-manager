package pricing

import "testing"

// TestApplyDiscount_RejectsNegativeSubtotal is the held-back F2P oracle for
// corpus item pricing-negative-subtotal (T-E40-F01-002). It fails at the
// fixture's base commit, where ApplyDiscount silently accepts a negative
// subtotal, and passes once reference.patch adds the validation.
func TestApplyDiscount_RejectsNegativeSubtotal(t *testing.T) {
	if _, err := ApplyDiscount(-100, 10); err == nil {
		t.Fatal("ApplyDiscount(-100, 10) expected error, got nil")
	}
}
