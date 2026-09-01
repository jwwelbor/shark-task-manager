// Package commands provides CLI command implementations.
//
// This file covers the T-E34-F01-005 rework (UAT rejection): shark run's
// claim acquisition (acquireRunLease) never carried Harness/HarnessVersion/
// HarnessModel into services.ClaimInput, so the claim tier of REQ-F-002/AC-08
// was unreachable through a real `shark run` -- next_run_harness_parity_test.go's
// TC-010 only proved HarnessResolver.Resolve *can* read a scripted claim
// (harnessMockClaimReader), never that shark run's own acquireRunLease
// *writes* one. This file drives the real production chain end-to-end: real
// *services.ClaimService (backed by an in-memory ClaimRepository double, per
// the CLI-tests-never-touch-a-real-DB rule -- full runRun needs DB-backed
// global services, which would violate that rule), real
// acquireRunLeaseForRunnableAction, real *services.HarnessResolver, and real
// runner.RunController.Run. No ClaimReader is scripted with a fixed claim;
// the claim the resolver reads is the one production code created.
package commands

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	claimrepo "github.com/jwwelbor/shark-task-manager/internal/repository/claim"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// inMemoryClaimRepo is a minimal, real (non-scripted) implementation of
// services.ClaimRepository: an in-memory map with the same single-grab
// (UNIQUE entity_type+entity_key) and session-scoped semantics as
// internal/repository/claim.Repository, without touching a real database.
// Building a genuine *services.ClaimService on top of this -- rather than
// mocking ClaimService/ClaimReader directly -- is what lets this test prove
// the real acquisition-to-resolution chain, not a scripted stand-in for it.
type inMemoryClaimRepo struct {
	mu     sync.Mutex
	claims map[string]*models.EntityClaim
	nextID int64
}

func newInMemoryClaimRepo() *inMemoryClaimRepo {
	return &inMemoryClaimRepo{claims: map[string]*models.EntityClaim{}}
}

func claimRepoKey(entityType, entityKey string) string { return entityType + "/" + entityKey }

func (r *inMemoryClaimRepo) Claim(_ context.Context, c *models.EntityClaim) (*models.EntityClaim, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := claimRepoKey(c.EntityType, c.EntityKey)
	if _, exists := r.claims[key]; exists {
		return nil, claimrepo.ErrAlreadyClaimed
	}
	r.nextID++
	stored := *c
	stored.ID = r.nextID
	stored.ClaimedAt = time.Now().UTC()
	stored.LastHeartbeat = stored.ClaimedAt
	r.claims[key] = &stored
	out := stored
	return &out, nil
}

func (r *inMemoryClaimRepo) Get(_ context.Context, entityType, entityKey string) (*models.EntityClaim, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.claims[claimRepoKey(entityType, entityKey)]
	if !ok {
		return nil, nil
	}
	out := *c
	return &out, nil
}

func (r *inMemoryClaimRepo) Release(_ context.Context, entityType, entityKey string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := claimRepoKey(entityType, entityKey)
	if _, ok := r.claims[key]; !ok {
		return false, nil
	}
	delete(r.claims, key)
	return true, nil
}

func (r *inMemoryClaimRepo) ReleaseSession(_ context.Context, entityType, entityKey, sessionID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := claimRepoKey(entityType, entityKey)
	c, ok := r.claims[key]
	if !ok || c.SessionID != sessionID {
		return false, nil
	}
	delete(r.claims, key)
	return true, nil
}

func (r *inMemoryClaimRepo) Renew(_ context.Context, entityType, entityKey, sessionID string, _ *float64, _ string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.claims[claimRepoKey(entityType, entityKey)]
	if !ok || c.SessionID != sessionID {
		return false, nil
	}
	c.LastHeartbeat = time.Now().UTC()
	return true, nil
}

func (r *inMemoryClaimRepo) ReclaimExpired(context.Context, time.Duration) (int64, error) {
	return 0, nil
}

func (r *inMemoryClaimRepo) List(context.Context) ([]*models.EntityClaim, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*models.EntityClaim, 0, len(r.claims))
	for _, c := range r.claims {
		cp := *c
		out = append(out, &cp)
	}
	return out, nil
}

var _ services.ClaimRepository = (*inMemoryClaimRepo)(nil)

