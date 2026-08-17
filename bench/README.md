# shark-bench corpus

## F08 lifecycle runner contract tier

The canonical multi-entity lifecycle runner is `bench/scripts/run-lifecycle.sh`.
Contract and dry-run modes keep provider execution out of the verification path,
while `lifecycle-worker-adapter.sh`, `lifecycle-prelude.sh`, and
`review-capture.sh` preserve prompt provenance, authorized prelude decisions,
stage-policy identity, and review findings as bounded artifacts. The complete
F08 contract tier is registered in `bench/scripts/tests/run-all.sh` as TC-060
through TC-065; run that wrapper to verify the lifecycle seam locally.

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

## I-04 lifecycle scenario corpus and adapter contract (E40-F05)

`bench/scenarios/` is I-04: a versioned lifecycle scenario corpus independent
of `bench/corpus/corpus.yaml` (REQ-F-001) covering the four lifecycle
families (`feature`, `bug`, `change_card`, `tech_debt`) on a second,
controlled Python fixture (`bench/fixture-py`, a git submodule, same
convention as `bench/fixture-repo`). `bench/fixture-repo` and its I-01
tooling above are unmodified and remain registered as the `go` compatibility
adapter. E40-F06, E40-F07, and E40-F08 read this section instead of
re-deriving the package shape, the adapter contract, or the Tier 2 sequence
from `bench/scenarios/**` or the scripts directly — the same role the
"Manifest schema" section above plays for I-01.

### I-04 scenario package schema

`bench/scenarios/scenarios.yaml`, schema-versioned via its top-level
`schema_version` (currently `"1.0"`), registers three things: `fixtures`
(`fixture_id` → `submodule_path`, both `py` and `go`), `adapters` (`name` →
`path`, `version`), and `scenarios` (a list of `packages/<scenario_id>`
directory paths, relative to `bench/scenarios/`). Resolving a package's
`fixture.fixture_id` or `adapter.name` means looking it up in these two maps
— an unregistered id is rejected naming the field (REQ-F-015).

Each `bench/scenarios/packages/<scenario_id>/package.yaml` carries the full
I-04 field inventory. Every field is required unless noted. Loading the same
package twice MUST yield byte-identical values for every field below
(REQ-F-002/AC-009).

| Field | Type | Contract |
|---|---|---|
| `schema_version` | string | Matches the index; the version the validator supports. |
| `scenario_id` | string | Unique lowercase-kebab identity; the package's own directory name. |
| `scenario_version` | integer | Incremented on any content change. |
| `entity_family` | enum | `feature` \| `bug` \| `change_card` \| `tech_debt`. |
| `stage_matrix.prelude.D01`…`.D05` | object | `{applicable: bool, reason: string}`; `reason` required when `applicable: false`. Family invariant (REQ-F-004): `feature` requires all five `true`; `bug`/`change_card`/`tech_debt` require all five `false`, each with a reason. |
| `stage_matrix.lifecycle` | object | `{mode: all_dispatched, evidence_required: true}` — a declarative rule, never an enumerated status list. |
| `fixture` | object | `{fixture_id, submodule_path, base_sha}`; `fixture_id` must resolve in `scenarios.yaml`'s `fixtures:` map. |
| `adapter` | object | `{name, version}`; `name` must resolve in `scenarios.yaml`'s `adapters:` map. |
| `toolchain_identity` | ordered list | `[{key, value}]`, captured from a real `adapter.sh identity` run and pinned at admission; opaque to consumers (REQ-F-008) — compare the whole ordered list for equality, never a named key. |
| `input.agent_visible` | path | The issue-style initial input; MUST resolve outside the package's own `evaluator/` subtree. |
| `replay_reference` | path, feature only | Opaque pointer to the I-06 response bundle E40-F07 consumes; absent for every other family. |
| `evaluator_only` | object | `{reference_solution, oracle_tests[], answer_keys[]}`, all paths under `evaluator/`, never reachable from `input.agent_visible`. |
| `final_predicate` | object | `{kind, …operands}` — see "Final predicate vocabulary" below. Every kind carries a `p2p_selection` operand `{include: [fixture-relative paths], exclude_test_ids: [ids]}` (REQ-F-017). |
| `resource_policy` | object | `{max_cost_usd, max_wall_clock_seconds, max_generated_tasks}`, all strictly positive (REQ-F-011). |
| `admission` | object | `{status, base_outcome, reference_outcome, toolchain_identity}`, written in place by `admit-scenario.sh` on an admitted verdict. Its `toolchain_identity` is the second encoding of the top-level field and MUST equal it element-for-element (AC-019). |

The contract validator (`tests/contracts/e40_i04_scenario_contract_test.go`,
TC-030) is the executable definition of this schema, reading only committed
`bench/scenarios/**` files — it requires no populated submodule (REQ-NF-003),
so it runs in CI without `git submodule update --init`. Treat this section
as a reader's map to that test, not a substitute for it.

### Final predicate vocabulary

A closed set of four kinds, one permitted per family, each evaluable from an
adapter's `test`/`lint` capability output alone (REQ-F-010) — no ledger file
is read and none is committed under `bench/scenarios/` (AC-021).
`bench/scripts/eval-predicate.sh <package.yaml> <test-output.json>
<lint-output.json>` is the single named owner of this arithmetic; nothing
else re-derives it.

| Kind | Family | Operands | True when |
|---|---|---|---|
| `f2p_p2p` | `bug` | `f2p_test_ids[]`, `p2p_selection` | Every `f2p_test_ids` entry is `pass` and every `p2p_selection` entry is `pass`. |
| `acceptance_tests` | `change_card` | `acceptance_test_ids[]`, `p2p_selection` | Every acceptance test is `pass` and every `p2p_selection` entry is `pass`. |
| `p2p_plus_rule_drop` | `tech_debt` | `p2p_selection`, `rule`, `max_remaining` | Every `p2p_selection` entry is `pass` and the count of `lint` issues whose `rule` matches is `<= max_remaining`. |
| `child_oracles_union` | `feature` | `integration_test_ids[]`, `child_oracles[]`, `p2p_selection` | Every integration test is `pass`, every declared child oracle evaluates true, and every `p2p_selection` entry is `pass`. |

Every kind's P2P clause is absolute (REQ-F-017, ADR-F05-10), not
base-relative: every entry the `p2p_selection` resolves to must be `pass`,
whichever state (base or reference-applied) `test`/`lint` output was
captured from. No base ledger is required or committed, because REQ-F-016
requires the fixture's full suite green at `base_sha` and admission check
(b) verifies the narrower `p2p_selection` subset per candidate —
`bench/scripts/verify-fixture-py-base.sh <base_sha>` is the separate, named
owner of the broader "the whole fixture is green at base_sha" claim
(AC-020, TC-040).

### Adapter capability contract

`bench/adapters/<name>/adapter.sh <capability> [args]` (REQ-F-006) is an
executable contract, not a library — the only file that may know a
fixture's language, package manager, or toolchain (REQ-F-007), with one
named exception below. Every generic scenario, evidence, or admission
script reaches a language-specific command through this interface.
`bench/adapters/<name>/adapter.yaml` declares `{name, version}`, registered
in `scenarios.yaml`'s `adapters:` map. The exception:
`bench/scripts/id-collectors/` (see "Three-root policy" below), a
test-identity collector this feature (E40-F06) owns because REQ-NF-006
freezes `bench/adapters/**` byte-unchanged — it is language-aware by
necessity, but is never reached through the `adapter.sh <capability>`
interface and adds no capability to any adapter.

