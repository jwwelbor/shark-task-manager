# Scope Boundaries

**Epic**: [Configurable Status Workflow System](./epic.md)

---

## Overview

This document explicitly defines what is **NOT** included in this epic to prevent scope creep and set clear expectations.

---

## Out of Scope

### Explicitly Excluded Features

**1. Legacy Task Data Migration**
- **What**: Automatic or manual migration of existing tasks from old hardcoded statuses to new configurable workflow statuses
- **Why It's Out of Scope**:
  - This epic targets new projects or projects without existing tasks
  - Migration adds significant complexity (data safety, rollback, testing)
  - Current Shark installations can adopt new workflow for new tasks only
  - Existing tasks can remain with legacy statuses (backward compatible)
- **Future Consideration**: Maybe - if user demand exists for migrating existing projects
- **Workaround**: For existing projects, use default workflow (matches legacy statuses) or manually update task statuses as needed
- **Important Clarification**: "Code migration" (refactoring Shark's codebase to use workflow config) IS in scope and has been completed in F01-F04. This exclusion refers ONLY to task data migration

**2. Workflow Automation / Transition Hooks**
- **What**: Automatic execution of scripts or webhooks when status transitions occur (e.g., send Slack notification when task moves to `ready_for_qa`)
- **Why It's Out of Scope**:
  - Adds significant complexity (error handling, async execution, security sandboxing)
  - Requires design decisions around hook configuration, retry logic, failure modes
  - Security implications of executing arbitrary code on status changes
  - Better suited for future epic focused on integrations/automation
- **Future Consideration**: Yes - planned for E12 "Workflow Automation & Integrations" (tentative)
- **Workaround**: Users can build external tools that poll `task_history` and react to status changes

**2. Multiple Parallel Workflows**
- **What**: Support for different workflows per task type (e.g., feature tasks use 14-status workflow, hotfix tasks use 4-status workflow)
- **Why It's Out of Scope**:
  - Increases configuration complexity (workflow selection logic, task type taxonomy)
  - Most teams have one primary workflow (can customize if needed)
  - Adds UI complexity (how to show which workflow applies?)
  - Better to validate single-workflow model before expanding
- **Future Consideration**: Possible Phase 2 enhancement if user demand exists
- **Workaround**: Use single workflow with optional statuses (not all tasks must visit all statuses)

**3. Time-Based Status Transitions**
- **What**: Automatic status changes after time duration (e.g., tasks in `in_progress` for >5 days auto-move to `on_hold`)
- **Why It's Out of Scope**:
  - Requires background job scheduler (not part of current architecture)
  - Time thresholds are subjective and error-prone (some tasks legitimately take weeks)
  - Better handled by human/agent review than automation
- **Future Consideration**: No - low value, high complexity
- **Workaround**: Use `shark task list --status=in_progress` with date filters (once filtering by duration is added)

**4. External System Integrations**
- **What**: Direct integration with Jira, GitHub Projects, Linear, etc. for workflow sync
- **Why It's Out of Scope**:
  - Each integration requires custom mapping logic, authentication, error handling
  - Maintenance burden for multiple external APIs
  - Better suited for separate integration epic or plugin system
- **Future Consideration**: Yes - separate epic for "External Integrations" (E13 tentative)
- **Workaround**: Manual sync or use Zapier/n8n for custom integrations

**5. Workflow Versioning / History**
- **What**: Track changes to workflow configuration over time, revert to previous workflow versions
- **Why It's Out of Scope**:
  - Config file versioning is handled by Git (users should commit `.sharkconfig.json`)
  - In-app versioning adds complexity without clear value
  - Edge case: what happens to tasks mid-flight when workflow changes? (deferred decision)
- **Future Consideration**: Maybe - low priority unless enterprise users request it
- **Workaround**: Use Git for config versioning

**6. Graphical Workflow Editor**
- **What**: GUI tool to visually design workflow (drag-and-drop nodes, connect transitions)
- **Why It's Out of Scope**:
  - Requires substantial frontend development (not core CLI focus)
  - JSON config is sufficient for power users
  - Most users will use templates, not custom workflows
- **Future Consideration**: Maybe - if web UI is built (separate epic), workflow editor could be added
- **Workaround**: Edit JSON directly or use Mermaid diagram + manual JSON translation

**7. Per-User Workflow Permissions**
- **What**: Role-based access control for status transitions (e.g., only tech leads can approve tasks)
- **Why It's Out of Scope**:
  - Shark currently has no user/role system
  - Adds authentication and authorization complexity
  - Most Shark users are AI agents (no user identity concept yet)
- **Future Consideration**: Yes - if multi-user/multi-agent auth is added (E14 tentative)
- **Workaround**: Trust model (all agents/users can perform all transitions; audit trail in task_history)

---

### Edge Cases & Scenarios Not Covered

**1. Workflow Changes with Tasks In-Flight**
- **Scenario**: Project has 50 tasks in status `in_development`. Admin changes workflow config to remove `in_development` status.
- **Impact**: Medium - tasks now have "undefined" status
- **Rationale**: Rare edge case; most config changes add statuses, not remove
- **Mitigation**:
  - Validation warning (not error) if any tasks have statuses being removed
  - Tasks with undefined statuses display with ⚠️ warning in `task list`
  - `shark task fix-status <task-key>` command to interactively choose new status
- **Deferred To**: Phase 3 or Phase 4 (low priority)

**2. Concurrent Status Updates (Race Condition)**
- **Scenario**: Two agents query `task next`, both get same task, both try to claim it simultaneously
- **Impact**: Low - database handles via optimistic locking (one update succeeds, other fails)
- **Rationale**: Edge case in practice (agents query at ~1 second intervals)
- **Mitigation**:
  - First agent's `set-status` succeeds
  - Second agent's `set-status` fails with "Task was modified by another agent"
  - Second agent re-queries, gets different task
- **Deferred To**: Not addressing further (current behavior is acceptable)

**3. Very Large Workflows (100+ Statuses)**
- **Scenario**: Enterprise team defines workflow with 150 statuses and 500 transition rules
- **Impact**: Low - most teams use 5-15 statuses
- **Rationale**: Performance testing validates up to 100 statuses; beyond that is untested
- **Mitigation**: Documentation recommends <50 statuses for maintainability
- **Deferred To**: Only address if real-world use case emerges

**4. Circular Workflow (No Terminal State)**
- **Scenario**: User defines workflow where statuses loop forever with no path to completion
- **Impact**: Medium - tasks can never be marked "done"
- **Rationale**: Validation detects this (REQ-F-002: detect dead-end statuses)
- **Mitigation**: `shark workflow validate` fails with error, user must fix config
- **Handled In**: Phase 1 (validation)

**5. Migration with Corrupted Task History**
- **Scenario**: Legacy project has task with `status='in_progress'` but `task_history` shows last entry as `status='completed'` (data inconsistency)
- **Impact**: Low - rare (indicates previous bug or manual DB edit)
- **Rationale**: Migration should preserve existing data, even if inconsistent
- **Mitigation**:
  - Migration uses current task status (not history) as source of truth
  - Warning logged: "Task T-X has inconsistent history, using current status"
- **Deferred To**: Phase 2 (migration) - log warning but don't block

---

## Alternative Approaches Considered But Rejected

**Alternative 1: Database-Level Workflow Validation**
- **Description**: Store workflow config in database, use CHECK constraints or triggers to enforce valid transitions
- **Pros**:
  - Database enforces constraints atomically (no race conditions)
  - Validation happens even if application layer is bypassed
- **Cons**:
  - Database triggers can't read JSON config file easily (would need to store config in DB)
  - Dynamic workflow changes require ALTER TABLE (slow, risky)
  - SQLite triggers are limited (no complex logic)
  - Violates principle of database as dumb storage, application as smart logic
- **Decision Rationale**: Application-layer validation is more flexible and maintainable; database provides ACID guarantees, not business logic

**Alternative 2: Separate Workflow Definition DSL**
- **Description**: Create custom domain-specific language for workflow definition (not JSON)
- **Pros**:
  - More expressive syntax (e.g., `draft -> refinement -> development -> done`)
  - Easier to write by hand
- **Cons**:
  - Requires custom parser (maintenance burden)
  - JSON is universal (every language/tool can parse it)
  - JSON Schema provides validation out-of-the-box
  - Custom DSL adds learning curve
- **Decision Rationale**: JSON is "good enough" and widely understood; custom DSL is over-engineering

**Alternative 3: Hardcoded Workflow with Extension Points**
- **Description**: Keep current hardcoded workflow, add hooks for custom statuses (e.g., `custom_1`, `custom_2`)
- **Pros**:
  - Simpler implementation (no config parsing)
  - Backward compatible by default
- **Cons**:
  - Not truly flexible (teams still constrained by hardcoded structure)
  - Doesn't solve multi-agent coordination problem
  - Generic status names (`custom_1`) are confusing
  - Doesn't support backward transitions
- **Decision Rationale**: Defeats the purpose of configurable workflows; too limited

**Alternative 4: Graph Database for Workflow**
- **Description**: Use graph database (e.g., Neo4j) to store workflow as nodes (statuses) and edges (transitions)
- **Pros**:
  - Native graph queries (find all paths from status A to B)
  - Visualizable out-of-the-box
- **Cons**:
  - Adds dependency on graph database (complexity, deployment, backup)
  - Overkill for simple directed graph (workflow is not complex enough to justify)
  - SQLite is sufficient for validation (in-memory graph from JSON config)
- **Decision Rationale**: JSON config + in-memory graph is simpler and sufficient

**Alternative 5: Automatic Migration Support**
- **Description**: Include automatic migration of legacy task statuses to new workflow statuses
- **Pros**:
  - Existing projects can adopt new workflows seamlessly
  - Preserves historical data with new structure
- **Cons**:
  - Adds significant complexity (migration planning, dry-run, rollback, data safety)
  - Requires extensive testing to prevent data loss
  - Not needed if targeting new projects
  - Can be added later if demand exists
- **Decision Rationale**: Migration is out of scope for initial release; backward compatibility via default workflow is sufficient

---

## Future Epic Candidates

Features that are natural follow-ons to this epic but require separate design and implementation:

| Future Epic Concept | Priority | Dependency | Estimated Size |
|---------------------|----------|------------|----------------|
| **E12: Workflow Automation & Integrations** | Medium | Depends on E11 | Large (8-10 weeks) |
| - Transition hooks (scripts/webhooks on status change) | | | |
| - External system sync (Jira, GitHub, Linear) | | | |
| - Notification system (Slack, email on status change) | | | |
| **E13: Workflow Analytics & Insights** | Low | Depends on E11 | Medium (4-6 weeks) |
| - Bottleneck detection (which statuses accumulate tasks?) | | | |
| - Cycle time analysis (average time in each status) | | | |
| - Workflow metrics dashboard | | | |
| **E14: Multi-User Workflow Permissions** | Low | Depends on E11 + auth system | Large (10-12 weeks) |
| - Role-based transition permissions | | | |
| - Approval chains (e.g., 2 approvers required for `completed`) | | | |
| - Audit log with user identity | | | |
| **E15: Multiple Workflow Support** | Low | Depends on E11 | Medium (5-7 weeks) |
| - Per-task-type workflows (feature, bug, spike) | | | |
| - Workflow selection UI | | | |
| - Workflow template library expansion | | | |

---

## Constraints & Assumptions

### Technical Constraints
- **SQLite limitations**: No native graph queries, no stored procedures (validation logic must be in Go)
- **CLI-first design**: All features must work from command line; GUI is optional future enhancement
- **Backward compatibility**: Existing projects must continue working without forced migration

### Business Constraints
- **Timeline**: Must deliver core workflow config + migration in <6 weeks (Phases 1-3)
- **Resources**: 1-2 developers available for implementation
- **Risk tolerance**: Zero tolerance for data loss during migration; medium tolerance for UX friction (can iterate)

### Assumptions
- **User technical proficiency**: Users comfortable editing JSON config files (target audience is developers)
- **Git usage**: Assume users track `.sharkconfig.json` in Git for versioning (no in-app versioning needed)
- **Single workflow suffices**: Most teams have one primary workflow; multiple workflows are edge case
- **Agent coordination**: AI agents will respect status signals (no rogue agents ignoring workflow)

---

*See also*: [Requirements](./requirements.md), [User Journeys](./user-journeys.md)
