// Package commands — cascade.go is the thin CLI wrapper over
// services.CascadeService. The service owns repository access and
// terminal-status filtering; this file only adapts the call.
package commands

import (
	"context"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// maxCascadeDepth bounds how deep cascade recursion can go. The hierarchy
// is Epic → Feature → Task (depth 2), so a small bound is sufficient.
// Anything deeper indicates a misconfigured workflow YAML and should fail
// loudly rather than spinning.
const maxCascadeDepth = 4

// listDispatchableChildren returns the ordered list of child entities to
// consider for cascade dispatch from parent (entityType, key). Thin wrapper
// around services.CascadeService.
func listDispatchableChildren(ctx context.Context, entityType, key string) ([]services.CascadeChild, error) {
	return cli.GetCascadeService().ListDispatchableChildren(ctx, entityType, key)
}
