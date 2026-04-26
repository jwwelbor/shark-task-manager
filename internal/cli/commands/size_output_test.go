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
// F3 — size_label in JSON output for bug, change-card, and idea (via buildEnrichedJSON)
// REQ-F-007: --field size_label must work for all 6 entity types.
// ---------------------------------------------------------------------------

func TestBuildEnrichedJSONBug_SizeLabelIncludedWhenNonNil(t *testing.T) {
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

	result, err := buildEnrichedJSON(bug, nil, nil)
	if err != nil {
		t.Fatalf("buildEnrichedJSON(bug): unexpected error: %v", err)
	}

	sizeLabelVal, ok := result["size_label"]
	if !ok {
		t.Fatal("buildEnrichedJSON(bug): expected 'size_label' key when Size=2, but it is absent")
	}
	if sizeLabelVal != "S" {
		t.Errorf("buildEnrichedJSON(bug): expected size_label='S', got %v", sizeLabelVal)
	}
}

func TestBuildEnrichedJSONBug_SizeLabelAbsentWhenNil(t *testing.T) {
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

	result, err := buildEnrichedJSON(bug, nil, nil)
	if err != nil {
		t.Fatalf("buildEnrichedJSON(bug): unexpected error: %v", err)
	}

	if v, ok := result["size_label"]; ok && v != nil {
		t.Errorf("buildEnrichedJSON(bug): expected 'size_label' absent or nil when Size=nil, got %v", v)
	}
}

func TestBuildEnrichedJSONChange_SizeLabelIncludedWhenNonNil(t *testing.T) {
	n := 8
	card := &models.ChangeCard{
		BaseEntity: models.BaseEntity{
			ID:    1,
			Key:   "CC-001",
			Title: "Test Change",
			Size:  &n,
		},
		Status:   models.ChangeCardStatus("draft"),
		Priority: 3,
	}

	result, err := buildEnrichedJSON(card, nil, nil)
	if err != nil {
		t.Fatalf("buildEnrichedJSON(change): unexpected error: %v", err)
	}

	sizeLabelVal, ok := result["size_label"]
	if !ok {
		t.Fatal("buildEnrichedJSON(change): expected 'size_label' key when Size=8, but it is absent")
	}
	if sizeLabelVal != "XL" {
		t.Errorf("buildEnrichedJSON(change): expected size_label='XL', got %v", sizeLabelVal)
	}
}

func TestBuildEnrichedJSONChange_SizeLabelAbsentWhenNil(t *testing.T) {
	card := &models.ChangeCard{
		BaseEntity: models.BaseEntity{
			ID:    2,
			Key:   "CC-002",
			Title: "Unsized Change",
			Size:  nil,
		},
		Status:   models.ChangeCardStatus("draft"),
		Priority: 2,
	}

	result, err := buildEnrichedJSON(card, nil, nil)
	if err != nil {
		t.Fatalf("buildEnrichedJSON(change): unexpected error: %v", err)
	}

	if v, ok := result["size_label"]; ok && v != nil {
		t.Errorf("buildEnrichedJSON(change): expected 'size_label' absent or nil when Size=nil, got %v", v)
	}
}

func TestIdeaGetJSON_SizeLabelIncludedWhenNonNil(t *testing.T) {
	n := 13
	idea := &models.Idea{
		ID:     1,
		Key:    "I-2026-01-01-01",
		Title:  "XXL Idea",
		Size:   &n,
		Status: models.IdeaStatusNew,
	}

	result := buildIdeaGetJSON(idea, []string{})

	sizeLabelVal, ok := result["size_label"]
	if !ok {
		t.Fatal("buildIdeaGetJSON: expected 'size_label' key when Size=13, but it is absent")
	}
	if sizeLabelVal != "XXL" {
		t.Errorf("buildIdeaGetJSON: expected size_label='XXL', got %v", sizeLabelVal)
	}
}

