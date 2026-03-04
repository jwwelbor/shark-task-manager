package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// MockChangeCardService — function-field pattern (no real DB)
// ---------------------------------------------------------------------------

// MockChangeCardService implements changeCardServicer for testing.
// Each method delegates to the corresponding Func field if set, otherwise
// returns a sensible default so callers don't panic on unimplemented methods.
type MockChangeCardService struct {
	CreateChangeCardFunc  func(ctx context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, error)
	GetChangeCardFunc     func(ctx context.Context, key string) (*models.ChangeCard, error)
	ListChangeCardsFunc   func(ctx context.Context, filters services.ChangeCardFilters) ([]*models.ChangeCard, error)
	UpdateChangeCardFunc  func(ctx context.Context, key string, updates services.ChangeCardUpdates) (*models.ChangeCard, error)
	DeleteChangeCardFunc  func(ctx context.Context, key string) error
	ApproveChangeCardFunc func(ctx context.Context, key string) (*models.ChangeCard, error)
}

func (m *MockChangeCardService) CreateChangeCard(ctx context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, error) {
	if m.CreateChangeCardFunc != nil {
		return m.CreateChangeCardFunc(ctx, input)
	}
	return nil, fmt.Errorf("CreateChangeCard not implemented in mock")
}

func (m *MockChangeCardService) GetChangeCard(ctx context.Context, key string) (*models.ChangeCard, error) {
	if m.GetChangeCardFunc != nil {
		return m.GetChangeCardFunc(ctx, key)
	}
	return nil, fmt.Errorf("GetChangeCard not implemented in mock")
}

func (m *MockChangeCardService) ListChangeCards(ctx context.Context, filters services.ChangeCardFilters) ([]*models.ChangeCard, error) {
	if m.ListChangeCardsFunc != nil {
		return m.ListChangeCardsFunc(ctx, filters)
	}
	return nil, fmt.Errorf("ListChangeCards not implemented in mock")
}

func (m *MockChangeCardService) UpdateChangeCard(ctx context.Context, key string, updates services.ChangeCardUpdates) (*models.ChangeCard, error) {
	if m.UpdateChangeCardFunc != nil {
		return m.UpdateChangeCardFunc(ctx, key, updates)
	}
	return nil, fmt.Errorf("UpdateChangeCard not implemented in mock")
}

func (m *MockChangeCardService) DeleteChangeCard(ctx context.Context, key string) error {
	if m.DeleteChangeCardFunc != nil {
		return m.DeleteChangeCardFunc(ctx, key)
	}
	return fmt.Errorf("DeleteChangeCard not implemented in mock")
}

func (m *MockChangeCardService) ApproveChangeCard(ctx context.Context, key string) (*models.ChangeCard, error) {
	if m.ApproveChangeCardFunc != nil {
		return m.ApproveChangeCardFunc(ctx, key)
	}
	return nil, fmt.Errorf("ApproveChangeCard not implemented in mock")
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// injectMockChangeCardSvc sets the package-level override and returns a
// cleanup function that restores the original nil value.
func injectMockChangeCardSvc(t *testing.T, mock changeCardServicer) func() {
	t.Helper()
	changeCardSvcOverride = mock
	return func() { changeCardSvcOverride = nil }
}

// newChangeCmd returns a minimal cobra.Command whose Context() is non-nil.
func newChangeCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	return cmd
}

// suppressOutput redirects stdout/stderr for the duration of the test and
// discards everything written.  Call the returned restore func in defer.
func suppressOutput(t *testing.T) func() {
	t.Helper()
	origStdout := os.Stdout
	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w
	return func() {
		_ = w.Close()
		_, _ = io.Copy(io.Discard, r)
		os.Stdout = origStdout
		os.Stderr = origStderr
	}
}

// captureOutput captures stdout during fn() and returns the captured bytes.
func captureOutput(t *testing.T, fn func()) []byte {
	t.Helper()
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stdout = origStdout
	return buf.Bytes()
}

// ---------------------------------------------------------------------------
// runChangeCreate tests
// ---------------------------------------------------------------------------

