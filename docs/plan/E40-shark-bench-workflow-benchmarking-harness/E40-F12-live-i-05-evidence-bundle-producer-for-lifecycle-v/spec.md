---
feature_key: E40-F12-live-i-05-evidence-bundle-producer-for-lifecycle-v
epic_key: E40
title: Live I-05 evidence bundle producer for lifecycle-v2 runs
spec_schema: 1
tier: standard
related-docs:
  - bench/evidence/i05-schema.yaml
  - bench/README.md
  - docs/plan/tech-debt/TD-132.md
---

# Specification: Live I-05 evidence bundle producer for lifecycle-v2 runs

Combined requirements + architecture specification. Business context is **not**
restated here.

- Business context and scope: **see epic PRD** (`../epic.md`), Phase 2 /
  lifecycle-v2 tranche.
- System-level decisions: **see** `../architecture.md#stage-evidence-and-isolation-contract`
  and `../architecture.md#lifecycle-run-record-contract`.
- Brownfield evidence and reuse decisions: **see** `research-report.md`
  ("Capability map", "Findings", "Decisions"). Its five Decisions are adopted
  verbatim and are not re-argued below.

---

## 0. Product critical-path guard

| File | State | Report |
|---|---|---|
| `docs/product/D01-vision-statement.md` | absent | `unresolved prerequisite: docs/product/D01-vision-statement.md missing` |
| `docs/product/D02-success-criteria.md` | absent | `unresolved prerequisite: docs/product/D02-success-criteria.md missing` |
| `docs/plan/product-delivery-roadmap.md` | absent | `unresolved prerequisite: docs/plan/product-delivery-roadmap.md missing` |
| `docs/plan/product-critical-path.md` | absent | `unresolved prerequisite: docs/plan/product-critical-path.md missing` |

All four are missing, so no roadmap gate can be named, no relationship to a gate
can be asserted, and no gate-advancement evidence can be linked. The guard is
advisory and does not stop this specification. Consequences carried forward:

- **Gate name** — unavailable (`docs/plan/product-critical-path.md` absent).
- **Relationship to the gate** — unassessable.
- **Executable advancement evidence** — the closest available live-golden-path
  evidence is `bench/scripts/e40-benchmark.sh preflight` against the six
  lifecycle-v2 scenario packages, which today reports every scenario blocked on
  `scenario_roots.<id>.i05_bundle_dir is not configured` (P5). REQ-F-016 makes
  clearing that blocker on a real run this feature's own executable evidence.
- **Side-quest call** — **proceed**. No named gate exists to move, but the work
  removes the single hard blocker in front of E40's first real six-family
  lifecycle-v2 baseline, and it closes two already-filed, still-open findings
  (TC-082, TD-132). Deferring keeps 0 of 6 families runnable.

---

## 1. Requirements

