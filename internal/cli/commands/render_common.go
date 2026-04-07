package commands

import (
	"fmt"
	"unicode"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/pterm/pterm"
)

// EntityDisplayOptions configures entity display rendering.
// TODO E15-F07: Commands will use this struct when refactored to service layer pattern.
type EntityDisplayOptions struct {
	// Entity metadata
	EntityType string // "epic", "feature", or "task"
	Key        string
	Status     string

	// Common display sections (nil/empty values skip section)
	BasicInfo          [][]string              // Key-value pairs for info table
	ValidTransitions   []string                // Allowed next statuses
	OrchestratorAction *config.PopulatedAction // Next action instruction
	RelatedDocs        []*models.Document      // Related documents
	Notes              []*models.EntityNote    // Entity notes
	ContextData        *models.ContextData     // Context information

	// Entity-specific content callback
	RenderSpecific func() // Callback for entity-specific sections (features table, tasks table, etc.)
}

// RenderEntity renders a complete entity display using EntityDisplayOptions.
// This function orchestrates section rendering in standard order.
//
// TODO E15-F07: Commands will use this function when refactored to service layer pattern.
// For now, this establishes the pattern and section ordering for future consolidation.
//
// Standard section order:
//  1. Header (entity type + key)
//  2. Basic Info (key-value table)
//  3. Valid Transitions (allowed next statuses)
//  4. Orchestrator Action (next action instruction)
//  5. Related Documents (linked artifacts)
//  6. [Entity-Specific Sections via RenderSpecific callback]
//  7. Notes (entity notes)
//  8. Context Data (progress, decisions, questions)
//
// Parameters:
//   - opts: EntityDisplayOptions struct with all display configuration
//
// Example:
//
//	opts := EntityDisplayOptions{
//	    EntityType: "feature",
//	    Key: "E07-F01",
//	    Status: "active",
//	    BasicInfo: [][]string{{"Title", "..."}},
//	    ValidTransitions: []string{"completed", "on_hold"},
//	    RenderSpecific: func() {
//	        renderFeatureTasksTable(tasks)
//	    },
//	}
//	RenderEntity(opts)
func RenderEntity(opts EntityDisplayOptions) {
	// 1. Header
	renderHeader(opts.EntityType, opts.Key)

	// 2. Basic Info
	renderBasicInfo(opts.BasicInfo)

	// 3. Valid Transitions
	renderValidTransitions(opts.Status, opts.ValidTransitions)

	// 4. Orchestrator Action
	renderOrchestratorAction(opts.OrchestratorAction)

	// 5. Related Documents
	renderRelatedDocuments(opts.RelatedDocs)

	// 6. Entity-Specific Sections (callback)
	if opts.RenderSpecific != nil {
		opts.RenderSpecific()
	}

	// 7. Notes
	renderNotes(opts.Notes)

	// 8. Context Data
	renderContextData(opts.ContextData)
}

// renderHeader displays the entity header section.
// Uses pterm.DefaultSection for consistent styling.
//
// Parameters:
//   - entityType: "epic", "feature", "task", "bug", or "change"
//   - key: entity key (e.g., "E07", "E07-F01", "E07-F01-001", "B001", "C001")
func renderHeader(entityType, key string) {
	pterm.DefaultSection.Printf("%s: %s", displayEntityTypeName(entityType), key)
	fmt.Println()
}

// displayEntityTypeName returns the human-readable display name for an entity type.
// Maps internal entity type strings to user-facing display names.
//
// Mappings:
//   - "bug"    -> "Bug"
//   - "change" -> "Change Card" (space, not hyphen per ADR-F06-003)
//   - others   -> capitalize first letter (e.g., "epic" -> "Epic")
//
// This produces headers like "Bug: B001", "Change Card: C001",
// matching the existing "Epic: E07" pattern.
func displayEntityTypeName(entityType string) string {
	switch entityType {
	case "bug":
		return "Bug"
	case "change":
		return "Change Card"
	case "tech_debt":
		return "Tech Debt"
	default:
		return capitalize(entityType)
	}
}

