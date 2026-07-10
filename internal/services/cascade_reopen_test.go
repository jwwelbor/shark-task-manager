package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	gosync "sync"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ============================================================
// Mock implementations for cascade dependencies
// ============================================================

// mockCascadeFeatureRepo implements CascadeFeatureRepo.
type mockCascadeFeatureRepo struct {
	GetByIDFunc         func(ctx context.Context, id int64) (*models.Feature, error)
	GetByIDTxFunc       func(ctx context.Context, tx *sql.Tx, id int64) (*models.Feature, error)
	UpdateStatusTxFunc  func(ctx context.Context, tx *sql.Tx, id int64, status string, agent *string, notes *string) error
	getByIDCalls        int
	getByIDTxCalls      int
	updateStatusTxCalls int
	lastUpdateID        int64
	lastUpdateStatus    string
	lastUpdateTx        *sql.Tx
}

func (m *mockCascadeFeatureRepo) GetByID(ctx context.Context, id int64) (*models.Feature, error) {
	m.getByIDCalls++
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("GetByID not implemented in mockCascadeFeatureRepo")
}

func (m *mockCascadeFeatureRepo) GetByIDTx(ctx context.Context, tx *sql.Tx, id int64) (*models.Feature, error) {
	m.getByIDTxCalls++
	if m.GetByIDTxFunc != nil {
		return m.GetByIDTxFunc(ctx, tx, id)
	}
	// Default: delegate to GetByIDFunc so tests that only set GetByIDFunc still work
	// for the in-tx re-fetch path.
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("GetByIDTx not implemented in mockCascadeFeatureRepo")
}

func (m *mockCascadeFeatureRepo) UpdateStatusTx(ctx context.Context, tx *sql.Tx, id int64, status string, agent *string, notes *string) error {
	m.updateStatusTxCalls++
	m.lastUpdateID = id
	m.lastUpdateStatus = status
	m.lastUpdateTx = tx
	if m.UpdateStatusTxFunc != nil {
		return m.UpdateStatusTxFunc(ctx, tx, id, status, agent, notes)
	}
	return nil
}

// mockCascadeEpicRepo implements CascadeEpicRepo.
type mockCascadeEpicRepo struct {
	GetByIDFunc         func(ctx context.Context, id int64) (*models.Epic, error)
	GetByIDTxFunc       func(ctx context.Context, tx *sql.Tx, id int64) (*models.Epic, error)
	UpdateStatusTxFunc  func(ctx context.Context, tx *sql.Tx, id int64, status string, agent *string, notes *string) error
	getByIDCalls        int
	getByIDTxCalls      int
	updateStatusTxCalls int
	lastUpdateID        int64
	lastUpdateStatus    string
	lastUpdateTx        *sql.Tx
}

func (m *mockCascadeEpicRepo) GetByID(ctx context.Context, id int64) (*models.Epic, error) {
	m.getByIDCalls++
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("GetByID not implemented in mockCascadeEpicRepo")
}

func (m *mockCascadeEpicRepo) GetByIDTx(ctx context.Context, tx *sql.Tx, id int64) (*models.Epic, error) {
	m.getByIDTxCalls++
	if m.GetByIDTxFunc != nil {
		return m.GetByIDTxFunc(ctx, tx, id)
	}
	// Default: delegate to GetByIDFunc so tests that only set GetByIDFunc still work
	// for the in-tx re-fetch path.
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("GetByIDTx not implemented in mockCascadeEpicRepo")
}

func (m *mockCascadeEpicRepo) UpdateStatusTx(ctx context.Context, tx *sql.Tx, id int64, status string, agent *string, notes *string) error {
	m.updateStatusTxCalls++
	m.lastUpdateID = id
	m.lastUpdateStatus = status
	m.lastUpdateTx = tx
	if m.UpdateStatusTxFunc != nil {
		return m.UpdateStatusTxFunc(ctx, tx, id, status, agent, notes)
	}
	return nil
}

// mockParentReopenHistoryQuerier implements ParentReopenHistoryQuerier.
type mockParentReopenHistoryQuerier struct {
	GetLastNonTerminalStatusFunc func(ctx context.Context, entityType models.EntityType, entityID int64, terminalStatuses []string) (string, bool, error)
	calls                        int
}

func (m *mockParentReopenHistoryQuerier) GetLastNonTerminalStatus(
	ctx context.Context,
	entityType models.EntityType,
	entityID int64,
	terminalStatuses []string,
) (string, bool, error) {
	m.calls++
	if m.GetLastNonTerminalStatusFunc != nil {
		return m.GetLastNonTerminalStatusFunc(ctx, entityType, entityID, terminalStatuses)
	}
	return "", false, nil
}

// mockEntityHistoryTxRecorder implements EntityHistoryTxRecorder.
type mockEntityHistoryTxRecorder struct {
	CreateTxFunc func(ctx context.Context, tx *sql.Tx, history *models.EntityHistory) error
	calls        int
	captured     []*models.EntityHistory
}

func (m *mockEntityHistoryTxRecorder) CreateTx(ctx context.Context, tx *sql.Tx, history *models.EntityHistory) error {
	m.calls++
	// Deep-copy the entity history so callers can inspect it after the test.
	h := *history
	m.captured = append(m.captured, &h)
	if m.CreateTxFunc != nil {
		return m.CreateTxFunc(ctx, tx, history)
	}
	return nil
}

// ============================================================
// Fake transaction — used where a real *sql.Tx is required but
// all actual DB work is intercepted by mocks.
// ============================================================

// fakeTx wraps a mock in-memory Tx that tracks Commit/Rollback calls.
type fakeTx struct {
	committed  bool
	rolledBack bool
	commitErr  error
}

func (f *fakeTx) Commit() error {
	if f.commitErr != nil {
		return f.commitErr
	}
	f.committed = true
	return nil
}

func (f *fakeTx) Rollback() error {
	f.rolledBack = true
	return nil
}

// txBeginner is the narrow interface used by cascadeParentReopens to open a
// transaction. This is satisfied by *dbconn.DB in production and by mockTxBeginner in tests.
// (The interface is defined in cascade_reopen.go alongside the implementation.)

// mockTxBeginner is a test double for the txBeginner interface.
// It returns a *sql.Tx that is backed by an in-memory fakeTx tracker.
// Because *sql.Tx cannot be constructed directly (it holds unexported fields),
// the cascade implementation under test must accept a txBeginner interface
// rather than a concrete *dbconn.DB.
type mockTxBeginner struct {
	tx         *fakeTx
	beginErr   error
	beginCalls int
}

func newMockTxBeginner() (*mockTxBeginner, *fakeTx) {
	ftx := &fakeTx{}
	return &mockTxBeginner{tx: ftx}, ftx
}

func (m *mockTxBeginner) BeginTxContext(ctx context.Context) (*sql.Tx, error) {
	m.beginCalls++
	if m.beginErr != nil {
		return nil, m.beginErr
	}
	// Return nil – the mock repos and history recorder accept nil *sql.Tx.
	// This reflects that in tests we intercept all DB calls at the mock level.
	return nil, nil
}

// ============================================================
// Test helpers
// ============================================================

func newTestFeature(id int64, epicID int64, status string) *models.Feature {
	return &models.Feature{
		BaseEntity: models.BaseEntity{
			ID:  id,
			Key: fmt.Sprintf("E07-F%02d", id),
		},
		EpicID: epicID,
		Status: models.FeatureStatus(status),
	}
}

func newTestEpic(id int64, status string) *models.Epic {
	return &models.Epic{
		BaseEntity: models.BaseEntity{
			ID:  id,
			Key: fmt.Sprintf("E%02d", id),
		},
		Status: models.EpicStatus(status),
	}
}

// ============================================================
// AC-01: Task backward transition reopens parent feature
// ============================================================

func TestCascade_TaskBackwardReopensFeature(t *testing.T) {
	ctx := context.Background()

	featureID := int64(101)
	epicID := int64(201)
	feature := newTestFeature(featureID, epicID, "completed") // terminal
	epic := newTestEpic(epicID, "in_development")             // non-terminal — not affected

	txBeginner, _ := newMockTxBeginner()
	featureRepo := &mockCascadeFeatureRepo{
		GetByIDFunc: func(_ context.Context, id int64) (*models.Feature, error) {
			if id == featureID {
				return feature, nil
			}
			return nil, fmt.Errorf("unexpected feature id %d", id)
		},
	}
	epicRepo := &mockCascadeEpicRepo{
		GetByIDFunc: func(_ context.Context, id int64) (*models.Epic, error) {
			if id == epicID {
				return epic, nil
			}
			return nil, fmt.Errorf("unexpected epic id %d", id)
		},
	}
	histQuerier := &mockParentReopenHistoryQuerier{
		GetLastNonTerminalStatusFunc: func(_ context.Context, entityType models.EntityType, entityID int64, _ []string) (string, bool, error) {
			if entityType == models.EntityTypeFeature && entityID == featureID {
				return "in_qa", true, nil
			}
			return "", false, nil
		},
	}
	histTx := &mockEntityHistoryTxRecorder{}

	wf := newTestTaskWorkflowService(t)

	deps := cascadeDeps{
		db:             txBeginner,
		featureRepo:    featureRepo,
		epicRepo:       epicRepo,
		historyQuerier: histQuerier,
		historyTx:      histTx,
		workflowSvc:    wf,
	}
	trigger := cascadeTrigger{
		triggerKey:  "E07-F01-003",
		triggerKind: "regression",
		triggerType: models.EntityTypeTask,
		startLeg:    cascadeLegFeature,
		featureID:   featureID,
	}

	cascadeParentReopens(ctx, deps, trigger)

	// Feature was terminal → should be updated.
	if featureRepo.updateStatusTxCalls != 1 {
		t.Errorf("expected 1 feature UpdateStatusTx call, got %d", featureRepo.updateStatusTxCalls)
	}
	if featureRepo.lastUpdateStatus != "in_qa" {
		t.Errorf("expected feature reopen to in_qa, got %q", featureRepo.lastUpdateStatus)
	}

	// Epic was non-terminal → no update.
	if epicRepo.updateStatusTxCalls != 0 {
		t.Errorf("expected 0 epic UpdateStatusTx calls (non-terminal), got %d", epicRepo.updateStatusTxCalls)
	}

	// Exactly one history row written (for feature).
	if histTx.calls != 1 {
		t.Errorf("expected 1 history row, got %d", histTx.calls)
	}
	row := histTx.captured[0]
	if row.EntityType != models.EntityTypeFeature {
		t.Errorf("expected history row EntityType=feature, got %q", row.EntityType)
	}
	if !strings.HasPrefix(*row.Notes, "auto_reopen: triggered by E07-F01-003 regression") {
		t.Errorf("notes should start with auto_reopen prefix, got %q", *row.Notes)
	}
}

