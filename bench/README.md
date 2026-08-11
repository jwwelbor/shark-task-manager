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

### Two-root run context

Each run uses two separate roots. Shark state, the scratch database,
transcripts, and generated planning documents belong to the scratch project.
The agent edits and tests code in the fixture checkout passed to
`shark run --workdir`.

Before dispatch, `run-one.sh` mirrors the scratch project's generated
`docs/plan/` tree into the fixture checkout. This lets the workflow prompt's
relative entity `file_path` resolve where the agent works. The mirrored files
are harness context, not fixture code or oracle inputs. The driver still
injects held-back F2P tests only after the run, after LOC and quality
measurements complete.

The bundled sibling canary is the default. Start a run without setting
`CANARY_BIN`:

```bash
bench/scripts/run-one.sh --item validate-sku-max-length --variant default --rep 1 --timeout 900 --out /tmp/shark-bench-runs
```

Set `CANARY_BIN` only to use a test or operator-specific replacement. Use
`--skip-canary` only when you intentionally bypass the preflight.

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
| `manifest.fixture_base_sha` | string | The fixture-repo commit this run's checkout was built from (`corpus.yaml`'s `fixture.base_sha`, resolved by `run-one.sh` during provisioning). G7 reproducibility (architecture.md#metric-collection-and-artifact-schema, uat-plan.md UAT-07: "pinned fixture SHA ... The report regenerates from the artifact directory alone, with no state outside it"). |
| `manifest.corpus_schema_version` | string | `corpus.yaml`'s own top-level `schema_version` at the time this item was read. |
| `manifest.p2p_set` | string | The item's own `p2p_set` name (`corpus.yaml`), resolved alongside `fixture_base_sha` in the same read. |
| `manifest.variant_bundle_sha256` | string | Content hash (sha256, sorted-by-path) over the workflow bundle `run-one.sh` installed for this run — pins which variant content produced the run without requiring `shark-data/` itself to survive as external state (ADR-002). |
| `manifest.shark_version` | string | The resolved `$SHARK_BIN --version` output, captured inside the scratch project (never the live repo, REQ-N-002/AC-11). |
| `manifest.shark_binary_sha256` | string | sha256 of the resolved `$SHARK_BIN` file itself, captured before provisioning (filesystem-only, no subprocess cwd to police). Under the harness's own PATH-stub test seam this pairs with a stub `shark_version`, not the real binary's — the pairing only describes the same binary on a real (non-stubbed) run. |
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

## Baseline aggregation, noise band, and replay

`bench/scripts/aggregate-runs.sh --root <artifact_root> [--variant <id>]
[--reps <n>] [--items <id[,id...]>]` (T-E40-F03-003/004) is a pure function of an artifact root: it reads only
the `record.jsonl` files matched by the pinned glob
`"$root"/*/*/rep-*/record.jsonl` (never `find` — a bare `*` never matches
the dot-prefixed `.incomplete/` quarantine root `run-batch.sh` writes) and
prints one `aggregate.json` document to stdout. No batch log, scratch
project, database, or network access (REQ-F-007, ADR-002).

### Record classification

| Classification | Rule |
|---|---|
| `complete` | Every one of `oracle`/`quality`/`loc` — the three post-run "observational" families — is present. |
| `explained_absence` | One or more of `oracle`/`quality`/`loc` is absent AND the record itself explains it: `outcome == "timeout"`; or `quality.toolchain_guard != "pass"`; or `errors[]` carries a `postrun_check_aborted` entry. |
| `anomaly` | One or more of `oracle`/`quality`/`loc` is absent with NO explanation on the record (the F-4 case, TD-079). Any `anomaly` makes the aggregator exit non-zero. |

Family presence — for classification, for `inventory[].families_present`,
and for every metric below — is read from the family block itself, **never**
from `sources`; a `sources.<family>` entry can be absent while the family
block it describes is present, and vice versa (REQ-F-007 2nd sentence,
TD-076's consumer-side consequence).

### Metric registry and exclusion reasons

Every aggregated metric is declared once with an id, a class (A/B/C, see
below), the I-02 field(s) it reads, and which `manifest.item_type` values
it applies to (`oracle_repro_confirmed` is bug-only; every other metric
applies to both `task` and `bug`). For each `(item_id, metric)` pair over
its contributing reps, every OTHER rep that did **not** contribute is
still named in that metric's `excluded[]` with one of these closed
reasons:

| Reason | Meaning |
|---|---|
| `outcome_timeout` | `outcome == "timeout"`: excluded from EVERY registry metric applicable to the item's type, whether or not the record could structurally have carried that metric's value (a timeout's `timing.harness_wall_ns` is the cap, not a measurement). |
| `toolchain_guard_abort` | The record is `explained_absence` via `quality.toolchain_guard != "pass"`, and this metric's own source field is individually absent — true even for a metric that lives inside the still-present `quality` family (e.g. `quality.fmt_clean`), since the guard aborts every other `quality` sub-field too. |
| `postrun_aborted` | Same as above, explained via an `errors[].kind == "postrun_check_aborted"` entry instead of the guard. |
| `unexplained_absence` | The record is classified `anomaly`: EVERY `oracle`/`quality`/`loc` metric is excluded with this reason, regardless of whether this particular metric's field happens to be present on the record. |
| `gate_not_executed` | This metric's source field EXISTS on the record but holds JSON `null` — `quality.fmt_clean`/`.vet_ok`/`.tests_pass` are documented above as `bool \| null`, "null means the gate could not be executed". A `null` is excluded, never coerced into a boolean/numeric measurement. |
| `partial_usage` | A Class C sum over `stages[].usage.*` (or a `step.<status>.tokens_input`/`.tokens_output`/`.cost_usd` metric) where at least one contributing `spawn_agent` stage exists but is missing `usage` or the specific sub-field — never summed over a subset. |
| `step_not_reached` | A `step.<status>.*` metric where this run's `stages[]` contains no entry at all with that status — distinct from a true zero (a step reached only via a non-agent, e.g. `advance_status`, stage genuinely cost nothing). |
| `family_absent` | Catch-all: the metric's source field, or its whole containing family/block (e.g. a wholly-absent `rejections` block), is simply missing, and none of the above more specific reasons applies. |
| `missing_run` | This `(item, rep)` pair has no `record.jsonl` at all under the expected rep matrix (`--reps`, or — absent that — the union of rep numbers observed anywhere in the root). Never silently reflected as a smaller `n` with an empty `excluded[]`; the item is also named in `flags.reduced_reps[]`. A declared item (`--items <id[,id...]>`) with NO record for ANY rep gets a `missing_run` entry for every expected rep on every applicable metric the same way, is named in `flags.missing_items[]` instead of `flags.reduced_reps[]`, and — unlike a partial rep hole — withholds `baseline_id` entirely (the declared item universe isn't fully covered). An observed item outside the declared `--items` set is left alone (still contributes normally) but is named in `flags.unexpected_items[]`. `--items` is authoritative in both directions the same way `--reps`/`expected_rep_set` is, and, like `--reps`, is never resolved from `batch-log.jsonl` or `corpus.yaml` (REQ-F-007) — purely the caller-supplied argument. |
| `unexpected_rep` | This record's `manifest.rep` is OUTSIDE the expected rep matrix (`--reps`, or — absent that — the union of rep numbers observed anywhere in the root) — e.g. `--reps 2` against a root that actually holds reps 1, 2, and 3. Never silently folded into the band while `provenance.reps`/`baseline_id` still name the smaller declared count; the item is also named in `flags.unexpected_reps[]`. `expected_rep_set` is authoritative in both directions — a rep below it is `missing_run`, a rep above it is `unexpected_rep`. |
| `invalid_value_type` | This metric's source field is present and non-null but is not the type its class requires (a genuine `bool` for Class A; a genuine `int`/`float`, not `bool`, for Class B/C) — excluded rather than coerced (REQ-N-005), e.g. a JSON string `"false"` never satisfies `oracle.f2p_resolved`. |
| `out_of_domain` | This metric's source field holds the RIGHT type but an out-of-domain value: a negative Class B/C value (every registered Class B/C metric is a non-negative count/measurement), or a non-finite Class C value (`NaN`/`Infinity`/`-Infinity` — reachable via the JSON-decode path, since Python's `json` module accepts those as bare tokens by default). Excluded rather than silently clamped into range, which is what the Class B band formula `max(0, min − 1)` would otherwise do to a negative `min`, breaking AC-12's `accept_lo <= min` invariant by construction. Distinct from `invalid_value_type`: that reason fires when the field's Python type itself is wrong; `out_of_domain` fires when the type is correct but the value falls outside the metric's declared domain. |

`rejections.by_gate`'s own zero-vs-excluded rule is distinct from the
table above and is **not** an exclusion: an omitted gate key within a
**present** `by_gate` object counts as `0` for that gate (still counted in
`n`) — only a wholly **absent** `rejections` block excludes the record
from every `rejections_*` metric (reason `family_absent`, unless
`outcome_timeout` already applies). The gate key universe, and the
`step.<status>.*` status universe, are each the union of keys/statuses
observed across the item's own contributing records.

### Band and acceptance interval

For each `(item_id, variant_id, metric)` over its contributing reps:

| Class | Observed statistics | Acceptance interval |
|---|---|---|
| A (binary) | `n`, `true_count`, `rate` | `accept_set`: the set of boolean values observed across reps. |
| B (integer count) | `n`, `min`, `median`, `max`, `mean`, `spread_abs`, `spread_rel` | `[max(0, min − 1), max + 1]`, unconditionally — every registered Class B metric is a non-negative count. |
| C (continuous) | `n`, `min`, `median`, `max`, `mean`, `spread_abs`, `spread_rel` | `r = max − min`; `r_eff = r` when `r > 0`, else `0.10 × abs(median)` when `median != 0`, else exactly `0`; interval `[min − r_eff, max + r_eff]`, lower-clamped at `0`. |

`spread_rel = spread_abs / median`, or `null` when `median == 0` (the
report then states the absolute spread only). A metric identically zero
across every rep has `r_eff = 0` and an exact `[0, 0]` interval — the
documented rule, not a coincidence of the `0.10 × median` formula.
`median` uses Python's `statistics.median`, which averages the two middle
values at even `n` — an integer-valued metric (e.g. `wall_clock_ns` at
`n == 2`, after a timeout exclusion) can therefore publish a non-integer
`median`; this is expected, not a formatting defect, and the report and
replay comparisons both read it as a float.

A metric with fewer than two contributing reps (`n < 2`) publishes no
interval/`accept_set` and is flagged `insufficient_reps` instead
(REQ-F-016). A metric whose `spread_abs` **strictly** exceeds its `mean`
is flagged `unusable` (REQ-F-018 — `>`, not `>=`). A task whose
`oracle.f2p_resolved` is identical across every one of at least two
contributing reps (all `true` or all `false`) is flagged
`non_discriminative`, listed in `flags.non_discriminative_tasks[]` — a
single contributing rep is never enough to call a result non-discriminative.

The regression signal is `oracle.p2p_regressions_count` and nothing else
(ADR-F03-06) — the `p2p_regressions_count` metric's own statistics are the
only aggregate field that may be read as a regression indicator.
`quality.tests_pass` feeds only its own `quality_tests_pass` Class A
metric; no other aggregate field is derived from it, because the fixture
carries a deliberately permanently-failing regression probe (T-004) that
would otherwise read as a constant "finding".

### Aggregate document (`aggregate.json`)

Sorted keys, fixed list order, no timestamp (REQ-N-004).

| Block | Content |
|---|---|
| `schema_version` | Pinned; the aggregate's own version, distinct from I-02's. |
| `input_digest` | sha256 over the sorted `"<sha256>  <relpath>"` lines of every contributing `record.jsonl` (REQ-F-019) — **computed**, not echoed. Each line's sha256 is over the record file's raw bytes; `relpath` is relative to `--root` (never the absolute path), so the same artifact set aggregated from two different locations digests identically. Publishes even when provenance is non-uniform — it identifies the exact (possibly-invalid) input set. |
| `provenance` | `model_ids[]`, `fixture_base_sha`, `variant_bundle_sha256`, `corpus_schema_version`, `shark_version`, `reps`, `uniform` (bool), `divergences[]` when not, and `unresolved_fields[]` (R2-F-4) when any of the five uniformity fields above was never given a usable value by ANY contributing record — each entry is `{field, reason}` with `reason` one of `all-exempt` (every contributing record's absence was individually explained, e.g. `model_ids` absent on an all-`timeout` batch) or `never-present` (defensive fallback for a shape not reachable under today's rules, kept so a future field/rule combination still gets a named reason instead of a silent drop). |
| `inventory` | Per `run_key`: `classification`, `outcome`, and the families present. |
| `outcomes` | Counts per outcome value, `timeout_rate`, `anomaly_count`. |
| `anomalies[]` | `run_key` + missing families, for the F-4 bucket. |
| `tasks[]` | Present only when `provenance.uniform` is `true`. Per item: `item_id`, `item_type`, `metrics{}` (statistics + interval/`accept_set` + `excluded[]` + `insufficient_reps`/`unusable` flags), `non_discriminative`. |
| `corpus` | Present only when `provenance.uniform` is `true`. Per metric: the `min`/`median`/`max` of the per-task `spread_rel` (nulls skipped); for `oracle_f2p_resolved`, additionally a corpus pass rate and a rep-slice band (the min/median/max of the per-rep-index corpus pass rates — a descriptive, non-paired rollup; the operative band for any comparison is the per-task one). |
| `flags` | Present only when `provenance.uniform` is `true`. `unusable_metrics[]` and `insufficient_reps[]` are `{item_id, metric}` lists; `non_discriminative_tasks[]` is a list of item ids; `reduced_reps[]` is a list of `{item_id, contributing_reps, published_reps}` naming every item whose contributing rep count fell below the published `provenance.reps` (REQ-N-005 — a band is never published over a silently reduced rep set); `unexpected_reps[]` is a list of `{item_id, unexpected_reps, published_reps}` naming every item that contributed a rep OUTSIDE the published `provenance.reps` matrix (R2-F-7 — the mirror-image mismatch: a rep count larger, not smaller, than declared). `missing_items[]` (NEW-2, present — possibly empty — whether or not `--items` was supplied) is a sorted list of item id **strings**: every declared item with zero records at all. `unexpected_items[]` (same shape, same always-present rule) is a sorted list of item id strings: every observed item outside the declared `--items` set. |
| `baseline_id` | Present only when `provenance.uniform` is `true`, both a single `variant_id` and `provenance.fixture_base_sha` are available, AND `flags.missing_items[]` is empty (NEW-2 — a declared item with zero records means the corpus isn't fully covered, so no confident baseline is stamped): `<variant_id>-<fixture_base_sha[:12]>-r<reps>` — 12 hex chars of the SHA, literal `r` + the integer rep count. |

**Non-uniform provenance (REQ-F-011).** `tasks[]`, `corpus`, `flags`, and
`baseline_id` are omitted entirely — never published "as if the batch
were valid" (AC-11) — while `schema_version`/`input_digest`/`provenance`/
`inventory`/`outcomes`/`anomalies` still print, since those are
independent of provenance validity and a non-uniform batch report still
needs to name its own (invalid) input set precisely. One `divergences[]`
shape carries its own named reason, `unpinned_field` (R2-F-11): every
NON-exempt contributing record agreeing on an absent value for one of the
five uniformity fields (all of them contribute an explicit JSON `null`,
never a real value) is unpinned agreement, not verified agreement, and is
recorded as `{field, reason: "unpinned_field", values: [{value: null,
run_keys: [...]}]}` rather than silently folded into `provenance[field]
= null` the way a genuinely single agreed-on value would be. Its
consequence is identical to any other divergence: `provenance.uniform` is
`false`, and `tasks[]`/`corpus`/`flags`/`baseline_id` are all suppressed
for the batch. A root whose records
span more than one `manifest.variant_id` with no `--variant` filter to
disambiguate is a separate, harder failure: nothing is printed at all
(same class as a structurally invalid record) — silently blending two
variants' runs under one `item_id` would be a fabricated result
(REQ-N-005), and `manifest.variant_id` is not one of REQ-F-011's five
uniformity fields (a batch is expected to be single-variant per REQ-F-001).

### Replay (`bench/scripts/replay-manifest.sh`)

`bench/scripts/replay-manifest.sh --record <path/to/record.jsonl> --band
<aggregate.json> --out <replay_artifact_root> [--corpus <corpus.yaml>]
[--skip-canary]` (T-E40-F03-006/007) re-runs one stored run against the
published band, to check whether the result still reproduces (epic G7).

**REQ-N-007's two preconditions**, documented here and (where cheaply
assertable) enforced before any replay dispatch:

1. **Ledger retention.** `bench/corpus/ledgers/<sha>/` is never deleted
   for any SHA a published manifest references. Replay reads the base-SHA
   ledger indirectly through `run-one.sh`, but its own precondition check
   asserts file-level content, not just directory existence: it requires
   `bench/corpus/ledgers/<sha>/tests.json` and `.../lint.json` to each
   exist as files, naming whichever is missing — deleting either file (or
   pruning the directory) is caught before dispatch, rather than silently
   breaking a future replay of a run against that SHA.
2. **Corpus item immutability.** A corpus item's seed file and held-back
   F2P test files are treated as immutable for any SHA a published
   manifest references (Q005). An ordinary curator edit to a seed or F2P
   file — without a fixture-repo SHA bump — would silently change what a
   "reproduction" of an old run actually re-executes.

**Dispatch order, before any API spend (REQ-F-027):** replay checks three
preconditions, collecting and reporting every failing one (never stopping
at the first):

- `bench/corpus/ledgers/<manifest.fixture_base_sha>/` exists.
- The stored record's `manifest.item_id` resolves in `--corpus`'s
  `items:` list.
- The `--band` aggregate has an entry for `(item_id, variant_id)`: a
  `tasks[]` entry with a matching `item_id`, and the aggregate's own
  `baseline_id` — the only place `aggregate.json` exposes its variant —
  reconstructs to `<variant_id>-<fixture_base_sha[:12]>-r<reps>` from the
  **band's own** `provenance.fixture_base_sha`/`reps` (never the
  manifest's, so a drifted `fixture_base_sha` never also manufactures a
  false band-lookup failure).

Any failure here makes zero `run-one.sh` invocations.

**Synthetic replay corpus (ADR-F03-02).** Once every precondition holds,
replay builds a synthetic single-item `corpus.yaml` in a temp directory —
`bench/corpus/corpus.yaml` itself is never edited. It carries the source
corpus's `fixture:`/`p2p_sets:` blocks verbatim (`fixture.base_sha`
overwritten from the manifest — corpus.yaml's own INVARIANT requires it
match the item's own `fixture_base_sha` byte-for-byte), one `items:` entry
(the resolved source item, with `fixture_base_sha`/`p2p_set` overwritten
from the manifest and `seed_path`/`f2p.paths` absolutized against the
*source* corpus's own directory), and no `negative_items:` key at all.
This is dispatched via `run-one.sh`'s existing `--corpus` flag — `run-one.
sh` itself is never modified (ADR-F02-06's "invoke, never re-derive").

**`corpus_drift` (REQ-F-028).** Any divergence between the *live* corpus
item's `fixture_base_sha`/`p2p_set` or the live corpus's top-level
`schema_version`, and the manifest's pinned values, is recorded as a
`corpus_drift` entry in `verification.json`'s `reasons[]` — one entry per
diverging field, naming both the expected (manifest) and actual (live)
value. The dispatched synthetic corpus always carries the **manifest's**
values regardless of any detected drift; drift is recorded, not a
precondition failure.

Replay passes the manifest's own stored `--rep` value (and `--timeout`,
when `manifest.timeout_cap_s` is present) to `run-one.sh` against a
distinct `--out` root, so the fresh record's `manifest.run_key` is
byte-identical to the stored one and is the join key for the comparison
(REQ-F-025).

**Post-dispatch comparison and the three-valued verdict (REQ-F-029/030,
ADR-F03-05).** After a successful dispatch, the fresh record's identity
fields — `manifest.run_key`, `.fixture_base_sha`, `.corpus_schema_version`,
`.p2p_set`, `.variant_bundle_sha256`, `.model_ids`, and `.shark_version` —
are ALL compared against the stored record's (the same set that drives the
`verification.json` `stored`/`replayed` display blocks — a field is never
displayed without also being enforced). A mismatch on
`variant_bundle_sha256`/`model_ids`/`run_key` is recorded with its own
named `reasons[]` entry (`variant_bundle_drift`, `model_version_drift`,
`run_key_mismatch` respectively); every other identity field gets a
generic `identity_mismatch` entry naming the field. Any of these — naming
both the expected (stored) and actual (fresh) value in full — yields the
verdict `invalid`, **never** `fail`: the inputs were not reproduced, so a
metric comparison would be meaningless. When multiple fields drift
simultaneously, every one is present as its own reasons[] entry, each with
its own expected/actual pair — none ever subsumes another. `corpus_drift`
(above) shares this same `reasons[]` list and vocabulary, so any
combination of drift — corpus, identity, or both — still yields one
`invalid` verdict naming every contributing reason. A freshly dispatched
replay whose own `aggregate-runs.sh` invocation exits non-zero (an
`anomaly` classification, REQ-F-009 — the aggregator itself refused to
certify the replayed record as a comparable datum) is likewise recorded
as its own `reasons[]` entry (`aggregator_anomaly`, carrying a `detail`
string instead of an expected/actual pair) and forces the same `invalid`
verdict, never a metric comparison run over data the aggregator would not
certify.

Only when every identity field matches does the per-metric comparison
run: the freshly dispatched single record is aggregated on its own (by
invoking `aggregate-runs.sh`, never re-derived) to compute its own
per-metric values, and each metric published in the `--band`'s matching
`tasks[]` entry is compared against it — every metric the band publishes
is accounted for in `metrics[]`, never silently dropped (REQ-N-005): a
metric is `pass`/`fail` when both sides can supply a value, `not_comparable`
(reason `metric_not_comparable`, forcing verdict `invalid`) when the band
published an interval the fresh record has no value for, and
`not_comparable` (reason `band_no_interval`, traced but **not** by itself
forcing `invalid`) when the band itself published no interval for that
metric at all (e.g. `insufficient_reps`, AC-13) — the band made no claim,
so a replay cannot have violated one. The verdict is `pass` when every
metric that DID get a `pass`/`fail` score falls inside its published
interval (or `accept_set`, for a Class A metric), or `fail` when at least
one does not — `fail` is metric-scoped in `verification.json`'s
`metrics[]` table (every comparable metric is listed with its own
`replayed_value` and interval/`accept_set` and its own `pass`/`fail`),
never a single bare bit. If NO metric ends up with a `pass`/`fail` score
(an empty `metrics[]`, or every entry `band_no_interval`), the verdict is
`invalid` (reason `no_comparable_metrics`) — never a vacuous pass.
