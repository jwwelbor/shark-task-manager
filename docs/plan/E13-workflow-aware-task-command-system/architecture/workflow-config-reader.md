# Workflow Configuration Reader Design

**Epic**: [E13 Workflow-Aware Task Command System](../epic.md)

**Last Updated**: 2026-01-11

---

## Overview

The Workflow Configuration Reader is a centralized service that loads, validates, and provides access to workflow configuration defined in `.sharkconfig.json`. It implements caching and provides a clean API for transition queries used by phase-aware commands.

**Design Principle**: Simple, fast, proven pattern (singleton with file modification time check)

---

## Architecture

### Component Location

**Primary Implementation**: `internal/workflow/service.go` (existing, enhance)

**Related Files**:
- `internal/config/workflow.go` - Config structs and JSON unmarshaling
- `.sharkconfig.json` - Workflow configuration file

### Existing Service (Base)

The project already has a `workflow.Service` that handles:
- Loading workflow config from `.sharkconfig.json`
- Providing status metadata (colors, descriptions, phases, agent types)
- Validating transitions
- Ordering statuses by phase

**Enhancements Needed**:
- Add phase-aware transition helpers
- Add agent type filtering logic
- Enhance error messages for invalid transitions

---

## Configuration File Structure

### .sharkconfig.json Format

```json
{
  "status_flow": {
    "draft": ["ready_for_refinement", "cancelled"],
    "ready_for_refinement": ["in_refinement", "cancelled"],
    "in_refinement": ["ready_for_development", "draft"],
    "ready_for_development": ["in_development", "ready_for_refinement"],
    "in_development": ["ready_for_code_review", "ready_for_refinement", "blocked"],
    "ready_for_code_review": ["in_code_review", "in_development"],
    "in_code_review": ["ready_for_qa", "in_development"],
    "ready_for_qa": ["in_qa"],
    "in_qa": ["ready_for_approval", "in_development", "blocked"],
    "ready_for_approval": ["in_approval"],
    "in_approval": ["completed", "ready_for_qa", "ready_for_refinement"],
    "blocked": ["ready_for_development", "ready_for_refinement"],
    "cancelled": [],
    "completed": []
  },
  "status_metadata": {
    "ready_for_development": {
      "phase": "development",
      "agent_types": ["developer", "ai-coder"],
      "color": "yellow",
      "description": "Spec complete, ready for implementation"
    },
    "in_development": {
      "phase": "development",
      "color": "yellow",
      "description": "Code implementation in progress"
    },
    "ready_for_code_review": {
      "phase": "review",
      "agent_types": ["tech-lead", "code-reviewer"],
      "color": "magenta",
      "description": "Code complete, awaiting review"
    }
    // ... more statuses ...
  },
  "special_statuses": {
    "_start_": ["draft", "ready_for_development"],
    "_complete_": ["completed", "cancelled"]
  }
}
```

### Configuration Fields

**status_flow** (required):
- Maps each status to array of valid next statuses
- Terminal statuses have empty array: `"completed": []`
- Bidirectional flows supported (e.g., dev ↔ refinement)

**status_metadata** (optional):
- Provides rich metadata for each status
- `phase`: Groups statuses into lifecycle phases
- `agent_types`: Lists which agents handle this status
- `color`: Display color for CLI output
- `description`: Human-readable explanation

**special_statuses** (required):
- `_start_`: Array of valid entry statuses for new tasks
- `_complete_`: Array of terminal statuses (no outgoing transitions)

---

## Service API

### Enhanced Workflow Service Methods

#### Core Transition Queries

```go
// GetValidTransitions returns valid next statuses from current status
// Returns empty slice if status is terminal or not found
func (s *Service) GetValidTransitions(currentStatus string) []string

// IsValidTransition checks if transition from → to is allowed
// Case-insensitive comparison
func (s *Service) IsValidTransition(from, to string) bool

// GetTransitionInfo returns detailed info about valid transitions
// Includes target status metadata (phase, agent types, color)
func (s *Service) GetTransitionInfo(currentStatus string) []TransitionInfo
```

