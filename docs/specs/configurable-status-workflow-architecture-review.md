# Configurable Status Workflow - Architecture Review

**Version**: 1.0
**Date**: 2025-12-29
**Reviewer**: Architect Agent
**Specification**: `/home/jwwelbor/projects/shark-task-manager/docs/specs/configurable-status-workflow.md`

---

## Executive Summary

The configurable status workflow specification is **architecturally sound** and well-aligned with the project's clean architecture principles. The design leverages configuration-driven behavior to replace hardcoded state transitions, which is appropriate for the stated goals of multi-agent collaboration and flexible workflows.

**Overall Assessment**: ✅ **APPROVED with recommendations**

**Key Strengths**:
- Minimal database schema changes (no breaking changes)
- Proper separation of concerns (config layer, repository layer, CLI layer)
- Backward compatibility via migration strategy
- `--force` escape hatch for operational flexibility
- Comprehensive testing strategy

**Key Concerns**:
1. Configuration loading strategy needs refinement
2. Repository initialization pattern will require careful refactoring
3. Transaction safety during status transitions must be preserved
4. Validation error messages need enhancement for developer experience

---

## 1. Technical Design Analysis

### 1.1 Configuration Schema Design ✅ **APPROVED**

**Strengths**:
- Uses existing `.sharkconfig.json` infrastructure (already parsed and loaded)
- JSON schema is clear and self-documenting
- Special keys (`_start_`, `_complete_`) provide explicit workflow boundaries
- Status metadata (color, description, phase, agent_types) supports rich UX

**Concerns**:
- **No schema validation**: Missing JSON schema definition for validation
- **No version field**: Config evolution will be difficult without versioning
- **Large config**: 14 statuses with transitions = ~50 lines of config (may overwhelm users)

**Recommendations**:
1. **Add config version field**:
   ```json
   {
     "status_flow_version": "1.0",
     "status_flow": { ... }
   }
   ```

2. **Provide schema validation** using JSON Schema or Go struct tags:
   ```go
   type WorkflowConfig struct {
       Version        string                    `json:"status_flow_version" validate:"required"`
       StatusFlow     StatusFlow                `json:"status_flow" validate:"required"`
       StatusMetadata map[string]StatusMetadata `json:"status_metadata,omitempty"`
   }
   ```

3. **Ship with multiple preset workflows** in `docs/workflows/`:
   - `simple.json` (3 statuses: todo → in_progress → done)
   - `kanban.json` (5 statuses)
   - `full.json` (14 statuses as shown in spec)
   - Users can copy and customize

---

### 1.2 Database Schema Changes ✅ **APPROVED (No Changes Required)**

**Analysis**:
The spec correctly identifies that **no schema changes are needed**. The `tasks.status` column is already `TEXT NOT NULL`, which supports arbitrary status values.

**Current Schema**:
```sql
CREATE TABLE tasks (
    ...
    status TEXT NOT NULL DEFAULT 'todo',
    ...
);
```

**Trigger Proposal** (from spec):
The spec proposes a validation trigger, but **I recommend against it** for these reasons:

1. **Validation belongs in application layer** (already proposed in repository layer)
2. **Database triggers add hidden behavior** that's hard to debug
3. **Config-driven validation cannot be enforced at DB level** without dynamic SQL
4. **Force flag would require trigger bypass mechanism** (complex)

**Recommendation**:
- ❌ **Do NOT implement the validation trigger**
- ✅ **Keep all validation in `TaskRepository.UpdateStatusForced()`**
- ✅ **Existing task_history trigger already records changes** (sufficient)

---

### 1.3 Configuration Management Package ✅ **APPROVED**

**Proposed Package**: `internal/config/workflow.go`

**Strengths**:
- Clear API surface (`LoadWorkflowConfig`, `CanTransition`, `GetValidNextStatuses`)
- Proper error handling signatures
- Logical home in `internal/config/` package

**Concerns**:
1. **File path coupling**: Hardcoded `.sharkconfig.json` path in spec
2. **No caching strategy**: Config will be loaded on every command
3. **No reload mechanism**: Config changes require process restart
4. **Missing validation**: No check for unreachable statuses or circular references

**Recommendations**:

#### 1.3.1 Configuration Loading Strategy

**Current Pattern** (from existing codebase):
```go
// internal/cli/root.go
func InitConfig() {
    configPath := filepath.Join(projectRoot, ".sharkconfig.json")
    // Config loaded once at CLI startup
}
```

**Proposed Pattern**:
```go
package config

var (
    workflowConfigCache *WorkflowConfig
    workflowConfigMutex sync.RWMutex
)

// LoadWorkflowConfig loads and caches workflow configuration
func LoadWorkflowConfig(configPath string) (*WorkflowConfig, error) {
    workflowConfigMutex.RLock()
    if workflowConfigCache != nil {
        defer workflowConfigMutex.RUnlock()
        return workflowConfigCache, nil
    }
    workflowConfigMutex.RUnlock()

    workflowConfigMutex.Lock()
    defer workflowConfigMutex.Unlock()

    // Double-check after acquiring write lock
    if workflowConfigCache != nil {
        return workflowConfigCache, nil
    }

    // Load from file
    cfg, err := loadWorkflowFromFile(configPath)
    if err != nil {
        // Fall back to default workflow
        cfg = DefaultWorkflowConfig()
    }

    // Validate config before caching
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid workflow config: %w", err)
    }

    workflowConfigCache = cfg
    return cfg, nil
}

// ReloadWorkflowConfig forces a reload from disk
func ReloadWorkflowConfig(configPath string) (*WorkflowConfig, error) {
    workflowConfigMutex.Lock()
    defer workflowConfigMutex.Unlock()

    workflowConfigCache = nil
    return LoadWorkflowConfig(configPath)
}
```

