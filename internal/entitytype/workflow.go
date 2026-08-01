// Package entitytype centralizes entity-type spelling aliases that cross
// package boundaries.
package entitytype

import "strings"

const (
	WorkflowEpic     = "epic"
	WorkflowFeature  = "feature"
	WorkflowTask     = "task"
	WorkflowSprint   = "sprint"
	WorkflowBug      = "bug"
	WorkflowChange   = "change"
	WorkflowTechDebt = "tech_debt"
	WorkflowQuestion = "question"
)

var workflowLevelAliases = map[string]string{
	"epic":     WorkflowEpic,
	"epics":    WorkflowEpic,
	"feature":  WorkflowFeature,
	"features": WorkflowFeature,
	"task":     WorkflowTask,
	"tasks":    WorkflowTask,
	"sprint":   WorkflowSprint,
	"sprints":  WorkflowSprint,
	"bug":      WorkflowBug,
	"bugs":     WorkflowBug,

	"change":       WorkflowChange,
	"changes":      WorkflowChange,
	"change_card":  WorkflowChange,
	"change_cards": WorkflowChange,
	"changecard":   WorkflowChange,
	"changecards":  WorkflowChange,

	"tech_debt":  WorkflowTechDebt,
	"tech_debts": WorkflowTechDebt,
	"techdebt":   WorkflowTechDebt,
	"techdebts":  WorkflowTechDebt,
	"td":         WorkflowTechDebt,

	"question":  WorkflowQuestion,
	"questions": WorkflowQuestion,
}

// NormalizeWorkflowLevel maps user/config/runtime entity-type spellings to
// canonical workflow slots. Hyphen and underscore variants are equivalent.
func NormalizeWorkflowLevel(raw string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	level, ok := workflowLevelAliases[normalized]
	return level, ok
}

// WorkflowLevelOrSelf normalizes known workflow-level aliases and leaves
// unknown values unchanged so legacy callers keep their existing error paths.
func WorkflowLevelOrSelf(raw string) string {
	if level, ok := NormalizeWorkflowLevel(raw); ok {
		return level
	}
	return raw
}
