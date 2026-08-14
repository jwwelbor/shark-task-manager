# Test Plan: E40-F05 - Lifecycle scenario corpus and adapter contract

**Created:** 2026-08-11
**Feature PRD:** `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F05-lifecycle-scenario-corpus-and-adapter-contract/spec.md`
**Task Spec:** Not yet decomposed. This plan is written directly against
`spec.md` (Step 1-3 use `spec.md` as both the incremental spec and the
traceability target, following the same posture the sibling F01-F04 test
plans used before task decomposition).
**Status:** APPROVED, with two named test-infrastructure gaps resolved below
(AC-020's owning script; the predicate-vocabulary wording disagreement
between `spec.md`'s table and prose) — see "Test infrastructure gaps."

## Scope and drift analysis

`spec.md` is incremental over `feature.md` and `architecture.md#lifecycle-scenario-package-contract`
/ `architecture.md#adapter-capability-contract`, both of which the epic
already fixed before this feature's own spec was written. Comparing `spec.md`
against `feature.md`:

- Every `spec.md` REQ-F-* and REQ-NF-* traces to a `feature.md` Scope,
  Acceptance boundary, or Contracts line: schema definition (REQ-F-001/002) →
  Scope bullet 1; four admitted seeds (REQ-F-014) → Scope bullet 2; Python
  fixture + Go compatibility adapter (REQ-F-005/006/016) → Scope bullet 3;
  agent-visible/evaluator-only separation (REQ-F-009) → Scope bullet 4;
  rejection of unrunnable/non-applicable/non-machine-checkable candidates
  (REQ-F-012/015) → Scope bullet 5. No REQ introduces a capability absent
  from `feature.md`'s Scope.
- Every `feature.md` Acceptance boundary bullet is covered: "loader accepts
  four families, rejects malformed with field named" → AC-001/AC-005;
  "re-loading yields same identity" → AC-009; "generic component selects
  through adapter without knowing language" → AC-011/AC-012; "feature
  declares D01-D05 applicable, others explicitly non-applicable" →
  AC-002/AC-003. No boundary bullet is left uncovered.
- `feature.md`'s Contracts section ("consume I-01 principles, not its Go
  schema"; "produce I-04") matches `spec.md`'s ADR-F05-01 and the Cross-feature
  interactions section verbatim — no semantic drift in what is consumed vs.
  produced.
- `spec.md`'s own "Durable unresolved decisions" section closes all three
  candidate open questions with a materiality argument (submodule vs.
  vendored fixture; change-card/tech-debt predicate form; provider-usage
  field names deferred to F06/X-09). No new Q### is warranted by this review
  either — the three candidates were already exhaustively covered.

**No drift found.** No BA or architecture refinement is required.

### Feature-level coverage check (component-changes table vs. AC/REQ)

`spec.md`'s Architecture "Component changes" table lists 14 new/modified
artifacts. Cross-referencing each against the AC list:

| Component | Owning REQ/AC |
|---|---|
| `bench/scenarios/scenarios.yaml` | REQ-F-001, AC-001 |
| `bench/scenarios/packages/<id>/package.yaml` | REQ-F-002..017, AC-001-004, AC-009, AC-013, AC-019 |
| `bench/scenarios/packages/<id>/input/prompt.md` | REQ-F-009, AC-004 |
| `bench/scenarios/packages/<id>/evaluator/` | REQ-F-009, AC-004 |
| `bench/adapters/go/{adapter.yaml,adapter.sh}` | REQ-F-006, AC-010, AC-011 |
| `bench/adapters/python/{adapter.yaml,adapter.sh}` | REQ-F-006, AC-010, AC-011, AC-020 |
| `bench/fixture-py/` + `.gitmodules` | REQ-F-016, AC-020 |
| `bench/scripts/checkout-scenario-fixture.sh` | REQ-NF-006, AC-017 |
| `bench/scripts/admit-scenario.sh` | REQ-F-012/013, AC-006, AC-007, AC-008 |
| `bench/scripts/eval-predicate.sh` | REQ-F-010, AC-014, AC-021 |
| `bench/scripts/tests/tc031_adapter_conformance_test.sh` | AC-010, AC-011, AC-012 |
| `bench/scripts/tests/run-all.sh` | registration only, no independent AC |
| `bench/README.md` | documentation — reviewed, not executed (see "Integration scenarios") |
| `tests/contracts/e40_i04_scenario_contract_test.go` | TC-030 owner; AC-001-005, AC-009, AC-016, AC-019 |
| `tests/contracts/testdata/e40_i04/` | AC-005 |

**One gap found:** AC-020 ("a fresh clone of `bench/fixture-py` at
`base_sha`... the adapter's `test` capability reports every entry `pass` at
that SHA... Verified by a bench script, not by TC-030") names no script in
the component-changes table. `admit-scenario.sh`'s own admission check (b)
only resolves the scenario's `p2p_selection` — a *subset* of the fixture's
tests — not the fixture's *entire* suite, so passing check (b) does not by
itself prove REQ-F-016's "full test suite green at `base_sha`," which is the
premise `p2p_selection`'s absolute semantics rest on (ADR-F05-10). This is a
real coverage gap, not a restatement of an existing check.

**Resolution required for this test plan to be actionable:** add
`bench/scripts/verify-fixture-py-base.sh <base_sha>` as a new deliverable —
checks out `bench/fixture-py` at `<base_sha>` via
`checkout-scenario-fixture.sh py <base_sha> <tmp_dir>`, asserts
`pyproject.toml` carries dependency, lint, and formatter configuration
sections, asserts no path under `<tmp_dir>` matches `bench/scenarios/**` or
`evaluator/`, then invokes `bench/adapters/python/adapter.sh test --checkout
<tmp_dir>` with **no** `--include` filter (the full suite, not a
`p2p_selection` subset) and asserts every returned entry's `outcome` is
`pass`. TC-040 below drives this script directly. Task decomposition must
create an explicit task for it; it is not optional research left to the
developer's judgment — the same posture F01's test plan took for
`diff-ledgers.sh`/`verify-clean-checkout.sh`.

This is why the plan below reaches **APPROVED** rather than
**NEEDS_REFINEMENT**: every AC has a test, and the one AC whose owning script
was unnamed in `spec.md` now has a concrete name, argument shape, and owning
test case.

## Test tiers

Mirrors F01's tiering, for the same reason: REQ-NF-003 keeps the schema
validator submodule-free so CI (`actions/checkout@v4`, no submodule init)
stays green, while execution-based admission genuinely needs the fixtures.

| Tier | Runs | Needs submodule? | Where |
|---|---|---|---|
| **Tier 1** | `make test` (CI + every dev machine) | No — reads only committed scenario artifacts | `tests/contracts/e40_i04_scenario_contract_test.go` (TC-030) |
| **Tier 1b** | Curator, manually or via `bench/scripts/tests/run-all.sh` | Yes for the adapter conformance run (both checkouts); no fixture-language branching in the harness itself | `bench/scripts/tests/tc031_adapter_conformance_test.sh` (TC-031) |
| **Tier 2** | Curator, at scenario-corpus build time and on every scenario/fixture/adapter change | Yes — `git submodule update --init` first (both `bench/fixture-repo` and `bench/fixture-py`) | `bench/scripts/{admit-scenario,eval-predicate,checkout-scenario-fixture,verify-fixture-py-base}.sh` against real checkouts |

Tier 2 is what REQ-NF-004's "byte-identical verdicts... at an unchanged
fixture SHA and toolchain identity" exercises. It is **not** gated by root
`make test`; `bench/README.md`'s new "I-04 scenario package schema" section
must name the exact Tier 2 invocation sequence so "curator re-runs it" is a
real, documented action (REQ-NF-004's reproducibility claim otherwise has no
re-runnable command).

## Determinism boundary (REQ-NF-004, AC-008)

Same class of incidental non-determinism as F01, now doubled across two
adapters:

- `adapter.sh test`'s JSON output must carry no wall-clock/duration field in
  the identity comparison — `admit-scenario.sh` and TC-034's comparison must
  normalize/ignore any timing field either adapter emits, not merely assume
  neither does.
- `adapter.sh lint`'s `issues` array is a multiset with position-independent
  identity (`rule`, `file`, `text` — REQ-F excludes line/column per the
  Adapter capability contract table); `admit-scenario.sh` must sort it into a
  fixed order before comparing two runs, or TC-034 could falsely report a
  mismatch from ordering alone.
- `admit-scenario.sh`'s per-candidate verdict output must be emitted in a
  fixed order (sorted by `scenario_id`) so two runs' full output is
  byte-comparable.

TC-034 provisions **two independent** `checkout-scenario-fixture.sh`
invocations into two different temp directories for the same `base_sha`, not
one reused checkout — otherwise a memoizing implementation could pass by
accident, the same trap F01's TC-006 was built to catch.

## AC test matrix

| AC | Requirement(s) | Tier | Technique | Test case | Input/setup | Expected outcome; edge and negative case |
|---|---|---|---|---|---|---|
| AC-001, AC-016 | REQ-F-001, REQ-F-002, REQ-F-005, REQ-F-009, REQ-F-010, REQ-F-011, REQ-NF-003 | 1 | Contract-surface enumeration | TC-030 | `TestTC030_I04ScenarioPackageContract` reads, via `os.ReadFile`, the committed `bench/scenarios/scenarios.yaml` and every `bench/scenarios/packages/*/package.yaml`, run both with the fixture submodules uninitialized (gitlink-only, matching CI) and after `git submodule update --init`. The decoder uses strict/unknown-field-rejecting unmarshalling, and one malformed fixture under `tests/contracts/testdata/e40_i04/` injects each of the four forbidden I-01 field names (`fixture.toolchain.go_version`, `golangci_lint_version`, `p2p_sets`, `reference_patch_path`) into an otherwise-valid package | Every package's REQ-F-002/005/009/010/011 field set is present and well-typed; every `input.agent_visible`/`evaluator_only`/`replay_reference`/predicate-operand path exists as a committed file; `fixture_id` and `adapter.name` resolve to entries registered in `scenarios.yaml`; `schema_version` matches what the validator supports. Both submodule states produce identical results (proves AC-016: no populated submodule required). Negative: a package referencing an unregistered `adapter.name`, or a predicate-operand path that does not exist on disk, fails naming that field; a package carrying any of the four forbidden I-01 field names is rejected as an unknown field — proving REQ-F-001's "no field... is inherited by name" mechanically, not by the real committed packages simply happening to omit them. |
| AC-002 | REQ-F-004 | 1 | Decision table (4 families x 5 prelude stages) | TC-030 | Same artifacts as above, asserting `stage_matrix.prelude.D01`-`.D05` for each of the four seeds | The `feature` package declares all five `true`; `bug`, `change_card`, `tech_debt` declare all five `false`, each carrying a non-empty `reason`. Negative: a `bug` package with one prelude stage `true`, or a `false` stage with an empty `reason`, is rejected naming the offending family and stage (REQ-F-004's own wording). |
| AC-003 | REQ-F-003 | 1 | Equivalence partitioning (valid rule shape vs. any enumerated-status shape) | TC-030 | Same artifacts, asserting `stage_matrix.lifecycle` shape | Every package's `stage_matrix.lifecycle` is exactly `{mode: all_dispatched, evidence_required: true}`. Negative: a malformed test fixture under `tests/contracts/testdata/e40_i04/` embeds a status-name list (e.g. `[in_progress, completed]`) in place of the rule and is rejected — proving the validator actually distinguishes the rule shape from an enumerated list, not merely that the real committed packages happen not to have one. |
| AC-004 | REQ-F-009 | 1 | Attack-class enumeration (leak surface: agent-visible path resolving into evaluator/) | TC-030 | Same artifacts; for each package, resolve `input.agent_visible` and compare its resolved path against the package's `evaluator/` subtree; separately check `replay_reference` presence/absence by family | No `input.agent_visible` path resolves inside `evaluator/`; `py-feature-recurring-tasks` carries `replay_reference` and the other three seeds do not. Negative: a malformed fixture whose `input.agent_visible` points at a file physically inside `evaluator/` is rejected; a `bug`/`change_card`/`tech_debt` fixture carrying a `replay_reference` field is rejected (family mismatch). |
| AC-005 | REQ-F-015 | 1 | Decision table (one malformed fixture per named rejection case, all three resource ceilings varied independently) | TC-030 | Table-driven subtests under `TestTC030_I04ScenarioPackageContract`, one fixture per row of the authoritative case table in "Test infrastructure" below (14 rows: 8 structural cases + 6 resource-ceiling cases covering `max_cost_usd`/`max_wall_clock_seconds`/`max_generated_tasks` each missing and each non-positive) | Each of the 14 named cases fails validation with the specific field named in the returned error, matching the case's own defect — not a generic "invalid package" message. Every one of the three ceiling fields is independently proven to gate on both "missing" and "non-positive" (not just one ceiling standing in for all three). Negative: a fixture correcting exactly one field passes, proving the validator isn't rejecting the whole file for unrelated reasons. |
| AC-006 | REQ-F-012, REQ-F-014 | 2 | Decision table (4 families x 6 admission checks, all pass) | TC-032 | `bench/scripts/admit-scenario.sh` against all four seed `scenario_id`s on a real `checkout-scenario-fixture.sh py <base_sha>` checkout | All four (`py-bug-due-date-boundary`, `py-change-priority-scale`, `py-techdebt-consolidate-validation`, `py-feature-recurring-tasks`) are admitted, one per family, with `base_outcome: false` and `reference_outcome: true` recorded for each. Negative: fewer than four admitted, or any pair sharing a family, fails the gate's own summary assertion. |
| AC-007 | REQ-F-011, REQ-F-012 | 2 | Decision table (one candidate per rejection check; check (b) has two independent trigger conditions, so seven candidates cover the six named checks) | TC-033 | `bench/scripts/admit-scenario.sh` against **seven** candidates built from a copy of `py-bug-due-date-boundary`, each mutated at test time for exactly one trigger: (a) fixture `Makefile`'s build target broken so `adapter.sh build` fails; (b-empty) `p2p_selection.include` repointed at a file with no matching tests (empty resolution); (b-fail) `p2p_selection` unchanged but one of its resolved entries deliberately broken so it fails at base; (c) `stage_matrix.prelude` mutated to violate REQ-F-004; (d) `final_predicate` operands loosened so the predicate is already true at base; (e) the reference patch under `evaluator/` corrupted so the predicate stays false after applying it; (f) `resource_policy.max_cost_usd` set to `0` | Each of the seven candidates is rejected naming exactly its own check — (a) "runnable base fixture," (b-empty)/(b-fail) both name check (b) ("p2p_selection resolution/at-base pass") despite differing triggers, (c) "stage matrix invariant," (d) "final predicate false at base," (e) "final predicate true after reference," (f) "resource policy: max_cost_usd." None of the seven appears in any admitted-set output. Negative: a candidate accidentally passing all checks (misconfigured mutation) is a plan-level red flag — TC-033 fails loudly rather than silently admitting it. |
| AC-008 | REQ-NF-004 | 2 | State transition (re-run stability, independent checkouts) | TC-034 | `bench/scripts/admit-scenario.sh` over all four seeds, invoked twice against two separately provisioned `checkout-scenario-fixture.sh` temp checkouts of the same `base_sha`/toolchain | Byte-identical verdict output and identical `base_outcome`/`reference_outcome` per candidate, both runs (per "Determinism boundary" normalization). Negative: a script that memoizes/re-reads a prior verdict instead of re-running the checks is caught because the two checkouts are provisioned independently. |
| AC-009 | REQ-F-002 | 1 | State transition (load twice, same state) | TC-030 | `TestTC030_I04ScenarioPackageContract` loads each package twice from the same committed files within one test run | `scenario_id`, `scenario_version`, `fixture.base_sha`, `stage_matrix`, `toolchain_identity`, and every resolved input path are byte-identical across the two loads. Negative: a loader with non-deterministic map iteration ordering leaking into the resolved-path list would fail this comparison. |
| AC-010, AC-011 | REQ-F-006, REQ-F-007 | 1b | Contract-surface enumeration (6 capabilities x 2 adapters, identical assertion set, closed-vocabulary boundary) | TC-031 | `bench/scripts/tests/tc031_adapter_conformance_test.sh` runs `identity`, `inject-tests`, `test`, `lint`, `build`, `format-check` against both `bench/adapters/go/adapter.sh` (on `bench/fixture-repo`) and `bench/adapters/python/adapter.sh` (on `bench/fixture-py`), applying the identical JSON-shape assertion set to both; additionally invokes each `adapter.sh` with a seventh, undefined capability name (e.g. `adapter.sh coverage --checkout <dir>`) | Every one of the six capabilities' stdout JSON validates against the shape fixed in the Adapter capability contract table for both adapters, and the harness asserts the capability set is exactly these six — no extra verb is silently accepted. `test`'s emitted ids are normalized: Go emits `<package import path>::<test name>`, Python emits `<module path>::<test name>`, for the same conceptual entry set; the harness performs no language-aware id parsing on either side. Negative: an adapter emitting an un-normalized id fails the shared shape assertion; the seventh, undefined capability name is rejected with a non-zero exit by both adapters, proving REQ-F-006's "closed set of six verbs" rather than merely asserting the six that exist. |
| AC-012 | REQ-F-007 | 1 | Attack-class enumeration (forbidden-token leak surface, including indirect language selection) | TC-031 | `grep -rE 'python|pytest|pip|go test|golangci-lint|go build|\.py"|\.go"'` over every generic scenario, evidence, and admission script (`bench/scripts/admit-scenario.sh`, `bench/scripts/eval-predicate.sh`, `bench/scripts/checkout-scenario-fixture.sh`, `bench/scripts/tests/*.sh` excluding `tc031_*` itself); additionally, source-level check that none of these scripts branches (`if`/`case`) on `fixture_id`, `adapter.name`, or a `scenario_id` substring — the closed set of allowed discriminants is "which `adapter.sh` to invoke," never a language-specific code path inline | Zero literal-token hits outside `bench/adapters/*/`, and zero conditional branches on fixture/adapter identity in the generic scripts. Negative: a hypothetical future edit inlining a `pytest` invocation, or an `if fixture_id == "python"` branch that calls language-specific logic without going through `adapter.sh`, is caught by this same check, mechanically rather than by code-review convention. |
| AC-013, AC-019 | REQ-F-008 | 1 + 2 | Boundary/state enumeration (two real, pinned toolchain states; two encodings) | TC-030 (cross-encoding), TC-035 (changed toolchain) | TC-030 asserts each package's top-level `toolchain_identity` equals `admission.toolchain_identity` element-for-element and in order. TC-035 provisions two **real, pinned, cache-warmed** interpreter environments (e.g. a second Python minor version, such as 3.11 alongside the pinned 3.12, both installed via the same dependency-lock mechanism the fixture uses — never a stubbed `python`/`pip` on `PATH` reporting a fabricated version string), invokes `bench/adapters/python/adapter.sh identity --checkout <dir>` once under each, and separately re-runs the full `admit-scenario.sh` for `py-bug-due-date-boundary` under each environment | TC-030: the two encodings agree exactly. TC-035: the two real toolchain environments produce two genuinely different ordered key/value lists, comparable only by equality — no consumer reads a named key — and the full admission re-run under each environment records the matching `admission.toolchain_identity` for that environment, proving the pin propagates end-to-end, not just at the `identity` capability boundary. Negative: an implementation reading `toolchain_identity[0].value` directly instead of comparing the whole ordered list would still "work" by accident here, which is exactly why AC-013 requires the equality-only comparison as the assertion, not key-indexed reads; a version-string stub would pass even if the adapter never actually interrogated the live toolchain, which real pinned interpreters catch. |
| AC-014 | REQ-F-010 | 2 | Boundary value analysis (tech-debt rule-count threshold, adjusted to the seed's actual `max_remaining: 0` floor) + decision table (all four predicate kinds x {base-false, reference-applied-true, P2P-regression-false}, uniformly) | TC-036 | `bench/scripts/eval-predicate.sh` invoked against real `test`/`lint` capability output captured by TC-032's own admission run (base-state and reference-applied-state JSON for all four seeds, reused rather than re-derived), plus dedicated boundary/negative states per kind, each kind carrying its own P2P-regression mutation so the absolute P2P clause is proven independently for every kind, not only `f2p_p2p`: **`f2p_p2p`** (`py-bug-due-date-boundary`) — base (repro fails, false), reference-applied (repro passes + `p2p_selection` green, true), P2P-regression (repro passes but one `p2p_selection` entry broken, false); **`acceptance_tests`** (`py-change-priority-scale`) — base (acceptance tests fail/absent, false), reference-applied (all acceptance tests pass + `p2p_selection` green, true), one acceptance test still failing post-reference (false), and separately a P2P-regression mutation with all acceptance tests passing but one `p2p_selection` entry broken (false); **`p2p_plus_rule_drop`** (`py-techdebt-consolidate-validation`, whose seed pins `max_remaining: 0`) — **two** lint-issue-count states, not three, because `max_remaining: 0` is the domain floor for a non-negative issue count and no "over-satisfied" state exists below it: count `= 1` (over threshold, false) and count `= 0` (exactly at threshold, true), plus a P2P-regression mutation at count `= 0` with one `p2p_selection` entry broken (false); **`child_oracles_union`** (`py-feature-recurring-tasks`) — all integration tests and child oracles green (true), one child oracle deliberately failed (false), one integration test failed with oracles green (false), and a P2P-regression mutation with integration tests and oracles green but one `p2p_selection` entry broken (false) | All four predicate kinds evaluate to the documented boolean from `test`/`lint` output alone, exercising base (false), reference-applied (true), and an independent P2P-regression negative (false) for every kind — the same admission gate boundary REQ-F-012 checks (d)/(e) already require, so TC-036 proves `eval-predicate.sh` is the correct single owner of that arithmetic rather than assuming it. The tech-debt boundary is exact at `max_remaining` (not off-by-one) and does not assert an impossible sub-floor state. Negative: an implementation using `<` instead of `<=` for `max_remaining` would fail the exactly-at-threshold case; one relying only on `integration_test_ids`/named-test-id lists and ignoring the shared `p2p_selection` clause would falsely report true on any of the four kinds' P2P-regression mutations above; one relying only on `child_oracles` and ignoring `integration_test_ids` would falsely pass the failed-integration-test case. |
| AC-015 | REQ-NF-001, REQ-NF-002 | 1 | Boundary/state enumeration (submodule populated vs. not) | TC-037 | Repo root `make fmt && make lint && make test`, run once with both fixture submodules uninitialized (CI-like) and once after `git submodule update --init`; separately `go list ./...` in both states | Both runs are green in both states; `go list ./...` lists no `bench/fixture-py`, `bench/fixture-repo`, or scenario/evaluator package in either state; `scripts/lint-positional-selection.sh`'s `internal`/`cmd`-only grep scope is unaffected. Negative: a stray Python `__init__.py`-adjacent Go shim or a scenario helper accidentally placed under `internal/` would surface here as a new `go list` entry. |
| AC-017 | REQ-NF-006 | 1 + 2 | Attack-class enumeration (frozen-interface regression) | TC-038 | (i) `git diff <pre-feature-ref> -- bench/scripts/checkout-fixture.sh` — byte diff; (ii) `bench/scripts/checkout-scenario-fixture.sh py <base_sha> <dest_dir>` and `... go <base_sha> <dest_dir>`, each resolving `submodule_path` from `scenarios.yaml`; (iii) confirm `admit.sh`/`build-ledgers.sh` (I-01 callers) still invoke the original `checkout-fixture.sh` by re-running TC-004/TC-006/TC-007 (F01's own tests) unmodified | (i) empty diff — byte-unchanged. (ii) both fixture ids resolve and clone at the requested SHA into `<dest_dir>`. (iii) F01's existing tests still pass unmodified, proving the I-01 callers were not silently repointed at the new script. Negative: any diff in (i), or an I-01 test failing after this feature lands, fails TC-038. |
| AC-018 | REQ-NF-004 | 2 | Attack-class enumeration (offline + warm cache, both isolation mechanisms independently exercised) | TC-039 | `bench/scripts/admit-scenario.sh` over all four seeds with pip/go module caches pre-warmed, run **twice** under two independent isolation mechanisms so neither is merely an untested fallback: once with `unshare --net` (Linux network-namespace block) and once with the portable `GOPROXY=off GOFLAGS=-mod=readonly PIP_NO_INDEX=1` plus a poisoned `PIP_INDEX_URL`/`http_proxy` pointed at a guaranteed-closed local port (mirrors F01's TC-013 technique, extended to `pip`) | The gate completes over all four seeds with no network access, under **both** mechanisms independently — proving the portable fallback actually works, not merely that it exists as an assumed equivalent to the Linux-only namespace block. Negative: an adapter falling back to `pip install` on a cache miss instead of failing loudly would hang or attempt network access under either mechanism, which the isolation harness catches as a non-zero exit or a detected outbound attempt. |
| AC-020 | REQ-F-016 | 2 | Contract-surface enumeration (fixture content) + equivalence partitioning (full suite, not p2p subset) | TC-040 | `bench/scripts/verify-fixture-py-base.sh <base_sha>` (new script — see "Test infrastructure gaps"): fresh `checkout-scenario-fixture.sh py <base_sha> <tmp>`, asserts `pyproject.toml` exists with dependency, lint, and formatter configuration sections, asserts no path under `<tmp>` matches `bench/scenarios/**` or `evaluator/`, then `bench/adapters/python/adapter.sh test --checkout <tmp>` with no `--include` filter | `pyproject.toml` carries all three configuration sections; the tree contains fixture source and tests only; every entry in the **unfiltered** `test` capability output is `pass`, with exactly one named, permanent exception — `tests.test_manager::test_recurring_task_generates_next_occurrence` — permitted to report `skip` (never `fail`); no other entry may be `skip` or `fail` (ADR-F05-11). Negative: this must not be satisfied merely by the `p2p_selection`-scoped check inside `admit-scenario.sh` (admission check (b)) — TC-040 explicitly omits `--include` so a scenario's narrow selection can't mask a red test elsewhere in the suite, and any *other* skip or a failure of the one named exception still rejects. |
| AC-021 | REQ-F-017 | 1 | Attack-class enumeration (hidden-baseline leak/read surface, not limited to one file-naming convention) | TC-041 | Static argument-trace check that `eval-predicate.sh`'s only file reads resolve to its three positional arguments (`<package.yaml>`, `<test-output.json>`, `<lint-output.json>`) — grep for every file-read construct (`cat`, `<`, `source`, `jq -f`, redirection) in the script and assert each resolved path traces to `$1`/`$2`/`$3`, not a literal or derived path; broaden the naming-convention grep beyond `ledger` to also match `baseline`, `snapshot`, and any path under a `ledgers/` or `bench/corpus/` directory; separately `find bench/scenarios -iname '*ledger*' -o -iname '*baseline*'` (expect zero results) | `eval-predicate.sh` reads only its three declared arguments — no hardcoded or derived path outside them — under any naming convention, not just the literal string "ledger"; no such file is committed anywhere under `bench/scenarios/`. Negative: a future edit reintroducing a base comparison under a differently-named file (e.g. `snapshot.json`) would be caught by the argument-trace check, which does not depend on the file's name, before it silently violates ADR-F05-10. |

## Acceptance-criteria review

Every AC above is unambiguous, testable, traceable to a `spec.md` REQ, and
specifies an exact expected output (a named failing check, an exact
threshold boundary, a byte-identical comparison, a specific field-agreement
assertion) rather than "works correctly" or "handles errors gracefully." No
AC is an open-ended robustness assertion. REQ-NF-004's "byte-identical" is
the closest candidate to open-endedness and is closed by the "Determinism
boundary" section above, the same way F01's plan closed the analogous claim.
Every runtime AC above has at least one explicit negative case in the matrix.

**Wording ambiguity in `spec.md`'s source of truth (flagged, not blocking).**
The Final predicate vocabulary table's "True when" column words three of the
four kinds base-relative — `f2p_p2p` ("no entry... **regressed** from `pass`
to `fail`"), `acceptance_tests` ("the P2P selection has **no regression**"),
`p2p_plus_rule_drop` ("the P2P selection has **no regression**") — while
`child_oracles_union` is worded absolute ("every P2P entry is `pass`"), and
the paragraph immediately below the table then declares all four absolute
("Each kind's P2P clause is absolute, not base-relative... no predicate
consults a stored base ledger"). On an admitted scenario (REQ-F-016's fixture
green at `base_sha`, verified by admission check (b)) the two readings
produce the same truth value, so this does not change what this test plan
asserts is true or false in any AC-014 case above. It does change what a
*base-relative* implementation would have to do differently from an
*absolute* one — the former reads a stored base ledger, which REQ-F-017 and
ADR-F05-10 exist specifically to eliminate, and which AC-021 forbids. This
plan treats the paragraph, not the three "regression"-worded table rows, as
authoritative (REQ-F-017's operand shape and ADR-F05-10's own text settle
it), and **TC-041 — not TC-036 — is the test that discriminates the two
readings**: TC-036's P2P-regression mutations break an entry strictly after
the base/reference-applied point, which reads `false` under either
semantics, so it cannot tell a compliant absolute implementation from a
non-compliant base-relative one. TC-041's static argument-trace and
no-stored-file check is what would catch an implementation that persisted or
read a base ledger despite `eval-predicate.sh` receiving only the current
`test`/`lint` output. **Recommendation to the spec/architecture owner:**
reword the three "regression"-worded table rows to match the paragraph's
absolute framing (e.g. `f2p_p2p`: "every entry the P2P selection resolves to
is `pass`") so the table and the prose agree; this is a wording fix, not a
behavior change, and does not block this plan's APPROVED status.

## ISTQB technique application

| AC | Technique(s) applied | Test cases | Rationale |
|---|---|---|---|
| AC-001, AC-016 | Contract-surface enumeration | TC-030 | I-04 is a cross-feature interaction surface (F06/F07/F08 read it); every field, cross-reference, and submodule-state combination must be enumerated, not sampled. |
| AC-002 | Decision table | TC-030 | Family x prelude-stage is a combinatorial invariant (4 families x 5 stages); a decision table forces every cell, not just the two committed extremes. |
| AC-003 | Equivalence partitioning | TC-030 | "Rule shape" vs. "any enumerated-status shape" is exactly two partitions; a malformed-fixture case proves the validator actually distinguishes them. |
| AC-004 | Attack-class enumeration | TC-030 | "Never resolves into evaluator/" is a defensive/leak-surface property — enumerate the leak, don't just assert the happy-path path resolves outside it. |
| AC-005 | Decision table | TC-030 | REQ-F-015 lists 11 distinct named rejection cases; each is a row a table-driven test must exercise independently with its own expected failing field. |
| AC-006, AC-007 | Decision table | TC-032, TC-033 | Admission is six boolean checks combining into accept/reject per candidate; a decision table forces every check to have its own named-rejection test, not just the all-pass path. |
| AC-008 | State transition (re-run stability) | TC-034 | "Reproducible" is a claim about two executions of the same state — the technique built to enumerate exactly that, proven across two independently provisioned checkouts. |
| AC-009 | State transition | TC-030 | Loading the same artifact twice within one process is the minimal state-transition case for identity stability. |
| AC-010, AC-011 | Contract-surface enumeration | TC-031 | The adapter boundary is itself a contract two implementations must both satisfy; the same assertion set must run against both, not a bespoke check per language. |
| AC-012 | Attack-class enumeration | TC-031 | "No generic component branches on language" is a leak-surface property; grep is the mechanical enumeration of that surface, not a convention. |
| AC-013, AC-019 | Boundary/state enumeration | TC-030, TC-035 | Two toolchain states and two field encodings are the two boundaries worth distinguishing; equality-only comparison (not key reads) is the property under test. |
| AC-014 | Boundary value analysis + decision table | TC-036 | The tech-debt threshold is an ordered boundary (`max_remaining` exact vs. one-over; the seed's `max_remaining: 0` has no sub-floor "over-satisfied" state, so BVA yields two states here, not three); all four predicate kinds are a decision table over family x {base-false, reference-applied-true, P2P-regression-false} — not merely two of the four, since every kind's P2P clause is independently absolute per ADR-F05-10 and must be independently proven, not assumed transitively from `f2p_p2p`'s coverage alone. |
| AC-015, AC-017, AC-018 | Attack-class / boundary-state enumeration | TC-037, TC-038, TC-039 | Repo hygiene regression, frozen-interface regression, and offline capability are each a defensive property against a specific class of silent breakage. |
| AC-020 | Contract-surface enumeration + equivalence partitioning | TC-040 | Fixture content is a contract surface (what REQ-F-016 requires present/absent); "full suite" vs. "p2p subset" is the partition this AC exists to keep distinct from admission check (b). |
| AC-021 | Attack-class enumeration | TC-041 | "No ledger file read or committed" is a structural leak/regression-surface property against ADR-F05-10's absolute-P2P design, enumerated by grep rather than asserted by convention. |

## Caller-Path Contracts

This feature is deterministic runtime tooling (bash scripts executing real
git/python/go/pytest/golangci-lint against real or test-time-mutated files,
plus one Go contract test), so `content-only` opt-outs apply only to
`bench/README.md`'s documentation additions (see "Integration scenarios").
Every other row below drives its real production entrypoint.

| TC | Entrypoint (exact invocation) | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-030 | `TestTC030_I04ScenarioPackageContract` (the contract test function itself) calling `os.ReadFile` + real YAML/JSON unmarshal against `bench/scenarios/scenarios.yaml` and `bench/scenarios/packages/*/package.yaml`, plus table-driven subtests reading `tests/contracts/testdata/e40_i04/*` | Real filesystem read of committed YAML, real parser | Do not substitute an in-memory struct for the committed package files; must parse the real files | A validator reading a hand-built in-memory manifest would stay green even if the real committed YAML were malformed — the same trap F01's TC-001 Caller-Path Contract names. |
| TC-031 | `bench/scripts/tests/tc031_adapter_conformance_test.sh` invoking `bench/adapters/go/adapter.sh <capability>` and `bench/adapters/python/adapter.sh <capability>` directly, once per capability, against real checkouts | Real subprocess invocation of both `adapter.sh` scripts; real `go test`/`golangci-lint`/`pytest`/fixture-linter underneath them | Do not stub either `adapter.sh`'s stdout with canned JSON; must invoke the real scripts against real checkouts | A harness asserting against a hand-authored JSON fixture instead of live adapter output would pass even if a real adapter emitted a malformed or un-normalized shape. |
| TC-032 | `bench/scripts/admit-scenario.sh <scenario_id>` for each of the four seed ids, against a real `checkout-scenario-fixture.sh py <base_sha>` checkout | Real git clone/checkout of the submodule; real adapter subprocess execution for `build`/`test`/`lint` | Do not stub any capability's output or hardcode a per-candidate boolean verdict | An admission gate hardcoding "admitted" for the four named seed ids would pass without ever having actually executed the checks. |
| TC-033 | `bench/scripts/admit-scenario.sh <mutated-candidate-path>`, once per one of the seven test-time-mutated copies of `py-bug-due-date-boundary` described in the AC test matrix | Same as TC-032 | Same as TC-032; additionally do not special-case a mutated candidate to force a specific rejection message | A gate that always reports the first of the six checks as the failure would pass TC-033 by coincidence unless each mutation's *specific* named check is asserted — including distinguishing the two independent triggers that both name check (b). |
| TC-034 | `bench/scripts/admit-scenario.sh` over all four seeds, invoked once against each of two independently provisioned `checkout-scenario-fixture.sh` temp checkouts | Same as TC-032, run twice from a clean state each time | Do not memoize or cache verdicts between the two invocations under test | A cached-verdict implementation would trivially "reproduce" without the checks having actually re-run, hiding real nondeterminism. |
| TC-035 | `bench/adapters/python/adapter.sh identity --checkout <dir>` invoked under two distinct, real, pinned interpreter environments, plus a full `bench/scripts/admit-scenario.sh py-bug-due-date-boundary` re-run under each | Real subprocess invocation reporting the actual live interpreter/tooling versions; real environment switch (e.g. two provisioned virtualenvs), not a `PATH`-stubbed binary | Do not hardcode either expected identity list; must derive both from live tool invocation in each environment; do not substitute a stub binary reporting a fabricated version string for either environment | An identity capability reading a constant instead of interrogating the live toolchain would report the same list under both environments, hiding a real toolchain change from E40-F09's future comparison-identity check; a stubbed binary would pass even if the adapter never actually queried the live toolchain. |
| TC-036 | `bench/scripts/eval-predicate.sh <package.yaml> <test-output.json> <lint-output.json>` against real `adapter.sh test`/`lint` output captured from all four seeds — base and reference-applied states reused from TC-032's own admission run, plus the dedicated boundary/negative-mutation states described in the AC test matrix | Real `test`/`lint` capability JSON as produced by the real adapter (TC-032's captured artifacts, or a fresh real adapter invocation for the dedicated boundary states) | Do not stub the lint-issue count or test-outcome set directly in the predicate evaluator's own test harness; the boundary and negative-mutation states must come from real fixture edits, real evaluator-only file placement, or real reference-patch mutation, not hand-authored JSON standing in for adapter output | A predicate evaluator reading a hand-crafted count instead of the real adapter's `lint` output could pass the boundary test while silently miscounting a real lint run; an implementation that ignores a predicate kind's shared `p2p_selection` clause would pass the happy-path states here but is caught by the dedicated P2P-regression negative mutation carried by every one of the four kinds, not only `f2p_p2p`. |
| TC-037 | Repo root `make fmt && make lint && make test` | Real `go`, `gofmt`, `golangci-lint` subprocesses via the Makefile | Do not run against a filtered subset of packages | A green result from a hand-picked package subset would hide a scenario/fixture package accidentally entering `./...`. |
| TC-038 | `git diff` against `bench/scripts/checkout-fixture.sh`; `bench/scripts/checkout-scenario-fixture.sh <fixture_id> <base_sha> <dest_dir>`; re-run of F01's committed TC-004/TC-006/TC-007 | Real git diff; real script execution; real F01 test re-run, unmodified | Do not hand-edit F01's tests to accommodate this feature; they must pass exactly as F01 left them | A silently repointed `admit.sh`/`build-ledgers.sh` (now calling the new script instead of the frozen one) would only surface if the *existing, unmodified* I-01 tests are re-run, not a fresh assertion invented for this feature. |
| TC-039 | `bench/scripts/admit-scenario.sh` under `unshare --net` (Linux) or the portable proxy-poison fallback, with pip/go caches pre-warmed | Real subprocess execution with genuine network isolation, not a code-level flag | Do not simulate "offline" by stubbing network calls in code; the environment itself must be offline; do not let a missing pinned interpreter/package fall back to a network install | An implementation checking only a config flag for "offline mode" while still depending on a network fallback in production would pass a fake test while remaining unsafe. |
| TC-040 | `bench/scripts/verify-fixture-py-base.sh <base_sha>` (new script) | Real git checkout of the submodule; real `adapter.sh test --checkout <dir>` with no `--include` filter | Do not pass a `p2p_selection`-scoped `--include` filter; must run the unfiltered full suite | A check reusing admission's `p2p_selection`-scoped result would pass even if a test outside that selection were red at `base_sha`, silently breaking ADR-F05-10's absolute-P2P premise. |
| TC-041 | Static argument-trace grep over `bench/scripts/eval-predicate.sh`'s file-read constructs; `grep -riE 'ledger|baseline|snapshot'` over the same file; `find bench/scenarios -iname '*ledger*' -o -iname '*baseline*'` | Real source grep and real filesystem find | Do not restrict the grep to a hand-picked line range or a single naming convention | A grep scoped to only the literal string "ledger" would miss a differently-named base-comparison file reintroduced elsewhere in the same script. |

## ISO 25010 coverage matrix

`N/A` cells are justified the same way F01's plan justified them: this is
offline curator/CI tooling with no production request path or end-user
journey (`uat-plan.md`'s "Not a product concern here"), consistent with
REQ-NF-003/004/005.

| AC | Functional | Performance | Compat | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001, AC-016 | ✅ TC-030 | N/A | ✅ TC-030 (no-submodule CI compat) | N/A | N/A | N/A | ✅ TC-030 (schema version gate) | N/A |
| AC-002 | ✅ TC-030 | N/A | N/A | N/A | ✅ TC-030 | N/A | N/A | N/A |
| AC-003 | ✅ TC-030 | N/A | N/A | N/A | N/A | N/A | ✅ TC-030 (rule vs. list distinction) | N/A |
| AC-004 | ✅ TC-030 | N/A | N/A | N/A | N/A | ✅ TC-030 (leak surface) | N/A | N/A |
| AC-005 | ✅ TC-030 | N/A | N/A | ✅ TC-030 (named failing field) | N/A | N/A | N/A | N/A |
| AC-006 | ✅ TC-032 | N/A | N/A | N/A | ✅ TC-032 | N/A | N/A | N/A |
| AC-007 | ✅ TC-033 | N/A | N/A | ✅ TC-033 (named failing check) | ✅ TC-033 | N/A | N/A | N/A |
| AC-008 | ✅ TC-034 | N/A | N/A | N/A | ✅ TC-034 | N/A | N/A | N/A |
| AC-009 | ✅ TC-030 | N/A | N/A | N/A | ✅ TC-030 | N/A | N/A | N/A |
| AC-010, AC-011 | ✅ TC-031 | N/A | ✅ TC-031 (same shape, two adapters) | N/A | N/A | N/A | ✅ TC-031 (no per-language consumer branch) | ✅ TC-031 (Go + Python parity) |
| AC-012 | ✅ TC-031 | N/A | N/A | N/A | N/A | ✅ TC-031 (branch-leak surface) | ✅ TC-031 | N/A |
| AC-013, AC-019 | ✅ TC-030, TC-035 | N/A | N/A | N/A | N/A | N/A | ✅ TC-035 (opaque comparison, no key reads) | N/A |
| AC-014 | ✅ TC-036 | N/A | N/A | N/A | ✅ TC-036 (threshold exact) | N/A | N/A | N/A |
| AC-015 | ✅ TC-037 | N/A | ✅ TC-037 (submodule states) | N/A | N/A | N/A | ✅ TC-037 (lint clean) | N/A |
| AC-017 | ✅ TC-038 | N/A | N/A | N/A | ✅ TC-038 (I-01 callers unaffected) | N/A | ✅ TC-038 (frozen interface) | N/A |
| AC-018 | ✅ TC-039 | ✅ TC-039 (warm-cache offline completion) | N/A | N/A | ✅ TC-039 | ✅ TC-039 (no silent network fallback) | N/A | ✅ TC-039 (both Linux-namespace and portable proxy-poison mechanisms independently exercised, not one assumed equivalent to the other) |
| AC-020 | ✅ TC-040 | N/A | N/A | N/A | ✅ TC-040 (full suite green) | N/A | N/A | N/A |
| AC-021 | ✅ TC-041 | N/A | N/A | N/A | N/A | ✅ TC-041 (no hidden ledger state) | ✅ TC-041 | N/A |

No coverage gaps: every non-`N/A` cell cites a TC; every `N/A` cell is
justified by this feature's offline-tooling, no-user-journey nature.

## Observability design

Same posture as F01: no metrics/trace spans, because this is offline
curator/CI tooling. Observability means the scripts' own machine-readable
output, which F06/F07/F08 and a human curator both depend on.

| Behavior | Log / stdout evidence | Trace/metric | Test assertion |
|---|---|---|---|
| Schema validation failure | `TestTC030_...` subtests report the exact failing field per REQ-F-015 case, not a generic "invalid" message | N/A | TC-030 |
| Admission verdict per candidate | `admit-scenario.sh` prints one JSON record per candidate, sorted by `scenario_id`, naming pass/fail per check and the exact failing check on rejection, plus `base_outcome`/`reference_outcome`/`toolchain_identity` on success | N/A | TC-032, TC-033, TC-034 |
| Predicate evaluation | `eval-predicate.sh` prints the boolean result plus the operand values it read from `test`/`lint` output (never a ledger path) | N/A | TC-036, TC-041 |
| Adapter capability output | Each `adapter.sh <capability>` call emits exactly one JSON document to stdout per the fixed shape, with exit status `0` for "ran successfully" distinct from the JSON's own pass/fail content | N/A | TC-031 |
| Toolchain identity | `adapter.sh identity` prints the ordered key/value list; `admit-scenario.sh` prints it again inside `admission.toolchain_identity` | N/A | TC-030 (cross-encoding), TC-035 (changes under a changed toolchain) |
| Fixture-base verification | `verify-fixture-py-base.sh` prints `CLEAN` plus the full-suite pass count on success, or the first offending file/test on failure | N/A | TC-040 |
| Offline/missing-tool failure | Fails naming the missing pinned interpreter/tool, not a generic error | N/A | TC-039 |

No new instrumentation beyond structured script/test output is required or
permitted.

## Cross-feature contract tests (I-04, I-01)

### Produces: I-04

Carried verbatim from `spec.md`'s Cross-feature interactions section:

| I-## | Producer | Consumers | Shape source | Contract test pointer | TC |
|---|---|---|---|---|---|
| I-04 | E40-F05 | E40-F06, E40-F07, E40-F08 | `architecture.md#lifecycle-scenario-package-contract` | `tests/contracts/e40_i04_scenario_contract_test.go#TC-030` | TC-030 |

Gate mode: `live` (per `E40-interaction-map.md` and `spec.md`'s own
Cross-feature interactions section; no `contract-only`/staged row is
declared for I-04, so no `review_basis` value is fabricated here). Closure
owner: E40-F05 code-review owner (`spec.md`). Required UAT evidence: UAT-08
— "all four admitted families load with stable identity, correct adapter
selection, final predicate, and the exact applicable/non-applicable stage
matrix; a malformed or non-runnable package is rejected with the failing
field named." TC-030 (identity/stage-matrix/rejection) plus TC-031/TC-032
(adapter selection, admitted-set correctness) together are this evidence.
When E40-F06, E40-F07, or E40-F08 are decomposed, each must reuse TC-030
verbatim as the shape source rather than writing a second reader — no twin
test, matching `spec.md`'s explicit instruction.

### Consumes: I-01

| I-## | Producer | Consumer | Shape source | Contract test pointer | TC |
|---|---|---|---|---|---|
| I-01 | E40-F01 | E40-F05 | `architecture.md#corpus-and-oracle-contract` | `tests/contracts/e40_i01_corpus_contract_test.go#TC-001` | TC-001 (existing, unmodified) |

F05 consumes I-01's admission/versioning/held-back-oracle *principles*, not
its Go-typed fields (ADR-F05-01) — there is no shared payload F05 reads
through TC-001's own assertions, so F05 does not extend or re-run TC-001.
The non-regression obligation instead is: `bench/corpus/corpus.yaml`,
`checkout-fixture.sh`, `build-ledgers.sh`, `diff-ledgers.sh`, and TC-001
itself are unmodified by this feature. TC-038 is F05's evidence for this —
it re-runs F01's own TC-004/TC-006/TC-007 (which transitively depend on
`checkout-fixture.sh` and the ledger scripts) unmodified and requires them
to still pass, rather than writing a second, F05-owned assertion over I-01's
shape. Gate mode: `live`.

## Cross-epic integration tests (X-##)

`spec.md`'s Cross-epic integrations section and `E40-cross-epic-map.md` both
state F05 produces, consumes, and validates **no X-## row** (X-07 → F02,
X-08 → F04, X-09 → F06, X-10 → F07, X-11/X-13 → F08, X-12 → F09). This plan
does not invent an X-## row, a contract pointer, or a deferral where none is
assigned. The one adjacency `spec.md` names — I-04's evaluator/usage
reference fields staying opaque so F06 can decode X-09 later without F05
having committed the shape — is exercised by TC-030's assertion that
`evaluator_only` fields are typed as opaque paths, not decoded structures;
no separate X-09 test belongs to F05.

## Integration scenarios

| Scenario | Boundary | Epic UAT contribution | Test evidence |
|---|---|---|---|
| Scenario package → future lifecycle consumers (I-04) | `bench/scenarios/*` file shape → E40-F06/F07/F08's not-yet-built readers | UAT-08 | TC-030 pins the shape those features must consume with no manual step |
| Admission gate → six named rejection checks | `admit-scenario.sh` → mutated candidates | UAT-08 ("a malformed or non-runnable package is rejected with the failing field named") | TC-033 |
| Adapter boundary → two language implementations | `adapter.sh` (Go, Python) → generic scenario/evidence/admission scripts | UAT-08 ("correct adapter selection"); epic G8 | TC-031, TC-032 |
| Fixture and submodule → shark's own quality gate | `bench/` tree → root `make fmt && make lint && make test` and `go list ./...` | Non-functional: repo hygiene, not a UAT scenario | TC-037 |
| Frozen I-01 interface → new sibling script | `checkout-fixture.sh` (unchanged) vs. `checkout-scenario-fixture.sh` (new) | Non-functional: no regression to Phase 1 (UAT-01, UAT-02, UAT-05-07 transitively) | TC-038 |
| Curator re-run → REQ-NF-004 reproducibility claim | `admit-scenario.sh`, `eval-predicate.sh`, `verify-fixture-py-base.sh` re-run on a clean checkout | UAT-08 (stable identity) | TC-034, TC-039, TC-040 |

Two verification-plan rows are intentionally **not** test cases, matching
`spec.md`'s own Verification plan table's stated method:

- **REQ-NF-005** (scenario tooling never touches the live shark database,
  `.sharkconfig.json`, or the live repository working tree, and never
  invokes shark project-initialisation commands) — verified by code review
  of `bench/scripts/*.sh` and `bench/adapters/*/adapter.sh` confirming no
  such invocation exists, matching `spec.md`'s own "Diff review" method, not
  an automated test.
- **`bench/README.md`'s new documentation sections** (I-04 schema and
  adapter capability contract) — this is prose describing an already-tested
  shape (TC-030, TC-031 assert the real shape); the documentation itself is
  reviewed for accuracy against the shape, not independently tested, per the
  Workflow's "Prompt-only changes" guidance applied to a docs-only delta.

## Test infrastructure

**Existing patterns to reuse:**
- `tests/contracts/e40_i01_corpus_contract_test.go` and
  `tests/contracts/e39_interactions_test.go` establish the
  repository-root-relative artifact-reading helper style (`filepath.Abs`,
  then `os.ReadFile`) and the `TestTC0NN_...` naming convention — TC-030
  follows this exactly, continuing the epic's sequential TC numbering (F01
  used TC-001-TC-013; the highest TC number in committed epic docs today is
  TC-029, so TC-030 is the next free slot, matching `spec.md`'s own explicit
  reference).
- `bench/scripts/tests/run-all.sh` and its `tcNNN_<description>_test.sh`
  naming convention (`bench/scripts/tests/tc003_clean_checkout_test.sh` …
  `tc020_zero_go_change_test.sh`) is the pattern TC-031 through TC-041's bash
  test scripts follow, registered in the same `run-all.sh`.
- `bench/scripts/checkout-fixture.sh` and `.gitmodules`'s
  `bench/fixture-repo` entry are the structural precedent
  `checkout-scenario-fixture.sh` and the new `bench/fixture-py` submodule
  follow, per ADR-F05-06 and ADR-F05-09.
- `internal/runner/dispatcher.go`'s `AgentDispatcher` interface is the
  pattern (not code) the two `adapter.sh` scripts' shared capability
  contract mirrors — nothing under `internal/runner` is imported or tested
  by this feature; it is cited here only because `spec.md` names it as the
  adapter's design precedent, not because F05 adds a test against it.

**New test infrastructure needed (this feature's own deliverables):**
- `tests/contracts/e40_i04_scenario_contract_test.go` — one Go file,
  `package contracts`, containing `TestTC030_I04ScenarioPackageContract` and
  its table-driven malformed-package subtests (AC-005). Per REQ-NF-001, this
  is the **only** Go file this feature adds.
- `tests/contracts/testdata/e40_i04/` — the 14 malformed-package fixtures
  AC-005/TC-030 requires, one per row of this authoritative case table:

  | # | Case | Expected failing field named |
  |---|---|---|
  | 1 | Unknown `entity_family` | `entity_family` |
  | 2 | Unregistered `fixture_id` | `fixture.fixture_id` |
  | 3 | Unregistered `adapter.name` | `adapter.name` |
  | 4 | `resource_policy.max_cost_usd` missing | `resource_policy.max_cost_usd` |
  | 5 | `resource_policy.max_cost_usd` non-positive (`0`) | `resource_policy.max_cost_usd` |
  | 6 | `resource_policy.max_wall_clock_seconds` missing | `resource_policy.max_wall_clock_seconds` |
  | 7 | `resource_policy.max_wall_clock_seconds` non-positive (`0`) | `resource_policy.max_wall_clock_seconds` |
  | 8 | `resource_policy.max_generated_tasks` missing | `resource_policy.max_generated_tasks` |
  | 9 | `resource_policy.max_generated_tasks` non-positive (`0`) | `resource_policy.max_generated_tasks` |
  | 10 | Prelude stage with no explicit boolean | the named `D0N` stage |
  | 11 | `false` prelude stage with no `reason` | the named `D0N` stage's `reason` |
  | 12 | Family-invariant violation (`bug` with one stage `true`) | the offending family and stage |
  | 13 | Unknown `final_predicate.kind` | `final_predicate.kind` |
  | 14 | Predicate kind not permitted for declared family (`f2p_p2p` on a `feature` package) | `final_predicate.kind` |

  Three further REQ-F-015 cases reuse fixtures already required elsewhere
  rather than adding new rows to this table: **"missing predicate operand
  path"** is covered by the AC-001 negative case above ("a predicate-operand
  path that does not exist on disk"); **"`input.agent_visible` resolving
  inside `evaluator/`"** is covered by the AC-004 negative case above (a
  malformed fixture whose `input.agent_visible` points inside `evaluator/`);
  and **the forbidden-I-01-field case** is covered by the AC-001 fixture
  added above (injecting `go_version`/`golangci_lint_version`/`p2p_sets`/
  `reference_patch_path`). All three live in the same
  `tests/contracts/testdata/e40_i04/` tree and the same `TestTC030_...`
  table-driven harness — REQ-F-015's full enumerated list is satisfied
  across these 14 dedicated fixtures plus these 3 shared ones (2 owned by
  AC-001, 1 owned by AC-004), not by inflating the dedicated count further.
- `bench/scripts/tests/tc031_adapter_conformance_test.sh` — already named in
  `spec.md`'s component table; drives TC-031.
- `bench/scripts/verify-fixture-py-base.sh <base_sha>` — **new, not yet in
  `spec.md`'s component table** (see "Test infrastructure gaps" below);
  drives TC-040.
- Six test-time-mutated candidate fixtures for TC-033 — built from a copy of
  `py-bug-due-date-boundary` at test-invocation time (not committed
  separately), the same "transient candidate" technique F01's TC-005 used
  for its own rejection-branch coverage that a committed negative fixture
  would otherwise require duplicating four more times.
- A second **real, pinned** Python interpreter environment for TC-035 (e.g. a
  second minor version installed alongside the pinned one, both resolvable
  via the fixture's own lock mechanism) — a `PATH`-stubbed binary reporting a
  fabricated version string is explicitly disallowed here (unlike F01's
  TC-011, which stubs a version string to test a *comparison* guard; TC-035
  tests whether `identity` actually *interrogates* the live toolchain, which
  a stub would falsify by construction).
- `bench/README.md`'s new "I-04 scenario package schema" and "Adapter
  capability contract" sections must name the exact Tier 2 curator command
  sequence (at minimum: `git submodule update --init` for both submodules,
  then `admit-scenario.sh`, `eval-predicate.sh`,
  `verify-fixture-py-base.sh`) so REQ-NF-004's reproducibility claim has one
  documented invocation, not an implied one.

### Test infrastructure gaps

**AC-020's owning script is not yet named in `spec.md`'s component table.**
`spec.md` states AC-020 is "Verified by a bench script, not by TC-030" but
the Architecture "Component changes" table lists no script that performs
the full-suite-green check independent of admission check (b)'s
`p2p_selection`-scoped check. This plan names the resolution concretely
(`bench/scripts/verify-fixture-py-base.sh <base_sha>`, TC-040) so task
decomposition has an unambiguous deliverable. Per the workflow's own rule
("codex returning CONCERNS... routes to revise now or document why deferred
with explicit owner/timeframe, not to NEEDS_REFINEMENT"), and because the
resolution here is fully specified (exact script name, argument shape,
assertions, and owning test), this is recorded as a named, owned item for
the spec/architecture owner to fold into `spec.md`'s component table before
or during task decomposition, not a blocker to APPROVED status:

- **Owner:** the E40-F05 spec/architecture owner.
- **Trigger:** before task decomposition creates the fixture-content
  verification task.
- **Exact ask:** add a row to `spec.md`'s Architecture component-changes
  table for `bench/scripts/verify-fixture-py-base.sh`, and note under
  REQ-F-016/AC-020 which script owns the full-suite-green check (mirroring
  this section).
- **Interim status:** until that lands, this test plan's resolution above is
  the authoritative source for the script's contract; task decomposition may
  proceed from it.

**`spec.md`'s Final predicate vocabulary table wording disagrees with its own
paragraph on base-relative vs. absolute P2P semantics** (see
"Acceptance-criteria review" above for the full analysis). Non-blocking, same
posture as the AC-020 item above — the paragraph and ADR-F05-10 are
authoritative, and this plan's ACs/TCs already test the absolute reading
exclusively.

- **Owner:** the E40-F05 spec/architecture owner.
- **Trigger:** before task decomposition creates the `eval-predicate.sh`
  implementation task, so the developer reads only the unambiguous
  paragraph/ADR wording, not the three "regression"-worded table rows.
- **Exact ask:** reword `f2p_p2p`, `acceptance_tests`, and
  `p2p_plus_rule_drop`'s "True when" cells to match `child_oracles_union`'s
  absolute framing ("every entry the P2P selection resolves to is `pass`"),
  removing "regressed"/"no regression" language from the table.
- **Interim status:** this test plan's TC-036 and TC-041 already implement
  and verify the absolute reading; no test design changes ride on this
  wording fix landing.

## Codex test-plan red-team

**Verdict:** CONCERNS (second pass), all findings closed by revision —
**Issues raised:** 9 (first pass, verdict FAIL) + 2 (second pass, verdict
CONCERNS)
**Issues addressed before development:** 11
**Issues deferred:** 1, named and owned (AC-020's `verify-fixture-py-base.sh`
component-table addition — codex's second pass explicitly confirmed this
deferral does not block APPROVED status)

The local read-only Codex CLI (`codex exec -s read-only`, high reasoning
effort) reviewed the drafted plan together with `spec.md` in two passes.

**First pass — FAIL.** Nine findings (condensed; full run against the
pre-revision draft):

1. **AC-001:** TC-030 never rejected the four forbidden I-01 field names
   (`go_version`, `golangci_lint_version`, `p2p_sets`,
   `reference_patch_path`), so REQ-F-001's "no field is inherited by name"
   was asserted by the real packages' omission, not proven mechanically.
2. **AC-005:** the case count and the "11 named cases" label didn't match
   what REQ-F-015 actually enumerates, and the three `resource_policy`
   ceilings weren't each independently varied for missing/non-positive.
3. **AC-007/TC-033:** labeled "six candidates" while needing seven, since
   check (b) has two independent trigger conditions (empty resolution vs. a
   failing entry at base).
4. **AC-010/AC-012/TC-031:** running six capabilities didn't prove the verb
   set is *closed*, and the grep missed indirect language branching via
   `fixture_id`/`adapter.name` conditionals.
5. **AC-013/TC-035:** the "second toolchain environment" was ambiguous
   between a real environment and a `PATH` stub, and never re-ran the full
   admission gate under each to prove end-to-end pin propagation.
6. **AC-014/TC-036 — BLOCKER:** claimed all four `final_predicate` kinds but
   only exercised `p2p_plus_rule_drop` and `child_oracles_union`; `f2p_p2p`
   and `acceptance_tests` had no executable true/false cases at all.
7. **AC-018/TC-039:** Portability was marked N/A despite a Linux-only
   `unshare` path and an untested "portable fallback."
8. **AC-020:** confirmed (definitively, on direct re-read of REQ-F-012(b)
   and REQ-F-016) as a genuine coverage gap — admission check (b) proves
   only the `p2p_selection` subset green at base, never the fixture's full
   suite REQ-F-016 requires.
9. **AC-021/TC-041:** `grep 'ledger'` catches only that literal spelling,
   not a differently-named base-comparison file or another read path.

**Revision applied between passes:** AC-001 gained a forbidden-field
rejection fixture; AC-005 gained an authoritative 14-case table with all
three ceilings independently varied for missing/non-positive; AC-007/TC-033
corrected to seven named candidates; AC-010/AC-012/TC-031 gained a
closed-vocabulary rejection assertion (a seventh, undefined capability name
must be rejected by both adapters) and a broadened branch-detection check;
AC-013/TC-035 now requires two real, pinned, cache-warmed interpreter
environments (stubs explicitly disallowed) with a full `admit-scenario.sh`
re-run under each; AC-014/TC-036 was rewritten to exercise all four
predicate kinds with base/reference-applied/P2P-regression states;
AC-018/TC-039 now runs both isolation mechanisms independently and
Portability moved from N/A to covered; AC-020's resolution
(`verify-fixture-py-base.sh`, TC-040) was retained with an explicit named
deferral; AC-021/TC-041 replaced the narrow grep with a static
argument-trace requirement plus a broadened naming-convention check.

**Second pass — CONCERNS**, verifying the revision directly:

1. **AC-014/TC-036 partly closed:** the four-kind matrix was present, but
   (a) the tech-debt boundary specified an impossible `max_remaining - 1`
   state, since the seed's `max_remaining: 0` is already the domain floor
   for a non-negative issue count, and (b) only `f2p_p2p` carried a
   dedicated P2P-regression mutation — the other three kinds' absolute P2P
   clauses (ADR-F05-10) were unproven.
2. **AC-020 confirmed non-blocking:** codex explicitly stated the named
   deferral (exact script, invocation, assertions, owner, and trigger) "need
   not prevent APPROVED once the remaining TC-036 concern is fixed" — the
   first-pass FAIL did not make this finding intrinsically blocking.
3. All other eight first-pass findings were confirmed "materially
   adequate" as revised: forbidden-field rejection, 14-case ceiling
   coverage, seven AC-007 candidates, closed-verb and indirect-branch
   checks, real pinned toolchains with end-to-end admission re-runs, both
   offline mechanisms, and the broadened argument-trace check.

**Final revision applied:** TC-036's tech-debt boundary corrected to two
states (count `= 1` false, count `= 0` true — no sub-floor state asserted),
and every one of the four predicate kinds now carries its own independent
P2P-regression mutation (`f2p_p2p`, `acceptance_tests`,
`p2p_plus_rule_drop`, and `child_oracles_union` each has a dedicated
"reference/threshold satisfied but a `p2p_selection` entry broken" false
case), closing the second pass's sole remaining concern. This plan is not
re-submitted for a third codex pass: the fix is a narrow, mechanical
correction to a state already validated as the right shape in pass two (a
boundary-value floor correction and a uniform extension of an existing
per-kind mutation pattern to the three kinds that lacked it), not a new
design surface.

## Recommendations

- [x] Ready for development — every AC in `spec.md` has a named test case,
  technique, ISO 25010 row, and caller-path contract, red-teamed by codex
  across two passes with all findings closed or explicitly deferred with an
  owner and trigger. The one implementation-surface gap found in the
  feature-level coverage check (`verify-fixture-py-base.sh` for AC-020) is
  resolved with a concrete script name, argument shape, and owning test case
  (TC-040); task decomposition must create an explicit task for it, and the
  spec/architecture owner must fold it into `spec.md`'s component-changes
  table before that task is dispatched.
- [ ] Needs BA refinement.
- [ ] Needs tech refinement.