func TestRunChangeCreate_Success(t *testing.T) {
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		CreateChangeCardFunc: func(ctx context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, error) {
			return &models.ChangeCard{Key: "CC-001", Title: input.Title, Status: "proposed"}, nil
		},
	})
	defer restore()

	// Reset globals
	changeLinkKey = ""
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""

	restore2 := suppressOutput(t)
	defer restore2()

	cmd := newChangeCmd()
	err := runChangeCreate(cmd, []string{"Add dark mode"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunChangeCreate_ServiceError(t *testing.T) {
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		CreateChangeCardFunc: func(ctx context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, error) {
			return nil, fmt.Errorf("db error")
		},
	})
	defer restore()

	changeLinkKey = ""
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""

	cmd := newChangeCmd()
	err := runChangeCreate(cmd, []string{"Add dark mode"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunChangeCreate_JSONOutput(t *testing.T) {
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		CreateChangeCardFunc: func(ctx context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, error) {
			return &models.ChangeCard{Key: "CC-001", Title: "Test", Status: "proposed"}, nil
		},
	})
	defer restore()

	changeLinkKey = ""
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""

	out := captureOutput(t, func() {
		cmd := newChangeCmd()
		_ = runChangeCreate(cmd, []string{"Test"})
	})

	if !bytes.Contains(out, []byte("CC-001")) {
		t.Errorf("expected JSON output to contain CC-001, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// runChangeGet tests
// ---------------------------------------------------------------------------

func TestRunChangeGet_Success(t *testing.T) {
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		GetChangeCardFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return &models.ChangeCard{Key: key, Title: "Some change", Status: "proposed"}, nil
		},
	})
	defer restore()

	restore2 := suppressOutput(t)
	defer restore2()

	cmd := newChangeCmd()
	err := runChangeGet(cmd, []string{"CC-001"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunChangeGet_NotFound(t *testing.T) {
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		GetChangeCardFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return nil, fmt.Errorf("change-card not found: %s", key)
		},
	})
	defer restore()

	cmd := newChangeCmd()
	err := runChangeGet(cmd, []string{"CC-999"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunChangeGet_JSONOutput(t *testing.T) {
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		GetChangeCardFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return &models.ChangeCard{Key: "CC-042", Title: "JSON change", Status: "approved"}, nil
		},
	})
	defer restore()

	out := captureOutput(t, func() {
		cmd := newChangeCmd()
		_ = runChangeGet(cmd, []string{"CC-042"})
	})

	if !bytes.Contains(out, []byte("CC-042")) {
		t.Errorf("expected JSON output to contain CC-042, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// runChangeList tests
// ---------------------------------------------------------------------------

func TestRunChangeList_Empty(t *testing.T) {
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		ListChangeCardsFunc: func(ctx context.Context, filters services.ChangeCardFilters) ([]*models.ChangeCard, error) {
			return []*models.ChangeCard{}, nil
		},
	})
	defer restore()

	changeStatusFilter = ""
	changeLinkFilter = ""

	restore2 := suppressOutput(t)
	defer restore2()

	cmd := newChangeCmd()
	err := runChangeList(cmd, []string{})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunChangeList_WithResults(t *testing.T) {
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		ListChangeCardsFunc: func(ctx context.Context, filters services.ChangeCardFilters) ([]*models.ChangeCard, error) {
			return []*models.ChangeCard{
				{Key: "CC-001", Title: "First change", Status: "proposed"},
				{Key: "CC-002", Title: "Second change", Status: "approved"},
			}, nil
		},
	})
	defer restore()

	changeStatusFilter = ""
	changeLinkFilter = ""

	restore2 := suppressOutput(t)
	defer restore2()

	cmd := newChangeCmd()
	err := runChangeList(cmd, []string{})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunChangeList_FilterRouting_EpicKey(t *testing.T) {
	var capturedFilters services.ChangeCardFilters
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		ListChangeCardsFunc: func(ctx context.Context, filters services.ChangeCardFilters) ([]*models.ChangeCard, error) {
			capturedFilters = filters
			return []*models.ChangeCard{}, nil
		},
	})
	defer restore()

	changeLinkFilter = "E07"
	changeStatusFilter = ""
	defer func() { changeLinkFilter = "" }()

	restore2 := suppressOutput(t)
	defer restore2()

	cmd := newChangeCmd()
	_ = runChangeList(cmd, []string{})

	if capturedFilters.EpicKey != "E07" {
		t.Errorf("expected EpicKey=E07, got %q", capturedFilters.EpicKey)
	}
	if capturedFilters.FeatureKey != "" {
		t.Errorf("expected empty FeatureKey, got %q", capturedFilters.FeatureKey)
	}
}

