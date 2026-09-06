---
feature_key: E40-F11-six-family-lifecycle-benchmark-readiness-and-cover
epic_key: E40
title: Six-family lifecycle benchmark readiness and coverage
description: Repair the E40 lifecycle benchmark blockers found during the 2026-09-04 full-family preflight and expand I-04 from four admitted families to all six delivery families: epic, feature, task, bug, change_card, and tech_debt. Preserve the retained failed-run evidence, make preflight fail closed, add spend-gated I-06 replay preparation, add real epic-root and task-root scenarios, and produce a fresh verified six-scenario baseline only after explicit provider-ceiling approval.
---

# Six-family lifecycle benchmark readiness and coverage

**Feature Key**: E40-F11-six-family-lifecycle-benchmark-readiness-and-cover

## Goal

Make the E40 lifecycle benchmark truthful, runnable, and complete across the six
delivery entity families: `epic`, `feature`, `task`, `bug`, `change_card`, and
`tech_debt`.

### Problem

The 2026-09-04 operator preflight selected the four families currently admitted
by I-04, but it exposed runtime and contract gaps that prevent a publication-
eligible baseline. Evaluator test-identity collection used the repository's
checked-out Python submodule instead of the package's admitted fixture SHA, the
feature scenario had no genuine I-06 replay result, and preflight reported
provider readiness even though the feature stopped at `unresolved_gate` and two
other scenarios failed stage resolution.

The benchmark also calls its current matrix “full-family” even though I-04
admits only `feature`, `bug`, `change_card`, and `tech_debt`. Shark has real
`epic` and `task` workflows, but setup creates an epic only as a feature parent
and does not create a task-root scenario. Generated tasks inside the feature
scenario do not replace an independently admitted task-root lifecycle scenario.

### Solution

Repair preflight and replay preparation first, then version the I-04 contract
for six delivery families. Add bounded epic-root and task-root packages with
held-back correctness checks and structural entity-graph assertions. Generalize
operator setup, spend planning, pilot attestations, aggregation, reports, tests,
and documentation. Capture a new baseline only after the six-scenario zero-
provider preflight passes and the operator explicitly approves provider limits.

### Impact

- Prevent provider spend when selected scenarios cannot run or required replay
  inputs are absent.
- Bind every evaluator and fixture claim to the admitted immutable identity.
- Replace ambiguous “full-family” language with a six-family delivery matrix.
- Produce one independently evaluable root scenario for every delivery family.
- Preserve failed and incomplete evidence without silently retrying or
  overwriting it.

## Observed evidence

The failed capture is evidence for this feature, not a baseline to retry or
promote.

- Candidate branch: `fix/e40-benchmark-capture`
- Candidate HEAD: `29566e4dbc6f4e84fd1056b7b00eaa0533e6bffb`
- Retained operator root:
  `/home/jwwel/e40-benchmarks/2026-09-04-e40-full-family`
- Preflight result:
  `/home/jwwel/e40-benchmarks/2026-09-04-e40-full-family/preflight/baseline/preflight-result.json`
- Preflight result SHA-256:
  `4c49f8e4839ffaf2371526a3f35fdc3e1afad3796bdcff414ed9dcb03fe02111`
- Preflight selected exactly four scenarios and made zero provider calls and
  zero live-database mutations.
- Preflight returned `pass_with_dry_run_limitations` with
  `dry_run_stage_resolution_failures=2` and
  `missing_real_runtime_inputs=[]`.
- `py-bug-due-date-boundary` resolved one dispatch but ended with
  `terminal=error` in dry-run mode.
- `py-change-priority-scale` failed evaluator collection with
  `ModuleNotFoundError: No module named 'taskmanager.legacy_records'`.
- `py-feature-recurring-tasks` resolved zero dispatches and ended with
  `terminal=unresolved_gate` because no genuine I-06 replay result was
  configured.
- `py-techdebt-consolidate-validation` failed evaluator collection with
  `ModuleNotFoundError: No module named 'taskmanager.bulk_import'`.
- The repository submodule was checked out at
  `2b030a9982a54473b023aebaaa35fe18bb5c42c6`, while every admitted scenario
  pins fixture SHA `964fa68e4c9e0c4e0f3756d9efd78b888c558fd9`. Both missing modules exist at
  the admitted SHA.