A closed set — six capabilities; adding a seventh requires an I-04
`schema_version` bump (that bump would touch every committed
`package.yaml`/`scenarios.yaml` file, I-04/E40-F05 corpus this feature's own
Integration Contracts row holds in `contract-only` gate mode). Each
capability writes one JSON document to stdout:

| Capability | Arguments | stdout JSON |
|---|---|---|
| `identity` | `--checkout <dir>` | `{adapter, version, toolchain_identity: [{key, value}]}` — ordered, opaque. |
| `inject-tests` | `--checkout <dir> --files <path>…` | `{injected: [{source, destination}]}`. Places evaluator-only test files where the toolchain discovers them. |
| `test` | `--checkout <dir> [--include <path>…] [--exclude-id <id>…] [--only-id <id>…]` | `{entries: [{id, outcome}]}`, `id` already normalized to `<module-or-package>::<test-name>`, `outcome` one of `pass`\|`fail`\|`skip`. `--include`/`--exclude-id` carry a `p2p_selection`; `--only-id` names a predicate's own test ids. |
| `lint` | `--checkout <dir>` | `{issues: [{rule, file, text}]}` — a multiset; identity excludes line/column so it is stable under position shifts. |
| `build` | `--checkout <dir>` | `{ok: bool, diagnostics: [string]}`. Used by admission check (a). |
| `format-check` | `--checkout <dir>` | `{ok: bool, offending_files: [string]}`. |

Exit status `0` means "the capability ran" — even when its *subject* is red
(a failing test, a lint issue, an unformatted file): that outcome is
reported IN the JSON, not via a non-zero exit. A non-zero exit means the
toolchain itself could not be invoked. This is what lets a generic
consumer — `admit-scenario.sh`, `eval-predicate.sh`, or a future E40-F06/08
component — read "did the check run" and "did the code pass" as two
independent signals without ever branching on which adapter answered.

Two adapters are registered today: `python` (`bench/adapters/python/`, the
four seed scenarios' fixture) and `go` (`bench/adapters/go/`, the I-01
compatibility adapter — its `test`/`lint` delegate to the unmodified
`bench/scripts/build-ledgers.sh`/`diff-ledgers.sh`, reshaped into this JSON
shape). `bench/scripts/tests/tc031_adapter_conformance_test.sh` runs the
identical assertion set against both, live, proving they emit the same
shape (AC-010) and normalized test ids (AC-011) without either adapter's
own scripts branching on which fixture is under test.

### Adding a new scenario package

1. Register the fixture (if new) and the adapter (if new) in
   `bench/scenarios/scenarios.yaml`'s `fixtures:`/`adapters:` maps.
2. Create `bench/scenarios/packages/<scenario_id>/package.yaml` with the
   full field inventory above, plus `input/prompt.md` (agent-visible) and an
   `evaluator/` subtree (`reference.patch`, any held-back oracle test
   files) — never cross-referenced from `input.agent_visible`.
3. Add the package's relative directory to `scenarios.yaml`'s `scenarios:`
   list.
4. Run `tests/contracts/e40_i04_scenario_contract_test.go` (`make test`) to
   confirm the schema is well-formed before attempting execution admission —
   it needs no populated submodule.
5. Run the Tier 2 sequence below to admit the package for real.

### Test tiers

Mirrors the I-01 tiering above, for the same reason (REQ-NF-003 keeps the
schema validator submodule-free so CI stays green without initializing
either submodule):

| Tier | Runs | Needs submodule? | Where |
|---|---|---|---|
| Tier 1 | `make test` (CI + every dev machine) | No — reads only committed scenario artifacts | `tests/contracts/e40_i04_scenario_contract_test.go` (TC-030) |
| Tier 1b | Curator, manually or via `bench/scripts/tests/run-all.sh` | Yes for the adapter conformance run (both checkouts); no fixture-language branching in the harness itself | `bench/scripts/tests/tc031_adapter_conformance_test.sh` (TC-031) |
| Tier 2 | Curator, at scenario-corpus build time and on every scenario/fixture/adapter change | Yes — `git submodule update --init` first (both `bench/fixture-repo` and `bench/fixture-py`) | `bench/scripts/{admit-scenario,eval-predicate,checkout-scenario-fixture,verify-fixture-py-base}.sh` against real checkouts |

### Tier 2 curator command sequence

A curator re-runs this exact sequence whenever a scenario package,
`bench/fixture-py`'s pinned `base_sha`, or an adapter changes, to confirm
REQ-NF-004's "byte-identical verdicts... at an unchanged fixture SHA and
toolchain identity" still holds:

```bash
# 1. Initialize both fixture submodules (once per clone/checkout).
git submodule update --init

# 2. Confirm the whole Python fixture is green at its pinned base_sha
#    (REQ-F-016/AC-020) -- broader than any one package's own
#    p2p_selection, so run this before admitting any package against it.
bench/scripts/verify-fixture-py-base.sh 964fa68e4c9e0c4e0f3756d9efd78b888c558fd9

# 3. Admit each of the four seed packages (REQ-F-012/013). Each writes its
#    own package.yaml's admission: block in place on an admitted verdict.
bench/scripts/admit-scenario.sh bench/scenarios/packages/py-bug-due-date-boundary/package.yaml
bench/scripts/admit-scenario.sh bench/scenarios/packages/py-change-priority-scale/package.yaml
bench/scripts/admit-scenario.sh bench/scenarios/packages/py-techdebt-consolidate-validation/package.yaml
bench/scripts/admit-scenario.sh bench/scenarios/packages/py-feature-recurring-tasks/package.yaml

# 4. To check one package's final_predicate directly against a captured
#    test/lint state (e.g. while authoring a new package), invoke the
#    single named predicate owner rather than re-deriving REQ-F-010:
#    bench/scripts/eval-predicate.sh <package.yaml> <test-output.json> <lint-output.json>

# 5. checkout-scenario-fixture.sh is the generic, fixture_id-keyed sibling
#    of checkout-fixture.sh (REQ-NF-006) -- use it directly to inspect a
#    fixture checkout without running the full admission gate:
#    bench/scripts/checkout-scenario-fixture.sh py 964fa68e4c9e0c4e0f3756d9efd78b888c558fd9 /tmp/shark-bench-scenario-checkout

# 6. Run the full bench self-test suite (I-01's Tier 1b/2 scripts plus
#    I-04's TC-031 through TC-041, run together).
bench/scripts/tests/run-all.sh
```

`admit-scenario.sh`'s own exit code is its per-candidate summary assertion:
`0` if admitted, `1` if rejected (naming the specific failing check), `2` on
a script/toolchain error that kept a check from even running. Re-running
step 3 against an unmutated package on an unchanged checkout/toolchain
reproduces byte-identical `package.yaml` content (REQ-NF-004,
`bench/scripts/tests/tc034_admission_determinism_test.sh`).

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
| `quality.gate_reasons` | object | Present only for null gates; maps a record field such as `fmt_clean` to its non-empty execution-failure reason. |
| `quality.postrun_abort` | string | Present only with `postrun_check_aborted` after a later post-run command fails; one of `build_ledgers`, `test_diff`, or `lint_diff`. |
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

