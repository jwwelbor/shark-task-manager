---
feature_key: E35-F06-harness-and-docs
epic_key: E35
title: Harness and docs
description: shark/SKILL.md dispatch loop becomes claim -> run -> release. validate gains outcome-map resolution, core-outcomes-present, every-old-status-has-an-alias-home, and bundle-root resolution checks. Update vocabulary in CLAUDE.md, workflow-configuration.md, workflow-profiles.md. Final integration; depends on F2/F3/F4.
size: M
---

# Harness and docs

**Feature Key**: E35-F06

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Design (single source of truth)**: [route-based-workflow-redesign.md](../../route-based-workflow-redesign.md) — §3, §8, §9

---

## Goal

### Problem

The engine pieces (F01–F05) change the contract the harness and docs assume: the
dispatch loop still speaks the old `advance`/`set-status` vocabulary, `shark validate`
doesn't check the new structures, and `CLAUDE.md` / workflow guides still describe
`ready_for_X`/`in_X` and the two-map split. Without this feature the redesign is
usable from the CLI but not wired into the harness or documented.

### Solution

Final integration (design §3, §9):

- Rewrite the `shark/SKILL.md` dispatch loop to **`claim → run → release`**: claim an
  unclaimed entity via `shark next`, run the rendered prompt, then `release(outcome)`
  with the skill's `{ outcome, reason, log }` payload (skills stay topology-blind).
- Extend `shark validate` with the redesign's structural checks:
  - outcome maps resolve (every target names a defined step);
  - core outcomes (`pass`/`fail`/`blocked`) present on every non-terminal step;
  - every old status has an alias home;
  - prompts/skills/agents resolve from the bundle root.
- Update vocabulary across the docs: `CLAUDE.md`, `workflow-configuration.md`,
  `workflow-profiles.md` — drop `ready_for_X`/`in_X` and `status_flow`/`status_metadata`,
  describe steps/outcomes/claim/release.

### Impact

- The harness loop is trivial again: claim, run, release — no status names in the loop.
- `shark validate` is the single gate that catches structural defects across F01–F05.
- Docs match the shipped engine; no stale marker/two-map vocabulary remains.

---

## Scope

- `shark/SKILL.md` (canonical embedded copy under `internal/sharkdata/default_data/`):
  rewrite the dispatch loop to claim → run → release; document the release payload
  contract (`outcome`/`reason`/`log`).
- `shark validate`: author the four structural checks above (consuming F02's
  outcome resolution, F04's bundle resolver, F05's alias maps).
- Docs: `CLAUDE.md`, `docs/cli-reference/workflow-configuration.md`,
  `docs/guides/workflow-profiles.md` — vocabulary and examples updated to the new model.
- Update any harness-facing examples that still call `advance`/`next-status` with a
  target instead of `release(outcome)`.

### Out of Scope

- The engine behaviors themselves (routing F02, lease F03, resolution F04, migration
  F05) — this feature *uses* and *documents* them.
- E32 docs unrelated to the marker/two-map vocabulary.

---

## Acceptance Criteria

1. The `shark/SKILL.md` dispatch loop is `claim → run → release`; it names no status
   and passes the skill's `{ outcome, reason, log }` payload to `release`.
2. `shark validate` fails on: an unresolved outcome target; a non-terminal step
   missing a core outcome; an old status with no alias home; a prompt/skill/agent that
   doesn't resolve from the bundle root. It passes on the canonical tree.
3. `CLAUDE.md`, `workflow-configuration.md`, and `workflow-profiles.md` contain no
   `ready_for_X`/`in_X` or `status_flow`/`status_metadata` references and describe the
   step/outcome/claim/release model.
4. A fresh end-to-end run (claim → run → release) advances a canonical entity through
   its workflow using only the rewritten harness loop.

---

## Verification

- `shark validate` exit-code tests for each failure mode and the passing canonical tree.
- Manual/e2e: drive one entity through several steps via the rewritten SKILL.md loop.
- Doc grep: `grep -rE "ready_for_|in_[a-z]+|status_flow|status_metadata"` over the
  updated docs returns nothing (outside historical/design references).
- `make fmt && make lint && make test` pass.

---

## Dependencies

- **Blocked by**: F02 (release + outcome resolution), F03 (claim/lease),
  F04 (bundle resolver), F05 (alias coverage). Effectively the integration capstone.
- **Blocks**: none — epic exit.

---

*Last Updated*: 2026-06-23
