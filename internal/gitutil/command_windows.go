//go:build windows

package gitutil

import (
	"context"
	"os/exec"
	"time"
)

// Command bounds pipe draining if a Git child inherits its output handles.
func Command(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.WaitDelay = 500 * time.Millisecond
	return cmd
}
