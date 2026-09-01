// Package commands provides CLI command implementations.
// This file contains tests for the `shark run` command helpers:
// - buildTransitioner: entity type dispatch (no DB for error path)
// - buildPlaceholderGenerator: entity type dispatch (no DB for nil path)
// - GetActionService caching (sync.Once behavior)
package commands

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

type mockRunClaimService struct {
	ttl        time.Duration
	claims     []services.ClaimInput
	releases   []runReleaseCall
	heartbeats []runHeartbeatCall
}

type mockRunCascadeChildrenService struct {
	state services.CascadeChildrenState
}

func (m *mockRunCascadeChildrenService) DescribeDispatchableChildren(context.Context, string, string) (services.CascadeChildrenState, error) {
	return m.state, nil
}

type runReleaseCall struct {
	entityType string
	entityKey  string
	sessionID  string
	outcome    string
	force      bool
}

type runHeartbeatCall struct {
	entityType string
	entityKey  string
	sessionID  string
	note       string
}

func (m *mockRunClaimService) Claim(ctx context.Context, in services.ClaimInput) (*models.EntityClaim, error) {
	m.claims = append(m.claims, in)
	return &models.EntityClaim{
		EntityType: in.EntityType,
		EntityKey:  in.EntityKey,
		SessionID:  "run-session-1",
	}, nil
}

func (m *mockRunClaimService) Release(ctx context.Context, entityType, entityKey, sessionID, outcome string, force bool) (bool, error) {
	m.releases = append(m.releases, runReleaseCall{
		entityType: entityType,
		entityKey:  entityKey,
		sessionID:  sessionID,
		outcome:    outcome,
		force:      force,
	})
	return true, nil
}

func (m *mockRunClaimService) Heartbeat(ctx context.Context, entityType, entityKey, sessionID string, progress *float64, note string) error {
	m.heartbeats = append(m.heartbeats, runHeartbeatCall{
		entityType: entityType,
		entityKey:  entityKey,
		sessionID:  sessionID,
		note:       note,
	})
	return nil
}

func (m *mockRunClaimService) TTL() time.Duration {
	if m.ttl > 0 {
		return m.ttl
	}
	return time.Hour
}

func withRunClaimSvcOverride(t *testing.T, svc runClaimServicer) {
	t.Helper()
	orig := runClaimSvcOverride
	runClaimSvcOverride = svc
	t.Cleanup(func() { runClaimSvcOverride = orig })
}

// ─── buildTransitioner ────────────────────────────────────────────────────────

func TestAcquireRunLease_DryRunSkipsClaim(t *testing.T) {
	mock := &mockRunClaimService{}
	withRunClaimSvcOverride(t, mock)

	lease, err := acquireRunLease(context.Background(), "bug", "B041", "", true, services.HarnessIdentity{})
	if err != nil {
		t.Fatalf("acquireRunLease dry-run: %v", err)
	}
	if lease != nil {
		t.Fatalf("dry-run lease = %#v, want nil", lease)
	}
	if len(mock.claims) != 0 {
		t.Fatalf("dry-run claimed %d times, want 0", len(mock.claims))
	}
}

func TestRunLease_ReleasesAcquiredSession(t *testing.T) {
	mock := &mockRunClaimService{}
	withRunClaimSvcOverride(t, mock)

	lease, err := acquireRunLease(context.Background(), "bug", "B041", "", false, services.HarnessIdentity{})
	if err != nil {
		t.Fatalf("acquireRunLease: %v", err)
	}
	if len(mock.claims) != 1 {
		t.Fatalf("claims = %d, want 1", len(mock.claims))
	}
	if got := mock.claims[0]; got.EntityType != "bug" || got.EntityKey != "B041" {
		t.Fatalf("claim input = %#v, want bug B041", got)
	}

	if err := lease.Release("completed"); err != nil {
		t.Fatalf("release: %v", err)
	}
	if len(mock.releases) != 1 {
		t.Fatalf("releases = %d, want 1", len(mock.releases))
	}
	got := mock.releases[0]
	if got.entityType != "bug" || got.entityKey != "B041" || got.sessionID != "run-session-1" {
		t.Fatalf("release target = %#v, want bug B041 session run-session-1", got)
	}
	if got.outcome != "completed" {
		t.Fatalf("release outcome = %q, want completed", got.outcome)
	}
	if got.force {
		t.Fatal("run release used force; want session-scoped release")
	}
}

