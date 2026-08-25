---
feature_key: E34-F08-tier-consistent-gates-and-final-integration-review
epic_key: E34
title: Tier-Consistent Gates and Final Integration Review
description: Align SIMPLE, STANDARD, and COMPLEX gate evidence with executable validation, and add an epic final integration review that verifies the full merged change and cross-feature interaction closure.
---

# Tier-Consistent Gates and Final Integration Review

**Feature Key**: E34-F08

## Goal

### Problem

Quality prompts sometimes expect artifacts that a feature's complexity route
does not create, and deterministic checks are often accepted as model-written
claims rather than runner output. Per-feature reviews can also miss code and
contracts introduced in earlier features or repair rounds. E04 demonstrated
both failures: one gate reported test totals that did not match the runner,
while a later whole-diff review found canonical and predicted-debt violations
that every scoped round had missed.

### Solution

Define one artifact and gate matrix for SIMPLE, STANDARD, and COMPLEX features.
Require gate reports to cite exact project-declared commands, exit status,
runner-native counts and skips, and bounded log pointers. Add a mandatory epic
`integration_review` step before completion that reviews the accumulated diff
from a recorded immutable base, closes cross-feature interactions, and verifies
defect guards, decisions, standards, and predicted debt without silently
superseding a failed feature gate.

### Impact

Every tier is judged against artifacts it actually produces, deterministic
evidence comes from tools, and the last epic gate evaluates the integrated
system rather than the last round's delta. The shipped E40 operator can later
measure pinned E34 scenarios, but provider-backed comparison does not block
this feature.

## Research findings

- The canonical feature workflow routes SIMPLE and STANDARD through a combined
  code-review/QA step, while COMPLEX receives a separate QA step. WWGM added
  correct SIMPLE-lite and tier-aware text locally, but whole-file overrides now
  hide newer upstream staged-integration checks.
- Current review prompts tell workers to run format, lint, and tests, but they
  do not define a uniform evidence record for exact command, exit status,
  runner-native totals, expected and unexpected skips, and a retained log.
- The canonical epic workflow moves from active directly to completed after
  feature work. It has no named whole-diff integration step with authority and
  closure rules.
- E40's benchmark operator is shipped and is the right follow-up surface for
  workflow outcomes, latency, and review-round behavior. Provider-backed
  comparison evidence is validation follow-up, not a delivery dependency.

## Tier contract

| Tier | Planning source | Test source | Same-model gate | Separate QA | Final UAT |
|---|---|---|---|---|---|
| SIMPLE | `feature.md` and validated `research-report.md` | Inline task ACs and test cases | Combined code review and QA | No | Yes |
| STANDARD | `spec.md` and `test-plan.md` | Test-plan cases and caller paths | Combined code review and QA | No | Yes |
| COMPLEX | `spec.md` and `test-plan.md` | Test-plan cases and caller paths | Craft code review | Yes | Yes |

The tier determines required artifacts and gate division, not a lower evidence
standard. Missing artifacts are failures only when the selected tier requires
them.

## Requirements

1. **REQ-F-001 — Canonical tier matrix**
   - Put the tier matrix in one reusable bundle reference consumed by
     assessment, planning, task review, development, code review, QA, and UAT.
   - Remove duplicated or contradictory tier descriptions from consuming
     prompts.
   - Assert through rendered-prompt tests that each tier names exactly its
     required artifacts and gates.

2. **REQ-F-002 — Executable gate evidence**
   - Discover validation commands from project guidance such as the documented
     quality gate, build targets, package scripts, or explicit test-plan
     commands.
   - Record the exact command, working directory, exit status, runner-native
     pass/fail/error/skip counts, expected-skip comparison, and a bounded log
     or artifact pointer.
   - Reject a prose-only total, an omitted exit status, a missing declared test
     case, or an unexpected skip.
   - Keep project-specific commands and environment setup in the project; the
     Shark bundle defines the evidence contract only.

3. **REQ-F-003 — Coverage and class completeness**
   - Verify every required AC, test-plan case, caller-path contract, I-##, and
     X-## against the tier-appropriate source.
   - Consume E34-F06's completed sweep and guard evidence for prior blocking
     defect classes.
   - A passing result cannot leave an unexplained blocking finding, missing
     contract test, or unverified required guard.

4. **REQ-F-004 — Epic integration-review workflow step**
   - Add a non-terminal `integration_review` step between active work and epic
     completion in the canonical route-based epic workflow.
   - Capture an immutable epic integration-base commit when execution begins.
     Bind each review to that base, candidate head, and the exact completed or
     staged feature commits and paths included in the candidate.
   - Add immutable `.shark/runs/<epic-run-id>/integration-events/*.json`,
     immutable archived candidate heads, and one atomic
     `integration-candidate.json` head. The epic active-entry coordinator
     captures the base before first feature dispatch, each feature completion
     writes a separate event, and integration-review dispatch binds candidate
     head plus tracked and untracked path digests.
   - Compute SHA-256 over canonical JSON excluding the object's own digest.
     Serialize updates under a run-scoped lock and compare-and-swap the prior
     head digest so concurrent feature completions are additive and stale
     writers fail without data loss.
   - Review the entire accumulated diff from the recorded integration base to
     the candidate head, not only the latest round or feature.
   - Include every completed or staged feature in the review inventory and
     detect untracked changed paths.
   - Fail closed on a missing or unreachable base and define handling for
     rebases, squash-merged feature branches, interleaved unrelated commits,
     dirty tracked files, and untracked candidate paths. Do not infer scope
     from `merge-base HEAD main` after work has landed on `main`.
   - For already-active epics with no pre-execution record, require an explicit
     operator backfill of a verified base and complete feature/event inventory;
     never infer or silently migrate the identity.

