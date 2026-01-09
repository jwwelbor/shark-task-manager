# E07-F16: Workflow Improvements - CX Design Review

## Executive Summary

This document provides a comprehensive Customer Experience (CX) review of the proposed workflow improvements for E07-F16, including status-based querying, workflow progression commands, and workflow state discovery.

**Status**: CX Validation Complete
**Reviewer**: CX Designer Agent
**Date**: 2026-01-08

---

## 1. Current State Analysis

### Existing Workflow System

The codebase has a well-designed configurable workflow system already in place:

**Configuration** (`.sharkconfig.json`):
- `status_flow`: Maps current status to valid next statuses (e.g., `"in_development" → ["ready_for_code_review", "blocked", ...]`)
- `status_metadata`: Defines UI properties (color, description, phase, agent_types)
- `special_statuses._start_`: Entry points for new tasks (default: `["draft", "ready_for_development"]`)
- `special_statuses._complete_`: Terminal statuses (default: `["completed", "cancelled"]`)

**Current Task Creation Behavior**:
- New tasks are created using the first status in `special_statuses._start_`
- Current config has `"_start_": ["draft", "ready_for_development"]` (correctly workflow-aware)
- **Problem**: Some legacy code still references hardcoded `TaskStatusTodo` constants

**Current CLI Commands**:
- `shark task list --status=<status>`: Existing filter by single status
- `shark task start`, `task complete`, `task approve`: Hardcoded status transitions (not workflow-aware)
- No command for automatic workflow progression
- No way to discover available next statuses for a task

### Critical Issues Identified

1. **Status Creation Mismatch**: Tasks created in `draft` or `ready_for_development` (from config), but hardcoded transitions still reference `todo`
2. **Limited Status Filtering**: Only supports single `--status` filter, no phase-based filtering
3. **Hardcoded Transitions**: Commands like `task start`, `task complete` don't respect workflow config
4. **No Workflow Discovery**: Users can't see available transitions for a task before acting
5. **No Automatic Progression**: No command to advance task through workflow without knowing next status

---

## 2. User Experience Requirements Validation

### Requirement 1: Status-Based Querying

**User Need**: "List all tasks in a specific workflow status"

**Proposed Command**:
```bash
shark task list status ready-for-development
shark task list --status=ready-for-development  # Flag variant
```

**CX Validation**: ✓ VALID & RECOMMENDED

**Rationale**:
- Aligns with existing `--status` filter in `task list`
- Pattern matches other list commands (`shark epic list`, `shark feature list`)
- Positional argument approach (`list status <name>`) is more discoverable than flags alone
- Supports quick filtering for next batch of work

**UX Recommendations**:
1. **Positional Argument Over Flag**: Use positional syntax for primary discovery
   - Primary: `shark task list status ready-for-development` (discoverable)
   - Alias: `shark task list --status=ready-for-development` (backward compatible)
   - Both should work identically

2. **Case Insensitivity**: Accept status names in any case
   - `shark task list status ready-for-development`
   - `shark task list status READY_FOR_DEVELOPMENT`
   - `shark task list status Ready For Development`
   - Normalize to snake_case internally

3. **Tab Completion**: Enable shell completion for status names
   - Users can discover available statuses via tab completion
   - Reduces memorization burden

4. **Validation Output**: Show available statuses if user provides invalid name
   ```
   $ shark task list status invalid
   Error: Status 'invalid' not found in workflow
   Available statuses:
     - draft
     - ready_for_development
     - in_development
     - ready_for_code_review
     - in_code_review
     - ready_for_qa
     - in_qa
     - ready_for_approval
     - in_approval
     - completed
     - cancelled
     - blocked
     - on_hold
   ```

---

### Requirement 2: Workflow Progression Command

**User Need**: "Automatically advance task to next status without knowing what comes next"

**Proposed Command**:
```bash
shark task next-status E11-F15-001
```

