package workflow

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
)

// WorkflowConfig defines the structure for configurable status workflows in .sharkconfig.json
//
// Example JSON configuration:
//
//	{
//	  "status_flow_version": "1.0",
//	  "status_flow": {
//	    "todo": ["in_progress", "blocked"],
//	    "in_progress": ["ready_for_review", "blocked"],
//	    "ready_for_review": ["completed", "in_progress", "blocked"],
//	    "completed": [],
//	    "blocked": ["todo", "in_progress"]
//	  },
//	  "status_metadata": {
//	    "todo": {
//	      "color": "gray",
//	      "description": "Task is ready to be started",
//	      "phase": "planning",
//	      "agent_types": ["business-analyst", "project-manager"]
//	    },
//	    "in_progress": {
//	      "color": "blue",
//	      "description": "Task is actively being worked on",
//	      "phase": "development",
//	      "agent_types": ["developer", "backend", "frontend"]
//	    },
//	    "ready_for_review": {
//	      "color": "yellow",
//	      "description": "Task implementation complete, awaiting review",
//	      "phase": "review",
//	      "agent_types": ["tech-lead", "qa"]
//	    },
//	    "completed": {
//	      "color": "green",
//	      "description": "Task approved and merged",
//	      "phase": "done",
//	      "agent_types": []
//	    },
//	    "blocked": {
//	      "color": "red",
//	      "description": "Task blocked by external dependency",
//	      "phase": "blocked",
//	      "agent_types": ["project-manager"]
//	    }
//	  },
//	  "special_statuses": {
//	    "_start_": ["todo"],
//	    "_complete_": ["completed"]
//	  },
//	  "require_rejection_reason": true
//	}
type WorkflowConfig struct {
	// Version of the workflow config schema (default: "1.0")
	// Used for future schema evolution and migration
	Version string `json:"status_flow_version"`

	// StatusFlow defines valid status transitions
	// Key: current status, Value: array of valid next statuses
	// Empty array means terminal status (no transitions out)
	StatusFlow map[string][]string `json:"status_flow"`

	// StatusMetadata provides additional metadata for each status
	// Optional: missing metadata fields default gracefully
	StatusMetadata map[string]StatusMetadata `json:"status_metadata"`

	// SpecialStatuses defines workflow entry and exit points
	// _start_: array of initial statuses (e.g., ["todo", "backlog"])
	// _complete_: array of terminal statuses (e.g., ["completed", "archived"])
	SpecialStatuses map[string][]string `json:"special_statuses"`

	// RequireRejectionReason specifies whether rejection reasons are required for backward transitions
	// When true: backward transitions must include a reason (--reason flag) or use --force to bypass
	// When false: backward transitions are allowed without a reason
	// Default: true (enabled)
	// Stored in config as "require_rejection_reason": true/false
	RequireRejectionReason bool `json:"require_rejection_reason"`

	// --- Shark 2.x route-based schema (E35-F01) ---
	//
	// Start names the entry step for the consolidated per-step schema. It is the
	// route-based analogue of special_statuses._start_[0]. Only meaningful when
	// Steps is populated.
	Start string `json:"start,omitempty"`

	// Steps is the consolidated per-step workflow definition (E35 D5). Each step
	// merges what used to be split across status_flow (its transition graph, now
	// expressed as outcomes) and status_metadata (color/phase/weight/action).
	//
	// When Steps is non-empty, buildWorkflowMapsFromSteps() projects it back onto
	// StatusFlow/StatusMetadata/SpecialStatuses so every existing reader keeps
	// working unchanged. Steps therefore becomes the source of truth while the
	// two legacy maps become a derived compatibility view.
	Steps map[string]*Step `json:"steps,omitempty"`
}

