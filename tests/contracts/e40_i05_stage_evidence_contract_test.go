// TC-042 verifies the I-05 stage evidence and evaluator isolation contract
// E40-F06 produces for E40-F08, E40-F09, and E40-F10 (spec.md "Produces:
// I-05"). Per REQ-NF-003/ADR-F06-09, this validator reads only in-repo
// artifacts -- bench/evidence/i05-schema.yaml and
// tests/contracts/testdata/e40_i05/{valid,invalid}/** -- and never a
// populated fixture submodule, so its result is identical whether
// bench/fixture-py or bench/fixture-repo is populated or gitlink-only,
// mirroring TestTC030_I04ScenarioPackageContract's own submodule-independence
// discipline.
//
// This task (T-E40-F06-001) covers AC-001, AC-002, AC-003, AC-015, AC-017
// (Go half), and AC-020. The usage-mapping.yaml fail-closed and
// verification_tier decision tables (AC-008, AC-022) are
// T-E40-F06-002's extension of this same test file.
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

// e40I05SupportedSchemaVersion is the i05-schema.yaml / bundle.json
// schema_version this validator understands (mirrors
// e40I04SupportedSchemaVersion).
const e40I05SupportedSchemaVersion = "1.0"

// e40I05RootSpec is one entry of i05-schema.yaml's roots map: the required
// worker_access mode for that root name (REQ-F-002).
type e40I05RootSpec struct {
	WorkerAccess string `yaml:"worker_access"`
}

// e40I05Schema decodes bench/evidence/i05-schema.yaml -- REQ-F-017's single
// machine-readable owner of I-05's schema_version and every closed
// vocabulary. The Go validator below reads every vocabulary from this
// struct; none of the vocab value lists are duplicated as Go constants.
type e40I05Schema struct {
	SchemaVersion        string                    `yaml:"schema_version"`
	Roots                map[string]e40I05RootSpec `yaml:"roots"`
	StageCategory        []string                  `yaml:"stage_category"`
	IntervalCategory     []string                  `yaml:"interval_category"`
	ArtifactType         []string                  `yaml:"artifact_type"`
	EdgeKind             []string                  `yaml:"edge_kind"`
	EvaluatorAccessPhase []string                  `yaml:"evaluator_access_phase"`
	StopOutcome          []string                  `yaml:"stop_outcome"`
	ErrorKind            []string                  `yaml:"error_kind"`
}

