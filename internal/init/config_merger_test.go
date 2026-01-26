package init

import (
	"testing"
)

// TestMerge_AddMissingFields tests adding new fields to existing config
func TestMerge_AddMissingFields(t *testing.T) {
	merger := NewConfigMerger()

	base := map[string]interface{}{
		"database": map[string]interface{}{"backend": "local"},
		"viewer":   "nano",
	}

	overlay := map[string]interface{}{
		"status_metadata": map[string]interface{}{
			"todo": map[string]interface{}{"color": "gray"},
		},
	}

	opts := ConfigMergeOptions{}
	merged, report, err := merger.Merge(base, overlay, opts)

	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// Check database is preserved
	if db, exists := merged["database"]; !exists {
		t.Error("expected database to exist")
	} else if dbMap, ok := db.(map[string]interface{}); !ok {
		t.Error("expected database to be map")
	} else if backend, ok := dbMap["backend"].(string); !ok || backend != "local" {
		t.Error("expected database.backend to be 'local'")
	}

	// Check viewer is preserved
	if viewer, ok := merged["viewer"].(string); !ok || viewer != "nano" {
		t.Error("expected viewer to be 'nano'")
	}

	// Check status_metadata was added
	if sm, exists := merged["status_metadata"]; !exists {
		t.Error("expected status_metadata to be added")
	} else if smMap, ok := sm.(map[string]interface{}); !ok {
		t.Error("expected status_metadata to be map")
	} else if todo, ok := smMap["todo"]; !ok {
		t.Error("expected status_metadata.todo to exist")
	} else if todoMap, ok := todo.(map[string]interface{}); !ok {
		t.Error("expected todo to be map")
	} else if color, ok := todoMap["color"].(string); !ok || color != "gray" {
		t.Error("expected color to be 'gray'")
	}

	// Check report
	if len(report.Added) == 0 {
		t.Error("expected Added to contain status_metadata")
	}

	if report.Stats == nil {
		t.Error("expected Stats to exist")
	}
}

// TestMerge_PreserveFields tests that preserve list is respected
func TestMerge_PreserveFields(t *testing.T) {
	merger := NewConfigMerger()

	base := map[string]interface{}{
		"database":    "old_db.db",
		"viewer":      "vim",
		"project_root": "/old/path",
	}

	overlay := map[string]interface{}{
		"database":    "new_db.db",
		"viewer":      "nano",
		"project_root": "/new/path",
	}

	opts := ConfigMergeOptions{
		PreserveFields: []string{"database", "viewer", "project_root"},
	}

	merged, report, err := merger.Merge(base, overlay, opts)

	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// All fields should be preserved
	if db, ok := merged["database"].(string); !ok || db != "old_db.db" {
		t.Error("expected database to be 'old_db.db'")
	}

	if viewer, ok := merged["viewer"].(string); !ok || viewer != "vim" {
		t.Error("expected viewer to be 'vim'")
	}

	if pr, ok := merged["project_root"].(string); !ok || pr != "/old/path" {
		t.Error("expected project_root to be '/old/path'")
	}

	// Check report shows preserved
	if len(report.Preserved) != 3 {
		t.Errorf("expected 3 preserved fields, got %d", len(report.Preserved))
	}
}

// TestMerge_OverwriteFields tests that overwrite list replaces fields
func TestMerge_OverwriteFields(t *testing.T) {
	merger := NewConfigMerger()

	base := map[string]interface{}{
		"status_metadata": map[string]interface{}{
			"old": map[string]interface{}{"color": "red"},
		},
		"viewer": "vim",
	}

	overlay := map[string]interface{}{
		"status_metadata": map[string]interface{}{
			"new": map[string]interface{}{"color": "blue"},
		},
	}

	opts := ConfigMergeOptions{
		OverwriteFields: []string{"status_metadata"},
	}

	merged, report, err := merger.Merge(base, overlay, opts)

	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// status_metadata should be replaced
	if sm, ok := merged["status_metadata"].(map[string]interface{}); !ok {
		t.Error("expected status_metadata to be map")
	} else if _, hasOld := sm["old"]; hasOld {
		t.Error("expected old status to be removed")
	} else if _, hasNew := sm["new"]; !hasNew {
		t.Error("expected new status to be added")
	}

	// viewer should be preserved
	if viewer, ok := merged["viewer"].(string); !ok || viewer != "vim" {
		t.Error("expected viewer to be 'vim'")
	}

	// Check report
	if len(report.Overwritten) != 1 {
		t.Errorf("expected 1 overwritten field, got %d", len(report.Overwritten))
	}
}

