package sharkdata

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TC-005 (manifest-load subtests): missing manifest, corrupt JSON, and wrong
// schema_version must all classify as invalid (ErrInvalidBaselineManifest),
// never silently upgraded or treated as valid.
func TestLoadOverrideBaselines(t *testing.T) {
	t.Run("missing file returns empty manifest, nil error", func(t *testing.T) {
		dataRoot := t.TempDir()

		manifest, err := LoadOverrideBaselines(dataRoot)

		if err != nil {
			t.Fatalf("expected nil error for missing manifest, got %v", err)
		}
		if manifest == nil {
			t.Fatal("expected non-nil manifest for missing file")
		}
		if manifest.SchemaVersion != 1 {
			t.Errorf("SchemaVersion = %d, want 1", manifest.SchemaVersion)
		}
		if manifest.Baselines == nil {
			t.Fatal("Baselines map should be initialized, not nil")
		}
		if len(manifest.Baselines) != 0 {
			t.Errorf("Baselines = %v, want empty map", manifest.Baselines)
		}
	})

	t.Run("corrupt JSON returns ErrInvalidBaselineManifest", func(t *testing.T) {
		dataRoot := t.TempDir()
		writeBaselineFile(t, dataRoot, "{not valid json")

		manifest, err := LoadOverrideBaselines(dataRoot)

		if !errors.Is(err, ErrInvalidBaselineManifest) {
			t.Fatalf("err = %v, want ErrInvalidBaselineManifest", err)
		}
		if manifest != nil {
			t.Errorf("manifest = %v, want nil on error", manifest)
		}
	})

	t.Run("wrong schema_version returns ErrInvalidBaselineManifest", func(t *testing.T) {
		dataRoot := t.TempDir()
		writeBaselineFile(t, dataRoot, `{"schema_version": 2, "baselines": {}}`)

		manifest, err := LoadOverrideBaselines(dataRoot)

		if !errors.Is(err, ErrInvalidBaselineManifest) {
			t.Fatalf("err = %v, want ErrInvalidBaselineManifest", err)
		}
		if manifest != nil {
			t.Errorf("manifest = %v, want nil on error", manifest)
		}
	})

	t.Run("valid manifest loads baselines", func(t *testing.T) {
		dataRoot := t.TempDir()
		writeBaselineFile(t, dataRoot, `{"schema_version": 1, "baselines": {"workflow/epic.yaml": "abc123"}}`)

		manifest, err := LoadOverrideBaselines(dataRoot)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if manifest.SchemaVersion != 1 {
			t.Errorf("SchemaVersion = %d, want 1", manifest.SchemaVersion)
		}
		if got := manifest.Baselines["workflow/epic.yaml"]; got != "abc123" {
			t.Errorf("Baselines[workflow/epic.yaml] = %q, want %q", got, "abc123")
		}
	})
}

