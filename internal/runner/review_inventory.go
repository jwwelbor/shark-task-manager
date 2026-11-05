package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gitutil"
)

// ReviewSourceInventory computes the same confined source-change digest used
// by bug gate admission. It does not alter the source index or working tree.
func ReviewSourceInventory(ctx context.Context, root string) (string, error) {
	return ReviewSourceInventoryForProject(ctx, root, ".")
}

// ReviewSourceInventoryForProject hashes source changes while excluding only
// untracked runtime files belonging to the designated project.
func ReviewSourceInventoryForProject(ctx context.Context, root, runtimeProject string) (string, error) {
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("review inventory requires an absolute project root")
	}
	run := func(args ...string) ([]byte, error) {
		cmd := gitutil.Command(ctx, append([]string{"-c", "core.fsmonitor=false", "-C", root}, args...)...)
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
	return bugSourceInventory(root, runtimeProject, run)
}

// ReviewReadArtifact safely reads one bounded regular file without following
// a final symlink. It is also used by hosts lacking Python's no-follow flags.
func ReviewReadArtifact(path string, limit int64) ([]byte, error) {
	if !filepath.IsAbs(path) || limit <= 0 || limit > 128*1024*1024 {
		return nil, fmt.Errorf("invalid review artifact path or size limit")
	}
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }() // Read-only directory handle.
	name := filepath.Base(path)
	info, err := root.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("review artifact must be a bounded regular file")
	}
	file, err := bugOpenVerifiedInventoryFile(root, name, info)
	if err != nil {
		return nil, err
	}
	return readBoundedReviewFile(file, limit)
}

func readBoundedReviewFile(file *os.File, limit int64) ([]byte, error) {
	data, readErr := io.ReadAll(io.LimitReader(file, limit+1))
	closeErr := file.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("review artifact exceeds size limit")
	}
	return data, nil
}
