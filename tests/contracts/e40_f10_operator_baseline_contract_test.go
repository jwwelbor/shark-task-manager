// TC-078 verifies F10's own schema-ownership contract (REQ-F-018):
// bench/reports/lifecycle-baseline-schema.yaml is the single owner of the
// aggregate.json field inventory, the retention-layout artifact inventory,
// the pilot-attestation field inventory, the refusal-reason vocabulary, the
// noise-band derivation-rule names (including the boundary-inclusivity
// rule), the REQ-F-011 share-partition cell names, the report view names,
// the phase-label values, F10's own digest rules, and the
// forbidden_effort_language / forbidden_composite_fields closed lists.
//
// T-E40-F10-001 implements the valid-fixture half only (schema shape plus
// tests/contracts/testdata/e40_f10/valid/{aggregate,retention-manifest,
// pilot-attestation}.json). T-E40-F10-002 extends this same test function
// with the invalid-fixture matrix (tests/contracts/testdata/e40_f10/invalid)
// per test-plan.md's Caller-Path Contract, which names one shared entrypoint
// (`TestTC078_F10OperatorBaselineContract`) rather than a twin test.
package contracts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type e40F10Schema struct {
	SchemaVersion string `yaml:"schema_version"`

	AggregateTopLevelFields []string          `yaml:"aggregate_top_level_fields"`
	AggregateRequiredFields []string          `yaml:"aggregate_required_fields"`
	AggregateProperties     map[string]string `yaml:"aggregate_properties"`

	RetentionRequiredArtifacts      []string `yaml:"retention_required_artifacts"`
	RetentionOptionalArtifacts      []string `yaml:"retention_optional_artifacts"`
	RetentionManifestRequiredFields []string `yaml:"retention_manifest_required_fields"`

	PilotAttestationRequiredFields []string `yaml:"pilot_attestation_required_fields"`

	RefusalReason     []string       `yaml:"refusal_reason"`
	RefusalExitStatus map[string]int `yaml:"refusal_exit_status"`

	NoiseBandDerivationRule      []string `yaml:"noise_band_derivation_rule"`
	NoiseBandBoundaryInclusivity string   `yaml:"noise_band_boundary_inclusivity"`

	SharePartitionCell     []string `yaml:"share_partition_cell"`
	SharePartitionResidual string   `yaml:"share_partition_residual"`

	ReportView []string `yaml:"report_view"`
	PhaseLabel []string `yaml:"phase_label"`

	DigestRules map[string]string `yaml:"digest_rules"`

	ForbiddenEffortLanguageVersion  int      `yaml:"forbidden_effort_language_version"`
	ForbiddenEffortLanguage         []string `yaml:"forbidden_effort_language"`
	ForbiddenCompositeFieldsVersion int      `yaml:"forbidden_composite_fields_version"`
	ForbiddenCompositeFields        []string `yaml:"forbidden_composite_fields"`

	ProviderAndNetworkBinaries []string `yaml:"provider_and_network_binaries"`
}