// TC-013 (partial — this task's slice): golden schema_version/baselines key
// names for the manifest file, asserted via marshal-and-compare so an
// accidental rename fails this test, not just at runtime.
func TestOverrideBaselineManifest_JSONKeyNames(t *testing.T) {
	manifest := OverrideBaselineManifest{
		SchemaVersion: 1,
		Baselines:     map[string]string{"workflow/epic.yaml": "deadbeef"},
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var golden map[string]interface{}
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if _, ok := golden["schema_version"]; !ok {
		t.Error(`golden JSON missing "schema_version" key`)
	}
	if _, ok := golden["baselines"]; !ok {
		t.Error(`golden JSON missing "baselines" key`)
	}
	if len(golden) != 2 {
		t.Errorf("golden JSON has %d keys, want exactly 2 (schema_version, baselines): %v", len(golden), golden)
	}
}

// AC-T3: Save writes sorted-key indented JSON at
// <dataRoot>/.shark-override-baselines.json.
func TestOverrideBaselineManifest_Save(t *testing.T) {
	t.Run("writes sorted-key indented JSON", func(t *testing.T) {
		dataRoot := t.TempDir()
		manifest := &OverrideBaselineManifest{
			SchemaVersion: 1,
			Baselines: map[string]string{
				"workflow/epic.yaml":     "sha-b",
				"prompts/feature/a.md":   "sha-a",
				"skills/quality/tool.md": "sha-c",
			},
		}

		if err := manifest.Save(dataRoot); err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		path := filepath.Join(dataRoot, ".shark-override-baselines.json")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read saved manifest: %v", err)
		}

		expected, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			t.Fatalf("failed to build expected JSON: %v", err)
		}
		if string(raw) != string(expected) {
			t.Errorf("saved JSON = %s, want %s", raw, expected)
		}

		// Sorted-key: prompts/feature/a.md before skills/... before workflow/...
		idxA := indexOf(t, string(raw), "prompts/feature/a.md")
		idxS := indexOf(t, string(raw), "skills/quality/tool.md")
		idxW := indexOf(t, string(raw), "workflow/epic.yaml")
		if !(idxA < idxS && idxS < idxW) {
			t.Errorf("keys not sorted in output: prompts=%d skills=%d workflow=%d", idxA, idxS, idxW)
		}
	})

	t.Run("round-trips through Load", func(t *testing.T) {
		dataRoot := t.TempDir()
		manifest := &OverrideBaselineManifest{
			SchemaVersion: 1,
			Baselines:     map[string]string{"workflow/epic.yaml": "sha-x"},
		}
		if err := manifest.Save(dataRoot); err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		loaded, err := LoadOverrideBaselines(dataRoot)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		if got := loaded.Baselines["workflow/epic.yaml"]; got != "sha-x" {
			t.Errorf("loaded Baselines[workflow/epic.yaml] = %q, want %q", got, "sha-x")
		}
	})
}

// manifestPath returns the fixed baseline manifest path under dataRoot.
func manifestPath(dataRoot string) string {
	return filepath.Join(dataRoot, ".shark-override-baselines.json")
}

// readManifestBytesIfExists returns the raw bytes of the manifest file, or
// nil if it does not exist. Used to assert "writes nothing" — a manifest
// that did not exist before a failed acknowledge call must still not exist
// after, and a manifest that did exist must be byte-for-byte unchanged.
func readManifestBytesIfExists(t *testing.T, dataRoot string) []byte {
	t.Helper()
	data, err := os.ReadFile(manifestPath(dataRoot))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}
	return data
}

