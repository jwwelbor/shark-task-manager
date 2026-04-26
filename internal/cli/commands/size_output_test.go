package commands

// size_output_test.go — E07-F42 T-009
//
// Tests for:
//   - formatSize(*int) helper (TC-F006-C, TC-F006-D)
//   - size field in JSON output from buildTaskGetJSON (TC-F006-A, TC-F006-B)
//   - size field in JSON output from buildEpicGetJSON
//   - size field in JSON output from buildFeatureGetJSON
//   - size field in buildTaskBasicInfo (human detail view)
//   - size field in buildBugBasicInfo (human detail view)
//   - size_label as an additional field accessible via --field
//
// Rules:
//   - Pure unit tests: no DB, no real service, no cobra command execution.
//   - All tests verify the output of helper functions directly.

import (
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ptrInt returns a pointer to the given int — test helper.
func ptrInt(n int) *int { return &n }

// ---------------------------------------------------------------------------
// TC-F006-C: formatSize — non-nil returns "<label> (<num>)"
// TC-F006-D: formatSize — nil returns "—"
// ---------------------------------------------------------------------------

func TestFormatSize_NilReturnsEmDash(t *testing.T) {
	// TC-F006-D
	result := formatSize(nil)
	if result != "—" {
		t.Errorf("formatSize(nil): expected %q, got %q", "—", result)
	}
}

func TestFormatSize_NonNilReturnsLabelAndNum(t *testing.T) {
	// TC-F006-C: table-driven for all 6 canonical values
	tests := []struct {
		size     *int
		expected string
	}{
		{ptrInt(1), "XS (1)"},
		{ptrInt(2), "S (2)"},
		{ptrInt(3), "M (3)"},
		{ptrInt(5), "L (5)"},
		{ptrInt(8), "XL (8)"},
		{ptrInt(13), "XXL (13)"},
	}
	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := formatSize(tc.size)
			if result != tc.expected {
				t.Errorf("formatSize(%d): expected %q, got %q", *tc.size, tc.expected, result)
			}
		})
	}
}

// Edge case: formatSize with an invalid (non-canonical) numeric value falls back to just the number.
func TestFormatSize_InvalidNumFallsBackToNumber(t *testing.T) {
	// Defensive: formatSize should not panic and should render something reasonable.
	// Per spec §3.6: "defensive: should never trigger" — returns Sprintf("%d", *s).
	result := formatSize(ptrInt(4))
	if result == "" || result == "—" {
		t.Errorf("formatSize(4): expected non-empty non-dash, got %q", result)
	}
	// The output should at least contain "4".
	if !strings.Contains(result, "4") {
		t.Errorf("formatSize(4): expected to contain '4', got %q", result)
	}
}

// ---------------------------------------------------------------------------
// TC-F006-A: buildTaskGetJSON includes "size" when non-nil
// TC-F006-B: buildTaskGetJSON omits "size" when nil (omitempty via JSON struct tag)
// ---------------------------------------------------------------------------

func TestBuildTaskGetJSON_SizeIncludedWhenNonNil(t *testing.T) {
	// TC-F006-A
	n := 5
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			ID:    1,
			Key:   "E07-F01-001",
			Title: "Test Task",
			Size:  &n,
		},
		Status:   models.TaskStatus("todo"),
		Priority: 3,
	}

	result := buildTaskGetJSON(task, nil, nil, nil, nil, nil, nil, nil, nil)

	if _, ok := result["size"]; !ok {
		t.Error("buildTaskGetJSON: expected 'size' key when Size is non-nil, but it is absent")
	}
	if result["size"] != &n && result["size"] != n {
		// Either a pointer or a direct int is acceptable; check the dereferenced value.
		sizeVal, ok := result["size"]
		if !ok {
			t.Error("buildTaskGetJSON: 'size' key missing")
			return
		}
		t.Logf("size value type: %T, value: %v", sizeVal, sizeVal)
		// Accept both *int(5) and int(5) — we just want the value to be 5.
		switch v := sizeVal.(type) {
		case *int:
			if *v != 5 {
				t.Errorf("buildTaskGetJSON: expected size=5, got %d", *v)
			}
		case int:
			if v != 5 {
				t.Errorf("buildTaskGetJSON: expected size=5, got %d", v)
			}
		case float64:
			if v != 5 {
				t.Errorf("buildTaskGetJSON: expected size=5 as float64, got %f", v)
			}
		default:
			t.Errorf("buildTaskGetJSON: unexpected size type %T", sizeVal)
		}
	}
}

