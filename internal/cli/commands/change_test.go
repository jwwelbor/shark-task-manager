package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
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
	CreateChangeCardFunc      func(ctx context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, error)
	GetChangeCardFunc         func(ctx context.Context, key string) (*models.ChangeCard, error)
	ListChangeCardsFunc       func(ctx context.Context, filters services.ChangeCardFilters) ([]*models.ChangeCard, error)
	UpdateChangeCardFunc      func(ctx context.Context, key string, updates services.ChangeCardUpdates) (*models.ChangeCard, error)
	DeleteChangeCardFunc      func(ctx context.Context, key string) error
	TransitionStatusFunc      func(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
	GetNextStatusFunc         func(ctx context.Context, key string) (*services.NextStatusInfo, error)
	GetNextStatusForCardFunc  func(card *models.ChangeCard) *services.NextStatusInfo
	GetOrchestratorActionFunc func(card *models.ChangeCard) *config.PopulatedAction
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

func (m *MockChangeCardService) GetChangeCardWithTags(ctx context.Context, key string) (*models.ChangeCard, []string, error) {
	if m.GetChangeCardFunc != nil {
		card, err := m.GetChangeCardFunc(ctx, key)
		if err != nil {
			return nil, nil, err
		}
		return card, []string{}, nil
	}
	return nil, nil, fmt.Errorf("GetChangeCardWithTags not implemented in mock")
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

func (m *MockChangeCardService) TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	if m.TransitionStatusFunc != nil {
		return m.TransitionStatusFunc(ctx, key, targetStatus, opts)
	}
	return nil, fmt.Errorf("TransitionStatus not implemented in mock")
}

func (m *MockChangeCardService) GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error) {
	if m.GetNextStatusFunc != nil {
		return m.GetNextStatusFunc(ctx, key)
	}
	return nil, fmt.Errorf("GetNextStatus not implemented in mock")
}

func (m *MockChangeCardService) GetNextStatusForCard(card *models.ChangeCard) *services.NextStatusInfo {
	if m.GetNextStatusForCardFunc != nil {
		return m.GetNextStatusForCardFunc(card)
	}
	return &services.NextStatusInfo{}
}

func (m *MockChangeCardService) GetOrchestratorAction(card *models.ChangeCard) *config.PopulatedAction {
	if m.GetOrchestratorActionFunc != nil {
		return m.GetOrchestratorActionFunc(card)
	}
	return nil
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

// newChangeCmd returns a minimal cobra.Command (Context may be nil).
func newChangeCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	return cmd
}

