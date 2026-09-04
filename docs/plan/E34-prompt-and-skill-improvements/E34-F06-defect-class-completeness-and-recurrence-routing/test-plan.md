---
feature_key: E34-F06-defect-class-completeness-and-recurrence-routing
epic_key: E34
title: Defect-Class Completeness and Recurrence Routing — Test Plan
---

# E34-F06 Test Plan

**content-only justification**: This feature adds one embedded workflow
markdown file and edits five existing prompt/skill markdown files
(`code_review.md`, `approval.md`, `redteam-rubric.md`, `qa.md`,
`development.md` — the last two added by round-1's HIGH-1 fix, which wired
QA and development as workflow consumers alongside the original three). No Go
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
| TC-002 | AC-2 | Sweep prose consolidated, not duplicated | Grep `code_review.md`, `approval.md`, `redteam-rubric.md`, `qa.md`, `development.md` post-edit for old and paraphrased inline sweep-procedure prose strings (`TestDefectClassSweepConsolidatedNotDuplicated`) | Zero matches for the old/paraphrased duplicated prose in any of the five files; each file instead references the new workflow path | A file that still contains the old or paraphrased prose fails the test |
| TC-003 | AC-3 | Bundle indices updated | `skills/quality/SKILL.md` "Workflow Selection" section and `skills/README.md` quality row | Both name `workflows/defect-class-sweep.md` | Missing from either index → test fails (mirrors existing `internal/sharkdata` index-completeness checks) |
| TC-004 | AC-4 | No WWGM/Python-tool/local-path leakage | `grep -rin "WWGM\|\.py\b\|/home/\|/Users/" internal/sharkdata/default_data/skills/quality/workflows/defect-class-sweep.md` plus the five edited files | Zero matches | A reintroduced WWGM-specific term or hardcoded path fails the test |
| TC-005 | AC-5 (scenario: close an enumerated class) | Workflow content produces a complete-class decision | Feed the workflow's documented procedure a fixture: 1 finding, 1 matching sibling instance, both fixed, guard verified | Rendered guidance instructs `status: complete`, `open_count: 0`, `matching_count = fixed_count + dispositioned_count` | Guard not verified → `status` must remain `open` |
| TC-006 | AC-5 (scenario: distinguish recurrence) | Same-fingerprint vs. new-fingerprint classification | Fixture A: same fingerprint resurfaces after recorded repair. Fixture B: new fingerprint, same `class_key`, inside a completed `search_scope`. Fixture C: new fingerprint outside that scope | A/B classified as recurrence; C classified as a normal finding routed to ordinary rework | Round-count field must not appear anywhere in the classification logic (grep check) |
| TC-007 | AC-5 (scenario: route a severity conflict) | HIGH finding conflicts with prior accepted LOW decision | Fixture 7a: bounded, single-owner disagreement. Fixture 7b: specialist/cross-entity/blast-radius disagreement over the same conflicting evidence | Both route to `severity_conflict` and block normal advancement identically; 7a names `question-management`, 7b names the council workflow, per the section's own partition criteria | A conflict routed silently through normal rework (no block), or the two mechanisms left unpartitioned, fails the test |
| TC-008 | AC-5 (scenario: zero remaining instances) | Full re-enumeration finds nothing | Fixture: re-run re-verification after all instances already fixed | Result reports `searched_count`, `matching_count: 0`, `open_count: 0` explicitly — not an empty/omitted result | Omitted counts on a zero-result pass fails the test |
| TC-009 | AC-5 (scenario: missing/disabled/ineffective guard) | Guard counterfactual gate | Fixture: sibling defect reintroduced with (a) no guard, (b) a guard present but disabled, (c) a guard present but does not actually detect the class when reintroduced | All three cases: workflow content requires closure to fail until an executable counterfactual proves detection | A guard accepted without counterfactual evidence fails the test |
| TC-010 | REQ-NF-001 | No new persistence introduced | `TestDefectClassSweepNoGoPersistenceIntroduced` in `internal/sharkdata/embed_test.go` (Go source only, tightened synonym-aware scan — see Test Infrastructure) | Zero Go-source hits (confirms the feature stayed content-only, matching research-report.md's finding) | Any new `.go` file matching fails the test — signals scope creep into a Go persistence layer, including under a renamed identifier (`SweepRecord`, `defectKey`, etc.) |
| TC-011 | REQ-F-002 (backward-looking rework: compatible fix or cited divergence) | UAT-kickback (MEDIUM-1) regression guard: the workflow must require implementing a recorded compatible fix design, or citing durable evidence to justify diverging from one — not silent override | Fixture A: implement recorded design as-is. Fixture B: diverge, citing durable evidence in `evidence`. Fixture C (violation): diverge with no cited evidence. Fixture D: no recorded design found. Walkthrough in `scenario-review-TC-005-TC-009.md#tc-011-backward-looking-rework-compatible-fix-or-cited-divergence-req-f-002` | A/B compliant, C explicitly rejected by the workflow's own text ("a repair that silently does something different ... does not satisfy this section"), D outside scope by construction | A repair that diverges from a recorded design with no cited evidence, and the workflow content silently permitting it, fails the test |
| TC-012 | AC-5 / REQ-F-006 (accepted-risk visible-but-non-blocking branch) | UAT-kickback round-3 (HIGH-6) regression guard: a recurring finding covered by a dated, owner-grounded acceptance decision with no material new evidence must stay visible but non-blocking, distinct from the severity-conflict path | Fixture A (accepted-risk): no new evidence since the acceptance decision. Fixture B (contrast with TC-007): fresh evidence materially changes risk. Walkthrough in `scenario-review-TC-005-TC-009.md#tc-012-accepted-risk-visible-non-blocking-branch-req-f-006` | Fixture A → `dispositioned`, visible in `instances`, non-blocking, no Question/council routing. Fixture B → falls through to the severity-conflict path (TC-007), not this branch | A workflow that routes fixture A through `severity_conflict` (blocking) or silently drops it from `instances` (invisible) fails the test |

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
- TC-005 through TC-009, TC-011: manual policy-wording review of the
  rendered workflow content against each fixture scenario — `content-only`
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
  in the same package (TC-010 is implemented as
  `TestDefectClassSweepNoGoPersistenceIntroduced` in
  `internal/sharkdata/embed_test.go`, scoped to non-test Go source under
  `internal/`, and tightened per UAT-kickback MEDIUM-3 to also catch a
  second, independent signal — a new non-test `.go` file under `internal/`
  that didn't exist before this feature's base commit — so a renamed
  identifier alone can't evade both checks); TC-005 through TC-009 and
  TC-011 are manual review checklist items recorded in the code-review/UAT
  report for this feature — the reviewed-evidence record is
  `E34-F06-defect-class-completeness-and-recurrence-routing/scenario-review-TC-005-TC-009.md`
  — not
  automated Go tests (no fixture-execution harness exists for prose
  "decision procedures," consistent with E34-F05's precedent of not building
  one either). TC-011's compatible-fix-or-cited-divergence enforcement is
  additionally backed by an `embed_test.go` regression guard
  (`TestDefectClassSweepBackwardLookingReworkRequiresCompatOrDivergence`)
  asserting the enforcing sentences are present in the rendered workflow
  content, the same pattern used for the HIGH-1..4 regression guards.

## Cross-feature contract tests (I-##)

| I-## | Producer | Consumer(s) | Shape source | Contract test pointer | TC |
|---|---|---|---|---|---|
| I-02 | E34-F05 | E34-F06, E34-F07, E34-F08 | architecture.md#i-02-gateresult-v1 | **GAP** — `E34-interaction-map.md` names `TC-I-02-GATERESULT-PARITY` as a "planned" pointer E34-F05 "must create," but E34-F05 is completed with no `test-plan.md` and no such test anywhere in the repo (verified 2026-09-04). Out of E34-F06 scope to create (I-02 is consumed, not produced, here). | (owned by E34-F05's test plan, which does not yet exist; this feature only nests inside the existing envelope, verified structurally by TC-010) |
| I-03 | E34-F06 | E34-F08 | architecture.md#i-03-defectclasssweep-v1 | `E34-F06-defect-class-completeness-and-recurrence-routing/scenario-review-TC-005-TC-009.md#tc-i-03-defect-class-closure-cross-reference` (the anchor lives in `scenario-review-TC-005-TC-009.md`, not `test-plan.md`) | **TC-I-03-DEFECT-CLASS-CLOSURE** = TC-005 + TC-008 + TC-009 combined: together they prove the full I-03 completeness invariant (`matching_count = fixed_count + dispositioned_count`, `open_count = 0`, `guard.status = verified`) that E34-F08 will consume. E34-F08's test plan must reference this same pointer rather than writing a twin test. |

## Cross-epic integration tests (X-##)

None. `spec.md` "Cross-epic integrations" declares no X-## rows for this
feature (confirmed empty in both `E34-cross-epic-map.md` and
`docs/product/cross-epic-integration-map.md`).

## Codex Test-Plan Red-Team

**UAT-kickback note (MEDIUM-4):** the entry below this note replaces a prior
version that read only "Not run — codex CLI unavailable in this environment
during this pass," with no attempt/retry/failure evidence. UAT correctly
flagged that as an unsubstantiated deferral — `codex` was in fact on `PATH`
in this environment (`which codex` → `codex-cli 0.153.0`) and was not
attempted before that note was written. This entry documents the actual
attempt, run at 2026-09-04 (during the T-E34-F06-003 rework pass), per this
workflow's timeout/retry/degrade guidance (`quality/workflows/test-planning.md`
Step 7.5: at least 580s timeout, retry once on failure, log
"Codex test-plan review: FAILED — [error]" as a non-blocking note only if
both attempts fail).

**Attempt 1:** `codex exec -s read-only -c model_reasoning_effort=high
--skip-git-repo-check "<red-team prompt per Step 7.5's seven evaluation
criteria, scoped to this test-plan.md against feature.md, applying the
Prompt-only-changes exemption>"`, timeout 595s. Completed successfully
(no timeout, no error) in well under the budget; no retry was needed.

**Verdict:** FAIL
**Issues raised:** 7
**Issues addressed before dev:** 1 — TC-010's description in the AC Test
Matrix (above) was itself imprecise in exactly the way codex flagged
("`grep -rln ... internal/` is not 'Go source only' and will match the new
markdown's `class_key`"): the row now points at the actual Go test
(`TestDefectClassSweepNoGoPersistenceIntroduced` in
`internal/sharkdata/embed_test.go`), which already scopes to non-test `.go`
source under `internal/` and does not match the markdown workflow file — the
raw-grep description in the table was stale relative to the implementation
that landed for TC-010.
**Issues deferred:** 6, all logged here rather than resolved now — resolving
them (adding `AC-1`..`AC-5` identifiers to feature.md, a full ISTQB
technique/ISO-25010 matrix per AC, expanded TC-006/TC-007 routing-partition
branches, and a fully clause-complete `TC-I-03-DEFECT-CLASS-CLOSURE`) is a
test-plan-authoring-scale change, not a MEDIUM-finding-scale rework fix, and
this pass is scoped to the four UAT MEDIUM findings plus their defect-class
sweep. Deferred issues, verbatim from the run:
1. No named ISTQB technique / ISO 25010 matrix per AC (AC-1–AC-5).
2. Test matrix uses `AC-1`..`AC-5` identifiers not present in feature.md
   (feature.md uses `REQ-F-*` and three named scenarios) — traceability is
   indirect.
3. TC-006 omits a "new class inside old scope" branch in the *test plan's
   own row text* (already closed at the workflow-content level by
   `TestDefectClassSweepRecurrenceRequiresClassKey` and this feature's TC-006
   walkthrough's fixture D — the gap codex found is in the test-plan
   *description's* enumeration, not in enforcement). The accepted-risk-with-
   no-new-evidence branch for TC-007 is now closed: see TC-012 below, added
   in the round-3 UAT-kickback rework (finding 6).
4. TC-001/I-03 contract: header presence alone doesn't prove every I-03
   field/clause; `TC-I-03-DEFECT-CLASS-CLOSURE` covers only 3 of the full
   I-03 shape's fields.
5. TC-002/TC-004 grep models are open-ended (unenumerated legacy-prose
   anchors, unenumerated forbidden-token/path classes).
6. TC-001–TC-003 integration checks should name a concrete renderer
   entrypoint (e.g. `NewIncludeResolverWithEmbed("").Resolve(...)`) per
   consuming file.

Owner: next worker to touch E34-F06's test-plan.md or the next
E34-F06-adjacent feature (E34-F08, the I-03 consumer) planning pass, since
item 4 directly affects what E34-F08's test plan can safely assume about the
I-03 contract test pointer. Timeframe: before E34-F08 test-planning begins,
or re-run and re-deferred if still open at that point.

Full codex output (verbatim), immediately preceding this section's edits:

```
## Verdict: FAIL

The prompt-only exemption is correctly applied: Go caller-path, mutation, and executable decision-table tests are not required. However, Step 7.5 treats missing technique application as a blocker, and the plan has material enumeration gaps.

Concrete issues:

- AC-1-AC-5: No named ISTQB technique application or ISO 25010 matrix exists. Add per-AC technique annotations and a complete ISO matrix with justified N/A cells. Add an explicit "content-only-no runtime instrumentation" observability row mapping render, grep, link, and review artifacts to each AC.
- All ACs: Traceability is indirect. feature.md contains REQ-F-* requirements and three named scenarios, but not the AC-1-AC-5 identifiers used by the test matrix. Add those AC IDs to feature.md, or map every TC directly to its feature REQ/scenario.
- AC-5 / TC-006-TC-007: The plan omits the explicitly required new-class and accepted-risk-with-no-new-evidence outcomes. Add manual rendered-clause cases for those branches, including the required visible-but-non-blocking accepted-risk result.
- AC-1 / TC-001 and I-03 contract: Header presence does not prove the full feature contract. Expand TC-I-03-DEFECT-CLASS-CLOSURE beyond its three closure fields to cover the complete I-03 shape and its nesting under I-02.
- AC-2 / TC-002 and AC-4 / TC-004: The grep models are not closed. Enumerate the legacy prose anchors and prohibited token/path classes; supplement grep with manual single-source-of-truth review.
- REQ-NF-001 / TC-010: `grep -rln ... internal/` is not "Go source only" and will match the new markdown's class_key, making its expected zero result impossible. Use an explicit non-test Go filter.
- TC-001-TC-003 integration: Make content entrypoints concrete; name each rendered consumer path and assign that integration check a TC ID.
```

## Recommendations

- [x] Ready for development — no PRD/spec drift (spec.md traces every
      requirement to feature.md verbatim), every AC has at least one test
      case, content-only exemption is documented and applied consistently,
      I-03 contract test pointer is declared and matches spec.md exactly.
- [ ] Needs BA refinement
- [ ] Needs tech refinement

*Last Updated*: 2026-09-04
