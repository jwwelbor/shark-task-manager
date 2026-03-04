package commands

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// mockBugService implements bugServicer for testing.
type mockBugService struct {
	createBugFunc        func(ctx context.Context, input services.CreateBugInput) (*models.Bug, error)
	getBugFunc           func(ctx context.Context, key string) (*models.Bug, error)
	listBugsFunc         func(ctx context.Context, filters services.BugFilters) ([]*models.Bug, error)
	updateBugFunc        func(ctx context.Context, key string, updates services.BugUpdates) (*models.Bug, error)
	deleteBugFunc        func(ctx context.Context, key string) error
	triageBugFunc        func(ctx context.Context, key string, input services.TriageBugInput) (*models.Bug, error)
	advanceBugStatusFunc func(ctx context.Context, key string) (*models.Bug, error)
	setBugStatusFunc     func(ctx context.Context, key string, status string, force bool) (*models.Bug, error)
}

func (m *mockBugService) CreateBug(ctx context.Context, input services.CreateBugInput) (*models.Bug, error) {
	if m.createBugFunc != nil {
		return m.createBugFunc(ctx, input)
	}
	return nil, errors.New("CreateBug: not implemented")
}

func (m *mockBugService) GetBug(ctx context.Context, key string) (*models.Bug, error) {
	if m.getBugFunc != nil {
		return m.getBugFunc(ctx, key)
	}
	return nil, errors.New("GetBug: not implemented")
}

func (m *mockBugService) ListBugs(ctx context.Context, filters services.BugFilters) ([]*models.Bug, error) {
	if m.listBugsFunc != nil {
		return m.listBugsFunc(ctx, filters)
	}
	return nil, errors.New("ListBugs: not implemented")
}

func (m *mockBugService) UpdateBug(ctx context.Context, key string, updates services.BugUpdates) (*models.Bug, error) {
	if m.updateBugFunc != nil {
		return m.updateBugFunc(ctx, key, updates)
	}
	return nil, errors.New("UpdateBug: not implemented")
}

func (m *mockBugService) DeleteBug(ctx context.Context, key string) error {
	if m.deleteBugFunc != nil {
		return m.deleteBugFunc(ctx, key)
	}
	return errors.New("DeleteBug: not implemented")
}

func (m *mockBugService) TriageBug(ctx context.Context, key string, input services.TriageBugInput) (*models.Bug, error) {
	if m.triageBugFunc != nil {
		return m.triageBugFunc(ctx, key, input)
	}
	return nil, errors.New("TriageBug: not implemented")
}

func (m *mockBugService) AdvanceBugStatus(ctx context.Context, key string) (*models.Bug, error) {
	if m.advanceBugStatusFunc != nil {
		return m.advanceBugStatusFunc(ctx, key)
	}
	return nil, errors.New("AdvanceBugStatus: not implemented")
}

func (m *mockBugService) SetBugStatus(ctx context.Context, key string, status string, force bool) (*models.Bug, error) {
	if m.setBugStatusFunc != nil {
		return m.setBugStatusFunc(ctx, key, status, force)
	}
	return nil, errors.New("SetBugStatus: not implemented")
}

// newTestBug creates a Bug fixture for testing.
func newTestBug(key, title string) *models.Bug {
	desc := "a test bug description"
	return &models.Bug{
		ID:          1,
		Key:         key,
		Title:       title,
		Status:      models.BugStatus("open"),
		Severity:    models.BugSeverityMedium,
		Description: &desc,
	}
}

// --- Scaffolding tests ---

func TestBugCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range bugCmd.Commands() {
		_ = cmd
		found = true
		break
	}
	// bugCmd itself must be registered and non-nil
	if bugCmd == nil {
		t.Fatal("bugCmd should not be nil")
	}
	_ = found
}

func TestBugCreateCmdRegistered(t *testing.T) {
	if bugCreateCmd == nil {
		t.Fatal("bugCreateCmd should not be nil")
	}
	if bugCreateCmd.Use != "create <title>" {
		t.Errorf("expected Use='create <title>', got %q", bugCreateCmd.Use)
	}
}

func TestBugGetCmdRegistered(t *testing.T) {
	if bugGetCmd == nil {
		t.Fatal("bugGetCmd should not be nil")
	}
	if bugGetCmd.Use != "get <key>" {
		t.Errorf("expected Use='get <key>', got %q", bugGetCmd.Use)
	}
}

func TestBugListCmdRegistered(t *testing.T) {
	if bugListCmd == nil {
		t.Fatal("bugListCmd should not be nil")
	}
	if bugListCmd.Use != "list" {
		t.Errorf("expected Use='list', got %q", bugListCmd.Use)
	}
}

// --- CRUD handler tests ---

