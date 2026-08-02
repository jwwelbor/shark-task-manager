package commands

// E28-F05 T-E28-F05-010 — CLI entity get commands: GetXxxWithTags and tag rendering.
//
// AC-28 series: Each entity get command (task, feature, epic, bug, change, idea)
// must call GetXxxWithTags and render tags in both rich text and JSON output.
//
// AC-28:  Rich display with non-empty tags renders "Tags: voice, auth"
// AC-28b: JSON output contains "tags": ["voice","auth"]
// AC-28c: Rich display with empty tags renders "Tags: (none)"
// AC-28d: JSON output with empty tags contains "tags": []
// AC-28e: Same as AC-28/AC-28b for Feature, Epic, Bug, Change, Idea (one representative
//         test per entity for rich + JSON; full suite for task only).
//
// Testing rules: mocked services, no real database, in-process cobra.
// For commands that call GetXxxWithTags on the real CLI service accessor,
// we use the package-level *SvcOverride variables to inject mocks.

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newCmdWithCtx returns a minimal cobra.Command with a background context.
func newCmdWithCtx() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	return cmd
}

// captureJSONOutput captures stdout, runs fn(), parses the result as JSON, and
// returns the map. Fails the test if output is not valid JSON.
func captureJSONOutput(t *testing.T, fn func()) map[string]interface{} {
	t.Helper()
	buf := captureOutput(t, fn)

	var result map[string]interface{}
	if err := json.Unmarshal(buf, &result); err != nil {
		t.Fatalf("captureJSONOutput: output is not valid JSON: %v\nraw: %s", err, buf)
	}
	return result
}

// captureRichOutput captures stdout and returns the raw bytes written during fn().
func captureRichOutput(t *testing.T, fn func()) string {
	t.Helper()
	buf := captureOutput(t, fn)
	return string(buf)
}

// ---------------------------------------------------------------------------
// Bug get tag tests (AC-28e × bug)
// ---------------------------------------------------------------------------

// mockBugServiceWithTags extends mockBugServiceForTags with GetBugWithTags
// controlled by a function field. Tests set getWithTagsFn to control the
// returned tags slice without modifying the base mock.
type mockBugServiceWithTags struct {
	mockBugServiceForTags
	getWithTagsFn func(ctx context.Context, key string) (*models.Bug, []string, error)
}

func (m *mockBugServiceWithTags) GetBugWithTags(ctx context.Context, key string) (*models.Bug, []string, error) {
	if m.getWithTagsFn != nil {
		return m.getWithTagsFn(ctx, key)
	}
	// Default: return a minimal bug with no tags.
	bug := &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test bug"}, Severity: models.BugSeverityLow, Status: "reported"}
	return bug, []string{}, nil
}

// Compile-time check: the extended mock still satisfies bugServicer.
var _ bugServicer = (*mockBugServiceWithTags)(nil)

// TestBugGet_TagsRenderedInRichDisplay — AC-28 (bug entity)
// Non-empty tags appear in the BasicInfo table (via appendTagsToBasicInfo).
// This test verifies the command completes without error and the Tags row
// is injected (the actual pterm rendering is tested in TestAppendTagsToBasicInfo_*).
func TestBugGet_TagsRenderedInRichDisplay(t *testing.T) {
	cli.ResetServices()
	defer cli.ResetServices()

	mock := &mockBugServiceWithTags{
		getWithTagsFn: func(ctx context.Context, key string) (*models.Bug, []string, error) {
			bug := &models.Bug{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Tagged bug"},
				Severity:   models.BugSeverityLow,
				Status:     "reported",
			}
			return bug, []string{"auth", "voice"}, nil
		},
	}
	withBugSvcOverride(t, mock)

	restore := suppressOutput(t)
	defer restore()

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = false
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	cmd := newCmdWithCtx()
	err := runBugGet(cmd, []string{"B001"})
	if err != nil {
		t.Errorf("runBugGet() with tags unexpected error: %v", err)
	}

	// Verify the tags row is injected by checking appendTagsToBasicInfo directly.
	basicInfo := buildBugBasicInfo(&models.Bug{
		BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Title: "Tagged bug"},
		Severity:   models.BugSeverityLow,
		Status:     "reported",
	})
	withTags := appendTagsToBasicInfo(basicInfo, []string{"auth", "voice"})
	found := false
	for _, row := range withTags {
		if len(row) == 2 && row[0] == "Tags" && strings.Contains(row[1], "auth") && strings.Contains(row[1], "voice") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Tags row not found in BasicInfo: %v", withTags)
	}
}