// TC-307/TC-308: CLI run preflight must recognize a direct Question block
// after reading candidate status and before action lookup, responder work, or
// any normal/dry-run lease attempt.
func TestRunLeasePreflight_BlockedCandidateSkipsActionAndClaim_TC307_TC308(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "dry_run"}[dryRun], func(t *testing.T) {
			claims := &mockRunClaimService{}
			withRunClaimSvcOverride(t, claims)
			actionCalls := 0
			lease, block, status, err := acquireRunLeaseForRunnableAction(
				context.Background(),
				fixedNextTransitioner{info: &services.NextStatusInfo{EntityType: models.EntityTypeFeature, EntityKey: "E39-F03", CurrentStatus: "active"}},
				&config.MockActionService{GetStatusActionFunc: func(context.Context, string) (*config.OrchestratorAction, error) {
					actionCalls++
					return nil, nil
				}},
				questionBlockerFunc(func(_ context.Context, entityType models.EntityType, key string) (*services.QuestionBlock, error) {
					if entityType != models.EntityTypeFeature || key != "E39-F03" {
						t.Fatalf("blocker candidate = %s %s, want feature E39-F03", entityType, key)
					}
					return &services.QuestionBlock{QuestionKey: "Q001", Summary: "Gate", ResolutionOwner: "owner", CurrentResponder: "alice"}, nil
				}),
				"feature", "E39-F03", dryRun, services.HarnessIdentity{},
			)
			if err != nil {
				t.Fatalf("blocked run preflight error = %v", err)
			}
			if lease != nil || block == nil || status != "active" {
				t.Fatalf("blocked run preflight lease=%#v block=%#v status=%q, want nil/non-nil/active", lease, block, status)
			}
			if actionCalls != 0 || len(claims.claims) != 0 {
				t.Fatalf("blocked preflight action/claim calls = %d/%d, want 0/0", actionCalls, len(claims.claims))
			}
		})
	}
}

// TestAcquireRunLeaseForRunnableActionPropagatesQuestionBlockerCheckError
// locks in that a questionBlocker.Check failure propagates out of
// acquireRunLeaseForRunnableAction and stops before action lookup or lease
// acquisition -- no existing test double for this function ever returns a
// non-nil error, so a regression that silently swallowed it would otherwise
// pass every other preflight test in this file.
func TestAcquireRunLeaseForRunnableActionPropagatesQuestionBlockerCheckError(t *testing.T) {
	claims := &mockRunClaimService{}
	withRunClaimSvcOverride(t, claims)
	actionCalls := 0
	checkErr := errors.New("Question blocker load candidate: repository unavailable")
	lease, block, _, err := acquireRunLeaseForRunnableAction(
		context.Background(),
		fixedNextTransitioner{info: &services.NextStatusInfo{EntityType: models.EntityTypeFeature, EntityKey: "E39-F03", CurrentStatus: "active"}},
		&config.MockActionService{GetStatusActionFunc: func(context.Context, string) (*config.OrchestratorAction, error) {
			actionCalls++
			return nil, nil
		}},
		questionBlockerFunc(func(context.Context, models.EntityType, string) (*services.QuestionBlock, error) {
			return nil, checkErr
		}),
		"feature", "E39-F03", false, services.HarnessIdentity{},
	)
	if err == nil {
		t.Fatal("acquireRunLeaseForRunnableAction() error = nil, want the propagated Question blocker error")
	}
	if !errors.Is(err, checkErr) {
		t.Fatalf("acquireRunLeaseForRunnableAction() error = %v, want it to wrap %v", err, checkErr)
	}
	if lease != nil || block != nil {
		t.Fatalf("blocker error lease=%#v block=%#v, want nil/nil", lease, block)
	}
	if actionCalls != 0 || len(claims.claims) != 0 {
		t.Fatalf("blocker error action/claim calls = %d/%d, want 0/0", actionCalls, len(claims.claims))
	}
}

