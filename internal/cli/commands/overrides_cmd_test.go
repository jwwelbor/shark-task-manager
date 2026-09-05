package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
)

// overridesCmdCanonicalFixturePath is a real embedded canonical file used so
// sharkdata.ReadEmbedded resolves a genuine counterpart, matching the
// internal/sharkdata test fixtures' convention.
const overridesCmdCanonicalFixturePath = "workflow/epic.yaml"

// chdirToProjectRoot creates a temp dir with a minimal .sharkconfig.json (so
// cli.FindProjectRoot resolves to it and config.ResolveSharkDataRoot defaults
// to <dir>/shark-data), chdirs into it, and restores the original working
// directory on cleanup. Returns the temp dir and the resolved shark-data
// root.
func chdirToProjectRoot(t *testing.T) (projectRoot, dataRoot string) {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sharkconfig.json"), []byte(`{}`), 0644))

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	return dir, filepath.Join(dir, "shark-data")
}

// writeOverridesCmdOverrideFile writes an override file at
// <dataRoot>/overrides/<relPath>, creating parent directories as needed.
func writeOverridesCmdOverrideFile(t *testing.T, dataRoot, relPath string, data []byte) string {
	t.Helper()
	full := filepath.Join(dataRoot, "overrides", filepath.FromSlash(relPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0755))
	require.NoError(t, os.WriteFile(full, data, 0644))
	return full
}

func setJSONMode(t *testing.T, enabled bool) {
	t.Helper()
	orig := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = enabled
	t.Cleanup(func() { cli.GlobalConfig.JSON = orig })
}

// ============================================================================
// TC-001 at the CLI layer: empty/absent overrides/ directory.
// ============================================================================

func TestRunOverridesStatus_JSON_EmptyOverrides(t *testing.T) {
	// TC-001 has two variants: no overrides/ directory at all, and an
	// overrides/ directory that exists but is empty. Both must classify
	// identically (all-zero summary, no rows).
	t.Run("no overrides directory at all", func(t *testing.T) {
		chdirToProjectRoot(t)
		setJSONMode(t, true)

		var runErr error
		out := captureStdout(t, func() {
			runErr = runOverridesStatus(&cobra.Command{}, nil)
		})
		require.NoError(t, runErr)

		var report sharkdata.OverrideStatusReport
		require.NoError(t, json.Unmarshal([]byte(out), &report))

		assert.Empty(t, report.Rows)
		assert.Equal(t, map[string]int{
			sharkdata.ClassificationCurrent:            0,
			sharkdata.ClassificationUpstreamChanged:    0,
			sharkdata.ClassificationIdenticalRedundant: 0,
			sharkdata.ClassificationOrphaned:           0,
			sharkdata.ClassificationBaselineUnknown:    0,
		}, report.Summary)
	})

	t.Run("empty overrides directory", func(t *testing.T) {
		_, dataRoot := chdirToProjectRoot(t)
		require.NoError(t, os.MkdirAll(filepath.Join(dataRoot, "overrides"), 0755))
		setJSONMode(t, true)

		var runErr error
		out := captureStdout(t, func() {
			runErr = runOverridesStatus(&cobra.Command{}, nil)
		})
		require.NoError(t, runErr)

		var report sharkdata.OverrideStatusReport
		require.NoError(t, json.Unmarshal([]byte(out), &report))

		assert.Empty(t, report.Rows)
		assert.Equal(t, map[string]int{
			sharkdata.ClassificationCurrent:            0,
			sharkdata.ClassificationUpstreamChanged:    0,
			sharkdata.ClassificationIdenticalRedundant: 0,
			sharkdata.ClassificationOrphaned:           0,
			sharkdata.ClassificationBaselineUnknown:    0,
		}, report.Summary)
	})
}

func TestRunOverridesStatus_Human_EmptyOverrides(t *testing.T) {
	chdirToProjectRoot(t)
	setJSONMode(t, false)

	var runErr error
	out := captureStdout(t, func() {
		runErr = runOverridesStatus(&cobra.Command{}, nil)
	})
	require.NoError(t, runErr)

	assert.Contains(t, out, "current:             0")
	assert.Contains(t, out, "baseline_unknown:    0")
	assert.Contains(t, out, "(no overrides found)")
}

func TestRunOverridesStatus_JSON_ResolvesDataRootLikeUpgrade(t *testing.T) {
	// AC-T1: status resolves project + data root exactly as runSharkUpgrade
	// does (cli.FindProjectRoot() + config.ResolveSharkDataRoot). Verified
	// here by placing an override under a custom shark_data_path and
	// confirming status finds it.
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sharkconfig.json"),
		[]byte(`{"shark_data_path":"custom-bundle"}`), 0644))

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	customDataRoot := filepath.Join(dir, "custom-bundle")
	writeOverridesCmdOverrideFile(t, customDataRoot, overridesCmdCanonicalFixturePath, []byte("orphan-if-no-counterpart"))

	setJSONMode(t, true)
	var runErr error
	out := captureStdout(t, func() {
		runErr = runOverridesStatus(&cobra.Command{}, nil)
	})
	require.NoError(t, runErr)

	var report sharkdata.OverrideStatusReport
	require.NoError(t, json.Unmarshal([]byte(out), &report))
	require.Len(t, report.Rows, 1)
	assert.Equal(t, overridesCmdCanonicalFixturePath, report.Rows[0].Path)
}

