// Package commands provides CLI command implementations.
//
// This file adapts the shared `shark next`/`shark run` epic-cascade
// pre-dispatch integration-evidence guard (ensureEpicIntegrationBaseCaptured,
// next.go) to runner.CascadeIntegrationGuard, so `shark run`'s
// RunController.handleCascade enforces the identical REQ-F-004 precondition
// that entityResolutionStrategy.resolveCascade enforces for `shark next`
// (UAT round-3 rejection Finding 1; docs/plan/tech-debt/TD-208.md).
//
// Both dispatch surfaces call ensureEpicIntegrationBaseCaptured directly —
// this file contributes no independent capture-or-block logic of its own,
// only the entity-type scoping and the runner.CascadeIntegrationGuard
// adaptation — so there is exactly one capture-or-block implementation
// shared by both entrypoints, not two independently-maintained copies.
package commands

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// cascadeIntegrationGuard implements runner.CascadeIntegrationGuard for
// `shark run`'s cascade path. commandLabel identifies the dispatch surface
// ("run") for the shared stderr-warning/pause-error prefixes
// ensureEpicIntegrationBaseCaptured produces.
type cascadeIntegrationGuard struct {
	commandLabel string
}

// EnsureBaseCaptured is a no-op for anything but an epic-level cascade,
// mirroring entityResolutionStrategy.resolveCascade's identical scoping: a
// nested feature-level cascade (e.g. a feature's own `active` step cascading
// into tasks) must never re-fire the epic integration-base capture for the
// wrong entity.
func (g cascadeIntegrationGuard) EnsureBaseCaptured(ctx context.Context, entityType, key string) error {
	if entityType != string(models.EntityTypeEpic) {
		return nil
	}
	return ensureEpicIntegrationBaseCaptured(ctx, g.commandLabel, key)
}