// TestPreflightCascadeQuestionBlockPropagatesCheckError locks in that a
// questionBlocker.Check failure during cascade traversal propagates out of
// preflightCascadeQuestionBlock instead of being treated as "no block."
func TestPreflightCascadeQuestionBlockPropagatesCheckError(t *testing.T) {
	checkErr := errors.New("Question blocker load candidate: repository unavailable")
	blocker := questionBlockerFunc(func(context.Context, models.EntityType, string) (*services.QuestionBlock, error) {
		return nil, checkErr
	})
	actions := &config.MockActionService{GetStatusActionFunc: func(context.Context, string) (*config.OrchestratorAction, error) {
		t.Fatal("blocker error must not reach action resolution")
		return nil, nil
	}}
	root := fixedNextTransitioner{info: &services.NextStatusInfo{CurrentStatus: "parent"}}

	got, _, err := preflightCascadeQuestionBlock(context.Background(), root, actions, &mockRunCascadeChildrenService{}, blocker,
		func(context.Context, string) (runner.EntityTransitioner, error) { return root, nil }, "epic", "E39")
	if err == nil {
		t.Fatal("preflightCascadeQuestionBlock() error = nil, want the propagated Question blocker error")
	}
	if !errors.Is(err, checkErr) {
		t.Fatalf("preflightCascadeQuestionBlock() error = %v, want it to wrap %v", err, checkErr)
	}
	if got != nil {
		t.Fatalf("preflightCascadeQuestionBlock() block = %#v, want nil on error", got)
	}
}

// TC-308: preflight may suppress the parent lease only when all selected
// cascade work is parked. A blocked first child must fall through to a later
// runnable sibling so the controller can dispatch it.
func TestPreflightCascadeQuestionBlockFallsThroughToLiveSibling_TC308(t *testing.T) {
	blocker := questionBlockerFunc(func(_ context.Context, entityType models.EntityType, key string) (*services.QuestionBlock, error) {
		if entityType == models.EntityTypeFeature && key == "E39-F03" {
			return &services.QuestionBlock{QuestionKey: "Q001", Summary: "Gate", ResolutionOwner: "owner", CurrentResponder: "alice"}, nil
		}
		return nil, nil
	})
	actions := &config.MockActionService{GetStatusActionFunc: func(_ context.Context, status string) (*config.OrchestratorAction, error) {
		switch status {
		case "parent":
			return &config.OrchestratorAction{Action: config.ActionCascade}, nil
		case "child":
			return &config.OrchestratorAction{Action: config.ActionSpawnAgent}, nil
		default:
			t.Fatalf("unexpected action status %q", status)
			return nil, nil
		}
	}}
	cascadeChildren := &mockRunCascadeChildrenService{state: services.CascadeChildrenState{Children: []services.CascadeChild{
		{Key: "E39-F03", EntityType: models.EntityTypeFeature},
		{Key: "E39-F04", EntityType: models.EntityTypeFeature},
	}}}
	root := fixedNextTransitioner{info: &services.NextStatusInfo{CurrentStatus: "parent"}}
	child := fixedNextTransitioner{info: &services.NextStatusInfo{CurrentStatus: "child"}}

	got, status, err := preflightCascadeQuestionBlock(context.Background(), root, actions, cascadeChildren, blocker,
		func(context.Context, string) (runner.EntityTransitioner, error) { return child, nil }, "epic", "E39")
	if err != nil {
		t.Fatalf("preflightCascadeQuestionBlock() error = %v", err)
	}
	if got != nil || status != "parent" {
		t.Fatalf("preflight = block=%#v status=%q, want live sibling/no block/parent", got, status)
	}
}

