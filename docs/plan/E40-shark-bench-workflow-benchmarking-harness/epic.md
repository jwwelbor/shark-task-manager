---
epic_key: E40
title: Shark Bench: workflow benchmarking harness
description: Benchmark program around Shark that measures workflow quality, cost, and lifecycle behavior from reproducible scenario inputs through planning, execution, review, and held-back evaluation. Phase 1 uses shark run for task and bug baselines; lifecycle v2 adds replayable product-design inputs, a canonical keyed Rider loop, stage evidence, exact candidate and policy identity, artifact-use and structured-finding telemetry, calibrated evaluation, controlled QA-versus-deep-review comparisons, and retained multi-entity baselines.
---

# Shark Bench: workflow benchmarking harness

**Epic Key**: E40 · **Last updated**: 2026-08-13

This document is the single source of business context for E40. Features reference
it; they must not restate it. Technical mechanics (architecture, per-metric
collection, entity matrix, phasing detail) live in
[shark-bench-design.md](./shark-bench-design.md) — phased design, updated for
lifecycle v2 on 2026-08-11.

### Delivery history and current tranche

- **Phase 1 is complete.** E40-F01 through E40-F04 remain the delivered Go
  fixture corpus, single-run collector, baseline/noise-band report, and
  `shark run` liveness foundation.
- **Lifecycle v2 is active.** E40-F05 through E40-F10 extend the same benchmark
  program to realistic feature, bug, change-card, and tech-debt lifecycles on a
  controlled Python fixture.
- Creating E40-F05 reopened E40 through Shark's normal parent-maintenance
  behavior. This does not revise the completion history of E40-F01 through
  E40-F04.

---

## 1. Problem Statement and Business Justification

Shark ships a default workflow bundle and lets users configure per-step
`provider`, `model`, `effort`, `skills`, and `prompt`. Today there is **no way to
tell whether any of those choices is good, or whether a change to them made
things better or worse**. Workflow tuning is done by feel: a maintainer edits a
prompt or swaps a model on the dev step, runs a few entities by hand, forms an
impression, and ships it. Every such decision is unfalsifiable — there is no
baseline to compare against, no measure of run-to-run variance, and therefore no
way to distinguish a real improvement from noise.

Three concrete costs follow:

- **Regressions ship silently.** A prompt edit that raises the code-review
  rejection rate or doubles token burn per task looks identical to a neutral edit.
- **Cost decisions are guesses.** "Would sonnet on the dev step be as good as opus
  at a third the spend?" is currently unanswerable, so the conservative (expensive)
  default persists by default, not by evidence.
- **User tuning is unsupported.** Advanced users configuring their own bundles get
  no method and no reference numbers.

Agentic-coding benchmarks (SWE-bench, Aider polyglot, terminal-bench) solved the
analogous problem for LLM harnesses using execution-based test oracles, isolated
runs, per-step cost capture, repetition with variance reporting, and paired
comparison. Shark needs the same discipline applied to its own workflow configs.

**Justification.** Shark's core promise is AI-driven development workflows; the
quality of the default bundle *is* the product. Bench data converts bundle
decisions from opinion into measured deltas and gives users evidence for their own
tuning. A secondary return is continuous end-to-end exercise of `shark run`: bench
design alone surfaced **B051** (`shark run TD-###` stalled silently at
`in_progress`), a high-severity runner defect no manual workflow had caught. B051
is now fixed — the pattern generalizes.

---

## 2. Goals and Success Criteria

G1-G7 preserve the Phase 1 contract. G8-G19 define the lifecycle v2 exit
contract. A later feature workflow may refine implementation detail, but it may
not orphan or silently weaken these epic-level outcomes.