// Step is one node in the consolidated route-based workflow schema (E35-F01,
// decision D5). It replaces the status_flow + status_metadata split: a step
// carries both its display/agent metadata and its routing (outcomes) in a
// single block.
//
// All fields are optional except that non-terminal, non-parking steps are
// expected to define an Outcomes map (validated separately).
type Step struct {
	// Phase groups the step for ordering/filtering (planning, development,
	// review, qa, approval, done, blocked, paused, …).
	Phase string `json:"phase,omitempty"`

	// Color is the display color for the step's status (red, green, cyan, …).
	Color string `json:"color,omitempty"`

	// Description is human-readable help text for the step.
	Description string `json:"description,omitempty"`

	// ProgressWeight is the step's contribution to weighted progress (0.0-1.0).
	ProgressWeight float64 `json:"progress_weight,omitempty"`

	// Responsibility records who owns work at this step (agent, human, qa_team,
	// none).
	Responsibility string `json:"responsibility,omitempty"`

	// AgentTypes lists agent types that should see entities parked at this step.
	// When empty and Agent is set, Agent is used as the single agent type.
	AgentTypes []string `json:"agent_types,omitempty"`

	// Action is the orchestrator action for this step (spawn_agent,
	// advance_status, check_or_resume, pause, archive, cascade,
	// wait_for_triage).
	Action string `json:"action,omitempty"`

	// Agent is the agent type to spawn (for spawn_agent). Route-based analogue
	// of orchestrator_action.agent_type.
	Agent string `json:"agent,omitempty"`

	// Provider is the AI provider for the dispatched agent (anthropic, openai).
	Provider string `json:"provider,omitempty"`

	// Model is the model override for the dispatched agent.
	Model string `json:"model,omitempty"`

	// Skills lists the skills the dispatched agent should load.
	Skills []string `json:"skills,omitempty"`

	// Prompt is the instruction template path/string for the step's agent.
	// Route-based rename of orchestrator_action.instruction_template (D5).
	Prompt string `json:"prompt,omitempty"`

	// Outcomes maps a semantic outcome name (pass, fail, blocked, plus extras)
	// to the target step the engine should route to. Replaces status_flow (D2).
	Outcomes map[string]string `json:"outcomes,omitempty"`

	// Aliases lists old status names that collapse into this step. Drives the
	// one-shot status migration, input compat shim, and history-read resolution
	// (E35-F05, §7).
	Aliases []string `json:"aliases,omitempty"`

	// Parking marks a step whose resume target is computed from history rather
	// than a static outcome (e.g. blocked, on_hold).
	Parking bool `json:"parking,omitempty"`

	// Terminal marks an end state with no outcomes (e.g. completed, cancelled).
	Terminal bool `json:"terminal,omitempty"`

	// BlocksFeature indicates entities at this step block parent feature/epic
	// progress.
	BlocksFeature bool `json:"blocks_feature,omitempty"`

	// IsPlanning indicates this step is a planning-phase step (the entity tracks
	// its own status rather than aggregating children).
	IsPlanning bool `json:"is_planning,omitempty"`

	// AggregatesFrom indicates this step derives progress from children
	// ("features", "tasks", or "").
	AggregatesFrom string `json:"aggregates_from,omitempty"`

	// ExcludeFromProgress indicates entities at this step are omitted from
	// progress calculations (e.g. cancelled).
	ExcludeFromProgress bool `json:"exclude_from_progress,omitempty"`

	// DisplayToken is a short, human-chosen status abbreviation for dense CLI
	// tables (e.g., "IP", "REV", "BLK"). Route-based analogue of
	// StatusMetadata.DisplayToken. When omitted, callers may derive a fallback.
	DisplayToken string `json:"display_token,omitempty"`

	// SprintBucket defines which sprint display bucket this step's status belongs
	// to ("ready", "in_progress", "blocked", "done", or "" to omit from the
	// sprint view). Route-based analogue of StatusMetadata.SprintBucket. When
	// nil, the sprint planner derives a bucket from the phase name.
	SprintBucket *string `json:"sprint_bucket,omitempty"`
}