Incremental over the epic. Every requirement below traces to an epic PRD
requirement already discharged in shape by E40-F06 (I-05 contract) and E40-F08
(the lifecycle runner); this feature adds only the missing *production* of that
contract. In particular, E40-F08 `spec.md` REQ-F-010 ("After each applicable
dispatch, the runner MUST write or reference the I-05 stage snapshot, time
ledger, candidate snapshot, artifact producer/consumer events, provider usage,
and evaluator-access ordering") is the parent requirement this feature closes;
nothing here is new epic scope.

### 1.1 Functional requirements

#### REQ-F-001 — Bundle directory is an explicit driver input

`bench/scripts/run-lifecycle.sh` MUST accept a new option
`--i05-bundle-dir <dir>` and MUST write the I-05 bundle there.

- **Priority**: Must-Have
- The option is **required** in `--mode live`; it is **optional** in
  `--mode contract` and `--mode dry-run` (offline suites that do not configure
  it keep their current behavior unchanged).
- `--mode resolve-route` MUST reject the option and MUST NEVER write a bundle.
  Route resolution output is already fenced off from live stage evidence by
  `verify-stage-evidence.sh`'s `reject_route_resolution_only()`
  (`bench/scripts/verify-stage-evidence.sh:620`); this keeps that fence intact
  at the producer end too.
- When the option is absent in `live` mode the run MUST fail before the first
  dispatch, naming the missing option — never run and silently produce no
  evidence.
- **Acceptance**: AC-001, AC-014.

#### REQ-F-002 — `bundle.json` is written and kept current

The producer MUST write `<i05_bundle_dir>/bundle.json` before the first
dispatch and MUST rewrite it after every dispatch and at run termination.

- **Priority**: Must-Have
- The ten fields `bench/README.md` "Bundle layout (I-05)" declares are all
  written and none is omitted: `schema_version`, `scenario`, `run_id`, `roots`,
  `stage_matrix_source`, `stages`, `terminal_status`, `stop_outcome` (optional),
  `publication_eligible`, `ineligibility_reasons`. `schema_version` MUST be read
  from `bench/evidence/i05-schema.yaml`'s `schema_version`, never hard-coded.
- **Additive join references** required by REQ-F-014 are the only fields written
  beyond those ten: top-level `scenario_id`, `scenario_version`, and
  `dispatches`. They are required because `evaluate-lifecycle.sh`
  `validate_identity_join()` resolves each join field as
  `i05.identity.<field>` then `i05.<field>` at **top level** (line 226-232) — a
  `scenario_id` nested inside the documented `scenario` object does not satisfy
  it. They are additive-only: `verify-stage-evidence.sh` `main()`
  (line 983) performs no exclusive field inventory on `bundle.json` and rejects
  no unknown top-level key, so no shipped validator is broken by their presence.
  No *other* field is invented, and no I-05 vocabulary is extended.
- `scenario` is `{scenario_id, scenario_version, entity_family}` copied verbatim
  from the I-04 package.
- `stages[]` is the ordered index `{dispatch_ordinal, stage_key, stage_category,
  snapshot_path, snapshot_digest}`; `dispatch_ordinal` is unique within the
  bundle and equals the I-07 `dispatches[].ordinal` for the same dispatch.
- **Acceptance**: AC-002, AC-003.

#### REQ-F-003 — One immutable stage snapshot per dispatch

After each `spawn_agent` dispatch completes (success or failure), the producer
MUST write `<i05_bundle_dir>/stages/<dispatch_ordinal>-<stage_key>.json`
carrying every field in `bench/README.md`'s "Stage-snapshot field reference",
plus the top-level `provider` field `bench/evidence/i05-schema.yaml`'s header
declares as required.

- **Priority**: Must-Have
- `snapshot_digest` is `sha256` over the canonical serialization of the snapshot
  **excluding** `snapshot_digest` itself, using the same canonical form as
  `run-lifecycle.sh`'s existing `canonical_digest()`
  (`json.dumps(..., sort_keys=True, separators=(",",":"), ensure_ascii=False)`),
  so recomputation reproduces it exactly.
- A snapshot is written once and never rewritten. A later dispatch never edits
  an earlier snapshot.
- `candidate` is written only for snapshots whose `stage_category` is `code` or
  `review`, and carries the six REQ-F-006 identity fields plus `test_suite_ids`
  and `test_suite_dir` that `replay-stage-evidence.sh` reads.
- **Acceptance**: AC-004, AC-005, AC-006.

#### REQ-F-004 — `stage_category` is derived through a closed table

The producer MUST resolve each dispatched step's `stage_category` from the
closed table in §2.4 and MUST NOT guess, default, or hard-code a value.

- **Priority**: Must-Have
- The step's workflow **phase** is read from `shark admin workflow list
  <workflow-level> --json` (`statuses[].phase`), invoked at most once per
  workflow level per run and cached in-process.
- A dispatched status with no phase, or a phase absent from the §2.4 table, MUST
  produce a snapshot whose `errors[]` carries `{kind: "unknown_stage_category"}`
  (the existing `bench/evidence/i05-schema.yaml` vocabulary) naming the status
  and phase. The producer MUST NOT substitute a plausible category.
- **Acceptance**: AC-007.

#### REQ-F-005 — The time ledger records only observed time

Each snapshot's `time_ledger` MUST reconcile to `[stage_start, stage_end)` under
`verify-stage-evidence.sh`'s rules, populated exclusively from intervals the
driver actually observed, per the closed table in §2.5.

- **Priority**: Must-Have
- `provider_active` MUST be populated **only** from explicit provider-active
  intervals reported by the worker envelope (§2.5). When the envelope reports
  none, the adapter window is recorded as `unclassified` and `provider_active`
  is absent or empty. Unattributable time is never routed to `provider_active`.
- Every interval is half-open `[start, end)` in integer nanoseconds on a single
  monotonic-derived timeline, non-overlapping across all categories, contained
  in the stage window.
- `reconciliation_epsilon_ns` MUST be a fixed, declared constant (§2.5), not
  widened per run to absorb whatever residual appeared.
- **Acceptance**: AC-008, AC-009.

#### REQ-F-006 — `access.jsonl` exists and is append-only

The producer MUST create `<i05_bundle_dir>/access.jsonl` at run start and MUST
only ever append to it.

- **Priority**: Must-Have
- The lifecycle driver itself performs no evaluator access during dispatch, so
  the file is legitimately empty at the end of a clean run; it MUST still exist
  so `verify-stage-evidence.sh --grant-access`'s existing
  `append_access_event()` (`bench/scripts/verify-stage-evidence.sh:410`) has a
  file to append to and so the bundle layout is complete.
- Any `evaluator_access` entry a snapshot carries MUST also be appended here.
- **Acceptance**: AC-010.

#### REQ-F-007 — `transcripts/` subdirectory is materialized

The producer MUST create `<i05_bundle_dir>/transcripts/` as a real directory
(never a symlink) and MUST write each dispatch's bounded worker transcript
artifact into it.

- **Priority**: Must-Have
- Rationale is load-bearing and non-obvious: `bench/scripts/lib/retain_pair`
  calls `copy_dir_artifact("transcripts", i05_bundle_dir + "/transcripts")`
  (line 381) and `refuse_missing_source()` refuses the **whole** pair, writing
  no `manifest.json`, when that directory does not exist. The documented I-05
  bundle-layout table does not mention it (research-report Finding 6).
- Transcript content is bounded by the existing `bounded()` helper and MUST NOT
  contain raw prompt bytes or provider credentials (REQ-NF-003).
- **Acceptance**: AC-011.

#### REQ-F-008 — `roots` carries a real `agent_fixture_checkout`

`bundle.json`'s `roots.agent_fixture_checkout` MUST be an existing directory
path on the machine running the evaluation.

- **Priority**: Must-Have
- Rationale: `evaluate-lifecycle.sh` `run_oracle()`
  (`bench/scripts/evaluate-lifecycle.sh:550`) skips the held-back oracle and
  appends a `missing_oracle` invalidity reason whenever this path is absent or
  not a directory; the missing `.oracle.json` sidecar then makes
  `retain_pair`'s `copy_artifact("oracle.json", …)` refuse the pair. Both
  failures are silent-looking and neither names the root as the cause.
- All three roots MUST be declared with their `i05-schema.yaml` `worker_access`
  values and MUST be pairwise non-nested absolute paths.
- **Acceptance**: AC-012.

#### REQ-F-009 — Stop-outcome eligibility triad

`bundle.json` MUST carry the REQ-F-014 triad consistently with the I-07 record
the same run writes.

- **Priority**: Must-Have
- Closed lifecycle table, §2.6. On a clean terminal run `stop_outcome` is absent
  and `publication_eligible` is `true`. On any non-clean termination
  `stop_outcome` is one of the ten `i05-schema.yaml` values,
  `publication_eligible` is `false`, and `ineligibility_reasons[]` is non-empty.
- The bundle's `stop_outcome` MUST equal the I-07 record's `outcome.terminal`
  whenever that value is in the closed set.
- **Acceptance**: AC-013.

#### REQ-F-010 — Partial evidence survives an aborted run

When a run terminates early (any `stop_outcome`), every snapshot already written
MUST remain present, readable, and indexed in `bundle.json`.

- **Priority**: Must-Have
- The producer writes incrementally inside the existing per-dispatch loop; it
  MUST NOT defer all writes to a single end-of-run flush, and MUST NOT delete or
  truncate earlier snapshots on failure.
- **Acceptance**: AC-014.

#### REQ-F-011 — Producer-owned reset, bounded to producer-owned entries

At run start, and only in the modes of REQ-F-001, the producer MUST remove
exactly the four entries it owns inside `i05_bundle_dir` — `bundle.json`,
`stages/`, `access.jsonl`, `transcripts/` — and MUST NOT remove the directory
itself or any other entry within it.

- **Priority**: Must-Have
- Rationale: `i05_bundle_dir` is an operator-configured, preflight-validated
  directory (§2.3, ADR-F12-02). Without the reset, repetition N of a scenario
  can inherit repetition N-1's snapshots and retain a mixed bundle; with an
  unbounded reset the producer could delete operator content it does not own.
- The reset MUST refuse and fail the run if any of the four entries is a symlink
  (matching `retain_pair`'s source-side symlink posture).
- **Acceptance**: AC-015.

#### REQ-F-012 — TD-132: `content_root` and a matching content digest

The producer MUST declare a `content_root` and MUST compute
`identity.shark_content_digest` with the same scheme
`evaluate-lifecycle.sh`'s `content_digest()` uses, so the crosscheck fires and
passes on a real run.

- **Priority**: Must-Have
- `content_root` is the **installed Shark-data canonical content tree inside the
  scratch Shark project** (ADR-F12-03).
- `shark_content_digest` MUST be replaced with the walk-based digest of
  `content_root`, matching `bench/scripts/evaluate-lifecycle.sh:102`'s
  `content_digest()` byte-for-byte in traversal order and separator bytes.
- The digest scheme change MUST be marked in the record so a pre-change retained
  baseline can never be silently compared against a post-change one: the I-07
  `identity` gains `content_digest_scheme: "walk_v1"` and any record lacking that
  marker is treated as the legacy whole-repo-tree scheme.
- A half-wiring — declaring `content_root` while leaving the legacy
  `git rev-parse HEAD^{tree}` digest in place — is explicitly forbidden: it
  converts a dead check into a guaranteed `identity_mismatch` and makes every
  pair ineligible.
- **Acceptance**: AC-016, AC-017.

#### REQ-F-013 — TC-082: the driver hands retention a real source

`bench/scripts/run-lifecycle-batch.sh` and `bench/scripts/run-review-comparison.sh`
MUST pass the configured `i05_bundle_dir` through to the lifecycle driver so the
bundle exists before `retain_pair` is invoked with it.

- **Priority**: Must-Have
- Today `dispatch_pair()` (`bench/scripts/run-lifecycle-batch.sh:822`) receives
  `i05_bundle_dir` and uses it for evaluation and retention but never passes it
  to `"$RUN_LIFECYCLE_BIN"`. `dispatch_gate()`
  (`bench/scripts/run-review-comparison.sh:718`) has the same shape.
- No change to `bench/scripts/lib/retain_pair` is in scope
  (research-report Decision 5): its fail-closed behavior is already correct and
  only lacked a real source.
- **Acceptance**: AC-018.

#### REQ-F-014 — The I-05 half of the evaluator join is complete

`bundle.json` MUST carry every join reference `evaluate-lifecycle.sh`
`validate_identity_join()` reads from the I-05 side that this feature owns:
`run_id`, `scenario_id`, `scenario_version`, `roots`, and a `dispatches`
reference list.

- **Priority**: Must-Have
- The `dispatches` list mirrors the I-07 record's dispatch references so the
  digest-equality check has a defined counterpart.
- `dispatch_id` / `dispatch_ordinal` at **record** level are deliberately **not**
  fabricated: they are dispatch-scoped names on a per-run bundle, and
  `../architecture.md#stage-evidence-and-isolation-contract` gives the bundle no
  such record-level identity (§3.2). Confirming that reading and assigning the
  evaluator-side fix is **Q008**; this requirement covers only the half the
  architecture grounds.
- **Acceptance**: AC-019.

#### REQ-F-015 — Producer failure is loud

A failure to write any bundle artifact MUST terminate the run with a named
error and MUST NOT be swallowed so the run continues producing an I-07 record
with no matching evidence.

- **Priority**: Must-Have
- The failure is recorded as the I-07 record's `outcome.terminal = "error"` with
  the writer's own message as `reason`, and `bundle.json` (if already written)
  gains `stop_outcome: "error"` with a matching `ineligibility_reasons[]` entry.
- **Acceptance**: AC-020.

#### REQ-F-016 — The shipped validator is the acceptance bar

`bench/scripts/verify-stage-evidence.sh <i05_bundle_dir>` MUST exit `0` on the
bundle a real run produces, with no modification to that script.

- **Priority**: Must-Have
- No parallel validator, no relaxed variant, no new vocabulary
  (research-report Decision 2).
- This is a scope fence on F12's own diff, not a perpetual freeze: pre-feature
  state is `01ce448b` (E40-F11, #213); satisfied at `8aba4712` (F12's own
  merge, #214). Later validator changes belong to E40-F06 (§1.4 item 1 of
  this spec, and test-plan.md's own TC-004 note: "wiring a new validator
  check is E40-F06 scope, not E40-F12's") and are not governed by this
  requirement.
- **Acceptance**: AC-021, AC-022.

#### REQ-F-017 — The adjacent I-07 identity gap is disclosed, not silently left

The feature MUST record, in this spec and in a durable Question, that a correct
I-05 bundle alone does **not** make a real pair evaluate non-failed.

- **Priority**: Must-Have
- The enumerated gap is §3.2. The Question is **Q008**.
- Acceptance for this feature MUST NOT claim "a non-failed evaluated pair" as
  achieved while Q008 is open.
- **Acceptance**: AC-023.

### 1.2 Non-functional requirements

| ID | Requirement | Measurement |
|---|---|---|
| REQ-NF-001 | The producer adds no new runtime dependency. It uses only the standard library and `PyYAML` already required by `run-lifecycle.sh`. | `run-lifecycle.sh` imports unchanged apart from stdlib modules already present |
| REQ-NF-002 | Per-dispatch producer overhead stays bounded and small relative to a provider dispatch: at most one write pass per artifact and one `shark admin workflow list` call per workflow level per run. | The guard suite times the producer's per-dispatch write path directly (it sits outside the stage window by §2.5) and asserts it is under 1s per dispatch on the reference fixture |
| REQ-NF-003 | No secret or raw prompt material is written to the bundle. Snapshots carry `prompt_digest`, never prompt text; transcripts pass through the existing `bounded()` truncation and the adapter's `SENSITIVE` redaction. | Grep of a produced bundle for the prompt bytes of the same run finds no match; `bench/scripts/verify-evidence-roots.sh` stays green |
| REQ-NF-004 | Bundle writes are deterministic: two runs with identical inputs produce byte-identical snapshots apart from timestamps, digests derived from real state, and observed durations. | Canonical serialization is `sort_keys=True, separators=(",",":")` throughout |
| REQ-NF-005 | `make fmt && make lint && make test` at repository root stays green, and `bench/scripts/tests/run-all.sh` stays green. | CI |

### 1.3 Acceptance criteria

| ID | Criterion |
|---|---|
| AC-001 | `run-lifecycle.sh --mode live` without `--i05-bundle-dir` exits non-zero before the first `shark next`, naming the missing option; `--mode resolve-route --i05-bundle-dir <dir>` exits non-zero naming the rejected option and leaves `<dir>` untouched. |
| AC-002 | After a run with at least one dispatch, `<i05_bundle_dir>/bundle.json` exists and carries exactly the ten documented top-level fields; `schema_version` equals `bench/evidence/i05-schema.yaml`'s `schema_version`; `scenario.{scenario_id,scenario_version,entity_family}` equal the I-04 package's values. |
| AC-003 | `bundle.json.stages[]` has one entry per dispatch, `dispatch_ordinal` values are unique and equal the I-07 `dispatches[].ordinal` set, and each `snapshot_digest` equals the recomputed digest of the file at `snapshot_path`. |
| AC-004 | Each `stages/<ordinal>-<stage_key>.json` exists, parses, and carries every field in `bench/README.md`'s stage-snapshot field reference plus a non-empty top-level `provider`. |
| AC-005 | Recomputing `snapshot_digest` over the snapshot minus that field reproduces the recorded value for every snapshot; mutating one byte of a snapshot makes `verify-stage-evidence.sh` exit `1` naming `snapshot_mutated`. |
| AC-006 | A `code`- or `review`-category snapshot carries all six candidate identity fields plus `test_suite_ids` and `test_suite_dir`; a `discovery`-category snapshot carries no `candidate` block and is still accepted. |
| AC-007 | For every phase in the §2.4 table, a dispatch at a status carrying that phase yields the tabulated `stage_category`. A dispatch at a status whose phase is absent from the table yields a snapshot with `errors[]` containing `{kind: "unknown_stage_category"}` naming the status and phase, and no substituted category. |
| AC-008 | `verify-stage-evidence.sh` accepts every produced `time_ledger`: no `ledger_overlap`, no `ledger_window_escape`, no `ledger_non_reconciling`. |
| AC-009 | With a worker envelope reporting no provider-active intervals, the adapter window appears wholly under `unclassified` and `provider_active` totals zero. With an envelope reporting intervals, those exact intervals appear under `provider_active` and the remainder of the adapter window under `unclassified`. |
| AC-010 | `<i05_bundle_dir>/access.jsonl` exists after every run, including a run with zero evaluator access, and `verify-stage-evidence.sh --grant-access` appends to it without creating it. |
| AC-011 | `<i05_bundle_dir>/transcripts/` exists as a real directory after every run; a direct `bench/scripts/lib/retain_pair` invocation against the produced bundle succeeds and writes a `manifest.json` whose `artifacts.evidence` and `artifacts.transcripts` entries both carry a real `sha256`. |
| AC-012 | `bundle.json.roots` declares all three roots with their `i05-schema.yaml` `worker_access` values, pairwise non-nested absolute paths, and an `agent_fixture_checkout` that `os.path.isdir()` accepts; `evaluate-lifecycle.sh` against that bundle does not append a `missing_oracle` reason for an absent checkout. |
| AC-013 | A clean run's `bundle.json` has no `stop_outcome` and `publication_eligible: true`. For each stop outcome the guard suite can induce end-to-end — `resource_limit` (a ceiling of 0.01 USD), `missing_outcome` (a stub worker returning no `recommended_outcome`), `error` (a stub `shark` exiting non-zero), `worker_failure` (an adapter exiting non-zero), `pause` (a stub worker returning `kind: "question"`) — the run yields that value, `publication_eligible: false`, a non-empty `ineligibility_reasons[]`, and `verify-stage-evidence.sh` accepts the bundle. The remaining five (`lease_loss`, `unresolved_gate`, `archive`, `cancellation`, `timeout`) are covered by direct unit invocation of the producer's triad writer with each value injected, asserting the same three postconditions — end-to-end induction is not required for them. |
| AC-014 | A run interrupted after dispatch N leaves snapshots 1..N present and readable, indexed in `bundle.json`, and `verify-stage-evidence.sh` still validates the partial bundle. |
| AC-015 | Running scenario S repetition 2 into an `i05_bundle_dir` holding repetition 1's bundle leaves no repetition-1 snapshot in the final bundle, while an unrelated operator-placed file in that same directory is still present afterwards. A symlink at any of the four producer-owned entries fails the run before any write. |
| AC-016 | `identity.shark_content_digest` on a real I-07 record equals `content_digest(bundle.json's content_root)` computed by `evaluate-lifecycle.sh`'s own function, and the record carries `content_digest_scheme: "walk_v1"`. |
| AC-017 | Corrupting one byte under `content_root` after the run makes `evaluate-lifecycle.sh` append the `identity_mismatch` reason at `/identity/shark_content_digest` — i.e. the crosscheck demonstrably fires, closing TD-132. |
| AC-018 | `run-lifecycle-batch.sh dispatch_pair` and `run-review-comparison.sh dispatch_gate` both pass the configured bundle directory to the lifecycle driver, and `bench/scripts/e40-benchmark.sh preflight` no longer reports the P5 `i05_bundle_dir is not configured` blocker for a configured scenario. |
| AC-019 | `evaluate-lifecycle.sh` against a real produced pair emits **no** `missing_join` or `contradictory_join` reason at `/join/run_id`, `/join/scenario_id`, `/join/scenario_version`, or `/join/dispatches`. Reasons at `/join/dispatch_id` and `/join/dispatch_ordinal` are expected and are Q008's subject, not a defect of this feature. |
| AC-020 | Making the bundle directory unwritable mid-run terminates the run with `outcome.terminal: "error"` naming the write failure; no I-07 record claims `publication_eligible: true` while its bundle is incomplete. |
| AC-021 | `bench/scripts/verify-stage-evidence.sh <i05_bundle_dir>` exits `0` on the bundle from a real `--mode live` run of at least one of the six lifecycle-v2 families, printing its fixed-order JSON summary. |
| AC-022 | `bench/scripts/verify-stage-evidence.sh` is byte-identical between its pre-feature state (`01ce448b`) and F12's own merge (`8aba4712`) (`git diff --exit-code` on that path across that fixed historical range). |
| AC-023 | This spec's §3.2 enumerates every I-07 identity and workflow-policy field the evaluator requires and the live producer omits, and Q008 exists in Shark carrying that enumeration. |

### 1.4 Out of scope

1. **Any change to the I-05 schema, its vocabularies, or `verify-stage-evidence.sh`.**
   Why: `bench/evidence/i05-schema.yaml` and its validator are E40-F06's shipped
   contract and are correct; the gap is production, not definition
   (research-report Decision 2). Future: none anticipated.
2. **Any change to `bench/scripts/lib/retain_pair`.** Why: its fail-closed
   behavior was already hardened at UAT-R3-01; it only lacked a real source
   directory (research-report Decision 5, Finding 4). Future: none.
3. **Closing the I-07 producer identity and workflow-policy surface (§3.2).**
   Why: those are E40-F08's I-07 contract fields, which
   `../architecture.md#lifecycle-run-record-contract` already requires and the
   live producer never emitted; reopening a completed feature to fix them is not
   this feature's call (**Q008**). Future: a follow-up feature once Q008 is
   confirmed. **Consequence stated plainly: feature.md's Impact ("unblocks a
   real, spend-eligible pilot/baseline run across all 6 families") is NOT met by
   this feature alone.** See §3.2.
4. **Removing `dispatch_id` / `dispatch_ordinal` from `validate_identity_join()`'s
   record-level join fields.** Why: E40-F09's shipped guard, and §3.2 shows the
   expectation traces to its per-dispatch fixtures rather than to the
   architecture — but E40-F12 does not edit another completed feature's guard
   (**Q008**). Future: as above.
5. **Parallel repetitions writing into one `i05_bundle_dir`.** Why: the flat
   layout every shipped driver and test assumes (ADR-F12-02) is safe only for
   sequential repetitions. Future: a per-run subdirectory layout, if parallel
   repetitions are ever introduced.
6. **Extending the worker adapter to measure provider-active time.** Why: the
   producer consumes such intervals when the envelope supplies them (REQ-F-005)
   but does not change the adapter or provider contract. Future: an adapter-side
   feature; until then `provider_active` is honestly empty rather than inflated.
7. **Any Go change to Shark itself.** Why: E40 benchmarks production Shark and
   does not take ownership of it (cross-epic map preamble). The producer reads
   only already-shipped public CLI surfaces.

---

## 2. Architecture

### 2.1 Component changes

| File | Change | Detail |
|---|---|---|
| `bench/scripts/run-lifecycle.sh` | **Modify** | Add `--i05-bundle-dir` to `parse_args()` (line 67) with the mode rules of REQ-F-001. Add an `I05BundleWriter` class (new, in the same embedded Python program) implementing REQ-F-002/003/005/006/007/009/010/011/015. Instantiate it in `main()` after `pre_dispatch_gates()` (line 737) and before the dispatch loop. Call it from inside the existing per-dispatch loop at line 871, immediately after `record["stages"].append(stage_record(dispatch, stage_candidate))`. Add the phase lookup of REQ-F-004. Replace `scenario_identity()`'s `shark_content_digest` computation (line 255) per REQ-F-012 and add `content_root` + `content_digest_scheme`. Add interval instrumentation per §2.5. |
| `bench/scripts/run-lifecycle-batch.sh` | **Modify** | `dispatch_pair()` (line 822): pass `--i05-bundle-dir "$i05_bundle_dir"` to `"$RUN_LIFECYCLE_BIN"` (the invocation at line 887). Move the existing `if [[ -z "$i05_bundle_dir" ]]` guard (line 902) to **before** the run rather than after it, so a misconfigured pair fails without burning provider spend. |
| `bench/scripts/run-review-comparison.sh` | **Modify** | `dispatch_gate()` (line 719): same pass-through and same guard reordering (its guard is at line 819). |
| `bench/evidence/stage-category-map.yaml` | **New** | The closed §2.4 phase → `stage_category` table, machine-readable, single owner. Mirrors the single-owner discipline `bench/evidence/i05-schema.yaml` and `bench/evidence/usage-mapping.yaml` already establish; the producer reads it rather than embedding a private copy. |
| `bench/README.md` | **Modify** | Add `transcripts/` to the "Bundle layout (I-05)" tree with a note that `retain_pair` requires it (closes research-report Finding 6's documentation gap). Document `--i05-bundle-dir`, `content_root`, and `content_digest_scheme`. |
| `docs/plan/tech-debt/TD-132.md` | **Modify** | Fill Impact and Resolution Plan with the ADR-F12-03 decision; mark resolved when AC-016/AC-017 pass. |
| `bench/scripts/tests/tc115_i05_producer_test.sh` | **New** | Guard suite for AC-001..AC-015, AC-019..AC-021, following the existing `tcNNN_*_test.sh` convention and registered in `bench/scripts/tests/run-all.sh`. |
| `bench/scripts/tests/tc116_content_identity_crosscheck_test.sh` | **New** | Guard suite for AC-016/AC-017 (TD-132), complementing `tc075_content-identity_x12_test.sh` which today can only exercise the branch with a hand-injected `content_root`. |
| `bench/scripts/tests/run-all.sh` | **Modify** | Register the two new suites. |
| `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-interaction-map.md` | **Modify** | Add the provisional revision-owner record for the I-05 staged edge (§4), following the E40-F11 / Q007 precedent for I-04 verbatim in form. |

No Go file is modified. No file under `internal/` or `cmd/` is touched, so the
CLAUDE.md service/repository layering rules do not apply; the applicable
conventions are `bench/scripts/`'s own (`set -euo pipefail` bash wrapper around
an embedded Python program, `sort_keys=True, separators=(",",":")` canonical
JSON, guards owned by a single named script).

### 2.2 Data model changes

No database schema, no migration, no `internal/db/db.go` change. The data model
changes are on-disk artifacts only.

**New on-disk tree** (created by the producer at `i05_bundle_dir`):

```
<i05_bundle_dir>/
├── bundle.json          # REQ-F-002
├── stages/
│   └── <dispatch_ordinal>-<stage_key>.json    # REQ-F-003
├── access.jsonl         # REQ-F-006
└── transcripts/
    └── <dispatch_ordinal>-<stage_key>.txt     # REQ-F-007
```

Field inventories for `bundle.json` and the stage snapshot are **not restated**
— they are `bench/README.md`'s "Bundle layout (I-05)" and "Stage-snapshot field
reference" tables, and this feature adds no field to either.

**Two additive fields on the I-07 record's `identity` block** (REQ-F-012):

| Field | Type | Contract |
|---|---|---|
| `content_root` | string | Absolute path to the installed Shark-data canonical content tree inside the scratch Shark project. Read by `evaluate-lifecycle.sh:275`'s `declared_content_root` (which already accepts `identity.content_root`, so no evaluator change is needed). |
| `content_digest_scheme` | string, closed set `{"walk_v1"}` | Marks which scheme produced `shark_content_digest`. A record without this field is the legacy whole-repo-tree scheme and is not comparable with a `walk_v1` record. |

`shark_content_digest`'s **value scheme** changes (whole-repo git-tree hash →
walk-based content hash of `content_root`); the field itself is unchanged in
name, type, and position.

### 2.3 Interface contracts

**`run-lifecycle.sh` CLI (extended):**

```
run-lifecycle.sh --scenario <package.yaml> --run-id <id> --root <key>
                 --scratch-root <dir> [--output <lifecycle.jsonl>]
                 [--limits <policy.yaml>] [--i05-bundle-dir <dir>]
                 [--mode contract|dry-run|resolve-route]
```

| Mode | `--i05-bundle-dir` | Producer behavior |
|---|---|---|
| `live` | **required** | Writes the full bundle |
| `contract` | optional | Writes the full bundle when supplied; no-op when absent |
| `dry-run` | optional | Writes the full bundle when supplied; no-op when absent |
| `resolve-route` | **rejected** | Never writes; exits `2` if supplied |

**Producer → validator contract:** `verify-stage-evidence.sh <i05_bundle_dir>`
exits `0`. That script is the single owner of I-05's validation arithmetic and
is unmodified (REQ-F-016).

**Driver → producer contract:** `dispatch_pair` / `dispatch_gate` pass the
operator-configured `scenario_roots.<id>.i05_bundle_dir` through unchanged. The
directory is **flat** — the same path the drivers then hand to
`evaluate-lifecycle.sh --i05` and to `retain_pair` (ADR-F12-02).

**Producer → Shark read contract:** `shark admin workflow list <level> --json`,
reading `levels[].statuses[].{name,phase}`. `internal/cli/commands/workflow.go:33`
`StatusDisplay` already carries `Phase string \`json:"phase,omitempty"\``, so
this is an existing shipped read surface consumed unchanged. Entity type from
`shark next`'s `entity_type` maps to the workflow level name with one
normalization: `tech-debt` → `tech_debt`.

### 2.4 Closed lifecycle table — workflow phase → I-05 `stage_category`

Behavior-bearing lifecycle field. Every phase the shipped default workflows
declare is assigned; there is **no fallthrough** and no default. Derived from
the live workflow (`shark admin workflow list <level> --json`, all six levels,
read 2026-09-08).

| Workflow phase | Statuses carrying it (level) | `stage_category` | Dispatchable |
|---|---|---|---|
| `research` | `research` (bug, epic, change, feature, tech_debt) | `discovery` | yes |
| `triage` | `assessment` (epic, feature) | `discovery` | yes |
| `refinement` | `refinement` (epic) | `specification` | yes |
| `specification` | `specification` (feature) | `specification` | yes |
| `design` | `design` (epic) | `specification` | yes |
| `decomposition` | `decomposition` (epic), `task_generation` (feature) | `planning` | yes |
| `planning` | `draft` (bug, epic, task, change, feature), `identified` (tech_debt) | `planning` | yes |
| `test_planning` | `test_planning` (feature) | `planning` | yes |
| `development` | `development` (bug, task, change), `in_progress` / `triaged` (tech_debt) | `code` | yes |
| `code_review` | `code_review` (bug, change, feature) | `review` | yes |
| `review` | `feature_review` / `integration_review` (epic), `task_review` (feature) | `review` | yes |
| `qa` | `qa` (bug, change, feature) | `qa` | yes |
| `approval` | `approval` (feature) | `uat` | yes |
| `execution` | `active` (epic, feature) | `code` | rarely — a parent execution status that normally cascades rather than dispatching; assigned so a direct dispatch is never uncategorized |
| `blocked` | `blocked` (bug, epic, task, change, feature) | — | **no** — parking status; `shark next` never returns `spawn_agent`, so no snapshot is written |
| `paused` | `on_hold` (bug, epic, task, change, feature) | — | **no** — parking status; as above |
| `done` | `completed`/`cancelled` (all levels), `resolved`/`wont_fix` (tech_debt) | — | **no** — terminal; as above |
| *(any phase not listed, or a status with no phase)* | — | **none assigned** | Snapshot carries `errors[]` `{kind: "unknown_stage_category"}` naming status and phase (REQ-F-004) |

`shipping` is deliberately **unproduced** by the shipped workflow: the
interaction map's boundary rule states "Finish-feature scope in lifecycle v2
stops at its controlled deep-review gate. PR feedback, CI, merge, and cleanup
remain deferred delivery-tail scenarios." No phase maps to it, and the producer
never emits it. It remains in `i05-schema.yaml`'s closed vocabulary for the
deferred delivery-tail scenarios.

### 2.5 Closed lifecycle table — time-ledger interval sources

Behavior-bearing lifecycle field. Every one of `i05-schema.yaml`'s six
`interval_category` values is assigned exactly one observed source, or is
declared unproduced with a reason. There is no default bucket other than the
explicit `unclassified` one.

Stage window: `stage_start` is captured immediately before the dispatch's
`shark next` call (`bench/scripts/run-lifecycle.sh:769`); `stage_end` is
captured after `refresh_candidate()` (line 868) — that is, after `release`
completes **and** after the last identity/digest computation that belongs to the
dispatch. The producer's own serialization and file write happen after
`stage_end` and are deliberately **outside** the window: they are bookkeeping
about the stage, not stage work, and including them would require knowing the
write duration before writing it. `tool_and_test` therefore covers the git and
filesystem work inside the window (`candidate_identity()` at line 794,
`refresh_candidate()` at line 868) and not the snapshot write itself. Both
are integer nanoseconds from a single `time.monotonic_ns()` origin taken once at
run start, so all intervals share one timeline. `reconciliation_epsilon_ns` is
the fixed constant `1_000_000` (1 ms), declared in the producer and never
widened per run.

| `interval_category` | Observed source | Produced |
|---|---|---|
| `provider_active` | Only explicit `provider_active` intervals in the worker envelope's top-level `time_ledger` block, clipped to the stage window. **Never** the adapter window as a whole. | Only when the envelope reports them |
| `tool_and_test` | `candidate_identity()` and `refresh_candidate()` git and filesystem walks (`run-lifecycle.sh:794`, `:868`). The producer's own bundle writes are outside the stage window (see above) and are never recorded in any category. | Always |
| `queue_or_claim_wait` | The `shark claim` call window (line 806) and the `shark release` call window (line 853) | Always |
| `replay_or_human_gate_wait` | The `route_worker_question()` window (line 823) when a worker returns `kind: "question"` | Only on a question handoff |
| `retry_or_backoff` | The heartbeat sleep windows inside `adapter_result()` (line 383) when the adapter process is polled after a failed heartbeat retry | Only on a heartbeat retry |
| `unclassified` | The adapter subprocess window minus any envelope-reported `provider_active` intervals, plus `shark next` / prompt-verification time, plus any residual within `reconciliation_epsilon_ns` | Always |

**Envelope placement note:** worker-reported intervals MUST be read from the
control envelope's **top level**, not from `evidence`. The adapter's
`SAFE_EVIDENCE_KEYS = {"type","path","digest","size_bytes","description"}`
(`bench/scripts/lifecycle-worker-adapter.sh`) strips anything else placed inside
`evidence`, so a timing block there would be silently discarded.

### 2.6 Closed lifecycle table — stop outcome → eligibility triad

| I-07 `outcome.terminal` | `bundle.json.stop_outcome` | `publication_eligible` | `ineligibility_reasons[]` |
|---|---|---|---|
| `complete` | absent | `true` | `[]` |
| `resource_limit` | `resource_limit` | `false` | names the ceiling from `limits.first_exceeded` |
| `lease_loss` | `lease_loss` | `false` | names the entity whose lease was lost |
| `missing_outcome` | `missing_outcome` | `false` | names the entity with no final outcome |
| `unresolved_gate` | `unresolved_gate` | `false` | names the blocking gate |
| `pause` | `pause` | `false` | names the routed Question key |
| `archive` | `archive` | `false` | names the archived entity |
| `error` | `error` | `false` | carries `outcome.reason` |
| `cancellation` | `cancellation` | `false` | names the signal |
| `worker_failure` | `worker_failure` | `false` | carries `outcome.reason` |
| `timeout` | `timeout` | `false` | names the exceeded window |

The eleven rows are total over `run-lifecycle.sh`'s reachable terminal values:
`complete` plus the ten `STOP_OUTCOMES` (`run-lifecycle.sh:47`), which are the
same ten `i05-schema.yaml` declares.

### 2.7 Key technical decisions

**ADR-F12-01 — The producer lives inside `run-lifecycle.sh`'s dispatch loop, not
in a shared library.**
Rationale: no bundle writer exists anywhere under `bench/scripts/` to extend —
every I-05-touching test hand-builds its own fixture (research-report Finding 2,
Decision 1). The loop already computes every field a snapshot needs
(`candidate_identity()` lines 111-127, `refresh_candidate()` lines 162-167, the
per-dispatch `evidence_refs` at line 798), so serializing in place avoids
re-deriving identity in a second place and the drift that invites. Follows the
established pattern of `run-lifecycle.sh` owning its own record assembly
(`make_record()`, `stage_record()`).
Rejected: a `bench/scripts/lib/i05_writer` module — one caller, no reuse today,
and CLAUDE.md Rule 2 forbids abstractions for single-use code.

**ADR-F12-02 — The bundle is written flat into the configured `i05_bundle_dir`,
not into a per-run subdirectory.**
Rationale: every shipped consumer treats `i05_bundle_dir` as the bundle root —
`evaluate-lifecycle.sh` resolves `os.path.join(args.i05, "bundle.json")`
(line 173), `retain_pair` copies from it directly (lines 380-381),
`run-review-comparison.sh:486` tests it for non-emptiness, and
`tc082`/`tc099`/`tc079` all configure a flat path. A `<dir>/<run_id>/` layout
would be closer to `bench/README.md`'s documented
`<evidence_root>/<scenario_id>/<run_id>/` tree but would require changing all of
those call sites and their fixtures for no behavior gained: repetition isolation
already comes from `retain_pair` copying each pair out immediately after it
completes. The documented tree is achieved by the **operator's configuration**
pointing `scenario_roots.<id>.i05_bundle_dir` at a per-scenario path.
Cost, stated: this is safe only for sequential repetitions, which is what
`run-lifecycle-batch.sh` does today. REQ-F-011's bounded reset stops a later
repetition inheriting an earlier one's snapshots. Parallel repetitions are out
of scope (§1.4 item 5).

**ADR-F12-03 — `content_root` is the installed Shark-data canonical content tree
in the scratch project, and `shark_content_digest` switches to the walk-based
scheme.**
Rationale: TD-132 names exactly two candidates — the installed
`internal/sharkdata/default_data` bundle or an installed copy in the scratch
project — and requires a decision. The cross-epic map's X-12 row already decides
the *intent*: "Derive one installed-content identity covering workflows,
prompts, skills, and agents so comparisons cannot mix content bundles", with the
note "It does not require E32 to add a benchmark-specific digest API if E40 can
deterministically hash the installed canonical tree." The scratch project's
installed tree is what actually drives the run under measurement, and it is what
varies between configurations under comparison; a whole-repo git-tree hash
additionally varies with every unrelated repository edit, which makes two
otherwise-identical runs incomparable for no semantic reason.
The digest scheme must change with it: `evaluate-lifecycle.sh`'s
`content_digest()` walks and hashes file contents, so declaring `content_root`
while retaining the git-tree hash converts a dead check into a guaranteed
`identity_mismatch` on every run — strictly worse than leaving TD-132 open.
Cost, stated: retained baselines produced before this change used the legacy
scheme and are not comparable with post-change baselines.
`content_digest_scheme: "walk_v1"` (REQ-F-012) makes that visible rather than
silent.
Rejected: a two-scheme compatibility path — it doubles the identity surface
permanently to preserve baselines that predate any real lifecycle-v2 run.

**ADR-F12-04 — `stage_category` is derived from the workflow phase via a
bench-owned closed table, read through an existing public Shark command.**
Rationale: `shark next --json` returns `status` but not `phase`
(`internal/cli/commands/next.go:331` `NextResponse` has no phase field), and the
I-04 scenario package's `stage_matrix.lifecycle` is `{mode, evidence_required}`
— neither enumerates stages nor categories. The workflow's own 18-value `phase`
vocabulary is the only authoritative source, and
`shark admin workflow list <level> --json` already exposes it
(`internal/cli/commands/workflow.go:36`). Reading it costs one call per workflow
level per run and requires no Go change, keeping E40's "benchmark production
Shark, do not take ownership of it" posture intact.
Rejected: (a) adding `phase` to `NextResponse` — a Go change to a public
contract E40 does not own; (b) a bench table keyed on status **name** — six
workflows' worth of status names, far more drift surface than 18 phases;
(c) parsing `shark-data/workflow/*.yaml` from disk — fails whenever the run uses
the embedded default bundle, which the scratch projects do.

**ADR-F12-05 — `provider_active` stays empty rather than being inferred from the
adapter window.**
Rationale: `bench/README.md`'s time-ledger rules state "Unknown or
unattributable time is never assigned to `provider_active` — the one property
UAT-16 turns on." The adapter window contains provider inference *plus* the
worker's tool and test execution, and the driver cannot separate them.
Attributing the whole window to `provider_active` would make the benchmark's
headline provider-time metric systematically wrong in the direction that
flatters it.
Cost, stated: until the adapter reports intervals, `provider_active` totals zero
on real runs and provider time is visible only as `unclassified`. That is a
truthful "not measured", not a measured zero, and REQ-F-005 plus AC-009 pin it.

**ADR-F12-06 — The producer resets only the four entries it owns.**
Rationale: `i05_bundle_dir` is operator-configured and preflight-validated as an
existing real directory (`bench/scripts/lib/e40_benchmark.py:2192`). Deleting
and recreating it would destroy operator content and could race the preflight
contract; leaving it untouched would let repetition N-1's snapshots survive into
repetition N's retained bundle. Removing exactly `bundle.json`, `stages/`,
`access.jsonl`, `transcripts/` is the minimum that makes each repetition's
bundle honest. The symlink refusal mirrors `retain_pair`'s existing source-side
posture (`refuse_symlink_containment`).

**ADR-F12-07 — The `i05_bundle_dir` configuration guard moves before the run.**
Rationale: `run-lifecycle-batch.sh:902` currently checks `i05_bundle_dir` only
**after** `run-lifecycle.sh` has already executed a full live dispatch sequence.
Once the producer needs that path as an input, the check must precede the run
anyway; doing so also stops a misconfigured scenario burning real provider spend
before failing. Same change in `run-review-comparison.sh:819`.

### 2.8 Integration with existing code

| Integration point | Existing signature / location | How this feature integrates |
|---|---|---|
| Argument parsing | `parse_args(argv)` — `bench/scripts/run-lifecycle.sh:67` | Add `--i05-bundle-dir` to the optional-option set alongside `--output`, `--limits`, `--mode`; add the mode-dependent validation of REQ-F-001 after the existing mode check at line 82 |
| Candidate identity | `candidate_identity(repo_root)` — line 111; `refresh_candidate(candidate, scratch)` — line 162 | Consumed unchanged. The snapshot's `candidate` block is `dict(candidate)` after `refresh_candidate()` — the same object the loop already assigns to `stage_candidate` at line 869 |
| Canonical digest | `canonical_digest(value)` — line 96 | Reused verbatim for `snapshot_digest` and for the bundle's stage index digests |
| Bounded serialization | `bounded(value)` — line 174 | Reused for transcript content and for any response fragment recorded in a snapshot |
| Scenario identity | `scenario_identity(scenario_path, scenario)` — line 241 | `shark_content_digest` computation at line 255 is replaced per REQ-F-012; `content_root` and `content_digest_scheme` are added to the returned dict |
| Stage record | `stage_record(dispatch, candidate)` — line 295 | Unchanged. The I-07 stage record and the I-05 snapshot are written from the same `dispatch` and `stage_candidate` objects, so the two records cannot disagree about `dispatch_ordinal`, `prompt_digest`, or `snapshot_digest` |
| Dispatch loop hook | `bench/scripts/run-lifecycle.sh:871` (`record["stages"].append(...)`) | The producer's per-dispatch write is invoked immediately after this line, inside the same `while queue` body, so partial evidence (REQ-F-010) follows automatically from the loop's existing early-`break` paths |
| Run termination | line 888 (`record["outcome"] = {...}`) and line 893 (final `output.write_text`) | The producer's final `bundle.json` rewrite (REQ-F-009's triad) is invoked between these two, from the same `terminal` / `reason` values, so §2.6's table is satisfied by construction |
| Signal handling | `release_on_signal(signum, _frame)` — line 750 | Unchanged. Snapshots already written stay on disk; the partial bundle is what AC-014 validates |
| Adapter result | `adapter_result(adapter, request, cwd, mode, shark)` — line 346 | The producer wraps the call site (line 816) with monotonic marks to derive the adapter window of §2.5, and reads any top-level `time_ledger` from the returned envelope |
| Batch driver | `dispatch_pair()` — `bench/scripts/run-lifecycle-batch.sh:822`; the `"$RUN_LIFECYCLE_BIN"` invocation at line 887; the guard at line 902 | Pass-through per REQ-F-013, guard move per ADR-F12-07 |
| Comparison driver | `dispatch_gate()` — `bench/scripts/run-review-comparison.sh:718`; guard at line 819; `retain_gate()` at line 620 | Same two changes |
| Retention | `bench/scripts/lib/retain_pair` `copy_dir_artifact("evidence", i05_bundle_dir, exclude_top_level=("transcripts",))` and `copy_dir_artifact("transcripts", i05_bundle_dir + "/transcripts")` — lines 380-381 | Unmodified. REQ-F-007 supplies the `transcripts/` directory these calls require |
| Validator | `bench/scripts/verify-stage-evidence.sh` — `main()` at line 983 | Unmodified; invoked as the acceptance gate (REQ-F-016) |
| Evaluator | `bench/scripts/evaluate-lifecycle.sh` — `validate_candidate_snapshots()` line 246, `declared_content_root` line 275, `run_oracle()` line 548 | Unmodified. REQ-F-008 and REQ-F-012 supply the inputs its already-shipped branches need |
| Preflight | `bench/scripts/lib/e40_benchmark.py` P5 check at lines 2176-2203 | Unmodified. AC-018 is satisfied by the operator configuring the path the drivers now consume |

