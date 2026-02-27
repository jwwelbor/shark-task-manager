# Epic Commands

Complete reference for managing epics in Shark Task Manager.

## Overview

Epics are the top-level organizational units in Shark's three-tier hierarchy (Epic > Feature > Task). They represent major product initiatives with multi-phase workflows from ideation through decomposition and active development.

**Epic Lifecycle (Advanced Profile):**
```
draft -> refinement -> research -> feasibility -> test planning -> decomposition -> active -> completed
```

**Epic Lifecycle (Basic Profile):**
```
draft -> active -> completed
```

All epic commands support case-insensitive keys (`E07`, `e07`), slugged keys (`E07-user-management`), and the `--json` flag for machine-readable output.

## Quick Reference

| Command | Description |
|---------|-------------|
| **CRUD** | |
| `shark epic create` | Create a new epic |
| `shark epic get` | Get epic details with features and rollups |
| `shark epic list` | List all epics with progress |
| `shark epic delete` | Delete an epic (cascade) |
| `shark epic update` | Update epic properties |
| **Lifecycle** | |
| `shark epic complete` | Complete all tasks in an epic |
| `shark epic next-status` | Advance epic to next workflow status |
| `shark epic set-status` | Set epic to a specific status |
| `shark epic status` | Show epic status summary dashboard |
| **Context and Notes** | |
| `shark epic context` | Manage structured context data |
| `shark epic note` | Add typed notes to an epic |
| `shark epic notes` | List notes for an epic |
| `shark epic resume` | Get full context for resuming work |

## Global Flags

All epic commands inherit these global flags:

| Flag | Description |
|------|-------------|
| `--json` | Output in JSON format (machine-readable) |
| `--field <name>` | Extract a single field from JSON output |
| `--config <path>` | Config file path (default: `.sharkconfig.json`) |
| `--db <path>` | Database file path (default: `shark-tasks.db`) |
| `--no-color` | Disable colored output |
| `-v, --verbose` | Enable verbose/debug output |

---

## CRUD Commands

### `shark epic create`

Create a new epic with auto-assigned key, folder structure, and database entry.

**Usage:**
```bash
shark epic create <title> [flags]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `<title>` | Epic title (required, positional) |

**Flags:**

| Flag | Description |
|------|-------------|
| `--key <string>` | Custom key (e.g., `E00`, `bugs`). Defaults to auto-generated next E## |
| `--description <string>` | Epic description |
| `--file <path>` | Custom file path (relative to root, must end in `.md`) |
| `--force` | Force reassignment if file already claimed by another entity |
| `--priority <string>` | Priority: low, medium, high (default: medium) |
| `--business-value <string>` | Business value: low, medium, high |
| `--status <string>` | Initial status: draft, active, completed, archived (default: draft) |

**Examples:**

```bash
# Basic epic creation
shark epic create "User Authentication System"

# With custom file path
shark epic create "Q1 2025 Roadmap" --file="docs/roadmap/2025-q1/epic.md"

# With priority and business value
shark epic create "Payment Gateway" --priority=high --business-value=high --json
```

**Behavior:**

1. Generates epic key (e.g., `E07`) or uses custom `--key`
2. Creates slug from title (e.g., "User Auth" -> `user-auth`)
3. Creates directory structure: `docs/plan/E##-slug/`
4. Creates `epic.md` file with frontmatter
5. Inserts record in database
6. Sets initial status to `draft`

**Generated File Structure** (`docs/plan/E07-user-auth/epic.md`):

```markdown
---
epic_key: E07
title: User Auth
status: draft
priority: medium
business_value: medium
created_at: 2026-02-25T10:30:00Z
updated_at: 2026-02-25T10:30:00Z
---

# User Auth

[Epic description goes here]
```

---

### `shark epic get`

Display detailed information about a specific epic including features, task rollups, and impediments.

**Usage:**
```bash
shark epic get <epic-key> [flags]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `<epic-key>` | Epic key (case-insensitive, numeric or slugged) |

**Key Formats Supported:**
- Numeric: `E07`
- Slugged: `E07-user-management-system`
- Case insensitive: `e07`, `e07-user-management-system`

**Examples:**

```bash
# Get epic by numeric key
shark epic get E07