#### Phase-Aware Helpers (NEW)

```go
// GetNextPhaseStatus determines standard forward flow status
// Given current status, returns the "ready_for_X" status in next phase
// Example: in_development → ready_for_code_review
func (s *Service) GetNextPhaseStatus(currentStatus string) (string, error)

// GetPreviousPhaseStatus determines standard backward flow status
// Used for rejection - finds earlier phase status
// Example: in_development → in_refinement (or ready_for_refinement)
func (s *Service) GetPreviousPhaseStatus(currentStatus string) (string, error)

// IsReadyForPhase checks if status matches "ready_for_*" pattern
func (s *Service) IsReadyForPhase(status string) bool

// IsInPhase checks if status matches "in_*" pattern
func (s *Service) IsInPhase(status string) bool

// GetPhaseFromStatus returns phase name for a status
// Example: "in_development" → "development"
func (s *Service) GetPhaseFromStatus(status string) string
```

#### Agent Type Queries (NEW)

```go
// GetStatusesByAgentType returns all statuses handled by agent type
// Example: GetStatusesByAgentType("backend") → ["ready_for_development", ...]
func (s *Service) GetStatusesByAgentType(agentType string) []string

// CanAgentClaimStatus checks if agent type can claim a ready_for_* status
// Returns true if status_metadata[status].agent_types contains agentType
// Returns true (permissive) if no agent_types defined for status
func (s *Service) CanAgentClaimStatus(status, agentType string) (bool, []string)
// Returns: (allowed bool, expectedAgentTypes []string)

// GetAgentTypesForStatus returns agent types that handle a status
func (s *Service) GetAgentTypesForStatus(status string) []string
```

#### Validation Methods (ENHANCE)

```go
// ValidateWorkflow checks workflow configuration for errors
// Returns all validation errors found (or empty slice if valid)
func (s *Service) ValidateWorkflow() []error

// IsValidStatus checks if status exists in workflow
func (s *Service) IsValidStatus(status string) bool

// NormalizeStatus returns canonical case for status name
// Example: "IN_DEVELOPMENT" → "in_development"
func (s *Service) NormalizeStatus(status string) string
```

---

## Caching Strategy

### File-Based Cache with Modification Time Check

**Rationale**: Balance between performance and freshness

**Implementation**:

```go
package workflow

import (
    "os"
    "sync"
    "time"
)

// Cache state (package-level)
var (
    cachedService *Service
    cachedModTime time.Time
    cacheMutex    sync.RWMutex
    configPath    string
)

// NewService creates or returns cached service
// Checks file modification time to detect config changes
func NewService(projectRoot string) *Service {
    path := filepath.Join(projectRoot, ".sharkconfig.json")

    cacheMutex.RLock()
    modTime := getFileModTime(path)
    if cachedService != nil && configPath == path && modTime.Equal(cachedModTime) {
        defer cacheMutex.RUnlock()
        return cachedService // Cache hit
    }
    cacheMutex.RUnlock()

    // Cache miss or outdated - reload
    cacheMutex.Lock()
    defer cacheMutex.Unlock()

    // Double-check (another goroutine might have loaded)
    modTime = getFileModTime(path)
    if cachedService != nil && configPath == path && modTime.Equal(cachedModTime) {
        return cachedService
    }

    // Load workflow config
    workflow := config.GetWorkflowOrDefault(path)

    cachedService = &Service{
        workflow:    workflow,
        projectRoot: projectRoot,
    }
    cachedModTime = modTime
    configPath = path

    return cachedService
}

// getFileModTime returns modification time or zero if file doesn't exist
func getFileModTime(path string) time.Time {
    info, err := os.Stat(path)
    if err != nil {
        return time.Time{}
    }
    return info.ModTime()
}

// ResetCache clears cached workflow (for testing)
func ResetCache() {
    cacheMutex.Lock()
    defer cacheMutex.Unlock()
    cachedService = nil
    cachedModTime = time.Time{}
    configPath = ""
}
```

