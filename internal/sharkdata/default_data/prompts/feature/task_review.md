Review tasks for feature {{.id}}: "{{.title}}".

Check for existing task review report at {{.review_base}}{{.id}}-task-review.md.
If report exists with PASS verdict, advance immediately. If FAIL, send back to task_generation.

---

TASK DECOMPOSITION REVIEW

This is a quality gate comparing the generated tasks against the feature specification. The goal is to catch gaps, ordering issues, or spec misalignment BEFORE development begins.

READ:
(1) Feature spec at {{.file_path}} for requirements, architecture decisions, and acceptance criteria
(2) Feature test-plan.md (sibling of spec.md) for expected test cases
(3) All task files: {{template "list_json" .}}, then read each task's file_path
(4) Parent epic context for cross-feature dependency awareness
(5) Parent interaction map if present
(6) Parent {{.epic_id}}-cross-epic-map.md and
    docs/product/cross-epic-integration-map.md if present

VERIFY:

## Requirements Coverage
- [ ] Every requirement in spec.md is addressed by at least one task
- [ ] Every acceptance criterion in spec.md maps to at least one task
- [ ] No spec requirements are missing or only partially covered

## Task Quality
- [ ] Each task file is 50 lines or less (not counting frontmatter)
- [ ] No code blocks in task files (references only)
- [ ] Each task references spec.md and test-plan.md sections (not copy-pasted content)
- [ ] Task scopes are coherent (each modifies a related set of files)
- [ ] Task titles accurately reflect their content
- [ ] TDD shape: no task is pure "write tests for X" following a preceding
      "implement X" task. Each implementation task owns its own tests — its
      Scope must list both implementation file(s) AND test file(s).
      Exception: a single feature-level integration/e2e test suite that
      spans multiple components may be its own task (goal must say so).
      If this rule is violated, FAIL and send back with reason: "merge
      test-only task(s) into their implementation task(s) for TDD."

## Ordering & Dependencies
- [ ] Execution order reflects actual dependencies
- [ ] Dependencies form a valid DAG (no circular chains)
- [ ] Foundation tasks (shared types, interfaces, DB schema) come first
- [ ] Tasks can be worked on in the specified order without blocking

## Scope Alignment
- [ ] Tasks stay within feature scope (no scope creep beyond spec.md)
- [ ] No unnecessary tasks that don't map to spec requirements
- [ ] Task granularity is appropriate (each is a focused unit of work)

### Integration coverage (STANDARD/COMPLEX only)

- Every CONTRACT-### declared for internal feature contracts appears in exactly
  one producer task and at least one consumer task
- Producer and consumer cite the same shape source
- Each contract has a single contract-test pointer that both sides reference
  verbatim
- No orphans (missing producer, missing consumer, mismatched pointer) is FAIL
- Every I-## the feature spec declares under "Cross-feature interactions" is
  mirrored in the producing/consuming task spec(s) under "Integration Contracts
  > Cross-feature"
- Every mirrored I-## keeps the same shape source and contract-test pointer from
  the feature spec
- Every X-## the feature spec declares under "Cross-epic integrations" is
  mirrored in the producing/consuming/validating task spec(s) under
  "Integration Contracts > Cross-epic"
- Every mirrored X-## keeps the same producer/consumer feature refs, contract /
  shape source, UX / CX handoff notes, and test coverage pointer or explicit
  progress-log deferral from the feature spec
- Missing X-## producer task, consumer task, validation task, mismatched shape
  source, or missing coverage disposition is FAIL

{{template "_review_output_policy" .}}

PRODUCE task review report at {{.review_base}}{{.id}}-task-review.md:
- If zero findings: compact PASS artifact only
  - Verdict: PASS
  - Scope reviewed: feature spec, test-plan, task count, and parent integration context checked
  - Checklist section totals: Requirements Coverage, Task Quality, Ordering & Dependencies, Scope Alignment
  - Integration row counts reviewed for CONTRACT-###, I-##, and X-## (if applicable)
  - Duration if known
  - `0 defects found`
- If any requirement gap, ordering issue, task quality issue, integration mismatch, or other finding exists: full detailed report
  - Verdict: FAIL
  - Requirements coverage matrix (spec requirement -> task mapping)
  - Integration coverage matrix for CONTRACT-###, I-##, and X-## rows, including producer task, consumer task(s), shape source, contract-test pointer, and closure status
  - Gaps identified
  - Ordering issues
  - Task quality issues
  - Recommendations

DECISION:
- ALL PASS -> end with `RECOMMENDED OUTCOME: pass`
- ANY FAIL -> end with `RECOMMENDED OUTCOME: fail` and include the specific gaps or issues to fix in your final summary
- Do NOT run Shark status commands yourself; the parent loop will apply the outcome and route the feature.
