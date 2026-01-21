package formatters

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/pterm/pterm"
)

// TaskTableConfig controls which columns to display and their formatting options.
// This provides flexibility for different views (e.g., task list vs feature get).
type TaskTableConfig struct {
	// Column visibility
	ShowKey            bool
	ShowTitle          bool
	ShowStatus         bool
	ShowPriority       bool
	ShowAgentType      bool
	ShowExecutionOrder bool
	ShowRejections     bool

	// Formatting options
	TitleMaxLength int
	ColorEnabled   bool
	UseHeader      bool

	// Table renderer
	UsePterm bool // If true, uses pterm.DefaultTable; otherwise returns headers/rows for cli.OutputTable
}

// DefaultTaskTableConfig returns the standard configuration used by "task list" command.
func DefaultTaskTableConfig() TaskTableConfig {
	return TaskTableConfig{
		ShowKey:            true,
		ShowTitle:          true,
		ShowStatus:         true,
		ShowPriority:       true,
		ShowAgentType:      true,
		ShowExecutionOrder: true,
		ShowRejections:     true,
		TitleMaxLength:     40,
		ColorEnabled:       true,
		UseHeader:          true,
		UsePterm:           false,
	}
}

// FeatureGetTaskTableConfig returns the configuration used by "feature get" command.
// This has a wider title column and uses pterm for rendering.
func FeatureGetTaskTableConfig() TaskTableConfig {
	return TaskTableConfig{
		ShowKey:            true,
		ShowTitle:          true,
		ShowStatus:         true,
		ShowPriority:       true,
		ShowAgentType:      true,
		ShowExecutionOrder: false, // Not shown in feature get
		ShowRejections:     false, // Not shown in feature get
		TitleMaxLength:     60,
		ColorEnabled:       true,
		UseHeader:          true,
		UsePterm:           true,
	}
}

// TaskTableResult contains the formatted table data.
type TaskTableResult struct {
	Headers []string
	Rows    [][]string
}

// FormatTaskTable formats a list of tasks as a table with configurable columns.
// This is the core function that consolidates all task table rendering logic.
//
// Parameters:
//   - tasks: slice of tasks to format
//   - workflowService: service for status color formatting
//   - config: controls which columns to show and how to format them
//
// Returns:
//   - TaskTableResult with headers and rows ready for rendering
func FormatTaskTable(
	tasks []*models.Task,
	workflowService *workflow.Service,
	config TaskTableConfig,
) TaskTableResult {
	result := TaskTableResult{
		Headers: buildHeaders(config),
		Rows:    make([][]string, 0, len(tasks)),
	}

	for _, task := range tasks {
		row := formatTaskRow(task, workflowService, config)
		result.Rows = append(result.Rows, row)
	}

	return result
}

// RenderTaskTable is a convenience function that formats and renders the table.
// It automatically chooses the appropriate renderer based on config.UsePterm.
//
// Parameters:
//   - tasks: slice of tasks to format and render
//   - workflowService: service for status color formatting
//   - config: controls which columns to show and how to format/render them
//
// Returns:
//   - error if rendering fails (currently always nil)
func RenderTaskTable(
	tasks []*models.Task,
	workflowService *workflow.Service,
	config TaskTableConfig,
) error {
	result := FormatTaskTable(tasks, workflowService, config)

	if config.UsePterm {
		// Use pterm for rendering (used by feature get)
		tableData := make([][]string, 0, len(result.Rows)+1)
		tableData = append(tableData, result.Headers)
		tableData = append(tableData, result.Rows...)
		_ = pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	} else {
		// Use cli.OutputTable for rendering (used by task list)
		cli.OutputTable(result.Headers, result.Rows)
	}

	return nil
}

// buildHeaders constructs the header row based on config
func buildHeaders(config TaskTableConfig) []string {
	headers := []string{}

	if config.ShowKey {
		headers = append(headers, "Key")
	}
	if config.ShowTitle {
		headers = append(headers, "Title")
	}
	if config.ShowStatus {
		headers = append(headers, "Status")
	}
	if config.ShowPriority {
		headers = append(headers, "Priority")
	}
	if config.ShowAgentType {
		headers = append(headers, "Agent Type")
	}
	if config.ShowExecutionOrder {
		headers = append(headers, "Order")
	}

	return headers
}

// formatTaskRow formats a single task into a table row
func formatTaskRow(
	task *models.Task,
	workflowService *workflow.Service,
	config TaskTableConfig,
) []string {
	row := []string{}

	if config.ShowKey {
		keyDisplay := task.Key
		if config.ShowRejections && task.RejectionCount > 0 {
			keyDisplay = task.Key + " " + formatRejectionIndicator(task.RejectionCount)
		}
		row = append(row, keyDisplay)
	}

	if config.ShowTitle {
		title := task.Title
		if len(title) > config.TitleMaxLength {
			title = title[:config.TitleMaxLength-3] + "..."
		}
		row = append(row, title)
	}

	if config.ShowStatus {
		statusDisplay := string(task.Status)
		if config.ColorEnabled && workflowService != nil {
			formatted := workflowService.FormatStatusForDisplay(string(task.Status), true)
			statusDisplay = formatted.Colored
		}
		row = append(row, statusDisplay)
	}

	if config.ShowPriority {
		row = append(row, fmt.Sprintf("%d", task.Priority))
	}

	if config.ShowAgentType {
		agentTypeStr := "none"
		if task.AgentType != nil {
			agentTypeStr = string(*task.AgentType)
		}
		row = append(row, agentTypeStr)
	}

	if config.ShowExecutionOrder {
		execOrder := "-"
		if task.ExecutionOrder != nil {
			execOrder = fmt.Sprintf("%d", *task.ExecutionOrder)
		}
		row = append(row, execOrder)
	}

	return row
}

// formatRejectionIndicator formats the rejection count as an indicator.
// This matches the existing behavior in task.go.
func formatRejectionIndicator(count int) string {
	// Use red circle with count
	return fmt.Sprintf("🔴×%d", count)
}
