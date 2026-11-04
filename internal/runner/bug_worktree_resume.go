package runner

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// RetainedBugWorktree finds the checkout previously created by shark run for
// this bug. Ambiguity is surfaced rather than choosing a potentially stale plan.
func RetainedBugWorktree(ctx context.Context, projectRoot, key string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-c", "core.fsmonitor=false", "-C", projectRoot, "worktree", "list", "--porcelain", "-z")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("list retained bug worktrees: %w", err)
	}
	prefix := "branch refs/heads/shark-run-" + sanitizeKeyForPath(key) + "-"
	var path string
	var matches []string
	for _, field := range strings.Split(string(out), "\x00") {
		if strings.HasPrefix(field, "worktree ") {
			path = strings.TrimPrefix(field, "worktree ")
		} else if strings.HasPrefix(field, prefix) && path != "" {
			matches = append(matches, path)
		}
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple retained worktrees for %s: %s; select one with --workdir", key, strings.Join(matches, ", "))
	}
	if len(matches) == 0 {
		return "", nil
	}
	return matches[0], nil
}