- Change-card diagnostic log:
  `/home/jwwel/e40-benchmarks/2026-09-04-e40-full-family/diagnostics/preflight-repro/change-run-lifecycle.log`
  with SHA-256
  `c14d4fa9de73a6d95a46aa5e1da3cba2838ecd7154b2f39909ea33fe08f9d845`.
- Tech-debt diagnostic log:
  `/home/jwwel/e40-benchmarks/2026-09-04-e40-full-family/diagnostics/preflight-repro/techdebt-run-lifecycle.log`
  with SHA-256
  `16be7b7799b678ad2a1a36d6a3b9d520d26fe68d172c132c7b2e42ac7790e196`.
- Candidate validation passed before preflight:
  - `validation/run-all.log` SHA-256
    `cd4756da9c471760c2a9f46130837336df42cb7f738e8772d3174bf46c17bdf9`
  - `validation/quality-gate.log` SHA-256
    `ab8626bb3189125c768cb98238c36a61810295f414e318301e89f3534bd12fa0`

## User stories

### Run a truthful preflight

As a benchmark operator, I want preflight to validate every selected scenario
and required runtime input without provider calls so that I can approve spend
from complete evidence.

### Benchmark every delivery family

As a Shark maintainer, I want independent epic-root and task-root scenarios in
addition to the existing four families so that “full-family” means all six
delivery lifecycles.

### Review publication evidence

As a reviewer, I want every scenario and repetition bound to exact fixture,
adapter, workflow, candidate, replay, and resource identities so that command
completion cannot be mistaken for benchmark validity.

## Functional requirements

### Preserve failed-run evidence

**REQ-F-001**: Preserve the existing 2026-09-04 operator root byte-for-byte.

The freeze is **unconditional**. It does not depend on the retained result's
recorded `status`, on whether `candidate_identity` matches the current
candidate, or on which subcommand is invoked. A registered root is frozen
because it is registered, not because it is incomplete.

- Register the root in a committed retention registry, by path and by a
  recorded path -> SHA-256 manifest of its whole tree.
- Every operator subcommand that acquires the operator-root lock or writes a
  byte under the operator root must refuse to run against a registered root:
  `setup`, `preflight`, `prepare-replay`, `pilot`, `baseline`, `variant`,
  `validate-variant`, `compare`, and `demo`.
- Do not use `--retry-incomplete` or any equivalent reclaim option on it.
- Do not promote it to a pilot or baseline.
- Classify the current result as `incomplete`.
- Prove whole-tree invariance by recomputing the manifest after every
  validation run and requiring it to match the registered manifest exactly --
  not merely that a single result file is unchanged.
- Use a new external operator root after implementation changes alter candidate,
  scenario, package, schema, fixture, or workflow identities.
- Keep the registered absolute path out of `bench/` source: `bench/` reads the
  registry, it does not embed an operator-specific path.

### Bind evaluator work to the admitted fixture

**REQ-F-002**: Run evaluator disclosure and test-identity collection against an
immutable checkout of each package's `fixture.base_sha`, never against the
repository submodule's incidental working-tree HEAD.

- Pass the checkout path explicitly through every collector caller.
- Verify the checkout SHA before collection.
- Keep the repository submodule byte-unchanged.
- Retain the fixture SHA and adapter identity in preflight and run provenance.
- Add a regression case where submodule HEAD differs from `fixture.base_sha`.

### Prepare I-06 replay through an explicit spend gate

**REQ-F-003**: Add one operator-owned replay preparation flow for scenarios with
an applicable I-06 prelude.

The flow must:

1. Preview the proposed provider calls and resource ceilings without calling a
   provider.
2. Require explicit provider-spend acknowledgement **and** all four strictly
   positive operator-supplied ceilings -- maximum cost, maximum wall-clock
   seconds, maximum provider calls, and **maximum generated tasks**. A missing,
   non-numeric, boolean, zero, or negative value for any of the four is a
   rejection, not a defaulted value.
3. Run the genuine replay producer rather than a fixture or hand-authored
   result.
4. Retain the transcript, result, usage, limits, and validation output.
5. Run `bench/scripts/verify-replay-result.sh` against the admitted reference
   bundle.
6. Expose the verified absolute result path for the generated operator config.

Ordinary preflight must only validate an existing replay. It must never create
one or make a provider call.

### Make preflight fail closed

**REQ-F-004**: A preflight may return `status=pass` only when every selected
scenario resolves successfully and every required runtime input is present and
valid.

