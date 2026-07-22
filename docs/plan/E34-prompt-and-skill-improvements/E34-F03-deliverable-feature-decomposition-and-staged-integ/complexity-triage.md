# Complexity triage report — E34-F03

**Feature**: Deliverable Feature Decomposition and Staged Integration Acceptance
**Score**: 13/27
**Tier**: STANDARD

## Scope validation

Keep this work as a feature. It defines one shared acceptance/readiness
capability across epic decomposition, feature specification, task review, QA,
UAT, and approval. It has multiple independently testable policy changes and
must keep a producer/consumer contract with E34-F02. It is not an atomic task.

## Dimension scores

### Technical complexity

1. **File impact: 3/3** — The known surface spans at least 10 embedded prompt,
   skill, template, and rendered-golden files: epic decomposition/design/review;
   feature specification, task review, QA, and code review; the interaction-map
   template; UAT/quality content; and prompt-rendering tests.
2. **Pattern novelty: 1/3** — The repository already carries interaction-map
   IDs, shared contract tests, and rendered-prompt goldens. This feature extends
   those patterns with explicit `live` and `contract-only` gate semantics rather
   than adding a workflow engine or new runtime architecture.
3. **Data model: 0/3** — The stated boundary changes bundled policy and evidence
   contracts. It does not require a database schema change.
4. **API surface: 1/3** — Existing CLI prompt output and documented workflow
   vocabulary gain fields and rules, but the feature does not introduce a new
   command, HTTP endpoint, or entity type.
5. **Cross-feature dependencies: 2/3** — E34-F03 is the acceptance/readiness
   producer for E34-F02 and coordinates the specification-writing, quality, and
   UAT surfaces. The dependency is directional; no circular dependency is
   required.
6. **UI complexity: 0/3** — No interactive UI is in scope. Prompt and report
   wording is the user-facing work.

### Execution complexity

7. **Task estimation: 2/3** — A credible decomposition is about six tasks:
   acceptance matrix and decomposition prompts; interaction-map schema and
   validation; feature/task-review rules; QA/UAT and approval semantics;
   E34-F02 consumer contract; and rendered-prompt regressions.
8. **Regression risk: 2/3** — These rules decide whether workflow reviews block,
   conditionally accept, or close work. Inconsistent wording across prompt
   surfaces could weaken security/integrity gates or falsely report incomplete
   integration as delivered.
9. **Execution effort: 2/3** — The implementation must align policy, templates,
   prompts, quality/UAT behavior, owner-decision vocabulary, and golden tests.
   The rubric heuristic estimates about 18 working days, so this is a 3–4 week
   cross-surface change rather than mechanical editing.

**Technical total**: 7/18
**Execution total**: 6/9
**Overall total**: 13/27

## Tier assignment

**Assigned tier**: STANDARD

**Rationale**: E34-F03 extends established prompt and interaction-map patterns,
but it crosses several gate-owning surfaces and changes high-consequence
acceptance language. The work is substantial enough to need validated research
and a deliberate task breakdown, but it does not add new persistence, APIs, or
a workflow runtime.

## Autonomous build feasibility

- Task count: about 6 (threshold <=10)
- Regression risk: 2 (threshold <=1)
- Execution effort: 2 (threshold <=1)
- Circular dependencies: no; E34-F03 produces the contract and E34-F02 consumes it

**Recommendation**: Manual, staged execution recommended. Start with research
to enumerate every producer and consumer prompt, template, and rendered-golden
test. Preserve the strict non-waiver rule for security, integrity, and unmet
current-feature criteria.

## Evidence reviewed

- `feature.md` for the proposed capability, source incidents, boundaries, and
  E34-F02 relationship.
- `epic.md`, `requirements.md`, and the E34-F02 feature brief for the parent
  scope and producer/consumer ordering.
- The E04 remediation plan for the live-vs-contract-only decision table and
  downstream closure requirements.
- Existing embedded decomposition, specification, task-review, QA, code-review,
  UAT, interaction-map, and rendered-prompt surfaces identified by repository
  search.