---

## 3. Cross-feature interactions

### 3.1 Declared interactions

Interaction IDs are used verbatim from `../E40-interaction-map.md`. No new ID is
minted here.

- **Produces**: **I-05** — Stage evidence and isolation contract; consumer
  features E40-F08 (canonical multi-entity lifecycle runner), E40-F09
  (calibrated evaluation and comparison identity), E40-F10 (operator workflow
  and retained lifecycle baseline).
  **Shape source**: `../architecture.md#stage-evidence-and-isolation-contract`.
  **Contract test pointer**:
  `tests/contracts/e40_i05_stage_evidence_contract_test.go#TC-042`.
  **Producer of record remains E40-F06.** E40-F12 supplies the *live production
  path* for a shape E40-F06 defined; it is not written in as producer. The
  map-assigned values for the I-05 staged edge stand **unaltered**:
  `gate_mode` `contract-only` until each consumer proves live production-path
  use; `activation_owner` E40-F08 / E40-F09 / E40-F10, each closing its own
  consumption independently; `closure_key` E40-F08 / E40-F09 / E40-F10
  respectively at each feature's own UAT; `counterpart_status` read live from
  Shark at review/UAT time (read 2026-09-08: E40-F06 `completed`, E40-F08
  `completed`, E40-F09 `completed`, E40-F10 `completed`); `review_basis`
  E40-F06's completed specification and the map row, present together at F06
  task_review; `demonstrability_disposition` `pending-integration` until each
  consumer's live wiring closes. Whether E40-F12's live producer *discharges*
  that activation obligation for the three named owners, or whether each must
  reopen, is **Q009** — unsettled, and no assigned value is changed by this
  feature.