// TestTC042_I05StageEvidenceContract is the shared contract test E40-F08,
// E40-F09, and E40-F10 must reuse verbatim (spec.md Cross-feature
// interactions: "no twin test is created").
func TestTC042_I05StageEvidenceContract(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	schemaPath := filepath.Join(repoRoot, "bench", "evidence", "i05-schema.yaml")
	testdataRoot := filepath.Join(repoRoot, "tests", "contracts", "testdata", "e40_i05")

	schema := e40I05ReadSchema(t, schemaPath)

	t.Run("schema_self_check", func(t *testing.T) {
		if schema.SchemaVersion != e40I05SupportedSchemaVersion {
			t.Errorf("i05-schema.yaml schema_version = %q, want %q", schema.SchemaVersion, e40I05SupportedSchemaVersion)
		}
		wantRoots := map[string]string{
			"agent_fixture_checkout": "read_write",
			"scratch_shark_project":  "authorized_surfaces_only",
			"evaluator_only":         "never_during_dispatch",
		}
		if len(schema.Roots) != len(wantRoots) {
			t.Errorf("i05-schema.yaml roots has %d entries, want %d", len(schema.Roots), len(wantRoots))
		}
		for name, wantAccess := range wantRoots {
			got, ok := schema.Roots[name]
			if !ok {
				t.Errorf("i05-schema.yaml roots missing %q", name)
				continue
			}
			if got.WorkerAccess != wantAccess {
				t.Errorf("i05-schema.yaml roots.%s.worker_access = %q, want %q", name, got.WorkerAccess, wantAccess)
			}
		}
		for _, vocab := range []struct {
			name   string
			values []string
		}{
			{"stage_category", schema.StageCategory},
			{"interval_category", schema.IntervalCategory},
			{"artifact_type", schema.ArtifactType},
			{"edge_kind", schema.EdgeKind},
			{"evaluator_access_phase", schema.EvaluatorAccessPhase},
			{"stop_outcome", schema.StopOutcome},
			{"error_kind", schema.ErrorKind},
		} {
			if len(vocab.values) == 0 {
				t.Errorf("i05-schema.yaml %s vocabulary is empty", vocab.name)
			}
		}
	})

	// AC-001, AC-002 (positive cell), AC-020 (bidirectional agreement):
	// every fixture bundle under valid/ must satisfy the full REQ-F-002/
	// 003/005/006/008/009 field inventory and every closed-vocabulary value
	// it uses must resolve against i05-schema.yaml.
	t.Run("valid_fixtures_field_inventory", func(t *testing.T) {
		validRoot := filepath.Join(testdataRoot, "valid")
		dirs := e40I05ListBundleDirs(t, validRoot)
		if len(dirs) == 0 {
			t.Fatalf("no valid fixture bundles found under %s", validRoot)
		}

		exercised := map[string]map[string]bool{
			"stage_category":         {},
			"interval_category":      {},
			"artifact_type":          {},
			"edge_kind":              {},
			"evaluator_access_phase": {},
			"stop_outcome":           {},
			"error_kind":             {},
		}

		for _, dir := range dirs {
			dir := dir
			t.Run(filepath.Base(dir), func(t *testing.T) {
				bundle, stages := e40I05ReadBundle(t, dir)
				errs := e40I05ValidateBundle(bundle, stages, dir, schema)
				if len(errs) != 0 {
					t.Errorf("valid fixture failed validation, want zero errors:\n%s", strings.Join(errs, "\n"))
				}
				e40I05CollectExercisedVocab(bundle, stages, exercised)
			})
		}

		// AC-020: a schema-declared value that no committed valid fixture
		// exercises is a legitimate, non-fatal coverage gap -- schema and
		// bundle fixtures may diverge in coverage -- surfaced as a log
		// note, not a failure.
		for _, vocab := range []struct {
			name   string
			values []string
		}{
			{"stage_category", schema.StageCategory},
			{"interval_category", schema.IntervalCategory},
			{"artifact_type", schema.ArtifactType},
			{"edge_kind", schema.EdgeKind},
			{"evaluator_access_phase", schema.EvaluatorAccessPhase},
			{"stop_outcome", schema.StopOutcome},
			{"error_kind", schema.ErrorKind},
		} {
			for _, v := range vocab.values {
				// evaluator_access_phase.pre_terminal is a negative-only
				// value by construction (REQ-F-012: every legitimate access
				// is granted post-terminal) -- no valid fixture will ever
				// exercise it, so it is excluded from this sweep rather than
				// permanently logged as an unaddressed coverage gap.
				if vocab.name == "evaluator_access_phase" && v == "pre_terminal" {
					continue
				}
				if !exercised[vocab.name][v] {
					t.Logf("declared but unexercised (AC-020, non-fatal): %s = %q is in i05-schema.yaml but no valid/ fixture uses it", vocab.name, v)
				}
			}
		}
	})

	// AC-002: three-root decision table.
	t.Run("root_policy_decision_table", func(t *testing.T) {
		cases := []struct {
			name    string
			dir     string
			wantAny []string
		}{
			{"missing_root", filepath.Join(testdataRoot, "invalid", "missing-root"), []string{"evaluator_only"}},
			{"nested_root_pair", filepath.Join(testdataRoot, "invalid", "nested-root"), []string{"agent_fixture_checkout", "scratch_shark_project"}},
		}
		for _, c := range cases {
			c := c
			t.Run(c.name, func(t *testing.T) {
				bundle, stages := e40I05ReadBundle(t, c.dir)
				errs := e40I05ValidateBundle(bundle, stages, c.dir, schema)
				if len(errs) == 0 {
					t.Fatalf("case %s: expected validation errors, got none", c.name)
				}
				for _, want := range c.wantAny {
					if !e40ContainsErrorMatching(errs, want) {
						t.Errorf("case %s: expected an error naming %q, got:\n%s", c.name, want, strings.Join(errs, "\n"))
					}
				}
			})
		}

		// Positive control: the comprehensive valid fixture declares all
		// three roots with distinct, pairwise non-nested paths and the
		// required worker_access modes.
		t.Run("valid_bundle_declares_all_three_roots", func(t *testing.T) {
			dir := filepath.Join(testdataRoot, "valid", "prelude-lifecycle")
			bundle, stages := e40I05ReadBundle(t, dir)
			errs := e40I05ValidateBundle(bundle, stages, dir, schema)
			if len(errs) != 0 {
				t.Errorf("valid bundle's root policy failed, want zero errors:\n%s", strings.Join(errs, "\n"))
			}
		})
	})

	// AC-003: REQ-F-004's dual completeness oracle -- prelude and lifecycle
	// halves evaluated by genuinely different oracles.
	t.Run("stage_completeness_dual_oracle", func(t *testing.T) {
		t.Run("prelude_missing_stage_is_named", func(t *testing.T) {
			dir := filepath.Join(testdataRoot, "invalid", "prelude-missing-stage")
			bundle, stages := e40I05ReadBundle(t, dir)
			errs := e40I05ValidateBundle(bundle, stages, dir, schema)
			if len(errs) == 0 {
				t.Fatal("expected a missing_stage violation, got none")
			}
			if !e40ContainsErrorMatching(errs, "missing_stage", "D03") {
				t.Errorf("expected a missing_stage error naming D03, got:\n%s", strings.Join(errs, "\n"))
			}
		})

		t.Run("lifecycle_unmatched_dispatch_is_named", func(t *testing.T) {
			dir := filepath.Join(testdataRoot, "invalid", "lifecycle-unmatched-dispatch")
			bundle, stages := e40I05ReadBundle(t, dir)
			errs := e40I05ValidateBundle(bundle, stages, dir, schema)
			if len(errs) == 0 {
				t.Fatal("expected an unmatched_dispatch violation, got none")
			}
			if !e40ContainsErrorMatching(errs, "unmatched_dispatch") {
				t.Errorf("expected an unmatched_dispatch error, got:\n%s", strings.Join(errs, "\n"))
			}
			// This is the REQ-F-004 defect TC-042 must catch: a validator
			// must never claim missing_stage for the lifecycle half, which
			// has no "should have been dispatched" oracle.
			if e40ContainsErrorMatching(errs, "missing_stage") {
				t.Errorf("lifecycle half must never produce a missing_stage verdict (REQ-F-004), got:\n%s", strings.Join(errs, "\n"))
			}
		})

		t.Run("lifecycle_duplicate_ordinal_is_named", func(t *testing.T) {
			dir := filepath.Join(testdataRoot, "invalid", "duplicate-dispatch-ordinal")
			bundle, stages := e40I05ReadBundle(t, dir)
			errs := e40I05ValidateBundle(bundle, stages, dir, schema)
			if len(errs) == 0 {
				t.Fatal("expected a duplicate_dispatch_ordinal violation, got none")
			}
			if !e40ContainsErrorMatching(errs, "duplicate_dispatch_ordinal") {
				t.Errorf("expected a duplicate_dispatch_ordinal error, got:\n%s", strings.Join(errs, "\n"))
			}
		})

		t.Run("lifecycle_consistent_has_no_missing_stage_verdict_available", func(t *testing.T) {
			dir := filepath.Join(testdataRoot, "valid", "lifecycle-only")
			bundle, stages := e40I05ReadBundle(t, dir)
			errs := e40I05ValidateBundle(bundle, stages, dir, schema)
			if len(errs) != 0 {
				t.Errorf("consistent lifecycle-only bundle failed validation, want zero errors:\n%s", strings.Join(errs, "\n"))
			}
			// Proves the two halves are evaluated by genuinely different
			// oracles rather than one shared rule wearing two names.
			if e40ContainsErrorMatching(errs, "missing_stage") {
				t.Errorf("lifecycle-only bundle must never surface a missing_stage verdict, got:\n%s", strings.Join(errs, "\n"))
			}
		})
	})

	// AC-015: table-driven REQ-F-016 malformed-bundle cases, each naming
	// the failing field. The usage-mapping-dependent fail-closed cases
	// (AC-008, AC-022) are T-E40-F06-002's addition to this table.
	t.Run("malformed_bundle_cases_req_f_016", func(t *testing.T) {
		// REQ-F-016 names exactly 11 malformed-bundle items for this task
		// (the zero-valued-usage-slot item is included; the twelfth item --
		// a zero-valued slot rejected because a *provider* is declared
		// unmapped -- is T-E40-F06-002's usage-mapping.yaml addition to
		// this same table). Two of the 11 named items are themselves
		// "A or B" pairs ("a missing or overlapping root", "an overlapping
		// or non-reconciling ledger"); both sub-variants are exercised
		// below for full decision-table coverage, so this table has 13
		// rows proving the 11 named items, not 13 distinct REQ-F-016 items.
		// wantAny: at least one reported error must name this field/kind
		// (existence check). onlyErrorsMatching: EVERY reported error must
		// match at least one of these substrings (purity check) -- proving
		// each fixture carries its one named defect and nothing else, per
		// the test-plan AC-015 row's "not merely correct when present, but
		// absent otherwise" discipline. Where a case legitimately produces
		// more than one error, onlyErrorsMatching documents why each extra
		// error is a direct mathematical consequence of the *same* injected
		// defect (e.g. overlapping intervals mechanically also break
		// reconciliation) rather than an unrelated defect the fixture
		// accidentally introduced.
		cases := []struct {
			dir                string
			wantAny            []string
			onlyErrorsMatching []string
		}{
			{"missing-root", []string{"evaluator_only"}, []string{"missing_root"}},                                     // item 1a: missing root
			{"nested-root", []string{"agent_fixture_checkout"}, []string{"overlapping_root"}},                          // item 1b: overlapping root
			{"unknown-stage-category", []string{"stage_category", "deployment"}, []string{"stage_category"}},           // item 2 (index + snapshot layers both catch it)
			{"unknown-interval-category", []string{"interval", "network_wait"}, []string{"unknown_interval_category"}}, // item 3
			// item 4a: two overlapping intervals mechanically double-count
			// their shared span, so the same injected defect also trips
			// ledger_non_reconciling -- not a second, unrelated defect.
			{"overlapping-ledger", []string{"ledger_overlap"}, []string{"ledger_overlap", "ledger_non_reconciling"}},
			{"non-reconciling-ledger", []string{"ledger_non_reconciling"}, []string{"ledger_non_reconciling"}},              // item 4b
			{"candidate-missing-field", []string{"candidate", "tree_digest"}, []string{"candidate_field_missing"}},          // item 5
			{"artifact-missing-field", []string{"artifact", "producer_stage"}, []string{"artifact_field_missing"}},          // item 6
			{"usage-slot-zero-when-absent", []string{"total_cost"}, []string{"usage.total_cost"}},                           // item 7
			{"evaluator-access-out-of-order", []string{"isolation_violation"}, []string{"isolation_violation"}},             // item 8
			{"stop-outcome-eligible-conflict", []string{"publication_eligible"}, []string{"publication_eligible_conflict"}}, // item 9
			{"duplicate-dispatch-ordinal", []string{"duplicate_dispatch_ordinal"}, []string{"duplicate_dispatch_ordinal"}},  // item 10
			{"unsupported-schema-version", []string{"schema_version"}, []string{"unsupported_schema_version"}},              // item 11
		}
		for _, c := range cases {
			c := c
			t.Run(c.dir, func(t *testing.T) {
				dir := filepath.Join(testdataRoot, "invalid", c.dir)
				bundle, stages := e40I05ReadBundle(t, dir)
				errs := e40I05ValidateBundle(bundle, stages, dir, schema)
				if len(errs) == 0 {
					t.Fatalf("case %s: expected validation errors, got none", c.dir)
				}
				for _, want := range c.wantAny {
					if !e40ContainsErrorMatching(errs, want) {
						t.Errorf("case %s: expected an error naming %q, got:\n%s", c.dir, want, strings.Join(errs, "\n"))
					}
				}
				for _, e := range errs {
					matches := false
					for _, allowed := range c.onlyErrorsMatching {
						if strings.Contains(e, allowed) {
							matches = true
							break
						}
					}
					if !matches {
						t.Errorf("case %s: unexpected error not matching this case's own defect class %v (fixture may be contaminated by an unrelated defect): %s\nall errors:\n%s", c.dir, c.onlyErrorsMatching, e, strings.Join(errs, "\n"))
					}
				}
			})
		}

		// Positive controls: correcting exactly the one field each case
		// above violates passes, proving the validator isn't rejecting the
		// whole bundle for an unrelated reason.
		t.Run("valid_baseline_passes", func(t *testing.T) {
			dir := filepath.Join(testdataRoot, "valid", "prelude-lifecycle")
			bundle, stages := e40I05ReadBundle(t, dir)
			if errs := e40I05ValidateBundle(bundle, stages, dir, schema); len(errs) != 0 {
				t.Errorf("valid baseline fixture failed validation, want zero errors:\n%s", strings.Join(errs, "\n"))
			}
		})
	})

	// AC-020: single-owner vocabulary agreement is bidirectional. A value
	// used by a fixture but absent from i05-schema.yaml must be rejected as
	// unknown -- reusing the unknown-stage-category case above proves the
	// "value present in neither" direction; the "declared but unexercised"
	// direction is asserted inside valid_fixtures_field_inventory above.
	t.Run("vocabulary_single_owner_bidirectional", func(t *testing.T) {
		dir := filepath.Join(testdataRoot, "invalid", "unknown-stage-category")
		bundle, stages := e40I05ReadBundle(t, dir)
		errs := e40I05ValidateBundle(bundle, stages, dir, schema)
		if !e40ContainsErrorMatching(errs, "stage_category", "deployment") {
			t.Errorf("expected an error naming the unknown stage_category value %q, got:\n%s", "deployment", strings.Join(errs, "\n"))
		}
		// A validator embedding a private copy of the vocabulary would
		// still accept "deployment" even if i05-schema.yaml never listed
		// it. Prove the schema is the actual source consulted: "deployment"
		// is genuinely absent from the schema's stage_category list.
		for _, v := range schema.StageCategory {
			if v == "deployment" {
				t.Fatalf("test fixture bug: %q must NOT be a declared stage_category value in i05-schema.yaml", v)
			}
		}
	})
}