**CX Validation**: ✓ VALID & RECOMMENDED with modifications

**Rationale**:
- Reduces cognitive load - users don't need to know workflow structure
- Supports common workflow progression patterns
- Enables fast task status updates for users in flow state

**Critical Challenge - Multiple Possible Transitions**:

The workflow config shows many statuses have **multiple valid next states**:
```
in_development → [ready_for_code_review, ready_for_refinement, blocked, on_hold]
in_approval    → [completed, ready_for_qa, ready_for_development, ready_for_refinement, on_hold]
in_code_review → [ready_for_qa, in_development, ready_for_refinement, on_hold]
```

**This is a UX Decision Point**: When multiple transitions are available, what should the command do?

**Option A: Interactive Selection (Recommended)**
```bash
$ shark task next-status E07-F20-001
Current status: in_development
Available transitions:
  1) ready_for_code_review - "Code complete, awaiting review" [agents: tech-lead, code-reviewer]
  2) ready_for_refinement  - "Awaiting specification and analysis" [agents: business-analyst, architect]
  3) blocked               - "Temporarily blocked by external dependency"
  4) on_hold               - "Intentionally paused"
Which status? [1-4]: 1
→ Task E07-F20-001 transitioned: in_development → ready_for_code_review
```

**Pros**:
- Never fails due to ambiguity
- Educational - shows users workflow structure
- Respects workflow design intent
- Safe - explicit user confirmation

**Cons**:
- Not suitable for scripts/automation (can use `--status` flag instead)
- Adds friction for terminal-based workflows

**Option B: Smart Default + Override (Alternative)**
```bash
$ shark task next-status E07-F20-001
# Heuristic: Use first terminal transition (toward completion)
→ Task E07-F20-001 transitioned: in_development → ready_for_code_review

$ shark task next-status E07-F20-001 --to=blocked
→ Task E07-F20-001 transitioned: in_development → blocked

$ shark task next-status E07-F20-001 --auto
# Non-interactive for scripts
```

**Pros**:
- Fast for common case (most workflows have "happy path")
- Scriptable with `--auto` flag
- Backward compatible with flag override

**Cons**:
- Heuristic might not match user intent
- Less educational
- Risk of unintended transitions

**CX Recommendation**: **Option A (Interactive)** is superior

Rationale:
- Task workflow progression is not high-frequency action (unlike code completion)
- Cognitive benefit of understanding workflow outweighs friction
- Interactive mode with `--status=<name>` escape hatch provides both safety and speed
- Aligns with Shark's philosophy of explicit, audited state transitions

**Recommended Command Structure**:
```go
// Interactive mode (primary)
shark task next-status E07-F20-001

// Non-interactive with explicit target (for scripts/automation)
shark task next-status E07-F20-001 --status=ready_for_code_review

// Show available transitions without changing
shark task next-status E07-F20-001 --preview
```

---

### Requirement 3: Phase-Based Querying

**User Need**: "Filter tasks by workflow phase (development, review, qa, etc.)"

**Proposed Command**:
```bash
shark task list phase development
shark task list phase review
```

**CX Validation**: ✓ VALID BUT SECONDARY NEED

**Rationale**:
- Provides higher-level task organization than individual statuses
- Useful for team coordination ("what's in review?")
- Phases exist in workflow metadata (planning, development, review, qa, approval, done)

**UX Considerations**:
1. **Not Primary Interaction**: Status filtering is more direct - keep as primary
2. **Phase Ambiguity**: Some statuses have `phase: "any"` (blocked, on_hold)
   - These appear in all phase filters? Exclude? Document behavior
3. **Multiple Phases**: Some tasks will be in multiple phases during transitions
   - Need clear semantics: "current phase only" vs "all matching"

**Recommendation**: Implement as **secondary feature**, not part of E07-F16 MVP
- Reason: Phase filtering is a "nice-to-have" for power users
- E07-F16 should focus on status filtering (primary need)
- Add phase filtering in E07-F17 after gathering actual usage patterns

