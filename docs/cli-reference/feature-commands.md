# Feature Commands

Complete reference for feature management commands in Shark Task Manager.

## Overview

Features are mid-level organizational units in Shark's hierarchy: **Epic > Feature > Task**. They represent user-facing capabilities or technical improvements that contain one or more tasks.

Features follow a complexity-adaptive workflow. Depending on the configured workflow profile, features may pass through simple linear statuses (basic profile) or a multi-phase lifecycle including scope validation, triage, refinement, test planning, task generation, and execution (advanced profile).

## Quick Reference

| Subcommand | Category | Description |
|------------|----------|-------------|
| `create` | CRUD | Create a new feature in an epic |
| `get` | CRUD | Get detailed feature information |
| `list` | CRUD | List features with optional filtering |
| `update` | CRUD | Update feature properties |
| `delete` | CRUD | Delete a feature (cascade deletes tasks) |
| `complete` | Lifecycle | Complete all tasks in a feature |
| `next-status` | Lifecycle | Advance feature to next workflow status |
| `set-status` | Lifecycle | Set feature to a specific workflow status |
| `context` | Context & Notes | Manage structured context data (set/get/clear) |
| `note` | Context & Notes | Add typed notes to a feature |
| `notes` | Context & Notes | List notes for a feature |
| `criteria` | Context & Notes | Show aggregated acceptance criteria |
| `resume` | Context & Notes | Get comprehensive context for resuming work |
| `tag add` | Tags | Attach a registered tag to a feature |
| `tag rm` | Tags | Detach a registered tag from a feature |

## Global Flags

All feature subcommands support these global flags:

| Flag | Description |
|------|-------------|
| `--json` | Output in JSON format (machine-readable) |
| `--field <name>` | Extract a single field from JSON output |
| `--db <path>` | Database file path (default: `shark-tasks.db`) |
| `--config <path>` | Config file path (default: `.sharkconfig.json`) |
| `--no-color` | Disable colored output |
| `-v, --verbose` | Enable verbose/debug output |

## Key Format Support

All feature commands accept keys in multiple formats (case-insensitive):

- Full key: `E07-F01`
- Short key: `F01`
- Slugged key: `E07-F01-authentication` or `F01-authentication`
- Case variants: `e07-f01`, `E07-f01`

---

## CRUD Commands

### `shark feature create`

Create a new feature with auto-assigned key and folder structure.

**Usage:**

```bash
# Positional syntax (recommended)
shark feature create <epic-key> "<title>" [flags]

# Flag syntax (legacy, still supported)
shark feature create --epic=<epic-key> "<title>" [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--epic <key>` | Epic key (alternative to positional argument) |
| `--description <text>` | Feature description |
| `--file <path>` | Custom file path (relative to root, must end in `.md`) |
| `--force` | Force reassignment if file already claimed |
| `--key <key>` | Custom key for the feature |
| `--order <n>` | Execution order within epic (lower runs first) |
| `--status <status>` | Initial status (default: `draft`) |
| `--size <value>` | Size estimate: `XS`, `S`, `M`, `L`, `XL`, `XXL` or numeric `1`, `2`, `3`, `5`, `8`, `13`. Optional; absence stores NULL. |
| `--tag <name>` | Tag to apply (repeatable). Tag must be registered; see `shark tags list`. |

**Examples:**

```bash
# Create feature with positional syntax
shark feature create E07 "JWT Token Management"

# Create with custom file path
shark feature create E07 "User Profiles" --file="docs/features/profiles/feature.md"

# Create with execution order and description
shark feature create E07 "OAuth Integration" --order=2 --description="Add OAuth 2.0 support"

# Create with size (label or numeric form)
shark feature create E07 "API Gateway" --size=M
shark feature create E07 "API Gateway" --size=3

# Create with one or more tags (tags must already be registered)
shark feature create E07 "OAuth Integration" --tag=auth --tag=voice
```

**JSON Output (example with `--size=M`):**

```bash
shark feature create E07 "JWT Token Management" --size=M --json
```