5. **REQ-F-005 — Integration closure checks**
   - Verify all applicable I-## and X-## producer/consumer contracts, live
     caller paths, shared tests, staged-edge declarations, and closure owners.
   - Consume E34-F07's I-04 impact sets and confirm every amendment or linked
     follow-up is accounted for.
   - Cross-check open review findings, completed class sweeps and guards,
     ADRs, project standards, and predicted-debt records naming changed paths.
   - Add shared contract test `TC-I-01-READINESS-SYMMETRY`, which reads the
     canonical architecture field list and verifies the producer, consumer,
     Rider verb, embedded skill, and interaction-map references in one test.

6. **REQ-F-006 — Gate authority**
   - Integration review is an additional gate; it does not rewrite an
     independent feature verdict or silently convert a rejected required gate
     into acceptance.
   - Epic completion requires every current required feature gate to be
     passing or to carry a valid disposition through the existing workflow's
     decision mechanism.
   - Do not introduce a global owner-approval configuration requirement.
   - Track WWGM's historical E04-F02 shipped-while-rejected inconsistency as a
     bounded record-reconciliation item under E34-F09, separate from this
     general rule.

7. **REQ-F-007 — Structured output and adoption manifest**
   - Emit E34-F05 GateResult for each quality gate and final integration review.
   - Produce **I-05 CanonicalAdoptionManifest v1** with changed canonical
     bundle paths, workflow compatibility notes, required override actions,
     validation commands, and version/baseline evidence for E34-F09.

8. **REQ-F-008 — Benchmark follow-up**
   - Define pinned E40 scenarios for tier routing, evidence fidelity,
     defect-class recurrence, and final integration closure.
   - Do not block E34-F08 acceptance on provider-backed execution or a
     benchmark delta.

9. **REQ-NF-001 — Provider and project neutrality**
   - Do not name WWGM scripts, Python environment variables, a specific LLM,
     or host-only commands in canonical evidence policy.
   - Preserve provider-neutral outcomes and current workflow configurability.

## Implementation plan

1. Add the shared tier/evidence reference and refactor all consumers to use it.
2. Update gate output policies to require GateResult plus executable evidence
   and to consume I-03 and I-04.
3. Add `integration_review` to canonical epic workflow YAML, create its prompt,
   skill workflow, transition outcomes, and failure routing, and implement the
   immutable integration-event log, CAS candidate-head/capture service, and
   explicit legacy backfill command.
4. Implement interaction, finding, guard, ADR/standards, predicted-debt, and
   changed-path closure checks in the final review procedure.
5. Produce I-05 and add tier-route, workflow, prompt-render, full-diff,
   authority, and compatibility tests.
6. Add non-blocking pinned E40 benchmark scenario requirements through the
   shipped operator.

## Acceptance scenarios

**Review each complexity tier consistently**

- Given equivalent SIMPLE, STANDARD, and COMPLEX fixtures,
- When their rendered review and UAT prompts are inspected,
- Then each requires exactly the matrix-defined artifacts and gates,
- And all three require the same executable evidence fields.

**Reject an unverifiable test claim**

- Given a gate says all tests passed but omits the exact command or runner
  reports an unexpected skip,
- When GateResult is validated,
- Then the gate cannot advance and reports the missing evidence class.

**Review the integrated epic**

- Given every feature-level gate has passed and the epic candidate includes
  changes from several features and rework rounds,
- When `integration_review` runs,
- Then it verifies the recorded base and candidate identity, reviews the
  complete accumulated diff and closes interactions,
  impact sets, findings, guards, decisions, standards, and predicted debt,
- And it cannot use its own PASS to overwrite a required feature rejection.

## Dependencies and interactions

- Depends on E34-F06 and E34-F07 in Shark.
- Consumes **I-02 GateResult v1**, **I-03 DefectClassSweep v1**, and
  **I-04 ChangeImpactSet v1**.
- Produces **I-05 CanonicalAdoptionManifest v1** for E34-F09.
- Reuses E34-F03 staged integration and interaction closure semantics.

## Out of scope

- Adding separate QA to every STANDARD feature.
- Numeric round-based escalation or automatic owner interruption.
- WWGM validation scripts, database setup, lint configuration, or model
  assignments.
- Changing historical E04 lifecycle records in this repository.
- Requiring provider-backed E40 comparison before implementation.

## Verification plan

- Exercise every tier route and assert required/forbidden artifact references.
- Render changed prompts and validate bundle references and workflow YAML.
- Test exact command evidence, nonzero exit, mismatched totals, missing tests,
  expected skips, and unexpected skips.
- Test integration review over multi-feature accumulated diffs, untracked
  changed paths, open I/X edges, stale decisions, incomplete sweeps, missing
  guards, and a rejected feature gate.
- Test pinned-base histories with independently squash-merged features,
  unrelated interleaved commits, rebases, a missing base, dirty tracked files,
  and untracked candidate paths.
- Race two feature-completion events and prove both survive; reject a stale CAS
  writer. Verify canonical digests, archived-head links, tamper detection,
  truncation, reordering, and crash recovery.
- Implement `TC-I-01-READINESS-SYMMETRY` as the structural guard for the full
  I-01 producer/consumer reference surface.
- Run `make fmt`, `make lint`, `make test`, and `git diff --check`.

*Last Updated*: 2026-08-05
