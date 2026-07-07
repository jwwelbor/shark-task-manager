package commands

// status_advance_otel_test.go — Integration test for T-E32-F07-004
//
// Verifies that runStatusAdvance emits a "shark.advance" OTel span with the
// required attributes to the file_jsonl exporter. The test:
//   1. Creates a minimal shark project in a tmpDir (config + SQLite DB).
//   2. Seeds an epic, feature, and task in "todo" status.
//   3. Initialises the file_jsonl OTel provider pointing at the tmpDir.
//   4. Calls runStatusAdvance directly.
//   5. Shuts down OTel so all spans are flushed to events.jsonl.
//   6. Reads the JSONL file and asserts ≥1 line contains the expected span.

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/observability"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalSharkConfig is the .sharkconfig.json written into the tmpDir. It
// embeds the workflow so we do not depend on any project-level config file,
// and enables the file_jsonl OTel exporter so spans land in events.jsonl.
const minimalSharkConfig = `{
  "observability": {
    "enabled": true,
    "tracing_enabled": true,
    "exporter": "file_jsonl"
  },
  "status_flow": {
    "todo": ["in_progress", "blocked"],
    "in_progress": ["completed", "blocked"],
    "completed": [],
    "blocked": ["todo", "in_progress"]
  },
  "status_metadata": {
    "todo":        {"color": "gray",  "phase": "planning",     "description": "Not started"},
    "in_progress": {"color": "blue",  "phase": "development",  "description": "In progress"},
    "completed":   {"color": "green", "phase": "done",         "description": "Done"},
    "blocked":     {"color": "red",   "phase": "any",          "description": "Blocked"}
  },
  "special_statuses": {
    "_start_":    ["todo"],
    "_complete_": ["completed"]
  },
  "epic_workflow": {
    "status_flow": {
      "draft":     ["active"],
      "active":    ["completed"],
      "completed": []
    },
    "status_metadata": {
      "draft":     {"color": "gray",  "phase": "planning", "description": "Draft"},
      "active":    {"color": "blue",  "phase": "execution","description": "Active"},
      "completed": {"color": "green", "phase": "done",     "description": "Done"}
    },
    "special_statuses": {
      "_start_":    ["draft"],
      "_complete_": ["completed"]
    }
  },
  "feature_workflow": {
    "status_flow": {
      "draft":     ["active"],
      "active":    ["completed"],
      "completed": []
    },
    "status_metadata": {
      "draft":     {"color": "gray",  "phase": "planning", "description": "Draft"},
      "active":    {"color": "blue",  "phase": "execution","description": "Active"},
      "completed": {"color": "green", "phase": "done",     "description": "Done"}
    },
    "special_statuses": {
      "_start_":    ["draft"],
      "_complete_": ["completed"]
    }
  }
}`

