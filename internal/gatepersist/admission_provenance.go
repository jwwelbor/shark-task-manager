package gatepersist

import (
	"context"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// BoundAdmissionVersion identifies operations with run-bound transition history.
const BoundAdmissionVersion = 1

func needsBoundAdmission(c *Coordinator, req Request) bool {
	return c.Admission != nil && req.EntityType == models.EntityTypeBug && req.Role == gateresult.RoleSuccess
}

// BoundTransitionReason identifies the result whose guarded transition was
// durably applied. It is shared with bug receipt validation.
func BoundTransitionReason(gate, outcome, runID, digest string) string {
	return fmt.Sprintf("gate %s outcome %s run %s operation %s", gate, outcome, runID, digest)
}

func (c *Coordinator) markBoundAdmission(req Request, state *gaterun.OperationState) error {
	if !needsBoundAdmission(c, req) || state.PersistenceState != gaterun.PersistenceStatePending || state.AdmissionVersion != 0 {
		return nil
	}
	state.AdmissionVersion = BoundAdmissionVersion
	if err := state.Save(req.RunDir); err != nil {
		return fmt.Errorf("gatepersist: save admission version: %w", err)
	}
	return nil
}

func (c *Coordinator) requireBoundTransition(ctx context.Context, req Request, state *gaterun.OperationState) error {
	if state.AdmissionVersion != BoundAdmissionVersion {
		return nil
	}
	entries, err := c.History.GetHistory(ctx, req.EntityType, req.EntityKey)
	if err != nil {
		return fmt.Errorf("gatepersist: read transition provenance: %w", err)
	}
	latest := latestTransition(entries)
	reason := BoundTransitionReason(req.Gate, req.OutcomeKey, req.RunID, state.OperationDigest)
	if !matchesBoundTransition(latest, req, reason, c.normalizeHistoryStatus) {
		return fmt.Errorf("gatepersist: target transition does not belong to this run")
	}
	return nil
}

func latestTransition(entries []*models.EntityHistory) *models.EntityHistory {
	var latest *models.EntityHistory
	for _, entry := range entries {
		if entry != nil && (latest == nil || entry.ID > latest.ID) {
			latest = entry
		}
	}
	return latest
}

func (c *Coordinator) normalizeHistoryStatus(entityType models.EntityType, status string) string {
	if normalizer, ok := c.Validator.(interface {
		NormalizeStatus(models.EntityType, string) string
	}); ok {
		return normalizer.NormalizeStatus(entityType, status)
	}
	return status
}

func (c *Coordinator) sameStatus(entityType models.EntityType, a, b string) bool {
	return strings.EqualFold(c.normalizeHistoryStatus(entityType, a), c.normalizeHistoryStatus(entityType, b))
}

func matchesBoundTransition(entry *models.EntityHistory, req Request, reason string, normalize func(models.EntityType, string) string) bool {
	if entry == nil || entry.Forced || entry.FromStatus == nil || entry.Notes == nil {
		return false
	}
	return strings.EqualFold(normalize(req.EntityType, *entry.FromStatus), normalize(req.EntityType, req.SourceStatus)) && strings.EqualFold(normalize(req.EntityType, entry.ToStatus), normalize(req.EntityType, req.TargetStatus)) && *entry.Notes == reason
}