func TestTC078_F10OperatorBaselineContract(t *testing.T) {
	// TC-078 Caller-Path Contract: read the committed schema and fixtures
	// through the real filesystem/parser seam; no in-memory record may
	// substitute for them (see test-plan.md's Caller-Path Contracts table).
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	schemaPath := filepath.Join(repoRoot, "bench", "reports", "lifecycle-baseline-schema.yaml")
	testdataRoot := filepath.Join(repoRoot, "tests", "contracts", "testdata", "e40_f10")

	schema := e40F10ReadSchema(t, schemaPath)

	t.Run("schema_owns_required_vocabulary", func(t *testing.T) {
		e40F10AssertSchemaShape(t, schema)
	})

	t.Run("valid_aggregate_fixture", func(t *testing.T) {
		record := e40F10ReadJSONFixture(t, filepath.Join(testdataRoot, "valid", "aggregate.json"))
		if errs := e40F10ValidateRequiredPointers(record, schema.AggregateRequiredFields); len(errs) > 0 {
			t.Fatalf("valid aggregate fixture violates schema:\n%s", strings.Join(errs, "\n"))
		}
		if errs := e40F10ValidateAggregateTypes(schema, record); len(errs) > 0 {
			t.Fatalf("valid aggregate fixture has malformed types:\n%s", strings.Join(errs, "\n"))
		}
		e40F10AssertAggregateSemantics(t, schema, record)
		e40F10AssertReferencesUpstreamVocabularies(t, repoRoot, record)
	})

	t.Run("valid_retention_manifest_fixture", func(t *testing.T) {
		record := e40F10ReadJSONFixture(t, filepath.Join(testdataRoot, "valid", "retention-manifest.json"))
		if errs := e40F10ValidateRequiredPointers(record, schema.RetentionManifestRequiredFields); len(errs) > 0 {
			t.Fatalf("valid retention-manifest fixture violates schema:\n%s", strings.Join(errs, "\n"))
		}
		artifacts, ok := record["artifacts"].(map[string]any)
		if !ok {
			t.Fatal("retention-manifest fixture missing /artifacts object")
		}
		for _, name := range schema.RetentionRequiredArtifacts {
			if name == "manifest.json" {
				continue // the manifest never digests its own not-yet-written bytes
			}
			entry, present := artifacts[name].(map[string]any)
			if !present {
				t.Errorf("retention-manifest fixture missing artifact entry %q", name)
				continue
			}
			digest, _ := entry["sha256"].(string)
			if !isDigest(digest) {
				t.Errorf("retention-manifest fixture artifact %q has malformed sha256 %q", name, digest)
			}
		}
	})

	t.Run("valid_pilot_attestation_fixture", func(t *testing.T) {
		record := e40F10ReadJSONFixture(t, filepath.Join(testdataRoot, "valid", "pilot-attestation.json"))
		if errs := e40F10ValidateRequiredPointers(record, schema.PilotAttestationRequiredFields); len(errs) > 0 {
			t.Fatalf("valid pilot-attestation fixture violates schema:\n%s", strings.Join(errs, "\n"))
		}
		digests, ok := record["inspected_artifact_digests"].(map[string]any)
		if !ok || len(digests) == 0 {
			t.Fatal("pilot-attestation fixture must carry at least one inspected-artifact digest")
		}
		for artifact, value := range digests {
			digest, ok := value.(string)
			if !ok || !isDigest(digest) {
				t.Errorf("pilot-attestation fixture digest for %q is malformed: %v", artifact, value)
			}
		}
	})

	// T-E40-F10-002 appends the invalid-fixture matrix here (missing
	// required field, wrong type, unknown refusal reason, malformed
	// noise-band derivation-rule name, malformed share-partition cell name,
	// unrecognized view name, wrong phase-label value, malformed digest),
	// walking tests/contracts/testdata/e40_f10/invalid/*.json the same way
	// TestTC067_I08LifecycleEvaluationContract walks testdata/e40_i08/invalid.
}

func e40F10ReadSchema(t *testing.T, path string) e40F10Schema {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read F10 schema: %v", err)
	}
	var schema e40F10Schema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		t.Fatalf("parse F10 schema: %v", err)
	}
	return schema
}