// TestMerge_NestedMaps tests deep merge of nested structures
func TestMerge_NestedMaps(t *testing.T) {
	merger := NewConfigMerger()

	base := map[string]interface{}{
		"settings": map[string]interface{}{
			"color":    true,
			"verbose":  false,
			"nested": map[string]interface{}{
				"deep": "value1",
			},
		},
	}

	overlay := map[string]interface{}{
		"settings": map[string]interface{}{
			"verbose": true,
			"nested": map[string]interface{}{
				"deep":  "value2",
				"new":   "value3",
			},
		},
	}

	opts := ConfigMergeOptions{}

	merged, report, err := merger.Merge(base, overlay, opts)

	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	settings, ok := merged["settings"].(map[string]interface{})
	if !ok {
		t.Fatal("expected settings to be map")
	}

	// color should be preserved from base
	if color, ok := settings["color"].(bool); !ok || !color {
		t.Error("expected color to be true")
	}

	// verbose should be updated
	if verbose, ok := settings["verbose"].(bool); !ok || !verbose {
		t.Error("expected verbose to be true")
	}

	// nested should be deeply merged
	nested, ok := settings["nested"].(map[string]interface{})
	if !ok {
		t.Fatal("expected nested to be map")
	}

	if deep, ok := nested["deep"].(string); !ok || deep != "value2" {
		t.Error("expected nested.deep to be 'value2'")
	}

	if newVal, ok := nested["new"].(string); !ok || newVal != "value3" {
		t.Error("expected nested.new to be 'value3'")
	}

	if report.Stats == nil {
		t.Error("expected Stats to exist")
	}
}

// TestMerge_Force tests force mode overwrites protected fields
func TestMerge_Force(t *testing.T) {
	merger := NewConfigMerger()

	base := map[string]interface{}{
		"database": "old_db.db",
		"viewer":   "vim",
	}

	overlay := map[string]interface{}{
		"database": "new_db.db",
		"viewer":   "nano",
	}

	opts := ConfigMergeOptions{
		PreserveFields: []string{"database", "viewer"},
		Force:          true,
	}

	merged, report, err := merger.Merge(base, overlay, opts)

	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// With force=true, protected fields should be overwritten
	if db, ok := merged["database"].(string); !ok || db != "new_db.db" {
		t.Error("expected database to be 'new_db.db' in force mode")
	}

	if viewer, ok := merged["viewer"].(string); !ok || viewer != "nano" {
		t.Error("expected viewer to be 'nano' in force mode")
	}

	// Check report shows overwritten
	if len(report.Overwritten) != 2 {
		t.Errorf("expected 2 overwritten fields, got %d", len(report.Overwritten))
	}
}

// TestDeepMerge_Simple tests simple map merge
func TestDeepMerge_Simple(t *testing.T) {
	merger := NewConfigMerger()

	base := map[string]interface{}{
		"a": 1,
		"b": 2,
	}

	overlay := map[string]interface{}{
		"b": 20,
		"c": 30,
	}

	result := merger.DeepMerge(base, overlay)

	if a, ok := result["a"].(int); !ok || a != 1 {
		t.Error("expected a to be 1")
	}

	if b, ok := result["b"].(int); !ok || b != 20 {
		t.Error("expected b to be 20")
	}

	if c, ok := result["c"].(int); !ok || c != 30 {
		t.Error("expected c to be 30")
	}
}

// TestDeepMerge_Nested tests nested map merge
func TestDeepMerge_Nested(t *testing.T) {
	merger := NewConfigMerger()

	base := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"a": 1,
				"b": 2,
			},
		},
	}

	overlay := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"b": 20,
				"c": 30,
			},
		},
	}

	result := merger.DeepMerge(base, overlay)

	level1, ok := result["level1"].(map[string]interface{})
	if !ok {
		t.Fatal("expected level1 to be map")
	}

	level2, ok := level1["level2"].(map[string]interface{})
	if !ok {
		t.Fatal("expected level2 to be map")
	}

	if a, ok := level2["a"].(int); !ok || a != 1 {
		t.Error("expected a to be 1")
	}

	if b, ok := level2["b"].(int); !ok || b != 20 {
		t.Error("expected b to be 20")
	}

	if c, ok := level2["c"].(int); !ok || c != 30 {
		t.Error("expected c to be 30")
	}
}

