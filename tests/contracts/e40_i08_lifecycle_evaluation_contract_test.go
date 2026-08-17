// Package contracts verifies the I-08 lifecycle evaluation record contract.
// TC-067 is the shared contract proof consumed by E40-F10.
package contracts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type e40I08Schema struct {
	SchemaVersion      string              `yaml:"schema_version"`
	TopLevelFields     []string            `yaml:"top_level_fields"`
	RequiredFields     []string            `yaml:"required_fields"`
	TruthResult        []string            `yaml:"truth_result"`
	CheckApplicability []string            `yaml:"check_applicability"`
	InvalidityReasons  []string            `yaml:"invalidity_reason"`
	DigestRules        map[string]string   `yaml:"digest_rules"`
	Properties         map[string]string   `yaml:"properties"`
	NestedVocabularies map[string][]string `yaml:"vocabularies"`
}

func TestTC067_I08LifecycleEvaluationContract(t *testing.T) {
	// TC-067: read the committed schema and fixtures through the real
	// filesystem/parser seam; no in-memory record may substitute for them.
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	schemaPath := filepath.Join(repoRoot, "bench", "evaluation", "i08-schema.yaml")
	testdataRoot := filepath.Join(repoRoot, "tests", "contracts", "testdata", "e40_i08")

	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read I-08 schema: %v", err)
	}
	var schema e40I08Schema
	if err := yaml.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("parse I-08 schema: %v", err)
	}
	if schema.SchemaVersion == "" {
		t.Fatal("I-08 schema must declare schema_version")
	}
	for _, field := range []string{
		"identity", "source_artifacts", "structural", "judge", "execution_oracle", "eligibility",
	} {
		if !e40I08ContainsString(schema.TopLevelFields, field) {
			t.Errorf("I-08 schema top_level_fields missing %q", field)
		}
	}
	for _, field := range []string{
		"/schema_version", "/evaluation_id", "/identity", "/source_artifacts", "/structural",
		"/judge", "/execution_oracle", "/eligibility",
	} {
		if !e40I08ContainsString(schema.RequiredFields, field) {
			t.Errorf("I-08 schema required_fields missing %q", field)
		}
	}
	if schema.DigestRules["algorithm"] != "sha256" || schema.DigestRules["encoding"] != "lowercase_hex" {
		t.Fatalf("I-08 schema must own lowercase SHA-256 digest rules: %#v", schema.DigestRules)
	}

	// TC-067 staged-edge proof: the shared contract test reads the committed
	// map/test-plan metadata and live Shark counterpart status. This prevents a
	// schema-only green result from hiding a drifted integration edge.
	mapPath := filepath.Join(repoRoot, "docs", "plan", "E40-shark-bench-workflow-benchmarking-harness", "E40-interaction-map.md")
	testPlanPath := filepath.Join(repoRoot, "docs", "plan", "E40-shark-bench-workflow-benchmarking-harness", "E40-F09-calibrated-evaluation-and-comparison-identity", "test-plan.md")
	mapText := string(readContractFile(t, mapPath))
	planText := string(readContractFile(t, testPlanPath))
	for _, row := range []string{"I-05", "I-07", "I-08"} {
		if !strings.Contains(mapText, row) || !strings.Contains(planText, row) {
			t.Errorf("TC-067 staged edge %s missing from interaction map or test plan", row)
		}
	}
	for _, field := range []string{"contract-only", "activation owners", "closure keys", "counterpart status", "review basis", "pending-integration"} {
		if !strings.Contains(planText, field) {
			t.Errorf("TC-067 staged edge metadata missing %q", field)
		}
	}
	for _, key := range []string{"E40-F06", "E40-F08", "E40-F10"} {
		cmd := exec.Command(filepath.Join(repoRoot, "bin", "shark"), "get", key, "--json")
		cmd.Dir = repoRoot
		if output, err := cmd.Output(); err != nil || len(output) == 0 {
			t.Errorf("TC-067 live counterpart status read failed for %s: %v", key, err)
		}
	}
	for name, values := range map[string][]string{
		"truth_result":        schema.TruthResult,
		"check_applicability": schema.CheckApplicability,
		"invalidity_reason":   schema.InvalidityReasons,
	} {
		if len(values) == 0 {
			t.Errorf("I-08 schema vocabulary %s is empty", name)
		}
	}

	validPath := filepath.Join(testdataRoot, "valid", "eligible.json")
	validRecord := readI08Record(t, validPath)
	if errs := validateI08Record(schema, validRecord); len(errs) > 0 {
		t.Fatalf("valid I-08 fixture violates schema:\n%s", strings.Join(errs, "\n"))
	}

	invalidEntries, err := os.ReadDir(filepath.Join(testdataRoot, "invalid"))
	if err != nil {
		t.Fatalf("read invalid I-08 fixtures: %v", err)
	}
	if len(invalidEntries) == 0 {
		t.Fatal("I-08 invalid fixture directory must not be empty")
	}
	for _, entry := range invalidEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		entry := entry
		t.Run(entry.Name(), func(t *testing.T) {
			record := readI08Record(t, filepath.Join(testdataRoot, "invalid", entry.Name()))
			errs := validateI08Record(schema, record)
			if len(errs) == 0 {
				t.Fatalf("invalid fixture %s unexpectedly passed", entry.Name())
			}
			joined := strings.Join(errs, "\n")
			if !strings.Contains(joined, "invalidity") && !strings.Contains(joined, "required") && !strings.Contains(joined, "malformed") {
				t.Fatalf("invalid fixture %s lacks a named path/reason: %v", entry.Name(), errs)
			}
		})
	}
}

func readContractFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read staged contract metadata %s: %v", path, err)
	}
	return data
}

func readI08Record(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("parse fixture %s: %v", path, err)
	}
	return record
}

func validateI08Record(schema e40I08Schema, record map[string]any) []string {
	var errs []string
	allowed := make(map[string]bool, len(schema.TopLevelFields)+2)
	allowed["schema_version"] = true
	allowed["evaluation_id"] = true
	for _, field := range schema.TopLevelFields {
		allowed[field] = true
	}
	for field := range record {
		if !allowed[field] {
			errs = append(errs, fmt.Sprintf("/%s: unknown field", field))
		}
	}
	for _, pointer := range schema.RequiredFields {
		if pointer == "/schema_version" || pointer == "/evaluation_id" {
			continue
		}
		value, ok := e40I08Pointer(record, pointer)
		if !ok || value == nil {
			errs = append(errs, fmt.Sprintf("%s: required field missing", pointer))
		}
	}
	if version, ok := record["schema_version"].(string); !ok || version != schema.SchemaVersion {
		errs = append(errs, "/schema_version: unsupported or malformed value")
	}
	if id, ok := record["evaluation_id"].(string); !ok || strings.TrimSpace(id) == "" {
		errs = append(errs, "/evaluation_id: required string missing")
	}
	for _, block := range []string{"structural", "judge", "execution_oracle", "eligibility"} {
		if value, ok := record[block].(map[string]any); !ok || value == nil {
			errs = append(errs, fmt.Sprintf("/%s: truth/eligibility block missing", block))
		}
	}
	identity, ok := record["identity"].(map[string]any)
	if !ok {
		errs = append(errs, "/identity: object required")
	} else {
		for _, field := range []string{"run_id", "scenario_id", "scenario_version", "fixture_id", "adapter_id", "adapter_version", "shark_binary_digest", "shark_content_digest"} {
			if value, present := identity[field]; !present || value == nil {
				errs = append(errs, fmt.Sprintf("/identity/%s: required", field))
			}
		}
		for _, field := range []string{"fixture_digest", "shark_binary_digest", "shark_content_digest"} {
			if digest, present := identity[field]; present {
				if value, ok := digest.(string); !ok || !isDigest(value) {
					errs = append(errs, fmt.Sprintf("/identity/%s: malformed digest", field))
				}
			}
		}
	}
	eligibility, ok := record["eligibility"].(map[string]any)
	if ok {
		if value, present := eligibility["aggregate_eligible"]; !present || value == nil {
			errs = append(errs, "/eligibility/aggregate_eligible: required")
		}
		if reasons, present := eligibility["invalidity_reasons"]; present {
			if list, ok := reasons.([]any); !ok {
				errs = append(errs, "/eligibility/invalidity_reasons: array required")
			} else {
				for i, reason := range list {
					if item, ok := reason.(map[string]any); !ok || item["code"] == nil {
						errs = append(errs, fmt.Sprintf("/eligibility/invalidity_reasons/%d: named invalidity required", i))
					}
				}
			}
		}
	}
	return errs
}

func e40I08Pointer(record map[string]any, pointer string) (any, bool) {
	var current any = record
	for _, segment := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[segment]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func isDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == strings.ToLower(value)
}

func e40I08ContainsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
