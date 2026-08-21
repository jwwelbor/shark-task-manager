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
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type e40F10Schema struct {
	SchemaVersion string `yaml:"schema_version"`

	AggregateTopLevelFields           []string          `yaml:"aggregate_top_level_fields"`
	AggregateRequiredFields           []string          `yaml:"aggregate_required_fields"`
	AggregateProperties               map[string]string `yaml:"aggregate_properties"`
	AggregateRequiredArraysMayBeEmpty []string          `yaml:"aggregate_required_arrays_may_be_empty"`

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
	i05Vocab, i08Vocab := e40F10ReadUpstreamVocab(t, repoRoot)

	t.Run("schema_owns_required_vocabulary", func(t *testing.T) {
		e40F10AssertSchemaShape(t, schema)
		e40F10AssertSchemaReferencesNeverRestates(t, schemaPath)
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

		// T-E40-F10-002: the combined validator every invalid-fixture case
		// below is also run through must accept the valid fixture cleanly,
		// or the invalid-fixture matrix would be proving nothing.
		if errs := e40F10ValidateAggregateRecord(schema, i05Vocab, i08Vocab, record); len(errs) > 0 {
			t.Fatalf("valid aggregate fixture fails the T-E40-F10-002 combined validator:\n%s", strings.Join(errs, "\n"))
		}
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

		if errs := e40F10ValidateRetentionManifestRecord(schema, record); len(errs) > 0 {
			t.Fatalf("valid retention-manifest fixture fails the T-E40-F10-002 combined validator:\n%s", strings.Join(errs, "\n"))
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

		if errs := e40F10ValidatePilotAttestationRecord(schema, record); len(errs) > 0 {
			t.Fatalf("valid pilot-attestation fixture fails the T-E40-F10-002 combined validator:\n%s", strings.Join(errs, "\n"))
		}
	})

	// T-E40-F10-002: a minimal refusal-reason vocabulary fixture. This is
	// NOT the full batch.json refusal-record shape (that driver is out of
	// scope for T-002 -- see spec.md's run-lifecycle-batch.sh /
	// spend-gate.sh component rows); it exists solely to prove
	// refusal_reason is schema-owned and closed (REQ-F-002/003/005/017,
	// AC-001), matching the Architecture component-changes row's explicit
	// "share partition, and refusal-reason cases" testdata note.
	t.Run("valid_refusal_reason_fixture", func(t *testing.T) {
		record := e40F10ReadJSONFixture(t, filepath.Join(testdataRoot, "valid", "refusal.json"))
		if errs := e40F10ValidateRefusalRecord(schema, record); len(errs) > 0 {
			t.Fatalf("valid refusal-reason fixture violates schema:\n%s", strings.Join(errs, "\n"))
		}
	})

	// T-E40-F10-002 AC-T1/AC-T2 and TC-078's exhaustive invalid-fixture
	// matrix. Each map below was generated once from the valid base
	// fixtures (one mutation per entry) and is walked here the same way
	// TestTC067_I08LifecycleEvaluationContract walks testdata/e40_i08/invalid,
	// except every entry also asserts the *specific* failing JSON path
	// named in test-plan.md's TC-078 "Notes for Agent" requirement, not
	// just a nonzero error count.
	t.Run("invalid_aggregate_fixtures", func(t *testing.T) {
		e40F10RunInvalidFixtureMatrix(t, filepath.Join(testdataRoot, "invalid"), e40F10InvalidAggregateWantPath,
			func(record map[string]any) []string {
				return e40F10ValidateAggregateRecord(schema, i05Vocab, i08Vocab, record)
			})
	})

	t.Run("invalid_retention_manifest_fixtures", func(t *testing.T) {
		e40F10RunInvalidFixtureMatrix(t, filepath.Join(testdataRoot, "invalid"), e40F10InvalidRetentionManifestWantPath,
			func(record map[string]any) []string {
				return e40F10ValidateRetentionManifestRecord(schema, record)
			})
	})

	t.Run("invalid_pilot_attestation_fixtures", func(t *testing.T) {
		e40F10RunInvalidFixtureMatrix(t, filepath.Join(testdataRoot, "invalid"), e40F10InvalidPilotAttestationWantPath,
			func(record map[string]any) []string {
				return e40F10ValidatePilotAttestationRecord(schema, record)
			})
	})

	t.Run("invalid_refusal_reason_fixtures", func(t *testing.T) {
		e40F10RunInvalidFixtureMatrix(t, filepath.Join(testdataRoot, "invalid"), e40F10InvalidRefusalWantPath,
			func(record map[string]any) []string {
				return e40F10ValidateRefusalRecord(schema, record)
			})
	})

	// Coverage check: every *.json committed under invalid/ must be
	// exercised by exactly one of the four want-path tables above, and
	// every table entry must name a file that actually exists on disk --
	// otherwise a fixture could silently stop being tested (or a table
	// entry could silently reference a deleted file) without any test
	// failing.
	t.Run("invalid_fixture_directory_matches_tables", func(t *testing.T) {
		e40F10AssertInvalidDirectoryCoverage(t, filepath.Join(testdataRoot, "invalid"),
			e40F10InvalidAggregateWantPath, e40F10InvalidRetentionManifestWantPath,
			e40F10InvalidPilotAttestationWantPath, e40F10InvalidRefusalWantPath)
	})
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

	// T-E40-F10-002 schema gap: a required top-level array (e.g. /scenarios)
	// can be present-but-empty and pass every other pointer/type check,
	// since an empty array has no elements to walk. /comparisons and
	// /invalid are the only two blocks this schema documents as
	// legitimately empty; every other required array MUST be enforced
	// non-empty by a validator, and this list is how a validator knows
	// which two are exempt rather than hard-coding the exemption.
	if len(schema.AggregateRequiredArraysMayBeEmpty) == 0 {
		t.Fatal("F10 schema aggregate_required_arrays_may_be_empty must not be empty")
	}
	for _, exempt := range []string{"/comparisons", "/invalid"} {
		if !e40F10ContainsString(schema.AggregateRequiredArraysMayBeEmpty, exempt) {
			t.Errorf("F10 schema aggregate_required_arrays_may_be_empty missing %q", exempt)
		}
	}
	for _, mustNotBeExempt := range []string{"/scenarios", "/noise_bands"} {
		if e40F10ContainsString(schema.AggregateRequiredArraysMayBeEmpty, mustNotBeExempt) {
			t.Errorf("F10 schema aggregate_required_arrays_may_be_empty must not exempt %q (REQ-F-007)", mustNotBeExempt)
		}
	}
}

// e40F10AssertSchemaReferencesNeverRestates proves the schema header's
// "references, and deliberately does NOT restate" claim structurally: the
// committed YAML file itself must carry no top-level stage_category,
// interval_category, or invalidity_reason key of its own (those belong to
// i05-schema.yaml / i08-schema.yaml only, per REQ-F-018).
func e40F10AssertSchemaReferencesNeverRestates(t *testing.T, schemaPath string) {
	t.Helper()
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read F10 schema: %v", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parse F10 schema: %v", err)
	}
	for _, restatedKey := range []string{"stage_category", "interval_category", "invalidity_reason"} {
		if _, present := raw[restatedKey]; present {
			t.Errorf("F10 schema must not restate upstream vocabulary key %q (REQ-F-018: reference i05/i08, never restate)", restatedKey)
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

// e40F10ReadUpstreamVocab reads bench/evidence/i05-schema.yaml and
// bench/evaluation/i08-schema.yaml -- the two upstream vocabularies F10's
// own schema deliberately references and never restates (REQ-F-018). Shared
// by T-001's positive-path assertion and T-002's invalid-fixture matrix so
// both read the same real files through the same seam rather than each
// re-implementing the read.
func e40F10ReadUpstreamVocab(t *testing.T, repoRoot string) (e40F10I05VocabSchema, e40F10I08VocabSchema) {
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
	return i05, i08
}

// e40F10AssertReferencesUpstreamVocabularies proves the schema header's
// claim -- "an aggregate or report that names a stage/interval category
// value not present in i05-schema.yaml is invalid... enforced by reading
// i05-schema.yaml directly rather than duplicating its list here"
// (REQ-F-018) -- against the valid fixture. This is the positive half T-001
// owns: the reference must actually resolve to real upstream members. The
// negative half (a bogus category must fail) is T-E40-F10-002's
// invalid_aggregate_fixtures matrix, which calls the same pure
// e40F10UpstreamVocabViolations this wraps.
func e40F10AssertReferencesUpstreamVocabularies(t *testing.T, repoRoot string, record map[string]any) {
	t.Helper()
	i05, i08 := e40F10ReadUpstreamVocab(t, repoRoot)
	for _, msg := range e40F10UpstreamVocabViolations(i05, i08, record) {
		t.Error(msg)
	}
}

// e40F10UpstreamVocabViolations is the pure check both
// e40F10AssertReferencesUpstreamVocabularies (valid fixture, expects zero
// results) and T-E40-F10-002's AC-T2 invalid-fixture cases (expect a
// specific named path) drive. Every returned string starts with the exact
// failing JSON pointer, per TC-078's "Notes for Agent" requirement.
func e40F10UpstreamVocabViolations(i05 e40F10I05VocabSchema, i08 e40F10I08VocabSchema, record map[string]any) []string {
	var errs []string

	for _, base := range []string{"time", "cost"} {
		block, _ := record[base].(map[string]any)
		if block == nil {
			continue
		}
		if stageCategory, ok := block["stage_category"].(map[string]any); ok {
			errs = append(errs, e40F10CategoryKeyViolations("/"+base+"/stage_category", stageCategory, i05.StageCategory)...)
		}
		if intervalCategory, ok := block["interval_category"].(map[string]any); ok {
			errs = append(errs, e40F10CategoryKeyViolations("/"+base+"/interval_category", intervalCategory, i05.IntervalCategory)...)
		}
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
			if value == "" {
				continue // absence is a required-field concern, not a vocabulary one
			}
			if !e40F10ContainsString(i08.TruthResult, value) {
				errs = append(errs, fmt.Sprintf("/quality/by_scenario[%d]/%s: %q is not a member of I-08 truth_result %v", i, field, value, i08.TruthResult))
			}
		}
		oracle, _ := scenario["execution_oracle"].(string)
		if oracle != "" && !e40F10ContainsString(i08.OracleResult, oracle) {
			errs = append(errs, fmt.Sprintf("/quality/by_scenario[%d]/execution_oracle: %q is not a member of I-08 oracle_result %v", i, oracle, i08.OracleResult))
		}
	}
	return errs
}

// e40F10CategoryKeyViolations checks that every key of partition (an I-05
// stage_category or interval_category partition object) is a member of
// known, except the required "unattributed" residual line which is F10's
// own vocabulary, not an I-05 category member.
func e40F10CategoryKeyViolations(basePointer string, partition map[string]any, known []string) []string {
	keys := make([]string, 0, len(partition))
	for key := range partition {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var errs []string
	for _, key := range keys {
		if key == "unattributed" {
			continue
		}
		if !e40F10ContainsString(known, key) {
			errs = append(errs, fmt.Sprintf("%s/%s: not a member of the referenced I-05 vocabulary %v (REQ-F-018: F10 references, never restates)", basePointer, key, known))
		}
	}
	return errs
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

// ---------------------------------------------------------------------------
// T-E40-F10-002: additional validators the invalid-fixture matrix requires.
// Each returns a []string of "<failing JSON pointer>: <reason>" messages so
// callers can assert the specific failing path named, per TC-078's "Notes
// for Agent" requirement, rather than a generic "invalid" result.
// ---------------------------------------------------------------------------

// e40F10UnknownTopLevelFieldViolations is AC-T1: an unknown/extra field
// beyond the schema's aggregate_top_level_fields must fail validation.
func e40F10UnknownTopLevelFieldViolations(schema e40F10Schema, record map[string]any) []string {
	allowed := make(map[string]bool, len(schema.AggregateTopLevelFields))
	for _, field := range schema.AggregateTopLevelFields {
		allowed[field] = true
	}
	keys := make([]string, 0, len(record))
	for key := range record {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var errs []string
	for _, key := range keys {
		if !allowed[key] {
			errs = append(errs, fmt.Sprintf("/%s: unknown field not present in schema aggregate_top_level_fields (AC-T1)", key))
		}
	}
	return errs
}

// e40F10RequiredArrayNonEmptyViolations closes the schema gap
// aggregate_required_arrays_may_be_empty documents: a required top-level
// array pointer (declared "array" in aggregate_properties, one path segment
// deep -- i.e. one of the ten aggregate.json blocks, not a nested array
// like /quality/by_scenario) must not be present-but-empty unless the
// schema explicitly exempts it.
func e40F10RequiredArrayNonEmptyViolations(schema e40F10Schema, record map[string]any) []string {
	mayBeEmpty := make(map[string]bool, len(schema.AggregateRequiredArraysMayBeEmpty))
	for _, pointer := range schema.AggregateRequiredArraysMayBeEmpty {
		mayBeEmpty[pointer] = true
	}
	pointers := make([]string, 0, len(schema.AggregateProperties))
	for pointer := range schema.AggregateProperties {
		pointers = append(pointers, pointer)
	}
	sort.Strings(pointers)
	var errs []string
	for _, pointer := range pointers {
		if schema.AggregateProperties[pointer] != "array" || strings.Contains(pointer, "[]") {
			continue
		}
		if len(e40F10SplitPointer(pointer)) != 1 {
			continue // only the ten top-level required blocks carry this rule
		}
		if mayBeEmpty[pointer] {
			continue
		}
		for _, match := range e40F10ResolvePointerValues(record, e40F10SplitPointer(pointer), "") {
			if list, ok := match.Value.([]any); ok && len(list) == 0 {
				errs = append(errs, fmt.Sprintf("%s: required array must not be empty", match.Path))
			}
		}
	}
	return errs
}

// e40F10PhaseLabelViolations checks /identity/phase, when present, is a
// member of the schema-owned phase_label vocabulary (REQ-F-017). Presence
// is a separate concern already owned by the required-pointer walk.
func e40F10PhaseLabelViolations(schema e40F10Schema, record map[string]any) []string {
	identity, ok := record["identity"].(map[string]any)
	if !ok {
		return nil
	}
	phase, ok := identity["phase"].(string)
	if !ok || phase == "" {
		return nil
	}
	if !e40F10ContainsString(schema.PhaseLabel, phase) {
		return []string{fmt.Sprintf("/identity/phase: not a member of the schema-owned phase_label vocabulary %v", schema.PhaseLabel)}
	}
	return nil
}

// e40F10SourceDigestViolations checks every /scenarios[]/source_digests
// entry is a well-formed lowercase sha256 hex digest. Dynamic map keys
// (lifecycle_jsonl, evaluation_jsonl, ...) fall outside aggregate_properties'
// static pointer hints, so this is a dedicated walk.
func e40F10SourceDigestViolations(record map[string]any) []string {
	scenarios, _ := record["scenarios"].([]any)
	var errs []string
	for i, raw := range scenarios {
		scenario, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		digests, ok := scenario["source_digests"].(map[string]any)
		if !ok {
			continue
		}
		keys := make([]string, 0, len(digests))
		for key := range digests {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			value, _ := digests[key].(string)
			if !isDigest(value) {
				errs = append(errs, fmt.Sprintf("/scenarios[%d]/source_digests/%s: malformed digest", i, key))
			}
		}
	}
	return errs
}

// e40F10SharePartitionViolations checks that /time/share_partition and
// /cost/share_partition carry no key beyond the REQ-F-011 six named shares
// plus the required "unattributed" residual -- a malformed share-partition
// cell name (spec.md "Data model changes" / share_partition_cell) must fail.
func e40F10SharePartitionViolations(schema e40F10Schema, record map[string]any) []string {
	allowed := make(map[string]bool, len(schema.SharePartitionCell)+1)
	for _, cell := range schema.SharePartitionCell {
		allowed[cell] = true
	}
	allowed[schema.SharePartitionResidual] = true

	var errs []string
	for _, base := range []string{"time", "cost"} {
		block, _ := record[base].(map[string]any)
		partition, ok := block["share_partition"].(map[string]any)
		if !ok {
			continue
		}
		keys := make([]string, 0, len(partition))
		for key := range partition {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if !allowed[key] {
				errs = append(errs, fmt.Sprintf("/%s/share_partition/%s: not a member of the schema-owned share_partition_cell vocabulary %v", base, key, schema.SharePartitionCell))
			}
		}
	}
	return errs
}

// e40F10NoiseBandDerivationRuleViolations checks every /noise_bands[]
// entry's derivation_rule is a member of the schema-owned
// noise_band_derivation_rule vocabulary.
func e40F10NoiseBandDerivationRuleViolations(schema e40F10Schema, record map[string]any) []string {
	bands, _ := record["noise_bands"].([]any)
	var errs []string
	for i, raw := range bands {
		band, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		rule, ok := band["derivation_rule"].(string)
		if !ok || rule == "" {
			continue // presence is a separate, required-pointer concern
		}
		if !e40F10ContainsString(schema.NoiseBandDerivationRule, rule) {
			errs = append(errs, fmt.Sprintf("/noise_bands[%d]/derivation_rule: not a member of the schema-owned noise_band_derivation_rule vocabulary %v", i, schema.NoiseBandDerivationRule))
		}
	}
	return errs
}

// e40F10ValidateAggregateRecord is the single combined aggregate.json
// validator: every invalid_aggregate_fixtures case below, and the valid
// aggregate fixture, are run through exactly this function so the
// invalid-fixture matrix proves something about the same code path the
// valid fixture proves clean.
func e40F10ValidateAggregateRecord(schema e40F10Schema, i05 e40F10I05VocabSchema, i08 e40F10I08VocabSchema, record map[string]any) []string {
	var errs []string
	errs = append(errs, e40F10UnknownTopLevelFieldViolations(schema, record)...)
	errs = append(errs, e40F10ValidateRequiredPointers(record, schema.AggregateRequiredFields)...)
	errs = append(errs, e40F10RequiredArrayNonEmptyViolations(schema, record)...)
	errs = append(errs, e40F10ValidateAggregateTypes(schema, record)...)
	errs = append(errs, e40F10PhaseLabelViolations(schema, record)...)
	errs = append(errs, e40F10SourceDigestViolations(record)...)
	errs = append(errs, e40F10SharePartitionViolations(schema, record)...)
	errs = append(errs, e40F10NoiseBandDerivationRuleViolations(schema, record)...)
	errs = append(errs, e40F10UpstreamVocabViolations(i05, i08, record)...)
	return errs
}

// e40F10ValidateRetentionManifestRecord combines the required-pointer walk
// with a digest-format check per retained artifact (missing/wrong-type is
// already caught by the pointer walk; malformed-format needs its own check
// since a present, non-empty string still isn't necessarily a valid digest).
func e40F10ValidateRetentionManifestRecord(schema e40F10Schema, record map[string]any) []string {
	errs := e40F10ValidateRequiredPointers(record, schema.RetentionManifestRequiredFields)
	artifacts, ok := record["artifacts"].(map[string]any)
	if !ok {
		return errs
	}
	for _, name := range schema.RetentionRequiredArtifacts {
		if name == "manifest.json" {
			continue // the manifest never digests its own not-yet-written bytes
		}
		entry, ok := artifacts[name].(map[string]any)
		if !ok {
			continue // presence already enforced by the required-pointer walk
		}
		digest, hasDigest := entry["sha256"].(string)
		if hasDigest && digest != "" && !isDigest(digest) {
			errs = append(errs, fmt.Sprintf("/artifacts/%s/sha256: malformed digest", name))
		}
		// UAT-R3-01 (round 3), T-E40-F10-001 fix requirement 1/3: a required
		// artifact's source_path being merely PRESENT (the required-pointer
		// walk above) is not enough -- it must also be non-empty. An empty
		// string is exactly retain_pair's pre-fix fabrication shape (a real,
		// present, digestible placeholder with source_path=="" claiming a
		// required artifact was retained when it never had a real source).
		sourcePath, hasSourcePath := entry["source_path"].(string)
		if hasSourcePath && sourcePath == "" {
			errs = append(errs, fmt.Sprintf("/artifacts/%s/source_path: empty -- a required artifact must carry a real source", name))
		}
	}
	return errs
}

// e40F10ValidatePilotAttestationRecord combines the required-pointer walk
// with the type/non-emptiness/digest-format checks REQ-F-005's four named
// fields need and no static aggregate_properties-style hint table covers.
func e40F10ValidatePilotAttestationRecord(schema e40F10Schema, record map[string]any) []string {
	errs := e40F10ValidateRequiredPointers(record, schema.PilotAttestationRequiredFields)

	if value, present := record["run_reference"]; present && value != nil {
		if _, ok := value.(string); !ok {
			errs = append(errs, "/run_reference: expected string")
		}
	}
	if value, present := record["operator_identity"]; present && value != nil {
		if _, ok := value.(string); !ok {
			errs = append(errs, "/operator_identity: expected string")
		}
	}
	if value, present := record["checklist_results"]; present && value != nil {
		switch list := value.(type) {
		case []any:
			if len(list) == 0 {
				errs = append(errs, "/checklist_results: required array must not be empty")
			}
		default:
			errs = append(errs, "/checklist_results: expected array")
		}
	}
	if value, present := record["inspected_artifact_digests"]; present && value != nil {
		digests, ok := value.(map[string]any)
		if !ok {
			errs = append(errs, "/inspected_artifact_digests: expected object")
		} else if len(digests) == 0 {
			errs = append(errs, "/inspected_artifact_digests: required object must not be empty")
		} else {
			keys := make([]string, 0, len(digests))
			for key := range digests {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				digest, _ := digests[key].(string)
				if !isDigest(digest) {
					errs = append(errs, fmt.Sprintf("/inspected_artifact_digests/%s: malformed digest", key))
				}
			}
		}
	}
	return errs
}

// e40F10ValidateRefusalRecord validates the minimal refusal-reason
// vocabulary fixture (see valid_refusal_reason_fixture's doc comment for
// why this is a dedicated, deliberately small record shape rather than the
// full batch.json refusal-record shape).
func e40F10ValidateRefusalRecord(schema e40F10Schema, record map[string]any) []string {
	reason, ok := record["refusal_reason"].(string)
	if !ok || reason == "" {
		return []string{"/refusal_reason: required field missing"}
	}
	if !e40F10ContainsString(schema.RefusalReason, reason) {
		return []string{fmt.Sprintf("/refusal_reason: not a member of the schema-owned refusal_reason vocabulary %v", schema.RefusalReason)}
	}
	return nil
}

// e40F10RunInvalidFixtureMatrix walks wantPath (fixture filename -> the
// specific failing JSON pointer substring its diagnostic must name),
// reads each fixture from invalidDir, runs validate, and asserts both that
// validation failed and that one of the returned messages names the
// expected path -- never accepting a bare nonzero error count as proof.
func e40F10RunInvalidFixtureMatrix(t *testing.T, invalidDir string, wantPath map[string]string, validate func(record map[string]any) []string) {
	t.Helper()
	names := make([]string, 0, len(wantPath))
	for name := range wantPath {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		name, want := name, wantPath[name]
		t.Run(name, func(t *testing.T) {
			record := e40F10ReadJSONFixture(t, filepath.Join(invalidDir, name))
			errs := validate(record)
			if len(errs) == 0 {
				t.Fatalf("invalid fixture %s unexpectedly passed validation", name)
			}
			joined := strings.Join(errs, "\n")
			if !strings.Contains(joined, want) {
				t.Fatalf("invalid fixture %s: diagnostic did not name the expected failing path %q; got:\n%s", name, want, joined)
			}
		})
	}
}

// e40F10AssertInvalidDirectoryCoverage proves the want-path tables and the
// committed invalid/ directory agree in both directions: every *.json file
// on disk is exercised by exactly one table, and every table entry names a
// file that still exists. Without this, a fixture could be added without a
// table entry (silently untested) or a table entry could survive a deleted
// fixture (silently untested the other way).
func e40F10AssertInvalidDirectoryCoverage(t *testing.T, invalidDir string, tables ...map[string]string) {
	t.Helper()
	entries, err := os.ReadDir(invalidDir)
	if err != nil {
		t.Fatalf("read invalid fixture directory: %v", err)
	}
	onDisk := make(map[string]int)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		onDisk[entry.Name()] = 0
	}
	tabled := make(map[string]int)
	for _, table := range tables {
		for name := range table {
			tabled[name]++
		}
	}
	for name := range onDisk {
		if tabled[name] == 0 {
			t.Errorf("invalid fixture %s exists on disk but is not exercised by any want-path table", name)
		}
	}
	for name, count := range tabled {
		if count > 1 {
			t.Errorf("invalid fixture %s is exercised by more than one want-path table", name)
		}
		if _, present := onDisk[name]; !present {
			t.Errorf("want-path table names %s but no such file exists under %s", name, invalidDir)
		}
	}
}

// ---------------------------------------------------------------------------
// T-E40-F10-002 want-path tables: filename (under
// tests/contracts/testdata/e40_f10/invalid/) -> the specific failing JSON
// pointer substring e40F10RunInvalidFixtureMatrix requires the validator's
// diagnostic to contain. Generated once from the T-E40-F10-001 valid base
// fixtures (one mutation per entry); e40F10AssertInvalidDirectoryCoverage
// proves this list and the committed fixture directory stay in sync.
// ---------------------------------------------------------------------------

var e40F10InvalidAggregateWantPath = map[string]string{
	"aggregate-block-artifact_use-empty.json":                   "/artifact_use/produced_count: required field missing",
	"aggregate-block-artifact_use-missing.json":                 "/artifact_use: required field missing",
	"aggregate-block-artifact_use-null.json":                    "/artifact_use: required field missing",
	"aggregate-block-artifact_use-wrong-type.json":              "/artifact_use: expected object",
	"aggregate-block-comparisons-missing.json":                  "/comparisons: required field missing",
	"aggregate-block-comparisons-null.json":                     "/comparisons: required field missing",
	"aggregate-block-comparisons-wrong-type.json":               "/comparisons: expected array",
	"aggregate-block-cost-empty.json":                           "/cost/stage_category: required field missing",
	"aggregate-block-cost-missing.json":                         "/cost: required field missing",
	"aggregate-block-cost-null.json":                            "/cost: required field missing",
	"aggregate-block-cost-wrong-type.json":                      "/cost: expected object",
	"aggregate-block-identity-empty.json":                       "/identity/schema_version: required field missing",
	"aggregate-block-identity-missing.json":                     "/identity: required field missing",
	"aggregate-block-identity-null.json":                        "/identity: required field missing",
	"aggregate-block-identity-wrong-type.json":                  "/identity: expected object",
	"aggregate-block-invalid-missing.json":                      "/invalid: required field missing",
	"aggregate-block-invalid-null.json":                         "/invalid: required field missing",
	"aggregate-block-invalid-wrong-type.json":                   "/invalid: expected array",
	"aggregate-block-noise_bands-empty.json":                    "/noise_bands: required array must not be empty",
	"aggregate-block-noise_bands-missing.json":                  "/noise_bands: required field missing",
	"aggregate-block-noise_bands-null.json":                     "/noise_bands: required field missing",
	"aggregate-block-noise_bands-wrong-type.json":               "/noise_bands: expected array",
	"aggregate-block-quality-empty.json":                        "/quality/by_scenario: required field missing",
	"aggregate-block-quality-missing.json":                      "/quality: required field missing",
	"aggregate-block-quality-null.json":                         "/quality: required field missing",
	"aggregate-block-quality-wrong-type.json":                   "/quality: expected object",
	"aggregate-block-review_value-empty.json":                   "/review_value/gates: required field missing",
	"aggregate-block-review_value-missing.json":                 "/review_value: required field missing",
	"aggregate-block-review_value-null.json":                    "/review_value: required field missing",
	"aggregate-block-review_value-wrong-type.json":              "/review_value: expected object",
	"aggregate-block-scenarios-empty.json":                      "/scenarios: required array must not be empty",
	"aggregate-block-scenarios-missing.json":                    "/scenarios: required field missing",
	"aggregate-block-scenarios-null.json":                       "/scenarios: required field missing",
	"aggregate-block-scenarios-wrong-type.json":                 "/scenarios: expected array",
	"aggregate-block-time-empty.json":                           "/time/lifecycle_wall_seconds: required field missing",
	"aggregate-block-time-missing.json":                         "/time: required field missing",
	"aggregate-block-time-null.json":                            "/time: required field missing",
	"aggregate-block-time-wrong-type.json":                      "/time: expected object",
	"digest-batch-policy-digest-malformed.json":                 "/identity/batch_policy_digest: expected lowercase sha256 hex digest",
	"digest-retention-root-digest-malformed.json":               "/identity/retention_root_digest: expected lowercase sha256 hex digest",
	"digest-source-digest-malformed.json":                       "/scenarios[0]/source_digests/lifecycle_jsonl: malformed digest",
	"noise-band-derivation-rule-malformed.json":                 "/noise_bands[0]/derivation_rule: not a member of the schema-owned noise_band_derivation_rule vocabulary",
	"share-partition-unknown-cell-cost.json":                    "/cost/share_partition/extra_cell: not a member of the schema-owned share_partition_cell vocabulary",
	"share-partition-unknown-cell-time.json":                    "/time/share_partition/extra_cell: not a member of the schema-owned share_partition_cell vocabulary",
	"subfield-eligibility-aggregate-eligible-missing.json":      "/scenarios[0]/eligibility/aggregate_eligible: required field missing",
	"subfield-eligibility-aggregate-eligible-wrong-type.json":   "/scenarios[0]/eligibility/aggregate_eligible: expected boolean",
	"subfield-eligibility-invalidity-reasons-missing.json":      "/scenarios[0]/eligibility/invalidity_reasons: required field missing",
	"subfield-eligibility-invalidity-reasons-wrong-type.json":   "/scenarios[0]/eligibility/invalidity_reasons: expected array",
	"subfield-eligibility-publication-eligible-missing.json":    "/scenarios[0]/eligibility/publication_eligible: required field missing",
	"subfield-eligibility-publication-eligible-wrong-type.json": "/scenarios[0]/eligibility/publication_eligible: expected boolean",
	"subfield-insufficient-reps-missing.json":                   "/noise_bands[0]/insufficient_reps: required field missing",
	"subfield-insufficient-reps-wrong-type.json":                "/noise_bands[0]/insufficient_reps: expected boolean",
	"subfield-phase-wrong-value.json":                           "/identity/phase: not a member of the schema-owned phase_label vocabulary",
	"unknown-top-level-field.json":                              "/_unexpected_field: unknown field",
	"vocab-stage-category-restated.json":                        "/time/stage_category/coding: not a member of the referenced I-05 vocabulary",
	"vocab-truth-result-restated.json":                          "/quality/by_scenario[0]/structural: \"success\" is not a member of I-08 truth_result",
}

var e40F10InvalidRetentionManifestWantPath = map[string]string{
	"retention-manifest-artifacts-missing.json":                    "/artifacts: required field missing",
	"retention-manifest-entity-history-json-entry-missing.json":    "/artifacts/entity-history.json: required field missing",
	"retention-manifest-entity-history-json-sha256-malformed.json": "/artifacts/entity-history.json/sha256: malformed digest",
	"retention-manifest-evaluation-jsonl-entry-missing.json":       "/artifacts/evaluation.jsonl: required field missing",
	"retention-manifest-evaluation-jsonl-sha256-malformed.json":    "/artifacts/evaluation.jsonl/sha256: malformed digest",
	"retention-manifest-evidence-entry-missing.json":               "/artifacts/evidence: required field missing",
	"retention-manifest-evidence-sha256-malformed.json":            "/artifacts/evidence/sha256: malformed digest",
	"retention-manifest-lifecycle-jsonl-entry-missing.json":        "/artifacts/lifecycle.jsonl: required field missing",
	"retention-manifest-lifecycle-jsonl-sha256-malformed.json":     "/artifacts/lifecycle.jsonl/sha256: malformed digest",
	"retention-manifest-oracle-json-entry-missing.json":            "/artifacts/oracle.json: required field missing",
	"retention-manifest-oracle-json-sha256-malformed.json":         "/artifacts/oracle.json/sha256: malformed digest",
	"retention-manifest-package-yaml-entry-missing.json":           "/artifacts/package.yaml: required field missing",
	"retention-manifest-package-yaml-sha256-malformed.json":        "/artifacts/package.yaml/sha256: malformed digest",
	"retention-manifest-package-yaml-sha256-missing.json":          "/artifacts/package.yaml/sha256: required field missing",
	"retention-manifest-package-yaml-source-path-missing.json":     "/artifacts/package.yaml/source_path: required field missing",
	"retention-manifest-rep-missing.json":                          "/rep: required field missing",
	"retention-manifest-scenario-id-missing.json":                  "/scenario_id: required field missing",
	"retention-manifest-transcripts-entry-missing.json":            "/artifacts/transcripts: required field missing",
	"retention-manifest-transcripts-sha256-malformed.json":         "/artifacts/transcripts/sha256: malformed digest",

	// UAT-R3-01 (round 3), T-E40-F10-001 fix requirement 3 ("add contract
	// fixtures proving each required artifact with absent provenance
	// fails"): source_path PRESENT but empty is distinct from source_path
	// entirely absent (the pre-existing "-source-path-missing" case above)
	// -- both must fail, but retain_pair's pre-fix fabrication produced the
	// PRESENT-but-empty shape specifically, so this is the exhaustive
	// seven-artifact form of exactly that regression.
	"retention-manifest-package-yaml-source-path-empty.json":        "/artifacts/package.yaml/source_path: empty",
	"retention-manifest-evidence-source-path-empty.json":            "/artifacts/evidence/source_path: empty",
	"retention-manifest-transcripts-source-path-empty.json":         "/artifacts/transcripts/source_path: empty",
	"retention-manifest-entity-history-json-source-path-empty.json": "/artifacts/entity-history.json/source_path: empty",
	"retention-manifest-lifecycle-jsonl-source-path-empty.json":     "/artifacts/lifecycle.jsonl/source_path: empty",
	"retention-manifest-evaluation-jsonl-source-path-empty.json":    "/artifacts/evaluation.jsonl/source_path: empty",
	"retention-manifest-oracle-json-source-path-empty.json":         "/artifacts/oracle.json/source_path: empty",
}

var e40F10InvalidPilotAttestationWantPath = map[string]string{
	"pilot-checklist-results-empty.json":                     "/checklist_results: required array must not be empty",
	"pilot-checklist-results-missing.json":                   "/checklist_results: required field missing",
	"pilot-checklist-results-wrong-type.json":                "/checklist_results: expected array",
	"pilot-inspected-artifact-digests-empty.json":            "/inspected_artifact_digests: required object must not be empty",
	"pilot-inspected-artifact-digests-malformed-digest.json": "/inspected_artifact_digests/lifecycle.jsonl: malformed digest",
	"pilot-inspected-artifact-digests-missing.json":          "/inspected_artifact_digests: required field missing",
	"pilot-operator-identity-missing.json":                   "/operator_identity: required field missing",
	"pilot-operator-identity-wrong-type.json":                "/operator_identity: expected string",
	"pilot-run-reference-missing.json":                       "/run_reference: required field missing",
	"pilot-run-reference-wrong-type.json":                    "/run_reference: expected string",
}

var e40F10InvalidRefusalWantPath = map[string]string{
	"refusal-unknown-reason.json": "/refusal_reason: not a member of the schema-owned refusal_reason vocabulary",
}
