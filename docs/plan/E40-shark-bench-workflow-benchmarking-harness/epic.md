---
epic_key: E40
title: Shark Bench: workflow benchmarking harness
description: Benchmark harness around shark (not inside it) that measures the effectiveness of a workflow configuration end-to-end and detects the effect of config changes (model, effort, prompt, per-step assignments). Uses shark run as the execution engine; captures per-stage wall-clock, tokens in/out, cost, LOC, rejections, defects, and code quality into JSONL artifacts; compares config variants paired per-task against a measured noise band.
---

# Shark Bench: workflow benchmarking harness

**Epic Key**: E40 · **Last Updated**: 2026-08-05

This document is the single source of business context for E40. Features reference
it; they must not restate it. Technical mechanics (architecture, per-metric
collection, entity matrix, phasing detail) live in
[shark-bench-design.md](./shark-bench-design.md) — agreed approach, 2026-08-05.

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

Each criterion below is measurable and verifiable at Phase 1 exit, **except G6**
(see the phase note below the table).

| # | Goal | Phase | Success criterion (testable) |
|---|---|---|---|
| G1 | A screened corpus exists | 1 | ≥10 corpus items (tasks and bugs) admitted; **100% pass the admission gate** — at the base commit, FAIL_TO_PASS tests fail and PASS_TO_PASS tests pass; applying the reference patch turns both green |
| G2 | Runs execute unattended | 1 | A full baseline batch (≥10 items × 3 reps = ≥30 runs) completes without human intervention; every run terminates in a recorded outcome (including `timeout`) rather than hanging the batch |
| G3 | Runs are observable while in flight | 1 | For every run, per-stage progress is visible during execution (stderr events in `--json` mode) and a per-run log exists at `.shark/runs/<run_id>/run.log` |
| G4 | Every run emits complete metrics | 1 | 100% of completed runs produce a JSONL artifact carrying: per-stage wall-clock, tokens in/out, cost, exact model IDs, rejection count per gate, oracle result (F2P/P2P), quality-gate results (fmt/vet/lint-delta/test), and LOC added/deleted split prod vs test |
| G5 | Run-to-run variance is quantified | 1 | A published **noise band** per headline metric (pass rate, rejections per gate, tokens per step, wall-clock, LOC), computed from the 3-rep baseline |
| G6 | Config changes are evaluable | **2** | A deliberate config change (e.g. model swap on the dev step) produces a **paired per-task** delta report against the baseline; deltas that do not clear the G5 noise band are reported as **"no detectable effect"** rather than as an improvement |
| G7 | Results are reproducible | 1 | Re-running a stored manifest (pinned fixture-repo SHA, pinned model IDs, variant bundle) reproduces the reported metrics within the published noise band |