### Cache Behavior

**Cache Hit**:
- Same file path
- File modification time unchanged
- Return cached service (no I/O)
- **Performance**: < 1ms

**Cache Miss**:
- Different file path
- File modified since last load
- File doesn't exist → default workflow
- **Performance**: < 50ms (includes JSON parsing)

**Thread Safety**:
- Read lock for cache checks (concurrent reads allowed)
- Write lock for reloading (exclusive)
- Double-check pattern prevents race conditions

---

## Transition Logic

### Standard Forward Flow (finish command)

**Pattern**: Move from `in_X` to `ready_for_Y` (next phase)

```go
func (s *Service) GetNextPhaseStatus(currentStatus string) (string, error) {
    // Get valid transitions
    validNext := s.GetValidTransitions(currentStatus)
    if len(validNext) == 0 {
        return "", fmt.Errorf("status %s is terminal", currentStatus)
    }

    // Find first "ready_for_*" status (standard forward flow)
    for _, nextStatus := range validNext {
        if strings.HasPrefix(nextStatus, "ready_for_") {
            return nextStatus, nil
        }
    }

    // No ready_for_ status found - check for terminal status
    for _, nextStatus := range validNext {
        if s.IsTerminalStatus(nextStatus) {
            return nextStatus, nil
        }
    }

    // Fallback: return first valid transition
    return validNext[0], nil
}
```

**Examples**:
- `in_development` → `ready_for_code_review`
- `in_code_review` → `ready_for_qa`
- `in_qa` → `ready_for_approval`
- `in_approval` → `completed` (terminal)

### Standard Backward Flow (reject command)

**Pattern**: Move to earlier phase (refinement or previous development stage)

```go
func (s *Service) GetPreviousPhaseStatus(currentStatus string) (string, error) {
    validNext := s.GetValidTransitions(currentStatus)
    if len(validNext) == 0 {
        return "", fmt.Errorf("status %s has no valid transitions", currentStatus)
    }

    // Priority 1: Look for refinement statuses
    for _, nextStatus := range validNext {
        if strings.Contains(nextStatus, "refinement") {
            return nextStatus, nil
        }
    }

    // Priority 2: Look for earlier development phase
    currentPhase := s.GetPhaseFromStatus(currentStatus)
    for _, nextStatus := range validNext {
        nextPhase := s.GetPhaseFromStatus(nextStatus)
        if isEarlierPhase(nextPhase, currentPhase) {
            return nextStatus, nil
        }
    }

    // Priority 3: Any backward transition (not ready_for_*)
    for _, nextStatus := range validNext {
        if !strings.HasPrefix(nextStatus, "ready_for_") && !s.IsTerminalStatus(nextStatus) {
            return nextStatus, nil
        }
    }

    return "", fmt.Errorf("no backward transition found from %s", currentStatus)
}

// Phase order for comparison
var phaseOrder = map[string]int{
    "planning":    0,
    "development": 1,
    "review":      2,
    "qa":          3,
    "approval":    4,
}

func isEarlierPhase(phase1, phase2 string) bool {
    return phaseOrder[phase1] < phaseOrder[phase2]
}
```

**Examples**:
- `in_development` → `in_refinement` (reject back to BA)
- `in_code_review` → `in_development` (reject back to developer)
- `in_qa` → `in_development` (reject back to development)

### Claim Transition (claim command)

**Pattern**: Move from `ready_for_X` to `in_X` (same phase)

```go
func (s *Service) GetClaimStatus(currentStatus string) (string, error) {
    if !strings.HasPrefix(currentStatus, "ready_for_") {
        return "", fmt.Errorf("cannot claim task in status %s (must be ready_for_*)", currentStatus)
    }

    // Extract phase: "ready_for_development" → "development"
    phase := strings.TrimPrefix(currentStatus, "ready_for_")

    // Target status: "in_<phase>"
    targetStatus := "in_" + phase

    // Validate transition is allowed
    if !s.IsValidTransition(currentStatus, targetStatus) {
        return "", fmt.Errorf("workflow does not allow transition from %s to %s", currentStatus, targetStatus)
    }

    return targetStatus, nil
}
```

