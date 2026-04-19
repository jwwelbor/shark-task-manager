// Package commands provides CLI command implementations.
// This file implements the slog-based run execution logging helpers for
// T-E07-F41-002: run_id generation and run.start/run.end event emission.
package commands

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
)

// generateRunID returns a new UUID string to be used as the correlation
// identifier for a single /run invocation (AC-T1).
func generateRunID() string {
	return uuid.New().String()
}

// runStartParams bundles the fields emitted in the run.start slog event.
// Corresponds to AC-T2 / REQ-F-001.
type runStartParams struct {
	// Args is the raw command-line arguments slice passed to runRun.
	Args []string

	// EntityKey is the normalized entity key being run (e.g. "E07-F01-001").
	EntityKey string

	// EntityType is the detected type string (e.g. "task", "feature", "epic").
	EntityType string

	// DryRun mirrors the --dry-run flag value.
	DryRun bool

	// Worktree mirrors the --worktree flag value.
	Worktree bool

	// WorktreePath is the path of the created worktree (empty when --worktree=false).
	WorktreePath string

	// RunID is the correlation identifier for this invocation.
	RunID string
}

// emitRunStart emits the run.start slog event at INFO level.
// No-ops when obs.Enabled is false (AC-T5).
func emitRunStart(obs config.ObservabilityConfig, p runStartParams) {
	if !obs.Enabled {
		return
	}
	slog.Info("run.start",
		"command", "run",
		"args", p.Args,
		"entity_key", p.EntityKey,
		"entity_type", p.EntityType,
		"dry_run", p.DryRun,
		"worktree", p.Worktree,
		"worktree_path", p.WorktreePath,
		"run_id", p.RunID,
	)
}

// emitRunEnd emits the run.end slog event.
// Level is INFO when result.Error is empty, ERROR otherwise (AC-T4).
// durationMS is the wall-clock duration of the run in milliseconds.
// No-ops when obs.Enabled is false (AC-T5).
func emitRunEnd(ctx context.Context, obs config.ObservabilityConfig, runID string, result *runner.RunResult, durationMS int64) {
	if !obs.Enabled {
		return
	}

	attrs := []any{
		"entity_key", result.EntityKey,
		"outcome", result.Outcome,
		"final_status", result.FinalStatus,
		"stages_completed", result.StagesCompleted,
		"duration_ms", durationMS,
		"run_id", runID,
	}
	if result.Error != "" {
		attrs = append(attrs, "error", result.Error)
		slog.ErrorContext(ctx, "run.end", attrs...)
		return
	}
	slog.InfoContext(ctx, "run.end", attrs...)
}
