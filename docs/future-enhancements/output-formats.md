# Configurable Output Formats

## Overview
Enable flexible output formatting for task data beyond the current JSON and table formats.

## Proposed Formats

### 1. Markdown (`.md`)
```markdown
# Task: T-E04-F01-001

**Status**: completed
**Priority**: 1
**Agent Type**: backend

## Description
Define Epic, Feature, Task, TaskHistory models

## Timestamps
- Created: 2025-12-16 02:16:57
- Started: 2025-12-15 20:16:57
- Completed: 2025-12-15 20:16:57

## Dependencies
None
```

### 2. YAML
```yaml
task:
  key: T-E04-F01-001
  title: Create ORM Models
  status: completed
  priority: 1
  agent_type: backend
  description: Define Epic, Feature, Task, TaskHistory models
  timestamps:
    created_at: 2025-12-16T02:16:57Z
    started_at: 2025-12-15T20:16:57Z
    completed_at: 2025-12-15T20:16:57Z
  dependencies: []
```

### 3. CSV (for bulk export)
```csv
key,title,status,priority,agent_type,created_at,started_at,completed_at
T-E04-F01-001,Create ORM Models,completed,1,backend,2025-12-16T02:16:57Z,2025-12-15T20:16:57Z,2025-12-15T20:16:57Z
```

## Implementation Plan

### Phase 1: Refactor Current Output
- Replace `--json` boolean flag with `--format` string flag
- Support values: `table` (default), `json`, `markdown`, `yaml`, `csv`
- Maintain `--json` as alias to `--format=json` for backwards compatibility

### Phase 2: Create Formatter Interface
```go
type TaskFormatter interface {
    FormatTask(task *models.Task) (string, error)
    FormatTaskList(tasks []*models.Task) (string, error)
}

type JSONFormatter struct{}
type MarkdownFormatter struct{}
type YAMLFormatter struct{}
type CSVFormatter struct{}
type TableFormatter struct{}
```

### Phase 3: Add Format-Specific Libraries
- YAML: `gopkg.in/yaml.v3`
- CSV: Use standard `encoding/csv`
- Markdown: Custom implementation

### Usage Examples
```bash
# Get task in markdown
shark task get T-E04-F01-001 --format=markdown > task.md

# Export all tasks to YAML
shark task list --format=yaml > tasks.yaml

# Generate CSV for spreadsheet import
shark task list --format=csv > tasks.csv

# Pipe markdown to documentation
shark task list --epic=E04 --format=markdown >> project-status.md
```

## Benefits
- **Documentation**: Export tasks as markdown for wikis/docs
- **Integration**: YAML/CSV for tooling integration
- **Portability**: Share task data in multiple formats
- **Automation**: Easier to process in different contexts

## Related Files
- `internal/cli/commands/task.go` - Add format flag and formatter selection
- `internal/formatters/` - New package for output formatters
- `internal/cli/root.go` - Add global `--format` flag option

## Priority
Low - Enhancement for future implementation