#### 1.3.2 Configuration Validation

**Add comprehensive validation**:
```go
// Validate checks workflow configuration for common errors
func (w *WorkflowConfig) Validate() error {
    // Check for _start_ and _complete_ special keys
    if _, ok := w.StatusFlow["_start_"]; !ok {
        return fmt.Errorf("missing required key: _start_")
    }
    if _, ok := w.StatusFlow["_complete_"]; !ok {
        return fmt.Errorf("missing required key: _complete_")
    }

    // Build set of all defined statuses
    definedStatuses := make(map[string]bool)
    for status := range w.StatusFlow {
        if !strings.HasPrefix(status, "_") {
            definedStatuses[status] = false // false = not yet reached
        }
    }

    // Validate transitions reference only defined statuses
    for from, toList := range w.StatusFlow {
        if strings.HasPrefix(from, "_") {
            continue // Skip special keys
        }
        for _, to := range toList {
            if !definedStatuses[to] {
                return fmt.Errorf("status %q transitions to undefined status %q", from, to)
            }
        }
    }

    // Check for unreachable statuses (graph reachability)
    startStatuses := w.StatusFlow["_start_"]
    reachable := make(map[string]bool)
    queue := append([]string{}, startStatuses...)

    for len(queue) > 0 {
        current := queue[0]
        queue = queue[1:]

        if reachable[current] {
            continue
        }
        reachable[current] = true

        nextStatuses := w.StatusFlow[current]
        queue = append(queue, nextStatuses...)
    }

    // Warn about unreachable statuses
    for status := range definedStatuses {
        if !reachable[status] {
            // Non-fatal warning
            fmt.Fprintf(os.Stderr, "WARNING: status %q is not reachable from any _start_ status\n", status)
        }
    }

    return nil
}
```

#### 1.3.3 Default Workflow Strategy

**Provide a sensible default** that matches current behavior:
```go
// DefaultWorkflowConfig returns a workflow matching the current hardcoded behavior
func DefaultWorkflowConfig() *WorkflowConfig {
    return &WorkflowConfig{
        StatusFlow: StatusFlow{
            "_start_":    []string{"todo"},
            "_complete_": []string{"completed", "archived"},

            "todo":             []string{"in_progress", "blocked"},
            "in_progress":      []string{"ready_for_review", "blocked", "todo"},
            "blocked":          []string{"todo"},
            "ready_for_review": []string{"completed", "in_progress"},
            "completed":        []string{"archived"},
            "archived":         []string{},
        },
        StatusMetadata: map[string]StatusMetadata{
            "todo":             {Color: "white", Description: "Not yet started", Phase: "planning"},
            "in_progress":      {Color: "yellow", Description: "Work in progress", Phase: "development"},
            "blocked":          {Color: "red", Description: "Blocked by dependency", Phase: "any"},
            "ready_for_review": {Color: "green", Description: "Ready for review", Phase: "review"},
            "completed":        {Color: "blue", Description: "Completed", Phase: "done"},
            "archived":         {Color: "gray", Description: "Archived", Phase: "done"},
        },
    }
}
```

---

### 1.4 Repository Layer Changes ⚠️ **APPROVED with CRITICAL refactoring**

**Current Implementation**:
```go
// internal/repository/task_repository.go (lines 452-509)
func isValidTransition(from models.TaskStatus, to models.TaskStatus) bool {
    validTransitions := map[models.TaskStatus][]models.TaskStatus{
        models.TaskStatusTodo: {
            models.TaskStatusInProgress,
            models.TaskStatusBlocked,
        },
        // ... hardcoded map
    }
    // ...
}

// Used in UpdateStatusForced (line 547):
if !isValidTransition(currentTaskStatus, newStatus) {
    return fmt.Errorf("invalid status transition from %s to %s", currentStatus, newStatus)
}
```

**Proposed Change**:
```go
type TaskRepository struct {
    db       *db.DB
    workflow *config.WorkflowConfig  // NEW
}

func NewTaskRepository(database *db.DB) *TaskRepository {
    cfg, err := config.LoadWorkflowConfig(".sharkconfig.json")
    if err != nil {
        cfg = config.DefaultWorkflowConfig()
    }

    return &TaskRepository{
        db:       database,
        workflow: cfg,
    }
}
```

**CRITICAL CONCERNS**:

#### 1.4.1 Hardcoded Config Path
❌ **Problem**: `.sharkconfig.json` path is hardcoded in `NewTaskRepository`

**Issue**: The codebase uses **auto-detection** to find project root (see CLAUDE.md):
> "Shark automatically finds the project root by walking up the directory tree"

The config path should be resolved from the project root, not hardcoded.

**Solution**:
```go
// Option A: Pass config to repository constructor (RECOMMENDED)
func NewTaskRepository(database *db.DB, cfg *config.WorkflowConfig) *TaskRepository {
    if cfg == nil {
        cfg = config.DefaultWorkflowConfig()
    }
    return &TaskRepository{
        db:       database,
        workflow: cfg,
    }
}

// Option B: Pass config path to repository
func NewTaskRepository(database *db.DB, configPath string) *TaskRepository {
    cfg, err := config.LoadWorkflowConfig(configPath)
    if err != nil {
        cfg = config.DefaultWorkflowConfig()
    }
    return &TaskRepository{
        db:       database,
        workflow: cfg,
    }
}
```

**Recommendation**: **Option A (dependency injection)**
- Keeps repository layer pure (no file system access)
- Config loaded once in CLI layer, passed to all repositories
- Easier to test (inject mock configs)
- Follows existing patterns (constructor injection)