// ============================================================================
// TC-007 at the CLI layer: acknowledge success.
// ============================================================================

func TestRunOverridesAcknowledge_JSON_Success(t *testing.T) {
	_, dataRoot := chdirToProjectRoot(t)
	writeOverridesCmdOverrideFile(t, dataRoot, overridesCmdCanonicalFixturePath, []byte("locally modified, no manifest entry"))

	setJSONMode(t, true)
	var runErr error
	out := captureStdout(t, func() {
		runErr = runOverridesAcknowledge(&cobra.Command{}, []string{overridesCmdCanonicalFixturePath})
	})
	require.NoError(t, runErr)

	var report sharkdata.OverrideStatusReport
	require.NoError(t, json.Unmarshal([]byte(out), &report))

	require.Len(t, report.Rows, 1)
	assert.Equal(t, sharkdata.ClassificationCurrent, report.Rows[0].Classification)

	manifest, err := sharkdata.LoadOverrideBaselines(dataRoot)
	require.NoError(t, err)
	assert.NotEmpty(t, manifest.Baselines[overridesCmdCanonicalFixturePath])
	assert.Equal(t, manifest.Baselines[overridesCmdCanonicalFixturePath], report.Rows[0].CanonicalSHA256)
}

func TestRunOverridesAcknowledge_Human_Success(t *testing.T) {
	_, dataRoot := chdirToProjectRoot(t)
	writeOverridesCmdOverrideFile(t, dataRoot, overridesCmdCanonicalFixturePath, []byte("locally modified, no manifest entry"))

	setJSONMode(t, false)
	var runErr error
	out := captureStdout(t, func() {
		runErr = runOverridesAcknowledge(&cobra.Command{}, []string{overridesCmdCanonicalFixturePath})
	})
	require.NoError(t, runErr)

	assert.Contains(t, out, "Acknowledged 1 override(s):")
	assert.Contains(t, out, overridesCmdCanonicalFixturePath)
	assert.Contains(t, out, "current:             1")
	// The acknowledged row is now "current", which carries no
	// SuggestedAction; the row line must not print a dangling " -- ".
	assert.Contains(t, out, "[current] "+overridesCmdCanonicalFixturePath+"\n")
	assert.NotContains(t, out, "[current] "+overridesCmdCanonicalFixturePath+" -- ")
}

// ============================================================================
// TC-008 at the CLI layer: acknowledge failure, no partial writes, non-zero
// exit (returned error, per AC-T2).
// ============================================================================

func TestRunOverridesAcknowledge_Failure_NoCanonicalCounterpart(t *testing.T) {
	_, dataRoot := chdirToProjectRoot(t)
	const orphanPath = "no/such/canonical.md"
	writeOverridesCmdOverrideFile(t, dataRoot, orphanPath, []byte("orphan content"))

	manifestPath := filepath.Join(dataRoot, ".shark-override-baselines.json")
	_, statErr := os.Stat(manifestPath)
	require.True(t, os.IsNotExist(statErr), "manifest should not exist before the failing call")

	setJSONMode(t, false)
	var runErr error
	out := captureStdout(t, func() {
		runErr = runOverridesAcknowledge(&cobra.Command{}, []string{orphanPath})
	})

	require.Error(t, runErr)
	assert.True(t, strings.Contains(runErr.Error(), orphanPath), "error %q must name the failing path %q", runErr.Error(), orphanPath)
	assert.Empty(t, out, "no report should be printed on a failed acknowledge")

	_, statErr = os.Stat(manifestPath)
	assert.True(t, os.IsNotExist(statErr), "manifest must not be created by a failed acknowledge call")
}

func TestRunOverridesAcknowledge_Failure_NoOverrideFile(t *testing.T) {
	_, dataRoot := chdirToProjectRoot(t)
	require.NoError(t, os.MkdirAll(filepath.Join(dataRoot, "overrides"), 0755))

	setJSONMode(t, true)
	var runErr error
	out := captureStdout(t, func() {
		runErr = runOverridesAcknowledge(&cobra.Command{}, []string{overridesCmdCanonicalFixturePath})
	})

	require.Error(t, runErr)
	assert.True(t, strings.Contains(runErr.Error(), overridesCmdCanonicalFixturePath),
		"error %q must name the failing path %q", runErr.Error(), overridesCmdCanonicalFixturePath)
	assert.Empty(t, out, "no report should be printed on a failed acknowledge")
}

// ============================================================================
// Command wiring
// ============================================================================

func TestOverridesCmd_RegisteredUnderAdmin(t *testing.T) {
	found := false
	for _, c := range adminCmd.Commands() {
		if c == overridesCmd {
			found = true
			break
		}
	}
	assert.True(t, found, "overridesCmd must be registered as a child of adminCmd")

	subNames := map[string]bool{}
	for _, c := range overridesCmd.Commands() {
		subNames[c.Name()] = true
	}
	assert.True(t, subNames["status"], "overridesCmd must have a status subcommand")
	assert.True(t, subNames["acknowledge"], "overridesCmd must have an acknowledge subcommand")
}