**If Implementing Phase Filtering**:
```bash
# Show tasks in development phase
shark task list --phase=development

# Combined filters
shark task list E04 --phase=development  # Epic + phase

# Phase names (from metadata)
shark task list --list-phases  # Show available phases
```

---

## 3. Command Syntax Patterns

### Consistency Analysis

Current Shark command patterns:

| Command | Syntax | Example |
|---------|--------|---------|
| Epic list | `list` | `shark epic list` |
| Feature list | `list [EPIC]` | `shark feature list E04` |
| Task list | `list [EPIC] [FEATURE]` | `shark task list E04 F01` |
| Task by status | `list --status=<s>` | `shark task list --status=todo` |

### Recommended Pattern for New Commands

**Pattern: Positional + Optional Filters**

```bash
# Status filtering (new)
shark task list status <status-name>      # Primary
shark task list --status=<status-name>    # Alias (backward compatible)

# Phase filtering (future)
shark task list phase <phase-name>        # Primary
shark task list --phase=<phase-name>      # Alias (future)

# Workflow progression (new)
shark task next-status <task-key>         # Interactive by default
shark task next-status <task-key> --status=<s>  # Explicit transition
shark task next-status <task-key> --preview     # Show available transitions
```

**Rationale**:
- Positional arguments are more discoverable via `shark task list --help`
- Reduces reliance on memorizing flag names
- Consistent with existing patterns (`shark task list EPIC FEATURE`)
- Flags still work for power users and scripts

---

## 4. Workflow Discovery - Best Practices

### Show Available Next Statuses When Needed

**Use Case**: User needs to know what transitions are available

**Recommended Approach - Show in Task Details**:
```bash
$ shark task get E07-F20-001
Key:          E07-F20-001
Title:        Implement JWT token validation
Status:       in_development (phase: development)
  Description: Code implementation in progress
  Color:       yellow
  Agents:      developer, ai-coder

Available Transitions:
  → ready_for_code_review  "Code complete, awaiting review"
                           [agents: tech-lead, code-reviewer]
  → ready_for_refinement   "Awaiting specification and analysis"
                           [agents: business-analyst, architect]
  → blocked                "Temporarily blocked by external dependency"
  → on_hold                "Intentionally paused"

Blocked? Use: shark task block E07-F20-001 --reason="..."
Advance status? Use: shark task next-status E07-F20-001
```

**Command for Viewing Workflow State**:
```bash
# Show complete workflow diagram
shark workflow list
shark workflow list --json

# Show transitions for specific status
shark workflow transitions in_development
shark workflow transitions in_development --json
```

---

## 5. Task Creation Status Fix

### Issue: Hardcoded Status vs. Workflow Config

**Current Problem**:
- Config defines `special_statuses._start_: ["draft", "ready_for_development"]`
- Creator.go correctly uses first start status
- But legacy hardcoded transitions still expect `todo`
- This causes inconsistency

**Fix Required**:
1. Update all `TaskStatusTodo` references in workflow-aware code
2. Use `workflow.SpecialStatuses[StartStatusKey][0]` for initial status
3. Remove hardcoded status constants from workflow-aware code
4. Keep constants only for backward compatibility/validation

**Impact**:
- Tasks created in correct workflow status
- Eliminates gap between creation status and workflow config
- Supports custom workflows with different entry points

---

## 6. Journey Flow Coherence

### Task Status Progression Journey

```
Creation
   ↓
initial_status (from _start_ config)
   ↓
intermediate_statuses (workflow-specific)
   ↓
terminal_status (from _complete_ config)
```

