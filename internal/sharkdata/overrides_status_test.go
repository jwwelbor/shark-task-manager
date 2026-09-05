package sharkdata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// canonicalFixturePath is a real embedded canonical file used across
// fixtures below so ReadEmbedded resolves a genuine counterpart.
const canonicalFixturePath = "workflow/epic.yaml"

func writeOverrideFile(t *testing.T, dataRoot, relPath string, data []byte) string {
	t.Helper()
	full := filepath.Join(dataRoot, "overrides", filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("failed to create override dir: %v", err)
	}
	if err := os.WriteFile(full, data, 0644); err != nil {
		t.Fatalf("failed to write override file: %v", err)
	}
	return full
}

func writeManifest(t *testing.T, dataRoot string, manifest *OverrideBaselineManifest) {
	t.Helper()
	if err := manifest.Save(dataRoot); err != nil {
		t.Fatalf("failed to save manifest fixture: %v", err)
	}
}

func canonicalBytes(t *testing.T) []byte {
	t.Helper()
	data, err := ReadEmbedded(canonicalFixturePath)
	if err != nil {
		t.Fatalf("failed to read canonical fixture %q: %v", canonicalFixturePath, err)
	}
	return data
}

func findRow(rows []OverrideRow, path string) *OverrideRow {
	for i := range rows {
		if rows[i].Path == path {
			return &rows[i]
		}
	}
	return nil
}

// TC-001: empty/absent overrides/ -> all-zero summary, no rows.
func TestOverrideStatusAt_EmptyOrAbsent(t *testing.T) {
	t.Run("no overrides directory at all", func(t *testing.T) {
		dataRoot := t.TempDir()

		report, err := OverrideStatusAt(dataRoot)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(report.Rows) != 0 {
			t.Errorf("Rows = %v, want empty", report.Rows)
		}
		wantSummary := map[string]int{
			ClassificationCurrent:            0,
			ClassificationUpstreamChanged:    0,
			ClassificationIdenticalRedundant: 0,
			ClassificationOrphaned:           0,
			ClassificationBaselineUnknown:    0,
		}
		for k, v := range wantSummary {
			if report.Summary[k] != v {
				t.Errorf("Summary[%q] = %d, want %d", k, report.Summary[k], v)
			}
		}
	})

	t.Run("empty overrides directory", func(t *testing.T) {
		dataRoot := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dataRoot, "overrides"), 0755); err != nil {
			t.Fatalf("failed to create overrides dir: %v", err)
		}

		report, err := OverrideStatusAt(dataRoot)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(report.Rows) != 0 {
			t.Errorf("Rows = %v, want empty", report.Rows)
		}
	})
}

