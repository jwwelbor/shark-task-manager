package commands

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// TestResolvedResultContract_DefaultsToLegacy proves the `shark next --json`
// wire layer always emits a non-empty result_contract (REQ-F-006: "shark
// next exposes the resolved value") even for a nil NextStatusInfo or one
// built by a caller (e.g. question_service.go's handoff literals) that
// predates the ResultContract field.
func TestResolvedResultContract_DefaultsToLegacy(t *testing.T) {
	if got := resolvedResultContract(nil); got != "legacy" {
		t.Errorf("expected legacy for nil nextInfo, got %q", got)
	}
	if got := resolvedResultContract(&services.NextStatusInfo{}); got != "legacy" {
		t.Errorf("expected legacy for an empty ResultContract, got %q", got)
	}
}

// TestResolvedResultContract_ResolvesConfiguredValue proves a populated
// ResultContract passes through unchanged.
func TestResolvedResultContract_ResolvesConfiguredValue(t *testing.T) {
	info := &services.NextStatusInfo{
		ResultContract: "gate_result_v1",
		OutcomeRoles:   map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess},
	}
	if got := resolvedResultContract(info); got != "gate_result_v1" {
		t.Errorf("expected gate_result_v1, got %q", got)
	}
}
