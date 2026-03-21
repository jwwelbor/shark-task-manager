package runner

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// ─── MockWorktreeCreator ──────────────────────────────────────────────────────

// MockWorktreeCreator is a test double for WorktreeCreator that records calls
// and lets tests control return values without executing real git commands.
type MockWorktreeCreator struct {
	CreateFunc func(ctx context.Context, repoRoot, path, branch string) error
	RemoveFunc func(ctx context.Context, path string) error

	// Recorded calls for assertion.
	CreateCalls []createCall
	RemoveCalls []string
}

type createCall struct {
	RepoRoot string
	Path     string
	Branch   string
}

func (m *MockWorktreeCreator) CreateWorktree(ctx context.Context, repoRoot, path, branch string) error {
	m.CreateCalls = append(m.CreateCalls, createCall{RepoRoot: repoRoot, Path: path, Branch: branch})
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, repoRoot, path, branch)
	}
	return nil
}

func (m *MockWorktreeCreator) RemoveWorktree(ctx context.Context, path string) error {
	m.RemoveCalls = append(m.RemoveCalls, path)
	if m.RemoveFunc != nil {
		return m.RemoveFunc(ctx, path)
	}
	return nil
}

// ─── sanitizeKeyForPath ───────────────────────────────────────────────────────

func TestSanitizeKeyForPath_TaskKey(t *testing.T) {
	got := sanitizeKeyForPath("E22-F06-001")
	if got != "E22-F06-001" {
		t.Errorf("sanitizeKeyForPath(%q) = %q, want %q", "E22-F06-001", got, "E22-F06-001")
	}
}

func TestSanitizeKeyForPath_SpecialChars(t *testing.T) {
	got := sanitizeKeyForPath("E22/F06 001")
	// Slashes and spaces should become hyphens; consecutive hyphens collapsed.
	if strings.Contains(got, "/") || strings.Contains(got, " ") {
		t.Errorf("sanitizeKeyForPath should remove special chars, got %q", got)
	}
}

func TestSanitizeKeyForPath_ConsecutiveHyphens(t *testing.T) {
	got := sanitizeKeyForPath("E22--F06---001")
	if strings.Contains(got, "--") {
		t.Errorf("sanitizeKeyForPath should collapse consecutive hyphens, got %q", got)
	}
}

func TestSanitizeKeyForPath_EmptyString(t *testing.T) {
	got := sanitizeKeyForPath("")
	// Should not panic; returns empty or safe string.
	_ = got
}

// ─── WorktreePaths ───────────────────────────────────────────────────────────

func TestWorktreePaths_Structure(t *testing.T) {
	now := time.Unix(1742500000, 0)
	path, branch := WorktreePaths(".shark-run-worktrees", "E22-F06-001", now)

	ts := "1742500000"
	expectedPathSuffix := "E22-F06-001-" + ts
	if !strings.HasSuffix(path, expectedPathSuffix) {
		t.Errorf("WorktreePaths path %q does not end with %q", path, expectedPathSuffix)
	}

	if !strings.HasPrefix(branch, "shark-run-") {
		t.Errorf("WorktreePaths branch %q does not start with 'shark-run-'", branch)
	}

	if !strings.Contains(branch, ts) {
		t.Errorf("WorktreePaths branch %q does not contain timestamp %q", branch, ts)
	}
}

func TestWorktreePaths_BasePathIncluded(t *testing.T) {
	now := time.Unix(1000, 0)
	path, _ := WorktreePaths(".shark-run-worktrees", "E01-F01", now)
	if !strings.HasPrefix(path, ".shark-run-worktrees/") {
		t.Errorf("worktree path %q should start with base dir", path)
	}
}

func TestWorktreePaths_UniquenessAcrossTimes(t *testing.T) {
	path1, branch1 := WorktreePaths(".shark-run-worktrees", "E22-F06-001", time.Unix(1000, 0))
	path2, branch2 := WorktreePaths(".shark-run-worktrees", "E22-F06-001", time.Unix(2000, 0))

	if path1 == path2 {
		t.Error("WorktreePaths should produce unique paths for different times")
	}
	if branch1 == branch2 {
		t.Error("WorktreePaths should produce unique branches for different times")
	}
}

// ─── WorktreeCreator interface (mock-based) ───────────────────────────────────

func TestMockWorktreeCreator_CreateRecordsCall(t *testing.T) {
	ctx := context.Background()
	mock := &MockWorktreeCreator{}

	err := mock.CreateWorktree(ctx, "/repo", "/path/to/wt", "shark-run-E22-001")
	if err != nil {
		t.Fatalf("CreateWorktree() unexpected error: %v", err)
	}

	if len(mock.CreateCalls) != 1 {
		t.Fatalf("expected 1 CreateWorktree call, got %d", len(mock.CreateCalls))
	}
	call := mock.CreateCalls[0]
	if call.RepoRoot != "/repo" {
		t.Errorf("RepoRoot = %q, want %q", call.RepoRoot, "/repo")
	}
	if call.Path != "/path/to/wt" {
		t.Errorf("Path = %q, want %q", call.Path, "/path/to/wt")
	}
	if call.Branch != "shark-run-E22-001" {
		t.Errorf("Branch = %q, want %q", call.Branch, "shark-run-E22-001")
	}
}

func TestMockWorktreeCreator_RemoveRecordsCall(t *testing.T) {
	ctx := context.Background()
	mock := &MockWorktreeCreator{}

	err := mock.RemoveWorktree(ctx, "/path/to/wt")
	if err != nil {
		t.Fatalf("RemoveWorktree() unexpected error: %v", err)
	}

	if len(mock.RemoveCalls) != 1 {
		t.Fatalf("expected 1 RemoveWorktree call, got %d", len(mock.RemoveCalls))
	}
	if mock.RemoveCalls[0] != "/path/to/wt" {
		t.Errorf("RemoveCalls[0] = %q, want %q", mock.RemoveCalls[0], "/path/to/wt")
	}
}

func TestMockWorktreeCreator_CreateReturnsError(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("git error: not a git repo")
	mock := &MockWorktreeCreator{
		CreateFunc: func(_ context.Context, _, _, _ string) error {
			return wantErr
		},
	}

	err := mock.CreateWorktree(ctx, "", "/path", "branch")
	if !errors.Is(err, wantErr) {
		t.Errorf("CreateWorktree() error = %v, want %v", err, wantErr)
	}
}

func TestMockWorktreeCreator_RemoveReturnsError(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("worktree not found")
	mock := &MockWorktreeCreator{
		RemoveFunc: func(_ context.Context, _ string) error {
			return wantErr
		},
	}

	err := mock.RemoveWorktree(ctx, "/path")
	if !errors.Is(err, wantErr) {
		t.Errorf("RemoveWorktree() error = %v, want %v", err, wantErr)
	}
}

// ─── DefaultWorktreeBaseDir constant ─────────────────────────────────────────

func TestDefaultWorktreeBaseDir_Value(t *testing.T) {
	if DefaultWorktreeBaseDir != ".shark-run-worktrees" {
		t.Errorf("DefaultWorktreeBaseDir = %q, want %q", DefaultWorktreeBaseDir, ".shark-run-worktrees")
	}
}
