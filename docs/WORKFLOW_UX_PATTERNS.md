# Workflow UX Patterns - Design Guidelines

## Overview

This document provides UX patterns and design guidelines for E07-F16 workflow improvements, ensuring consistent user experience across all workflow-related commands.

---

## Pattern 1: Positional Arguments for Discovery

### Principle
Use positional arguments for common filtering operations to make commands discoverable via `--help`.

### Examples

**✓ CORRECT - Positional Arguments**
```bash
shark task list status ready-for-development
shark task list phase development
shark task list epic E04
shark task list feature F01
```

**✗ AVOID - Flag-Only Approaches**
```bash
shark task list --status=ready-for-development
shark task list --phase=development
shark --epic=E04 task list
```

### Why This Works
- Users see positional arguments in help text
- More discoverable than flag-only interfaces
- Consistent with existing Shark patterns
- Natural language feel ("list tasks by status")

### Implementation
```go
var taskListCmd = &cobra.Command{
	Use: "list [EPIC] [FEATURE] [status] [phase]",
	Short: "List tasks",
	Long: `List tasks with optional filtering by epic, feature, status, or phase.

Examples:
  shark task list                           List all tasks
  shark task list E04                       Filter by epic
  shark task list E04 F01                   Filter by epic and feature
  shark task list status ready-for-development  Filter by status
  shark task list phase development        Filter by phase (future)`,
}
```

---

## Pattern 2: Case-Insensitive Input

### Principle
Accept user input in any case format and normalize internally.

### Examples

**✓ All Valid**
```bash
shark task list status ready-for-development
shark task list status READY_FOR_DEVELOPMENT
shark task list status Ready-For-Development
shark task list status Ready_For_Development
```

### Implementation
```go
func normalizeStatus(input string) string {
	// Replace spaces/hyphens with underscores, convert to lowercase
	normalized := strings.ToLower(input)
	normalized = strings.ReplaceAll(normalized, " ", "_")
	normalized = strings.ReplaceAll(normalized, "-", "_")
	return normalized
}

// In command handler
userInput := args[0]
normalizedStatus := normalizeStatus(userInput)
```

### Why This Works
- Reduces user cognitive load
- More forgiving of typos and formatting variations
- Common in user-friendly CLIs

---

## Pattern 3: Helpful Error Messages

### Principle
When user input is invalid, show what's available plus how to fix it.

### Error Message Template

```
Error: <what went wrong>

Available <resource>s:
  - <option1>
  - <option2>
  - <option3>

Tips:
  • Use 'shark <resource> list' to see all options
  • Use shell completion: shark task list status <TAB>
  • <relevant hint>
```

### Example

```bash
$ shark task list status invalid_status
Error: Status 'invalid_status' not found in workflow configuration

Available statuses:
  draft
  ready_for_refinement
  in_refinement
  ready_for_development
  in_development
  ready_for_code_review
  in_code_review
  ready_for_qa
  in_qa
  ready_for_approval
  in_approval
  completed
  cancelled
  blocked
  on_hold

Tips:
  • Use 'shark workflow list' to see statuses with descriptions
  • Use shell completion: shark task list status <TAB>
  • Check .sharkconfig.json for custom workflow
```

### Implementation
```go
func showAvailableStatuses(workflow *config.WorkflowConfig) error {
	cli.Error("Status not found")

	statuses := make([]string, 0, len(workflow.StatusFlow))
	for status := range workflow.StatusFlow {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)

	fmt.Println("\nAvailable statuses:")
	for _, status := range statuses {
		metadata, ok := workflow.StatusMetadata[status]
		if ok && metadata.Description != "" {
			fmt.Printf("  - %s (%s)\n", status, metadata.Description)
		} else {
			fmt.Printf("  - %s\n", status)
		}
	}

	fmt.Println("\nTips:")
	fmt.Println("  • Use 'shark workflow list' for full workflow diagram")
	fmt.Println("  • Use shell completion: shark task list status <TAB>")

	return fmt.Errorf("invalid status")
}
```

---

## Pattern 4: Interactive Selection for Ambiguity

### Principle
When multiple valid options exist, ask the user to choose explicitly rather than guessing.

### Format

```
<current state>

Available options:
  1) <option1> [metadata]
     <description>

  2) <option2> [metadata]
     <description>

  N) <optionN> [metadata]
     <description>

Enter selection [1-N] or Ctrl+C: _
```

