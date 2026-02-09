---
epic_key: E16
title: Multi-Level Workflow System
description: Extend shark's configurable workflow engine from task-only to epic, feature, and task levels with level-specific status flows, orchestrator actions, and cascading state management
---

# Multi-Level Workflow System

**Epic Key**: E16

---

## Goal

### Problem

Shark's configurable workflow system (delivered in E11) applies only to **tasks**. Epics and features have hardcoded statuses (`draft`, `active`, `completed`, `archived`) with no configurable transitions, no workflow enforcement, and no orchestrator action responses.

This creates a structural gap in the PDLC (Product Development Lifecycle):

1. **Planning work has no workflow tracking.** When a BA refines requirements or an architect designs a system, that work happens at the feature level. But features have no refinement statuses, so this planning work gets pushed down into task-level statuses (`ready_for_refinement_ba`, `ready_for_refinement_tech`), which is semantically wrong. Tasks shouldn't exist until feature-level planning is complete.

2. **No orchestrator guidance at epic/feature level.** The `orchestrator_action` mechanism (proposed in the workflow-aware task system) tells the orchestrator what agent to spawn when a task reaches a given status. Epics and features have no equivalent, so orchestration tools like `/build` must hardcode all dispatch logic.

3. **The "active" status is overloaded.** Currently, when a feature is `active`, it means "some tasks exist and we're tracking them." There's no distinction between "feature is being planned" and "feature is being built." An orchestrator can't tell if a feature needs a BA, an architect, or developers.

4. **No upward escalation path.** When task-level work discovers a design flaw, there's no mechanism to push the parent feature back to a refinement state.

### Solution

Extend the `.sharkconfig.json` workflow configuration to support **three independent workflow definitions**: one each for epics, features, and tasks. Each level gets its own status flow, metadata, and orchestrator actions. When an entity reaches its "active" terminal planning state, it transitions to **aggregation mode** where its progress is derived from children below (the current behavior).

### Impact

- **Enable end-to-end PDLC orchestration**: A single `/feature` command can drive a feature from draft through BA refinement, technical design, task generation, and handoff to `/build` -- all tracked by shark
- **Clean separation of concerns**: Planning workflows live at the right level; task workflows are pure execution
- **Orchestrator becomes config-driven at all levels**: No hardcoded dispatch logic; shark tells the orchestrator what to do at every status transition, for every entity type
- **Foundation for autonomous pipelines**: Epic decomposition, feature refinement, and task execution can each be fully automated with appropriate agents

---

## Business Value

**Rating**: High

This is the architectural keystone that connects shark's existing task workflow engine to the full product development lifecycle. Without it, orchestration tools must maintain separate, hardcoded knowledge of what happens at each level. With it, shark becomes a **unified state machine** that can drive autonomous development from epic vision through production deployment.

---

## Current State

### Epic Entity

| Field | Current | Notes |
|-------|---------|-------|
| Statuses | `draft`, `active`, `completed`, `archived` | Hardcoded in Go, not configurable |
| Status transitions | Any → any (no validation) | No workflow enforcement |
| Progress | Aggregated from features/tasks | Only meaningful when `active` |
| `next-status` command | Does not exist | Only `shark task next-status` exists |
| Orchestrator actions | None | No action metadata on status changes |

### Feature Entity

| Field | Current | Notes |
|-------|---------|-------|
| Statuses | `draft`, `active`, `completed`, `archived` | Hardcoded in Go, not configurable |
| Status transitions | Any → any (no validation) | No workflow enforcement |
| Progress | Aggregated from tasks | Only meaningful when `active` |
| `next-status` command | Does not exist | Only `shark task next-status` exists |
| Orchestrator actions | None | No action metadata on status changes |

### Task Entity (Reference -- Already Implemented)

| Field | Current | Notes |
|-------|---------|-------|
| Statuses | Configurable via `.sharkconfig.json` | E11 delivered this |
| Status transitions | Validated against `status_flow` | Enforced, with `--force` override |
| Progress | Own status is progress | Not aggregated |
| `next-status` command | `shark task next-status <id>` | Advances to next logical status |
| Orchestrator actions | Partially implemented | `show-actions` command exists, action metadata in config |

---

## Proposed Design