// TC-308: when every selected child is blocked, preflight returns the first
// compact handoff and keeps the parent lease-free in normal and dry-run paths.
func TestPreflightCascadeQuestionBlockAllBlockedReturnsCompactPause_TC308(t *testing.T) {
	blocker := questionBlockerFunc(func(_ context.Context, entityType models.EntityType, key string) (*services.QuestionBlock, error) {
		if entityType != models.EntityTypeFeature {
			return nil, nil
		}
		return &services.QuestionBlock{QuestionKey: "Q" + key[len(key)-2:], Summary: key, ResolutionOwner: "owner", CurrentResponder: "alice"}, nil
	})
	actions := &config.MockActionService{GetStatusActionFunc: func(_ context.Context, status string) (*config.OrchestratorAction, error) {
		if status == "parent" {
			return &config.OrchestratorAction{Action: config.ActionCascade}, nil
		}
		return &config.OrchestratorAction{Action: config.ActionSpawnAgent}, nil
	}}
	cascadeChildren := &mockRunCascadeChildrenService{state: services.CascadeChildrenState{Children: []services.CascadeChild{
		{Key: "E39-F03", EntityType: models.EntityTypeFeature},
		{Key: "E39-F04", EntityType: models.EntityTypeFeature},
	}}}
	root := fixedNextTransitioner{info: &services.NextStatusInfo{CurrentStatus: "parent"}}
	child := fixedNextTransitioner{info: &services.NextStatusInfo{CurrentStatus: "child"}}

	got, status, err := preflightCascadeQuestionBlock(context.Background(), root, actions, cascadeChildren, blocker,
		func(context.Context, string) (runner.EntityTransitioner, error) { return child, nil }, "epic", "E39")
	if err != nil {
		t.Fatalf("preflightCascadeQuestionBlock() error = %v", err)
	}
	if got == nil || got.QuestionKey != "Q03" || got.Summary != "E39-F03" || status != "parent" {
		t.Fatalf("preflight = block=%#v status=%q, want first compact child block/parent", got, status)
	}
}

// TC-104: the production run lease boundary must inspect a Question pause
// action before it derives a responder-bound claim identity. A Question with
// no responder is a valid parked checkpoint, not a run failure or a claim.
func TestRunLeasePreflight_TopLevelQuestionPauseSkipsResponderClaim_TC104(t *testing.T) {
	claims := &mockRunClaimService{}
	withRunClaimSvcOverride(t, claims)

	transitioner := fixedNextTransitioner{info: &services.NextStatusInfo{
		EntityType:    models.EntityTypeQuestion,
		EntityKey:     "Q001",
		CurrentStatus: "ready_for_resolution",
	}}
	actions := &config.MockActionService{GetStatusActionFunc: func(_ context.Context, status string) (*config.OrchestratorAction, error) {
		if status != "ready_for_resolution" {
			t.Fatalf("preflight status = %q, want ready_for_resolution", status)
		}
		return &config.OrchestratorAction{Action: config.ActionPause}, nil
	}}

	lease, block, _, err := acquireRunLeaseForRunnableAction(context.Background(), transitioner, actions, nil, "question", "Q001", false, services.HarnessIdentity{})
	if err != nil {
		t.Fatalf("top-level Question pause preflight: %v", err)
	}
	if lease != nil {
		t.Fatalf("pause preflight lease = %#v, want nil", lease)
	}
	if block != nil {
		t.Fatalf("pause preflight block = %#v, want nil", block)
	}
	if len(claims.claims) != 0 {
		t.Fatalf("pause preflight claims = %#v, want none", claims.claims)
	}
}

// TC-104: cascade children enter the same lease boundary. A ready Question
// has a terminal dispatch signal and must stop before either action rendering
// or responder-bound claim derivation.
func TestRunLeasePreflight_CascadeReadyQuestionSkipsActionAndClaim_TC104(t *testing.T) {
	claims := &mockRunClaimService{}
	withRunClaimSvcOverride(t, claims)

	transitioner := fixedNextTransitioner{info: &services.NextStatusInfo{
		EntityType:    models.EntityTypeQuestion,
		EntityKey:     "Q001",
		CurrentStatus: "ready_for_resolution",
		IsTerminal:    true,
	}}
	actions := &config.MockActionService{GetStatusActionFunc: func(context.Context, string) (*config.OrchestratorAction, error) {
		t.Fatal("cascade ready Question looked up an action before stopping")
		return nil, nil
	}}

	lease, block, _, err := acquireRunLeaseForRunnableAction(context.Background(), transitioner, actions, nil, "question", "Q001", false, services.HarnessIdentity{})
	if err != nil {
		t.Fatalf("cascade ready Question preflight: %v", err)
	}
	if lease != nil {
		t.Fatalf("cascade ready Question lease = %#v, want nil", lease)
	}
	if block != nil {
		t.Fatalf("cascade ready Question block = %#v, want nil", block)
	}
	if len(claims.claims) != 0 {
		t.Fatalf("cascade ready Question claims = %#v, want none", claims.claims)
	}
}