// TestRunStatusAdvance_EmitsSharkAdvanceSpan is the primary acceptance test for
// T-E32-F07-004. It drives runStatusAdvance end-to-end and verifies that the
// "shark.advance" span is written to events.jsonl with the expected attributes.
//
// NOTE: This test MUST run serially (no t.Parallel()). It os.Chdir()s into a
// temp project root and drives the global cli.GetDB()/workflow singletons;
// running concurrently with any cwd- or singleton-dependent test would race.
// The package relies on the default serial test execution for this.
func TestRunStatusAdvance_EmitsSharkAdvanceSpan(t *testing.T) {
	// ── 1. Set up the isolated project root ───────────────────────────────────

	tmpDir := t.TempDir()

	// Write the .sharkconfig.json that the project root detection relies on.
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	require.NoError(t, os.WriteFile(configPath, []byte(minimalSharkConfig), 0644))

	// ── 2. Seed a real SQLite database in the tmpDir ──────────────────────────

	dbPath := filepath.Join(tmpDir, "shark-tasks.db")
	rawDB, err := db.InitDB(dbPath)
	require.NoError(t, err, "InitDB must succeed for the test database")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repoDb := repository.NewDB(rawDB)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	epic := &models.Epic{
		BaseEntity: models.BaseEntity{Key: "E01", Title: "OTel Test Epic"},
		Status:     models.EpicStatus("draft"),
		Priority:   models.PriorityMedium,
	}
	require.NoError(t, epicRepo.Create(ctx, epic), "seed epic")

	feature := &models.Feature{
		BaseEntity: models.BaseEntity{Key: "E01-F01", Title: "OTel Test Feature"},
		EpicID:     epic.ID,
		Status:     models.FeatureStatus("draft"),
	}
	require.NoError(t, featureRepo.Create(ctx, feature), "seed feature")

	task := &models.Task{
		BaseEntity: models.BaseEntity{Key: "T-E01-F01-001", Title: "OTel Test Task"},
		FeatureID:  feature.ID,
		Status:     models.TaskStatus("todo"),
		Priority:   5,
	}
	require.NoError(t, taskRepo.Create(ctx, task), "seed task")

	// Close the seeding connection; the global cli.GetDB() will open its own.
	rawDB.Close()

	// ── 3. Switch CWD so project root detection finds the tmpDir ─────────────

	origWD, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origWD) })
	require.NoError(t, os.Chdir(tmpDir))

	// Reset all global CLI singletons so this test runs in isolation.
	origConfigFile := cli.GlobalConfig.ConfigFile
	origDBPath := cli.GlobalConfig.DBPath
	configFlag := cli.RootCmd.PersistentFlags().Lookup("config")
	dbFlag := cli.RootCmd.PersistentFlags().Lookup("db")
	origConfigChanged := configFlag != nil && configFlag.Changed
	origDBChanged := dbFlag != nil && dbFlag.Changed
	cli.GlobalConfig.ConfigFile = configPath
	cli.GlobalConfig.DBPath = dbPath
	require.NoError(t, cli.RootCmd.PersistentFlags().Set("config", configPath))
	require.NoError(t, cli.RootCmd.PersistentFlags().Set("db", dbPath))
	cli.ResetDB()
	cli.ResetServices()
	cli.ResetWorkflowService()
	cli.ResetObservability()
	config.ClearWorkflowCache()
	t.Cleanup(func() {
		cli.GlobalConfig.ConfigFile = origConfigFile
		cli.GlobalConfig.DBPath = origDBPath
		require.NoError(t, cli.RootCmd.PersistentFlags().Set("config", origConfigFile))
		require.NoError(t, cli.RootCmd.PersistentFlags().Set("db", origDBPath))
		if configFlag != nil {
			configFlag.Changed = origConfigChanged
		}
		if dbFlag != nil {
			dbFlag.Changed = origDBChanged
		}
		cli.ResetDB()
		cli.ResetServices()
		cli.ResetWorkflowService()
		cli.ResetObservability()
		config.ClearWorkflowCache()
	})

	// ── 4. Initialize the file_jsonl OTel provider ────────────────────────────

	obsCfg := config.ObservabilityConfig{
		Enabled:        true,
		TracingEnabled: true,
		Exporter:       "file_jsonl",
	}
	// InitProviderWithRoot creates shark-data/.stats/events.jsonl under tmpDir.
	shutdown, err := observability.InitProviderWithRoot(obsCfg, tmpDir)
	require.NoError(t, err, "InitProviderWithRoot must succeed")

	// ── 5. Run the command under test ─────────────────────────────────────────

	// runStatusAdvance reads GlobalConfig.JSON; ensure it is false (human output).
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = false
	t.Cleanup(func() { cli.GlobalConfig.JSON = origJSON })

	advErr := runStatusAdvance(nil, []string{"T-E01-F01-001"})
	assert.NoError(t, advErr, "runStatusAdvance must succeed for a task in 'todo' status")

	// ── 6. Flush spans by shutting down the provider ──────────────────────────

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	require.NoError(t, shutdown(shutdownCtx), "OTel provider must shut down cleanly")

	// ── 7. Read and parse events.jsonl ────────────────────────────────────────

	eventsPath := filepath.Join(tmpDir, "shark-data", ".stats", "events.jsonl")
	f, err := os.Open(eventsPath)
	require.NoError(t, err, "events.jsonl must exist after shutdown")
	defer f.Close()

	type spanRecord struct {
		SpanName string                 `json:"span_name"`
		Attrs    map[string]interface{} `json:"attrs"`
	}

	var advanceRecord *spanRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var rec spanRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue // skip lines that don't parse
		}
		if rec.SpanName == "shark.advance" {
			rec := rec // capture loop variable
			advanceRecord = &rec
			break
		}
	}
	require.NoError(t, scanner.Err(), "scanning events.jsonl must not produce an error")
	require.NotNil(t, advanceRecord, "events.jsonl must contain at least one 'shark.advance' span line")

	// ── 8. Assert required attributes are present ─────────────────────────────

	attrs := advanceRecord.Attrs
	assert.Equal(t, "T-E01-F01-001", attrs["entity_key"],
		"entity_key attribute must match the task key")
	assert.Equal(t, "task", attrs["entity_type"],
		"entity_type attribute must be 'task'")
	assert.Equal(t, "todo", attrs["from_status"],
		"from_status attribute must be the task's initial status")
	assert.NotEmpty(t, attrs["to_status"],
		"to_status attribute must be present and non-empty")
	assert.NotEqual(t, attrs["from_status"], attrs["to_status"],
		"to_status must differ from from_status (a real transition occurred)")
}