- **Consumes**: **I-04** — Lifecycle scenario package contract; producer feature
  E40-F05.
  **Shape source**: `../architecture.md#lifecycle-scenario-package-contract`.
  **Contract test pointer**:
  `tests/contracts/e40_i04_scenario_contract_test.go#TC-030`.
  E40-F12 reads a read-only slice already inside E40-F08's declared consumption
  (`stage_matrix.lifecycle`, `fixture`, `adapter`, `resource_policy`) plus
  `scenario_id`, `scenario_version`, and `entity_family`, in order to populate
  `bundle.json`'s `scenario` block and `stage_matrix_source`
  (`{package_path, package_digest, prelude, lifecycle}`). It adds no field to
  the package and refines no part of the shape. The map-assigned staged values
  stand unaltered: `gate_mode` `contract-only` until each consumer proves live
  production-path use; `activation_owner` and `closure_key` E40-F06 / E40-F07 /
  E40-F08, each closing its own slice at its own UAT; `counterpart_status` read
  live (2026-09-08: E40-F05 `completed`, E40-F06/F07/F08 `completed`);
  `review_basis` E40-F05's completed `spec.md` and the map row at F05
  task_review; `demonstrability_disposition` `pending-integration`. E40-F11's
  provisional revision-owner record for this row (Q007) is untouched by this
  feature.