#### 1.4.2 Repository Initialization Impact

**Current Call Sites** (need refactoring):
```bash
$ grep -r "NewTaskRepository" internal/cli/commands/
# Multiple commands create repository inline:
repo := repository.NewTaskRepository(db)
```

**Impact**: Every command file will need updating to pass config.

**Recommendation**:
1. **Create factory function** in CLI layer:
   ```go
   // internal/cli/repository_factory.go
   package cli

   var (
       globalWorkflowConfig *config.WorkflowConfig
       configLoadOnce       sync.Once
   )

   func GetWorkflowConfig() *config.WorkflowConfig {
       configLoadOnce.Do(func() {
           cfg, err := config.LoadWorkflowConfig(filepath.Join(ProjectRoot, ".sharkconfig.json"))
           if err != nil {
               cfg = config.DefaultWorkflowConfig()
           }
           globalWorkflowConfig = cfg
       })
       return globalWorkflowConfig
   }

   func NewTaskRepository(db *db.DB) *repository.TaskRepository {
       return repository.NewTaskRepository(db, GetWorkflowConfig())
   }
   ```

2. **Update all command files** to use CLI factory instead of repository constructor:
   ```go
   // Before:
   repo := repository.NewTaskRepository(db)

   // After:
   repo := cli.NewTaskRepository(db)
   ```

This minimizes changes to command files while centralizing config loading.

#### 1.4.3 Validation Logic Refactoring

**Current Implementation** (lines 470-509):
```go
func isValidTransition(from models.TaskStatus, to models.TaskStatus) bool {
    validTransitions := map[models.TaskStatus][]models.TaskStatus{ /* hardcoded */ }
    // ...
}
```

**Proposed Replacement**:
```go
// Remove isValidTransition entirely

// In UpdateStatusForced:
if !force {
    if err := r.workflow.CanTransition(string(currentTaskStatus), string(newStatus)); err != nil {
        return fmt.Errorf("invalid status transition: %w\nValid next statuses: %v\nUse --force to override",
            err, r.workflow.GetValidNextStatuses(string(currentTaskStatus)))
    }
}
```

**Enhanced Error Messages** (developer experience):
```go
// internal/config/workflow.go
func (w *WorkflowConfig) CanTransition(from, to string) error {
    validNext, ok := w.StatusFlow[from]
    if !ok {
        return fmt.Errorf("unknown status: %q", from)
    }

    for _, allowed := range validNext {
        if allowed == to {
            return nil // Valid transition
        }
    }

    // Generate helpful error message
    if len(validNext) == 0 {
        return fmt.Errorf("cannot transition from terminal status %q", from)
    }

    return fmt.Errorf("cannot transition from %q to %q (allowed: %s)",
        from, to, strings.Join(validNext, ", "))
}
```

#### 1.4.4 Special Method Refactoring

**Current Special Methods** (need updating):
- `BlockTask()` - hardcoded validation: only from `todo` or `in_progress`
- `UnblockTask()` - hardcoded validation: only from `blocked`
- `ReopenTask()` - hardcoded validation: only from `ready_for_review`

**Proposed Refactoring**:
```go
// BlockTask should use workflow config
func (r *TaskRepository) BlockTask(ctx context.Context, taskID int64, reason string, force bool) error {
    // Get current status
    task, err := r.GetByID(ctx, taskID)
    if err != nil {
        return err
    }

    // Validate transition to 'blocked'
    if !force {
        if err := r.workflow.CanTransition(string(task.Status), "blocked"); err != nil {
            return fmt.Errorf("cannot block task: %w", err)
        }
    }

    // Proceed with blocking...
}

// Similarly for UnblockTask and ReopenTask
```

**Important**: These methods currently assume specific status names (`blocked`, `ready_for_review`). With custom workflows, these may not exist!

**Recommendation**:
1. **Deprecate hardcoded convenience methods** or make them config-aware:
   ```go
   // Check if 'blocked' status exists in workflow
   func (r *TaskRepository) BlockTask(...) error {
       if _, exists := r.workflow.StatusFlow["blocked"]; !exists {
           return fmt.Errorf("'blocked' status not defined in workflow")
       }
       // ...
   }
   ```

2. **Add generic method** (as spec proposes):
   ```go
   func (r *TaskRepository) SetStatus(ctx context.Context, taskKey, newStatus string, force bool, notes *string) error {
       // Generic status transition
   }
   ```

---

### 1.5 CLI Commands ✅ **APPROVED**

**New Commands**:
- `shark workflow list` - ✅ Good
- `shark workflow validate` - ✅ Essential for debugging config
- `shark workflow graph` - ✅ Nice to have (use Mermaid)

**Updated Commands**:
- `shark task create --status=draft` - ✅ Good (validates against `_start_`)
- `shark task set-status <key> <status> [--force]` - ✅ Essential

**Existing Commands** (remain for convenience):
- `shark task start` - maps to `set-status in_progress` (or config equivalent)
- `shark task complete` - maps to `set-status ready_for_review`
- `shark task approve` - maps to `set-status completed`

**Concern**: What if custom workflow doesn't have `in_progress` or `completed`?

**Recommendation**:
1. **Map convenience commands to workflow metadata**:
   ```json
   {
     "status_metadata": {
       "in_development": {
         "semantic_role": "in_progress"  // NEW field
       },
       "done": {
         "semantic_role": "completed"
       }
     }
   }
   ```

2. **Or deprecate convenience commands** and force users to use `set-status`:
   - Pro: Simpler, no ambiguity
   - Con: Breaks existing scripts/workflows

