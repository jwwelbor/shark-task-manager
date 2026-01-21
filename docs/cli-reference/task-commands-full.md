# Task Commands - Full Reference

Complete reference for all task management commands.

## `shark task create`

Create a new task within a feature.

**Positional Syntax (Recommended):**
```bash
# 3-argument format: epic, feature, title
shark task create <epic-key> <feature-key> "<title>" [flags]

# 2-argument format: combined epic-feature, title
shark task create <epic-feature-key> "<title>" [flags]
```

**Flag Syntax (Legacy, still supported):**
```bash
shark task create --epic=<epic-key> --feature=<feature-key> --title="<title>" [flags]
```

**Optional Flags:**
- `--agent <type>`: Agent type (any non-empty string)
  - **Recommended types** (have specific templates): `frontend`, `backend`, `api`, `testing`, `devops`, `general`
  - **Custom types** (use general template): `architect`, `business-analyst`, `qa`, `tech-lead`, `product-manager`, `ux-designer`, etc.
  - Custom agent types are fully supported and stored exactly as entered
- `--priority <1-10>`: Priority (1 = highest, 10 = lowest, default: 5)
- `--description <string>`: Detailed description
- `--depends-on <task-keys>`: Comma-separated list of dependency task keys
- `--file <path>`: Custom file path (relative to root, must include .md)
- `--force`: Reassign file if already claimed by another task
- `--json`: Output in JSON format

**Examples:**

```bash
# Create task with positional syntax - 3 arguments (recommended)
shark task create E07 F01 "Implement JWT validation"
shark task create e07 f01 "Implement JWT validation"  # Case insensitive

# Create task with positional syntax - 2 arguments
shark task create E07-F01 "Implement JWT validation"
shark task create e07-f01 "Implement JWT validation"  # Case insensitive

# Create task with flag syntax (legacy)
shark task create --epic=E07 --feature=F01 --title="Implement JWT validation"

# Create task with agent and priority
shark task create E07 F01 "Implement JWT validation" --agent=backend --priority=3

# Create task with custom agent type (multi-agent workflows)
shark task create E07 F01 "Design system architecture" --agent=architect --priority=2
shark task create E07 F01 "Elaborate user requirements" --agent=business-analyst --priority=4
shark task create E07 F01 "Create test strategy" --agent=qa --priority=3

# Create task with dependencies
shark task create E07 F01 "Add token refresh" \
  --agent=backend \
  --depends-on="E07-F01-001,E07-F01-002"

# Create task with custom file path
shark task create E07 F01 "Legacy auth migration" \
  --file="docs/tasks/legacy/auth-migration.md" \
  --force
```

---

## `shark task list`

List tasks with optional filtering.

**Usage:**
```bash
shark task list [EPIC] [FEATURE] [--status=<status>] [--agent=<type>] [--with-actions] [--json]
# OR (flag syntax, backward compatible)
shark task list [--epic=<epic-key>] [--feature=<feature-key>] [--status=<status>] [--agent=<type>] [--with-actions] [--json]
```

**Filter Flags:**
- `--status <status>`: Filter by status (`todo`, `in_progress`, `ready_for_review`, `completed`, `blocked`)
- `--agent <type>`: Filter by agent type
- `--with-actions`: Include orchestrator actions with each task (optional, for batch orchestrator polling)

**Examples:**

```bash
# List all tasks
shark task list

# List tasks in epic (positional)
shark task list E07

# List tasks in epic and feature (positional)
shark task list E07 F01
shark task list E07-F01  # Alternative format

# List tasks in epic and feature (flag syntax)
shark task list --epic=E07 --feature=F01

# Filter by status
shark task list --status=todo --json
shark task list --status=in_progress --json

# Filter by agent (standard types)
shark task list --agent=backend --json

# Filter by agent (custom types)
shark task list --agent=architect --json
shark task list --agent=business-analyst --json

# Combine filters
shark task list E07 --agent=backend --status=todo --json

# List with orchestrator actions (for orchestrator polling)
shark task list --status=ready_for_development --with-actions --json
shark task list E07 F01 --with-actions --json

# Combine all filters
shark task list E07 --agent=backend --status=ready_for_development --with-actions --json
```

**About the --with-actions Flag:**