### Configuration Structure

Extend `.sharkconfig.json` to support per-level workflow definitions. The existing top-level `status_flow` and `status_metadata` become the `task` workflow (backward compatible). New `epic_workflow` and `feature_workflow` sections are added.

```json
{
  "epic_workflow": {
    "version": "1.0",
    "status_flow": {
      "draft":                    ["ready_for_research", "active", "cancelled"],
      "ready_for_research":       ["in_research", "cancelled"],
      "in_research":              ["ready_for_refinement", "blocked"],
      "ready_for_refinement":     ["in_refinement", "blocked"],
      "in_refinement":            ["ready_for_decomposition", "blocked"],
      "ready_for_decomposition":  ["in_decomposition", "blocked"],
      "in_decomposition":         ["active", "blocked"],
      "active":                   ["completed", "on_hold"],
      "completed":                [],
      "cancelled":                [],
      "on_hold":                  ["ready_for_research", "ready_for_refinement", "active"],
      "blocked":                  ["ready_for_research", "ready_for_refinement", "ready_for_decomposition"]
    },
    "status_metadata": {
      "draft": {
        "description": "Epic captured, not yet researched",
        "phase": "planning",
        "color": "gray",
        "is_planning": true,
        "orchestrator_action": {
          "action": "wait_for_triage",
          "instruction_template": "Epic {id} needs review before research begins."
        }
      },
      "ready_for_research": {
        "description": "Ready for market/feasibility research",
        "phase": "research",
        "color": "purple",
        "is_planning": true,
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "researcher",
          "skills": ["discovery", "research"],
          "instruction_template": "Research market, competitors, and feasibility for epic {id}. Review epic document and produce research findings."
        }
      },
      "in_research": {
        "description": "Research in progress",
        "phase": "research",
        "color": "purple",
        "is_planning": true
      },
      "ready_for_refinement": {
        "description": "Research complete, ready for BA to write epic PRD",
        "phase": "refinement",
        "color": "orange",
        "is_planning": true,
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "business-analyst",
          "skills": ["specification-writing"],
          "instruction_template": "Write comprehensive epic PRD for {id} using research artifacts. Define scope, success criteria, and high-level feature areas."
        }
      },
      "in_refinement": {
        "description": "BA refining epic requirements",
        "phase": "refinement",
        "color": "orange",
        "is_planning": true
      },
      "ready_for_decomposition": {
        "description": "Epic PRD complete, ready to decompose into features",
        "phase": "decomposition",
        "color": "yellow",
        "is_planning": true,
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "product-manager",
          "skills": ["specification-writing"],
          "instruction_template": "Decompose epic {id} into features. Create each feature in shark with draft status. Reference epic PRD for scope."
        }
      },
      "in_decomposition": {
        "description": "Features being generated from epic",
        "phase": "decomposition",
        "color": "yellow",
        "is_planning": true
      },
      "active": {
        "description": "Features exist, progress aggregated from children",
        "phase": "execution",
        "color": "blue",
        "is_planning": false,
        "aggregates_from": "features"
      },
      "completed": {
        "description": "All features complete",
        "phase": "done",
        "color": "green",
        "is_planning": false
      },
      "cancelled": {
        "description": "Epic abandoned",
        "phase": "done",
        "color": "gray",
        "is_planning": false
      },
      "on_hold": {
        "description": "Epic intentionally paused",
        "phase": "paused",
        "color": "orange",
        "is_planning": true
      },
      "blocked": {
        "description": "Epic blocked by external dependency",
        "phase": "blocked",
        "color": "red",
        "is_planning": true
      }
    },
    "special_statuses": {
      "_start_": ["draft"],
      "_complete_": ["completed", "cancelled"],
      "_aggregation_": ["active"]
    }
  },

  "feature_workflow": {
    "version": "1.0",
    "status_flow": {
      "draft":                        ["ready_for_refinement_ba", "active", "cancelled"],
      "ready_for_refinement_ba":      ["in_refinement_ba", "cancelled"],
      "in_refinement_ba":             ["ready_for_refinement_tech", "draft", "blocked"],
      "ready_for_refinement_tech":    ["in_refinement_tech", "blocked"],
      "in_refinement_tech":           ["ready_for_task_generation", "ready_for_refinement_ba", "blocked"],
      "ready_for_task_generation":    ["in_task_generation", "blocked"],
      "in_task_generation":           ["ready_to_build", "blocked"],
      "ready_to_build":               ["active", "blocked"],
      "active":                       ["completed", "on_hold"],
      "completed":                    [],
      "cancelled":                    [],
      "on_hold":                      ["ready_for_refinement_ba", "ready_for_refinement_tech", "active"],
      "blocked":                      ["ready_for_refinement_ba", "ready_for_refinement_tech", "ready_for_task_generation", "ready_to_build", "draft"]
    },
    "status_metadata": {
      "draft": {
        "description": "Feature identified, not yet refined",
        "phase": "planning",
        "color": "gray",
        "is_planning": true,
        "orchestrator_action": {
          "action": "wait_for_triage",
          "instruction_template": "Feature {id} needs triage before refinement begins."
        }
      },
      "ready_for_refinement_ba": {
        "description": "Ready for business analysis and requirements specification",
        "phase": "refinement",
        "color": "cyan",
        "is_planning": true,
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "business-analyst",
          "skills": ["specification-writing", "shark-task-management"],
          "instruction_template": "Write feature PRD for {id}. Define user stories, acceptance criteria, and business rules. Read epic context for scope boundaries."
        }
      },
      "in_refinement_ba": {
        "description": "BA actively refining requirements",
        "phase": "refinement",
        "color": "blue",
        "is_planning": true
      },
      "ready_for_refinement_tech": {
        "description": "Requirements complete, ready for technical architecture",
        "phase": "refinement",
        "color": "cyan",
        "is_planning": true,
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "architect",
          "skills": ["architecture", "specification-writing", "shark-task-management"],
          "instruction_template": "Design technical architecture for feature {id}. Create architecture doc, API contracts, data model, and security design. Reference feature PRD for requirements."
        }
      },
      "in_refinement_tech": {
        "description": "Architect designing technical solution",
        "phase": "refinement",
        "color": "blue",
        "is_planning": true
      },
      "ready_for_task_generation": {
        "description": "Architecture complete, ready to generate implementation tasks",
        "phase": "decomposition",
        "color": "yellow",
        "is_planning": true,
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "product-manager",
          "skills": ["specification-writing", "shark-task-management"],
          "instruction_template": "Generate implementation tasks for feature {id}. Create tasks in shark with ready_for_development status. Reference PRD and architecture docs."
        }
      },
      "in_task_generation": {
        "description": "Tasks being generated from design documents",
        "phase": "decomposition",
        "color": "yellow",
        "is_planning": true
      },
      "ready_to_build": {
        "description": "All tasks generated and specified, ready for autonomous build",
        "phase": "execution",
        "color": "green",
        "is_planning": true,
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "tech-director",
          "skills": ["build", "shark-task-management"],
          "instruction_template": "Execute /build for feature {id}. All tasks have specifications and are ready for development. Drive tasks through dev, review, QA, and approval."
        }
      },
      "active": {
        "description": "Build in progress, progress aggregated from tasks",
        "phase": "execution",
        "color": "blue",
        "is_planning": false,
        "aggregates_from": "tasks"
      },
      "completed": {
        "description": "All tasks complete, feature delivered",
        "phase": "done",
        "color": "green",
        "is_planning": false
      },
      "cancelled": {
        "description": "Feature abandoned",
        "phase": "done",
        "color": "gray",
        "is_planning": false
      },
      "on_hold": {
        "description": "Feature intentionally paused",
        "phase": "paused",
        "color": "orange",
        "is_planning": true
      },
      "blocked": {
        "description": "Feature blocked by external dependency",
        "phase": "blocked",
        "color": "red",
        "is_planning": true
      }
    },
    "special_statuses": {
      "_start_": ["draft"],
      "_complete_": ["completed", "cancelled"],
      "_aggregation_": ["active"]
    }
  },

  "status_flow_version": "1.0",
  "status_flow": { "... existing task workflow, unchanged ..." },
  "status_metadata": { "... existing task metadata, unchanged ..." },
  "special_statuses": { "... existing task special statuses, unchanged ..." }
}
```