### Example
```bash
$ shark task next-status E07-F20-001
Current status: in_development (phase: development)

Available transitions:
  1) ready_for_code_review (phase: review)
     "Code complete, awaiting review"
     [agents: tech-lead, code-reviewer]

  2) ready_for_refinement (phase: planning)
     "Awaiting specification and analysis"
     [agents: business-analyst, architect]

  3) blocked (phase: any)
     "Temporarily blocked by external dependency"

  4) on_hold (phase: any)
     "Intentionally paused"

Enter selection [1-4] or Ctrl+C: 1
✓ Transitioned: in_development → ready_for_code_review
```

### Implementation
```go
func selectFromOptions(current string, options []string, metadata map[string]*config.StatusMetadata) (string, error) {
	if len(options) == 1 {
		// Only one option - no need to ask
		return options[0], nil
	}

	// Show all options
	fmt.Printf("Current status: %s\n\n", current)
	fmt.Println("Available transitions:")

	for i, option := range options {
		meta, ok := metadata[option]
		phase := "unknown"
		if ok && meta.Phase != "" {
			phase = meta.Phase
		}

		desc := option
		if ok && meta.Description != "" {
			desc = meta.Description
		}

		agents := ""
		if ok && len(meta.AgentTypes) > 0 {
			agents = fmt.Sprintf("[agents: %s]", strings.Join(meta.AgentTypes, ", "))
		}

		fmt.Printf("  %d) %s (phase: %s)\n", i+1, option, phase)
		fmt.Printf("     %q\n", desc)
		if agents != "" {
			fmt.Printf("     %s\n", agents)
		}
		fmt.Println()
	}

	// Get user selection
	var selection int
	for {
		fmt.Printf("Enter selection [1-%d] or Ctrl+C: ", len(options))
		_, err := fmt.Scanf("%d", &selection)
		if err != nil || selection < 1 || selection > len(options) {
			fmt.Println("Invalid selection")
			continue
		}
		break
	}

	return options[selection-1], nil
}
```

---

## Pattern 5: Preview Mode for Exploration

### Principle
Allow users to explore what would happen without making changes.

### Examples

**Status Filtering Preview**
```bash
# Show what would be filtered (same as regular list)
shark task list status in_development --preview
```

**Workflow Transition Preview**
```bash
# Show available transitions without changing
shark task next-status E07-F20-001 --preview

# JSON output for scripting
shark task next-status E07-F20-001 --preview --json
```

### Implementation
```go
// For next-status command
if preview {
	// Don't update database, just show options
	transitionOptions := workflow.StatusFlow[currentStatus]

	result := map[string]interface{}{
		"current_status": currentStatus,
		"available_transitions": buildTransitionDetails(transitionOptions, metadata),
		"would_change": false,
	}

	return cli.OutputJSON(result)
}
```

---

## Pattern 6: Backward Compatibility with Flags

### Principle
Support both new positional syntax and old flag syntax for backward compatibility.

### Examples

**Status Filtering**
```bash
# New positional syntax (primary)
shark task list status in_development

# Old flag syntax (backward compatible)
shark task list --status=in_development

# Both work identically
```

**Epic/Feature Filtering**
```bash
# New positional syntax (already supported)
shark task list E04 F01

# Old flag syntax (already supported)
shark task list --epic=E04 --feature=F01

# Both work identically
```

### Implementation
```go
var taskListCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle positional arguments
		var epicArg, featureArg, statusArg, phaseArg string

		if len(args) > 0 {
			epicArg = args[0]
		}
		if len(args) > 1 {
			featureArg = args[1]
		}
		if len(args) > 2 {
			statusArg = args[2]
		}
		if len(args) > 3 {
			phaseArg = args[3]
		}

		// Handle flags (override positional if provided)
		epicFlag, _ := cmd.Flags().GetString("epic")
		if epicFlag != "" {
			epicArg = epicFlag
		}

		statusFlag, _ := cmd.Flags().GetString("status")
		if statusFlag != "" {
			statusArg = statusFlag
		}

		// Use resolved values
		// ... rest of command logic
	},
}

func init() {
	taskListCmd.Flags().StringP("epic", "e", "", "Filter by epic")
	taskListCmd.Flags().StringP("feature", "f", "", "Filter by feature")
	taskListCmd.Flags().String("status", "", "Filter by status")
	taskListCmd.Flags().String("phase", "", "Filter by phase")
}
```

---

## Pattern 7: JSON Output for Integration

### Principle
All user-facing commands must support `--json` output for scripting and tool integration.

