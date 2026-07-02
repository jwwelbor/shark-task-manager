---
name: assessment-scope-validation
mode: scope_validation
---

# Workflow: Scope Validation

**Purpose**: Determine if a candidate work item belongs at feature level or task level, or whether it should be split or consolidated.

## Process

### Step 1: Apply Classification Decision Tree

Use the decision tree from `../references/scope-criteria.md`:

```text
Is the work...
|
|-- User-visible or stakeholder-visible?
|   `-- YES -> Likely a FEATURE
|
|-- Requires multiple implementation steps?
|   `-- YES -> Likely a FEATURE
|
|-- Crosses multiple domains/layers?
|   `-- YES -> Likely a FEATURE
|
|-- A single technical operation?
|   `-- YES -> Likely a TASK
|
`-- Part of implementing a larger feature?
    `-- YES -> Likely a TASK
```

### Step 2: Check Size Thresholds

**Feature thresholds**:

- Implementation tasks: 2-15
- Lines of code (new): 100-2000
- Lines of code (refactor): 500-5000
- Time estimate: 1-10 days

**Task thresholds**:

- Lines of code: 20-500
- Time estimate: 2 hours - 2 days
- Acceptance criteria: 2-5

### Step 3: Make Classification Decision

**Classify as FEATURE if**:

- Delivers standalone user or stakeholder value
- Requires 2+ meaningful implementation steps
- Crosses multiple system layers
- Estimated 1-10 days of work

**Classify as TASK if**:

- Atomic unit of work
- Single cohesive purpose
- Completable in <2 days
- Part of a larger feature

**Classify as CONSOLIDATE if**:

- Multiple tasks share the same user goal
- Tasks must be deployed together
- Individually they have no stakeholder value

**Classify as SPLIT if**:

- Feature has 15+ tasks
- Estimated >10 days
- Can deliver value incrementally
- Multiple independent user benefits are bundled together

### Step 4: Document Decision

```markdown
# Scope Validation — {work_item_description}

**Classification**: {FEATURE|TASK|SPLIT|CONSOLIDATE}

**Analysis**:
- User-visible: {yes|no}
- Multi-step: {yes|no}
- Integration: {yes|no}
- Estimated tasks: {count}
- Estimated time: {duration}

**Decision Rationale**: {explanation}

**Recommended Action**: {create as feature | create as task | split into N features | consolidate into feature X}
```

### Step 5: Return Structured Output

Return: `classification`, `decision_rationale`, `recommended_action`.
