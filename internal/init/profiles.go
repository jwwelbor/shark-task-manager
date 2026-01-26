package init

import (
	"fmt"
	"strings"
)

// basicProfile defines the basic 5-status workflow
var basicProfile = &WorkflowProfile{
	Name:        "basic",
	Description: "Simple workflow for solo developers",
	StatusMetadata: map[string]*StatusMetadata{
		"todo": {
			Color:          "gray",
			Phase:          "planning",
			ProgressWeight: 0.0,
			Responsibility: "none",
			BlocksFeature:  false,
			Description:    "Task not started",
		},
		"in_progress": {
			Color:          "yellow",
			Phase:          "development",
			ProgressWeight: 0.5,
			Responsibility: "agent",
			BlocksFeature:  false,
			Description:    "Work in progress",
		},
		"ready_for_review": {
			Color:          "magenta",
			Phase:          "review",
			ProgressWeight: 0.75,
			Responsibility: "human",
			BlocksFeature:  true,
			Description:    "Awaiting human review",
		},
		"completed": {
			Color:          "green",
			Phase:          "done",
			ProgressWeight: 1.0,
			Responsibility: "none",
			BlocksFeature:  false,
			Description:    "Task finished",
		},
		"blocked": {
			Color:          "red",
			Phase:          "any",
			ProgressWeight: 0.0,
			Responsibility: "none",
			BlocksFeature:  true,
			Description:    "Blocked by external dependency",
		},
	},
}