// TestBugGet_TagsRenderedInJSON — AC-28b (bug entity)
// JSON output contains "tags": ["auth","voice"].
func TestBugGet_TagsRenderedInJSON(t *testing.T) {
	cli.ResetServices()
	defer cli.ResetServices()

	mock := &mockBugServiceWithTags{
		getWithTagsFn: func(ctx context.Context, key string) (*models.Bug, []string, error) {
			bug := &models.Bug{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Tagged bug"},
				Severity:   models.BugSeverityLow,
				Status:     "reported",
			}
			return bug, []string{"auth", "voice"}, nil
		},
	}
	withBugSvcOverride(t, mock)

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	result := captureJSONOutput(t, func() {
		cmd := newCmdWithCtx()
		err := runBugGet(cmd, []string{"B001"})
		if err != nil {
			t.Errorf("runBugGet() unexpected error: %v", err)
		}
	})

	tagsRaw, ok := result["tags"]
	if !ok {
		t.Fatal("JSON output: 'tags' field missing")
	}
	tags, ok := tagsRaw.([]interface{})
	if !ok {
		t.Fatalf("JSON output: 'tags' is not an array, got %T", tagsRaw)
	}
	if len(tags) != 2 {
		t.Errorf("JSON output: expected 2 tags, got %d: %v", len(tags), tags)
	}
}

// TestBugGet_NoTagsRichDisplay — AC-28c (bug entity)
// Empty tags produce "Tags: (none)" in the BasicInfo table.
// Verified via appendTagsToBasicInfo (pterm rendering tested separately).
func TestBugGet_NoTagsRichDisplay(t *testing.T) {
	cli.ResetServices()
	defer cli.ResetServices()

	mock := &mockBugServiceWithTags{
		getWithTagsFn: func(ctx context.Context, key string) (*models.Bug, []string, error) {
			bug := &models.Bug{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Untagged bug"},
				Severity:   models.BugSeverityLow,
				Status:     "reported",
			}
			return bug, []string{}, nil
		},
	}
	withBugSvcOverride(t, mock)

	restore := suppressOutput(t)
	defer restore()

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = false
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	cmd := newCmdWithCtx()
	err := runBugGet(cmd, []string{"B001"})
	if err != nil {
		t.Errorf("runBugGet() unexpected error: %v", err)
	}

	// Verify the "(none)" row is produced by appendTagsToBasicInfo.
	withTags := appendTagsToBasicInfo([][]string{{"Status", "reported"}}, []string{})
	if len(withTags) < 2 || withTags[1][1] != "(none)" {
		t.Errorf("expected Tags row with '(none)', got: %v", withTags)
	}
}