// ============================================================
// AC-02: Task backward transition also reopens grandparent epic
// ============================================================

func TestCascade_TaskBackwardReopensEpic(t *testing.T) {
	ctx := context.Background()

	featureID := int64(102)
	epicID := int64(202)
	feature := newTestFeature(featureID, epicID, "completed")
	epic := newTestEpic(epicID, "completed")

	txBeginner, _ := newMockTxBeginner()
	featureRepo := &mockCascadeFeatureRepo{
		GetByIDFunc: func(_ context.Context, id int64) (*models.Feature, error) {
			return feature, nil
		},
	}
	epicRepo := &mockCascadeEpicRepo{
		GetByIDFunc: func(_ context.Context, id int64) (*models.Epic, error) {
			return epic, nil
		},
	}
	histQuerier := &mockParentReopenHistoryQuerier{
		GetLastNonTerminalStatusFunc: func(_ context.Context, entityType models.EntityType, entityID int64, _ []string) (string, bool, error) {
			switch entityType {
			case models.EntityTypeFeature:
				return "in_qa", true, nil
			case models.EntityTypeEpic:
				return "in_development", true, nil
			}
			return "", false, nil
		},
	}
	histTx := &mockEntityHistoryTxRecorder{}

	wf := newTestTaskWorkflowService(t)

	deps := cascadeDeps{
		db:             txBeginner,
		featureRepo:    featureRepo,
		epicRepo:       epicRepo,
		historyQuerier: histQuerier,
		historyTx:      histTx,
		workflowSvc:    wf,
	}
	trigger := cascadeTrigger{
		triggerKey:  "E07-F01-003",
		triggerKind: "regression",
		triggerType: models.EntityTypeTask,
		startLeg:    cascadeLegFeature,
		featureID:   featureID,
	}

	cascadeParentReopens(ctx, deps, trigger)

	if featureRepo.updateStatusTxCalls != 1 {
		t.Errorf("expected 1 feature update, got %d", featureRepo.updateStatusTxCalls)
	}
	if featureRepo.lastUpdateStatus != "in_qa" {
		t.Errorf("feature should reopen to in_qa, got %q", featureRepo.lastUpdateStatus)
	}

	if epicRepo.updateStatusTxCalls != 1 {
		t.Errorf("expected 1 epic update, got %d", epicRepo.updateStatusTxCalls)
	}
	if epicRepo.lastUpdateStatus != "in_development" {
		t.Errorf("epic should reopen to in_development, got %q", epicRepo.lastUpdateStatus)
	}

	if histTx.calls != 2 {
		t.Errorf("expected 2 history rows (feature + epic), got %d", histTx.calls)
	}

	// Both rows must use the same *sql.Tx (nil in tests, but consistent).
	if histTx.captured[0] == nil || histTx.captured[1] == nil {
		t.Fatal("expected non-nil captured history rows")
	}
}

// ============================================================
// AC-03: Feature backward transition reopens parent epic only
// ============================================================

func TestCascade_FeatureBackwardReopensEpic(t *testing.T) {
	ctx := context.Background()

	featureID := int64(103)
	epicID := int64(203)
	feature := newTestFeature(featureID, epicID, "in_qa") // trigger is the feature itself
	epic := newTestEpic(epicID, "completed")

	txBeginner, _ := newMockTxBeginner()
	featureRepo := &mockCascadeFeatureRepo{
		GetByIDFunc: func(_ context.Context, id int64) (*models.Feature, error) {
			return feature, nil
		},
	}
	epicRepo := &mockCascadeEpicRepo{
		GetByIDFunc: func(_ context.Context, id int64) (*models.Epic, error) {
			return epic, nil
		},
	}
	histQuerier := &mockParentReopenHistoryQuerier{
		GetLastNonTerminalStatusFunc: func(_ context.Context, entityType models.EntityType, _ int64, _ []string) (string, bool, error) {
			if entityType == models.EntityTypeEpic {
				return "in_development", true, nil
			}
			return "", false, nil
		},
	}
	histTx := &mockEntityHistoryTxRecorder{}

	wf := newTestTaskWorkflowService(t)

	deps := cascadeDeps{
		db:             txBeginner,
		featureRepo:    featureRepo,
		epicRepo:       epicRepo,
		historyQuerier: histQuerier,
		historyTx:      histTx,
		workflowSvc:    wf,
	}
	// startLeg = cascadeLegEpic → skip feature update, only update epic.
	trigger := cascadeTrigger{
		triggerKey:  "E07-F01",
		triggerKind: "regression",
		triggerType: models.EntityTypeFeature,
		startLeg:    cascadeLegEpic,
		featureID:   featureID,
	}

	cascadeParentReopens(ctx, deps, trigger)

	// Feature must NOT be updated (it's the trigger).
	if featureRepo.updateStatusTxCalls != 0 {
		t.Errorf("expected 0 feature updates (feature is trigger), got %d", featureRepo.updateStatusTxCalls)
	}

	// Epic must be updated.
	if epicRepo.updateStatusTxCalls != 1 {
		t.Errorf("expected 1 epic update, got %d", epicRepo.updateStatusTxCalls)
	}
	if epicRepo.lastUpdateStatus != "in_development" {
		t.Errorf("epic should reopen to in_development, got %q", epicRepo.lastUpdateStatus)
	}

	// Exactly one history row (for epic).
	if histTx.calls != 1 {
		t.Errorf("expected 1 history row (epic only), got %d", histTx.calls)
	}
}

// ============================================================
// epicID path: cascadeLegEpic + epicID skips feature lookup
// ============================================================

// TestCascade_EpicIDPathSkipsFeatureLookup verifies that when cascadeTrigger has
// startLeg=cascadeLegEpic AND epicID != 0, the cascade goes directly to the epic
// without calling featureRepo.GetByID at all. This is the code path used by
// maybeReopenParentEpic (CreateFeature trigger) after the REQ-F-003 refactor.
func TestCascade_EpicIDPathSkipsFeatureLookup(t *testing.T) {
	ctx := context.Background()

	epicID := int64(299)
	epic := newTestEpic(epicID, "completed") // terminal — should be reopened

	txBeginner, _ := newMockTxBeginner()
	featureRepo := &mockCascadeFeatureRepo{
		GetByIDFunc: func(_ context.Context, id int64) (*models.Feature, error) {
			t.Errorf("featureRepo.GetByID should NOT be called on the epicID path, got id=%d", id)
			return nil, fmt.Errorf("unexpected call")
		},
	}
	epicRepo := &mockCascadeEpicRepo{
		GetByIDFunc: func(_ context.Context, id int64) (*models.Epic, error) {
			if id == epicID {
				return epic, nil
			}
			return nil, fmt.Errorf("unexpected epic id %d", id)
		},
	}
	histQuerier := &mockParentReopenHistoryQuerier{
		GetLastNonTerminalStatusFunc: func(_ context.Context, entityType models.EntityType, _ int64, _ []string) (string, bool, error) {
			if entityType == models.EntityTypeEpic {
				return "in_development", true, nil
			}
			return "", false, nil
		},
	}
	histTx := &mockEntityHistoryTxRecorder{}

	wf := newTestTaskWorkflowService(t)

	deps := cascadeDeps{
		db:             txBeginner,
		featureRepo:    featureRepo,
		epicRepo:       epicRepo,
		historyQuerier: histQuerier,
		historyTx:      histTx,
		workflowSvc:    wf,
	}
	// epicID path: startLeg=cascadeLegEpic and epicID is set — no featureID needed.
	trigger := cascadeTrigger{
		triggerKey:  "E07-F05",
		triggerKind: "creation",
		triggerType: models.EntityTypeFeature,
		startLeg:    cascadeLegEpic,
		epicID:      epicID,
	}

	cascadeParentReopens(ctx, deps, trigger)

	// featureRepo.GetByID must not have been called.
	if featureRepo.getByIDCalls != 0 {
		t.Errorf("expected 0 featureRepo.GetByID calls, got %d", featureRepo.getByIDCalls)
	}

	// Feature must NOT be updated (epicID path has no feature leg).
	if featureRepo.updateStatusTxCalls != 0 {
		t.Errorf("expected 0 feature updates, got %d", featureRepo.updateStatusTxCalls)
	}

	// Epic must be reopened.
	if epicRepo.updateStatusTxCalls != 1 {
		t.Errorf("expected 1 epic update, got %d", epicRepo.updateStatusTxCalls)
	}
	if epicRepo.lastUpdateStatus != "in_development" {
		t.Errorf("epic should reopen to in_development, got %q", epicRepo.lastUpdateStatus)
	}

	// Exactly one history row (for epic only).
	if histTx.calls != 1 {
		t.Errorf("expected 1 history row (epic only), got %d", histTx.calls)
	}
	row := histTx.captured[0]
	if row.EntityType != models.EntityTypeEpic {
		t.Errorf("expected history row EntityType=epic, got %q", row.EntityType)
	}
	if !strings.HasPrefix(*row.Notes, "auto_reopen:") {
		t.Errorf("notes should start with auto_reopen: prefix, got %q", *row.Notes)
	}
}