The `create` command returns a **planning-mode** response — a shell for you to fill in — not the stored feature record. `size` and `size_label` are **not** present in create output; use `shark feature get <key> --json` (or `--field size_label`) to retrieve them after creation.

```json
{
  "entity_type": "feature",
  "key": "E07-F01",
  "title": "JWT Token Management",
  "status": "created",
  "file_path": "docs/plan/E07-user-authentication-system/E07-F01-jwt-token-management/feature.md",
  "file_state": "placeholder",
  "requires_editing": true,
  "required_actions": ["edit feature.md to add description and acceptance criteria"],
  "next_commands": ["shark feature get E07-F01", "shark feature next-status E07-F01"]
}
```


---

### `shark feature get`

Display detailed information about a specific feature including progress, work breakdown, action items, and task status.

**Usage:**

```bash
shark feature get <feature-key> [flags]
```

**Examples:**

```bash
# Get feature details
shark feature get E07-F01

# Get by short key
shark feature get F01

# JSON output
shark feature get E07-F01 --json
```

**Output includes:**

- Feature metadata (title, status, progress, file path)
- Progress breakdown (weighted and completion percentages)
- Work summary (tasks by responsibility: agent, human, QA, blocked)
- Action items (tasks in actionable statuses grouped by status)
- Task status breakdown ordered by workflow phase

**Table Output Example:**

```
Feature: E07-F01 - JWT Token Management
Status: active
Progress: 80.0% (weighted) | 73.3% (completion)
Total Tasks: 15

Work Breakdown
  Agent Work:     8 tasks
  Human Work:     2 tasks
  QA Work:        1 task
  Blocked:        0 tasks

Action Items (3 tasks require attention)
  ready_for_qa (1):
    T-E07-F01-012 - Token refresh endpoint tests
  ready_for_approval (2):
    T-E07-F01-013 - Token validation middleware
    T-E07-F01-014 - JWT secret rotation
```

**JSON Output:**

```json
{
  "feature": {
    "id": 201,
    "key": "E07-F01",
    "slug": "jwt-token-management",
    "title": "JWT Token Management",
    "epic_id": 7,
    "epic_key": "E07",
    "status": "active"
  },
  "progress": {
    "weighted_pct": 80.0,
    "completion_pct": 73.3,
    "total_tasks": 15
  },
  "work_breakdown": {
    "agent": 8,
    "human": 2,
    "qa_team": 1,
    "none": 1,
    "blocked": 0
  },
  "action_items": {
    "ready_for_qa": [{"key": "T-E07-F01-012", "title": "Token refresh endpoint tests"}],
    "ready_for_approval": [{"key": "T-E07-F01-013", "title": "Token validation middleware"}]
  },
  "health": "healthy",
  "status_breakdown": [
    {"status": "completed", "count": 11, "phase": "done", "color": "green"},
    {"status": "ready_for_approval", "count": 2, "phase": "review", "color": "magenta"}
  ]
}
```

---

### `shark feature list`

List features with optional filtering by epic, status, or sort order.

**Usage:**

```bash
# Positional syntax (recommended)
shark feature list [EPIC] [flags]

# Flag syntax (legacy)
shark feature list [--epic=<epic-key>] [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-e, --epic <key>` | Filter by epic key |
| `--status <status>` | Filter by status |
| `--all` | Show all features including completed |
| `--sort-by <field>` | Sort by: `key`, `progress`, `status` |

**Examples:**

```bash
# List all non-completed features
shark feature list

# List features in epic E07
shark feature list E07

# List all features including completed, sorted by progress
shark feature list --all --sort-by=progress

# JSON output filtered by epic
shark feature list E07 --json
```

**Health Indicators:**

The table output includes health indicators:

| Indicator | Meaning |
|-----------|---------|
| Green | Healthy - no blockers, progressing normally |
| Yellow | Warning - minor issues, some blocked or stalled tasks |
| Red | Critical - multiple blockers, approval tasks overdue |

**Progress Format:** Shows dual progress as `weighted% | completion%`. Weighted progress accounts for configured `progress_weight` per status. Completion progress is the raw ratio of completed tasks.

