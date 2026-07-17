package sharkdata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTC101_SharkAttackRoster_ValidatesDefaultRoster exercises the production
// bundle validator after materializing the embedded shark-attack skill.
func TestTC101_SharkAttackRoster_ValidatesDefaultRoster(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	rosterPath := filepath.Join(root, SharkDataDirName, "skills", "shark-attack", "context", "roster-schema.yaml")
	data, err := os.ReadFile(rosterPath)
	require.NoError(t, err)
	for _, want := range []string{
		"team: shark-attack",
		"chair: tech-director",
		"id: developer",
		"persona: developer",
		"model_tier:",
	} {
		assert.Contains(t, string(data), want)
	}

	report, err := Validate(root)
	require.NoError(t, err)
	assert.False(t, report.HasErrors(), "default shark-attack roster must validate: %+v", report.Issues)
}

// TestSharkAttackRoster_ValidatesEffectiveOverride verifies that a project
// override replaces the default roster while retaining structural validation.
func TestSharkAttackRoster_ValidatesEffectiveOverride(t *testing.T) {
	const overrideRosterPath = "overrides/" + sharkAttackRosterPath

	t.Run("override is consulted before the default roster", func(t *testing.T) {
		root := t.TempDir()
		_, err := Init(root)
		require.NoError(t, err)

		defaultPath := filepath.Join(root, SharkDataDirName, filepath.FromSlash(sharkAttackRosterPath))
		defaultData, err := os.ReadFile(defaultPath)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(defaultPath, []byte(strings.Replace(string(defaultData), "chair: tech-director\n", "", 1)), 0644))

		overridePath := filepath.Join(root, SharkDataDirName, filepath.FromSlash(overrideRosterPath))
		require.NoError(t, os.MkdirAll(filepath.Dir(overridePath), 0755))
		override := strings.Replace(string(defaultData), "- Implement scoped work", "- Return bounded council evidence", 1)
		require.NoError(t, os.WriteFile(overridePath, []byte(override), 0644))

		report, err := Validate(root)
		require.NoError(t, err)
		assert.False(t, report.HasErrors(), "a structurally valid project override must validate: %+v", report.Issues)
	})

	t.Run("symlinked override is rejected", func(t *testing.T) {
		root := t.TempDir()
		_, err := Init(root)
		require.NoError(t, err)

		overridePath := filepath.Join(root, SharkDataDirName, filepath.FromSlash(overrideRosterPath))
		require.NoError(t, os.MkdirAll(filepath.Dir(overridePath), 0755))
		target := filepath.Join(root, "external-roster.yaml")
		require.NoError(t, os.WriteFile(target, []byte("team: shark-attack\n"), 0644))
		require.NoError(t, os.Symlink(target, overridePath))

		report, err := Validate(root)
		require.NoError(t, err)
		assert.True(t, report.HasErrors(), "symlinked overrides must fail validation")
		assertReportHasErrorContaining(t, report, "symlinks are not allowed")
	})
}

// TestTC102_SharkAttackRoster_ReportsFieldSpecificInvalidRoster verifies that
// the production bundle validator rejects malformed roster structure.
func TestTC102_SharkAttackRoster_ReportsFieldSpecificInvalidRoster(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(t *testing.T, rosterPath string)
		wantErr string
	}{
		{
			name: "missing chair",
			mutate: func(t *testing.T, rosterPath string) {
				data, err := os.ReadFile(rosterPath)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(rosterPath, []byte(strings.Replace(string(data), "chair: tech-director\n", "", 1)), 0644))
			},
			wantErr: "chair",
		},
		{
			name: "duplicate member id",
			mutate: func(t *testing.T, rosterPath string) {
				data, err := os.ReadFile(rosterPath)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(rosterPath, []byte(strings.Replace(string(data), "id: qa", "id: developer", 1)), 0644))
			},
			wantErr: "duplicates member",
		},
		{
			name: "empty responsibility",
			mutate: func(t *testing.T, rosterPath string) {
				data, err := os.ReadFile(rosterPath)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(rosterPath, []byte(strings.Replace(string(data), "- Implement scoped work", "- \"\"", 1)), 0644))
			},
			wantErr: "responsibilities[0] must not be empty",
		},
		{
			name: "unknown persona",
			mutate: func(t *testing.T, rosterPath string) {
				data, err := os.ReadFile(rosterPath)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(rosterPath, []byte(strings.Replace(string(data), "persona: developer", "persona: unknown-persona", 1)), 0644))
			},
			wantErr: "members[5].persona",
		},
		{
			name: "missing acknowledgement flag",
			mutate: func(t *testing.T, rosterPath string) {
				data, err := os.ReadFile(rosterPath)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(rosterPath, []byte(strings.Replace(string(data), "  acknowledge_after_read: true\n", "", 1)), 0644))
			},
			wantErr: "communication.acknowledge_after_read is required",
		},
		{
			name: "unsupported top-level field",
			mutate: func(t *testing.T, rosterPath string) {
				data, err := os.ReadFile(rosterPath)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(rosterPath, append(data, []byte("unsupported: true\n")...), 0644))
			},
			wantErr: "unsupported is not a supported roster field",
		},
		{
			name: "non-canonical council root",
			mutate: func(t *testing.T, rosterPath string) {
				data, err := os.ReadFile(rosterPath)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(rosterPath, []byte(strings.Replace(string(data), "memory_root: docs/council", "memory_root: project/coordination", 1)), 0644))
			},
			wantErr: "memory_root must equal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			_, err := Init(root)
			require.NoError(t, err)

			rosterPath := filepath.Join(root, SharkDataDirName, "skills", "shark-attack", "context", "roster-schema.yaml")
			tt.mutate(t, rosterPath)

			report, err := Validate(root)
			require.NoError(t, err)
			assert.True(t, report.HasErrors(), "invalid roster must fail validation: %+v", report.Issues)
			assertReportHasErrorContaining(t, report, tt.wantErr)
		})
	}
}