- **Consumes**: **I-07** — Lifecycle run record contract; producer feature
  E40-F08.
  **Shape source**: `../architecture.md#lifecycle-run-record-contract`.
  **Contract test pointer**:
  `tests/contracts/e40_i07_lifecycle_run_contract_test.go#TC-061`.
  E40-F12 reads the run record's `identity`, `dispatches[]`, `stages[]`, and
  `outcome` to populate the I-05 bundle from the same in-process objects, and
  writes two additive `identity` fields (`content_root`,
  `content_digest_scheme`) plus a changed `shark_content_digest` value scheme
  (REQ-F-012, ADR-F12-03). The map-assigned staged values for I-07 stand
  unaltered: `gate_mode` `contract-only` until each consumer proves live
  production-path use; `activation_owner` E40-F09; E40-F10; `closure_key`
  E40-F09 / E40-F10 at each feature's own UAT; `counterpart_status` read live
  (2026-09-08: E40-F08 `completed`); `review_basis` E40-F08's completed
  specification and the map row at F08 task_review;
  `demonstrability_disposition` `pending-integration`.
  The two additive fields are additive-only and break no existing reader
  (`evaluate-lifecycle.sh:275` already accepts `identity.content_root`). The
  **value-scheme** change to `shark_content_digest` is the material one and is
  recorded in ADR-F12-03 with its stated cost.

