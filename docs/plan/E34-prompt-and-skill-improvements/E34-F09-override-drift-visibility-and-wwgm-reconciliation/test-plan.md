---
feature_key: E34-F09-override-drift-visibility-and-wwgm-reconciliation
epic_key: E34
title: Override Drift Visibility and WWGM Reconciliation — Test Plan
---

# E34-F09 Test Plan

This is a real Go/CLI feature (new package-level functions in
`internal/sharkdata` plus a new `internal/cli/commands/overrides_cmd.go`
command group) — full Caller-Path Contracts apply to every TC below, per this
repo's `.claude/rules/testing/architecture.md` golden rule: CLI-level tests
use mocked repositories (there are none here — the new code is filesystem-only,
no DB); package-level tests in `internal/sharkdata` exercise `OverrideStatusAt`
etc. directly against a temp-directory fixture tree, matching the existing
pattern other `internal/sharkdata` tests use for embedded/override-tree
fixtures.

## Traceability Matrix

| Spec Acceptance Criterion | Test Case | Covered? |
|---|---|---|
| Empty/absent `overrides/` → all-zero summary | TC-001 | Yes |
| `upstream_changed` (baseline ≠ current canonical) | TC-002 | Yes |
| `identical_redundant` (override bytes == canonical bytes) | TC-003 | Yes |
| `orphaned` (no canonical counterpart) | TC-004 | Yes |
| `baseline_unknown` (missing/corrupt/unrecognized-version manifest) | TC-005 | Yes |
| Symlink under `overrides/` → `baseline_unknown`, never followed | TC-006 | Yes |
| `acknowledge` success updates only the manifest, bytes/mtime-visible content unchanged | TC-007 | Yes |
| `acknowledge` failure (no counterpart / no override file) writes nothing | TC-008 | Yes |
| `shark admin upgrade [--json/--dry-run]` includes populated `overrides` counts, schema-stable | TC-009 | Yes |
| Real (non-dry-run) upgrade leaves override inventory, bytes, mode bits, and baseline manifest byte-identical | TC-010 | Yes |
| Classification order (symlink > orphaned > baseline_unknown > identical_redundant > upstream_changed/current) | TC-011 | Yes |
| Path safety (absolute path / `..` segment rejected) | TC-012 | Yes |
| JSON field names and enum spellings are golden-asserted (REQ-NF-001) | TC-013 | Yes |
| Process ACs (REQ-F-005–008, WWGM reconciliation checklist / linked work item / safeguards) | Deferred — verified outside this feature's Go suite, per spec.md | N/A (process, not code) |

## Caller-Path Contracts

| TC | Entrypoint | Contract |
|---|---|---|
| TC-001–TC-006, TC-011–TC-013 | `sharkdata.OverrideStatusAt(dataRoot)` | Production entrypoint — no mock, real `filepath.WalkDir` over a temp-directory fixture tree built by the test |
| TC-007, TC-008 | `sharkdata.AcknowledgeOverrides(dataRoot, paths)` | Production entrypoint — same temp-directory fixture pattern |
| TC-009, TC-010 | `runSharkUpgrade` (CLI command) via `internal/cli/commands` test harness | Production entrypoint — internal — function under test is the production entrypoint; mocks nothing (filesystem-only, no repository dependency) |

## Test Cases

- **TC-001** — AC "empty/no overrides dir": `OverrideStatusAt` on a `dataRoot`
  with no `overrides/` directory (and separately, with an empty one) returns
  `Rows: []` and `Summary` with all five keys present at `0`. Both variants
  run as subtests.
- **TC-002** — AC "upstream_changed": fixture has an override file, a
  manifest entry whose recorded SHA-256 differs from the current embedded
  canonical file's SHA-256. Assert `Classification == "upstream_changed"`,
  all three digest fields non-empty.
- **TC-003** — AC "identical_redundant": override file bytes equal canonical
  bytes; assert `identical_redundant` regardless of whether a baseline entry
  exists (run as two subtests: with and without a baseline entry, both must
  classify identically).
- **TC-004** — AC "orphaned": override file with no canonical counterpart at
  that relative path; assert `orphaned`, `CanonicalSHA256 == ""`.
- **TC-005** — AC "baseline_unknown": four subtests — missing manifest file,
  manifest present but path has no entry, manifest JSON is corrupt/unparseable,
  manifest `schema_version` is not `1`. All four must classify `baseline_unknown`,
  never silently upgraded or treated as `current`.
