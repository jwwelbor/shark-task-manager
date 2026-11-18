package contracts

import (
	"strings"
	"testing"
)

func TestB077CodexNativeRouteRequiresVerifiedMappingAndIdentity(t *testing.T) {
	run := readF09RepositoryFile(t, "skills/shark-rider/verbs/run.md")
	contract := readF09RepositoryFile(t, "skills/shark-rider/context/host-adapter-contract.md")
	codexProvider := readF09RepositoryFile(t, "skills/shark-attack/providers/codex.md")

	preflight := strings.Index(run, "### Route attestation pre-flight (B077)")
	claim := strings.Index(run, "Claim the concrete entity returned by Shark")
	spawn := strings.Index(run, "Spawn the host worker using a host-safe adapter")
	if preflight < 0 || claim < 0 || spawn < 0 || preflight > claim || preflight > spawn {
		t.Fatal("B077: route attestation must run before claim and worker spawn")
	}
	routePreflight := strings.ToLower(strings.Join(strings.Fields(run[preflight:claim]), " "))

	for _, required := range []string{
		"response.provider` is `codex",
		"native codex",
		"explicit model-selection mapping",
		"effective provider/model identity",
		"do not spawn",
		"blocked_external",
		"do not infer the model from `response.agent_type`, host worker class, or `response.effort`",
		"response.prompt` byte-for-byte unchanged",
	} {
		if !strings.Contains(routePreflight, required) {
			t.Errorf("B077: route pre-flight must contain %q", required)
		}
	}

	contractText := strings.ToLower(strings.Join(strings.Fields(contract), " "))
	for _, required := range []string{
		"route_status",
		"requested_provider",
		"requested_model",
		"host_worker_class",
		"effort",
		"effective_provider",
		"effective_model",
		"worker_id",
		"session_id",
		"independently read from host session metadata",
		"ephemeral dispatch result",
		"durable provenance",
	} {
		if !strings.Contains(contractText, required) {
			t.Errorf("B077: host-adapter contract must declare %q", required)
		}
	}

	if !strings.Contains(codexProvider, "Scope: OpenAI Codex CLI only") {
		t.Fatal("B077: native Codex routing must not treat Codex CLI evidence as a native host mapping")
	}
}