### 3.2 Adjacent gap this feature does NOT close (REQ-F-017)

**Stated plainly: shipping a correct I-05 bundle does not, by itself, produce a
non-failed evaluated pair.** feature.md's Impact is therefore not met by
E40-F12 alone.

`bench/scripts/evaluate-lifecycle.sh` computes
`aggregate = not unique and …` (line 616) and exits `1` when `aggregate` is
false (line 160); `run-lifecycle-batch.sh:918` then records the pair `failed`.
Every field below is required by the evaluator and never emitted by the live
I-07 producer, so each one alone is sufficient to fail every pair:

| Evaluator requirement | Source | Live producer state |
|---|---|---|
| `identity.dispatch_id`, `identity.dispatch_ordinal` | `validate_identity_join` join_fields, line 221 | `scenario_identity()` (line 241) emits neither; record top level has neither |
| `canonical_digest(i07.dispatches) == canonical_digest(i05.dispatches)` | line 241-243 | I-07 `dispatches[]` entries are rich objects with no `dispatch_id`; F09's own fixtures use a lightweight `[{dispatch_id, dispatch_ordinal, transition}]` list |
| `identity.toolchain_identity` | `validate_producer_identity`, line 291 | never emitted (the I-04 package carries `toolchain_identity`; it is not copied) |
| `identity.rendered_prompt_digests` | line 292 | never emitted (per-dispatch `prompt_sha256` exists but is not aggregated) |
| `identity.provider_identity` (list of `{stage, provider, model, effort}`) | line 292, 310 | never emitted |
| `identity.judge_identity` (`{model, configuration}`) | line 293, 315 | never emitted |
| `identity.reference_digests` | line 293 | never emitted |
| `identity.resource_policy_digest` | line 294 | never emitted |
| `workflow_policy.enabled_gates`, `.gate_order` | `validate_workflow_policy_identity`, line 322 | `make_record()` (line 277) emits `[]`, which the validator treats as missing |
| `workflow_policy.reviewer.effort` | line 331 | `main()` line 740 sets `effort: ""` |
| `workflow_policy.rendered_prompt_digest` | line 322 | never emitted |
| `workflow_policy.deep_review_bundle_digest` | line 322, 341 | never emitted; must equal `deep_review_digest(repo_root)` over nine named repository files |
| `workflow_policy.workflow_policy_identity_digest` | line 322, 337 | never emitted |