// TC-007: acknowledge success updates only the manifest — the returned
// report reclassifies the acknowledged path as current, the manifest's new
// entry equals the current canonical SHA-256, and the override file's bytes
// and os.Stat mtime are unchanged before/after (explicit byte + mtime
// comparison, not just "no error").
func TestAcknowledgeOverrides_Success(t *testing.T) {
	t.Run("baseline_unknown path becomes current", func(t *testing.T) {
		dataRoot := t.TempDir()
		overridePath := writeOverrideFile(t, dataRoot, canonicalFixturePath, []byte("locally modified, no manifest entry"))

		beforeBytes, err := os.ReadFile(overridePath)
		if err != nil {
			t.Fatalf("failed to read override file before acknowledge: %v", err)
		}
		beforeInfo, err := os.Stat(overridePath)
		if err != nil {
			t.Fatalf("failed to stat override file before acknowledge: %v", err)
		}

		// Sanity: this path starts baseline_unknown (no manifest at all).
		preReport, err := OverrideStatusAt(dataRoot)
		if err != nil {
			t.Fatalf("unexpected error from pre-check OverrideStatusAt: %v", err)
		}
		preRow := findRow(preReport.Rows, canonicalFixturePath)
		if preRow == nil || preRow.Classification != ClassificationBaselineUnknown {
			t.Fatalf("precondition failed: got row %+v, want classification %q", preRow, ClassificationBaselineUnknown)
		}

		report, err := AcknowledgeOverrides(dataRoot, []string{canonicalFixturePath})
		if err != nil {
			t.Fatalf("AcknowledgeOverrides failed: %v", err)
		}

		row := findRow(report.Rows, canonicalFixturePath)
		if row == nil {
			t.Fatalf("row for %q not found in %v", canonicalFixturePath, report.Rows)
		}
		if row.Classification != ClassificationCurrent {
			t.Errorf("Classification = %q, want %q", row.Classification, ClassificationCurrent)
		}

		wantSHA := sha256Hex(canonicalBytes(t))
		manifest, err := LoadOverrideBaselines(dataRoot)
		if err != nil {
			t.Fatalf("failed to reload manifest: %v", err)
		}
		if got := manifest.Baselines[canonicalFixturePath]; got != wantSHA {
			t.Errorf("manifest Baselines[%q] = %q, want %q", canonicalFixturePath, got, wantSHA)
		}

		afterBytes, err := os.ReadFile(overridePath)
		if err != nil {
			t.Fatalf("failed to read override file after acknowledge: %v", err)
		}
		if string(afterBytes) != string(beforeBytes) {
			t.Errorf("override file bytes changed: before=%q after=%q", beforeBytes, afterBytes)
		}
		afterInfo, err := os.Stat(overridePath)
		if err != nil {
			t.Fatalf("failed to stat override file after acknowledge: %v", err)
		}
		if !afterInfo.ModTime().Equal(beforeInfo.ModTime()) {
			t.Errorf("override file mtime changed: before=%v after=%v", beforeInfo.ModTime(), afterInfo.ModTime())
		}
	})

	t.Run("upstream_changed path becomes current", func(t *testing.T) {
		dataRoot := t.TempDir()
		overridePath := writeOverrideFile(t, dataRoot, canonicalFixturePath, []byte("locally modified, stale baseline"))
		writeManifest(t, dataRoot, &OverrideBaselineManifest{
			SchemaVersion: 1,
			Baselines:     map[string]string{canonicalFixturePath: "0000000000000000000000000000000000000000000000000000000000000000"[:64]}, //nolint:gosec // stale sentinel digest, not a real hash
		})

		beforeBytes, err := os.ReadFile(overridePath)
		if err != nil {
			t.Fatalf("failed to read override file before acknowledge: %v", err)
		}
		beforeInfo, err := os.Stat(overridePath)
		if err != nil {
			t.Fatalf("failed to stat override file before acknowledge: %v", err)
		}

		preReport, err := OverrideStatusAt(dataRoot)
		if err != nil {
			t.Fatalf("unexpected error from pre-check OverrideStatusAt: %v", err)
		}
		preRow := findRow(preReport.Rows, canonicalFixturePath)
		if preRow == nil || preRow.Classification != ClassificationUpstreamChanged {
			t.Fatalf("precondition failed: got row %+v, want classification %q", preRow, ClassificationUpstreamChanged)
		}

		report, err := AcknowledgeOverrides(dataRoot, []string{canonicalFixturePath})
		if err != nil {
			t.Fatalf("AcknowledgeOverrides failed: %v", err)
		}

		row := findRow(report.Rows, canonicalFixturePath)
		if row == nil {
			t.Fatalf("row for %q not found in %v", canonicalFixturePath, report.Rows)
		}
		if row.Classification != ClassificationCurrent {
			t.Errorf("Classification = %q, want %q", row.Classification, ClassificationCurrent)
		}

		wantSHA := sha256Hex(canonicalBytes(t))
		manifest, err := LoadOverrideBaselines(dataRoot)
		if err != nil {
			t.Fatalf("failed to reload manifest: %v", err)
		}
		if got := manifest.Baselines[canonicalFixturePath]; got != wantSHA {
			t.Errorf("manifest Baselines[%q] = %q, want %q", canonicalFixturePath, got, wantSHA)
		}

		afterBytes, err := os.ReadFile(overridePath)
		if err != nil {
			t.Fatalf("failed to read override file after acknowledge: %v", err)
		}
		if string(afterBytes) != string(beforeBytes) {
			t.Errorf("override file bytes changed: before=%q after=%q", beforeBytes, afterBytes)
		}
		afterInfo, err := os.Stat(overridePath)
		if err != nil {
			t.Fatalf("failed to stat override file after acknowledge: %v", err)
		}
		if !afterInfo.ModTime().Equal(beforeInfo.ModTime()) {
			t.Errorf("override file mtime changed: before=%v after=%v", beforeInfo.ModTime(), afterInfo.ModTime())
		}
	})
}