### Output Format

**List with Status Filter**
```bash
$ shark task list status in_development --json
[
  {
    "key": "E07-F20-001",
    "title": "Implement JWT token validation",
    "status": "in_development",
    "phase": "development",
    "priority": 5,
    "agent_type": "backend",
    "feature_id": 123,
    "created_at": "2026-01-08T14:22:00Z",
    "updated_at": "2026-01-08T14:22:00Z"
  },
  ...
]
```

**Task Get with Transitions**
```bash
$ shark task get E07-F20-001 --json
{
  "key": "E07-F20-001",
  "title": "Implement JWT token validation",
  "status": "in_development",
  "phase": "development",
  "priority": 5,
  "available_transitions": [
    {
      "status": "ready_for_code_review",
      "description": "Code complete, awaiting review",
      "phase": "review",
      "agent_types": ["tech-lead", "code-reviewer"]
    },
    {
      "status": "ready_for_refinement",
      "description": "Awaiting specification and analysis",
      "phase": "planning",
      "agent_types": ["business-analyst", "architect"]
    },
    {
      "status": "blocked",
      "description": "Temporarily blocked by external dependency",
      "phase": "any",
      "agent_types": []
    },
    {
      "status": "on_hold",
      "description": "Intentionally paused",
      "phase": "any",
      "agent_types": []
    }
  ]
}
```

**Transition Preview**
```bash
$ shark task next-status E07-F20-001 --preview --json
{
  "task_key": "E07-F20-001",
  "current_status": "in_development",
  "current_phase": "development",
  "available_transitions": [
    {
      "status": "ready_for_code_review",
      "description": "Code complete, awaiting review",
      "phase": "review"
    },
    ...
  ],
  "would_change": false
}
```

### Implementation
```go
if cli.GlobalConfig.JSON {
	return cli.OutputJSON(taskWithTransitions)
}

// Human-readable output
displayTaskWithTransitions(task)
```

---

## Pattern 8: Status Metadata in Display

### Principle
When showing status information, include relevant metadata from workflow config.

### Display Template

```
Status: <status-name> (phase: <phase>)
  → <description>
  → Color: <color> | Agents: <agent-types>
```

### Example
```
Status: in_development (phase: development)
  → Code implementation in progress
  → Color: yellow | Agents: developer, ai-coder
```

### Implementation
```go
func displayStatus(status string, metadata map[string]*config.StatusMetadata) {
	meta, ok := metadata[status]
	if !ok {
		fmt.Printf("Status: %s\n", status)
		return
	}

	fmt.Printf("Status: %s", status)
	if meta.Phase != "" && meta.Phase != "any" {
		fmt.Printf(" (phase: %s)", meta.Phase)
	}
	fmt.Println()

	if meta.Description != "" {
		fmt.Printf("  → %s\n", meta.Description)
	}

	var metaInfo []string
	if meta.Color != "" {
		metaInfo = append(metaInfo, fmt.Sprintf("Color: %s", meta.Color))
	}
	if len(meta.AgentTypes) > 0 {
		metaInfo = append(metaInfo, fmt.Sprintf("Agents: %s", strings.Join(meta.AgentTypes, ", ")))
	}
	if len(metaInfo) > 0 {
		fmt.Printf("  → %s\n", strings.Join(metaInfo, " | "))
	}
}
```

---

## Pattern 9: Workflow Config Validation

### Principle
Validate that task statuses exist in current workflow configuration.

### Implementation

```go
func validateTaskStatusInWorkflow(ctx context.Context, status string, workflow *config.WorkflowConfig) error {
	// Check if status exists in workflow
	_, exists := workflow.StatusFlow[status]
	if !exists {
		return fmt.Errorf(
			"task status '%s' not found in workflow configuration. "+
			"This may happen if workflow was recently changed. "+
			"Use 'shark workflow list' to see valid statuses",
			status,
		)
	}
	return nil
}

// When filtering by status
statusStr, _ := cmd.Flags().GetString("status")
if statusStr != "" {
	if err := validateTaskStatusInWorkflow(ctx, statusStr, workflow); err != nil {
		return err
	}
}
```

---

## Pattern 10: Shell Completion Support

### Principle
Enable tab completion for common values to reduce memorization burden.

### Implementation

