# Scope Boundaries

**Epic**: [Workflow-Aware Task Command System](./epic.md)

---

## Overview

This document explicitly defines what is **NOT** included in the workflow-aware task command system epic. Clear boundaries prevent scope creep and set realistic expectations.

---

## Out of Scope

### Explicitly Excluded Features

**1. Analytics CLI Separation (`shark-analytics`)**
- **Why It's Out of Scope**: Mentioned in original idea I-2026-01-09-02 but separate concern
- **Rationale**:
  - Separating analytics/reporting commands into dedicated CLI is independent work
  - This epic focuses on workflow-aware phase commands, not command namespace reorganization
  - Analytics separation has different requirements (data aggregation, visualization, export)
- **Future Consideration**: Create Epic E14 for `shark-analytics` CLI
- **Workaround**: Keep `history` and `sessions` commands in main shark CLI for now

**2. Workflow Designer UI**
- **Why It's Out of Scope**: Visual workflow configuration editor
- **Rationale**:
  - Workflow config is `.sharkconfig.json` (text file) - sufficient for MVP
  - UI adds significant complexity (web framework, diagram rendering, validation UI)
  - Power users comfortable with JSON editing
  - Would delay epic by 4-6 weeks minimum
- **Future Consideration**: Epic for web-based workflow designer if user demand emerges
- **Workaround**: Provide example workflow templates and validation command

**3. Automatic Migration of Existing Tasks**
- **Why It's Out of Scope**: Auto-converting existing task statuses to new workflow
- **Rationale**:
  - Risky - could corrupt task data if migration logic has bugs
  - Each user's workflow is different - no one-size-fits-all migration
  - Manual migration (or staying with deprecated commands) is safer
  - Adds 2-3 weeks of testing and rollback mechanisms
- **Future Consideration**: Migration tool if users report difficulty
- **Workaround**:
  - Deprecated commands continue working (backward compatibility)
  - Users can manually update task statuses if desired
  - New tasks use new commands from day one

**4. Workflow Change Impact Analysis**
- **Why It's Out of Scope**: Tool that shows what happens if workflow config changes
- **Rationale**:
  - Nice-to-have but not critical for MVP
  - Complex to implement (simulate all transitions, detect orphaned states)
  - Users can test workflow changes in dev environment
  - Workflow validation (`shark workflow validate`) covers basic correctness
- **Future Consideration**: Add if users accidentally break workflows frequently
- **Mitigation**: Clear documentation on workflow design, validation on startup

**5. Multi-Tenancy / Team Isolation**
- **Why It's Out of Scope**: Different workflows for different teams in same shark instance
- **Rationale**:
  - Current shark assumes single team/project per database
  - Multi-tenancy requires epic/feature-level workflow assignment
  - Database schema changes (workflow_id foreign key)
  - Out of scope for this epic focused on command interface
- **Future Consideration**: Epic E15 for multi-team support
- **Workaround**: Use separate shark databases for teams with different workflows

**6. Workflow Execution Hooks / Triggers**
- **Why It's Out of Scope**: Custom scripts that run on status transitions (e.g., notify Slack on task completion)
- **Rationale**:
  - Hook system is separate epic (event system, webhook support)
  - This epic focused on command semantics, not event orchestration
  - Would add 3-4 weeks of implementation time
- **Future Consideration**: Epic for webhook/event system
- **Workaround**: AI orchestrator can implement notification logic externally

**7. Undo/Rollback for Status Transitions**
- **Why It's Out of Scope**: Ability to revert a `finish` or `claim` command
- **Rationale**:
  - Adds complexity (state snapshots, audit trail)
  - `reject` command serves similar purpose (send task backward)
  - Undo is rarely needed if error messages are clear
  - Can manually fix with `update --status` if needed
- **Future Consideration**: Add if users frequently make transition mistakes
- **Mitigation**: Require confirmation for bulk operations, clear error messages

---

### Edge Cases & Scenarios Not Covered

**1. Concurrent Status Updates (Race Conditions)**
- **Edge Case**: Two agents try to claim same task simultaneously
- **Impact**: Medium - could lead to double assignment
- **Rationale**: SQLite transactions provide some protection, but distributed race conditions are complex
- **Mitigation**:
  - Database UNIQUE constraint on (task_id, status, session_id)
  - Optimistic locking (check status before update)
  - Error message if claim fails due to concurrent update
- **Future**: Implement distributed locking if AI orchestrator runs multi-instance

**2. Workflow Config Hot-Reload**
- **Edge Case**: Workflow config changed while tasks are in progress
- **Impact**: Low - tasks mid-workflow might have invalid transitions
- **Rationale**: Requires watching config file, invalidating cache, re-validating in-flight tasks
- **Mitigation**:
  - Require shark restart after workflow changes (documented)
  - Validation on startup ensures config is correct before processing tasks
  - In-progress tasks continue with old workflow until finished
- **Future**: Add config reload command if demand exists