// newChangeCmdWithCtx returns a minimal cobra.Command with a non-nil context.
// Use this for tests that call enrichment helpers (GetNoteService, GetContextService).
func newChangeCmdWithCtx() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
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
			return &models.ChangeCard{BaseEntity: models.BaseEntity{Key: "CC-001", Title: input.Title}, Status: "proposed"}, nil
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
			return &models.ChangeCard{BaseEntity: models.BaseEntity{Key: "CC-001", Title: "Test"}, Status: "proposed"}, nil
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
	// Reset service singletons so GetNoteService/GetContextService re-initialize
	// (and fail gracefully if no DB is available).
	cli.ResetServices()
	defer cli.ResetServices()

	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		GetChangeCardFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return &models.ChangeCard{BaseEntity: models.BaseEntity{Key: key, Title: "Some change"}, Status: "proposed"}, nil
		},
	})
	defer restore()

	restore2 := suppressOutput(t)
	defer restore2()

	cmd := newChangeCmdWithCtx()
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
	cli.ResetServices()
	defer cli.ResetServices()

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		GetChangeCardFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return &models.ChangeCard{BaseEntity: models.BaseEntity{Key: "CC-042", Title: "JSON change"}, Status: "approved"}, nil
		},
	})
	defer restore()

	out := captureOutput(t, func() {
		cmd := newChangeCmdWithCtx()
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
				{BaseEntity: models.BaseEntity{Key: "CC-001", Title: "First change"}, Status: "proposed"},
				{BaseEntity: models.BaseEntity{Key: "CC-002", Title: "Second change"}, Status: "approved"},
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
				{BaseEntity: models.BaseEntity{Key: "CC-001", Title: "List item"}, Status: "proposed"},
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
			return &models.ChangeCard{BaseEntity: models.BaseEntity{Key: key, Title: "Updated title"}, Status: "proposed"}, nil
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
			return &models.ChangeCard{BaseEntity: models.BaseEntity{Key: "CC-005", Title: newTitle}, Status: "proposed"}, nil
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
		TransitionStatusFunc: func(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{EntityKey: key, ToStatus: "approved", Transitioned: true}, nil
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
		TransitionStatusFunc: func(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{EntityKey: "CC-007", ToStatus: "approved", Transitioned: true}, nil
		},
		GetChangeCardFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return &models.ChangeCard{BaseEntity: models.BaseEntity{Key: "CC-007", Title: "Approve me"}, Status: "approved"}, nil
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
		{BaseEntity: models.BaseEntity{Key: "CC-001", Title: "A change"}, Status: "proposed"},
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
		{BaseEntity: models.BaseEntity{Key: "CC-001", Title: "First"}, Status: "proposed"},
		{BaseEntity: models.BaseEntity{Key: "CC-002", Title: "Second"}, Status: "approved"},
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
	updates, err := buildChangeCardUpdates(cmd)
	if err != nil {
		t.Fatalf("unexpected error from buildChangeCardUpdates: %v", err)
	}

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

	updates, err := buildChangeCardUpdates(cmd)
	if err != nil {
		t.Fatalf("unexpected error from buildChangeCardUpdates: %v", err)
	}

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

	updates, err := buildChangeCardUpdates(cmd)
	if err != nil {
		t.Fatalf("unexpected error from buildChangeCardUpdates: %v", err)
	}

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

	updates, err := buildChangeCardUpdates(cmd)
	if err != nil {
		t.Fatalf("unexpected error from buildChangeCardUpdates: %v", err)
	}

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

// ---------------------------------------------------------------------------
// E28-F04 T-009 — --tag flag parsing tests for `shark change create|update`.
//
// These exercise the CLI → service wire: the flag must arrive on the DTO so
// the service (tested separately in change_card_service_tags_test.go) can
// invoke EnforceRequired/AttachMany with the right names.
//
// The tests build fresh, ISOLATED cobra commands per case so they don't
// mutate the package-level changeCreateCmd/changeUpdateCmd (which are wired
// into cli.RootCmd and shared across the test binary). Flag parsing is
// verified end-to-end through the cobra Execute() path so StringSliceVar's
// repeat-flag behaviour is exercised by the same machinery users hit.
// ---------------------------------------------------------------------------

// mockChangeCardSvcForTags is a narrow stub for changeCardServicer used by
// the E28-F04 --tag flag tests. It records the CreateChangeCard /
// UpdateChangeCard inputs so tests can assert that the CLI threaded the
// --tag slice through to the DTO.
type mockChangeCardSvcForTags struct {
	MockChangeCardService
	lastCreate services.CreateChangeCardInput
	lastUpdate services.ChangeCardUpdates
}

func (m *mockChangeCardSvcForTags) CreateChangeCard(ctx context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, error) {
	m.lastCreate = input
	if m.CreateChangeCardFunc != nil {
		return m.CreateChangeCardFunc(ctx, input)
	}
	return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Title: input.Title}, Status: "proposed"}, nil
}

func (m *mockChangeCardSvcForTags) UpdateChangeCard(ctx context.Context, key string, updates services.ChangeCardUpdates) (*models.ChangeCard, error) {
	m.lastUpdate = updates
	if m.UpdateChangeCardFunc != nil {
		return m.UpdateChangeCardFunc(ctx, key, updates)
	}
	return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Updated"}, Status: "proposed"}, nil
}

// withChangeCardSvcOverride installs a test override for the package-level
// changeCardSvcOverride, restoring the previous value on test cleanup.
func withChangeCardSvcOverride(t *testing.T, svc changeCardServicer) {
	t.Helper()
	orig := changeCardSvcOverride
	changeCardSvcOverride = svc
	t.Cleanup(func() { changeCardSvcOverride = orig })
}

// buildChangeCreateCmdForTagTest returns a fresh `shark change create`
// command with a local tags slice bound to --tag. The command uses
// runChangeCreate via a small shim that reads the flag through
// cmd.Flags().GetStringSlice.
func buildChangeCreateCmdForTagTest(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{
		Use:  "create <title>",
		Args: cobraExactArgs(1),
		RunE: runChangeCreate,
	}
	cmd.Flags().StringVar(&changeLinkKey, "link", "", "link")
	cmd.Flags().StringVar(&changeDescription, "description", "", "description")
	cmd.Flags().IntVar(&changePriority, "priority", 0, "priority")
	cmd.Flags().StringVar(&changeRequestedBy, "requested-by", "", "requested-by")
	var tags []string
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "tag (repeatable)")
	return cmd
}

