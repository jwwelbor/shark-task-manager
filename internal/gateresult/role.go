package gateresult

import (
	"strconv"
	"strings"
)

// ValidateRole checks REQ-F-004/I-02 gate-completeness invariants for one
// already-structurally-valid GateResult against the semantic role the
// parent's workflow configuration assigned to the observed opaque outcome
// key (REQ-F-006's outcome_roles map — owned by the workflow layer, not this
// package). role is therefore always an explicit caller-supplied parameter:
// GateResult carries no role/outcome_roles field of its own, so a worker can
// never select its own validation rules.
//
// mainEntityKey is the bound main entity the parent observed this result
// for. It is required so the main-entity-kickback invariant can be checked
// regardless of role.
func ValidateRole(role OutcomeRole, result *GateResult, mainEntityKey string) error {
	if result == nil {
		return newValidationError("", ErrorClassSchema, "gate result is required")
	}
	if strings.TrimSpace(mainEntityKey) == "" {
		return newValidationError("main_entity_key", ErrorClassSchema, "is required")
	}

	switch role {
	case RoleSuccess, RoleRouteRework, RoleKickbackRework, RoleBlocked, RoleHold, RoleCancelled:
	default:
		return newValidationError("role", ErrorClassRole, "is not a supported outcome role")
	}

	// Role-independent: every kickback key must differ from the bound main
	// entity key regardless of role, so a pre-transition write cannot
	// invalidate the guarded source state.
	for i, k := range result.Kickbacks {
		if k.EntityKey == mainEntityKey {
			return newValidationError(fieldIndex("kickbacks", i)+".entity_key", ErrorClassRole, "must target a different entity from the bound main entity")
		}
	}

	switch role {
	case RoleSuccess:
		if len(result.Kickbacks) > 0 {
			return newValidationError("kickbacks", ErrorClassRole, "a success outcome must contain no kickback")
		}
		for i, f := range result.Findings {
			if f.Disposition == DispositionOpen || f.Disposition == DispositionSeverityConflict {
				return newValidationError(fieldIndex("findings", i)+".disposition", ErrorClassRole, "a success outcome must contain no open or severity_conflict blocking finding")
			}
		}
	case RoleRouteRework:
		if len(result.Kickbacks) > 0 {
			return newValidationError("kickbacks", ErrorClassRole, "a route_rework outcome must contain no kickback; its configured main-entity transition is the rework route")
		}
	case RoleKickbackRework:
		if len(result.Kickbacks) == 0 {
			return newValidationError("kickbacks", ErrorClassRole, "a kickback_rework outcome requires at least one child or cross-entity kickback")
		}
	case RoleBlocked, RoleHold, RoleCancelled:
		if len(result.Kickbacks) == 0 && strings.TrimSpace(result.NoKickbackReason) == "" {
			return newValidationError("no_kickback_reason", ErrorClassRole, "is required for a "+string(role)+" outcome with no kickback")
		}
	}
	return nil
}

func fieldIndex(field string, index int) string {
	return field + "[" + strconv.Itoa(index) + "]"
}
