# Technical Design: Orchestrator Actions for Status Transitions

**Feature**: E07-F21 - Add Orchestrator Actions to Status Transitions
**Author**: Architect Agent
**Date**: 2026-01-15
**Status**: Architecture Review
**Version**: 1.0

---

## Executive Summary

This technical design specifies the architecture for adding orchestrator action metadata to Shark's status transition responses. The feature enables AI Agent Orchestrators to receive complete execution instructions in status transition responses, eliminating separate queries and reducing API calls by 50%.

**Core Architecture Pattern**: Configuration-driven action definitions with runtime template population and in-memory caching.

**Key Design Principles**:
- **Appropriate**: Uses proven configuration patterns (JSON schema, file-based config)
- **Proven**: Similar to GitHub Actions workflow syntax and Argo Workflows
- **Simple**: Minimalist implementation with clear separation of concerns

**Integration Points**:
- Configuration system (`.sharkconfig.json`)
- Task repository layer (status update methods)
- CLI command handlers (task update, start, complete, etc.)
- Template engine (variable substitution)

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Component Design](#2-component-design)
3. [Data Flow](#3-data-flow)
4. [Integration Points](#4-integration-points)
5. [Performance Considerations](#5-performance-considerations)
6. [Error Handling Strategy](#6-error-handling-strategy)
7. [Testing Architecture](#7-testing-architecture)
8. [Migration Strategy](#8-migration-strategy)
9. [Security Considerations](#9-security-considerations)
10. [Design Decisions](#10-design-decisions)

---

## 1. Architecture Overview

### 1.1 System Context

```
┌─────────────────────────────────────────────────────────────┐
│                    AI Agent Orchestrator                     │
└────────────┬────────────────────────────────────┬───────────┘
             │                                     │
             │ shark task update                   │ (optional)
             │ --status ready_for_development      │ shark task list
             │ --json                              │ --with-actions
             │                                     │
             ▼                                     ▼
┌─────────────────────────────────────────────────────────────┐
│                     Shark CLI Commands                       │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  task update │ task start │ task complete │ etc.       │ │
│  └─────────────┬──────────────────────────────────────────┘ │
└────────────────┼──────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│              Task Repository Layer                           │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  UpdateStatus(taskKey, newStatus)                      │ │
│  │    → (Task, OrchestratorAction, error)                 │ │
│  └─────────────┬──────────────────────────────────────────┘ │
└────────────────┼──────────────────────────────────────────┘
                 │
                 │ Query action for status
                 │ Populate template
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│              Config Service Layer                            │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  GetStatusActionPopulated(status, taskID)              │ │
│  │    → PopulatedAction                                   │ │
│  └─────────────┬──────────────────────────────────────────┘ │
└────────────────┼──────────────────────────────────────────┘
                 │
                 │ Read cached config
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│              In-Memory Cache                                 │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  status_metadata[status].orchestrator_action           │ │
│  └─────────────┬──────────────────────────────────────────┘ │
└────────────────┼──────────────────────────────────────────┘
                 │
                 │ Load on startup
                 │ Validate schema
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│              .sharkconfig.json                               │
│  {                                                           │
│    "status_metadata": {                                      │
│      "ready_for_development": {                             │
│        "orchestrator_action": {                             │
│          "action": "spawn_agent",                           │
│          "agent_type": "developer",                         │
│          "skills": ["tdd", "implementation"],               │
│          "instruction_template": "Implement {task_id}..."   │
│        }                                                     │
│      }                                                       │
│    }                                                         │
│  }                                                           │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 Component Layers

**Layer 1: Configuration Storage**
- `.sharkconfig.json` - JSON file with `orchestrator_action` in `status_metadata`
- Validated at load time (fail-fast)
- Version controlled with project

**Layer 2: Config Service**
- In-memory cache of workflow configuration
- Query API: `GetStatusAction()`, `GetStatusActionPopulated()`
- Template variable population
- Thread-safe with `sync.RWMutex`

**Layer 3: Repository Layer**
- Task status update methods
- Orchestrator action retrieval
- Integration point between business logic and config

**Layer 4: CLI Commands**
- Status transition handlers
- JSON/human-readable output formatting
- Error handling and user messaging

### 1.3 Design Patterns

**Pattern 1: Service Layer Pattern**
- Config service abstracts configuration access
- Enables mocking for tests
- Provides caching and performance optimization

**Pattern 2: Template Method Pattern**
- Abstract template rendering interface
- Simple string replacement implementation for v1
- Extensible to advanced template engines (Phase 4)

**Pattern 3: Strategy Pattern**
- Different action types (spawn_agent, pause, wait_for_triage, archive)
- Validated polymorphically via schema
- Handled generically in response serialization

---

## 2. Component Design

### 2.1 Configuration Schema

**File**: `internal/config/orchestrator_action.go`

#### OrchestratorAction Struct

```go
type OrchestratorAction struct {
    // Action type: spawn_agent, pause, wait_for_triage, archive
    Action string `json:"action" yaml:"action"`

    // Agent type to spawn (required for spawn_agent)
    AgentType string `json:"agent_type,omitempty" yaml:"agent_type,omitempty"`

    // Skills for agent (required for spawn_agent)
    Skills []string `json:"skills,omitempty" yaml:"skills,omitempty"`

    // Instruction template with {task_id} placeholder
    InstructionTemplate string `json:"instruction_template" yaml:"instruction_template"`
}

// Action type constants
const (
    ActionSpawnAgent    = "spawn_agent"
    ActionPause         = "pause"
    ActionWaitForTriage = "wait_for_triage"
    ActionArchive       = "archive"
)
```

#### Validation Method

```go
// Validate checks OrchestratorAction schema correctness
func (oa *OrchestratorAction) Validate() error {
    // 1. Validate action enum
    validActions := []string{ActionSpawnAgent, ActionPause, ActionWaitForTriage, ActionArchive}
    if !contains(validActions, oa.Action) {
        return fmt.Errorf("invalid action type: %s", oa.Action)
    }

    // 2. Validate instruction_template (required for all)
    if strings.TrimSpace(oa.InstructionTemplate) == "" {
        return errors.New("instruction_template is required")
    }

    // 3. Validate spawn_agent requirements
    if oa.Action == ActionSpawnAgent {
        if strings.TrimSpace(oa.AgentType) == "" {
            return errors.New("agent_type required for spawn_agent")
        }
        if len(oa.Skills) == 0 {
            return errors.New("skills required for spawn_agent")
        }
    }

    return nil
}
```

#### StatusMetadata Extension

```go
type StatusMetadata struct {
    Color       string              `json:"color" yaml:"color"`
    Description string              `json:"description" yaml:"description"`
    Phase       string              `json:"phase" yaml:"phase"`
    AgentTypes  []string            `json:"agent_types,omitempty" yaml:"agent_types,omitempty"`

    // NEW FIELD
    OrchestratorAction *OrchestratorAction `json:"orchestrator_action,omitempty" yaml:"orchestrator_action,omitempty"`
}
```

**Key Design Decisions**:
- Optional field (`omitempty`) for backward compatibility
- Pointer type allows nil distinction (no action vs. empty action)
- Validation at config load time (fail-fast)

---

### 2.2 Template Engine

**File**: `internal/template/renderer.go`

#### Interface

```go
package template

// TemplateRenderer renders instruction templates with variable substitution
type TemplateRenderer interface {
    // Render replaces variables in template with context values
    Render(template string, context map[string]string) string
}

// NewRenderer creates default simple string renderer
func NewRenderer() TemplateRenderer {
    return &simpleRenderer{}
}
```

#### Simple Implementation (Phase 1)

```go
type simpleRenderer struct{}

func (r *simpleRenderer) Render(template string, context map[string]string) string {
    result := template
    for key, value := range context {
        placeholder := fmt.Sprintf("{%s}", key)
        result = strings.ReplaceAll(result, placeholder, value)
    }
    return result
}
```

**Phase 1 Support**: `{task_id}` only
**Future**: `{epic_id}`, `{feature_id}`, `{task_title}`, conditional logic

**Performance**: <1ms per render (simple string replacement, no regex)

---

### 2.3 Config Service Layer

**File**: `internal/config/action_service.go`

#### Service Interface

```go
type ActionService interface {
    // GetStatusAction returns raw action for status (nil if not defined)
    GetStatusAction(ctx context.Context, status string) (*OrchestratorAction, error)

    // GetStatusActionPopulated returns action with {task_id} populated
    GetStatusActionPopulated(ctx context.Context, status string, taskID string) (*PopulatedAction, error)

    // GetAllActions returns all actions indexed by status
    GetAllActions(ctx context.Context) (map[string]*OrchestratorAction, error)

    // ValidateActions checks all actions are valid
    ValidateActions(ctx context.Context) (*ValidationResult, error)

    // Reload forces config reload from disk
    Reload(ctx context.Context) error
}

// PopulatedAction is action with template variables replaced
type PopulatedAction struct {
    Action      string   `json:"action"`
    AgentType   string   `json:"agent_type,omitempty"`
    Skills      []string `json:"skills,omitempty"`
    Instruction string   `json:"instruction"` // Template populated
}
```

#### Cache Implementation

```go
type DefaultActionService struct {
    mu             sync.RWMutex      // Protects workflow field
    configPath     string
    workflow       *WorkflowConfig   // Cached config
    templateEngine TemplateRenderer
}

func (s *DefaultActionService) GetStatusActionPopulated(ctx context.Context, status string, taskID string) (*PopulatedAction, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    // 1. Lookup action from cached config
    metadata, exists := s.workflow.StatusMetadata[status]
    if !exists {
        return nil, &StatusNotFoundError{Status: status}
    }

    if metadata.OrchestratorAction == nil {
        return nil, nil // No action - not an error
    }

    action := metadata.OrchestratorAction

    // 2. Populate template
    context := map[string]string{"task_id": taskID}
    instruction := s.templateEngine.Render(action.InstructionTemplate, context)

    // 3. Return populated action
    return &PopulatedAction{
        Action:      action.Action,
        AgentType:   action.AgentType,
        Skills:      action.Skills,
        Instruction: instruction,
    }, nil
}
```

**Caching Benefits**:
- Initial load: <100ms (parse + validate JSON)
- Cached queries: <1ms (map lookup + string replacement)
- Thread-safe read/write with RWMutex
- Reload on config changes (manual or file watch)

---

### 2.4 Repository Layer Integration

**File**: `internal/repository/task_repository.go`

#### Enhanced UpdateStatus Method

```go
// UpdateStatus updates task status and returns orchestrator action
func (r *TaskRepository) UpdateStatus(ctx context.Context, taskKey string, newStatus string) (*models.Task, *models.OrchestratorAction, error) {
    // 1. Begin transaction
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, nil, fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback()

    // 2. Update task status in database
    task, err := r.updateTaskStatus(ctx, tx, taskKey, newStatus)
    if err != nil {
        return nil, nil, fmt.Errorf("update status: %w", err)
    }

    // 3. Commit transaction
    if err := tx.Commit(); err != nil {
        return nil, nil, fmt.Errorf("commit transaction: %w", err)
    }

    // 4. Get orchestrator action (after commit - not part of transaction)
    action, err := r.getOrchestratorAction(ctx, task, newStatus)
    if err != nil {
        // Log warning but don't fail - action is optional
        log.Warnf("Failed to get orchestrator action for status %s: %v", newStatus, err)
        return task, nil, nil
    }

    return task, action, nil
}

// getOrchestratorAction retrieves and populates action for status
func (r *TaskRepository) getOrchestratorAction(ctx context.Context, task *models.Task, status string) (*models.OrchestratorAction, error) {
    // Query config service
    actionService, err := r.configManager.GetActionService()
    if err != nil {
        return nil, fmt.Errorf("get action service: %w", err)
    }

    // Get populated action
    populatedAction, err := actionService.GetStatusActionPopulated(ctx, status, task.Key)
    if err != nil {
        return nil, fmt.Errorf("get status action: %w", err)
    }

    if populatedAction == nil {
        return nil, nil // No action defined
    }

    // Convert to models.OrchestratorAction
    return &models.OrchestratorAction{
        Action:      populatedAction.Action,
        AgentType:   populatedAction.AgentType,
        Skills:      populatedAction.Skills,
        Instruction: populatedAction.Instruction,
    }, nil
}
```

**Key Design Points**:
- Action retrieval outside transaction (not DB operation)
- Graceful degradation (missing action returns nil, not error)
- Config service dependency injection via repository constructor

---

### 2.5 CLI Response Enhancement

**File**: `internal/cli/commands/task.go`

#### JSON Response with OrchestratorAction

```go
// TaskUpdateResponse includes optional orchestrator action
type TaskUpdateResponse struct {
    *models.Task
    OrchestratorAction *models.OrchestratorAction `json:"orchestrator_action,omitempty"`
}

func runTaskUpdate(cmd *cobra.Command, args []string) error {
    // ... parse arguments ...

    // Update task status
    task, action, err := repo.UpdateStatus(ctx, taskKey, newStatus)
    if err != nil {
        return fmt.Errorf("update failed: %w", err)
    }

    // Build response
    response := &TaskUpdateResponse{
        Task:                task,
        OrchestratorAction: action, // nil if no action defined
    }

    // Output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(response)
    }

    // Human-readable output
    displayTaskUpdate(task)
    displayOrchestratorAction(action)
    return nil
}
```

#### Human-Readable Display

```go
func displayOrchestratorAction(action *models.OrchestratorAction) {
    if action == nil {
        cli.Info("Next Action: None configured")
        return
    }

    cli.Info("Next Action:")
    cli.Infof("  Type: %s", action.Action)

    if action.AgentType != "" {
        cli.Infof("  Agent: %s", action.AgentType)
    }

    if len(action.Skills) > 0 {
        cli.Infof("  Skills: %s", strings.Join(action.Skills, ", "))
    }

    // Truncate instruction for display
    instruction := action.Instruction
    if len(instruction) > 100 {
        instruction = instruction[:97] + "..."
    }
    cli.Infof("  Instruction: %s", instruction)
}
```

**Backward Compatibility**:
- `omitempty` tag omits field when nil
- Existing orchestrators ignore unknown field
- Field presence check: `if response.OrchestratorAction != nil { ... }`

---

## 3. Data Flow

### 3.1 Configuration Load (Startup)

```
[1] Shark CLI Starts
     │
     ▼
[2] Load .sharkconfig.json
     │
     ▼
[3] Parse status_metadata
     │
     ▼
[4] Validate orchestrator_action schemas
     │  - Check action enum
     │  - Check required fields
     │  - Check template syntax
     │
     ├─ Valid ────────▶ [5] Cache in memory
     │                       │
     │                       ▼
     │                  [6] Ready for queries
     │
     └─ Invalid ─────▶ [7] Print validation errors
                           │
                           ▼
                      [8] Exit with code 2
```

**Performance**: <100ms for typical config (10KB JSON, 15 statuses)

---

### 3.2 Status Transition (Runtime)

```
[1] Agent/Orchestrator: shark task update T-E07-F21-001 --status ready_for_development
     │
     ▼
[2] CLI Command Handler
     │  - Parse arguments
     │  - Validate task key
     │
     ▼
[3] Repository.UpdateStatus(task, newStatus)
     │
     ├─ [3a] Begin transaction
     │
     ├─ [3b] UPDATE tasks SET status = ? WHERE key = ?
     │
     ├─ [3c] INSERT INTO task_history (...)
     │
     ├─ [3d] Commit transaction
     │
     └─ [3e] Query orchestrator action ────┐
                                           │
                                           ▼
[4] ConfigService.GetStatusActionPopulated("ready_for_development", "T-E07-F21-001")
     │
     ├─ [4a] Lookup cached status_metadata
     │
     ├─ [4b] Extract orchestrator_action
     │
     └─ [4c] Template.Render(instruction_template, {"task_id": "T-E07-F21-001"})
          │
          ▼
[5] Return (Task, PopulatedAction)
     │
     ▼
[6] CLI Serialize to JSON
     │
     ▼
[7] Output Response
     {
       "key": "T-E07-F21-001",
       "status": "ready_for_development",
       "orchestrator_action": {
         "action": "spawn_agent",
         "agent_type": "developer",
         "skills": ["tdd", "implementation"],
         "instruction": "Implement task T-E07-F21-001..."
       }
     }
```

**Performance**: <10ms additional latency (config service: 1ms, template: <1ms)

---

### 3.3 Error Scenarios

**Scenario 1: No Orchestrator Action Defined**
```
Status transition → Repository update success
                  → ConfigService returns nil (no action)
                  → Response omits orchestrator_action field
                  → Success (backward compatible)
```

**Scenario 2: Config Service Error**
```
Status transition → Repository update success
                  → ConfigService query fails
                  → Log warning
                  → Response omits orchestrator_action field
                  → Success (graceful degradation)
```

**Scenario 3: Invalid Action Schema (Load Time)**
```
Shark startup → Parse .sharkconfig.json
              → Validate orchestrator_action
              → Validation fails (missing agent_type)
              → Print error: "Status 'ready_for_development' missing agent_type for spawn_agent"
              → Exit code 2 (fail-fast)
```

---

## 4. Integration Points

### 4.1 Existing Shark Architecture

**Config Package** (`internal/config/`)
- **Extension Point**: StatusMetadata struct
- **New Components**: OrchestratorAction struct, ActionService interface
- **Change Impact**: Low (additive changes, optional field)

**Repository Package** (`internal/repository/`)
- **Extension Point**: UpdateStatus method signature
- **New Components**: getOrchestratorAction helper method
- **Change Impact**: Medium (signature change, but graceful fallback)

**CLI Package** (`internal/cli/commands/`)
- **Extension Point**: Task command response serialization
- **New Components**: TaskUpdateResponse struct, displayOrchestratorAction helper
- **Change Impact**: Low (additive, backward compatible JSON)

### 4.2 External Integration (Orchestrator)

**API Contract**:
```json
{
  "orchestrator_action": {
    "action": "spawn_agent | pause | wait_for_triage | archive",
    "agent_type": "string (required for spawn_agent)",
    "skills": ["array", "of", "strings"],
    "instruction": "populated template string"
  }
}
```

**Orchestrator Usage**:
```go
// Parse response
var response TaskUpdateResponse
json.Unmarshal(output, &response)

// Check for action
if response.OrchestratorAction != nil {
    switch response.OrchestratorAction.Action {
    case "spawn_agent":
        SpawnAgent(
            response.OrchestratorAction.AgentType,
            response.OrchestratorAction.Skills,
            response.OrchestratorAction.Instruction,
        )
    case "pause":
        log.Info("Task paused:", response.OrchestratorAction.Instruction)
    // ... handle other actions
    }
}
```

**Backward Compatibility**:
- Field omitted when nil (not null)
- Existing orchestrators ignore unknown field
- Gradual adoption (add actions incrementally)

---

## 5. Performance Considerations

### 5.1 Performance Requirements

**From PRD**:
- REQ-NF-001: Response time impact <10ms
- REQ-NF-002: Config load time <100ms

### 5.2 Performance Analysis

**Config Load (Once at Startup)**:
- JSON parse: ~30ms (10KB config)
- Schema validation: ~20ms (15 statuses × 1ms)
- Cache initialization: ~5ms
- **Total: ~55ms** ✅ (target: <100ms)

**Runtime Query (Per Transition)**:
- Status lookup in map: ~0.1ms
- Template rendering: ~0.5ms (simple string replace)
- Object construction: ~0.1ms
- **Total: ~0.7ms** ✅ (target: <10ms)

**Memory Overhead**:
- Cached config: ~50KB (typical workflow)
- Per-status action: ~200 bytes
- Total: <100KB for full workflow

### 5.3 Optimization Strategies

**Caching**:
- In-memory cache eliminates file I/O
- Map lookups are O(1)
- Config immutable after load (thread-safe reads)

**Template Engine**:
- Simple string replacement (no regex)
- Single pass over template string
- Context map pre-built

**Lazy Loading** (Future):
- Load config only when first action query occurs
- Trade startup time for first-query latency

---

## 6. Error Handling Strategy

### 6.1 Error Categories

**Category 1: Configuration Errors (Load Time)**
- Invalid action type enum
- Missing required fields (agent_type for spawn_agent)
- Empty instruction_template
- Malformed JSON

**Handling**: Fail-fast at config load, exit code 2, clear error message

**Category 2: Runtime Errors**
- Status not found in config
- Config service unavailable
- Template population failure

**Handling**: Log warning, graceful degradation (omit action from response)

**Category 3: Optional Field (Not an Error)**
- orchestrator_action not defined for status

**Handling**: Return nil, omit field from JSON, no error or warning

### 6.2 Error Messages

**Good Error Message**:
```
Error: Invalid orchestrator_action in status 'ready_for_development'
  Field: agent_type
  Problem: Missing required field for spawn_agent action
  Fix: Add "agent_type": "developer" to orchestrator_action
```

**Bad Error Message**:
```
Error: validation failed
```

**Error Message Template**:
```
Error: <brief description>
  Field: <field name>
  Problem: <what's wrong>
  Fix: <how to resolve>
```

### 6.3 Graceful Degradation

**Principle**: Missing orchestrator actions don't break Shark functionality.

**Implementation**:
- Config service returns nil for missing actions (not error)
- Repository continues after action query failure
- CLI omits field from response (backward compatible)
- Orchestrators implement fallback logic

**Example**:
```go
action, err := configService.GetStatusAction(ctx, status)
if err != nil {
    log.Warnf("Failed to get action: %v", err)
    action = nil // Continue without action
}

// Response serialization handles nil action
response := TaskUpdateResponse{
    Task:                task,
    OrchestratorAction: action, // nil omitted by omitempty
}
```

---

## 7. Testing Architecture

### 7.1 Test Strategy

**Unit Tests** (Fast, Isolated)
- Config parsing and validation
- Template rendering
- Action service queries
- Repository action retrieval

**Integration Tests** (Real Components)
- Config load with orchestrator_action
- Repository + Config service integration
- CLI command + Repository integration

**End-to-End Tests** (Full Stack)
- Task status transition with action response
- Orchestrator workflow simulation
- Backward compatibility verification

### 7.2 Testing Pyramid

```
                     ▲
                    / \
                   /   \
                  /  E2E \    (5 tests)
                 /       \
                /---------\
               /Integration\  (20 tests)
              /             \
             /---------------\
            /   Unit Tests    \ (100 tests)
           /___________________\
```

**Unit Tests (80%)**: Fast feedback, high coverage of individual components

**Integration Tests (15%)**: Verify component interactions

**E2E Tests (5%)**: Critical paths, orchestrator workflows

### 7.3 Mocking Strategy

**Config Service Mocking** (CLI Tests):
```go
type MockActionService struct {
    GetStatusActionPopulatedFunc func(ctx, status, taskID string) (*PopulatedAction, error)
}

func TestTaskUpdate_WithAction(t *testing.T) {
    mockService := &MockActionService{
        GetStatusActionPopulatedFunc: func(ctx, status, taskID string) (*PopulatedAction, error) {
            return &PopulatedAction{
                Action: "spawn_agent",
                AgentType: "developer",
            }, nil
        },
    }

    // Test CLI command with mock service
}
```

**Repository Mocking** (Integration Tests):
```go
type MockTaskRepository struct {
    UpdateStatusFunc func(ctx, taskKey, status string) (*Task, *OrchestratorAction, error)
}
```

---

## 8. Migration Strategy

### 8.1 Backward Compatibility

**Principle**: Existing Shark installations continue to work unchanged.

**Implementation**:
1. `orchestrator_action` is optional field (`omitempty` tag)
2. Missing actions omit field from response (not null)
3. Config validation only runs when field present
4. Repository gracefully handles missing config service

**Compatibility Matrix**:

| Shark Version | Config Without Action | Config With Action | Orchestrator |
|---------------|----------------------|--------------------|--------------|
| v1.x (old)    | ✅ Works             | ⚠️ Ignores field    | Not supported |
| v2.0 (new)    | ✅ Works             | ✅ Returns action   | ✅ Receives action |

### 8.2 Migration Path

**Phase 1: Deploy Shark v2.0** (Week 1)
- Install new Shark version with orchestrator_action support
- Existing configs work unchanged
- No orchestrator changes yet

**Phase 2: Add Orchestrator Actions to Config** (Week 2)
- Update `.sharkconfig.json` with orchestrator_action for key statuses
- Validate with `shark workflow validate-actions`
- Test with `shark config get-status-action ready_for_development`

**Phase 3: Update Orchestrators** (Week 3)
- Modify orchestrators to parse `orchestrator_action` field
- Implement action handling (spawn_agent, pause, etc.)
- Test with real workflows

**Phase 4: Remove Hardcoded Logic** (Week 4)
- Remove hardcoded workflow mappings from orchestrators
- Rely entirely on action responses
- Monitor for issues

### 8.3 Rollback Plan

**If Issues Arise**:
1. Orchestrators can ignore `orchestrator_action` field (backward compatible)
2. Remove orchestrator_action from configs (revert to old behavior)
3. Rollback to Shark v1.x if critical issues
4. Keep hardcoded fallback in orchestrators during transition

---

## 9. Security Considerations

### 9.1 Configuration Security

**Threat**: Malicious config injection

**Mitigation**:
- `.sharkconfig.json` is project file (not user input)
- Version controlled with code
- Config validation at load time

**Best Practice**:
- Review config changes in pull requests
- Use `shark workflow validate-actions` in CI/CD
- Restrict config file permissions (0644)

### 9.2 Template Injection

**Threat**: Template injection via task_id

**Analysis**:
- `{task_id}` is controlled value (Shark-generated key like "T-E07-F21-001")
- No user input directly in template
- Simple string replacement (not eval)

**Mitigation**:
- Validate task_id format before template population
- Use allowlist for template variables
- Phase 2+: Escape special characters if dynamic templates added

### 9.3 Instruction Content

**Consideration**: instructions may contain sensitive information

**Guidance**:
- Avoid secrets in instruction templates
- Do not include credentials, API keys, or PII
- Instructions logged for debugging (ensure no sensitive data)

---

## 10. Design Decisions

### 10.1 Why File-Based Config (Not Database)?

**Decision**: Store orchestrator_action in `.sharkconfig.json` (not database)

**Rationale**:
- ✅ Version controlled with project
- ✅ Easy to edit and review (JSON/YAML)
- ✅ Validated at load time (fail-fast)
- ✅ Standard practice for workflow configs (GitHub Actions, Argo)
- ✅ No migration needed for database schema
- ❌ Less dynamic than database storage

**Future**: Phase 3 may add database-backed workflows for versioning/history

### 10.2 Why Omit Field (Not Return Null)?

**Decision**: Omit `orchestrator_action` field when nil (not return null)

**Rationale**:
- ✅ More idiomatic JSON (missing fields = not applicable)
- ✅ Reduces response payload size
- ✅ Easier orchestrator check: `if (action)` vs `if (action !== null)`
- ✅ Go's `omitempty` tag handles naturally

**Implementation**: `json:"orchestrator_action,omitempty"`

### 10.3 Why Simple Template Engine (Not text/template)?

**Decision**: Use simple string replacement for Phase 1

**Rationale**:
- ✅ Sufficient for `{task_id}` use case
- ✅ No external dependencies
- ✅ <1ms performance
- ✅ Easy to understand and debug
- ✅ Future-proof (can upgrade to text/template in Phase 4)

**Trade-off**: No conditionals, loops, or functions (yet)

### 10.4 Why In-Memory Cache (Not Redis)?

**Decision**: Cache workflow config in memory with `sync.RWMutex`

**Rationale**:
- ✅ Simple implementation (no Redis dependency)
- ✅ Fast queries (<1ms map lookup)
- ✅ Config is small (~50KB) and immutable
- ✅ Thread-safe read/write with RWMutex

**Trade-off**: Manual reload needed for config changes (acceptable)

### 10.5 Why Action Outside Transaction?

**Decision**: Query orchestrator action after database commit

**Rationale**:
- ✅ Action retrieval not a database operation (config service)
- ✅ Failed action query doesn't rollback status update
- ✅ Graceful degradation (update succeeds even if action fails)
- ✅ Simpler transaction management

**Implementation**: `UpdateStatus` commits transaction before calling `getOrchestratorAction`

---

## Appendix A: File Structure

```
internal/
├── config/
│   ├── orchestrator_action.go              (NEW - T-E07-F21-001)
│   ├── orchestrator_action_test.go         (NEW - T-E07-F21-001)
│   ├── action_service.go                   (NEW - T-E07-F21-004)
│   ├── action_service_test.go              (NEW - T-E07-F21-004)
│   ├── mock_action_service.go              (NEW - T-E07-F21-004)
│   └── config.go                           (MODIFY - extend StatusMetadata)
│
├── template/
│   ├── renderer.go                         (NEW - T-E07-F21-002)
│   └── renderer_test.go                    (NEW - T-E07-F21-002)
│
├── repository/
│   ├── task_repository.go                  (MODIFY - UpdateStatus signature)
│   └── task_repository_test.go             (MODIFY - test action return)
│
├── cli/commands/
│   ├── task.go                             (MODIFY - T-E07-F21-006)
│   ├── task_test.go                        (MODIFY - test with mocks)
│   └── workflow.go                         (NEW - validate-actions command)
│
└── models/
    └── task.go                             (MODIFY - add OrchestratorAction)
```

---

## Appendix B: API Reference

### Config Service API

```go
// ActionService provides orchestrator action queries
type ActionService interface {
    GetStatusAction(ctx context.Context, status string) (*OrchestratorAction, error)
    GetStatusActionPopulated(ctx context.Context, status, taskID string) (*PopulatedAction, error)
    GetAllActions(ctx context.Context) (map[string]*OrchestratorAction, error)
    ValidateActions(ctx context.Context) (*ValidationResult, error)
    Reload(ctx context.Context) error
}
```

### Repository API

```go
// UpdateStatus updates task status and returns orchestrator action
func (r *TaskRepository) UpdateStatus(
    ctx context.Context,
    taskKey string,
    newStatus string,
) (*models.Task, *models.OrchestratorAction, error)
```

### CLI Response API

```json
{
  "key": "T-E07-F21-001",
  "status": "ready_for_development",
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "developer",
    "skills": ["test-driven-development", "implementation"],
    "instruction": "Launch a developer agent to implement task T-E07-F21-001..."
  }
}
```

---

## Appendix C: Configuration Examples

### Minimal Example

```json
{
  "status_metadata": {
    "ready_for_development": {
      "color": "yellow",
      "phase": "development",
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "developer",
        "skills": ["implementation"],
        "instruction_template": "Implement task {task_id}"
      }
    }
  }
}
```

### Complete Example (All Action Types)

```json
{
  "status_metadata": {
    "ready_for_development": {
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "developer",
        "skills": ["tdd", "implementation", "shark"],
        "instruction_template": "Launch developer agent for task {task_id}. Write tests first."
      }
    },
    "blocked": {
      "orchestrator_action": {
        "action": "pause",
        "instruction_template": "Task {task_id} blocked. Do not spawn agent."
      }
    },
    "draft": {
      "orchestrator_action": {
        "action": "wait_for_triage",
        "instruction_template": "Task {task_id} needs triage. Awaiting human decision."
      }
    },
    "completed": {
      "orchestrator_action": {
        "action": "archive",
        "instruction_template": "Task {task_id} completed. No further action needed."
      }
    }
  }
}
```

---

## Appendix D: Performance Benchmarks

**Target Metrics**:
- Config load: <100ms
- Cached query: <1ms
- Template render: <1ms
- Total latency impact: <10ms

**Benchmark Results** (Expected):

```
BenchmarkConfigLoad-8                 20     50ms/op
BenchmarkGetStatusAction-8         10000      0.1ms/op
BenchmarkTemplateRender-8          50000      0.02ms/op
BenchmarkUpdateStatusWithAction-8   5000      2ms/op
```

**Memory Usage**:
- Cached config: ~50KB
- Per-action overhead: ~200 bytes
- Total: <100KB for typical workflow

---

## Conclusion

This technical design provides a solid architectural foundation for orchestrator actions feature. Key strengths:

✅ **Separation of Concerns**: Config, service, repository, CLI layers clearly defined
✅ **Performance**: In-memory caching meets <10ms latency requirement
✅ **Backward Compatibility**: Optional field, graceful degradation
✅ **Testability**: Service interface enables comprehensive mocking
✅ **Simplicity**: Minimal complexity, easy to understand and maintain
✅ **Extensibility**: Template engine and action types easily extended

**Ready for Implementation**: All 13 tasks have clear specifications aligned with this architecture.

---

**Document Status**: ✅ Architecture Review Complete
**Next Steps**: Proceed with task implementation (T-E07-F21-001 through T-E07-F21-013)
