package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// ContextData represents structured resume context for a task
type ContextData struct {
	Progress                 *ProgressContext             `json:"progress,omitempty"`
	ImplementationDecisions  map[string]string            `json:"implementation_decisions,omitempty"`
	OpenQuestions            []string                     `json:"open_questions,omitempty"`
	Blockers                 []BlockerContext             `json:"blockers,omitempty"`
	AcceptanceCriteriaStatus []AcceptanceCriterionContext `json:"acceptance_criteria_status,omitempty"`
	RelatedTasks             []string                     `json:"related_tasks,omitempty"`
}

// ProgressContext tracks what's done, what's current, and what remains
type ProgressContext struct {
	CompletedSteps []string `json:"completed_steps,omitempty"`
	CurrentStep    *string  `json:"current_step,omitempty"`
	RemainingSteps []string `json:"remaining_steps,omitempty"`
}

// BlockerContext represents a blocker with type and timestamp
type BlockerContext struct {
	Description  string    `json:"description"`
	BlockerType  string    `json:"blocker_type"`
	BlockedSince time.Time `json:"blocked_since"`
}

// AcceptanceCriterionContext tracks individual acceptance criteria status
type AcceptanceCriterionContext struct {
	Criterion string `json:"criterion"`
	Status    string `json:"status"` // pending, in_progress, complete, failed, na
}

// Validate validates the ContextData structure
func (cd *ContextData) Validate() error {
	// Validate blocker types if any
	for _, blocker := range cd.Blockers {
		if blocker.Description == "" {
			return fmt.Errorf("blocker description cannot be empty")
		}
		if blocker.BlockerType == "" {
			return fmt.Errorf("blocker type cannot be empty")
		}
		if blocker.BlockedSince.IsZero() {
			return fmt.Errorf("blocker blocked_since timestamp is required")
		}
	}

	// Validate acceptance criteria status if any
	validACStatuses := map[string]bool{
		"pending":     true,
		"in_progress": true,
		"complete":    true,
		"failed":      true,
		"na":          true,
	}
	for _, ac := range cd.AcceptanceCriteriaStatus {
		if ac.Criterion == "" {
			return fmt.Errorf("acceptance criterion cannot be empty")
		}
		if !validACStatuses[ac.Status] {
			return fmt.Errorf("invalid acceptance criterion status: %s (must be one of: pending, in_progress, complete, failed, na)", ac.Status)
		}
	}

	return nil
}

// ToJSON converts ContextData to JSON string
func (cd *ContextData) ToJSON() (string, error) {
	if err := cd.Validate(); err != nil {
		return "", fmt.Errorf("invalid context data: %w", err)
	}
	data, err := json.Marshal(cd)
	if err != nil {
		return "", fmt.Errorf("failed to marshal context data: %w", err)
	}
	return string(data), nil
}

// FromJSON parses JSON string into ContextData
func FromJSON(jsonStr string) (*ContextData, error) {
	if jsonStr == "" {
		return &ContextData{}, nil
	}

	var cd ContextData
	if err := json.Unmarshal([]byte(jsonStr), &cd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context data: %w", err)
	}

	if err := cd.Validate(); err != nil {
		return nil, fmt.Errorf("invalid context data: %w", err)
	}

	return &cd, nil
}

// Merge merges another ContextData into this one (partial updates)
// Non-nil fields in other will override this object's fields
func (cd *ContextData) Merge(other *ContextData) {
	if other.Progress != nil {
		if cd.Progress == nil {
			cd.Progress = &ProgressContext{}
		}
		if other.Progress.CompletedSteps != nil {
			cd.Progress.CompletedSteps = other.Progress.CompletedSteps
		}
		if other.Progress.CurrentStep != nil {
			cd.Progress.CurrentStep = other.Progress.CurrentStep
		}
		if other.Progress.RemainingSteps != nil {
			cd.Progress.RemainingSteps = other.Progress.RemainingSteps
		}
	}

	if other.ImplementationDecisions != nil {
		if cd.ImplementationDecisions == nil {
			cd.ImplementationDecisions = make(map[string]string)
		}
		for k, v := range other.ImplementationDecisions {
			cd.ImplementationDecisions[k] = v
		}
	}

	if other.OpenQuestions != nil {
		cd.OpenQuestions = other.OpenQuestions
	}

	if other.Blockers != nil {
		cd.Blockers = other.Blockers
	}

	if other.AcceptanceCriteriaStatus != nil {
		cd.AcceptanceCriteriaStatus = other.AcceptanceCriteriaStatus
	}

	if other.RelatedTasks != nil {
		cd.RelatedTasks = other.RelatedTasks
	}
}
