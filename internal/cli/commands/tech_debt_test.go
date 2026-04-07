package commands

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// MockTechDebtService — function-field pattern (no real DB)
// ---------------------------------------------------------------------------

// MockTechDebtService implements techDebtServicer for testing.
type MockTechDebtService struct {
	CreateTechDebtFunc           func(ctx context.Context, input services.CreateTechDebtInput) (*models.TechDebt, error)
	GetTechDebtFunc              func(ctx context.Context, key string) (*models.TechDebt, error)
	ListTechDebtsFunc            func(ctx context.Context, filters services.TechDebtFilters) ([]*models.TechDebt, error)
	UpdateTechDebtFunc           func(ctx context.Context, key string, updates services.TechDebtUpdates) (*models.TechDebt, error)
	DeleteTechDebtFunc           func(ctx context.Context, key string) error
	TriageTechDebtFunc           func(ctx context.Context, key string, input services.TriageTechDebtInput) (*models.TechDebt, error)
	GetNextStatusForTechDebtFunc func(td *models.TechDebt) *services.NextStatusInfo
	GetOrchestratorActionFunc    func(td *models.TechDebt) *config.PopulatedAction
}

func (m *MockTechDebtService) CreateTechDebt(ctx context.Context, input services.CreateTechDebtInput) (*models.TechDebt, error) {
	if m.CreateTechDebtFunc != nil {
		return m.CreateTechDebtFunc(ctx, input)
	}
	return nil, fmt.Errorf("CreateTechDebt not implemented in mock")
}

func (m *MockTechDebtService) GetTechDebt(ctx context.Context, key string) (*models.TechDebt, error) {
	if m.GetTechDebtFunc != nil {
		return m.GetTechDebtFunc(ctx, key)
	}
	return nil, fmt.Errorf("GetTechDebt not implemented in mock")
}

func (m *MockTechDebtService) ListTechDebts(ctx context.Context, filters services.TechDebtFilters) ([]*models.TechDebt, error) {
	if m.ListTechDebtsFunc != nil {
		return m.ListTechDebtsFunc(ctx, filters)
	}
	return nil, fmt.Errorf("ListTechDebts not implemented in mock")
}

func (m *MockTechDebtService) UpdateTechDebt(ctx context.Context, key string, updates services.TechDebtUpdates) (*models.TechDebt, error) {
	if m.UpdateTechDebtFunc != nil {
		return m.UpdateTechDebtFunc(ctx, key, updates)
	}
	return nil, fmt.Errorf("UpdateTechDebt not implemented in mock")
}

func (m *MockTechDebtService) DeleteTechDebt(ctx context.Context, key string) error {
	if m.DeleteTechDebtFunc != nil {
		return m.DeleteTechDebtFunc(ctx, key)
	}
	return fmt.Errorf("DeleteTechDebt not implemented in mock")
}

func (m *MockTechDebtService) TriageTechDebt(ctx context.Context, key string, input services.TriageTechDebtInput) (*models.TechDebt, error) {
	if m.TriageTechDebtFunc != nil {
		return m.TriageTechDebtFunc(ctx, key, input)
	}
	return nil, fmt.Errorf("TriageTechDebt not implemented in mock")
}

func (m *MockTechDebtService) GetNextStatusForTechDebt(td *models.TechDebt) *services.NextStatusInfo {
	if m.GetNextStatusForTechDebtFunc != nil {
		return m.GetNextStatusForTechDebtFunc(td)
	}
	return &services.NextStatusInfo{}
}

func (m *MockTechDebtService) GetOrchestratorAction(td *models.TechDebt) *config.PopulatedAction {
	if m.GetOrchestratorActionFunc != nil {
		return m.GetOrchestratorActionFunc(td)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// injectMockTdSvc sets the package-level override and returns a
// cleanup function that restores the original nil value.
func injectMockTdSvc(t *testing.T, mock techDebtServicer) func() {
	t.Helper()
	tdSvcOverride = mock
	return func() { tdSvcOverride = nil }
}

// newTdCmd returns a minimal cobra.Command (Context may be nil).
func newTdCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	return cmd
}

// newTdCmdWithCtx returns a minimal cobra.Command with a non-nil context.
func newTdCmdWithCtx() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	return cmd
}

// ---------------------------------------------------------------------------
// runTdCreate tests
// ---------------------------------------------------------------------------