These are **E40-F08's I-07 producer contract** and **E40-F09's evaluator
expectations**, not E40-F06's I-05 shape.

#### Architecture adjudication

`../architecture.md#lifecycle-run-record-contract` (the I-07 shape source)
already decides most of this. It states I-07 "records workflow-policy identity:
enabled gates, gate order, reviewer provider, model, effort, prompt digest, full
review-bundle digest, and whether fixes are allowed between gates. For deep
review, the bundle digest covers the skill, all angle prompts, the consolidator
prompt, and the diff-selection script used by the run." That is exactly
`validate_workflow_policy_identity()`'s requirement set, including
`deep_review_bundle_digest`'s coverage matching `DEEP_REVIEW_FILES`
(`bench/scripts/evaluate-lifecycle.sh:126-136`). The same section requires
"scenario identity ... prompt and worker-result reference ... usage, cost, and
elapsed time", covering `rendered_prompt_digests` and `provider_identity`.

**Conclusion: on the workflow-policy and identity rows above, the evaluator is
right and the live I-07 producer drifted from its own architecture.**
`make_record()` emitting `enabled_gates: []`, `prompt_digest: "0"*64`, a
misnamed `review_bundle_digest`, and `effort: ""` is an implementation gap in
E40-F08, not an over-strict guard in E40-F09.