func e40F10AssertSchemaShape(t *testing.T, schema e40F10Schema) {
	t.Helper()

	if schema.SchemaVersion == "" {
		t.Error("F10 schema must declare schema_version")
	}

	for _, block := range []string{
		"identity", "scenarios", "time", "cost", "quality",
		"review_value", "artifact_use", "noise_bands", "comparisons", "invalid",
	} {
		if !e40F10ContainsString(schema.AggregateTopLevelFields, block) {
			t.Errorf("F10 schema aggregate_top_level_fields missing %q", block)
		}
	}
	if len(schema.AggregateRequiredFields) == 0 {
		t.Error("F10 schema aggregate_required_fields must not be empty")
	}

	if len(schema.RetentionRequiredArtifacts) != 8 {
		t.Errorf("F10 schema retention_required_artifacts must enumerate all 8 retained artifacts (package.yaml, evidence, transcripts, entity-history.json, lifecycle.jsonl, evaluation.jsonl, oracle.json, manifest.json), got %d", len(schema.RetentionRequiredArtifacts))
	}
	for _, artifact := range []string{
		"package.yaml", "evidence", "transcripts", "entity-history.json",
		"lifecycle.jsonl", "evaluation.jsonl", "oracle.json", "manifest.json",
	} {
		if !e40F10ContainsString(schema.RetentionRequiredArtifacts, artifact) {
			t.Errorf("F10 schema retention_required_artifacts missing %q", artifact)
		}
	}

	for _, field := range []string{
		"/run_reference", "/checklist_results", "/operator_identity", "/inspected_artifact_digests",
	} {
		if !e40F10ContainsString(schema.PilotAttestationRequiredFields, field) {
			t.Errorf("F10 schema pilot_attestation_required_fields missing %q (REQ-F-005)", field)
		}
	}

	if len(schema.RefusalReason) == 0 {
		t.Error("F10 schema refusal_reason vocabulary must not be empty (REQ-F-002/003/005/017)")
	}
	if schema.RefusalExitStatus["spend_gate_refusal"] == 0 {
		t.Error("F10 schema refusal_exit_status.spend_gate_refusal must be a nonzero, distinct exit status")
	}
	if schema.RefusalExitStatus["usage_error"] == schema.RefusalExitStatus["spend_gate_refusal"] {
		t.Error("F10 schema refusal_exit_status must distinguish usage_error from spend_gate_refusal (REQ-F-002)")
	}

	if len(schema.NoiseBandDerivationRule) == 0 {
		t.Error("F10 schema noise_band_derivation_rule must not be empty")
	}
	// AC-T1: the boundary-inclusivity rule must be stated explicitly, not
	// left as test-plan.md TC-088's deferred open question.
	if strings.TrimSpace(schema.NoiseBandBoundaryInclusivity) == "" {
		t.Fatal("F10 schema must state noise_band_boundary_inclusivity (AC-T1 / TC-088 deferred item)")
	}
	lowerBoundary := strings.ToLower(schema.NoiseBandBoundaryInclusivity)
	if !strings.Contains(lowerBoundary, "closed") && !strings.Contains(lowerBoundary, "open") {
		t.Errorf("F10 schema noise_band_boundary_inclusivity must name an open or closed interval rule, got: %s", schema.NoiseBandBoundaryInclusivity)
	}

	wantShares := []string{"pre_code", "review", "rework", "first_pass_code", "wait", "shipping"}
	if len(schema.SharePartitionCell) != len(wantShares) {
		t.Errorf("F10 schema share_partition_cell must name exactly the six REQ-F-011 shares, got %v", schema.SharePartitionCell)
	}
	for _, share := range wantShares {
		if !e40F10ContainsString(schema.SharePartitionCell, share) {
			t.Errorf("F10 schema share_partition_cell missing %q (REQ-F-011)", share)
		}
	}
	if schema.SharePartitionResidual != "unattributed" {
		t.Errorf("F10 schema share_partition_residual must be %q, got %q", "unattributed", schema.SharePartitionResidual)
	}

	for _, view := range []string{"headline", "stage_diagnostic"} {
		if !e40F10ContainsString(schema.ReportView, view) {
			t.Errorf("F10 schema report_view missing %q (REQ-F-009)", view)
		}
	}

	if !e40F10ContainsString(schema.PhaseLabel, "lifecycle_v2") {
		t.Error("F10 schema phase_label missing \"lifecycle_v2\" (REQ-F-017)")
	}

	if schema.DigestRules["algorithm"] != "sha256" || schema.DigestRules["encoding"] != "lowercase_hex" {
		t.Fatalf("F10 schema must own lowercase SHA-256 digest rules: %#v", schema.DigestRules)
	}

	// REQ-F-013: TC-086's exact required literal terms, including the four
	// "non-obvious ones" the test-plan calls out explicitly.
	for _, term := range []string{
		"human minute", "human hour", "human effort", "effort saved",
		"time equivalent", "time saved", "person-hour", "FTE",
	} {
		if !e40F10ContainsCaseInsensitive(schema.ForbiddenEffortLanguage, term) {
			t.Errorf("F10 schema forbidden_effort_language missing required literal term %q (TC-086)", term)
		}
	}
	if schema.ForbiddenEffortLanguageVersion < 1 {
		t.Error("F10 schema forbidden_effort_language_version must be a positive, versioned integer")
	}

	// REQ-F-016: TC-088's exact required literal terms.
	for _, term := range []string{"efficiency", "value_score", "roi", "composite", "weighted_score", "blended"} {
		if !e40F10ContainsCaseInsensitive(schema.ForbiddenCompositeFields, term) {
			t.Errorf("F10 schema forbidden_composite_fields missing required literal term %q (TC-088)", term)
		}
	}
	if schema.ForbiddenCompositeFieldsVersion < 1 {
		t.Error("F10 schema forbidden_composite_fields_version must be a positive, versioned integer")
	}
	// TC-088 target (i): no aggregate property name may itself match a
	// forbidden_composite_fields term (the list's own declaration is
	// excluded, per test-plan.md, since it necessarily names what it
	// forbids -- schema.AggregateProperties never contains that list).
	for pointer := range schema.AggregateProperties {
		lowerPointer := strings.ToLower(pointer)
		for _, term := range schema.ForbiddenCompositeFields {
			if strings.Contains(lowerPointer, strings.ToLower(term)) {
				t.Errorf("F10 schema aggregate_properties pointer %q matches forbidden composite term %q (REQ-F-016)", pointer, term)
			}
		}
	}

	// AC-T2: the enumerated provider/network binary list the PATH-shim
	// denial harness (TC-079/080/090) reuses.
	for _, binary := range []string{"claude", "codex"} {
		if !e40F10ContainsString(schema.ProviderAndNetworkBinaries, binary) {
			t.Errorf("F10 schema provider_and_network_binaries missing %q", binary)
		}
	}
}

