---
feature_key: E34-F09-override-drift-visibility-and-wwgm-reconciliation
epic_key: E34
title: Override Drift Visibility and WWGM Reconciliation
description: Expose canonical-versus-override drift through Shark administration commands, promote reusable WWGM review improvements upstream, and track a deliberate WWGM rebase and cleanup after upstream delivery.
---

# Override Drift Visibility and WWGM Reconciliation

**Feature Key**: E34-F09

## Goal

### Problem

Shark overrides fully replace canonical bundle files and survive upgrades, but
`shark admin upgrade` only reports that it skipped them. An operator cannot
tell whether an override still matches its canonical baseline, hides upstream
changes, duplicates the default, or has lost its counterpart. The 2026-08-24
reconciliation found ten WWGM overrides: seven prompt/skill files contain
reusable or stale policy, while three workflow files carry or may carry
project-specific routing/model decisions.

### Solution

Add read-only override drift classification with stable JSON output and
explicit baseline provenance. Promote reusable WWGM quality semantics into the
canonical E34 features, then reconcile WWGM in one linked follow-up: remove
obsolete prompt/skill overrides, rebuild retained workflow policy on the new
canonical files, add missing project-local deterministic safeguards, and
resolve the existing CC-007 and CC-008 records without creating duplicates.

### Impact

Operators can see override risk before or after an upgrade without diffing by
hand. Upstream improvements reach customized projects deliberately, WWGM keeps
only true local policy, and later E40 benchmarks can compare canonical and
customized configurations from a known baseline.

## Research findings

- `shark admin upgrade` refreshes every canonical file and skips the entire
  `overrides/` subtree. Its summary lists skipped paths but does not compare
  their content or remember the canonical version from which they diverged.
- Resolution is replace-only: an override at a relative path fully masks the
  canonical file. Therefore a small local amendment can hide unrelated future
  improvements in that file.
- WWGM's approval, UAT skill, and UAT rubric overrides contain reusable tier,
  staged-edge, and disposition behavior. Its epic review adds a reusable
  deferred-consumer rule. Those changes should move upstream through
  E34-F06–F08.
- WWGM's code-review, task-review, development, and epic-review overrides are
  behind the current canonical staged-integration content. The epic and
  feature workflow overrides carry intentional epic order and `gpt-5.6-sol`
  model assignments. The newer sprint workflow override also removes the
  canonical research step; that routing change needs explicit owner
  re-ratification rather than automatic upstream promotion. Every whole-file
  workflow replacement can hide later canonical steps.
- WWGM lacks a root `AGENTS.md` and the deterministic project checks proposed
  for method length, test selection, test database setup, unexpected skips,
  standards, and bare-assert linting. These are WWGM adoption work, not generic
  Shark bundle commands.
- WWGM currently ignores the entire generated `shark-data/` tree, including
  `overrides/`. Any retained workflow policy therefore needs an explicitly
  versioned home or unignore rule; local preservation across one upgrade is
  not sufficient recovery evidence for a fresh clone.

## Override status contract

`shark admin overrides status [--json]` classifies every regular override by
relative path. JSON rows use this stable vocabulary:

| Classification | Meaning |
|---|---|
| `current` | Override differs intentionally and the recorded canonical baseline still equals the current embedded canonical digest. |
| `upstream_changed` | The current embedded canonical digest differs from the recorded baseline for this override. |
| `identical_redundant` | Override bytes equal the current canonical counterpart and the override can be removed without changing resolution. |
| `orphaned` | No canonical counterpart exists in the current embedded bundle. |
| `baseline_unknown` | A counterpart exists, but Shark has no trustworthy canonical baseline for the override. |

Each row includes relative path, classification, override SHA-256, current
canonical SHA-256 when present, recorded baseline SHA-256 when known, and a
bounded suggested action. Summary counts use the same classification keys.

## Requirements

1. **REQ-F-001 — Read-only drift status**
   - Add the `shark admin overrides status` command and `--json` support.
   - Walk only regular files under the resolved data root's `overrides/`
     subtree; reject symlinks and escaping paths consistently with bundle
     validation.
   - Compare bytes by SHA-256 without printing file contents.
   - Sort rows by relative path for deterministic human and JSON output.