// e40I05ReadSchema reads and parses the real committed i05-schema.yaml.
func e40I05ReadSchema(t *testing.T, path string) *e40I05Schema {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read i05 schema %s: %v", path, err)
	}
	var schema e40I05Schema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		t.Fatalf("parse i05 schema %s: %v", path, err)
	}
	return &schema
}

// e40I05ListBundleDirs lists the immediate subdirectories of root, each one
// bundle fixture.
func e40I05ListBundleDirs(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read %s: %v", root, err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(root, e.Name()))
		}
	}
	sort.Strings(dirs)
	return dirs
}

// e40I05StageFile pairs a stage snapshot's decoded content with the
// snapshot_path-relative name of the file it was read from, so callers can
// cross-reference bundle.json's stages[] index against the files actually
// present on disk (REQ-F-004's lifecycle "observed dispatch" oracle).
type e40I05StageFile struct {
	RelPath string
	Content map[string]interface{}
}

// e40I05ReadBundle reads and JSON-decodes one bundle directory's bundle.json
// plus every file under stages/. Real filesystem reads of committed JSON,
// per this task's Caller-Path Contract (TC-042 row): a validator reading a
// hand-built in-memory manifest would stay green even if a real committed
// bundle fixture were malformed.
func e40I05ReadBundle(t *testing.T, dir string) (map[string]interface{}, []e40I05StageFile) {
	t.Helper()
	bundlePath := filepath.Join(dir, "bundle.json")
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read %s: %v", bundlePath, err)
	}
	var bundle map[string]interface{}
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatalf("parse %s: %v", bundlePath, err)
	}

	stagesDir := filepath.Join(dir, "stages")
	entries, err := os.ReadDir(stagesDir)
	if err != nil {
		t.Fatalf("read %s: %v", stagesDir, err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	var stages []e40I05StageFile
	for _, name := range files {
		p := filepath.Join(stagesDir, name)
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		var stage map[string]interface{}
		if err := json.Unmarshal(raw, &stage); err != nil {
			t.Fatalf("parse %s: %v", p, err)
		}
		stages = append(stages, e40I05StageFile{RelPath: "stages/" + name, Content: stage})
	}
	return bundle, stages
}

func e40I05AsMap(v interface{}) (map[string]interface{}, bool) {
	m, ok := v.(map[string]interface{})
	return m, ok
}

func e40I05AsStringSlice(v interface{}) []string {
	list, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func e40I05StringSet(values []string) map[string]bool {
	m := make(map[string]bool, len(values))
	for _, v := range values {
		m[v] = true
	}
	return m
}

// e40I05ValidateBundle applies the full REQ-F-002/003/004/005/006/008/
// 009/012/014/016/017 field inventory to one decoded bundle plus its stage
// snapshots, returning one description per violation. dir is the bundle's
// own directory, used only for error context.
func e40I05ValidateBundle(bundle map[string]interface{}, stageFiles []e40I05StageFile, dir string, schema *e40I05Schema) []string {
	var errs []string
	addf := func(format string, args ...interface{}) {
		errs = append(errs, fmt.Sprintf(format, args...))
	}

	schemaVersion, _ := bundle["schema_version"].(string)
	if schemaVersion != e40I05SupportedSchemaVersion {
		addf("unsupported_schema_version: schema_version = %q, want %q", schemaVersion, e40I05SupportedSchemaVersion)
	}

	scenario, ok := e40I05AsMap(bundle["scenario"])
	if !ok {
		addf("scenario: missing or not an object")
	} else {
		for _, f := range []string{"scenario_id", "entity_family"} {
			if s, _ := scenario[f].(string); strings.TrimSpace(s) == "" {
				addf("scenario.%s: empty", f)
			}
		}
	}

	if s, _ := bundle["run_id"].(string); strings.TrimSpace(s) == "" {
		addf("run_id: empty")
	}

	errs = append(errs, e40I05ValidateRoots(bundle["roots"], schema)...)

	matrix, ok := e40I05AsMap(bundle["stage_matrix_source"])
	if !ok {
		addf("stage_matrix_source: missing or not an object")
		matrix = nil
	}

	// stages[] index: dispatch_ordinal must be unique within the bundle
	// (REQ-F-004), and each entry's declared stage_category/snapshot_path
	// must resolve against the schema and an actual file on disk.
	indexList, _ := bundle["stages"].([]interface{})
	seenOrdinal := map[float64]string{}
	indexedRelPaths := map[string]bool{}
	stageCategorySet := e40I05StringSet(schema.StageCategory)

	for i, raw := range indexList {
		entry, ok := e40I05AsMap(raw)
		if !ok {
			addf("stages[%d]: not an object", i)
			continue
		}
		ordinal, hasOrdinal := e40I05AsNumber(entry["dispatch_ordinal"])
		if !hasOrdinal {
			addf("stages[%d].dispatch_ordinal: missing or not a number", i)
		} else if prevKey, dup := seenOrdinal[ordinal]; dup {
			addf("duplicate_dispatch_ordinal: dispatch_ordinal %v is used by both %q and %q", ordinal, prevKey, entry["stage_key"])
		} else {
			seenOrdinal[ordinal] = fmt.Sprintf("%v", entry["stage_key"])
		}

		category, _ := entry["stage_category"].(string)
		if !stageCategorySet[category] {
			addf("unknown_stage_category: stages[%d].stage_category = %q not declared in i05-schema.yaml", i, category)
		}

		relPath, _ := entry["snapshot_path"].(string)
		if strings.TrimSpace(relPath) == "" {
			addf("stages[%d].snapshot_path: empty", i)
		} else {
			indexedRelPaths[relPath] = true
			if !e40FileExists(filepath.Join(dir, relPath)) {
				addf("stages[%d].snapshot_path = %q does not exist on disk", i, relPath)
			}
		}
	}

	// REQ-F-004 lifecycle half: every real snapshot file on disk (an
	// "observed dispatch") must have a matching stages[] index entry --
	// unmatched_dispatch otherwise. This oracle applies uniformly; the
	// prelude half's *additional* missing_stage oracle is layered in below
	// and never substitutes for this one.
	for _, sf := range stageFiles {
		if !indexedRelPaths[sf.RelPath] {
			addf("unmatched_dispatch: %s is a real stage snapshot on disk with no matching stages[] index entry (REQ-F-004)", sf.RelPath)
		}
	}

	// REQ-F-004 prelude half: an applicable prelude stage (D01-D05) with no
	// snapshot is a named missing_stage failure. This check is scoped
	// strictly to stage keys declared in stage_matrix_source.prelude and
	// never runs against the lifecycle half, which has no "should have been
	// dispatched" oracle (ADR-F06-02).
	if matrix != nil {
		prelude, _ := e40I05AsMap(matrix["prelude"])
		indexedStageKeys := map[string]bool{}
		for _, raw := range indexList {
			if entry, ok := e40I05AsMap(raw); ok {
				if key, _ := entry["stage_key"].(string); key != "" {
					indexedStageKeys[key] = true
				}
			}
		}
		stageNames := make([]string, 0, len(prelude))
		for name := range prelude {
			stageNames = append(stageNames, name)
		}
		sort.Strings(stageNames)
		for _, name := range stageNames {
			stage, ok := e40I05AsMap(prelude[name])
			if !ok {
				continue
			}
			applicable, _ := stage["applicable"].(bool)
			if applicable && !indexedStageKeys[name] {
				addf("missing_stage: stage_matrix_source.prelude.%s.applicable is true but no stages[] entry exists for %s", name, name)
			}
		}
	}

	// Stage snapshot content validation.
	for _, sf := range stageFiles {
		errs = append(errs, e40I05ValidateStageSnapshot(sf.Content, sf.RelPath, schema)...)
	}

	// REQ-F-014: a stop outcome paired with publication_eligible: true is
	// always rejected.
	if stopOutcome, present := bundle["stop_outcome"]; present {
		if s, _ := stopOutcome.(string); !e40I05StringSet(schema.StopOutcome)[s] {
			addf("stop_outcome = %q not declared in i05-schema.yaml", s)
		}
		if eligible, _ := bundle["publication_eligible"].(bool); eligible {
			addf("publication_eligible_conflict: stop_outcome %q is present but publication_eligible is true (REQ-F-014)", stopOutcome)
		}
		reasons := e40I05AsStringSlice(bundle["ineligibility_reasons"])
		if len(reasons) == 0 {
			addf("ineligibility_reasons: must be non-empty when stop_outcome is present (REQ-F-014)")
		}
	}

	return errs
}

// e40I05ValidateRoots checks REQ-F-002: all three roots declared, pairwise
// disjoint paths, and the required worker_access mode per root.
func e40I05ValidateRoots(v interface{}, schema *e40I05Schema) []string {
	var errs []string
	m, ok := e40I05AsMap(v)
	if !ok {
		return []string{"roots: missing or not an object"}
	}

	names := make([]string, 0, len(schema.Roots))
	for name := range schema.Roots {
		names = append(names, name)
	}
	sort.Strings(names)

	type resolved struct {
		name string
		path string
	}
	var present []resolved

	for _, name := range names {
		spec := schema.Roots[name]
		entry, ok := e40I05AsMap(m[name])
		if !ok {
			errs = append(errs, fmt.Sprintf("missing_root: roots.%s is required and missing", name))
			continue
		}
		path, _ := entry["path"].(string)
		if strings.TrimSpace(path) == "" {
			errs = append(errs, fmt.Sprintf("roots.%s.path: empty", name))
		}
		access, _ := entry["worker_access"].(string)
		if access != spec.WorkerAccess {
			errs = append(errs, fmt.Sprintf("roots.%s.worker_access = %q, want %q", name, access, spec.WorkerAccess))
		}
		if digest, _ := entry["identity_digest"].(string); strings.TrimSpace(digest) == "" {
			errs = append(errs, fmt.Sprintf("roots.%s.identity_digest: empty", name))
		}
		if strings.TrimSpace(path) != "" {
			present = append(present, resolved{name: name, path: path})
		}
	}

	for i := 0; i < len(present); i++ {
		for j := i + 1; j < len(present); j++ {
			a, b := present[i], present[j]
			if e40I05PathsOverlap(a.path, b.path) {
				errs = append(errs, fmt.Sprintf("overlapping_root: roots.%s (%q) and roots.%s (%q) are not pairwise disjoint (REQ-F-002)", a.name, a.path, b.name, b.path))
			}
		}
	}

	return errs
}

// e40I05PathsOverlap reports whether a and b are equal or one is nested
// inside the other, treating both as slash-separated directory paths. This
// is a string-level containment check over the bundle's declared paths (the
// fixture roots are not real directories on disk), matching the level of
// this Go contract validator; the execution-time guard
// (verify-evidence-roots.sh, REQ-F-011) inspects the real live filesystem.
func e40I05PathsOverlap(a, b string) bool {
	a = strings.TrimRight(a, "/")
	b = strings.TrimRight(b, "/")
	if a == b {
		return true
	}
	return strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}

// e40I05ValidateStageSnapshot checks the REQ-F-003/005/006/008/009/012
// field inventory for one stage snapshot.
func e40I05ValidateStageSnapshot(stage map[string]interface{}, label string, schema *e40I05Schema) []string {
	var errs []string
	addf := func(format string, args ...interface{}) {
		errs = append(errs, fmt.Sprintf("%s: "+format, append([]interface{}{label}, args...)...))
	}

	category, _ := stage["stage_category"].(string)
	if !e40I05StringSet(schema.StageCategory)[category] {
		addf("unknown_stage_category: stage_category = %q not declared in i05-schema.yaml", category)
	}

	if s, _ := stage["prompt_digest"].(string); strings.TrimSpace(s) == "" {
		addf("prompt_digest: empty")
	}
	if s, _ := stage["snapshot_digest"].(string); strings.TrimSpace(s) == "" {
		addf("snapshot_digest: empty")
	}

	errs = append(errs, e40I05ValidateArtifacts(stage["artifacts"], schema, label)...)
	errs = append(errs, e40I05ValidateUsage(stage["usage"], stage["errors"], label)...)
	errs = append(errs, e40I05ValidateTimeLedger(stage["time_ledger"], schema, label)...)
	errs = append(errs, e40I05ValidateEvaluatorAccess(stage["evaluator_access"], schema, label)...)

	if category == "code" || category == "review" {
		errs = append(errs, e40I05ValidateCandidate(stage["candidate"], label)...)
	}

	return errs
}

func e40I05AsNumber(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

// e40I05ValidateCandidate checks REQ-F-006: every code/review snapshot
// carries the full candidate block. A candidate identified by base_commit
// alone (any one of the other four fields missing) is rejected naming that
// field (ADR-009).
func e40I05ValidateCandidate(v interface{}, label string) []string {
	m, ok := e40I05AsMap(v)
	if !ok {
		return []string{fmt.Sprintf("%s: candidate_field_missing: candidate block missing or not an object (required for code/review snapshots, REQ-F-006)", label)}
	}
	var errs []string
	for _, f := range []string{"base_commit", "tree_digest", "binary_diff_digest", "changed_path_digest", "test_suite_digest"} {
		if s, _ := m[f].(string); strings.TrimSpace(s) == "" {
			errs = append(errs, fmt.Sprintf("%s: candidate_field_missing: candidate.%s is required and missing (REQ-F-006, ADR-009)", label, f))
		}
	}
	if _, present := m["dirty_untracked_manifest"]; !present {
		errs = append(errs, fmt.Sprintf("%s: candidate_field_missing: candidate.dirty_untracked_manifest is required and missing (REQ-F-006)", label))
	}
	return errs
}

// e40I05ValidateArtifacts checks REQ-F-008: every artifact record carries
// artifact_type, path, digest, size_bytes, and producer_stage. consumers is
// intentionally not required here -- an absent consumers key and an empty
// consumers: [] are both legal, distinct states (ADR-F06-07); the empty-
// versus-absent distinction itself is bench script tc046's job
// (AC-006), not this task's.
func e40I05ValidateArtifacts(v interface{}, schema *e40I05Schema, label string) []string {
	list, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var errs []string
	artifactTypeSet := e40I05StringSet(schema.ArtifactType)
	edgeKindSet := e40I05StringSet(schema.EdgeKind)
	for i, raw := range list {
		m, ok := e40I05AsMap(raw)
		if !ok {
			errs = append(errs, fmt.Sprintf("%s: artifact_field_missing: artifacts[%d] is not an object", label, i))
			continue
		}
		aType, _ := m["artifact_type"].(string)
		if !artifactTypeSet[aType] {
			errs = append(errs, fmt.Sprintf("%s: artifacts[%d].artifact_type = %q not declared in i05-schema.yaml", label, i, aType))
		}
		if s, _ := m["path"].(string); strings.TrimSpace(s) == "" {
			errs = append(errs, fmt.Sprintf("%s: artifact_field_missing: artifacts[%d].path is required and missing", label, i))
		}
		if s, _ := m["digest"].(string); strings.TrimSpace(s) == "" {
			errs = append(errs, fmt.Sprintf("%s: artifact_field_missing: artifacts[%d].digest is required and missing", label, i))
		}
		if _, ok := e40I05AsNumber(m["size_bytes"]); !ok {
			errs = append(errs, fmt.Sprintf("%s: artifact_field_missing: artifacts[%d].size_bytes is required and missing", label, i))
		}
		if s, _ := m["producer_stage"].(string); strings.TrimSpace(s) == "" {
			errs = append(errs, fmt.Sprintf("%s: artifact_field_missing: artifacts[%d].producer_stage is required and missing", label, i))
		}
		if consumersRaw, present := m["consumers"]; present {
			consumers, _ := consumersRaw.([]interface{})
			for j, cRaw := range consumers {
				c, ok := e40I05AsMap(cRaw)
				if !ok {
					continue
				}
				edgeKind, _ := c["edge_kind"].(string)
				if !edgeKindSet[edgeKind] {
					errs = append(errs, fmt.Sprintf("%s: artifacts[%d].consumers[%d].edge_kind = %q not declared in i05-schema.yaml", label, i, j, edgeKind))
				}
			}
		}
	}
	return errs
}

// e40I05ValidateUsage checks REQ-F-016's usage-slot consistency rule: when
// errors[] records a usage_slot_unavailable entry for a slot, that slot
// MUST be genuinely absent from usage -- never present as zero (or any
// other value). Provider-mapping-specific fail-closed logic (AC-007, AC-008,
// AC-022) is T-E40-F06-002's addition; this task validates only the
// bundle-internal absent-vs-present consistency REQ-F-016 names.
func e40I05ValidateUsage(usageV, errorsV interface{}, label string) []string {
	usage, _ := e40I05AsMap(usageV)
	errList, _ := errorsV.([]interface{})

	var errs []string
	for _, raw := range errList {
		e, ok := e40I05AsMap(raw)
		if !ok {
			continue
		}
		kind, _ := e["kind"].(string)
		if kind != "usage_slot_unavailable" {
			continue
		}
		slot, _ := e["slot"].(string)
		if slot == "" {
			errs = append(errs, fmt.Sprintf("%s: usage_slot_unavailable error missing its slot field", label))
			continue
		}
		if _, present := usage[slot]; present {
			errs = append(errs, fmt.Sprintf("%s: usage.%s is present (must be absent -- errors[] reports it usage_slot_unavailable, REQ-F-016/REQ-F-009)", label, slot))
		}
	}
	return errs
}

// e40I05ValidateTimeLedger checks REQ-F-005: six half-open, pairwise
// disjoint interval categories reconciling to [stage_start, stage_end)
// within reconciliation_epsilon_ns, with any residual attributed to
// unclassified.
func e40I05ValidateTimeLedger(v interface{}, schema *e40I05Schema, label string) []string {
	m, ok := e40I05AsMap(v)
	if !ok {
		return []string{fmt.Sprintf("%s: time_ledger missing or not an object (REQ-F-005)", label)}
	}
	var errs []string

	start, hasStart := e40I05AsNumber(m["stage_start"])
	end, hasEnd := e40I05AsNumber(m["stage_end"])
	epsilon, hasEpsilon := e40I05AsNumber(m["reconciliation_epsilon_ns"])
	if !hasStart || !hasEnd {
		errs = append(errs, fmt.Sprintf("%s: time_ledger.stage_start/stage_end missing or not numeric", label))
		return errs
	}
	if !hasEpsilon {
		errs = append(errs, fmt.Sprintf("%s: time_ledger.reconciliation_epsilon_ns missing or not numeric", label))
	}

	intervalsRaw, ok := e40I05AsMap(m["intervals"])
	if !ok {
		errs = append(errs, fmt.Sprintf("%s: time_ledger.intervals missing or not an object", label))
		return errs
	}

	intervalCategorySet := e40I05StringSet(schema.IntervalCategory)
	type tagged struct {
		category string
		start    float64
		end      float64
	}
	var all []tagged
	var total float64

	categoryNames := make([]string, 0, len(intervalsRaw))
	for name := range intervalsRaw {
		categoryNames = append(categoryNames, name)
	}
	sort.Strings(categoryNames)

	for _, name := range categoryNames {
		if !intervalCategorySet[name] {
			errs = append(errs, fmt.Sprintf("%s: unknown_interval_category: time_ledger.intervals has key %q not declared in i05-schema.yaml", label, name))
			// Deliberately do NOT skip this category's intervals below: an
			// unknown category name is its own, independent defect. The
			// wall-clock time under that name is still real elapsed time,
			// so it still counts toward the reconciliation total and
			// overlap check -- an unrecognized label must not also
			// manufacture an unrelated ledger_non_reconciling verdict for
			// time that was, in fact, fully accounted for.
		}
		list, _ := intervalsRaw[name].([]interface{})
		for _, ivRaw := range list {
			pair, _ := ivRaw.([]interface{})
			if len(pair) != 2 {
				errs = append(errs, fmt.Sprintf("%s: time_ledger.intervals.%s: each interval must be a [start, end) pair", label, name))
				continue
			}
			s, sOK := e40I05AsNumber(pair[0])
			e, eOK := e40I05AsNumber(pair[1])
			if !sOK || !eOK || e <= s {
				errs = append(errs, fmt.Sprintf("%s: time_ledger.intervals.%s: malformed interval %v", label, name, ivRaw))
				continue
			}
			all = append(all, tagged{category: name, start: s, end: e})
			total += e - s
		}
	}

	// Overlap: sort by start and compare each interval against every
	// later-starting interval that begins before it ends. Half-open
	// [start, end) means adjacency (b.start == a.end) is NOT an overlap.
	sort.Slice(all, func(i, j int) bool { return all[i].start < all[j].start })
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].start >= all[i].end {
				break
			}
			errs = append(errs, fmt.Sprintf("%s: ledger_overlap: %s [%v,%v) overlaps %s [%v,%v) (REQ-F-005)", label, all[i].category, all[i].start, all[i].end, all[j].category, all[j].start, all[j].end))
		}
	}

	if hasEpsilon {
		residual := (end - start) - total
		if residual < 0 {
			residual = -residual
		}
		if residual > epsilon {
			errs = append(errs, fmt.Sprintf("%s: ledger_non_reconciling: residual %v ns exceeds reconciliation_epsilon_ns %v (REQ-F-005)", label, residual, epsilon))
		}
	}

	return errs
}

