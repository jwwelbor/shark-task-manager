package sharkdata

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