// e40F10AssertAggregateSemantics checks the small set of cross-field rules
// that a bare required-pointer walk cannot express: partitions must
// reconcile to lifecycle wall time (REQ-F-010) and the eligibility/outcome
// enums must use values the schema and its referenced upstream vocabularies
// actually define.
func e40F10AssertAggregateSemantics(t *testing.T, schema e40F10Schema, record map[string]any) {
	t.Helper()

	timeBlock, _ := record["time"].(map[string]any)
	wallSeconds, _ := timeBlock["lifecycle_wall_seconds"].(float64)
	if wallSeconds <= 0 {
		t.Fatalf("aggregate fixture /time/lifecycle_wall_seconds must be positive, got %v", timeBlock["lifecycle_wall_seconds"])
	}

	for _, partitionName := range []string{"stage_category", "interval_category", "share_partition"} {
		partition, ok := timeBlock[partitionName].(map[string]any)
		if !ok {
			t.Errorf("aggregate fixture /time/%s must be an object", partitionName)
			continue
		}
		var sum float64
		for _, value := range partition {
			n, ok := value.(float64)
			if !ok {
				t.Errorf("aggregate fixture /time/%s has a non-numeric cell: %v", partitionName, value)
				continue
			}
			sum += n
		}
		if sum != wallSeconds {
			t.Errorf("aggregate fixture /time/%s does not reconcile to lifecycle_wall_seconds: sum=%v want=%v (REQ-F-010/REQ-F-011)", partitionName, sum, wallSeconds)
		}
	}

	shares, _ := timeBlock["share_partition"].(map[string]any)
	for _, share := range schema.SharePartitionCell {
		if _, present := shares[share]; !present {
			t.Errorf("aggregate fixture /time/share_partition missing declared share %q", share)
		}
	}
	if _, present := shares[schema.SharePartitionResidual]; !present {
		t.Errorf("aggregate fixture /time/share_partition missing residual line %q (REQ-F-011: printed even at zero)", schema.SharePartitionResidual)
	}

	scenarios, _ := record["scenarios"].([]any)
	for i, raw := range scenarios {
		scenario, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		eligibility, _ := scenario["eligibility"].(map[string]any)
		if _, ok := eligibility["aggregate_eligible"].(bool); !ok {
			t.Errorf("scenarios[%d]/eligibility/aggregate_eligible must be boolean", i)
		}
		if _, ok := eligibility["publication_eligible"].(bool); !ok {
			t.Errorf("scenarios[%d]/eligibility/publication_eligible must be boolean", i)
		}
	}
}

type e40F10I05VocabSchema struct {
	StageCategory    []string `yaml:"stage_category"`
	IntervalCategory []string `yaml:"interval_category"`
}

type e40F10I08VocabSchema struct {
	TruthResult  []string `yaml:"truth_result"`
	OracleResult []string `yaml:"oracle_result"`
}

