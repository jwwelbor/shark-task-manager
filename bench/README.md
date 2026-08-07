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

> The run directory layout and the full I-02 `record.jsonl` field reference
> are T-E40-F02-007's scope (the envelope parser task) and land in this
> section when that task completes. What follows is only the Q003 closure
> (REQ-F-021, AC-17): the confirmed `claude --output-format json` envelope
> field names the parser depends on.

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

**Structural surprise worth flagging for T-E40-F02-007**: the captured
envelope has **no top-level `model` field at all** — not null, not empty,
absent. REQ-F-021's named fallback ("if `modelUsage` is absent, exact model
IDs come from the envelope's top-level `model` field") assumes that field
exists whenever `modelUsage` doesn't; this capture cannot confirm that,
because `modelUsage` was present here. The envelope parser must treat "both
`modelUsage` and `model` absent" as its own named parse error
(`errors[].kind: "envelope_parse_error"`, REQ-F-012/AC-05) rather than
assuming the fallback field is always reachable.

Every fixture under `bench/scripts/testdata/run/` that asserts envelope
shape (`clean-completed/run/transcripts/*.log`, and the three
`missing-envelope-field/{modelUsage,num_turns,duration_api_ms}/` variants)
uses these confirmed names with hand-authored synthetic values — never the
raw captured transcript, which may carry real API content.

**Coverage gap, inherited by T-E40-F02-007**: no fixture here exercises the
`model`-fallback branch (including `missing-envelope-field/modelUsage/`,
which omits `modelUsage` but does **not** add a top-level `model` field,
matching what was actually observed). This is deliberate, not an oversight
— no captured envelope has ever carried a top-level `model` field, so
fabricating one to drive that branch would mean testing against an
unobserved shape, exactly what Q003's capture-before-fixture rule exists to
prevent. If T-E40-F02-007 implements the fallback branch, its own fixture
is necessarily authored on an unobserved shape and must say so in its own
comment/commit.