// TestCascade_EpicIDPathNoOpWhenEpicNonTerminal verifies the epicID path is a no-op
// when the epic is already non-terminal, without touching the feature layer.
func TestCascade_EpicIDPathNoOpWhenEpicNonTerminal(t *testing.T) {
	ctx := context.Background()

	epicID := int64(298)
	epic := newTestEpic(epicID, "in_development") // non-terminal

	txBeginner, _ := newMockTxBeginner()
	featureRepo := &mockCascadeFeatureRepo{
		GetByIDFunc: func(_ context.Context, id int64) (*models.Feature, error) {
			t.Errorf("featureRepo.GetByID should NOT be called on the epicID path, got id=%d", id)
			return nil, fmt.Errorf("unexpected call")
		},
	}
	epicRepo := &mockCascadeEpicRepo{
		GetByIDFunc: func(_ context.Context, id int64) (*models.Epic, error) {
			return epic, nil
		},
	}
	histTx := &mockEntityHistoryTxRecorder{}
	wf := newTestTaskWorkflowService(t)

	deps := cascadeDeps{
		db:             txBeginner,
		featureRepo:    featureRepo,
		epicRepo:       epicRepo,
		historyQuerier: &mockParentReopenHistoryQuerier{},
		historyTx:      histTx,
		workflowSvc:    wf,
	}
	trigger := cascadeTrigger{
		triggerKey:  "E07-F06",
		triggerKind: "creation",
		triggerType: models.EntityTypeFeature,
		startLeg:    cascadeLegEpic,
		epicID:      epicID,
	}

	cascadeParentReopens(ctx, deps, trigger)

	if featureRepo.getByIDCalls != 0 {
		t.Errorf("expected 0 featureRepo.GetByID calls, got %d", featureRepo.getByIDCalls)
	}
	if epicRepo.updateStatusTxCalls != 0 {
		t.Errorf("expected 0 epic updates (non-terminal), got %d", epicRepo.updateStatusTxCalls)
	}
	if histTx.calls != 0 {
		t.Errorf("expected 0 history rows (no-op), got %d", histTx.calls)
	}
	// Transaction must not have been opened (early return before BeginTxContext).
	if txBeginner.beginCalls != 0 {
		t.Errorf("expected 0 BeginTx calls on no-op, got %d", txBeginner.beginCalls)
	}
}

// ============================================================
// AC-04: Non-terminal feature is skipped; epic still checked
// ============================================================

func TestCascade_NonTerminalFeatureContinuesToEpic(t *testing.T) {
	ctx := context.Background()

	featureID := int64(104)
	epicID := int64(204)
	feature := newTestFeature(featureID, epicID, "in_qa") // non-terminal
	epic := newTestEpic(epicID, "completed")              // terminal

	txBeginner, _ := newMockTxBeginner()
	featureRepo := &mockCascadeFeatureRepo{
		GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) {
			return feature, nil
		},
	}
	epicRepo := &mockCascadeEpicRepo{
		GetByIDFunc: func(_ context.Context, _ int64) (*models.Epic, error) {
			return epic, nil
		},
	}
	histQuerier := &mockParentReopenHistoryQuerier{
		GetLastNonTerminalStatusFunc: func(_ context.Context, entityType models.EntityType, _ int64, _ []string) (string, bool, error) {
			if entityType == models.EntityTypeEpic {
				return "in_development", true, nil
			}
			return "", false, nil
		},
	}
	histTx := &mockEntityHistoryTxRecorder{}
	wf := newTestTaskWorkflowService(t)

	deps := cascadeDeps{
		db:             txBeginner,
		featureRepo:    featureRepo,
		epicRepo:       epicRepo,
		historyQuerier: histQuerier,
		historyTx:      histTx,
		workflowSvc:    wf,
	}
	trigger := cascadeTrigger{
		triggerKey:  "E07-F01-003",
		triggerKind: "regression",
		triggerType: models.EntityTypeTask,
		startLeg:    cascadeLegFeature,
		featureID:   featureID,
	}

	cascadeParentReopens(ctx, deps, trigger)

	// Feature non-terminal → no update.
	if featureRepo.updateStatusTxCalls != 0 {
		t.Errorf("expected 0 feature updates (already non-terminal), got %d", featureRepo.updateStatusTxCalls)
	}

	// Epic terminal → must be updated.
	if epicRepo.updateStatusTxCalls != 1 {
		t.Errorf("expected 1 epic update, got %d", epicRepo.updateStatusTxCalls)
	}

	// One history row only (for epic).
	if histTx.calls != 1 {
		t.Errorf("expected 1 history row (epic only), got %d", histTx.calls)
	}
	if histTx.captured[0].EntityType != models.EntityTypeEpic {
		t.Errorf("history row should be for epic, got %q", histTx.captured[0].EntityType)
	}
}

// ============================================================
// AC-05: Both ancestors non-terminal — complete no-op
// ============================================================

func TestCascade_AllAncestorsNonTerminalNoOp(t *testing.T) {
	ctx := context.Background()

	featureID := int64(105)
	epicID := int64(205)
	feature := newTestFeature(featureID, epicID, "in_qa") // non-terminal
	epic := newTestEpic(epicID, "in_development")         // non-terminal

	txBeginner, _ := newMockTxBeginner()
	featureRepo := &mockCascadeFeatureRepo{
		GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) {
			return feature, nil
		},
	}
	epicRepo := &mockCascadeEpicRepo{
		GetByIDFunc: func(_ context.Context, _ int64) (*models.Epic, error) {
			return epic, nil
		},
	}
	histQuerier := &mockParentReopenHistoryQuerier{}
	histTx := &mockEntityHistoryTxRecorder{}
	wf := newTestTaskWorkflowService(t)

	deps := cascadeDeps{
		db:             txBeginner,
		featureRepo:    featureRepo,
		epicRepo:       epicRepo,
		historyQuerier: histQuerier,
		historyTx:      histTx,
		workflowSvc:    wf,
	}
	trigger := cascadeTrigger{
		triggerKey:  "E07-F01-003",
		triggerKind: "regression",
		triggerType: models.EntityTypeTask,
		startLeg:    cascadeLegFeature,
		featureID:   featureID,
	}

	cascadeParentReopens(ctx, deps, trigger)

	// No DB writes at all.
	if featureRepo.updateStatusTxCalls != 0 {
		t.Errorf("expected 0 feature updates, got %d", featureRepo.updateStatusTxCalls)
	}
	if epicRepo.updateStatusTxCalls != 0 {
		t.Errorf("expected 0 epic updates, got %d", epicRepo.updateStatusTxCalls)
	}
	if histTx.calls != 0 {
		t.Errorf("expected 0 history rows, got %d", histTx.calls)
	}
	// BeginTx must NOT be called when both ancestors are non-terminal
	// (optimisation: we check after feature leg before opening a tx).
	if txBeginner.beginCalls != 0 {
		t.Errorf("expected 0 BeginTxContext calls for full no-op, got %d", txBeginner.beginCalls)
	}
}

// ============================================================
// AC-06: History row format
// ============================================================

func TestCascade_HistoryRowFormat(t *testing.T) {
	ctx := context.Background()

	featureID := int64(106)
	epicID := int64(206)
	feature := newTestFeature(featureID, epicID, "completed")
	epic := newTestEpic(epicID, "completed")

	txBeginner, _ := newMockTxBeginner()
	featureRepo := &mockCascadeFeatureRepo{
		GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) { return feature, nil },
	}
	epicRepo := &mockCascadeEpicRepo{
		GetByIDFunc: func(_ context.Context, _ int64) (*models.Epic, error) { return epic, nil },
	}
	histQuerier := &mockParentReopenHistoryQuerier{
		GetLastNonTerminalStatusFunc: func(_ context.Context, et models.EntityType, _ int64, _ []string) (string, bool, error) {
			switch et {
			case models.EntityTypeFeature:
				return "in_qa", true, nil
			case models.EntityTypeEpic:
				return "in_development", true, nil
			}
			return "", false, nil
		},
	}
	histTx := &mockEntityHistoryTxRecorder{}
	wf := newTestTaskWorkflowService(t)

	deps := cascadeDeps{
		db: txBeginner, featureRepo: featureRepo, epicRepo: epicRepo,
		historyQuerier: histQuerier, historyTx: histTx, workflowSvc: wf,
	}
	trigger := cascadeTrigger{
		triggerKey: "E07-F01-003", triggerKind: "regression",
		triggerType: models.EntityTypeTask, startLeg: cascadeLegFeature,
		featureID: featureID,
	}

	cascadeParentReopens(ctx, deps, trigger)

	if histTx.calls != 2 {
		t.Fatalf("expected 2 history rows, got %d", histTx.calls)
	}

	// Feature row checks.
	featureRow := histTx.captured[0]
	if featureRow.EntityType != models.EntityTypeFeature {
		t.Errorf("first row should be feature, got %q", featureRow.EntityType)
	}
	if featureRow.FromStatus == nil || *featureRow.FromStatus != "completed" {
		t.Errorf("feature FromStatus should be completed, got %v", featureRow.FromStatus)
	}
	if featureRow.ToStatus != "in_qa" {
		t.Errorf("feature ToStatus should be in_qa, got %q", featureRow.ToStatus)
	}
	if featureRow.Notes == nil || !strings.HasPrefix(*featureRow.Notes, "auto_reopen: triggered by E07-F01-003 regression (task)") {
		t.Errorf("feature notes wrong format: %v", featureRow.Notes)
	}
	if featureRow.ChangedBy == nil || *featureRow.ChangedBy != "system" {
		t.Errorf("feature ChangedBy should be system, got %v", featureRow.ChangedBy)
	}
	// No fallback suffix expected.
	if featureRow.Notes != nil && strings.Contains(*featureRow.Notes, "[fallback:") {
		t.Errorf("unexpected fallback suffix in feature notes: %q", *featureRow.Notes)
	}

	// Epic row checks.
	epicRow := histTx.captured[1]
	if epicRow.EntityType != models.EntityTypeEpic {
		t.Errorf("second row should be epic, got %q", epicRow.EntityType)
	}
	if epicRow.ToStatus != "in_development" {
		t.Errorf("epic ToStatus should be in_development, got %q", epicRow.ToStatus)
	}
	if epicRow.Notes == nil || !strings.HasPrefix(*epicRow.Notes, "auto_reopen: triggered by E07-F01-003 regression (task)") {
		t.Errorf("epic notes wrong format: %v", epicRow.Notes)
	}
}