// TC-104: The CLI run preflight receives the same responder-less dispatch
// signal as keyed next and the controller. It must avoid both action lookup
// and responder-bound claims for every checkpoint in dry-run and real-run
// modes, including F01's unconfigured draft compatibility state.
func TestRunLeasePreflight_QuestionNoResponderParity_TC104(t *testing.T) {
	for _, status := range []string{"draft", "open", "answering", "ready_for_resolution"} {
		t.Run(status, func(t *testing.T) {
			for _, dryRun := range []bool{false, true} {
				t.Run(map[bool]string{false: "run", true: "dry_run"}[dryRun], func(t *testing.T) {
					claims := &mockRunClaimService{}
					withRunClaimSvcOverride(t, claims)
					transitioner := fixedNextTransitioner{info: &services.NextStatusInfo{
						EntityType: models.EntityTypeQuestion, EntityKey: "Q001", CurrentStatus: status, IsTerminal: true,
					}}
					actions := &config.MockActionService{GetStatusActionFunc: func(context.Context, string) (*config.OrchestratorAction, error) {
						t.Fatal("Question no-responder preflight looked up a workflow action")
						return nil, nil
					}}

					lease, block, _, err := acquireRunLeaseForRunnableAction(context.Background(), transitioner, actions, nil, "question", "Q001", dryRun, services.HarnessIdentity{})
					if err != nil {
						t.Fatalf("Question %s preflight: %v", status, err)
					}
					if lease != nil || block != nil || len(claims.claims) != 0 {
						t.Fatalf("Question %s preflight lease=%#v block=%#v claims=%#v, want none", status, lease, block, claims.claims)
					}
				})
			}
		})
	}
}

// TestBuildTransitioner_UnsupportedType verifies that buildTransitioner returns
// an error for an entity type it does not recognize. This code path does not
// call any global service accessor, so no database is required.
func TestBuildTransitioner_UnsupportedType(t *testing.T) {
	ctx := context.Background()
	transitioner, err := buildTransitioner(ctx, "unknown_entity")

	if err == nil {
		t.Fatal("expected error for unsupported entity type, got nil")
	}
	if transitioner != nil {
		t.Errorf("expected nil transitioner for unsupported type, got %T", transitioner)
	}
	// Error message must identify the unsupported type.
	if err != nil && err.Error() == "" {
		t.Error("expected non-empty error message for unsupported entity type")
	}
}

// TestBuildTransitioner_UnsupportedType_ErrorMessage verifies the error message
// includes the unsupported type name so callers can diagnose the problem.
func TestBuildTransitioner_UnsupportedType_ErrorMessage(t *testing.T) {
	ctx := context.Background()
	_, err := buildTransitioner(ctx, "widget")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if len(msg) == 0 {
		t.Error("error message is empty")
	}
	// The type string "widget" should appear in the error for diagnosability.
	found := false
	for i := 0; i+len("widget") <= len(msg); i++ {
		if msg[i:i+len("widget")] == "widget" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("error message %q does not contain the unsupported type name %q", msg, "widget")
	}
}

// ─── buildPlaceholderGenerator ────────────────────────────────────────────────

// TestBuildPlaceholderGenerator_UnsupportedType verifies that
// buildPlaceholderGenerator returns nil for an unknown entity type.
// Per the contract: "controller handles nil gracefully."
func TestBuildPlaceholderGenerator_UnsupportedType(t *testing.T) {
	ctx := context.Background()
	gen := buildPlaceholderGenerator(ctx, "unknown_entity")

	if gen != nil {
		t.Errorf("expected nil PlaceholderGenerator for unsupported entity type, got %T", gen)
	}
}

