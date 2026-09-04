---
feature_key: E34-F06-defect-class-completeness-and-recurrence-routing
epic_key: E34
title: Defect-Class Completeness and Recurrence Routing — Test Plan
---

# E34-F06 Test Plan

**content-only justification**: This feature adds one embedded workflow
markdown file and edits three existing prompt/skill markdown files. No Go
source, CLI command, schema, or deterministic runtime behavior changes.
Per CLAUDE.md's prompt-only testing guidance and this plan's own "Prompt-only
changes" gate, tests below use the production template renderer, the
embedded-bundle validators already exercised by `internal/sharkdata/embed_test.go`
and `internal/templates/includes_test.go`, direct-file existence checks, and
manual policy-wording review — not caller-path/mutation/decision-table tests,
which apply only to deterministic runtime behavior this feature does not add.

## AC Test Matrix

| TC | AC | Description | Input/Setup | Expected outcome | Edge cases |
|----|----|--------------|--------------|-------------------|------------|
| TC-001 | AC-1 | New workflow file renders cleanly | `internal/sharkdata/default_data/skills/quality/workflows/defect-class-sweep.md` present, run through `internal/templates` production renderer (extend `TestIncludeResolver_*` table or add a sibling test) | Renderer returns no error; output contains the section headers named in spec.md REQ-F-001 (class naming, search scope, enumeration, zero-result reporting, instance evidence, guard selection, closure, three-part re-verification) | Missing section header → test fails; malformed `{{include}}` syntax → renderer error surfaces |
| TC-002 | AC-2 | Sweep prose consolidated, not duplicated | Grep `code_review.md`, `approval.md`, `redteam-rubric.md` post-edit for the old inline sweep-procedure prose strings | Zero matches for the old duplicated prose in all three files; each file instead references the new workflow path | A file that still contains the old prose fails the test |
| TC-003 | AC-3 | Bundle indices updated | `skills/quality/SKILL.md` "Workflow Selection" section and `skills/README.md` quality row | Both name `workflows/defect-class-sweep.md` | Missing from either index → test fails (mirrors existing `internal/sharkdata` index-completeness checks) |
| TC-004 | AC-4 | No WWGM/Python-tool/local-path leakage | `grep -rin "WWGM\|\.py\b\|/home/\|/Users/" internal/sharkdata/default_data/skills/quality/workflows/defect-class-sweep.md` plus the three edited files | Zero matches | A reintroduced WWGM-specific term or hardcoded path fails the test |
| TC-005 | AC-5 (scenario: close an enumerated class) | Workflow content produces a complete-class decision | Feed the workflow's documented procedure a fixture: 1 finding, 1 matching sibling instance, both fixed, guard verified | Rendered guidance instructs `status: complete`, `open_count: 0`, `matching_count = fixed_count + dispositioned_count` | Guard not verified → `status` must remain `open` |
| TC-006 | AC-5 (scenario: distinguish recurrence) | Same-fingerprint vs. new-fingerprint classification | Fixture A: same fingerprint resurfaces after recorded repair. Fixture B: new fingerprint, same `class_key`, inside a completed `search_scope`. Fixture C: new fingerprint outside that scope | A/B classified as recurrence; C classified as a normal finding routed to ordinary rework | Round-count field must not appear anywhere in the classification logic (grep check) |
| TC-007 | AC-5 (scenario: route a severity conflict) | HIGH finding conflicts with prior accepted LOW decision | Fixture: fresh evidence materially changes risk on a previously-accepted fingerprint | Workflow content routes to `severity_conflict`, blocks normal advancement, references Question/council mechanisms by name | A conflict routed silently through normal rework (no block) fails the test |
| TC-008 | AC-5 (scenario: zero remaining instances) | Full re-enumeration finds nothing | Fixture: re-run re-verification after all instances already fixed | Result reports `searched_count`, `matching_count: 0`, `open_count: 0` explicitly — not an empty/omitted result | Omitted counts on a zero-result pass fails the test |
| TC-009 | AC-5 (scenario: missing/disabled/ineffective guard) | Guard counterfactual gate | Fixture: sibling defect reintroduced with (a) no guard, (b) a guard present but disabled, (c) a guard present but does not actually detect the class when reintroduced | All three cases: workflow content requires closure to fail until an executable counterfactual proves detection | A guard accepted without counterfactual evidence fails the test |
| TC-010 | REQ-NF-001 | No new persistence introduced | `grep -rln "DefectClassSweep\|class_key" internal/` (Go source only) | Zero Go-source hits (confirms the feature stayed content-only, matching research-report.md's finding) | Any new `.go` file matching fails the test — signals scope creep into a Go persistence layer |

## Caller-Path Contracts

Not applicable in the traditional sense — no test case in this plan drives a
Go function with a production argument shape, because this feature adds no
deterministic runtime code. Per the "Prompt-only changes" exemption, each
test case above instead names its concrete entrypoint:

- TC-001, TC-003: `internal/templates` production renderer / direct bundle
  index file read — entrypoint is the renderer function already exercised by
  `internal/templates/includes_test.go` (e.g. sibling to
  `TestIncludeResolver_BasicInclude`).
- TC-002, TC-004, TC-010: direct grep/file-content check against the
  checked-in bundle files — entrypoint is the file itself, `content-only`
  justification.
- TC-005 through TC-009: manual policy-wording review of the rendered
  workflow content against each fixture scenario — `content-only`
  justification; there is no executable classifier to unit-test because the
  "classification" is instructional prose a future AI worker follows, not a
  Go decision function.

## Integration Scenarios

- **Consuming gate → new workflow reference**: `code_review.md`,
  `approval.md`, and `redteam-rubric.md` each `{{include}}` or textually
  reference `skills/quality/workflows/defect-class-sweep.md`. Verify via the
  same renderer path as TC-001 that each of the three consuming prompts
  resolves the reference without a broken-include error — this is the
  integration boundary between "the new workflow exists" (TC-001) and "gates
  actually use it" (TC-002).
- **E34-F08 downstream consumption**: out of this feature's test scope — TC
  pointer `TC-I-03-DEFECT-CLASS-CLOSURE` (below) proves this feature's I-03
  *shape*; E34-F08's own test plan proves it *consumes* that shape.

## Test Infrastructure

- **Existing to reuse**: `internal/templates/includes_test.go`'s
  `TestIncludeResolver_*` table pattern (add entries rather than a new test
  file) for TC-001 and the integration-scenario include checks;
  `internal/sharkdata/embed_test.go`'s `TestEmbedded_SkillsContainNoBareSharkCLIRefs`
  and `TestEmbedded_AgentsDescribeRoleNotWorkflow` remain in force for the new
  file (no new helper needed — the new file must pass these existing global
  gates by construction, since it contains no bare `shark <verb>` string and
  is workflow content, not an agent persona).
- **New test helpers needed**: none. TC-002/TC-004/TC-010 are one-line grep
  assertions addable to `embed_test.go` or a lightweight companion test file
  in the same package; TC-005 through TC-009 are manual review checklist
  items recorded in the code-review/UAT report for this feature, not
  automated Go tests (no fixture-execution harness exists for prose
  "decision procedures," consistent with E34-F05's precedent of not building
  one either).

## Cross-feature contract tests (I-##)

| I-## | Producer | Consumer(s) | Shape source | Contract test pointer | TC |
|---|---|---|---|---|---|
| I-02 | E34-F05 | E34-F06, E34-F07, E34-F08 | architecture.md#i-02-gateresult-v1 | `E34-F05-structured-gate-results-and-parent-owned-persisten/test-plan.md#TC-I-02-GATERESULT-PARITY` | (owned by E34-F05's test plan; this feature only nests inside the existing envelope, verified structurally by TC-010) |
| I-03 | E34-F06 | E34-F08 | architecture.md#i-03-defectclasssweep-v1 | `E34-F06-defect-class-completeness-and-recurrence-routing/test-plan.md#TC-I-03-DEFECT-CLASS-CLOSURE` | **TC-I-03-DEFECT-CLASS-CLOSURE** = TC-005 + TC-008 + TC-009 combined: together they prove the full I-03 completeness invariant (`matching_count = fixed_count + dispositioned_count`, `open_count = 0`, `guard.status = verified`) that E34-F08 will consume. E34-F08's test plan must reference this same pointer rather than writing a twin test. |

## Cross-epic integration tests (X-##)

None. `spec.md` "Cross-epic integrations" declares no X-## rows for this
feature (confirmed empty in both `E34-cross-epic-map.md` and
`docs/product/cross-epic-integration-map.md`).

## Codex Test-Plan Red-Team

**Verdict:** Not run — codex CLI unavailable in this environment during this
pass.
**Issues raised:** 0
**Issues addressed before dev:** 0
**Issues deferred:** 1 — codex red-team pass on this test plan. Owner: next
worker to touch E34-F06 with codex access; timeframe: before task_review
closes, or documented again as deferred if codex remains unavailable at that
point. Logged as a non-blocking note rather than gating this content-only,
STANDARD-tier plan on tool availability, per this workflow's own timeout/
retry/degrade guidance.

## Recommendations

- [x] Ready for development — no PRD/spec drift (spec.md traces every
      requirement to feature.md verbatim), every AC has at least one test
      case, content-only exemption is documented and applied consistently,
      I-03 contract test pointer is declared and matches spec.md exactly.
- [ ] Needs BA refinement
- [ ] Needs tech refinement

*Last Updated*: 2026-09-04