func TestIdeaGetJSON_SizeLabelAbsentWhenNil(t *testing.T) {
	idea := &models.Idea{
		ID:     2,
		Key:    "I-2026-01-01-02",
		Title:  "Unsized Idea",
		Size:   nil,
		Status: models.IdeaStatusNew,
	}

	result := buildIdeaGetJSON(idea, []string{})

	if v, ok := result["size_label"]; ok && v != nil {
		t.Errorf("buildIdeaGetJSON: expected 'size_label' absent or nil when Size=nil, got %v", v)
	}
}

// ---------------------------------------------------------------------------
// F4 — Size column in list-view tables for all 6 entity types.
// ---------------------------------------------------------------------------

func TestPrintTaskTable_SizeColumn(t *testing.T) {
	n := 3
	tasks := []*models.Task{
		{
			BaseEntity: models.BaseEntity{
				Key:   "E07-F01-001",
				Title: "Sized Task",
				Size:  &n,
			},
			Status:   models.TaskStatus("todo"),
			Priority: 5,
		},
		{
			BaseEntity: models.BaseEntity{
				Key:   "E07-F01-002",
				Title: "Unsized Task",
				Size:  nil,
			},
			Status:   models.TaskStatus("in_progress"),
			Priority: 3,
		},
	}

	rows := buildTaskListRows(tasks)
	if len(rows) != 2 {
		t.Fatalf("buildTaskListRows: expected 2 rows, got %d", len(rows))
	}

	// Row for sized task: should include "M (3)" as Size cell.
	sizedRow := rows[0]
	sizeFound := false
	for _, cell := range sizedRow {
		if cell == "M (3)" {
			sizeFound = true
			break
		}
	}
	if !sizeFound {
		t.Errorf("buildTaskListRows: expected 'M (3)' cell for sized task, row=%v", sizedRow)
	}

	// Row for unsized task: should include "—" as Size cell.
	unsizedRow := rows[1]
	dashFound := false
	for _, cell := range unsizedRow {
		if cell == "—" {
			dashFound = true
			break
		}
	}
	if !dashFound {
		t.Errorf("buildTaskListRows: expected '—' cell for unsized task, row=%v", unsizedRow)
	}
}

func TestPrintBugTable_SizeColumn(t *testing.T) {
	n := 5
	bugs := []*models.Bug{
		{
			BaseEntity: models.BaseEntity{
				Key:   "B001",
				Title: "Sized Bug",
				Size:  &n,
			},
			Status:   models.BugStatus("reported"),
			Severity: models.BugSeverity("high"),
		},
		{
			BaseEntity: models.BaseEntity{
				Key:   "B002",
				Title: "Unsized Bug",
				Size:  nil,
			},
			Status:   models.BugStatus("confirmed"),
			Severity: models.BugSeverity("low"),
		},
	}

	rows := buildBugListRows(bugs)
	if len(rows) != 2 {
		t.Fatalf("buildBugListRows: expected 2 rows, got %d", len(rows))
	}

	// Sized bug row: should contain "L (5)".
	sizeFound := false
	for _, cell := range rows[0] {
		if cell == "L (5)" {
			sizeFound = true
			break
		}
	}
	if !sizeFound {
		t.Errorf("buildBugListRows: expected 'L (5)' for sized bug, row=%v", rows[0])
	}

	// Unsized bug row: should contain "—".
	dashFound := false
	for _, cell := range rows[1] {
		if cell == "—" {
			dashFound = true
			break
		}
	}
	if !dashFound {
		t.Errorf("buildBugListRows: expected '—' for unsized bug, row=%v", rows[1])
	}
}

