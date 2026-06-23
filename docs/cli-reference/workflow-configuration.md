# Workflow Configuration

Deep dive into Shark's workflow system - status flows, metadata, orchestrator actions, and lifecycle management.

> **Shark 2.x route-based schema (E35):** A consolidated per-step schema
> (`steps:` with `outcomes:`, `aliases:`, a claim/session lease, and a master
> index file) is supported alongside everything documented below. The two
> shapes coexist — the loader derives the legacy `status_flow`/`status_metadata`
> maps from `steps:`. See the
> [Route-Based Workflow Guide](../guides/route-based-workflow.md).

## Overview

Shark's workflow system enables AI-driven, multi-stage development workflows through:
- **Status flows**: Define valid transitions between statuses
- **Status metadata**: Configure color, phase, progress weight, responsibility, agent assignment
- **Orchestrator actions**: Auto-spawn agents, advance statuses, pause workflows
- **Special statuses**: Mark lifecycle boundaries (start, complete, aggregation)

Five entity types each have their own workflow configuration:

| Entity | Config Key | Key Format | Description |
|--------|-----------|------------|-------------|
| Epic | `epic_workflow` | `E07` | Top-level organizational units |
| Feature | `feature_workflow` | `E07-F01` | Feature areas within an epic |
| Task | `status_flow` / `status_metadata` | `E07-F01-001` | Atomic work items |
| Bug | `bug_workflow` | `B001` | Defect tracking (standalone) |
| Change-Card | `change_workflow` | `CC-001` | Lightweight changes (standalone) |

Bugs and change-cards are **standalone entities** — they are not nested under epics or features, though they can optionally link to them.

## Workflow Structure

```json
{
  "epic_workflow": {
    "version": "1.0",
    "status_flow": {
      "draft": ["ready_for_refinement", "cancelled"],
      "ready_for_refinement": ["in_refinement"]
    },
    "status_metadata": {
      "draft": {
        "description": "Epic captured, not yet refined",
        "phase": "planning",
        "color": "gray",
        "progress_weight": 0,
        "is_planning": true,
        "responsibility": "none",
        "orchestrator_action": { }
      }
    },
    "special_statuses": {
      "_start_": ["draft"],
      "_complete_": ["completed", "cancelled"],
      "_aggregation_": ["active"]
    }
  }
}
```

## Status Flow

Defines valid status transitions as a directed graph.

### Structure

```json
{
  "status_flow": {
    "from_status": ["to_status_1", "to_status_2"],
    "another_status": ["next_status"]
  }
}
```

### Example: Task Workflow

```json
{
  "status_flow": {
    "draft": ["ready_for_refinement_ba", "ready_for_refinement_tech", "cancelled"],
    "ready_for_refinement_ba": ["in_refinement_ba", "cancelled"],
    "in_refinement_ba": ["ready_for_refinement_tech", "ready_for_development"],
    "ready_for_development": ["in_development", "cancelled"],
    "in_development": ["ready_for_code_review", "blocked"],
    "ready_for_code_review": ["in_code_review"],
    "in_code_review": ["ready_for_qa", "in_development"],
    "ready_for_qa": ["in_qa"],
    "in_qa": ["ready_for_approval", "in_development"],
    "ready_for_approval": ["in_approval"],
    "in_approval": ["completed", "ready_for_qa"],
    "completed": [],
    "cancelled": [],
    "blocked": ["ready_for_development", "cancelled"],
    "on_hold": ["ready_for_refinement_ba", "cancelled"]
  }
}
```

### Validation Rules

1. **All statuses referenced must exist** in status_metadata
2. **No self-loops** (status cannot transition to itself)
3. **Acyclic for terminal states** (completed, cancelled cannot have outgoing edges)
4. **Reachability** (all statuses should be reachable from start statuses)

### Common Patterns

#### Linear Flow

```json
{
  "status_flow": {
    "draft": ["in_progress"],
    "in_progress": ["done"],
    "done": []
  }
}
```

#### Branching Flow

```json
{
  "status_flow": {
    "draft": ["ready_for_ba", "ready_for_tech"],  // Two paths
    "ready_for_ba": ["in_ba"],
    "ready_for_tech": ["in_tech"],
    "in_ba": ["development"],
    "in_tech": ["development"],
    "development": ["done"]
  }
}
```