// TestDetectChanges tests change detection accuracy
func TestDetectChanges(t *testing.T) {
	merger := NewConfigMerger()

	old := map[string]interface{}{
		"a":      1,
		"b":      2,
		"keep":   "same",
	}

	new := map[string]interface{}{
		"a":      1,
		"b":      20,
		"keep":   "same",
		"added":  3,
	}

	report := merger.DetectChanges(old, new)

	// Check added
	if len(report.Added) != 1 || report.Added[0] != "added" {
		t.Errorf("expected 'added' in Added, got %v", report.Added)
	}

	// Check overwritten
	if len(report.Overwritten) != 1 || report.Overwritten[0] != "b" {
		t.Errorf("expected 'b' in Overwritten, got %v", report.Overwritten)
	}

	// Check preserved (both 'a' and 'keep' should be preserved)
	if len(report.Preserved) != 2 {
		t.Errorf("expected 2 preserved fields, got %d: %v", len(report.Preserved), report.Preserved)
	}
	expectedPreserved := map[string]bool{"a": true, "keep": true}
	for _, p := range report.Preserved {
		if !expectedPreserved[p] {
			t.Errorf("unexpected preserved field: %s", p)
		}
	}
}

// TestCalculateStats tests statistics calculation
func TestCalculateStats(t *testing.T) {
	merger := NewConfigMerger()

	base := map[string]interface{}{
		"a": 1,
		"b": 2,
	}

	merged := map[string]interface{}{
		"a": 1,
		"b": 2,
		"c": 3,
	}

	overlay := map[string]interface{}{
		"status_metadata": map[string]interface{}{
			"todo":     map[string]interface{}{},
			"progress": map[string]interface{}{},
		},
		"status_flow": map[string]interface{}{
			"todo":     []string{},
			"progress": []string{},
		},
		"special_statuses": map[string]interface{}{
			"blocked": []string{},
		},
		"c": 3,
	}

	stats := merger.calculateStats(base, merged, overlay)

	if stats == nil {
		t.Fatal("expected stats to exist")
	}

	if stats.StatusesAdded != 2 {
		t.Errorf("expected 2 statuses added, got %d", stats.StatusesAdded)
	}

	if stats.FlowsAdded != 2 {
		t.Errorf("expected 2 flows added, got %d", stats.FlowsAdded)
	}

	if stats.GroupsAdded != 1 {
		t.Errorf("expected 1 group added, got %d", stats.GroupsAdded)
	}

	if stats.FieldsPreserved != 2 {
		t.Errorf("expected 2 fields preserved, got %d", stats.FieldsPreserved)
	}
}

// TestMerge_EmptyBase tests merge into empty config
func TestMerge_EmptyBase(t *testing.T) {
	merger := NewConfigMerger()

	base := map[string]interface{}{}

	overlay := map[string]interface{}{
		"status_metadata": map[string]interface{}{
			"todo": map[string]interface{}{"color": "gray"},
		},
		"viewer": "nano",
	}

	opts := ConfigMergeOptions{}

	merged, report, err := merger.Merge(base, overlay, opts)

	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	if len(merged) != 2 {
		t.Errorf("expected 2 fields in merged, got %d", len(merged))
	}

	if len(report.Added) != 2 {
		t.Errorf("expected 2 added fields, got %d", len(report.Added))
	}
}

// TestMerge_EmptyOverlay tests merge with empty overlay
func TestMerge_EmptyOverlay(t *testing.T) {
	merger := NewConfigMerger()

	base := map[string]interface{}{
		"database": "db.db",
		"viewer":   "vim",
	}

	overlay := map[string]interface{}{}

	opts := ConfigMergeOptions{}

	merged, report, err := merger.Merge(base, overlay, opts)

	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// Should preserve all base fields
	if db, ok := merged["database"].(string); !ok || db != "db.db" {
		t.Error("expected database to be 'db.db'")
	}

	if viewer, ok := merged["viewer"].(string); !ok || viewer != "vim" {
		t.Error("expected viewer to be 'vim'")
	}

	if len(report.Added) != 0 {
		t.Errorf("expected no added fields, got %d", len(report.Added))
	}
}

