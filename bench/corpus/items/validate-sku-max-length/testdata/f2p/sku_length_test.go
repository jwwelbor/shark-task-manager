package validate

import (
	"strings"
	"testing"
)

// TestSKU_RejectsOverlongValue is the held-back F2P oracle for corpus item
// validate-sku-max-length (T-E40-F01-002). It fails at the fixture's base
// commit, where SKU has no length limit, and passes once reference.patch
// adds the 40-character maximum.
func TestSKU_RejectsOverlongValue(t *testing.T) {
	overlong := strings.Repeat("a", 41)
	if err := SKU(overlong); err == nil {
		t.Fatalf("SKU(%d chars) expected error, got nil", len(overlong))
	}
}