#### Looping Flow (Rework)

```json
{
  "status_flow": {
    "draft": ["in_development"],
    "in_development": ["ready_for_qa"],
    "ready_for_qa": ["in_qa"],
    "in_qa": ["completed", "in_development"],  // Loop back for rework
    "completed": []
  }
}
```

#### Escape Hatches

```json
{
  "status_flow": {
    "any_status": ["blocked", "on_hold", "cancelled"],  // Escape hatches
    "blocked": ["any_status"],  // Resume from block
    "on_hold": ["any_status"],  // Resume from hold
    "cancelled": []  // Terminal
  }
}
```

## Status Metadata

Defines configuration for each status.

### Structure

```json
{
  "status_metadata": {
    "status_name": {
      "description": "Human-readable description",
      "phase": "planning | development | review | qa | approval | done | any",
      "color": "gray | yellow | green | red | blue | cyan | magenta | orange | purple | white",
      "progress_weight": 0.0,
      "is_planning": true,
      "responsibility": "none | agent | human | qa_team",
      "agent_types": ["agent-type-1", "agent-type-2"],
      "blocks_feature": false,
      "aggregates_from": "tasks | features",
      "orchestrator_action": {
        "action": "spawn_agent | advance_status | pause | archive",
        "agent_type": "agent-type",
        "skills": ["skill-1", "skill-2"],
        "instruction_template": "path/to/template.tmpl"
      }
    }
  }
}
```

### Core Fields

#### `description` (string, required)

Human-readable description of this status.

**Example:**
```json
"description": "Ready for technical architecture design"
```

#### `phase` (string, required)

Workflow phase for grouping and ordering.

**Valid values:**
- `planning` - Requirements, analysis, design
- `development` - Implementation
- `review` - Code review, technical review
- `qa` - Testing and validation
- `approval` - Final sign-off
- `done` - Terminal states (completed, cancelled)
- `any` - Special states (blocked, on_hold) that don't belong to a phase
- `research` - Discovery and investigation
- `feasibility` - Feasibility assessment
- `refinement` - Requirements refinement
- `test_planning` - Test strategy creation
- `decomposition` - Breaking down into smaller units
- `execution` - Active implementation
- `validation` - Scope validation
- `triage` - Complexity assessment
- `intervention` - Human decision required
- `paused` - Intentionally paused
- `blocked` - Blocked by dependency

**Usage:**
- Groups statuses in UI displays
- Orders workflow stages
- Drives progress calculations

#### `color` (string, required)

ANSI color for terminal display.

**Valid values:**
- `gray` - Neutral, draft, not started
- `yellow` - In progress, active work
- `green` - Success, completed, passed
- `red` - Error, failed, critical, blocked
- `blue` - In progress, active (alternative)
- `cyan` - Ready for work, awaiting attention
- `magenta` - Review, waiting for feedback
- `orange` - Warning, on hold
- `purple` - Approval, final stages
- `white` - Completed, done
- `lime` - QA, testing
- `teal` - Feasibility checks

**Example:**
```json
{
  "draft": {"color": "gray"},
  "ready_for_development": {"color": "cyan"},
  "in_development": {"color": "yellow"},
  "completed": {"color": "white"}
}
```

#### `progress_weight` (number, required)

Contribution to weighted progress calculation (0.0 - 1.0).

**Purpose:**
- Calculate weighted progress percentage
- Reflect true completion vs simple task count
- Account for backend-heavy vs frontend-heavy features

**Guidelines:**
- `0.0` - Not started (draft, todo)
- `0.1-0.3` - Planning phases
- `0.5` - Active development
- `0.7-0.9` - Review and testing
- `1.0` - Complete

**Example:**
```json
{
  "draft": {"progress_weight": 0.0},
  "ready_for_refinement": {"progress_weight": 0.05},
  "in_refinement": {"progress_weight": 0.1},
  "ready_for_development": {"progress_weight": 0.32},
  "in_development": {"progress_weight": 0.5},
  "ready_for_code_review": {"progress_weight": 0.75},
  "ready_for_qa": {"progress_weight": 0.8},
  "ready_for_approval": {"progress_weight": 0.9},
  "completed": {"progress_weight": 1.0}
}
```

