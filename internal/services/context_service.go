package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// Bug is an alias to avoid import issues - the actual type comes from models.
type Bug = models.Bug

// ContextService provides business logic for context data operations across all entity types.
type ContextService struct {
	registry *EntityRegistry
}

// NewContextService creates a new ContextService with injected dependencies.
func NewContextService(registry *EntityRegistry) (*ContextService, error) {
	if registry == nil {
		return nil, fmt.Errorf("ContextService: EntityRegistry must not be nil")
	}
	return &ContextService{
		registry: registry,
	}, nil
}

// GetContext returns the parsed context data for an entity.
// Returns nil if the entity has no context data.
func (s *ContextService) GetContext(ctx context.Context, entityType models.EntityType, entityKey string) (*models.ContextData, error) {
	contextJSON, err := s.getContextJSON(ctx, entityType, entityKey)
	if err != nil {
		return nil, err
	}

	if contextJSON == nil || *contextJSON == "" || *contextJSON == "{}" {
		return nil, nil
	}

	contextData, err := models.FromJSON(*contextJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to parse context data: %w", err)
	}

	return contextData, nil
}

// SetContextField sets a single field in the entity's context data using merge semantics.
func (s *ContextService) SetContextField(ctx context.Context, entityType models.EntityType, entityKey string, field string, value string) error {
	// Validate field name
	if !isValidContextField(field) {
		return fmt.Errorf("invalid context field: %s", field)
	}

	// Get current context data
	//
	// KNOWN LIMITATION (deep-review 2026-08-01, non-blocker): for Questions,
	// this read-merge-write has a TOCTOU race against QuestionRepository.
	// persistTransition's guarded writes (ConfigureWorkflow, RecordResponse,
	// Resolve, Withdraw/Supersede) -- the emulated EntityRepository.
	// UpdateContextData path (entity_adapter.go) has no compare-and-swap, so
	// a workflow transition committed between this read and that write could
	// be silently lost. An outright refusal was tried and reverted: it broke
	// the legitimate case of setting a generic field (e.g. current_step) on
	// an already-configured Question, which TC-406 exercises deliberately.
	// A correct fix needs either a guarded UpdateContextData variant on the
	// EntityRepository contract (affects all 9 adapters) or a retry loop
	// scoped to Question; deferred as tech debt rather than rushed here.
	contextJSON, err := s.getContextJSON(ctx, entityType, entityKey)
	if err != nil {
		return err
	}

	var contextData *models.ContextData
	if contextJSON != nil && *contextJSON != "" {
		contextData, err = models.FromJSON(*contextJSON)
		if err != nil {
			return fmt.Errorf("failed to parse existing context data: %w", err)
		}
	} else {
		contextData = &models.ContextData{}
	}

	// Update the specified field
	if err := updateContextField(contextData, field, value); err != nil {
		return fmt.Errorf("failed to update field: %w", err)
	}

	// Convert back to JSON while retaining fields that the generic ContextData
	// DTO intentionally does not own. Question's bounded workflow state is one
	// such field; serializing only ContextData here would silently erase it.
	jsonStr, err := mergeGenericContextData(contextJSON, contextData)
	if err != nil {
		return fmt.Errorf("failed to serialize context data: %w", err)
	}

	return s.setContextJSON(ctx, entityType, entityKey, &jsonStr)
}

// ClearContext removes all context data from an entity.
func (s *ContextService) ClearContext(ctx context.Context, entityType models.EntityType, entityKey string) error {
	contextJSON, err := s.getContextJSON(ctx, entityType, entityKey)
	if err != nil {
		return err
	}
	if entityType == models.EntityTypeQuestion {
		hasQuestionState, err := hasQuestionOwnedContextData(contextJSON)
		if err != nil {
			return fmt.Errorf("inspect Question context data before clear: %w", err)
		}
		if hasQuestionState {
			return fmt.Errorf("cannot clear context for configured Question: generic context clear would discard Question-owned state")
		}
	}
	return s.setContextJSON(ctx, entityType, entityKey, nil)
}

var genericContextFieldNames = []string{
	"progress",
	"implementation_decisions",
	"open_questions",
	"blockers",
	"metadata",
}

// mergeGenericContextData preserves fields outside ContextData's generic
// contract while retaining the established generic serialization behavior for
// its own fields. In particular, Question-owned I-02 fields must survive a
// generic context update because this service cannot validate or recreate
// their workflow semantics.
func mergeGenericContextData(existing *string, contextData *models.ContextData) (string, error) {
	fields := make(map[string]json.RawMessage)
	if existing != nil && *existing != "" {
		if err := json.Unmarshal([]byte(*existing), &fields); err != nil {
			return "", fmt.Errorf("decode existing raw context data: %w", err)
		}
	}
	for _, name := range genericContextFieldNames {
		delete(fields, name)
	}
	genericJSON, err := contextData.ToJSON()
	if err != nil {
		return "", err
	}
	var genericFields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(genericJSON), &genericFields); err != nil {
		return "", fmt.Errorf("decode serialized generic context data: %w", err)
	}
	for name, value := range genericFields {
		fields[name] = value
	}
	merged, err := json.Marshal(fields)
	if err != nil {
		return "", fmt.Errorf("encode merged context data: %w", err)
	}
	return string(merged), nil
}

