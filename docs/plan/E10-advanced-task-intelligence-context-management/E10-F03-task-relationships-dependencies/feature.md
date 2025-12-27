# Feature PRD: E10-F03 Task Relationships & Dependencies

**Feature Key**: E10-F03
**Epic**: [E10: Advanced Task Intelligence & Context Management](../epic.md)
**Status**: Draft
**Priority**: Should Have (Phase 2)
**Execution Order**: 3

---

## Goal

### Problem

Development workflows involve complex dependencies between tasks, but the current Shark system only supports basic textual `depends_on` field (comma-separated task keys) with no bidirectional visibility, relationship typing, or dependency graph analysis. When an AI agent blocks on a dependency, the system cannot answer:
- "What tasks are blocked waiting for this task to complete?"
- "What is the full dependency chain for this feature?"
- "Which tasks are related but not blocking?"
- "What tasks were spawned from UAT findings or bugs in this task?"

**Real Example from E13**: T-E13-F05-004 (ThemeToggle Component) depends on both T-E13-F05-003 (useTheme Composable) and T-E13-F05-001 (Dark Mode CSS Variables). T-E13-F05-007 (Integration Testing) blocks deployment until complete. T-E13-F05-002 (Flash Prevention) is related to T-E13-F05-003 (both use same localStorage key) but not a hard dependency. Without typed relationships, product managers cannot see the dependency graph, tech leads cannot identify what's blocked, and AI agents cannot understand which tasks should be sequenced vs parallelized.

### Solution

Replace the simple `depends_on` text field with a rich relationship system using a `task_relationships` table that models seven distinct relationship types:
- **depends_on**: Task Y cannot start until Task X completes (hard dependency)
- **blocks**: Task X blocks Task Y from proceeding (inverse of depends_on, explicitly tracked)
- **related_to**: Tasks share common code/concerns but no blocking relationship
- **follows**: Task Y naturally comes after Task X in sequence (soft ordering, not blocking)
- **spawned_from**: Task Y was created from UAT findings or bugs discovered in Task X
- **duplicates**: Tasks represent duplicate work that should be merged
- **references**: Task Y consults or uses output/artifacts from Task X

The system provides CLI commands for:
- Creating typed relationships: `shark task link <task> --depends-on <other>`, `--blocks <other>`, etc.
- Querying relationships: `shark task deps <task>` (all relationships), `shark task blocked-by <task>` (what blocks this), `shark task blocks <task>` (what this blocks)
- Visualizing dependencies: `shark task graph <task>` (ASCII dependency graph)

### Impact

**For Product Managers**:
- **Complete visibility**: See full dependency graphs to understand feature complexity and critical path
- **Blocker identification**: Query "what's blocked by this incomplete task" to prioritize work
- **Risk assessment**: Identify high-fanout tasks (blocking many downstream tasks) for focused attention
- **5-10 uses per week** to understand feature dependencies and report on delivery risks

**For AI Development Agents**:
- **Intelligent task selection**: Avoid starting tasks with incomplete dependencies
- **Context awareness**: See related tasks to reuse patterns and maintain consistency
- **Efficient unblocking**: Identify what will unblock when current task completes
- **Spawned task tracking**: Link bug fixes and improvements back to source tasks

**For Tech Leads**:
- **Code review prioritization**: Review high-impact tasks blocking others first
- **Refactoring planning**: Understand ripple effects via related_to relationships
- **Pattern enforcement**: See which tasks follow architectural patterns from earlier tasks

**For Developers**:
- **Work sequencing**: Understand natural task order via follows relationships
- **Duplicate detection**: Identify duplicate work via duplicates relationships before implementation
- **Learning**: Discover reference implementations via references relationships

---

## User Personas

### Persona 1: Product Manager

**Profile**:
- **Role/Title**: Product manager overseeing feature development and delivery
- **Experience Level**: 3+ years product management, responsible for scope and delivery timelines
- **Key Characteristics**:
  - Tracks feature completion and identifies delivery risks
  - Needs to understand critical path and dependency chains
  - Reports on delivery status and blockers to stakeholders

**Goals Related to This Feature**:
1. Visualize full dependency graph for a feature to understand complexity
2. Identify tasks blocking downstream work to prioritize unblocking
3. Track spawned tasks from UAT findings to ensure quality issues are addressed
4. Report on delivery risks based on dependency chains

**Pain Points This Feature Addresses**:
- **Hidden Dependencies**: Cannot see full dependency chain or what's blocked by incomplete tasks
- **Critical Path Invisibility**: No way to identify high-impact tasks blocking many downstream tasks
- **Duplicate Work**: Tasks representing duplicate work not identified until late
- **Spawned Task Tracking**: Bug fixes and improvements from UAT not linked to source tasks

**Success Looks Like**:
Product managers can query "what does T-E13-F05-003 block" and see 3 downstream tasks, visualize the full dependency graph for feature E13-F05 with critical path highlighted, and identify all spawned tasks from UAT findings to ensure quality issues are tracked.

---

### Persona 2: AI Development Agent (Claude Code)

**Profile**:
- **Role/Title**: AI-powered development agent performing implementation tasks
- **Experience Level**: Expert technical knowledge, works autonomously selecting and executing tasks
- **Key Characteristics**:
  - Uses `shark task next` to select next available task
  - Should avoid tasks with incomplete dependencies
  - Benefits from seeing related tasks to maintain consistency