// TC-008: acknowledge failure — no canonical counterpart, or no override
// file at all — returns a non-nil error naming the path and writes nothing:
// the manifest is byte-for-byte unchanged (or still absent) before/after.
func TestAcknowledgeOverrides_Failure_NoPartialWrites(t *testing.T) {
	t.Run("path has no canonical counterpart", func(t *testing.T) {
		dataRoot := t.TempDir()
		const orphanPath = "no/such/canonical.md"
		writeOverrideFile(t, dataRoot, orphanPath, []byte("orphan content"))

		before := readManifestBytesIfExists(t, dataRoot)

		report, err := AcknowledgeOverrides(dataRoot, []string{orphanPath})

		if err == nil {
			t.Fatal("expected error for path with no canonical counterpart, got nil")
		}
		if !strings.Contains(err.Error(), orphanPath) {
			t.Errorf("error %q does not name path %q", err.Error(), orphanPath)
		}
		if report != nil {
			t.Errorf("report = %v, want nil on failure", report)
		}

		after := readManifestBytesIfExists(t, dataRoot)
		if before == nil && after != nil {
			t.Errorf("manifest was created by a failed acknowledge call: %s", after)
		}
		if before != nil && string(before) != string(after) {
			t.Errorf("manifest changed by a failed acknowledge call: before=%s after=%s", before, after)
		}
	})

	t.Run("path has no override file at all", func(t *testing.T) {
		dataRoot := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dataRoot, "overrides"), 0755); err != nil {
			t.Fatalf("failed to create overrides dir: %v", err)
		}
		writeManifest(t, dataRoot, &OverrideBaselineManifest{SchemaVersion: 1, Baselines: map[string]string{"other/path.yaml": "abc123"}})

		before := readManifestBytesIfExists(t, dataRoot)

		report, err := AcknowledgeOverrides(dataRoot, []string{canonicalFixturePath})

		if err == nil {
			t.Fatal("expected error for path with no override file, got nil")
		}
		if !strings.Contains(err.Error(), canonicalFixturePath) {
			t.Errorf("error %q does not name path %q", err.Error(), canonicalFixturePath)
		}
		if report != nil {
			t.Errorf("report = %v, want nil on failure", report)
		}

		after := readManifestBytesIfExists(t, dataRoot)
		if string(before) != string(after) {
			t.Errorf("manifest changed by a failed acknowledge call: before=%s after=%s", before, after)
		}
	})

	t.Run("one invalid path among several aborts the whole call with zero mutation", func(t *testing.T) {
		dataRoot := t.TempDir()
		writeOverrideFile(t, dataRoot, canonicalFixturePath, []byte("valid override, would succeed alone"))
		const orphanPath = "no/such/canonical.md"
		writeOverrideFile(t, dataRoot, orphanPath, []byte("orphan content"))

		before := readManifestBytesIfExists(t, dataRoot)

		report, err := AcknowledgeOverrides(dataRoot, []string{canonicalFixturePath, orphanPath})

		if err == nil {
			t.Fatal("expected error when one of several paths is invalid, got nil")
		}
		if report != nil {
			t.Errorf("report = %v, want nil on failure", report)
		}

		after := readManifestBytesIfExists(t, dataRoot)
		if before == nil && after != nil {
			t.Errorf("manifest was created despite one invalid path in a multi-path call: %s", after)
		}
		if before != nil && string(before) != string(after) {
			t.Errorf("manifest changed despite one invalid path in a multi-path call: before=%s after=%s", before, after)
		}
	})
}

func writeBaselineFile(t *testing.T, dataRoot, content string) {
	t.Helper()
	path := filepath.Join(dataRoot, ".shark-override-baselines.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture manifest: %v", err)
	}
}

func indexOf(t *testing.T, haystack, needle string) int {
	t.Helper()
	idx := -1
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			idx = i
			break
		}
	}
	if idx == -1 {
		t.Fatalf("needle %q not found in haystack", needle)
	}
	return idx
}