The converse holds for the two join rows.
`../architecture.md#stage-evidence-and-isolation-contract` describes I-05 as a
per-run bundle of per-stage snapshots and nowhere gives the bundle a
record-level `dispatch_id` or `dispatch_ordinal`. **On those two rows the
evaluator's expectation is not grounded in the architecture** and appears to
come from F09's per-dispatch test fixtures
(`bench/scripts/tests/tc069_calibration_boundary_test.sh:346`,
`tc071_review_finding_normalization_test.sh:277`).

E40-F12 acts on the part it owns and defers the rest: it emits the I-05-side
join references the architecture supports (REQ-F-014) and does **not** fabricate
record-level dispatch identity for a per-run bundle. Confirming this
adjudication and assigning the two fixes is **Q008**.

**What must land after Q008 is confirmed** for a non-failed pair: (a) extending
the I-07 producer to emit the identity and workflow-policy surface the
architecture already requires, and (b) removing `dispatch_id` /
`dispatch_ordinal` from `validate_identity_join()`'s record-level join fields,
leaving the `dispatches`-digest equality as the dispatch-granularity proof.
Both are a follow-up feature under E40; neither is in this feature's scope.

---

## 4. Cross-epic integrations

X-## IDs are used verbatim from `../E40-cross-epic-map.md` and
`docs/product/cross-epic-integration-map.md`. No new X-## is minted here.

- **Validates**: **X-12** — Derive one installed-content identity covering
  workflows, prompts, skills, and agents so comparisons cannot mix content
  bundles. Producer epic E32 (E32-F04 Migrate canonical content into shark-data);
  consumer epic E40 (E40-F09); owning feature **E40-F09**.
  **Contract / shape source**: `E32-F04 Shark-data canonical-content contract;
  E40 architecture "Lifecycle evaluation record contract"`.
  **UX / CX handoff notes**: `Operators see a comparison rejected with the
  differing content digest rather than a misleading quality delta.` E40-F12 is
  what makes that visible behavior real: ADR-F12-03 hashes the installed
  canonical tree deterministically — the exact affordance the map's own note
  contemplates ("It does not require E32 to add a benchmark-specific digest API
  if E40 can deterministically hash the installed canonical tree") — and
  AC-017 proves the rejection fires. E40-F09 remains the owning feature; E40-F12
  supplies the live producer half.
  **Test coverage**: `test-plan.md` TC for AC-016 and AC-017;
  `bench/scripts/tests/tc116_content_identity_crosscheck_test.sh`, complementing
  the existing `bench/scripts/tests/tc075_content-identity_x12_test.sh` which
  can only reach the branch with a hand-injected `content_root`.

- **Consumes**: **X-09** — Reuse the audited provider-usage field mapping for
  stage evidence and comparison identity; do not invent missing token, cost,
  model, session, or timing fields. Producer epic E27 (E27-F15); consumer epic
  E40 (E40-F06 contract owner; E40-F08 runtime writer); owning feature
  **E40-F06**.
  **Contract / shape source**: `E40 architecture "Stage evidence and isolation
  contract"; E27-F15 approved usage metadata contract and implementation
  artifacts`.
  **UX / CX handoff notes**: `Internal. Missing required provider identity
  invalidates the run instead of degrading to an incomparable aggregate.`
  E40-F12 populates each snapshot's `usage` block by semantic slot name through
  `bench/evidence/usage-mapping.yaml`, never by a hard-coded envelope path, and
  records an unresolvable slot as absent with a matching `usage_slot_unavailable`
  entry in `errors[]` — never zero, never null (REQ-F-003, the field reference
  in `bench/README.md`).
  **Test coverage**: `test-plan.md` TC for AC-004 (the required top-level
  `provider` field and the usage fail-closed posture); the existing
  `bench/scripts/canary-usagemapping.sh` and
  `bench/scripts/tests/tc047_usage_mapping_canary_test.sh` stay green unmodified.

- **Consumes**: **X-11** — Reuse the canonical host-side keyed loop: dispatch
  response, claim, unchanged prompt, heartbeat, semantic outcome, transition,
  release, prompt provenance, and bounded resume. Producer epic E38 (E38-F07;
  E38-F09); consumer epic E40 (E40-F08); owning feature **E40-F08**.
  **Contract / shape source**: `E38-F07 and E38-F09 feature contracts; Shark
  Rider run procedure; E40 architecture "Lifecycle v2 controller boundary"`.
  **UX / CX handoff notes**: `The benchmark records and schedules the loop but
  does not create a second workflow engine, claim store, or prompt assembler.`
  E40-F12 only **records** the loop's already-observed events (claim window,
  adapter window, release window, `prompt_sha256`, semantic outcome) into stage
  evidence; it adds no dispatch, claim, or routing behavior, and makes no Go
  change (§1.4 item 7). The `shark admin workflow list --json` read of
  ADR-F12-04 is an ordinary read of an already-shipped public CLI surface, not a
  new cross-epic seam, and therefore adds no X-## row.
  **Test coverage**: `test-plan.md` TC for AC-008 and AC-009 (the ledger
  intervals derived from the loop's windows); the existing E40 UAT-11 and
  UAT-12 scenarios remain X-11's coverage of record.

---

## 5. Durable unresolved decisions

| Question | Decision needed | Why material | Status |
|---|---|---|---|
| **Q008** | Confirm §3.2's architecture adjudication and assign both fixes: extend the live I-07 producer to the identity and workflow-policy surface `architecture.md#lifecycle-run-record-contract` already requires, and drop `dispatch_id` / `dispatch_ordinal` from `validate_identity_join()`'s record-level join fields. | Determines whether a non-failed evaluated pair is reachable at all, and which of two completed features (E40-F08, E40-F09) carries each fix. E40-F12 owns neither contract; §3.2 shows the architecture decides the substance, but the ownership and reopening call is not this feature's to make. | open; `--blocking` set on the record. **No `question_blocks` link to E40-F12 exists** — the parent loop owns entity gates. Resolution owner `architect@E40` |
| **Q009** | Does E40-F12 discharge I-05's activation obligation for E40-F08/F09/F10, or must each consumer reopen to close its own slice? | The interaction map assigns `activation_owner` and `closure_key` to three features that are all `completed` yet none of which can have closed its slice against a live bundle that never existed. Epic completion depends on the answer, and the map's boundary rule forbids a feature spec altering an assigned value unilaterally. | open; resolution owner `architect@E40` |
| **Q011** | Which feature closes E40-F12's revision to the I-07 lifecycle run record shape (two additive `identity` fields plus the `shark_content_digest` value-scheme change, ADR-F12-03/TD-132)? E40-interaction-map.md's "Provisional revision-owner record (E40-F12)" leaves E40-F08 as producer of record pending this answer. | The revision is additive-only on the two new fields but materially changes `shark_content_digest`'s value scheme; E40-F09/F10's consumer updates against the new scheme are not yet recorded as absorbed anywhere. The map's boundary rule forbids E40-F12 from altering the assigned producer unilaterally. | open; resolution owner `architect@E40` |

**Non-material rationale recorded here rather than as Questions:**

- Bundle directory layout (flat vs per-run subdirectory): decided in ADR-F12-02
  on shipped-consumer evidence; the cost (sequential repetitions only) is stated
  and scoped out in §1.4 item 5.
- `content_root` target and digest scheme: decided in ADR-F12-03. TD-132 asked
  for a decision to be recorded, not escalated, and the cross-epic map's X-12
  note already establishes the intent.
- `provider_active` staying empty: decided in ADR-F12-05 on `bench/README.md`'s
  own stated ledger rule; it is a truthful "not measured" with a named
  follow-up, not an open decision.
- Phase lookup source: decided in ADR-F12-04; the rejected alternatives are
  recorded there.

---

*Last updated*: 2026-09-08
