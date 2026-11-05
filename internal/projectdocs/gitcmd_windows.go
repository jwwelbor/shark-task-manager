//go:build windows

package projectdocs

import (
	"context"
	"os/exec"
	"time"
)

// WaitDelay bounds pipe draining if a Git child inherits its output handles.
func gitCommand(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.WaitDelay = 500 * time.Millisecond
	return cmd
}
