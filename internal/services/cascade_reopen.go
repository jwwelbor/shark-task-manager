package services

// cascade_reopen.go is the single source of truth for all parent-reopen logic.
//
// It exports the following package-level identifiers (all unexported — cascade logic
// is wired in task_service.go and feature_service.go):
//
//   - cascadeParentReopens  — walks the parent chain and reopens terminal ancestors
//   - resolveReopenTarget   — three-step fallback resolver for the reopen target status
//   - buildAutoReopenNotes  — formats the structured notes prefix per REQ-F-007
//   - cascadeDeps           — bundles all runtime dependencies
//   - cascadeTrigger        — describes what fired the cascade
//   - cascadeLeg            — typed constant for "start at feature" vs "start at epic"
//   - txBeginner            — narrow interface for BeginTxContext
//   - levelWorkflow         — narrow per-level workflow interface
//   - levelWorkflowProvider — narrow interface wrapping *workflow.Service
//   - Repo interfaces        — ParentReopenHistoryQuerier, CascadeFeatureRepo,
//                              CascadeEpicRepo, EntityHistoryTxRecorder
//
// All dependency injection is done through the cascadeDeps struct; no global state.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// workflowProviderAdapter wraps *workflow.Service and implements levelWorkflowProvider.
// This is necessary because *workflow.Service.ForLevel returns *workflow.Service
// (a concrete type), while levelWorkflowProvider.ForLevel must return levelWorkflow
// (an interface). The adapter bridges the concrete→interface gap for production wiring.
type workflowProviderAdapter struct {
	svc *workflow.Service
}

func (a *workflowProviderAdapter) ForLevel(level string) levelWorkflow {
	return a.svc.ForLevel(level)
}

// ============================================================
// Narrow repository interfaces (defined at point of use per convention)
// ============================================================

// ParentReopenHistoryQuerier is the narrow interface the cascade needs for
// resolving each ancestor's prior non-terminal status.
type ParentReopenHistoryQuerier interface {
	GetLastNonTerminalStatus(
		ctx context.Context,
		entityType models.EntityType,
		entityID int64,
		terminalStatuses []string,
	) (string, bool, error)
}

// CascadeFeatureRepo is the narrow read+Tx-update interface the cascade needs
// for the feature ancestor leg.
type CascadeFeatureRepo interface {
	GetByID(ctx context.Context, id int64) (*models.Feature, error)
	// GetByIDTx reads the feature inside an existing transaction, providing
	// snapshot isolation for the in-tx re-fetch that enforces idempotency
	// under concurrent cascades (REQ-F-008 / AC-T2).
	GetByIDTx(ctx context.Context, tx *sql.Tx, id int64) (*models.Feature, error)
	UpdateStatusTx(ctx context.Context, tx *sql.Tx, id int64, status string, agent *string, notes *string) error
}

// CascadeEpicRepo is the narrow read+Tx-update interface the cascade needs for
// the epic ancestor leg.
type CascadeEpicRepo interface {
	GetByID(ctx context.Context, id int64) (*models.Epic, error)
	// GetByIDTx reads the epic inside an existing transaction, providing
	// snapshot isolation for the in-tx re-fetch that enforces idempotency
	// under concurrent cascades (REQ-F-008 / AC-T2).
	GetByIDTx(ctx context.Context, tx *sql.Tx, id int64) (*models.Epic, error)
	UpdateStatusTx(ctx context.Context, tx *sql.Tx, id int64, status string, agent *string, notes *string) error
}

// EntityHistoryTxRecorder writes history rows inside a cascade-owned transaction.
type EntityHistoryTxRecorder interface {
	CreateTx(ctx context.Context, tx *sql.Tx, history *models.EntityHistory) error
}

// txBeginner is the narrow interface the cascade uses to open a transaction.
// Satisfied by *dbconn.DB in production and by test doubles in tests.
type txBeginner interface {
	BeginTxContext(ctx context.Context) (*sql.Tx, error)
}

