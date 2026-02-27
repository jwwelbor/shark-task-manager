package commands

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/status"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProgressRequest(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		epicFlag    string
		wantEpicKey string
		wantErr     bool
	}{
		{"no args", []string{}, "", "", false},
		{"epic positional", []string{"E05"}, "", "E05", false},
		{"epic positional overrides flag", []string{"E05"}, "E07", "E05", false},
		{"epic flag only", []string{}, "E05", "E05", false},
		{"combined feature format", []string{"E05-F02"}, "", "E05", false},
		{"too many args", []string{"E05", "F02", "extra"}, "", "", true},
		{"lowercase epic normalized", []string{"e05"}, "", "E05", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("epic", "", "")
			cmd.Flags().String("recent", "", "")
			cmd.Flags().Bool("include-archived", false, "")
			if tt.epicFlag != "" {
				err := cmd.Flags().Set("epic", tt.epicFlag)
				require.NoError(t, err)
			}

			req, err := parseProgressRequest(cmd, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantEpicKey, req.EpicKey)
		})
	}
}

func TestParseProgressRequest_Flags(t *testing.T) {
	t.Run("recent window flag", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("epic", "", "")
		cmd.Flags().String("recent", "", "")
		cmd.Flags().Bool("include-archived", false, "")
		err := cmd.Flags().Set("recent", "7d")
		require.NoError(t, err)

		req, err := parseProgressRequest(cmd, []string{})
		assert.NoError(t, err)
		assert.Equal(t, "7d", req.RecentWindow)
	})

	t.Run("include-archived flag", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("epic", "", "")
		cmd.Flags().String("recent", "", "")
		cmd.Flags().Bool("include-archived", false, "")
		err := cmd.Flags().Set("include-archived", "true")
		require.NoError(t, err)

		req, err := parseProgressRequest(cmd, []string{})
		assert.NoError(t, err)
		assert.True(t, req.IncludeArchived)
	})
}

func TestOutputProgressJSON(t *testing.T) {
	dashboard := &status.StatusDashboard{
		Summary: &status.ProjectSummary{
			OverallProgress: 75.0,
		},
		Epics:        []*status.EpicSummary{},
		ActiveTasks:  map[string][]*status.TaskInfo{},
		BlockedTasks: []*status.BlockedTaskInfo{},
	}

	// Capture stdout
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	outputErr := outputProgressJSON(dashboard)

	err = w.Close()
	require.NoError(t, err)
	os.Stdout = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	output := buf.String()

	assert.NoError(t, outputErr)
	assert.Contains(t, output, `"overall_progress": 75`)
	assert.Contains(t, output, `"epics"`)
}

func TestRunProgress_EndToEnd(t *testing.T) {
	t.Skip("Blocked on mock injection for GetStatusService -- matches status_test.go precedent")
}