**Goals Related to This Feature**:
1. Avoid starting tasks with incomplete hard dependencies
2. Discover related tasks to reuse patterns and maintain consistency
3. Understand what will unblock when current task completes (motivation)
4. Link spawned tasks back to source tasks for context

**Pain Points This Feature Addresses**:
- **Wasted Work**: Starting tasks only to discover blocking dependencies mid-implementation
- **Pattern Inconsistency**: Missing related tasks that established patterns to follow
- **Motivation Gap**: Not knowing how completing current task unblocks downstream work
- **Lost Context**: Spawned bug fix tasks don't link back to source task for context

**Success Looks Like**:
`shark task next` automatically excludes tasks with incomplete dependencies, agent can query `shark task deps T-E13-F05-004` to see it depends on completed T-E13-F05-003 and T-E13-F05-001 (safe to start), and agent can see that completing current task will unblock 3 downstream tasks (motivation).

---

### Persona 3: Human Developer (Technical Lead)

**Profile**:
- **Role/Title**: Senior developer or tech lead managing code quality and architecture
- **Experience Level**: 5+ years development, responsible for architectural decisions
- **Key Characteristics**:
  - Reviews code and prioritizes review queue
  - Plans refactoring and understands ripple effects
  - Enforces architectural patterns across tasks

**Goals Related to This Feature**:
1. Prioritize code review for high-impact tasks blocking others
2. Understand refactoring ripple effects via related_to relationships
3. Ensure tasks follow established patterns via follows/references relationships
4. Track duplicate work to merge or cancel redundant tasks

**Pain Points This Feature Addresses**:
- **Review Prioritization**: Cannot identify which reviews are blocking downstream work
- **Refactoring Blindness**: Don't know which tasks are related and will be affected by changes
- **Pattern Drift**: Cannot see which tasks should follow architectural patterns from earlier tasks
- **Duplicate Work**: Duplicate tasks not discovered until code review or post-merge

**Success Looks Like**:
Tech leads can query `shark task blocks T-E13-F05-003` to see 2 tasks waiting on review, prioritize accordingly, and query `shark task related T-E13-F05-002` to see all tasks sharing localStorage patterns for refactoring planning.

---

## User Stories

### Must-Have Stories

**Story 1**: As a product manager, I want to see which tasks depend on T-E13-F05-003 so that I understand downstream impacts of delays on this task.

**Acceptance Criteria**:
- [ ] CLI command `shark task deps <task-key>` shows all relationships (incoming and outgoing)
- [ ] Output separates dependencies by type: depends_on, blocks, related_to, follows, spawned_from, duplicates, references
- [ ] Each relationship shows direction (this task → other, other → this task)
- [ ] Related task status is shown (completed, in_progress, blocked, todo)
- [ ] Output is human-readable with clear labeling

---

**Story 2**: As an AI agent, I want to link T-E13-F05-004 as depending on T-E13-F05-003 so that dependencies are tracked and I avoid starting tasks with incomplete dependencies.

**Acceptance Criteria**:
- [ ] CLI command `shark task link <task-key> --depends-on <other-task>` creates depends_on relationship
- [ ] Similar flags for other types: `--blocks`, `--related-to`, `--follows`, `--spawned-from`, `--duplicates`, `--references`
- [ ] Multiple relationships can be created in single command: `--depends-on T1,T2`
- [ ] Relationships are bidirectional (creating depends_on from A to B allows querying "what depends on B")
- [ ] Error if either task does not exist
- [ ] Prevent duplicate relationships (same from/to/type)

---

**Story 3**: As a developer, I want to find what tasks are blocked by T-E13-F05-003 so that I know what will unblock when this task completes.

**Acceptance Criteria**:
- [ ] CLI command `shark task blocked-by <task-key>` shows all tasks with depends_on relationship pointing to this task
- [ ] CLI command `shark task blocks <task-key>` shows all tasks this task blocks (outgoing blocks relationships)
- [ ] Output shows task key, title, and current status
- [ ] Output highlights tasks in todo or in_progress status (actionable blockers)
- [ ] Completed dependencies are shown but de-emphasized

---

**Story 4**: As a product manager, I want to visualize the dependency graph for T-E13-F05-004 so that I understand the full dependency chain and critical path.

**Acceptance Criteria**:
- [ ] CLI command `shark task graph <task-key>` generates ASCII dependency graph
- [ ] Graph shows upstream dependencies (depends_on relationships traversed recursively)
- [ ] Graph shows downstream dependents (tasks depending on this task, recursively)
- [ ] Graph uses visual indicators for task status (✓ completed, ○ todo, • in_progress, ✗ blocked)
- [ ] Graph clearly shows direction of dependencies with arrows or indentation
- [ ] Circular dependencies are detected and flagged as error

---

### Should-Have Stories

**Story 5**: As a tech lead, I want to find all tasks related to T-E13-F05-002 so that I can understand shared concerns and refactoring impacts.

**Acceptance Criteria**:
- [ ] CLI command `shark task related <task-key>` shows all related tasks regardless of relationship type
- [ ] Output groups by relationship type (depends_on, blocks, related_to, etc.)
- [ ] Can filter by specific types: `shark task related <task-key> --type related_to,follows`

---

**Story 6**: As an AI agent, I want `shark task next` to exclude tasks with incomplete dependencies so that I only work on tasks ready to start.

**Acceptance Criteria**:
- [ ] `shark task next` queries task_relationships for depends_on relationships
- [ ] Tasks with any incomplete depends_on relationships are excluded from results
- [ ] Tasks with only completed dependencies are candidates for selection
- [ ] Behavior documented in CLI help text