func TestRunChangeList_FilterRouting_FeatureKey(t *testing.T) {
	var capturedFilters services.ChangeCardFilters
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		ListChangeCardsFunc: func(ctx context.Context, filters services.ChangeCardFilters) ([]*models.ChangeCard, error) {
			capturedFilters = filters
			return []*models.ChangeCard{}, nil
		},
	})
	defer restore()

	changeLinkFilter = "E07-F03"
	changeStatusFilter = ""
	defer func() { changeLinkFilter = "" }()

	restore2 := suppressOutput(t)
	defer restore2()

	cmd := newChangeCmd()
	_ = runChangeList(cmd, []string{})

	if capturedFilters.FeatureKey != "E07-F03" {
		t.Errorf("expected FeatureKey=E07-F03, got %q", capturedFilters.FeatureKey)
	}
	if capturedFilters.EpicKey != "" {
		t.Errorf("expected empty EpicKey, got %q", capturedFilters.EpicKey)
	}
}

func TestRunChangeList_ServiceError(t *testing.T) {
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		ListChangeCardsFunc: func(ctx context.Context, filters services.ChangeCardFilters) ([]*models.ChangeCard, error) {
			return nil, fmt.Errorf("database error")
		},
	})
	defer restore()

	changeStatusFilter = ""
	changeLinkFilter = ""

	cmd := newChangeCmd()
	err := runChangeList(cmd, []string{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunChangeList_JSONOutput(t *testing.T) {
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		ListChangeCardsFunc: func(ctx context.Context, filters services.ChangeCardFilters) ([]*models.ChangeCard, error) {
			return []*models.ChangeCard{
				{Key: "CC-001", Title: "List item", Status: "proposed"},
			}, nil
		},
	})
	defer restore()

	changeStatusFilter = ""
	changeLinkFilter = ""

	out := captureOutput(t, func() {
		cmd := newChangeCmd()
		_ = runChangeList(cmd, []string{})
	})

	if !bytes.Contains(out, []byte("CC-001")) {
		t.Errorf("expected JSON output to contain CC-001, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// runChangeUpdate tests
// ---------------------------------------------------------------------------

func TestRunChangeUpdate_Success(t *testing.T) {
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		UpdateChangeCardFunc: func(ctx context.Context, key string, updates services.ChangeCardUpdates) (*models.ChangeCard, error) {
			return &models.ChangeCard{Key: key, Title: "Updated title", Status: "proposed"}, nil
		},
	})
	defer restore()

	changeTitle = ""
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""
	changeAssignedTo = ""

	restore2 := suppressOutput(t)
	defer restore2()

	cmd := newTestChangeUpdateCmd()
	err := runChangeUpdate(cmd, []string{"CC-001"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunChangeUpdate_ServiceError(t *testing.T) {
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		UpdateChangeCardFunc: func(ctx context.Context, key string, updates services.ChangeCardUpdates) (*models.ChangeCard, error) {
			return nil, fmt.Errorf("change-card not found: %s", key)
		},
	})
	defer restore()

	changeTitle = ""
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""
	changeAssignedTo = ""

	cmd := newTestChangeUpdateCmd()
	err := runChangeUpdate(cmd, []string{"CC-999"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunChangeUpdate_JSONOutput(t *testing.T) {
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	newTitle := "Updated title"
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		UpdateChangeCardFunc: func(ctx context.Context, key string, updates services.ChangeCardUpdates) (*models.ChangeCard, error) {
			return &models.ChangeCard{Key: "CC-005", Title: newTitle, Status: "proposed"}, nil
		},
	})
	defer restore()

	changeTitle = newTitle
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""
	changeAssignedTo = ""
	defer func() { changeTitle = "" }()

	out := captureOutput(t, func() {
		cmd := newTestChangeUpdateCmd()
		_ = cmd.Flags().Set("title", newTitle)
		_ = runChangeUpdate(cmd, []string{"CC-005"})
	})

	if !bytes.Contains(out, []byte("CC-005")) {
		t.Errorf("expected JSON output to contain CC-005, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// runChangeDelete tests
// ---------------------------------------------------------------------------

func TestRunChangeDelete_Force_Success(t *testing.T) {
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		DeleteChangeCardFunc: func(ctx context.Context, key string) error {
			return nil
		},
	})
	defer restore()

	changeForce = true
	defer func() { changeForce = false }()

	restore2 := suppressOutput(t)
	defer restore2()

	cmd := newChangeCmd()
	err := runChangeDelete(cmd, []string{"CC-001"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunChangeDelete_Force_NotFound(t *testing.T) {
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		DeleteChangeCardFunc: func(ctx context.Context, key string) error {
			return fmt.Errorf("change-card not found: %s", key)
		},
	})
	defer restore()

	changeForce = true
	defer func() { changeForce = false }()

	cmd := newChangeCmd()
	err := runChangeDelete(cmd, []string{"CC-999"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunChangeDelete_Force_JSONOutput(t *testing.T) {
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		DeleteChangeCardFunc: func(ctx context.Context, key string) error {
			return nil
		},
	})
	defer restore()

	changeForce = true
	defer func() { changeForce = false }()

	out := captureOutput(t, func() {
		cmd := newChangeCmd()
		_ = runChangeDelete(cmd, []string{"CC-003"})
	})

	if !bytes.Contains(out, []byte("CC-003")) {
		t.Errorf("expected JSON output to contain CC-003, got: %s", out)
	}
}

func TestRunChangeDelete_NoForce_NotFound(t *testing.T) {
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		GetChangeCardFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return nil, fmt.Errorf("change-card not found: %s", key)
		},
	})
	defer restore()

	changeForce = false

	cmd := newChangeCmd()
	err := runChangeDelete(cmd, []string{"CC-999"})
	if err == nil {
		t.Fatal("expected error for not-found get, got nil")
	}
}

// ---------------------------------------------------------------------------
// runChangeApprove tests — success path only (error paths call os.Exit)
// ---------------------------------------------------------------------------

func TestRunChangeApprove_Success(t *testing.T) {
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		ApproveChangeCardFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return &models.ChangeCard{Key: key, Title: "Approved change", Status: "approved"}, nil
		},
	})
	defer restore()

	restore2 := suppressOutput(t)
	defer restore2()

	cmd := newChangeCmd()
	err := runChangeApprove(cmd, []string{"CC-001"})
	if err != nil {
		t.Errorf("expected no error on approve success, got %v", err)
	}
}

