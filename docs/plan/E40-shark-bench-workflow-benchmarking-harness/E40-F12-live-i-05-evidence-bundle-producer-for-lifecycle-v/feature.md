---
feature_key: E40-F12-live-i-05-evidence-bundle-producer-for-lifecycle-v
epic_key: E40
title: Live I-05 evidence bundle producer for lifecycle-v2 runs
description: No production code path emits a live I-05 evidence bundle (bundle.json/stages/*.json/access.jsonl) during a real E40 lifecycle-v2 run; run-lifecycle.sh only folds bounded snippets into the I-07 JSONL. Closes an already-flagged, still-open gap (F10 review finding TC-082, TD-132) and unblocks the first real six-family baseline.
related-docs:
  - bench/evidence/i05-schema.yaml
  - bench/README.md
---

# Live I-05 evidence bundle producer for lifecycle-v2 runs

**Feature Key**: E40-F12-live-i-05-evidence-bundle-producer-for-lifecycle-v

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md) _(if available)_

---

## Goal

### Problem
F06 fully specified the I-05 stage-evidence contract (schema, three-root isolation model, verifiers) and F08/F09/F10 built the offline driver, evaluator, and retention/comparison tooling against it — but nothing was ever wired to actually *produce* a real bundle during a live run. `run-lifecycle.sh` (the F08 driver `pilot`/`baseline` invoke per rep) only folds bounded `evidence_refs` snippets inline into the I-07 JSONL; it never writes `bundle.json`/`stages/*.json`/`access.jsonl` to `i05_bundle_dir`. Every script that touches `bundle.json` elsewhere only reads/verifies one, or is a unit test using a hand-built fixture.

Consequence, verified empirically via a real `e40-benchmark.sh preflight` run against all 6 lifecycle-v2 families (2026-09-08): every scenario is blocked on `scenario_roots.<id>.i05_bundle_dir is not configured` (P5), and `run-lifecycle-batch.sh:902-905` treats a missing bundle as fatal per pair (`"cannot evaluate, recorded failed"`). This is not a config gap — it is the single hard blocker standing between E40 and its first real lifecycle-v2 baseline.

This also closes two already-open, related findings that never got fixed: E40-F10's round-1 code-review finding TC-082 (`retain_pair()` never copies real I-05 content into the retention root — an empty dir currently satisfies every downstream check), and TD-132 (`run-lifecycle.sh` never sets `content_root`, leaving `evaluate-lifecycle.sh`'s content-digest crosscheck dead in every real run).

### Solution
Build the missing producer: materialize a real I-05 bundle (`bundle.json` + per-stage snapshots) at `i05_bundle_dir` as each stage dispatches inside `run-lifecycle.sh`'s existing per-dispatch loop, using the already-fully-specified `bench/evidence/i05-schema.yaml` contract and the candidate identity / evidence data the driver already computes internally. Wire the resulting bundle through to close TC-082 (real retention copy) and TD-132 (`content_root` populated so the crosscheck fires).

### Impact
Unblocks a real, spend-eligible `pilot`/`baseline` run across all 6 lifecycle-v2 families — currently 0 of 6 can produce a non-`failed` evaluated pair.

---

## User Stories

### Must-Have Stories

**Story 1**: As a [user persona], I want to [perform an action] so that I can [achieve a benefit].

**Acceptance Criteria**:
- [ ] [Specific testable criterion 1]
- [ ] [Specific testable criterion 2]
- [ ] [Specific testable criterion 3]

---

## Requirements

### Functional Requirements

1. **REQ-F-001**: [Requirement Title]
   - **Description**: [Clear, specific, testable requirement statement]
   - **Priority**: [Must-Have | Should-Have | Could-Have]
   - **Acceptance Criteria**:
     - [ ] [Specific criterion 1]
     - [ ] [Specific criterion 2]

### Non-Functional Requirements

1. **REQ-NF-001**: [Non-functional requirement]
   - **Description**: [Specific target or constraint]
   - **Measurement**: [How it will be measured]

---

## Acceptance Criteria

**Scenario 1: [Primary Use Case]**
- **Given** [initial context/state]
- **When** [user action is performed]
- **Then** [expected outcome]
- **And** [additional outcome]

---

## Out of Scope

1. **[Feature/Capability]**
   - **Why**: [Reasoning]
   - **Future**: [Whether this may be addressed later]

---

## Success Metrics

1. **[Metric Name]**
   - **What**: [What data point is tracked]
   - **Target**: [Specific goal]
   - **Measurement**: [How to measure]

---

*Last Updated*: 2026-09-08