// e40I05ValidateEvaluatorAccess checks REQ-F-012's ordering: every
// evaluator_access event's phase must be post_terminal -- access is only
// ever legitimately granted after the applicable stage or scenario reaches
// terminal status (injection via adapter.sh inject-tests, or an in-place
// oracle read). A pre_terminal phase is itself the isolation_violation.
func e40I05ValidateEvaluatorAccess(v interface{}, schema *e40I05Schema, label string) []string {
	list, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var errs []string
	phaseSet := e40I05StringSet(schema.EvaluatorAccessPhase)
	for i, raw := range list {
		m, ok := e40I05AsMap(raw)
		if !ok {
			errs = append(errs, fmt.Sprintf("%s: evaluator_access[%d] is not an object", label, i))
			continue
		}
		for _, f := range []string{"accessor", "artifact_path", "digest", "phase", "granted_at"} {
			if s, _ := m[f].(string); strings.TrimSpace(s) == "" {
				errs = append(errs, fmt.Sprintf("%s: evaluator_access[%d].%s is required and missing", label, i, f))
			}
		}
		phase, _ := m["phase"].(string)
		if !phaseSet[phase] {
			errs = append(errs, fmt.Sprintf("%s: evaluator_access[%d].phase = %q not declared in i05-schema.yaml", label, i, phase))
			continue
		}
		if phase == "pre_terminal" {
			errs = append(errs, fmt.Sprintf("%s: isolation_violation: evaluator_access[%d] occurred pre_terminal (REQ-F-012 permits access only after terminal status)", label, i))
		}
	}
	return errs
}