// capitalize capitalizes the first letter of a string
func capitalize(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// truncateRunes truncates a string to maxLen runes, appending "..." if truncated.
// Unlike byte-based slicing (s[:n]), this safely handles multi-byte UTF-8 characters
// (CJK, emoji, etc.) without producing corrupted output.
func truncateRunes(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// renderBasicInfo renders key-value info table.
// Expects [][]string with format: []{"Label", "Value"}
//
// Parameters:
//   - info: key-value pairs for display (e.g., [["Title", "..."], ["Status", "..."]])
//
// Example:
//
//	info := [][]string{
//	    {"Title", "User Authentication"},
//	    {"Status", "active"},
//	    {"Priority", "high"},
//	}
//	renderBasicInfo(info)
func renderBasicInfo(info [][]string) {
	if len(info) == 0 {
		return
	}
	_ = pterm.DefaultTable.WithData(info).Render()
	fmt.Println()
}

// renderValidTransitions displays allowed next statuses.
// Shows a simple list of valid transitions from current status.
//
// Parameters:
//   - status: current status (for context display)
//   - transitions: list of allowed next statuses
//
// Example:
//
//	renderValidTransitions("in_development", []string{"ready_for_code_review", "blocked"})
//	// Output:
//	// Valid Transitions
//	// ━━━━━━━━━━━━━━━━━━
//	//   - ready_for_code_review
//	//   - blocked
func renderValidTransitions(status string, transitions []string) {
	if len(transitions) == 0 {
		return
	}

	pterm.DefaultSection.Println("Valid Transitions")
	for _, transition := range transitions {
		fmt.Printf("  - %s\n", transition)
	}
	fmt.Println()
}

// renderOrchestratorAction displays next action instruction.
// Reuses existing displayOrchestratorAction implementation.
//
// Parameters:
//   - action: populated action from workflow config (nil if no action configured)
//
// Note: This is a thin wrapper around existing displayOrchestratorAction()
// to maintain consistency with current rendering while making it reusable.
func renderOrchestratorAction(action *config.PopulatedAction) {
	// Delegate to existing implementation in orchestrator_display.go
	displayOrchestratorAction(action)
}

// renderRelatedDocuments displays list of related documents.
// Shows document title, type, and file path.
//
// Parameters:
//   - docs: list of related documents ([]*models.Document)
//
// Example:
//
//	docs := []*models.Document{
//	    {Title: "PRD", Type: "prd", FilePath: "docs/plan/E07-F31/feature.md"},
//	    {Title: "Research Report", Type: "research", FilePath: "docs/plan/E07-F31/research.md"},
//	}
//	renderRelatedDocuments(docs)
func renderRelatedDocuments(docs []*models.Document) {
	if len(docs) == 0 {
		return
	}

	pterm.DefaultSection.Println("Related Documents")
	for _, doc := range docs {
		fmt.Printf("  - %s (%s)\n", doc.Title, doc.FilePath)
	}
	fmt.Println()
}

// renderNotes displays entity notes with truncation.
// Shows most recent 10 notes by default.
//
// Parameters:
//   - notes: list of entity notes ([]*models.EntityNote)
//
// Format: [type] date  content (truncated to 80 chars)
func renderNotes(notes []*models.EntityNote) {
	if len(notes) == 0 {
		return
	}

	maxDisplay := 10
	totalNotes := len(notes)
	if totalNotes > maxDisplay {
		pterm.DefaultSection.Printf("Notes (showing %d of %d)", maxDisplay, totalNotes)
	} else {
		pterm.DefaultSection.Printf("Notes (%d)", totalNotes)
	}
	fmt.Println()

	displayCount := totalNotes
	if displayCount > maxDisplay {
		displayCount = maxDisplay
	}
	for i := totalNotes - displayCount; i < totalNotes; i++ {
		note := notes[i]
		dateStr := note.CreatedAt.Format("2006-01-02")
		content := truncateRunes(note.Content, 77)
		fmt.Printf("  [%s] %s  %s\n", note.NoteType, dateStr, content)
	}
	fmt.Println()
}

// renderContextData displays context information.
// Delegates to existing printContextData() implementation.
//
// Parameters:
//   - contextData: context data structure (*models.ContextData)
//
// Note: Only renders if context has actual content (progress, decisions, questions, etc.)
func renderContextData(contextData *models.ContextData) {
	if contextData == nil {
		return
	}

	// Check if context has any content
	hasContent := contextData.Progress != nil ||
		len(contextData.ImplementationDecisions) > 0 ||
		len(contextData.OpenQuestions) > 0 ||
		len(contextData.Blockers) > 0

	if !hasContent {
		return
	}

	pterm.DefaultSection.Println("Context")
	fmt.Println()
	printContextData(contextData)
}

// GetValidTransitions extracts valid next statuses from workflow config.
// Returns empty array if status not found in status_flow or config is nil.
//
// Parameters:
//   - status: current status string
//   - workflow: workflow configuration (*config.WorkflowConfig)
//
// Returns:
//   - []string: list of allowed next statuses (empty if not found)
//
// Example:
//
//	cfg, _ := config.LoadWorkflowConfig(".sharkconfig.json")
//	transitions := GetValidTransitions("in_development", cfg)
//	// Returns: ["ready_for_code_review", "blocked", "on_hold"]
func GetValidTransitions(status string, workflow *config.WorkflowConfig) []string {
	if workflow == nil {
		return []string{}
	}

	transitions, ok := workflow.StatusFlow[status]
	if !ok {
		return []string{}
	}

	return transitions
}
