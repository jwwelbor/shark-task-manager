---
feature_key: E34-F09-override-drift-visibility-and-wwgm-reconciliation
epic_key: E34
title: Override Drift Visibility and WWGM Reconciliation — Specification
---

# E34-F09 Specification

See [Epic PRD](../epic.md) for business context and
[architecture.md](../architecture.md) for the E34-wide I-05 contract and the
"Override baseline architecture" decisions this spec implements verbatim. See
[research-report.md](./research-report.md) for the Capability map this spec
builds on. See [feature.md](./feature.md) for the full requirements text
(REQ-F-001–008, REQ-NF-001) — this spec adds only file- and function-level
implementation detail feature.md does not carry.

This feature has two parts with very different shapes:

- **Go/CLI part** (REQ-F-001–004, REQ-NF-001): a new read-only classifier and
  baseline-provenance mechanism inside `internal/sharkdata` plus two new `shark
  admin overrides` subcommands. This is the only part this spec designs code
  for.
- **Cross-repository reconciliation part** (REQ-F-005–008): promoting WWGM's
  reusable prompt/skill behavior through E34-F06–F08, and later reconciling
  the WWGM repository itself using the Go tooling this spec builds. No Shark
  Go code implements this part — it is WWGM-repository and process work,
  scoped in feature.md's "Implementation plan" steps 4–6 and "WWGM override
  disposition" table. This spec does not restate that table; see feature.md.

## Requirements (incremental over epic)

Traces to feature.md REQ-F-001 through REQ-F-004 and REQ-NF-001 for the Go/CLI
surface (feature.md already states these at full fidelity). REQ-F-005–008 are
process/cross-repository requirements with no Shark code surface; this spec
does not add implementation detail for them beyond noting the consumption
point of I-05 (see "Cross-feature interactions" below).

### Functional

- **REQ-F-001 (spec)**: Add `shark admin overrides status [--json]`, wired as
  a new `overridesCmd` command group in a new file
  `internal/cli/commands/overrides_cmd.go`, following the `cloudCmd` /
  `configCmd` nested-group pattern in `internal/cli/commands/cloud.go`
  (`adminCmd.AddCommand(overridesCmd)`, then `overridesCmd.AddCommand(...)` for
  `status` and `acknowledge`). The command resolves the project root via
  `cli.FindProjectRoot()` and the data root via
  `config.ResolveSharkDataRoot(root, configBytes)` — the same two calls
  `runSharkUpgrade` already uses in `internal/cli/commands/sharkdata_cmd.go` —
  then delegates to a new `sharkdata.OverrideStatusAt(dataRoot)` function.