// TestCascade_HistoryRowFormat_CreationTrigger verifies the creation-trigger
// format of the notes field (separate edge case from AC-06).
func TestCascade_HistoryRowFormat_CreationTrigger(t *testing.T) {
	ctx := context.Background()

	featureID := int64(1061)
	epicID := int64(2061)
	feature := newTestFeature(featureID, epicID, "completed")
	epic := newTestEpic(epicID, "in_development") // non-terminal

	txBeginner, _ := newMockTxBeginner()
	featureRepo := &mockCascadeFeatureRepo{
		GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) { return feature, nil },
	}
	epicRepo := &mockCascadeEpicRepo{
		GetByIDFunc: func(_ context.Context, _ int64) (*models.Epic, error) { return epic, nil },
	}
	histQuerier := &mockParentReopenHistoryQuerier{
		GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, _ []string) (string, bool, error) {
			return "in_qa", true, nil
		},
	}
	histTx := &mockEntityHistoryTxRecorder{}
	wf := newTestTaskWorkflowService(t)

	deps := cascadeDeps{
		db: txBeginner, featureRepo: featureRepo, epicRepo: epicRepo,
		historyQuerier: histQuerier, historyTx: histTx, workflowSvc: wf,
	}
	trigger := cascadeTrigger{
		triggerKey: "E07-F01-005", triggerKind: "creation",
		triggerType: models.EntityTypeTask, startLeg: cascadeLegFeature,
		featureID: featureID,
	}

	cascadeParentReopens(ctx, deps, trigger)

	if histTx.calls != 1 {
		t.Fatalf("expected 1 history row (feature only), got %d", histTx.calls)
	}
	row := histTx.captured[0]
	if row.Notes == nil || !strings.Contains(*row.Notes, "creation (task)") {
		t.Errorf("notes should contain creation trigger, got %v", row.Notes)
	}
}

// ============================================================
// AC-07: Fallback to aggregation status
// ============================================================

func TestResolveReopenTarget_FallbackAggregation(t *testing.T) {
	ctx := context.Background()

	histQuerier := &mockParentReopenHistoryQuerier{
		GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, _ []string) (string, bool, error) {
			return "", false, nil // no history
		},
	}

	// Use a workflow with a designated aggregation status.
	levelWf := &mockLevelWorkflow{
		isTerminalFunc:               func(s string) bool { return s == "completed" || s == "archived" },
		getTerminalStatusesFunc:      func() []string { return []string{"completed", "archived"} },
		primaryAggregationStatusFunc: func() (string, error) { return "active", nil },
		getInitialStatusStringFunc:   func() string { return "draft" },
	}

	status, fallbackKind, err := resolveReopenTarget(ctx, histQuerier, models.EntityTypeEpic, 99, levelWf)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if fallbackKind != "aggregation" {
		t.Errorf("expected fallbackKind=aggregation, got %q", fallbackKind)
	}
	if status != "active" {
		t.Errorf("expected status=%q (designated aggregation status), got %q", "active", status)
	}
}

// TestResolveReopenTarget_HistoryError verifies that an error from the history
// querier is returned (not silently swallowed into a fallback).
func TestResolveReopenTarget_HistoryError(t *testing.T) {
	ctx := context.Background()

	expectedErr := errors.New("simulated DB error")
	histQuerier := &mockParentReopenHistoryQuerier{
		GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, _ []string) (string, bool, error) {
			return "", false, expectedErr
		},
	}
	wfAdapter := &workflowProviderAdapter{svc: newTestEpicWorkflowServiceForBackward(t)}
	levelWf := wfAdapter.ForLevel("epic")

	_, _, err := resolveReopenTarget(ctx, histQuerier, models.EntityTypeEpic, 99, levelWf)
	if err == nil {
		t.Fatal("expected error to be returned, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected wrapped expectedErr, got %v", err)
	}
}

// ============================================================
// AC-08: Fallback to initial status when no aggregation statuses
// ============================================================

func TestResolveReopenTarget_FallbackInitial(t *testing.T) {
	ctx := context.Background()

	histQuerier := &mockParentReopenHistoryQuerier{
		GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, _ []string) (string, bool, error) {
			return "", false, nil
		},
	}

	// Build a mock levelWorkflow with no aggregation statuses.
	levelWf := &mockLevelWorkflow{
		isTerminalFunc:          func(s string) bool { return s == "completed" || s == "archived" },
		getTerminalStatusesFunc: func() []string { return []string{"completed", "archived"} },
		primaryAggregationStatusFunc: func() (string, error) {
			return "", &config.NoCandidateError{Selection: "aggregation (reopen-target)"} // none configured
		},
		getInitialStatusStringFunc: func() string { return "draft" },
	}

	status, fallbackKind, err := resolveReopenTarget(ctx, histQuerier, models.EntityTypeEpic, 99, levelWf)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if fallbackKind != "initial" {
		t.Errorf("expected fallbackKind=initial, got %q", fallbackKind)
	}
	if status != "draft" {
		t.Errorf("expected status=draft (initial), got %q", status)
	}
}

// ============================================================
// AC-09: Idempotent on second regression — no duplicate history rows
// ============================================================

func TestCascade_IdempotentOnSecondRegression(t *testing.T) {
	ctx := context.Background()

	featureID := int64(109)
	epicID := int64(209)

	// First call: both terminal.
	featureTerminal := newTestFeature(featureID, epicID, "completed")
	epicTerminal := newTestEpic(epicID, "completed")

	// After first cascade: non-terminal.
	featureNonTerminal := newTestFeature(featureID, epicID, "in_qa")
	epicNonTerminal := newTestEpic(epicID, "in_development")

	callCount := 0
	featureRepo := &mockCascadeFeatureRepo{
		GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) {
			if callCount == 0 {
				return featureTerminal, nil
			}
			return featureNonTerminal, nil
		},
	}
	epicRepo := &mockCascadeEpicRepo{
		GetByIDFunc: func(_ context.Context, _ int64) (*models.Epic, error) {
			if callCount == 0 {
				return epicTerminal, nil
			}
			return epicNonTerminal, nil
		},
	}

	histQuerier := &mockParentReopenHistoryQuerier{
		GetLastNonTerminalStatusFunc: func(_ context.Context, et models.EntityType, _ int64, _ []string) (string, bool, error) {
			switch et {
			case models.EntityTypeFeature:
				return "in_qa", true, nil
			case models.EntityTypeEpic:
				return "in_development", true, nil
			}
			return "", false, nil
		},
	}
	histTx := &mockEntityHistoryTxRecorder{}
	wf := newTestTaskWorkflowService(t)

	txBeginner, _ := newMockTxBeginner()
	deps := cascadeDeps{
		db: txBeginner, featureRepo: featureRepo, epicRepo: epicRepo,
		historyQuerier: histQuerier, historyTx: histTx, workflowSvc: wf,
	}
	trigger := cascadeTrigger{
		triggerKey: "E07-F01-003", triggerKind: "regression",
		triggerType: models.EntityTypeTask, startLeg: cascadeLegFeature,
		featureID: featureID,
	}

	// First regression.
	cascadeParentReopens(ctx, deps, trigger)
	afterFirst := histTx.calls
	callCount++ // switch to non-terminal state

	// Second regression — ancestors are now non-terminal.
	cascadeParentReopens(ctx, deps, trigger)
	afterSecond := histTx.calls

	if afterFirst != 2 {
		t.Errorf("expected 2 history rows after first regression, got %d", afterFirst)
	}
	if afterSecond != 2 {
		t.Errorf("expected no additional history rows after second regression (idempotent), still want 2, got %d", afterSecond)
	}
}

// ============================================================
// AC-13: Cascade transaction failure is non-blocking
// ============================================================