2. **REQ-F-002 — Baseline provenance**
   - Store per-path canonical baseline digests in a Shark-owned manifest that
     never contains override bytes. Its location is
     `<resolved shark_data_path>/.shark-override-baselines.json`; it is not
     hardcoded to a repository-relative `shark-data/` directory.
   - Leave a discovered override without trustworthy provenance as
     `baseline_unknown`; status, upgrade, and dry-run never create or advance a
     baseline.
   - Add an explicit `shark admin overrides acknowledge <path>...` operation
     that records the current canonical digest only after the operator has
     manually reconciled that override.
   - An unknown or invalid baseline must classify as `baseline_unknown`, not
     `current`.

3. **REQ-F-003 — Upgrade visibility**
   - Extend human and JSON upgrade summaries with override classification
     counts and the status command needed for detail.
   - Preserve the existing added/updated/unchanged/skipped fields for backward
     compatibility.
   - `--dry-run` computes the same classifications without writing canonical
     files or baseline metadata.

4. **REQ-F-004 — Safe operator control**
   - Status and upgrade never rewrite, merge, delete, or auto-disable an
     override.
   - Acknowledge changes metadata only; it requires an existing override and
     current canonical counterpart.
   - Errors and suggestions name relative paths and digests, never file
     contents, credentials, rendered prompts, or transcripts.

5. **REQ-F-005 — Promote reusable WWGM behavior**
   - E34-F06 owns already-dispositioned recurrence and severity conflict.
   - E34-F07 owns cross-epic deferred-consumer closure and decision impact.
   - E34-F08 owns SIMPLE-lite/tier artifact rules, staged-edge final closure,
     and executable evidence.
   - E34-F09 must verify that these canonical changes cover the reusable WWGM
     hunks before authorizing local override removal.

6. **REQ-F-006 — WWGM reconciliation work item**
   - After I-05 is available, create or reuse exactly one linked WWGM change
     item that accounts for every current override and the local safeguards
     below. Link or resolve CC-007 and CC-008 instead of duplicating them.
   - Remove prompt and skill overrides whose behavior is fully canonical;
     rebase any retained amendment on the new full canonical file.
   - Retain WWGM's intentional epic workflow ordering and `gpt-5.6-sol` model
     assignments, but rebuild those workflow overrides so they include new
     canonical steps and fields.
   - Re-ratify the sprint workflow's planning-to-active routing. Remove the
     override if WWGM should use canonical sprint research; otherwise rebuild
     it from the current canonical workflow and record the local decision.
   - Make every retained WWGM policy reproducible from version control rather
     than relying on the ignored generated `shark-data/` tree.
   - Add WWGM-local exact method-length and test-selection checks, test database
     setup, unexpected-skip enforcement, architecture standards, bare-assert
     lint guard, and a thin root `AGENTS.md` that points to canonical project
     guidance.
   - Reconcile the historical E04-F02 shipped-while-UAT-rejected record through
     an explicit WWGM decision or current-gate correction. Do not convert this
     cleanup into a global owner-approval setting.

7. **REQ-F-007 — Evidence threshold for routing infrastructure**
   - Do not upstream WWGM's proposed `rules.py` selector or editor hook in this
     feature. Record them in the WWGM work item as deferred candidates until
     more than one project or measured context growth demonstrates the need.
   - Keep the root `AGENTS.md` project-local and thin.

8. **REQ-F-008 — E40 comparison follow-up**
   - After E40 is ready, benchmark a canonical configuration and a reconciled
     WWGM-style configuration from recorded bundle and baseline digests.
   - Do not block F09 delivery on E40 execution.

9. **REQ-NF-001 — Compatibility and determinism**
   - Preserve existing upgrade behavior for projects with no overrides.
   - Keep JSON field names, enum values, row order, and digest encoding stable
     and covered by tests.

## Implementation plan

1. Add the baseline manifest model, safe path walker, digest classifier, and
   unit tests in `internal/sharkdata`.