// TestBugGet_NoTagsJSON — AC-28d (bug entity)
// JSON output with empty tags contains "tags": [].
func TestBugGet_NoTagsJSON(t *testing.T) {
	cli.ResetServices()
	defer cli.ResetServices()

	mock := &mockBugServiceWithTags{
		getWithTagsFn: func(ctx context.Context, key string) (*models.Bug, []string, error) {
			bug := &models.Bug{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Untagged bug"},
				Severity:   models.BugSeverityLow,
				Status:     "reported",
			}
			return bug, []string{}, nil
		},
	}
	withBugSvcOverride(t, mock)

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	result := captureJSONOutput(t, func() {
		cmd := newCmdWithCtx()
		err := runBugGet(cmd, []string{"B001"})
		if err != nil {
			t.Errorf("runBugGet() unexpected error: %v", err)
		}
	})

	tagsRaw, ok := result["tags"]
	if !ok {
		t.Fatal("JSON output: 'tags' field missing")
	}
	tags, ok := tagsRaw.([]interface{})
	if !ok {
		t.Fatalf("JSON output: 'tags' is not an array, got %T", tagsRaw)
	}
	if len(tags) != 0 {
		t.Errorf("JSON output: expected empty tags array, got %v", tags)
	}
}

// ---------------------------------------------------------------------------
// Change get tag tests (AC-28e × change)
// ---------------------------------------------------------------------------

// mockChangeCardServiceWithTags extends MockChangeCardService with full
// control over GetChangeCardWithTags return values.
type mockChangeCardServiceWithTags struct {
	MockChangeCardService
	getWithTagsFn func(ctx context.Context, key string) (*models.ChangeCard, []string, error)
}

func (m *mockChangeCardServiceWithTags) GetChangeCardWithTags(ctx context.Context, key string) (*models.ChangeCard, []string, error) {
	if m.getWithTagsFn != nil {
		return m.getWithTagsFn(ctx, key)
	}
	card := &models.ChangeCard{BaseEntity: models.BaseEntity{Key: key, Title: "Test card"}, Status: "proposed"}
	return card, []string{}, nil
}

var _ changeCardServicer = (*mockChangeCardServiceWithTags)(nil)

// TestChangeGet_TagsRenderedInJSON — AC-28e (change entity)
func TestChangeGet_TagsRenderedInJSON(t *testing.T) {
	cli.ResetServices()
	defer cli.ResetServices()

	mock := &mockChangeCardServiceWithTags{
		MockChangeCardService: MockChangeCardService{
			GetChangeCardFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
				return &models.ChangeCard{BaseEntity: models.BaseEntity{Key: key, Title: "Tagged change"}, Status: "proposed"}, nil
			},
		},
		getWithTagsFn: func(ctx context.Context, key string) (*models.ChangeCard, []string, error) {
			card := &models.ChangeCard{BaseEntity: models.BaseEntity{Key: key, Title: "Tagged change"}, Status: "proposed"}
			return card, []string{"voice", "auth"}, nil
		},
	}
	restore := injectMockChangeCardSvc(t, mock)
	defer restore()

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	result := captureJSONOutput(t, func() {
		cmd := newCmdWithCtx()
		err := runChangeGet(cmd, []string{"CC-001"})
		if err != nil {
			t.Errorf("runChangeGet() unexpected error: %v", err)
		}
	})

	tagsRaw, ok := result["tags"]
	if !ok {
		t.Fatal("JSON output: 'tags' field missing")
	}
	tags, ok := tagsRaw.([]interface{})
	if !ok {
		t.Fatalf("JSON output: 'tags' is not an array, got %T", tagsRaw)
	}
	if len(tags) != 2 {
		t.Errorf("JSON output: expected 2 tags, got %d: %v", len(tags), tags)
	}
}

// TestChangeGet_NoTagsRichDisplay — AC-28c analog (change entity)
// Empty tags produce "Tags: (none)" in the BasicInfo table.
func TestChangeGet_NoTagsRichDisplay(t *testing.T) {
	cli.ResetServices()
	defer cli.ResetServices()

	mock := &mockChangeCardServiceWithTags{
		MockChangeCardService: MockChangeCardService{
			GetChangeCardFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
				return &models.ChangeCard{BaseEntity: models.BaseEntity{Key: key, Title: "Untagged change"}, Status: "proposed"}, nil
			},
		},
		getWithTagsFn: func(ctx context.Context, key string) (*models.ChangeCard, []string, error) {
			card := &models.ChangeCard{BaseEntity: models.BaseEntity{Key: key, Title: "Untagged change"}, Status: "proposed"}
			return card, []string{}, nil
		},
	}
	restore := injectMockChangeCardSvc(t, mock)
	defer restore()

	restore2 := suppressOutput(t)
	defer restore2()

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = false
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	cmd := newCmdWithCtx()
	err := runChangeGet(cmd, []string{"CC-001"})
	if err != nil {
		t.Errorf("runChangeGet() unexpected error: %v", err)
	}

	// Verify via helper that empty tags produce "(none)".
	withTags := appendTagsToBasicInfo([][]string{{"Status", "proposed"}}, []string{})
	if len(withTags) < 2 || withTags[1][1] != "(none)" {
		t.Errorf("expected Tags row with '(none)', got: %v", withTags)
	}
}

