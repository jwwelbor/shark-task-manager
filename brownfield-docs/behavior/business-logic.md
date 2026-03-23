# Business Logic

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22
> Phase: 4 — Behavior Analysis

## Business Domains

### 1. Entity Lifecycle Management

**Purpose**: Manage hierarchical work items (Epic → Feature → Task) through configurable workflow states.

**Key Business Rules**:
- Epics contain features; features contain tasks — strict hierarchy enforced by foreign keys
- Bugs and ChangeCards are standalone entities optionally linked to any parent
- Status transitions are config-driven, not hardcoded
- Backward transitions require documented reasons
- Forced transitions require both `--force` and `--reason` flags
- All status changes create audit trail entries

**Core Operations**:
- Create/update/delete entities at any level
- Advance entities through workflow states
- Cascade status changes from parent to children
- Block/unblock entities with reason tracking

### 2. Progress & Health Tracking

**Purpose**: Calculate and display progress metrics across the entity hierarchy.

**Key Business Rules**:
- **Weighted progress**: `(Σ(status_weight × task_count) / total_tasks) × 100%`
  - Weights defined in `.sharkconfig.json` per status (e.g., todo=0%, in_progress=50%, completed=100%)
- **Completion progress**: Simple `completed_tasks / total_tasks × 100%`
- Feature progress rolls up from task breakdown
- Epic progress aggregates from feature rollups
- Health status derived from blockers and approval age:
  - `healthy`: No blockers, approvals < 3 days
  - `warning`: Approvals > 3 days or minor blockers
  - `critical`: Multiple blockers or high-priority blocked tasks

**Cross-Domain**: Progress drives feature status display, epic dashboards, and analytics

### 3. Dependency Management

**Purpose**: Track and enforce dependencies between entities.

**Key Business Rules**:
- Tasks can depend on other tasks (stored as JSON array of keys in `depends_on` field)
- Starting a task validates all dependencies are completed
- Circular dependencies detected and rejected
- Supported relationship types: `depends_on`, `blocks`, `related_to`, `follows`, `spawned_from`, `duplicates`, `references`
- Cross-entity relationships supported via `entity_relationships` table

### 4. Workflow Configuration

**Purpose**: Define how entities flow through statuses, who owns each status, and what happens on transitions.

**Key Business Rules**:
- Two profiles: Basic (5 statuses, solo dev) and Advanced (19 statuses, team TDD)
- Each status has: color, phase, progress weight, responsibility, blocks_feature flag
- Agent routing: BA → Tech Lead → Developer → QA → Product Owner (advanced only)
- Orchestrator actions triggered on specific status transitions
- Profile switching preserves database config and viewer settings

### 5. File System Integration

**Purpose**: Maintain markdown specification files alongside database state.

**Key Business Rules**:
- Database is source of truth for status (never synced from files)
- Entity files live at `docs/plan/{epic}/{feature}/tasks/{task-key}.md`
- File discovery scans filesystem to find entity hierarchy
- File sync is one-directional: filesystem → database for content
- Atomic file writes prevent race conditions (O_EXCL flag)
- Template system generates status-specific content (80+ templates)

### 6. Idea Management

**Purpose**: Capture ideas before they're ready for formal tracking.

**Key Business Rules**:
- Ideas have lifecycle: `new → on_hold → converted → archived`
- Ideas can be promoted to Epic, Feature, or Task
- Promotion records the source idea and target entity key

## Business Constraints & Invariants

| Constraint | Enforcement | Location |
|-----------|-------------|----------|
| Epic key format `E##` | Model validation + DB CHECK | `models/validation.go`, schema |
| Feature belongs to epic | Foreign key CASCADE | Schema |
| Task belongs to feature | Foreign key CASCADE | Schema |
| Priority range 1-10 | Model validation + DB CHECK | `models/validation.go`, schema |
| Status must be valid | Workflow service validation | `workflow/service.go` |
| Transitions must be allowed | Workflow service validation | `workflow/service.go` |
| Dependencies must be met | Service validation | `services/task_service.go` |
| No circular dependencies | Cycle detection | `services/entity_relationship_service.go` |
| Slug uniqueness per entity | Numeric key + slug compound lookup | `repository/*.go` |

See also: [Workflows](workflows.md) | [Decision Logic](decision-logic.md) | [Error Handling](error-handling.md)