**Recommendation**: **Keep convenience commands as aliases** that resolve to semantic roles in config.

---

## 2. Architectural Patterns & Consistency

### 2.1 Alignment with Clean Architecture ✅ **EXCELLENT**

**Layering**:
```
CLI Layer (commands/)
    ↓ calls
Repository Layer (repository/)
    ↓ calls
Database Layer (db/)
    ↓ uses
Models Layer (models/)

Config Layer (config/) ← loaded by CLI, injected into Repository
```

**Proper Separation**:
- ✅ Config layer is independent
- ✅ Repository layer doesn't know about CLI
- ✅ Database layer is isolated
- ✅ Models define data contracts

**Dependency Injection**:
- ✅ Repository receives config via constructor
- ✅ No global state (except cached config in CLI layer)
- ✅ Testable (can inject mock configs)

**Assessment**: Design aligns perfectly with existing architecture.

---

### 2.2 Consistency with Existing Patterns ✅ **GOOD**

**Configuration Patterns**:
- ✅ Uses existing `.sharkconfig.json`
- ✅ Follows existing config struct patterns (see `internal/config/config.go`)
- ✅ Uses pointer fields for optional config (`*bool`, `*string`)

**Repository Patterns**:
- ✅ Constructor injection (`NewTaskRepository(db *DB)`)
- ✅ Context-aware methods (`UpdateStatus(ctx context.Context, ...)`)
- ✅ Error wrapping (`fmt.Errorf("context: %w", err)`)
- ✅ Transaction management (BEGIN, ROLLBACK, COMMIT)

**CLI Patterns**:
- ✅ Cobra command structure
- ✅ Global flags (`--json`, `--force`, `--verbose`)
- ✅ Output formatting (JSON vs. table)

**Assessment**: Design is consistent with existing codebase conventions.

---

## 3. Migration Strategy Analysis

### 3.1 Migration Options Comparison

