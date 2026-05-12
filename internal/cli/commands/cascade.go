// Package commands — cascade.go implements engine-internal cascade resolution
// for `shark next`. When an entity's status maps to action "cascade", the
// engine looks one level down, picks the first dispatchable child, and
// recurses on the child's key. Cascade never reaches the harness on the
// wire — the harness only ever sees `spawn_agent`, `pause`, or `archive`.
//
// This file is intentionally a thin CLI wrapper: it adapts the
// command-layer types (`cascadeChild`) to and from the service layer
// (`services.CascadeService`) and delegates all repository access and
// terminal-status business logic to the service. See B029 for the
// architectural rationale (fat-controller fix).
package commands

import (
	"context"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// maxCascadeDepth bounds how deep cascade recursion can go. The hierarchy
// is Epic → Feature → Task (depth 2), so a small bound is sufficient.
// Anything deeper indicates a misconfigured workflow YAML and should fail
// loudly rather than spinning.
const maxCascadeDepth = 4

// cascadeChild is a (key, entityType) pair the resolver hands back to
// runNext for recursion. Kept as a command-layer type so callers continue to
// import only from `commands`. The service returns a structurally identical
// `services.CascadeChild` which we map into this type below.
type cascadeChild struct {
	Key        string
	EntityType string
}

// listDispatchableChildren returns the ordered list of child entities to
// consider for cascade dispatch from parent (entityType, key).
//
// This function is now a thin wrapper around services.CascadeService —
// repository construction and terminal-status filtering happen there.
func listDispatchableChildren(ctx context.Context, entityType, key string) ([]cascadeChild, error) {
	svc := cli.GetCascadeService()
	children, err := svc.ListDispatchableChildren(ctx, entityType, key)
	if err != nil {
		return nil, err
	}
	if len(children) == 0 {
		return nil, nil
	}
	out := make([]cascadeChild, len(children))
	for i, c := range children {
		out[i] = cascadeChild{Key: c.Key, EntityType: c.EntityType}
	}
	return out, nil
}

// isTerminalStatus reports whether a status is terminal for the given
// workflow level. Retained as a package-level helper for the
// cascade_terminal_status_test.go regression test (B028) which exercises
// the workflow-level delegation contract directly.
//
// New code should not call this helper; use services.CascadeService instead.
func isTerminalStatus(wf *workflow.Service, s string) bool {
	if wf == nil {
		return false
	}
	return wf.IsTerminalStatus(s)
}