**Example Journey - Current Config**:
```
CREATE TASK
   ↓
draft [ready_for_refinement, cancelled, on_hold]
   ↓
ready_for_refinement [in_refinement, cancelled, on_hold]
   ↓
in_refinement [ready_for_development, draft, blocked, on_hold]
   ↓
ready_for_development [in_development, ready_for_refinement, cancelled, on_hold]
   ↓
in_development [ready_for_code_review, ready_for_refinement, blocked, on_hold]
   ↓
ready_for_code_review [in_code_review, in_development, on_hold]
   ↓
in_code_review [ready_for_qa, in_development, ready_for_refinement, on_hold]
   ↓
ready_for_qa [in_qa, on_hold]
   ↓
in_qa [ready_for_approval, in_development, ready_for_refinement, blocked, on_hold]
   ↓
ready_for_approval [in_approval, on_hold]
   ↓
in_approval [completed, ready_for_qa, ready_for_development, ready_for_refinement, on_hold]
   ↓
completed (TERMINAL)
```

**Coherence Validation**: ✓ PASS

- All states are reachable from `_start_` statuses
- All states have path to `_complete_` statuses
- Error recovery paths exist (can return to earlier stages)
- Flexibility for interruptions (blocked, on_hold, cancelled)
- Phase progression is logical

**Emotion Arc**: ✓ APPROPRIATE

- Clear progress feeling (status moves toward completion)
- Escape hatches prevent user frustration (blocked, on_hold)
- Not overly complex (13 states, but each has clear purpose)

---

## 7. Accessibility & Inclusivity

### Status Naming Conventions

**Current Status Names**: Use snake_case, descriptive
- ✓ Clear intent (e.g., "in_development", not "dev" or "wip")
- ✓ Consistent naming pattern
- ✓ Self-documenting

**Color Accessibility**:
- Verify in `status_metadata.color` values
- Colors present in metadata but should be tested for WCAG AA contrast
- Recommendation: Always show status name in addition to color in CLI output

**Agent Targeting**:
- `status_metadata[status].agent_types` helps task routing
- Filters like `shark task next --agent=developer` enable agent-specific views

### Status Names for Non-Native English Speakers

**Current**: All statuses in English (standard for technical tools)

**Recommendation**: Keep as-is
- Rationale: Status names are identifiers in config and database
- Not a UX barrier - config is admin-level setting
- CLI can support localized descriptions if needed later

---

## 8. Edge Cases & Error Handling

### Case 1: Invalid Status in Filter

**Scenario**: User types `shark task list status draft_review` (typo)

**Handling**:
```
Error: Status 'draft_review' not found in workflow configuration

Available statuses:
  draft, ready_for_refinement, in_refinement, ready_for_development,
  in_development, ready_for_code_review, in_code_review, ready_for_qa,
  in_qa, ready_for_approval, in_approval, completed, cancelled, blocked, on_hold

Tip: Use 'shark workflow list' to see complete workflow with descriptions
     Use 'shark task list --status=<TAB>' for shell completion
```

### Case 2: Multiple Possible Transitions

**Scenario**: User runs `shark task next-status E07-F20-001` when 4 options exist

**Handling** (Interactive Mode - Recommended):
```
Current status: in_development

Available transitions:
  1) ready_for_code_review (phase: review)
     Code complete, awaiting review
     Agents: tech-lead, code-reviewer

  2) ready_for_refinement (phase: planning)
     Awaiting specification and analysis
     Agents: business-analyst, architect

  3) blocked (phase: any)
     Temporarily blocked by external dependency

  4) on_hold (phase: any)
     Intentionally paused

Enter selection [1-4] or press Ctrl+C to cancel: 1
Transitioned: in_development → ready_for_code_review
```

### Case 3: No Available Transitions (Terminal Status)

**Scenario**: Task is in `completed` status (terminal)

**Handling**:
```
Error: Task E07-F20-001 is in 'completed' status (terminal)

This task cannot be transitioned further. Completed tasks are final.

Options:
  • View task: shark task get E07-F20-001
  • Reopen task: (not available via next-status - explicit command needed)
```

