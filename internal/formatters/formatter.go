package formatters

import "github.com/jwwelbor/shark-task-manager/internal/models"

// TaskFormatter defines the interface for formatting task output
// TODO: Implement this interface for multiple output formats
// See docs/future-enhancements/output-formats.md for details
type TaskFormatter interface {
	// FormatTask formats a single task
	FormatTask(task *models.Task) (string, error)
	
	// FormatTaskList formats a list of tasks
	FormatTaskList(tasks []*models.Task) (string, error)
}

// Format represents supported output formats
type Format string

const (
	FormatTable    Format = "table"
	FormatJSON     Format = "json"
	FormatMarkdown Format = "markdown" // TODO: Implement
	FormatYAML     Format = "yaml"     // TODO: Implement
	FormatCSV      Format = "csv"      // TODO: Implement
)

// GetFormatter returns the appropriate formatter for the given format
// TODO: Implement formatters for markdown, yaml, csv
func GetFormatter(format Format) (TaskFormatter, error) {
	// Current implementation uses existing CLI output methods
	// Future: Create separate formatter implementations
	return nil, nil
}