func TestRunChangeApprove_JSONOutput(t *testing.T) {
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		ApproveChangeCardFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return &models.ChangeCard{Key: "CC-007", Title: "Approve me", Status: "approved"}, nil
		},
	})
	defer restore()

	out := captureOutput(t, func() {
		cmd := newChangeCmd()
		_ = runChangeApprove(cmd, []string{"CC-007"})
	})

	if !bytes.Contains(out, []byte("CC-007")) {
		t.Errorf("expected JSON output to contain CC-007, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// printChangeCardList tests
// ---------------------------------------------------------------------------

// TestPrintChangeCardList_LinkedEntityHeader verifies that printChangeCardList
// returns no error for a non-empty list (the "Linked Entity" header is verified
// by reading the hardcoded headers slice in the implementation).
func TestPrintChangeCardList_LinkedEntityHeader(t *testing.T) {
	cards := []*models.ChangeCard{
		{Key: "CC-001", Title: "A change", Status: "proposed"},
	}

	restore := suppressOutput(t)
	defer restore()

	err := printChangeCardList(cards)
	if err != nil {
		t.Errorf("printChangeCardList returned unexpected error: %v", err)
	}
}

// TestPrintChangeCardList_NonEmpty verifies that a non-empty list is rendered
// without errors.
func TestPrintChangeCardList_NonEmpty(t *testing.T) {
	cards := []*models.ChangeCard{
		{Key: "CC-001", Title: "First", Status: "proposed"},
		{Key: "CC-002", Title: "Second", Status: "approved"},
	}

	restore := suppressOutput(t)
	defer restore()

	err := printChangeCardList(cards)
	if err != nil {
		t.Errorf("printChangeCardList returned unexpected error: %v", err)
	}
}

// newTestChangeUpdateCmd builds a cobra.Command with the flags used by
// buildChangeCardUpdates.
func newTestChangeUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "update"}
	cmd.Flags().StringVar(&changeTitle, "title", "", "Title")
	cmd.Flags().StringVar(&changeDescription, "description", "", "Description")
	cmd.Flags().IntVar(&changePriority, "priority", 0, "Priority")
	cmd.Flags().StringVar(&changeRequestedBy, "requested-by", "", "Requested by")
	cmd.Flags().StringVar(&changeAssignedTo, "assigned-to", "", "Assigned to")
	return cmd
}