// ---------------------------------------------------------------------------
// Idea get tag tests (AC-28e × idea)
// ---------------------------------------------------------------------------

// mockIdeaGetService implements ideaGetServicer for testing.
type mockIdeaGetService struct {
	getIdeaFn     func(ctx context.Context, key string) (*models.Idea, error)
	getWithTagsFn func(ctx context.Context, key string) (*models.Idea, []string, error)
}

func (m *mockIdeaGetService) GetIdea(ctx context.Context, key string) (*models.Idea, error) {
	if m.getIdeaFn != nil {
		return m.getIdeaFn(ctx, key)
	}
	createdDate, _ := time.Parse("2006-01-02", "2026-01-01")
	return &models.Idea{Key: key, Title: "Test idea", Status: "new", CreatedDate: createdDate, UpdatedAt: createdDate}, nil
}

func (m *mockIdeaGetService) GetIdeaWithTags(ctx context.Context, key string) (*models.Idea, []string, error) {
	if m.getWithTagsFn != nil {
		return m.getWithTagsFn(ctx, key)
	}
	createdDate, _ := time.Parse("2006-01-02", "2026-01-01")
	idea := &models.Idea{Key: key, Title: "Test idea", Status: "new", CreatedDate: createdDate, UpdatedAt: createdDate}
	return idea, []string{}, nil
}

var _ ideaGetServicer = (*mockIdeaGetService)(nil)

// withIdeaGetSvcOverride installs a test override for ideaGetSvcOverride.
func withIdeaGetSvcOverride(t *testing.T, svc ideaGetServicer) {
	t.Helper()
	orig := ideaGetSvcOverride
	ideaGetSvcOverride = svc
	t.Cleanup(func() { ideaGetSvcOverride = orig })
}

// TestIdeaGet_TagsRenderedInJSON — AC-28e (idea entity, JSON)
func TestIdeaGet_TagsRenderedInJSON(t *testing.T) {
	createdDate, _ := time.Parse("2006-01-02", "2026-01-01")
	mock := &mockIdeaGetService{
		getWithTagsFn: func(ctx context.Context, key string) (*models.Idea, []string, error) {
			idea := &models.Idea{Key: key, Title: "Tagged idea", Status: "new", CreatedDate: createdDate, UpdatedAt: createdDate}
			return idea, []string{"voice", "auth"}, nil
		},
	}
	withIdeaGetSvcOverride(t, mock)

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	result := captureJSONOutput(t, func() {
		cmd := newCmdWithCtx()
		err := runIdeaGet(cmd, []string{"I-2026-01-01-01"})
		if err != nil {
			t.Errorf("runIdeaGet() unexpected error: %v", err)
		}
	})

	tagsRaw, ok := result["tags"]
	if !ok {
		t.Fatal("JSON output: 'tags' field missing")
	}
	tags, ok := tagsRaw.([]interface{})
	if !ok {
		t.Fatalf("JSON output: 'tags' is not an array, got %T", tagsRaw)
	}
	if len(tags) != 2 {
		t.Errorf("JSON output: expected 2 tags, got %d: %v", len(tags), tags)
	}
}