**Calculation:**
```
Weighted Progress = Σ(task_progress_weight) / task_count * 100%
```

#### `responsibility` (string, required)

Who is responsible for work in this status.

**Valid values:**
- `none` - No one (draft, blocked, on_hold)
- `agent` - AI agent
- `human` - Human developer or product owner
- `qa_team` - QA team or QA agent

**Usage:**
- Work breakdown reporting
- Capacity planning
- Agent routing
- Human intervention detection

**Example:**
```json
{
  "draft": {"responsibility": "none"},
  "ready_for_development": {"responsibility": "none"},  // Awaiting pickup
  "in_development": {"responsibility": "agent"},
  "ready_for_approval": {"responsibility": "human"},
  "in_qa": {"responsibility": "qa_team"}
}
```

### Optional Fields

#### `is_planning` (boolean, optional)

Whether this status is part of planning phase (not implementation).

**Default:** `false`

**Usage:**
- Filter planning vs execution tasks
- Calculate time spent in planning
- Separate "thinking" from "doing"

**Example:**
```json
{
  "draft": {"is_planning": true},
  "in_refinement": {"is_planning": true},
  "in_development": {"is_planning": false},
  "completed": {"is_planning": false}
}
```

#### `blocks_feature` (boolean, optional)

Whether tasks in this status block feature completion.

**Default:** `false`

**Usage:**
- Calculate action items (blockers)
- Identify critical path tasks
- Drive escalation logic

**Example:**
```json
{
  "blocked": {"blocks_feature": true},
  "ready_for_approval": {"blocks_feature": true},  // Must approve to complete
  "in_development": {"blocks_feature": false}
}
```

#### `agent_types` ([]string, optional)

Agent types that can work on tasks in this status.

**Usage:**
- Agent routing for `shark task list --agent=<type>`
- Orchestrator knows which agent to spawn
- Multi-agent type support (e.g., tech-lead or code-reviewer)

**Example:**
```json
{
  "ready_for_development": {
    "agent_types": ["developer"]
  },
  "ready_for_code_review": {
    "agent_types": ["tech-lead", "code-reviewer"]
  },
  "ready_for_approval": {
    "agent_types": ["product-manager", "client"]
  }
}
```

#### `aggregates_from` (string, optional)

For aggregation statuses: where to aggregate progress from.

**Valid values:**
- `"tasks"` - Feature aggregates progress from tasks
- `"features"` - Epic aggregates progress from features

**Usage:**
- `active` status for features (aggregates from tasks)
- `active` status for epics (aggregates from features)

**Example:**
```json
{
  "feature_workflow": {
    "status_metadata": {
      "active": {
        "aggregates_from": "tasks",
        "description": "Build in progress, progress aggregated from tasks"
      }
    }
  },
  "epic_workflow": {
    "status_metadata": {
      "active": {
        "aggregates_from": "features",
        "description": "Features exist, progress aggregated from children"
      }
    }
  }
}
```

## Orchestrator Actions

Defines what happens when an entity enters a status.

### Structure

```json
{
  "orchestrator_action": {
    "action": "spawn_agent | advance_status | pause | archive",
    "agent_type": "agent-type",
    "provider": "anthropic | openai",
    "model": "opus | sonnet | haiku | codex",
    "skills": ["skill-1", "skill-2"],
    "instruction_template": "path/to/template.tmpl"
  }
}
```

### Action Types

#### `spawn_agent`

Spawn an agent to perform work.

**Required fields:**
- `agent_type` - Agent to spawn (e.g., "researcher", "developer")
- `instruction_template` - Path to template file (e.g., "task/ready_for_development.tmpl")

**Recommended fields:**
- `provider` - AI provider for dispatch (e.g., `anthropic`, `openai`). Surfaced verbatim in `shark next` output. Empty is legal — the run controller may default — but `shark admin workflow validate-actions` warns when missing on `spawn_agent` / `check_or_resume`, since downstream tooling expects it populated.
- `model` - Model identifier (e.g., `opus`, `sonnet`, `haiku`, `codex`). Should match the provider (anthropic models for `anthropic`, codex for `openai`).

**Optional fields:**
- `skills` - Skills/tools to load (e.g., ["test-driven-development", "implementation"])