---

### Edge Case & Error Stories

**Error Story 1**: As a user, when I try to create a circular dependency (A depends on B, B depends on A), I want to receive a clear error so that I don't create invalid dependency graphs.

**Acceptance Criteria**:
- [ ] System detects circular dependencies when creating relationships
- [ ] Error message: "Circular dependency detected: T-A → T-B → T-A"
- [ ] Exit code 3 (invalid state)
- [ ] Relationship is not created

**Error Story 2**: As a user, when I try to link a non-existent task, I want a clear error message so that I can correct my mistake.

**Acceptance Criteria**:
- [ ] Error message: "Task T-XYZ not found"
- [ ] Exit code 1 (not found)
- [ ] Suggests checking task key with `shark task list`

**Error Story 3**: As a user, when I create a duplicate relationship (same from/to/type), I want the system to prevent duplicates.

**Acceptance Criteria**:
- [ ] Database UNIQUE constraint on (from_task_id, to_task_id, relationship_type)
- [ ] Error message: "Relationship already exists: T-A depends_on T-B"
- [ ] Exit code 3 (invalid state)

---

## Requirements

### Functional Requirements

**Category: Relationship Creation**

1. **REQ-F-010**: Bidirectional Task Relationships
   - **Description**: System must support typed relationships between tasks with seven distinct types
   - **User Story**: Links to Story 1, Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] CREATE TABLE task_relationships with columns: id, from_task_id, to_task_id, relationship_type, created_at
     - [ ] Relationship types: depends_on, blocks, related_to, follows, spawned_from, duplicates, references
     - [ ] Foreign keys enforce task existence (from_task_id → tasks.id, to_task_id → tasks.id)
     - [ ] CASCADE DELETE when either task is deleted
     - [ ] CHECK constraint enforces valid relationship_type values
     - [ ] UNIQUE constraint prevents duplicate relationships (from_task_id, to_task_id, relationship_type)

---

**Category: Relationship Management**

2. **REQ-F-011**: Relationship Management Commands
   - **Description**: System must provide CLI commands for creating and viewing relationships
   - **User Story**: Links to Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `shark task link <task-key> --depends-on <other-task>` creates depends_on relationship
     - [ ] Similar flags for other types: `--blocks`, `--related-to`, `--follows`, `--spawned-from`, `--duplicates`, `--references`
     - [ ] Multiple relationships in one command: `--depends-on T1,T2,T3`
     - [ ] `shark task deps <task-key>` shows all relationships (incoming and outgoing)
     - [ ] `shark task graph <task-key>` generates dependency graph visualization
     - [ ] All commands support `--json` flag for machine-readable output

---

**Category: Relationship Querying**

3. **REQ-F-012**: Relationship Querying
   - **Description**: System must support queries based on relationships
   - **User Story**: Links to Story 3, Story 5
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `shark task blocked-by <task-key>` shows incoming depends_on relationships (tasks depending on this)
     - [ ] `shark task blocks <task-key>` shows outgoing blocks relationships (tasks this blocks)
     - [ ] `shark task related <task-key>` shows all related tasks regardless of type
     - [ ] Output includes relationship type, direction, task status
     - [ ] Can filter by relationship type: `--type depends_on,blocks`

---

**Category: Dependency Validation**

4. **REQ-F-013**: Circular Dependency Detection
   - **Description**: System must detect and prevent circular dependencies
   - **User Story**: Links to Error Story 1
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] When creating depends_on or blocks relationship, traverse graph to detect cycles
     - [ ] Error if cycle detected: "Circular dependency detected: T-A → T-B → T-C → T-A"
     - [ ] Relationship not created if cycle would result
     - [ ] Performance target: cycle detection <100ms for graphs with <500 tasks

---

**Category: Task Selection Integration**

5. **REQ-F-014**: Dependency-Aware Task Selection
   - **Description**: `shark task next` must exclude tasks with incomplete dependencies
   - **User Story**: Links to Story 6
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] `shark task next` queries task_relationships for depends_on relationships
     - [ ] Tasks with incomplete depends_on dependencies are excluded
     - [ ] Tasks with only completed dependencies are candidates
     - [ ] Performance target: dependency check adds <200ms to task selection

---

### Non-Functional Requirements

**Performance**

1. **REQ-NF-001**: Relationship Query Performance
   - **Description**: Querying all relationships for a task must complete in <200ms
   - **Measurement**: Execute `shark task deps <task-key>` and measure execution time
   - **Target**: p95 < 200ms for tasks with up to 50 relationships
   - **Justification**: Frequently queried during task selection and planning; must be fast

2. **REQ-NF-002**: Graph Generation Performance
   - **Description**: Generating dependency graph must complete in <2 seconds
   - **Measurement**: Execute `shark task graph <task-key>` and measure execution time
   - **Target**: p95 < 2 seconds for graphs with up to 100 tasks
   - **Justification**: Graph generation is information-dense; users tolerate slight delay for comprehensive view

**Data Integrity**

3. **REQ-NF-003**: Foreign Key Enforcement
   - **Description**: All relationships must enforce referential integrity via foreign keys
   - **Implementation**: task_relationships.from_task_id → tasks.id, task_relationships.to_task_id → tasks.id
   - **Compliance**: SQLite `PRAGMA foreign_keys = ON`
   - **Risk Mitigation**: Prevents orphaned relationships when tasks are deleted

