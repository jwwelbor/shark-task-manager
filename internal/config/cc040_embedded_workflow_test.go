package config

// cc040_embedded_workflow_test.go verifies that Pass 3 of
// defaultWorkflowDataLoader reads from the embedded FS instead of calling
// emptyMLW.GetWorkflowForLevel() (CC-040).
//
// Acceptance criteria:
//   - A zero-config project (no shark-data/ on disk, no inline workflow blocks)
//     gets the same statuses/transitions as the embedded canonical YAML.
//   - The embedded task workflow includes the route-based steps (draft,
//     development, completed, etc.) that the hardcoded Go default does NOT
//     include.

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/workflow"
	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
)

// TestDefaultWorkflowDataLoader_Pass3_AllSlotsPopulated verifies that when
// no disk workflow dir exists and no inline config is present, Pass 3 populates
// all entity slots and includes route-based statuses from the embedded YAMLs.
// Counter-factual: the hardcoded Go DefaultWorkflow does NOT contain "draft" or
// "development" — so if Pass 3 were using hardcoded defaults, this test fails.
func TestDefaultWorkflowDataLoader_Pass3_AllSlotsPopulated(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	result, err := defaultWorkflowDataLoader(configPath)
	if err != nil {
		t.Fatalf("defaultWorkflowDataLoader: %v", err)
	}

	// Verify all entity slots are present.
	for _, entityType := range workflow.EntityTypes() {
		if _, ok := result[entityType]; !ok {
			t.Errorf("entity slot %q missing from result", entityType)
		}
	}

	// Counter-factual assertion: the task slot must contain "draft", which exists
	// in the embedded task.yaml but NOT in the hardcoded Go DefaultWorkflow.
	// If Pass 3 fell back to hardcoded defaults, "draft" would be absent.
	taskStatuses, ok := result["task"]
	if !ok {
		t.Fatal("task slot missing from result")
	}
	if _, hasDraft := taskStatuses["draft"]; !hasDraft {
		t.Error("task slot missing \"draft\" status — Pass 3 may be using hardcoded Go defaults instead of embedded YAML")
	}
}

// TestDefaultWorkflowDataLoader_Pass3_TaskMatchesEmbeddedStatuses verifies
// that the task workflow loaded by Pass 3 has the same status set as the
// embedded task.yaml — specifically the route-based steps (draft, development,
// etc.) that are NOT present in the hardcoded Go DefaultWorkflow.
func TestDefaultWorkflowDataLoader_Pass3_TaskMatchesEmbeddedStatuses(t *testing.T) {
	// Parse the embedded task.yaml directly to get the expected status set.
	embeddedBytes, err := sharkdata.ReadEmbedded("workflow/task.yaml")
	if err != nil {
		t.Fatalf("sharkdata.ReadEmbedded(workflow/task.yaml): %v", err)
	}
	embeddedCfg, err := workflow.ParseWorkflowYAMLBytes(embeddedBytes, "embedded:workflow/task.yaml")
	if err != nil {
		t.Fatalf("ParseWorkflowYAMLBytes: %v", err)
	}

	// Build a zero-config project and run the loader.
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	result, err := defaultWorkflowDataLoader(configPath)
	if err != nil {
		t.Fatalf("defaultWorkflowDataLoader: %v", err)
	}

	taskSlot, ok := result["task"]
	if !ok {
		t.Fatal("task slot missing from result")
	}

	// Collect status names from the loader result and from the embedded YAML.
	loaderStatuses := make([]string, 0, len(taskSlot))
	for status := range taskSlot {
		loaderStatuses = append(loaderStatuses, status)
	}
	sort.Strings(loaderStatuses)

	embeddedStatuses := make([]string, 0, len(embeddedCfg.StatusMetadata))
	for status := range embeddedCfg.StatusMetadata {
		embeddedStatuses = append(embeddedStatuses, status)
	}
	sort.Strings(embeddedStatuses)

	// Verify the sets match.
	if len(loaderStatuses) != len(embeddedStatuses) {
		t.Errorf("status count mismatch: loader=%d embedded=%d\nloader=%v\nembedded=%v",
			len(loaderStatuses), len(embeddedStatuses), loaderStatuses, embeddedStatuses)
		return
	}
	for i, s := range loaderStatuses {
		if s != embeddedStatuses[i] {
			t.Errorf("status mismatch at index %d: loader=%q embedded=%q", i, s, embeddedStatuses[i])
		}
	}
}

// TestDefaultWorkflowDataLoader_Pass3_EmbeddedRicherThanHardcoded verifies
// that the embedded task workflow contains route-based steps (e.g. "draft",
// "development") that are NOT present in the legacy hardcoded DefaultWorkflow.
// This is the key CC-040 regression: Pass 3 must ship the richer embedded
// content, not the stale hardcoded Go struct.
func TestDefaultWorkflowDataLoader_Pass3_EmbeddedRicherThanHardcoded(t *testing.T) {
	// These statuses exist in the embedded task.yaml (route-based steps) but
	// are NOT present in the hardcoded DefaultWorkflow Go struct.
	routeBasedStatuses := []string{"draft", "development"}

	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	result, err := defaultWorkflowDataLoader(configPath)
	if err != nil {
		t.Fatalf("defaultWorkflowDataLoader: %v", err)
	}

	taskSlot, ok := result["task"]
	if !ok {
		t.Fatal("task slot missing from result")
	}

	for _, status := range routeBasedStatuses {
		if _, found := taskSlot[status]; !found {
			t.Errorf("expected route-based status %q in Pass 3 task slot (embedded YAML), but it was absent — Pass 3 may be using the hardcoded Go default instead of the embedded YAML", status)
		}
	}
}

// TestDefaultWorkflowDataLoader_Pass3_AllEmbeddedEntitiesLoaded verifies
// that for every entity type that has an embedded YAML file, the loader
// produces a non-empty status map consistent with the embedded content.
func TestDefaultWorkflowDataLoader_Pass3_AllEmbeddedEntitiesLoaded(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	result, err := defaultWorkflowDataLoader(configPath)
	if err != nil {
		t.Fatalf("defaultWorkflowDataLoader: %v", err)
	}

	// For each entity type that has an embedded YAML, verify the loader's
	// status set matches the embedded YAML's status set.
	for _, entityType := range workflow.EntityTypes() {
		filename := workflow.EmbeddedWorkflowFilename(entityType)
		if filename == "" {
			continue // no embedded YAML for this entity type
		}
		relPath := "workflow/" + filename
		embeddedBytes, readErr := sharkdata.ReadEmbedded(relPath)
		if readErr != nil {
			// Some entity types (e.g. sprint) may not have an embedded YAML
			// yet — skip rather than fail.
			continue
		}
		embeddedCfg, parseErr := workflow.ParseWorkflowYAMLBytes(embeddedBytes, "embedded:"+relPath)
		if parseErr != nil {
			t.Errorf("entity %q: parse embedded YAML: %v", entityType, parseErr)
			continue
		}

		loaderSlot, ok := result[entityType]
		if !ok {
			t.Errorf("entity %q: missing from loader result", entityType)
			continue
		}

		// Every status in the embedded YAML must appear in the loader result.
		for status := range embeddedCfg.StatusMetadata {
			if _, found := loaderSlot[status]; !found {
				t.Errorf("entity %q: embedded status %q missing from loader result", entityType, status)
			}
		}
	}
}