**Example:**
```json
{
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "developer",
    "provider": "anthropic",
    "model": "sonnet",
    "skills": ["test-driven-development", "implementation"],
    "instruction_template": "task/ready_for_development.tmpl"
  }
}
```

**Behavior:**
1. Orchestrator detects entity in this status
2. Reads instruction_template file
3. Renders template with entity data
4. Spawns agent with rendered instructions
5. Agent performs work
6. Agent advances entity status on completion

#### `advance_status`

Automatically advance to next status.

**Required fields:**
- `instruction_template` - Instructions for orchestrator (usually inline string)

**Example:**
```json
{
  "orchestrator_action": {
    "action": "advance_status",
    "instruction_template": "Epic {id} is in draft. Execute `shark epic next-status {id}` to advance to refinement."
  }
}
```

**Behavior:**
1. Orchestrator detects entity in this status
2. Executes `shark <entity> next-status <id>`
3. Entity moves to next valid status

**Use cases:**
- Draft → ready_for_refinement (no work needed, just advance)
- Automatic progression through workflow gates

#### `pause`

Pause workflow, do not spawn agents.

**Required fields:**
- `instruction_template` - Explanation of why paused

**Example:**
```json
{
  "orchestrator_action": {
    "action": "pause",
    "instruction_template": "Epic {id} is blocked. Do not spawn agent. Wait for blocker resolution."
  }
}
```

**Behavior:**
1. Orchestrator detects entity in this status
2. Logs pause reason
3. Does NOT spawn any agents
4. Waits for manual status change

**Use cases:**
- `blocked` - External dependency
- `on_hold` - Product decision pending
- `intervention_required` - Human decision needed

#### `archive`

Mark as complete, do nothing.

**Required fields:**
- `instruction_template` - Completion message

**Example:**
```json
{
  "orchestrator_action": {
    "action": "archive",
    "instruction_template": "Task {task_id} is completed. No agent action needed."
  }
}
```

**Behavior:**
1. Orchestrator detects entity in this status
2. Logs completion
3. No further orchestration

**Use cases:**
- `completed` - Work done
- `cancelled` - Work abandoned

### Template Variables

Available in `instruction_template`:

**For Epics:**
- `{id}` - Epic key (e.g., "E07")
- `{title}` - Epic title
- `{file_path}` - Path to epic.md

**For Features:**
- `{id}` - Feature key (e.g., "E07-F01")
- `{title}` - Feature title
- `{file_path}` - Path to feature.md
- `{epic_id}` - Parent epic key

**For Tasks:**
- `{task_id}` - Task key (e.g., "T-E07-F01-001")
- `{title}` - Task title
- `{file_path}` - Path to task.md
- `{epic_id}` - Parent epic key
- `{feature_id}` - Parent feature key

## Special Statuses

Mark lifecycle boundaries and workflow gates.

### Structure

```json
{
  "special_statuses": {
    "_start_": ["draft", "todo"],
    "_complete_": ["completed", "cancelled"],
    "_aggregation_": ["active"]
  }
}
```

### `_start_`

Initial statuses when entity is created.

**Purpose:**
- Define entry point(s) to workflow
- `shark task create` defaults to first start status
- Multiple start statuses = branching entry points

**Example:**
```json
{
  "_start_": ["draft"]
}
```

**Multi-entry example:**
```json
{
  "_start_": ["draft", "ready_for_development", "todo"]
}
```

### `_complete_`

Terminal statuses (no outgoing transitions).

**Purpose:**
- Mark work as finished
- Stop orchestration
- Calculate completion percentage

**Example:**
```json
{
  "_complete_": ["completed", "cancelled"]
}
```

**Common patterns:**
- `completed` - Work done successfully
- `cancelled` - Work abandoned
- `obsolete` - No longer needed
- `duplicate` - Duplicate of another task

### `_aggregation_`

Statuses that aggregate progress from children.

**Purpose:**
- Features aggregate from tasks
- Epics aggregate from features
- Progress is calculated, not directly set

**Example:**
```json
{
  "feature_workflow": {
    "special_statuses": {
      "_aggregation_": ["active"]
    },
    "status_metadata": {
      "active": {
        "aggregates_from": "tasks"
      }
    }
  }
}
```