func TestBuildTaskGetJSON_SizeAbsentWhenNil(t *testing.T) {
	// TC-F006-B: nil Size should not appear in the JSON map.
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			ID:    2,
			Key:   "E07-F01-002",
			Title: "Unsized Task",
			Size:  nil,
		},
		Status:   models.TaskStatus("todo"),
		Priority: 3,
	}

	result := buildTaskGetJSON(task, nil, nil, nil, nil, nil, nil, nil, nil)

	if v, ok := result["size"]; ok {
		// size should be absent or nil when task.Size == nil
		if v != nil {
			t.Errorf("buildTaskGetJSON: expected 'size' absent or nil when Size=nil, got %v (%T)", v, v)
		}
	}
	// Acceptable: key absent OR key present with nil value.
}

// ---------------------------------------------------------------------------
// buildEpicGetJSON includes "size" when non-nil
// ---------------------------------------------------------------------------

func TestBuildEpicGetJSON_SizeIncludedWhenNonNil(t *testing.T) {
	n := 8
	epic := &models.Epic{
		BaseEntity: models.BaseEntity{
			ID:    1,
			Key:   "E07",
			Title: "Test Epic",
			Size:  &n,
		},
		Status: models.EpicStatusActive,
	}
	data := &EpicGetData{
		FeaturesWithDetails: nil,
		RelatedDocs:         nil,
		FeatureRollup:       map[string]int{},
		TaskRollup:          map[string]int{},
		BlockedTasks:        nil,
	}
	result := buildEpicGetJSON(epic, data, nil)

	sizeVal, ok := result["size"]
	if !ok {
		t.Fatal("buildEpicGetJSON: expected 'size' key when Size is non-nil, but it is absent")
	}
	_ = sizeVal // We verified presence; value check via assertSizeValue below.
	assertSizeValue(t, "buildEpicGetJSON", sizeVal, 8)
}

func TestBuildEpicGetJSON_SizeAbsentWhenNil(t *testing.T) {
	epic := &models.Epic{
		BaseEntity: models.BaseEntity{
			ID:    2,
			Key:   "E08",
			Title: "Unsized Epic",
			Size:  nil,
		},
		Status: models.EpicStatusActive,
	}
	data := &EpicGetData{
		FeatureRollup: map[string]int{},
		TaskRollup:    map[string]int{},
	}
	result := buildEpicGetJSON(epic, data, nil)

	if v, ok := result["size"]; ok && v != nil {
		t.Errorf("buildEpicGetJSON: expected 'size' absent or nil when Size=nil, got %v", v)
	}
}

// ---------------------------------------------------------------------------
// buildFeatureGetJSON includes "size" when non-nil
// ---------------------------------------------------------------------------

func TestBuildFeatureGetJSON_SizeIncludedWhenNonNil(t *testing.T) {
	n := 3
	feature := &models.Feature{
		BaseEntity: models.BaseEntity{
			ID:    1,
			Key:   "E07-F01",
			Title: "Test Feature",
			Size:  &n,
		},
		Status: models.FeatureStatusActive,
	}
	data := &FeatureGetData{}
	result := buildFeatureGetJSON(feature, data, nil)

	sizeVal, ok := result["size"]
	if !ok {
		t.Fatal("buildFeatureGetJSON: expected 'size' key when Size is non-nil, but it is absent")
	}
	assertSizeValue(t, "buildFeatureGetJSON", sizeVal, 3)
}