func TestCascade_TxFailureIsNonBlocking(t *testing.T) {
	ctx := context.Background()

	t.Run("BeginTx_failure", func(t *testing.T) {
		featureID := int64(1130)
		epicID := int64(2130)
		feature := newTestFeature(featureID, epicID, "completed")
		epic := newTestEpic(epicID, "completed")

		txBeginner := &mockTxBeginner{beginErr: errors.New("simulated BeginTx failure")}
		featureRepo := &mockCascadeFeatureRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) { return feature, nil },
		}
		epicRepo := &mockCascadeEpicRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Epic, error) { return epic, nil },
		}
		histTx := &mockEntityHistoryTxRecorder{}
		wf := newTestTaskWorkflowService(t)

		deps := cascadeDeps{
			db: txBeginner, featureRepo: featureRepo, epicRepo: epicRepo,
			historyQuerier: &mockParentReopenHistoryQuerier{
				GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, _ []string) (string, bool, error) {
					return "in_qa", true, nil
				},
			},
			historyTx: histTx, workflowSvc: wf,
		}
		trigger := cascadeTrigger{
			triggerKey: "E07-F01-003", triggerKind: "regression",
			triggerType: models.EntityTypeTask, startLeg: cascadeLegFeature,
			featureID: featureID,
		}

		// Must not panic or return an error — cascade is best-effort.
		// (cascadeParentReopens returns nothing; errors are logged.)
		cascadeParentReopens(ctx, deps, trigger)

		// No DB writes despite terminal ancestors — begin failed.
		if featureRepo.updateStatusTxCalls != 0 {
			t.Errorf("expected 0 feature updates after BeginTx failure, got %d", featureRepo.updateStatusTxCalls)
		}
		if histTx.calls != 0 {
			t.Errorf("expected 0 history rows after BeginTx failure, got %d", histTx.calls)
		}
	})

	t.Run("UpdateStatusTx_feature_failure", func(t *testing.T) {
		featureID := int64(1131)
		epicID := int64(2131)
		feature := newTestFeature(featureID, epicID, "completed")
		epic := newTestEpic(epicID, "completed")

		txBeginner, _ := newMockTxBeginner()
		featureRepo := &mockCascadeFeatureRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) { return feature, nil },
			UpdateStatusTxFunc: func(_ context.Context, _ *sql.Tx, _ int64, _ string, _ *string, _ *string) error {
				return errors.New("simulated feature update failure")
			},
		}
		epicRepo := &mockCascadeEpicRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Epic, error) { return epic, nil },
		}
		histTx := &mockEntityHistoryTxRecorder{}
		wf := newTestTaskWorkflowService(t)

		deps := cascadeDeps{
			db: txBeginner, featureRepo: featureRepo, epicRepo: epicRepo,
			historyQuerier: &mockParentReopenHistoryQuerier{
				GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, _ []string) (string, bool, error) {
					return "in_qa", true, nil
				},
			},
			historyTx: histTx, workflowSvc: wf,
		}
		trigger := cascadeTrigger{
			triggerKey: "E07-F01-003", triggerKind: "regression",
			triggerType: models.EntityTypeTask, startLeg: cascadeLegFeature,
			featureID: featureID,
		}

		// Must not panic.
		cascadeParentReopens(ctx, deps, trigger)

		// Epic must NOT have been updated (stopped after feature failure).
		if epicRepo.updateStatusTxCalls != 0 {
			t.Errorf("expected 0 epic updates after feature update failure, got %d", epicRepo.updateStatusTxCalls)
		}
		if histTx.calls != 0 {
			t.Errorf("expected 0 history rows after feature update failure, got %d", histTx.calls)
		}
	})

	// ---- new subtests added to close F-01 ----

	t.Run("Feature_history_write_failure", func(t *testing.T) {
		// cascade_reopen.go:279 — CreateTx error on the feature history row.
		featureID := int64(1132)
		epicID := int64(2132)
		feature := newTestFeature(featureID, epicID, "completed")
		epic := newTestEpic(epicID, "completed")

		txBeginner, _ := newMockTxBeginner()
		featureRepo := &mockCascadeFeatureRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) { return feature, nil },
			// UpdateStatusTx succeeds — the error fires on the subsequent history write.
		}
		epicRepo := &mockCascadeEpicRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Epic, error) { return epic, nil },
		}
		histTx := &mockEntityHistoryTxRecorder{
			CreateTxFunc: func(_ context.Context, _ *sql.Tx, h *models.EntityHistory) error {
				if h.EntityType == models.EntityTypeFeature {
					return errors.New("simulated feature history write failure")
				}
				return nil
			},
		}
		wf := newTestTaskWorkflowService(t)

		deps := cascadeDeps{
			db: txBeginner, featureRepo: featureRepo, epicRepo: epicRepo,
			historyQuerier: &mockParentReopenHistoryQuerier{
				GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, _ []string) (string, bool, error) {
					return "in_qa", true, nil
				},
			},
			historyTx: histTx, workflowSvc: wf,
		}
		trigger := cascadeTrigger{
			triggerKey: "E07-F01-003", triggerKind: "regression",
			triggerType: models.EntityTypeTask, startLeg: cascadeLegFeature,
			featureID: featureID,
		}

		// Must not panic — cascade is best-effort.
		cascadeParentReopens(ctx, deps, trigger)

		// The cascade returns after the feature history write fails; epic must not be touched.
		if epicRepo.updateStatusTxCalls != 0 {
			t.Errorf("expected 0 epic updates after feature history write failure, got %d", epicRepo.updateStatusTxCalls)
		}
	})

	t.Run("Feature_refetch_failure_inside_tx", func(t *testing.T) {
		// cascade_reopen.go:249 — second GetByID on feature repo inside tx fails.
		featureID := int64(1133)
		epicID := int64(2133)
		feature := newTestFeature(featureID, epicID, "completed")
		epic := newTestEpic(epicID, "completed")

		calls := 0
		txBeginner, _ := newMockTxBeginner()
		featureRepo := &mockCascadeFeatureRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) {
				calls++
				if calls == 1 {
					// First call (outer phase pre-flight) succeeds.
					return feature, nil
				}
				// Second call (in-tx re-fetch) fails.
				return nil, errors.New("simulated feature re-fetch failure")
			},
		}
		epicRepo := &mockCascadeEpicRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Epic, error) { return epic, nil },
		}
		histTx := &mockEntityHistoryTxRecorder{}
		wf := newTestTaskWorkflowService(t)

		deps := cascadeDeps{
			db: txBeginner, featureRepo: featureRepo, epicRepo: epicRepo,
			historyQuerier: &mockParentReopenHistoryQuerier{
				GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, _ []string) (string, bool, error) {
					return "in_qa", true, nil
				},
			},
			historyTx: histTx, workflowSvc: wf,
		}
		trigger := cascadeTrigger{
			triggerKey: "E07-F01-003", triggerKind: "regression",
			triggerType: models.EntityTypeTask, startLeg: cascadeLegFeature,
			featureID: featureID,
		}

		cascadeParentReopens(ctx, deps, trigger)

		// Cascade must return immediately; no DB writes.
		if featureRepo.updateStatusTxCalls != 0 {
			t.Errorf("expected 0 feature updates after in-tx re-fetch failure, got %d", featureRepo.updateStatusTxCalls)
		}
		if histTx.calls != 0 {
			t.Errorf("expected 0 history rows after in-tx re-fetch failure, got %d", histTx.calls)
		}
	})

	t.Run("Epic_update_failure", func(t *testing.T) {
		// cascade_reopen.go:305 — UpdateStatusTx error on the epic leg.
		featureID := int64(1134)
		epicID := int64(2134)
		feature := newTestFeature(featureID, epicID, "completed")
		epic := newTestEpic(epicID, "completed")

		txBeginner, _ := newMockTxBeginner()
		featureRepo := &mockCascadeFeatureRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) { return feature, nil },
			// Feature update and history write succeed.
		}
		epicRepo := &mockCascadeEpicRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Epic, error) { return epic, nil },
			UpdateStatusTxFunc: func(_ context.Context, _ *sql.Tx, _ int64, _ string, _ *string, _ *string) error {
				return errors.New("simulated epic update failure")
			},
		}
		histTx := &mockEntityHistoryTxRecorder{}
		wf := newTestTaskWorkflowService(t)

		deps := cascadeDeps{
			db: txBeginner, featureRepo: featureRepo, epicRepo: epicRepo,
			historyQuerier: &mockParentReopenHistoryQuerier{
				GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, _ []string) (string, bool, error) {
					return "in_qa", true, nil
				},
			},
			historyTx: histTx, workflowSvc: wf,
		}
		trigger := cascadeTrigger{
			triggerKey: "E07-F01-003", triggerKind: "regression",
			triggerType: models.EntityTypeTask, startLeg: cascadeLegFeature,
			featureID: featureID,
		}

		cascadeParentReopens(ctx, deps, trigger)

		// Feature was updated (epic failure happens after).
		if featureRepo.updateStatusTxCalls != 1 {
			t.Errorf("expected 1 feature update before epic failure, got %d", featureRepo.updateStatusTxCalls)
		}
		// No epic history row written.
		epicHistoryRows := 0
		for _, h := range histTx.captured {
			if h.EntityType == models.EntityTypeEpic {
				epicHistoryRows++
			}
		}
		if epicHistoryRows != 0 {
			t.Errorf("expected 0 epic history rows after epic update failure, got %d", epicHistoryRows)
		}
	})

	t.Run("Epic_history_write_failure", func(t *testing.T) {
		// cascade_reopen.go:315 — CreateTx error on the epic history row.
		featureID := int64(1135)
		epicID := int64(2135)
		feature := newTestFeature(featureID, epicID, "completed")
		epic := newTestEpic(epicID, "completed")

		txBeginner, _ := newMockTxBeginner()
		featureRepo := &mockCascadeFeatureRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) { return feature, nil },
		}
		epicRepo := &mockCascadeEpicRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Epic, error) { return epic, nil },
			// UpdateStatusTx succeeds; error fires on the subsequent history write.
		}
		histTx := &mockEntityHistoryTxRecorder{
			CreateTxFunc: func(_ context.Context, _ *sql.Tx, h *models.EntityHistory) error {
				if h.EntityType == models.EntityTypeEpic {
					return errors.New("simulated epic history write failure")
				}
				return nil
			},
		}
		wf := newTestTaskWorkflowService(t)

		deps := cascadeDeps{
			db: txBeginner, featureRepo: featureRepo, epicRepo: epicRepo,
			historyQuerier: &mockParentReopenHistoryQuerier{
				GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, _ []string) (string, bool, error) {
					return "in_qa", true, nil
				},
			},
			historyTx: histTx, workflowSvc: wf,
		}
		trigger := cascadeTrigger{
			triggerKey: "E07-F01-003", triggerKind: "regression",
			triggerType: models.EntityTypeTask, startLeg: cascadeLegFeature,
			featureID: featureID,
		}

		cascadeParentReopens(ctx, deps, trigger)

		// The feature history row was written successfully.
		// The epic CreateTx was called (the mock captures before returning error) but returned an error,
		// causing the cascade to return — this is the path we want to exercise.
		// We verify the write attempt count via histTx.calls: feature (1) + epic attempt (1) = 2.
		if histTx.calls != 2 {
			t.Errorf("expected 2 CreateTx calls (feature success + epic attempt), got %d", histTx.calls)
		}
		// Confirm there is exactly one successful feature row.
		featureHistoryRows := 0
		for _, h := range histTx.captured {
			if h.EntityType == models.EntityTypeFeature {
				featureHistoryRows++
			}
		}
		if featureHistoryRows != 1 {
			t.Errorf("expected 1 feature history row, got %d", featureHistoryRows)
		}
	})

	t.Run("Epic_refetch_failure_inside_tx", func(t *testing.T) {
		// cascade_reopen.go:293 — second GetByID on epic repo inside tx fails.
		featureID := int64(1136)
		epicID := int64(2136)
		feature := newTestFeature(featureID, epicID, "completed")
		epic := newTestEpic(epicID, "completed")

		epicCalls := 0
		txBeginner, _ := newMockTxBeginner()
		featureRepo := &mockCascadeFeatureRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) { return feature, nil },
		}
		epicRepo := &mockCascadeEpicRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Epic, error) {
				epicCalls++
				if epicCalls == 1 {
					// First call (outer phase pre-flight) succeeds.
					return epic, nil
				}
				// Second call (in-tx re-fetch) fails.
				return nil, errors.New("simulated epic re-fetch failure")
			},
		}
		histTx := &mockEntityHistoryTxRecorder{}
		wf := newTestTaskWorkflowService(t)

		deps := cascadeDeps{
			db: txBeginner, featureRepo: featureRepo, epicRepo: epicRepo,
			historyQuerier: &mockParentReopenHistoryQuerier{
				GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, _ []string) (string, bool, error) {
					return "in_qa", true, nil
				},
			},
			historyTx: histTx, workflowSvc: wf,
		}
		trigger := cascadeTrigger{
			triggerKey: "E07-F01-003", triggerKind: "regression",
			triggerType: models.EntityTypeTask, startLeg: cascadeLegFeature,
			featureID: featureID,
		}

		cascadeParentReopens(ctx, deps, trigger)

		// Epic re-fetch failed inside tx; no epic writes expected.
		if epicRepo.updateStatusTxCalls != 0 {
			t.Errorf("expected 0 epic updates after in-tx re-fetch failure, got %d", epicRepo.updateStatusTxCalls)
		}
		epicHistoryRows := 0
		for _, h := range histTx.captured {
			if h.EntityType == models.EntityTypeEpic {
				epicHistoryRows++
			}
		}
		if epicHistoryRows != 0 {
			t.Errorf("expected 0 epic history rows after epic re-fetch failure, got %d", epicHistoryRows)
		}
	})

	t.Run("Commit_failure", func(t *testing.T) {
		// REQ-N-006 / U-03: commit failure path must be exercised.
		//
		// mockTxBeginner returns (nil, nil) so the cascade's `if tx != nil` block
		// is skipped. To actually reach tx.Commit(), we wire cascadeDeps.commitFn
		// with a function that records the call and returns an error. The cascade
		// treats commitFn as the commit hook; when nil it falls back to tx.Commit().
		// This approach exercises the slog.Warn("cascade: commit failed") branch
		// without needing a real *sql.Tx.
		featureID := int64(1137)
		epicID := int64(2137)
		feature := newTestFeature(featureID, epicID, "completed")
		epic := newTestEpic(epicID, "completed")

		commitCalled := false
		commitErr := errors.New("simulated commit failure")

		// Use a txBeginner that returns non-nil tx so commitFn is reached.
		// Since *sql.Tx cannot be constructed directly, we use a sentinel approach:
		// commitFn is called regardless of the tx value when deps.commitFn is set.
		txBeginner, _ := newMockTxBeginner()

		featureRepo := &mockCascadeFeatureRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) { return feature, nil },
		}
		epicRepo := &mockCascadeEpicRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Epic, error) { return epic, nil },
		}
		histTx := &mockEntityHistoryTxRecorder{}
		wf := newTestTaskWorkflowService(t)

		deps := cascadeDeps{
			db: txBeginner, featureRepo: featureRepo, epicRepo: epicRepo,
			historyQuerier: &mockParentReopenHistoryQuerier{
				GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, _ []string) (string, bool, error) {
					return "in_qa", true, nil
				},
			},
			historyTx:   histTx,
			workflowSvc: wf,
			// commitFn intercepts the commit path and returns a failure.
			// This exercises the slog.Warn("cascade: commit failed") branch.
			commitFn: func(_ *sql.Tx) error {
				commitCalled = true
				return commitErr
			},
		}
		trigger := cascadeTrigger{
			triggerKey: "E07-F01-003", triggerKind: "regression",
			triggerType: models.EntityTypeTask, startLeg: cascadeLegFeature,
			featureID: featureID,
		}

		// Must not panic — cascade is best-effort.
		cascadeParentReopens(ctx, deps, trigger)

		// commitFn must have been called (proving the commit path was exercised).
		if !commitCalled {
			t.Error("expected commitFn to be called (commit path not exercised)")
		}

		// The writes reached the mock repos, but the commit failed. This is correct
		// best-effort behaviour — the cascade logs a WARN and returns without panicking.
		if featureRepo.updateStatusTxCalls != 1 {
			t.Errorf("expected feature UpdateStatusTx to have been called once before commit, got %d", featureRepo.updateStatusTxCalls)
		}
		if epicRepo.updateStatusTxCalls != 1 {
			t.Errorf("expected epic UpdateStatusTx to have been called once before commit, got %d", epicRepo.updateStatusTxCalls)
		}
	})
}