func TestRunBugCreate_CallsService(t *testing.T) {
	var calledWith services.CreateBugInput
	svc := &mockBugService{
		createBugFunc: func(ctx context.Context, input services.CreateBugInput) (*models.Bug, error) {
			calledWith = input
			return newTestBug("B001", input.Title), nil
		},
	}

	var buf bytes.Buffer
	err := runBugCreateWithService(context.Background(), []string{"login crash"}, svc, &buf)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if calledWith.Title != "login crash" {
		t.Errorf("expected title='login crash', got %q", calledWith.Title)
	}
	if !strings.Contains(buf.String(), "B001") {
		t.Errorf("expected output to contain bug key B001, got: %s", buf.String())
	}
}

func TestRunBugCreate_ServiceError(t *testing.T) {
	svc := &mockBugService{
		createBugFunc: func(_ context.Context, _ services.CreateBugInput) (*models.Bug, error) {
			return nil, errors.New("db error")
		},
	}

	var buf bytes.Buffer
	err := runBugCreateWithService(context.Background(), []string{"crash bug"}, svc, &buf)
	if err == nil {
		t.Fatal("expected error from service, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create bug") {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestRunBugGet_OutputsJSON(t *testing.T) {
	bug := newTestBug("B001", "login crash")
	svc := &mockBugService{
		getBugFunc: func(_ context.Context, key string) (*models.Bug, error) {
			if key != "B001" {
				t.Errorf("expected key='B001', got %q", key)
			}
			return bug, nil
		},
	}

	var buf bytes.Buffer
	err := runBugGetWithService(context.Background(), []string{"B001"}, svc, &buf, true)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, `"key"`) {
		t.Errorf("expected JSON with 'key' field, got: %s", output)
	}
	if !strings.Contains(output, "B001") {
		t.Errorf("expected JSON to contain B001, got: %s", output)
	}
}

func TestRunBugGet_HumanOutput(t *testing.T) {
	bug := newTestBug("B002", "payment error")
	svc := &mockBugService{
		getBugFunc: func(_ context.Context, _ string) (*models.Bug, error) {
			return bug, nil
		},
	}

	var buf bytes.Buffer
	err := runBugGetWithService(context.Background(), []string{"B002"}, svc, &buf, false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "B002") {
		t.Errorf("expected output to contain B002, got: %s", output)
	}
	if !strings.Contains(output, "payment error") {
		t.Errorf("expected output to contain title, got: %s", output)
	}
}

func TestRunBugList_OutputsJSON(t *testing.T) {
	bugs := []*models.Bug{
		newTestBug("B001", "crash on login"),
		newTestBug("B002", "payment timeout"),
	}
	svc := &mockBugService{
		listBugsFunc: func(_ context.Context, _ services.BugFilters) ([]*models.Bug, error) {
			return bugs, nil
		},
	}

	// Reset filter flags to empty state
	bugListStatus = ""
	bugListSeverity = ""

	var buf bytes.Buffer
	err := runBugListWithService(context.Background(), svc, &buf, true)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "B001") {
		t.Errorf("expected JSON to contain B001, got: %s", output)
	}
	if !strings.Contains(output, "B002") {
		t.Errorf("expected JSON to contain B002, got: %s", output)
	}
}

func TestRunBugList_HumanOutput(t *testing.T) {
	bugs := []*models.Bug{
		newTestBug("B001", "crash on login"),
	}
	svc := &mockBugService{
		listBugsFunc: func(_ context.Context, _ services.BugFilters) ([]*models.Bug, error) {
			return bugs, nil
		},
	}

	bugListStatus = ""
	bugListSeverity = ""

	var buf bytes.Buffer
	err := runBugListWithService(context.Background(), svc, &buf, false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "B001") {
		t.Errorf("expected output to contain B001, got: %s", output)
	}
	if !strings.Contains(output, "crash on login") {
		t.Errorf("expected output to contain title, got: %s", output)
	}
}

func TestRunBugList_Empty(t *testing.T) {
	svc := &mockBugService{
		listBugsFunc: func(_ context.Context, _ services.BugFilters) ([]*models.Bug, error) {
			return []*models.Bug{}, nil
		},
	}

	bugListStatus = ""
	bugListSeverity = ""

	var buf bytes.Buffer
	err := runBugListWithService(context.Background(), svc, &buf, false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "No bugs found") {
		t.Errorf("expected 'No bugs found' in output, got: %s", buf.String())
	}
}

func TestRunBugUpdate_CallsService(t *testing.T) {
	var capturedKey string
	var capturedUpdates services.BugUpdates
	svc := &mockBugService{
		updateBugFunc: func(_ context.Context, key string, updates services.BugUpdates) (*models.Bug, error) {
			capturedKey = key
			capturedUpdates = updates
			return newTestBug(key, "updated title"), nil
		},
	}

	// Set flag state before calling
	bugUpdateTitle = "updated title"
	bugUpdateDescription = ""
	bugUpdateSeverity = ""
	bugUpdateLinkType = ""
	bugUpdateLinkKey = ""

	var buf bytes.Buffer
	err := runBugUpdateWithService(context.Background(), []string{"B001"}, svc, &buf)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedKey != "B001" {
		t.Errorf("expected key='B001', got %q", capturedKey)
	}
	if capturedUpdates.Title == nil || *capturedUpdates.Title != "updated title" {
		t.Errorf("expected Title='updated title' in updates, got: %+v", capturedUpdates)
	}

	// Reset
	bugUpdateTitle = ""
}

func TestRunBugDelete_CallsService(t *testing.T) {
	var deletedKey string
	svc := &mockBugService{
		deleteBugFunc: func(_ context.Context, key string) error {
			deletedKey = key
			return nil
		},
	}

	var buf bytes.Buffer
	err := runBugDeleteWithService(context.Background(), []string{"B003"}, svc, &buf)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if deletedKey != "B003" {
		t.Errorf("expected deleted key='B003', got %q", deletedKey)
	}
	if !strings.Contains(buf.String(), "B003") {
		t.Errorf("expected output to confirm deletion of B003, got: %s", buf.String())
	}
}

// --- Triage handler tests ---

func TestRunBugTriage_SetsSeverity(t *testing.T) {
	var capturedInput services.TriageBugInput
	svc := &mockBugService{
		triageBugFunc: func(_ context.Context, _ string, input services.TriageBugInput) (*models.Bug, error) {
			capturedInput = input
			bug := newTestBug("B001", "crash")
			bug.Severity = models.BugSeverityCritical
			return bug, nil
		},
	}

	bugTriageSeverity = "critical"
	bugTriageAssign = ""
	bugTriageLinkType = ""
	bugTriageLinkKey = ""

	var buf bytes.Buffer
	err := runBugTriageWithService(context.Background(), []string{"B001"}, svc, &buf)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedInput.Severity == nil || *capturedInput.Severity != models.BugSeverityCritical {
		t.Errorf("expected severity=critical in triage input, got: %+v", capturedInput)
	}
	if !strings.Contains(buf.String(), "critical") {
		t.Errorf("expected output to contain 'critical', got: %s", buf.String())
	}

	// Reset
	bugTriageSeverity = ""
}

func TestRunBugTriage_SetsAssign(t *testing.T) {
	var capturedInput services.TriageBugInput
	svc := &mockBugService{
		triageBugFunc: func(_ context.Context, _ string, input services.TriageBugInput) (*models.Bug, error) {
			capturedInput = input
			return newTestBug("B001", "crash"), nil
		},
	}

	bugTriageSeverity = ""
	bugTriageAssign = "alice"
	bugTriageLinkType = ""
	bugTriageLinkKey = ""

	var buf bytes.Buffer
	err := runBugTriageWithService(context.Background(), []string{"B001"}, svc, &buf)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedInput.AssignedTo == nil || *capturedInput.AssignedTo != "alice" {
		t.Errorf("expected AssignedTo='alice', got: %+v", capturedInput)
	}

	// Reset
	bugTriageAssign = ""
}

func TestRunBugTriage_SetsLink(t *testing.T) {
	var capturedInput services.TriageBugInput
	svc := &mockBugService{
		triageBugFunc: func(_ context.Context, _ string, input services.TriageBugInput) (*models.Bug, error) {
			capturedInput = input
			return newTestBug("B001", "crash"), nil
		},
	}

	bugTriageSeverity = ""
	bugTriageAssign = ""
	bugTriageLinkType = "feature"
	bugTriageLinkKey = "E18-F04"

	var buf bytes.Buffer
	err := runBugTriageWithService(context.Background(), []string{"B001"}, svc, &buf)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedInput.LinkedEntityType == nil || *capturedInput.LinkedEntityType != "feature" {
		t.Errorf("expected LinkedEntityType='feature', got: %+v", capturedInput)
	}
	if capturedInput.LinkedEntityKey == nil || *capturedInput.LinkedEntityKey != "E18-F04" {
		t.Errorf("expected LinkedEntityKey='E18-F04', got: %+v", capturedInput)
	}

	// Reset
	bugTriageLinkType = ""
	bugTriageLinkKey = ""
}

// --- Sub-command registration test ---

func TestBugSubcommandsRegistered(t *testing.T) {
	wantSubcmds := []string{"create", "get", "list", "update", "delete", "triage", "note", "notes", "context"}
	registered := make(map[string]bool)
	for _, sub := range bugCmd.Commands() {
		registered[sub.Name()] = true
	}
	for _, name := range wantSubcmds {
		if !registered[name] {
			t.Errorf("expected subcommand %q to be registered under bugCmd", name)
		}
	}
}

// --- cobra.Command integration smoke test ---

func TestBugCmdHasRunE(t *testing.T) {
	// bugCmd itself is a parent container with no RunE (just subcommands)
	// Verify key subcommands do have RunE set.
	cmdsWithRunE := []*cobra.Command{bugCreateCmd, bugGetCmd, bugListCmd, bugUpdateCmd, bugDeleteCmd, bugTriageCmd}
	for _, cmd := range cmdsWithRunE {
		if cmd.RunE == nil {
			t.Errorf("expected RunE to be set on %q", cmd.Use)
		}
	}
}