**Notes Column:** Shows count of tasks in actionable statuses (`ready_for_*`).

**JSON Output:**

```json
{
  "id": 1,
  "key": "E07-F01",
  "title": "Authentication",
  "epic_key": "E07",
  "progress": 70.5,
  "weighted_progress": 70.5,
  "completion_progress": 50.0,
  "health_status": "warning",
  "action_items_count": 4,
  "blocked_count": 0
}
```

---

### `shark feature update`

Update a feature's properties such as title, description, status, or execution order.

**Usage:**

```bash
shark feature update <feature-key> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--title <text>` | New title |
| `--description <text>` | New description |
| `--status <status>` | New status |
| `--order <n>` | New execution order (`-1` = no change) |
| `--parallel` | Set `--order` without renumbering siblings (preserve duplicate-order parallel groups). Pairs with `--order`; without `--order`, has no effect. |
| `--key <key>` | New key (must be unique, no spaces) |
| `--file <path>` | New file path |
| `--force` | Force reassignment if file already claimed |
| `--size <value>` | New size: `XS`\|`S`\|`M`\|`L`\|`XL`\|`XXL` or `1`\|`2`\|`3`\|`5`\|`8`\|`13`. Use `clear` to remove the size (set to NULL). Flag absent = no change. |
| `--tag <name>` | Tag to apply additively (repeatable). Empty = no change; use `shark feature tag rm` to detach. |

**Examples:**

```bash
# Update title
shark feature update E07-F01 --title="JWT Token Generation and Validation"

# Update execution order (renumbers siblings to keep them sequential)
shark feature update E07-F01 --order=1

# Set order WITHOUT renumbering siblings — for forming parallel-work groups
# (multiple features at the same order run as a parallel batch)
shark feature update E07-F02 --order=1 --parallel

# Set or update size
shark feature update E07-F01 --size=S
shark feature update E07-F01 --size=2

# Clear size (set back to NULL)
shark feature update E07-F01 --size=clear

# Update multiple fields
shark feature update E07-F01 --status=active --description="Updated scope" --json

# Additive tagging (does not detach existing tags)
shark feature update E07-F01 --tag=auth
shark feature update E07-F01 --tag=auth --tag=voice
```

> `--tag` on update is **additive only**. Use `shark feature tag rm E07-F01 <name>` to detach a single tag.

---

### `shark feature delete`

Delete a feature from the database. This CASCADE deletes all tasks belonging to the feature.

**WARNING:** This action cannot be undone.

**Usage:**

```bash
shark feature delete <feature-key> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--force` | Force deletion even if feature has tasks |

**Examples:**

```bash
# Delete feature with no tasks
shark feature delete E04-F02

# Force delete feature and all its tasks
shark feature delete E04-F02 --force

# JSON output
shark feature delete E04-F02 --force --json
```

---

## Lifecycle Commands

### `shark feature complete`

Mark all tasks in a feature as completed.

Without `--force`, the command fails if any tasks are not already completed. With `--force`, all tasks are completed regardless of their current status.

**Usage:**

```bash
shark feature complete <feature-key> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--force` | Force completion of all tasks regardless of status |

**Examples:**

```bash
# Complete feature (fails if incomplete tasks exist)
shark feature complete E04-F02

# Force complete all tasks
shark feature complete E04-F02 --force

# JSON output
shark feature complete E04-F02 --json
```

---

### `shark feature next-status`

Progress a feature through its configured workflow by selecting from available transitions.

When a feature has multiple valid next statuses, this command auto-selects the first one. For automation and scripting, use `--status` to specify the target directly. Use `--preview` to inspect available transitions without making changes.

**Usage:**

```bash
shark feature next-status <feature-key> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--status <name>` | Transition directly to this status (non-interactive) |
| `--preview` | Show available transitions without making changes |
| `--force` | Bypass workflow validation (administrative override) |
| `--reason <text>` | Reason for backward or forced transitions |
| `--agent <name>` | Agent or user performing the transition |

