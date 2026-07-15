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

// TestTC102_SharkAttackRoster_ReportsFieldSpecificInvalidRoster verifies that
// the production bundle validator rejects unsafe roster semantics.
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
			name: "unsafe memory root",
			mutate: func(t *testing.T, rosterPath string) {
				data, err := os.ReadFile(rosterPath)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(rosterPath, []byte(strings.Replace(string(data), "memory_root: docs/council", "memory_root: ../outside", 1)), 0644))
			},
			wantErr: "memory_root",
		},
		{
			name: "absolute inbox root",
			mutate: func(t *testing.T, rosterPath string) {
				data, err := os.ReadFile(rosterPath)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(rosterPath, []byte(strings.Replace(string(data), "inbox_root: docs/council/inbox", "inbox_root: /tmp/council", 1)), 0644))
			},
			wantErr: "communication.inbox_root",
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
			name: "status mutation responsibility",
			mutate: func(t *testing.T, rosterPath string) {
				data, err := os.ReadFile(rosterPath)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(rosterPath, []byte(strings.Replace(string(data), "- Implement scoped work", "- status_mutation", 1)), 0644))
			},
			wantErr: "responsibilities",
		},
		{
			name: "symlink escape",
			mutate: func(t *testing.T, rosterPath string) {
				externalPath := filepath.Join(t.TempDir(), "roster.yaml")
				require.NoError(t, os.WriteFile(externalPath, []byte("team: shark-attack\n"), 0644))
				require.NoError(t, os.Remove(rosterPath))
				require.NoError(t, os.Symlink(externalPath, rosterPath))
			},
			wantErr: "must not be a symlink",
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