func TestPrintIdeaList_SizeColumn(t *testing.T) {
	n := 1
	ideas := []*models.Idea{
		{
			Key:    "I-2026-01-01-01",
			Title:  "Sized Idea",
			Size:   &n,
			Status: models.IdeaStatusNew,
		},
		{
			Key:    "I-2026-01-01-02",
			Title:  "Unsized Idea",
			Size:   nil,
			Status: models.IdeaStatusNew,
		},
	}

	rows := buildIdeaListRows(ideas)
	if len(rows) != 2 {
		t.Fatalf("buildIdeaListRows: expected 2 rows, got %d", len(rows))
	}

	// Sized idea row: should contain "XS (1)".
	sizeFound := false
	for _, cell := range rows[0] {
		if cell == "XS (1)" {
			sizeFound = true
			break
		}
	}
	if !sizeFound {
		t.Errorf("buildIdeaListRows: expected 'XS (1)' for sized idea, row=%v", rows[0])
	}

	// Unsized idea row: should contain "—".
	dashFound := false
	for _, cell := range rows[1] {
		if cell == "—" {
			dashFound = true
			break
		}
	}
	if !dashFound {
		t.Errorf("buildIdeaListRows: expected '—' for unsized idea, row=%v", rows[1])
	}
}

func TestRenderEpicListTable_SizeColumn(t *testing.T) {
	n := 8
	epics := []EpicWithProgress{
		{
			Epic: &models.Epic{
				BaseEntity: models.BaseEntity{
					Key:   "E07",
					Title: "Sized Epic",
					Size:  &n,
				},
				Status:   models.EpicStatusActive,
				Priority: "high",
			},
			ProgressPct: 50.0,
		},
		{
			Epic: &models.Epic{
				BaseEntity: models.BaseEntity{
					Key:   "E08",
					Title: "Unsized Epic",
					Size:  nil,
				},
				Status:   models.EpicStatusActive,
				Priority: "medium",
			},
			ProgressPct: 25.0,
		},
	}

	rows := buildEpicListRows(epics)
	if len(rows) != 2 {
		t.Fatalf("buildEpicListRows: expected 2 rows, got %d", len(rows))
	}

	// Sized epic row: should contain "XL (8)".
	sizeFound := false
	for _, cell := range rows[0] {
		if cell == "XL (8)" {
			sizeFound = true
			break
		}
	}
	if !sizeFound {
		t.Errorf("buildEpicListRows: expected 'XL (8)' for sized epic, row=%v", rows[0])
	}

	// Unsized epic row: should contain "—".
	dashFound := false
	for _, cell := range rows[1] {
		if cell == "—" {
			dashFound = true
			break
		}
	}
	if !dashFound {
		t.Errorf("buildEpicListRows: expected '—' for unsized epic, row=%v", rows[1])
	}
}

func TestPrintChangeCardList_SizeColumn(t *testing.T) {
	n := 3
	cards := []*models.ChangeCard{
		{
			BaseEntity: models.BaseEntity{
				Key:   "CC-001",
				Title: "Sized Change",
				Size:  &n,
			},
			Status:   models.ChangeCardStatus("draft"),
			Priority: 2,
		},
		{
			BaseEntity: models.BaseEntity{
				Key:   "CC-002",
				Title: "Unsized Change",
				Size:  nil,
			},
			Status:   models.ChangeCardStatus("draft"),
			Priority: 1,
		},
	}

	rows := buildChangeCardListRows(cards)
	if len(rows) != 2 {
		t.Fatalf("buildChangeCardListRows: expected 2 rows, got %d", len(rows))
	}

	// Sized change row: should contain "M (3)".
	sizeFound := false
	for _, cell := range rows[0] {
		if cell == "M (3)" {
			sizeFound = true
			break
		}
	}
	if !sizeFound {
		t.Errorf("buildChangeCardListRows: expected 'M (3)' for sized change, row=%v", rows[0])
	}

	// Unsized change row: should contain "—".
	dashFound := false
	for _, cell := range rows[1] {
		if cell == "—" {
			dashFound = true
			break
		}
	}
	if !dashFound {
		t.Errorf("buildChangeCardListRows: expected '—' for unsized change, row=%v", rows[1])
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