4. **REQ-NF-004**: Cascade Deletion
   - **Description**: Deleting a task must cascade delete all related relationships
   - **Implementation**: FOREIGN KEY ... ON DELETE CASCADE
   - **Compliance**: SQLite foreign key constraints
   - **Risk Mitigation**: Prevents data inconsistency and orphaned relationships

5. **REQ-NF-005**: Unique Relationship Constraint
   - **Description**: System must prevent duplicate relationships (same from/to/type)
   - **Implementation**: UNIQUE constraint on (from_task_id, to_task_id, relationship_type)
   - **Compliance**: SQLite UNIQUE constraint
   - **Risk Mitigation**: Prevents accidental duplicate relationship creation

**Usability**

6. **REQ-NF-006**: Human-Readable Output
   - **Description**: All CLI commands must provide human-readable output by default
   - **Implementation**: Format tables, use unicode symbols (✓○•✗), group by relationship type
   - **Testing**: Manual review of CLI output for readability
   - **Risk Mitigation**: Improves developer experience and adoption

7. **REQ-NF-007**: JSON Output Mode
   - **Description**: All CLI commands must support `--json` flag
   - **Implementation**: Marshal results to JSON when `--json` flag present
   - **Testing**: Verify all commands produce valid, parseable JSON
   - **Risk Mitigation**: Enables AI agent automation and scripting

**Backward Compatibility**

8. **REQ-NF-008**: Migration from depends_on Field
   - **Description**: Existing depends_on text field values must be migrated to task_relationships table
   - **Implementation**: Migration script parses comma-separated task keys, creates depends_on relationships
   - **Testing**: Test migration with existing database containing depends_on values
   - **Risk Mitigation**: Existing dependency data is preserved

---

## Database Schema

### New Table: task_relationships

```sql
CREATE TABLE task_relationships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_task_id INTEGER NOT NULL,
    to_task_id INTEGER NOT NULL,
    relationship_type TEXT CHECK (relationship_type IN (
        'depends_on',    -- Task from_task depends on to_task completing (hard dependency)
        'blocks',        -- Task from_task blocks to_task from proceeding (explicit blocker)
        'related_to',    -- Tasks share common code/concerns (soft relationship)
        'follows',       -- Task from_task naturally follows to_task (sequence, not blocking)
        'spawned_from',  -- Task from_task was created from UAT/bugs in to_task
        'duplicates',    -- Tasks represent duplicate work (should merge)
        'references'     -- Task from_task consults/uses output of to_task
    )) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (to_task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    UNIQUE(from_task_id, to_task_id, relationship_type)
);

-- Index for finding all relationships for a task (in either direction)
CREATE INDEX idx_task_relationships_from ON task_relationships(from_task_id);
CREATE INDEX idx_task_relationships_to ON task_relationships(to_task_id);

-- Index for filtering by relationship type
CREATE INDEX idx_task_relationships_type ON task_relationships(relationship_type);

-- Composite index for specific relationship queries
CREATE INDEX idx_task_relationships_from_type ON task_relationships(from_task_id, relationship_type);
CREATE INDEX idx_task_relationships_to_type ON task_relationships(to_task_id, relationship_type);
```

**Schema Design Rationale**:
- **Separate table vs. TEXT field**: Enables bidirectional queries, typed relationships, and referential integrity
- **CHECK constraint for relationship_type**: Seven fixed types unlikely to change; simpler than separate types table
- **UNIQUE constraint on (from, to, type)**: Prevents duplicate relationships but allows same task pair with different types (e.g., A depends_on B AND A related_to B)
- **Indexes on both from_task_id and to_task_id**: Enables fast bidirectional queries ("what does X depend on" and "what depends on X")
- **ON DELETE CASCADE**: Relationships meaningless without both tasks; cascade prevents orphaned relationships

---

### Migration from Existing depends_on Field

**Migration Strategy**:

```sql
-- Migration to populate task_relationships from existing depends_on field

-- Step 1: Parse comma-separated depends_on field and create relationships
-- Example: Task T-E13-F05-004 has depends_on = "T-E13-F05-003,T-E13-F05-001"
-- Creates:
--   task_relationships: from_task_id=404, to_task_id=403, type='depends_on'
--   task_relationships: from_task_id=404, to_task_id=401, type='depends_on'

-- Step 2: Deprecate depends_on field (keep for backward compatibility, mark as read-only)
-- ALTER TABLE tasks ADD COLUMN depends_on_deprecated TEXT;
-- UPDATE tasks SET depends_on_deprecated = depends_on;
-- (depends_on field still populated from task_relationships for backward compatibility)

-- Step 3: Future migration can drop depends_on field once all clients updated
```

**Backward Compatibility**:
- Existing `depends_on` field maintained for read-only backward compatibility
- New code reads from `task_relationships` table
- Migration script runs automatically on first launch after upgrade

---

## CLI Commands Specification

### Command: `shark task link`

**Purpose**: Create typed relationship between tasks

**Syntax**:
```bash
shark task link <task-key> --<relationship-type> <other-task-key>[,<other-task-key>...] [--json]
```

**Arguments**:
- `<task-key>`: Required. Source task key (e.g., T-E13-F05-004)
- `--depends-on <keys>`: Create depends_on relationship (comma-separated for multiple)
- `--blocks <keys>`: Create blocks relationship
- `--related-to <keys>`: Create related_to relationship
- `--follows <keys>`: Create follows relationship
- `--spawned-from <keys>`: Create spawned_from relationship
- `--duplicates <keys>`: Create duplicates relationship
- `--references <keys>`: Create references relationship
- `--json`: Optional. Output JSON

