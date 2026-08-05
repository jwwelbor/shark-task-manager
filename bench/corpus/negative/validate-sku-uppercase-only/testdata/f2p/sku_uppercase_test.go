package validate

import "testing"

// TestSKU_RejectsLowercaseLetters is the held-back F2P oracle for negative
// corpus item validate-sku-uppercase-only (T-E40-F01-004, rejection branch
// (d)). SKU has no uppercase-only rule at the fixture's base commit, so
// this test fails at base as expected. reference.patch is deliberately
// incomplete: it adds only a TODO comment above SKU and never adds the
// guard clause, so the test is still red after the patch is applied.
// bench/scripts/admit.sh (T-E40-F01-006) must reject this candidate at
// check (d) — F2P still red after the patch — never admitting it.
func TestSKU_RejectsLowercaseLetters(t *testing.T) {
	if err := SKU("sku-1"); err == nil {
		t.Fatal("SKU(\"sku-1\") expected error for lowercase letters, got nil")
	}
}