Preflight must block on:

- Missing or invalid replay results.
- Missing lifecycle adapters or provider commands required for the approved
  execution mode.
- Evaluator import or test-identity collection failures.
- `terminal=error`, `unresolved_gate`, or any other non-successful stage
  resolution.
- Missing, unsafe, or identity-mismatched scenario roots and fixture checkouts.

`pass_with_dry_run_limitations` is diagnostic information, not an acceptable
spend gate for pilot or baseline execution.

### Resolve lifecycle stages without live-worker artifacts

**REQ-F-005**: Add a deterministic stage-resolution-only path that traces the
configured success route without requiring files that only a live worker can
produce.

- Keep evidence validation active in live pilot and baseline runs.
- Distinguish a route-resolution defect from an expected absence of live
  artifacts.
- Report each selected route, provider, model, effort, and terminal state.
- Do not synthesize evidence that could later be mistaken for live evidence.

### Produce an honest provider-call plan

**REQ-F-006**: Preflight must produce a conservative call plan before spend
approval.

The plan must separate:

- Fixed replay-prelude calls.
- Fixed root-lifecycle calls.
- Review-gate calls.
- Descendant calls bounded by the package's declared entity graph and generated-
  task ceiling.
- The total proposed call ceiling, cost ceiling, wall-clock ceiling, and
  generated-task ceiling.
- The package, workflow, model, provider, and resource-policy evidence used to
  derive each limit.

Do not present a dynamic estimate as an exact count. Present fixed calls, bounded
calls, and the resulting worst-case ceiling separately.

### Version I-04 for six delivery families

**REQ-F-007**: Bump the I-04 schema version and extend its closed entity-family
vocabulary to exactly:

- `epic`
- `feature`
- `task`
- `bug`
- `change_card`
- `tech_debt`

Update every package, schema validator, admission rule, predicate mapping,
consumer, test, README section, and demo statement that assumes four families.

`sprint` and `question` remain orchestration and control workflows, not delivery-
scenario roots. Document this boundary so “full-family” is unambiguous.

### Preserve the feature-only product-design prelude

**REQ-F-008**: Keep D01-D05 applicable to `feature` scenarios only unless a new
architecture decision explicitly broadens I-06.

- `feature`: D01-D05 must be `applicable: true`.
- `epic`, `task`, `bug`, `change_card`, and `tech_debt`: D01-D05 must be
  `applicable: false` with a non-empty family-specific reason.
- Avoid duplicating D01-D05 inside the epic workflow's own assessment,
  refinement, research, design, and decomposition stages.

### Declare expected entity graphs

**REQ-F-009**: Extend the scenario package contract with a declarative expected-
entity-graph shape for hierarchical scenarios.

The contract must support:

- Root family.
- Allowed and required descendant families.
- Minimum and maximum descendant counts.
- Required terminal states.
- Whether unexpected descendants invalidate the run.
- Required planning-artifact and provenance assertions.

Use this contract both for structural evaluation and conservative provider-call
planning.

### Add a task-root scenario

**REQ-F-010**: Admit one bounded task-root Python scenario. The recommended seed
is `py-task-delete-task`.

- Seed a dedicated host epic and host feature, then run the target task as the
  scenario root.
- Exercise the real task `draft -> development -> completed` route.
- Expect no generated descendants.
- Add held-back tests for successful deletion, missing-ID behavior, and existing
  behavior regressions.
- Add a task-specific final predicate, such as `task_acceptance_tests`, without
  weakening the absolute P2P clause.
- Require base predicate false, reference predicate true, and full fixture P2P
  green.

The exact fixture behavior may change during curation if admission evidence
shows a better bounded task, but the replacement must remain atomic and distinct
from the four existing scenarios.

### Add an epic-root scenario

**REQ-F-011**: Admit one bounded epic-root Python scenario. The recommended seed
is `py-epic-task-organization`.

- Exercise epic assessment, refinement, research, design, decomposition,
  feature review, active cascade, and terminal completion.
- Require exactly two intended features and a bounded task set, with every
  required descendant reaching a terminal state.
- Use a small integrated fixture outcome, such as task tagging plus filtered
  listing, that requires two coherent feature slices without becoming an
  unbounded product build.
- Add a descendant-oracle union that combines held-back implementation tests
  with the expected-entity-graph and required-artifact assertions.