# Get epic by slugged key
shark epic get E07-user-authentication --json

# Case insensitive
shark epic get e07
```

**Table Output:**
```
Epic: E07 - User Authentication System
Status: active | Priority: high | Business Value: high
Progress: 45.0%

Features (3)
Key          Title                    Status              Progress
E07-F01      JWT Token Management     active                 80.0%
E07-F02      OAuth Integration        in_refinement_tech     20.0%
E07-F03      Session Management       draft                   0.0%

Task Status Rollup
Status                  Count
completed                  12
in_development              5
ready_for_development       8
draft                      10
Total Tasks: 35

Impediments
  T-E07-F01-005 "Setup OAuth providers" (Feature: Authentication, 2 days)
```

**JSON Output:**
```json
{
  "epic": {
    "id": 7,
    "key": "E07",
    "slug": "user-authentication-system",
    "title": "User Authentication System",
    "status": "active",
    "priority": "high",
    "business_value": "high",
    "file_path": "docs/plan/E07-user-authentication-system/epic.md"
  },
  "progress_pct": 45.0,
  "features": [
    {
      "key": "E07-F01",
      "slug": "jwt-token-management",
      "title": "JWT Token Management",
      "status": "active",
      "progress_pct": 80.0,
      "task_count": 15
    }
  ],
  "task_status_rollup": {
    "completed": 12,
    "in_development": 5,
    "ready_for_development": 8,
    "draft": 10
  },
  "total_tasks": 35,
  "impediments": [
    {
      "task_key": "T-E07-F01-005",
      "task_title": "Setup OAuth providers",
      "feature_key": "E07-F01",
      "reason": "Waiting for OAuth provider approval",
      "blocked_since": "2026-01-14T10:00:00Z",
      "age_days": 2
    }
  ]
}
```

**Rollup Details:**

- **Feature Status Rollup:** Distribution of features by status across the epic
- **Task Status Rollup:** Aggregate task counts across all features by status
- **Impediments:** All blocked tasks with blocker reason and age

---

### `shark epic list`

List all epics with progress information.

**Usage:**
```bash
shark epic list [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--status <string>` | Filter by status: draft, active, completed, archived |
| `--sort-by <string>` | Sort by: key, progress, status (default: key) |

**Examples:**

```bash
# List all epics
shark epic list

# Filter by status
shark epic list --status=active

# Sort by progress, JSON output
shark epic list --sort-by=progress --json
```

**Table Output:**
```
Key  Title                          Status    Progress  Features
E04  Task Management CLI Core       active       75.0%         7
E07  User Authentication System     active       45.0%         3
E08  Analytics Dashboard            draft         0.0%         0
```

**JSON Output:**
```json
[
  {
    "id": 4,
    "key": "E04",
    "slug": "task-management-cli-core",
    "title": "Task Management CLI Core",
    "status": "active",
    "priority": "high",
    "business_value": "high",
    "file_path": "docs/plan/E04-task-management-cli-core/epic.md",
    "progress_pct": 75.0,
    "feature_count": 7
  }
]
```

**Progress Calculation:**

- **Planning statuses** (draft through decomposition): Progress equals the `progress_weight` from status metadata
- **Active status**: Progress is aggregated from child features: `Sum(Feature Progress) / Feature Count`

---

### `shark epic delete`

Delete an epic and all its features and tasks via CASCADE.

**Usage:**
```bash
shark epic delete <epic-key> [flags]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `<epic-key>` | Epic key to delete (case-insensitive) |

**Flags:**

| Flag | Description |
|------|-------------|
| `--force` | Force deletion even if epic has features |

Without `--force`, the command fails if the epic has any features. This prevents accidental cascade deletion.

**Examples:**

```bash
# Delete epic with no features
shark epic delete E05

# Force delete epic with features (cascades to all children)
shark epic delete E05 --force

# JSON output
shark epic delete E05 --force --json
```

---

### `shark epic update`

Update an epic's properties: title, description, status, priority, file path, or business value.

**Usage:**
```bash
shark epic update <epic-key> [flags]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `<epic-key>` | Epic key to update (case-insensitive) |

**Flags:**

| Flag | Description |
|------|-------------|
| `--title <string>` | New title |
| `--description <string>` | New description |
| `--status <string>` | New status: draft, active, completed, archived |
| `--priority <string>` | New priority: low, medium, high |
| `--business-value <string>` | New business value: low, medium, high |
| `--key <string>` | New key (must be unique, no spaces) |
| `--file <path>` | New file path (e.g., `docs/custom/epic.md`) |
| `--force` | Force reassignment if file already claimed |

**Examples:**

```bash
# Update title
shark epic update E07 --title="Authentication and Authorization System"

# Update priority and business value
shark epic update E07 --priority=high --business-value=high

# Change file path
shark epic update E07 --file="docs/roadmap/auth-system.md" --json
```

Note: For workflow-aware status transitions with validation, use `shark epic set-status` or `shark epic next-status` instead of `--status` on update.

---

## Lifecycle Commands

### `shark epic complete`

Mark all tasks in an epic as completed.

**Usage:**
```bash
shark epic complete <epic-key> [flags]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `<epic-key>` | Epic key (case-insensitive, numeric or slugged) |

**Flags:**

| Flag | Description |
|------|-------------|
| `--force` | Force completion of all tasks regardless of their current status |

Without `--force`, the command fails if any tasks are in an incomplete status.

**Examples:**

```bash
# Complete epic (fails if any tasks are incomplete)
shark epic complete E07

# Force complete all tasks regardless of status
shark epic complete E07 --force

# JSON output
shark epic complete E07 --force --json
```

---

### `shark epic next-status`

Progress an epic through its configured workflow by advancing to the next valid status.

When multiple valid next statuses exist, the command auto-selects the first one. Use `--status` for explicit selection.

**Usage:**
```bash
shark epic next-status <epic-key> [flags]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `<epic-key>` | Epic key (case-insensitive) |

**Flags:**

| Flag | Description |
|------|-------------|
| `--status <name>` | Transition directly to this status (non-interactive) |
| `--preview` | Show available transitions without making changes |
| `--force` | Bypass workflow validation (administrative override, requires `--reason`) |
| `--reason <text>` | Reason for backward or forced transitions |
| `--agent <name>` | Agent or user performing the transition |

**Examples:**

```bash
# Auto-advance to next status
shark epic next-status E16

# Preview available transitions
shark epic next-status E16 --preview

# Direct transition to a specific status
shark epic next-status E16 --status=active

# Backward transition with reason
shark epic next-status E16 --status=draft --reason="Requirements changed"
```

**Behavior:**

1. Checks current status of the epic
2. Looks up valid transitions from `status_flow` in configuration
3. If `--status` provided, transitions directly to that status
4. Otherwise advances to the first valid next status
5. Records transition in history
6. Triggers status cascade if applicable

---

### `shark epic set-status`

Set an epic to a specific workflow status with validation and backward transition guards.

**Usage:**
```bash
shark epic set-status <epic-key> <status> [flags]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `<epic-key>` | Epic key (case-insensitive) |
| `<status>` | Target status name |

**Flags:**

| Flag | Description |
|------|-------------|
| `--reason <text>` | Reason for backward or forced transitions (required for backward transitions) |
| `--force` | Bypass workflow validation (administrative override, requires `--reason`) |
| `--agent <name>` | Agent or user performing the transition |

**Backward Transition Rules:**
- Moving to an earlier workflow phase (e.g., active -> draft) requires `--reason`
- Using `--force` always requires `--reason` to document the override
- Forward transitions do not require `--reason`

**Examples:**

```bash
# Forward transition
shark epic set-status E16 active

# Backward transition (requires --reason)
shark epic set-status E16 draft --reason="Requirements changed significantly"

# Force transition (bypasses validation, requires --reason)
shark epic set-status E16 custom --force --reason="Administrative override"
```

**JSON Output:**
```json
{
  "entity_type": "epic",
  "entity_key": "E16",
  "from_status": "draft",
  "to_status": "active",
  "transitioned": true,
  "is_backward": false,
  "child_count": 3
}
```

---

### `shark epic status`

Display a summary dashboard of all epics with completion percentages and task counts.

**Usage:**
```bash
shark epic status [flags]
```

**Examples:**

```bash
# Show epic status dashboard
shark epic status

# JSON output
shark epic status --json
```

---

## Context and Notes Commands

### `shark epic context`

Manage structured resume context data for epics. Context data tracks progress, decisions, questions, and blockers for coordinated work across sessions.

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `set` | Set or update a context field |
| `get` | Display current context data |
| `clear` | Remove all context data |

#### `shark epic context set`

Set or update a specific field in epic context data.

**Usage:**
```bash
shark epic context set <epic-key> --field <field> --value <value>
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--field <string>` | Context field to update (required) |
| `--value <string>` | Field value (required) |

**Supported Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `current_step` | String | Current work step description |
| `completed_steps` | JSON array | Steps already completed |
| `remaining_steps` | JSON array | Steps still to do |
| `implementation_decisions` | JSON object | Decision key-value pairs |
| `open_questions` | JSON array | Unanswered questions |
| `blockers` | JSON array | Blocker objects |
| `acceptance_criteria_status` | JSON array | Criterion status objects |
| `related_tasks` | JSON array | Related task keys |

**Examples:**

```bash
# Set current step
shark epic context set E07 --field current_step --value "Defining epic scope"

# Set completed steps
shark epic context set E07 --field completed_steps --value '["Research","Design review"]'

# Set open questions
shark epic context set E07 --field open_questions --value '["What API version?","OAuth providers?"]'
```

#### `shark epic context get`

Display the current context data for an epic.

**Usage:**
```bash
shark epic context get <epic-key>
```

**Examples:**

```bash
# Get context data
shark epic context get E07

# JSON output
shark epic context get E07 --json
```

#### `shark epic context clear`

Remove all context data from an epic.

**Usage:**
```bash
shark epic context clear <epic-key>
```

**Examples:**

```bash
# Clear all context
shark epic context clear E07
```

---

### `shark epic note`

Add typed notes to an epic for context, decisions, and documentation.

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `add` | Add a typed note to an epic |

#### `shark epic note add`

Add a typed note to an epic.

**Usage:**
```bash
shark epic note add <epic-key> --type <type> <content> [flags]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `<epic-key>` | Epic key (case-insensitive) |
| `<content>` | Note content (quoted string) |

**Flags:**

| Flag | Description |
|------|-------------|
| `-t, --type <string>` | Note type (required, see table below) |
| `-c, --created-by <string>` | Creator name (optional) |

**Note Types:**

| Type | Description |
|------|-------------|
| `comment` | General observation |
| `decision` | Why we chose X over Y |
| `blocker` | What is blocking progress |
| `solution` | How we solved a problem |
| `reference` | External links, documentation |
| `implementation` | What we actually built |
| `testing` | Test results, coverage |
| `future` | Future improvements, TODOs |
| `question` | Unanswered questions |

**Examples:**

```bash
# Add a decision note
shark epic note add E07 --type decision "Chose microservices architecture for scalability"

# Add a blocker note with creator
shark epic note add E07 --type blocker "Waiting on infrastructure team" --created-by alice

# Add a reference note
shark epic note add E07 --type reference "https://example.com/design-doc"
```

---

### `shark epic notes`

List all notes for an epic, optionally filtered by type.

**Usage:**
```bash
shark epic notes <epic-key> [flags]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `<epic-key>` | Epic key (case-insensitive) |

**Flags:**

| Flag | Description |
|------|-------------|
| `-t, --type <string>` | Filter by note type (comma-separated for multiple) |

**Examples:**

```bash
# List all notes
shark epic notes E07

# List only decision notes
shark epic notes E07 --type decision

# List decision and solution notes
shark epic notes E07 --type decision,solution

# JSON output
shark epic notes E07 --json
```

---

### `shark epic resume`

Get all context needed to resume work on an epic in a single command. This is the recommended way for agents to get up to speed on an epic.

**Usage:**
```bash
shark epic resume <epic-key> [flags]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `<epic-key>` | Epic key (case-insensitive) |

**Includes:**
- Epic details (title, description, status, priority)
- Context data (progress, decisions, questions, blockers)
- Epic notes (chronologically ordered)
- Feature summaries (with task counts and progress)
- Task rollup (aggregate counts by status)

**Examples:**

```bash
# Resume epic work
shark epic resume E07

# JSON output (recommended for agents)
shark epic resume E07 --json
```

---

## Common Workflows

### Create Epic and Begin Refinement

```bash
# 1. Create epic
shark epic create "User Authentication System" --priority=high

# 2. Add initial context
shark epic context set E07 --field current_step --value "Initial scoping"

# 3. Advance to refinement
shark epic next-status E07

# 4. Verify status
shark epic get E07
```

### Track Decisions During Development

```bash
# Record architectural decisions
shark epic note add E07 --type decision "Using JWT for stateless auth"
shark epic note add E07 --type decision "PostgreSQL for user store"

# Record blockers
shark epic note add E07 --type blocker "Waiting on SSO provider credentials"

# Review decisions later
shark epic notes E07 --type decision
```

### Monitor Epic Progress

```bash
# Check overall status with rollups
shark epic get E07

# View status dashboard across all epics
shark epic status

# Check for impediments
shark epic get E07 --json | jq '.impediments'
```

### Resume Work on an Epic

```bash
# Get full context in one command
shark epic resume E07 --json

# Update progress tracking
shark epic context set E07 --field current_step --value "Code review phase"
shark epic context set E07 --field completed_steps --value '["Research","Design","Implementation"]'
```

### Complete an Epic

```bash
# Verify all features are done
shark epic get E07

# Complete all remaining tasks
shark epic complete E07

# Or set status directly
shark epic set-status E07 completed
```

## Epic Status Flow (Advanced Profile)

**Planning Phase:**
1. `draft` - Initial capture
2. `ready_for_refinement` - Ready for PRD
3. `in_refinement` - BA writing PRD
4. `ready_for_research` - PRD complete
5. `in_research` - Research in progress

**Feasibility Phase:**
6. `ready_for_feasibility_review_ba` - Business feasibility
7. `in_feasibility_review_ba` - BA reviewing
8. `ready_for_feasibility_review_tech` - Technical feasibility
9. `in_feasibility_review_tech` - Architect reviewing
10. `intervention_required` - Human decision needed

**Execution Phase:**
11. `ready_for_test_planning` - UAT planning
12. `in_test_planning` - QA creating plan
13. `ready_for_decomposition` - Ready for feature breakdown
14. `in_decomposition` - PM generating features
15. `active` - Features exist, work in progress

**Terminal States:**
16. `completed` - All features done
17. `cancelled` - Epic abandoned
18. `on_hold` - Temporarily paused
19. `blocked` - External blocker

### Agent Routing by Status

In the advanced profile, different statuses route to different agent types:

| Status | Agent Type | Role |
|--------|-----------|------|
| `ready_for_refinement` | business-analyst | Writes PRD |
| `ready_for_research` | researcher | Conducts research |
| `ready_for_feasibility_review_ba` | business-analyst | Business feasibility |
| `ready_for_feasibility_review_tech` | architect | Technical feasibility |
| `ready_for_test_planning` | qa | Creates UAT plan |
| `ready_for_decomposition` | product-manager | Generates features |

## JSON Output Notes

All epic commands support the `--json` flag for machine-readable output. Additionally:

- Use `--field <name>` to extract a single field from JSON output (e.g., `--field status`)
- Smart dispatchers (`shark get E07`, `shark list`) also work with epics
- The `shark epic resume` command provides the most comprehensive JSON payload for agent consumption

## Related Documentation

- [Feature Commands](feature-commands.md)
- [Task Commands](task-commands.md)
- [Key Formats](key-formats.md) - Case insensitive and slugged keys
- [File Paths](file-paths.md) - Custom file path organization
- [JSON API Fields](json-api-fields.md) - Enhanced JSON response fields
- [Workflow Profiles](../guides/workflow-profiles.md) - Status flow configuration
