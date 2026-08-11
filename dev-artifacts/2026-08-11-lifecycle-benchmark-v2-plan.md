# Lifecycle benchmark v2: realistic Shark workflow evaluation

## Planning status

The approved backlog placement was applied on 2026-08-11. E40-F05 through
E40-F10 now exist as draft features under E40. Their creation reopened E40 to
`active`; E40-F01 through E40-F04 remain completed.

The recommended Shark home is **multiple new features under E40**, not a new
epic. E40 already owns end-to-end workflow benchmarking, and its design and
epic documents explicitly reserve later phases for feature Mode A/Mode B,
change-card and tech-debt scenarios, configuration comparisons, and epic-level
work. The four existing E40 features remain the completed v1 foundation.

Creating E40-F05 automatically reopened the terminal epic through Shark's
creation-trigger parent maintenance. Preserve that event in entity history and
keep E40 open until all accepted v2 features complete.

The rejected alternative was a new epic that preserved E40 as an immutable v1
milestone. The owner selected the ongoing-program model instead.

## Triage observations and implemented feature breakdown

The live Shark inventory contained no duplicate lifecycle-benchmark entity.
Before placement, E40 was completed with four completed features and 32
completed tasks, but its charter already covered this work. The new children
now keep E40 active. Existing E40 tech-debt items are narrow harness defects,
not alternate parents for v2. Link any applicable item as a prerequisite rather
than duplicating it; for example, shared parser work should consume TD-084 if
another bench script triggers its stated threshold.

The following outcome-oriented features now exist under E40:

| Order | Feature | Demonstrable outcome | Principal scope | Depends on |
|---:|---|---|---|---|
| 5 | E40-F05 — Lifecycle scenario corpus and adapter contract | Four versioned scenario families load and run against the controlled Python fixture without embedding language assumptions in Shark's workflow contracts | Scenario package schema, applicability matrix, Python fixture and adapter, fixture identity, seeded feature/bug/change-card/tech-debt cases, Go compatibility-adapter boundary | Completed E40 corpus and harness contracts |
| 6 | E40-F06 — Stage evidence and evaluator isolation | Every applicable stage emits traceable evidence while hidden references and execution truth remain unavailable to workers | Agent-visible, scratch, and evaluator-only roots; stage snapshot and lineage schema; prompt and artifact digests; admission checks; held-back execution-oracle boundary | E40-F05 |
| 7 | E40-F07 — Replayable product-design prelude | A feature scenario reproducibly executes its real D01-D05 Rider prelude without live human or network input | Host-side product-design controller, scripted stakeholder responses, frozen research responses, human-question/research routing, non-applicable-stage semantics, replay digests | E40-F05; X-10 |
| 8 | E40-F08 — Canonical multi-entity lifecycle runner | Every eligible entity produced by a scenario is driven through its real keyed lifecycle with bounded, auditable outcomes | `next` dispatch, deterministic fork scheduling, claims, heartbeats, exact prompt handoff, semantic outcomes, transitions, release, Questions and human gates, all-eligible-task execution, resource ceilings and named stop outcomes | E40-F05, E40-F06, E40-F07; X-11, X-13 |
| 9 | E40-F09 — Calibrated evaluation and comparison identity | A run can be accepted or rejected using reproducible quality evidence, and incompatible runs cannot be aggregated | Structural evaluation, evaluator-only reference comparison, calibrated LLM judge, execution-oracle result, complete provenance identity, invalid-inventory and aggregate-rejection rules | E40-F06, E40-F08; X-12 |
| 10 | E40-F10 — Operator workflow and retained lifecycle baseline | An operator can preview, pilot, execute, inspect, and report a provider-backed baseline without accidental spend | Dry-run/spend gates, pilot and baseline commands, per-stage diagnostics, lifecycle headline report, machine-readable aggregates, retained raw artifacts, noise bands and publication gate | E40-F08, E40-F09 |

Keep benchmark-owned adapters, replays, scenarios, evaluators, reports, and
operator commands in these E40 features. If implementation exposes a missing
generic Shark or Rider capability, triage that production change separately
under the epic that owns the affected runtime contract and link it as a
dependency. Do not move the benchmark wrapper itself into a runner, prompt, or
workflow epic merely because it calls those surfaces.

## Goal

Create a language-agnostic lifecycle benchmark for Shark. Treat E40 as the
harness foundation and its seeded task corpus as low-cost harness acceptance
coverage, not as the product baseline.

Use a controlled Python task-manager fixture for the first benchmark project.
Evaluate fixed scenario inputs through each scenario family's real Shark
lifecycle, including applicable planning artifacts, decomposition, execution,
and review. Start with feature delivery, bug investigation, change requests,
and tech-debt remediation.

Do not use live or historical OSS issues as the primary corpus. Use them only
as design inspiration. Keep the benchmark reproducible,
privately-oracle-backed, and resistant to task-quality and contamination
problems in issue and pull-request datasets.

## Define each scenario lifecycle

Use an explicit stage matrix instead of forcing every entity family through
the same artifacts:

| Scenario family | Initial input | Evaluated lifecycle |
|---|---|---|
| Feature delivery | Scripted stakeholder brief and evidence bundle | D01 through D05 via the product-design Rider action, then the applicable epic and feature planning, decomposition, child-task execution, and review gates |
| Bug investigation | Bug seed with a reproducible symptom | Bug research, development, code review, and QA |
| Change request | Change-card seed with a machine-checkable acceptance predicate | Change research, development, code review, and QA |
| Tech-debt remediation | Tech-debt seed with current-state evidence and a target condition | Tech-debt research, triage, implementation or resume, and resolution |

Record non-applicable stages explicitly. Do not classify an intentionally
non-applicable D-artifact as missing for bug, change-request, or tech-debt
scenarios.

The benchmark has two orchestration layers:

1. The host-side product-design controller runs the D01 through D05 prelude for
   feature-delivery scenarios. It records Rider progress and artifact lineage.
2. The keyed lifecycle controller starts from the scenario's created Shark
   entity and drives that entity and its descendants through the canonical
   parent loop.

Use a versioned replay bundle for every interaction that would otherwise depend
on mutable outside state. The bundle supplies scripted stakeholder responses,
interview or proxy-research evidence, and frozen research-tool responses. Route
`AskUserQuestion` and research requests through the replay adapter. Disable live
network research during scored runs. Hash every replay input and record which
response each stage consumed.

## Create the lifecycle corpus

- Implement the lifecycle corpus through E40-F05. Its creation reopened E40 and
  preserved the completed v1 features as historical foundation.
- Add a versioned scenario package for each benchmark case. Each package
  contains:
  - A scenario ID and version.
  - A stage-applicability matrix and actual initial entity family.
  - A fixed project fixture SHA and language-specific execution adapter.
  - An agent-visible initial input appropriate to the entity family.
  - A harness-only replay bundle that reveals only the current authorized
    stakeholder, evidence, or research response through the replay adapter.
  - An evaluator-only set of approved reference artifacts for each applicable
    planning and decomposition stage.
  - An evaluator-only implementation oracle and reference patch used only
    after execution.
- Keep every approved reference artifact, judge answer key, reference patch,
  and hidden test in an evaluator-only root. Never copy or mount that root into
  the agent-visible scratch project or fixture checkout.
- Add admission checks that prove evaluator-only files are absent at every
  dispatch boundary and become readable only by post-stage or post-run
  evaluation.
- Use a controlled Python task-manager fixture first. Keep language-specific
  skills with the fixture and adapter; keep Shark's workflow and planning
  contracts language-agnostic.

## Run and capture the real workflow

For each keyed dispatch, the lifecycle runner must:

1. Call `shark next <key> --json` and preserve the returned response.
2. Resolve a hierarchy fork through a recorded deterministic scheduling policy,
   then dispatch every eligible candidate.
3. Claim the returned concrete entity before starting its worker.
4. Send `response.prompt` to the worker unchanged.
5. Heartbeat the claim while the worker or a routed consultation is active.
6. Persist the worker's bounded evidence and semantic outcome.
7. Apply the configured outcome transition and record the resulting status.
8. Release the claim on every success, failure, cancellation, or exception
   path.

Stop and record a named outcome on `pause`, `archive`, `error`, lease loss,
missing semantic outcome, or an unresolved human gate. Satisfy Questions and
human decisions only from the scenario's versioned response bundle. If the
bundle has no authorized response, record `unresolved_gate`; do not invent an
answer or force a transition.

Capture the rendered prompt and its digest, generated document, consumed replay
inputs, entity graph, claim and status history, agent transcript, semantic
outcome, usage, cost, elapsed time, and artifact digest after every stage.

Execute every eligible task the decomposition generates. Record an explicit
reason for every skipped or ineligible task; do not replace the generated plan
with a fixed task budget. Apply scenario-level safety ceilings for provider
cost, wall time, and generated task count. If a ceiling is reached, stop the
scenario with `resource_limit`, mark it invalid for baseline publication, and
retain all partial evidence. Do not publish a silently truncated plan.

Retain E40's split-root isolation. Keep Shark state, planning documents,
transcripts, and run logs in the scratch project; mirror only the planning
context required by the worker into the fixture checkout. Keep evaluator-only
references and oracles outside both agent-visible roots until their evaluation
phase.

## Evaluate artifacts and outcomes

Freeze and compare snapshots for every applicable stage in the scenario's
matrix. Each snapshot records its input and response lineage, content digest,
tokens, cost, elapsed time, output LOC, rejection or rework count, and errors.

Evaluate each applicable stage with both mechanisms:

1. Deterministic structural checks for required artifacts, entity ownership,
   links, dependencies, status transitions, traceability, and executable-task
   eligibility.
2. A versioned, calibrated LLM-judge rubric that compares generated artifacts
   against evaluator-only references and records its model, configuration,
   rubric and prompt digests, rationale, score, usage, and cost.