// TestBuildCreateChangeCardInput_Title verifies that the title is passed through.
func TestBuildCreateChangeCardInput_Title(t *testing.T) {
	// Reset global flag variables before use
	changeLinkKey = ""
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""

	input := buildCreateChangeCardInput("Add dark mode toggle")

	if input.Title != "Add dark mode toggle" {
		t.Errorf("expected title %q, got %q", "Add dark mode toggle", input.Title)
	}
}

// TestBuildCreateChangeCardInput_LinkEpic verifies that a bare epic key (e.g. E07)
// is placed in EpicKey and not FeatureKey.
func TestBuildCreateChangeCardInput_LinkEpic(t *testing.T) {
	changeLinkKey = "E07"
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""
	defer func() { changeLinkKey = "" }()

	input := buildCreateChangeCardInput("Some change")

	if input.EpicKey != "E07" {
		t.Errorf("expected EpicKey %q, got %q", "E07", input.EpicKey)
	}
	if input.FeatureKey != "" {
		t.Errorf("expected empty FeatureKey, got %q", input.FeatureKey)
	}
}

// TestBuildCreateChangeCardInput_LinkFeature verifies that a key containing "-F"
// (e.g. E07-F03) is placed in FeatureKey and not EpicKey.
func TestBuildCreateChangeCardInput_LinkFeature(t *testing.T) {
	changeLinkKey = "E07-F03"
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""
	defer func() { changeLinkKey = "" }()

	input := buildCreateChangeCardInput("Some change")

	if input.FeatureKey != "E07-F03" {
		t.Errorf("expected FeatureKey %q, got %q", "E07-F03", input.FeatureKey)
	}
	if input.EpicKey != "" {
		t.Errorf("expected empty EpicKey, got %q", input.EpicKey)
	}
}

// TestBuildCreateChangeCardInput_Priority verifies that a non-zero priority is set.
func TestBuildCreateChangeCardInput_Priority(t *testing.T) {
	changeLinkKey = ""
	changeDescription = ""
	changePriority = 8
	changeRequestedBy = ""
	defer func() { changePriority = 0 }()

	input := buildCreateChangeCardInput("Priority change")

	if input.Priority != 8 {
		t.Errorf("expected Priority 8, got %d", input.Priority)
	}
}

// TestBuildCreateChangeCardInput_ZeroPriorityOmitted verifies that a zero priority
// is not forwarded (it remains 0 in the input, which is fine as the service handles
// the default).
func TestBuildCreateChangeCardInput_ZeroPriorityOmitted(t *testing.T) {
	changeLinkKey = ""
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""

	input := buildCreateChangeCardInput("Some change")

	if input.Priority != 0 {
		t.Errorf("expected Priority 0 when flag not set, got %d", input.Priority)
	}
}

// TestBuildCreateChangeCardInput_AllFields verifies all optional fields together.
func TestBuildCreateChangeCardInput_AllFields(t *testing.T) {
	changeLinkKey = "E10"
	changeDescription = "Detailed description"
	changePriority = 5
	changeRequestedBy = "alice"
	defer func() {
		changeLinkKey = ""
		changeDescription = ""
		changePriority = 0
		changeRequestedBy = ""
	}()

	input := buildCreateChangeCardInput("Full change")

	want := services.CreateChangeCardInput{
		Title:       "Full change",
		Description: "Detailed description",
		EpicKey:     "E10",
		Priority:    5,
		RequestedBy: "alice",
	}

	if input.Title != want.Title {
		t.Errorf("Title: expected %q, got %q", want.Title, input.Title)
	}
	if input.Description != want.Description {
		t.Errorf("Description: expected %q, got %q", want.Description, input.Description)
	}
	if input.EpicKey != want.EpicKey {
		t.Errorf("EpicKey: expected %q, got %q", want.EpicKey, input.EpicKey)
	}
	if input.Priority != want.Priority {
		t.Errorf("Priority: expected %d, got %d", want.Priority, input.Priority)
	}
	if input.RequestedBy != want.RequestedBy {
		t.Errorf("RequestedBy: expected %q, got %q", want.RequestedBy, input.RequestedBy)
	}
}

