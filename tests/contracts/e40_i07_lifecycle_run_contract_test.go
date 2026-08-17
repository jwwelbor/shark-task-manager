// Package contracts verifies the I-07 lifecycle run contract consumed by
// E40-F09 and E40-F10.
package contracts

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type e40I07Schema struct {
	SchemaVersion   string            `yaml:"schema_version"`
	RequiredFields  []string          `yaml:"required_fields"`
	TerminalOutcome []string          `yaml:"terminal_outcome"`
	DispatchOutcome []string          `yaml:"dispatch_outcome"`
	GateState       []string          `yaml:"gate_state"`
	DigestRules     map[string]string `yaml:"digest_rules"`
	Properties      map[string]any    `yaml:"properties"`
}

func TestTC061_I07LifecycleRunContract(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	schemaPath := filepath.Join(repoRoot, "bench", "runs", "i07-schema.yaml")
	testdataRoot := filepath.Join(repoRoot, "tests", "contracts", "testdata", "e40_i07")

	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read I-07 schema: %v", err)
	}
	var schema e40I07Schema
	if err := yaml.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("parse I-07 schema: %v", err)
	}
	if schema.SchemaVersion == "" {
		t.Fatal("i07-schema.yaml must declare schema_version")
	}
	for _, field := range []string{"/identity/run_id", "/entity_graph/root_key", "/dispatches", "/stages", "/workflow_policy/enabled_gates", "/review_gates", "/questions", "/limits/max_cost_usd", "/outcome/terminal"} {
		if !containsString(schema.RequiredFields, field) {
			t.Errorf("i07-schema.yaml required_fields missing %q", field)
		}
	}
	for name, values := range map[string][]string{
		"terminal_outcome": schema.TerminalOutcome,
		"dispatch_outcome": schema.DispatchOutcome,
		"gate_state":       schema.GateState,
	} {
		if len(values) == 0 {
			t.Errorf("i07-schema.yaml %s vocabulary is empty", name)
		}
	}
	if schema.DigestRules["algorithm"] != "sha256" {
		t.Errorf("i07-schema.yaml digest_rules.algorithm = %q, want sha256", schema.DigestRules["algorithm"])
	}

	t.Run("valid_fixture", func(t *testing.T) {
		assertValidatorResult(t, repoRoot, filepath.Join(testdataRoot, "valid", "complete.jsonl"), true, "")
	})

	validFixture := filepath.Join(testdataRoot, "valid", "complete.jsonl")
	cases := []struct {
		name   string
		want   string
		mutate func(map[string]any)
	}{
		{"malformed_field", "/identity/run_id", func(record map[string]any) { record["identity"].(map[string]any)["run_id"] = nil }},
		{"unexpected_field", "unexpected", func(record map[string]any) { record["unexpected"] = true }},
		{"unsupported_outcome", "terminal", func(record map[string]any) { record["outcome"].(map[string]any)["terminal"] = "unknown" }},
		{"duplicate_ordinal", "ordinal", func(record map[string]any) {
			dispatches := record["dispatches"].([]any)
			record["dispatches"] = append(dispatches, dispatches[0])
		}},
		{"missing_stop_reason", "/outcome/reason", func(record map[string]any) {
			outcome := record["outcome"].(map[string]any)
			outcome["terminal"] = "resource_limit"
			outcome["reason"] = ""
			outcome["publication_eligible"] = false
		}},
		{"eligibility_conflict", "publication_eligible", func(record map[string]any) { record["outcome"].(map[string]any)["publication_eligible"] = false }},
		{"identity_mismatch", "identity", func(record map[string]any) {
			record["stages"].([]any)[0].(map[string]any)["candidate"].(map[string]any)["identity_digest"] = strings.Repeat("0", 64)
		}},
		{"missing_model_usage", "usage", func(record map[string]any) {
			record["stages"].([]any)[0].(map[string]any)["usage"].(map[string]any)["model"] = ""
		}},
		{"prompt_digest_mismatch", "prompt_sha256", func(record map[string]any) {
			record["dispatches"].([]any)[0].(map[string]any)["evidence_refs"].(map[string]any)["prompt_sha256"] = strings.Repeat("e", 64)
		}},
		{"candidate_snapshot_mismatch", "candidate", func(record map[string]any) {
			record["stages"].([]any)[0].(map[string]any)["evidence_refs"].(map[string]any)["candidate_snapshot_digest"] = strings.Repeat("0", 64)
		}},
		{"missing_artifact_consumption", "consumption", func(record map[string]any) {
			delete(record["stages"].([]any)[0].(map[string]any)["artifacts"].([]any)[0].(map[string]any), "consumers")
		}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			fixture := writeMutationFixture(t, validFixture, tc.mutate)
			assertValidatorResult(t, repoRoot, fixture, false, tc.want)
		})
	}
}

func writeMutationFixture(t *testing.T, source string, mutate func(map[string]any)) string {
	t.Helper()
	contents, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read valid fixture for mutation: %v", err)
	}
	var record map[string]any
	if err := json.Unmarshal(contents, &record); err != nil {
		t.Fatalf("decode valid fixture for mutation: %v", err)
	}
	mutate(record)
	mutated, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("encode mutated fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), "mutation.jsonl")
	if err := os.WriteFile(path, append(mutated, '\n'), 0o600); err != nil {
		t.Fatalf("write mutated fixture: %v", err)
	}
	return path
}

func assertValidatorResult(t *testing.T, repoRoot, fixture string, wantValid bool, wantMessage string) {
	t.Helper()
	schema := filepath.Join(repoRoot, "bench", "runs", "i07-schema.yaml")
	cmd := exec.Command(filepath.Join(repoRoot, "bench", "scripts", "verify-lifecycle-run.sh"), fixture, "--schema", schema)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if wantValid {
		if err != nil {
			t.Fatalf("validator rejected valid fixture: %v\nstderr: %s", err, stderr.String())
		}
		var verdict map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &verdict); err != nil {
			t.Fatalf("validator output is not JSON: %v\nstdout: %s", err, stdout.String())
		}
		if verdict["result"] != "accepted" {
			t.Fatalf("validator result = %v, want accepted", verdict["result"])
		}
		return
	}
	if err == nil {
		t.Fatalf("validator accepted invalid fixture; stdout: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), wantMessage) {
		t.Fatalf("validator diagnostic %q does not name %q", stderr.String(), wantMessage)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