**Examples**:
- `ready_for_development` → `in_development`
- `ready_for_code_review` → `in_code_review`
- `ready_for_qa` → `in_qa`

---

## Validation Rules

### Workflow Configuration Validation

**Performed on Load** (ValidateWorkflow):

1. **No Orphaned Statuses**
   - Every status in `status_flow` keys appears in `status_metadata`
   - Every status in `status_metadata` appears in `status_flow`

2. **Terminal Status Integrity**
   - All `_complete_` statuses have empty transition arrays
   - All empty transition arrays are in `_complete_` list

3. **Valid Transition Targets**
   - All transition targets exist as `status_flow` keys
   - No transitions to non-existent statuses

4. **Start Status Validity**
   - All `_start_` statuses exist in workflow
   - At least one start status defined

5. **Phase Consistency**
   - All statuses have a defined phase (or "any")
   - Phases follow logical order

**Error Examples**:
```
Error: Orphaned status 'legacy_qa' in status_metadata but not in status_flow
Error: Status 'in_qa' has transition to 'ready_for_final_approval' which does not exist
Error: Terminal status 'completed' has outgoing transitions (should be empty array)
Error: No start statuses defined in special_statuses._start_
```

### Runtime Transition Validation

**Performed on Command Execution**:

```go
func (s *Service) ValidateTransition(from, to string, commandType string) error {
    // 1. Normalize status names
    from = s.NormalizeStatus(from)
    to = s.NormalizeStatus(to)

    // 2. Check both statuses exist
    if !s.IsValidStatus(from) {
        return fmt.Errorf("unknown status: %s", from)
    }
    if !s.IsValidStatus(to) {
        return fmt.Errorf("unknown status: %s", to)
    }

    // 3. Check transition is allowed in workflow
    if !s.IsValidTransition(from, to) {
        validNext := s.GetValidTransitions(from)
        return fmt.Errorf(
            "invalid transition from %s to %s. Valid next statuses: %v",
            from, to, validNext,
        )
    }

    // 4. Command-specific validation
    switch commandType {
    case "claim":
        if !strings.HasPrefix(from, "ready_for_") {
            return fmt.Errorf("can only claim tasks in ready_for_* status, current status: %s", from)
        }
        if !strings.HasPrefix(to, "in_") {
            return fmt.Errorf("claim must transition to in_* status, target status: %s", to)
        }
    case "finish":
        if !strings.HasPrefix(from, "in_") {
            return fmt.Errorf("can only finish tasks in in_* status, current status: %s", from)
        }
    case "reject":
        // Reject should move to earlier phase
        fromPhase := s.GetPhaseFromStatus(from)
        toPhase := s.GetPhaseFromStatus(to)
        if !isEarlierPhase(toPhase, fromPhase) && toPhase != fromPhase {
            return fmt.Errorf("reject should move to earlier phase, %s → %s is forward/lateral", from, to)
        }
    }

    return nil
}
```

---

## Error Handling

### Missing Configuration

```go
// If .sharkconfig.json doesn't exist or is invalid, fall back to default
workflow := config.GetWorkflowOrDefault(configPath)
if workflow == defaultWorkflow {
    log.Warn("Using default workflow (no .sharkconfig.json found)")
}
```

**Default Workflow**:
```json
{
  "status_flow": {
    "todo": ["in_progress", "cancelled"],
    "in_progress": ["ready_for_review", "todo"],
    "ready_for_review": ["completed", "in_progress"],
    "completed": [],
    "cancelled": []
  },
  "special_statuses": {
    "_start_": ["todo"],
    "_complete_": ["completed", "cancelled"]
  }
}
```

### Invalid Configuration

**Validation Errors Logged** (non-fatal):
```
WARN: Workflow validation failed: status 'in_qa' has no outgoing transitions
WARN: Using default workflow as fallback
```

**Fatal Errors** (command execution):
- Transition validation failure → show error, suggest --force
- Status not found → show error with valid statuses