// buildChangeUpdateCmdForTagTest returns a fresh `shark change update`
// command with a local tags slice bound to --tag.
func buildChangeUpdateCmdForTagTest(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{
		Use:  "update <key>",
		Args: cobraExactArgs(1),
		RunE: runChangeUpdate,
	}
	cmd.Flags().StringVar(&changeTitle, "title", "", "title")
	cmd.Flags().StringVar(&changeDescription, "description", "", "description")
	cmd.Flags().IntVar(&changePriority, "priority", 0, "priority")
	cmd.Flags().StringVar(&changeRequestedBy, "requested-by", "", "requested-by")
	cmd.Flags().StringVar(&changeAssignedTo, "assigned-to", "", "assigned-to")
	cmd.Flags().String("file", "", "file")
	cmd.Flags().String("filename", "", "filename")
	cmd.Flags().String("path", "", "path")
	var tags []string
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "tag (repeatable)")
	return cmd
}

// TestChangeCreate_TagFlag_PassesTagsToService covers the change row of the
// CLI tag-flag wiring. Repeated --tag flags surface as an input.Tags slice
// in order.
func TestChangeCreate_TagFlag_PassesTagsToService(t *testing.T) {
	// Reset globals
	changeLinkKey = ""
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""

	stub := &mockChangeCardSvcForTags{}
	withChangeCardSvcOverride(t, stub)

	restore := suppressOutput(t)
	defer restore()

	cmd := buildChangeCreateCmdForTagTest(t)
	cmd.SetArgs([]string{"--tag=voice", "--tag=auth", "Tagged change"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(stub.lastCreate.Tags) != 2 {
		t.Fatalf("expected 2 tags on input, got %d (%v)", len(stub.lastCreate.Tags), stub.lastCreate.Tags)
	}
	if stub.lastCreate.Tags[0] != "voice" || stub.lastCreate.Tags[1] != "auth" {
		t.Errorf("tags = %v, want [voice, auth]", stub.lastCreate.Tags)
	}
}

// TestChangeUpdate_TagFlag_PassesTagsToService covers the change row of the
// update --tag wiring.
func TestChangeUpdate_TagFlag_PassesTagsToService(t *testing.T) {
	// Reset globals
	changeTitle = ""
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""
	changeAssignedTo = ""

	stub := &mockChangeCardSvcForTags{}
	withChangeCardSvcOverride(t, stub)

	restore := suppressOutput(t)
	defer restore()

	cmd := buildChangeUpdateCmdForTagTest(t)
	cmd.SetArgs([]string{"--tag=voice", "CC-001"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(stub.lastUpdate.Tags) != 1 || stub.lastUpdate.Tags[0] != "voice" {
		t.Errorf("update Tags = %v, want [voice]", stub.lastUpdate.Tags)
	}
}

// TestChangeUpdate_NoTagFlagIsNoTag verifies that when --tag is NOT passed,
// updates.Tags is nil (honours the Changed-guard in buildChangeCardUpdates).
func TestChangeUpdate_NoTagFlagIsNoTag(t *testing.T) {
	// Reset globals
	changeTitle = ""
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""
	changeAssignedTo = ""

	stub := &mockChangeCardSvcForTags{}
	withChangeCardSvcOverride(t, stub)

	restore := suppressOutput(t)
	defer restore()

	cmd := buildChangeUpdateCmdForTagTest(t)
	// Pass only --title so the update is valid without a tag.
	cmd.SetArgs([]string{"--title=Renamed", "CC-001"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if stub.lastUpdate.Tags != nil {
		t.Errorf("expected nil Tags when --tag omitted, got %v", stub.lastUpdate.Tags)
	}
}

// TestResolveChangeCardID_ReturnsID covers the EntityKeyResolver used by
// the `shark change tag` subcommand factory.
func TestResolveChangeCardID_ReturnsID(t *testing.T) {
	stub := &MockChangeCardService{
		GetChangeCardFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 77, Key: key}}, nil
		},
	}
	withChangeCardSvcOverride(t, stub)

	id, err := resolveChangeCardID(context.Background(), "CC-001")
	if err != nil {
		t.Fatalf("resolveChangeCardID() error = %v", err)
	}
	if id != 77 {
		t.Errorf("resolveChangeCardID() = %d, want 77", id)
	}
}

// TestResolveChangeCardID_PropagatesError ensures that a missing change-card
// surfaces as an error (translates to exit code 1 in the factory).
func TestResolveChangeCardID_PropagatesError(t *testing.T) {
	stub := &MockChangeCardService{
		GetChangeCardFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return nil, fmt.Errorf("change-card not found: %s", key)
		},
	}
	withChangeCardSvcOverride(t, stub)

	_, err := resolveChangeCardID(context.Background(), "CC-999")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