**3. Circular Workflow Loops**
- **Edge Case**: Workflow allows task to cycle infinitely (e.g., dev → review → dev → review...)
- **Impact**: Medium - task could get stuck in loop
- **Rationale**: Workflow validation can detect simple cycles but not all logical loops
- **Mitigation**:
  - Warn on workflow validation if cycles detected
  - Track task history - humans/PM can intervene if task bounces too many times
  - Not automated prevention - requires human judgment
- **Future**: Add max-transition-count rule if abuse occurs

**4. Orphaned Tasks After Workflow Change**
- **Edge Case**: Task has status "in_legacy_qa" but new workflow removes that status
- **Impact**: High - task becomes unmanageable
- **Rationale**: Workflow evolution is complex - can't auto-migrate safely
- **Mitigation**:
  - `shark workflow validate --migration` checks for orphaned statuses (future)
  - Document workflow evolution best practices
  - Manual status update if orphaned: `shark task update --status=<valid-status>`
- **Future**: Migration validation tool

**5. Agent Type Mismatches**
- **Edge Case**: Developer with agent=frontend claims task meant for agent=backend
- **Impact**: Low - wrong person works on task
- **Rationale**: Enforcement requires rigid RBAC system, reduces flexibility
- **Mitigation**:
  - `claim` validates agent type against `status_metadata.agent_types` (warning, not error)
  - AI orchestrator respects agent types strictly
  - Humans can override with `--force` flag
- **Future**: Add strict mode if needed

---

## Alternative Approaches Considered But Rejected

**Alternative 1: Extend Existing Commands Instead of New Commands**
- **Description**: Add `--phase-aware` flag to existing `start`, `complete`, etc.
- **Pros**:
  - No new commands to learn
  - Backward compatible by default
- **Cons**:
  - Confusing dual behavior (`start` vs. `start --phase-aware`)
  - Doesn't solve semantic confusion (complete vs. approve vs. next-status)
  - Hard to deprecate old behavior cleanly
  - Perpetuates poor naming choices
- **Decision Rationale**: Clean break with new command names (`claim`, `finish`) is clearer and allows gradual migration

**Alternative 2: Single "Advance" Command Instead of Claim/Finish**
- **Description**: One command `shark task advance <task-id>` handles all transitions
- **Pros**:
  - Simplest possible interface
  - Workflow config entirely drives behavior
- **Cons**:
  - Loses semantic clarity (advance from ready to in? or in to ready?)
  - Can't distinguish claim (start work) from finish (complete work)
  - No work session tracking (when did work start vs. end?)
  - Harder to implement rejection (backward advance?)
- **Decision Rationale**: Separate claim/finish commands provide clearer intent and enable session tracking

**Alternative 3: Workflow Engine with State Machine**
- **Description**: Implement full workflow engine (like Temporal, Camunda) instead of simple config
- **Pros**:
  - Very powerful - supports complex workflows, parallel paths, conditional branches
  - Industry-standard approach
- **Cons**:
  - Massive scope increase (12-16 weeks implementation)
  - Overkill for task management CLI
  - Adds heavy dependency
  - Steep learning curve for users
- **Decision Rationale**: Simple config-based workflow is 80/20 solution - covers most use cases with 20% of effort

**Alternative 4: GraphQL API for Workflow Queries**
- **Description**: Expose workflow transitions via GraphQL instead of CLI commands
- **Pros**:
  - Flexible querying
  - API-first design
  - Good for web UI
- **Cons**:
  - Scope creep - API design is separate epic
  - CLI users want CLI, not API
  - Adds complexity (GraphQL server, schema, resolvers)
  - AI orchestrator uses CLI, not API
- **Decision Rationale**: Keep epic focused on CLI commands; API can be added later if needed

---

## Future Epic Candidates

Features/capabilities that are natural follow-ons to this epic:

| Future Epic Concept | Priority | Dependency |
|---------------------|----------|------------|
| **E14: Analytics CLI Separation** | Medium | None - can be done independently |
| **E15: Multi-Team Workflow Support** | Low | Depends on E13 workflow foundation |
| **E16: Workflow Designer UI** | Low | Depends on E13 for backend logic |
| **E17: Webhook/Event System** | Medium | Depends on E13 for workflow transitions |
| **E18: Workflow Migration Tools** | Low | Depends on E13 + user feedback |
| **E19: Advanced Workflow Engine** | Very Low | Depends on E13 if simple workflows prove insufficient |

---

## Scope Change Process

If during implementation we identify a **Must Have** feature that's currently out of scope:

1. **Document the requirement** - Why is it critical? What breaks without it?
2. **Estimate effort** - How much time would it add?
3. **Assess impact** - Can we defer to next release? Is there a workaround?
4. **Get approval** - Stakeholder decision on scope expansion vs. deferral
5. **Update this document** - Move from Out of Scope to In Scope (or reject and document why)

**Guiding Principle**: Prefer deferral over scope expansion. Ship a focused, high-quality epic rather than a bloated, half-finished one.

---

*See also*: [Requirements](./requirements.md), [Success Metrics](./success-metrics.md)