```go
// Status completion function
func completeStatuses(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	workflow, err := config.LoadWorkflowConfig(".sharkconfig.json")
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	for status := range workflow.StatusFlow {
		if strings.HasPrefix(status, toComplete) {
			completions = append(completions, status)
		}
	}

	sort.Strings(completions)
	return completions, cobra.ShellCompDirectiveDefault
}

// Register completion in command
var taskListCmd = &cobra.Command{
	...
	ValidArgsFunction: completeStatuses,
}
```

---

## Design Consistency Checklist

### Commands
- [ ] Help text shows examples with positional arguments
- [ ] `--help` mentions all filtering options
- [ ] Long form has usage examples
- [ ] Flag equivalents are documented

### Input Handling
- [ ] User input is case-insensitive
- [ ] Invalid input shows helpful error with available options
- [ ] Both positional and flag syntax work
- [ ] Errors are actionable and suggest fixes

### Output
- [ ] Human-readable output is clean and scannable
- [ ] JSON output is valid and complete
- [ ] Status information includes relevant metadata
- [ ] Available transitions are shown when relevant

### Error Messages
- [ ] Errors clearly state what went wrong
- [ ] Errors show what the user can do about it
- [ ] Errors suggest relevant commands to run
- [ ] Error exit codes are consistent (2 for config/workflow errors)

### Documentation
- [ ] Command help text is complete and clear
- [ ] Examples show both positional and flag syntax
- [ ] Related commands are mentioned
- [ ] Edge cases are documented

---

## Configuration

### Required Workflow Config Structure

```json
{
  "status_flow_version": "1.0",
  "status_flow": {
    "<status>": ["<next-status>", ...],
    ...
  },
  "status_metadata": {
    "<status>": {
      "color": "<color>",
      "description": "<description>",
      "phase": "<phase>",
      "agent_types": ["<agent-type>", ...]
    },
    ...
  },
  "special_statuses": {
    "_start_": ["<initial-status>", ...],
    "_complete_": ["<terminal-status>", ...]
  }
}
```

### Required Fields
- `status_flow`: Map of all statuses and valid transitions
- `status_metadata[status].description`: Human-readable status description
- `status_metadata[status].phase`: Workflow phase for grouping
- `special_statuses._start_`: Initial statuses for new tasks
- `special_statuses._complete_`: Terminal statuses

### Optional Fields
- `status_metadata[status].color`: Display color for status
- `status_metadata[status].agent_types`: Agent types that target this status

---

## Testing Patterns

### Test Status Filtering

```go
func TestTaskListStatusFilter(t *testing.T) {
	tests := []struct {
		name           string
		statusInput    string
		expectedCount  int
		shouldError    bool
	}{
		{
			name:          "valid status",
			statusInput:   "in_development",
			expectedCount: 5,
			shouldError:   false,
		},
		{
			name:        "case insensitive",
			statusInput: "IN_DEVELOPMENT",
			expectedCount: 5,
			shouldError: false,
		},
		{
			name:        "hyphen separator",
			statusInput: "in-development",
			expectedCount: 5,
			shouldError: false,
		},
		{
			name:          "invalid status",
			statusInput:   "invalid_status",
			expectedCount: 0,
			shouldError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test implementation
		})
	}
}
```

### Test Interactive Transitions

```go
func TestNextStatusInteractive(t *testing.T) {
	tests := []struct {
		name              string
		currentStatus     string
		availableOptions  []string
		userSelection     string
		expectedNewStatus string
		shouldError       bool
	}{
		{
			name:              "single option",
			currentStatus:     "in_approval",
			availableOptions:  []string{"completed"},
			userSelection:     "1",
			expectedNewStatus: "completed",
			shouldError:       false,
		},
		{
			name:              "multiple options user chooses 2",
			currentStatus:     "in_development",
			availableOptions:  []string{"ready_for_code_review", "blocked", "on_hold"},
			userSelection:     "2",
			expectedNewStatus: "blocked",
			shouldError:       false,
		},
		{
			name:             "invalid selection",
			currentStatus:    "in_development",
			availableOptions: []string{"ready_for_code_review", "blocked"},
			userSelection:    "99",
			shouldError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test implementation
		})
	}
}
```

---

## Summary

These patterns ensure:
- **Consistency**: All workflow commands follow the same UX principles
- **Discoverability**: Positional arguments visible in help text
- **Safety**: Interactive mode prevents unintended transitions
- **Learnability**: Users can understand and explore the workflow
- **Power**: Scripts and automation can use explicit flags
- **Integration**: JSON output enables tool chaining

Follow these patterns across all E07-F16 implementation to deliver a cohesive user experience.