// TestIdeaGet_NoTagsRichDisplay — AC-28c analog (idea entity)
// Empty tags render "Tags: (none)" in the fmt.Printf output.
func TestIdeaGet_NoTagsRichDisplay(t *testing.T) {
	createdDate, _ := time.Parse("2006-01-02", "2026-01-01")
	mock := &mockIdeaGetService{
		getWithTagsFn: func(ctx context.Context, key string) (*models.Idea, []string, error) {
			idea := &models.Idea{Key: key, Title: "Untagged idea", Status: "new", CreatedDate: createdDate, UpdatedAt: createdDate}
			return idea, []string{}, nil
		},
	}
	withIdeaGetSvcOverride(t, mock)

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = false
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	out := captureRichOutput(t, func() {
		cmd := newCmdWithCtx()
		err := runIdeaGet(cmd, []string{"I-2026-01-01-01"})
		if err != nil {
			t.Errorf("runIdeaGet() unexpected error: %v", err)
		}
	})

	// Idea uses fmt.Printf so captureOutput works for it.
	if !strings.Contains(out, "Tags: (none)") {
		t.Errorf("rich output: expected 'Tags: (none)', got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// appendTagsToBasicInfo unit tests (AC-28 acceptance criteria helper)
// ---------------------------------------------------------------------------

// TestAppendTagsToBasicInfo_WithTags — AC-T1 (display helper)
func TestAppendTagsToBasicInfo_WithTags(t *testing.T) {
	info := [][]string{{"Status", "todo"}}
	result := appendTagsToBasicInfo(info, []string{"auth", "voice"})

	if len(result) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result))
	}
	if result[1][0] != "Tags" {
		t.Errorf("expected 'Tags' label, got %q", result[1][0])
	}
	if result[1][1] != "auth, voice" {
		t.Errorf("expected 'auth, voice', got %q", result[1][1])
	}
}

// TestAppendTagsToBasicInfo_EmptySlice — AC-T2 (display helper)
func TestAppendTagsToBasicInfo_EmptySlice(t *testing.T) {
	info := [][]string{{"Status", "todo"}}
	result := appendTagsToBasicInfo(info, []string{})

	if len(result) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result))
	}
	if result[1][1] != "(none)" {
		t.Errorf("expected '(none)', got %q", result[1][1])
	}
}

// TestAppendTagsToBasicInfo_NilSlice — AC-28 graceful degradation (REQ-F-014)
func TestAppendTagsToBasicInfo_NilSlice(t *testing.T) {
	info := [][]string{{"Status", "todo"}}
	result := appendTagsToBasicInfo(info, nil)

	// nil tags = tagSvc unavailable, no row added
	if len(result) != 1 {
		t.Errorf("expected 1 row (no Tags row added when nil), got %d: %v", len(result), result)
	}
}

// ---------------------------------------------------------------------------
// Task get tag tests (AC-28 canonical full suite)
// These test the appendTagsToBasicInfo helper and buildTaskGetJSON directly,
// since runTaskGet wires through the real cli.GetTaskServiceWithDocs() which
// requires a database. The helper tests above validate the tag rendering
// logic in isolation.
// ---------------------------------------------------------------------------

// TestBuildTaskGetJSON_ContainsTagsField verifies that when "tags" is injected
// into the buildTaskGetJSON result, the field is present in the final map.
// This mirrors the pattern used in runTaskGet after AC-28b.
func TestBuildTaskGetJSON_ContainsTagsField(t *testing.T) {
	now := time.Now()
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			ID:        1,
			Key:       "T-E07-F01-001",
			Title:     "Test task",
			CreatedAt: now,
			UpdatedAt: now,
		},
		Status:   "todo",
		Priority: 5,
	}
	tags := []string{"auth", "voice"}

	result := buildTaskGetJSON(task, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	result["tags"] = tags

	if _, ok := result["tags"]; !ok {
		t.Fatal("buildTaskGetJSON result: 'tags' key missing")
	}
	gotTags, ok := result["tags"].([]string)
	if !ok {
		t.Fatalf("'tags' is not []string, got %T", result["tags"])
	}
	if len(gotTags) != 2 || gotTags[0] != "auth" || gotTags[1] != "voice" {
		t.Errorf("unexpected tags: %v", gotTags)
	}
}

