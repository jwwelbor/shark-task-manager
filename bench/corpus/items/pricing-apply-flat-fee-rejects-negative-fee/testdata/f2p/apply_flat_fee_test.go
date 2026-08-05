package pricing

import "testing"

// TestApplyFlatFee_RejectsNegativeFee is the held-back F2P oracle for
// corpus item pricing-apply-flat-fee-rejects-negative-fee (T-E40-F01-003).
// It fails at the fixture's base commit, where ApplyFlatFee silently
// accepts a negative fee, and passes once reference.patch adds the
// validation.
func TestApplyFlatFee_RejectsNegativeFee(t *testing.T) {
	if _, err := ApplyFlatFee(1000, -50); err == nil {
		t.Fatal("ApplyFlatFee(1000, -50) expected error, got nil")
	}
}