| # | Goal | Phase | Success criterion (testable) |
|---|---|---|---|
| G1 | A screened corpus exists | 1 | ≥10 corpus items (tasks and bugs) admitted; **100% pass the admission gate** — at the base commit, FAIL_TO_PASS tests fail and PASS_TO_PASS tests pass; applying the reference patch turns both green |
| G2 | Runs execute unattended | 1 | A full baseline batch (≥10 items × 3 reps = ≥30 runs) completes without human intervention; every run terminates in a recorded outcome (including `timeout`) rather than hanging the batch |
| G3 | Runs are observable while in flight | 1 | For every run, per-stage progress is visible during execution (stderr events in `--json` mode) and a per-run log exists at `.shark/runs/<run_id>/run.log` |
| G4 | Every run emits complete metrics | 1 | 100% of completed runs produce a JSONL artifact carrying: per-stage wall-clock, tokens in/out, cost, exact model IDs, rejection count per gate, oracle result (F2P/P2P), quality-gate results (fmt/vet/lint-delta/test), and LOC added/deleted split prod vs test |
| G5 | Run-to-run variance is quantified | 1 | A published **noise band** per headline metric (pass rate, rejections per gate, tokens per step, wall-clock, LOC), computed from the 3-rep baseline |
| G6 | Config changes are evaluable | v2 | A deliberate config change (e.g. model swap on the dev step) produces a **paired per-task** delta report against the baseline; deltas that do not clear the G5 noise band are reported as **"no detectable effect"** rather than as an improvement |
| G7 | Results are reproducible | 1 | Re-running a stored manifest (pinned fixture-repo SHA, pinned model IDs, variant bundle) reproduces the reported metrics within the published noise band |
| G8 | Real lifecycle scenarios are admitted | v2 | At least one versioned feature, bug, change-card, and tech-debt scenario passes fixture, stage-matrix, adapter, and oracle admission against a controlled Python task-manager fixture |
| G9 | Stage evidence is complete and isolated | v2 | Every applicable stage emits an immutable, replayable snapshot; evaluator-only references, patches, answer keys, and tests are absent at every worker dispatch boundary |
| G10 | Product-design input is replayable | v2 | Feature scenarios execute D01-D05 through the existing Shark Rider product-design action using only versioned stakeholder and research responses; missing authorized input stops as `unresolved_gate` |
| G11 | The real keyed lifecycle executes | v2 | The lifecycle runner preserves every keyed dispatch response, claim, heartbeat, unchanged prompt handoff, semantic outcome, transition, release, Question decision, and named stop outcome for the scenario root and all eligible descendants |
| G12 | Safety ceilings preserve truth | v2 | Positive provider-cost, wall-time, and generated-task ceilings are required; reaching one stops and invalidates the entire scenario as `resource_limit` while retaining partial evidence |
| G13 | Artifact and implementation quality are independently evaluated | v2 | Applicable artifacts pass deterministic structural checks and a calibrated versioned judge; implementation correctness comes from a held-back execution oracle, never terminal workflow status or worker self-report |
| G14 | Comparisons have complete identity | v2 | Scenario, fixture, adapter, Shark binary, installed content, prompt, provider/model/effort, judge, reference, and resource-policy identity are present and uniform; mixed or incomplete runs are rejected from aggregates with reasons retained |
| G15 | Provider-backed baselines are deliberate and inspectable | v2 | Dry-run and report operations make no provider calls; provider-backed runs require explicit spend and safety limits; one retained, inspected pilot per scenario family precedes repeated baseline publication |
| G16 | Lifecycle cost separates work from coordination | v2 | Every applicable stage records a non-overlapping time ledger for provider-active work, tools and tests, queue or claim wait, replay or human-gate wait, retry or backoff, and unclassified time; reports group stages as discovery, specification, planning, code, review, QA, UAT, or shipping without double counting wall time |
| G17 | Review-gate value is measurable | v2 | Code review, QA, UAT, and controlled finish-feature deep-review comparisons retain structured findings with gate, round, severity, defect class, fingerprint, affected criterion, disposition, confirmation, first-seen gate, recurrence, resolution candidate, and duplicate linkage; reports show unique confirmed yield, overlap, false positives where truth labels permit, downstream escapes, and cost per gate |
| G18 | Artifact use and replayed human burden are visible | v2 | Every produced artifact has a typed producer record and downstream consumption or access edges; reports identify reused and orphaned artifacts. Replayed product-design stages record request and response counts, payload size, revision count, and unresolved gates without presenting those proxies as observed human minutes |
| G19 | The reviewed candidate and workflow policy are exact | v2 | Every code-producing or review stage pins the base commit, candidate tree and diff digests, changed-path digest, dirty and untracked manifest, test-suite digest, enabled gates, gate order, reviewer configuration and full review-bundle digest, and whether fixes were allowed; independent and sequential review comparisons reject mismatched candidates or policies |