**Examples:**

```bash
# Auto-advance to next status
shark feature next-status E07-F01

# Preview available transitions
shark feature next-status E07-F01 --preview

# Direct transition to specific status
shark feature next-status E07-F01 --status=active

# Backward transition with reason
shark feature next-status E07-F01 --status=draft --reason="Scope change detected"
```

**JSON Output:**

```json
{
  "entity_type": "feature",
  "entity_key": "E07-F01",
  "from_status": "draft",
  "to_status": "ready_for_scope_validation",
  "transitioned": true,
  "is_backward": false,
  "child_count": 15
}
```

---

### `shark feature set-status`

Set a feature to a specific workflow status with validation and backward transition guards.

Backward transitions (moving to an earlier workflow phase) require `--reason`. Using `--force` bypasses workflow validation entirely but also requires `--reason` to document the override. Forward transitions do not require `--reason`.

**Usage:**

```bash
shark feature set-status <feature-key> <status> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--reason <text>` | Reason for backward or forced transitions (required for backward) |
| `--force` | Bypass workflow validation (requires `--reason`) |
| `--agent <name>` | Agent or user performing the transition |

**Examples:**

```bash
# Forward transition
shark feature set-status E07-F01 active

# Backward transition (requires reason)
shark feature set-status E07-F01 draft --reason="Requirements changed"

# Force transition (bypasses validation, requires reason)
shark feature set-status E07-F01 custom --force --reason="Administrative override"
```

**JSON Output:**

```json
{
  "entity_type": "feature",
  "entity_key": "E07-F01",
  "from_status": "draft",
  "to_status": "active",
  "transitioned": true,
  "is_backward": false,
  "child_count": 15
}
```

---

## Context & Notes Commands

### `shark feature context`

Manage structured resume context data for features. Context data tracks progress, implementation decisions, open questions, blockers, acceptance criteria status, and related tasks.

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `set` | Set or update a context field |
| `get` | Display current context data |
| `clear` | Remove all context data |

#### `shark feature context set`

Set or update a specific field in feature context data.

**Usage:**

```bash
shark feature context set <feature-key> --field <field-name> --value <value>
```

**Supported Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `current_step` | String | Current work step description |
| `completed_steps` | JSON array | List of completed steps |
| `remaining_steps` | JSON array | List of remaining steps |
| `implementation_decisions` | JSON object | Key-value pairs of decisions |
| `open_questions` | JSON array | Unanswered questions |
| `blockers` | JSON array | Blocker objects |
| `acceptance_criteria_status` | JSON array | Criterion status objects |
| `related_tasks` | JSON array | Related task keys |

**Flags:**

| Flag | Description |
|------|-------------|
| `--field <name>` | Context field to update (required) |
| `--value <value>` | Field value (required) |

**Examples:**

```bash
# Set current step
shark feature context set E07-F01 --field current_step --value "Implementing auth endpoint"

# Set completed steps (JSON array)
shark feature context set E07-F01 --field completed_steps --value '["Design","DB Schema","API Contract"]'

# Set open questions
shark feature context set E07-F01 --field open_questions --value '["Should we support refresh tokens?"]'
```

#### `shark feature context get`

Display the current context data for a feature.

**Usage:**

```bash
shark feature context get <feature-key>
```

**Examples:**

```bash
# View context data
shark feature context get E07-F01

# JSON output
shark feature context get E07-F01 --json
```

#### `shark feature context clear`

Remove all context data from a feature.

**Usage:**

```bash
shark feature context clear <feature-key>
```

**Examples:**

```bash
# Clear all context
shark feature context clear E07-F01

# Clear with JSON confirmation
shark feature context clear E07-F01 --json
```

---

### `shark feature note`

Parent command for managing typed notes on features. Notes provide structured documentation for decisions, blockers, solutions, and other context.

#### `shark feature note add`

Add a typed note to a feature for context, decisions, and documentation.

**Usage:**

```bash
shark feature note add <feature-key> --type <type> "<content>" [flags]
```

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
| `future` | Future improvements, TODO items |
| `question` | Unanswered questions |