// StatusMetadata provides UI and agent-targeting metadata for a status
// All fields are optional
type StatusMetadata struct {
	// Color for display in CLI/UI (e.g., "red", "green", "blue", "#FF5733")
	// Used for colored terminal output (unless --no-color)
	Color string `json:"color,omitempty"`

	// DisplayToken is a short, human-chosen status abbreviation for dense CLI tables
	// (e.g., "IP", "REV", "BLK"). When omitted, callers may derive a fallback token.
	DisplayToken string `json:"display_token,omitempty"`

	// Human-readable description of what this status means
	Description string `json:"description,omitempty"`

	// Workflow phase grouping (e.g., "planning", "development", "review", "qa", "done")
	// Used for task filtering: `shark task list --phase=development`
	Phase string `json:"phase,omitempty"`

	// Agent types that should see tasks in this status
	// Used for agent-targeted queries: `shark task list --agent=qa`
	// Examples: ["developer", "backend", "frontend", "qa", "business-analyst", "tech-lead"]
	AgentTypes []string `json:"agent_types,omitempty"`

	// ProgressWeight indicates how much this status contributes to overall progress (0.0-1.0)
	// Used by CalculateProgress() to recognize partial completion:
	// - 0.0: not started (todo, draft, backlog)
	// - 0.5: in progress (in_development, in_progress, in_review)
	// - 0.9: nearly complete (ready_for_approval)
	// - 1.0: complete (completed, archived)
	// Default: 0.0 if not specified
	ProgressWeight float64 `json:"progress_weight"`

	// Responsibility defines who is responsible for work in this status
	// Values: "agent", "human", "qa_team", "none"
	// Used for work breakdown calculations (E07-F23)
	Responsibility string `json:"responsibility,omitempty"`

	// BlocksFeature indicates if tasks in this status block the feature progress
	// Used to identify blocked work in work breakdown calculations
	BlocksFeature bool `json:"blocks_feature,omitempty"`

	// OrchestratorAction specifies the action for orchestrators when task enters this status
	// Optional field for workflow-driven agent spawning (Phase 1 feature)
	OrchestratorAction *action.OrchestratorAction `json:"orchestrator_action,omitempty" yaml:"orchestrator_action,omitempty"`

	// IsPlanning indicates this status is a planning phase status.
	// When true, the entity has its own workflow status (not aggregating children).
	// When false (or omitted), the entity may aggregate progress from children.
	// Used by E16-F03 to control display behavior.
	IsPlanning bool `json:"is_planning,omitempty"`

	// AggregatesFrom indicates this status derives progress from children.
	// Values: "features" (epic aggregates features), "tasks" (feature aggregates tasks), "" (none).
	// Used by E16-F03 to switch between workflow display and progress display.
	AggregatesFrom string `json:"aggregates_from,omitempty"`

	// ExcludeFromProgress indicates this status should be excluded from progress calculations.
	// When true, entities in this status are not counted in either the numerator or denominator
	// of progress percentages (treated like a soft-delete for progress purposes).
	// Typical use: "cancelled" statuses should not drag down progress percentages.
	ExcludeFromProgress bool `json:"exclude_from_progress,omitempty"`

	// SprintBucket defines which sprint display bucket this status belongs to.
	// Values: "ready", "in_progress", "blocked", "done", or "" (omit from sprint view).
	// When set, the sprint planner uses this value directly instead of deriving
	// a bucket from the phase name. This allows each workflow to explicitly control
	// how its statuses appear in the sprint board.
	SprintBucket *string `json:"sprint_bucket,omitempty"`
}

// Special status keys used in SpecialStatuses map
const (
	// StartStatusKey defines initial statuses where new tasks begin
	StartStatusKey = "_start_"

	// CompleteStatusKey defines terminal statuses where tasks end
	CompleteStatusKey = "_complete_"

	// AggregationStatusKey identifies statuses where an entity switches from
	// its own workflow tracking to aggregating progress from children.
	// Used by epic/feature workflows to distinguish planning from execution phases.
	AggregationStatusKey = "_aggregation_"
)

// Default version for workflow configs
const DefaultWorkflowVersion = "1.0"

// GetStatusMetadata returns metadata for a given status
// Returns empty metadata if status not found
func (w *WorkflowConfig) GetStatusMetadata(status string) (StatusMetadata, bool) {
	if w.StatusMetadata == nil {
		return StatusMetadata{}, false
	}
	meta, found := w.StatusMetadata[status]
	return meta, found
}

