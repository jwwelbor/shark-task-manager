package gatepersist

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/keys"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// KickbackValidationError reports that a kickback's target_status is not a
// status/outcome defined in its target entity's own configured workflow —
// the target-entity workflow-membership check internal/gateresult's package
// doc defers to this coordinator (REQ-F-002). Every kickback is checked
// before any write, so a single invalid kickback rejects the whole result
// without partial mutation.
type KickbackValidationError struct {
	EntityKey    string
	TargetStatus string
}

func (e *KickbackValidationError) Error() string {
	return fmt.Sprintf("gatepersist: kickback target entity %s has no configured status/outcome %q in its workflow", e.EntityKey, e.TargetStatus)
}

// KickbackConflictError reports that a durable kickback record already
// exists for this suboperation ID with different target status or reason
// than the one now being applied — a conflicting replay, which must fail
// closed rather than silently reapplying different content under an ID
// already recorded complete (REQ-F-003).
type KickbackConflictError struct {
	EntityKey string
}

func (e *KickbackConflictError) Error() string {
	return fmt.Sprintf("gatepersist: kickback for entity %s already durably recorded with different target status or reason; conflicting replay rejected", e.EntityKey)
}

// kickbackEntityType resolves the entity type of a kickback's target key
// from its key shape (e.g. "T-E07-F01-002" -> task, "E07-F02" -> feature).
// An unrecognized key shape is itself a validation failure — a kickback
// cannot target an entity this system cannot address.
func kickbackEntityType(entityKey string) (models.EntityType, error) {
	ks := keys.NewKeyService()
	detected := ks.DetectEntityType(entityKey)
	if detected == keys.EntityTypeUnknown {
		return "", fmt.Errorf("gatepersist: kickback entity key %q does not match any known entity key shape", entityKey)
	}
	entityType := models.EntityType(string(detected))
	if !models.ValidEntityTypes[entityType] {
		return "", fmt.Errorf("gatepersist: kickback entity key %q resolved to unsupported entity type %q", entityKey, detected)
	}
	return entityType, nil
}

// validateKickbacks checks every kickback's target-entity workflow
// membership up front, before any write, per the acceptance criterion "a
// kickback targeting an invalid status for its target entity's workflow is
// rejected without partial mutation." It returns the resolved entity type
// for each kickback (by entity key) so the caller does not re-derive it.
//
// It also re-checks, in two layers, that no kickback targets the bound main
// entity — internal/gateresult.ValidateRole already enforces a syntactic
// version of this role-independently before this package is reached, but a
// gate worker must never be able to bypass the guarded main-entity
// transition through a kickback, so this coordinator does not trust that
// upstream check alone:
//
//  1. A cheap syntactic (keys.KeyService.Normalize) comparison, same as
//     ValidateRole's — catches slugged/short-form/T-prefixed aliases without
//     a repository round-trip.
//  2. The AUTHORITATIVE check: resolve both the main entity and every
//     kickback target through resolver (IdentityResolver), the same
//     repository-backed key resolution production transitions use, and
//     compare resolved (entityType, id). Normalize alone cannot fold every
//     alias production resolution folds — e.g. a feature's bare "F05"
//     suffix has no epic context for Normalize to fold into "E34-F05", but
//     FeatureRepository.GetByKey's suffix-match resolves both to the same
//     row (code-review round 12 finding, reopening round 11's fix for
//     feature-typed gates specifically). Every kickback is resolved
//     (not only same-type ones), so an unresolvable kickback target also
//     fails closed here, before any write.
//
// It also rejects, via the SAME resolver-backed (entityType, id) comparison,
// any two kickbacks within this result that resolve to the same real
// entity — the authoritative counterpart of gateresult.Validate()'s own
// Normalize-based kickback-entity_key dedup, which is syntactic-only for the
// same reason ValidateRole's main-entity check is (code-review round 12
// finding: two aliases of one real entity must not each get an
// independently-applied kickback transition).
func validateKickbacks(ctx context.Context, kickbacks []gateresult.Kickback, mainEntityType models.EntityType, mainEntityKey string, validator StatusValidator, resolver IdentityResolver) (map[string]models.EntityType, error) {
	ks := keys.NewKeyService()
	canonicalMain := ks.Normalize(mainEntityKey)

	mainID, err := resolver.ResolveEntityID(ctx, mainEntityType, mainEntityKey)
	if err != nil {
		return nil, fmt.Errorf("gatepersist: resolve bound main entity %s %s: %w", mainEntityType, mainEntityKey, err)
	}

	// seenKickbackIdentities tracks the resolved (entityType, id) tuple of
	// every kickback processed so far, for the cross-kickback dedup check
	// below. This is the AUTHORITATIVE version of the raw-string dedup
	// gateresult.Validate() performs on entity_key (upgraded to a
	// Normalize-based comparison as a syntactic first pass, code-review
	// round 12): two kickbacks whose entity_key values are different
	// textual aliases of the SAME real entity (e.g. a feature's bare suffix
	// "F05" vs. its full form "E34-F05", which Normalize alone cannot fold
	// for the same reason it cannot fold that alias against the main
	// entity, see IdentityResolver's doc comment) would otherwise both pass
	// every check above and get applied as two sequential, independently
	// workflow-legal transitions to one real entity within a single gate
	// result -- defeating the one-kickback-per-entity design intent.
	type resolvedIdentity struct {
		entityType models.EntityType
		id         int64
	}
	seenKickbackIdentities := make(map[resolvedIdentity]string, len(kickbacks))

	entityTypes := make(map[string]models.EntityType, len(kickbacks))
	for _, k := range kickbacks {
		// Layer 1: syntactic. See internal/gateresult.ValidateRole's
		// matching check for why a slugged/short-form alias of
		// mainEntityKey must be rejected here too
		// (authorization-bypass-via-key-aliasing, code-review round 11).
		if ks.Normalize(k.EntityKey) == canonicalMain {
			return nil, fmt.Errorf("gatepersist: kickback entity_key %q must differ from the bound main entity", k.EntityKey)
		}
		entityType, err := kickbackEntityType(k.EntityKey)
		if err != nil {
			return nil, err
		}

		// Layer 2: authoritative, repository-backed. Resolved for every
		// kickback regardless of whether entityType matches
		// mainEntityType, so a kickback naming a nonexistent entity also
		// fails closed here rather than surfacing later at Transition time
		// after other writes have already landed.
		kickbackID, err := resolver.ResolveEntityID(ctx, entityType, k.EntityKey)
		if err != nil {
			return nil, fmt.Errorf("gatepersist: resolve kickback entity_key %q: %w", k.EntityKey, err)
		}
		if entityType == mainEntityType && kickbackID == mainID {
			return nil, fmt.Errorf("gatepersist: kickback entity_key %q resolves to the bound main entity %q via repository-backed key resolution; a kickback must target a different entity", k.EntityKey, mainEntityKey)
		}

		// Layer 3: cross-kickback authoritative dedup. Every kickback is
		// checked against every OTHER kickback already processed, resolved
		// through the same repository-backed lookup, so two aliases of one
		// real entity fail closed here before any write.
		identity := resolvedIdentity{entityType: entityType, id: kickbackID}
		if priorKey, exists := seenKickbackIdentities[identity]; exists {
			return nil, fmt.Errorf("gatepersist: kickback entity_key %q resolves to the same real entity as kickback entity_key %q via repository-backed key resolution; each kickback must target a distinct entity", k.EntityKey, priorKey)
		}
		seenKickbackIdentities[identity] = k.EntityKey

		if !validator.IsValidStatus(entityType, k.TargetStatus) {
			return nil, &KickbackValidationError{EntityKey: k.EntityKey, TargetStatus: k.TargetStatus}
		}
		entityTypes[k.EntityKey] = entityType
	}
	return entityTypes, nil
}