// TestBuildPlaceholderGenerator_UnsupportedType_EmptyString verifies that
// the empty string also produces nil (not a panic).
func TestBuildPlaceholderGenerator_UnsupportedType_EmptyString(t *testing.T) {
	ctx := context.Background()
	gen := buildPlaceholderGenerator(ctx, "")

	if gen != nil {
		t.Errorf("expected nil PlaceholderGenerator for empty entity type, got %T", gen)
	}
}

// ─── GetActionService caching ─────────────────────────────────────────────────

// setupActionServiceTestDir creates a temporary directory with a minimal
// .sharkconfig.json so that GetActionService can initialize without error.
// It changes the working directory to the temp dir and returns a cleanup
// function that resets service globals and restores the original directory.
func setupActionServiceTestDir(t *testing.T) func() {
	t.Helper()

	tmpDir := t.TempDir()

	// Minimal config sufficient for config.NewActionService to succeed.
	configContent := `{}`
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir to tmpDir: %v", err)
	}

	return func() {
		cli.ResetServices()
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}
}

// TestGetActionService_ReturnsNonNil verifies that GetActionService returns a
// non-nil service when a valid .sharkconfig.json exists in the project root.
func TestGetActionService_ReturnsNonNil(t *testing.T) {
	cleanup := setupActionServiceTestDir(t)
	defer cleanup()

	ctx := context.Background()
	svc, err := cli.GetActionService(ctx)
	if err != nil {
		t.Fatalf("GetActionService() unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("GetActionService() returned nil service")
	}
}

// TestGetActionService_SameInstanceOnRepeatCalls verifies that repeated calls
// to GetActionService return the same instance (sync.Once caching behavior).
func TestGetActionService_SameInstanceOnRepeatCalls(t *testing.T) {
	cleanup := setupActionServiceTestDir(t)
	defer cleanup()

	ctx := context.Background()

	svc1, err := cli.GetActionService(ctx)
	if err != nil {
		t.Fatalf("GetActionService() first call error: %v", err)
	}

	svc2, err := cli.GetActionService(ctx)
	if err != nil {
		t.Fatalf("GetActionService() second call error: %v", err)
	}

	// Both calls must return the exact same pointer (sync.Once caching).
	if svc1 != svc2 {
		t.Errorf("GetActionService() returned different instances on repeated calls: %p vs %p", svc1, svc2)
	}
}

// TestFindProjectRoot_ResolvesTempDirWithConfig verifies that cli.FindProjectRoot
// — the function runRun calls to populate RunOptions.ProjectRoot — correctly
// resolves to a directory containing .sharkconfig.json. An empty ProjectRoot
// would cause maybeWriteTranscript to short-circuit silently.
func TestFindProjectRoot_ResolvesTempDirWithConfig(t *testing.T) {
	cleanup := setupActionServiceTestDir(t)
	defer cleanup()

	root, err := cli.FindProjectRoot()
	if err != nil {
		t.Fatalf("FindProjectRoot() returned error: %v", err)
	}
	if root == "" {
		t.Fatal("FindProjectRoot() returned empty string; RunOptions.ProjectRoot would be empty")
	}

	// The resolved root must contain .sharkconfig.json (the marker the test
	// helper wrote). If FindProjectRoot ever stops honoring that marker the
	// transcript directory path would land in an unexpected location.
	if _, err := os.Stat(filepath.Join(root, ".sharkconfig.json")); err != nil {
		t.Errorf("FindProjectRoot() resolved to %s which does not contain .sharkconfig.json: %v", root, err)
	}
}

// TestGetActionService_ReturnsErrorAfterReset verifies that after ResetServices()
// a new call to GetActionService re-initializes from the config (sync.Once resets).
func TestGetActionService_ResetAndReinit(t *testing.T) {
	cleanup := setupActionServiceTestDir(t)
	defer cleanup()

	ctx := context.Background()

	// First initialization.
	svc1, err := cli.GetActionService(ctx)
	if err != nil {
		t.Fatalf("GetActionService() first call error: %v", err)
	}

	// Reset clears cached state.
	cli.ResetServices()

	// Second initialization — should succeed again (same config file still present).
	svc2, err := cli.GetActionService(ctx)
	if err != nil {
		t.Fatalf("GetActionService() after reset error: %v", err)
	}
	if svc2 == nil {
		t.Fatal("GetActionService() returned nil after reset")
	}

	// The two instances may or may not be equal by pointer — what matters
	// is that both are valid non-nil services.
	_ = svc1
}

// ─── AST source-guard test infrastructure (TC-013, TC-018, TC-025) ──────────
//
// runRun has no mock seam: test-plan.md's "Considered and rejected" section
// evaluated extending runClaimSvcOverride-style injection to buildTransitioner
// / cli.Get*Service and rejected it as disproportionate for a STANDARD-scored
// feature (six-plus call sites would need refactoring into injectable seams).
// D7 sanctions go/parser source-invariant checks for exactly this class of
// property (wiring order, writer target) — syntactic properties, not
// behavioral ones, following the codebase's existing source-invariant
// validator convention (e.g. internal/services/portfolio_advice_service_test.go).
//
// parseRunGoSource is shared by run_test.go and run_worktree_test.go (same
// package).

// parseRunGoSource parses run.go — this package's production source file,
// not the test file — and returns its FileSet plus the runRun FuncDecl.
func parseRunGoSource(t *testing.T) (*token.FileSet, *ast.FuncDecl) {
	t.Helper()

	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() could not locate run_test.go")
	}
	runGoPath := filepath.Join(filepath.Dir(testFile), "run.go")

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, runGoPath, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", runGoPath, err)
	}

	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "runRun" {
			return fset, fn
		}
	}
	t.Fatalf("%s does not declare a runRun function", runGoPath)
	return nil, nil
}