// e40F10AssertReferencesUpstreamVocabularies proves the schema header's
// claim -- "an aggregate or report that names a stage/interval category
// value not present in i05-schema.yaml is invalid... enforced by reading
// i05-schema.yaml directly rather than duplicating its list here"
// (REQ-F-018) -- by actually reading bench/evidence/i05-schema.yaml and
// bench/evaluation/i08-schema.yaml and checking the valid fixture's
// stage_category/interval_category/structural/judge/execution_oracle
// values against them. The negative half (a bogus category must fail) is
// T-E40-F10-002's invalid-fixture matrix; this is the positive half T-001
// owns: the reference must actually resolve to real upstream members.
func e40F10AssertReferencesUpstreamVocabularies(t *testing.T, repoRoot string, record map[string]any) {
	t.Helper()

	i05Path := filepath.Join(repoRoot, "bench", "evidence", "i05-schema.yaml")
	i05Data, err := os.ReadFile(i05Path)
	if err != nil {
		t.Fatalf("read I-05 schema (REQ-F-018 reference target): %v", err)
	}
	var i05 e40F10I05VocabSchema
	if err := yaml.Unmarshal(i05Data, &i05); err != nil {
		t.Fatalf("parse I-05 schema: %v", err)
	}
	if len(i05.StageCategory) == 0 || len(i05.IntervalCategory) == 0 {
		t.Fatal("I-05 schema stage_category/interval_category must not be empty")
	}

	i08Path := filepath.Join(repoRoot, "bench", "evaluation", "i08-schema.yaml")
	i08Data, err := os.ReadFile(i08Path)
	if err != nil {
		t.Fatalf("read I-08 schema (REQ-F-018 reference target): %v", err)
	}
	var i08 e40F10I08VocabSchema
	if err := yaml.Unmarshal(i08Data, &i08); err != nil {
		t.Fatalf("parse I-08 schema: %v", err)
	}
	if len(i08.TruthResult) == 0 || len(i08.OracleResult) == 0 {
		t.Fatal("I-08 schema truth_result/oracle_result must not be empty")
	}

	timeBlock, _ := record["time"].(map[string]any)
	costBlock, _ := record["cost"].(map[string]any)
	for _, block := range []map[string]any{timeBlock, costBlock} {
		e40F10AssertCategoryKeysKnown(t, block, "stage_category", i05.StageCategory)
		e40F10AssertCategoryKeysKnown(t, block, "interval_category", i05.IntervalCategory)
	}

	quality, _ := record["quality"].(map[string]any)
	byScenario, _ := quality["by_scenario"].([]any)
	for i, raw := range byScenario {
		scenario, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		for _, field := range []string{"structural", "judge"} {
			value, _ := scenario[field].(string)
			if !e40F10ContainsString(i08.TruthResult, value) {
				t.Errorf("quality/by_scenario[%d]/%s value %q is not a member of I-08 truth_result %v", i, field, value, i08.TruthResult)
			}
		}
		oracle, _ := scenario["execution_oracle"].(string)
		if !e40F10ContainsString(i08.OracleResult, oracle) {
			t.Errorf("quality/by_scenario[%d]/execution_oracle value %q is not a member of I-08 oracle_result %v", i, oracle, i08.OracleResult)
		}
	}
}

func e40F10AssertCategoryKeysKnown(t *testing.T, block map[string]any, partitionName string, known []string) {
	t.Helper()
	partition, ok := block[partitionName].(map[string]any)
	if !ok {
		return
	}
	for key := range partition {
		if key == "unattributed" {
			continue // the required residual line, not a category member
		}
		if !e40F10ContainsString(known, key) {
			t.Errorf("%s cell %q is not a member of the referenced I-05 vocabulary %v (REQ-F-018: F10 references, never restates)", partitionName, key, known)
		}
	}
}

// e40F10ValidateAggregateTypes applies the discriminating type hints the
// schema's aggregate_properties block declares (object/array/boolean/
// digest), the same non-exhaustive-hint pattern the schema's own comment
// documents. Pointers containing a "[]" segment (e.g.
// /scenarios[]/eligibility/aggregate_eligible) are resolved against every
// array element via e40F10ResolvePointerValues, so a type hint under an
// array is enforced per-element rather than silently skipped.
func e40F10ValidateAggregateTypes(schema e40F10Schema, record map[string]any) []string {
	var errs []string
	for pointer, kind := range schema.AggregateProperties {
		segments := e40F10SplitPointer(pointer)
		matches := e40F10ResolvePointerValues(record, segments, "")
		if len(matches) == 0 {
			// Presence is already enforced by aggregate_required_fields;
			// a hint-only pointer absent here would be a required-field
			// failure caught elsewhere, so skip rather than double-report.
			continue
		}
		for _, match := range matches {
			errs = append(errs, e40F10CheckType(match.Path, kind, match.Value)...)
		}
	}
	return errs
}

func e40F10CheckType(pointer, kind string, value any) []string {
	switch kind {
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return []string{fmt.Sprintf("%s: expected object, got %T", pointer, value)}
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return []string{fmt.Sprintf("%s: expected array, got %T", pointer, value)}
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return []string{fmt.Sprintf("%s: expected boolean, got %T", pointer, value)}
		}
	case "string":
		if _, ok := value.(string); !ok {
			return []string{fmt.Sprintf("%s: expected string, got %T", pointer, value)}
		}
	case "integer":
		n, ok := value.(float64)
		if !ok || n != float64(int64(n)) {
			return []string{fmt.Sprintf("%s: expected integer, got %v", pointer, value)}
		}
	case "digest":
		s, ok := value.(string)
		if !ok || !isDigest(s) {
			return []string{fmt.Sprintf("%s: expected lowercase sha256 hex digest, got %v", pointer, value)}
		}
	}
	return nil
}

