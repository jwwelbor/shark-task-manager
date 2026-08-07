# shark-bench corpus

`bench/` is the E40-F01 benchmark corpus: a self-contained fixture repo
(`bench/fixture-repo`, a git submodule) at a pinned base commit, a
machine-readable manifest of admitted and negative items
(`bench/corpus/corpus.yaml`), and the scripts that admit, ledger, and diff
against that fixture (`bench/scripts/`). E40-F02 (the run driver) reads this
document instead of re-deriving the manifest shape, the `p2p_set`
resolution rule, or the diff method from `corpus.yaml` or the scripts
directly (I-01).

## Manifest schema (REQ-F-002)

`bench/corpus/corpus.yaml`, schema-versioned via its top-level
`schema_version` field (currently `"1.0"`), has four top-level sections:

- **`fixture`** — `submodule_path` (the fixture's location relative to the
  repo root), `base_sha` (the single commit every item and ledger is
  evaluated against), and a `toolchain` block (`go_version`,
  `golangci_lint_version`, `goos`, `goarch`, `golangci_config_sha256`) —
  the same axes `build-ledgers.sh` records on every ledger and
  `diff-ledgers.sh --toolchain-guard` compares against.
- **`p2p_sets`** — a map from a `p2p_set` identifier to `packages` (Go
  package patterns), `run_selector` (an optional `-run` regex), and
  `exclude_tests` (fully-qualified test identities to subtract). See
  "`p2p_set` resolution rule" below.
- **`items`** — the admitted corpus (REQ-F-006: ≥10). Each entry carries:
  `id`, `type` (`task` or `bug`), `prompt_path`, `seed_path`, an `f2p`
  block (`paths`: held-back F2P source file(s); `test_names`: fully
  qualified `<package import path>::<test name>` identities — a bug
  item's `f2p.test_names` doubles as its repro-test set), `p2p_set` (an
  identifier resolved against the `p2p_sets` map), `reference_patch_path`,
  and `fixture_base_sha` (which MUST equal the top-level `fixture.base_sha`
  — `tests/contracts/e40_i01_corpus_contract_test.go#TC-001` asserts this
  byte-identity on every `make test` run). `seed_path` points at a
  standalone YAML file (REQ-F-002's "entity seed spec"), itself carrying
  `type` (MUST match the item's own `type`), `title`, `description`, and,
  for `type: bug` only, `severity` — TC-001 asserts all four by reading
  the file, not just its existence.
- **`negative_items`** — the committed rejection-path corpus (REQ-F-007:
  exactly one item per admission-gate branch (a), (b), and (d)), same
  field shape as `items`, stored apart and never emitted into any run
  list `bench/scripts/admit.sh` or E40-F02 consumes. Branches (c) and (e)
  need no committed entry — they are built transiently at test time (see
  the corpus.yaml comments above `negative_items:` and
  `bench/scripts/tests/tc005_admit_rejection_branches_test.sh`).

All paths inside an item entry are relative to `bench/corpus/`. The
contract validator (`tests/contracts/e40_i01_corpus_contract_test.go#TC-001`)
is the executable definition of this schema — it asserts every field's
presence and type, that every named prompt/seed/patch/F2P path exists on
disk, that every item's `p2p_set` resolves in `p2p_sets`, and that
`schema_version` is a version the validator supports. Treat this section as
a reader's map to that test, not a substitute for it.

## `p2p_set` resolution rule (REQ-F-003)

An item names exactly one `p2p_set` identifier, which MUST resolve to a
named entry in `p2p_sets`. Resolving a `p2p_set` for a given item means:

1. Look up the named entry's `packages` (and `run_selector`, if any) — the
   candidate universe of tests.
2. Subtract that entry's own `exclude_tests` (deliberately, permanently
   broken tests unrelated to any single item, carried by the set itself —
   see `default`'s exclusion of
   `pkg/inventory::TestStock_PermanentlyFailingRegressionProbe` in
   `corpus.yaml`).
3. Subtract the item's own `f2p.test_names` — always, for every item,
   regardless of which set it names, so no set needs to list an item's F2P
   tests itself.

The result is that item's P2P (pass-to-pass) set: the tests that must stay
green both before and after a candidate patch is applied. `default` is the
fixture's whole suite (`./...`) minus its own permanent-failure exclusion;
most items name it. An item may name a different set when its rejection
depends on the set definition itself rather than on any F2P injection —
`corpus.yaml`'s `full_suite_no_exclusions` entry (named only by the
`cart-add-item-rejects-negative-quantity` negative item) is the one case in
this corpus: it is identical to `default` except it does **not** exclude
the permanent-failure probe, so that candidate's own P2P set is red at base
for a reason intrinsic to the set, not to its F2P files.