// findMethodCallPos returns the position of the earliest call `<recv>.<method>(...)`
// found anywhere within root, or token.NoPos if no such call exists.
func findMethodCallPos(root ast.Node, recv, method string) token.Pos {
	pos := token.NoPos
	ast.Inspect(root, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != method {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != recv {
			return true
		}
		if pos == token.NoPos || call.Pos() < pos {
			pos = call.Pos()
		}
		return true
	})
	return pos
}

// findFuncCallPos returns the position of the earliest call to the bare
// (non-method) function named name, found anywhere within root, or
// token.NoPos if no such call exists.
func findFuncCallPos(root ast.Node, name string) token.Pos {
	pos := token.NoPos
	ast.Inspect(root, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok || ident.Name != name {
			return true
		}
		if pos == token.NoPos || call.Pos() < pos {
			pos = call.Pos()
		}
		return true
	})
	return pos
}

// TestRunRunLivenessTeardownIsClosure is TC-018 (test-plan.md): runRun's
// liveness-recorder teardown defer must be
// `defer func() { rec.Stop(); rec.Finish(runResult) }()` — a closure —
// never a direct `defer rec.Finish(runResult)`. A direct method-value defer
// evaluates its arguments at registration time and would capture nil
// forever, silently losing the run-end outcome even though it would pass
// every liveness_test.go unit test in isolation (D6 edit 2; the emitRunEnd
// defer already above in runRun documents this exact trap). AC-06.
func TestRunRunLivenessTeardownIsClosure(t *testing.T) {
	_, runRunDecl := parseRunGoSource(t)

	var closureFound, directCaptureFound bool
	ast.Inspect(runRunDecl, func(n ast.Node) bool {
		def, ok := n.(*ast.DeferStmt)
		if !ok {
			return true
		}
		switch fun := def.Call.Fun.(type) {
		case *ast.SelectorExpr:
			if recv, ok := fun.X.(*ast.Ident); ok && recv.Name == "rec" && fun.Sel.Name == "Finish" {
				directCaptureFound = true
			}
		case *ast.FuncLit:
			ast.Inspect(fun.Body, func(inner ast.Node) bool {
				call, ok := inner.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				recv, ok := sel.X.(*ast.Ident)
				if ok && recv.Name == "rec" && sel.Sel.Name == "Finish" {
					closureFound = true
				}
				return true
			})
		}
		return true
	})

	if directCaptureFound {
		t.Fatal("runRun defers rec.Finish directly (defer rec.Finish(runResult)) — this evaluates the argument at registration time and captures nil forever; must be defer func() { rec.Stop(); rec.Finish(runResult) }()")
	}
	if !closureFound {
		t.Fatal("runRun has no `defer func() { ...; rec.Finish(...) }()` teardown closure for the liveness recorder")
	}
}