// ============================================================
// Outer-phase lookup failure subtests (lines 166 and 177)
// ============================================================

func TestCascade_OuterLookupFailureIsNonBlocking(t *testing.T) {
	ctx := context.Background()

	t.Run("Initial_feature_lookup_failure", func(t *testing.T) {
		// cascade_reopen.go:166 — the very first GetByID on the feature repo returns an error.
		featureID := int64(1140)

		txBeginner, _ := newMockTxBeginner()
		featureRepo := &mockCascadeFeatureRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) {
				return nil, errors.New("simulated outer feature lookup failure")
			},
		}
		epicRepo := &mockCascadeEpicRepo{}
		histTx := &mockEntityHistoryTxRecorder{}
		wf := newTestTaskWorkflowService(t)

		deps := cascadeDeps{
			db: txBeginner, featureRepo: featureRepo, epicRepo: epicRepo,
			historyQuerier: &mockParentReopenHistoryQuerier{},
			historyTx:      histTx, workflowSvc: wf,
		}
		trigger := cascadeTrigger{
			triggerKey: "E07-F01-003", triggerKind: "regression",
			triggerType: models.EntityTypeTask, startLeg: cascadeLegFeature,
			featureID: featureID,
		}

		// Must not panic.
		cascadeParentReopens(ctx, deps, trigger)

		// Nothing must have been written.
		if featureRepo.updateStatusTxCalls != 0 {
			t.Errorf("expected 0 feature updates after outer feature lookup failure, got %d", featureRepo.updateStatusTxCalls)
		}
		if epicRepo.updateStatusTxCalls != 0 {
			t.Errorf("expected 0 epic updates after outer feature lookup failure, got %d", epicRepo.updateStatusTxCalls)
		}
		if histTx.calls != 0 {
			t.Errorf("expected 0 history rows after outer feature lookup failure, got %d", histTx.calls)
		}
		if txBeginner.beginCalls != 0 {
			t.Errorf("expected 0 BeginTxContext calls after outer feature lookup failure, got %d", txBeginner.beginCalls)
		}
	})

	t.Run("Initial_epic_lookup_failure", func(t *testing.T) {
		// cascade_reopen.go:177 — GetByID on the epic repo returns an error in the outer phase.
		featureID := int64(1141)
		epicID := int64(2141)
		// Feature is non-terminal so we reach the epic lookup even in the feature-leg path.
		// Use cascadeLegEpic (feature trigger) — the feature is looked up only for EpicID,
		// and featureNeedsReopen is false, so we always proceed to the epic lookup.
		feature := newTestFeature(featureID, epicID, "in_qa") // non-terminal: featureNeedsReopen = false

		txBeginner, _ := newMockTxBeginner()
		featureRepo := &mockCascadeFeatureRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) { return feature, nil },
		}
		epicRepo := &mockCascadeEpicRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Epic, error) {
				return nil, errors.New("simulated outer epic lookup failure")
			},
		}
		histTx := &mockEntityHistoryTxRecorder{}
		wf := newTestTaskWorkflowService(t)

		deps := cascadeDeps{
			db: txBeginner, featureRepo: featureRepo, epicRepo: epicRepo,
			historyQuerier: &mockParentReopenHistoryQuerier{},
			historyTx:      histTx, workflowSvc: wf,
		}
		trigger := cascadeTrigger{
			triggerKey: "E07-F01", triggerKind: "regression",
			triggerType: models.EntityTypeFeature, startLeg: cascadeLegEpic,
			featureID: featureID,
		}

		cascadeParentReopens(ctx, deps, trigger)

		if epicRepo.updateStatusTxCalls != 0 {
			t.Errorf("expected 0 epic updates after outer epic lookup failure, got %d", epicRepo.updateStatusTxCalls)
		}
		if histTx.calls != 0 {
			t.Errorf("expected 0 history rows after outer epic lookup failure, got %d", histTx.calls)
		}
		if txBeginner.beginCalls != 0 {
			t.Errorf("expected 0 BeginTxContext calls after outer epic lookup failure, got %d", txBeginner.beginCalls)
		}
	})
}

