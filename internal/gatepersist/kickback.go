package gatepersist

import (
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
// It also re-checks, defense-in-depth, that no kickback targets the bound
// main entity — internal/gateresult.ValidateRole already enforces this
// role-independently before this package is reached, but a gate worker must
// never be able to bypass the guarded main-entity transition through a
// kickback, so this coordinator does not trust that upstream check alone.
func validateKickbacks(kickbacks []gateresult.Kickback, mainEntityKey string, validator StatusValidator) (map[string]models.EntityType, error) {
	entityTypes := make(map[string]models.EntityType, len(kickbacks))
	for _, k := range kickbacks {
		if k.EntityKey == mainEntityKey {
			return nil, fmt.Errorf("gatepersist: kickback entity_key %q must differ from the bound main entity", k.EntityKey)
		}
		entityType, err := kickbackEntityType(k.EntityKey)
		if err != nil {
			return nil, err
		}
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
const kickbackTokenPrefix = "[gatepersist:sub="

var kickbackTokenPattern = regexp.MustCompile(`\[gatepersist:sub=([0-9a-f]{64}):digest=([0-9a-f]{64})\]`)

// buildKickbackReason appends the bounded suboperation token and content
// digest to reason.
func buildKickbackReason(reason, suboperationID, contentDigest string) string {
	return strings.TrimSpace(reason) + " " + kickbackTokenPrefix + suboperationID + ":digest=" + contentDigest + "]"
}

// parseKickbackToken extracts the suboperation ID and content digest
// embedded by buildKickbackReason from a history entry's Notes field, if
// present.
func parseKickbackToken(notes string) (suboperationID, contentDigest string, ok bool) {
	m := kickbackTokenPattern.FindStringSubmatch(notes)
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}
