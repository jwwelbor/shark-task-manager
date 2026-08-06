// Package commands provides CLI command implementations.
// This file tests the worktree integration helpers in the run command.
package commands

import (
	"context"
	"errors"
	"go/ast"
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

// TestRunWorktreeCleanupWarningTargetsStderr is TC-013 (test-plan.md): the
// --worktree cleanup defer's warning call in runRun must be
// fmt.Fprintf(os.Stderr, ...), never a bare fmt.Printf (which would leak onto
// stdout, corrupting `shark run --json`'s output — research Finding 3 /
// Decision 3, AC-03/REQ-F-003). Source-guard (go/parser), not a runtime
// test: runRun has no mock seam (test-plan.md "Considered and rejected"),
// and T-E40-F04-006's TC-002 proves the whole-file invariant; this test
// additionally proves the *specific* writer target at this one call site —
// TC-002's whole-file guard alone would already catch a reversion to a bare
// fmt.Printf, but not confirm which writer a corrected Fprintf targets.
// parseRunGoSource is defined in run_test.go (same package).
func TestRunWorktreeCleanupWarningTargetsStderr(t *testing.T) {
	_, runRunDecl := parseRunGoSource(t)

	// Locate `if removeErr := creator.RemoveWorktree(...); removeErr != nil { ... }`.
	var cleanupIf *ast.IfStmt
	ast.Inspect(runRunDecl, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok || ifStmt.Init == nil {
			return true
		}
		assign, ok := ifStmt.Init.(*ast.AssignStmt)
		if !ok || len(assign.Rhs) != 1 {
			return true
		}
		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if ok && sel.Sel.Name == "RemoveWorktree" {
			cleanupIf = ifStmt
		}
		return true
	})
	if cleanupIf == nil {
		t.Fatal("could not find `if removeErr := ...RemoveWorktree(...); removeErr != nil { ... }` inside runRun")
	}

	// Within that if-block, find the fmt.Printf/Fprintf warning call.
	var warnCall *ast.CallExpr
	var warnSel *ast.SelectorExpr
	ast.Inspect(cleanupIf.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "fmt" {
			return true
		}
		if sel.Sel.Name == "Printf" || sel.Sel.Name == "Fprintf" {
			warnCall, warnSel = call, sel
		}
		return true
	})
	if warnCall == nil {
		t.Fatal("worktree cleanup defer's if-block contains no fmt.Printf/Fprintf call")
	}

	if warnSel.Sel.Name != "Fprintf" {
		t.Fatalf("worktree cleanup warning call is fmt.%s(...), want fmt.Fprintf(os.Stderr, ...)", warnSel.Sel.Name)
	}
	if len(warnCall.Args) == 0 {
		t.Fatal("fmt.Fprintf call has no arguments")
	}
	firstArg, ok := warnCall.Args[0].(*ast.SelectorExpr)
	if !ok {
		t.Fatalf("fmt.Fprintf's first argument is %T, want os.Stderr", warnCall.Args[0])
	}
	pkgIdent, ok := firstArg.X.(*ast.Ident)
	if !ok || pkgIdent.Name != "os" || firstArg.Sel.Name != "Stderr" {
		t.Fatal("fmt.Fprintf's first argument is not os.Stderr")
	}
}