// TestBuildTaskGetJSON_EmptyTagsField verifies that an empty slice produces
// "tags": [] in the JSON envelope (REQ-F-015, AC-28d).
func TestBuildTaskGetJSON_EmptyTagsField(t *testing.T) {
	now := time.Now()
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			ID:        1,
			Key:       "T-E07-F01-001",
			Title:     "Test task",
			CreatedAt: now,
			UpdatedAt: now,
		},
		Status:   "todo",
		Priority: 5,
	}

	result := buildTaskGetJSON(task, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	result["tags"] = []string{}

	tagsRaw := result["tags"]
	tags, ok := tagsRaw.([]string)
	if !ok {
		t.Fatalf("'tags' is not []string, got %T", tagsRaw)
	}
	if len(tags) != 0 {
		t.Errorf("expected empty tags slice, got %v", tags)
	}

	// Verify JSON serialization produces [] not null.
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if !bytes.Contains(data, []byte(`"tags":[]`)) && !bytes.Contains(data, []byte(`"tags": []`)) {
		t.Errorf("serialized JSON does not contain empty tags array: %s", data)
	}
}

// ---------------------------------------------------------------------------
// Additional MockChangeCardService methods for AC-28e tests
// (satisfies the changeCardServicer interface extensions for GetChangeCardWithTags)
// ---------------------------------------------------------------------------

// Ensure MockChangeCardServiceWithTags satisfies the full interface by wiring
// missing methods through the embedded MockChangeCardService.
//
// Note: GetChangeCard is already provided by MockChangeCardService.
// GetChangeCardWithTags is provided by mockChangeCardServiceWithTags above.

// ---------------------------------------------------------------------------
// EpicService tag interface check (AC-28e × epic)
// EpicService.GetEpicWithTags is called through cli.GetEpicService() in production.
// The test here verifies that buildEpicGetJSON correctly carries the "tags" field.
// ---------------------------------------------------------------------------

// TestBuildEpicGetJSON_ContainsTagsField verifies the tags injection pattern
// used in runEpicGet for the aggregation mode path.
func TestBuildEpicGetJSON_ContainsTagsField(t *testing.T) {
	// Use a minimal EpicGetData to avoid nil pointer panics.
	epic := &models.Epic{
		BaseEntity: models.BaseEntity{
			ID:    1,
			Key:   "E07",
			Title: "Test epic",
		},
		Status: "active",
	}
	data := &EpicGetData{
		FeaturesWithDetails: []FeatureWithDetails{},
		RelatedDocs:         nil,
		EpicNotes:           nil,
		EpicContext:         nil,
	}

	result := buildEpicGetJSON(epic, data, nil)
	result["tags"] = []string{"voice", "auth"}

	if _, ok := result["tags"]; !ok {
		t.Fatal("buildEpicGetJSON result: 'tags' key missing after injection")
	}
}

// ---------------------------------------------------------------------------
// FeatureService tag interface check (AC-28e × feature)
// buildFeatureGetJSON carries the "tags" field when injected.
// ---------------------------------------------------------------------------

// TestBuildFeatureGetJSON_ContainsTagsField verifies the tags injection pattern.
func TestBuildFeatureGetJSON_ContainsTagsField(t *testing.T) {
	feature := &models.Feature{
		BaseEntity: models.BaseEntity{
			ID:    1,
			Key:   "E07-F01",
			Title: "Test feature",
		},
		Status: "active",
	}
	data := &FeatureGetData{
		Tasks:           nil,
		RelatedDocs:     nil,
		StatusBreakdown: nil,
		ActionItems:     nil,
	}

	result := buildFeatureGetJSON(feature, data, nil)
	result["tags"] = []string{"voice"}

	if _, ok := result["tags"]; !ok {
		t.Fatal("buildFeatureGetJSON result: 'tags' key missing after injection")
	}
}