### Key Design Concept: The Aggregation Threshold

The `_aggregation_` special status is the bridge between this feature and the current behavior:

```
PLANNING STATES                    AGGREGATION STATES
(entity has its own status)        (entity derives progress from children)

draft → research → refine → ...  →  active  →  completed
                                       ↑
                                       │
                              "aggregation threshold"
                                       │
                          Before this: entity workflow
                          After this:  current behavior (unchanged)
```

**Before `active`**: The entity has its own workflow. `shark epic get E16` shows status `in_refinement`, not progress bars. The entity is being planned.

**At/after `active`**: The entity aggregates from children. `shark epic get E16` shows task/feature progress breakdowns, exactly as it does today. The entity is being executed.

This means **all existing behavior is preserved**. Any epic or feature currently at `active` continues to work identically. The new planning states are opt-in: you only enter them if you transition an entity out of `draft` into a planning status instead of directly to `active`.

### The Shortcut: Direct to Active

For users who don't need planning workflows, the direct path remains:

```
draft → active     (current behavior, always valid)
```

This is explicitly included in the `status_flow` for both epic and feature. Users can skip the entire planning workflow and go straight to aggregation mode.

---

## Functional Requirements

### FR-1: Level-Specific Workflow Configuration

**Description**: Support separate workflow definitions for epic, feature, and task levels in `.sharkconfig.json`.