- **REQ-F-002 (spec)**: Add `internal/sharkdata/overrides_status.go` with:
  - `OverrideStatusAt(dataRoot string) (*OverrideStatusReport, error)` — walks
    `<dataRoot>/overrides/` with `filepath.WalkDir`, following the existing
    symlink-rejection pattern in `resolveEffectiveSharkAttackRoster`
    (`internal/sharkdata/embed.go`, checks `info.Mode()&os.ModeSymlink != 0`
    and fails closed) applied per-file rather than to one roster path. A
    symlink anywhere under `overrides/` is reported as a `baseline_unknown`
    row with a suggested action of "replace symlink with a regular file", not
    silently skipped and not resolved/read.
  - For each regular file found, compute its relative path (POSIX-normalized
    relative to `overrides/`, matching the join convention `walkEmbedded`
    already uses for canonical bundle paths in `embed.go`), then look up the
    corresponding canonical file at that relative path via the package-local
    `ReadEmbedded(relPath)` (already exported from `internal/sharkdata`,
    called unqualified as a package sibling) to get the current canonical
    bytes, and read the
    baseline manifest (REQ-F-002 baseline type below) for the recorded digest.
  - `OverrideRow` fields: `Path`, `Classification`, `OverrideSHA256`,
    `CanonicalSHA256` (empty when no counterpart), `BaselineSHA256` (empty
    when unknown), `SuggestedAction`.
  - Classification logic (five states, matching feature.md's "Override status
    contract" table verbatim). Order matters — each step below applies only
    if no earlier step matched:
    0. Path is a symlink (or any other non-regular file) anywhere under
       `overrides/` → `baseline_unknown`, regardless of whether a canonical
       counterpart exists at that path. This step runs before the counterpart
       check in step 1 specifically so a symlinked path is never
       misclassified as `orphaned` when a counterpart does exist; the file is
       never opened to compare bytes. `SuggestedAction` names the symlink
       problem explicitly (e.g. "replace symlink with a regular file").
    1. No canonical counterpart → `orphaned`.
    2. Canonical counterpart exists, no trustworthy baseline entry for this
       path → `baseline_unknown` (covers missing manifest, missing entry,
       corrupt JSON, and any manifest schema_version this binary does not
       recognize — never silently upgraded to a lower/no-op version).
    3. Override bytes SHA-256 equal canonical bytes SHA-256 → `identical_redundant`
       (checked before baseline comparison — bytes equality is definitive
       regardless of baseline state).
    4. Baseline SHA-256 differs from current canonical SHA-256 →
       `upstream_changed`.
    5. Baseline SHA-256 equals current canonical SHA-256 (and override bytes
       differ from canonical, else case 3 already matched) → `current`.
  - Rows sorted by `Path` (Go `sort.Slice` on the POSIX relative path string)
    for deterministic human and JSON output, matching REQ-F-001's determinism
    requirement.
  - `OverrideStatusReport` also carries `Summary map[string]int` keyed
    by the same five classification strings (zero-valued entries included for
    JSON stability, mirroring the existing five-key vocabulary rather than
    omitting empty classes). JSON envelope keys are fixed as `"overrides"`
    (the row array) and `"summary"` (the counts map) — both names are part of
    REQ-NF-001's stability contract alongside the per-row field names below.
- **REQ-F-002 (baseline manifest)**: Add
  `internal/sharkdata/override_baseline.go` with:
  - `type OverrideBaselineManifest struct { SchemaVersion int; Baselines map[string]string }`
    (JSON tags `schema_version`, `baselines`), matching the "schema version
    and a map from normalized relative canonical path to SHA-256" contract in
    architecture.md's "Override baseline architecture" section verbatim.
  - `LoadOverrideBaselines(dataRoot string) (*OverrideBaselineManifest, error)`
    reads `<dataRoot>/.shark-override-baselines.json`; a missing file returns
    an empty manifest (`SchemaVersion: 1, Baselines: map[string]string{}`) and
    a nil error — status must still function on a project with zero
    acknowledgements. A file that exists but fails to parse, or whose
    `schema_version` is not exactly `1`, returns a typed
    `ErrInvalidBaselineManifest` sentinel; callers (status, upgrade) treat
    every path as `baseline_unknown` rather than surfacing the parse error as
    a hard failure, per REQ-F-004 ("never rewrite... an override") and
    REQ-F-002 ("An unknown or invalid baseline must classify as
    `baseline_unknown`, not `current`").
  - `(*OverrideBaselineManifest) Save(dataRoot string) error` writes the file
    with `os.WriteFile(..., 0644)` after `json.MarshalIndent` with sorted map
    keys (Go's `encoding/json` already sorts map keys on marshal), so the file
    is diff-stable in version control.
  - The manifest file never contains override bytes, only paths and SHA-256
    hex digests — enforced by the struct shape itself (no bytes field exists
    to populate).
- **REQ-F-002 (acknowledge)**: Add `shark admin overrides acknowledge
  <path>...` (variadic, `cobra.MinimumNArgs(1)`) to
  `internal/cli/commands/overrides_cmd.go`, delegating to a new
  `sharkdata.AcknowledgeOverrides(dataRoot string, paths []string) (*OverrideStatusReport, error)`
  in `override_baseline.go`. For each path: reject it (return an error naming
  the path, no partial writes to disk) unless a regular override file exists
  at `<dataRoot>/overrides/<path>` **and** a canonical counterpart exists via
  `ReadEmbedded` — matching REQ-F-004 ("Acknowledge... requires an
  existing override and current canonical counterpart"). On success for every
  path, load the manifest, set `Baselines[path]` to the current canonical
  SHA-256 for every path, `Save` once, and return the refreshed status report
  so the CLI can print the new classifications (expected to move to
  `current`). Acknowledge never touches override bytes.
- **REQ-F-003 (spec)**: Extend `runSharkUpgrade` in
  `internal/cli/commands/sharkdata_cmd.go` to call
  `sharkdata.OverrideStatusAt(dataRoot)` after computing `summary` (for both
  the real run and `--dry-run` — `OverrideStatusAt` is inherently read-only so
  no branching is needed there) and add an `overrides` object to both the
  human and JSON output:
  - JSON: add `"overrides": {"current": N, "upstream_changed": N,
    "identical_redundant": N, "orphaned": N, "baseline_unknown": N}` alongside
    the existing `added`/`updated`/`unchanged`/`skipped_overrides` keys (all
    four preserved unchanged, satisfying REQ-F-003's backward-compatibility
    clause).
  - Human output: append a line `  overrides: <counts> (run 'shark admin
    overrides status' for detail)` after the existing four summary lines.
  - A project with zero overrides produces all-zero counts, not an omitted
    field, so JSON consumers can rely on the key's presence.
- **REQ-F-004 (spec)**: No code path in `overrides_status.go`,
  `override_baseline.go`, or the two new CLI commands opens an override or
  canonical file in write mode except `AcknowledgeOverrides`'s single
  `Save` of the baseline manifest. `status` and `upgrade` (including
  `--dry-run`) call only `OverrideStatusAt`, which contains no `os.WriteFile`
  or `os.Remove` call. Error and row messages interpolate only `Path`,
  `OverrideSHA256`, `CanonicalSHA256`, `BaselineSHA256`, and
  `SuggestedAction` — never file contents — matching the existing `%q`-quoted,
  content-free error convention in `embed.go`'s validators.

### Non-functional

- **REQ-NF-001 (spec)**: `OverrideRow` JSON field names
  (`path`, `classification`, `override_sha256`, `canonical_sha256`,
  `baseline_sha256`, `suggested_action`), the five classification string
  values, and `.shark-override-baselines.json`'s `schema_version`/`baselines`
  keys are covered by golden-output-style table tests in
  `internal/sharkdata/overrides_status_test.go` asserting exact JSON key names
  and enum spellings, so an accidental rename fails CI rather than only
  breaking at runtime.
- Path safety: `overrides_status.go` reuses the same relative-path
  normalization Shark already applies to prompt includes (reject absolute
  paths and any segment equal to `..`, per the pattern in
  `validatePromptIncludes` in `embed.go`) before joining a walked path onto
  `dataRoot`, so a maliciously named override file cannot escape the data
  root when its path is echoed back or looked up.

### Acceptance criteria

- Given an empty `overrides/` directory (or no `overrides/` directory at
  all), `shark admin overrides status --json` exits 0 and returns
  `{"overrides": [], "summary": {"current": 0, "upstream_changed": 0,
  "identical_redundant": 0, "orphaned": 0, "baseline_unknown": 0}}`.
- Given an override with a recorded baseline whose digest no longer matches
  the embedded canonical digest, status reports `upstream_changed` with all
  three digests populated and non-empty.
- Given an override whose bytes equal the current canonical bytes, status
  reports `identical_redundant` regardless of whether a baseline entry
  exists.
- Given an override with no canonical counterpart, status reports `orphaned`
  with `canonical_sha256` empty.
- Given an override with a canonical counterpart but no manifest entry (or a
  corrupt/absent manifest), status reports `baseline_unknown`, never
  `current`.
- Given a symlink under `overrides/`, status reports that path as
  `baseline_unknown` with a suggested action naming the symlink problem, and
  never follows it to read file contents outside `overrides/`.
- Running `shark admin overrides acknowledge <path>` for a `baseline_unknown`
  or `upstream_changed` path with a canonical counterpart updates only
  `.shark-override-baselines.json`; a re-run of `status` immediately after
  reports that path as `current`. The override file's bytes and mtime-visible
  content are unchanged (byte-for-byte compared before/after in the test).
  Running `acknowledge` for a path with no canonical counterpart, or no
  override file, fails with a non-zero exit and writes nothing.
- `shark admin upgrade --json` and `shark admin upgrade --dry-run --json` both
  include a populated `overrides` counts object with the same schema; running
  `--dry-run` twice in a row against a repo with a `baseline_unknown` row
  leaves `.shark-override-baselines.json` and every override file byte-for-byte
  unchanged.
- Given a project with sentinel override files (fixed bytes and fixed file
  mode, e.g. `0644`) and a populated `.shark-override-baselines.json`, running
  a real (non-dry-run) `shark admin upgrade` leaves the override inventory
  (the set of paths under `overrides/`), every override file's byte digest,
  every override file's mode bits, and the baseline manifest's bytes
  identical before and after — a real upgrade only ever writes canonical
  files outside `overrides/` and, for `overrides_status.go`/`upgrade`
  specifically, never writes `.shark-override-baselines.json` at all (only
  `acknowledge` writes it). This is asserted with an explicit `os.Stat` mode
  comparison in the test, not only a byte comparison, matching feature.md's
  verification-plan phrase "byte digests, and baseline manifest remain
  identical" together with "mode metadata."
- **Process ACs for REQ-F-005–008** (verified outside this feature's Go test
  suite, as part of closing the linked WWGM work item — listed here so every
  requirement has a stated, checkable exit condition per this spec's exit
  gate):
  - A path-by-path WWGM reconciliation checklist exists and its path column
    covers every path present in a `shark admin overrides status --json` run
    against the live WWGM override inventory at the time I-05 is consumed —
    no override path is silently absent from the checklist.
  - Exactly one linked WWGM adoption work item exists for this reconciliation
    (not zero, not more than one); CC-007 and CC-008 are linked to it or
    resolved by it, never duplicated by a new record covering the same
    finding.
  - The WWGM-local deterministic safeguards feature.md's REQ-F-006 lists
    (method-length/test-selection checks, test database setup, unexpected-skip
    enforcement, standards, bare-assert lint, thin root `AGENTS.md`) are
    present in WWGM's version-controlled tree (not only in the generated,
    gitignored `shark-data/` tree) once the work item closes.
- `make fmt && make lint && make test` pass with the new files included.

### Out of scope

- Everything feature.md's "Out of scope" section already states (automatic
  merge/delete/rewrite of overrides; storing override contents in the
  manifest or telemetry; upstreaming WWGM's rules/scripts/models/order as
  universal defaults; a universal owner-approval config change; blocking on
  E40).
- Any Go implementation of REQ-F-005 through REQ-F-008: those are
  cross-repository/process requirements executed by the linked WWGM work item
  and the E40 epic, using the CLI this spec ships as a tool, not a Shark code
  change.
- A `shark admin overrides diff` or content-printing command. Feature.md's
  REQ-F-004 and the "never... print file contents" contract text this spec
  implements both rule that out; an operator diffs manually with the digests
  status provides.

## Architecture

### Component changes

| File | Change |
|---|---|
| `internal/sharkdata/overrides_status.go` | New. `OverrideStatusAt`, `OverrideRow`, `OverrideStatusReport`, classification constants, symlink-safe walker. |
| `internal/sharkdata/override_baseline.go` | New. `OverrideBaselineManifest`, `LoadOverrideBaselines`, `Save`, `AcknowledgeOverrides`, `ErrInvalidBaselineManifest`. |
| `internal/sharkdata/overrides_status_test.go` | New. Five-classification table tests, symlink rejection, JSON field/enum golden assertions, path-safety tests. |
| `internal/sharkdata/override_baseline_test.go` | New. Missing/corrupt/wrong-version manifest handling, acknowledge success/failure paths, byte-identity of override files across acknowledge. |
| `internal/cli/commands/overrides_cmd.go` | New. `overridesCmd` group (`adminCmd.AddCommand(overridesCmd)`), `status` and `acknowledge` subcommands, human/JSON rendering. |
| `internal/cli/commands/overrides_cmd_test.go` | New. CLI-level human/JSON output tests, mirroring the existing `sharkdata_cmd_test.go` structure. |
| `internal/cli/commands/sharkdata_cmd.go` | Modified. `runSharkUpgrade` gains the `sharkdata.OverrideStatusAt` call and the `overrides` counts block in both JSON and human output; `Long` help text for `sharkUpgradeCmd` gains one line pointing at `shark admin overrides status`. |
| `internal/cli/commands/sharkdata_cmd_test.go` | Modified. Existing upgrade-output tests updated to expect the new `overrides` key (additive; no existing assertion is removed). |
| `docs/cli-reference/setup-commands.md` (or the nearest existing admin-command reference doc) | Modified. Document the two new subcommands and the extended upgrade output, per this repo's existing pattern of keeping `docs/cli-reference/` in sync with new commands. |

No new command touches `internal/services/`, `internal/repository/`, or any
SQLite schema — this feature is entirely file-based (embedded bundle + one
project-local JSON manifest) and has no database persistence, consistent with
`internal/sharkdata` having no repository dependency today.

### Data model changes

No SQLite schema change; no migration; `CurrentSchemaVersion` in
`internal/db/db.go` is untouched (see `.claude/rules/database-critical.md`,
which applies only to that schema, not this feature).

The one new persistent artifact is a project-local JSON file,
`<resolved shark_data_path>/.shark-override-baselines.json` (path fixed by
architecture.md's "Override baseline architecture" section — not
hard-coded to a repository-relative `shark-data/` directory, matching
REQ-F-002's `<resolved shark_data_path>` requirement: it is built from the
same `dataRoot` value `config.ResolveSharkDataRoot` returns, so a
custom `shark_data_path` relocates the manifest alongside the bundle it
describes):

```json
{
  "schema_version": 1,
  "baselines": {
    "prompts/feature/approval.md": "<sha256 hex>",
    "workflow/epic.yaml": "<sha256 hex>"
  }
}
```

Keys are POSIX-normalized paths relative to `overrides/` (i.e. the same
relative path used both under `overrides/<path>` and as the canonical
counterpart's embedded path) — not absolute paths, and not prefixed with
`overrides/`, so a moved `shark_data_path` does not invalidate recorded
baselines.

### API / interface contracts

New Go-internal contracts (no HTTP API; `internal/sharkdata` has no HTTP
handler today and this feature does not add one):

```
OverrideStatusAt(dataRoot string) (*OverrideStatusReport, error)

type OverrideStatusReport struct {
    Rows    []OverrideRow
    Summary map[string]int // keys: current, upstream_changed, identical_redundant, orphaned, baseline_unknown
}

type OverrideRow struct {
    Path            string
    Classification  string
    OverrideSHA256  string
    CanonicalSHA256 string // "" when orphaned
    BaselineSHA256  string // "" when baseline_unknown
    SuggestedAction string
}

LoadOverrideBaselines(dataRoot string) (*OverrideBaselineManifest, error)
(*OverrideBaselineManifest) Save(dataRoot string) error
AcknowledgeOverrides(dataRoot string, paths []string) (*OverrideStatusReport, error)
```

CLI surface:

```
shark admin overrides status [--json]
shark admin overrides acknowledge <relative-override-path>... [--json]
```

JSON output shape for `status`:

```json
{
  "overrides": [
    {
      "path": "workflow/sprint.yaml",
      "classification": "upstream_changed",
      "override_sha256": "...",
      "canonical_sha256": "...",
      "baseline_sha256": "...",
      "suggested_action": "review upstream canonical change before rebasing this override"
    }
  ],
  "summary": {
    "current": 2,
    "upstream_changed": 1,
    "identical_redundant": 0,
    "orphaned": 0,
    "baseline_unknown": 3
  }
}
```

Extended `shark admin upgrade --json` shape (additive key only):

```json
{
  "dry_run": false,
  "added": [], "updated": [], "unchanged": [], "skipped_overrides": [],
  "overrides": {"current": 2, "upstream_changed": 1, "identical_redundant": 0, "orphaned": 0, "baseline_unknown": 3}
}
```

### Key technical decisions

1. **Classification lives in `internal/sharkdata`, not `internal/cli/commands`**
   — mirrors the existing split where `embed.go`'s `ValidateAt`/`UpgradeAt`
   contain all logic and `sharkdata_cmd.go` is a thin adapter (per this file's
   own header comment: "these handlers are thin adapters"). Keeps the
   classifier unit-testable without cobra/CLI plumbing.
2. **Reuse `sharkdata.ReadEmbedded` for canonical lookup rather than
   re-walking the embedded FS per override** — `ReadEmbedded` is exported
   from the same `internal/sharkdata` package the new classifier lives in
   (`embed.go`) and is the established way to fetch one canonical file's
   bytes by relative path; no new embedded-FS traversal helper is needed, and
   `overrides_status.go` calls it unqualified as a package-internal sibling.
3. **Baseline manifest is a flat `map[string]string`, not a richer struct
   with timestamps or acknowledger identity** — feature.md's "Out of scope"
   and REQ-F-004 rule out storing anything beyond what a digest comparison
   needs; a flat map is the simplest structure that satisfies REQ-NF-001's
   determinism requirement (stable JSON key order via Go's map-key sorting on
   marshal).
4. **`identical_redundant` is checked before baseline lookup** — bytes
   equality is a stronger, self-contained signal than any baseline state;
   checking it first avoids a spurious `upstream_changed` result on a file
   that happens to be byte-identical to canonical today even though its
   recorded baseline is stale.
5. **Symlinks classify as `baseline_unknown` with a corrective suggested
   action rather than erroring the whole `status` call** — consistent with
   REQ-F-001's "reject symlinks... consistently with bundle validation" and
   with this feature's read-only, never-fail-the-whole-command posture;
   `resolveEffectiveSharkAttackRoster`'s existing symlink check errors out
   entirely because it validates one required roster file, whereas `status`
   walks an open-ended tree of independent override files and a single bad
   entry must not hide every other row's classification.
6. **No new sentinel error type beyond `ErrInvalidBaselineManifest`** — every
   other failure mode (missing manifest, missing canonical counterpart) is
   modeled as a classification value or a plain CLI-level `error`, per the
   project's guidance to avoid unnecessary abstraction for single-use code.

### Integration with existing code

- `internal/cli/commands/sharkdata_cmd.go`: `runSharkUpgrade` (existing
  function, currently lines ~257–293) gains one call to
  `sharkdata.OverrideStatusAt(dataRoot)` after `summary, err :=
  sharkdata.UpgradeAt(...)` succeeds, and the new `overrides` block is added
  to both the `cli.OutputJSON` map literal and the human `fmt.Printf` block
  immediately below it. No existing field, key, or line is removed.
- `internal/sharkdata/embed.go`: no changes to `UpgradeAt`, `ValidateAt`, or
  any validator — this feature only reads what those functions already
  compute (via a fresh `OverrideStatusAt` call) and reads `ReadEmbedded`
  for canonical bytes; it does not alter upgrade/validate behavior or touch
  `DiffSummary`.
- `internal/config/aliases.go` / `internal/config/config.go`:
  `config.ResolveSharkDataRoot` (existing, exported) is called unchanged from
  the new `overrides_cmd.go`, identically to how `sharkdata_cmd.go` already
  calls it — no changes to `internal/config`.
- `admin.go`'s `adminCmd` gains one more child command group
  (`overridesCmd`), following the exact registration pattern
  `cloudCmd`/`configCmd`/`workflowCmd` already use.

## Cross-feature interactions

### Consumes

- **I-05** — CanonicalAdoptionManifest v1. Producer: E34-F08. Shape source:
  [architecture.md#i-05-canonicaladoptionmanifest-v1](../architecture.md#i-05-canonicaladoptionmanifest-v1).
  Contract test: **N/A** — this feature's Go/CLI surface
  (`status`/`acknowledge`/upgrade counts) does not itself read or parse the
  I-05 manifest, so there is no Go-level contract test to anchor (see
  test-plan.md's Cross-feature contract tests table, which records this
  explicitly rather than naming a nonexistent anchor). I-05 is consumed by the
  process step in feature.md's Implementation-plan step 4 ("Consume I-05 and
  produce a path-by-path WWGM reconciliation checklist"), which is
  human/agent work product, not a Go code path. No Go type in this spec
  deserializes I-05; recording that explicitly here avoids implying a
  manifest-parsing component that does not exist in this feature's scope.

No I-## row in `E34-interaction-map.md` names E34-F09 as a producer.

## Cross-epic integrations

### Produces

- **X-14** — Compare canonical E34 policy with a reconciled project-style
  configuration, retaining identity for tier, evidence, recurrence, and
  integration-closure scenarios. Producer: E34, owning feature E34-F09 per
  the map row (this feature produces the artifact identity X-14 compares;
  it does not itself run or validate the comparison — that is E40's role
  as consumer). Consumer: E40 — Shark Bench. Contract / shape
  source: [architecture.md#i-05-canonicaladoptionmanifest-v1](../architecture.md#i-05-canonicaladoptionmanifest-v1)
  and this spec's REQ-F-008 pointer to feature.md § Requirements 8, matching
  `E34-cross-epic-map.md` row X-14 verbatim. UX/CX handoff notes: none — this
  is a CLI-only artifact contract with no end-user screen, per the cross-epic
  map's header note. Test coverage: **explicit deferral** — X-14's own row
  states its test coverage pointer is `TBD — E40 decomposition must add the
  benchmark scenario and coverage pointer`; this spec does not create that
  pointer (out of scope per REQ-F-008: "Do not block F09 delivery on E40
  execution"). This spec's contribution to X-14 is limited to the
  digest-stable `.shark-override-baselines.json` manifest and the
  `override_sha256`/`canonical_sha256` fields `status` emits, which give E40 a
  recorded, reproducible baseline identity to compare against once its
  benchmark scenario exists.

## Durable unresolved decisions

None material for the Go/CLI surface this spec designs. The three
process-level decisions feature.md leaves open (sprint-routing
re-ratification in REQ-F-006, the `rules.py`/editor-hook evidence threshold in
REQ-F-007, and the E40 scenario definition in REQ-F-008) are already recorded
as explicit, non-blocking deferrals in feature.md's own text — they are
WWGM-repository and E40-epic decisions to be made when that work executes, not
open architecture or requirements gaps in this Shark-repository feature. No
Q### is warranted: feature.md's rationale for deferring each one already
exists in prose (see feature.md "Requirements" 6–8 and "Out of scope"), and
creating a Q### here would duplicate a decision this spec has no authority or
new information to resolve.

*Last Updated*: 2026-09-04