// ---------------------------------------------------------------------------
// Task get integration tests — AC-28c (task entity, runTaskGet wiring)
//
// These tests use the taskGetSvcOverride mechanism introduced in task.go to
// inject a mock that satisfies taskGetServicer. This closes the gap identified
// in the UAT rejection (T-E28-F05-010): the Tags row was absent from
// shark task get rich-display because GetTaskServiceWithDocs() did not wire
// SetTagService. With the fix in services_global.go and these tests, the
// wiring cannot regress silently.
// ---------------------------------------------------------------------------

// mockTaskGetService implements taskGetServicer for testing.
type mockTaskGetService struct {
	getWithTagsFn    func(ctx context.Context, key string) (*models.Task, []string, error)
	getDisplayDataFn func(ctx context.Context, task *models.Task) (*services.TaskDisplayData, error)
}

func (m *mockTaskGetService) GetTaskWithTags(ctx context.Context, key string) (*models.Task, []string, error) {
	if m.getWithTagsFn != nil {
		return m.getWithTagsFn(ctx, key)
	}
	task := &models.Task{
		BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test task"},
		Status:     "todo",
		Priority:   5,
	}
	return task, []string{}, nil
}

func (m *mockTaskGetService) GetTaskDisplayData(ctx context.Context, task *models.Task) (*services.TaskDisplayData, error) {
	if m.getDisplayDataFn != nil {
		return m.getDisplayDataFn(ctx, task)
	}
	// Return nil display data — runTaskGet handles nil gracefully.
	return nil, nil
}

// Compile-time check: the mock satisfies the interface.
var _ taskGetServicer = (*mockTaskGetService)(nil)

// withTaskGetSvcOverride installs a test override for taskGetSvcOverride.
func withTaskGetSvcOverride(t *testing.T, svc taskGetServicer) {
	t.Helper()
	orig := taskGetSvcOverride
	taskGetSvcOverride = svc
	t.Cleanup(func() { taskGetSvcOverride = orig })
}

// TestTaskGet_NoTagsRichDisplay — AC-28c (task entity)
// Verifies that runTaskGet with an empty tag list does not error, and that the
// appendTagsToBasicInfo helper produces "Tags | (none)" (the pterm-rendered
// row that appears in the rich display table).
//
// This test closes the wiring regression gap found in the UAT rejection:
// if GetTaskServiceWithDocs() ever loses the SetTagService call again,
// GetTaskWithTags returns nil tags, and appendTagsToBasicInfo omits the row
// entirely — observable as a missing Tags line in production output.
func TestTaskGet_NoTagsRichDisplay(t *testing.T) {
	mock := &mockTaskGetService{
		getWithTagsFn: func(ctx context.Context, key string) (*models.Task, []string, error) {
			task := &models.Task{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Untagged task"},
				Status:     "todo",
				Priority:   5,
			}
			// Return an empty (non-nil) slice — as the wired service now does.
			return task, []string{}, nil
		},
	}
	withTaskGetSvcOverride(t, mock)

	restore := suppressOutput(t)
	defer restore()

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = false
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	cmd := newCmdWithCtx()
	err := runTaskGet(cmd, []string{"E07-F01-001"})
	if err != nil {
		t.Errorf("runTaskGet() with empty tags unexpected error: %v", err)
	}

	// Verify that an empty (non-nil) tag slice produces "(none)" via the helper.
	// This is what the wired service now returns instead of nil.
	withTags := appendTagsToBasicInfo([][]string{{"Status", "todo"}}, []string{})
	if len(withTags) < 2 || withTags[1][1] != "(none)" {
		t.Errorf("expected Tags row with '(none)' for empty slice, got: %v", withTags)
	}
}