- **TC-006** — AC "symlink handling": a symlink placed under `overrides/`
  (pointing anywhere, including outside `dataRoot`) classifies
  `baseline_unknown` with a `SuggestedAction` naming the symlink problem;
  assert the file is never opened/read (fixture target file has sentinel
  content that must not appear in any error message or row field).
- **TC-007** — AC "acknowledge success": call `AcknowledgeOverrides` for a
  `baseline_unknown` and a separate `upstream_changed` path (two subtests),
  each with a canonical counterpart; assert the returned report reclassifies
  both as `current`; assert `.shark-override-baselines.json`'s new entry
  equals the current canonical SHA-256; assert the override file's bytes and
  `os.Stat` mtime are unchanged before/after (explicit byte + mtime
  comparison, not just "no error").
- **TC-008** — AC "acknowledge failure, no partial writes": two subtests —
  (a) path has no canonical counterpart, (b) path has no override file at all.
  Both must return a non-zero-exit error naming the path and leave
  `.shark-override-baselines.json` byte-for-byte unchanged (assert via
  before/after file read, not just "no new entry").
- **TC-009** — AC "upgrade output schema": run `shark admin upgrade --json`
  and `--dry-run --json` against a fixture with a mix of all five
  classifications; assert the `overrides` object has all five keys with
  correct counts in both modes; assert the four pre-existing keys
  (`added`/`updated`/`unchanged`/`skipped_overrides`) are still present and
  unchanged in shape (regression guard against accidentally dropping them).
  A third subtest with zero overrides asserts all-zero counts are present
  (not an omitted key).
- **TC-010** — AC "upgrade byte-identity guarantee": fixture with sentinel
  override files (fixed bytes, fixed `0644` mode) and a populated baseline
  manifest. Run a real (non-dry-run) `shark admin upgrade`. Assert: the set
  of paths under `overrides/` is unchanged, every override file's SHA-256 is
  unchanged, every override file's `os.FileMode` bits are unchanged (explicit
  `os.Stat().Mode()` comparison — not only a byte-content comparison), and
  `.shark-override-baselines.json`'s bytes are unchanged (only `acknowledge`
  writes that file; a plain upgrade must not).
- **TC-011** — Classification-order regression guard: a fixture engineered
  to be simultaneously eligible for two classifications if order were wrong
  (e.g. a symlink whose target bytes would equal canonical bytes) must
  resolve to the higher-priority classification (symlink →
  `baseline_unknown`, checked before the `identical_redundant` byte
  comparison) per spec.md's stated step order.
- **TC-012** — Path-safety regression guard: an override path containing an
  absolute path or a `..` segment (constructed via a crafted fixture, since
  `filepath.WalkDir` itself won't normally produce these — this tests the
  normalization/rejection logic directly with a hand-built input) is rejected
  before being joined onto `dataRoot`; assert no read outside `dataRoot`
  occurs (e.g. via a sentinel file placed just outside the fixture root that
  must never appear in any row).
- **TC-013** — REQ-NF-001 golden-output test: assert the exact JSON key names
  (`path`, `classification`, `override_sha256`, `canonical_sha256`,
  `baseline_sha256`, `suggested_action`) and the five classification string
  values (`current`, `upstream_changed`, `identical_redundant`, `orphaned`,
  `baseline_unknown`) via `encoding/json` marshal-and-compare against a fixed
  golden map — an accidental rename fails this test, not just at runtime.
  Also assert `.shark-override-baselines.json`'s `schema_version`/`baselines`
  key names the same way.

## Cross-feature contract tests

| I-## | Role | Contract test |
|---|---|---|
| I-05 (CanonicalAdoptionManifest v1) | Consumes (informational — this feature's Go/CLI surface does not parse I-05; see spec.md "Cross-feature interactions") | N/A — no Go-level contract test; the process step (feature.md Implementation-plan step 4) is verified outside this test suite |

No I-## row names E34-F09 as a producer (spec.md confirmed via
`E34-interaction-map.md` grep).

## Exit gate

- Every spec.md AC has at least one TC (see Traceability Matrix above); no
  row is "Covered? No".
- `make fmt && make lint && make test` pass with the new files included.
- `git diff --check` clean (no whitespace errors).

*Last Updated*: 2026-09-04