// TestBuildChangeCardUpdates_NoFlags verifies that when no flags are changed,
// all update fields remain nil.
func TestBuildChangeCardUpdates_NoFlags(t *testing.T) {
	changeTitle = ""
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""
	changeAssignedTo = ""

	cmd := newTestChangeUpdateCmd()
	updates := buildChangeCardUpdates(cmd)

	if updates.Title != nil {
		t.Errorf("expected nil Title, got %v", updates.Title)
	}
	if updates.Description != nil {
		t.Errorf("expected nil Description, got %v", updates.Description)
	}
	if updates.Priority != nil {
		t.Errorf("expected nil Priority, got %v", updates.Priority)
	}
	if updates.RequestedBy != nil {
		t.Errorf("expected nil RequestedBy, got %v", updates.RequestedBy)
	}
	if updates.AssignedTo != nil {
		t.Errorf("expected nil AssignedTo, got %v", updates.AssignedTo)
	}
}

// TestBuildChangeCardUpdates_TitleChanged verifies that a changed title flag
// results in a non-nil Title pointer.
func TestBuildChangeCardUpdates_TitleChanged(t *testing.T) {
	changeTitle = "New title"
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""
	changeAssignedTo = ""
	defer func() { changeTitle = "" }()

	cmd := newTestChangeUpdateCmd()
	// Simulate the user passing --title on the CLI
	_ = cmd.Flags().Set("title", "New title")

	updates := buildChangeCardUpdates(cmd)

	if updates.Title == nil {
		t.Fatal("expected non-nil Title")
	}
	if *updates.Title != "New title" {
		t.Errorf("expected Title %q, got %q", "New title", *updates.Title)
	}
	// Other fields should remain nil
	if updates.Description != nil {
		t.Errorf("expected nil Description, got %v", updates.Description)
	}
}

// TestBuildChangeCardUpdates_MultipleFlags verifies several changed flags at once.
func TestBuildChangeCardUpdates_MultipleFlags(t *testing.T) {
	changeTitle = "Updated title"
	changeDescription = "Updated description"
	changePriority = 7
	changeRequestedBy = "bob"
	changeAssignedTo = ""
	defer func() {
		changeTitle = ""
		changeDescription = ""
		changePriority = 0
		changeRequestedBy = ""
	}()

	cmd := newTestChangeUpdateCmd()
	_ = cmd.Flags().Set("title", "Updated title")
	_ = cmd.Flags().Set("description", "Updated description")
	_ = cmd.Flags().Set("priority", "7")
	_ = cmd.Flags().Set("requested-by", "bob")

	updates := buildChangeCardUpdates(cmd)

	if updates.Title == nil || *updates.Title != "Updated title" {
		t.Errorf("Title: expected %q, got %v", "Updated title", updates.Title)
	}
	if updates.Description == nil || *updates.Description != "Updated description" {
		t.Errorf("Description: expected %q, got %v", "Updated description", updates.Description)
	}
	if updates.Priority == nil || *updates.Priority != 7 {
		t.Errorf("Priority: expected 7, got %v", updates.Priority)
	}
	if updates.RequestedBy == nil || *updates.RequestedBy != "bob" {
		t.Errorf("RequestedBy: expected %q, got %v", "bob", updates.RequestedBy)
	}
	if updates.AssignedTo != nil {
		t.Errorf("AssignedTo: expected nil, got %v", updates.AssignedTo)
	}
}

// TestBuildChangeCardUpdates_AssignedTo verifies that the assigned-to flag works.
func TestBuildChangeCardUpdates_AssignedTo(t *testing.T) {
	changeTitle = ""
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""
	changeAssignedTo = "carol"
	defer func() { changeAssignedTo = "" }()

	cmd := newTestChangeUpdateCmd()
	_ = cmd.Flags().Set("assigned-to", "carol")

	updates := buildChangeCardUpdates(cmd)

	if updates.AssignedTo == nil || *updates.AssignedTo != "carol" {
		t.Errorf("AssignedTo: expected %q, got %v", "carol", updates.AssignedTo)
	}
	// Other fields should remain nil
	if updates.Title != nil {
		t.Errorf("expected nil Title, got %v", updates.Title)
	}
}

// TestPrintChangeCardList_Empty verifies that an empty slice prints a message
// and does not panic.
func TestPrintChangeCardList_Empty(t *testing.T) {
	// Should not panic and should return nil
	err := printChangeCardList(nil)
	if err != nil {
		t.Errorf("expected nil error for empty list, got %v", err)
	}
}