## Ledger diff method (REQ-F-011) and toolchain guard (REQ-F-010)

`bench/scripts/build-ledgers.sh <checkout_dir> <output_dir>` (T-E40-F01-008)
runs the fixture's own test suite and lint config against a checkout and
writes two ledgers: `tests.json` and `lint.json`. Each records a
`toolchain` block (Go version, `golangci-lint` version, `GOOS`/`GOARCH`, a
content hash of `.golangci.yml`) but never compares it against anything —
`build-ledgers.sh` only records the toolchain that produced the ledger.

`bench/scripts/diff-ledgers.sh` is the single tool that implements the
diff method described here and the toolchain comparison — E40-F02 invokes
it directly instead of re-deriving either from prose.

### `--kind=lint|test` — applying the diff

```
bench/scripts/diff-ledgers.sh --kind=lint --base=<base-ledger.json> --post=<post-ledger.json>
bench/scripts/diff-ledgers.sh --kind=test --base=<base-ledger.json> --post=<post-ledger.json>
```

Both forms take two ledger-shaped JSON files (the same shape
`build-ledgers.sh` writes) — no fixture checkout is needed, so the method
is drivable from hand-authored synthetic ledgers as well as real ones.

**`--kind=lint` (lint ledger identity, ADR-F01-03):** a lint issue's
identity is the tuple `(from_linter, path, text)` — line and column are
excluded, because an agent's patch shifts line numbers in a touched file
and including position would report every pre-existing issue as new.
`lint_new_issues` is the **multiset** difference, post-run minus base,
**floored at zero per identity**: for every identity, the reported new
count is `max(post_count - base_count, 0)`. A duplicated identity in the
base ledger is tracked at its own depth (e.g. 2), not collapsed to a
single known/unknown flag — a set-based (not multiset-based) diff would
silently under-report a genuinely re-duplicated issue.

**`--kind=test` (test ledger transitions):** an entry's identity is
`<package>::<test>` (subtests are distinct, slash-named identities).
`p2p_regressions` is every identity that was `pass` at base and is `fail`
at post — that is the *only* transition classified as a regression. An
identity present at base and absent at post is reported separately as
**removed**, never as a regression and never silently dropped. A base
`fail`/`skip` entry that is still or newly `fail` at post produces no
entry at all — the base was already failing (or skipped), so post-`fail`
is not a new regression.

Both forms print a single JSON object to stdout with the entries in a
fixed sort order (so two independent runs over the same input are
byte-comparable) plus the applicable counts. This script's own exit code
reflects only whether the diff computation itself succeeded — it never
encodes "new issues found" or "regression found" as failure; interpreting
those counts is the consumer's decision.

### `--toolchain-guard` — the fail-closed comparison

```
bench/scripts/diff-ledgers.sh --toolchain-guard --base=<ledger.json>
```