**Phase 1 exit owns G1–G5 and G7.** G6 requires variant bundles and the paired
per-task comparison report, both of which are Phase 2 scope (§3) — no Phase 1
feature owns G6, and F03 explicitly excludes comparison. Stating G6 as a Phase 1
exit criterion would leave it orphaned; this phase placement is decision Q002
(see [architecture.md](architecture.md#delivery-boundaries-and-traceability) and
[uat-plan.md](uat-plan.md)). G7, by contrast, needs no comparison — only a
replay of a stored manifest, which is Phase 1 scope owned by E40-F03.

**Explicit non-goal on sensitivity.** A 10×3 matrix detects coarse effects
(roughly ≥20pp pass-rate shifts, large token/cost shifts). Detecting subtle prompt
tweaks is *not* a Phase 1 goal; the noise band exists precisely to stop such
claims being made from this data.

---

## 3. Scope

### In scope (Phase 1)

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

### Out of scope

Out of scope for E40 entirely:

- Mining our own OSS issues for corpus material.
- New shark database tables or schema changes for bench data (JSONL artifacts are
  the store).
- A hosted dashboard, CI integration, or scheduled bench runs.
- Benchmarking non-shark agent harnesses, or cross-harness comparison.
- Any claim of statistical significance beyond the published noise band.

Deferred to later phases of this epic (planned, not in Phase 1):

- **Phase 2**: core instrumentation G1 (decode the agent usage envelope into
  StageLog), G2 (per-stage outcome + `outcome_source`), G3 (`--stage-timeout`);
  variant bundles and the matrix runner with per-run budget caps; the paired
  comparison report (**G6**, with UAT-3 and UAT-4); feature-level benching (Mode A
  pre-seeded children, then Mode B agent-generated); change-card and tech-debt
  entity coverage.
- **Phase 3**: epics as benched entities; a SWE-bench Verified slice; a
  rubric/LLM-judge reviewer; codex usage parity (G4 in the design doc); post-hoc
  adversarial defect window; corpus rotation policy.

---

## 4. Constraints and Assumptions

**Constraints**

- **`shark run` is the execution engine.** Bench is a harness *around* shark, not
  a feature inside it. It may not fork, reimplement, or bypass the run controller.
- **One Go-side change in Phase 1.** F01–F03 require no changes to shark itself;
  **F04 does** — `--json` runs are silent until completion today, which unattended
  batches cannot tolerate. Any further core change is Phase 2 by definition.
- **Never run against the live repo or database.** All provisioning goes through
  `scripts/shark-scratch-env.sh`; the shark-config guardrail hook blocks the
  alternative.
- **Exact model IDs are pinned per run** and recorded in every manifest.
  Comparisons across unpinned model versions are invalid and must not be reported.
- **JSONL artifacts are the source of truth.** Reports are derived; no bench state
  lives in a shark database.
- **Comparison is paired per-task only.** Cross-task aggregation of a variant
  against a baseline is not a permitted method.
- **Held-back tests never enter the repo the agent sees**; they are injected only
  at terminal status during post-run checks.
- The epic design phase must produce `E40-interaction-map.md` (four features with
  F01→F02→F03 producer/consumer handoffs). Stable `I-##` IDs are assigned there
  only, once each shape source resolves to a section in `architecture.md` — this
  PRD deliberately invents none.

**Assumptions**

- Agent dispatch persists the provider's JSON envelope (usage, cost, model IDs)
  per stage when transcripts are captured, so Phase 1 can parse it harness-side
  without core changes. If this proves false, G1 moves into Phase 1 scope.
- A curated fixture repo produces oracle signal representative enough for
  *relative* config comparison, even though absolute pass rates will not transfer
  to real-world repos.
- API spend for a full baseline batch (≈30 multi-stage runs) is acceptable and
  bounded by the per-run timeout cap; matrix runs in Phase 2 will additionally need
  per-run budget caps.
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

---

## 6. High-Level Acceptance Criteria (UAT Scenarios)

**Phase 1 exit is gated on UAT-1, UAT-2, UAT-5, UAT-6, and UAT-7.** UAT-3 and
UAT-4 are stated here for completeness but are **Phase 2 criteria** — they
require the variant bundles and paired comparison report deferred with G6 (§3,
§2 phase note; decision Q002). Full detail and coverage pointers for every
scenario, including the Phase 2 deferral rationale, live in
[uat-plan.md](uat-plan.md).

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

**UAT-3 — (Phase 2) A config delta inside the noise band is reported as no effect**
*Given* a published baseline and its noise band, *when* an operator runs a variant
bundle over the same corpus and the paired per-task delta on every headline metric
falls inside the band, *then* the comparison report states **"no detectable
effect"** and does not present the variant as an improvement or a regression.
Deferred to Phase 2 with G6 — no Phase 1 feature builds the variant-bundle
comparison this requires.

**UAT-4 — (Phase 2) A config delta outside the noise band is reported as a measured change**
*Given* the same baseline, *when* an operator swaps the model on the dev step and
the paired per-task delta on at least one headline metric exceeds the band,
*then* the report names the metric, the direction and size of the delta, the
per-task pairs behind it, and the exact model IDs on both sides — and re-running
the stored manifest reproduces the finding within the band. Deferred to Phase 2
with G6, for the same reason as UAT-3.

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
