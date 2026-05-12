// Package commands — cascade.go implements engine-internal cascade resolution
// for `shark next`. When an entity's status maps to action "cascade", the
// engine looks one level down, picks the first dispatchable child, and
// recurses on the child's key. Cascade never reaches the harness on the
// wire — the harness only ever sees `spawn_agent`, `pause`, or `archive`.
package commands

import (
	"context"
	"fmt"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// maxCascadeDepth bounds how deep cascade recursion can go. The hierarchy
// is Epic → Feature → Task (depth 2), so a small bound is sufficient.
// Anything deeper indicates a misconfigured workflow YAML and should fail
// loudly rather than spinning.
const maxCascadeDepth = 4

// cascadeChild is a (key, entityType) pair the resolver hands back to
// runNext for recursion. The pair is enough because runNext re-detects the
// entity type from the key shape anyway.
type cascadeChild struct {
	Key        string
	EntityType string
}

// listDispatchableChildren returns the ordered list of child entities to
// consider for cascade dispatch from parent (entityType, key).
//
// Ordering: the underlying repository queries already sort by
// (execution_order NULLS LAST, priority ASC, created_at ASC, key ASC).
// We pass that order through untouched so the caller picks "first
// in-progress task; else first todo by order" simply by iterating.
//
// Terminal-status children (completed, cancelled, archived) are filtered
// out — they can't be dispatched and would only waste a recursion. The
// caller is still free to receive a "pause" response from the recursion
// for non-terminal-but-stuck children (e.g. blocked).
func listDispatchableChildren(ctx context.Context, entityType, key string) ([]cascadeChild, error) {
	db, err := cli.GetDB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	wfSvc := cli.GetWorkflowService()

	switch entityType {
	case "feature":
		taskRepo := repository.NewTaskRepository(db)
		tasks, err := taskRepo.ListByFeatureKey(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks for feature %s: %w", key, err)
		}
		taskWf := wfSvc.ForLevel(workflow.LevelTask)
		out := make([]cascadeChild, 0, len(tasks))
		for _, t := range tasks {
			if isTerminalStatus(taskWf, string(t.Status)) {
				continue
			}
			out = append(out, cascadeChild{Key: t.Key, EntityType: "task"})
		}
		return out, nil

	case "epic":
		// Resolve the epic ID once, then list its features.
		epicRepo := repository.NewEpicRepository(db)
		featureRepo := repository.NewFeatureRepository(db)

		epic, err := epicRepo.GetByKey(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("failed to get epic %s: %w", key, err)
		}
		features, err := featureRepo.ListByEpic(ctx, epic.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list features for epic %s: %w", key, err)
		}
		featureWf := wfSvc.ForLevel(workflow.LevelFeature)
		out := make([]cascadeChild, 0, len(features))
		for _, f := range features {
			if isTerminalStatus(featureWf, string(f.Status)) {
				continue
			}
			out = append(out, cascadeChild{Key: f.Key, EntityType: "feature"})
		}
		return out, nil
	}

	// Tasks, bugs, change-cards, tech-debt are leaf entities — they have
	// no children. Returning an empty slice (not an error) lets the caller
	// fall through to "no dispatchable child → pause".
	return nil, nil
}

// isTerminalStatus reports whether a status is terminal (no productive
// dispatch possible) for the given workflow level. Delegates to
// workflow.Service.IsTerminalStatus, which reads the configured terminal
// set from the per-level workflow YAML (special_statuses._complete_).
//
// Using the workflow service rather than a hardcoded literal list keeps
// cascade correctness aligned with custom workflows that rename terminal
// statuses (e.g. "shipped" instead of "completed"). See B028.
func isTerminalStatus(wf *workflow.Service, s string) bool {
	if wf == nil {
		return false
	}
	return wf.IsTerminalStatus(s)
}