**Flags:**

| Flag | Description |
|------|-------------|
| `-t, --type <type>` | Note type (required) |
| `-c, --created-by <name>` | Creator name (optional) |

**Examples:**

```bash
# Add a decision note
shark feature note add E07-F01 --type decision "Using JWT for stateless auth"

# Add a blocker note with creator
shark feature note add E07-F01 --type blocker "API spec incomplete" --created-by architect

# Add a reference note
shark feature note add E07-F01 --type reference "https://jwt.io/introduction"
```

---

### `shark feature notes`

List all notes for a feature, optionally filtered by type.

**Usage:**

```bash
shark feature notes <feature-key> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-t, --type <types>` | Filter by note type (comma-separated for multiple) |

**Examples:**

```bash
# List all notes
shark feature notes E07-F01

# List decision notes only
shark feature notes E07-F01 --type decision

# List multiple types
shark feature notes E07-F01 --type decision,solution

# JSON output
shark feature notes E07-F01 --json
```

---

### `shark feature criteria`

Show aggregated acceptance criteria across all tasks in a feature.

Displays total criteria count, breakdown by status (pending, in_progress, complete, failed, na), overall completion percentage, and optional per-task breakdown.

**Usage:**

```bash
shark feature criteria <feature-key> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-t, --by-task` | Show per-task breakdown |

**Examples:**

```bash
# Show aggregated criteria
shark feature criteria E10-F04

# Show per-task breakdown
shark feature criteria E10-F04 --by-task

# JSON output
shark feature criteria E10-F04 --json
```

---

### `shark feature resume`

Get all context needed to resume work on a feature in a single command. This is the recommended entry point when returning to a feature after a break.

**Includes:**
- Feature details (title, description, status, progress)
- Context data (current step, decisions, questions, blockers)
- Feature notes (chronologically ordered)
- Task summaries (with status and priority)
- Task rollup (aggregate counts by status)

**Usage:**

```bash
shark feature resume <feature-key> [flags]
```

**Examples:**

```bash
# Resume work on a feature
shark feature resume E07-F01

# JSON output for programmatic consumption
shark feature resume E07-F01 --json
```

---

## Tag Commands

