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

// describeDispatchableChildren returns the ordered dispatchable children plus
// summary counts for the parent. Thin wrapper around services.CascadeService.
func describeDispatchableChildren(ctx context.Context, entityType, key string) (services.CascadeChildrenState, error) {
	return cli.GetCascadeService().DescribeDispatchableChildren(ctx, entityType, key)
}

// nextDescribeDispatchableChildren is a test seam for resolveNext/tryCascade.
var nextDescribeDispatchableChildren = describeDispatchableChildren