// e40F10ResolvedValue is one concrete value a pointer resolved to (a
// "[]" segment can fan a single pointer out to many concrete values, one
// per array element).
type e40F10ResolvedValue struct {
	Value any
	Path  string
}

// e40F10ResolvePointerValues walks segments against current the same way
// e40F10WalkPointerSegments does, but instead of reporting presence errors
// it collects every concrete value the pointer resolves to (one per array
// element for each "[]" segment traversed), each tagged with its own
// concrete (index-expanded) path for error reporting. A pointer segment
// that cannot be resolved (missing key, wrong container type, empty
// array) yields zero results rather than an error -- presence is a
// separate concern owned by e40F10ValidateRequiredPointers.
func e40F10ResolvePointerValues(current any, segments []string, path string) []e40F10ResolvedValue {
	if len(segments) == 0 {
		return []e40F10ResolvedValue{{Value: current, Path: path}}
	}
	segment := segments[0]
	rest := segments[1:]

	if strings.HasSuffix(segment, "[]") {
		key := strings.TrimSuffix(segment, "[]")
		obj, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		raw, present := obj[key]
		if !present {
			return nil
		}
		list, ok := raw.([]any)
		if !ok {
			return nil
		}
		var results []e40F10ResolvedValue
		for i, elem := range list {
			results = append(results, e40F10ResolvePointerValues(elem, rest, fmt.Sprintf("%s/%s[%d]", path, key, i))...)
		}
		return results
	}

	obj, ok := current.(map[string]any)
	if !ok {
		return nil
	}
	value, present := obj[segment]
	if !present {
		return nil
	}
	return e40F10ResolvePointerValues(value, rest, path+"/"+segment)
}

// e40F10ValidateRequiredPointers walks each JSON-Pointer-style path in
// pointers against record, honoring the schema's own "[]" convention
// ("for every element of this array" -- see lifecycle-baseline-schema.yaml's
// header comment) by recursing into every array element for the remainder
// of the pointer.
func e40F10ValidateRequiredPointers(record map[string]any, pointers []string) []string {
	var errs []string
	for _, pointer := range pointers {
		segments := e40F10SplitPointer(pointer)
		errs = append(errs, e40F10WalkPointerSegments(record, segments, "")...)
	}
	return errs
}

func e40F10SplitPointer(pointer string) []string {
	trimmed := strings.Trim(pointer, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func e40F10WalkPointerSegments(current any, segments []string, path string) []string {
	if len(segments) == 0 {
		if current == nil {
			return []string{path + ": required field is nil"}
		}
		return nil
	}
	segment := segments[0]
	rest := segments[1:]

	if strings.HasSuffix(segment, "[]") {
		key := strings.TrimSuffix(segment, "[]")
		obj, ok := current.(map[string]any)
		if !ok {
			return []string{fmt.Sprintf("%s/%s: expected object to hold array %q", path, segment, key)}
		}
		raw, present := obj[key]
		if !present || raw == nil {
			return []string{fmt.Sprintf("%s/%s: required array missing", path, key)}
		}
		list, ok := raw.([]any)
		if !ok {
			return []string{fmt.Sprintf("%s/%s: expected array, got %T", path, key, raw)}
		}
		var errs []string
		for i, elem := range list {
			errs = append(errs, e40F10WalkPointerSegments(elem, rest, fmt.Sprintf("%s/%s[%d]", path, key, i))...)
		}
		return errs
	}

	obj, ok := current.(map[string]any)
	if !ok {
		return []string{fmt.Sprintf("%s/%s: expected object", path, segment)}
	}
	value, present := obj[segment]
	if !present || value == nil {
		return []string{fmt.Sprintf("%s/%s: required field missing", path, segment)}
	}
	return e40F10WalkPointerSegments(value, rest, path+"/"+segment)
}

func e40F10ReadJSONFixture(t *testing.T, path string) map[string]any {
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

func e40F10ContainsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func e40F10ContainsCaseInsensitive(values []string, want string) bool {
	lowerWant := strings.ToLower(want)
	for _, value := range values {
		if strings.ToLower(value) == lowerWant {
			return true
		}
	}
	return false
}
