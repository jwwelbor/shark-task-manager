---
inputs:
  # This reference is loaded by the assessment skill during scope_validation mode.
  # It expects the caller (SKILL.md craft) to have:
  #   - work_item_description (string) — what the candidate work is
  #   - estimated_task_count (integer, optional)
  #   - estimated_loc (integer, optional)
  #   - estimated_duration_days (number, optional)
  # No shark CLI is invoked from inside this reference.
outputs:
  # classification (FEATURE | TASK | SPLIT | CONSOLIDATE) with rationale.
---

# Scope Validation — Feature vs Task Classification

Complete criteria for determining whether work belongs at the feature level or task level.

## Core Principle

**Features represent user-facing value. Tasks represent implementation steps.**

- **Feature**: Delivers standalone value to users/stakeholders
- **Task**: Technical step toward delivering a feature

## Classification Decision Tree

```
Is the work...
│
├─ User-visible or stakeholder-visible?
│  └─ YES → Likely a FEATURE
│
├─ Requires multiple implementation steps?
│  └─ YES → Likely a FEATURE
│
├─ Crosses multiple domains/layers?
│  └─ YES → Likely a FEATURE
│
├─ A single technical operation?
│  └─ YES → Likely a TASK
│
└─ Part of implementing a larger feature?
   └─ YES → Likely a TASK
```

## Feature Classification Criteria

### Minimum Feature Requirements

A feature MUST satisfy at least ONE of these:

1. **User-Facing Value**: Changes visible/usable by end users
2. **Stakeholder Deliverable**: Reportable progress to product owner/business
3. **Integration Point**: Requires coordination across multiple systems/teams
4. **Multi-Step Implementation**: Breaks down into 2+ meaningful tasks

### Feature Size Thresholds

| Metric | Too Small (Task) | Right Size (Feature) | Too Large (Split) |
|--------|------------------|---------------------|-------------------|
| Implementation tasks | 1 task | 2-15 tasks | 15+ tasks |
| Lines of code (new) | <100 | 100-2000 | 2000+ |
| Lines of code (refactor) | <500 | 500-5000 | 5000+ |
| Time estimate | <1 day | 1-10 days | 10+ days |
| Complexity score | 0-3 | 4-15 | 16+ |
| Developer handoffs | 0 | 1-3 | 3+ |

**Note**: These are guidelines, not hard rules. Context matters.

## Task Classification Criteria

### Characteristics of Tasks

A task is:

- **Atomic**: Single cohesive unit of work
- **Assignable**: One developer can complete it
- **Testable**: Clear acceptance criteria, can be verified
- **Completable**: Finite scope, not open-ended
- **Sequential**: Has clear dependencies (can be ordered)

### Task Size Thresholds

| Metric | Too Small (Consolidate) | Right Size (Task) | Too Large (Split) |
|--------|------------------------|-------------------|-------------------|
| Lines of code | <20 | 20-500 | 500+ |
| Time estimate | <2 hours | 2 hours - 2 days | 2+ days |
| Acceptance criteria | 1 | 2-5 | 5+ |
| Files touched | 0-1 | 2-10 | 10+ |

## Common Classification Scenarios

### Scenario 1: Database Schema Change

**Work**: "Add user_preferences table"

**Analysis**:

- User-facing? YES (users can save preferences)
- Multi-step? YES (schema, migration, API, UI)
- Integration? YES (backend + frontend + database)

**Classification**: FEATURE

**Tasks**:

1. Design schema for user_preferences table
2. Create migration script
3. Add repository methods for preferences
4. Create API endpoints for preferences CRUD
5. Update frontend to use preferences API
6. Write tests for preferences feature

---

### Scenario 2: Refactoring

**Work**: "Extract validation logic into separate service"

**Analysis**:

- User-facing? NO (internal refactoring)
- Multi-step? DEPENDS (if affects many files, YES)
- Integration? DEPENDS (if crosses layers, YES)