**Requirements**:

1. Parse `epic_workflow` section from config if present
2. Parse `feature_workflow` section from config if present
3. Existing top-level `status_flow` / `status_metadata` remains the task workflow (backward compatible)
4. If `epic_workflow` is absent, use default hardcoded statuses: `draft`, `active`, `completed`, `archived` (current behavior)
5. If `feature_workflow` is absent, use default hardcoded statuses: `draft`, `active`, `completed`, `archived` (current behavior)
6. Validate each workflow independently (no cross-references between levels)

**Backward Compatibility**: Existing `.sharkconfig.json` files with only task-level workflow continue to work unchanged. Epic and feature default to current hardcoded behavior.

### FR-2: Epic Status Transitions with Validation

**Description**: Apply workflow validation to epic status changes, matching the existing task behavior.

**Requirements**:

1. `shark epic update <key> --status <new>` validates transition against `epic_workflow.status_flow`
2. Invalid transitions are rejected with error message showing valid options
3. `--force` flag overrides validation (matching task behavior)
4. History/audit trail records epic status changes (if history system supports it)

**CLI Changes**:

```bash
# Existing command, enhanced behavior:
shark epic update E16 --status ready_for_research

# Response (success):
Epic E16 updated: draft → ready_for_research

# Response (invalid transition):
Error: Cannot transition epic E16 from 'draft' to 'in_refinement'
Valid transitions from 'draft': ready_for_research, active, cancelled
Use --force to override workflow validation.
```

### FR-3: Feature Status Transitions with Validation

**Description**: Apply workflow validation to feature status changes.

**Requirements**:

1. `shark feature update <key> --status <new>` validates transition against `feature_workflow.status_flow`
2. Same error handling and `--force` behavior as epic and task levels
3. History/audit trail records feature status changes

**CLI Changes**:

```bash
shark feature update E16-F01 --status ready_for_refinement_ba

# Response:
Feature E16-F01 updated: draft → ready_for_refinement_ba
```

### FR-4: `next-status` Command for Epic and Feature

**Description**: Add `shark epic next-status` and `shark feature next-status` commands, mirroring the existing `shark task next-status`.

**Requirements**:

1. `shark epic next-status <key>` advances epic to the next logical status in its workflow
2. `shark feature next-status <key>` advances feature to the next logical status in its workflow
3. "Next logical status" = first entry in the `status_flow` array for the current status (same logic as task `next-status`)
4. Returns the transition details and orchestrator action (if defined) in the response

**CLI**:

```bash
shark epic next-status E16
# Response:
Epic E16: draft → ready_for_research

shark feature next-status E16-F01
# Response:
Feature E16-F01: ready_for_refinement_ba → in_refinement_ba
```

**JSON response** (when `--json` flag used):