Reads `<ledger.json>`'s recorded `toolchain` block and compares it against
the live environment (the same `go env GOVERSION`/`GOOS`/`GOARCH` and
`golangci-lint version` invocations `build-ledgers.sh` uses to record
them, plus a content hash of the fixture's `.golangci.yml`). If any field
disagrees, it exits non-zero, naming every mismatched axis on stderr,
**before** either diff mode above would run. `diff-ledgers.sh` is the
**single named owner** of this comparison — `build-ledgers.sh` never
performs it.

A post-run check executing under a different toolchain than the one that
produced a ledger cannot meaningfully diff against it (a version bump in
`golangci-lint` or the Go toolchain can change which issues are reported
independent of any code change) — the guard exists so that failure is
loud and specific rather than a silent, misleading diff.

## Test tiers and the Tier 2 curator command sequence

Verification of this corpus spans three tiers (test-plan.md), because
`spec.md` (ADR-F01-05) deliberately keeps the `bench/fixture-repo` submodule
out of CI:

| Tier | Runs | Needs submodule? | Where |
|---|---|---|---|
| Tier 1 | `make test` (CI + every dev machine) | No — reads only committed corpus artifacts | `tests/contracts/e40_i01_corpus_contract_test.go` |
| Tier 1b | Curator, manually or via the bench self-test wrapper | No — synthetic JSON only, no fixture execution | `bench/scripts/diff-ledgers.sh` invoked with hand-authored fixtures under `bench/scripts/testdata/` |
| Tier 2 | Curator, at corpus build time and on every corpus/fixture change | Yes — `git submodule update --init` first | `bench/scripts/{checkout-fixture,build-ledgers,admit,diff-ledgers,verify-clean-checkout}.sh` against a real checkout |

Tier 2 is what the feature's Success Metric — "the admission gate re-run on
a clean checkout reproduces identical results" — exercises. It is **not**
gated by root `make test`, since CI's checkout does not initialize
submodules, so it is the curator's own responsibility to run it, on the
following exact sequence:

```bash
# 1. Initialize the fixture submodule (once per clone/checkout).
git submodule update --init

# 2. Run the admission gate over the full corpus (REQ-F-005/006/007).
#    Evaluates every entry in corpus.yaml's items: (never negative_items:)
#    against a fresh checkout-fixture.sh checkout of fixture.base_sha.
bench/scripts/admit.sh bench/corpus/corpus.yaml

# 3. Build the base-SHA test and lint ledgers (REQ-F-008/009).
bench/scripts/checkout-fixture.sh <fixture.base_sha> /tmp/shark-bench-ledger-checkout
bench/scripts/build-ledgers.sh /tmp/shark-bench-ledger-checkout /tmp/shark-bench-ledger-out

# 4. Confirm the ledgers agree with the live toolchain before trusting any
#    diff against them (REQ-F-010) -- run this before either diff mode.
bench/scripts/diff-ledgers.sh --toolchain-guard --base=/tmp/shark-bench-ledger-out/tests.json

# 5. Prove held-back F2P tests never leak into a checkout (REQ-F-004,
#    AC-001). This is also invoked automatically as checkout-fixture.sh's
#    own exit path (step 2's and step 3's checkouts were already verified
#    clean as a side effect), so a standalone run here is a direct check
#    against any checkout obtained another way.
bench/scripts/verify-clean-checkout.sh /tmp/shark-bench-ledger-checkout bench/corpus/corpus.yaml

# 6. Run the full bench self-test suite (Tier 1b + Tier 2 scripts driven
#    together, TC-003 through TC-011 and TC-013).
bench/scripts/tests/run-all.sh
```

`admit.sh`'s own exit code is the admission gate's summary assertion (0
only if every item in `items:` was admitted, at least 10 were admitted, and
at least one admitted `bug` item confirms its repro oracle); see
`bench/scripts/admit.sh`'s header comment for the full five-check
breakdown and its `--item`/`--patch` flags for evaluating a single
candidate (including the transient rejection-branch (c)/(e) candidates).

A curator re-runs this sequence whenever `bench/corpus/corpus.yaml` or
`bench/fixture-repo`'s pinned commit changes, to confirm REQ-F-012 (a clean
checkout at an unchanged base SHA and toolchain reproduces byte-identical
verdicts and identity sets) still holds.

## Run driver and artifact schema