// TC-002: upstream_changed when baseline SHA differs from current canonical SHA.
func TestOverrideStatusAt_UpstreamChanged(t *testing.T) {
	dataRoot := t.TempDir()
	writeOverrideFile(t, dataRoot, canonicalFixturePath, []byte("locally modified override content"))
	writeManifest(t, dataRoot, &OverrideBaselineManifest{
		SchemaVersion: 1,
		Baselines:     map[string]string{canonicalFixturePath: "0000000000000000000000000000000000000000000000000000000000000000"[:64]}, //nolint:gosec // stale sentinel digest, not a real hash
	})

	report, err := OverrideStatusAt(dataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := findRow(report.Rows, canonicalFixturePath)
	if row == nil {
		t.Fatalf("row for %q not found in %v", canonicalFixturePath, report.Rows)
	}
	if row.Classification != ClassificationUpstreamChanged {
		t.Errorf("Classification = %q, want %q", row.Classification, ClassificationUpstreamChanged)
	}
	if row.OverrideSHA256 == "" || row.CanonicalSHA256 == "" || row.BaselineSHA256 == "" {
		t.Errorf("expected all three digests populated, got override=%q canonical=%q baseline=%q",
			row.OverrideSHA256, row.CanonicalSHA256, row.BaselineSHA256)
	}
	if report.Summary[ClassificationUpstreamChanged] != 1 {
		t.Errorf("Summary[upstream_changed] = %d, want 1", report.Summary[ClassificationUpstreamChanged])
	}
}

// TC-003: identical_redundant when override bytes equal canonical bytes,
// regardless of baseline entry presence (two subtests).
func TestOverrideStatusAt_IdenticalRedundant(t *testing.T) {
	t.Run("without a baseline entry", func(t *testing.T) {
		dataRoot := t.TempDir()
		writeOverrideFile(t, dataRoot, canonicalFixturePath, canonicalBytes(t))

		report, err := OverrideStatusAt(dataRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		row := findRow(report.Rows, canonicalFixturePath)
		if row == nil {
			t.Fatalf("row not found in %v", report.Rows)
		}
		if row.Classification != ClassificationIdenticalRedundant {
			t.Errorf("Classification = %q, want %q", row.Classification, ClassificationIdenticalRedundant)
		}
	})

	t.Run("with a stale baseline entry", func(t *testing.T) {
		dataRoot := t.TempDir()
		writeOverrideFile(t, dataRoot, canonicalFixturePath, canonicalBytes(t))
		writeManifest(t, dataRoot, &OverrideBaselineManifest{
			SchemaVersion: 1,
			Baselines:     map[string]string{canonicalFixturePath: "deadbeef"},
		})

		report, err := OverrideStatusAt(dataRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		row := findRow(report.Rows, canonicalFixturePath)
		if row == nil {
			t.Fatalf("row not found in %v", report.Rows)
		}
		if row.Classification != ClassificationIdenticalRedundant {
			t.Errorf("Classification = %q, want %q", row.Classification, ClassificationIdenticalRedundant)
		}
	})
}

// TC-004: orphaned when there is no canonical counterpart.
func TestOverrideStatusAt_Orphaned(t *testing.T) {
	dataRoot := t.TempDir()
	writeOverrideFile(t, dataRoot, "no/such/canonical.md", []byte("orphan content"))

	report, err := OverrideStatusAt(dataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := findRow(report.Rows, "no/such/canonical.md")
	if row == nil {
		t.Fatalf("row not found in %v", report.Rows)
	}
	if row.Classification != ClassificationOrphaned {
		t.Errorf("Classification = %q, want %q", row.Classification, ClassificationOrphaned)
	}
	if row.CanonicalSHA256 != "" {
		t.Errorf("CanonicalSHA256 = %q, want empty", row.CanonicalSHA256)
	}
}

// TC-005: baseline_unknown for missing manifest, missing entry, corrupt JSON,
// and unrecognized schema_version — never silently treated as current.
func TestOverrideStatusAt_BaselineUnknown(t *testing.T) {
	t.Run("missing manifest file", func(t *testing.T) {
		dataRoot := t.TempDir()
		writeOverrideFile(t, dataRoot, canonicalFixturePath, []byte("differs from canonical"))

		report, err := OverrideStatusAt(dataRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		row := findRow(report.Rows, canonicalFixturePath)
		if row == nil || row.Classification != ClassificationBaselineUnknown {
			t.Fatalf("got row %+v, want classification %q", row, ClassificationBaselineUnknown)
		}
	})

	t.Run("manifest present but path has no entry", func(t *testing.T) {
		dataRoot := t.TempDir()
		writeOverrideFile(t, dataRoot, canonicalFixturePath, []byte("differs from canonical"))
		writeManifest(t, dataRoot, &OverrideBaselineManifest{SchemaVersion: 1, Baselines: map[string]string{"other/path.yaml": "abc"}})

		report, err := OverrideStatusAt(dataRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		row := findRow(report.Rows, canonicalFixturePath)
		if row == nil || row.Classification != ClassificationBaselineUnknown {
			t.Fatalf("got row %+v, want classification %q", row, ClassificationBaselineUnknown)
		}
	})

	t.Run("manifest JSON is corrupt", func(t *testing.T) {
		dataRoot := t.TempDir()
		writeOverrideFile(t, dataRoot, canonicalFixturePath, []byte("differs from canonical"))
		if err := os.WriteFile(filepath.Join(dataRoot, ".shark-override-baselines.json"), []byte("{not valid"), 0644); err != nil {
			t.Fatalf("failed to write corrupt manifest: %v", err)
		}

		report, err := OverrideStatusAt(dataRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		row := findRow(report.Rows, canonicalFixturePath)
		if row == nil || row.Classification != ClassificationBaselineUnknown {
			t.Fatalf("got row %+v, want classification %q", row, ClassificationBaselineUnknown)
		}
	})

	t.Run("manifest schema_version is not 1", func(t *testing.T) {
		dataRoot := t.TempDir()
		writeOverrideFile(t, dataRoot, canonicalFixturePath, []byte("differs from canonical"))
		body, _ := json.Marshal(map[string]interface{}{"schema_version": 99, "baselines": map[string]string{}})
		if err := os.WriteFile(filepath.Join(dataRoot, ".shark-override-baselines.json"), body, 0644); err != nil {
			t.Fatalf("failed to write manifest: %v", err)
		}

		report, err := OverrideStatusAt(dataRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		row := findRow(report.Rows, canonicalFixturePath)
		if row == nil || row.Classification != ClassificationBaselineUnknown {
			t.Fatalf("got row %+v, want classification %q", row, ClassificationBaselineUnknown)
		}
	})
}

// TC-006: a symlink under overrides/ classifies baseline_unknown with a
// suggested action naming the symlink problem, and its target is never read
// (sentinel content must never leak into any row field).
func TestOverrideStatusAt_SymlinkHandling(t *testing.T) {
	dataRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataRoot, "overrides"), 0755); err != nil {
		t.Fatalf("failed to create overrides dir: %v", err)
	}

	sentinel := filepath.Join(t.TempDir(), "sentinel-target.md")
	const sentinelContent = "SENTINEL-SECRET-CONTENT-MUST-NOT-LEAK"
	if err := os.WriteFile(sentinel, []byte(sentinelContent), 0644); err != nil {
		t.Fatalf("failed to write symlink target: %v", err)
	}

	linkPath := filepath.Join(dataRoot, "overrides", "linked.md")
	if err := os.Symlink(sentinel, linkPath); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}

	report, err := OverrideStatusAt(dataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := findRow(report.Rows, "linked.md")
	if row == nil {
		t.Fatalf("row not found in %v", report.Rows)
	}
	if row.Classification != ClassificationBaselineUnknown {
		t.Errorf("Classification = %q, want %q", row.Classification, ClassificationBaselineUnknown)
	}
	if row.SuggestedAction == "" {
		t.Error("expected a non-empty SuggestedAction naming the symlink problem")
	}
	for _, field := range []string{row.SuggestedAction, row.OverrideSHA256, row.CanonicalSHA256, row.BaselineSHA256} {
		if field == sentinelContent {
			t.Errorf("sentinel content leaked into row field: %q", field)
		}
	}
}

// TC-011: classification-order regression guard. A symlink whose target
// bytes would equal canonical bytes must still classify baseline_unknown
// (symlink check runs before the identical_redundant byte comparison), not
// identical_redundant.
func TestOverrideStatusAt_ClassificationOrder_SymlinkBeforeIdentical(t *testing.T) {
	dataRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataRoot, "overrides"), 0755); err != nil {
		t.Fatalf("failed to create overrides dir: %v", err)
	}

	// Target file's bytes are byte-identical to the canonical fixture, so if
	// classification order were wrong (byte-compare before symlink check),
	// this would misclassify as identical_redundant.
	targetDir := t.TempDir()
	target := filepath.Join(targetDir, "identical-target.yaml")
	if err := os.WriteFile(target, canonicalBytes(t), 0644); err != nil {
		t.Fatalf("failed to write symlink target: %v", err)
	}

	linkPath := filepath.Join(dataRoot, "overrides", canonicalFixturePath)
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		t.Fatalf("failed to create parent dir: %v", err)
	}
	if err := os.Symlink(target, linkPath); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}

	report, err := OverrideStatusAt(dataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := findRow(report.Rows, canonicalFixturePath)
	if row == nil {
		t.Fatalf("row not found in %v", report.Rows)
	}
	if row.Classification != ClassificationBaselineUnknown {
		t.Errorf("Classification = %q, want %q (symlink check must precede identical_redundant)", row.Classification, ClassificationBaselineUnknown)
	}
}

// TC-012: path-safety regression guard. normalizeOverrideRelPath must reject
// an absolute path or a path with a ".." segment before any join onto
// dataRoot — filepath.WalkDir itself won't produce these from a real
// directory tree, so this tests the normalization function directly with a
// hand-built input, per test-plan.md TC-012.
func TestNormalizeOverrideRelPath_RejectsUnsafePaths(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"absolute path", "/etc/passwd"},
		{"leading dotdot segment", "../outside.md"},
		{"embedded dotdot segment", "a/../../outside.md"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := normalizeOverrideRelPath(tc.in); err == nil {
				t.Errorf("normalizeOverrideRelPath(%q) = nil error, want rejection", tc.in)
			}
		})
	}

	if got, err := normalizeOverrideRelPath("workflow/epic.yaml"); err != nil {
		t.Errorf("normalizeOverrideRelPath(safe path) unexpected error: %v", err)
	} else if got != "workflow/epic.yaml" {
		t.Errorf("normalizeOverrideRelPath(safe path) = %q, want unchanged", got)
	}
}