## I-05 stage evidence and isolation contract (E40-F06)

`bench/evidence/` is I-05: a separate, file-backed, schema-versioned
evidence bundle format independent of I-02's run-record schema (REQ-F-001).
I-05 records nothing live — F06 defines and validates the schema, the
three-root isolation guard, the bundle validator, the replay guard, and the
X-09 usage-mapping canary; E40-F08 populates real bundles during a real
lifecycle run, E40-F09 consumes them for comparison identity, and E40-F10
derives reports from them. E40-F08/F09/F10 read this section instead of
re-deriving the bundle shape, the ledger rules, or the guard invocation
order from `bench/evidence/**` or the scripts directly — the same role the
"Manifest schema" and "I-04 scenario package schema" sections play for
I-01 and I-04.

The single machine-readable owner of every closed vocabulary below —
`stage_category`, interval category, `artifact_type`, `edge_kind`,
`evaluator_access.phase`, stop outcome, and `errors[].kind` — is
`bench/evidence/i05-schema.yaml` (REQ-F-017). The Go contract validator
(`tests/contracts/e40_i05_stage_evidence_contract_test.go`, TC-042) and
every guard script under `bench/scripts/` read that file at call time
rather than embedding a private copy, so a vocabulary change cannot land
in one consumer and not the other. Treat this section as a reader's map to
that file and to `bench/evidence/usage-mapping.yaml`, not a substitute for
either.

### Three-root policy (REQ-F-002)

Every bundle declares exactly three roots, each with a fixed `worker_access`
mode. The three paths MUST be pairwise disjoint — no root may be nested
inside, or equal to, another; a bundle declaring fewer than three roots, or
an overlapping pair, is rejected naming the offending pair.

| Root | `worker_access` | Meaning |
|---|---|---|
| `agent_fixture_checkout` | `read_write` | The I-04 fixture checkout the worker edits directly. |
| `scratch_shark_project` | `authorized_surfaces_only` | The scratch Shark project the lifecycle runs against — Shark writes here too, not only the worker. |
| `evaluator_only` | `never_during_dispatch` | `reference_solution`, `oracle_tests[]`, `answer_keys[]` — reachable by neither agent-visible root at any dispatch boundary (REQ-F-010/011). |

Two guards enforce this against the *live* roots, never only the declared
I-04 package layout:

- `bench/scripts/verify-evidence-roots.sh <package.yaml> <fixture_checkout>
  <scratch_project> <evaluator_root>` — one script serving both the
  REQ-F-010 admission-time check (once per candidate package, against a
  fresh checkout) and the REQ-F-011 dispatch-boundary check (immediately
  before every worker dispatch, against the live in-flight roots). It walks
  **both** agent-visible roots independently — a guard that walks only
  `--workdir` misses everything Shark writes into the scratch project
  (ADR-F06-03) — and derives every evaluator-only name, digest, and test
  identity from `<package.yaml>` at call time, never from a hardcoded list
  (REQ-F-010, AC-010). Exit `0` = both roots clean ("CLEAN" on stdout).
  Exit `1` = an isolation violation, naming the offending root, path, and
  matched evaluator-only source. Exit `2` = a script/usage/authoring error.
  For an `oracle_tests[]` entry, one signal (`derived_test_identity`) derives
  the file's REAL, normalized test identity(ies) rather than approximating
  them from the file's own name (T-E40-F06-003 round-4 UAT fix) — it
  resolves the package's declared `adapter.name` through TWO registries
  (never a raw path join of that candidate-controlled string, round-3
  code-review fix): `scenarios.yaml`'s own `adapters:` map confirms it names
  a real, registered I-04 adapter; `bench/scripts/id-collectors/registry.yaml`
  then maps that same name to a collector script this feature owns. This
  keeps the capability outside `bench/adapters/**` entirely (REQ-NF-006
  freezes that tree) while still leaving the generic guard itself ignorant
  of any language's syntax (REQ-F-007) — it only shells out to the resolved
  collector, e.g. `bench/scripts/id-collectors/python-collect-ids.sh
  --checkout <dir> --file <path>` (a real toolchain-collection subprocess,
  no test body ever executed), emitting `{ids: [{id, name}]}` for every
  identity a file defines. The python collector derives each `name` from
  pytest's own STRUCTURED collection data (`item.originalname`, read via a
  `pytest_collection_modifyitems` plugin hook) rather than parsing
  `--collect-only`'s free-form text output — a bare `def` function name,
  parametrize-suffix-free, with no custom `ids=` content ever mixed in,
  even when that custom id itself contains a space or `::` (T-E40-F06-003
  round-4 code-review fix; the prior text-parsing implementation silently
  dropped or mis-derived exactly those two shapes).
- Evaluator access after the isolation boundary is lifted only in the
  REQ-F-012 order — see "Evaluator access ordering" below.

### Bundle layout (I-05)

One directory per run under an operator-supplied evidence output root,
mirroring the run-directory convention "Run directory layout" documents
for I-02:

```
<evidence_root>/<scenario_id>/<run_id>/
├── bundle.json                 # identity, roots, stage index, stop outcome, eligibility
├── stages/
│   └── <dispatch_ordinal>-<stage_key>.json   # one immutable snapshot per stage
└── access.jsonl                # append-only evaluator_access events
```

`bundle.json` top-level fields (every field required unless marked):

| Field | Type | Contract |
|---|---|---|
| `schema_version` | string | The I-05 version `i05-schema.yaml` declares and TC-042 supports. |
| `scenario` | object | `{scenario_id, scenario_version, entity_family}`, copied verbatim from the I-04 package. |
| `run_id` | string | The lifecycle run this bundle belongs to. Opaque to F06; E40-F08 assigns it. |
| `roots` | object | The three roots above, each `{path, worker_access, identity_digest}`. Pairwise disjoint. |
| `stage_matrix_source` | object | `{package_path, package_digest, prelude, lifecycle}` — a snapshot of the I-04 halves this bundle's completeness is evaluated against, taken at run time (REQ-F-004). |
| `stages` | array | Ordered index `{dispatch_ordinal, stage_key, stage_category, snapshot_path, snapshot_digest}`. `dispatch_ordinal` is unique within the bundle. |
| `terminal_status` | object | `{reached: bool, reached_at}` — gates the `--grant-access inject-tests` broker (REQ-F-012). |
| `stop_outcome` | string, optional | Absent on a clean terminal run; one of the ten values below otherwise. |
| `publication_eligible` | bool | `false` whenever `stop_outcome` is present (REQ-F-014). |
| `ineligibility_reasons` | array of string | Non-empty whenever `publication_eligible` is `false`. |

### Stage-snapshot field reference

`stages/<dispatch_ordinal>-<stage_key>.json`, one immutable, content-addressed
document per dispatched stage:

| Field | Type | Contract |
|---|---|---|
| `dispatch_ordinal` | integer | Matches the bundle index entry; unique per bundle. |
| `entity` | object | `{entity_key, entity_type}` for the concrete dispatched entity — never the cascade parent. |
| `stage_key` | string | The workflow step or prelude stage (`D01`–`D05`) this dispatch served. |
| `stage_category` | enum | One of `discovery`, `specification`, `planning`, `code`, `review`, `qa`, `uat`, `shipping` (REQ-F-003). |
| `prompt_digest` | string | `sha256` of the rendered prompt. The prompt text itself is never stored in the snapshot. |
| `input_lineage` | array | `{source_kind, path, digest}` for every input the stage consumed. |
| `replay_lineage` | array, feature family only | `{replay_reference, entry_digest}` pointers into the I-06 bundle; opaque interior (E40-F07 owns it). |
| `artifacts` | array | `{artifact_type, path, digest, size_bytes, producer_stage, consumers}` (REQ-F-008). `consumers: []` ("orphan", no consumer observed) and an absent `consumers` key ("consumption evidence not collected") are two distinct, never-coerced states. |
| `usage` | object | Semantic slots from `usage-mapping.yaml` (see "Usage slot table" below). A slot the mapping could not resolve is absent, with a matching `usage_slot_unavailable` entry in `errors[]` — never zero, never null. |
| `time_ledger` | object | See "Time ledger rules" below. |
| `candidate` | object, `code`/`review` only | REQ-F-006's six required fields — `base_commit`, `tree_digest`, `binary_diff_digest`, `changed_path_digest`, `dirty_untracked_manifest` (ordered `{path, digest, tracked}`), `test_suite_digest` — plus two replay-guard fields `replay-stage-evidence.sh` reads: `test_suite_ids` (the recorded normalized `<module-or-package>::<test-name>` id set the replay guard diffs against a live `<adapter> test` invocation to name a differing test by id, since the opaque `test_suite_digest` alone cannot) and `test_suite_dir` (excluded from the file-drift walk as the test-suite check's own domain). `base_commit` is one field of the identity, never the identity alone (REQ-F-006, ADR-009). |
| `errors` | array | `{kind, detail, …}`, `kind` resolving against `i05-schema.yaml`'s `error_kind` vocabulary. |
| `rework_count` | integer | Re-entries into this stage for this entity. |
| `evaluator_access` | array | REQ-F-012 events; also appended to the bundle's `access.jsonl`. |
| `snapshot_digest` | string | `sha256` over the canonical serialization excluding this field (REQ-F-015). Recomputing it must reproduce the recorded value; a mismatch is `snapshot_mutated`. |

`bench/scripts/verify-stage-evidence.sh <bundle_dir>` is the single named
owner of this validation arithmetic — field inventory, root policy,
REQ-F-004's completeness split (prelude `missing_stage` vs. lifecycle
duplicate/unmatched dispatch), ledger reconciliation, candidate identity,
artifact records, usage fail-closed posture, access ordering, stop-outcome
eligibility, and snapshot immutability — so F08/F09/F10 invoke it rather
than re-deriving I-05's semantics, the discipline `eval-predicate.sh`
established for I-04 and `diff-ledgers.sh` for I-01. Exit `0` = valid (a
fixed-order JSON summary on stdout, sorted by `dispatch_ordinal`). Exit
`1` = a named rejection verdict on stderr. Exit `2` = a script/usage error.

### Time ledger rules (REQ-F-005)

Every snapshot's `time_ledger` is `{stage_start, stage_end,
reconciliation_epsilon_ns, intervals: {<category>: [[start, end), …]}}`
over six categories: `provider_active`, `tool_and_test`,
`queue_or_claim_wait`, `replay_or_human_gate_wait`, `retry_or_backoff`,
`unclassified`.

- Every interval is a genuine **half-open** `[start, end)` span
  (`start < end`), fully contained in `[stage_start, stage_end)`. An
  interval that escapes the stage window is rejected naming the interval
  and the window.
- No two intervals overlap, across **any** two categories — rejected
  naming both offending categories and both intervals.
- The union of all intervals must reconcile to `[stage_start, stage_end)`
  within `reconciliation_epsilon_ns`; any residual **within** the epsilon
  lands in `unclassified`, never in `provider_active`. A residual larger
  than the epsilon is rejected naming its magnitude.
- Unknown or unattributable time is never assigned to `provider_active` —
  the one property UAT-16 turns on.

`verify-stage-evidence.sh` is this rule set's single owner; nothing else
re-derives it.

### Usage slot table (X-09)

`bench/evidence/usage-mapping.yaml` binds each semantic slot I-05 records
to a concrete provider envelope path, never a field name written inline
into `i05-schema.yaml` and never E27-F15's unmerged Go structs
(ADR-F06-04). It carries its own `schema_version`, a `verified_from`
provenance block, and one block per provider. Consumers read a slot by its
semantic name and never hard-code an envelope path.

| Semantic slot | `anthropic_claude_cli` envelope path | `verification_tier` | Required identity slot? |
|---|---|---|---|
| `total_cost` | `total_cost_usd` | `real_capture` | Yes |
| `input_tokens` | `usage.input_tokens` | `real_capture` | Yes |
| `output_tokens` | `usage.output_tokens` | `real_capture` | Yes |
| `cache_read_input_tokens` | `usage.cache_read_input_tokens` | `real_capture` | Yes |
| `cache_creation_input_tokens` | `usage.cache_creation_input_tokens` | `real_capture` | Yes |
| `model_ids` | sorted keys of `modelUsage` | `real_capture` | Yes |
| `api_active_duration_ms` | `duration_api_ms` | `real_capture` | Yes |
| `turn_count` | `num_turns` | `real_capture` | Yes |
| `provider_session_id` | `session_id` | `unverified` | No — deliberately excluded (ADR-F06-12) so no `unverified` slot gates G14 comparison identity. |

`openai_codex_cli` is declared `unmapped`: `buildCodexArgs` on `main` does
not pass `--json`, so a codex stage's transcript stdout is not a decodable
envelope today, and every slot fails closed. A provider declared `unmapped`
MUST NOT be decoded by guess.

`bench/scripts/canary-usagemapping.sh [--transcript <path>]` re-verifies
every `anthropic_claude_cli` slot against a **real captured envelope**,
defaulting to the committed fixtures under `bench/scripts/testdata/run/`
and accepting an operator-supplied live transcript. It tells apart two
REQ-F-019 drift classes rather than conflating them:

- **envelope-field drift** — one mapped path absent from a real captured
  envelope: fails naming that slot and path
  (`usage_slot_unavailable slot=<slot> envelope_path=<path>`).
- **envelope-availability drift** — the transcript's `---STDOUT---` block
  is no longer decodable JSON at all (e.g. a lifecycle change that starts
  persisting assistant prose instead of the raw envelope): fails as **one**
  whole-source failure (`envelope_source_unavailable transcript=<path>`),
  never as nine independent per-slot failures.

Every diagnostic goes to stderr, ending in exactly one line reading `PASS`
or `FAIL: <field>` — the same convention `canary-runsurface.sh` already
established. Exit `0` = PASS. Exit `1` = a named drift verdict. Exit `2` =
a script/usage error.

### Evaluator access ordering (REQ-F-012)

Evaluator-only material becomes readable only after its authorized
boundary, and every read appends one `evaluator_access` event
(`{accessor, artifact_path, digest, phase, granted_at}`, `phase` one of
`pre_terminal`/`post_terminal`) to the bundle:

1. Absent from both agent-visible roots at every dispatch boundary
   (`verify-evidence-roots.sh`, above).
2. After the applicable stage or scenario reaches `terminal_status`, a
   held-back oracle test MAY be placed into the fixture checkout, and only
   through I-04's `adapter.sh inject-tests` capability:
   `verify-stage-evidence.sh <bundle_dir> --grant-access inject-tests
   --accessor <name> --adapter <adapter.sh> --checkout <dir> --files
   <path>…`. Requested before `terminal_status.reached`, this is rejected
   as `isolation_violation` and the adapter is never invoked.
3. The post-run oracle reads reference solutions and answer keys **in
   place** from the `evaluator_only` root, never by copying them into the
   worker checkout first:
   `verify-stage-evidence.sh <bundle_dir> --grant-access in-place-read
   --accessor <name> --evaluator-root <dir> --artifact <rel_path>
   --checkout <dir>`. A read performed by first copying the file into the
   worker checkout is rejected naming the violation, not merely warned.

### Replay and immutability (REQ-F-013, REQ-F-015)

`bench/scripts/replay-stage-evidence.sh <bundle_dir> [--checkout
<fixture_checkout>] [--adapter <adapter_path>]` re-evaluates a **stored**
bundle against its named roots with no worker rerun and no provider call —
every field it needs is resolvable from the bundle plus the caller-supplied
roots. For every indexed stage:

1. Recomputes `snapshot_digest` over the snapshot's own canonical
   serialization (excluding that field). A mismatch is `snapshot_mutated`,
   naming the stage; when the mismatched content also shows an `artifacts[]`
   entry missing its `consumers` key, it additionally (never instead)
   reports `artifact_consumption_record_missing`.
2. Only when the digest matches and `stage_category` is `code`/`review`,
   two independent drift checks against `--checkout`, sourced from the
   snapshot's own `candidate` block: file drift
   (`tracked_file_changed`/`untracked_file_changed`, naming the path) from
   `dirty_untracked_manifest` (excluding `candidate.test_suite_dir`, the
   test-suite check's own domain below), and test-suite drift
   (`test_suite_changed`, naming the differing test id) from a live
   `<adapter> test --checkout` invocation diffed against
   `candidate.test_suite_ids`, the recorded normalized id set.

Exit `0` = replay clean (a JSON summary on stdout). Exit `1` = one or more
named drift/mutation verdicts on stderr. Exit `2` = a script/usage error.

### Stop outcomes and partial evidence (REQ-F-014)

A bundle terminating in one of the ten named stop outcomes —
`resource_limit`, `lease_loss`, `missing_outcome`, `unresolved_gate`,
`pause`, `archive`, `error`, `cancellation`, `worker_failure`, `timeout` —
retains its partial stage snapshots and sets `publication_eligible: false`
with a non-empty `ineligibility_reasons[]`. Partial evidence is never
discarded, and is never readable as a valid baseline contribution. A
bundle pairing any stop outcome with `publication_eligible: true` is
rejected as `publication_eligible_conflict`.

### Tier 2 guard invocation sequence

The four guard scripts above are offline once the fixtures and caches are
present — zero provider calls, byte-identical verdicts across repeated
runs at an unchanged bundle, fixture SHA, and toolchain identity
(REQ-NF-004):

```bash
# 1. Admission time, once per I-04 candidate package, against a fresh
#    checkout (REQ-F-010):
bench/scripts/verify-evidence-roots.sh \
  bench/scenarios/packages/py-feature-recurring-tasks/package.yaml \
  <fresh_fixture_checkout> <empty_scratch_project> <evaluator_root>

