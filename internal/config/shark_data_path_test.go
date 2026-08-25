package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestGetSharkDataPath_DefaultAndParse covers the nil-safe accessor and the
// manager parse path for shark_data_path.
func TestGetSharkDataPath_DefaultAndParse(t *testing.T) {
	custom := "bundle/content"
	empty := ""
	tests := []struct {
		name string
		cfg  *Config
		want string
	}{
		{"nil config", nil, DefaultSharkDataPath},
		{"absent field", &Config{}, DefaultSharkDataPath},
		{"empty field", &Config{SharkDataPath: &empty}, DefaultSharkDataPath},
		{"custom field", &Config{SharkDataPath: &custom}, "bundle/content"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.GetSharkDataPath(); got != tt.want {
				t.Errorf("GetSharkDataPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestManagerLoad_ParsesSharkDataPath verifies Manager.Load populates the
// SharkDataPath field from raw JSON (and leaves it nil when absent so the
// default applies).
func TestManagerLoad_ParsesSharkDataPath(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantPath string
	}{
		{"present", `{"shark_data_path": "custom-bundle"}`, "custom-bundle"},
		{"absent defaults", `{}`, DefaultSharkDataPath},
		{"empty defaults", `{"shark_data_path": ""}`, DefaultSharkDataPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			cfgPath := filepath.Join(tmp, ".sharkconfig.json")
			if err := os.WriteFile(cfgPath, []byte(tt.json), 0644); err != nil {
				t.Fatal(err)
			}
			cfg, err := NewManager(cfgPath).Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got := cfg.GetSharkDataPath(); got != tt.wantPath {
				t.Errorf("GetSharkDataPath() = %q, want %q", got, tt.wantPath)
			}
		})
	}
}

// TestResolveSharkDataRoot covers default resolution, custom relative paths,
// absolute paths, and rejection of `..`-escapes.
func TestResolveSharkDataRoot(t *testing.T) {
	root := t.TempDir()
	absBundle := filepath.Join(t.TempDir(), "shared", "bundle")

	tests := []struct {
		name        string
		configBytes []byte
		want        string
		wantErr     bool
	}{
		{
			name:        "nil bytes defaults to shark-data",
			configBytes: nil,
			want:        filepath.Join(root, DefaultSharkDataPath),
		},
		{
			name:        "absent field defaults to shark-data",
			configBytes: []byte(`{}`),
			want:        filepath.Join(root, DefaultSharkDataPath),
		},
		{
			name:        "custom relative path",
			configBytes: []byte(`{"shark_data_path": "content/bundle"}`),
			want:        filepath.Join(root, "content", "bundle"),
		},
		{
			// A leading-dots component is an in-project dir name, not an
			// escape. Guards against the naive HasPrefix(rel, "..") check
			// that would wrongly reject it.
			name:        "in-project dir name with leading dots allowed",
			configBytes: []byte(`{"shark_data_path": "..foo/bundle"}`),
			want:        filepath.Join(root, "..foo", "bundle"),
		},
		{
			name:        "absolute path honored verbatim",
			configBytes: []byte(`{"shark_data_path": "` + absBundle + `"}`),
			want:        absBundle,
		},
		{
			name:        "escape via .. rejected",
			configBytes: []byte(`{"shark_data_path": "../escape"}`),
			wantErr:     true,
		},
		{
			name:        "deep escape via .. rejected",
			configBytes: []byte(`{"shark_data_path": "a/b/../../../escape"}`),
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveSharkDataRoot(root, tt.configBytes)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (resolved=%q)", got)
				}
				if !errors.Is(err, ErrSharkDataPathEscapes) {
					t.Errorf("expected ErrSharkDataPathEscapes, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Compare cleaned absolute forms (root from t.TempDir is already absolute).
			wantAbs, _ := filepath.Abs(tt.want)
			if got != filepath.Clean(wantAbs) {
				t.Errorf("ResolveSharkDataRoot() = %q, want %q", got, filepath.Clean(wantAbs))
			}
		})
	}
}

// TestResolveWorkflowDir_DerivesFromSharkDataPath verifies that when
// workflow_config is empty, the default workflow directory derives from
// <shark_data_path>/workflow rather than the hardcoded shark-data/workflow.
func TestResolveWorkflowDir_DerivesFromSharkDataPath(t *testing.T) {
	tmp := t.TempDir()
	// Custom bundle root with a workflow/ dir inside it.
	if err := os.MkdirAll(filepath.Join(tmp, "custom-bundle", "workflow"), 0755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"shark_data_path": "custom-bundle"}`

	dir, overrides := resolveWorkflowDir(tmp, []byte(cfg))
	want := filepath.Join(tmp, "custom-bundle", "workflow")
	if dir != want {
		t.Errorf("workflowDir = %q, want %q", dir, want)
	}
	wantOverrides := filepath.Join(tmp, "custom-bundle", "overrides", "workflow")
	if overrides != wantOverrides {
		t.Errorf("overridesDir = %q, want %q", overrides, wantOverrides)
	}
}

// TestResolveWorkflowDir_ExplicitWorkflowConfigWinsOverSharkDataPath verifies
// that an explicit workflow_config always takes precedence over the
// shark_data_path-derived default.
func TestResolveWorkflowDir_ExplicitWorkflowConfigWinsOverSharkDataPath(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "explicit", "wf"), 0755); err != nil {
		t.Fatal(err)
	}
	// Both fields set: workflow_config must win, shark_data_path must NOT
	// influence the resolved workflow dir.
	cfg := `{"shark_data_path": "custom-bundle", "workflow_config": "explicit/wf"}`

	dir, _ := resolveWorkflowDir(tmp, []byte(cfg))
	want := filepath.Join(tmp, "explicit", "wf")
	if dir != want {
		t.Errorf("workflowDir = %q, want %q (explicit workflow_config should win)", dir, want)
	}
}

// TestResolveWorkflowDir_DefaultUnaffectedWhenSharkDataPathAbsent confirms
// backward compatibility: with neither field set, the workflow dir is still
// <projectRoot>/shark-data/workflow.
func TestResolveWorkflowDir_DefaultUnaffectedWhenSharkDataPathAbsent(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "shark-data", "workflow"), 0755); err != nil {
		t.Fatal(err)
	}
	dir, _ := resolveWorkflowDir(tmp, []byte(`{}`))
	want := filepath.Join(tmp, "shark-data", "workflow")
	if dir != want {
		t.Errorf("workflowDir = %q, want %q", dir, want)
	}
}

// TestBundleSeparation_WorkflowConfigFileDoesNotDriveSharkDataRoot asserts the
// core separation invariant: when workflow_config points at a file under
// examples/ and shark_data_path is absent, the bundle root still resolves to
// <projectRoot>/shark-data — NOT the examples/ directory. This mirrors this
// repo's own .sharkconfig.json.
func TestBundleSeparation_WorkflowConfigFileDoesNotDriveSharkDataRoot(t *testing.T) {
	tmp := t.TempDir()
	cfg := `{"workflow_config": "examples/route-based-codex/workflow.yaml"}`

	dataRoot, err := ResolveSharkDataRoot(tmp, []byte(cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(tmp, DefaultSharkDataPath)
	if dataRoot != want {
		t.Errorf("bundle root = %q, want %q (workflow_config must not drive shark_data root)", dataRoot, want)
	}
}

// TestResolveSharkDataRoot_ExpandsHomePrefix verifies a "~/"-prefixed
// shark_data_path is expanded to the user's home directory (treated as an
// absolute shared-bundle root) rather than joined literally under projectRoot.
func TestResolveSharkDataRoot_ExpandsHomePrefix(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)        // unix: os.UserHomeDir() reads $HOME
	t.Setenv("USERPROFILE", home) // windows parity
	root := t.TempDir()

	got, err := ResolveSharkDataRoot(root, []byte(`{"shark_data_path": "~/shared/bundle"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(home, "shared", "bundle")
	if got != want {
		t.Errorf("ResolveSharkDataRoot() = %q, want %q (\"~/\" should expand to home, not %q)",
			got, want, filepath.Join(root, "~", "shared", "bundle"))
	}
}

// TestResolveSharkDataRoot_MalformedJSONFallsBackToDefault verifies that
// unparseable config bytes fall back to <projectRoot>/shark-data without error,
// so downstream path joins always get a usable bundle root.
func TestResolveSharkDataRoot_MalformedJSONFallsBackToDefault(t *testing.T) {
	root := t.TempDir()
	got, err := ResolveSharkDataRoot(root, []byte(`{ this is not json`))
	if err != nil {
		t.Fatalf("unexpected error on malformed JSON: %v", err)
	}
	want := filepath.Join(root, DefaultSharkDataPath)
	if got != want {
		t.Errorf("ResolveSharkDataRoot() = %q, want %q", got, want)
	}
}

// TestResolveWorkflowDir_AbsoluteSharkDataPath verifies the absolute-bundle
// branch: an absolute shark_data_path yields <bundle>/workflow directly,
// never joined under projectRoot.
func TestResolveWorkflowDir_AbsoluteSharkDataPath(t *testing.T) {
	projectRoot := t.TempDir()
	bundle := t.TempDir() // absolute, outside projectRoot
	if err := os.MkdirAll(filepath.Join(bundle, "workflow"), 0755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"shark_data_path": "` + bundle + `"}`

	dir, _ := resolveWorkflowDir(projectRoot, []byte(cfg))
	want := filepath.Join(bundle, "workflow")
	if dir != want {
		t.Errorf("workflowDir = %q, want %q (absolute bundle must not be joined under projectRoot)", dir, want)
	}
}