**Classification**:

- If <5 files, behavior-preserving, <1 day → TASK
- If 5+ files, multiple layers, 1-3 days → FEATURE

**Threshold**: Use complexity triage to decide.

---

### Scenario 3: Bug Fix

**Work**: "Fix login redirect after authentication"

**Analysis**:

- User-facing? YES (users affected)
- Multi-step? DEPENDS (simple fix vs architectural issue)

**Classification**:

- Simple fix (<1 hour, 1 file) → TASK
- Complex fix (multiple files, investigation required) → FEATURE

**Rule of thumb**: If fix requires investigation + design + implementation → FEATURE.

---

### Scenario 4: New API Endpoint

**Work**: "Add GET /api/users/:id/tasks endpoint"

**Analysis**:

- User-facing? YES (if consumed by frontend)
- Multi-step? YES (route + handler + service + repository + tests)
- Integration? YES (API + business logic + database)

**Classification**: FEATURE (even if "simple")

**Why**: Endpoint is a user-facing integration point.

---

### Scenario 5: Configuration Change

**Work**: "Update production database connection string"

**Analysis**:

- User-facing? NO
- Multi-step? NO
- Integration? NO

**Classification**: TASK (operational change, not development)

**Note**: Some operational changes don't belong in the project tracker at all (use ops runbook instead).

---

### Scenario 6: Documentation

**Work**: "Document API authentication flow"

**Analysis**:

- User-facing? YES (developers are users)
- Multi-step? DEPENDS (simple vs comprehensive)

**Classification**:

- Single doc, <1 day → TASK
- Multiple docs, diagrams, examples → FEATURE

---

### Scenario 7: Service Layer Refactoring

**Work**: "Create TaskService and extract business logic from CLI commands"

**Analysis**:

- User-facing? NO (internal architecture)
- Multi-step? YES (12 tasks)
- Integration? YES (CLI + service + repository)
- Lines to refactor? ~7000
- Time estimate? 3-4 weeks

**Classification**: FEATURE (despite being internal)

**Why**: Massive scope, high complexity (~15/27), behavior-preserving requirement.

**Could it be split?**: YES — by user-value layer:

- Core operations (get, list)
- Lifecycle operations (start, complete, approve)
- Advanced operations (block, dependencies)

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Feature Too Small

**Problem**: "Add loading spinner" as a feature.

**Why wrong**: Single UI change, <1 hour, 1 file.

**Fix**: Make it a task under "Improve UX feedback" feature.

---

### Anti-Pattern 2: Task Too Large

**Problem**: "Implement user authentication" as a task.

**Why wrong**:

- Multi-step (login, logout, session, tokens)
- Multiple files (UI, API, database)
- Multiple days of work

**Fix**: Make it a feature with tasks:

1. Design authentication flow
2. Implement JWT token generation
3. Create login endpoint
4. Create logout endpoint
5. Add session management
6. Update frontend to use auth

---

### Anti-Pattern 3: Consolidating Too Much

**Problem**: Combining "Email notifications" + "SMS notifications" + "Push notifications" into one feature.

**Why wrong**: Each is independently valuable, different implementation.

**Fix**: Three separate features (unless they share significant infrastructure).

**Exception**: If building unified notification system first, then implementations as tasks within that feature.

---

### Anti-Pattern 4: Splitting Too Much

**Problem**: Making "Write unit test for login" a separate feature from "Implement login".

**Why wrong**: Tests are part of implementing the feature, not separate value.

**Fix**: Tests are tasks within the login feature.

---

## Consolidation Guidelines

### When to Consolidate Tasks into Feature

Consolidate when:

- Tasks share same user-facing goal
- Tasks must be deployed together
- Tasks are sequential dependencies
- Individually, tasks have no stakeholder value

**Example — GOOD consolidation**:

- Tasks: "Create database table" + "Add API endpoint" + "Update UI"
- Feature: "User profile editing"
- Why: Each task alone delivers no value; feature delivers complete capability.

### When NOT to Consolidate

Keep separate when:

- Each provides independent value
- Can be deployed independently
- Different stakeholders care about each
- Parallel development possible

**Example — BAD consolidation**:

- Feature: "Improve performance"
- Tasks: "Optimize database queries" + "Add caching" + "Reduce bundle size"
- Why wrong: Each is independently valuable, different domains.
- Fix: Three separate features.

---

### When to Consolidate Features into Epic

Consolidate features when they share:

1. **Business objective**: Same quarterly goal or initiative
2. **User journey**: Same end-to-end user flow
3. **Technical foundation**: Shared architecture/infrastructure
4. **Timeline**: Released together as coherent product update

**Example — GOOD epic consolidation**:

- Epic: "E-commerce checkout flow"
- Features: "Shopping cart", "Payment processing", "Order confirmation"
- Why: Together they form a complete user journey.

### When NOT to Consolidate Features

Keep features in separate epics when:

- Different business objectives
- Can be delivered independently
- Different user personas benefit
- No shared implementation

---

## Complexity Triage Integration

After classifying as feature, run complexity triage to determine tier:

```
Scope Validation:
  Is it a feature? → YES
  Is it too large? → Check complexity score

Complexity Triage:
  Score: 15/27
  Tier: COMPLEX
  Recommendation: Consider splitting

Decision:
  Split into 3 features OR
  Accept as COMPLEX and plan manual execution
```

See `complexity-dimensions.md` for full triage criteria.

---

## Decision Flowchart

```
START: New work item
  │
  ├─ Is it user-facing OR multi-step OR integration point?
  │  │
  │  YES → Candidate FEATURE
  │  │     │
  │  │     ├─ Run complexity triage
  │  │     │  │
  │  │     │  ├─ Score 0-6: SIMPLE feature
  │  │     │  ├─ Score 7-15: STANDARD feature
  │  │     │  └─ Score 16+: COMPLEX feature → Consider splitting
  │  │     │
  │  │     └─ Estimate tasks
  │  │        │
  │  │        ├─ 1 task: Too small, make it a TASK instead
  │  │        ├─ 2-15 tasks: Right size FEATURE
  │  │        └─ 15+ tasks: Too large, SPLIT into multiple features
  │  │
  │  NO → Candidate TASK
  │       │
  │       ├─ Can it be completed in <2 days?
  │       │  │
  │       │  YES → TASK (proceed with task creation)
  │       │  NO → Too large, either:
  │       │       - Split into multiple tasks OR
  │       │       - Promote to FEATURE
  │       │
  │       └─ Does it deliver value independently?
  │          │
  │          YES → Reconsider as FEATURE
  │          NO → TASK (part of a feature)
```

---

## Validation Checklist

Use this checklist when validating feature scope:

### Feature Level Validation

- [ ] Delivers standalone value to users/stakeholders
- [ ] Breaks down into 2-15 meaningful tasks
- [ ] Estimated 1-10 days of implementation
- [ ] Has clear acceptance criteria
- [ ] Fits within one epic's business objective
- [ ] Not too similar to existing features (check for duplication)
- [ ] Complexity score appropriate for intended tier

### Task Level Validation

- [ ] Atomic unit of work (single cohesive purpose)
- [ ] Completable in <2 days by one developer
- [ ] Clear acceptance criteria (2-5 criteria)
- [ ] Fits within parent feature's scope
- [ ] Dependencies identified (if any)
- [ ] Not too similar to other tasks (check for duplication)

---

## Common Questions

### Q: When should a bug fix be a feature vs task?

**A**: Use the "investigation + design + implementation" test.

- Simple fix (known root cause, <1 file, <1 hour) → TASK
- Complex fix (requires investigation, affects multiple files, architectural change) → FEATURE

**Example**:

- "Fix typo in error message" → TASK
- "Fix memory leak in background job processor" → FEATURE (requires investigation, profiling, design, testing)

### Q: Should tests be separate features?

**A**: NO. Tests are part of implementing the feature, not separate deliverables.

**Exception**: "Add test coverage to legacy code" can be a feature if:

- It's a deliberate initiative (not part of new feature work)
- Stakeholder cares about test coverage metrics
- Requires significant effort (days of work)

### Q: How do I handle "technical debt" work?

**A**: Depends on scope.

- Small cleanup (<1 day) → TASK in "Tech debt" feature
- Large refactoring (multi-day, multi-file) → FEATURE
- Architectural change (affects multiple features) → EPIC

### Q: What about documentation?

**A**:

- Inline code comments → Part of task/feature implementation
- API docs for new endpoint → Part of endpoint feature
- Comprehensive developer guide → Separate FEATURE (if multi-day effort)
- Single README update → TASK

### Q: When should I split a large feature?

**A**: Split when ANY of these hold:

- Complexity score ≥16 (COMPLEX tier)
- 15+ implementation tasks
- 10+ days estimated
- Multiple teams required
- Can deliver value incrementally

**How to split**: by user value or by technical layers.

- User value: "User login" + "User registration" + "Password reset"
- Technical layers: "API layer" + "Service layer" + "UI layer"

**Prefer**: user-value splitting (delivers working features incrementally).

---

## Worked Examples

### Example 1: Should Have Been Split

**Original Classification**: FEATURE — service-layer completion (~7000 LoC refactor)

**Analysis**:

- Complexity score: 15/27 (high STANDARD, borderline COMPLEX)
- Tasks: 12 (above STANDARD threshold of 7)
- Lines to refactor: ~7000
- Time estimate: 3-4 weeks
- Regression risk: HIGH (behavior-preserving, 100% test pass required)

**Recommendation**: SPLIT into 3 features.

**Proposed split**:

- **Core operations** (get, list, filters): score 5/27, 4 tasks, 1 week
- **Lifecycle operations** (start, complete, approve, reopen): score 6/27, 4 tasks, 1 week
- **Advanced operations** (block/unblock, dependencies, notes): score 4/27, 4 tasks, 1 week

**Benefit**: each split-feature fits autonomous-build feasibility, deliverable value incrementally.

---

### Example 2: Right-Sized Feature

**Classification**: FEATURE — unified entity display rendering

**Analysis**:

- Complexity score: 4/27 (SIMPLE / low STANDARD)
- Tasks: 2-3
- Lines: ~500 new
- Time estimate: 5-7 days
- Regression risk: LOW (additive changes)

**Validation**: CORRECT.

- Delivers standalone value (unified rendering helpers)
- Right number of tasks (2-3)
- Appropriate complexity
- Autonomous build feasible

---

### Example 3: Task vs Feature Decision

**Work**: "Add `--agent` flag to task create command"

**Initial thought**: simple change, just a task?

**Analysis**:

- User-facing? YES (CLI users)
- Multi-step? YES (CLI parsing + service layer + repository + validation)
- Integration? YES (crosses CLI → service → DB)
- Files: 3-4
- Time: 1-2 days

**Classification**: FEATURE (even though it seems "simple")

**Why**: CLI feature additions are user-facing integration points.

**Tasks**:

1. Add `--agent` flag to CLI command definition
2. Update service to accept agentType
3. Add validation for agent types
4. Update repository Create method
5. Add tests for agent type filtering

---

## Summary

**Use this reference when**:

- Creating new work items (before creating feature/task)
- Reviewing existing features (scope validation)
- Deciding whether to split or consolidate
- Triaging work into epics/features/tasks

**Key takeaway**: features deliver user value, tasks implement features. When in doubt, run complexity triage.

**Next step after scope validation**: run complexity triage (see `complexity-dimensions.md`).