# 2. Dispatch boundary, immediately before EVERY worker dispatch, against
#    the live in-flight roots (REQ-F-011) -- E40-F08's dispatch loop calls
#    this, not F06.
bench/scripts/verify-evidence-roots.sh \
  <package.yaml> <live_fixture_checkout> <live_scratch_project> <evaluator_root>

# 3. After a stage or run completes, validate the bundle's field shape,
#    completeness, ledger, candidate identity, artifacts, usage, access
#    ordering, and eligibility in one pass:
bench/scripts/verify-stage-evidence.sh <evidence_root>/<scenario_id>/<run_id>

# 4. Grant evaluator access only through the broker, only in the
#    authorized order (REQ-F-012) -- see "Evaluator access ordering" above.

# 5. Re-evaluate a stored bundle with no worker rerun and no provider call
#    (REQ-F-013/015):
bench/scripts/replay-stage-evidence.sh <bundle_dir> --checkout <fixture_checkout> --adapter <adapter.sh>

# 6. Re-verify the X-09 usage mapping against a real captured envelope
#    (run at usage-mapping.yaml authoring time, or as a manual Tier 2
#    spot-check against a live transcript):
bench/scripts/canary-usagemapping.sh
bench/scripts/canary-usagemapping.sh --transcript <live_transcript_path>