// TestDeepCopy tests deep copy prevents mutation of original
func TestDeepCopy_NoMutation(t *testing.T) {
	original := map[string]interface{}{
		"nested": map[string]interface{}{
			"value": "original",
		},
	}

	copied := deepCopy(original)

	// Modify copy
	if nested, ok := copied["nested"].(map[string]interface{}); ok {
		nested["value"] = "modified"
	}

	// Original should be unchanged
	if nested, ok := original["nested"].(map[string]interface{}); ok {
		if value, ok := nested["value"].(string); !ok || value != "original" {
			t.Error("expected original to be unchanged")
		}
	}
}

// TestDeepCopySlice tests deep copy of slices
func TestDeepCopySlice_NoMutation(t *testing.T) {
	original := []interface{}{
		map[string]interface{}{"a": 1},
		"string",
		42,
	}

	copied := deepCopySlice(original)

	// Modify copy
	if m, ok := copied[0].(map[string]interface{}); ok {
		m["a"] = 999
	}

	// Original should be unchanged
	if m, ok := original[0].(map[string]interface{}); ok {
		if a, ok := m["a"].(int); !ok || a != 1 {
			t.Error("expected original to be unchanged")
		}
	}
}

// TestContains tests string slice contains check
func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !contains(slice, "b") {
		t.Error("expected contains to find 'b'")
	}

	if contains(slice, "d") {
		t.Error("expected contains to not find 'd'")
	}

	if !contains([]string{}, "a") == contains([]string{}, "a") {
		// Empty slice should not contain anything
		emptySlice := []string{}
		if contains(emptySlice, "a") {
			t.Error("expected empty slice to not contain 'a'")
		}
	}
}