// UnmarshalJSON implements custom unmarshaling for WorkflowConfig
// Ensures RequireRejectionReason defaults to true when not specified in JSON
func (w *WorkflowConfig) UnmarshalJSON(data []byte) error {
	// Use alias to avoid infinite recursion
	type Alias WorkflowConfig
	aux := &struct {
		RequireRejectionReason *bool `json:"require_rejection_reason"`
		*Alias
	}{
		Alias: (*Alias)(w),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Set default value if not specified in JSON
	if aux.RequireRejectionReason == nil {
		w.RequireRejectionReason = true
	} else {
		w.RequireRejectionReason = *aux.RequireRejectionReason
	}

	return nil
}

// GetStatusesByAgentType returns all statuses that include the given agent type
// Returns empty slice if no statuses match
func (w *WorkflowConfig) GetStatusesByAgentType(agentType string) []string {
	if w.StatusMetadata == nil {
		return []string{}
	}

	var statuses []string
	for status, meta := range w.StatusMetadata {
		for _, at := range meta.AgentTypes {
			if at == agentType {
				statuses = append(statuses, status)
				break
			}
		}
	}
	return statuses
}

// GetStatusesByPhase returns all statuses in the given phase, sorted
// alphabetically for deterministic ordering (StatusMetadata is a map, so
// unsorted iteration would make callers that pick the first result, e.g.
// SprintService's phase-derived status helpers, non-deterministic across
// calls). Returns empty slice if no statuses match.
func (w *WorkflowConfig) GetStatusesByPhase(phase string) []string {
	if w.StatusMetadata == nil {
		return []string{}
	}

	var statuses []string
	for status, meta := range w.StatusMetadata {
		if meta.Phase == phase {
			statuses = append(statuses, status)
		}
	}
	sort.Strings(statuses)
	return statuses
}

// getPhaseOrder returns the ordering of phases for backward transition detection
// Lower numbers represent earlier phases in the workflow
// Returns -1 for unknown/any phases (which don't participate in backward detection)
func getPhaseOrder(phase string) int {
	phaseOrder := map[string]int{
		"planning":    0,
		"development": 1,
		"review":      2,
		"qa":          3,
		"approval":    4,
		"done":        5,
		"any":         -1, // Special phase that doesn't participate in order
		"blocked":     -1, // Special phase that doesn't participate in order
	}

	if order, found := phaseOrder[phase]; found {
		return order
	}

	// Unknown phases are treated as non-participating (-1)
	return -1
}

// IsBackwardTransition determines if a transition from one status to another is backward
// based on phase ordering. A backward transition is one where the new phase is ordered
// before (lower order number) the current phase.
//
// Returns:
//   - (false, nil) for forward transitions or same phase
//   - (true, nil) for backward transitions
//   - (false, error) if either status is not found in metadata
//
// Special cases:
//   - Transitions to/from "any" phase are not considered backward
//   - Transitions to/from "blocked" phase are not considered backward
//   - If either status lacks phase metadata, returns (false, nil) - not backward
func (w *WorkflowConfig) IsBackwardTransition(fromStatus, toStatus string) (bool, error) {
	// Get metadata for both statuses
	fromMeta, fromFound := w.GetStatusMetadata(fromStatus)
	toMeta, toFound := w.GetStatusMetadata(toStatus)

	// If either status is not found in metadata, return error
	if !fromFound || !toFound {
		return false, fmt.Errorf("status not found in metadata: from=%s (found=%v), to=%s (found=%v)",
			fromStatus, fromFound, toStatus, toFound)
	}

	// If either status lacks phase information, treat as not backward
	if fromMeta.Phase == "" || toMeta.Phase == "" {
		return false, nil
	}

	// Get phase orders
	fromOrder := getPhaseOrder(fromMeta.Phase)
	toOrder := getPhaseOrder(toMeta.Phase)

	// If either phase is "any" or "blocked" (order = -1), not backward
	if fromOrder == -1 || toOrder == -1 {
		return false, nil
	}

	// Backward if new phase order is less than current phase order
	return toOrder < fromOrder, nil
}