// TestRunRunLivenessStartPrecedesPreflight is TC-025 (test-plan.md): the
// liveness recorder's Start() call must occur, in source order, before both
// preflight/lease-acquisition calls (preflightCascadeQuestionBlock,
// acquireRunLeaseForRunnableAction) in runRun — so a run that pauses at a
// Question block still leaves a log (D6 edit 2's stated reason for the
// ordering). An implementation that constructs the recorder correctly but
// starts it after either preflight call would pass every liveness_test.go
// test while silently breaking liveness for every Question-paused run. AC-10.
func TestRunRunLivenessStartPrecedesPreflight(t *testing.T) {
	fset, runRunDecl := parseRunGoSource(t)

	recStartPos := findMethodCallPos(runRunDecl, "rec", "Start")
	preflightPos := findFuncCallPos(runRunDecl, "preflightCascadeQuestionBlock")
	leasePos := findFuncCallPos(runRunDecl, "acquireRunLeaseForRunnableAction")

	if recStartPos == token.NoPos {
		t.Fatal("runRun does not call rec.Start()")
	}
	if preflightPos == token.NoPos {
		t.Fatal("runRun does not call preflightCascadeQuestionBlock")
	}
	if leasePos == token.NoPos {
		t.Fatal("runRun does not call acquireRunLeaseForRunnableAction")
	}

	if recStartPos >= preflightPos {
		t.Errorf("rec.Start() at %s must precede preflightCascadeQuestionBlock at %s", fset.Position(recStartPos), fset.Position(preflightPos))
	}
	if recStartPos >= leasePos {
		t.Errorf("rec.Start() at %s must precede acquireRunLeaseForRunnableAction at %s", fset.Position(recStartPos), fset.Position(leasePos))
	}
}

// TestRunRunRunControllerDepsAlwaysSetsGateIngest is F-1's regression guard
// (T-E34-F05-004 rework, code-review-20260901T061603Z-E34-F05.md): every
// runner.RunControllerDeps{...} composite literal runRun constructs — the
// top-level controller and the cascade-child controller inside the runChild
// closure — must set a non-nil GateIngest field. Before this fix neither
// site set it at all, so RunController.gateIngest stayed nil and every
// gate_result_v1 step (including the shipped change.code_review/change.qa
// steps) failed closed inside ingestGateResultForDispatch with "requires a
// configured GateResult persistence coordinator" the moment a real operator
// ran `shark run` in the foreground — only --apply-result/--resume-run
// (which call buildGateCoordinator directly) were reachable. Every existing
// controller-level gate test hand-builds RunControllerDeps with GateIngest
// pre-populated, which is exactly why this shipped undetected: this test
// inspects run.go's own actual construction sites via go/parser (the
// existing D7 source-invariant convention for a function with no mock
// seam — see the "AST source-guard test infrastructure" comment above)
// instead of a hand-built RunControllerDeps.
func TestRunRunRunControllerDepsAlwaysSetsGateIngest(t *testing.T) {
	_, runRunDecl := parseRunGoSource(t)

	var sitesChecked int
	ast.Inspect(runRunDecl, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		sel, ok := lit.Type.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != "RunControllerDeps" {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "runner" {
			return true
		}
		sitesChecked++

		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "GateIngest" {
				continue
			}
			if ident, ok := kv.Value.(*ast.Ident); ok && ident.Name == "nil" {
				t.Fatalf("runner.RunControllerDeps{} at %v sets GateIngest to a literal nil", lit.Pos())
			}
			return true
		}
		t.Fatalf("runner.RunControllerDeps{} at %v does not set GateIngest at all — every gate_result_v1 step will fail closed with \"requires a configured GateResult persistence coordinator\" through this construction site", lit.Pos())
		return true
	})

	if sitesChecked < 2 {
		t.Fatalf("expected to find 2 runner.RunControllerDeps{} construction sites in runRun (top-level + cascade child), found %d — this test's own detection may have broken", sitesChecked)
	}
}