Retroactively attach or detach a registered tag on an existing feature. See [Tags → Retroactive Tagging](tags.md#retroactive-tagging-shark-entity-tag-addrm) for the cross-entity overview.

### `shark feature tag add`

Attach a registered tag to a feature. Re-running with the same arguments is a no-op (idempotent).

**Usage:**
```bash
shark feature tag add <feature-key> <name> [--json]
```

**Examples:**
```bash
shark feature tag add E07-F01 auth
shark feature tag add E07-F01 auth --json   # idempotent
```

**Exit codes:**
| Process exit | Internal class | Condition |
|------|----------------|-----------|
| 0 | — | Tag attached (or already attached) |
| 1 | `not_found` | Feature key not found |
| 2 | `db_error` | Database error |
| 3 | `unregistered_tag` | Tag name not in vocabulary |
| 3 | `validation` | Name validation error |

On an unregistered tag (process exit **3**, internal class `unregistered_tag`), stderr renders the SC-2 vocabulary snippet + `To add it:` remediation line. See [Tags → Unregistered Tag Errors](tags.md#unregistered-tag-errors).

### `shark feature tag rm`

Detach a registered tag from a feature. Removing an unattached tag is a no-op as long as the tag name is in the vocabulary.

**Usage:**
```bash
shark feature tag rm <feature-key> <name> [--json]
```

**Examples:**
```bash
shark feature tag rm E07-F01 auth
shark feature tag rm E07-F01 auth   # idempotent no-op
```

**Exit codes:**
| Process exit | Internal class | Condition |
|------|----------------|-----------|
| 0 | — | Tag detached (or was not attached) |
| 1 | `not_found` | Feature key not found, or tag name not in vocabulary |
| 2 | `db_error` | Database error |
| 3 | `validation` | Name validation error |

---

## Workflow Guidance

### Complexity-Adaptive Workflow

In the advanced workflow profile, features follow different paths based on their complexity tier:

**SIMPLE tier (complexity score 0-3):**
```
draft > scope_validation > triage > task_generation > ready_to_build > active > completed
```

**STANDARD tier (score 4-7):**
```
draft > scope_validation > triage > refinement_ba > refinement_tech >
test_planning > task_generation > ready_to_build > active > completed
```

**COMPLEX tier (score 8+):**
```
draft > scope_validation > triage > research > refinement_ba > refinement_tech >
test_planning > task_generation > ready_to_build > active > completed
```

### Complexity Scoring

Features are scored across 6 dimensions (0-3 each, max 18):

1. **File Impact**: 1-3 files=0, 4-10=2, 10+=3
2. **Pattern Novelty**: existing=0, adapt=1, minor new=2, major new=3
3. **Data Model**: no change=0, modify tables=2, new tables=3
4. **API Surface**: no change=0, modify endpoints=2, new endpoints=3
5. **Cross-Feature Deps**: isolated=0, 1-2 deps=1, 3+ deps=2, circular=3
6. **UI Complexity**: modify=0, new simple=1, new complex=2, interactive=3

### Agent Routing (Advanced Profile)

Features spawn different agents based on their current status:

| Status | Agent | Responsibility |
|--------|-------|----------------|
| `ready_for_scope_validation` | business-analyst | Validate feature scope |
| `ready_for_triage` | researcher | Complexity assessment |
| `ready_for_research` | researcher | Codebase research |
| `ready_for_refinement_ba` | business-analyst | Write PRD |
| `ready_for_refinement_tech` | architect | Design architecture |
| `ready_for_test_planning` | qa | Create test strategy |
| `ready_for_task_generation` | product-manager | Generate tasks |
| `ready_to_build` | tech-director | Autonomous build |

### Progress Calculation

**Planning statuses:** Progress equals the configured `progress_weight` for the current status.

**Active status:** Progress is aggregated from child tasks:
- **Weighted**: `sum(task_weight * task_count) / total_tasks`
- **Completion**: `completed_tasks / total_tasks`

### Health Status

| Health | Criteria |
|--------|----------|
| Healthy | No blocked tasks, all approval tasks < 3 days old |
| Warning | 1-2 blocked tasks, or approval tasks 3-7 days old |
| Critical | 3+ blocked tasks, or approval tasks > 7 days old, or no progress in 7+ days |

### Common Workflows

**Create and validate a feature:**

```bash
# 1. Create the feature
shark feature create E07 "JWT Token Management"

# 2. Advance to scope validation
shark feature next-status E07-F01

# 3. Check status after agent validates
shark feature get E07-F01
```

**Monitor feature progress:**

```bash
# View detailed progress and action items
shark feature get E07-F01

# Check acceptance criteria completion
shark feature criteria E07-F01

# List all features in an epic
shark feature list E07
```

**Resume work on a feature:**

```bash
# Get full context
shark feature resume E07-F01

# Check recent notes and decisions
shark feature notes E07-F01 --type decision

# Update progress context
shark feature context set E07-F01 --field current_step --value "Implementing refresh tokens"
```

---

## JSON Output Notes

All feature commands support `--json` for machine-readable output. Key considerations:

- Use `--json` when integrating with scripts or AI agent workflows.
- Use `--field <name>` to extract a single field from JSON output (e.g., `--field status`).
- List commands return JSON arrays; get/resume commands return JSON objects.
- The `complete`, `next-status`, and `set-status` commands return transition result objects.
- Error responses in JSON mode include structured error information.

---

## Related Documentation

- [Epic Commands](epic-commands.md)
- [Task Commands](task-commands.md)
- [Key Formats](key-formats.md) - Case insensitive and slugged keys
- [File Paths](file-paths.md) - Custom file path organization
- [JSON API Fields](json-api-fields.md) - Enhanced JSON response fields
- [Workflow Profiles](../guides/workflow-profiles.md) - Basic and advanced workflow profiles