Do not treat a workflow status advance, worker self-report, or terminal outcome
as artifact approval or execution correctness. The held-back execution oracle
is the authoritative implementation-quality signal.

Publish two views from the same lifecycle data:

- **Lifecycle baseline:** Per-scenario quality, cost, elapsed time, rework, and
  final oracle outcomes across the scenario's applicable lifecycle. This is the
  headline result.
- **Stage diagnostic view:** Replays of frozen, genuinely Shark-generated stage
  inputs, including decomposition snapshots. Use this view to isolate a
  planning regression from an execution regression; do not present it as a
  separate product baseline.

## Pin comparison identity

Record and require uniform values for every comparison identity field:

- Scenario ID and version, stage matrix, and agent-visible input or replay
  digest.
- Fixture SHA, adapter version, language toolchain, and execution-environment
  identity.
- Shark version and binary digest.
- A complete installed-content digest covering workflows, prompts, skills, and
  agents.
- Per-stage provider, model, effort, and exact rendered-prompt digest.
- Judge model and configuration plus rubric, prompt, and reference digests.
- Resource ceilings and other experiment flags that can change execution.

Reject aggregates when any required identity field is missing or when
contributing runs disagree. Do not publish task, scenario, or corpus bands for
non-uniform inputs. Keep the invalid inventory and divergence reasons so the
operator can diagnose the rejected batch.

## Add operator commands

Add commands that require explicit output roots and never silently incur
provider spend:

```bash
make benchmark-lifecycle-dry-run BENCH_OUT=/path/to/preview
make benchmark-lifecycle-pilot BENCH_OUT=/path/to/pilot \
  ALLOW_PROVIDER_SPEND=1 MAX_PROVIDER_USD=10 MAX_WALL_SECONDS=3600 \
  MAX_GENERATED_TASKS=20
make benchmark-lifecycle BENCH_OUT=/path/to/baseline \
  ALLOW_PROVIDER_SPEND=1 MAX_PROVIDER_USD=100 MAX_WALL_SECONDS=14400 \
  MAX_GENERATED_TASKS=20
make benchmark-lifecycle-report BENCH_OUT=/path/to/baseline
```

Treat the numeric values above as operator examples, not benchmark defaults.
Provider-backed commands must refuse to start unless the operator supplies the
spend acknowledgement and positive resource ceilings. The dry-run command
must show the scenario matrix, selected stages, planned provider calls, and
resource ceilings without making provider calls. The report command must read
persisted results and must not call a provider.

The report command writes machine-readable aggregate JSON and a human-readable
artifact comparison report.

## Verify the benchmark

- Test every scenario family's initial entity, applicable stages, and explicit
  non-applicable-stage handling.
- Test the product-design replay adapter with scripted stakeholder, evidence,
  and research responses. Prove scored runs cannot reach live network research
  or unrecorded human input.
- Test the keyed parent loop: fork selection, claim, exact prompt handoff,
  heartbeat, semantic outcome persistence, configured transition, and release
  on every exit path.
- Test pause, archive, error, Question routing, unresolved human gates, lease
  loss, worker failure, missing outcomes, and rework transitions.
- Test that every selected stage produces its matching snapshot.
- Test missing or invalid applicable D01 through D05 artifacts, broken entity
  links, unsupported transitions, and non-executable generated tasks. Each
  failure must be named by structural evaluation.
- Test adapter isolation so language-specific commands cannot leak into
  Shark's generic workflow layer.
- Test that all evaluator-only references, judge answer keys, reference
  patches, and hidden execution tests remain unavailable until their evaluation
  phase.
- Test cost, wall-time, and generated-task ceilings. A breached ceiling must
  produce `resource_limit`, retain partial evidence, and prevent baseline
  publication.
- Reject aggregates that mix or omit any comparison identity field.
- Calibrate the judge against a small human-scored set before it can gate
  baseline publication. Persist disagreements for rubric refinement and do
  not retune against the held-out evaluation set.
- Run one retained provider-backed pilot for each scenario family before any
  repeated baseline. Inspect raw stage artifacts, judge rationale, generated
  graph, transcripts, claim and transition history, and post-run oracle
  evidence.

Establish the first baseline only when every selected scenario has complete
applicable-stage lineage, uniform comparison identity, passing structural
evaluation, calibrated judge evidence, and an execution oracle result. Publish
stage-level noise bands from repeated runs. Compare future changes only against
the matching scenario, stage, adapter, and provenance band.

## Assumptions

- Python is the first execution adapter. The existing Go harness becomes a
  compatibility adapter, not the benchmark's global assumption.
- Feature-delivery scenarios own the D01 through D05 product-design prelude.
  Bug, change-request, and tech-debt scenarios start from their keyed entity.
- All eligible generated tasks execute unless a scenario-level safety ceiling
  invalidates and stops the whole scenario. Reports normalize at scenario level
  and record task-count variation.
- E40 task and bug items remain low-cost harness regression coverage.
- The new lifecycle features extend E40's runner and collection surface; they
  do not redefine whether the four completed v1 features were delivered.
