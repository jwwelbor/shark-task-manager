// Package commands — cascade.go implements engine-internal cascade resolution
// for `shark next`. When an entity (typically a feature or epic) is in a
// status whose orchestrator action is "cascade", the engine looks one level
// down, picks the first dispatchable child, and recurses runNext on the
// child's key. The wire response carries the child's dispatch step plus a
// `resolved_via` chain that audits the parents skipped through.
//
// Per the 2026-05-11 design decision, cascade never reaches the harness on
// the wire — the harness only ever sees `spawn_agent`, `pause`, or
// `archive`. This keeps the contract narrow and avoids teaching the
// harness about parent/child entity topology.
package commands

import (
	"context"
	"fmt"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
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

	switch entityType {
	case "feature":
		taskRepo := repository.NewTaskRepository(db)
		tasks, err := taskRepo.ListByFeatureKey(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks for feature %s: %w", key, err)
		}
		out := make([]cascadeChild, 0, len(tasks))
		for _, t := range tasks {
			if isTerminalTaskStatus(t.Status) {
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
		out := make([]cascadeChild, 0, len(features))
		for _, f := range features {
			if isTerminalFeatureStatus(f.Status) {
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

// isTerminalTaskStatus reports whether a task status is in a terminal state
// from which no dispatch will ever be productive. The exact terminal set is
// workflow-defined; we hardcode the names that ship with the default short
// workflow plus "archived"/"done" for compatibility with custom workflows.
//
// This is intentionally a static list: cascade resolution runs before the
// per-entity action service is consulted for the child, and we don't want
// to pay the action-service round-trip just to filter children. If a
// custom workflow renames "completed" to something exotic, the recursion
// will still bottom out at "pause" — just one wasted call.
func isTerminalTaskStatus(s models.TaskStatus) bool {
	switch string(s) {
	case "completed", "cancelled", "archived", "done":
		return true
	}
	return false
}

// isTerminalFeatureStatus mirrors isTerminalTaskStatus for features.
func isTerminalFeatureStatus(s models.FeatureStatus) bool {
	switch string(s) {
	case "completed", "cancelled", "archived", "done":
		return true
	}
	return false
}