The `--with-actions` flag includes `orchestrator_action` metadata with each task. This is useful for orchestrators that need to batch-fetch ready tasks and their execution instructions.

**Without flag (default):**
```json
[
  {
    "id": 123,
    "key": "E01-F03-002",
    "status": "ready_for_development",
    "title": "Implement feature X"
  }
]
```

**With --with-actions flag:**
```json
[
  {
    "id": 123,
    "key": "E01-F03-002",
    "status": "ready_for_development",
    "title": "Implement feature X",
    "orchestrator_action": {
      "action": "spawn_agent",
      "agent_type": "developer",
      "skills": ["test-driven-development", "implementation"],
      "instruction": "Launch a developer agent to implement task E01-F03-002..."
    }
  }
]
```

See [Task Update API Response Format](orchestrator-actions.md) section for complete action field documentation.

---

## `shark task get`

Get detailed information about a specific task.

**Usage:**
```bash
shark task get <task-key> [--json]
```

**Supports:**
- Short format: `E07-F01-001` (recommended)
- Traditional format: `T-E07-F01-001`
- Slugged keys: `E07-F01-001-implement-jwt-validation`, `T-E07-F01-001-implement-jwt-validation`
- Case insensitive: `e07-f01-001`, `t-e07-f01-001`

**Examples:**

```bash
# Get task details (short format, recommended)
shark task get E07-F01-001
shark task get e07-f01-001  # Case insensitive

# Get task details (traditional format)
shark task get T-E07-F01-001

# Get task details (JSON)
shark task get E07-F01-001 --json

# Using slugged key
shark task get E07-F01-001-implement-jwt-validation --json
```

---

## `shark task next`

Find the next available task to work on.

**Flags:**
- `--agent <type>`: Filter by agent type
- `--epic <epic-key>`: Filter by epic
- `--json`: Output in JSON format

**Examples:**

```bash
# Get next available task (any agent)
shark task next --json

# Get next task for specific agent (standard types)
shark task next --agent=backend --json
shark task next --agent=frontend --json

# Get next task for specific agent (custom types)
shark task next --agent=architect --json
shark task next --agent=business-analyst --json

# Get next task in specific epic
shark task next --epic=E07 --json

# Combine filters
shark task next --epic=E07 --agent=backend --json
```

**Returns:**
- Tasks in `todo` status
- With all dependencies completed
- Sorted by priority (1 = highest)

---

## `shark task start`

Start working on a task (transition from `todo` to `in_progress`).

**Usage:**
```bash
shark task start <task-key> [--agent=<agent-id>] [--json]
```

**Examples:**

```bash
# Start task (short format, recommended)
shark task start E07-F01-001
shark task start e07-f01-001  # Case insensitive

# Start task (traditional format)
shark task start T-E07-F01-001

# Start task with agent tracking
shark task start E07-F01-001 --agent="ai-agent-001" --json

# Using slugged key
shark task start E07-F01-001-implement-jwt-validation --json
```

---

## `shark task complete`

Mark task as ready for review (transition from `in_progress` to `ready_for_review`).

**Usage:**
```bash
shark task complete <task-key> [--notes="..."] [--json]
```

**Examples:**

```bash
# Mark task complete (short format, recommended)
shark task complete E07-F01-001
shark task complete e07-f01-001  # Case insensitive

# Mark task complete with notes
shark task complete E07-F01-001 --notes="Implementation complete, all tests passing" --json
```

---

## `shark task approve`

Approve and mark task as completed (transition from `ready_for_review` to `completed`).

**Usage:**
```bash
shark task approve <task-key> [--notes="..."] [--json]
```

**Examples:**

```bash
# Approve task (short format, recommended)
shark task approve E07-F01-001
shark task approve e07-f01-001  # Case insensitive

# Approve task with notes
shark task approve E07-F01-001 --notes="LGTM, approved" --json
```

---

## `shark task reopen`

Reopen task for rework (transition from `ready_for_review` to `in_progress`).

**Usage:**
```bash
shark task reopen <task-key> [--rejection-reason="..."] [--reason-doc="..."] [--notes="..."] [--force] [--json]
```