| Aspect | Option A: Auto | Option B: Explicit Command | Option C: Coexist |
|--------|----------------|----------------------------|-------------------|
| **User Experience** | Transparent | Explicit confirmation | Confusing (2 systems) |
| **Risk** | Medium (auto-mutation) | Low (user controls) | High (data inconsistency) |
| **Rollback** | Difficult | Easy (don't run) | N/A |
| **Testing** | Complex | Simple | Very complex |
| **Documentation** | Minimal | Clear migration guide | Extensive |

**Spec Recommendation**: Option B (explicit migration command)

**Architect Assessment**: ✅ **AGREE - Option B is best**

**Rationale**:
1. **Explicit is better than implicit** (Python Zen applies to CLIs)
2. **User control**: Shows what will change before changing it
3. **Safer**: No surprises on first run
4. **Auditable**: Migration can be logged and verified
5. **Testable**: Can test migration logic in isolation

### 3.2 Migration Implementation Details

**Proposed Command**:
```bash
$ shark migrate workflow [--dry-run] [--json]
```

**Implementation**:
```go
// internal/cli/commands/migrate.go
func RunWorkflowMigration(ctx context.Context, dryRun bool) error {
    db := getDB()
    repo := NewTaskRepository(db)
    cfg := GetWorkflowConfig()

    // Define legacy-to-new mapping
    mapping := map[string]string{
        "todo":             "ready_for_development",  // or keep as "todo" if exists in new config
        "in_progress":      "in_development",
        "ready_for_review": "ready_for_review",       // keep same if exists
        "completed":        "completed",
        "blocked":          "blocked",
    }

    // Find tasks with legacy statuses
    tasks, err := repo.GetTasksWithLegacyStatuses(ctx)
    if err != nil {
        return err
    }

    if len(tasks) == 0 {
        fmt.Println("No tasks need migration")
        return nil
    }

    // Show migration plan
    fmt.Printf("Found %d tasks with legacy statuses:\n", len(tasks))
    statusCounts := make(map[string]int)
    for _, task := range tasks {
        statusCounts[string(task.Status)]++
    }
    for old, count := range statusCounts {
        new := mapping[old]
        fmt.Printf("  - %d tasks: %s → %s\n", count, old, new)
    }

    if dryRun {
        fmt.Println("\nDry run mode - no changes made")
        return nil
    }

    // Confirm with user
    fmt.Print("\nProceed with migration? [y/N]: ")
    var response string
    fmt.Scanln(&response)
    if strings.ToLower(response) != "y" {
        fmt.Println("Migration cancelled")
        return nil
    }

    // Perform migration in transaction
    tx, err := db.BeginTxContext(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    for _, task := range tasks {
        newStatus := mapping[string(task.Status)]
        _, err := tx.ExecContext(ctx,
            "UPDATE tasks SET status = ? WHERE id = ?",
            newStatus, task.ID)
        if err != nil {
            return fmt.Errorf("failed to migrate task %s: %w", task.Key, err)
        }
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("migration transaction failed: %w", err)
    }

    fmt.Printf("✓ Successfully migrated %d tasks\n", len(tasks))
    return nil
}
```

**Additional Safety**:
1. **Backup database before migration**:
   ```bash
   cp shark-tasks.db shark-tasks.db.backup-$(date +%Y%m%d)
   ```

2. **Validate new statuses exist in config** before migration:
   ```go
   for old, new := range mapping {
       if _, exists := cfg.StatusFlow[new]; !exists {
           return fmt.Errorf("target status %q not defined in workflow config", new)
       }
   }
   ```

3. **Create task_history entries** for migration:
   ```sql
   INSERT INTO task_history (task_id, old_status, new_status, changed_by, notes)
   VALUES (?, ?, ?, 'system', 'Migrated from legacy status workflow')
   ```

---

## 4. Performance & Scalability

### 4.1 Configuration Loading Performance ✅ **GOOD**

**Analysis**:
- Config loaded **once per CLI invocation** (cached in `cli` package)
- JSON parsing is fast (microseconds for ~50-line config)
- In-memory lookups for status validation (O(1) map access)

**Benchmark Estimate**:
```
LoadWorkflowConfig: ~100μs (initial load)
CanTransition: ~500ns (map lookup)
GetValidNextStatuses: ~1μs (array copy)
```

**Impact**: Negligible (<0.1ms per command)

**Recommendation**: ✅ No performance concerns

### 4.2 Database Performance ✅ **NO IMPACT**

**Analysis**:
- No schema changes = no migration overhead
- Status field remains TEXT (no type change)
- Validation moved from hardcoded map to config map (same complexity: O(1))
- No additional database queries

**Existing Indexes**:
```sql
CREATE INDEX idx_tasks_status ON tasks(status);
```

This index remains valid and efficient for arbitrary status values.

**Recommendation**: ✅ No performance impact

### 4.3 Scalability ✅ **EXCELLENT**

**Workflow Complexity Scaling**:
- 14 statuses × average 3 transitions = 42 edges in workflow graph
- Map lookups remain O(1) regardless of workflow size
- Config validation is O(V + E) where V = statuses, E = transitions
  - For 14 statuses: ~50 operations (negligible)

**Concurrent Access**:
- Config is read-only after load (no locking needed)
- Repository instances can share config safely
- No contention or race conditions

**Recommendation**: ✅ Scales well to complex workflows (100+ statuses if needed)

---

## 5. Testing Strategy Completeness

### 5.1 Proposed Testing Strategy ✅ **COMPREHENSIVE**

**Spec Proposes**:
- Unit tests (workflow validation, edge cases)
- Integration tests (full workflow, backward transitions, force flag)
- Manual testing (multi-agent simulation, config errors)

**Assessment**: Well-structured, covers all layers

### 5.2 Critical Test Cases (Must Have)

#### Unit Tests (internal/config/workflow_test.go)
```go
func TestWorkflowValidation(t *testing.T) {
    tests := []struct {
        name      string
        config    *WorkflowConfig
        expectErr bool
    }{
        {
            name: "valid simple workflow",
            config: &WorkflowConfig{
                StatusFlow: StatusFlow{
                    "_start_": []string{"todo"},
                    "_complete_": []string{"done"},
                    "todo": []string{"done"},
                    "done": []string{},
                },
            },
            expectErr: false,
        },
        {
            name: "missing _start_ key",
            config: &WorkflowConfig{
                StatusFlow: StatusFlow{
                    "_complete_": []string{"done"},
                    "todo": []string{"done"},
                },
            },
            expectErr: true,
        },
        {
            name: "transition to undefined status",
            config: &WorkflowConfig{
                StatusFlow: StatusFlow{
                    "_start_": []string{"todo"},
                    "_complete_": []string{"done"},
                    "todo": []string{"in_progress"},  // undefined
                },
            },
            expectErr: true,
        },
        {
            name: "circular workflow",
            config: &WorkflowConfig{
                StatusFlow: StatusFlow{
                    "_start_": []string{"a"},
                    "_complete_": []string{"c"},
                    "a": []string{"b"},
                    "b": []string{"a"},  // circular
                    "c": []string{},
                },
            },
            expectErr: false,  // Circular is OK (not terminal)
        },
    }
    // ...
}

func TestCanTransition(t *testing.T) {
    cfg := DefaultWorkflowConfig()

    tests := []struct {
        from      string
        to        string
        expectErr bool
    }{
        {"todo", "in_progress", false},
        {"todo", "completed", true},           // invalid
        {"completed", "todo", true},           // terminal → anything
        {"in_progress", "blocked", false},
        {"unknown_status", "todo", true},      // undefined status
    }
    // ...
}
```

#### Repository Tests (internal/repository/task_workflow_test.go)
```go
func TestTaskRepository_StatusTransitionWithWorkflow(t *testing.T) {
    ctx := context.Background()
    database := test.GetTestDB()

    // Create custom workflow
    customWorkflow := &config.WorkflowConfig{
        StatusFlow: config.StatusFlow{
            "_start_": []string{"draft"},
            "_complete_": []string{"done"},
            "draft": []string{"in_progress"},
            "in_progress": []string{"done"},
            "done": []string{},
        },
    }

    db := repository.NewDB(database)
    repo := repository.NewTaskRepository(db, customWorkflow)

    // Clean up test data
    database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'TEST-WF-001'")

    epicID, featureID := test.SeedTestData()

    task := &models.Task{
        FeatureID: featureID,
        Key:       "TEST-WF-001",
        Title:     "Test Workflow",
        Status:    "draft",  // Custom status
        Priority:  5,
    }

    err := repo.Create(ctx, task)
    assert.NoError(t, err)

    // Test valid transition
    err = repo.UpdateStatus(ctx, task.ID, "in_progress", nil, nil)
    assert.NoError(t, err)

    // Test invalid transition (force=false)
    err = repo.UpdateStatus(ctx, task.ID, "draft", nil, nil)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid status transition")

    // Test invalid transition with force
    err = repo.UpdateStatusForced(ctx, task.ID, "draft", nil, nil, true)
    assert.NoError(t, err)  // Should succeed with force

    // Cleanup
    database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
}
```

#### Integration Tests (cmd/test-workflow/main.go)
```go
// Full end-to-end workflow test
func TestFullWorkflowIntegration(t *testing.T) {
    // 1. Create custom workflow config
    // 2. Initialize database
    // 3. Create task with start status
    // 4. Transition through all statuses
    // 5. Verify task_history records
    // 6. Test force flag
    // 7. Test invalid transitions
}
```

#### CLI Tests (internal/cli/commands/workflow_test.go)
```go
func TestWorkflowListCommand(t *testing.T) {
    // Test workflow list output (JSON and table)
}

func TestWorkflowValidateCommand(t *testing.T) {
    // Test validation with valid and invalid configs
}

func TestTaskSetStatusCommand(t *testing.T) {
    // Test set-status command with validation
}
```

### 5.3 Testing Gaps to Address

**Missing from Spec**:
1. **Config reload testing**: What happens if config changes mid-process?
2. **Migration rollback testing**: Can we undo a migration?
3. **Concurrent status updates**: Race conditions with workflow validation?
4. **Backward compatibility**: Can old CLI work with new database?

**Recommendations**:
1. Add **config versioning tests** (reject old/new incompatible configs)
2. Add **concurrency tests** (multiple agents updating status simultaneously)
3. Add **migration rollback** (restore from backup script)

---

## 6. Technical Risks & Dependencies

### 6.1 High-Risk Areas 🔴

#### Risk 1: Breaking Existing Workflows
**Severity**: HIGH
**Probability**: MEDIUM

**Scenario**: Users with existing tasks in `todo`, `in_progress`, etc. upgrade and their tasks become "invalid" in new workflow.

**Mitigation**:
1. ✅ **Spec addresses this** with migration strategy (Option B)
2. ✅ **Default workflow preserves current statuses** (no breaking change if no custom config)
3. ⚠️ **Add explicit version check**: Refuse to run if config version > CLI version

**Additional Safeguard**:
```go
// Detect legacy statuses and warn user
func (r *TaskRepository) DetectLegacyStatuses(ctx context.Context) ([]string, error) {
    query := `SELECT DISTINCT status FROM tasks WHERE status NOT IN (?)`
    // Query all statuses not in current workflow

    if len(legacyStatuses) > 0 {
        fmt.Fprintf(os.Stderr, "WARNING: %d tasks use statuses not in workflow config\n", len(legacyStatuses))
        fmt.Fprintf(os.Stderr, "Run 'shark migrate workflow' to update\n")
    }
}
```

#### Risk 2: Configuration Complexity
**Severity**: MEDIUM
**Probability**: HIGH

**Scenario**: Users misconfigure workflow (unreachable statuses, no terminal state, etc.) and commands fail silently or confusingly.

**Mitigation**:
1. ✅ **Spec includes `shark workflow validate`** command
2. ✅ **Config validation in load path** (proposed above)
3. ⚠️ **Add validation to `shark init`**: Validate config on project initialization

**Additional Safeguard**:
```bash
# Run validation on every command (dev mode)
if [ "$SHARK_DEBUG" = "1" ]; then
    shark workflow validate || echo "WARNING: Invalid workflow config"
fi
```

#### Risk 3: Repository Initialization Refactoring
**Severity**: MEDIUM
**Probability**: HIGH

**Scenario**: Updating 20+ command files to pass config to repository breaks existing commands or introduces bugs.

**Mitigation**:
1. ✅ **Use CLI factory pattern** (proposed in 1.4.2)
2. ✅ **Compile-time checking**: Go compiler will catch signature mismatches
3. ⚠️ **Add integration tests** for all commands after refactoring

**Testing Strategy**:
```bash
# Before refactoring
make test > test-output-before.txt

# After refactoring
make test > test-output-after.txt

# Compare
diff test-output-before.txt test-output-after.txt
# Should show no new failures
```

### 6.2 Medium-Risk Areas 🟡

#### Risk 4: Status Name Collisions
**Severity**: LOW
**Probability**: MEDIUM

**Scenario**: User creates custom workflow with status name that conflicts with database reserved keywords or Go constants.

**Mitigation**:
1. ✅ **Status is TEXT field** (no enum constraint)
2. ⚠️ **Add validation**: Reject statuses with special characters, SQL keywords

**Validation**:
```go
func ValidateStatusName(status string) error {
    // Alphanumeric + underscore only
    if !regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(status) {
        return fmt.Errorf("invalid status name: %q (must be lowercase alphanumeric)", status)
    }

    // Reject SQL keywords
    sqlKeywords := []string{"select", "insert", "update", "delete", "drop", "table"}
    for _, keyword := range sqlKeywords {
        if status == keyword {
            return fmt.Errorf("invalid status name: %q is a reserved keyword", status)
        }
    }

    return nil
}
```

#### Risk 5: Task History Integrity
**Severity**: LOW
**Probability**: LOW

**Scenario**: Status transitions with `--force` bypass validation and create invalid history entries.

**Mitigation**:
1. ✅ **Existing trigger** already records all status changes (see db.go:~300)
2. ✅ **Force flag is logged** (spec line 544: `fmt.Printf("WARNING: Forced status update...")`)
3. ⚠️ **Add `forced` column** to task_history:
   ```sql
   ALTER TABLE task_history ADD COLUMN forced BOOLEAN DEFAULT FALSE;
   ```

### 6.3 Low-Risk Areas 🟢

#### Risk 6: Performance Degradation
**Severity**: LOW
**Probability**: LOW

**Analysis**: Config lookups are O(1), no database changes, same transaction patterns.

**Mitigation**: ✅ None needed (negligible risk)

---

## 7. Dependencies & Integration Points

### 7.1 Internal Dependencies

**Direct Dependencies**:
1. `internal/config/` package (new)
   - ✅ No circular dependencies
   - ✅ Clean interface

2. `internal/repository/task_repository.go` (modification)
   - ✅ Requires config injection
   - ⚠️ Affects 20+ CLI commands (see 1.4.2)

3. `internal/cli/commands/` (multiple files)
   - ✅ Factory pattern minimizes changes
   - ⚠️ Testing burden (must re-test all commands)

4. `internal/models/task.go` (minimal changes)
   - ⚠️ **Remove hardcoded TaskStatus constants?**
   - **Recommendation**: Keep constants for backward compatibility, but don't enforce them

### 7.2 External Dependencies

**Config File** (`.sharkconfig.json`):
- ✅ Already exists
- ✅ Already has version control
- ⚠️ **Add JSON schema** for IDE validation

**Database** (`shark-tasks.db`):
- ✅ No schema changes required
- ✅ Existing indexes work
- ✅ Backward compatible

**CLI Clients**:
- ⚠️ **Scripts using hardcoded statuses** may break
- **Mitigation**: Document migration in CHANGELOG, provide migration guide

---

## 8. Recommendations Summary

### 8.1 Architecture Improvements

| # | Recommendation | Priority | Effort |
|---|----------------|----------|--------|
| 1 | Add config version field (`status_flow_version`) | HIGH | 1 hour |
| 2 | Implement config validation (unreachable statuses, undefined references) | HIGH | 4 hours |
| 3 | Use CLI factory pattern for repository creation | HIGH | 8 hours |
| 4 | Ship preset workflows (simple, kanban, full) | MEDIUM | 2 hours |
| 5 | Add semantic role mapping for convenience commands | MEDIUM | 4 hours |
| 6 | Reject validation trigger in database | HIGH | 0 hours (don't implement) |
| 7 | Add `forced` column to task_history | LOW | 2 hours |
| 8 | Add JSON schema for `.sharkconfig.json` | MEDIUM | 3 hours |

### 8.2 Migration Approach

**Selected Strategy**: ✅ **Option B - Explicit Migration Command**

**Implementation Steps**:
1. Implement `shark migrate workflow` command
2. Add `--dry-run` flag to preview changes
3. Create database backup before migration
4. Validate target statuses exist in new workflow
5. Record migration in task_history
6. Add rollback instructions to docs

**Timeline**: 2-3 days

### 8.3 Design Patterns

**Recommended Patterns**:
1. **Dependency Injection**: Pass `WorkflowConfig` to repositories (not file paths)
2. **Factory Pattern**: CLI layer creates repositories with injected config
3. **Singleton Pattern**: Cache workflow config per CLI process (not global)
4. **Strategy Pattern**: Workflow config defines transition strategy (not hardcoded)

**Anti-Patterns to Avoid**:
1. ❌ **Database triggers for validation** (use application layer)
2. ❌ **Global mutable state** (config should be immutable after load)
3. ❌ **Hardcoded config paths in repositories** (inject from CLI)
4. ❌ **Silent auto-migration** (require explicit user action)

### 8.4 Missing Technical Considerations

**Not Addressed in Spec**:
1. **Config versioning and evolution** - How to upgrade workflows over time?
2. **Workflow analytics** - Track time in each status, bottleneck detection
3. **Multi-workflow support** - Different workflows for bug vs. feature vs. spike
4. **Status aliases** - Short names (e.g., `wip` → `in_progress`)
5. **Transition hooks** - Run scripts on status change (future enhancement)
6. **Audit compliance** - Immutable audit trail of forced transitions

**Recommendations**:
1. ✅ **Add to "Future Enhancements" section** in spec
2. ⚠️ **Design config schema to support** these (forward compatibility)
3. ⚠️ **Document extension points** for future work

---

## 9. Final Assessment

### 9.1 Technical Soundness ✅ **APPROVED**

The design is architecturally sound, well-aligned with existing patterns, and solves the stated problem effectively.

**Strengths**:
- ✅ Minimal invasive changes (no database schema changes)
- ✅ Backward compatible via migration
- ✅ Proper separation of concerns
- ✅ Comprehensive testing strategy
- ✅ Force flag escape hatch for operational flexibility

**Weaknesses** (addressable):
- ⚠️ Config loading strategy needs refinement (caching, validation)
- ⚠️ Repository initialization requires careful refactoring
- ⚠️ Convenience commands need semantic mapping

### 9.2 Implementation Readiness 🟡 **READY with REFINEMENTS**

**Blocking Issues** (must address before implementation):
1. **Config validation** - Add comprehensive validation (unreachable statuses, circular refs)
2. **Repository factory pattern** - Define CLI-layer factory to minimize command changes
3. **Migration command** - Implement Option B (explicit migration)

**Non-Blocking Improvements** (can address in iterations):
1. Config versioning
2. Preset workflows
3. Workflow analytics
4. JSON schema

### 9.3 Risk Level 🟢 **LOW-MEDIUM**

**Overall Risk**: **MEDIUM** (manageable with proper testing)

**High-Risk Areas**:
- Repository initialization refactoring (affects many files)
- Legacy status migration (user data at risk)

**Mitigation**:
- ✅ Comprehensive test coverage (unit, integration, manual)
- ✅ Explicit migration with dry-run and confirmation
- ✅ Database backups before migration
- ✅ Rollback documentation

### 9.4 Recommendation: PROCEED ✅

**Verdict**: **APPROVED for implementation** with the following conditions:

1. ✅ **Implement all HIGH priority recommendations** (items 1-3, 6 in section 8.1)
2. ✅ **Use Option B for migration** (explicit command with confirmation)
3. ✅ **Add comprehensive testing** (coverage target: >90% for workflow package)
4. ✅ **Document migration guide** (with rollback instructions)
5. ✅ **Phased rollout**: Beta test with 2-3 workflows before full release

---

## 10. Next Steps

### 10.1 Pre-Implementation Tasks
- [ ] **Update spec** with architecture recommendations from this review
- [ ] **Create ADR** (Architecture Decision Record) for workflow system
- [ ] **Define config JSON schema** for validation
- [ ] **Create preset workflows** (simple, kanban, full)
- [ ] **Write migration guide** (user-facing documentation)

### 10.2 Implementation Phases

**Phase 1: Core Infrastructure** (3-4 days)
- [ ] Implement `internal/config/workflow.go`
- [ ] Add config validation logic
- [ ] Create default workflow config
- [ ] Add unit tests (>90% coverage)

**Phase 2: Repository Integration** (2-3 days)
- [ ] Add CLI factory pattern (`cli.NewTaskRepository()`)
- [ ] Refactor `TaskRepository` to accept config
- [ ] Update `isValidTransition()` to use config
- [ ] Refactor special methods (`BlockTask`, `UnblockTask`, `ReopenTask`)
- [ ] Add repository tests with custom workflows

**Phase 3: CLI Commands** (2-3 days)
- [ ] Implement `shark workflow list`
- [ ] Implement `shark workflow validate`
- [ ] Implement `shark task set-status`
- [ ] Update existing commands (`start`, `complete`, `approve`)
- [ ] Add `--force` flag to all status commands

**Phase 4: Migration** (2 days)
- [ ] Implement `shark migrate workflow`
- [ ] Add dry-run mode
- [ ] Add backup creation
- [ ] Add task_history logging
- [ ] Test on sample database

**Phase 5: Testing & Documentation** (2-3 days)
- [ ] Integration tests (full workflow scenarios)
- [ ] Manual testing with multi-agent simulation
- [ ] Write user guide (migration instructions)
- [ ] Update CLAUDE.md with workflow architecture
- [ ] Create release notes

**Total Estimated Effort**: 11-15 days

### 10.3 Success Criteria
- [ ] All existing tests pass
- [ ] New tests achieve >90% coverage
- [ ] Migration tested on 3+ real projects
- [ ] Documentation complete and reviewed
- [ ] No breaking changes to existing workflows (without migration)

---

## Appendix A: Refactoring Checklist

### Files to Create
- [ ] `internal/config/workflow.go` (new)
- [ ] `internal/config/workflow_test.go` (new)
- [ ] `internal/cli/repository_factory.go` (new)
- [ ] `internal/cli/commands/workflow.go` (new)
- [ ] `internal/cli/commands/migrate.go` (new)
- [ ] `docs/workflows/simple.json` (new)
- [ ] `docs/workflows/kanban.json` (new)
- [ ] `docs/workflows/full.json` (new)
- [ ] `docs/WORKFLOW_MIGRATION_GUIDE.md` (new)

### Files to Modify
- [ ] `.sharkconfig.json` (add `status_flow` section)
- [ ] `internal/config/config.go` (add workflow fields)
- [ ] `internal/repository/task_repository.go` (inject config, replace hardcoded logic)
- [ ] `internal/repository/task_repository_test.go` (add workflow tests)
- [ ] `internal/cli/commands/task.go` (use factory, add set-status)
- [ ] `internal/cli/commands/helpers.go` (add workflow helpers)
- [ ] All command files in `internal/cli/commands/` (use factory: ~20 files)
- [ ] `CLAUDE.md` (document workflow architecture)
- [ ] `README.md` (mention configurable workflows)

### Files to Remove (Optional)
- [ ] `internal/models/task.go` (deprecate hardcoded TaskStatus constants) - **Recommendation**: Keep for backward compat

---

## Appendix B: Open Questions for Product Team

1. **Default Workflow**: Ship with simple (3 statuses) or full (14 statuses)?
   - **Architect Recommendation**: Simple by default, full as opt-in

2. **Convenience Commands**: Keep `start`, `complete`, `approve` or deprecate?
   - **Architect Recommendation**: Keep as aliases with semantic role mapping

3. **Config Location**: `.sharkconfig.json` or separate `workflow.json`?
   - **Architect Recommendation**: Keep in `.sharkconfig.json` (single source)

4. **Validation Strictness**: Error or warning for invalid statuses in database?
   - **Architect Recommendation**: Warning + suggest migration (don't break existing)

5. **Transition Hooks**: Future support for running scripts on status change?
   - **Architect Recommendation**: Yes, but phase 2 (not MVP)

---

## Appendix C: References

**Specification**: `/home/jwwelbor/projects/shark-task-manager/docs/specs/configurable-status-workflow.md`

**Existing Codebase Files**:
- `/home/jwwelbor/projects/shark-task-manager/internal/repository/task_repository.go` (lines 452-746)
- `/home/jwwelbor/projects/shark-task-manager/internal/config/config.go`
- `/home/jwwelbor/projects/shark-task-manager/internal/models/task.go`
- `/home/jwwelbor/projects/shark-task-manager/internal/models/validation.go`
- `/home/jwwelbor/projects/shark-task-manager/internal/db/db.go`
- `/home/jwwelbor/projects/shark-task-manager/CLAUDE.md`

**Related Documents**:
- Clean Architecture principles: https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html
- State Machine Patterns: https://refactoring.guru/design-patterns/state
- Go Dependency Injection: https://github.com/google/wire

---

**Reviewed By**: Architect Agent
**Date**: 2025-12-29
**Status**: ✅ **APPROVED with recommendations**
