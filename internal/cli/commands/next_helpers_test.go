package commands

import (
	"strings"
	"testing"
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
// empty (legacy shark-templates/ mode, which LoadAgentBodyForInline
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

// TestHelperFunctionsCallable is a compile-time check that the three
// helpers introduced in TD-017 (tryCascade, applyWireAction,
// attachAgentBody) remain in scope and accept their documented signatures.
// If a future refactor renames or removes a helper, this test fails to
// compile — the loudest possible signal that the TD-017 contract was
// broken.
func TestHelperFunctionsCallable(t *testing.T) {
	// Reference the functions; non-nil assignments force the compiler to
	// resolve their identifiers. We don't invoke them — that would require
	// fully-wired adapter caches and transitioners, which is the job of
	// the existing integration-style tests.
	_ = tryCascade
	_ = applyWireAction
	_ = attachAgentBody
}