func TestBuildFeatureGetJSON_SizeAbsentWhenNil(t *testing.T) {
	feature := &models.Feature{
		BaseEntity: models.BaseEntity{
			ID:    2,
			Key:   "E07-F02",
			Title: "Unsized Feature",
			Size:  nil,
		},
		Status: models.FeatureStatusActive,
	}
	data := &FeatureGetData{}
	result := buildFeatureGetJSON(feature, data, nil)

	if v, ok := result["size"]; ok && v != nil {
		t.Errorf("buildFeatureGetJSON: expected 'size' absent or nil when Size=nil, got %v", v)
	}
}

// ---------------------------------------------------------------------------
// buildTaskBasicInfo includes "Size" row when non-nil
// ---------------------------------------------------------------------------

func TestBuildTaskBasicInfo_SizeIncludedWhenNonNil(t *testing.T) {
	n := 5
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			ID:    1,
			Key:   "E07-F01-001",
			Title: "Test Task",
			Size:  &n,
		},
		Status:   models.TaskStatus("todo"),
		Priority: 3,
	}

	info := buildTaskBasicInfo(task, nil, nil, nil)
	sizeFound := false
	for _, row := range info {
		if len(row) >= 2 && row[0] == "Size" {
			sizeFound = true
			if !strings.Contains(row[1], "L") || !strings.Contains(row[1], "5") {
				t.Errorf("buildTaskBasicInfo: Size row expected 'L (5)', got %q", row[1])
			}
		}
	}
	if !sizeFound {
		t.Error("buildTaskBasicInfo: expected 'Size' row when task.Size=5, but not found")
	}
}

func TestBuildTaskBasicInfo_SizeAbsentWhenNil(t *testing.T) {
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			ID:    2,
			Key:   "E07-F01-002",
			Title: "Unsized Task",
			Size:  nil,
		},
		Status:   models.TaskStatus("todo"),
		Priority: 3,
	}

	info := buildTaskBasicInfo(task, nil, nil, nil)
	for _, row := range info {
		if len(row) >= 1 && row[0] == "Size" {
			t.Error("buildTaskBasicInfo: 'Size' row should not appear when task.Size=nil")
		}
	}
}

// ---------------------------------------------------------------------------
// buildBugBasicInfo includes "Size" row when non-nil
// ---------------------------------------------------------------------------

func TestBuildBugBasicInfo_SizeIncludedWhenNonNil(t *testing.T) {
	n := 2
	bug := &models.Bug{
		BaseEntity: models.BaseEntity{
			ID:    1,
			Key:   "B001",
			Title: "Test Bug",
			Size:  &n,
		},
		Status:   models.BugStatus("reported"),
		Severity: models.BugSeverity("medium"),
	}

	info := buildBugBasicInfo(bug)
	sizeFound := false
	for _, row := range info {
		if len(row) >= 2 && row[0] == "Size" {
			sizeFound = true
			if !strings.Contains(row[1], "S") || !strings.Contains(row[1], "2") {
				t.Errorf("buildBugBasicInfo: Size row expected 'S (2)', got %q", row[1])
			}
		}
	}
	if !sizeFound {
		t.Error("buildBugBasicInfo: expected 'Size' row when bug.Size=2, but not found")
	}
}

func TestBuildBugBasicInfo_SizeAbsentWhenNil(t *testing.T) {
	bug := &models.Bug{
		BaseEntity: models.BaseEntity{
			ID:    2,
			Key:   "B002",
			Title: "Unsized Bug",
			Size:  nil,
		},
		Status:   models.BugStatus("reported"),
		Severity: models.BugSeverity("low"),
	}

	info := buildBugBasicInfo(bug)
	for _, row := range info {
		if len(row) >= 1 && row[0] == "Size" {
			t.Error("buildBugBasicInfo: 'Size' row should not appear when bug.Size=nil")
		}
	}
}

// ---------------------------------------------------------------------------
// size_label is included in buildTaskGetJSON when non-nil
// REQ-F-007: --field size_label should extract the label string
// ---------------------------------------------------------------------------

