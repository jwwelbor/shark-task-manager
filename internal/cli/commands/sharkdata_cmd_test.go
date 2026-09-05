package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// ensureWorkflowConfigField
// ============================================================================

func TestEnsureWorkflowConfigField_CreatesConfigWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	updated, migratedFrom, err := ensureWorkflowConfigField(dir, defaultWorkflowConfigDir)
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Empty(t, migratedFrom)

	configPath := filepath.Join(dir, ".sharkconfig.json")
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, defaultWorkflowConfigDir, raw["workflow_config"])
}

func TestEnsureWorkflowConfigField_AddsFieldToExistingConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"some_other_key":"value"}`), 0644))

	updated, migratedFrom, err := ensureWorkflowConfigField(dir, defaultWorkflowConfigDir)
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Empty(t, migratedFrom)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, defaultWorkflowConfigDir, raw["workflow_config"])
	// Existing key must be preserved.
	assert.Equal(t, "value", raw["some_other_key"])
}

func TestEnsureWorkflowConfigField_MigratesExplicitJSONWorkflowConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")
	jsonWorkflowPath := "legacy/workflow.json"
	require.NoError(t, os.WriteFile(configPath,
		[]byte(`{"workflow_config":"`+jsonWorkflowPath+`"}`), 0644))

	updated, migratedFrom, err := ensureWorkflowConfigField(dir, defaultWorkflowConfigDir)
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, jsonWorkflowPath, migratedFrom)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, defaultWorkflowConfigDir, raw["workflow_config"])
	assert.Equal(t, "shark-data", raw["shark_data_path"])
}

func TestEnsureWorkflowConfigField_MigratesSharkWorkflowTarget(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")
	jsonWorkflowPath := ".sharkworkflow.json"
	require.NoError(t, os.WriteFile(configPath,
		[]byte(`{"workflow_config":"`+jsonWorkflowPath+`","shark_data_path":"shark-data/"}`), 0644))

	updated, migratedFrom, err := ensureWorkflowConfigField(dir, defaultWorkflowConfigDir)
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, jsonWorkflowPath, migratedFrom)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, defaultWorkflowConfigDir, raw["workflow_config"])
	assert.Equal(t, "shark-data/", raw["shark_data_path"])
}

func TestEnsureWorkflowConfigField_RespectsYAMLSharkWorkflowIndex(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")
	indexPath := ".sharkworkflow.yaml"
	require.NoError(t, os.WriteFile(configPath,
		[]byte(`{"workflow_config":"`+indexPath+`","shark_data_path":"shark-data/"}`), 0644))

	updated, migratedFrom, err := ensureWorkflowConfigField(dir, defaultWorkflowConfigDir)
	require.NoError(t, err)
	assert.False(t, updated)
	assert.Empty(t, migratedFrom)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, indexPath, raw["workflow_config"])
}

func TestEnsureWorkflowConfigField_RespectsCustomPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")
	customPath := "my-custom-workflows/"
	// Include shark_data_path so the function has nothing to add and truly
	// returns updated=false when it finds a non-legacy workflow_config.
	require.NoError(t, os.WriteFile(configPath,
		[]byte(`{"workflow_config":"`+customPath+`","shark_data_path":"shark-data/"}`), 0644))

	updated, migratedFrom, err := ensureWorkflowConfigField(dir, defaultWorkflowConfigDir)
	require.NoError(t, err)
	assert.False(t, updated)
	assert.Empty(t, migratedFrom)

	// Config must be untouched.
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, customPath, raw["workflow_config"])
}

func TestEnsureWorkflowConfigField_IdempotentOnAlreadyCorrectPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")
	// Include shark_data_path so no field is missing and the function has
	// nothing to add — both fields are already correct.
	require.NoError(t, os.WriteFile(configPath,
		[]byte(`{"workflow_config":"`+defaultWorkflowConfigDir+`","shark_data_path":"shark-data/"}`), 0644))

	// Both fields are present and non-legacy, so the function should leave
	// the config untouched (false, "", nil).
	updated, migratedFrom, err := ensureWorkflowConfigField(dir, defaultWorkflowConfigDir)
	require.NoError(t, err)
	assert.False(t, updated)
	assert.Empty(t, migratedFrom)
}

func TestWorkflowConfigDirForDataRoot_ProjectRelative(t *testing.T) {
	dir := t.TempDir()

	got := workflowConfigDirForDataRoot(dir, filepath.Join(dir, "custom-bundle"))

	assert.Equal(t, "custom-bundle/workflow/", got)
}

func TestRunSharkInstallData_JSONMigratesDeprecatedConfigWithCustomBundle(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")
	require.NoError(t, os.WriteFile(configPath,
		[]byte(`{"shark_data_path":"custom-bundle","workflow_config":"legacy/workflow.json"}`), 0644))

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	t.Cleanup(func() { cli.GlobalConfig.JSON = origJSON })

	var runErr error
	out := captureStdout(t, func() {
		runErr = runSharkInstallData(&cobra.Command{}, nil)
	})
	require.NoError(t, runErr)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &payload))
	assert.Equal(t, "extracted", payload["status"])
	assert.Equal(t, "legacy/workflow.json", payload["migrated_from"])
	assert.Equal(t, true, payload["config_updated"])

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, "custom-bundle/workflow/", raw["workflow_config"])
	assert.Equal(t, "custom-bundle", raw["shark_data_path"])

	_, err = os.Stat(filepath.Join(dir, "custom-bundle", "workflow", "task.yaml"))
	require.NoError(t, err)
}

func TestRunSharkInstallData_JSONMigratesDeprecatedConfigWithAbsoluteBundle(t *testing.T) {
	dir := t.TempDir()
	bundleDir := filepath.Join(t.TempDir(), "shared-bundle")
	configPath := filepath.Join(dir, ".sharkconfig.json")
	require.NoError(t, os.WriteFile(configPath,
		[]byte(`{"shark_data_path":"`+filepath.ToSlash(bundleDir)+`","workflow_config":"legacy/workflow.json"}`), 0644))

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	t.Cleanup(func() { cli.GlobalConfig.JSON = origJSON })

	var runErr error
	out := captureStdout(t, func() {
		runErr = runSharkInstallData(&cobra.Command{}, nil)
	})
	require.NoError(t, runErr)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &payload))
	assert.Equal(t, "extracted", payload["status"])
	assert.Equal(t, "legacy/workflow.json", payload["migrated_from"])
	assert.Equal(t, true, payload["config_updated"])

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	expectedWorkflowConfig := filepath.ToSlash(filepath.Join(bundleDir, "workflow")) + "/"
	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, expectedWorkflowConfig, raw["workflow_config"])
	assert.Equal(t, filepath.ToSlash(bundleDir), raw["shark_data_path"])

	_, err = os.Stat(filepath.Join(bundleDir, "workflow", "task.yaml"))
	require.NoError(t, err)
}

func TestPrintConfigUpdateMessage_MigrationMentionsTarget(t *testing.T) {
	out := captureStdout(t, func() {
		printConfigUpdateMessage(true, "legacy/workflow.json", "custom-bundle/workflow/")
	})

	assert.True(t, strings.Contains(out, `deprecated JSON target "legacy/workflow.json"`))
	assert.True(t, strings.Contains(out, `"custom-bundle/workflow/"`))
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return string(out)
}

// captureStdoutAndStderr captures both streams for a single call, used by
// the OverrideStatusAt-failure regression test where the warning goes to
// stderr while the (still-successful) upgrade summary goes to stdout.
func captureStdoutAndStderr(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()

	origOut, origErr := os.Stdout, os.Stderr
	rOut, wOut, err := os.Pipe()
	require.NoError(t, err)
	rErr, wErr, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout, os.Stderr = wOut, wErr
	defer func() { os.Stdout, os.Stderr = origOut, origErr }()

	fn()

	require.NoError(t, wOut.Close())
	require.NoError(t, wErr.Close())
	outBytes, err := io.ReadAll(rOut)
	require.NoError(t, err)
	errBytes, err := io.ReadAll(rErr)
	require.NoError(t, err)
	require.NoError(t, rOut.Close())
	require.NoError(t, rErr.Close())
	return string(outBytes), string(errBytes)
}

// ============================================================================
// runSharkUpgrade -- overrides summary (T-E34-F09-005, TC-009/TC-010)
// ============================================================================

// Canonical fixture paths reused across the mixed-classification fixture
// below. Each is a real embedded file so sharkdata.ReadEmbedded resolves a
// genuine counterpart, matching the internal/sharkdata test fixtures'
// convention (see overrides_cmd_test.go's overridesCmdCanonicalFixturePath).
const (
	upgradeCurrentFixturePath         = "workflow/epic.yaml"
	upgradeUpstreamChangedFixturePath = "workflow/task.yaml"
	upgradeIdenticalFixturePath       = "workflow/bug.yaml"
	upgradeBaselineUnknownFixturePath = "workflow/change.yaml"
	upgradeOrphanedFixturePath        = "not/a/real/canonical.md"
)

func sha256HexForTest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// setupUpgradeFixtureRoot creates a temp project root with a minimal
// .sharkconfig.json, materializes shark-data/ via sharkdata.InitAt (UpgradeAt
// refuses to run against a missing destination), chdirs into the project
// root, and restores the original working directory on cleanup.
func setupUpgradeFixtureRoot(t *testing.T) (projectRoot, dataRoot string) {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sharkconfig.json"), []byte(`{}`), 0644))

	dataRoot = filepath.Join(dir, "shark-data")
	_, err := sharkdata.InitAt(dataRoot)
	require.NoError(t, err)

	// InitAt materializes the embedded overrides/.gitkeep scaffold file,
	// which OverrideStatusAt classifies as "orphaned" (no canonical
	// counterpart at that relative path) -- pre-existing behavior of
	// overrides_status.go, unrelated to this task. Remove it so these
	// fixtures start from a genuinely empty overrides/ directory.
	require.NoError(t, os.Remove(filepath.Join(dataRoot, "overrides", ".gitkeep")))

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	return dir, dataRoot
}

// buildMixedClassificationOverridesFixture writes one override file for each
// of the five drift classifications under dataRoot/overrides/, per TC-009's
// "fixture with a mix of all five classifications". Returns the expected
// summary counts.
func buildMixedClassificationOverridesFixture(t *testing.T, dataRoot string) map[string]int {
	t.Helper()

	epicCanonical, err := sharkdata.ReadEmbedded(upgradeCurrentFixturePath)
	require.NoError(t, err)
	taskCanonical, err := sharkdata.ReadEmbedded(upgradeUpstreamChangedFixturePath)
	require.NoError(t, err)
	bugCanonical, err := sharkdata.ReadEmbedded(upgradeIdenticalFixturePath)
	require.NoError(t, err)

	// current: override differs from canonical; recorded baseline equals the
	// current canonical digest (canonical has not moved since acknowledge).
	writeOverridesCmdOverrideFile(t, dataRoot, upgradeCurrentFixturePath,
		append([]byte("locally customized (current): "), epicCanonical...))
	// upstream_changed: override differs from canonical; recorded baseline is
	// stale relative to the current canonical digest.
	writeOverridesCmdOverrideFile(t, dataRoot, upgradeUpstreamChangedFixturePath,
		append([]byte("locally customized (upstream_changed): "), taskCanonical...))
	// identical_redundant: override bytes exactly equal canonical bytes.
	writeOverridesCmdOverrideFile(t, dataRoot, upgradeIdenticalFixturePath, bugCanonical)
	// baseline_unknown: override differs from canonical; no manifest entry.
	writeOverridesCmdOverrideFile(t, dataRoot, upgradeBaselineUnknownFixturePath,
		[]byte("no baseline entry for this one"))
	// orphaned: no canonical counterpart exists at this relative path.
	writeOverridesCmdOverrideFile(t, dataRoot, upgradeOrphanedFixturePath,
		[]byte("orphaned override, no canonical counterpart"))

	manifest := &sharkdata.OverrideBaselineManifest{
		SchemaVersion: 1,
		Baselines: map[string]string{
			upgradeCurrentFixturePath:         sha256HexForTest(epicCanonical),
			upgradeUpstreamChangedFixturePath: strings.Repeat("d", 64), // deliberately stale digest
		},
	}
	require.NoError(t, manifest.Save(dataRoot))

	return map[string]int{
		sharkdata.ClassificationCurrent:            1,
		sharkdata.ClassificationUpstreamChanged:    1,
		sharkdata.ClassificationIdenticalRedundant: 1,
		sharkdata.ClassificationOrphaned:           1,
		sharkdata.ClassificationBaselineUnknown:    1,
	}
}

// listOverridePaths returns the sorted, POSIX-relative set of regular file
// paths under <dataRoot>/overrides/, used to assert the override inventory
// (the set of paths) is unchanged across a real upgrade (TC-010).
func listOverridePaths(t *testing.T, dataRoot string) []string {
	t.Helper()

	overridesDir := filepath.Join(dataRoot, "overrides")
	var paths []string
	err := filepath.WalkDir(overridesDir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(overridesDir, p)
		if relErr != nil {
			return relErr
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	require.NoError(t, err)
	sort.Strings(paths)
	return paths
}

// withUpgradeDryRun sets the package-level upgradeDryRun flag var for the
// duration of the test (runSharkUpgrade reads it directly, not via cobra
// flag parsing in these unit tests) and restores it on cleanup.
func withUpgradeDryRun(t *testing.T, dryRun bool) {
	t.Helper()
	orig := upgradeDryRun
	upgradeDryRun = dryRun
	t.Cleanup(func() { upgradeDryRun = orig })
}

// TestRunSharkUpgrade_JSON_OverridesSummary_MixedClassifications is TC-009's
// first subtest: a real (non-dry-run) `shark admin upgrade --json` against a
// fixture with all five classifications must report all five counts, and
// must not disturb the four pre-existing keys' presence/shape (AC-T1).
func TestRunSharkUpgrade_JSON_OverridesSummary_MixedClassifications(t *testing.T) {
	_, dataRoot := setupUpgradeFixtureRoot(t)
	expectedCounts := buildMixedClassificationOverridesFixture(t, dataRoot)

	setJSONMode(t, true)
	withUpgradeDryRun(t, false)

	var runErr error
	out := captureStdout(t, func() {
		runErr = runSharkUpgrade(&cobra.Command{}, nil)
	})
	require.NoError(t, runErr)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &payload))

	for _, key := range []string{"added", "updated", "unchanged", "skipped_overrides"} {
		val, present := payload[key]
		require.True(t, present, "pre-existing key %q must still be present", key)
		// AC-T1 requires the four pre-existing keys "unchanged" in shape, not
		// just present -- assert each still serializes as list-shaped (a JSON
		// array, or null for an empty Go nil slice -- encoding/json's normal
		// rendering of []string(nil)), not, e.g., silently narrowed to a bare
		// count.
		if val == nil {
			continue
		}
		_, isArray := val.([]interface{})
		assert.True(t, isArray, "pre-existing key %q must still serialize as an array (or null), got %T", key, val)
	}

	overridesRaw, ok := payload["overrides"].(map[string]interface{})
	require.True(t, ok, `payload must contain an "overrides" object`)
	assert.Len(t, overridesRaw, 5, "overrides object must always carry all five classification keys")

	for classification, expectedCount := range expectedCounts {
		gotCount, present := overridesRaw[classification]
		require.True(t, present, "overrides summary missing key %q", classification)
		assert.Equal(t, float64(expectedCount), gotCount, "unexpected count for %q", classification)
	}
}

// TestRunSharkUpgrade_JSON_DryRun_OverridesSummary is TC-009's second
// subtest: `--dry-run --json` must report the same populated overrides
// object (OverrideStatusAt is read-only, so no branching is needed).
func TestRunSharkUpgrade_JSON_DryRun_OverridesSummary(t *testing.T) {
	_, dataRoot := setupUpgradeFixtureRoot(t)
	expectedCounts := buildMixedClassificationOverridesFixture(t, dataRoot)

	setJSONMode(t, true)
	withUpgradeDryRun(t, true)

	var runErr error
	out := captureStdout(t, func() {
		runErr = runSharkUpgrade(&cobra.Command{}, nil)
	})
	require.NoError(t, runErr)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &payload))
	assert.Equal(t, true, payload["dry_run"])

	overridesRaw, ok := payload["overrides"].(map[string]interface{})
	require.True(t, ok, `payload must contain an "overrides" object`)

	for classification, expectedCount := range expectedCounts {
		gotCount, present := overridesRaw[classification]
		require.True(t, present, "overrides summary missing key %q", classification)
		assert.Equal(t, float64(expectedCount), gotCount, "unexpected count for %q", classification)
	}
}

// TestRunSharkUpgrade_JSON_ZeroOverrides_AllZeroCounts is TC-009's third
// subtest: a project with zero overrides must still carry all five keys,
// all zero-valued -- not an omitted field.
func TestRunSharkUpgrade_JSON_ZeroOverrides_AllZeroCounts(t *testing.T) {
	setupUpgradeFixtureRoot(t)

	setJSONMode(t, true)
	withUpgradeDryRun(t, false)

	var runErr error
	out := captureStdout(t, func() {
		runErr = runSharkUpgrade(&cobra.Command{}, nil)
	})
	require.NoError(t, runErr)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &payload))

	overridesRaw, ok := payload["overrides"].(map[string]interface{})
	require.True(t, ok, `payload must contain an "overrides" object even with zero overrides`)

	for _, classification := range []string{
		sharkdata.ClassificationCurrent,
		sharkdata.ClassificationUpstreamChanged,
		sharkdata.ClassificationIdenticalRedundant,
		sharkdata.ClassificationOrphaned,
		sharkdata.ClassificationBaselineUnknown,
	} {
		gotCount, present := overridesRaw[classification]
		require.True(t, present, "overrides summary missing key %q", classification)
		assert.Equal(t, float64(0), gotCount, "expected zero count for %q", classification)
	}
}

// TestRunSharkUpgrade_Human_OverridesSummaryLine asserts the human-readable
// upgrade output gains the new "overrides:" line pointing at the detail
// command, without disturbing the four pre-existing summary lines.
func TestRunSharkUpgrade_Human_OverridesSummaryLine(t *testing.T) {
	_, dataRoot := setupUpgradeFixtureRoot(t)
	buildMixedClassificationOverridesFixture(t, dataRoot)

	setJSONMode(t, false)
	withUpgradeDryRun(t, false)

	var runErr error
	out := captureStdout(t, func() {
		runErr = runSharkUpgrade(&cobra.Command{}, nil)
	})
	require.NoError(t, runErr)

	assert.Contains(t, out, "added:")
	assert.Contains(t, out, "updated:")
	assert.Contains(t, out, "unchanged:")
	assert.Contains(t, out, "overrides skipped:")
	assert.Contains(t, out, "overrides:")
	assert.Contains(t, out, "shark admin overrides status")
	assert.Contains(t, out, sharkdata.ClassificationCurrent+"=1")
	assert.Contains(t, out, sharkdata.ClassificationUpstreamChanged+"=1")
	assert.Contains(t, out, sharkdata.ClassificationIdenticalRedundant+"=1")
	assert.Contains(t, out, sharkdata.ClassificationOrphaned+"=1")
	assert.Contains(t, out, sharkdata.ClassificationBaselineUnknown+"=1")

	// Spec.md: the new line is appended "after the existing four summary
	// lines" -- assert ordering, not just presence.
	skippedIdx := strings.Index(out, "overrides skipped:")
	overridesLineIdx := strings.Index(out, "overrides: "+sharkdata.ClassificationCurrent)
	require.NotEqual(t, -1, skippedIdx)
	require.NotEqual(t, -1, overridesLineIdx)
	assert.Less(t, skippedIdx, overridesLineIdx, "overrides summary line must come after the four pre-existing summary lines")
}

// TestRunSharkUpgrade_FreshInit_ReportsGitkeepScaffoldAsOrphaned documents a
// real, user-visible consequence of wiring OverrideStatusAt into upgrade:
// InitAt materializes the embedded overrides/.gitkeep scaffold file (so a
// fresh overrides/ directory exists on disk), and OverrideStatusAt has no
// canonical counterpart at that relative path, so it classifies .gitkeep as
// "orphaned" (existing overrides_status.go behavior, unrelated to this
// task). This means EVERY freshly-initialized project reports
// orphaned=1 out of the box -- a feature-level finding worth flagging to
// QA/UAT (the suggested_action text -- "consider removing this override" --
// is misleading for a scaffold file shark placed deliberately). This test
// pins that reality so a future change to either InitAt's scaffold or
// OverrideStatusAt's classification is a deliberate, visible decision.
func TestRunSharkUpgrade_FreshInit_ReportsGitkeepScaffoldAsOrphaned(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sharkconfig.json"), []byte(`{}`), 0644))
	dataRoot := filepath.Join(dir, "shark-data")
	_, err := sharkdata.InitAt(dataRoot) // deliberately NOT removing .gitkeep, unlike setupUpgradeFixtureRoot
	require.NoError(t, err)

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	setJSONMode(t, true)
	withUpgradeDryRun(t, false)

	var runErr error
	out := captureStdout(t, func() {
		runErr = runSharkUpgrade(&cobra.Command{}, nil)
	})
	require.NoError(t, runErr)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &payload))

	overridesRaw, ok := payload["overrides"].(map[string]interface{})
	require.True(t, ok, `payload must contain an "overrides" object`)
	assert.Equal(t, float64(1), overridesRaw[sharkdata.ClassificationOrphaned],
		"a freshly-initialized project's bundled overrides/.gitkeep scaffold classifies as orphaned")

	statusReport, err := sharkdata.OverrideStatusAt(dataRoot)
	require.NoError(t, err)
	require.Len(t, statusReport.Rows, 1)
	assert.Equal(t, ".gitkeep", statusReport.Rows[0].Path)
	assert.Equal(t, sharkdata.ClassificationOrphaned, statusReport.Rows[0].Classification)
}

// TestRunSharkUpgrade_OverridesStatusFailure_DoesNotDropUpgradeOutput pins
// the fallback behavior when OverrideStatusAt itself fails (e.g. an
// unreadable subdirectory under overrides/): a real upgrade has already
// written files by that point, so the command must still succeed, still
// print the four pre-existing summary lines, and still emit a schema-stable
// (all-zero) "overrides" line -- with the failure surfaced as a stderr
// warning instead of aborting the command.
func TestRunSharkUpgrade_OverridesStatusFailure_DoesNotDropUpgradeOutput(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission-based fixture requires a non-root user")
	}

	_, dataRoot := setupUpgradeFixtureRoot(t)

	// An unreadable subdirectory under overrides/ makes filepath.WalkDir
	// (inside OverrideStatusAt) fail to descend into it, propagating a
	// real error out of OverrideStatusAt -- unlike a missing/corrupt
	// baseline manifest, which OverrideStatusAt already tolerates.
	blockedDir := filepath.Join(dataRoot, "overrides", "blocked")
	require.NoError(t, os.MkdirAll(blockedDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(blockedDir, "file.txt"), []byte("x"), 0644))
	require.NoError(t, os.Chmod(blockedDir, 0000))
	t.Cleanup(func() { _ = os.Chmod(blockedDir, 0755) })

	setJSONMode(t, false)
	withUpgradeDryRun(t, false)

	var runErr error
	stdout, stderr := captureStdoutAndStderr(t, func() {
		runErr = runSharkUpgrade(&cobra.Command{}, nil)
	})

	require.NoError(t, runErr, "an OverrideStatusAt failure must not turn an already-applied upgrade into a hard error")
	assert.Contains(t, stdout, "added:")
	assert.Contains(t, stdout, "updated:")
	assert.Contains(t, stdout, "unchanged:")
	assert.Contains(t, stdout, "overrides skipped:")
	assert.Contains(t, stdout, "overrides: "+sharkdata.ClassificationCurrent+"=0")
	assert.Contains(t, stderr, "warning")
	assert.Contains(t, stderr, "overrides status")
}

// TestRunSharkUpgrade_RealUpgrade_OverridesByteIdentity is TC-010: a real
// (non-dry-run) upgrade must leave the override inventory, every override
// file's bytes and mode bits, and the baseline manifest's bytes identical --
// a plain upgrade only ever writes canonical files outside overrides/ and
// never writes .shark-override-baselines.json (AC-T2; only `acknowledge`
// writes that file).
func TestRunSharkUpgrade_RealUpgrade_OverridesByteIdentity(t *testing.T) {
	_, dataRoot := setupUpgradeFixtureRoot(t)

	const sentinelPath = upgradeCurrentFixturePath
	sentinelBytes := []byte("sentinel override content, fixed bytes, must not change")
	overrideFullPath := writeOverridesCmdOverrideFile(t, dataRoot, sentinelPath, sentinelBytes)
	require.NoError(t, os.Chmod(overrideFullPath, 0644))

	epicCanonical, err := sharkdata.ReadEmbedded(sentinelPath)
	require.NoError(t, err)
	manifest := &sharkdata.OverrideBaselineManifest{
		SchemaVersion: 1,
		Baselines: map[string]string{
			sentinelPath: sha256HexForTest(epicCanonical),
		},
	}
	require.NoError(t, manifest.Save(dataRoot))

	manifestPath := filepath.Join(dataRoot, ".shark-override-baselines.json")
	manifestBefore, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	overridesBefore := listOverridePaths(t, dataRoot)

	beforeStat, err := os.Stat(overrideFullPath)
	require.NoError(t, err)

	setJSONMode(t, false)
	withUpgradeDryRun(t, false)

	var runErr error
	captureStdout(t, func() {
		runErr = runSharkUpgrade(&cobra.Command{}, nil)
	})
	require.NoError(t, runErr)

	assert.Equal(t, overridesBefore, listOverridePaths(t, dataRoot), "override inventory (set of paths) must be unchanged")

	afterBytes, err := os.ReadFile(overrideFullPath)
	require.NoError(t, err)
	assert.Equal(t, sentinelBytes, afterBytes, "override file bytes must be unchanged")

	afterStat, err := os.Stat(overrideFullPath)
	require.NoError(t, err)
	assert.Equal(t, beforeStat.Mode(), afterStat.Mode(), "override file mode bits must be unchanged")

	manifestAfter, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	assert.Equal(t, manifestBefore, manifestAfter, "a real upgrade must never write .shark-override-baselines.json")
}