**Phase 1 exit owns G1-G5 and G7.** G6 remains the paired configuration-change
criterion and is now owned by the v2 evaluation and operator-reporting tranche,
primarily E40-F09 and E40-F10. G8-G19 are jointly gated by UAT-08 through UAT-19.
See [architecture.md](architecture.md#delivery-boundaries-and-traceability) and
[uat-plan.md](uat-plan.md) for feature ownership and observable scenarios.

**Explicit non-goal on sensitivity.** A 10×3 matrix detects coarse effects
(roughly ≥20pp pass-rate shifts, large token/cost shifts). Detecting subtle prompt
tweaks is *not* a Phase 1 goal; the noise band exists precisely to stop such
claims being made from this data.

---

## 3. Scope

### Delivered scope (Phase 1)

- **Fixture corpus and oracles** — a purpose-built Go fixture repo with a real
  test suite, plus a corpus manifest of ~10 screened tasks and bugs (issue-style
  prompt, entity seed spec, held-back FAIL_TO_PASS tests, PASS_TO_PASS set), with
  base-SHA test and lint ledgers captured at corpus build. (E40-F01)
- **Bench harness** — scratch-project provisioning, variant-bundle install, entity
  seeding via the shark CLI, `shark run --json --workdir <fixture-checkout>`
  invocation under a timeout cap against a harness-owned fixture checkout
  (ADR-001), and collection of RunResult, stage transcripts, scratch-DB session
  history, and post-run checks in that checkout into one JSONL artifact per
  run. (E40-F02)
- **Baseline report and noise band** — aggregation of the Phase 1 matrix into the
  published baseline and per-metric noise band. (E40-F03)
- **`shark run` liveness** — stderr progress events in `--json` mode, stage-scoped
  heartbeats, correct cascade child labeling, and an unconditional per-run log
  file. (E40-F04)
- Entity types benched in Phase 1: **tasks and bugs**.

### Active scope (lifecycle v2)

- **E40-F05 — Lifecycle scenario corpus and adapter contract.** Add versioned
  feature, bug, change-card, and tech-debt scenarios, a controlled Python
  fixture, and a language-neutral adapter boundary. Produces I-04.
- **E40-F06 — Stage evidence and evaluator isolation.** Define the three-root
  isolation model, immutable stage evidence, decomposed time ledger, exact
  candidate snapshot, and artifact producer/consumer records. Produces I-05.
- **E40-F07 — Replayable product-design prelude.** Run feature scenarios through
  D01-D05 using versioned stakeholder and research responses, including explicit
  replayed human-interaction proxies. Produces I-06.
- **E40-F08 — Canonical multi-entity lifecycle runner.** Drive the real keyed
  Shark lifecycle for the root and every eligible descendant while capturing
  stage spans, workflow policy, artifact use, and structured review findings.
  Produces I-07.
- **E40-F09 — Calibrated evaluation and comparison identity.** Combine
  structural checks, a calibrated artifact judge, held-back execution truth,
  review-finding confirmation, paired gate evaluation, and fail-closed identity
  validation. Produces I-08.
- **E40-F10 — Operator workflow and retained lifecycle baseline.** Add safe
  preview, pilot, run, inspection, reporting, noise-band, and publication
  operations, including independent and sequential QA-versus-finish-feature
  deep-review comparisons, that consume I-07 and I-08.

### Out of scope

Out of scope for E40 entirely:

- Mining our own OSS issues for corpus material.
- New shark database tables or schema changes for bench data (JSONL artifacts are
  the store).
- A hosted dashboard, CI integration, or scheduled bench runs.
- Benchmarking non-shark agent harnesses, or cross-harness comparison.
- Any claim of statistical significance beyond the published noise band.

Still deferred after lifecycle v2:

- Epics as primary scenario roots and the full D06-D14 product-design arc.
- Sprint planning, execution, close, and retrospective scenarios.
- The delivery tail after the controlled finish-feature review comparison: PR
  feedback, CI wait and retry, merge, and branch or worktree cleanup.
- A SWE-bench Verified slice, corpus rotation policy, and post-hoc adversarial
  defect window.
- A hosted dashboard, scheduled service, or CI-triggered provider spend.
- Generic Shark, Rider, Question, usage-decoder, or Shark-data changes that the
  benchmark may expose. Triage those under their owning epics and link them to
  E40; do not absorb them into the benchmark wrapper.

---

## 4. Constraints and Assumptions

**Constraints**

- **Use the phase-appropriate public execution contract.** Phase 1 uses
  `shark run` unchanged. Lifecycle v2 uses a host-side controller over the
  canonical keyed dispatch, claim, heartbeat, semantic-outcome, transition, and
  release APIs so it can record each stage and route replayed decisions. It may
  not reconstruct prompts, routing, workflow state, claims, or Questions outside
  Shark.
- **One Go-side change in Phase 1.** F01-F03 required no changes to Shark itself;
  **F04 does** — `--json` runs are silent until completion today, which unattended
  batches cannot tolerate. This is a historical Phase 1 constraint, not a claim
  that lifecycle v2 can never expose a core defect.
- **Never run against the live repo or database.** All provisioning goes through
  `scripts/shark-scratch-env.sh`; the shark-config guardrail hook blocks the
  alternative.
- **Exact model IDs are pinned per run** and recorded in every manifest.
  Comparisons across unpinned model versions are invalid and must not be reported.
- **JSONL artifacts are the source of truth.** Reports are derived; no bench state
  lives in a shark database.
- **Comparison is paired per-task only.** Cross-task aggregation of a variant
  against a baseline is not a permitted method.
- **Evaluator-only material never enters an agent-visible root**; references,
  answer keys, patches, and hidden tests become readable only by their authorized
  post-stage or post-run evaluator.
- `E40-interaction-map.md` is the stable cross-feature contract registry for all
  ten features. I-01 through I-03 preserve Phase 1; I-04 through I-08 define
  lifecycle v2. Every row names a producer, consumer, and architecture shape.

**Assumptions**

- Phase 1 artifacts preserve the provider JSON envelope they historically
  parsed harness-side. Lifecycle v2 verifies the audited field mapping through
  F06/X-09 and fails closed when required usage or model identity is absent.
- A curated fixture repo produces oracle signal representative enough for
  *relative* config comparison, even though absolute pass rates will not transfer
  to real-world repos.
- Provider spend is never implicit. Every provider-backed pilot or baseline run
  requires explicit acknowledgement plus positive cost, wall-time, and
  generated-task ceilings.
- Corpus items sit in a discriminative band — items every config aces or every
  config fails are dropped rather than kept as filler.

---

## 5. Stakeholder Impact

| Stakeholder | Impact |
|---|---|
| **Shark maintainers** (primary) | Default-bundle decisions — model, effort, prompt wording, per-step assignment — become evidence-backed. Workflow and prompt edits gain a regression guard before becoming the default. |
| **Advanced shark users** | Get a documented method and reference numbers for evaluating their own workflow variants, plus a stated sensitivity floor so they don't over-read small deltas. |
| **Runner / core maintainers** | Bench exercises `shark run` end-to-end and unattended, surfacing runner defects that manual use misses (B051 precedent). F04 also improves day-to-day `shark run` observability for every user, not just bench. |
| **Cost owner** | Accepts recurring API spend for baseline and matrix runs, in exchange for the ability to justify or reduce per-step model spend with measurements. |
| **Feature/task authors (corpus curators)** | Take on ongoing work: writing reference patches and held-back tests, and keeping the corpus in its discriminative band as models improve. |
| **Product and research participants** | Supply versioned replay inputs for feature scenarios once; scored runs consume those frozen responses rather than interrupting people or reaching mutable external sources. |
| **Benchmark operators and reviewers** | Gain retained stage evidence, invalid-run reasons, calibrated artifact judgments, and execution-oracle truth, but must inspect one real pilot per scenario family before publishing a repeated baseline. |

---

## 6. High-Level Acceptance Criteria (UAT Scenarios)

**Phase 1 exit was gated on UAT-1, UAT-2, UAT-5, UAT-6, and UAT-7.** Lifecycle
v2 retains those regression contracts, assigns UAT-3 and UAT-4 to E40-F09 and
E40-F10, and adds UAT-8 through UAT-19. Full detail and coverage pointers live
in [uat-plan.md](uat-plan.md).

**UAT-1 — Unattended baseline batch produces a report with a noise band**
*Given* an admitted corpus of ≥10 items and the default workflow bundle,
*when* an operator starts the Phase 1 baseline batch (10 items × 3 reps) and walks
away, *then* the batch completes without intervention, every run yields a JSONL
artifact, and the aggregated report states, per headline metric, the baseline value
and the observed run-to-run spread (the noise band).

**UAT-2 — A broken corpus item is rejected at admission, not benched**
*Given* a candidate corpus item whose reference patch does **not** turn its
FAIL_TO_PASS tests green (or whose PASS_TO_PASS set is already red at the base
commit), *when* the admission gate runs, *then* the item is rejected with the
failing check named, and it does not appear in any benchmark run.

**UAT-3 — A config delta inside the noise band is reported as no effect**
*Given* a published baseline and its noise band, *when* an operator runs a variant
bundle over the same corpus and the paired per-task delta on every headline metric
falls inside the band, *then* the comparison report states **"no detectable
effect"** and does not present the variant as an improvement or a regression.
E40-F09 validates comparison identity and E40-F10 owns the operator report.

**UAT-4 — A config delta outside the noise band is reported as a measured change**
*Given* the same baseline, *when* an operator swaps the model on the dev step and
the paired per-task delta on at least one headline metric exceeds the band,
*then* the report names the metric, the direction and size of the delta, the
per-task pairs behind it, and the exact model IDs on both sides — and re-running
the stored manifest reproduces the finding within the band. E40-F09 validates
the paired evidence and E40-F10 owns publication.

**UAT-5 — A stuck run is bounded, recorded, and does not hang the batch**
*Given* a run whose stage exceeds the configured timeout cap, *when* the cap is
reached, *then* the run terminates, its artifact records `outcome=timeout` with the
stage it stalled in, and the batch proceeds to the next run.

**UAT-6 — An in-flight run is observable**
*Given* a run started with `--json`, *when* an operator watches stderr or tails
`.shark/runs/<run_id>/run.log`, *then* they see per-stage progress with the entity
key (correct for cascade children), stage status, agent/provider, and stage plus
total elapsed time — while stdout still delivers a single clean RunResult at the
end.

**UAT-7 — Results reproduce from a stored manifest** (Phase 1, owned by E40-F03)
*Given* a stored manifest (pinned fixture-repo SHA, pinned exact model IDs,
variant bundle id) from a prior run, *when* an operator invokes the replay
command against it, *then* E40-F02's single-run command re-executes with those
exact pinned inputs, and the resulting metrics fall within the noise band
originally published for that manifest — verified from the manifest and artifact
directory alone, with no other state.

### Lifecycle v2 acceptance summary

- **UAT-08:** all four admitted scenario families load with correct applicable
  and non-applicable stages.
- **UAT-09:** evaluator-only material is absent at every dispatch boundary and
  becomes readable only by the authorized evaluator.
- **UAT-10:** feature scenarios reproduce D01-D05 from the versioned replay
  bundle; missing input stops as `unresolved_gate`.
- **UAT-11:** the keyed lifecycle records fork selection, claim, heartbeat,
  unchanged prompt handoff, semantic outcome, transition, and release for every
  eligible entity.
- **UAT-12:** Questions, lease loss, worker failure, missing outcomes, and safety
  ceilings stop with the correct named outcome and retain partial evidence.
- **UAT-13:** structural checks, calibrated judge evidence, and the held-back
  execution oracle remain distinct and all gate publication.
- **UAT-14:** aggregates reject missing or mixed comparison identity and retain
  the invalid inventory.
- **UAT-15:** dry-run and report paths spend nothing; retained pilots for all four
  families are inspected before a repeated lifecycle baseline is published.
- **UAT-16:** stage time partitions into provider, tool/test, wait, retry, and
  unclassified intervals whose total reconciles to lifecycle wall time.
- **UAT-17:** structured findings support an independent frozen-candidate and an
  actual sequential QA-versus-finish-feature deep-review comparison without
  treating duplicate or unconfirmed findings as unique value.
- **UAT-18:** artifact producer/consumer edges identify reused and orphaned
  outputs, and D01-D05 reports replayed interaction proxies without claiming
  measured human effort.
- **UAT-19:** changing the candidate tree, untracked manifest, test suite, gate
  set, gate order, reviewer configuration, or fix policy invalidates a paired
  comparison.

The full Given/When/Then scenarios and feature ownership live in
[uat-plan.md](uat-plan.md).