**Examples**:
```bash
# Single dependency
shark task link T-E13-F05-004 --depends-on T-E13-F05-003

# Multiple dependencies
shark task link T-E13-F05-004 --depends-on T-E13-F05-003,T-E13-F05-001

# Multiple relationship types in one command
shark task link T-E13-F05-004 \
  --depends-on T-E13-F05-003,T-E13-F05-001 \
  --related-to T-E13-F05-002

# Spawned task from UAT findings
shark task link T-E13-F05-008 --spawned-from T-E13-F05-002

# Reference implementation
shark task link T-E13-F05-005 --references T-E13-F05-003
```

**Output**:
```
Created 3 relationships for T-E13-F05-004:
  depends_on → T-E13-F05-003 (Create useTheme Composable)
  depends_on → T-E13-F05-001 (Define Dark Mode CSS Variables)
  related_to → T-E13-F05-002 (Implement Flash Prevention Script)
```

**Error Cases**:
- Task not found → Exit code 1, message "Task T-XYZ not found"
- Circular dependency → Exit code 3, message "Circular dependency detected: T-A → T-B → T-A"
- Duplicate relationship → Exit code 3, message "Relationship already exists: T-A depends_on T-B"
- Invalid relationship type → Exit code 3, message "Invalid flag: --invalid (must be one of: --depends-on, --blocks, ...)"

---

### Command: `shark task deps`

**Purpose**: View all relationships for a task

**Syntax**:
```bash
shark task deps <task-key> [--type <types>] [--json]
```

**Arguments**:
- `<task-key>`: Required. Task key
- `--type <types>`: Optional. Filter by relationship types (comma-separated)
- `--json`: Optional. Output JSON

**Examples**:
```bash
# View all relationships
shark task deps T-E13-F05-004

# View only dependencies
shark task deps T-E13-F05-004 --type depends_on,blocks
```

**Output (Human-Readable)**:
```
T-E13-F05-004: Create ThemeToggle Component

Dependencies (this task depends on):
  ✓ T-E13-F05-003: Create useTheme() Composable (completed 2025-12-26)
  ✓ T-E13-F05-001: Define Dark Mode CSS Variables (completed 2025-12-26)

Related Tasks:
  ✓ T-E13-F05-002: Implement Flash Prevention Script (shares localStorage key)

Blocks (this task blocks):
  ○ T-E13-F05-007: Dark Mode Integration Testing (waiting on this task)
  ○ DEPLOY-E13: Deploy E13 to Staging (waiting on this task)

Spawned Tasks (created from this task):
  (none)

References (this task references):
  (none)

Legend: ✓ completed | • in_progress | ○ todo | ✗ blocked
```

**JSON Output**:
```json
{
  "task_key": "T-E13-F05-004",
  "task_title": "Create ThemeToggle Component",
  "relationships": {
    "depends_on": [
      {
        "task_key": "T-E13-F05-003",
        "title": "Create useTheme() Composable",
        "status": "completed",
        "completed_at": "2025-12-26T14:30:00Z"
      },
      {
        "task_key": "T-E13-F05-001",
        "title": "Define Dark Mode CSS Variables",
        "status": "completed",
        "completed_at": "2025-12-26T10:15:00Z"
      }
    ],
    "related_to": [
      {
        "task_key": "T-E13-F05-002",
        "title": "Implement Flash Prevention Script",
        "status": "completed"
      }
    ],
    "blocks": [
      {
        "task_key": "T-E13-F05-007",
        "title": "Dark Mode Integration Testing",
        "status": "todo"
      },
      {
        "task_key": "DEPLOY-E13",
        "title": "Deploy E13 to Staging",
        "status": "todo"
      }
    ]
  }
}
```

---

### Command: `shark task blocked-by`

**Purpose**: Show what blocks this task (incoming dependencies)

**Syntax**:
```bash
shark task blocked-by <task-key> [--json]
```

**Example**:
```bash
shark task blocked-by T-E13-F05-007
```

**Output**:
```
T-E13-F05-007: Dark Mode Integration Testing

Blocked by (must complete first):
  • T-E13-F05-004: Create ThemeToggle Component (in_progress)
  ○ T-E13-F05-006: Update Documentation (todo)

Legend: ✓ completed | • in_progress | ○ todo | ✗ blocked
```

---

### Command: `shark task blocks`

**Purpose**: Show what this task blocks (outgoing blockers)

**Syntax**:
```bash
shark task blocks <task-key> [--json]
```

**Example**:
```bash
shark task blocks T-E13-F05-003
```

**Output**:
```
T-E13-F05-003: Create useTheme() Composable

Blocks (waiting on this task):
  • T-E13-F05-004: Create ThemeToggle Component (in_progress, can proceed now)
  ○ T-E13-F05-005: Theme Persistence Settings Page (todo, can start now)

This task is completed - all downstream tasks are unblocked.
```

---

### Command: `shark task graph`

**Purpose**: Visualize dependency graph (ASCII art)

**Syntax**:
```bash
shark task graph <task-key> [--depth <n>] [--json]
```

**Arguments**:
- `<task-key>`: Required. Root task for graph
- `--depth <n>`: Optional. Limit graph depth (default: unlimited)
- `--json`: Optional. Output structured graph data

