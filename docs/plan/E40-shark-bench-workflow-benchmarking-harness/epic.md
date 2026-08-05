---
epic_key: E40
title: Shark Bench: workflow benchmarking harness
description: Benchmark harness around shark (not inside it) that measures the effectiveness of a workflow configuration end-to-end and detects the effect of config changes (model, effort, prompt, per-step assignments). Uses shark run as the execution engine; captures per-stage wall-clock, tokens in/out, cost, LOC, rejections, defects, and code quality into JSONL artifacts; compares config variants paired per-task against a measured noise band.
---

# Shark Bench: workflow benchmarking harness

**Epic Key**: E40

---

## Goal

### Problem

There is no way to measure how effective a shark workflow configuration is, or whether a change to it (model, effort level, prompt, per-step model assignments) made things better or worse. Workflow tuning is currently done by feel. Agentic-coding benchmarks (SWE-bench, Aider polyglot, terminal-bench) solve the analogous problem for LLM harnesses with execution-based test oracles, isolated runs, and paired config comparison — shark needs the same discipline applied to its own workflow configs.

### Solution

Build shark bench as a harness *around* shark, with `shark run` as the execution engine (it already drives every entity type through its workflow, dispatches agents headlessly via `claude --output-format json`, supports `--worktree` isolation, and emits per-stage logs). The harness provisions a scratch shark project plus a pinned fixture repo per run, installs a workflow-config variant bundle, invokes `shark run --json --worktree` under a timeout, and collects metrics from the RunResult, stage transcripts, and post-run checks in the worktree into one JSONL artifact per run. Config variants are compared paired per-task against a noise band measured from repeated baseline runs.

Full design: [shark-bench-design.md](./shark-bench-design.md).

### Impact

- A trustworthy baseline for the default workflow: pass rate, rejection rate, tokens/cost per step, wall-clock, LOC — with run-to-run spread quantified.
- Config changes (model, effort, prompt, per-step assignment) evaluated as measured deltas against that noise band instead of anecdote.
- Regression guard: workflow/prompt edits that raise rejection rates or token burn are caught before becoming the default.

---

## Business Value

**Rating**: High

Shark's core promise is AI-driven development workflows; without measurement, every workflow/prompt/model decision is a guess. Bench data directly informs default-bundle choices and gives users evidence for their own tuning. It also exercises `shark run` end-to-end continuously, surfacing runner defects (B051 was found during bench design).

---

## Quick Reference

**Primary Users**: shark maintainers tuning default workflow bundles; advanced users evaluating their own config variants.

**Key Features**:
- Curated fixture corpus (tasks + bugs first) with execution-based oracles (FAIL_TO_PASS + PASS_TO_PASS)
- Bench harness: scratch-env provisioning, variant bundle install, `shark run` invocation, metric collection to JSONL
- Baseline + noise-band report; paired per-task A/B comparison of config variants
- Core instrumentation (Phase 2): token usage + per-stage outcome in StageLog, stage timeout

**Success Criteria**:
- Phase 1 baseline: ~10 screened tasks × 3 reps completing unattended, with published noise band
- A deliberate config change (e.g., model swap on the dev step) produces a measurable, reproducible delta report

**Phases**: P1 baseline (zero Go changes) → P2 config matrix + core instrumentation → P3 epics, SWE-bench Verified slice, rubric reviewer.

---

*Last Updated*: 2026-08-05
