package gateresult

import (
	"strconv"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/keys"
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
	//
	// The comparison is canonical-identity-aware, not raw string equality:
	// entity_key is arbitrary bounded free text (boundedText performs no
	// key-shape validation), and production key resolution
	// (internal/repository/task.TaskRepository.GetByKey, via
	// parseSluggedKey) resolves a slugged alias of a key to the same
	// database row as its bare form. Comparing raw strings let a worker
	// submit e.g. "T-E34-F05-001-<slug>" as a kickback's entity_key when the
	// bound main entity is "T-E34-F05-001": textually different, but the
	// same entity once resolved — an authorization bypass of the guarded
	// main-entity transition (code-review round 11 finding). keys.KeyService
	// normalizes both sides to their canonical, slug-stripped, uppercased
	// form (task keys additionally collapse the short/T-prefixed alias
	// pair), matching parseSluggedKey's notion of "same entity" without
	// requiring database access at this structural-validation layer. An
	// unrecognized key shape normalizes to its uppercased self, which still
	// correctly compares equal only to an identical unrecognized string.
	//
	// IMPORTANT — this check is a SYNTACTIC FIRST PASS ONLY, not the
	// authoritative one: this package has no database access (by design —
	// it is the pure model/parser layer), and keys.KeyService.Normalize
	// cannot fold every alias production repository resolution folds. In
	// particular a feature's bare suffix form ("F05") has no epic context
	// for Normalize to resolve against, but
	// internal/repository/feature.FeatureRepository.GetByKey's suffix-match
	// resolves it to the same row as its full form ("E34-F05") — a gap this
	// check cannot catch (code-review round 12 finding, which reopened
	// round 11's fix for feature-typed gates specifically). The
	// authoritative check lives in internal/gatepersist.validateKickbacks,
	// which has repository access via its injected
	// gatepersist.IdentityResolver and resolves both the main entity and
	// every kickback target through the SAME repository-backed lookup
	// production transitions use before comparing identity. Any caller with
	// database access that needs to know "is this the same entity" MUST use
	// that pattern (resolve-then-compare via IdentityResolver), not a
	// Normalize-based string comparison — this check here exists only to
	// reject cheaply, defense-in-depth, before gatepersist is ever reached.
	ks := keys.NewKeyService()
	canonicalMain := ks.Normalize(mainEntityKey)
	for i, k := range result.Kickbacks {
		if ks.Normalize(k.EntityKey) == canonicalMain {
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