// TC-013: REQ-NF-001 golden-output test — exact JSON field names and enum
// spellings, so an accidental rename fails CI rather than only at runtime.
func TestOverrideRow_JSONFieldNames_Golden(t *testing.T) {
	row := OverrideRow{
		Path:            "workflow/epic.yaml",
		Classification:  ClassificationCurrent,
		OverrideSHA256:  "aaa",
		CanonicalSHA256: "bbb",
		BaselineSHA256:  "ccc",
		SuggestedAction: "ddd",
	}

	data, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("failed to marshal OverrideRow: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal into map: %v", err)
	}

	want := map[string]interface{}{
		"path":             "workflow/epic.yaml",
		"classification":   "current",
		"override_sha256":  "aaa",
		"canonical_sha256": "bbb",
		"baseline_sha256":  "ccc",
		"suggested_action": "ddd",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("field %q = %v, want %v", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d fields, want %d: %v", len(got), len(want), got)
	}

	wantClassifications := []string{
		ClassificationCurrent,
		ClassificationUpstreamChanged,
		ClassificationIdenticalRedundant,
		ClassificationOrphaned,
		ClassificationBaselineUnknown,
	}
	wantSpellings := map[string]string{
		ClassificationCurrent:            "current",
		ClassificationUpstreamChanged:    "upstream_changed",
		ClassificationIdenticalRedundant: "identical_redundant",
		ClassificationOrphaned:           "orphaned",
		ClassificationBaselineUnknown:    "baseline_unknown",
	}
	for _, c := range wantClassifications {
		if c != wantSpellings[c] {
			t.Errorf("classification constant %q does not match golden spelling %q", c, wantSpellings[c])
		}
	}

	report := &OverrideStatusReport{Rows: []OverrideRow{row}, Summary: map[string]int{ClassificationCurrent: 1}}
	reportData, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("failed to marshal OverrideStatusReport: %v", err)
	}
	var envelope map[string]interface{}
	if err := json.Unmarshal(reportData, &envelope); err != nil {
		t.Fatalf("failed to unmarshal envelope: %v", err)
	}
	if _, ok := envelope["overrides"]; !ok {
		t.Error(`envelope missing "overrides" key`)
	}
	if _, ok := envelope["summary"]; !ok {
		t.Error(`envelope missing "summary" key`)
	}
}
