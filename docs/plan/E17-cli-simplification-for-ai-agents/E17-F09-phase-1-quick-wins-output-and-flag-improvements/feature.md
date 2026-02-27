---
feature_key: E17-F09
epic_key: E17
title: Phase 1 Quick Wins: Output and Flag Improvements
description: Groups XS-complexity atomic changes from E17 Phase 1 that each affect a single file and deliver a single capability. Consolidating them under one feature avoids feature overhead for changes that are each 5-15 lines of code. Includes SHARK_OUTPUT env var and flag normalization.
execution_order: 1
phase: 1
complexity: XS
status: in_scope_validation
dependencies: []
depended_on_by:
  - E17-F03 (SHARK_OUTPUT=json must be active when F03 JSON error output is active)
---

# Phase 1 Quick Wins: Output and Flag Improvements

**Feature Key**: E17-F09
**Phase**: 1 (Must-Have)
**Complexity**: XS (each task is 5-15 lines)
**Execution Order**: 1 (implement first -- smallest changes, immediate agent value)

---

## Purpose

This feature is a consolidation container for E17 Phase 1 changes that individually do not meet the feature threshold (single file, single capability, XS complexity). Rather than carry two standalone "features" with full feature overhead, they are grouped here as tasks.

**Original feature entries cancelled and converted:**
- E17-F04 (SHARK_OUTPUT Environment Variable) -> T-E17-F09-001
- E17-F05 (Flag Normalization) -> candidate for T-E17-F09-002

---

## Scope

### Problem

AI agents must pass `--json` on every shark command invocation (no session-wide setting), and flag names are inconsistent across commands (`--execution-order` vs `--order`, `--show-all` vs `--all`). These are small ergonomic friction points with simple, isolated fixes.

### Solution

Two atomic changes to the CLI:
1. Read `SHARK_OUTPUT` env var in root.go to enable session-wide JSON mode
2. Normalize flag names using Cobra's `MarkDeprecated()` to establish `--order` and `--all` as primary names

### What This Feature Does

- Adds `SHARK_OUTPUT=json` environment variable support (session-wide JSON mode)
- Normalizes `--execution-order` to `--order` and `--show-all` to `--all` (with deprecated aliases)

### What This Feature Does NOT Do

- Does not change default output format
- Does not remove existing flags (backward compatible)
- Does not introduce new commands

---

## Tasks

| Task | Title | Complexity | File(s) Changed |
|------|-------|------------|-----------------|
| T-E17-F09-001 | SHARK_OUTPUT Environment Variable | XS | `internal/cli/root.go` |
| T-E17-F09-002 | Flag Normalization | XS | `internal/cli/commands/*.go` |

---

## Acceptance Criteria

- [ ] `SHARK_OUTPUT=json shark get E17-F09` returns JSON output
- [ ] `SHARK_OUTPUT=json` applies to all commands
- [ ] `--json` flag overrides env var
- [ ] `PM_OUTPUT=json` also works as fallback
- [ ] `--order` accepted everywhere `--execution-order` was accepted
- [ ] `--execution-order` kept as hidden alias with deprecation warning on stderr
- [ ] `--all` replaces `--show-all` on list commands
- [ ] `--show-all` kept as hidden alias with deprecation warning on stderr
- [ ] All existing tests pass (`make test` green)

---

## Dependencies

### Depends On

None. Both tasks are standalone changes.

### Depended On By

- **E17-F03 (Structured JSON Errors)**: When `SHARK_OUTPUT=json` is active (T-E17-F09-001), F03's structured error output should also be active.
- **E17-F08 (Phase 2 unified create)**: Uses normalized flag names from T-E17-F09-002.

---

*Last Updated*: 2026-02-25
