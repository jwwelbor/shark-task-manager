// Package runner provides the AgentDispatcher interface and related types for
// invoking external AI agents (Claude, Codex, etc.) as part of the E22 run loop.
// This file implements git worktree creation and cleanup for agent isolation.
package runner

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// WorktreeCreator abstracts git worktree creation and removal so that the run
// command can be tested without executing real git operations.
//
// The interface is defined at point of use (consumer side) per Go conventions.
type WorktreeCreator interface {
	// CreateWorktree creates a new git worktree at the given path on a new branch.
	// Returns the absolute path to the created worktree.
	CreateWorktree(ctx context.Context, repoRoot, path, branch string) error

	// RemoveWorktree removes the git worktree at the given path.
	// Uses --force to handle unclean states.
	RemoveWorktree(ctx context.Context, path string) error
}

// GitWorktreeCreator is the production implementation of WorktreeCreator that
// executes real git commands via os/exec.
type GitWorktreeCreator struct {
	// cmdFactory creates an *exec.Cmd. Defaults to exec.CommandContext.
	// Tests may replace this to capture commands without execution.
	cmdFactory func(ctx context.Context, name string, args ...string) *exec.Cmd
}

// NewGitWorktreeCreator creates a GitWorktreeCreator using real git commands.
func NewGitWorktreeCreator() *GitWorktreeCreator {
	return &GitWorktreeCreator{
		cmdFactory: exec.CommandContext,
	}
}

// CreateWorktree runs: git worktree add <path> -b <branch>
func (g *GitWorktreeCreator) CreateWorktree(ctx context.Context, repoRoot, path, branch string) error {
	factory := g.cmdFactory
	if factory == nil {
		factory = exec.CommandContext
	}

	cmd := factory(ctx, "git", "worktree", "add", path, "-b", branch)
	if repoRoot != "" {
		cmd.Dir = repoRoot
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree add failed: %w\noutput: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// RemoveWorktree runs: git worktree remove --force <path>
func (g *GitWorktreeCreator) RemoveWorktree(ctx context.Context, path string) error {
	factory := g.cmdFactory
	if factory == nil {
		factory = exec.CommandContext
	}

	cmd := factory(ctx, "git", "worktree", "remove", "--force", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove failed: %w\noutput: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Precompiled regexes for sanitizeKeyForPath, avoiding per-call compilation.
var (
	invalidPathCharsRe   = regexp.MustCompile(`[^a-zA-Z0-9-]+`)
	consecutiveHyphensRe = regexp.MustCompile(`-{2,}`)
)

// sanitizeKeyForPath replaces characters that are unsafe in file and branch
// names with hyphens. Consecutive hyphens are collapsed to one.
func sanitizeKeyForPath(key string) string {
	s := invalidPathCharsRe.ReplaceAllString(key, "-")
	s = consecutiveHyphensRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// WorktreePaths computes the worktree directory path and git branch name for
// the given entity key and current time. The caller should use these values
// when invoking WorktreeCreator.CreateWorktree.
//
//   - basePath: the parent directory for worktrees (e.g. ".shark-run-worktrees")
//   - entityKey: the entity identifier (e.g. "E22-F06-001")
//   - now: the reference time (pass time.Now() in production; fixed value in tests)
//
// Returns (worktreePath, branchName).
func WorktreePaths(basePath, entityKey string, now time.Time) (string, string) {
	safe := sanitizeKeyForPath(entityKey)
	ts := fmt.Sprintf("%d", now.Unix())
	suffix := safe + "-" + ts
	path := basePath + "/" + suffix
	branch := "shark-run-" + suffix
	return path, branch
}

// DefaultWorktreeBaseDir is the directory (relative to project root) where
// shark creates agent worktrees when --worktree is set.
const DefaultWorktreeBaseDir = ".shark-run-worktrees"