```json
{
  "success": true,
  "entity_type": "feature",
  "entity_id": "E16-F01",
  "transition": {
    "from": "ready_for_refinement_ba",
    "to": "in_refinement_ba",
    "timestamp": "2026-02-08T15:30:00Z"
  },
  "orchestrator_action": null
}
```

```json
{
  "success": true,
  "entity_type": "feature",
  "entity_id": "E16-F01",
  "transition": {
    "from": "in_refinement_ba",
    "to": "ready_for_refinement_tech",
    "timestamp": "2026-02-08T16:00:00Z"
  },
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "architect",
    "skills": ["architecture", "specification-writing", "shark-task-management"],
    "instruction": "Design technical architecture for feature E16-F01. Create architecture doc, API contracts, data model, and security design. Reference feature PRD for requirements."
  }
}
```

### FR-5: Orchestrator Actions for Epic and Feature Levels

**Description**: Include `orchestrator_action` in status metadata for epic and feature workflows, returned in transition responses.

**Requirements**:

1. `orchestrator_action` is optional on any status (same as task level)
2. When present, it is returned in the JSON response of `update --status` and `next-status` commands
3. `{id}` placeholder in `instruction_template` is replaced with the entity key (e.g., `E16`, `E16-F01`)
4. Action types: `spawn_agent`, `pause`, `wait_for_triage`, `archive` (same set as task level)
5. `shark workflow show-actions` extended to show actions for all three levels

**CLI**:

```bash
shark workflow show-actions
# Response:
Epic Workflow Actions:
  ready_for_research      → spawn_agent (researcher)
  ready_for_refinement    → spawn_agent (business-analyst)
  ready_for_decomposition → spawn_agent (product-manager)
  ...

Feature Workflow Actions:
  ready_for_refinement_ba   → spawn_agent (business-analyst)
  ready_for_refinement_tech → spawn_agent (architect)
  ready_for_task_generation → spawn_agent (product-manager)
  ready_to_build            → spawn_agent (tech-director)
  ...

Task Workflow Actions:
  ready_for_development   → spawn_agent (developer)
  ready_for_code_review   → spawn_agent (tech-lead)
  ...
```

### FR-6: Aggregation Threshold Behavior

**Description**: When an entity reaches an aggregation status (marked with `_aggregation_` or `aggregates_from`), its display behavior switches from showing its own workflow status to showing aggregated child progress.

**Requirements**:

1. New `_aggregation_` key in `special_statuses` identifies which statuses trigger aggregation mode
2. Status metadata can include `"aggregates_from": "features"` or `"aggregates_from": "tasks"` to indicate what children to aggregate
3. `shark epic get <key>` behavior:
   - **If status is a planning status** (`is_planning: true`): Show the epic's own status, description, and workflow position. Do NOT show aggregated progress (there may be no children yet).
   - **If status is an aggregation status** (`is_planning: false`): Show aggregated progress from features/tasks (current behavior, unchanged).
4. `shark feature get <key>` behavior:
   - **If status is a planning status**: Show the feature's own status and workflow position.
   - **If status is an aggregation status**: Show aggregated progress from tasks (current behavior, unchanged).
5. `shark epic status` and `shark status` commands respect the same logic: planning-phase entities show their workflow status, aggregation-phase entities show progress.

**Example -- Epic in planning phase**:

```
$ shark epic get E16

Epic: E16 - Multi-Level Workflow System
Status: in_refinement (Refinement phase)
Phase: refinement

Workflow Position:
  draft → research → [in_refinement] → decomposition → active → completed
                       ^^^^^^^^^^^
                       YOU ARE HERE

No features yet (epic is still being refined).
```

**Example -- Epic in aggregation phase**:

```
$ shark epic get E16

Epic: E16 - Multi-Level Workflow System
Status: active

Features: 3 total
  E16-F01  Core Workflow Engine        active     ████████░░  80%
  E16-F02  CLI Command Extensions      in_refinement_tech  (planning)
  E16-F03  Migration & Compatibility   draft               (planning)

Tasks: 15 total | 8 completed | 4 in_progress | 2 ready | 1 blocked
Progress: ████████░░░░ 62%
```

Note how features in planning states show `(planning)` instead of a progress bar -- they don't have tasks to aggregate yet.

### FR-7: Workflow Validation for All Levels