2. Add `admin overrides status` and `acknowledge`, plus JSON and human output.
3. Integrate classification counts into upgrade and dry-run paths without
   changing override bytes or baseline metadata.
4. Consume I-05 and produce a path-by-path WWGM reconciliation checklist.
5. Create/reuse the single WWGM adoption item, link CC-007/CC-008, execute the
   cleanup and local safeguards there, and validate both projects.
6. Record non-blocking E40 comparison scenarios with canonical and baseline
   digests.

## WWGM override disposition

| Current override | Disposition after upstream work |
|---|---|
| `prompts/feature/approval.md` | Promote tier and disposition behavior through F06/F08, then remove if no local delta remains. |
| `skills/uat/references/redteam-rubric.md` | Promote shared staged-edge/disposition policy, deduplicate canonical text, then remove. |
| `prompts/epic/feature_review.md` | Promote the general deferred-consumer rule through F07; rebase or remove after canonical staged checks are restored. |
| `prompts/feature/code_review.md` | Promote SIMPLE-lite/tier behavior through F08, then remove the stale full replacement. |
| `prompts/feature/task_review.md` | Promote SIMPLE-lite behavior and F07 naming checks, then remove the stale full replacement. |
| `prompts/task/development.md` | Promote generic tier and defect-sweep behavior through F06/F08; keep WWGM commands in project guidance, then remove. |
| `skills/uat/SKILL.md` | Promote the generic undecomposed-producer feature-gate accommodation through F07/F08 without weakening the live-wiring/security floor, then remove. |
| `workflow/epic.yaml` | Retain WWGM order/model policy, rebuilt from the post-F08 canonical workflow. |
| `workflow/feature.yaml` | Retain WWGM model policy, rebuilt from the post-F08 canonical workflow. |
| `workflow/sprint.yaml` | Do not promote as-is. Re-ratify the local removal of sprint research, then either remove the override or rebuild it from post-F10 canonical routing. |

## Acceptance scenarios

**Classify an upstream-changed override**

- Given an override has a recorded canonical baseline and the binary ships a
  changed canonical counterpart,
- When the operator runs status or upgrade dry-run,
- Then the path is `upstream_changed` with all three available digests,
- And no override or baseline file is changed.

**Classify legacy and redundant overrides**

- Given one legacy override has no baseline, one equals canonical, and one has
  no canonical counterpart,
- When status runs,
- Then they classify as `baseline_unknown`, `identical_redundant`, and
  `orphaned` respectively in deterministic order.

**Reconcile WWGM without losing policy**

- Given F06–F08 have shipped and I-05 identifies the canonical changes,
- When the linked WWGM item is completed,
- Then reusable prompt/skill overrides are removed, retained workflow policy
  includes all current canonical steps, local deterministic safeguards pass,
  and CC-007/CC-008 are linked or resolved rather than duplicated.

## Dependencies and interactions

- Depends on E34-F08 in Shark.
- Consumes **I-05 CanonicalAdoptionManifest v1**.
- Reads WWGM's current override tree and existing CC-007/CC-008 records without
  changing WWGM during this planning feature.

## Out of scope

- Automatic three-way merge, deletion, or rewriting of overrides.
- Storing override contents in the baseline manifest or telemetry.
- Upstreaming WWGM application rules, Python scripts, database setup, lint
  configuration, model selections, or workflow order as universal defaults.
- A universal owner-approval config change.
- Blocking delivery on E40.

## Verification plan

- Unit-test all five classifications, path safety, symlinks, missing/corrupt
  manifests, deterministic ordering, `baseline_unknown` preservation, and
  explicit acknowledge.
- CLI-test human/JSON output and backward-compatible upgrade fields.
- Assert dry-run changes no file or manifest; run a mutating upgrade with
  sentinel override bytes and mode metadata and assert the override inventory,
  byte digests, and baseline manifest remain identical; status never prints
  content.
- Validate the full WWGM reconciliation checklist against I-05 and the live
  override inventory before the linked WWGM item closes.
- Run `make fmt`, `make lint`, `make test`, and `git diff --check`.

*Last Updated*: 2026-08-05