**Flags:**
- `--rejection-reason <string>`: Required for backward transitions. Explanation of why task is being sent back (e.g., "Missing error handling on line 67")
- `--reason-doc <path>`: Optional path to detailed document explaining rejection (e.g., code review file or bug report)
- `--notes <string>`: General notes about the task (different from rejection reason)
- `--force`: Bypass rejection reason requirement (not recommended - impairs feedback quality)
- `--json`: Output in JSON format

**Backward Transition Detection:**
When reopening a task from `ready_for_review`, the system automatically detects this as a backward transition in the workflow. A `--rejection-reason` flag is required to explain the rejection. This helps developers understand what needs to be fixed.

**Examples:**

```bash
# Reopen task with required rejection reason (short format, recommended)
shark task reopen E07-F01-001 --rejection-reason="Missing error handling for database.Query() on line 67"
shark task reopen e07-f01-001 --rejection-reason="Missing error handling for database.Query() on line 67"  # Case insensitive

# Reopen with detailed rejection reason
shark task reopen E07-F01-001 --rejection-reason="Tests fail on empty user input. Add input validation before processing."

# Reopen with rejection reason and linked document
shark task reopen E07-F01-001 \
  --rejection-reason="Found 3 critical issues. See code review document for details." \
  --reason-doc="docs/reviews/E07-F01-001-code-review.md"

# Reopen with both rejection reason and general notes
shark task reopen E07-F01-001 \
  --rejection-reason="Missing null check in error path" \
  --notes="Developer acknowledged, fixing now" \
  --json

# Force bypass rejection reason (not recommended)
shark task reopen E07-F01-001 --force
```

**About Rejection Reasons:**
- **Required for backward transitions**: The system enforces providing a reason when sending tasks backward in workflow (e.g., from review back to development)
- **Stored as rejection history**: Reasons are stored in the database and accessible via `shark task get`
- **Visible in task details**: Developers see rejection history when retrieving task with `shark task get <task-key>`
- **Prevents repeat mistakes**: Clear rejection reasons help developers fix issues on first attempt

---

## `shark task block`

Block a task (transition to `blocked` status).

**Usage:**
```bash
shark task block <task-key> --reason="..." [--json]
```

**Examples:**

```bash
# Block task with reason (short format, recommended)
shark task block E07-F01-001 --reason="Waiting for API design approval"
shark task block e07-f01-001 --reason="Waiting for API design approval"  # Case insensitive

# Block task with JSON output
shark task block E07-F01-001 --reason="Blocked by external dependency" --json
```

---

## `shark task unblock`

Unblock a task (transition from `blocked` to `todo`).

**Usage:**
```bash
shark task unblock <task-key> [--json]
```

**Examples:**

```bash
# Unblock task (short format, recommended)
shark task unblock E07-F01-001
shark task unblock e07-f01-001  # Case insensitive

# Unblock task with JSON output
shark task unblock E07-F01-001 --json
```

---

## `shark task next-status`

Transition a task to the next valid status in the workflow.

**Usage:**
```bash
shark task next-status <task-key> [--status=<status>] [--json]
```

**Flags:**
- `--status <status>`: Explicitly specify target status (skips selection)
- `--json`: Output in JSON format

**Behavior:**

When multiple valid transitions are available, behavior depends on the `interactive_mode` configuration:

**Non-Interactive Mode (Default):**
- Automatically selects the first valid transition from workflow configuration
- Prints info message showing which status was selected
- Example: `ℹ Auto-selected next status: in_qa (from 2 options)`

**Interactive Mode (Opt-In):**
- Displays interactive prompt with numbered options
- Waits for user input (1-N or Ctrl+C to cancel)
- Requires `interactive_mode: true` in `.sharkconfig.json`

**When Only One Transition Available:**
- Always auto-selects the single option (both modes)
- No prompt or selection message needed

**Examples:**

