{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Generate tasks for feature {{.id}}: "{{.title}}".

Check for existing tasks: {{template "list_json" .}}. If tasks exist covering all feature requirements with proper sequencing, advance immediately.

---

{{include: skills/specification-writing/workflows/write-task.md}}

READ:
(1) Feature spec.md for requirements, architecture, and file paths
(2) Feature test-plan.md for test cases
(3) Parent epic context for dependency awareness
(4) Feature spec.md "Cross-feature interactions" section and parent
    interaction map if present

## Step 0 - Identify contracts before decomposing

Separate internal feature contracts from cross-feature wires:

- Interfaces that stay inside this feature may use local CONTRACT-### IDs if the
  feature spec defines them.
- Interfaces that cross OUTSIDE this feature use I-## from the epic's
  interaction map.
- Those I-## rows are declared in the feature spec's "Cross-feature
  interactions" section; mirror them in the producing/consuming task spec's
  "Integration Contracts > Cross-feature" subsection.
- Do NOT invent new CONTRACT-### IDs for cross-feature wires.
- Mirror the same shape source and contract-test pointer from the feature spec.

PRODUCE tasks via shark CLI. Each task call MUST pass --size:

```
{{template "create_task" .}} "<title>" --order=N --size=<1|2|3|5|8|13>
```

{{template "_sizing_task" .}}

Then write task spec file (50 LINES MAX, not counting frontmatter):

```markdown
---
key: <assigned key>
title: "<title>"
epic: <epic>
feature: {{.id}}
agent: developer
priority: N
execution_order: N
size: <1|2|3|5|8|13>   # mirror of --size; required
dependencies: [<prior task keys>]
---

# <Title>

## Goal
One paragraph: what this task accomplishes and why.

## Scope
- Files to modify: (list with paths)
- Files to create: (list with paths, if any)

## Acceptance Criteria
Reference spec.md: "See spec.md AC-1, AC-2, AC-3"
Additional task-specific criteria (if any):
- AC-T1: <specific to this task>

## Test Cases
Reference test-plan.md: "See test-plan.md Section 1, cases 1.1-1.4"

## Integration Contracts

### Cross-feature
- I-##: produces|consumes; shape source: <spec.md/architecture.md section>;
  contract test: <test-plan.md TC or test file pointer>

## Design Reference
"See spec.md Architecture Section 2 for component design"
"See spec.md Architecture Section 3 for data model"
```

CRITICAL RULES:
- 50 lines MAX per task file (not counting frontmatter)
- NO code blocks in task files (no Go, SQL, etc.)
- REFERENCE parent docs by section — do NOT copy content
- NO ADRs, user stories, or edge cases in tasks (those live in spec.md)
- Each task modifies a coherent set of files (not too broad, not too narrow)
- TDD STRUCTURE: Implementation code and its tests live in the SAME task.
  NEVER create a task titled "Write tests for X" that follows a task titled
  "Implement X" — that is test-after, not TDD. Each task's Scope section must
  list BOTH the implementation file(s) AND the corresponding test file(s).
  Developer agents are instructed to write failing tests first (from
  test-plan.md), then implement to make them pass.
  Exception: a feature-level integration/e2e suite that exercises multiple
  components may be its own task, since it cannot be co-located with a single
  implementation file. State this exception explicitly in the task's Goal.
- Cross-feature wires use I-## only. Do NOT invent new CONTRACT-### IDs for
  cross-feature wires or rewrite the shape source/contract-test pointer.

EXIT GATE:
- All spec.md requirements covered by tasks
- Task dependencies form a valid DAG
- Each task is 50 lines or less
- Zero code blocks in any task file
- Every task references spec.md and test-plan.md sections
- Every implementation task's Scope includes its own test file(s); no
  standalone "write tests for X" task follows an "implement X" task
- Every task carries a non-empty size; no task is sized 8/XL or 13/XXL
  (decompose first — `shark list --json | jq` to verify sizes)
- Multi-feature epic: every I-## this feature produces or consumes appears in
  the relevant task spec's "Integration Contracts > Cross-feature" subsection
