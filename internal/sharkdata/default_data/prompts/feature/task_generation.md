{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Generate tasks for feature {{.id}}: "{{.title}}".

Check for existing tasks: {{template "list_json" .}}. If tasks exist covering all feature requirements with proper sequencing, advance immediately.

---

## Step 0: Determine Complexity Tier

Check `shark feature notes {{.id}}` (or {{template "get_json" .}}) for the most recent `COMPLEXITY: <TIER>` decision note — assessment records one, and an escalation below may have added a later, superseding one. **The latest such note wins.**

- **SIMPLE** -> follow "SIMPLE-lite mode" below; the spec and test-plan gates are waived, but validated research is never waived.
- **STANDARD**, **COMPLEX**, or no COMPLEXITY note found -> the included workflow's Hard Gates apply as written.

### Prompt-only scope

For embedded prompt, skill, template, or documentation-only features, accept a
test plan that uses renderer/include checks, golden snapshots, and documented
file-reference validation. Do not block task generation because the plan lacks
runtime caller-path, decision-table, or wording-mutation tests.

### SIMPLE-lite mode

A SIMPLE feature does not require `spec.md` or `test-plan.md` before tasks can be written. It does require the validated unified `research-report.md` and its Capability map. Instead:

- Each task file's Acceptance Criteria and Test Cases sections are written **inline and concrete** rather than by TC-ID reference: for code changes, enumerable pass/fail conditions; for doc-only tasks, verification steps (links resolve, lint/format passes, reviewer checklist).
- Use the research report's Capability map to prevent duplicate work; do not substitute assessment grep output.

**Escalation valve**: if lite decomposition would need more than 3 tasks, or any task is sized 5 or larger, this feature is not actually SIMPLE. Stop immediately, explain the reason, and include BOTH of these lines in your final response so the parent loop can persist the superseding decision and route back through test_planning:

`COMPLEXITY NOTE: COMPLEXITY: STANDARD (supersedes SIMPLE; task-generation found <reason>)`

`RECOMMENDED OUTCOME: fail`

Do not continue generating lite tasks after escalating — test_planning will produce the missing test-plan.md before task_generation runs again.

{{include: skills/specification-writing/workflows/write-task.md}}

READ:
(1) Validated feature research-report.md for vocabulary, source evidence, and the Capability map
(2) Feature spec.md for requirements, architecture, and file paths (required for STANDARD/COMPLEX)
(2) Feature test-plan.md for test cases
(3) Parent epic context for dependency awareness
(4) Feature spec.md "Cross-feature interactions" section and parent
    interaction map if present
(5) Feature spec.md "Cross-epic integrations" section, parent
    {{.epic_id}}-cross-epic-map.md, and docs/product/cross-epic-integration-map.md
    if present

## Step 0 - Identify contracts before decomposing

Separate internal feature contracts, cross-feature wires, and cross-epic
integrations:

- Interfaces that stay inside this feature may use local CONTRACT-### IDs if the
  feature spec defines them.
- Interfaces that cross OUTSIDE this feature use I-## from the epic's
  interaction map.
- Interfaces or journey handoffs that cross EPIC boundaries use X-## from
  docs/product/cross-epic-integration-map.md and the parent
  {{.epic_id}}-cross-epic-map.md.
- Those I-## rows are declared in the feature spec's "Cross-feature
  interactions" section; mirror them in the producing/consuming task spec's
  "Integration Contracts > Cross-feature" subsection.
- Those X-## rows are declared in the feature spec's "Cross-epic integrations"
  section; mirror them in the producing/consuming/validating task spec's
  "Integration Contracts > Cross-epic" subsection.
- Do NOT invent new CONTRACT-### IDs for cross-feature wires.
- Do NOT invent I-## IDs for cross-epic integrations or X-## IDs for
  cross-feature interactions.
- Mirror the same shape source and contract-test pointer from the feature spec.
- For each staged I-##, mirror the map-assigned gate mode, counterpart entities
  and a current status read live from Shark, shared-contract evidence, activation owner, closure key, and
  review basis in every relevant task. `live` is the default; never create a
  `contract-only` declaration after feature specification or invent its values.
- Required staged fields include `review_basis`; do not omit it from task specs.

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

### Cross-epic
- X-##: produces|consumes|validates; producer/consumer epic+feature:
  <epic/feature refs>; contract / shape source: <product map source>;
  coverage: <test-plan.md TC, test file pointer, or progress deferral>

## Design Reference
"See spec.md Architecture Section 2 for component design"
"See spec.md Architecture Section 3 for data model"
```

CRITICAL RULES:
- Draft each task file by reference from the start: cite spec.md/test-plan.md
  sections and file:line locations, don't narrate them. A task file that
  needs trimming after a first draft was written the wrong way, not just
  too long.
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
- Cross-epic integrations use X-## only. Put X-## work in the distinct
  "Integration Contracts > Cross-epic" subsection and keep it separate from
  I-## cross-feature work.

Before returning, run a line count on every task file you wrote or are
reusing (excluding frontmatter): `wc -l <file>` minus frontmatter lines. Any
file over 50 lines must be trimmed now, in this pass — do not rely on
task_review to catch it.

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
- Every X-## this feature produces, consumes, or validates appears in the
  relevant task spec's "Integration Contracts > Cross-epic" subsection with
  producer/consumer feature refs, matching contract / shape source, and coverage
  pointer or progress-log deferral