**Behavior:**
- `shark feature get E07-F01` calculates progress from task statuses
- Feature cannot manually set `active` status progress
- Feature is `active` when tasks are being worked

## Workflow Profiles

Pre-configured workflow templates.

### Basic Profile (5 statuses)

Simple linear workflow:

```json
{
  "status_flow": {
    "todo": ["in_progress"],
    "in_progress": ["ready_for_review", "blocked"],
    "ready_for_review": ["completed", "in_progress"],
    "completed": [],
    "blocked": ["todo", "in_progress"]
  },
  "special_statuses": {
    "_start_": ["todo"],
    "_complete_": ["completed"]
  }
}
```

**Use for:**
- Solo developers
- Simple projects
- Prototyping

### Advanced Profile (19 statuses)

Comprehensive TDD workflow:

```
draft → ready_for_refinement_ba → in_refinement_ba →
ready_for_refinement_tech → in_refinement_tech →
ready_for_development → in_development →
ready_for_code_review → in_code_review →
ready_for_qa → in_qa →
ready_for_approval → in_approval →
completed
```

**Use for:**
- Development teams
- TDD workflows
- Multi-stage reviews

**See:** `docs/guides/workflow-profiles.md` for full details.

## Example Workflows

### Minimal Task Workflow

```json
{
  "status_flow": {
    "todo": ["done"],
    "done": []
  },
  "status_metadata": {
    "todo": {
      "description": "Not started",
      "phase": "planning",
      "color": "gray",
      "progress_weight": 0,
      "responsibility": "none"
    },
    "done": {
      "description": "Complete",
      "phase": "done",
      "color": "green",
      "progress_weight": 1.0,
      "responsibility": "none"
    }
  },
  "special_statuses": {
    "_start_": ["todo"],
    "_complete_": ["done"]
  }
}
```

### Task Workflow with Agent Routing

```json
{
  "status_flow": {
    "todo": ["ready_for_development"],
    "ready_for_development": ["in_development"],
    "in_development": ["ready_for_review"],
    "ready_for_review": ["completed", "in_development"],
    "completed": []
  },
  "status_metadata": {
    "todo": {
      "description": "Not started",
      "phase": "planning",
      "color": "gray",
      "progress_weight": 0,
      "responsibility": "none"
    },
    "ready_for_development": {
      "description": "Ready for implementation",
      "phase": "development",
      "color": "cyan",
      "progress_weight": 0.2,
      "responsibility": "none",
      "agent_types": ["developer"],
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "developer",
        "skills": ["implementation", "test-driven-development"],
        "instruction_template": "task/ready_for_development.tmpl"
      }
    },
    "in_development": {
      "description": "Implementation in progress",
      "phase": "development",
      "color": "yellow",
      "progress_weight": 0.5,
      "responsibility": "agent"
    },
    "ready_for_review": {
      "description": "Awaiting code review",
      "phase": "review",
      "color": "magenta",
      "progress_weight": 0.8,
      "responsibility": "agent",
      "agent_types": ["tech-lead"],
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "tech-lead",
        "skills": ["quality"],
        "instruction_template": "task/ready_for_code_review.tmpl"
      }
    },
    "completed": {
      "description": "Work done",
      "phase": "done",
      "color": "white",
      "progress_weight": 1.0,
      "responsibility": "none",
      "orchestrator_action": {
        "action": "archive",
        "instruction_template": "Task {task_id} completed."
      }
    }
  },
  "special_statuses": {
    "_start_": ["todo"],
    "_complete_": ["completed"]
  }
}
```

## Best Practices

### Status Naming

**Use `ready_for_X` / `in_X` pattern:**
```
ready_for_development  → in_development
ready_for_code_review  → in_code_review
ready_for_qa           → in_qa
```

**Benefits:**
- Clear ownership (ready = awaiting, in = active)
- Easy to filter (all `ready_for_*` need attention)
- Natural language flow

### Progress Weights

**Distribute weights progressively:**
```json
{
  "draft": 0.0,
  "refinement": 0.1,
  "development": 0.5,
  "review": 0.75,
  "qa": 0.85,
  "approval": 0.95,
  "completed": 1.0
}
```

**Avoid:**
- Equal spacing (0, 0.33, 0.66, 1.0) - doesn't reflect real effort
- Back-heavy weights (planning=0.8, dev=0.9) - inflates early progress