// e40I05CollectExercisedVocab records every closed-vocabulary value a valid
// fixture bundle actually uses, so the caller can report (AC-020,
// non-fatal) which i05-schema.yaml-declared values no committed valid
// fixture exercises.
func e40I05CollectExercisedVocab(bundle map[string]interface{}, stageFiles []e40I05StageFile, out map[string]map[string]bool) {
	if s, _ := bundle["stop_outcome"].(string); s != "" {
		out["stop_outcome"][s] = true
	}
	if indexList, ok := bundle["stages"].([]interface{}); ok {
		for _, raw := range indexList {
			if entry, ok := e40I05AsMap(raw); ok {
				if c, _ := entry["stage_category"].(string); c != "" {
					out["stage_category"][c] = true
				}
			}
		}
	}
	for _, sf := range stageFiles {
		if c, _ := sf.Content["stage_category"].(string); c != "" {
			out["stage_category"][c] = true
		}
		if ledger, ok := e40I05AsMap(sf.Content["time_ledger"]); ok {
			if intervals, ok := e40I05AsMap(ledger["intervals"]); ok {
				for name, v := range intervals {
					if list, ok := v.([]interface{}); ok && len(list) > 0 {
						out["interval_category"][name] = true
					}
				}
			}
		}
		if artifacts, ok := sf.Content["artifacts"].([]interface{}); ok {
			for _, raw := range artifacts {
				if a, ok := e40I05AsMap(raw); ok {
					if t, _ := a["artifact_type"].(string); t != "" {
						out["artifact_type"][t] = true
					}
					if consumers, ok := a["consumers"].([]interface{}); ok {
						for _, cRaw := range consumers {
							if c, ok := e40I05AsMap(cRaw); ok {
								if ek, _ := c["edge_kind"].(string); ek != "" {
									out["edge_kind"][ek] = true
								}
							}
						}
					}
				}
			}
		}
		if accessList, ok := sf.Content["evaluator_access"].([]interface{}); ok {
			for _, raw := range accessList {
				if a, ok := e40I05AsMap(raw); ok {
					if p, _ := a["phase"].(string); p != "" {
						out["evaluator_access_phase"][p] = true
					}
				}
			}
		}
		if errList, ok := sf.Content["errors"].([]interface{}); ok {
			for _, raw := range errList {
				if e, ok := e40I05AsMap(raw); ok {
					if k, _ := e["kind"].(string); k != "" {
						out["error_kind"][k] = true
					}
				}
			}
		}
	}
}