---

## Performance Targets

### Caching Performance

**Cache Hit**:
- Target: < 1ms
- Actual: ~0.1ms (in-memory lookup)

**Cache Miss**:
- Target: < 50ms
- Breakdown:
  - File I/O: ~10ms
  - JSON parsing: ~20ms
  - Validation: ~10ms
  - Cache update: ~1ms

**Memory Usage**:
- Workflow config: ~50KB
- Cached service: ~100KB
- Acceptable for CLI tool

### Command Overhead

**Per-Command Workflow Access**:
- Cache hit (normal case): < 1ms
- Cache miss (config changed): < 50ms

**90th Percentile Target**:
- Total command time: < 500ms
- Workflow overhead: < 10% of total

---

## Testing Strategy

### Unit Tests

```go
func TestWorkflowService_GetNextPhaseStatus(t *testing.T) {
    tests := []struct{
        name           string
        currentStatus  string
        expectedNext   string
        expectError    bool
    }{
        {"development to review", "in_development", "ready_for_code_review", false},
        {"review to qa", "in_code_review", "ready_for_qa", false},
        {"approval to completed", "in_approval", "completed", false},
        {"terminal status", "completed", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            service := NewService("/tmp/test-project")
            next, err := service.GetNextPhaseStatus(tt.currentStatus)

            if tt.expectError && err == nil {
                t.Error("expected error but got none")
            }
            if !tt.expectError && err != nil {
                t.Errorf("unexpected error: %v", err)
            }
            if next != tt.expectedNext {
                t.Errorf("expected %s, got %s", tt.expectedNext, next)
            }
        })
    }
}
```

### Integration Tests

**Test Workflow Files**:
- `testdata/workflow-simple.json` - 5-state workflow
- `testdata/workflow-complex.json` - 10-state enterprise workflow
- `testdata/workflow-invalid.json` - intentionally broken config

```go
func TestWorkflowService_LoadAndValidate(t *testing.T) {
    // Load valid workflow
    service := NewServiceWithConfig("testdata/workflow-simple.json")
    errors := service.ValidateWorkflow()
    assert.Empty(t, errors, "valid workflow should have no errors")

    // Load invalid workflow
    service = NewServiceWithConfig("testdata/workflow-invalid.json")
    errors = service.ValidateWorkflow()
    assert.NotEmpty(t, errors, "invalid workflow should have errors")
}
```

### Caching Tests

```go
func TestWorkflowService_CachingBehavior(t *testing.T) {
    // First load
    start := time.Now()
    service1 := NewService("/tmp/test-project")
    duration1 := time.Since(start)

    // Second load (should be cached)
    start = time.Now()
    service2 := NewService("/tmp/test-project")
    duration2 := time.Since(start)

    assert.Same(t, service1, service2, "should return cached service")
    assert.Less(t, duration2, duration1/10, "cached load should be 10x faster")

    // Modify file and reload
    touchFile("/tmp/test-project/.sharkconfig.json")
    service3 := NewService("/tmp/test-project")
    assert.NotSame(t, service1, service3, "should reload after file modification")
}
```

---

## Migration Notes

### Existing Code Impact

**Files to Modify**:
- `internal/workflow/service.go` - Add new methods
- `internal/cli/commands/task_claim.go` - New file using service
- `internal/cli/commands/task_finish.go` - New file using service
- `internal/cli/commands/task_reject.go` - New file using service

**Backward Compatibility**:
- Existing `workflow.Service` methods unchanged
- New methods are additions only
- No breaking changes to API

### Default Workflow Support

**Guaranteed to Work**:
- Projects with no `.sharkconfig.json` use default workflow
- Default workflow supports claim/finish/reject commands
- Commands work identically with default or custom workflows

---

## References

- [System Architecture](./system-architecture.md)
- [Command Specifications](./command-specifications.md)
- [Transition Validation](./transition-validation.md)
- [Epic Requirements](../requirements.md) - REQ-F-004, REQ-F-005