// hasQuestionOwnedContextData identifies the private persisted fields that a
// generic clear cannot safely reconstruct. It intentionally checks raw JSON:
// even malformed private data must not be silently discarded by a generic
// operation that does not own it.
func hasQuestionOwnedContextData(contextJSON *string) (bool, error) {
	if contextJSON == nil || *contextJSON == "" {
		return false, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(*contextJSON), &fields); err != nil {
		return false, fmt.Errorf("decode raw context data: %w", err)
	}
	for _, name := range []string{"question_state", "question_terminal_provenance"} {
		if _, found := fields[name]; found {
			return true, nil
		}
	}
	return false, nil
}

// getContextJSON retrieves the raw context JSON string for an entity via the registry.
func (s *ContextService) getContextJSON(ctx context.Context, entityType models.EntityType, entityKey string) (*string, error) {
	repo, err := s.registry.GetRepository(entityType)
	if err != nil {
		return nil, fmt.Errorf("unsupported entity type for context: %w", err)
	}
	entity, err := repo.GetByKey(ctx, entityKey)
	if err != nil {
		return nil, fmt.Errorf("%s not found: %s: %w", entityType, entityKey, err)
	}
	return repo.GetContextData(ctx, entity.GetID())
}

// setContextJSON writes the raw context JSON string for an entity via the registry.
func (s *ContextService) setContextJSON(ctx context.Context, entityType models.EntityType, entityKey string, contextJSON *string) error {
	repo, err := s.registry.GetRepository(entityType)
	if err != nil {
		return fmt.Errorf("unsupported entity type for context: %w", err)
	}
	entity, err := repo.GetByKey(ctx, entityKey)
	if err != nil {
		return fmt.Errorf("%s not found: %s: %w", entityType, entityKey, err)
	}
	return repo.UpdateContextData(ctx, entity.GetID(), contextJSON)
}

// isValidContextField checks if a field name is a valid context field.
func isValidContextField(field string) bool {
	validFields := map[string]bool{
		"current_step":             true,
		"completed_steps":          true,
		"remaining_steps":          true,
		"implementation_decisions": true,
		"open_questions":           true,
		"blockers":                 true,
	}
	return validFields[field]
}

// updateContextField updates a specific field in the context data.
// This is extracted from the CLI layer for reuse across entity types.
func updateContextField(cd *models.ContextData, field, value string) error {
	switch field {
	case "current_step":
		if cd.Progress == nil {
			cd.Progress = &models.ProgressContext{}
		}
		cd.Progress.CurrentStep = &value

	case "completed_steps":
		var steps []string
		if err := json.Unmarshal([]byte(value), &steps); err != nil {
			return fmt.Errorf("invalid JSON for completed_steps: %w", err)
		}
		if cd.Progress == nil {
			cd.Progress = &models.ProgressContext{}
		}
		cd.Progress.CompletedSteps = steps

	case "remaining_steps":
		var steps []string
		if err := json.Unmarshal([]byte(value), &steps); err != nil {
			return fmt.Errorf("invalid JSON for remaining_steps: %w", err)
		}
		if cd.Progress == nil {
			cd.Progress = &models.ProgressContext{}
		}
		cd.Progress.RemainingSteps = steps

	case "implementation_decisions":
		var decisions map[string]string
		if err := json.Unmarshal([]byte(value), &decisions); err != nil {
			return fmt.Errorf("invalid JSON for implementation_decisions: %w", err)
		}
		if cd.ImplementationDecisions == nil {
			cd.ImplementationDecisions = make(map[string]string)
		}
		for k, v := range decisions {
			cd.ImplementationDecisions[k] = v
		}

	case "open_questions":
		var questions []string
		if err := json.Unmarshal([]byte(value), &questions); err != nil {
			return fmt.Errorf("invalid JSON for open_questions: %w", err)
		}
		cd.OpenQuestions = questions

	case "blockers":
		var blockers []models.BlockerContext
		if err := json.Unmarshal([]byte(value), &blockers); err != nil {
			return fmt.Errorf("invalid JSON for blockers: %w", err)
		}
		cd.Blockers = blockers

	default:
		return fmt.Errorf("unsupported context field: %s", field)
	}

	return nil
}