**Description**: Extend `shark workflow validate` to cover epic and feature workflows.

**Requirements**:

1. Validate `epic_workflow` section if present: check for unreachable statuses, missing terminal states, orphan statuses
2. Validate `feature_workflow` section if present: same checks
3. Validate task workflow: existing behavior, unchanged
4. Report per-level: "Epic workflow: valid", "Feature workflow: valid", "Task workflow: valid"
5. `shark workflow validate-actions` extended to check all three levels

### FR-8: Workflow List for All Levels

**Description**: Extend `shark workflow list` to display all configured workflows.

**Requirements**:

1. Show epic workflow (if configured, or "default" if using hardcoded)
2. Show feature workflow (if configured, or "default" if using hardcoded)
3. Show task workflow (existing behavior)
4. Visually distinguish planning statuses from aggregation statuses

**Example**:

```
$ shark workflow list

Epic Workflow (custom):
  draft → ready_for_research → in_research → ready_for_refinement →
  in_refinement → ready_for_decomposition → in_decomposition → [active] →
  completed

Feature Workflow (custom):
  draft → ready_for_refinement_ba → in_refinement_ba →
  ready_for_refinement_tech → in_refinement_tech →
  ready_for_task_generation → in_task_generation → ready_to_build →
  [active] → completed

Task Workflow (custom):
  draft → ready_for_development → in_development →
  ready_for_code_review → in_code_review → ready_for_qa → in_qa →
  ready_for_approval → in_approval → completed

[active] = aggregation threshold (progress derived from children)
```

### FR-9: Upward Escalation (Feature Block Propagation)

**Description**: When all tasks in a feature are blocked, or a task-level block cites a feature-level concern, provide tooling to push the feature back to a planning state.

**Requirements**:

This is NOT automatic propagation. It is explicit commands with guardrails:

1. `shark feature set-status <key> <status>` allows setting any valid status (with workflow validation)
2. When moving a feature BACKWARD (from `active` to a planning status like `ready_for_refinement_tech`), require `--reason` flag
3. Log the reason in feature history/notes
4. Do NOT automatically change child task statuses (tasks remain as-is; the orchestrator decides what to do with them)

**Example**:

```bash
shark feature set-status E16-F01 ready_for_refinement_tech \
  --reason "Architecture redesign needed: API contract conflicts with database schema discovered during T-E16-F01-003"

# Response:
Feature E16-F01: active → ready_for_refinement_tech
Reason logged. 15 tasks remain in current states.
Warning: Feature moved backward to planning phase. Tasks are unchanged.
```

### FR-10: Notes and Context for Epic and Feature Levels

**Description**: Extend the note and context system (currently task-only) to epic and feature entities.

**Requirements**:

1. `shark epic note add <key> --type <type> "<content>"` -- add notes to epics
2. `shark feature note add <key> --type <type> "<content>"` -- add notes to features
3. `shark epic context set <key> --field <field> --value "<value>"` -- set context on epics
4. `shark feature context set <key> --field <field> --value "<value>"` -- set context on features
5. Same note types as tasks: `comment`, `decision`, `blocker`, `solution`, `reference`, `implementation`, `testing`, `future`, `question`
6. `shark epic resume <key>` and `shark feature resume <key>` -- get full context for resuming work (notes, context, history, current status)

**Rationale**: Agents working on epic/feature-level refinement need the same persistent context tracking that task-level agents already have. A BA refining a feature PRD needs to record decisions, a researcher analyzing market feasibility needs to log findings.

---

## Non-Functional Requirements

### NFR-1: Backward Compatibility

- Existing `.sharkconfig.json` files without `epic_workflow` or `feature_workflow` sections continue to work unchanged
- Epics and features default to current hardcoded statuses (`draft`, `active`, `completed`, `archived`) when no custom workflow is configured
- The `draft → active` shortcut is always valid, even in custom workflows
- Existing `shark epic update --status active` and `shark feature update --status active` commands work identically to today

### NFR-2: Performance

- Workflow validation for all three levels completes in < 100ms
- `next-status` commands for epic/feature have same performance characteristics as task `next-status`
- Aggregation calculations (when in `active` state) are unchanged from current behavior

### NFR-3: Configuration Validation