### Case 4: Workflow Mismatch

**Scenario**: Task has status that's not in current workflow config

**Handling**:
```
Warning: Task E07-F20-001 has status 'legacy_status' which is not defined in current workflow.

This can happen if:
  • Workflow configuration was recently changed
  • Task was imported from different system
  • Database is out of sync with config

Options:
  • Update workflow: edit .sharkconfig.json to include 'legacy_status'
  • Migrate task: shark task update E07-F20-001 --status=draft
  • View current workflow: shark workflow list
```

---

## 9. Comparison to Alternative Patterns

### Alternative 1: Subcommand for Status Filtering

```bash
shark task status list E04  # NOT recommended
shark task status list-by-phase development  # NOT recommended
```

**Why Not Recommended**:
- Adds extra command level (less discoverable)
- Inconsistent with existing `shark task list --status=...`
- More characters to type

### Alternative 2: Multiple Shorthand Commands

```bash
shark in-development   # List tasks in development status - NOT recommended
shark ready-for-qa     # List tasks ready for QA
shark blocked          # List blocked tasks
```

**Why Not Recommended**:
- Creates too many commands
- Hard to discover
- Inconsistent with existing pattern
- Not extendable for new statuses

### Alternative 3: Configuration-Based Shortcuts

```bash
shark task list @development  # Uses phase/phase-based shortcut from config
shark task list #in-progress  # Uses status shortcut
```

**Why Not Recommended**:
- Non-obvious syntax
- Requires config knowledge
- Not accessible to new users

---

## 10. Recommended Implementation Order

### Phase 1 (E07-F16 MVP)

**Highest Priority - Status Filtering**
1. ✓ Implement positional `shark task list status <name>` syntax
2. ✓ Add case-insensitive status name handling
3. ✓ Show error with available statuses for invalid input
4. ✓ Update task creation to respect workflow config

**Medium Priority - Workflow Progression**
5. ✓ Implement `shark task next-status <task-key>` with interactive mode
6. ✓ Add `--status=<name>` override for non-interactive use
7. ✓ Add `--preview` flag to show available transitions

**Enhancement - Task Discovery**
8. ✓ Show available transitions in `shark task get <task-key>` output
9. ✓ Add transitions listing to workflow command: `shark workflow transitions <status>`

### Phase 2 (E07-F17 - Future)

10. Phase-based filtering: `shark task list phase <phase>`
11. Agent-aware filtering: `shark task list --agent=developer`
12. Shell completion support for status and phase names

### Phase 3 (E07-F18 - Future)

13. Workflow templates for common patterns
14. Custom workflow import/export
15. Workflow migration tools

---

## 11. Recommended Response to User Questions

### Question 1: Is `shark task list status <status-name>` the right UX pattern?

**Answer**: ✓ YES - Strongly Recommended

**Rationale**:
- Positional arguments are more discoverable than flags alone
- Consistent with existing `shark task list E04 F01` pattern
- "status" is a well-understood concept in task management
- Case-insensitive handling removes mental burden
- Can keep `--status=<name>` as alias for backward compatibility

**Implementation**:
```bash
# Primary (discoverable)
shark task list status ready-for-development
shark task list status READY_FOR_DEVELOPMENT

# Alias (backward compatible)
shark task list --status=ready-for-development

# Both should work identically
```

---

### Question 2: Is `shark task next-status <task-key>` the right command?

**Answer**: ✓ YES - With Interactive Mode Strongly Recommended