// ============================================================
// Helper: mock level workflow (for AC-08 fallback initial test)
// ============================================================

type mockLevelWorkflow struct {
	isTerminalFunc               func(string) bool
	getTerminalStatusesFunc      func() []string
	primaryAggregationStatusFunc func() (string, error)
	getInitialStatusStringFunc   func() string
}

func (m *mockLevelWorkflow) IsTerminalStatus(status string) bool {
	if m.isTerminalFunc != nil {
		return m.isTerminalFunc(status)
	}
	return false
}

func (m *mockLevelWorkflow) GetTerminalStatuses() []string {
	if m.getTerminalStatusesFunc != nil {
		return m.getTerminalStatusesFunc()
	}
	return nil
}

func (m *mockLevelWorkflow) PrimaryAggregationStatus() (string, error) {
	if m.primaryAggregationStatusFunc != nil {
		return m.primaryAggregationStatusFunc()
	}
	return "", &config.NoCandidateError{Selection: "aggregation (reopen-target)"}
}

func (m *mockLevelWorkflow) GetInitialStatusString() string {
	if m.getInitialStatusStringFunc != nil {
		return m.getInitialStatusStringFunc()
	}
	return ""
}

// ============================================================
// Additional helper: task-level workflow service for tests
// ============================================================

func newTestTaskWorkflowService(t *testing.T) levelWorkflowProvider {
	t.Helper()
	return &workflowProviderAdapter{svc: newTestEpicWorkflowServiceForBackward(t)}
}

// ============================================================
// Nil ChangedAt population test (per UAT observation 4.5)
// ============================================================

func TestCascade_HistoryRowHasChangedAt(t *testing.T) {
	ctx := context.Background()
	before := time.Now()

	featureID := int64(110)
	epicID := int64(210)
	feature := newTestFeature(featureID, epicID, "completed")
	epic := newTestEpic(epicID, "in_development")

	txBeginner, _ := newMockTxBeginner()
	featureRepo := &mockCascadeFeatureRepo{
		GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) { return feature, nil },
	}
	epicRepo := &mockCascadeEpicRepo{
		GetByIDFunc: func(_ context.Context, _ int64) (*models.Epic, error) { return epic, nil },
	}
	histQuerier := &mockParentReopenHistoryQuerier{
		GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, _ []string) (string, bool, error) {
			return "in_qa", true, nil
		},
	}
	histTx := &mockEntityHistoryTxRecorder{}
	wf := newTestTaskWorkflowService(t)

	deps := cascadeDeps{
		db: txBeginner, featureRepo: featureRepo, epicRepo: epicRepo,
		historyQuerier: histQuerier, historyTx: histTx, workflowSvc: wf,
	}
	trigger := cascadeTrigger{
		triggerKey: "E07-F01-003", triggerKind: "regression",
		triggerType: models.EntityTypeTask, startLeg: cascadeLegFeature,
		featureID: featureID,
	}

	cascadeParentReopens(ctx, deps, trigger)

	after := time.Now()
	if histTx.calls == 0 {
		t.Fatal("expected at least one history row")
	}
	row := histTx.captured[0]
	if row.ChangedAt.IsZero() {
		t.Error("expected ChangedAt to be set, got zero time")
	}
	if row.ChangedAt.Before(before) || row.ChangedAt.After(after) {
		t.Errorf("ChangedAt %v not in range [%v, %v]", row.ChangedAt, before, after)
	}
}

func TestCascade_RegressionReopensForwardAdvancedParentsFromHistory(t *testing.T) {
	ctx := context.Background()

	featureID := int64(310)
	epicID := int64(410)
	feature := newTestFeature(featureID, epicID, "completed")
	epic := newTestEpic(epicID, "completed")

	txBeginner, _ := newMockTxBeginner()
	featureRepo := &mockCascadeFeatureRepo{
		GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) { return feature, nil },
		GetByIDTxFunc: func(_ context.Context, _ *sql.Tx, _ int64) (*models.Feature, error) {
			return feature, nil
		},
	}
	epicRepo := &mockCascadeEpicRepo{
		GetByIDFunc: func(_ context.Context, _ int64) (*models.Epic, error) { return epic, nil },
		GetByIDTxFunc: func(_ context.Context, _ *sql.Tx, _ int64) (*models.Epic, error) {
			return epic, nil
		},
	}
	histQuerier := &mockParentReopenHistoryQuerier{
		GetLastNonTerminalStatusFunc: func(_ context.Context, entityType models.EntityType, entityID int64, terminalStatuses []string) (string, bool, error) {
			switch entityType {
			case models.EntityTypeFeature:
				return "ready_for_code_review", true, nil
			case models.EntityTypeEpic:
				return "ready_for_release", true, nil
			default:
				return "", false, fmt.Errorf("unexpected entity type %q", entityType)
			}
		},
	}
	histTx := &mockEntityHistoryTxRecorder{}
	deps := cascadeDeps{
		db:             txBeginner,
		featureRepo:    featureRepo,
		epicRepo:       epicRepo,
		historyQuerier: histQuerier,
		historyTx:      histTx,
		workflowSvc:    newTestTaskWorkflowService(t),
	}
	trigger := cascadeTrigger{
		triggerKey:  "E07-F44-001",
		triggerKind: "regression",
		triggerType: models.EntityTypeTask,
		startLeg:    cascadeLegFeature,
		featureID:   featureID,
	}

	cascadeParentReopens(ctx, deps, trigger)

	if featureRepo.lastUpdateStatus != "ready_for_code_review" {
		t.Fatalf("expected feature reopen target ready_for_code_review, got %q", featureRepo.lastUpdateStatus)
	}
	if epicRepo.lastUpdateStatus != "ready_for_release" {
		t.Fatalf("expected epic reopen target ready_for_release, got %q", epicRepo.lastUpdateStatus)
	}
	if len(histTx.captured) != 2 {
		t.Fatalf("expected 2 history rows, got %d", len(histTx.captured))
	}
	if histTx.captured[0].ToStatus != "ready_for_code_review" {
		t.Fatalf("expected feature history to_status ready_for_code_review, got %q", histTx.captured[0].ToStatus)
	}
	if histTx.captured[1].ToStatus != "ready_for_release" {
		t.Fatalf("expected epic history to_status ready_for_release, got %q", histTx.captured[1].ToStatus)
	}
}

// ============================================================
// Empty terminal-set guard (per UAT observation 4.3)
// ============================================================

func TestCascade_EmptyTerminalSetGuard(t *testing.T) {
	// The cascade must NEVER call historyQuerier with an empty terminalStatuses slice
	// when computing the reopen target. The workflow service must return a non-empty
	// terminal set for the guard to be meaningful.
	//
	// We verify this by using a real workflow service whose terminal set is non-empty
	// and asserting that the GetLastNonTerminalStatus call receives at least one entry.
	ctx := context.Background()

	featureID := int64(111)
	epicID := int64(211)
	feature := newTestFeature(featureID, epicID, "completed")
	epic := newTestEpic(epicID, "completed")

	var capturedTerminalStatuses []string
	txBeginner, _ := newMockTxBeginner()
	featureRepo := &mockCascadeFeatureRepo{
		GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) { return feature, nil },
	}
	epicRepo := &mockCascadeEpicRepo{
		GetByIDFunc: func(_ context.Context, _ int64) (*models.Epic, error) { return epic, nil },
	}
	histQuerier := &mockParentReopenHistoryQuerier{
		GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, terminalStatuses []string) (string, bool, error) {
			capturedTerminalStatuses = terminalStatuses
			return "in_qa", true, nil
		},
	}
	histTx := &mockEntityHistoryTxRecorder{}
	wf := newTestTaskWorkflowService(t)

	deps := cascadeDeps{
		db: txBeginner, featureRepo: featureRepo, epicRepo: epicRepo,
		historyQuerier: histQuerier, historyTx: histTx, workflowSvc: wf,
	}
	trigger := cascadeTrigger{
		triggerKey: "E07-F01-003", triggerKind: "regression",
		triggerType: models.EntityTypeTask, startLeg: cascadeLegFeature,
		featureID: featureID,
	}

	cascadeParentReopens(ctx, deps, trigger)

	if len(capturedTerminalStatuses) == 0 {
		t.Error("empty terminal-set guard violated: GetLastNonTerminalStatus called with empty terminalStatuses")
	}
}

// ============================================================
// Concurrent cascade idempotency (U-01 / AC-T2 / REQ-F-008)
// ============================================================