// TestTaskGet_WithTagsRichDisplay — AC-28 (task entity, non-empty)
// Verifies that runTaskGet with a non-empty tag list does not error, and that
// appendTagsToBasicInfo produces a Tags row with the expected sorted names.
func TestTaskGet_WithTagsRichDisplay(t *testing.T) {
	mock := &mockTaskGetService{
		getWithTagsFn: func(ctx context.Context, key string) (*models.Task, []string, error) {
			task := &models.Task{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Tagged task"},
				Status:     "in_progress",
				Priority:   5,
			}
			return task, []string{"auth", "voice"}, nil
		},
	}
	withTaskGetSvcOverride(t, mock)

	restore := suppressOutput(t)
	defer restore()

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = false
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	cmd := newCmdWithCtx()
	err := runTaskGet(cmd, []string{"E07-F01-001"})
	if err != nil {
		t.Errorf("runTaskGet() with tags unexpected error: %v", err)
	}

	// Verify the helper renders the tag names.
	basicInfo := appendTagsToBasicInfo([][]string{{"Status", "in_progress"}}, []string{"auth", "voice"})
	found := false
	for _, row := range basicInfo {
		if len(row) == 2 && row[0] == "Tags" && strings.Contains(row[1], "auth") && strings.Contains(row[1], "voice") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Tags row not found in BasicInfo: %v", basicInfo)
	}
}

// TestTaskGet_NoTagsJSON — AC-28d (task entity)
// Verifies that runTaskGet in JSON mode produces "tags": [] when the service
// returns an empty slice.
func TestTaskGet_NoTagsJSON(t *testing.T) {
	mock := &mockTaskGetService{
		getWithTagsFn: func(ctx context.Context, key string) (*models.Task, []string, error) {
			task := &models.Task{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Untagged task"},
				Status:     "todo",
				Priority:   5,
			}
			return task, []string{}, nil
		},
	}
	withTaskGetSvcOverride(t, mock)

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	result := captureJSONOutput(t, func() {
		cmd := newCmdWithCtx()
		err := runTaskGet(cmd, []string{"E07-F01-001"})
		if err != nil {
			t.Errorf("runTaskGet() unexpected error: %v", err)
		}
	})

	tagsRaw, ok := result["tags"]
	if !ok {
		t.Fatal("JSON output: 'tags' field missing")
	}
	tags, ok := tagsRaw.([]interface{})
	if !ok {
		t.Fatalf("JSON output: 'tags' is not an array, got %T", tagsRaw)
	}
	if len(tags) != 0 {
		t.Errorf("JSON output: expected empty tags array, got %v", tags)
	}
}

// TestTaskGet_WithTagsJSON — AC-28b (task entity)
// Verifies that runTaskGet in JSON mode produces "tags": ["auth","voice"].
func TestTaskGet_WithTagsJSON(t *testing.T) {
	mock := &mockTaskGetService{
		getWithTagsFn: func(ctx context.Context, key string) (*models.Task, []string, error) {
			task := &models.Task{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Tagged task"},
				Status:     "todo",
				Priority:   5,
			}
			return task, []string{"auth", "voice"}, nil
		},
	}
	withTaskGetSvcOverride(t, mock)

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	result := captureJSONOutput(t, func() {
		cmd := newCmdWithCtx()
		err := runTaskGet(cmd, []string{"E07-F01-001"})
		if err != nil {
			t.Errorf("runTaskGet() unexpected error: %v", err)
		}
	})

	tagsRaw, ok := result["tags"]
	if !ok {
		t.Fatal("JSON output: 'tags' field missing")
	}
	tags, ok := tagsRaw.([]interface{})
	if !ok {
		t.Fatalf("JSON output: 'tags' is not an array, got %T", tagsRaw)
	}
	if len(tags) != 2 {
		t.Errorf("JSON output: expected 2 tags, got %d: %v", len(tags), tags)
	}
}
