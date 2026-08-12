package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// TestAttachAgentBody_NoAgentType verifies the pure-logic degenerate branch
// of attachAgentBody — when agentType is empty, the prompt must pass
// through untouched. This is the cheapest correctness check that the
// helper extracted in TD-017 was wired in without changing observable
// behavior for the no-agent path.
func TestAttachAgentBody_NoAgentType(t *testing.T) {
	const prompt = "instruction prompt body"
	got, err := attachAgentBody(prompt, "", map[string]string{"task_id": "T-E01-F01-001"})
	if err != nil {
		t.Fatalf("attachAgentBody returned unexpected error: %v", err)
	}
	if got != prompt {
		t.Errorf("attachAgentBody with empty agentType should return prompt unchanged; got %q want %q", got, prompt)
	}
}

// TestAttachAgentBody_MissingAgentFile verifies that when the data root is
// empty (non-bundle prompt mode, which LoadAgentBodyForInline
// short-circuits on), the prompt also passes through untouched. This
// covers the graceful-degradation branch documented on the helper.
//
// We rely on LoadAgentBodyForInline's documented contract: root=="" or
// agentType=="" returns (_, false). The orchestrator engine's IncludeRoot()
// is "" by default in unit-test context (no project root configured).
func TestAttachAgentBody_GracefulDegradation(t *testing.T) {
	const prompt = "instruction prompt body"
	// agentType is non-empty, but with no orchestrator root configured,
	// LoadAgentBodyForInline returns false and attachAgentBody returns
	// the prompt unchanged. If a future change makes the orchestrator
	// engine resolve agent files in tests, this assertion will need to
	// be updated — but the contract on "graceful degradation when the
	// agent file is unresolvable" must remain.
	got, err := attachAgentBody(prompt, "developer", map[string]string{})
	if err != nil {
		// In unit-test context, root is "" so LoadAgentBodyForInline
		// returns false and attachAgentBody never runs the lint. An
		// error here would mean a future change wired a real agent
		// body in, and the missing vars caused the lint to fire —
		// that's a setup change worth surfacing here rather than
		// silently passing.
		t.Fatalf("attachAgentBody returned unexpected error: %v", err)
	}
	if got != prompt && !strings.Contains(got, prompt) {
		t.Errorf("attachAgentBody should either return the prompt unchanged or prepend an agent body containing it; got %q", got)
	}
}

// ─── maxCascadeDepth guard ────────────────────────────────────────────────────

// TestResolveNext_CascadeDepthGuardFires verifies that resolveNext returns an
// action="error" response (rather than panicking or spinning) when depth
// exceeds maxCascadeDepth. This is the runaway-workflow protection path.
//
// The test calls resolveNext with depth=maxCascadeDepth+1 — one deeper than
// the allowed limit — using a cache backed by stub builders so no real DB
// is needed. The depth guard fires before any adapter lookup, so the stubs
// are never actually called, but they must be installed to satisfy the
// function-variable types.
func TestResolveNext_CascadeDepthGuardFires(t *testing.T) {
	// Install the package-level stub builders used by next_cache_test.go.
	// They are the same types (stubTransitioner / stubPlaceholderGenerator)
	// defined in that file; the depth guard fires before get() is ever
	// called, so the stubs remain untouched.
	_, _, restore := installCountingBuilders(t)
	defer restore()

	actionSvc := &action.MockActionService{}
	cache := &nextAdapterCache{
		entries:       map[string]*nextAdapters{},
		actionSvcRoot: actionSvc,
	}

	// Call at depth = maxCascadeDepth+1: the guard fires immediately.
	resp, err := resolveNext(context.Background(), cache, "task", "E07-F01-001", maxCascadeDepth+1)
	if err != nil {
		t.Fatalf("resolveNext should not return an error for depth-exceeded; got: %v", err)
	}
	if resp.Action != "error" {
		t.Errorf("expected action=%q when cascade depth exceeded, got %q", "error", resp.Action)
	}
	if resp.Error == "" {
		t.Error("expected non-empty Error field when cascade depth exceeded")
	}
	if !strings.Contains(resp.Error, "cascade depth limit") {
		t.Errorf("Error field should mention cascade depth limit; got %q", resp.Error)
	}
}

// ─── applyWireAction: spawn_agent with non-empty prompt ──────────────────────

// TestApplyWireAction_SpawnAgentPopulatesAllFields verifies the AC requirement
// that `action="spawn_agent"` produces a NextResponse with non-empty Prompt,
// AgentType, Provider, and Model — i.e. every field the harness needs to
// actually spawn an agent.
//
// This is the unit-level proof for the acceptance criterion:
//
//	"Golden test or integration test exercises full spawn_agent response with
//	non-empty prompt."
//
// We drive applyWireAction directly so we don't need a real database or
// workflow config. The function is pure given the mock inputs.
func TestApplyWireAction_SpawnAgentPopulatesAllFields(t *testing.T) {
	populated := &action.PopulatedAction{
		Action:      "spawn_agent",
		AgentType:   "developer",
		Provider:    "anthropic",
		Model:       "claude-sonnet-4-5",
		Instruction: "Implement the acceptance criteria for E07-F01-001.",
	}
	nextInfo := &services.NextStatusInfo{
		AvailableTransitions: []services.TransitionInfoWithAction{
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "in_development"}},
		},
	}

	// Build a minimal cache with stub builders; applyWireAction only uses
	// the cache for the "advance_and_recurse" branch, which we don't hit.
	cache := &nextAdapterCache{
		entries:       map[string]*nextAdapters{},
		actionSvcRoot: &action.MockActionService{},
	}
	partialResp := NextResponse{
		EntityKey:  "E07-F01-001",
		EntityType: "task",
		Status:     "ready_for_development",
	}

	got, handled, err := applyWireAction(
		context.Background(),
		cache,
		nextResolutionStrategy(cache),
		"task",
		"E07-F01-001",
		0, // depth
		"spawn_agent",
		populated,
		nextInfo,
		stubTransitioner{},
		partialResp,
	)
	if err != nil {
		t.Fatalf("applyWireAction returned unexpected error: %v", err)
	}
	if handled {
		// spawn_agent is not a terminal branch; caller still runs attachAgentBody.
		t.Error("applyWireAction returned handled=true for spawn_agent; expected false")
	}
	if got.Action != "spawn_agent" {
		t.Errorf("action=%q want %q", got.Action, "spawn_agent")
	}
	if got.AgentType != "developer" {
		t.Errorf("agent_type=%q want %q", got.AgentType, "developer")
	}
	if got.Provider != "anthropic" {
		t.Errorf("provider=%q want %q", got.Provider, "anthropic")
	}
	if got.Model != "claude-sonnet-4-5" {
		t.Errorf("model=%q want %q", got.Model, "claude-sonnet-4-5")
	}
	if got.Prompt == "" {
		t.Error("Prompt must be non-empty for spawn_agent response; got empty string")
	}
	if !strings.Contains(got.Prompt, "E07-F01-001") {
		t.Errorf("Prompt should contain the entity key; got %q", got.Prompt)
	}
}