func TestRunTdCreate_Success(t *testing.T) {
	restore := injectMockTdSvc(t, &MockTechDebtService{
		CreateTechDebtFunc: func(ctx context.Context, input services.CreateTechDebtInput) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{Key: "TD-001", Title: input.Title},
				Status:     "identified",
				Category:   input.Category,
				Severity:   input.Severity,
			}, nil
		},
	})
	defer restore()

	// Reset globals
	tdCategory = ""
	tdSeverity = ""
	tdEffortEstimate = ""
	tdDescription = ""

	restore2 := suppressOutput(t)
	defer restore2()

	cmd := newTdCmd()
	cmd.Flags().String("category", "", "")
	cmd.Flags().String("severity", "", "")
	cmd.Flags().String("effort-estimate", "", "")
	cmd.Flags().String("description", "", "")
	cmd.Flags().String("file", "", "")
	cmd.Flags().Bool("force", false, "")
	err := runTdCreate(cmd, []string{"Refactor auth module"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunTdCreate_ServiceError(t *testing.T) {
	restore := injectMockTdSvc(t, &MockTechDebtService{
		CreateTechDebtFunc: func(ctx context.Context, input services.CreateTechDebtInput) (*models.TechDebt, error) {
			return nil, fmt.Errorf("db error")
		},
	})
	defer restore()

	tdCategory = ""
	tdSeverity = ""
	tdEffortEstimate = ""
	tdDescription = ""

	cmd := newTdCmd()
	cmd.Flags().String("category", "", "")
	cmd.Flags().String("severity", "", "")
	cmd.Flags().String("effort-estimate", "", "")
	cmd.Flags().String("description", "", "")
	cmd.Flags().String("file", "", "")
	cmd.Flags().Bool("force", false, "")
	err := runTdCreate(cmd, []string{"Bad item"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunTdCreate_JSONOutput(t *testing.T) {
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	restore := injectMockTdSvc(t, &MockTechDebtService{
		CreateTechDebtFunc: func(ctx context.Context, input services.CreateTechDebtInput) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{Key: "TD-001", Title: "Test"},
				Status:     "identified",
				Category:   "code-quality",
				Severity:   "medium",
			}, nil
		},
	})
	defer restore()

	tdCategory = ""
	tdSeverity = ""
	tdEffortEstimate = ""
	tdDescription = ""

	out := captureOutput(t, func() {
		cmd := newTdCmd()
		cmd.Flags().String("category", "", "")
		cmd.Flags().String("severity", "", "")
		cmd.Flags().String("effort-estimate", "", "")
		cmd.Flags().String("description", "", "")
		cmd.Flags().String("file", "", "")
		cmd.Flags().Bool("force", false, "")
		_ = runTdCreate(cmd, []string{"Test"})
	})

	if !bytes.Contains(out, []byte("TD-001")) {
		t.Errorf("expected JSON output to contain TD-001, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// runTdGet tests
// ---------------------------------------------------------------------------

func TestRunTdGet_Success(t *testing.T) {
	cli.ResetServices()
	defer cli.ResetServices()

	restore := injectMockTdSvc(t, &MockTechDebtService{
		GetTechDebtFunc: func(ctx context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{Key: key, Title: "Some tech debt"},
				Status:     "identified",
				Category:   "code-quality",
				Severity:   "medium",
			}, nil
		},
	})
	defer restore()

	restore2 := suppressOutput(t)
	defer restore2()

	cmd := newTdCmdWithCtx()
	err := runTdGet(cmd, []string{"TD-001"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunTdGet_NotFound(t *testing.T) {
	restore := injectMockTdSvc(t, &MockTechDebtService{
		GetTechDebtFunc: func(ctx context.Context, key string) (*models.TechDebt, error) {
			return nil, fmt.Errorf("tech-debt not found: %s", key)
		},
	})
	defer restore()

	cmd := newTdCmd()
	err := runTdGet(cmd, []string{"TD-999"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunTdGet_JSONOutput(t *testing.T) {
	cli.ResetServices()
	defer cli.ResetServices()

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	restore := injectMockTdSvc(t, &MockTechDebtService{
		GetTechDebtFunc: func(ctx context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{Key: "TD-042", Title: "JSON debt"},
				Status:     "triaged",
				Category:   "architecture",
				Severity:   "high",
			}, nil
		},
	})
	defer restore()

	out := captureOutput(t, func() {
		cmd := newTdCmdWithCtx()
		_ = runTdGet(cmd, []string{"TD-042"})
	})

	if !bytes.Contains(out, []byte("TD-042")) {
		t.Errorf("expected JSON output to contain TD-042, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// runTdList tests
// ---------------------------------------------------------------------------

func TestRunTdList_Empty(t *testing.T) {
	restore := injectMockTdSvc(t, &MockTechDebtService{
		ListTechDebtsFunc: func(ctx context.Context, filters services.TechDebtFilters) ([]*models.TechDebt, error) {
			return []*models.TechDebt{}, nil
		},
	})
	defer restore()

	tdStatus = ""
	tdCategory = ""
	tdSeverity = ""

	restore2 := suppressOutput(t)
	defer restore2()

	cmd := newTdCmd()
	cmd.Flags().String("status", "", "")
	cmd.Flags().String("category", "", "")
	cmd.Flags().String("severity", "", "")
	cmd.Flags().Bool("all", false, "")
	err := runTdList(cmd, []string{})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunTdList_WithResults(t *testing.T) {
	restore := injectMockTdSvc(t, &MockTechDebtService{
		ListTechDebtsFunc: func(ctx context.Context, filters services.TechDebtFilters) ([]*models.TechDebt, error) {
			return []*models.TechDebt{
				{BaseEntity: models.BaseEntity{Key: "TD-001", Title: "First debt"}, Status: "identified", Category: "code-quality", Severity: "medium"},
				{BaseEntity: models.BaseEntity{Key: "TD-002", Title: "Second debt"}, Status: "triaged", Category: "architecture", Severity: "high"},
			}, nil
		},
	})
	defer restore()

	tdStatus = ""
	tdCategory = ""
	tdSeverity = ""

	restore2 := suppressOutput(t)
	defer restore2()

	cmd := newTdCmd()
	cmd.Flags().String("status", "", "")
	cmd.Flags().String("category", "", "")
	cmd.Flags().String("severity", "", "")
	cmd.Flags().Bool("all", false, "")
	err := runTdList(cmd, []string{})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunTdList_ServiceError(t *testing.T) {
	restore := injectMockTdSvc(t, &MockTechDebtService{
		ListTechDebtsFunc: func(ctx context.Context, filters services.TechDebtFilters) ([]*models.TechDebt, error) {
			return nil, fmt.Errorf("database error")
		},
	})
	defer restore()

	tdStatus = ""
	tdCategory = ""
	tdSeverity = ""

	cmd := newTdCmd()
	cmd.Flags().String("status", "", "")
	cmd.Flags().String("category", "", "")
	cmd.Flags().String("severity", "", "")
	cmd.Flags().Bool("all", false, "")
	err := runTdList(cmd, []string{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunTdList_JSONOutput(t *testing.T) {
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	restore := injectMockTdSvc(t, &MockTechDebtService{
		ListTechDebtsFunc: func(ctx context.Context, filters services.TechDebtFilters) ([]*models.TechDebt, error) {
			return []*models.TechDebt{
				{BaseEntity: models.BaseEntity{Key: "TD-001", Title: "List item"}, Status: "identified", Category: "testing", Severity: "low"},
			}, nil
		},
	})
	defer restore()

	tdStatus = ""
	tdCategory = ""
	tdSeverity = ""

	out := captureOutput(t, func() {
		cmd := newTdCmd()
		cmd.Flags().String("status", "", "")
		cmd.Flags().String("category", "", "")
		cmd.Flags().String("severity", "", "")
		cmd.Flags().Bool("all", false, "")
		_ = runTdList(cmd, []string{})
	})

	if !bytes.Contains(out, []byte("TD-001")) {
		t.Errorf("expected JSON output to contain TD-001, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// runTdUpdate tests
// ---------------------------------------------------------------------------

func TestRunTdUpdate_Success(t *testing.T) {
	restore := injectMockTdSvc(t, &MockTechDebtService{
		UpdateTechDebtFunc: func(ctx context.Context, key string, updates services.TechDebtUpdates) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{Key: key, Title: "Updated title"},
				Status:     "identified",
				Category:   "code-quality",
				Severity:   "medium",
			}, nil
		},
	})
	defer restore()

	restore2 := suppressOutput(t)
	defer restore2()

	cmd := newTestTdUpdateCmd()
	_ = cmd.Flags().Set("title", "Updated title")
	err := runTdUpdate(cmd, []string{"TD-001"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunTdUpdate_ServiceError(t *testing.T) {
	restore := injectMockTdSvc(t, &MockTechDebtService{
		UpdateTechDebtFunc: func(ctx context.Context, key string, updates services.TechDebtUpdates) (*models.TechDebt, error) {
			return nil, fmt.Errorf("tech-debt not found: %s", key)
		},
	})
	defer restore()

	cmd := newTestTdUpdateCmd()
	_ = cmd.Flags().Set("title", "Updated title")
	err := runTdUpdate(cmd, []string{"TD-999"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunTdUpdate_NoFlags(t *testing.T) {
	restore := injectMockTdSvc(t, &MockTechDebtService{})
	defer restore()

	cmd := newTestTdUpdateCmd()
	err := runTdUpdate(cmd, []string{"TD-001"})
	if err == nil {
		t.Fatal("expected error for no flags, got nil")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("at least one update flag")) {
		t.Errorf("expected 'at least one update flag' error, got: %v", err)
	}
}

func TestRunTdUpdate_JSONOutput(t *testing.T) {
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	restore := injectMockTdSvc(t, &MockTechDebtService{
		UpdateTechDebtFunc: func(ctx context.Context, key string, updates services.TechDebtUpdates) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{Key: "TD-005", Title: "Updated"},
				Status:     "identified",
				Category:   "code-quality",
				Severity:   "medium",
			}, nil
		},
	})
	defer restore()

	out := captureOutput(t, func() {
		cmd := newTestTdUpdateCmd()
		_ = cmd.Flags().Set("title", "Updated")
		_ = runTdUpdate(cmd, []string{"TD-005"})
	})

	if !bytes.Contains(out, []byte("TD-005")) {
		t.Errorf("expected JSON output to contain TD-005, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// runTdDelete tests
// ---------------------------------------------------------------------------

func TestRunTdDelete_Force_Success(t *testing.T) {
	restore := injectMockTdSvc(t, &MockTechDebtService{
		DeleteTechDebtFunc: func(ctx context.Context, key string) error {
			return nil
		},
	})
	defer restore()

	restore2 := suppressOutput(t)
	defer restore2()

	cmd := newTdCmd()
	cmd.Flags().Bool("force", true, "")
	_ = cmd.Flags().Set("force", "true")
	err := runTdDelete(cmd, []string{"TD-001"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunTdDelete_Force_NotFound(t *testing.T) {
	restore := injectMockTdSvc(t, &MockTechDebtService{
		DeleteTechDebtFunc: func(ctx context.Context, key string) error {
			return fmt.Errorf("tech-debt not found: %s", key)
		},
	})
	defer restore()

	cmd := newTdCmd()
	cmd.Flags().Bool("force", true, "")
	_ = cmd.Flags().Set("force", "true")
	err := runTdDelete(cmd, []string{"TD-999"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunTdDelete_Force_JSONOutput(t *testing.T) {
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	restore := injectMockTdSvc(t, &MockTechDebtService{
		DeleteTechDebtFunc: func(ctx context.Context, key string) error {
			return nil
		},
	})
	defer restore()

	out := captureOutput(t, func() {
		cmd := newTdCmd()
		cmd.Flags().Bool("force", true, "")
		_ = cmd.Flags().Set("force", "true")
		_ = runTdDelete(cmd, []string{"TD-003"})
	})

	if !bytes.Contains(out, []byte("TD-003")) {
		t.Errorf("expected JSON output to contain TD-003, got: %s", out)
	}
}

func TestRunTdDelete_NoForce_NotFound(t *testing.T) {
	restore := injectMockTdSvc(t, &MockTechDebtService{
		GetTechDebtFunc: func(ctx context.Context, key string) (*models.TechDebt, error) {
			return nil, fmt.Errorf("tech-debt not found: %s", key)
		},
	})
	defer restore()

	cmd := newTdCmd()
	cmd.Flags().Bool("force", false, "")
	err := runTdDelete(cmd, []string{"TD-999"})
	if err == nil {
		t.Fatal("expected error for not-found get, got nil")
	}
}

// ---------------------------------------------------------------------------
// runTdTriage tests
// ---------------------------------------------------------------------------

func TestRunTdTriage_Success(t *testing.T) {
	restore := injectMockTdSvc(t, &MockTechDebtService{
		TriageTechDebtFunc: func(ctx context.Context, key string, input services.TriageTechDebtInput) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{Key: key, Title: "Triaged item"},
				Status:     "triaged",
				Category:   models.TechDebtCategory(input.Category),
				Severity:   models.TechDebtSeverity(input.Severity),
			}, nil
		},
	})
	defer restore()

	restore2 := suppressOutput(t)
	defer restore2()

	cmd := newTdCmd()
	cmd.Flags().String("severity", "high", "")
	cmd.Flags().String("category", "architecture", "")
	cmd.Flags().String("effort-estimate", "", "")
	err := runTdTriage(cmd, []string{"TD-001"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunTdTriage_ServiceError(t *testing.T) {
	restore := injectMockTdSvc(t, &MockTechDebtService{
		TriageTechDebtFunc: func(ctx context.Context, key string, input services.TriageTechDebtInput) (*models.TechDebt, error) {
			return nil, fmt.Errorf("tech-debt not found: %s", key)
		},
	})
	defer restore()

	cmd := newTdCmd()
	cmd.Flags().String("severity", "high", "")
	cmd.Flags().String("category", "", "")
	cmd.Flags().String("effort-estimate", "", "")
	err := runTdTriage(cmd, []string{"TD-999"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunTdTriage_JSONOutput(t *testing.T) {
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	restore := injectMockTdSvc(t, &MockTechDebtService{
		TriageTechDebtFunc: func(ctx context.Context, key string, input services.TriageTechDebtInput) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{Key: "TD-007", Title: "Triage me"},
				Status:     "triaged",
				Category:   "dependency",
				Severity:   "critical",
			}, nil
		},
	})
	defer restore()

	out := captureOutput(t, func() {
		cmd := newTdCmd()
		cmd.Flags().String("severity", "critical", "")
		cmd.Flags().String("category", "dependency", "")
		cmd.Flags().String("effort-estimate", "", "")
		_ = runTdTriage(cmd, []string{"TD-007"})
	})

	if !bytes.Contains(out, []byte("TD-007")) {
		t.Errorf("expected JSON output to contain TD-007, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// Helper function tests
// ---------------------------------------------------------------------------

func TestTruncateTdString_Short(t *testing.T) {
	result := truncateTdString("Hello", 10)
	if result != "Hello" {
		t.Errorf("expected %q, got %q", "Hello", result)
	}
}

func TestTruncateTdString_Exact(t *testing.T) {
	result := truncateTdString("Hello", 5)
	if result != "Hello" {
		t.Errorf("expected %q, got %q", "Hello", result)
	}
}

func TestTruncateTdString_Long(t *testing.T) {
	result := truncateTdString("Hello World", 8)
	if result != "Hello..." {
		t.Errorf("expected %q, got %q", "Hello...", result)
	}
}

func TestTruncateTdString_VeryShortMax(t *testing.T) {
	result := truncateTdString("Hello", 3)
	if result != "Hel" {
		t.Errorf("expected %q, got %q", "Hel", result)
	}
}

func TestTruncateTdString_Empty(t *testing.T) {
	result := truncateTdString("", 10)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestTruncateTdString_Table(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		maxLen   int
		expected string
	}{
		{"short string no truncation", "hi", 10, "hi"},
		{"exact length no truncation", "hello", 5, "hello"},
		{"one over max", "hello!", 5, "he..."},
		{"long title", "This is a very long tech-debt title that should be truncated", 20, "This is a very lo..."},
		{"empty string", "", 10, ""},
		{"maxLen 1", "abc", 1, "a"},
		{"maxLen 2", "abc", 2, "ab"},
		{"maxLen 3", "abc", 3, "abc"},
		{"maxLen 4 with truncation", "abcde", 4, "a..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateTdString(tt.s, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateTdString(%q, %d): expected %q, got %q",
					tt.s, tt.maxLen, tt.expected, result)
			}
		})
	}
}

// TestPrintTechDebtTable_NonEmpty verifies that a non-empty list is rendered without errors.
func TestPrintTechDebtTable_NonEmpty(t *testing.T) {
	items := []*models.TechDebt{
		{BaseEntity: models.BaseEntity{Key: "TD-001", Title: "First"}, Status: "identified", Category: "code-quality", Severity: "medium"},
		{BaseEntity: models.BaseEntity{Key: "TD-002", Title: "Second"}, Status: "triaged", Category: "architecture", Severity: "high"},
	}

	restore := suppressOutput(t)
	defer restore()

	err := printTechDebtTable(items)
	if err != nil {
		t.Errorf("printTechDebtTable returned unexpected error: %v", err)
	}
}

// newTestTdUpdateCmd builds a cobra.Command with the flags used by runTdUpdate.
func newTestTdUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "update"}
	cmd.Flags().StringVar(&tdTitle, "title", "", "Title")
	cmd.Flags().StringVar(&tdCategory, "category", "", "Category")
	cmd.Flags().StringVar(&tdSeverity, "severity", "", "Severity")
	cmd.Flags().StringVar(&tdEffortEstimate, "effort-estimate", "", "Effort estimate")
	cmd.Flags().StringVar(&tdDescription, "description", "", "Description")
	cmd.Flags().String("file", "", "File path")
	cmd.Flags().String("filename", "", "Alias")
	cmd.Flags().String("path", "", "Alias")
	return cmd
}