```bash
# Non-interactive mode (default) - auto-selects first transition
shark task next-status E07-F23-006
# Output:
# ℹ Auto-selected next status: in_qa (from 2 options)
# ✅ Task T-E07-F23-006 transitioned: ready_for_qa → in_qa

# Interactive mode (when enabled in config)
# Requires: { "interactive_mode": true } in .sharkconfig.json
shark task next-status E07-F23-006
# Output:
# Task: T-E07-F23-006
# Current status: ready_for_qa
#
# Available transitions:
#   1) in_qa
#   2) on_hold
#
# Enter selection [1-2]: 1
# ✅ Task T-E07-F23-006 transitioned: ready_for_qa → in_qa

# Explicit status (skips selection in both modes)
shark task next-status E07-F23-006 --status=in_qa
# ✅ Task T-E07-F23-006 transitioned: ready_for_qa → in_qa

# JSON output
shark task next-status E07-F23-006 --json
# Returns available transitions if multiple options

# Case insensitive
shark task next-status e07-f23-006
shark task next-status T-E07-F23-006  # Traditional format also works
```

**Auto-Selection Logic:**

The first transition in the workflow configuration is selected:

```json
{
  "status_flow": {
    "ready_for_qa": ["in_qa", "on_hold"]
    //               ^^^^^^^^ <- This is auto-selected (non-interactive mode)
  }
}
```

**Configuration Impact:**

| Config Setting | Multiple Transitions | Single Transition |
|----------------|---------------------|-------------------|
| `interactive_mode: false` (default) | Auto-selects first option | Auto-selects only option |
| `interactive_mode: true` | Shows interactive prompt | Auto-selects only option |
| `--status` flag provided | Uses specified status | Uses specified status |

**Use Cases:**

- **Agent/Automation Workflows:** Use default non-interactive mode
- **CI/CD Pipelines:** Use default non-interactive mode
- **Human Manual Operations:** Enable interactive mode in config
- **Explicit Control:** Use `--status` flag to specify exact transition

**Related Configuration:**
- See [Interactive Mode Configuration](interactive-mode.md) for details on `interactive_mode` setting
- See [Workflow Configuration](workflow-config.md) for status flow definitions

---

## Agent Type Flexibility

Shark supports flexible agent type assignment to accommodate diverse team structures and multi-agent workflows. Any non-empty string can be used as an agent type.

### Recommended Agent Types

These types have specific templates available:
- `frontend` - Frontend development and UI implementation
- `backend` - Backend development and API implementation
- `api` - API design and integration
- `testing` - Test development and quality assurance
- `devops` - DevOps and infrastructure
- `general` - General purpose tasks

### Custom Agent Types

Any custom string can be used as an agent type for specialized roles or multi-agent workflows. Custom types automatically use the `general` template:
- `architect` - System design and architecture decisions
- `business-analyst` - Requirements elaboration and user stories
- `qa` - Test planning and quality assurance
- `tech-lead` - Technical coordination and code review
- `product-manager` - Feature planning and prioritization
- `ux-designer` - UI/UX design and prototyping
- `data-engineer` - Data pipeline and ETL tasks
- `ml-specialist` - Machine learning model development
- `security-auditor` - Security review and compliance

### Template Fallback Behavior

- **Standard agent types** use role-specific templates (if available)
- **Custom agent types** automatically fall back to the `general` template
- All agent types are stored exactly as entered (no normalization)
- Custom agent types work seamlessly with filtering and task assignment

### Examples

```bash
# Standard agent type (uses backend template)
shark task create E07 F01 "Build API endpoint" --agent=backend

# Custom agent type (uses general template)
shark task create E07 F01 "Design system architecture" --agent=architect
shark task create E07 F01 "Write user stories" --agent=business-analyst
shark task create E07 F01 "Create test plan" --agent=qa

# Query by custom agent type
shark task list --agent=architect
shark task next --agent=business-analyst --json
```

### Multi-Agent Workflows

Custom agent types enable coordination of multiple AI agents or team members:
- Assign specific agent types to each role
- Use `--agent=<type>` to query work for specific agents
- Filter task lists by agent type for role-based views
- Each agent can retrieve their next task: `shark task next --agent=<type> --json`

---

## Related Documentation

- [Feature Commands](feature-commands.md)
- [Rejection Reasons](rejection-reasons.md) - Document rejection feedback
- [Orchestrator Actions](orchestrator-actions.md) - API response format
- [JSON API Fields](json-api-fields.md) - Enhanced JSON response fields
- [Key Formats](key-formats.md) - Case insensitive and slugged keys
- [Interactive Mode](interactive-mode.md) - Configure interactive prompts
- [Workflow Configuration](workflow-config.md) - Status flows and phases