`bench/scripts/run-one.sh` (the driver) provisions a scratch shark project,
seeds one corpus item, dispatches `shark run` under a timeout cap, and hands
off to `bench/scripts/collect-run.sh` (the collector, ADR-F02-01): a **pure
function** of a completed run directory that emits one I-02 JSONL record to
stdout. This section documents the run directory layout the driver writes
and the collector reads, the I-02 record's field reference, and (below) the
Q003 closure: the confirmed `claude --output-format json` envelope field
names the collector's `stages[].usage` extraction depends on.

### Run directory layout

```
<artifact_root>/<item_id>/<variant_id>/rep-<n>/
├── record.jsonl          # the I-02 record (exactly one line)
├── run/
│   ├── stdout.json       # shark run --json stdout, verbatim
│   ├── stderr.ndjson     # liveness stream, verbatim
│   ├── run.log           # copied from <scratch>/.shark/runs/<run_id>/run.log
│   ├── exit_status       # process exit status + whether the cap fired
│   └── transcripts/      # copied from <scratch>/.shark/runs/<run_id>/*.log
├── post/
│   ├── tests.json  lint.json          # post-run ledgers (build-ledgers.sh)
│   ├── test-diff.json  lint-diff.json # diff-ledgers.sh stdout, verbatim
│   └── f2p.json  fmt.txt  vet.txt  test.txt  numstat.txt
└── meta.json             # driver-captured manifest inputs
```

`collect-run.sh` reads only this directory (plus the scratch DB and fixture
checkout paths named in `meta.json`), so every collector behaviour is
testable from committed fixtures with no API spend. `post/` is entirely
optional: a run directory with no `post/toolchain-guard.json` at all (e.g. a
timed-out run) simply gets no `oracle`/`quality`/`loc` blocks on its record,
never a fabricated one. See `collect-run.sh`'s own header comment for the
exact machine-readable shape of each `post/*.json` file — that contract is
this task's own pinned extension of the layout above, since neither
`spec.md` nor `test-plan.md` pins the post-run outputs' exact shape beyond
the raw filenames.

### I-02 record schema field reference

One record per run, one line, sorted keys (REQ-N-004). Every field below is
**conditionally present**: a field whose source data never arrived is
absent from the record, never a zero, empty string, or null-filled
placeholder (REQ-N-005) — the "Notes" column below states the condition.