**Rationale**:
- Reduces cognitive load (users don't memorize workflow)
- Prevents unintended transitions when multiple options exist
- Educational (teaches users the workflow)
- Safe with explicit confirmation

**Command Structure**:
```bash
# Interactive (primary, recommended)
shark task next-status E07-F20-001

# Non-interactive override
shark task next-status E07-F20-001 --status=ready_for_code_review

# Preview without changing
shark task next-status E07-F20-001 --preview
```

**When Multiple Transitions Exist** (Recommended Behavior):
```
Current: in_development
Available:
  1) ready_for_code_review
  2) ready_for_refinement
  3) blocked
  4) on_hold
Choose [1-4]:
```

---

### Question 3: Should we support querying by phase?

**Answer**: ✓ VALID NEED, But Not MVP

**Rationale**:
- Useful for high-level team coordination
- Helps understand task distribution across workflow phases
- Secondary to status filtering (status is more direct)

**Recommendation**:
- Implement in Phase 2 (E07-F17)
- Gather usage data from status filtering first
- Don't add complexity prematurely

**If Implementing**:
```bash
shark task list phase development  # All tasks in development phase
shark task list phase review       # All tasks being reviewed
shark task list phase qa           # All QA-related tasks
```

---

### Question 4: How to handle multiple possible next statuses?

**Answer**: Interactive Selection is Superior

**Why Interactive Mode Wins**:

| Criteria | Interactive | Smart Default | Heuristic |
|----------|-------------|---------------|-----------|
| Safety | Excellent | Good | Fair |
| Discoverability | Excellent | Fair | Poor |
| Educational | Excellent | Fair | Poor |
| Speed | Good | Excellent | Excellent |
| Automation | Fair (use --status flag) | Good | Good |
| Error Prevention | Excellent | Fair | Poor |

**Recommended Approach**:
```bash
# Interactive (shows all options, user chooses)
shark task next-status E07-F20-001

# Explicit (for scripts/automation)
shark task next-status E07-F20-001 --status=ready_for_code_review

# Preview (show options without changing)
shark task next-status E07-F20-001 --preview
```

This covers all use cases while prioritizing safety and education.

---

### Question 5: Should we show available next statuses in `task get`?

**Answer**: ✓ YES - Strongly Recommended

**Benefits**:
- Reduces need to run separate command
- Educational (users learn workflow)
- Reduces context switching
- Enables faster decision-making

**Recommended Output**:
```
$ shark task get E07-F20-001

Key: E07-F20-001
Title: Implement JWT token validation
Status: in_development (phase: development)

Description: Code implementation in progress
Priority: 5
Agent: backend
Created: 2026-01-08 14:22:00

Available Transitions:
  → ready_for_code_review  "Code complete, awaiting review"
                           [agents: tech-lead, code-reviewer]
  → ready_for_refinement   "Awaiting specification"
  → blocked                "Blocked by external issue"
  → on_hold                "Paused"

Next Steps:
  • Advance status: shark task next-status E07-F20-001
  • View workflow: shark workflow list
  • See all transitions: shark workflow transitions in_development
```

---

## 12. Implementation Validation Checklist

### Functional Requirements
- [ ] `shark task list status <name>` filters by workflow status
- [ ] `shark task list --status=<name>` works as alias
- [ ] Case-insensitive status name handling
- [ ] Invalid status shows helpful error with available options
- [ ] `shark task next-status <task-key>` shows available transitions
- [ ] Multiple transitions: interactive selection or `--status=<name>` override
- [ ] Terminal status (completed, cancelled): proper error handling
- [ ] Tasks created in correct workflow entry status (not hardcoded "todo")

### UX Requirements
- [ ] Help text shows examples for both status-based and phase-based queries
- [ ] `--help` output mentions positional arguments
- [ ] Command completes successfully with `--json` output
- [ ] Status names discoverable via shell completion or `--help`
- [ ] Error messages are actionable and show available options
- [ ] Interactive mode is user-friendly (numbered options, clear prompts)

### Documentation
- [ ] README updated with workflow examples
- [ ] CLI_REFERENCE.md shows new commands with examples
- [ ] WORKFLOW_GUIDE.md documents status transitions
- [ ] Help text examples include common filtering patterns

### Testing
- [ ] Test valid status names (all statuses in workflow)
- [ ] Test invalid status names (show helpful error)
- [ ] Test case-insensitive input (Draft, DRAFT, draft)
- [ ] Test task get shows available transitions
- [ ] Test next-status with single vs. multiple transitions
- [ ] Test next-status with terminal status (error handling)
- [ ] Test JSON output format for filtering and transitions
- [ ] Test backward compatibility with `--status` flag

---

## 13. Risk Mitigation

### Risk: Users Confused by Multiple Status Options

**Mitigation**:
- Interactive mode by default (forces awareness)
- Show agent types for each transition
- Clear descriptions from metadata
- Workflow visualization available (`shark workflow list`)

### Risk: Workflow Config Becomes Out of Sync

**Mitigation**:
- Validation command: `shark workflow validate`
- Database migration if status removed from workflow
- Warning when task has status not in current workflow
- Clear error messages guide resolution

### Risk: Breaking Change for Users with Custom Workflows

**Mitigation**:
- Backward compatible: `--status` flag still works
- Support both positional and flag syntax
- Default workflow unchanged from current behavior
- Migration guide for custom workflows

### Risk: Performance Impact on Large Task Lists

**Mitigation**:
- Status filtering at repository layer (uses database index)
- Phase filtering can use same index as status
- No additional queries needed beyond existing pattern
- Pagination support for large result sets (future)

---

## 14. Success Metrics

### Adoption Metrics
- Measure usage of `shark task list status <name>` vs. `--status` flag
- Track `shark task next-status` usage rate
- Monitor help command hits for workflow/status documentation

### Quality Metrics
- Error rate for invalid status names (should be low after helpful error messages)
- Time to complete task status update workflow
- User feedback on command discoverability

### Workflow Metrics
- Average task progression speed (time in each status)
- Frequency of workflow deviations (blocked, on_hold, back to earlier stage)
- Task completion rate by phase

---

## 15. Conclusion

### Summary of Recommendations

1. **Status-Based Querying** ✓ APPROVED
   - Implement both positional (`status <name>`) and flag (`--status=<name>`) syntax
   - Add case-insensitive handling and helpful error messages
   - Provide shell completion support

2. **Workflow Progression** ✓ APPROVED
   - Implement `shark task next-status <task-key>` with interactive mode
   - Show all available transitions to user
   - Provide `--status=<name>` override for non-interactive use
   - Add `--preview` to show transitions without changing

3. **Phase-Based Querying** ✓ APPROVED FOR FUTURE
   - Valid need but secondary priority
   - Implement in E07-F17 after gathering usage data
   - Will leverage same status filtering infrastructure

4. **Task Creation Fix** ✓ APPROVED
   - Remove hardcoded status references
   - Respect workflow config `special_statuses._start_` for task creation
   - Ensures consistency between config and actual task creation

5. **Workflow Discovery** ✓ APPROVED
   - Show available transitions in `shark task get`
   - Add `shark workflow transitions <status>` command
   - Make workflow visible and learnable

### UX Philosophy

These improvements follow Shark's core principles:
- **Explicit**: No hidden state transitions; users see all options
- **Safe**: Require confirmation for state changes; prevent errors
- **Educational**: Show users the workflow; help them learn
- **Flexible**: Support both interactive and scripted usage
- **Configurable**: Workflow driven by config, not hardcoded logic

The recommendations prioritize **safety and discoverability** over maximum speed, recognizing that task status changes are not high-frequency operations and the cognitive benefit of understanding workflow outweighs the friction of interactive selection.

---

## Appendix A: Current Workflow Configuration

```json
{
  "status_flow_version": "1.0",
  "special_statuses": {
    "_start_": ["draft", "ready_for_development"],
    "_complete_": ["completed", "cancelled"]
  },
  "status_metadata": {
    "draft": {
      "color": "gray",
      "description": "Task created but not yet refined",
      "phase": "planning",
      "agent_types": ["business-analyst", "architect"]
    },
    "ready_for_refinement": {
      "color": "cyan",
      "description": "Awaiting specification and analysis",
      "phase": "planning",
      "agent_types": ["business-analyst", "architect"]
    },
    "in_refinement": {
      "color": "blue",
      "description": "Being analyzed and specified",
      "phase": "planning",
      "agent_types": ["business-analyst", "architect"]
    },
    "ready_for_development": {
      "color": "yellow",
      "description": "Spec complete, ready for implementation",
      "phase": "development",
      "agent_types": ["developer", "ai-coder"]
    },
    "in_development": {
      "color": "yellow",
      "description": "Code implementation in progress",
      "phase": "development",
      "agent_types": ["developer", "ai-coder"]
    },
    "ready_for_code_review": {
      "color": "magenta",
      "description": "Code complete, awaiting review",
      "phase": "review",
      "agent_types": ["tech-lead", "code-reviewer"]
    },
    "in_code_review": {
      "color": "magenta",
      "description": "Under code review",
      "phase": "review",
      "agent_types": ["tech-lead", "code-reviewer"]
    },
    "ready_for_qa": {
      "color": "green",
      "description": "Ready for quality assurance testing",
      "phase": "qa",
      "agent_types": ["qa", "test-engineer"]
    },
    "in_qa": {
      "color": "green",
      "description": "Being tested",
      "phase": "qa",
      "agent_types": ["qa", "test-engineer"]
    },
    "ready_for_approval": {
      "color": "purple",
      "description": "Awaiting final approval",
      "phase": "approval",
      "agent_types": ["product-manager", "client"]
    },
    "in_approval": {
      "color": "purple",
      "description": "Under final review",
      "phase": "approval",
      "agent_types": ["product-manager", "client"]
    },
    "completed": {
      "color": "white",
      "description": "Task finished and approved",
      "phase": "done",
      "agent_types": []
    },
    "cancelled": {
      "color": "gray",
      "description": "Task abandoned or deprecated",
      "phase": "done",
      "agent_types": []
    },
    "blocked": {
      "color": "red",
      "description": "Temporarily blocked by external dependency",
      "phase": "any",
      "agent_types": []
    },
    "on_hold": {
      "color": "orange",
      "description": "Intentionally paused",
      "phase": "any",
      "agent_types": []
    }
  },
  "status_flow": {
    "draft": ["ready_for_refinement", "cancelled", "on_hold"],
    "ready_for_refinement": ["in_refinement", "cancelled", "on_hold"],
    "in_refinement": ["ready_for_development", "draft", "blocked", "on_hold"],
    "ready_for_development": ["in_development", "ready_for_refinement", "cancelled", "on_hold"],
    "in_development": ["ready_for_code_review", "ready_for_refinement", "blocked", "on_hold"],
    "ready_for_code_review": ["in_code_review", "in_development", "on_hold"],
    "in_code_review": ["ready_for_qa", "in_development", "ready_for_refinement", "on_hold"],
    "ready_for_qa": ["in_qa", "on_hold"],
    "in_qa": ["ready_for_approval", "in_development", "ready_for_refinement", "blocked", "on_hold"],
    "ready_for_approval": ["in_approval", "on_hold"],
    "in_approval": ["completed", "ready_for_qa", "ready_for_development", "ready_for_refinement", "on_hold"],
    "completed": [],
    "cancelled": [],
    "blocked": ["ready_for_development", "ready_for_refinement", "cancelled"],
    "on_hold": ["ready_for_refinement", "ready_for_development", "cancelled"]
  }
}
```

---

## Document Information

- **Author**: CX Designer Agent
- **Date**: 2026-01-08
- **Status**: Complete - Ready for Implementation
- **Next Steps**: Share with team for feedback before development sprint
- **Related Epic**: E07 (CLI Improvements)
- **Related Feature**: E07-F16 (Workflow Improvements)