**Example**:
```bash
shark task graph T-E13-F05-004
```

**Output (ASCII Graph)**:
```
Dependency Graph for T-E13-F05-004: Create ThemeToggle Component

Upstream Dependencies (must complete first):
  ✓ T-E13-F05-003: Create useTheme() Composable
    ✓ T-E13-F05-001: Define Dark Mode CSS Variables (shared dep)
  ✓ T-E13-F05-001: Define Dark Mode CSS Variables

  • T-E13-F05-004: Create ThemeToggle Component ← YOU ARE HERE

Downstream Dependents (waiting on this):
  ○ T-E13-F05-007: Dark Mode Integration Testing
    ○ DEPLOY-E13: Deploy E13 to Staging
      ○ RELEASE-1.0: Release v1.0

Related (not blocking):
  ✓ T-E13-F05-002: Implement Flash Prevention Script

Legend: ✓ completed | • in_progress | ○ todo | ✗ blocked
```

**Alternative Output (Tree Format)**:
```
T-E13-F05-004: Create ThemeToggle Component [IN_PROGRESS]
│
├─ DEPENDS ON:
│  ├─ ✓ T-E13-F05-003: Create useTheme() Composable
│  │  └─ ✓ T-E13-F05-001: Define Dark Mode CSS Variables
│  └─ ✓ T-E13-F05-001: Define Dark Mode CSS Variables
│
├─ BLOCKS:
│  ├─ ○ T-E13-F05-007: Dark Mode Integration Testing
│  │  └─ ○ DEPLOY-E13: Deploy E13 to Staging
│  │     └─ ○ RELEASE-1.0: Release v1.0
│
└─ RELATED TO:
   └─ ✓ T-E13-F05-002: Implement Flash Prevention Script
```

---

## User Journeys

### Journey 1: Product Manager Understands Feature Dependencies

**Persona**: Product Manager

**Scenario**: PM needs to report on E13-F05 feature delivery timeline and identify critical path

**Steps**:
1. PM lists all feature tasks: `shark task list E13-F05`
2. Identifies key task: T-E13-F05-004 (ThemeToggle Component)
3. Visualizes dependencies: `shark task graph T-E13-F05-004`
4. Output shows:
   - Upstream: Depends on T-E13-F05-003 (completed) and T-E13-F05-001 (completed)
   - Downstream: Blocks T-E13-F05-007 (Integration Testing) which blocks deployment
5. Identifies T-E13-F05-004 is on critical path for deployment
6. Checks what's blocked: `shark task blocks T-E13-F05-004`
7. Sees 2 tasks waiting (Integration Testing, Deployment)
8. Reports to stakeholders: "T-E13-F05-004 is critical path; completing it unblocks integration testing and deployment"

**Outcome**: PM has complete visibility into dependency chain, identifies critical path, and provides accurate delivery forecast

---

### Journey 2: AI Agent Avoids Starting Task with Incomplete Dependencies

**Persona**: AI Development Agent

**Scenario**: Agent selects next task but should avoid tasks with incomplete dependencies

**Steps**:
1. Agent queries: `shark task next --agent frontend`
2. System internally checks task_relationships for each candidate task
3. T-E13-F05-007 (Integration Testing) has depends_on relationship to T-E13-F05-004 (status: in_progress)
4. T-E13-F05-007 excluded from results (incomplete dependency)
5. T-E13-F05-004 returned as next task (all dependencies completed)
6. Agent starts T-E13-F05-004 confidently
7. During implementation, agent creates spawned task for bug fix:
   ```bash
   shark task create --epic E13 --feature F05 --title "Fix Safari Theme Persistence Bug"
   shark task link T-E13-F05-009 --spawned-from T-E13-F05-004
   ```
8. Spawned task is linked back to source task for context

**Outcome**: Agent only works on tasks with completed dependencies, avoids wasted work, and maintains traceability for spawned tasks

---

### Journey 3: Tech Lead Prioritizes Code Review Based on Blockers

**Persona**: Human Developer (Technical Lead)

**Scenario**: Tech lead has 5 tasks in `ready_for_review` status and must prioritize review queue

**Steps**:
1. Lists tasks awaiting review: `shark task list --status ready_for_review`
2. For each task, checks what it blocks:
   ```bash
   shark task blocks T-E13-F05-003
   shark task blocks T-E13-F05-006
   shark task blocks T-E13-F02-004
   ```
3. T-E13-F05-003 output shows:
   - Blocks T-E13-F05-004 (in_progress, agent currently working)
   - Blocks T-E13-F05-005 (todo, waiting to start)
4. T-E13-F05-006 output shows: "No downstream tasks blocked"
5. Tech lead prioritizes T-E13-F05-003 for immediate review (unblocks 2 tasks)
6. Reviews and approves T-E13-F05-003
7. Agent working on T-E13-F05-004 can now complete without blocker
8. T-E13-F05-005 can be started by another agent

**Outcome**: Tech lead prioritizes high-impact reviews, reduces blockers, increases team velocity

---

### Journey 4: Developer Discovers Related Implementation Pattern

**Persona**: Human Developer

**Scenario**: Developer starting new task involving localStorage needs to find related implementations

