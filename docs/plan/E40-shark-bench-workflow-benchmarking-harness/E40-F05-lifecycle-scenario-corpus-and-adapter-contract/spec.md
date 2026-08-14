---
feature_key: E40-F05-lifecycle-scenario-corpus-and-adapter-contract
epic_key: E40
title: Lifecycle scenario corpus and adapter contract — combined requirements and architecture
date: 2026-08-11
---

# E40-F05 Specification: Lifecycle scenario corpus and adapter contract

Business context is not restated here. See epic PRD
[epic.md](../epic.md) §"Success gates" (G8) and §"Feature breakdown" (E40-F05),
and [feature.md](feature.md) for this feature's outcome, scope, and acceptance
boundary. System-level decisions are in [architecture.md](../architecture.md) —
this spec implements the shape already fixed by
[Lifecycle scenario package contract](../architecture.md#lifecycle-scenario-package-contract)
and the I-04 row of [E40-interaction-map.md](../E40-interaction-map.md).

Capability reuse is settled by the validated
[research report](research-report.md) Capability map. In summary: I-01's
admission and versioning *principles* are extended, its Go/golangci-lint fields
are not (Capability map row 1); the Go fixture and its ledger tooling are reused
unmodified as the compatibility adapter (row 2); the execution-adapter borrows
the `internal/runner.AgentDispatcher` *pattern only*, not its code (row 3); the
fixture-repo submodule convention is reused for the Python fixture (row 4). This
feature re-implements none of those.

---

## Requirements

### Functional requirements

| ID | Requirement | Traces to |
|---|---|---|
| REQ-F-001 | I-04 MUST be its own schema-versioned format, independent of `bench/corpus/corpus.yaml`. It comprises an index (`bench/scenarios/scenarios.yaml`) declaring `schema_version`, the registered fixtures, the registered adapters, and the scenario list; plus one `package.yaml` per scenario carrying the full field inventory in REQ-F-002 through REQ-F-009. No field of the I-01 Go manifest (`fixture.toolchain.go_version`, `golangci_lint_version`, `p2p_sets`, `reference_patch_path`) is inherited by name. | architecture Lifecycle scenario package contract; feature.md Contracts (consumes I-01 principles only); research Decision 1 |
| REQ-F-002 | Every package MUST carry a stable identity block: `scenario_id` (unique, lowercase-kebab), `scenario_version` (integer, incremented on any content change), and `entity_family` (exactly one of `feature`, `bug`, `change_card`, `tech_debt`). Loading the same package twice MUST yield byte-identical identity, fixture identity, stage matrix, and resolved input reference paths. | feature.md Acceptance boundary 2; architecture I-04 field list |
| REQ-F-003 | Every package MUST carry a `stage_matrix` with two parts. `stage_matrix.prelude` MUST declare an explicit boolean for each of the five product-design prelude stages `D01`–`D05`, with a `reason` string required on every `false`. `stage_matrix.lifecycle` MUST declare `mode: all_dispatched` and `evidence_required: true` — a declarative rule resolved at run time against the variant workflow bundle, never an enumerated status list. | feature.md Acceptance boundary 4; architecture Lifecycle scenario package contract; ADR-F05-02 |
| REQ-F-004 | The stage matrix MUST satisfy the family invariant: `entity_family: feature` requires all five prelude stages `true`; `bug`, `change_card`, and `tech_debt` require all five `false` with a reason. A package violating the invariant MUST be rejected naming the offending family and stage. | feature.md Acceptance boundary 4; epic G8; UAT-08 |
| REQ-F-005 | Every package MUST carry a `fixture` block (`fixture_id`, `submodule_path`, `base_sha`) and an `adapter` block (`name`, `version`). `fixture_id` and `adapter.name` MUST resolve to entries registered in `scenarios.yaml`. The four seed scenarios MUST name the controlled Python fixture; the existing Go fixture MUST remain registered as the `go` compatibility adapter with `bench/fixture-repo` unchanged. | feature.md Scope 3; research Decisions 3 and 6 |
| REQ-F-006 | The execution adapter MUST be an executable contract, not a library: `bench/adapters/<name>/adapter.sh <capability> [args]`, declared by `bench/adapters/<name>/adapter.yaml` (`name`, `version`). The capability vocabulary is a **closed set of six verbs** — `identity`, `inject-tests`, `test`, `lint`, `build`, `format-check` — each emitting one JSON document on stdout in the shape fixed in [Adapter capability contract](#adapter-capability-contract). Adding a verb requires an I-04 `schema_version` bump. | feature.md Scope (adapter boundary); architecture "The adapter owns language-specific commands"; research Decision 2 |
| REQ-F-007 | No generic lifecycle, evidence, or evaluation component may branch on fixture language, package manager, or toolchain. Every language-specific command MUST be reached through `adapter.sh`. Test identities emitted by `test` MUST already be normalized to `<module-or-package>::<test-name>`, so no consumer performs language-aware identity parsing. | feature.md Acceptance boundary 3; epic G8; ADR-F05-03 |
| REQ-F-008 | Toolchain identity MUST be produced by the adapter's `identity` capability as an **ordered list of opaque key/value pairs**, pinned into the package's `toolchain_identity` field at admission. Consumers MUST treat the list as an opaque ordered sequence suitable for equality comparison and hashing, and MUST NOT read individual keys by name. | architecture I-04 field list ("toolchain identity"); X-12 comparison identity (owned by E40-F09); research finding 6 |
| REQ-F-009 | Every package MUST separate `input.agent_visible` (the initial issue-style input the worker sees) from `evaluator_only` references (reference solution, held-back oracle tests, judge answer keys). Evaluator-only material MUST live under the package's `evaluator/` subtree and MUST NOT be reachable from `input.agent_visible`. Feature-family packages MUST additionally carry a `replay_reference` pointing at the versioned response bundle E40-F07 consumes; the reference is an opaque path, and its interior shape is I-06's, not I-04's. | feature.md Scope 4; architecture Stage evidence and isolation contract; ADR-F05-05 |
| REQ-F-010 | Every package MUST carry a `final_predicate` naming exactly one kind from the closed per-family vocabulary in [Final predicate vocabulary](#final-predicate-vocabulary), with its required operands present: `f2p_p2p` (bug), `acceptance_tests` (change_card), `p2p_plus_rule_drop` (tech_debt), `child_oracles_union` (feature). Each kind MUST be evaluable to a boolean solely from the `test` and `lint` capability outputs, with no human judgement and no reading of workflow status. | shark-bench-design.md §3 per-entity measurement matrix; epic G13; feature.md Scope 5 |
| REQ-F-011 | Every package MUST carry a `resource_policy` with `max_cost_usd`, `max_wall_clock_seconds`, and `max_generated_tasks`, each present and **strictly positive**. A missing or non-positive ceiling MUST be rejected naming the field. | architecture I-04 field list; epic G12 |
| REQ-F-012 | An admission gate MUST evaluate each candidate by execution against a fresh checkout of its fixture at `base_sha`: (a) the fixture checks out and the adapter's `build` capability succeeds — the "runnable base fixture" check; (b) the predicate's `p2p_selection` resolves to a non-empty entry set and every entry is `pass` at base; (c) the stage matrix satisfies REQ-F-004; (d) the final predicate is **false at base**; (e) the final predicate is **true after the evaluator-only reference solution is applied**; (f) the resource policy satisfies REQ-F-011. A candidate failing any check MUST be rejected with that specific check named and MUST NOT enter the admitted set. | feature.md Scope 5 and Acceptance boundary 1; epic G8; UAT-08 |
| REQ-F-013 | Each package MUST carry an `admission` block recording `status: admitted`, the reproducible `base_outcome` and `reference_outcome` from checks (c) and (d), and the `toolchain_identity` under which they were observed. Re-running the gate on a clean checkout at the same fixture SHA and toolchain MUST produce an identical verdict for every candidate. | architecture I-04 field list ("admission status with reproducible base and reference outcomes"); feature.md Acceptance boundary 2 |
| REQ-F-014 | The admitted set MUST contain exactly one seed scenario per lifecycle family — all four on the controlled Python fixture, all four passing REQ-F-012 — with these `scenario_id`s: `py-bug-due-date-boundary`, `py-change-priority-scale`, `py-techdebt-consolidate-validation`, `py-feature-recurring-tasks`. Their subjects are fixed in [Seed scenarios](#seed-scenarios). | feature.md Scope 2; epic G8; UAT-08 |
| REQ-F-015 | A schema validator MUST reject a malformed package with the **failing field named** — unknown `entity_family`, unregistered `fixture_id` or `adapter.name`, missing or non-positive resource ceiling, prelude stage without an explicit boolean, `false` prelude stage without a reason, family-invariant violation, unknown `final_predicate.kind`, a predicate kind not permitted for the declared family, a missing operand path, or an `input.agent_visible` path resolving inside `evaluator/`. | feature.md Acceptance boundary 1; UAT-08 |
| REQ-F-016 | The controlled Python fixture MUST be a self-contained Python task-manager repository held at a pinned base commit, carrying its own dependency/config manifest (`pyproject.toml`), the lint configuration the adapter's `lint` capability targets, the formatter configuration `format-check` targets, and a real test suite discoverable by the `test` capability. Its working tree MUST contain fixture code only — no scenario packages, no evaluator-only material, no shark source. Its full test suite MUST be green at `base_sha` — every entry `pass` — with exactly one named, permanent exception: `tests.test_manager::test_recurring_task_generates_next_occurrence`, committed `@pytest.mark.skip`, which is the `py-feature-recurring-tasks` seed's own proof that the recurring-task capability is genuinely absent at base (mirroring ADR-F05-09's documentation of the Go fixture's permanently-failing regression probe, the same pattern applied to a `skip` outcome instead of a `fail`). Any *other* skipped or failing entry MUST still reject the fixture as not green. This is what makes the absolute P2P semantics of REQ-F-017 sound. | E40-F01 REQ-F-001 precedent; feature.md Scope 3; ADR-F05-09, ADR-F05-11 |
| REQ-F-017 | A `p2p_selection` operand MUST be an object `{include: [fixture-relative paths], exclude_test_ids: [ids]}`, resolved by passing it to the adapter's `test` capability. Its clause in a final predicate is **absolute**: every entry the selection resolves to MUST be `pass`. No base ledger is required or committed for I-04, because REQ-F-016 requires the fixture suite green at `base_sha` and admission check (b) verifies it per scenario. | ADR-F05-10; feature.md Acceptance boundary 1 |

### Non-functional requirements

| ID | Requirement | Traces to |
|---|---|---|
| REQ-NF-001 | This feature changes no shark product code. It adds no file under `internal/` or `cmd/`, no schema, no migration, and no service. Its only addition inside the shark Go module is one test-only contract validator, mirroring E40-F01's REQ-NF-001 posture. | architecture Delivery boundaries ("Extend the bench corpus"); research Scope |
| REQ-NF-002 | With the Python fixture and scenario packages committed, `make fmt && make lint && make test` at the repository root MUST stay green, and `go list ./...` MUST list no fixture or evaluator-only package. Python sources are inert to the Go toolchain; `scripts/lint-positional-selection.sh` greps only `internal` and `cmd`, so `bench/` stays outside its reach and any future widening MUST preserve this. | E40-F01 REQ-NF-002 precedent (`AC-011`) |
| REQ-NF-003 | The I-04 schema validator MUST read in-repo artifacts only — `bench/scenarios/**` and `bench/adapters/*/adapter.yaml` — and MUST NOT require any populated submodule. `.github/workflows/ci.yml` uses `actions/checkout@v4`, which does not initialise submodules, so a validator touching a fixture tree would fail in CI and force a workflow change this feature does not need. | ADR-F01-05 precedent; `.github/workflows/ci.yml` |
| REQ-NF-004 | The admission gate MUST complete offline once the fixtures and their dependency caches are present, and MUST produce byte-identical verdicts across repeated runs at an unchanged fixture SHA and toolchain identity. | E40-F01 REQ-NF-005/REQ-F-012 precedent; epic G7 reproducibility principle |
| REQ-NF-005 | Scenario tooling MUST never touch the live shark database, `.sharkconfig.json`, or the live repository working tree, and MUST never invoke shark project-initialisation commands. All work happens inside a caller-supplied destination directory. | E40-F01 REQ-NF-003; `bench/scripts/checkout-fixture.sh` header; repo config-guardrail hook |
| REQ-NF-006 | `bench/scripts/checkout-fixture.sh`'s frozen `<base_sha> <dest_dir>` interface MUST NOT change — its header states the interface is fixed once `admit.sh`, `build-ledgers.sh`, and E40-F02 depend on it. Multi-fixture checkout is added as a **sibling script**, not as a new argument or altered contract. | `bench/scripts/checkout-fixture.sh:1-18`; ADR-F05-06 |

### Acceptance criteria

| ID | Criterion |
|---|---|
| AC-001 | TC-030 asserts, for every package in `bench/scenarios/scenarios.yaml`, that the REQ-F-002/005/009/010/011 field inventory is present and well-typed, that every referenced input, evaluator, and predicate-operand path exists, that `fixture_id` and `adapter.name` resolve to registered entries, and that the index's `schema_version` is the version the validator supports. |
| AC-002 | TC-030 asserts the REQ-F-004 family invariant for all four seeds: the `feature` package declares `D01`–`D05` all `true`; the `bug`, `change_card`, and `tech_debt` packages declare all five `false`, each with a non-empty reason. |
| AC-003 | TC-030 asserts `stage_matrix.lifecycle` is exactly `mode: all_dispatched` with `evidence_required: true` and carries no enumerated status list — a package embedding workflow status names is rejected. |
| AC-004 | TC-030 asserts every `input.agent_visible` path resolves outside the package's `evaluator/` subtree, and that every feature-family package carries a `replay_reference` while no other family does. |
| AC-005 | For each malformed-package case enumerated in REQ-F-015, the validator exits non-zero and its message names the failing field. Verified by table-driven fixtures under `tests/contracts/testdata/`. |
| AC-006 | `bench/scripts/admit-scenario.sh` admits all four seed packages named in REQ-F-014 — `py-bug-due-date-boundary`, `py-change-priority-scale`, `py-techdebt-consolidate-validation`, `py-feature-recurring-tasks` — and its output reports one admitted scenario per family with the `base_outcome` false and `reference_outcome` true recorded for each. |
| AC-007 | A candidate whose fixture fails the adapter's `build` capability at `base_sha` is rejected naming check (a); a candidate whose `p2p_selection` resolves to an empty set or contains a failing entry at base is rejected naming check (b); a candidate whose final predicate is already true at base is rejected naming check (d); a candidate whose predicate stays false after the reference solution is rejected naming check (e); a candidate with `max_cost_usd: 0` is rejected naming check (f) and the field. |
| AC-008 | Re-running `admit-scenario.sh` on a clean checkout at the same fixture SHA and toolchain produces an identical verdict and identical `base_outcome`/`reference_outcome` values for every candidate. |
| AC-009 | Loading the same package twice yields byte-identical `scenario_id`, `scenario_version`, `fixture.base_sha`, `stage_matrix`, `toolchain_identity`, and resolved input paths. |
| AC-010 | Both adapters implement all six capabilities. For each capability, `bench/adapters/go/adapter.sh` and `bench/adapters/python/adapter.sh` emit JSON validating against the same shape, verified by `bench/scripts/tests/tc031_adapter_conformance_test.sh` running the identical assertion set against both. |
| AC-011 | `adapter.sh test` emits normalized identities: the Go adapter emits `<package import path>::<test name>` and the Python adapter emits `<module path>::<test name>` for the same conceptual entry set, and neither consumer-side script performs language-aware parsing. |
| AC-012 | A grep of every generic scenario, evidence, and admission script for `python`, `pytest`, `pip`, `go test`, `golangci-lint`, and `go build` returns hits only inside `bench/adapters/*/`, proving REQ-F-007 mechanically rather than by convention. |
| AC-013 | `adapter.sh identity` emits an ordered key/value list; the admission gate pins it verbatim into `toolchain_identity`, and re-running under a changed toolchain produces a different list, so E40-F09 can detect the difference by equality alone without reading a named key. |
| AC-014 | Each of the four final-predicate kinds evaluates to a boolean from `test` and `lint` outputs alone: `p2p_plus_rule_drop` is proven by a tech-debt seed whose named lint rule count drops to its threshold, and `child_oracles_union` is proven by a feature seed whose integration set plus declared child oracles are all green under the reference solution. |
| AC-015 | `make fmt && make lint && make test` at the repository root is green with the Python fixture and scenario packages committed, and `go list ./...` lists no fixture or evaluator-only package. |
| AC-016 | TC-030 passes in CI without `git submodule update --init`, and `.github/workflows/ci.yml` is unchanged by this feature. |
| AC-017 | `bench/scripts/checkout-fixture.sh` is byte-unchanged by this feature; multi-fixture checkout is provided by `bench/scripts/checkout-scenario-fixture.sh`, and the existing I-01 callers continue to invoke the original script. |
| AC-018 | With the network disabled and dependency caches warm, the admission gate completes over all four seeds. |
| AC-019 | TC-030 asserts the toolchain identity agrees across its two encodings: each package's top-level `toolchain_identity` equals its `admission.toolchain_identity`, element for element and in order. |
| AC-020 | A fresh clone of `bench/fixture-py` at `base_sha` contains `pyproject.toml` with lint and formatter configuration, a discoverable test suite, and no scenario package or evaluator-only file; the adapter's `test` capability reports every entry `pass` at that SHA, with exactly one named, permanent exception — `tests.test_manager::test_recurring_task_generates_next_occurrence` — which is permitted (and expected) to report `skip` rather than `pass`; it MUST NOT report `fail`, and no *other* entry may report `skip` or `fail` (REQ-F-016). Verified by a bench script, not by TC-030, which must not require a populated submodule. |
| AC-021 | Every predicate evaluation performed by `eval-predicate.sh` reads only `test` and `lint` capability output — no ledger file is read, and none is committed under `bench/scenarios/`. |

### Out of scope for this feature

- Driving any Shark entity through its workflow, claiming, heartbeating, or transitioning (E40-F08).
- Enforcing evaluator-only invisibility at a dispatch boundary, and capturing stage evidence (E40-F06). F05 **declares** the evaluator-only root and the agent-visible input; it does not police the runtime boundary. `bench/scripts/verify-clean-checkout.sh` is corpus.yaml-shaped and is not generalized here.
- Replaying stakeholder or research interactions, and defining the interior of the replay bundle (E40-F07 / I-06).
- Evaluating artifact quality, calibrating a judge, computing comparison identity, or publishing a baseline (E40-F09, E40-F10).
- Decoding provider-usage envelope fields (X-09 / E40-F06). I-04 carries opaque references only.
- LOC prod/test classification as an adapter capability — see ADR-F05-04.
- Epic-family scenarios, deferred by shark-bench-design.md §3 until after lifecycle v2.

---

## Architecture

### Component changes

| File | Change |
|---|---|
| `bench/scenarios/scenarios.yaml` | New. I-04 index: `schema_version`, registered `fixtures` (`fixture_id` → `submodule_path`), registered `adapters` (`name` → `path`, `version`), and the `scenarios` list of package paths. Plays the role `bench/corpus/corpus.yaml` plays for I-01, without inheriting its fields. |
| `bench/scenarios/packages/<scenario_id>/package.yaml` | New, one per scenario. The I-04 package: identity, family, `stage_matrix`, `fixture`, `adapter`, `toolchain_identity`, `input`, `replay_reference` (feature only), `evaluator_only`, `final_predicate`, `resource_policy`, `admission`. |
| `bench/scenarios/packages/<scenario_id>/input/prompt.md` | New. The agent-visible initial input for the scenario root. |
| `bench/scenarios/packages/<scenario_id>/evaluator/` | New. Evaluator-only subtree: `reference.patch`, held-back oracle test files, and any judge answer key. Referenced by `evaluator_only`, never by `input.agent_visible`. |
| `bench/adapters/go/adapter.yaml`, `bench/adapters/go/adapter.sh` | New. The compatibility adapter. Its `test`/`lint` capabilities **delegate to `bench/scripts/build-ledgers.sh` and `bench/scripts/diff-ledgers.sh` unmodified** and reshape their output into the capability JSON; `build`/`format-check` invoke the fixture `Makefile` targets. |
| `bench/adapters/python/adapter.yaml`, `bench/adapters/python/adapter.sh` | New. The controlled Python fixture's adapter: pytest for `test`, the fixture's configured linter for `lint`, an import/compile check for `build`, a formatter check for `format-check`, and `inject-tests` placing evaluator-only test files where pytest discovers them. |
| `bench/fixture-py/` + `.gitmodules` | New submodule at a pinned SHA for the controlled Python task-manager fixture (REQ-F-016), following the existing `bench/fixture-repo` convention (research Decision 6). Contents: `pyproject.toml` (dependencies plus lint and formatter configuration), package source, and a test suite green at `base_sha`. **Implementation prerequisite**: the fixture repository must be created and pushed before any SHA can be pinned, exactly as `jwwelbor/shark-bench-fixture` was for I-01 — the first task in this feature's build order, not an assumption (ADR-F05-09). |
| `bench/scripts/checkout-scenario-fixture.sh` | New sibling of `checkout-fixture.sh`. `checkout-scenario-fixture.sh <fixture_id> <base_sha> <dest_dir>` resolves `submodule_path` from `scenarios.yaml` and clones that submodule at the SHA. Generic over fixtures; adds no argument to and changes no byte of `checkout-fixture.sh` (REQ-NF-006). |
| `bench/scripts/admit-scenario.sh` | New. The REQ-F-012 execution gate: checkout, `build`, stage-matrix invariant, predicate-at-base, predicate-after-reference, resource policy. Rejects naming the failing check; writes the reproducible `admission` block. |
| `bench/scripts/eval-predicate.sh` | New. Evaluates one `final_predicate` to a boolean from `test` and `lint` capability output. The single owner of the REQ-F-010 arithmetic, so `admit-scenario.sh` and later E40-F09 invoke it rather than re-deriving the semantics — the same "single named owner" discipline `diff-ledgers.sh` established for I-01. |
| `bench/scripts/verify-fixture-py-base.sh` | New. Runs the adapter's `test` capability with no `--include`/`--exclude-id` filter against `bench/fixture-py` at `base_sha` and asserts every entry is `pass`, with the one named exception AC-020/REQ-F-016 permit (`tests.test_manager::test_recurring_task_generates_next_occurrence`, ADR-F05-11) (AC-020, TC-040). Admission check (b) alone only proves the narrower `p2p_selection` subset is green; this script is the named owner of the broader "fixture is green at base_sha" claim so `admit-scenario.sh` does not silently under-cover it. |
| `bench/scripts/tests/tc031_adapter_conformance_test.sh` | New. Runs one assertion set against both adapters (AC-010, AC-011) and the REQ-F-007 grep (AC-012). Registered in `bench/scripts/tests/run-all.sh`. |
| `bench/scripts/tests/run-all.sh` | Modified. Registers the new bench test cases. |
| `bench/README.md` | Modified. Adds an "I-04 scenario package schema" section and the adapter capability contract, so E40-F06/F07/F08 read the shape instead of re-deriving it — the role its "Manifest schema" section plays for I-01. |
| `tests/contracts/e40_i04_scenario_contract_test.go` | New, **the only Go file this feature adds** (REQ-NF-001). `package contracts`, TC-030, repository-root-relative artifact reading, in-repo artifacts only (REQ-NF-003), following `tests/contracts/e40_i01_corpus_contract_test.go`. |
| `tests/contracts/testdata/e40_i04/` | New. Malformed-package fixtures for AC-005. |

### Data model changes

None. No shark table, column, migration, or `CurrentSchemaVersion` bump. I-04 is
file-backed under `bench/`, consistent with ADR-002 (JSONL/file artifacts are the
only store) and the architecture's "E40 adds no Shark database table."

### API / interface contracts

#### Lifecycle scenario package (I-04)

`package.yaml` field inventory. Every field is required unless marked.

| Field | Type | Contract |
|---|---|---|
| `schema_version` | string | Matches the index; the version TC-030 supports. |
| `scenario_id` | string | Unique lowercase-kebab identity. |
| `scenario_version` | integer | Incremented on any content change (REQ-F-002). |
| `entity_family` | enum | `feature` \| `bug` \| `change_card` \| `tech_debt`. |
| `stage_matrix.prelude.D01`…`.D05` | object | `{applicable: bool, reason: string}`; `reason` required when `applicable: false`. |
| `stage_matrix.lifecycle` | object | `{mode: all_dispatched, evidence_required: true}`. No status enumeration (ADR-F05-02). |
| `fixture` | object | `{fixture_id, submodule_path, base_sha}`; `fixture_id` registered in the index. |
| `adapter` | object | `{name, version}`; `name` registered in the index. |
| `toolchain_identity` | ordered list | `[{key, value}]`, opaque to consumers (REQ-F-008). |
| `input.agent_visible` | path | The issue-style initial input; must resolve outside `evaluator/`. |
| `replay_reference` | path, feature only | Opaque pointer to the I-06 response bundle. Absent for other families. |
| `evaluator_only` | object | `{reference_solution, oracle_tests[], answer_keys[]}` — all under `evaluator/`. |
| `final_predicate` | object | `{kind, …operands}` per the vocabulary below. Every kind carries a `p2p_selection` operand shaped `{include: [fixture-relative paths], exclude_test_ids: [ids]}` (REQ-F-017). |
| `resource_policy` | object | `{max_cost_usd, max_wall_clock_seconds, max_generated_tasks}`, all strictly positive. |
| `admission` | object | `{status, base_outcome, reference_outcome, toolchain_identity}`. Its `toolchain_identity` is the second encoding of the top-level field and MUST equal it; TC-030 asserts the agreement (AC-019), following AC-013's cross-encoding precedent. |

#### Adapter capability contract

`bench/adapters/<name>/adapter.sh <capability> [args]`. Exactly six capabilities;
each writes one JSON document to stdout and uses exit status `0` for "ran
successfully", non-zero for "could not run". A capability whose *subject* failed
(a red test, a lint issue) still exits `0` and reports the failure in JSON — so a
consumer never conflates "the toolchain broke" with "the code is wrong."

| Capability | Arguments | stdout JSON |
|---|---|---|
| `identity` | `--checkout <dir>` | `{adapter, version, toolchain_identity: [{key, value}]}` — ordered, opaque (REQ-F-008). |
| `inject-tests` | `--checkout <dir> --files <path>…` | `{injected: [{source, destination}]}`. Places evaluator-only test files where the toolchain discovers them — the language-specific placement rule lives here and nowhere else. |
| `test` | `--checkout <dir> [--include <fixture-relative path>…] [--exclude-id <id>…] [--only-id <id>…]` | `{entries: [{id, outcome}]}` with `id` already normalized to `<module-or-package>::<test-name>` and `outcome` one of `pass`\|`fail`\|`skip` (REQ-F-007). Subtests are distinct entries under their full name, matching the I-01 ledger convention. `--include`/`--exclude-id` carry a `p2p_selection` (REQ-F-017); `--only-id` names a predicate's own test ids. Selection is expressed in fixture-relative paths and normalized ids, so it stays language-neutral — translating them into package patterns or pytest node selectors is the adapter's job. |
| `lint` | `--checkout <dir>` | `{issues: [{rule, file, text}]}` — a multiset; identity excludes line and column so it is stable under position shifts, reusing ADR-F01-03's reasoning rather than re-deriving it. |
| `build` | `--checkout <dir>` | `{ok: bool, diagnostics: [string]}`. Used by admission check (a). |
| `format-check` | `--checkout <dir>` | `{ok: bool, offending_files: [string]}`. |

#### Final predicate vocabulary

Closed set of four kinds, one permitted per family, each evaluable from `test`
and `lint` output alone (REQ-F-010). Sourced from
[shark-bench-design.md](../shark-bench-design.md) §3 rather than invented here.

| Kind | Family | Operands | True when |
|---|---|---|---|
| `f2p_p2p` | `bug` | `f2p_test_ids[]`, `p2p_selection` | Every `f2p_test_ids` entry is `pass` **and** every entry in the P2P selection is `pass`. The bug seed's F2P set is its author-written repro test. |
| `acceptance_tests` | `change_card` | `acceptance_test_ids[]`, `p2p_selection` | Every acceptance test is `pass` and every P2P selection entry is `pass`. This is the design doc's "machine-checkable acceptance predicate, required at admission." |
| `p2p_plus_rule_drop` | `tech_debt` | `p2p_selection`, `rule`, `max_remaining` | Every P2P selection entry is `pass` **and** the count of `lint` issues whose `rule` matches is `<= max_remaining`. Expresses "the debt is gone" from the existing `lint` capability — no complexity tool is introduced (ADR-F05-04). |
| `child_oracles_union` | `feature` | `integration_test_ids[]`, `child_oracles[]`, `p2p_selection` | Every integration test is `pass`, every declared child oracle evaluates true, and every P2P entry is `pass`. |

Each kind's P2P clause is **absolute**, not base-relative (REQ-F-017, ADR-F05-10):
every entry the `p2p_selection` resolves to must be `pass`. `p2p_plus_rule_drop`'s
lint clause is likewise absolute via `max_remaining`. This is what makes
REQ-F-010's "evaluable from `test` and `lint` output alone" literally true — no
predicate consults a stored base ledger.

### Seed scenarios

The four REQ-F-014 seeds, all on `bench/fixture-py` at its pinned `base_sha`,
packaged at `bench/scenarios/packages/<scenario_id>/`.

| `scenario_id` | Family | Subject on the Python task-manager fixture | Predicate kind |
|---|---|---|---|
| `py-bug-due-date-boundary` | `bug` | The overdue check compares strictly (`<`) against the current time, so a task due at exactly the boundary is never reported overdue. The evaluator-only oracle is an author-written repro test asserting the boundary case. | `f2p_p2p` |
| `py-change-priority-scale` | `change_card` | Change the priority scale from three string levels to an integer 1–5 scale, converting existing stored records. Evaluator-only acceptance tests cover the new scale, the conversion of each old level, and rejection of out-of-range values. | `acceptance_tests` |
| `py-techdebt-consolidate-validation` | `tech_debt` | The same task-field validation is duplicated inline across three modules; a named lint rule flags each copy. Consolidating them into one validator drops that rule's issue count to `max_remaining: 0` while behavior is preserved. | `p2p_plus_rule_drop` |
| `py-feature-recurring-tasks` | `feature` | Add recurring-task scheduling: a recurrence rule on a task, generation of the next occurrence on completion, and listing of upcoming occurrences. Evaluator-only integration tests cover the end-to-end path; `child_oracles` cover each declared sub-capability. | `child_oracles_union` |

`py-feature-recurring-tasks` is the only seed carrying a `replay_reference`
(REQ-F-009), since only the feature family runs the D01–D05 prelude.

### Key technical decisions

**ADR-F05-01 — I-04 is a new schema, not an extension of `corpus.yaml`.** I-01's
`fixture.toolchain` block (`go_version`, `golangci_lint_version`, `goos`,
`goarch`, `golangci_config_sha256`) and its `p2p_sets` (Go package patterns and
`-run` selectors) have no Python analog, and its validator decodes exactly those
Go-typed fields. A literal reuse attempt breaks on its own toolchain semantics
before reaching any Python-specific gap (research finding 2). What carries over
is the *form* — a schema-versioned manifest, an execution-based reproducible
admission gate, held-back oracles stored outside the fixture, and a "single named
owner" script for each piece of arithmetic. Rejected alternative: retrofitting
`corpus.yaml` with a language discriminator, which would make every I-01 consumer
handle a Python branch it has no use for.

**ADR-F05-02 — The lifecycle half of the stage matrix is a rule, not a list.**
ADR-003 makes a config variant a **workflow YAML bundle**, so the dispatched
status set differs per variant while a scenario package is variant-independent by
construction. An enumerated status list would therefore be wrong for every
variant but one — not merely redundant with the workflow definition. It would
also make the package a second, drifting copy of workflow routing, which ADR-006
forbids. The prelude half *is* enumerated because those five stages are the
benchmark's own (E40-F07 drives them through the Rider action), not the workflow
engine's. One consequence binds E40-F06: with the lifecycle half resolved at run
time, stage completeness is **one snapshot per dispatch**, and a "named
missing-stage failure" is only detectable for the prelude half. Rejected
alternative: a per-variant matrix keyed by bundle identity, which would couple
every scenario package to the variant catalogue F09 owns.

**ADR-F05-03 — The adapter is an executable shell contract with a closed verb
set.** Research Decision 2 fixes the *shape* on `internal/runner.AgentDispatcher`
— one small interface, one implementation per concrete tool, selection driven by
data rather than inline branching. It does not fix the *medium*, and all of
`bench/` is bash. A Go adapter binary under `bench/` would also fight
REQ-NF-001's "one test-only Go file" posture. The verb set is closed and the JSON
shapes are fixed here because I-04's real contract surface is what F06, F07, and
F08 call — leaving the vocabulary open would let three features each invent their
own. Rejected alternative: a per-capability config map of raw command strings in
`package.yaml`, which would put Python commands back inside the scenario data the
feature's Outcome statement excludes.

**ADR-F05-04 — LOC prod/test classification is deliberately not an adapter
capability.** It is language-specific, so if a lifecycle consumer needed it, it
would belong behind the adapter. No lifecycle v2 consumer does: I-05's snapshot
field list, I-07's run record, and I-08's evaluation record carry no LOC field,
and epic G4's LOC metric is a **Phase 1** gate owned by E40-F02 on the Go-only
path. Adding a seventh verb now would be speculative (Rule 2). The schema is
versioned; if E40-F09 later needs LOC, it arrives as an adapter capability
addition with a `schema_version` bump, which is exactly what versioning is for.

**ADR-F05-05 — Evaluator and usage references stay opaque.** E27-F15 (X-09) is
confirmed unmerged on `main` — its commit lives only on
`E27-F15-cross-session-usage-tracking`, and the on-`main` dispatcher has no
`modelUsage`/`num_turns`/`duration_api_ms`/`total_cost_usd` decode (research
finding 4). I-04 therefore names evaluator artifacts and the replay bundle by
path only and types nothing about decoded provider-usage fields. That decode
remains E40-F06's job under X-09. The same opacity rule applies to
`toolchain_identity`: an ordered key/value list is comparable and hashable by
E40-F09 without any consumer knowing Python's or Go's key names.

**ADR-F05-06 — Multi-fixture checkout is a sibling script, not a changed
interface.** `bench/scripts/checkout-fixture.sh`'s header freezes its
`<base_sha> <dest_dir>` interface once `admit.sh`, `build-ledgers.sh`, and
E40-F02 depend on it. Adding a fixture argument would break three committed
callers to serve one new one. `checkout-scenario-fixture.sh` resolves
`submodule_path` from `scenarios.yaml` and leaves the original byte-unchanged, so
the Go family keeps working exactly as-is — the "wrap, don't rewrite" posture
research Decision 3 fixed for the whole Go tooling set.

**ADR-F05-07 — Schema validation and execution admission are separate owners.**
`tests/contracts/e40_i04_scenario_contract_test.go` (TC-030) validates structure
from in-repo artifacts only and never requires a populated submodule, because
CI's `actions/checkout@v4` does not initialise submodules; a validator touching a
fixture tree would fail in CI and force a workflow change this feature does not
need. Execution-based admission lives in `bench/scripts/admit-scenario.sh`, which
does require the fixtures. This mirrors E40-F01's TC-001/`admit.sh` split
(ADR-F01-05) rather than inventing a second convention.

**ADR-F05-08 — B051 is resolved infrastructure, not a tracked precondition.** The
parent epic research report's caution predates this feature and predates the
merge. `internal/runner/controller.go:517` groups `ActionCheckOrResume` with
`ActionSpawnAgent` in the dispatch switch on `main`, agreeing with B051's
`completed` status, so the tech-debt seed is admissible with no outstanding
dependency (research finding 3, Decision 5).

**ADR-F05-09 — The Python fixture is a new external repository, pinned as a
submodule.** I-01 established the convention (`.gitmodules` →
`jwwelbor/shark-bench-fixture`, pinned SHA, cloned per run), and research
Decision 6 carries it forward. Two consequences are binding. First, the fixture
repository must exist and be pushed before a SHA can be pinned, so creating and
seeding it is the **first task** in this feature's build order rather than an
assumption the schema work can proceed past. Second, REQ-F-016's "green at
`base_sha`" requirement is a deliberate divergence from the Go fixture, which
commits `TestStock_PermanentlyFailingRegressionProbe` — a permanently red test
that exists to exercise I-01's rejection branch (b). The Python fixture needs no
such probe because I-04's rejection branches are exercised by malformed packages
(AC-005) and by admission check fixtures (AC-007), not by a poisoned suite. A
green base is what makes ADR-F05-10's absolute semantics available — "green"
carrying exactly one further, differently-motivated named exception at the
individual-test level, unrelated to rejection-branch testing; see ADR-F05-11.

**ADR-F05-10 — P2P truth is absolute, not base-relative, so I-04 commits no
ledgers.** I-01 needed base-SHA test and lint ledgers because F02 counts
*regressions* across an external Go suite that is deliberately not green at base.
I-04's fixture is authored for this benchmark and required green at `base_sha`
(REQ-F-016), verified per scenario by admission check (b), so "no entry regressed
from `pass` to `fail`" and "every entry is `pass`" are equivalent — and only the
second is evaluable from a single `test` invocation. Choosing the second keeps
REQ-F-010's "evaluable from `test` and `lint` output alone" literally true,
removes a whole class of ledger-staleness bug from lifecycle v2, and drops the
ledger generation, storage, and toolchain-guard machinery I-04 would otherwise
have to duplicate. Rejected alternative: per-fixture-SHA ledgers mirroring I-01,
which buys nothing on a controlled green fixture and would make every fixture
commit a two-file update with the drift risk `corpus.yaml`'s own header warns
about.

The green-at-base invariant this ADR rests on is a property of a repository that
does not exist yet, pinned by a SHA chosen later, so name its enforcement
explicitly: **admission check (b) is the guard**. If the fixture ever acquires a
red test — a future contributor mirroring the Go fixture's permanently-failing
probe, or a dependency upgrade breaking a test at the pinned SHA — every absolute
P2P clause would otherwise become quietly unsatisfiable. Check (b) resolves each
scenario's `p2p_selection` at base and rejects a non-empty failing entry set with
that check named, converting a silent unsatisfiable predicate into a loud
admission rejection. Any future change to check (b) must preserve this coupling.

**ADR-F05-11 — `tests.test_manager::test_recurring_task_generates_next_occurrence`
is a permanently-`skip`ped, named exception to REQ-F-016's "green at `base_sha`",
committed at the fixture's very first commit.** The `py-feature-recurring-tasks`
seed's `child_oracles_union` predicate (REQ-F-014, seed 4 of 4) requires the
recurring-task capability to be genuinely **absent** at base (check (d),
REQ-F-012) and present only after its evaluator-only reference patch is applied
(check (e)) — the capability's absence at base is this seed's entire subject, not
an incidental gap. `tests/test_manager.py`'s `@pytest.mark.skip(reason=
"recurring-task scheduling is not implemented in the fixture's base state")`
placeholder is the fixture's own committed proof of that absence, and all four
seed packages' `final_predicate.p2p_selection.exclude_test_ids` already exclude
this same id for the identical reason: a `skip` is not `pass`, so leaving it
inside any package's own absolute P2P selection would make that selection
unsatisfiable at base by construction, not by regression.

This is a **different** exception from the one ADR-F05-09 declines to add: that
ADR is about *not* needing a Go-style `TestStock_PermanentlyFailingRegressionProbe`
to exercise I-04's own rejection branches (a fixture-authoring choice). This ADR
is about one specific seed's predicate design requiring one specific capability's
absence to be provable at base at all — a property of the corpus, not of I-04's
admission machinery. It mirrors ADR-F05-09's Go-fixture precedent in shape (a
named, permanent, non-transient test outcome carved out of an otherwise-absolute
"every entry passes" claim, documented at the spec level rather than left as an
undocumented implementation detail) while differing in kind (`skip`, not `fail` —
the Go probe proves I-01's rejection path handles a failing entry; this
placeholder proves a capability is absent, which `skip` states more precisely
than a contrived `fail` would).

`bench/scripts/verify-fixture-py-base.sh` is the sole enforcement point
(REQ-F-016, AC-020, TC-040): its `ALLOWED_SKIP_ID` constant names exactly this
one id as the only outcome permitted to be `skip` rather than `pass`; any other
`skip`, or any `fail`, still rejects the fixture as not green — including a
regression that turns this same test into an unexpected `fail`, or a second,
undocumented skip appearing anywhere else in the suite. This is what keeps
AC-020 non-vacuous rather than opening a general "skips are fine" reading.
Rejected alternative: implementing a minimal stub of the recurring-task
capability so `test_manager.py`'s placeholder passes unmodified at base (UAT
Finding 4's resolution path 1) — rejected not because it would flip check (d)
(a stub sufficient for this one shallow unit test would not necessarily satisfy
`evaluator/test_recurring.py`'s stricter child-oracle assertions, so check (d)
could remain technically satisfied), but because it would replace a structural,
self-evident absence — the capability simply does not exist in
`taskmanager/manager.py`/`models.py` at base — with an implementation detail
that happens to be shallow enough not to satisfy the oracle, a property that
depends on how narrowly the stub is written rather than on the capability's
actual presence. That is exactly the fragile, engineered-to-dodge-the-oracle
invariant the committed `skip` marker's structural guarantee avoids, and it
undermines the one seed whose entire design depends on the capability's
absence being real, not stubbed-around.

### Integration with existing code

Nothing under `internal/` or `cmd/` is called, imported, or extended. The
integration surfaces are conventions and executables, not Go call paths:

- **Wrapped, unmodified Go tooling** — `bench/adapters/go/adapter.sh` shells out
  to `bench/scripts/build-ledgers.sh` and `bench/scripts/diff-ledgers.sh` and
  reshapes their output. Neither script is edited, so the I-01 ledger and
  toolchain-guard semantics stay under their existing single owner
  (research Decision 3).
- **Fixture checkout** — `bench/scripts/checkout-scenario-fixture.sh` follows the
  structure of `bench/scripts/checkout-fixture.sh` (clone submodule, checkout
  SHA, verify, never touch the live tree) without altering it (ADR-F05-06).
- **Contract-test convention** — `tests/contracts/e40_i04_scenario_contract_test.go`
  joins `package contracts` and follows the repository-root-relative
  artifact-reading helper style of `tests/contracts/e40_i01_corpus_contract_test.go`
  and `tests/contracts/e39_interactions_test.go`.
- **Submodule convention** — `.gitmodules` gains `bench/fixture-py` alongside
  `bench/fixture-repo`, same pinned-SHA pattern, no vendored Python source.
- **Adapter pattern, not adapter code** — `internal/runner.AgentDispatcher`
  (`Name`, `BuildCommand`, `Dispatch`) is the shape borrowed; none of
  `internal/runner`'s code is imported or copied.
- **Scratch-environment discipline** — scenario tooling creates only temporary
  fixture checkouts and never invokes shark project initialisation, keeping
  `scripts/shark-scratch-env.sh` E40-F02's concern and the config-guardrail hook
  satisfied (REQ-NF-005).

---

## Cross-feature interactions

### Produces: I-04 — Lifecycle scenario package

| Property | Contract |
|---|---|
| Consumers | E40-F06 Stage evidence and evaluator isolation; E40-F07 Replayable product-design prelude; E40-F08 Canonical multi-entity lifecycle runner |
| Shape source | [Lifecycle scenario package contract](../architecture.md#lifecycle-scenario-package-contract) |
| Payload | Versioned scenario identity, family, stage matrix, fixture and adapter, visible input, replay and evaluator references, resource policy, final predicate, and admission result |
| Style | File artifact |
| Shared contract test | `tests/contracts/e40_i04_scenario_contract_test.go#TC-030` |
| Consumer reads | `bench/scenarios/scenarios.yaml`, `bench/scenarios/packages/*/package.yaml`, and the I-04 schema and adapter capability sections of `bench/README.md` |
| Consumer invokes | `bench/adapters/<name>/adapter.sh <capability>`, `bench/scripts/checkout-scenario-fixture.sh`, and `bench/scripts/eval-predicate.sh`, rather than re-deriving language commands or predicate arithmetic |
| Consumer split | E40-F07 reads `stage_matrix.prelude` and `replay_reference`; E40-F06 reads `evaluator_only`, `toolchain_identity`, and both stage-matrix halves; E40-F08 reads `stage_matrix.lifecycle`, `adapter`, `fixture`, and `resource_policy` |
| Test scope | TC-030 reads in-repo scenario artifacts only and requires no populated submodule, so `.github/workflows/ci.yml` is unchanged (ADR-F05-07) |
| Gate mode | `contract-only`, staged by [the interaction map](../E40-interaction-map.md#i-04-staged-edge) — F05's producer necessarily runs before its consumers are decomposed |
| Activation owner | E40-F06 (its slice); E40-F07 (its slice); E40-F08 (its slice) — each closes its own consumption independently at its own UAT |
| Closure key | E40-F06 / E40-F07 / E40-F08, respectively |
| Counterpart status | Read live from Shark at review/UAT time; not copied here as a fact that would go stale |
| Review basis | This spec.md and the interaction map row, present together at F05 task_review |
| Demonstrability disposition | `pending-integration` until each consumer's live wiring closes |
| Closure owner (F05 side) | E40-F05 code-review owner, for the producer half of the contract only |
| Required UAT evidence | UAT-08 (F05): all four admitted families load with stable identity, correct adapter selection, final predicate, and the exact applicable/non-applicable stage matrix; a malformed or non-runnable package is rejected with the failing field named. Each consumer's own UAT additionally proves its live wiring per the interaction map's activation-owner closure requirement. |

E40-F06, E40-F07, and E40-F08 must copy the shape source and the contract-test
pointer above verbatim; the same test proves every side of this contract and no
twin test is created. This `contract-only` staging is a predeclared handoff,
not a waiver — an open internal activation obligation blocks epic completion
until each consumer closes it.

### Consumes: I-01 — Corpus and oracle contract

| Property | Contract |
|---|---|
| Producer | E40-F01 Benchmark corpus v1: fixture repo and screened task/bug set |
| Shape source | [Corpus and oracle contract](../architecture.md#corpus-and-oracle-contract) |
| Payload | V1 manifest, entity seed, held-back F2P tests, P2P set, reference patch, fixture SHA, and base ledgers |
| Style | File artifact |
| Shared contract test | `tests/contracts/e40_i01_corpus_contract_test.go#TC-001` |
| What F05 consumes | The **admission, versioning, and held-back-oracle principles** — reproducible execution-based admission, a schema-versioned manifest, oracles stored outside the fixture, and a single named owner per piece of arithmetic |
| What F05 does not consume | The v1 Go manifest's fields and validator. The interaction map's own I-01 row states F05 "consumes the admission and oracle principles without making the Go manifest the global schema" (ADR-F05-01) |
| Non-regression obligation | `bench/corpus/corpus.yaml`, `bench/scripts/checkout-fixture.sh`, `build-ledgers.sh`, `diff-ledgers.sh`, and TC-001 are unmodified by this feature; the Go family keeps working byte-identically as the compatibility adapter |
| Gate mode | live, as assigned by [the interaction map](../E40-interaction-map.md) |

---

## Cross-epic integrations

E40-F05 produces, consumes, and validates **no X-## row**. Of the seven rows in
[E40-cross-epic-map.md](../E40-cross-epic-map.md), X-07 is owned by E40-F02, X-08
by E40-F04, X-09 by E40-F06, X-10 by E40-F07, X-11 and X-13 by E40-F08, and X-12
by E40-F09. F05 touches no E22, E27, E36, E38, E39, or E32 surface: it does not
parse `RunResult`, `StageLog`, or a transcript envelope; it does not decode a
provider-usage envelope; it does not invoke the Rider product-design action, the
keyed dispatch loop, or the Question lifecycle; and it does not hash installed
Shark-data content.

The one adjacency worth naming is X-09. I-04's evaluator and usage reference
fields are deliberately opaque paths precisely so that E40-F06 — the row's owning
feature — can decide the decoded-usage mapping when E27-F15 lands, without
E40-F05 having committed the shape first (ADR-F05-05).

---

## Durable unresolved decisions

No material decision remains open in this feature, and no new Q### is filed.
Applying the materiality test in `skills/question-management/SKILL.md` to the
three candidates:

1. **Python fixture hosting (submodule vs. vendored).** Closed by research
   Decision 6 and ADR-F05-09. Non-material for the same reason
   ADR-F01-01 recorded for the Go fixture: every consumer reaches the fixture
   through the package's pinned `base_sha`, so no scope, acceptance criterion,
   cross-feature contract, or entity gate changes with the answer.
2. **Change-card and tech-debt final-predicate form.** Not open — both are
   specified in [shark-bench-design.md](../shark-bench-design.md) §3 (a required
   machine-checkable acceptance predicate for change-cards; P2P preservation plus
   a structural "debt is gone" predicate for tech-debt). REQ-F-010 expresses both
   from the existing `test` and `lint` capabilities, so no new tooling decision is
   pending.
3. **Provider-usage field names.** Open, but **not F05's**. Epic Q003 remains an
   E40-F06/X-09 research obligation; ADR-F05-05 keeps I-04 clear of it rather
   than resolving it here.

Epic-level Questions Q001 and Q002 remain resolved. Q004's Phase 1 cascade
attribution constraint is superseded for lifecycle v2 by E40-F08's per-dispatch
I-07 record and is not this feature's concern; the underlying `shark run` defect
remains open on its own surface.

---

## Verification plan

| Requirement | Evidence |
|---|---|
| REQ-F-001, REQ-F-002 | AC-001, AC-009, and `tests/contracts/e40_i04_scenario_contract_test.go#TC-030` |
| REQ-F-003, REQ-F-004 | AC-002, AC-003 — epic G8, UAT-08 |
| REQ-F-005 | AC-001, AC-017 |
| REQ-F-006 | AC-010 |
| REQ-F-007 | AC-011, AC-012 |
| REQ-F-008 | AC-013, AC-019 |
| REQ-F-009 | AC-004 |
| REQ-F-010 | AC-014 |
| REQ-F-011 | AC-007 (resource-policy branch) |
| REQ-F-012, REQ-F-013 | AC-006, AC-007, AC-008 — UAT-08 |
| REQ-F-014 | AC-006 — epic G8 |
| REQ-F-015 | AC-005 — UAT-08 |
| REQ-F-016 | AC-020, AC-015 |
| REQ-F-017 | AC-007 (check (b) branch), AC-014, AC-021 |
| REQ-NF-001, REQ-NF-002 | AC-015 |
| REQ-NF-003 | AC-016 |
| REQ-NF-004 | AC-008, AC-018 |
| REQ-NF-005 | Diff review: no shark initialisation command and no live-root write in `bench/scripts/` or `bench/adapters/` |
| REQ-NF-006 | AC-017 |

---

*Last Updated*: 2026-08-11