func TestBuildTaskGetJSON_SizeLabelIncludedWhenNonNil(t *testing.T) {
	n := 5
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			ID:    3,
			Key:   "E07-F01-003",
			Title: "Labeled Task",
			Size:  &n,
		},
		Status:   models.TaskStatus("todo"),
		Priority: 1,
	}

	result := buildTaskGetJSON(task, nil, nil, nil, nil, nil, nil, nil, nil)

	sizeLabelVal, ok := result["size_label"]
	if !ok {
		t.Fatal("buildTaskGetJSON: expected 'size_label' key when Size=5, but it is absent")
	}
	if sizeLabelVal != "L" {
		t.Errorf("buildTaskGetJSON: expected size_label='L', got %v", sizeLabelVal)
	}
}

func TestBuildTaskGetJSON_SizeLabelAbsentWhenNil(t *testing.T) {
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			ID:    4,
			Key:   "E07-F01-004",
			Title: "No Label Task",
			Size:  nil,
		},
		Status:   models.TaskStatus("todo"),
		Priority: 1,
	}

	result := buildTaskGetJSON(task, nil, nil, nil, nil, nil, nil, nil, nil)

	if v, ok := result["size_label"]; ok && v != nil {
		t.Errorf("buildTaskGetJSON: expected 'size_label' absent or nil when Size=nil, got %v", v)
	}
}

// ---------------------------------------------------------------------------
// buildEpicGetJSON includes "size_label" when non-nil
// ---------------------------------------------------------------------------

func TestBuildEpicGetJSON_SizeLabelIncludedWhenNonNil(t *testing.T) {
	n := 13
	epic := &models.Epic{
		BaseEntity: models.BaseEntity{
			ID:    3,
			Key:   "E09",
			Title: "XXL Epic",
			Size:  &n,
		},
		Status: models.EpicStatusActive,
	}
	data := &EpicGetData{
		FeatureRollup: map[string]int{},
		TaskRollup:    map[string]int{},
	}
	result := buildEpicGetJSON(epic, data, nil)

	sizeLabelVal, ok := result["size_label"]
	if !ok {
		t.Fatal("buildEpicGetJSON: expected 'size_label' key when Size=13, but it is absent")
	}
	if sizeLabelVal != "XXL" {
		t.Errorf("buildEpicGetJSON: expected size_label='XXL', got %v", sizeLabelVal)
	}
}

// ---------------------------------------------------------------------------
// buildFeatureGetJSON includes "size_label" when non-nil
// ---------------------------------------------------------------------------

func TestBuildFeatureGetJSON_SizeLabelIncludedWhenNonNil(t *testing.T) {
	n := 1
	feature := &models.Feature{
		BaseEntity: models.BaseEntity{
			ID:    3,
			Key:   "E07-F03",
			Title: "XS Feature",
			Size:  &n,
		},
		Status: models.FeatureStatusActive,
	}
	data := &FeatureGetData{}
	result := buildFeatureGetJSON(feature, data, nil)

	sizeLabelVal, ok := result["size_label"]
	if !ok {
		t.Fatal("buildFeatureGetJSON: expected 'size_label' key when Size=1, but it is absent")
	}
	if sizeLabelVal != "XS" {
		t.Errorf("buildFeatureGetJSON: expected size_label='XS', got %v", sizeLabelVal)
	}
}

// ---------------------------------------------------------------------------
// assertSizeValue is a shared helper for checking size values in JSON maps.
// Accepts *int, int, or float64 (JSON numbers deserialize as float64).
// ---------------------------------------------------------------------------

func assertSizeValue(t *testing.T, label string, sizeVal interface{}, expected int) {
	t.Helper()
	switch v := sizeVal.(type) {
	case *int:
		if *v != expected {
			t.Errorf("%s: expected size=%d, got *int(%d)", label, expected, *v)
		}
	case int:
		if v != expected {
			t.Errorf("%s: expected size=%d, got int(%d)", label, expected, v)
		}
	case float64:
		if int(v) != expected {
			t.Errorf("%s: expected size=%d, got float64(%f)", label, expected, v)
		}
	default:
		t.Errorf("%s: unexpected size type %T = %v", label, sizeVal, sizeVal)
	}
}