// advancedProfile defines the comprehensive 19-status TDD workflow
var advancedProfile = &WorkflowProfile{
	Name:        "advanced",
	Description: "Comprehensive TDD workflow for teams",
	StatusMetadata: map[string]*StatusMetadata{
		"draft": {
			Color:          "gray",
			Phase:          "planning",
			ProgressWeight: 0.0,
			Responsibility: "none",
			BlocksFeature:  false,
			Description:    "Task created but not yet refined",
		},
		"ready_for_refinement_ba": {
			Color:          "cyan",
			Phase:          "planning",
			ProgressWeight: 0.05,
			Responsibility: "human",
			BlocksFeature:  false,
			Description:    "Awaiting business analysis",
		},
		"in_refinement_ba": {
			Color:          "cyan",
			Phase:          "planning",
			ProgressWeight: 0.1,
			Responsibility: "human",
			BlocksFeature:  false,
			Description:    "Under business analysis",
		},
		"ready_for_refinement_tech": {
			Color:          "blue",
			Phase:          "planning",
			ProgressWeight: 0.15,
			Responsibility: "human",
			BlocksFeature:  false,
			Description:    "Awaiting technical review",
		},
		"in_refinement_tech": {
			Color:          "blue",
			Phase:          "planning",
			ProgressWeight: 0.2,
			Responsibility: "human",
			BlocksFeature:  false,
			Description:    "Under technical review",
		},
		"ready_for_development": {
			Color:          "orange",
			Phase:          "planning",
			ProgressWeight: 0.25,
			Responsibility: "none",
			BlocksFeature:  false,
			Description:    "Ready for development",
		},
		"in_development": {
			Color:          "yellow",
			Phase:          "development",
			ProgressWeight: 0.5,
			Responsibility: "agent",
			BlocksFeature:  false,
			Description:    "Code implementation in progress",
		},
		"ready_for_code_review": {
			Color:          "magenta",
			Phase:          "review",
			ProgressWeight: 0.75,
			Responsibility: "human",
			BlocksFeature:  true,
			Description:    "Awaiting code review",
		},
		"in_code_review": {
			Color:          "magenta",
			Phase:          "review",
			ProgressWeight: 0.8,
			Responsibility: "agent",
			BlocksFeature:  false,
			Description:    "Under code review",
		},
		"changes_requested": {
			Color:          "orange",
			Phase:          "development",
			ProgressWeight: 0.6,
			Responsibility: "agent",
			BlocksFeature:  false,
			Description:    "Code review changes requested",
		},
		"ready_for_qa": {
			Color:          "cyan",
			Phase:          "qa",
			ProgressWeight: 0.85,
			Responsibility: "none",
			BlocksFeature:  false,
			Description:    "Ready for QA testing",
		},
		"in_qa": {
			Color:          "green",
			Phase:          "qa",
			ProgressWeight: 0.85,
			Responsibility: "qa_team",
			BlocksFeature:  false,
			Description:    "Being tested",
		},
		"qa_failed": {
			Color:          "orange",
			Phase:          "development",
			ProgressWeight: 0.5,
			Responsibility: "agent",
			BlocksFeature:  false,
			Description:    "QA testing failed",
		},
		"ready_for_approval": {
			Color:          "cyan",
			Phase:          "approval",
			ProgressWeight: 0.9,
			Responsibility: "none",
			BlocksFeature:  true,
			Description:    "Ready for final approval",
		},
		"in_approval": {
			Color:          "purple",
			Phase:          "approval",
			ProgressWeight: 0.95,
			Responsibility: "human",
			BlocksFeature:  false,
			Description:    "Under final review",
		},
		"completed": {
			Color:          "white",
			Phase:          "done",
			ProgressWeight: 1.0,
			Responsibility: "none",
			BlocksFeature:  false,
			Description:    "Task finished and approved",
		},
		"blocked": {
			Color:          "red",
			Phase:          "any",
			ProgressWeight: 0.0,
			Responsibility: "none",
			BlocksFeature:  true,
			Description:    "Temporarily blocked by external dependency",
		},
		"cancelled": {
			Color:          "gray",
			Phase:          "done",
			ProgressWeight: 0.0,
			Responsibility: "none",
			BlocksFeature:  false,
			Description:    "Task abandoned or deprecated",
		},
		"on_hold": {
			Color:          "orange",
			Phase:          "any",
			ProgressWeight: 0.0,
			Responsibility: "none",
			BlocksFeature:  false,
			Description:    "Intentionally paused",
		},
	},
	StatusFlow: map[string][]string{
		"draft": {
			"ready_for_refinement_ba",
			"cancelled",
			"on_hold",
		},
		"ready_for_refinement_ba": {
			"in_refinement_ba",
			"on_hold",
		},
		"in_refinement_ba": {
			"ready_for_refinement_tech",
			"draft",
			"blocked",
			"on_hold",
		},
		"ready_for_refinement_tech": {
			"in_refinement_tech",
			"on_hold",
		},
		"in_refinement_tech": {
			"ready_for_development",
			"draft",
			"blocked",
			"on_hold",
		},
		"ready_for_development": {
			"in_development",
			"ready_for_refinement_ba",
			"cancelled",
			"on_hold",
		},
		"in_development": {
			"ready_for_code_review",
			"ready_for_refinement_ba",
			"blocked",
			"on_hold",
		},
		"ready_for_code_review": {
			"in_code_review",
			"in_development",
			"on_hold",
		},
		"in_code_review": {
			"ready_for_qa",
			"changes_requested",
			"in_development",
			"ready_for_refinement_ba",
			"on_hold",
		},
		"changes_requested": {
			"ready_for_code_review",
			"in_development",
			"on_hold",
		},
		"ready_for_qa": {
			"in_qa",
			"on_hold",
		},
		"in_qa": {
			"ready_for_approval",
			"qa_failed",
			"in_development",
			"ready_for_refinement_ba",
			"blocked",
			"on_hold",
		},
		"qa_failed": {
			"ready_for_code_review",
			"in_development",
			"on_hold",
		},
		"ready_for_approval": {
			"in_approval",
			"on_hold",
		},
		"in_approval": {
			"completed",
			"ready_for_qa",
			"ready_for_development",
			"ready_for_refinement_ba",
			"on_hold",
		},
		"completed": {},
		"blocked": {
			"ready_for_development",
			"ready_for_refinement_ba",
			"cancelled",
		},
		"cancelled": {},
		"on_hold": {
			"ready_for_refinement_ba",
			"ready_for_development",
			"cancelled",
		},
	},
	SpecialStatuses: map[string][]string{
		"_start_": {
			"draft",
			"ready_for_development",
		},
		"_complete_": {
			"completed",
			"cancelled",
		},
		"_blocked_": {
			"blocked",
			"on_hold",
		},
	},
	StatusFlowVersion: "1.0",
}

// profileRegistry maps profile names to profile definitions
var profileRegistry = map[string]*WorkflowProfile{
	"basic":    basicProfile,
	"advanced": advancedProfile,
}

// GetProfile retrieves a workflow profile by name (case-insensitive)
func GetProfile(name string) (*WorkflowProfile, error) {
	profile, exists := profileRegistry[strings.ToLower(name)]
	if !exists {
		available := ListProfiles()
		return nil, fmt.Errorf("profile not found: %s (available: %s)", name, strings.Join(available, ", "))
	}
	return profile, nil
}

// ListProfiles returns a list of available profile names
func ListProfiles() []string {
	return []string{"basic", "advanced"}
}
