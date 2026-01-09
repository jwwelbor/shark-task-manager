# E07-F16 Workflow Improvements: CX Recommendations

## Quick Answers to Your Questions

### Q1: Is `shark task list status <status-name>` the right UX pattern?

**Answer: ✓ YES - Strongly Recommended**

Use **positional syntax** as primary interface, **keep flag as alias**:

```bash
# Primary (discoverable)
shark task list status ready-for-development

# Alias (backward compatible, for scripts)
shark task list --status=ready-for-development
```

**Why This Works**:
- Positional arguments are more discoverable (users see them in `--help`)
- Matches existing pattern: `shark task list E04 F01` (epic/feature filtering)
- Less reliance on flag memorization
- Case-insensitive for user friendliness

**Add Helpful Error for Invalid Status**:
```
$ shark task list status invalid_status
Error: Status 'invalid_status' not found

Available statuses:
  - draft
  - ready_for_refinement
  - in_refinement
  - ready_for_development
  ... (etc)

Use 'shark workflow list' to see all statuses with descriptions
```

---

### Q2: Is `shark task next-status <task-key>` the right command?

**Answer: ✓ YES - With Interactive Mode**

**Recommended Command Structure**:

```bash
# Interactive mode (PRIMARY - shows all options, user chooses)
shark task next-status E11-F15-001

# Non-interactive with explicit target (for scripts)
shark task next-status E11-F15-001 --status=ready_for_code_review

# Preview without changing (show options without transitioning)
shark task next-status E11-F15-001 --preview
```

**Why Interactive Mode Wins**:
- Users don't need to know workflow structure
- Educational (shows what transitions are valid)
- Prevents mistakes when multiple options exist
- Safe with explicit confirmation
- Reduces cognitive load

**Example Interactive Flow**:
```
$ shark task next-status E07-F20-001
Current status: in_development (phase: development)

Available transitions:
  1) ready_for_code_review (phase: review)
     "Code complete, awaiting review"
     Agents: tech-lead, code-reviewer

  2) ready_for_refinement (phase: planning)
     "Awaiting specification and analysis"
     Agents: business-analyst, architect

  3) blocked (phase: any)
     "Temporarily blocked by external dependency"

  4) on_hold (phase: any)
     "Intentionally paused"

Enter selection [1-4] or Ctrl+C: 1
✓ Transitioned: in_development → ready_for_code_review
```

---

### Q3: Should we support querying by phase?

**Answer: ✓ VALID NEED - But Phase 2 (Not MVP)**

**Reason**: Phase filtering is a secondary need
- Status filtering is more direct (users think in terms of specific statuses)
- Phase is higher-level abstraction that should be learned after status filtering
- Gather usage data from status filtering first before adding phase dimension

**Recommendation**:
- Implement status filtering in **E07-F16 (MVP)**
- Implement phase filtering in **E07-F17 (Phase 2)**
- Use same underlying index (status) for performance

**If Implementing Phase Filtering**:
```bash
shark task list phase development  # All tasks in development phase
shark task list phase review       # All tasks being reviewed
shark task list --phase=qa         # Flag syntax (backup)
```

---

### Q4: How to handle multiple possible next statuses?

**Answer: Interactive Selection (Shown Above)**

**Why NOT Smart Default**:
| Approach | Interactive | Smart Default |
|----------|-------------|---------------|
| Safety | ✓ Excellent | Fair (can guess wrong) |
| Clarity | ✓ Shows all options | Hidden heuristic |
| Education | ✓ Users learn workflow | Users don't know what happened |
| Automation | ✓ Use `--status` flag | Guess might be wrong |
| Speed | Good | Very fast |

**In Your Config**:
Many statuses have multiple transitions. Examples:
```
in_development      → [ready_for_code_review, ready_for_refinement, blocked, on_hold]
in_code_review      → [ready_for_qa, in_development, ready_for_refinement, on_hold]
in_qa               → [ready_for_approval, in_development, ready_for_refinement, blocked, on_hold]
ready_for_approval  → [completed, ready_for_qa, ready_for_development, ready_for_refinement, on_hold]
```

**Interactive mode is essential** to avoid confusion.

**For Automation/Scripts**:
```bash
# Use explicit --status when you know the target
shark task next-status E07-F20-001 --status=ready_for_code_review
```

---

### Q5: Should we show available next statuses in `shark task get`?

**Answer: ✓ YES - Strongly Recommended**

**Integration in Task Details Output**:

```bash
$ shark task get E07-F20-001

Key: E07-F20-001
Title: Implement JWT token validation
Status: in_development (phase: development)
  → Code implementation in progress
  → Color: yellow | Agents: developer, ai-coder

Priority: 5
Agent: backend
Created: 2026-01-08

Dependencies: (none)

Available Transitions:
  → ready_for_code_review
     "Code complete, awaiting review"
     [agents: tech-lead, code-reviewer]

  → ready_for_refinement
     "Awaiting specification and analysis"
     [agents: business-analyst, architect]

  → blocked
     "Temporarily blocked by external dependency"

  → on_hold
     "Intentionally paused"

Next Steps:
  • Advance status: shark task next-status E07-F20-001
  • View workflow: shark workflow list
  • See all transitions: shark workflow transitions in_development
```

**Benefits**:
- Reduces context switching (no need to run separate command)
- Educational (users learn workflow by seeing examples)
- Reduces time to next action
- Enables better decision-making

---

## Recommended Implementation Order

### E07-F16 MVP (Current Sprint)

1. **Status-Based Filtering**
   - Implement `shark task list status <name>`
   - Add `--status=<name>` flag as alias
   - Case-insensitive handling
   - Helpful error messages with available statuses

2. **Task Creation Fix**
   - Remove hardcoded `TaskStatusTodo` references
   - Use workflow config `special_statuses._start_[0]` for initial status
   - Ensures tasks created in correct workflow state

3. **Workflow Progression**
   - Implement `shark task next-status <task-key>` with interactive mode
   - Add `--status=<name>` for explicit non-interactive transitions
   - Add `--preview` to show available transitions

4. **Workflow Discovery**
   - Show available transitions in `shark task get` output
   - Add transitions listing to `shark workflow` command

### E07-F17 Phase 2 (Future Sprint)

5. **Phase-Based Filtering**
   - Implement `shark task list phase <name>`
   - Leverage existing status filtering infrastructure

6. **Enhancement**
   - Shell completion for status and phase names
   - Agent-aware filtering (`--agent=developer`)
   - Performance optimizations if needed

---

## Critical Issues to Fix First

Before implementation, address these issues:

1. **Task Creation Status**: Some code still hardcodes `todo` status
   - Current config expects tasks in `draft` or `ready_for_development`
   - Mismatch causes issues with workflow validation
   - Fix: Use `workflow.SpecialStatuses[StartStatusKey][0]` instead

2. **Hardcoded Status Constants**: Legacy references in workflow-aware code
   - Keep constants for backward compatibility
   - But remove from workflow-aware commands and creation logic
   - Update tests to use workflow config instead

---

## JSON Output Support

All new commands must support `--json`:

```bash
# Status filtering with JSON
shark task list status in_development --json
[
  {
    "key": "E07-F20-001",
    "title": "Implement JWT token validation",
    "status": "in_development",
    "priority": 5,
    ...
  }
]

# Task get with available transitions
shark task get E07-F20-001 --json
{
  "key": "E07-F20-001",
  "title": "Implement JWT token validation",
  "status": "in_development",
  "available_transitions": [
    {
      "status": "ready_for_code_review",
      "description": "Code complete, awaiting review",
      "phase": "review",
      "agent_types": ["tech-lead", "code-reviewer"]
    },
    ...
  ]
}

# Next status preview
shark task next-status E07-F20-001 --preview --json
{
  "current_status": "in_development",
  "available_transitions": [
    ...
  ]
}
```

---

## Backward Compatibility

All changes must be backward compatible:

- `shark task list --status=todo` continues to work
- Existing hardcoded commands (`shark task start`) still function
- Database schema unchanged
- Workflow config defaults to existing behavior

---

## Testing Checklist

- [ ] Valid status names filter correctly
- [ ] Invalid status names show helpful errors
- [ ] Case-insensitive status input (Draft, DRAFT, draft all work)
- [ ] `shark task next-status` shows all available transitions
- [ ] Multiple transitions: interactive mode works with numbered choices
- [ ] Single transition: advances immediately or asks confirmation
- [ ] Terminal status (completed): appropriate error message
- [ ] `--status` override works in next-status command
- [ ] `--preview` shows transitions without changing status
- [ ] `shark task get` displays available transitions
- [ ] All JSON output is valid and complete
- [ ] Backward compatibility with existing commands

---

## Document Location

**Full CX Design Review**: `/docs/WORKFLOW_UX_DESIGN_REVIEW.md`

This document contains:
- Deep dive on each requirement
- Edge case handling
- Risk mitigation strategies
- Success metrics
- Complete workflow configuration reference
- Comparison to alternative patterns
- Accessibility considerations

---

## Summary

Your proposed workflow improvements are **well-conceived and aligned with user needs**. The recommendations above optimize for:

1. **Discoverability**: Positional arguments visible in help text
2. **Safety**: Interactive mode prevents unintended transitions
3. **Simplicity**: Users don't memorize workflow structure
4. **Flexibility**: Scripts can use explicit `--status` flags
5. **Coherence**: Workflow progression feels natural and learnable

All five requirements are **valid and recommended for implementation**, with phase-based querying deferred to Phase 2 to keep MVP focused and gather usage data first.