# 7. Run the full bench self-test suite (I-01/I-04's scripts plus I-05's
#    TC-043 through TC-051, run together). TC-042 (the Go contract
#    validator) runs under `make test`, not here -- the same split TC-030
#    already established for I-04.
bench/scripts/tests/run-all.sh
```

`verify-evidence-roots.sh` never invokes a dispatcher of any kind — the
property a PATH-stubbed dispatcher's empty invocation log proves — so step
2 always completes before any provider spend. Steps 3 through 6 never make
a provider call either; `replay-stage-evidence.sh`'s own test-suite drift
check invokes only the I-04 adapter's `test` capability, never a provider.

## I-06 product-design replay contract (E40-F07)

I-06 wraps the existing Shark Rider product-design action (X-10) and its
bundled D01-D05 methodology — never forks it. `skills/shark-rider/verbs/
product-design.md` and every file under `internal/sharkdata/default_data/
skills/product-design/**` are byte-frozen (REQ-NF-006(a)); the routing this
feature needs arrives instead as a benchmark-owned, digest-pinned preamble
(`bench/replay/preamble.md`) prepended to the dispatch. F07 defines and
validates the I-06 schema, the replay bundle for the seed feature scenario,
the resolver, the prelude host-side adapter, and the isolation/lineage
guards; it does not start the keyed Shark entity lifecycle (E40-F08 / I-07)
and does not score artifact quality (E40-F09 / I-08). E40-F08 and E40-F10
read this section instead of re-deriving the document shapes, the
`entry_digest` join rule, the resolver's matching semantics, the live-egress
proof, or the guard invocation order from `bench/replay/**` or the scripts
directly — the same role the "I-05 stage evidence and isolation contract"
section above plays for I-05.

The single machine-readable owner of every closed vocabulary below —
`document_kinds`, `stage`, `request_kind`, `artifact_type`, `edge_kind`, the
`replayed_interaction_proxies` field set, `terminal_outcome`, and
`error_kind` — is `bench/replay/i06-schema.yaml` (REQ-F-018). The live-egress
denial set is owned by the separate `bench/replay/live-egress-tools.yaml`,
per REQ-F-018's single-owner rule applied per vocabulary. The Go contract
validator (`tests/contracts/e40_i06_product_design_replay_contract_test.go`,
TC-052) and every bench guard script under `bench/scripts/` read both files
at call time rather than embedding a private copy. Treat this section as a
reader's map to those two files, not a substitute for either.

### Two-document split (REQ-F-001)

I-06 is **two** schema-versioned, file-backed documents with distinct roles
— never one bundle-of-snapshots the way I-05 is. Neither redefines,
retypes, or duplicates an I-04 or I-05 field; both reference those
contracts by path and digest only.

- **Document A — the replay bundle** (I-06 *input*). The committed,
  versioned file I-04's `package.yaml` `replay_reference` points at.
- **Document B — the replay result** (I-06 *output*). The per-run document
  E40-F08 consumes; the interaction map calls this "the product-design
  replay result."

TC-052 validates both shapes from `document_kinds.bundle`/`document_kinds
.result` in `i06-schema.yaml` and rejects a document of one kind supplied
where the other is expected, naming the expected kind — a result opened as
a bundle, or the reverse, fails loudly rather than half-validating.

#### Document A fields — replay bundle

| Field | Type | Contract |
|---|---|---|
| `schema_version` | string | The I-06 version `i06-schema.yaml` declares. |
| `bundle_version` | string | Bumped whenever any entry changes; recorded in the result so a rerun against a different bundle is visible, not silent. |
| `scenario_binding` | object | `{scenario_id, scenario_version}`; `scenario_id` MUST equal the owning package's (REQ-F-014). |
| `entries` | array | Ordered authorized entries, each `{entry_id, stage, ordinal, request_kind, topic_key, required, response, response_digest, entry_digest}`. `stage` is `D01`-`D05`; `request_kind` is `human_question` or `research_query`; `ordinal` is unique within its stage; `response` is inline text or a `{path, digest}` reference resolved and contained relative to the bundle file; `entry_digest` is the REQ-F-003 join key (below). Every entry is single-use within a prelude run. |

The seed fixture is `bench/scenarios/packages/py-feature-recurring-tasks/
evaluator/replay/reference-bundle.json` — the one I-04 carve-out
REQ-NF-006(c) permits this feature to write; `package.yaml`'s
`replay_reference` pointer is unchanged.

#### Document B fields — replay result

| Field | Type | Contract |
|---|---|---|
| `schema_version` | string | As above. |
| `scenario` | object | `{scenario_id, scenario_version, entity_family}`, copied verbatim from the I-04 package. |
| `run_id` | string | Opaque to F07; E40-F08 assigns it. |
| `replay_bundle` | object | `{replay_reference, bundle_path, bundle_digest, bundle_version}` — the exact input this run consumed. |
| `preamble_digest` | string | `sha256` of `bench/replay/preamble.md` as dispatched. |
| `artifact_root` | object | `{path, identity_digest, root_kind: "scratch_shark_project"}` (REQ-F-015; see "Artifact placement" below). |
| `stages` | array | One record per `D01`-`D05`: `{stage, applicable, reason?, artifacts[], consumed_entries[]}`. `reason` is required and copied verbatim when `applicable` is `false`. |
| `stages[].artifacts` | array | `{artifact_type, path, digest, size_bytes, produced_at, revision_index, prompt_digest, input_digests[], consumed_entries[], consumers[]}`. `consumers: []` ("orphan") and an absent `consumers` key ("consumption evidence not collected") are distinct and never coerced — the same rule I-05 fixed for its own artifact records. `consumers[]` edges are `{consuming_stage, edge_kind, observed_at}`. |
| `stages[].consumed_entries` | array | `{entry_id, entry_digest, request_kind, topic_key, supplied_at, request_bytes, response_bytes}` — written only by the resolver ledger (see "Lineage reconciliation" below). |
| `replayed_interaction_proxies` | object | REQ-F-011 closed field set (see "Interaction-volume proxies" below). |
| `artifact_consumption_edges` | array | `{producer_stage, artifact_path, consuming_stage, edge_kind}` — the flattened cross-stage view E40-F10 reads for reuse and orphan counts. |
| `terminal_outcome` | enum | The closed set (see "terminal_outcome and the I-07 seam" below). |
| `i07_stop_mapping` | string, required on every stop outcome | The I-07 bucket E40-F08 propagates. |
| `publication_eligible` | bool | `false` for every outcome other than `complete` and `not_applicable`. |
| `ineligibility_reasons` | array of string | Non-empty whenever `publication_eligible` is `false`. |

### `entry_digest` — the I-05 join key (REQ-F-003)

`entry_digest` is a `sha256` over one `entries[]` element's canonical
serialization, excluding the digest field itself — decoded-object keys
sorted lexicographically at every nesting level, compact separators, no
`\uXXXX`-escaping, integers with no fractional part or leading zero
(`i06-schema.yaml`'s `entry_digest:` block; equivalent to `jq -cS` or
Python's `json.dumps(obj, sort_keys=True, separators=(",", ":"),
ensure_ascii=False)`). It is recomputable by any consumer from the stored
bundle alone, and it is the **single field** two contracts join on: E40-F08
writes it verbatim into each I-05 stage snapshot's `replay_lineage[]
.entry_digest` alongside the bundle path as `replay_reference`
(`bench/evidence/**`'s own `replay_lineage[]` field notes this interior as
"opaque — E40-F07 owns it"). A one-byte edit to any entry field changes its
recomputed digest; a result whose `consumed_entries[].entry_digest` values
are not a subset of the bundle's own freshly recomputed digest set is
rejected `replay_bundle_mutated`, naming the entry — whether the cited
bundle entry was edited after the digest was recorded or the result cites a
digest no bundle entry produces (a join-key spoof).

### Resolver semantics — `bench/scripts/replay-answer.sh` (REQ-F-006/007)

```
replay-answer.sh --bundle <path> --stage <D01|...|D05> \
                 --kind <human_question|research_query> --topic <key>
```

The **single named owner** of request matching, response supply, and
consumption recording — no other script, test, or the dispatched session
itself may re-derive these semantics; the session reaches this script only
through the preamble's routing instruction.

Matching is **ordinal-primary with a topic assertion** (ADR-F07-04), never
a match against the model's own literal request text, which regenerates
differently every run. The resolver looks at the bundle's `entries[]` for
the caller's `--stage`, finds the **lowest ordinal not yet consumed** in
this bundle's own consumption ledger, and supplies its response only when
that entry's own `request_kind`/`topic_key` both equal the caller-supplied
`--kind`/`--topic`. There is no nearest, partial, or fuzzy match — a
one-character near-miss fails exactly like any other disagreement. Three,
and only three, outcomes exist:

- **Exit 0 — supplied.** The lowest-unconsumed-ordinal entry's
  `request_kind`/`topic_key` both match. The response bytes are written
  verbatim to stdout (no added newline), and exactly one consumption record
  is appended to the ledger before the response is printed.
- **Exit 1 — `replay_desync`.** An unconsumed entry exists for the stage,
  but its `request_kind` or `topic_key` disagrees with the caller's. Stderr
  names both the expected (bundle-declared) and supplied (caller-given)
  kind/topic. No response is printed; no consumption record is appended.
- **Exit 1 — `unresolved_gate`.** No unconsumed entry remains for the stage
  (exhausted, or the bundle never declared one). Stderr names stage, kind,
  and topic. REQ-F-008 requires the *caller* (`run-prelude.sh`, ultimately
  E40-F08's dispatch loop) to stop the prelude on this outcome and never
  invent, paraphrase, or degrade to a default answer — this script's own
  job is only to report the gate precisely enough for that caller to do so.
- **Exit 2 — `ScriptError`.** A malformed bundle or an unresolvable
  `{path, digest}` response (escapes the bundle directory, missing file, or
  resolved bytes that do not hash to the declared digest) — never a
  matching verdict.

**Ledger side-file.** This script is the single writer of the consumption
ledger, at `<bundle_path>.consumption.jsonl`, one compact JSON object per
successful supply, appended in call order: `{entry_id, entry_digest, stage,
ordinal, request_kind, topic_key, response_digest, supplied_at}`. A missing
side-file means every entry in the bundle is unconsumed — a fresh scratch
project's own bundle copy simply has no side-file yet, which is how two
independent scored passes over the identical committed bundle, driven by
the same call sequence, each start from an empty ledger and consume a
byte-identical response sequence (REQ-F-007, AC-005). No network call is
made anywhere in this script.

### Live-egress denial and the observational binding gate (REQ-F-004/005)

`bench/replay/live-egress-tools.yaml` is the one owner of the **live-egress
set** a scored prelude dispatch must not reach: `AskUserQuestion`,
`WebSearch`, and `WebFetch` — `WebFetch` included even though the wrapped
bundle's own "Tools Used" section never names it, because "cannot reach
live research" is a reachability property of the session, not a
documentation property of the bundle (ADR-F07-02). Enforcement is
**tool-name-scoped and session-wide**, never a per-call-site list. Both
halves read this one file at call time, so adding or removing a tool
changes both enforcement and detection with no script edit:

| Half | Mechanism | Binding? |
|---|---|---|
| Structural denial | `bench/scripts/run-prelude.sh` emits one `--disallowedTools <tool>` argument per set member and refuses to dispatch — `argv_incomplete`, exit 1, before any subprocess spend — if any member is missing from the final constructed argv | No — belt-and-braces |
| Observational detection | `bench/scripts/verify-replay-isolation.sh <transcript_path>` scans the retained scored-run transcript for tool-use records naming any set member; one hit is `live_interaction_reached`, naming the tool, stage, and transcript line | **Yes** — this is the contract |

The binding half rests on no assumption about a provider CLI's own denial
semantics (ADR-F07-03): a transcript is JSONL, one JSON object per
non-blank line; a `{"type": "tool_use", "stage": "D0X", "tool_name": "..."}`
record is a tool invocation, and any `tool_name` that is not itself a
live-egress-set member is ordinary, permitted tool use (`Read`, `Write`,
`Bash` for `replay-answer.sh`, ...). The scanner **fails closed, never
open**: a transcript in which it recognizes zero `tool_use` records
anywhere is refused (`ScriptError`, exit 2), never silently certified
clean, because a real D01-D05 session always uses at least one tool. Exit
0 = clean (`CLEAN` on stdout).

### Bundle-disclosure guard (REQ-F-012)

`bench/scripts/verify-replay-isolation.sh <bundle_path> <fixture_checkout>
<scratch_project>` is the second invocation shape the same script
reserves, dispatched on argument count (three positional args) rather than
a subcommand — a **new** guard, not an edit to F06's frozen
`verify-evidence-roots.sh`, because the replay bundle is not
`evaluator_only` material and falls outside that guard's contract by
design (ADR-F07-07). It proves the replay bundle — and any copy of it that
is byte-identical in content, however it is named — is absent from
**both** agent-visible roots at every dispatch. A planted bundle or
bundle-derived file is a named `bundle_bulk_disclosure` violation
identifying the root (`agent_fixture_checkout` or `scratch_shark_project`)
and the exact path. `replay_reference` is authorized input the session
legitimately consumes one entry at a time (ADR-F07-07) — a single
resolved entry response sitting in an agent-visible root is not a
violation; only the whole bundle, or a content-identical copy, is. Exit
0 = both roots clean (`CLEAN` on stdout). This guard is invoked against the
**live** in-flight roots, immediately before every scored dispatch — the
same "against the live roots, not only the declared package layout"
discipline F06's dispatch-boundary check established; `run-prelude.sh`
does not call it internally, the caller's dispatch loop does (E40-F08's
job, mirroring "E40-F08's dispatch loop calls this, not F06" above).

### Lineage reconciliation (REQ-F-009)

`bench/scripts/verify-replay-result.sh <result_path> <bundle_path>` is the
resolver ledger's reconciliation check: every artifact's claimed lineage
must reconcile against `<bundle_path>.consumption.jsonl`, the same
single-writer ledger `replay-answer.sh` appends to — never trusted
alongside it, always reconciled against it. Two failures are named and
kept distinguishable, because they have different causes and different
downstream owners:

- **`unattributed_artifact`.** A stage's artifact claims (in its own
  `consumed_entries[]`) an `{entry_id, entry_digest}` pair the ledger never
  recorded for that stage — fabrication or resolver bypass, named on the
  entry and the artifact's stage. Or: a stage produced at least one
  artifact whose combined `consumed_entries[]` intersects **none** of the
  bundle's own `required: true` entries for that stage — the artifact
  exists but nothing it was required to ground itself in was ever
  attributed to it, named on the stage.
- **`unresolved_gate`.** A stage that produced no artifact at all is
  untouched by either check above — REQ-F-008 already requires
  `unresolved_gate` to stop the prelude before an artifact for that stage
  is ever produced, so "zero artifacts, zero required entries consumed" is
  the shape a genuine gate takes, not a rejection this script raises. This
  script passes the result's own top-level `terminal_outcome` straight
  through unexamined on success — never asserting or deriving it, since
  full `terminal_outcome` vocabulary validation is TC-052's job — which is
  how a genuine `unresolved_gate` run is reported without this script ever
  manufacturing an `unattributed_artifact` verdict for it. A validator that
  collapsed the two into one verdict, in either direction, is rejected by
  AC-006.

A missing `<bundle_path>.consumption.jsonl` means the ledger is **empty**
(zero consumptions occurred), a valid state, not a script error — mirroring
`replay-answer.sh`'s own "missing side-file = every entry unconsumed"
convention. Exit 0 prints one fixed-order JSON summary (`stages[]` sorted
by `stage`, each stage's `artifacts[]` sorted by `path`, each artifact's
claimed entries sorted by the bundle's `ordinal`) — REQ-NF-004
byte-identical verdicts. Exit 1 = a named rejection on stderr. Exit 2 = a
script/usage/authoring error.

This script currently checks `stages[].artifacts[].consumed_entries[]`
reconciliation only — REQ-F-009's own two-verdict split, the slice
T-E40-F07-007 decomposed into. The remaining Document B field inventory
(REQ-F-001/002/010/011/013/017/019: proxy closure, non-applicable
completeness, stop-outcome eligibility, the full closed-vocabulary sweep)
is TC-052's job, over the static fixtures under
`tests/contracts/testdata/e40_i06/{valid,invalid}/` — the same
schema-validator/execution-guard split ADR-F07-10 fixes and ADR-F06-09/
ADR-F05-07 already established for I-05/I-04. Consumers needing the full
result-shape check invoke TC-052 (`go test ./tests/contracts/...`), not
this script alone.

### Interaction-volume proxies (REQ-F-011)

`replayed_interaction_proxies` carries a required `measurement_kind:
"replayed_interaction_proxy"` discriminator and exactly the closed field
set `i06-schema.yaml`'s `replayed_interaction_proxies_fields:` declares:
`authorized_request_count`, `authorized_response_count`,
`request_bytes_total`, `response_bytes_total`,
`revision_or_replacement_count`, `replay_wait_ns`, `replay_wait_category`,
`unresolved_gate_count`. Any field outside this set, any `measurement_kind`
other than the discriminator, and any field name or unit expressing human
time, stakeholder minutes, or cognitive effort is rejected naming the
offending field — the structural half of UAT-18's "no report labels these
as human minutes." `replay_wait_category` MUST be I-05's own
`replay_or_human_gate_wait` interval-category value, referenced by name,
not redefined. `replay_wait_ns` is the harness's own resolver-latency
measurement for a local file read — no F07 component synthesizes, pads, or
models a human-latency delay (REQ-NF-007); a value exceeding
`i06-schema.yaml`'s `replay_wait_ns_plausibility_ceiling` is rejected
`proxy_wait_implausible` as a synthesized delay. `unresolved_gate_count`
must agree with `terminal_outcome`: an `unresolved_gate` outcome paired
with a zero count is rejected `unresolved_gate_count_inconsistent` — the
gate-count value can contradict the terminal outcome it exists to explain
even when the proxy block's own field inventory is otherwise valid.

### Artifact placement (REQ-F-015/016)

The prelude dispatch's working directory is I-05's own
`roots.scratch_shark_project` — D01-D05 artifacts and `docs/product/
progress.md` are planning documents written there, never into the
agent-visible fixture checkout and never into the evaluator-only root. F07
introduces **no fourth root**. `bench/scripts/run-prelude.sh --package
<package.yaml> --result-out <path> --artifact-root <scratch_shark_project>`
pins the dispatched subprocess's `cwd` to that root and records its
realpath plus a recomputable identity digest as the result's
`artifact_root` object. `bench/replay/preamble.md` — the digest-pinned,
methodology-free interaction-routing preamble — is read from disk and
prepended to every dispatch; its `sha256` is recorded as `preamble_digest`
so prompt identity is pinned for E40-F09. The preamble references the
resolver only by two environment-exported names
(`$REPLAY_ANSWER_SCRIPT`/`$REPLAY_BUNDLE_PATH`), never by baking a
scenario-specific path into the preamble's own bytes, so the same file and
the same digest apply to every scenario. Before writing a `complete`
result, `run-prelude.sh` positively asserts every artifact produced under
`<artifact_root>/docs/product/`, other than `progress.md` (the wrapped
action's own derived record), matches the bundle's own Output Standard
filename pattern `D0X-*.md` — a violation is named and rejected before any
result is written. The provider binary is resolved from `PATH` (never a
hardcoded absolute path), mirroring `internal/runner/claude_dispatcher.go`'s
own `exec.LookPath("claude")` resolution read there for **posture only** —
nothing under `internal/` is imported (REQ-NF-001).

### Non-applicable families (REQ-F-013/014)

When every `stage_matrix.prelude.D01`-`.D05` entry of an I-04 package is
`applicable: false`, `run-prelude.sh --package <package.yaml> --result-out
<path>` never invokes the Rider action — `main()` returns after writing the
result, before `build_disallowed_args`/dispatch is ever reached, so a
PATH-stubbed dispatcher records zero invocations — and still writes a
replay result whose `terminal_outcome` is `not_applicable` and whose
per-stage records carry `{applicable: false, reason}` copied **verbatim**
from the package. An absent result for such a scenario is itself a named
failure, never an accepted absence.

Before any dispatch, `check_consistency()` enforces REQ-F-014's read-only
assertion over I-04, regardless of whether the package turns out
applicable: a package with any `prelude.D0X.applicable: true` must carry a
non-empty `replay_reference` resolving to a bundle whose
`scenario_binding.scenario_id` equals the package's `scenario_id`; a
package whose prelude stages are all `applicable: false` must not carry
`replay_reference`. Either violation is rejected naming the package and the
offending field. No I-04 file is edited to satisfy this check
(REQ-NF-006(c)).

### `terminal_outcome` and the I-07 seam (REQ-F-017)

`terminal_outcome` is exactly one value of the closed, twelve-value set
`i06-schema.yaml` declares: `complete`, `not_applicable`,
`unresolved_gate`, `replay_desync`, `live_interaction_reached`,
`unattributed_artifact`, `bundle_bulk_disclosure`, `resource_limit`,
`error`, `cancellation`, `worker_failure`, `timeout`. Any value other than
`complete`/`not_applicable` retains partial evidence, sets
`publication_eligible: false`, and carries a non-empty
`ineligibility_reasons[]`; a result pairing a stop outcome with
`publication_eligible: true` is rejected.

Four values are **I-06-local diagnostics** that I-07's own stop vocabulary
does not contain — `replay_desync`, `live_interaction_reached`,
`unattributed_artifact`, `bundle_bulk_disclosure`. F07 does not widen
E40-F08's I-07 enum with these; instead every stop outcome carries a
required `i07_stop_mapping` field naming the I-07 bucket E40-F08
propagates:

| `terminal_outcome` | `i07_stop_mapping` |
|---|---|
| `replay_desync` | `unresolved_gate` |
| `live_interaction_reached` | `error` |
| `unattributed_artifact` | `error` |
| `bundle_bulk_disclosure` | `error` |
| every other stop value | itself |

`terminal_outcome` keeps the specific F07 cause, verbatim, both in that
field and in `ineligibility_reasons[]`; `i07_stop_mapping` is only the
propagation instruction. A stop outcome missing `i07_stop_mapping`, or
naming a value outside I-07's own stop vocabulary, is rejected.

### Tier 2 guard invocation sequence

Every guard below is offline once fixtures are present — zero provider
calls, byte-identical verdicts across repeated runs at an unchanged bundle
and package (REQ-NF-004). `bench/scripts/run-prelude.sh` is the **only**
provider-calling path in this feature, and no test in it exercises that
path — every test drives a PATH-stubbed dispatcher instead.

```bash
# 1. Before any dispatch, once per candidate package: REQ-F-014's read-only
#    consistency assertion runs inside run-prelude.sh itself (no separate
#    invocation needed) whenever --package is supplied.

# 2. Dispatch boundary, immediately before EVERY scored dispatch, against
#    the LIVE in-flight roots (REQ-F-012) -- E40-F08's dispatch loop calls
#    this, not run-prelude.sh itself:
bench/scripts/verify-replay-isolation.sh \
  <bundle_path> <live_fixture_checkout> <live_scratch_project>

# 3. The scored dispatch itself (the only provider-calling path):
bench/scripts/run-prelude.sh --package <package.yaml> \
  --result-out <result_path> --artifact-root <scratch_shark_project>

# 4. Observational binding gate, over the retained transcript from step 3
#    (REQ-F-005) -- proves the isolation independently of step 3's own
#    --disallowedTools argv, which is belt-and-braces only:
bench/scripts/verify-replay-isolation.sh <retained_transcript_path>

# 5. Lineage reconciliation against the resolver's own consumption ledger
#    (REQ-F-009):
bench/scripts/verify-replay-result.sh <result_path> <bundle_path>

# 6. Full Document A/B field-shape, vocabulary, and eligibility validation
#    against static fixtures (REQ-F-001/002/010/011/013/017/019) -- runs
#    under `make test`, not here, the same split TC-030/TC-042 already
#    established for I-04/I-05:
go test ./tests/contracts/... -run TestTC052_I06ProductDesignReplayContract

# 7. Run the full bench self-test suite (TC-053 through TC-059, alongside
#    every earlier tier's cases):
bench/scripts/tests/run-all.sh
```

`replay-answer.sh` is invoked indirectly, inside step 3's dispatched
session, only through the preamble's routing instruction — no test in this
feature calls it as a substitute for a live dispatch except tc054, which
drives it directly as the resolver itself (there is no caller above it to
substitute).