// levelWorkflow is the narrow per-level workflow interface the cascade needs.
// Satisfied by *workflow.Service (after ForLevel() scoping).
type levelWorkflow interface {
	IsTerminalStatus(status string) bool
	GetTerminalStatuses() []string
	PrimaryAggregationStatus() (string, error)
	GetInitialStatusString() string
}

// levelWorkflowProvider is the provider interface that returns a scoped
// levelWorkflow for a given entity level string.
// NOTE: *workflow.Service.ForLevel returns a concrete *workflow.Service, not the
// levelWorkflow interface. Production wiring requires a thin adapter (e.g.
// workflowServiceAdapter) that bridges the concrete→interface gap — the same adapter
// used in the test file.  See T-003 for production wiring.
type levelWorkflowProvider interface {
	ForLevel(level string) levelWorkflow
}

// ============================================================
// cascadeLeg — typed start-leg discriminant
// ============================================================

type cascadeLeg string

const (
	// cascadeLegFeature means the cascade starts at the feature leg (task trigger).
	cascadeLegFeature cascadeLeg = "feature"
	// cascadeLegEpic means the cascade skips the feature leg and starts at the epic
	// (feature trigger — the feature itself is the triggering entity, not an ancestor).
	cascadeLegEpic cascadeLeg = "epic"
)

// ============================================================
// cascadeDeps — all runtime dependencies bundled for injection
// ============================================================

// cascadeDeps bundles all dependencies the cascade helper needs.
// Held by TaskService and FeatureService and passed into the helper at call time.
// All fields are required; a nil field causes the cascade to silently disable itself
// (see TaskService.cascadeEnabled / FeatureService.cascadeEnabled).
//
// commitFn is optional. When nil, tx.Commit() is called directly (production
// default). Providing a non-nil commitFn allows tests to intercept the commit
// path without requiring a real *sql.Tx.
type cascadeDeps struct {
	db             txBeginner
	featureRepo    CascadeFeatureRepo
	epicRepo       CascadeEpicRepo
	historyQuerier ParentReopenHistoryQuerier
	historyTx      EntityHistoryTxRecorder
	workflowSvc    levelWorkflowProvider
	// commitFn, if non-nil, is called instead of tx.Commit(). Production callers
	// leave this nil; tests may set it to intercept the commit path.
	commitFn func(tx *sql.Tx) error
}

// ============================================================
// cascadeTrigger — describes what fired the cascade
// ============================================================

// cascadeTrigger describes what fired the cascade so the audit row can
// reference the trigger entity correctly.
type cascadeTrigger struct {
	triggerKey  string            // e.g. "E07-F01-003"
	triggerKind string            // "regression" or "creation"
	triggerType models.EntityType // models.EntityTypeTask or models.EntityTypeFeature
	startLeg    cascadeLeg        // cascadeLegFeature or cascadeLegEpic
	featureID   int64             // used when startLeg == cascadeLegFeature: feature to reopen and walk up from
	epicID      int64             // used when startLeg == cascadeLegEpic && epicID != 0: bypasses feature lookup entirely
}

// ============================================================
// cascadeParentReopens — main cascade entry point
// ============================================================