**Steps**:
1. Developer assigned T-E13-F05-005 (Theme Persistence Settings Page)
2. Checks dependencies and related tasks: `shark task deps T-E13-F05-005`
3. Output shows related_to relationship: T-E13-F05-003 (useTheme Composable) and T-E13-F05-002 (Flash Prevention)
4. Both tasks involve localStorage and theme preference
5. Developer views notes: `shark task notes T-E13-F05-003 --type decision,implementation`
6. Learns singleton pattern was used for theme state
7. Implements settings page using same pattern for consistency
8. Creates relationship: `shark task link T-E13-F05-005 --follows T-E13-F05-003` (pattern established by earlier task)

**Outcome**: Developer discovers established patterns via relationships, maintains architectural consistency, avoids re-inventing patterns

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Create Dependency Relationship**
- **Given** tasks T-E13-F05-004 and T-E13-F05-003 exist in database
- **When** user executes `shark task link T-E13-F05-004 --depends-on T-E13-F05-003`
- **Then** task_relationships record created with from_task_id=404, to_task_id=403, relationship_type='depends_on'
- **And** success message displayed: "Created 1 relationship for T-E13-F05-004: depends_on → T-E13-F05-003"

**Scenario 2: Query All Relationships**
- **Given** task T-E13-F05-004 has 2 depends_on relationships, 1 related_to relationship, and blocks 2 downstream tasks
- **When** user executes `shark task deps T-E13-F05-004`
- **Then** output shows all 5 relationships grouped by type (depends_on, related_to, blocks)
- **And** each relationship shows related task key, title, and status
- **And** completed dependencies show completion date

**Scenario 3: Query Blocked Tasks**
- **Given** task T-E13-F05-003 is completed and blocks tasks T-E13-F05-004 (in_progress) and T-E13-F05-005 (todo)
- **When** user executes `shark task blocks T-E13-F05-003`
- **Then** output shows 2 tasks: T-E13-F05-004 and T-E13-F05-005
- **And** output indicates "This task is completed - all downstream tasks are unblocked"

**Scenario 4: Circular Dependency Detection**
- **Given** tasks T-A, T-B, T-C exist
- **And** T-A depends_on T-B, T-B depends_on T-C
- **When** user executes `shark task link T-C --depends-on T-A`
- **Then** error message displayed: "Circular dependency detected: T-C → T-A → T-B → T-C"
- **And** relationship is NOT created
- **And** exit code is 3 (invalid state)

**Scenario 5: Duplicate Relationship Prevention**
- **Given** relationship already exists: T-E13-F05-004 depends_on T-E13-F05-003
- **When** user executes `shark task link T-E13-F05-004 --depends-on T-E13-F05-003`
- **Then** error message displayed: "Relationship already exists: T-E13-F05-004 depends_on T-E13-F05-003"
- **And** exit code is 3 (invalid state)

**Scenario 6: Dependency Graph Visualization**
- **Given** task T-E13-F05-004 has upstream dependencies (T-E13-F05-003, T-E13-F05-001) and downstream dependents (T-E13-F05-007)
- **When** user executes `shark task graph T-E13-F05-004`
- **Then** ASCII graph displayed showing:
  - Upstream section with completed dependencies marked with ✓
  - Current task marked with "YOU ARE HERE"
  - Downstream section with waiting tasks marked with ○
  - Clear visual hierarchy using indentation or tree structure

**Scenario 7: Task Next Excludes Incomplete Dependencies**
- **Given** task T-E13-F05-007 depends_on T-E13-F05-004 (status: in_progress, not completed)
- **When** user executes `shark task next --agent frontend`
- **Then** T-E13-F05-007 is NOT in results (excluded due to incomplete dependency)
- **And** only tasks with all completed dependencies are returned

---

## Out of Scope

### Explicitly Excluded

1. **Automatic Dependency Detection**
   - **Why**: Complex AI/code analysis feature; high implementation cost
   - **Future**: Phase 3 enhancement if AI analysis proves valuable
   - **Workaround**: Manual relationship creation by agents and developers during task creation

2. **Dependency Weight/Priority**
   - **Why**: Adds complexity; simple dependency types sufficient for Phase 2
   - **Future**: Could add priority field if users need weighted critical path analysis
   - **Workaround**: Use relationship types (depends_on vs. follows) to distinguish hard vs. soft dependencies

3. **Dependency Reason/Notes**
   - **Why**: Adds schema complexity; notes system (E10-F01) handles this use case
   - **Future**: Not planned; use blocker notes for reasoning
   - **Workaround**: Add blocker note explaining why dependency exists

4. **Dependency Change History**
   - **Why**: Audit trail for relationships low priority; task_history covers status changes
   - **Future**: Phase 3 if compliance requires relationship audit trail
   - **Workaround**: N/A (relationships rarely change after creation)

5. **Visual Graph Rendering (HTML/SVG)**
   - **Why**: CLI tool with ASCII output; web UI out of scope
   - **Future**: Web UI enhancement if Shark adds GUI
   - **Workaround**: ASCII graph with tree structure for CLI

6. **Dependency Templates (Common Patterns)**
   - **Why**: Limited reuse patterns identified; premature optimization
   - **Future**: Phase 3 if common dependency patterns emerge
   - **Workaround**: Manual relationship creation

---

## Success Metrics

### Primary Metrics

1. **Relationship Adoption Rate**
   - **What**: Percentage of tasks with at least 1 relationship (depends_on, blocks, or related_to)
   - **Target**: 60% of tasks in multi-task features
   - **Timeline**: 1 month after Phase 2 release
   - **Measurement**: SQL query: `SELECT COUNT(*) FROM tasks WHERE id IN (SELECT DISTINCT from_task_id FROM task_relationships)`