| Field | Type | Notes |
|---|---|---|
| `schema_version` | string | Pinned (`"1.0"`). |
| `manifest.item_id` / `.item_type` / `.variant_id` / `.rep` / `.timeout_cap_s` / `.seeded_keys` | — | Copied from `meta.json` (driver args + corpus.yaml + `shark create ... --json` responses, AC-16). |
| `manifest.run_key` | string | `<item_id>::<variant_id>::rep<rep>` (REQ-F-018). |
| `manifest.run_id` / `.run_id_source` | string / string | `"liveness_stream"` or `"fallback_newest_dir"` (REQ-F-008, ADR-F02-02). |
| `manifest.model_ids` | array of string | Deduplicated, sorted, canonical model IDs across every `spawn_agent` stage whose envelope resolved `modelUsage`. **Absent** when no stage resolved one (e.g. every `spawn_agent` stage's envelope was missing `modelUsage`). |
| `manifest.model_id_source` | string | `"modelUsage"` whenever `manifest.model_ids` is present — the shipped parser never falls back to a top-level `model` field (see "Confirmed envelope field names" below). Absent alongside `manifest.model_ids` when absent. |
| `outcome` | string | One of `completed` / `paused` / `failed` / `already_terminal` / `no_action` (the five `RunResult` values, copied unchanged) plus the harness-assigned `timeout`. |
| `timeout_detail` | object | Present **only** when `outcome == "timeout"`, otherwise absent (never a zero-valued object): `{stage_index, status, action, agent_type, provider, source}`, `source` is `"liveness_stream"` or `"scratch_db_status_fallback"`. |
| `runresult.final_status` / `.stages_completed` / `.total_duration_ns` / `.error` / `.question_block` | — | Copied verbatim from `RunResult`. The whole `runresult` block is genuinely **absent** (not null-filled) on the timeout path — no `RunResult` was ever delivered. |
| `stages[].index` / `.status` / `.action` / `.agent_type` / `.provider` / `.duration_ns` / `.exit_code` | — | Copied verbatim from `RunResult.Stages`, 1-based `index`. |
| `stages[].transcript_path` | string | Present only for a `spawn_agent` stage whose transcript resolved cleanly: count-correct, filename-component-correct, and readable (non-empty). |
| `stages[].usage` | object | Present only for a `spawn_agent` stage with a resolved `transcript_path`. Each sub-field is checked and recorded **independently** — see the table below; a field missing from the transcript's envelope is simply absent from `usage`, never zeroed, and the whole `usage` key itself is absent (not `{}`) if every sub-field failed. |
| `timing.harness_wall_ns` | integer | Driver-measured wall clock (`meta.json`), includes process spawn/teardown. |
| `rejections.by_gate` | object | Gate status → rejection count, inferred purely from `RunResult.Stages` re-entry (REQ-F-014). |
| `rejections.rework_loops` | integer | Sum of `by_gate`'s values. |
| `rejections.crosscheck.entity_history_backward_transitions` | integer | The same "re-entry into an already-seen value" measure applied to the scratch DB's `entity_history.to_status` (ADR-F02-09). |
| `rejections.crosscheck.work_session_outcomes` | integer | Count of `work_sessions` rows for the entity with `outcome = 'blocked'`. Supplementary — does **not** participate in the agree/disagree comparison. |
| `rejections.crosscheck.agrees` | bool | `entity_history_backward_transitions == rework_loops`. The whole `rejections.crosscheck` sub-object is present only when `meta.json` supplies `scratch_db_path` + `entity_key` + `item_type`. |
| `oracle.f2p_resolved` / `.repro_confirmed` / `.p2p_regressions[]` / `.p2p_regressions_count` / `.removed[]` / `.removed_count` | — | Copied from `post/f2p.json` / `post/test-diff.json`, byte-identical to `diff-ledgers.sh`'s own output fields — never recomputed (ADR-F02-06, AC-09). Absent entirely when the toolchain guard aborted or `post/` never ran. |
| `quality.fmt_clean` / `.vet_ok` / `.tests_pass` | bool \| null | `null` means the gate could not be executed — never a silent pass (REQ-F-016). |
| `quality.lint_new_issues[]` / `.lint_new_issues_count` | — | Copied from `post/lint-diff.json`, byte-identical to `diff-ledgers.sh`'s output. |
| `quality.toolchain_guard` | string | `"pass"`, or the mismatched-axis detail text when the guard aborted the whole post-run phase. |
| `loc.prod_added` / `.prod_deleted` / `.test_added` / `.test_deleted` / `.files_touched` | integer | From `post/numstat.txt`; a `_test.go` path's counts go to the `test_*` fields, every other path's to `prod_*`. |
| `errors[].kind` | string | Closed **seven-value** set: `envelope_parse_error`, `stage_join_error`, `transcript_missing`, `crosscheck_disagreement`, `crosscheck_resolution_error`, `postrun_check_aborted`, `usage_unavailable`. The last is reserved for a future codex-dispatched stage (Phase 1 dispatches Claude only and never emits it). |
| `errors[].detail` | string | Always present; names the missing field, the mismatched counts, or the mismatched axis. |
| `errors[].stage_index` / `.path` | integer / string | Present where applicable (a stage-scoped or transcript-scoped error); absent otherwise. |
| `sources.<family>` | string | Per-metric-family provenance, closed **five-value** set: `runresult` / `transcript` / `scratch_db` / `postrun` / `liveness` (REQ-N-007). The timeout-stage-attribution family's key is the literal `sources.stalled_stage` — `"liveness"` when resolved from the stream, `"scratch_db"` from the DB-status fallback. This is not "or an equivalent key" — `collect-run.sh` emits exactly `stalled_stage`. |

`stages[].usage`'s own sub-fields, each independently checked against the
transcript envelope (REQ-F-011/012):

| `usage.*` sub-field | Envelope source | Notes |
|---|---|---|
| `num_turns` | top-level `num_turns` | |
| `duration_api_ms` | top-level `duration_api_ms` | |
| `total_cost_usd` | top-level `total_cost_usd` | |
| `input_tokens` | `usage.input_tokens` | The envelope's own flat `usage` sub-object (snake_case) — distinct from `modelUsage`'s per-model camelCase fields. |
| `output_tokens` | `usage.output_tokens` | |
| `cache_read_input_tokens` | `usage.cache_read_input_tokens` | |
| `cache_creation_input_tokens` | `usage.cache_creation_input_tokens` | |
| `model_ids` | sorted `modelUsage` object keys | Also feeds `manifest.model_ids`/`.model_id_source` (deduplicated across every stage). |

A field missing or renamed in the transcript's envelope produces one
`envelope_parse_error` entry naming that exact field and the transcript
path, and that field alone is absent from `usage` — every other
successfully-parsed field is still recorded (AC-05: "the corresponding
metric is absent, not zero", not "the whole stage's usage is dropped").

### Confirmed claude CLI JSON envelope field names (Q003 closure, REQ-F-021)

Before this capture, the exact spelling and presence of `modelUsage`,
`num_turns`, and `duration_api_ms` in a real `claude --output-format json`
result envelope were unconfirmed in-repo (`spec.md` "Durable unresolved
decisions" Q003) — the only prior in-repo artifact
(`E27-F15-cross-session-usage-tracking`'s `testdata/claude-usage-result.json`)
is a 7-field hand-authored unit-test fixture that corroborates none of the
three.

**Capture provenance**: one real `bench/scripts/run-one.sh` invocation
against corpus item `validate-sku-max-length` (a small, real,
single-file task dispatched to a real `claude` CLI, not a stub), captured
2026-08-06, `claude` CLI version `2.1.223`, run id
`2aed0d9d-7db0-40b3-b756-e3a2404b42af`, transcript
`.shark/runs/2aed0d9d-7db0-40b3-b756-e3a2404b42af/2-development-anthropic.log`
(scratch project, not committed — never the raw captured file, per
test-plan.md's fixture-authoring rule).

| Field | Confirmed? | Shape observed |
|---|---|---|
| `modelUsage` | **Present**, exact spelling `modelUsage` (camelCase) | Object keyed by the canonical model ID string (e.g. `"claude-sonnet-5"`), not a flat field. Each value is a per-model usage object: `inputTokens`, `outputTokens`, `cacheReadInputTokens`, `cacheCreationInputTokens`, `webSearchRequests`, `costUSD`, `contextWindow`, `maxOutputTokens`, `canonicalModel`, `provider` (all camelCase, distinct from the flat `usage.*` sub-object's snake_case field names). |
| `num_turns` | **Present**, exact spelling `num_turns` (snake_case) | Top-level integer. |
| `duration_api_ms` | **Present**, exact spelling `duration_api_ms` (snake_case) | Top-level integer, milliseconds. |
| `usage` | Present (already partially corroborated by the E27-F15 fixture) | Top-level object, snake_case keys (`input_tokens`, `cache_creation_input_tokens`, `cache_read_input_tokens`, `output_tokens`, plus nested `server_tool_use`/`service_tier`/`cache_creation`/`inference_geo`/`iterations`/`speed`). |
| `total_cost_usd` | Present | Top-level float. |

**Model-ID source used**: `modelUsage` was present in the captured envelope,
so `manifest.model_id_source == "modelUsage"` for this capture — the exact
model ID is read from `modelUsage`'s own object key(s) (e.g.
`"claude-sonnet-5"`), not from a separate value field.

**Structural surprise, flagged for T-E40-F02-007** (now resolved, below):
the captured envelope has **no top-level `model` field at all** — not null,
not empty, absent. REQ-F-021's named fallback ("if `modelUsage` is absent,
exact model IDs come from the envelope's top-level `model` field") assumes
that field exists whenever `modelUsage` doesn't; this capture cannot
confirm that, because `modelUsage` was present here. The envelope parser
must treat "both `modelUsage` and `model` absent" as its own named parse
error (`errors[].kind: "envelope_parse_error"`, REQ-F-012/AC-05) rather than
assuming the fallback field is always reachable.

Every fixture under `bench/scripts/testdata/run/` that asserts envelope
shape (`clean-completed/run/transcripts/*.log`, and the three
`missing-envelope-field/{modelUsage,num_turns,duration_api_ms}/` variants)
uses these confirmed names with hand-authored synthetic values — never the
raw captured transcript, which may carry real API content.

**Coverage gap, flagged for T-E40-F02-007** (now resolved, below): no
fixture here exercises the `model`-fallback branch (including
`missing-envelope-field/modelUsage/`, which omits `modelUsage` but does
**not** add a top-level `model` field, matching what was actually
observed). This was deliberate, not an oversight — no captured envelope has
ever carried a top-level `model` field, so fabricating one to drive that
branch would mean testing against an unobserved shape, exactly what Q003's
capture-before-fixture rule exists to prevent.

**Resolved (T-E40-F02-007, closes Q003): no `model`-fallback branch is
implemented.** Given zero observed evidence of a top-level `model` field
across every real capture to date, the shipped parser does not implement
the fallback at all — a `modelUsage`-absent envelope is unconditionally its
own fail-loud `envelope_parse_error` naming `"modelUsage"`, and
`manifest.model_ids`/`manifest.model_id_source` are simply absent from the
record for that run (never a silent reach for the unconfirmed `model`
field, never a fabricated `model_id_source` value). `spec.md`'s REQ-F-021
sentence and its "Durable unresolved decisions" Q003 bullet are both
amended to match (E40-F02 decision note, 2026-08-07). If a future capture
ever observes a real `model` field, implementing the fallback then is a
new, evidence-backed change — not a retroactive justification of this one.
`missing-envelope-field/modelUsage/`'s existing fixture already exercises
this exact path: `manifest.model_id_source` is asserted absent, not `"model"`
(TC-015c).

### Test tiers for the run driver and collector

Mirrors the corpus's own three-tier split above, applied to
`run-one.sh`/`collect-run.sh`/`canary-runsurface.sh` (test-plan.md "Test
tiers"):

| Tier | Runs | Needs a scratch project/dispatch? | Where |
|---|---|---|---|
| Tier 1 | `make test` (CI + every dev machine) | No | `tests/contracts/e40_i02_artifact_contract_test.go` |
| Tier 1b | Curator, manually or via `bench/scripts/tests/run-all.sh` | No — synthetic fixtures + PATH-stubbed `shark`/`claude` only | `tc014_run_one_smoke_test.sh`, `tc015_collect_run_record_test.sh` |
| Tier 2 | Curator, via `bench/scripts/tests/run-all.sh`, before every corpus/harness release | **Yes** — provisions its own throwaway scratch project and dispatches a real `shark run` (PATH-stubbed `claude` only) | `tc016_canary_runsurface_test.sh` |

`tc016_canary_runsurface_test.sh`'s precondition (deferred from
T-E40-F02-005, folded in here): unlike F01's Tier 2 scripts, it does **not**
need the `bench/fixture-repo` submodule — `canary-runsurface.sh` provisions
its own scratch project via `scripts/shark-scratch-env.sh` (the same real
provisioning path `run-one.sh` uses, not a second implementation of it) and
tears it down on exit regardless of outcome. It is registered in
`bench/scripts/tests/run-all.sh` alongside the Tier 1b scripts but is not a
`make test` resident (ADR-F02-10) — it is a harness preflight that aborts a
batch on drift, not a CI gate. Known flake (T-E40-F02-005): do not run
`bench/scripts/tests/run-all.sh` concurrently with `make test` (`tc014`'s
sub-case (g) has resource contention with a concurrently-running Go test
binary) — run them sequentially.
