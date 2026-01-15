package config

import "context"

// MockActionService is a mock implementation of ActionService for testing
type MockActionService struct {
	GetStatusActionFunc          func(ctx context.Context, status string) (*OrchestratorAction, error)
	GetStatusActionPopulatedFunc func(ctx context.Context, status string, taskID string) (*PopulatedAction, error)
	GetAllActionsFunc            func(ctx context.Context) (map[string]*OrchestratorAction, error)
	ValidateActionsFunc          func(ctx context.Context) (*ValidationResult, error)
	ReloadFunc                   func(ctx context.Context) error
}

// GetStatusAction implements ActionService
func (m *MockActionService) GetStatusAction(ctx context.Context, status string) (*OrchestratorAction, error) {
	if m.GetStatusActionFunc != nil {
		return m.GetStatusActionFunc(ctx, status)
	}
	return nil, nil
}

// GetStatusActionPopulated implements ActionService
func (m *MockActionService) GetStatusActionPopulated(ctx context.Context, status string, taskID string) (*PopulatedAction, error) {
	if m.GetStatusActionPopulatedFunc != nil {
		return m.GetStatusActionPopulatedFunc(ctx, status, taskID)
	}
	return nil, nil
}

// GetAllActions implements ActionService
func (m *MockActionService) GetAllActions(ctx context.Context) (map[string]*OrchestratorAction, error) {
	if m.GetAllActionsFunc != nil {
		return m.GetAllActionsFunc(ctx)
	}
	return map[string]*OrchestratorAction{}, nil
}

// ValidateActions implements ActionService
func (m *MockActionService) ValidateActions(ctx context.Context) (*ValidationResult, error) {
	if m.ValidateActionsFunc != nil {
		return m.ValidateActionsFunc(ctx)
	}
	return &ValidationResult{Valid: true}, nil
}

// Reload implements ActionService
func (m *MockActionService) Reload(ctx context.Context) error {
	if m.ReloadFunc != nil {
		return m.ReloadFunc(ctx)
	}
	return nil
}