// TestDeepEqual tests deep equality check
func TestDeepEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        interface{}
		b        interface{}
		expected bool
	}{
		{
			name:     "equal strings",
			a:        "hello",
			b:        "hello",
			expected: true,
		},
		{
			name:     "different strings",
			a:        "hello",
			b:        "world",
			expected: false,
		},
		{
			name:     "equal maps",
			a:        map[string]interface{}{"a": 1, "b": 2},
			b:        map[string]interface{}{"a": 1, "b": 2},
			expected: true,
		},
		{
			name:     "different map values",
			a:        map[string]interface{}{"a": 1},
			b:        map[string]interface{}{"a": 2},
			expected: false,
		},
		{
			name:     "different map keys",
			a:        map[string]interface{}{"a": 1},
			b:        map[string]interface{}{"b": 1},
			expected: false,
		},
		{
			name:     "nested maps equal",
			a:        map[string]interface{}{"nested": map[string]interface{}{"a": 1}},
			b:        map[string]interface{}{"nested": map[string]interface{}{"a": 1}},
			expected: true,
		},
		{
			name:     "equal ints",
			a:        42,
			b:        42,
			expected: true,
		},
		{
			name:     "different ints",
			a:        42,
			b:        43,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deepEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("deepEqual(%v, %v) = %v, expected %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// TestMerge_ComplexScenario tests a realistic complex scenario
func TestMerge_ComplexScenario(t *testing.T) {
	merger := NewConfigMerger()

	base := map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "local",
			"url":     "./shark-tasks.db",
		},
		"project_root": "/my/project",
		"viewer":       "vim",
		"status_metadata": map[string]interface{}{
			"todo": map[string]interface{}{
				"color": "gray",
				"phase": "planning",
			},
		},
	}

	overlay := map[string]interface{}{
		"status_metadata": map[string]interface{}{
			"todo": map[string]interface{}{
				"color":           "gray",
				"phase":           "planning",
				"progress_weight": 0.0,
			},
			"in_progress": map[string]interface{}{
				"color":           "yellow",
				"phase":           "development",
				"progress_weight": 50.0,
			},
		},
		"status_flow": map[string]interface{}{
			"todo": []string{"in_progress", "blocked"},
		},
		"special_statuses": map[string]interface{}{
			"blocked": []string{"in_progress", "todo"},
		},
	}

	opts := ConfigMergeOptions{
		PreserveFields:  []string{"database", "project_root", "viewer"},
		OverwriteFields: []string{"status_metadata", "status_flow", "special_statuses"},
	}

	merged, report, err := merger.Merge(base, overlay, opts)

	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// Verify protected fields are preserved
	if db, ok := merged["database"].(map[string]interface{}); !ok {
		t.Error("expected database to exist")
	} else if backend, ok := db["backend"].(string); !ok || backend != "local" {
		t.Error("expected database.backend to be 'local'")
	}

	if pr, ok := merged["project_root"].(string); !ok || pr != "/my/project" {
		t.Error("expected project_root to be preserved")
	}

	if viewer, ok := merged["viewer"].(string); !ok || viewer != "vim" {
		t.Error("expected viewer to be preserved")
	}

	// Verify status_metadata was replaced
	if sm, ok := merged["status_metadata"].(map[string]interface{}); !ok {
		t.Error("expected status_metadata to be map")
	} else if _, hasOld := sm["todo"].(map[string]interface{}); !hasOld {
		t.Error("expected todo to exist in status_metadata")
	}

	// Verify new fields added
	if _, hasFlow := merged["status_flow"]; !hasFlow {
		t.Error("expected status_flow to be added")
	}

	if _, hasSpecial := merged["special_statuses"]; !hasSpecial {
		t.Error("expected special_statuses to be added")
	}

	// Verify report accuracy
	// Note: Preserved only tracks fields in overlay that are protected
	// Since overlay doesn't contain database/project_root/viewer, they're not in Preserved
	if len(report.Overwritten) != 1 {
		t.Errorf("expected 1 overwritten field (status_metadata), got %d: %v", len(report.Overwritten), report.Overwritten)
	}

	// Added should contain status_flow and special_statuses
	expectedAdded := map[string]bool{"status_flow": true, "special_statuses": true}
	for _, a := range report.Added {
		if !expectedAdded[a] {
			t.Errorf("unexpected added field: %s", a)
		}
	}

	if report.Stats == nil {
		t.Error("expected stats to exist")
	}
}

// TestMerge_PreserveWithoutForce_Workflow tests workflow config protection
func TestMerge_PreserveWithoutForce_Workflow(t *testing.T) {
	merger := NewConfigMerger()

	base := map[string]interface{}{
		"status_metadata": map[string]interface{}{
			"custom_status": map[string]interface{}{"color": "purple"},
		},
	}

	overlay := map[string]interface{}{
		"status_metadata": map[string]interface{}{
			"todo": map[string]interface{}{"color": "gray"},
		},
	}

	// Without force, and status_metadata not in OverwriteFields
	opts := ConfigMergeOptions{
		PreserveFields: []string{"status_metadata"},
		Force:          false,
	}

	merged, report, err := merger.Merge(base, overlay, opts)

	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// status_metadata should be preserved (not overwritten)
	if sm, ok := merged["status_metadata"].(map[string]interface{}); !ok {
		t.Error("expected status_metadata to exist")
	} else if _, hasCustom := sm["custom_status"]; !hasCustom {
		t.Error("expected custom_status to be preserved")
	} else if _, hasTodo := sm["todo"]; hasTodo {
		t.Error("expected todo to not be added (protected field)")
	}

	// Should be in preserved list
	if len(report.Preserved) != 1 || report.Preserved[0] != "status_metadata" {
		t.Errorf("expected status_metadata in preserved, got %v", report.Preserved)
	}
}

// TestMerge_ForceUnlocksWorkflowConfig tests force flag overrides protection
func TestMerge_ForceUnlocksWorkflowConfig(t *testing.T) {
	merger := NewConfigMerger()

	base := map[string]interface{}{
		"status_metadata": map[string]interface{}{
			"custom_status": map[string]interface{}{"color": "purple"},
		},
	}

	overlay := map[string]interface{}{
		"status_metadata": map[string]interface{}{
			"todo": map[string]interface{}{"color": "gray"},
		},
	}

	// With force=true, protected fields can be overwritten
	opts := ConfigMergeOptions{
		PreserveFields: []string{"status_metadata"},
		Force:          true,
	}

	merged, report, err := merger.Merge(base, overlay, opts)

	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// status_metadata should be replaced
	if sm, ok := merged["status_metadata"].(map[string]interface{}); !ok {
		t.Error("expected status_metadata to exist")
	} else if _, hasCustom := sm["custom_status"]; hasCustom {
		t.Error("expected custom_status to be removed (force mode)")
	} else if _, hasTodo := sm["todo"]; !hasTodo {
		t.Error("expected todo to be added (force mode)")
	}

	// Should be in overwritten list
	if len(report.Overwritten) != 1 || report.Overwritten[0] != "status_metadata" {
		t.Errorf("expected status_metadata in overwritten, got %v", report.Overwritten)
	}
}