// TestRunClaimTierReachableThroughRealAcquisition covers the T-E34-F01-005
// rework: with no --harness override on either the preflight lease
// acquisition or the controller run, SHARK_HARNESS=codex at claim-acquisition
// time must (1) actually land on the claim acquireRunLeaseForRunnableAction
// creates, and (2) survive to decide the controller's render even after the
// env var that seeded it changes -- proving the value came from the
// persisted claim (the claim tier), not a live env re-read (the env tier).
// This is the discriminating assertion: seeding ClaimInput from
// override-only (rather than override-else-env) would leave the claim's
// harness fields empty here and fail step (1) outright.
func TestRunClaimTierReachableThroughRealAcquisition(t *testing.T) {
	unsetHarnessEnv(t)
	t.Setenv("SHARK_HARNESS", "codex")

	repo := newInMemoryClaimRepo()
	ttl := time.Hour
	claimSvc := services.NewClaimService(repo, &ttl)
	withRunClaimSvcOverride(t, claimSvc)

	transitioner := fixedNextTransitioner{info: &services.NextStatusInfo{CurrentStatus: "in_progress"}}
	renderer, templateName := harnessDispatchRenderer(t, harnessIfTemplate)
	actionSvc := &config.MockActionService{
		GetStatusActionFunc: func(context.Context, string) (*config.OrchestratorAction, error) {
			return &config.OrchestratorAction{Action: config.ActionSpawnAgent}, nil
		},
		GetStatusActionPopulatedFunc: func(_ context.Context, _ string, vars map[string]string) (*config.PopulatedAction, error) {
			rendered, err := renderer.Render(templateName, vars)
			if err != nil {
				return nil, err
			}
			return &config.PopulatedAction{Action: "spawn_agent", Instruction: rendered}, nil
		},
	}

	// Step 1: real acquisition. No --harness flag override -- only env feeds
	// this claim.
	lease, block, _, err := acquireRunLeaseForRunnableAction(
		context.Background(), transitioner, actionSvc, nil, "task", "E01-F01-001", false, services.HarnessIdentity{},
	)
	if err != nil {
		t.Fatalf("acquireRunLeaseForRunnableAction: %v", err)
	}
	if block != nil {
		t.Fatalf("unexpected Question block: %#v", block)
	}
	if lease == nil {
		t.Fatal("acquireRunLeaseForRunnableAction returned a nil lease for a dispatchable action")
	}
	t.Cleanup(func() { _ = lease.Release("completed") })

	stored, err := claimSvc.Get(context.Background(), "task", "E01-F01-001")
	if err != nil {
		t.Fatalf("claimSvc.Get: %v", err)
	}
	if stored == nil {
		t.Fatal("acquireRunLeaseForRunnableAction did not persist a claim")
	}
	if stored.Harness != "codex" {
		t.Fatalf("persisted claim harness = %q, want %q (acquireRunLease must carry the env-resolved harness identity into ClaimInput)",
			stored.Harness, "codex")
	}

	// Step 2: change the env AFTER the claim exists, then drive the real
	// controller. If the render still reflects "codex", the value came from
	// the persisted claim, not from a live env re-read.
	t.Setenv("SHARK_HARNESS", "claude")

	var captured string
	ctrl, err := runner.NewRunController(runner.RunControllerDeps{
		Transitioner: transitioner,
		Placeholders: fixedNextPlaceholders{vars: map[string]string{}},
		ActionSvc:    actionSvc,
		WorkflowSvc:  workflow.NewService(""),
		Dispatchers:  map[string]runner.AgentDispatcher{"": &parityDispatcher{captured: &captured}},
		PromptAssembler: runner.PromptAssemblerFunc(func(_ context.Context, input runner.PromptAssemblyInput) (string, error) {
			return assembleDispatchPrompt(input.Instruction, input.AgentType, input.Vars)
		}),
		HarnessResolver: services.NewHarnessResolver(claimSvc),
	})
	if err != nil {
		t.Fatalf("NewRunController: %v", err)
	}

	if _, err := ctrl.Run(context.Background(), "E01-F01-001", runner.RunOptions{
		EntityType: "task",
		SessionID:  lease.sessionID,
	}); err != nil {
		t.Fatalf("RunController.Run() returned unexpected error: %v", err)
	}
	if captured == "" {
		t.Fatal("dispatcher never captured a rendered prompt")
	}
	if !strings.Contains(captured, harnessBranchB) {
		t.Fatalf("rendered prompt = %q, want the codex branch (%s) reflecting the persisted claim -- env has since changed to claude",
			captured, harnessBranchB)
	}
}