// kickbackTokenPrefix marks the bounded machine-readable suboperation token
// this package embeds in a kickback transition's audit reason, per
// architecture.md step 7: "kickback reasons/history store its bounded
// machine token." It is appended after the worker-supplied human reason so
// the reason remains readable while still carrying a durable, parseable
// suboperation identity AND content digest for reconciliation (reconcile.go).
//
// The content digest is embedded alongside the suboperation ID because
// REQ-F-003 requires a conflicting replay to fail closed on a differing
// target status OR reason, not status alone (round-2 UAT rejection: the
// original token carried only the suboperation ID, so reconcile.go could
// only compare target status — TD-178's gap). op.contentDigest() already covers
// entity key, target status, AND reason (operations.go), so embedding it
// here gives reconcile.go the same full-content comparison notes already
// get via metaContentDigest.
//
// The run_id is embedded for the same reason a note's metadata carries
// metaRunID: gaterun.ComputeOperationDigest (and therefore
// operation.suboperationID) never includes run_id, so two different runs
// against the same entity/source_status/gate/envelope legitimately derive
// the identical suboperation ID. Without run_id in the token,
// reconcile.go's kickback branch could not filter candidate history records
// by the run it is reconciling for — the asymmetry with the notes branch
// two lines above it (code-review round 11 finding) — and could misread a
// DIFFERENT run's durably-applied kickback as this run's own completed
// suboperation.
const kickbackTokenPrefix = "[gatepersist:sub="

var kickbackTokenPattern = regexp.MustCompile(`\[gatepersist:sub=([0-9a-f]{64}):digest=([0-9a-f]{64}):run=([A-Za-z0-9._-]{1,128})\]`)

// buildKickbackReason appends the bounded suboperation token, content
// digest, and run ID to reason.
func buildKickbackReason(reason, suboperationID, contentDigest, runID string) string {
	return strings.TrimSpace(reason) + " " + kickbackTokenPrefix + suboperationID + ":digest=" + contentDigest + ":run=" + runID + "]"
}

// parseKickbackToken extracts the suboperation ID, content digest, and run
// ID embedded by buildKickbackReason from a history entry's Notes field, if
// present.
func parseKickbackToken(notes string) (suboperationID, contentDigest, runID string, ok bool) {
	m := kickbackTokenPattern.FindStringSubmatch(notes)
	if m == nil {
		return "", "", "", false
	}
	return m[1], m[2], m[3], true
}