// cascadeParentReopens walks the parent chain from trigger up to the epic,
// reopening any terminal ancestor inside a single owned transaction.
//
// CONTRACT:
//   - Best-effort: errors do NOT propagate to the caller. They are logged via
//     slog.Warn with structured fields including triggerKey, ancestor, and error.
//   - Atomic across the parent chain: on any error the cascade Tx rolls back and
//     neither parent is updated.
//   - Idempotent: ancestors already non-terminal are skipped (no update, no history
//     row), but the walk continues up the chain.
//   - Profile-agnostic: terminal classification uses workflow.Service.IsTerminalStatus
//     for the appropriate level. No status names are hardcoded.
func cascadeParentReopens(ctx context.Context, deps cascadeDeps, trigger cascadeTrigger) {
	featureWf := deps.workflowSvc.ForLevel("feature")
	epicWf := deps.workflowSvc.ForLevel("epic")

	// --------------------------------------------------------
	// Phase 1: pre-flight check — collect which ancestors need
	// reopening without opening a transaction yet. If neither
	// ancestor is terminal, return immediately (AC-05).
	// --------------------------------------------------------

	var feature *models.Feature
	var epic *models.Epic
	var err error

	var featureNeedsReopen bool

	if trigger.startLeg == cascadeLegEpic && trigger.epicID != 0 {
		// Epic-only path: caller already has the epic record and its ID.
		// Skip the feature lookup entirely — there is no feature to reopen on this leg.
		epic, err = deps.epicRepo.GetByID(ctx, trigger.epicID)
		if err != nil || epic == nil {
			slog.Warn("cascade: epic lookup failed",
				"trigger_key", trigger.triggerKey,
				"epic_id", trigger.epicID,
				"error", err,
			)
			return
		}
	} else {
		// Feature+epic path: look up the feature first to get its EpicID.
		feature, err = deps.featureRepo.GetByID(ctx, trigger.featureID)
		if err != nil || feature == nil {
			slog.Warn("cascade: feature lookup failed",
				"trigger_key", trigger.triggerKey,
				"feature_id", trigger.featureID,
				"error", err,
			)
			return
		}

		featureNeedsReopen = trigger.startLeg == cascadeLegFeature && featureWf.IsTerminalStatus(string(feature.Status))

		epic, err = deps.epicRepo.GetByID(ctx, feature.EpicID)
		if err != nil || epic == nil {
			slog.Warn("cascade: epic lookup failed",
				"trigger_key", trigger.triggerKey,
				"epic_id", feature.EpicID,
				"error", err,
			)
			return
		}
	}

	epicNeedsReopen := epicWf.IsTerminalStatus(string(epic.Status))

	// Full no-op check: nothing to do, don't even open a transaction.
	if !featureNeedsReopen && !epicNeedsReopen {
		return
	}

	// --------------------------------------------------------
	// Phase 2: resolve reopen targets for the ancestors that
	// need reopening (before opening the transaction).
	// --------------------------------------------------------

	var featureTarget, epicTarget string
	var featureFallback, epicFallback string

	if featureNeedsReopen {
		featureTarget, featureFallback, err = resolveReopenTarget(ctx, deps.historyQuerier,
			models.EntityTypeFeature, feature.ID, featureWf)
		if err != nil {
			slog.Warn("cascade: resolve target failed (feature)",
				"trigger_key", trigger.triggerKey,
				"error", err,
			)
			return
		}
	}

	if epicNeedsReopen {
		epicTarget, epicFallback, err = resolveReopenTarget(ctx, deps.historyQuerier,
			models.EntityTypeEpic, epic.ID, epicWf)
		if err != nil {
			slog.Warn("cascade: resolve target failed (epic)",
				"trigger_key", trigger.triggerKey,
				"error", err,
			)
			return
		}
	}

	// --------------------------------------------------------
	// Phase 3: open transaction and apply all updates atomically.
	// Re-fetch ancestor statuses inside the transaction to
	// implement idempotency (ADR-004 / REQ-F-008).
	// --------------------------------------------------------

	tx, err := deps.db.BeginTxContext(ctx)
	if err != nil {
		slog.Warn("cascade: failed to begin transaction",
			"trigger_key", trigger.triggerKey,
			"error", err,
		)
		return
	}
	if tx != nil {
		defer func() { _ = tx.Rollback() }()
	}

	now := time.Now()

	// --- Feature leg ---
	if featureNeedsReopen {
		// Re-fetch inside the cascade transaction for idempotency (AC-T2 / REQ-F-008).
		// GetByIDTx routes the read through tx, ensuring snapshot isolation under
		// concurrent cascades on the same ancestor.
		freshFeature, ferr := deps.featureRepo.GetByIDTx(ctx, tx, feature.ID)
		if ferr != nil || freshFeature == nil {
			slog.Warn("cascade: feature re-fetch failed inside tx",
				"trigger_key", trigger.triggerKey,
				"feature_key", feature.Key,
				"feature_id", feature.ID,
				"error", ferr,
			)
			return
		}
		// Still terminal after re-fetch?
		if featureWf.IsTerminalStatus(string(freshFeature.Status)) {
			notes := buildAutoReopenNotes(trigger, featureFallback)
			if uerr := deps.featureRepo.UpdateStatusTx(ctx, tx, feature.ID, featureTarget, nil, &notes); uerr != nil {
				slog.Warn("cascade: feature update failed",
					"trigger_key", trigger.triggerKey,
					"feature_key", feature.Key,
					"error", uerr,
				)
				return
			}
			changedBy := "system"
			// Use freshFeature.Status (in-tx read) so the audit row reflects the
			// value that was actually used in the idempotency decision (U-04).
			fromStatus := string(freshFeature.Status)
			if herr := deps.historyTx.CreateTx(ctx, tx, &models.EntityHistory{
				EntityType: models.EntityTypeFeature,
				EntityID:   feature.ID,
				FromStatus: &fromStatus,
				ToStatus:   featureTarget,
				ChangedBy:  &changedBy,
				Notes:      &notes,
				ChangedAt:  now,
			}); herr != nil {
				slog.Warn("cascade: feature history write failed",
					"trigger_key", trigger.triggerKey,
					"feature_key", feature.Key,
					"error", herr,
				)
				return
			}
		}
		// Else: already non-terminal after re-fetch — idempotent skip, continue to epic.
	}

	// --- Epic leg ---
	if epicNeedsReopen {
		// Re-fetch inside the cascade transaction for idempotency (AC-T2 / REQ-F-008).
		// GetByIDTx routes the read through tx, ensuring snapshot isolation under
		// concurrent cascades on the same ancestor.
		freshEpic, eerr := deps.epicRepo.GetByIDTx(ctx, tx, epic.ID)
		if eerr != nil || freshEpic == nil {
			slog.Warn("cascade: epic re-fetch failed inside tx",
				"trigger_key", trigger.triggerKey,
				"epic_key", epic.Key,
				"epic_id", epic.ID,
				"error", eerr,
			)
			return
		}
		// Still terminal after re-fetch?
		if epicWf.IsTerminalStatus(string(freshEpic.Status)) {
			notes := buildAutoReopenNotes(trigger, epicFallback)
			if uerr := deps.epicRepo.UpdateStatusTx(ctx, tx, epic.ID, epicTarget, nil, &notes); uerr != nil {
				slog.Warn("cascade: epic update failed",
					"trigger_key", trigger.triggerKey,
					"epic_key", epic.Key,
					"error", uerr,
				)
				return
			}
			changedBy := "system"
			// Use freshEpic.Status (in-tx read) so the audit row reflects the
			// value that was actually used in the idempotency decision (U-04).
			epicFromStatus := string(freshEpic.Status)
			if herr := deps.historyTx.CreateTx(ctx, tx, &models.EntityHistory{
				EntityType: models.EntityTypeEpic,
				EntityID:   epic.ID,
				FromStatus: &epicFromStatus,
				ToStatus:   epicTarget,
				ChangedBy:  &changedBy,
				Notes:      &notes,
				ChangedAt:  now,
			}); herr != nil {
				slog.Warn("cascade: epic history write failed",
					"trigger_key", trigger.triggerKey,
					"epic_key", epic.Key,
					"error", herr,
				)
				return
			}
		}
		// Else: already non-terminal after re-fetch — idempotent skip.
	}

	// Commit. If deps.commitFn is set, use it regardless of whether tx is nil
	// (this allows tests to exercise the commit path when the test txBeginner
	// returns nil). In production deps.commitFn is always nil and we call
	// tx.Commit() directly only when tx is non-nil.
	if deps.commitFn != nil {
		if cerr := deps.commitFn(tx); cerr != nil {
			slog.Warn("cascade: commit failed",
				"trigger_key", trigger.triggerKey,
				"error", cerr,
			)
			return
		}
	} else if tx != nil {
		if cerr := tx.Commit(); cerr != nil {
			slog.Warn("cascade: commit failed",
				"trigger_key", trigger.triggerKey,
				"error", cerr,
			)
			return
		}
	}
}