- Reject missing, extra, non-terminal, or provenance-incomplete descendants.
- Require base predicate false, reference predicate true, and full fixture P2P
  green.

The exact fixture outcome may change during curation, but the scenario must
remain bounded, multi-feature, independently evaluable, and distinct from the
feature-root recurring-tasks scenario.

### Isolate every scenario root

**REQ-F-012**: Key setup roots by `scenario_id`, not by `entity_family`, and seed
an independent hierarchy for every scenario.

- Epic scenario: dedicated target epic.
- Feature scenario: dedicated host epic and target feature.
- Task scenario: dedicated host epic, host feature, and target task.
- Bug, change card, and tech debt: dedicated standalone roots.
- Do not reuse an epic or feature across scenarios.
- Capture every assigned key from Shark's JSON response.
- Keep benchmark state inside the external scratch project; never mutate the
  repository's live `shark-tasks.db` during benchmark execution.

This removes the current one-package-per-family dictionary assumption and keeps
future same-family scenarios from silently overwriting each other.

### Generalize pilot, retention, aggregation, and reports

**REQ-F-013**: Make all downstream lifecycle-v2 consumers work with the six-
family matrix without private family lists.

- Require one inspected pilot-ledger attestation per selected family.
- Preserve all scenario/repetition evidence and manifest provenance.
- Aggregate and report every selected family independently.
- Keep zero findings distinct from missing measurements.
- Preserve produced, consumed, reused, and orphan artifact counts.
- Preserve noise bands and insufficient-repetition warnings.
- Name every unavailable metric and upstream contract gap.

### Capture a fresh baseline

**REQ-F-014**: After implementation and validation, capture a new six-scenario
baseline through the normal spend gates.

1. Run a zero-provider preflight without a scenario filter.
2. Require exactly six selected scenarios and six distinct delivery families.
3. Show the provider-call plan and proposed resource ceilings with evidence.
4. Wait for explicit operator approval.
5. Run one pilot repetition for all six scenarios.
6. Inspect every pilot pair and record six factual ledger attestations.
7. Verify the pilot ledger.
8. Reuse pilot repetition 1 and add repetitions 2 and 3.
9. Require all 18 scenario/repetition pairs.
10. Run the offline retention verifier and independently validate every
    `lifecycle.jsonl` and `evaluation.jsonl`.
11. Require completed operator status, `publication_eligible=true`,
    `aggregate.json.invalid=[]`, and eligibility for every pair.
12. Reconcile `reports/headline.md`, `reports/stage-diagnostic.md`,
    `aggregate.json`, and retained raw evidence.

Do not claim improvement or regression without a controlled comparison using
the same scenario matrix, repetitions, identities, and resource policy.

## Non-functional requirements

### Safety

**REQ-NF-001**: Do not read or mutate the repository's live benchmark database
during setup, preflight, pilot, baseline, replay preparation, verification, or
reporting. Use new external roots and isolated scratch databases.

### Spend control

**REQ-NF-002**: Preflight and replay preview must make zero provider calls.
Provider-backed work requires explicit acknowledgement and strictly positive,
operator-approved cost, wall-clock, generated-task, and provider-call ceilings.

### Evidence retention

**REQ-NF-003**: Preserve incomplete and failed artifacts. Never retry or reclaim
an incomplete pair until an operator has inspected its retained failure and
recorded why retrying is appropriate.

### Identity and determinism

**REQ-NF-004**: Pin and verify scenario, package, fixture, adapter, toolchain,
candidate, Shark binary, content, prompt, workflow, policy, replay, model,
provider, effort, resource, and repetition identities.

### Isolation

**REQ-NF-005**: Keep agent-visible fixture code, scratch Shark state, and
evaluator-only material in separate roots. Reject symlink, traversal, identity,
or provenance escapes at every read and write boundary.

## Acceptance scenarios

### Reject incidental submodule identity

- **Given** the repository's `bench/fixture-py` checkout differs from a
  package's admitted `fixture.base_sha`
- **When** preflight collects evaluator test identities
- **Then** it uses an immutable checkout at the admitted SHA
- **And** both `taskmanager.legacy_records` and `taskmanager.bulk_import`
  collect successfully
- **And** it does not modify the repository submodule.

### Block a missing replay

- **Given** a selected feature scenario has no configured replay result
- **When** the operator runs preflight
- **Then** preflight makes zero provider calls
- **And** returns a blocked result naming the missing I-06 input
- **And** does not report provider readiness.