### Phase Organization

**Group related statuses:**
```json
{
  "planning": ["draft", "refinement", "ready_for_dev"],
  "development": ["in_development"],
  "review": ["ready_for_review", "in_review"],
  "qa": ["ready_for_qa", "in_qa"],
  "done": ["completed"]
}
```

### Escape Hatches

**Always provide:**
```json
{
  "any_status": ["blocked", "on_hold", "cancelled"]
}
```

**Rationale:**
- External blockers happen
- Product priorities change
- Need emergency exits

## Migration Guide

### Adding New Status

1. Add to `status_flow`:
   ```json
   {
     "new_status": ["next_status"]
   }
   ```

2. Add to `status_metadata`:
   ```json
   {
     "new_status": {
       "description": "...",
       "phase": "...",
       "color": "...",
       "progress_weight": 0.0,
       "responsibility": "none"
     }
   }
   ```

3. Update inbound transitions:
   ```json
   {
     "previous_status": ["new_status", "other_status"]
   }
   ```

### Removing Status

1. Check for tasks in that status:
   ```bash
   shark task list --status=old_status
   ```

2. Migrate tasks to new status:
   ```bash
   shark task update <key> --status=new_status
   ```

3. Remove from `status_flow` and `status_metadata`

### Changing Status Flow

1. Backup config:
   ```bash
   cp .sharkconfig.json .sharkconfig.backup.json
   ```

2. Update `status_flow` in your workflow file (the file referenced by
   `workflow_config` in `.sharkconfig.json` — typically
   `shark-templates/.sharkworkflow-short.json`)

3. Validate:
   ```bash
   shark admin workflow validate
   ```

4. The new flow takes effect on the next `shark` command — no apply step
   needed, since `workflow_config` is read on every invocation.

## Bug and Change-Card Workflows

Bugs and change-cards use the same configuration structure as other entities, under their own top-level keys: `bug_workflow` and `change_workflow`.

### Key Differences from Task Workflows

| Aspect | Tasks | Bugs | Change-Cards |
|--------|-------|------|--------------|
| Config key | `status_flow` / `status_metadata` | `bug_workflow` | `change_workflow` |
| Bundled workflows | `.sharkworkflow-short.json` / `.sharkworkflow.json` (in `shark-templates/`, selected via `workflow_config`) | Custom only | Custom only |
| Severity field | — | `critical`, `high`, `medium`, `low` | — |
| Priority field | — | — | 1–10 |
| Default start status | `todo` / `draft` | `reported` | `proposed` |
| Terminal statuses | `completed`, `cancelled` | `resolved`, `wont_fix`, `duplicate` | `completed`, `declined` |

### Default Bug Workflow

```
reported → triaged → in_fix → in_verification → resolved
                  ↘ wont_fix
         ↘ duplicate
```

Agents routed by status: `business-analyst` (reported), `developer` (triaged, in_fix), `qa` (in_verification).

### Default Change-Card Workflow

```
proposed → approved → in_progress → completed
         ↘ declined
```

Agents routed by status: `business-analyst` (proposed), `developer` (approved, in_progress).

### Customizing Bug and Change-Card Workflows

Add or modify statuses in `.sharkconfig.json` under `bug_workflow` or `change_workflow`. The structure is identical to `epic_workflow`:

```json
{
  "bug_workflow": {
    "version": "1.0",
    "status_flow": { ... },
    "status_metadata": { ... },
    "special_statuses": { ... }
  },
  "change_workflow": {
    "version": "1.0",
    "status_flow": { ... },
    "status_metadata": { ... },
    "special_statuses": { ... }
  }
}
```

See [Configuration — Bug Workflow](configuration.md#bug-workflow-configuration) and [Configuration — Change-Card Workflow](configuration.md#change-card-workflow-configuration) for complete examples.

---

## Related Documentation

- **[configuration.md](configuration.md)** - Config file reference (includes bug/change workflow examples)
- **[bug-commands.md](bug-commands.md)** - Bug CLI commands
- **[change-commands.md](change-commands.md)** - Change-card CLI commands
- **[template-system.md](template-system.md)** - Template configuration
- **[workflow-profiles.md](../guides/workflow-profiles.md)** - Pre-configured task workflows