2. **Dependency Graph Usage**
   - **What**: Frequency of `shark task graph` and `shark task deps` usage
   - **Target**: 10+ uses per week across team
   - **Timeline**: 1 month after Phase 2 release
   - **Measurement**: CLI usage analytics

3. **Wasted Work Reduction**
   - **What**: Reduction in tasks started then blocked mid-implementation
   - **Target**: 50% reduction in tasks transitioning todo → in_progress → blocked
   - **Timeline**: 2 months after Phase 2 release
   - **Measurement**: Database query on task_history transitions

---

### Secondary Metrics

- **Review Prioritization**: Tech leads use `shark task blocks` to prioritize reviews (5+ uses per week)
- **Spawned Task Tracking**: 80% of bug fix tasks have spawned_from relationship to source task
- **Critical Path Identification**: PMs identify critical path using dependency graphs (3+ uses per sprint)

---

## Dependencies & Integrations

### Dependencies

- **Existing Tables**: Requires `tasks` table (already exists)
- **CLI Framework**: Uses Cobra command structure in `internal/cli/commands/`
- **Repository Pattern**: Follows existing repository pattern from `internal/repository/`

### Integration Requirements

- **Task Next Integration**: `shark task next` must query task_relationships to exclude tasks with incomplete dependencies (REQ-F-014)
- **Note Integration**: Blocker notes from E10-F01 provide reasoning for dependencies
- **Task Creation Integration**: Future enhancement to suggest relationships during task creation based on epic/feature

### Downstream Dependencies

Features that depend on E10-F03:
- **E10-F04**: Acceptance Criteria & Search (search can filter by tasks with specific relationships)
- **E10-F05**: Work Sessions & Resume Context (resume command can show blocked dependencies)

---

## Implementation Plan

### Database Migration

**Migration File**: `internal/db/migrations/012_add_task_relationships.sql`

```sql
-- Create task_relationships table
CREATE TABLE IF NOT EXISTS task_relationships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_task_id INTEGER NOT NULL,
    to_task_id INTEGER NOT NULL,
    relationship_type TEXT CHECK (relationship_type IN (
        'depends_on', 'blocks', 'related_to', 'follows',
        'spawned_from', 'duplicates', 'references'
    )) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (to_task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    UNIQUE(from_task_id, to_task_id, relationship_type)
);

-- Indexes for bidirectional queries
CREATE INDEX idx_task_relationships_from ON task_relationships(from_task_id);
CREATE INDEX idx_task_relationships_to ON task_relationships(to_task_id);
CREATE INDEX idx_task_relationships_type ON task_relationships(relationship_type);
CREATE INDEX idx_task_relationships_from_type ON task_relationships(from_task_id, relationship_type);
CREATE INDEX idx_task_relationships_to_type ON task_relationships(to_task_id, relationship_type);

-- Migrate existing depends_on field to task_relationships
-- (Migration code will parse comma-separated depends_on field and create relationships)
```

**Backward Compatibility**:
- Existing `depends_on` field maintained for backward compatibility (populated from task_relationships)
- Migration automatically populates task_relationships from existing depends_on values

---

### Code Organization

- **Repository**: `internal/repository/task_relationship_repository.go` (new file)
- **Models**: `internal/models/task_relationship.go` (new file)
- **Commands**:
  - `internal/cli/commands/task_link.go` (new file for `task link`)
  - `internal/cli/commands/task_deps.go` (new file for `task deps`, `task blocked-by`, `task blocks`, `task graph`)
  - Modify `internal/cli/commands/task_next.go` to integrate dependency checking
- **Tests**:
  - `internal/repository/task_relationship_repository_test.go`
  - `internal/cli/commands/task_link_test.go`
  - `internal/cli/commands/task_deps_test.go`

---

### Testing Strategy

1. **Unit Tests**: Repository methods (CreateRelationship, GetRelationships, DetectCycle, GetDependencyGraph)
2. **Integration Tests**: Full CLI commands with real database and multiple tasks
3. **Performance Tests**:
   - Load 100 tasks with 500 relationships, measure query performance
   - Measure cycle detection performance on large graphs (500+ tasks)
   - Measure graph generation performance
4. **Edge Cases**:
   - Circular dependency detection (A→B→C→A)
   - Self-dependency prevention (A→A)
   - Orphaned relationship cleanup (task deletion cascades)
   - Duplicate relationship prevention
   - Large dependency graphs (100+ tasks, 500+ relationships)
   - Task with no relationships

---

## Open Questions

- **Q1**: Should the system automatically create inverse relationships (e.g., creating A depends_on B also creates B blocks A)?
  - **Recommendation**: No - keeps relationships explicit and prevents confusion. Users create specific relationship types as needed.
- **Q2**: Should `shark task graph` support multiple output formats (ASCII tree, DOT format for Graphviz)?
  - **Recommendation**: ASCII tree for Phase 2, DOT format for Phase 3 if users request it
- **Q3**: Should circular dependencies be prevented (error on creation) or allowed with warning?
  - **Recommendation**: Prevent with error - circular dependencies indicate modeling problem that should be fixed
- **Q4**: Should spawned_from relationships be automatically created when `shark task create` includes `--spawned-from` flag?
  - **Recommendation**: Yes - add `--spawned-from` flag to `shark task create` for convenience

---

*Last Updated*: 2025-12-26
*Status*: Ready for Review
*Author*: BusinessAnalyst Agent
