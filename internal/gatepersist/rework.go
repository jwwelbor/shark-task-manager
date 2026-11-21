package gatepersist

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// KickbackQuestionBlockedError reports an open or answering Question that
// blocks a child review-rework transition.
type KickbackQuestionBlockedError struct {
	EntityKey   string
	QuestionKey string
}

func (e *KickbackQuestionBlockedError) Error() string {
	return fmt.Sprintf("gatepersist: kickback target %s is blocked by Question %s", e.EntityKey, e.QuestionKey)
}

// preflightKickbacks checks every target's direct Question gate and resolves
// every still-pending destination before any new target-store operation is
// applied. Completed operations are not re-routed: their actual destination
// is verified by the history reconciler.
func (c *Coordinator) preflightKickbacks(ctx context.Context, req Request, entityTypes map[string]models.EntityType, state *gaterun.OperationState, digest string) error {
	if len(req.Result.Kickbacks) == 0 {
		return nil
	}
	if c.ReworkRoutes == nil {
		return fmt.Errorf("gatepersist: rework route resolver is required for child kickbacks")
	}
	if c.QuestionBlocker == nil {
		return fmt.Errorf("gatepersist: QuestionBlocker is required for child kickbacks")
	}
	if c.RoutedRework == nil {
		return fmt.Errorf("gatepersist: routed rework transitioner is required for child kickbacks")
	}

	for _, kickback := range req.Result.Kickbacks {
		if err := c.preflightKickback(ctx, req, kickback, entityTypes, state, digest); err != nil {
			return err
		}
	}
	return nil
}

func (c *Coordinator) preflightKickback(ctx context.Context, req Request, kickback gateresult.Kickback, entityTypes map[string]models.EntityType, state *gaterun.OperationState, digest string) error {
	entityType, ok := entityTypes[kickback.EntityKey]
	if !ok {
		return fmt.Errorf("gatepersist: missing resolved entity type for kickback target %s", kickback.EntityKey)
	}
	if !c.RoutedRework.SupportsReviewReworkTarget(entityType) {
		return fmt.Errorf("gatepersist: no transaction-bound review rework path for %s targets", entityType)
	}
	if state != nil {
		subID := gaterun.DeriveSuboperationID(digest, kindKickback, kickback.EntityKey)
		if state.HasCompleted(subID) {
			return nil
		}
	}
	if err := c.checkKickbackQuestion(ctx, entityType, kickback.EntityKey); err != nil {
		return err
	}
	return c.validateKickbackRoute(ctx, req, kickback, entityType)
}

func (c *Coordinator) validateKickbackRoute(ctx context.Context, req Request, kickback gateresult.Kickback, entityType models.EntityType) error {
	fromStatus, err := c.Status.CurrentStatus(ctx, entityType, kickback.EntityKey)
	if err != nil {
		return fmt.Errorf("gatepersist: read kickback source status for %s: %w", kickback.EntityKey, err)
	}
	target, err := c.ReworkRoutes.ResolveReworkOutcome(entityType, fromStatus, req.OutcomeKey)
	if err != nil {
		return fmt.Errorf("gatepersist: resolve kickback route for %s from %s using outcome %s: %w", kickback.EntityKey, fromStatus, req.OutcomeKey, err)
	}
	if !c.Validator.IsValidStatus(entityType, target) {
		return fmt.Errorf("gatepersist: configured kickback route for %s targets unknown %s status %q", kickback.EntityKey, entityType, target)
	}
	if !c.sameStatus(entityType, target, kickback.TargetStatus) {
		return fmt.Errorf("gatepersist: GateResult v1 target_status %q for %s does not match configured route %q from %s on outcome %q", kickback.TargetStatus, kickback.EntityKey, target, fromStatus, req.OutcomeKey)
	}
	return nil
}

func (c *Coordinator) checkKickbackQuestions(ctx context.Context, req Request) error {
	if req.Result == nil || len(req.Result.Kickbacks) == 0 {
		return nil
	}
	if c.QuestionBlocker == nil {
		return fmt.Errorf("gatepersist: QuestionBlocker is required for child kickbacks")
	}
	for _, kickback := range req.Result.Kickbacks {
		entityType, err := kickbackEntityType(kickback.EntityKey)
		if err != nil {
			return err
		}
		if err := c.checkKickbackQuestion(ctx, entityType, kickback.EntityKey); err != nil {
			return err
		}
	}
	return nil
}

func (c *Coordinator) checkKickbackQuestion(ctx context.Context, entityType models.EntityType, entityKey string) error {
	block, err := c.QuestionBlocker.Check(ctx, entityType, entityKey)
	if err != nil {
		return fmt.Errorf("gatepersist: check Question gate for kickback target %s: %w", entityKey, err)
	}
	if block != nil {
		return &KickbackQuestionBlockedError{EntityKey: entityKey, QuestionKey: block.QuestionKey}
	}
	return nil
}