// TestRunClaimTierDryRun_RendersViaEnvNotClaim covers the --dry-run edge of
// the same fix: acquireRunLease returns (nil, nil) before touching the claim
// when dryRun is true (unchanged by this fix), so no claim is ever created.
// The controller's HarnessResolver.Resolve must still fall through to the
// env tier directly for the render to reflect SHARK_HARNESS -- proving
// dry-run and real-run agree on the rendered identity (env-tier vs
// claim-tier respectively) precisely because the claim is itself seeded from
// env when no override is given.
func TestRunClaimTierDryRun_RendersViaEnvNotClaim(t *testing.T) {
	unsetHarnessEnv(t)
	t.Setenv("SHARK_HARNESS", "codex")

	repo := newInMemoryClaimRepo()
	ttl := time.Hour
	claimSvc := services.NewClaimService(repo, &ttl)
	withRunClaimSvcOverride(t, claimSvc)

	transitioner := fixedNextTransitioner{info: &services.NextStatusInfo{CurrentStatus: "in_progress"}}
	renderer, templateName := harnessDispatchRenderer(t, harnessIfTemplate)
	actionSvc := &config.MockActionService{
		GetStatusActionFunc: func(context.Context, string) (*config.OrchestratorAction, error) {
			return &config.OrchestratorAction{Action: config.ActionSpawnAgent}, nil
		},
		GetStatusActionPopulatedFunc: func(_ context.Context, _ string, vars map[string]string) (*config.PopulatedAction, error) {
			rendered, err := renderer.Render(templateName, vars)
			if err != nil {
				return nil, err
			}
			return &config.PopulatedAction{Action: "spawn_agent", Instruction: rendered}, nil
		},
	}

	lease, block, _, err := acquireRunLeaseForRunnableAction(
		context.Background(), transitioner, actionSvc, nil, "task", "E01-F01-001", true, services.HarnessIdentity{},
	)
	if err != nil {
		t.Fatalf("acquireRunLeaseForRunnableAction (dry-run): %v", err)
	}
	if block != nil {
		t.Fatalf("unexpected Question block: %#v", block)
	}
	if lease != nil {
		t.Fatalf("dry-run lease = %#v, want nil (no claim created)", lease)
	}
	if stored, getErr := claimSvc.Get(context.Background(), "task", "E01-F01-001"); getErr != nil || stored != nil {
		t.Fatalf("dry-run must not create a claim: stored=%#v err=%v", stored, getErr)
	}

	// handleSpawnAgent calls PromptAssembler.AssemblePrompt before its
	// dry-run branch, but returns before ever reaching Dispatch -- so unlike
	// TestRunClaimTierReachableThroughRealAcquisition, the prompt must be
	// captured at the assembler, not the dispatcher (whose Dispatch method
	// dry-run never calls).
	var captured string
	ctrl, err := runner.NewRunController(runner.RunControllerDeps{
		Transitioner: transitioner,
		Placeholders: fixedNextPlaceholders{vars: map[string]string{}},
		ActionSvc:    actionSvc,
		WorkflowSvc:  workflow.NewService(""),
		Dispatchers:  map[string]runner.AgentDispatcher{"": &parityDispatcher{captured: new(string)}},
		PromptAssembler: runner.PromptAssemblerFunc(func(_ context.Context, input runner.PromptAssemblyInput) (string, error) {
			rendered, err := assembleDispatchPrompt(input.Instruction, input.AgentType, input.Vars)
			if err == nil {
				captured = rendered
			}
			return rendered, err
		}),
		HarnessResolver: services.NewHarnessResolver(claimSvc),
	})
	if err != nil {
		t.Fatalf("NewRunController: %v", err)
	}

	if _, err := ctrl.Run(context.Background(), "E01-F01-001", runner.RunOptions{
		EntityType: "task",
		DryRun:     true,
	}); err != nil {
		t.Fatalf("RunController.Run() returned unexpected error: %v", err)
	}
	if captured == "" {
		t.Fatal("prompt assembler never captured a rendered prompt")
	}
	if !strings.Contains(captured, harnessBranchB) {
		t.Fatalf("rendered prompt = %q, want the codex branch (%s) sourced from env (no claim exists in dry-run)", captured, harnessBranchB)
	}
}

