# Task: E05-F01-T01 - Implement Status Dashboard Service Data Structures

**Feature**: E05-F01 Status Dashboard & Reporting
**Epic**: E05 Task Management CLI Capabilities
**Task Key**: E05-F01-T01

## Description

Implement the foundational data structures for the Status Dashboard service layer. These DTOs (Data Transfer Objects) define the contracts for aggregating project data and returning dashboard information to CLI commands for display.

This task creates the core models that represent:
- Dashboard output structure (StatusDashboard, ProjectSummary, EpicSummary)
- Task grouping for active/blocked sections (TaskInfo, BlockedTaskInfo, CompletionInfo)
- Request/response objects for service interface
- Error types for validation failures

**Why This Matters**: The data structures form the contract between the service layer and command layer. Correct modeling ensures type safety, proper JSON serialization, and clear separation of concerns.

## What You'll Build

A new package `internal/status/` with complete data models:

```
internal/status/
├── models.go          # All data structures (300-400 LOC)
├── errors.go          # Error types and helpers (50-80 LOC)
├── status.go          # Service interface (stub implementation)
└── status_test.go     # Unit tests for models
```

## Success Criteria

- [x] `StatusDashboard` struct with all required fields (Summary, Epics, ActiveTasks, BlockedTasks, RecentCompletions, Filter)
- [x] `ProjectSummary` struct with CountBreakdown and StatusBreakdown nested types
- [x] `EpicSummary` struct with progress, health, and task counts
- [x] `TaskInfo`, `BlockedTaskInfo`, `CompletionInfo` structs for task list sections
- [x] `StatusRequest` struct with validation method
- [x] `StatusError` type implementing error interface
- [x] All structs have proper JSON tags matching documented schema
- [x] Constants for valid timeframes and agent type ordering
- [x] Package compiles without errors
- [x] Unit tests verify JSON marshaling correctness
- [x] All tests pass with `go test ./internal/status`

## Implementation Notes

### Key Design Decisions

1. **Separate DTOs from Models**: StatusDashboard and related structs are specifically for output formatting, not general-purpose task data. This keeps them focused and prevents tight coupling with the database schema.

2. **JSON Tag Consistency**: All exported fields have `json` tags using snake_case naming to match the documented output schema. Use `omitempty` for optional fields.

3. **Nullable Fields**: Use `*string` and `*time.Time` for optional fields (blocked_reason, agent_type). This allows distinguishing between "not provided" and "empty string".

4. **Agent Type Ordering**: Define canonical ordering in constants to ensure consistent, predictable output:
   ```go
   var AgentTypesOrder = []string{
       "frontend", "backend", "api", "testing", "devops", "general", "unassigned",
   }
   ```

5. **Timeframe Validation**: Predefined set of valid timeframes for completions window:
   ```go
   var ValidTimeframes = map[string]bool{
       "24h": true, "1d": true, "7d": true, "48h": true, "30d": true, "90d": true,
   }
   ```

### Data Structure Hierarchy

```
StatusDashboard
├── ProjectSummary
│   ├── Epics (CountBreakdown)
│   ├── Features (CountBreakdown)
│   ├── Tasks (StatusBreakdown)
│   ├── OverallProgress (float64)
│   └── BlockedCount (int)
├── Epics ([]*EpicSummary)
├── ActiveTasks (map[string][]*TaskInfo)
├── BlockedTasks ([]*BlockedTaskInfo)
├── RecentCompletions ([]*CompletionInfo)
└── Filter (*DashboardFilter)
```

### StatusRequest Validation

The `Validate()` method should check:
- If EpicKey is provided, validate format (should match pattern E[0-9]+)
- If RecentWindow is provided, verify it's in ValidTimeframes
- Return descriptive error messages for validation failures

Example:
```go
func (r *StatusRequest) Validate() error {
    if r.EpicKey != "" && !isValidEpicKey(r.EpicKey) {
        return fmt.Errorf("invalid epic key format: %s", r.EpicKey)
    }
    if r.RecentWindow != "" && !ValidTimeframes[r.RecentWindow] {
        return fmt.Errorf("invalid timeframe: %s (valid: 24h, 7d, 30d, 90d)", r.RecentWindow)
    }
    return nil
}
```

### Error Handling

Create StatusError type that implements error interface:

```go
type StatusError struct {
    Message string
    Code    int  // Exit code
}

func (e *StatusError) Error() string {
    return e.Message
}

func NewStatusError(message string) error {
    return &StatusError{Message: message, Code: 1}
}
```

## Dependencies

- Go standard library: encoding/json, fmt, time
- No external dependencies for this task
- Will depend on models.Task once service is built

## Related Tasks

- **E05-F01-T02**: Database Queries - Uses these models as output structs
- **E05-F01-T03**: CLI Command - Imports and uses these models
- **E05-F01-T04**: Output Formatting - Receives StatusDashboard and formats for display

## Acceptance Criteria

**Functional**:
- [ ] All 13 data structures defined with correct fields and JSON tags
- [ ] StatusRequest.Validate() correctly identifies invalid inputs
- [ ] StatusError implements error interface with proper Error() method
- [ ] Constants defined: ValidTimeframes, AgentTypesOrder
- [ ] Code follows Go naming conventions and style

**Testing**:
- [ ] Unit test: JSON marshaling of StatusDashboard produces valid JSON
- [ ] Unit test: StatusRequest.Validate() accepts valid inputs
- [ ] Unit test: StatusRequest.Validate() rejects invalid inputs
- [ ] Unit test: EpicSummary with all fields marshals correctly
- [ ] Unit test: CompletionInfo with nil fields (CompletedAgo) marshals correctly

**Build**:
- [ ] `go build ./internal/status` succeeds without errors
- [ ] `go test ./internal/status -v` runs successfully (test file may have stubs)
- [ ] No Go linting issues: `golangci-lint run ./internal/status`
- [ ] Code formatted: `gofmt -l ./internal/status/` shows no changes needed

## Verification Steps

```bash
# Build the package
go build ./internal/status
echo "Build succeeded"

# Run basic tests
go test ./internal/status -v

# Check JSON marshaling
cat > /tmp/test_json.go << 'EOF'
package main
import (
    "encoding/json"
    "fmt"
    "github.com/yourusername/shark-task-manager/internal/status"
)
func main() {
    dashboard := &status.StatusDashboard{
        Summary: &status.ProjectSummary{
            Epics: &status.CountBreakdown{Total: 1, Active: 1},
        },
    }
    data, _ := json.MarshalIndent(dashboard, "", "  ")
    fmt.Println(string(data))
}
EOF

# Format check
gofmt -l ./internal/status/
```

## Implementation Checklist

See Phase 1 in implementation-checklist.md:
- [ ] Task 1.1: Define output data models (StatusDashboard, ProjectSummary, etc.)
- [ ] Task 1.2: Define request/response types (StatusRequest, StatusError)
- [ ] Task 1.3: Define service interface
- [ ] Task 1.4: Implement service constructor
- [ ] Task 1.5: Implement basic GetDashboard flow (stubs)
- [ ] Task 1.6: Error handling setup
- [ ] Task 1.7: Basic unit tests
