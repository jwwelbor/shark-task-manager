# Business Logic

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 4 — Behavior Analysis

## Business Domains

### 1. Project Hierarchy Management

**Domain**: Organizing work into a hierarchical structure of Epics, Features, and Tasks.

**Key Business Rules**:
- Epics contain Features; Features contain Tasks
- Each entity has a unique key (auto-generated, case-insensitive)
- Keys support dual format: numeric (`E07-F01-001`) and slugged (`E07-F01-001-task-name`)
- Cascading delete: deleting an Epic removes all its Features and Tasks
- Progress rolls up: Feature progress = completed tasks / total tasks; Epic progress = aggregate feature progress

**Core Operations**:
- CRUD for Epics, Features, Tasks, Bugs, Change Cards, Ideas
- Entity file creation in `docs/plan/` filesystem hierarchy
- Entity search across all types
- Related document management

**Source**: `internal/services/task_service.go`, `feature_service.go`, `epic_service.go`

### 2. Workflow & Status Management

**Domain**: Managing entity lifecycle through configurable status flows.

**Key Business Rules**:
- Status transitions are validated against `.sharkconfig.json` workflow definitions
- Each entity level (epic, feature, task, bug, change) has its own status flow
- Two workflow profiles: Basic (5 statuses) and Advanced (19 statuses)
- Backward transitions can require rejection reasons
- Force mode bypasses transition validation
- Terminal statuses: `completed`, `cancelled`
- Agent routing: specific statuses are assigned to specific agent types

**Core Operations**:
- Advance status to next valid state
- Set status directly (with optional force)
- Get valid transition options
- View status history
- Orchestrator action resolution (determine what agent should handle current status)

**Transition Feature Strategies** (polymorphic via EntityService):
- **DefaultTransitionFeatures** (Epic/Feature/Task): Backward detection enabled, rejection notes created, orchestrator actions resolved
- **SimpleTransitionFeatures** (Bug/ChangeCard): No backward detection, no rejection notes, orchestrator actions still resolved

**Source**: `internal/services/entity_service.go`, `internal/workflow/service.go`

### 3. Dependency Management

**Domain**: Tracking relationships between tasks.

**Key Business Rules**:
- 7 relationship types: `depends_on`, `blocks`, `related_to`, `follows`, `spawned_from`, `duplicates`, `references`
- Circular dependency detection
- Dependency validation before status transitions
- Automatic unblocking: when a blocking task completes, blocked tasks are notified

**Core Operations**:
- Link/unlink tasks with relationship types
- View dependency tree (with configurable depth)
- Check blocked-by / blocks relationships
- Validate dependencies before advancing status

**Source**: `internal/services/task_dependency_service.go`, `internal/repository/task_relationship_repository.go`

### 4. Context & Resume

**Domain**: Providing full context for resuming work on an entity.

**Key Business Rules**:
- Context aggregates entity details, parent context, notes, and history
- Resume assembles a complete picture for an AI agent or developer to continue work
- Context fields are key-value pairs stored per entity
- Notes support 10 types: comment, decision, blocker, solution, reference, implementation, testing, future, question, rejection

**Core Operations**:
- Set/get/clear context fields
- Add notes with type classification
- Resume with full context assembly
- Entity note CRUD

**Source**: `internal/services/context_service.go`, `resume_service.go`, `note_service.go`

### 5. Progress & Analytics

**Domain**: Calculating and displaying project health metrics.

**Key Business Rules**:
- Weighted progress uses `progress_weight` from status metadata (e.g., `in_progress` = 50%, `completed` = 100%)
- Completion progress = completed tasks / total tasks
- Health indicators: healthy (no blockers), warning (aging approvals), critical (multiple blockers)
- Work breakdown categorizes remaining work by responsibility (agent, human, QA)
- Action items identify tasks in statuses that `blocks_feature: true`

**Core Operations**:
- Feature/epic progress calculation (weighted and completion)
- Work remaining by responsibility
- Feature health assessment
- Action item identification
- Impediment tracking (blocked tasks with age)

**Source**: `internal/status/`, `internal/services/epic_analytics_service.go`

### 6. Idea Management

**Domain**: Capturing and promoting ideas into the project hierarchy.

**Key Business Rules**:
- Ideas have a separate lifecycle: new → on_hold / converted / archived
- Ideas can be promoted to epics, features, or tasks
- Promotion records conversion metadata (type, key, timestamp)
- Ideas have priority, ordering, and dependency tracking

**Source**: `internal/cli/commands/idea.go`, `internal/repository/idea_repository.go`

## Cross-Domain Interactions

| From | To | Interaction |
|------|----|-------------|
| Workflow | Progress | Status changes trigger progress recalculation |
| Dependency | Workflow | Dependencies validated before status advance |
| Dependency | Workflow | Completing a task may unblock dependents |
| Hierarchy | Progress | Feature progress depends on child task statuses |
| Context | Resume | Resume aggregates context from entity and ancestors |
| Idea | Hierarchy | Idea promotion creates entities in the hierarchy |

---

See also: [Workflows](workflows.md) | [Decision Logic](decision-logic.md) | [Error Handling](error-handling.md)