- Invalid `epic_workflow` or `feature_workflow` configs are caught at parse time with actionable error messages
- Missing required fields (e.g., status in `status_flow` not in `status_metadata`) produce clear warnings
- `shark workflow validate` catches all configuration issues before they cause runtime errors

---

## Migration Path

### Phase 1: No Migration Needed

This feature is additive. No data migration is required.

- Existing epics at `active` status: unchanged behavior
- Existing features at `active` status: unchanged behavior
- Existing `.sharkconfig.json` without new sections: default behavior

### Phase 2: Opt-In Adoption

Users add `epic_workflow` and/or `feature_workflow` sections to their `.sharkconfig.json` when ready.

Existing entities can be transitioned to new planning statuses via:
```bash
shark epic update E16 --status draft          # Reset to start of workflow
shark feature update E16-F01 --status draft   # Reset to start of workflow
```

### Phase 3: Existing Entity Cleanup (Optional)

If users want to retroactively assign planning statuses to entities that skipped the workflow:
```bash
# Feature that was already refined but never tracked:
shark feature update E16-F01 --status ready_to_build --force
# This skips the planning workflow but correctly positions the feature
```

---

## Feature Decomposition (Proposed)

| Feature | Description | Priority |
|---------|-------------|----------|
| E16-F01 | **Core Workflow Engine** -- Parse level-specific configs, validate transitions for epic/feature, `next-status` commands | P0 |
| E16-F02 | **Orchestrator Actions** -- Return `orchestrator_action` in epic/feature transition responses, extend `show-actions` | P1 |
| E16-F03 | **Display & Aggregation Threshold** -- Update `get`/`status` commands to respect `is_planning` vs aggregation mode | P1 |
| E16-F04 | **Notes & Context for Epic/Feature** -- Extend note and context commands to epic and feature entities | P2 |
| E16-F05 | **Backward Transition & Escalation** -- `set-status` with `--reason`, backward transition guards | P2 |
| E16-F06 | **Workflow Visualization** -- Update `workflow list` to show all three levels, planning vs aggregation distinction | P3 |

---

## Open Questions

1. **Should `active` be configurable or always the aggregation threshold?** The current proposal hardcodes `active` as the threshold via `_aggregation_` special status. Alternative: any status marked with `aggregates_from` triggers aggregation, allowing multiple aggregation states.

2. **Should `shark feature next-status` auto-transition to `active` after `ready_to_build`?** Or should that be a separate explicit step? The `ready_to_build → active` transition is where `/build` takes over, so making it explicit may be clearer.

3. **Should epic completion be auto-detected?** When all features reach `completed`, should the epic auto-complete? Or require explicit `shark epic next-status`?

4. **History table schema**: Does the existing history/audit system need schema changes to support epic and feature status transitions, or can it use the same table with an `entity_type` column?

5. **Feature status in `shark task list` output**: When listing tasks, should the parent feature's workflow status be visible? This would help orchestrators understand context without a separate query.

---

## Success Criteria

1. `shark epic next-status` and `shark feature next-status` work with configurable workflows
2. Orchestrator actions are returned in transition responses for all entity levels
3. Existing behavior is 100% preserved when no custom epic/feature workflow is configured
4. `shark workflow list` shows all three workflow levels
5. `shark workflow validate` validates all three levels
6. Epics in planning states show workflow position; epics in `active` show aggregated progress (current behavior)
7. Features in planning states show workflow position; features in `active` show aggregated progress (current behavior)
8. The `draft → active` shortcut remains valid at all levels

---

## References

- **E11 - Configurable Status Workflow System**: Foundation that delivered task-level configurable workflows
- **E13 - Workflow-Aware Task Command System**: Phase-aware commands and orchestrator actions for tasks
- **Shark-Claude Integration Notes**: `/home/jwwelbor/.claude/docs/architecture/shark-claude-integration-notes.md`
- **Orchestrator Actions Feature Request**: `/home/jwwelbor/.claude/docs/architecture/shark-orchestrator-actions-feature-request.md` (if exists)
- **WormwoodGM .sharkconfig.json**: `/home/jwwelbor/projects/wormwoodGM/.sharkconfig.json` -- reference implementation of rich task workflow
