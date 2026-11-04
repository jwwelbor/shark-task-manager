package runner

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ReviewSourceInventory computes the same confined source-change digest used
// by bug gate admission. It does not alter the source index or working tree.
func ReviewSourceInventory(ctx context.Context, root string) (string, error) {
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("review inventory requires an absolute project root")
	}
	run := func(args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("review inventory git %s: %w", args[0], err)
		}
		return output, nil
	}
	top, err := run("rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	if err := sameBugProject(root, strings.TrimSpace(string(top))); err != nil {
		return "", fmt.Errorf("review inventory project root: %w", err)
	}
	return bugSourceInventory(root, run)
}