// ============================================================
// resolveReopenTarget — three-step fallback resolver
// ============================================================

// resolveReopenTarget implements the three-step fallback chain for choosing
// where to reopen an ancestor (REQ-F-004):
//
//  1. Most recent non-terminal status from entity_history.
//  2. First aggregation status from the workflow profile.
//  3. Initial status from the workflow profile.
//
// Returns (status, fallbackKind, error) where fallbackKind is:
//   - "" on a history hit (no fallback)
//   - "aggregation" when step 2 fires
//   - "initial" when step 3 fires
//
// Errors from GetLastNonTerminalStatus are returned to the caller; they must
// NOT be silently swallowed into a fallback.
func resolveReopenTarget(
	ctx context.Context,
	historyQuerier ParentReopenHistoryQuerier,
	entityType models.EntityType,
	entityID int64,
	levelWf levelWorkflow,
) (string, string, error) {
	// Step 1: look up the most recent non-terminal status from history.
	terminalSet := levelWf.GetTerminalStatuses()
	// Guard against empty terminal set (U-05 / REQ-F-005): a misconfigured custom
	// workflow profile could return an empty slice, which would make the NOT IN (...)
	// clause vacuous — matching every row including terminal ones. Skip history lookup
	// entirely and fall through to the aggregation/initial fallback chain.
	if len(terminalSet) > 0 {
		target, found, err := historyQuerier.GetLastNonTerminalStatus(ctx, entityType, entityID, terminalSet)
		if err != nil {
			return "", "", fmt.Errorf("history lookup failed for %s %d: %w", entityType, entityID, err)
		}
		if found {
			return target, "", nil
		}
	} else {
		slog.Warn("cascade: resolveReopenTarget called with empty terminal set; skipping history lookup",
			"entity_type", entityType,
			"entity_id", entityID,
		)
	}

	// Step 2: aggregation status fallback — the workflow's designated
	// aggregation step (primary: true breaks a multi-candidate tie).
	target, aggErr := levelWf.PrimaryAggregationStatus()
	if aggErr == nil {
		return target, "aggregation", nil
	}
	var noCandidate *config.NoCandidateError
	if !errors.As(aggErr, &noCandidate) {
		// An ambiguous designation must surface to the caller, never be
		// swallowed into an arbitrary pick.
		return "", "", aggErr
	}

	// Step 3: initial status fallback.
	return levelWf.GetInitialStatusString(), "initial", nil
}

// ============================================================
// buildAutoReopenNotes — structured notes prefix per REQ-F-007
// ============================================================

// buildAutoReopenNotes produces the structured notes string for an auto-reopen
// history row per REQ-F-007.
//
// Format:
//
//	auto_reopen: triggered by <triggerKey> <triggerKind> (<triggerType>)
//	auto_reopen: triggered by <triggerKey> <triggerKind> (<triggerType>) [fallback: <fallbackKind>]
func buildAutoReopenNotes(trigger cascadeTrigger, fallbackKind string) string {
	base := fmt.Sprintf("auto_reopen: triggered by %s %s (%s)",
		trigger.triggerKey, trigger.triggerKind, trigger.triggerType)
	if fallbackKind != "" {
		return base + " [fallback: " + fallbackKind + "]"
	}
	return base
}