// TestAcquireRunLease_FlagOverrideBeatsEnvOnClaim locks in per-field
// precedence (override wins) at claim-creation time: with --harness=claude
// passed as an override and SHARK_HARNESS=codex in the environment, the
// persisted claim must carry "claude", not "codex".
func TestAcquireRunLease_FlagOverrideBeatsEnvOnClaim(t *testing.T) {
	unsetHarnessEnv(t)
	t.Setenv("SHARK_HARNESS", "codex")

	mock := &mockRunClaimService{}
	withRunClaimSvcOverride(t, mock)

	override := services.HarnessIdentity{Type: "Claude", Version: " 2.1.0 ", Model: " opus "}
	if _, err := acquireRunLease(context.Background(), "bug", "B041", "", false, override); err != nil {
		t.Fatalf("acquireRunLease: %v", err)
	}
	if len(mock.claims) != 1 {
		t.Fatalf("claims = %d, want 1", len(mock.claims))
	}
	got := mock.claims[0]
	if got.Harness != "claude" {
		t.Fatalf("claim Harness = %q, want %q (override wins over env, normalized lowercase)", got.Harness, "claude")
	}
	if got.HarnessVersion != "2.1.0" {
		t.Fatalf("claim HarnessVersion = %q, want %q (trimmed, not lowercased)", got.HarnessVersion, "2.1.0")
	}
	if got.HarnessModel != "opus" {
		t.Fatalf("claim HarnessModel = %q, want %q (trimmed, not lowercased)", got.HarnessModel, "opus")
	}
}

// TestAcquireRunLease_NoOverrideNoEnvLeavesClaimHarnessEmpty locks in the
// zero case: no flag override and no env var must leave the claim's harness
// fields empty, matching `shark claim` with no --harness flag.
func TestAcquireRunLease_NoOverrideNoEnvLeavesClaimHarnessEmpty(t *testing.T) {
	unsetHarnessEnv(t)

	mock := &mockRunClaimService{}
	withRunClaimSvcOverride(t, mock)

	if _, err := acquireRunLease(context.Background(), "bug", "B041", "", false, services.HarnessIdentity{}); err != nil {
		t.Fatalf("acquireRunLease: %v", err)
	}
	if len(mock.claims) != 1 {
		t.Fatalf("claims = %d, want 1", len(mock.claims))
	}
	got := mock.claims[0]
	if got.Harness != "" || got.HarnessVersion != "" || got.HarnessModel != "" {
		t.Fatalf("claim harness fields = %#v, want all empty", got)
	}
}

// TestCascadeChildLease_InheritsParentHarnessOverride locks in the second
// half of the rework's sweep: a cascade child's claim acquisition
// (run.go's runChild closure, via acquireRunLeaseForRunnableAction) must
// receive the parent's --harness override too, not only the parent's own
// lease and the controller's template rendering. Before this fix,
// childOpts.HarnessOverride reached childController.Run (rendering) via
// `childOpts := opts` but was silently dropped before reaching the child's
// own acquireRunLeaseForRunnableAction call, which only forwarded
// childOpts.DryRun.
func TestCascadeChildLease_InheritsParentHarnessOverride(t *testing.T) {
	unsetHarnessEnv(t)

	mock := &mockRunClaimService{}
	withRunClaimSvcOverride(t, mock)

	transitioner := fixedNextTransitioner{info: &services.NextStatusInfo{CurrentStatus: "in_progress"}}
	actions := &config.MockActionService{GetStatusActionFunc: func(context.Context, string) (*config.OrchestratorAction, error) {
		return &config.OrchestratorAction{Action: config.ActionSpawnAgent}, nil
	}}

	// childOpts.HarnessOverride, as it would be copied from the parent's
	// RunOptions via `childOpts := opts` in controller.go's handleCascade.
	childHarnessOverride := services.HarnessIdentity{Type: "codex"}

	lease, block, _, err := acquireRunLeaseForRunnableAction(
		context.Background(), transitioner, actions, nil, "feature", "E01-F02", false, childHarnessOverride,
	)
	if err != nil {
		t.Fatalf("acquireRunLeaseForRunnableAction: %v", err)
	}
	if block != nil {
		t.Fatalf("unexpected Question block: %#v", block)
	}
	if lease == nil {
		t.Fatal("acquireRunLeaseForRunnableAction returned a nil lease for a dispatchable action")
	}
	if len(mock.claims) != 1 {
		t.Fatalf("claims = %d, want 1", len(mock.claims))
	}
	if got := mock.claims[0].Harness; got != "codex" {
		t.Fatalf("cascade child claim Harness = %q, want %q (parent's --harness override must reach child claim construction)", got, "codex")
	}
}
