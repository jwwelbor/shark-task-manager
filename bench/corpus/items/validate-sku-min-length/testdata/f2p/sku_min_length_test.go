package validate

import "testing"

// TestSKU_RejectsTooShortValue is the held-back F2P oracle for corpus item
// validate-sku-min-length (T-E40-F01-003). It fails at the fixture's base
// commit, where SKU accepts a single-character value, and passes once
// reference.patch adds a minimum-length check.
func TestSKU_RejectsTooShortValue(t *testing.T) {
	if err := SKU("a"); err == nil {
		t.Fatal("SKU(\"a\") expected error, got nil")
	}
}