### Pass a complete six-family preflight

- **Given** all six packages are admitted and all runtime inputs are valid
- **When** the operator runs preflight without a scenario filter
- **Then** exactly six scenarios and six families are selected
- **And** every stage route resolves successfully
- **And** D01-D05 are applicable only to the feature scenario
- **And** provider calls and live-database mutations both equal zero
- **And** the output includes the fixed, bounded, and worst-case provider-call
  counts plus supporting identities.

### Complete a task-root lifecycle

- **Given** the admitted task scenario's dedicated hierarchy and fixture
- **When** the lifecycle runner executes the task root
- **Then** the task reaches `completed`
- **And** the held-back oracle runs only after terminal completion and passes
- **And** no unexpected descendant or candidate change exists.

### Complete an epic-root lifecycle

- **Given** the admitted epic scenario's expected entity graph
- **When** the lifecycle runner executes the epic root and descendants
- **Then** the epic and every required descendant reach terminal completion
- **And** the exact feature/task topology and required planning artifacts pass
  structural evaluation
- **And** the held-back integrated implementation oracle passes after terminal
  completion.

### Publish a six-family baseline

- **Given** a verified six-family pilot and explicit baseline ceiling approval
- **When** the operator completes three repetitions per scenario
- **Then** all 18 pairs are present and independently eligible
- **And** the operator result, manifest, aggregate, pilot ledger, reports, and
  raw evidence reconcile
- **And** no invalidity reason remains.

## Required validation

Run these gates in order:

1. Add focused regression tests for admitted-fixture checkout binding, missing
   replay readiness, fail-closed stage resolution, provider-call planning,
   six-family admission, scenario-root isolation, and six-family reporting.
2. Admit all six packages twice against the same fixture and toolchain; require
   byte-identical results.
3. Run `bash bench/scripts/tests/run-all.sh`.
4. Run `make fmt && make lint && make test`.
5. Record the exact HEAD and SHA-256 digest of every validation log.
6. Run setup under a new external operator root.
7. Run the zero-provider six-family preflight and inspect its structured output.
8. Stop before provider-backed work and request explicit ceiling approval.

## Implementation sequence

1. Preserve and register the failed-run evidence.
2. Add failing regression tests for fixture identity and preflight readiness.
3. Fix admitted-fixture checkout propagation.
4. Add the spend-gated replay preparation flow.
5. Separate route resolution from live evidence validation.
6. Make preflight fail closed and add the conservative call plan.
7. Version I-04 and add the entity-graph contract.
8. Curate and admit the task-root scenario.
9. Curate and admit the epic-root scenario.
10. Generalize setup, retention, ledger, aggregate, report, and documentation
    surfaces.
11. Run the complete offline validation suite.
12. Run a fresh preflight, request spend approval, pilot, inspect, attest, and
    capture the 18-pair baseline.

## Out of scope

- Running provider-backed replay, pilot, or baseline work during feature
  capture or planning.
- Reusing, modifying, retrying, or promoting
  `/home/jwwel/e40-benchmarks/2026-09-04-e40-full-family`.
- Comparing the new baseline with another variant before a controlled variant
  with matching identities and resource policy exists.
- Treating `sprint` or `question` as delivery-scenario roots.
- Creating implementation tasks during triage. Task generation belongs to the
  feature workflow after assessment, research, specification, and test
  planning.

## Definition of done

- I-04 admits exactly six delivery families with one real seed per family.
- Epic and task packages pass schema, admission, isolation, predicate, and
  determinism checks.
- Preflight fails on every reproduced 2026-09-04 blocker and passes when each
  blocker is resolved.
- Preflight makes zero provider calls and reports an evidence-backed worst-case
  call and resource ceiling.
- Full benchmark and repository quality gates pass at the exact candidate HEAD.
- A six-family pilot has six inspected and verified attestations.
- A three-repetition baseline contains 18 eligible pairs.
- `operator-result.json` reports `status=completed` and
  `publication_eligible=true`.
- `benchmark-manifest.json` records all six scenarios with `reps=3`.
- `aggregate.json` contains `invalid=[]`.
- `reports/headline.md` and `reports/stage-diagnostic.md` exist and reconcile
  with retained evidence.
- The final verdict is based on eligibility evidence, not command completion.

*Last Updated*: 2026-09-04