// TestCascade_ConcurrentCascadeOnSameAncestorWritesExactlyOneHistoryRow verifies
// that when two cascades fire concurrently against the same terminal ancestor, the
// GetByIDTx in-tx re-fetch makes exactly one of them a no-op (idempotent skip) so
// only a single history row is written.
//
// Because *sql.Tx cannot be constructed in tests and the mock repos accept nil tx,
// we simulate the race at the mock level: the first cascade's in-tx re-fetch still
// sees "completed" (terminal) and proceeds; the second cascade's in-tx re-fetch sees
// "in_qa" (non-terminal, as if the first cascade had already committed) and skips.
// This proves the code path that prevents the duplicate write is exercised.
func TestCascade_ConcurrentCascadeOnSameAncestorWritesExactlyOneHistoryRow(t *testing.T) {
	ctx := context.Background()

	featureID := int64(500)
	epicID := int64(600)

	// Shared history recorder — we count rows to assert idempotency.
	var mu gosync.Mutex
	histTx := &mockEntityHistoryTxRecorder{}

	wf := newTestTaskWorkflowService(t)
	trigger := cascadeTrigger{
		triggerKey:  "E07-F01-003",
		triggerKind: "regression",
		triggerType: models.EntityTypeTask,
		startLeg:    cascadeLegFeature,
		featureID:   featureID,
	}

	// Simulate two concurrent cascades: the first proceeds (in-tx re-fetch sees
	// terminal), the second is a no-op (in-tx re-fetch sees non-terminal because
	// the first has already "committed").
	//
	// We implement this by giving each cascade its own mock repos. Cascade 1's
	// GetByIDTx returns "completed" (still terminal). Cascade 2's GetByIDTx
	// returns "in_qa" (non-terminal — simulating that cascade 1 already committed).

	makeFeatureRepo := func(inTxStatus string) *mockCascadeFeatureRepo {
		feature := newTestFeature(featureID, epicID, "completed")
		freshFeature := newTestFeature(featureID, epicID, inTxStatus)
		return &mockCascadeFeatureRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Feature, error) {
				return feature, nil
			},
			GetByIDTxFunc: func(_ context.Context, _ *sql.Tx, _ int64) (*models.Feature, error) {
				return freshFeature, nil
			},
		}
	}

	sharedHistTx := &mockEntityHistoryTxRecorder{
		CreateTxFunc: func(ctx context.Context, tx *sql.Tx, history *models.EntityHistory) error {
			mu.Lock()
			defer mu.Unlock()
			h := *history
			histTx.captured = append(histTx.captured, &h)
			histTx.calls++
			return nil
		},
	}

	// epicRepo and histQuerier are constructed inside makeDeps so each goroutine
	// gets its own instance. This eliminates the data races on the unsynchronized
	// call-counter fields (getByIDCalls, calls) that would otherwise be mutated
	// concurrently by both goroutines. (BUG-QA-001)
	makeDeps := func(featureRepo *mockCascadeFeatureRepo) cascadeDeps {
		epicRepo := &mockCascadeEpicRepo{
			GetByIDFunc: func(_ context.Context, _ int64) (*models.Epic, error) {
				return newTestEpic(epicID, "in_development"), nil // non-terminal
			},
		}
		histQuerier := &mockParentReopenHistoryQuerier{
			GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, _ []string) (string, bool, error) {
				return "in_qa", true, nil
			},
		}
		return cascadeDeps{
			db:             &mockTxBeginner{},
			featureRepo:    featureRepo,
			epicRepo:       epicRepo,
			historyQuerier: histQuerier,
			historyTx:      sharedHistTx,
			workflowSvc:    wf,
		}
	}

	var wg gosync.WaitGroup
	wg.Add(2)

	// Cascade 1: in-tx re-fetch still sees terminal → proceeds to write.
	go func() {
		defer wg.Done()
		cascadeParentReopens(ctx, makeDeps(makeFeatureRepo("completed")), trigger)
	}()

	// Cascade 2: in-tx re-fetch sees non-terminal (simulating C1 already committed)
	// → idempotent skip, no history row for feature.
	go func() {
		defer wg.Done()
		cascadeParentReopens(ctx, makeDeps(makeFeatureRepo("in_qa")), trigger)
	}()

	wg.Wait()

	mu.Lock()
	featureRows := 0
	for _, h := range histTx.captured {
		if h.EntityType == models.EntityTypeFeature {
			featureRows++
		}
	}
	mu.Unlock()

	if featureRows != 1 {
		t.Errorf("expected exactly 1 feature history row under concurrent cascades (idempotency), got %d", featureRows)
	}
}

// ============================================================
// resolveReopenTarget — empty terminal set guard (U-05 direct test)
// ============================================================

// TestResolveReopenTarget_EmptyTerminalSetFallsBackToAggregation verifies that when
// the workflow profile returns an empty terminal set, resolveReopenTarget does NOT
// call GetLastNonTerminalStatus (which would issue a vacuous NOT IN () query) and
// instead falls through to the aggregation fallback.
func TestResolveReopenTarget_EmptyTerminalSetFallsBackToAggregation(t *testing.T) {
	ctx := context.Background()

	querier := &mockParentReopenHistoryQuerier{
		GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, _ []string) (string, bool, error) {
			t.Error("GetLastNonTerminalStatus must NOT be called when terminal set is empty")
			return "", false, nil
		},
	}

	levelWf := &mockLevelWorkflow{
		getTerminalStatusesFunc: func() []string { return nil }, // empty
		primaryAggregationStatusFunc: func() (string, error) {
			return "in_development", nil
		},
		getInitialStatusStringFunc: func() string { return "draft" },
	}

	target, fallback, err := resolveReopenTarget(ctx, querier, models.EntityTypeFeature, 42, levelWf)
	if err != nil {
		t.Fatalf("resolveReopenTarget returned unexpected error: %v", err)
	}
	if target != "in_development" {
		t.Errorf("expected aggregation fallback target %q, got %q", "in_development", target)
	}
	if fallback != "aggregation" {
		t.Errorf("expected fallback kind %q, got %q", "aggregation", fallback)
	}
}

// TestResolveReopenTarget_EmptyTerminalSetFallsBackToInitial verifies that when
// both the terminal set and aggregation statuses are empty, the initial status is used.
func TestResolveReopenTarget_EmptyTerminalSetFallsBackToInitial(t *testing.T) {
	ctx := context.Background()

	querier := &mockParentReopenHistoryQuerier{
		GetLastNonTerminalStatusFunc: func(_ context.Context, _ models.EntityType, _ int64, _ []string) (string, bool, error) {
			t.Error("GetLastNonTerminalStatus must NOT be called when terminal set is empty")
			return "", false, nil
		},
	}

	levelWf := &mockLevelWorkflow{
		getTerminalStatusesFunc: func() []string { return []string{} }, // empty slice
		primaryAggregationStatusFunc: func() (string, error) {
			return "", &config.NoCandidateError{Selection: "aggregation (reopen-target)"}
		},
		getInitialStatusStringFunc: func() string { return "draft" },
	}

	target, fallback, err := resolveReopenTarget(ctx, querier, models.EntityTypeEpic, 99, levelWf)
	if err != nil {
		t.Fatalf("resolveReopenTarget returned unexpected error: %v", err)
	}
	if target != "draft" {
		t.Errorf("expected initial fallback target %q, got %q", "draft", target)
	}
	if fallback != "initial" {
		t.Errorf("expected fallback kind %q, got %q", "initial", fallback)
	}
}

// ============================================================
// buildAutoReopenNotes helper tests
// ============================================================

// ============================================================
// AC-11: Bug status transitions do not trigger cascade reopens
// ============================================================

// TestCascade_BugDoesNotTriggerCascade verifies that BugService has no cascade
// reopen infrastructure. Bugs are standalone entities unrelated to the
// epic→feature→task hierarchy, so no cascade hook should ever fire for them.
//
// This is a structural test: it confirms that BugService has none of the cascade
// dependency fields (cascadeDB, cascadeFeatureRepo, cascadeEpicRepo,
// cascadeHistQuerier, cascadeHistTx) and exposes no SetCascadeDeps method.
// The reflect loop exhaustively checks field names so that future accidental
// additions are caught automatically.
func TestCascade_BugDoesNotTriggerCascade(t *testing.T) {
	bugServiceType := reflect.TypeOf(BugService{})

	cascadeFieldNames := []string{
		"cascadeDB",
		"cascadeFeatureRepo",
		"cascadeEpicRepo",
		"cascadeHistQuerier",
		"cascadeHistTx",
	}

	for _, fieldName := range cascadeFieldNames {
		if _, ok := bugServiceType.FieldByName(fieldName); ok {
			t.Errorf("BugService must not have cascade field %q — bugs are standalone entities and must never trigger parent cascade reopens", fieldName)
		}
	}

	// Also verify there is no SetCascadeDeps method on BugService or *BugService.
	bugServicePtrType := reflect.TypeOf(&BugService{})
	if _, ok := bugServiceType.MethodByName("SetCascadeDeps"); ok {
		t.Error("BugService must not have a SetCascadeDeps method")
	}
	if _, ok := bugServicePtrType.MethodByName("SetCascadeDeps"); ok {
		t.Error("*BugService must not have a SetCascadeDeps method")
	}
}

func TestBuildAutoReopenNotes(t *testing.T) {
	tests := []struct {
		name            string
		trigger         cascadeTrigger
		fallbackKind    string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "regression_no_fallback",
			trigger: cascadeTrigger{
				triggerKey: "E07-F01-003", triggerKind: "regression",
				triggerType: models.EntityTypeTask,
			},
			fallbackKind:    "",
			wantContains:    []string{"auto_reopen:", "E07-F01-003", "regression", "(task)"},
			wantNotContains: []string{"[fallback:"},
		},
		{
			name: "creation_with_fallback_aggregation",
			trigger: cascadeTrigger{
				triggerKey: "E07-F01-005", triggerKind: "creation",
				triggerType: models.EntityTypeTask,
			},
			fallbackKind: "aggregation",
			wantContains: []string{"auto_reopen:", "E07-F01-005", "creation", "[fallback: aggregation]"},
		},
		{
			name: "regression_with_fallback_initial",
			trigger: cascadeTrigger{
				triggerKey: "E07-F02", triggerKind: "regression",
				triggerType: models.EntityTypeFeature,
			},
			fallbackKind: "initial",
			wantContains: []string{"auto_reopen:", "E07-F02", "(feature)", "[fallback: initial]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notes := buildAutoReopenNotes(tt.trigger, tt.fallbackKind)
			for _, want := range tt.wantContains {
				if !strings.Contains(notes, want) {
					t.Errorf("notes %q should contain %q", notes, want)
				}
			}
			for _, notWant := range tt.wantNotContains {
				if strings.Contains(notes, notWant) {
					t.Errorf("notes %q should NOT contain %q", notes, notWant)
				}
			}
		})
	}
}
