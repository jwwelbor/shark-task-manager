// Package commands provides CLI command implementations.
// This file tests the worktree integration helpers in the run command.
package commands

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/runner"
)

// ─── mockWorktreeCreator ──────────────────────────────────────────────────────

// mockWorktreeCreator is a test double for runner.WorktreeCreator.
type mockWorktreeCreator struct {
	createFunc func(ctx context.Context, repoRoot, path, branch string) error
	removeFunc func(ctx context.Context, path string) error

	createCalls []struct{ repoRoot, path, branch string }
	removeCalls []string
}

func (m *mockWorktreeCreator) CreateWorktree(ctx context.Context, repoRoot, path, branch string) error {
	m.createCalls = append(m.createCalls, struct{ repoRoot, path, branch string }{repoRoot, path, branch})
	if m.createFunc != nil {
		return m.createFunc(ctx, repoRoot, path, branch)
	}
	return nil
}

func (m *mockWorktreeCreator) RemoveWorktree(ctx context.Context, path string) error {
	m.removeCalls = append(m.removeCalls, path)
	if m.removeFunc != nil {
		return m.removeFunc(ctx, path)
	}
	return nil
}

// Compile-time check: mockWorktreeCreator satisfies runner.WorktreeCreator.
var _ runner.WorktreeCreator = (*mockWorktreeCreator)(nil)

// ─── setupWorktree ────────────────────────────────────────────────────────────

// TestSetupWorktree_Success verifies that setupWorktree calls CreateWorktree
// and returns the worktree path as the working directory.
func TestSetupWorktree_Success(t *testing.T) {
	ctx := context.Background()
	mock := &mockWorktreeCreator{}

	workingDir, worktreePath, err := setupWorktree(ctx, "E22-F06-001", mock)
	if err != nil {
		t.Fatalf("setupWorktree() unexpected error: %v", err)
	}

	// workingDir and worktreePath must be the same path.
	if workingDir != worktreePath {
		t.Errorf("workingDir %q != worktreePath %q", workingDir, worktreePath)
	}

	// Must be non-empty.
	if workingDir == "" {
		t.Error("setupWorktree() returned empty workingDir")
	}

	// CreateWorktree must have been called exactly once.
	if len(mock.createCalls) != 1 {
		t.Fatalf("expected 1 CreateWorktree call, got %d", len(mock.createCalls))
	}

	call := mock.createCalls[0]

	// Path must contain the entity key.
	if !strings.Contains(call.path, "E22-F06-001") {
		t.Errorf("worktree path %q should contain entity key %q", call.path, "E22-F06-001")
	}

	// Branch must start with "shark-run-".
	if !strings.HasPrefix(call.branch, "shark-run-") {
		t.Errorf("branch %q should start with 'shark-run-'", call.branch)
	}
}

// TestSetupWorktree_CreateError verifies that setupWorktree propagates errors
// from CreateWorktree and does not return a working directory.
func TestSetupWorktree_CreateError(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("git: not a repository")
	mock := &mockWorktreeCreator{
		createFunc: func(_ context.Context, _, _, _ string) error {
			return wantErr
		},
	}

	workingDir, worktreePath, err := setupWorktree(ctx, "E22-F06-001", mock)
	if err == nil {
		t.Fatal("setupWorktree() expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("setupWorktree() error = %v, want to wrap %v", err, wantErr)
	}
	if workingDir != "" {
		t.Errorf("workingDir should be empty on error, got %q", workingDir)
	}
	if worktreePath != "" {
		t.Errorf("worktreePath should be empty on error, got %q", worktreePath)
	}
}

// TestSetupWorktree_PathContainsTimestamp verifies that the generated worktree
// path contains a numeric timestamp (not just the entity key).
func TestSetupWorktree_PathContainsTimestamp(t *testing.T) {
	ctx := context.Background()
	mock := &mockWorktreeCreator{}

	_, _, err := setupWorktree(ctx, "E22-F06-001", mock)
	if err != nil {
		t.Fatalf("setupWorktree() unexpected error: %v", err)
	}

	if len(mock.createCalls) == 0 {
		t.Fatal("no CreateWorktree calls recorded")
	}

	path := mock.createCalls[0].path
	// Path should be longer than just the base dir + entity key, due to timestamp suffix.
	minExpectedLen := len(runner.DefaultWorktreeBaseDir) + len("/E22-F06-001-") + 1
	if len(path) < minExpectedLen {
		t.Errorf("worktree path %q appears to be missing timestamp suffix (len=%d, want>%d)",
			path, len(path), minExpectedLen)
	}
}
