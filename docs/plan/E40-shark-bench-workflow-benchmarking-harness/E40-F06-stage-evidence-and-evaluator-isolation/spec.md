---
feature_key: E40-F06-stage-evidence-and-evaluator-isolation
epic_key: E40
title: Stage evidence and evaluator isolation — combined requirements and architecture
date: 2026-08-14
---

# E40-F06 Specification: Stage evidence and evaluator isolation

Business context is not restated here. See epic PRD [epic.md](../epic.md)
§"Success gates" (G9, G16, G18, G19) and §"Feature breakdown" (E40-F06), and
[feature.md](feature.md) for this feature's outcome, scope, and acceptance
boundary. System-level decisions are in [architecture.md](../architecture.md) —
this spec implements the shape already fixed by
[Stage evidence and isolation contract](../architecture.md#stage-evidence-and-isolation-contract),
ADR-007, ADR-008, ADR-009, and the I-05 row of
[E40-interaction-map.md](../E40-interaction-map.md).

Capability reuse is settled by the validated
[research report](research-report.md) Capability map. In summary: `RunResult` /
`StageLog` (`internal/runner/controller.go:84-135`) is **extended additively and
never mutated** — X-07/X-08 and F02's canary treat its JSON keys as frozen (row
1); the transcript byte format is **referenced read-only**, never reparsed or
redefined (row 2); `DefaultDisallowedTools` supplies the *structural-guard
pattern only*, not its mechanism (row 3); `bench/scripts/canary-runsurface.sh`
supplies the *real-invocation-over-re-derivation discipline* (row 4); I-04's
`scenarios.yaml` / `package.yaml` are **read-only inputs** and their
schema-versioning form is mirrored, not imported (row 5); X-09's provider-usage
mapping is **verified before use, never assumed** (row 6); the existing two-root
run split gains a genuinely new third root (row 8). This feature re-implements
none of those.

**Scope boundary that binds every requirement below.** F06 delivers the I-05
*schema, validator, guard scripts, and documented contract*. It captures no live
stage evidence and calls no provider. E40-F08 populates the runtime records,
E40-F09 validates comparison identity, and E40-F10 derives reports
([feature.md](feature.md) 2026-08-13 amendment). Any requirement that implied
F06 itself observed a running lifecycle would be an I-05 boundary violation; see
[Out of scope](#out-of-scope-for-this-feature).

---

## Requirements

### Functional requirements

| ID | Requirement | Traces to |
|---|---|---|
| REQ-F-001 | I-05 MUST be its own schema-versioned, file-backed bundle format under an explicit evidence output root, independent of the I-02 record schema. A bundle comprises one `bundle.json` (identity, three-root policy, stage index, stop outcome, eligibility) plus one immutable snapshot document per stage. I-05 MUST NOT require any change to the `RunResult` / `StageLog` JSON key set or to the transcript byte format; every I-05 field is new evidence keyed to a stage, addressed separately. | architecture Stage evidence and isolation contract; research Decision 1; Capability map rows 1-2; ADR-F06-01 |
| REQ-F-002 | Every bundle MUST declare all three roots as `roots.agent_fixture_checkout`, `roots.scratch_shark_project`, and `roots.evaluator_only`, each carrying an absolute path, its `worker_access` mode (`read_write` \| `authorized_surfaces_only` \| `never_during_dispatch` respectively), and a digest of the root's declared identity. The three paths MUST be pairwise disjoint — no root may be nested inside, or equal to, another. A bundle declaring fewer than three roots, or overlapping roots, MUST be rejected naming the offending root pair. | architecture three-root table; ADR-007; feature.md Scope 1; Capability map row 8 |
| REQ-F-003 | Every stage snapshot MUST carry the full field inventory fixed in [Stage snapshot (I-05)](#stage-snapshot-i-05): scenario and entity identity, dispatch ordinal, `stage_category`, a top-level `provider` identity, rendered-prompt digest, input and replay lineage, output artifact records, usage evidence, elapsed-time ledger, `errors[]`, `rework_count`, and `evaluator_access[]`. `stage_category` MUST be exactly one value of the closed set `discovery` \| `specification` \| `planning` \| `code` \| `review` \| `qa` \| `uat` \| `shipping`. `provider` is REQUIRED on every snapshot; a snapshot with no `provider` claim MUST be rejected as `missing_provider`, never silently accepted as out of scope for the REQ-F-009/REQ-F-018 checks it gates. | feature.md Scope 2-3; epic G9, G16 |
| REQ-F-004 | Stage completeness MUST be evaluated separately for the two halves of I-04's `stage_matrix`, because the halves have different applicability oracles (ADR-F05-02, inherited). **Prelude half:** the applicable set is enumerated ahead of time (`stage_matrix.prelude.D01`–`.D05`), so a stage declared `applicable: true` with no snapshot MUST be reported as a named `missing_stage` failure identifying the stage. **Lifecycle half:** `stage_matrix.lifecycle.mode: all_dispatched` is resolved at run time, so there is no "should have been dispatched" oracle; completeness MUST instead be exactly one snapshot per **observed** dispatch, and both a duplicate snapshot for one dispatch ordinal and an observed dispatch with no snapshot MUST be reported as named failures. A validator that claims a `missing_stage` verdict for the lifecycle half MUST be rejected. | feature.md Acceptance boundary 1; F05 spec ADR-F05-02 consequence ("stage completeness is one snapshot per dispatch"); ADR-F06-02 |
| REQ-F-005 | Every stage snapshot MUST carry a `time_ledger` of six intervals — `provider_active`, `tool_and_test`, `queue_or_claim_wait`, `replay_or_human_gate_wait`, `retry_or_backoff`, `unclassified` — expressed as **half-open** `[start, end)` interval lists that are pairwise non-overlapping by construction. The union of all intervals MUST reconcile to the stage's `[stage_start, stage_end)` wall clock within a declared `reconciliation_epsilon_ns`, with any residual assigned to `unclassified`. Unknown or unattributable time MUST NOT be assigned to `provider_active`, and a ledger whose intervals overlap, escape the stage window, or leave a residual larger than the epsilon un-assigned MUST be rejected naming the offending category pair or the residual magnitude. | feature.md Scope 3; epic G16; UAT-16; ADR-F06-06 |
| REQ-F-006 | Every snapshot whose `stage_category` is `code` or `review` MUST additionally carry a `candidate` block: `base_commit`, `tree_digest`, `binary_diff_digest`, `changed_path_digest`, `dirty_untracked_manifest` (an ordered list of `{path, digest, tracked: bool}`), and `test_suite_digest`. A `candidate` block whose identity rests on `HEAD`, a branch name, or a commit id alone MUST be rejected — `base_commit` is one field of the identity, never the identity. | feature.md Scope 4; epic G19; ADR-009; UAT-19 |
| REQ-F-007 | `test_suite_digest` MUST be computed from I-04's **normalized** test-identity set — the sorted `<module-or-package>::<test-name>` id list emitted by `adapter.sh test` — together with `adapter.name`, `adapter.version`, and the package's `toolchain_identity`. It MUST NOT be computed by hashing test file contents or by any rule that requires knowing the fixture's language, so that no generic evidence component branches on Python, Go, or a package manager. | E40-F05 REQ-F-007 (no generic component branches on language); F05 adapter capability contract; ADR-F06-05 |
| REQ-F-008 | Every artifact a stage produces MUST be recorded as `{artifact_type, path, digest, size_bytes, producer_stage, consumers}`. `consumers` is an array of typed edges `{consuming_stage, edge_kind, observed_at}` where `edge_kind` ∈ `read` \| `modified` \| `referenced` \| `evaluator_access`. The **present-but-empty** array (`consumers: []`) means "no consumer observed" — an orphan — and the **absent key** means "consumption evidence was not collected". The two states MUST be distinguishable at schema level; a validator MUST NOT coerce an absent key to an empty array, nor an empty array to absent. | feature.md Scope 5; epic G18; UAT-18; ADR-F06-07 |
| REQ-F-009 | Usage evidence MUST be declared as **semantic slots** — `total_cost`, `input_tokens`, `output_tokens`, `cache_read_input_tokens`, `cache_creation_input_tokens`, `model_ids`, `provider_session_id`, `api_active_duration_ms`, `turn_count` — each bound to a concrete provider envelope path by a versioned mapping declaration (`bench/evidence/usage-mapping.yaml`), never by a field name written inline into the I-05 schema. The mapping MUST carry its own `schema_version`, a `verified_from` provenance block naming the real capture it was confirmed against, and one entry per provider. An envelope field that the mapping names but the envelope does not carry MUST produce one named `usage_slot_unavailable` error identifying that exact slot and envelope path, and that slot MUST be **absent** from the snapshot — never zero, never null, never inferred, never silently dropped. A provider with no verified mapping MUST be declared `unmapped` and MUST fail closed rather than being decoded by guess. | X-09; architecture Metric collection ("E40-F06 verifies the current E27-F15 field mapping"; "A named, tested fallback source is acceptable; silently dropping the field is not"); research Decision 2; ADR-F06-04 |
| REQ-F-018 | Every slot binding MUST declare a `verification_tier` of exactly `real_capture` (the envelope path was observed in a real captured provider envelope) or `unverified` (corroborated only by a hand-authored fixture, a design document, or upstream source). An `unverified` slot MAY be decoded opportunistically, but a snapshot MUST NOT present it as verified identity. Promotion to `real_capture` requires a canary run that observes the path in a real envelope (REQ-F-019). The mapping MUST additionally declare a `required_identity_slots` list — the slots I-05 asserts as provider identity for comparison — and **every listed slot MUST be `real_capture`**; a mapping listing an `unverified` slot as required MUST be rejected naming the slot. `provider_session_id` is deliberately **not** in that list (ADR-F06-12), so no `unverified` slot gates comparison identity. | X-09 ("do not invent missing token, cost, model, session, or timing fields"); epic G14; ADR-008 fail-closed posture; ADR-F06-04, ADR-F06-12 |
| REQ-F-019 | The X-09 canary MUST fail loud, naming the exact slot and envelope path, when the *upstream* provider-usage contract changes in a way that removes I-05's evidence source. Two named drift classes MUST be detected: (a) **envelope-field drift** — a mapped path absent from a real captured envelope; and (b) **envelope-availability drift** — a transcript whose `---STDOUT---` block is no longer a decodable provider envelope at all, which would eliminate the retained source I-05's usage slots read. The canary MUST NOT infer availability from the presence of the transcript file; it MUST decode the block. | architecture ADR-004 canary discipline ("converts a silent metric corruption into a loud failure"); X-09 UX/CX note ("Missing required provider identity invalidates the run"); ADR-F06-11 |
| REQ-F-010 | An **admission-time** isolation check MUST prove, for each I-04 scenario package before any dispatch of that scenario, that every path reachable from `evaluator_only` (`reference_solution`, `oracle_tests[]`, `answer_keys[]`) and every test identity those files define is absent from a fresh checkout of the package's fixture at `fixture.base_sha`. Names MUST be derived from the package at call time, never from a hardcoded list. | feature.md Scope 6; epic G9; UAT-09 |
| REQ-F-011 | A **dispatch-boundary** isolation check MUST run immediately before every worker dispatch and MUST fail the dispatch **before any provider call is made** if any evaluator-only file, its content digest, or a test identity it defines is present in *either* agent-visible root — the fixture checkout **and** the scratch Shark project. The check MUST inspect the actual live roots at dispatch time; a check that asserts only the declared I-04 package layout, or only walks `--workdir`, MUST be treated as incomplete and rejected. Its failure message MUST name the offending root, the offending path, and the evaluator-only source it matched. | feature.md Acceptance boundary 2; architecture three-root table; research Decisions 3-4; ADR-F06-03 |
| REQ-F-012 | Evaluator-only material MUST become readable **only after** its authorized boundary and only in a recorded way, in this order: (a) absent from both agent-visible roots at every dispatch boundary (REQ-F-011); (b) after the applicable stage or scenario reaches terminal status, held-back oracle tests MAY be placed into the fixture checkout **only** through I-04's `adapter.sh inject-tests` capability; (c) the post-run execution oracle MUST read reference solutions and answer keys **in place from the evaluator-only root**, without copying them into the worker checkout before execution completes. Every read of evaluator-only material MUST append one `evaluator_access` event `{accessor, artifact_path, digest, phase, granted_at}` to the bundle. A copy or read that violates the ordering MUST be a named `isolation_violation`, not a warning. | feature.md Acceptance boundary 3, Scope 7; ADR-007; ADR-F06-08 |
| REQ-F-013 | A stored bundle MUST be re-evaluable without rerunning the worker and without any provider call: every field an evaluator needs MUST be resolvable from the bundle plus the named roots. A replay of a `code` or `review` snapshot MUST detect and name each of four drift kinds independently — `tracked_file_changed`, `untracked_file_changed`, `test_suite_changed`, `artifact_consumption_record_missing` — reporting the specific path, test id, or artifact that drifted. | feature.md Acceptance boundary 4 and 6; epic G19; UAT-09, UAT-19 |
| REQ-F-014 | A bundle terminating in a named stop outcome (`resource_limit`, `lease_loss`, `missing_outcome`, `unresolved_gate`, `pause`, `archive`, `error`, `cancellation`, `worker_failure`, `timeout`) MUST retain its partial evidence and MUST set `publication_eligible: false` with a non-empty `ineligibility_reasons[]`. Partial evidence MUST NOT be discarded, and MUST NOT be readable as a valid baseline contribution. A bundle carrying a stop outcome and `publication_eligible: true` MUST be rejected. | feature.md Scope 8, Acceptance boundary; epic G12; ADR-008 |
| REQ-F-015 | Each stage snapshot MUST be immutable and content-addressed: `snapshot_digest` is a `sha256` over the snapshot's canonical serialization excluding the digest field itself, and the bundle's stage index MUST record that digest. Recomputing the digest over a stored snapshot MUST reproduce the recorded value; a mismatch MUST be reported as `snapshot_mutated` naming the stage. | feature.md Outcome ("immutable"); architecture "immutable, replayable snapshot" (G9); REQ-F-013 |
| REQ-F-016 | A schema validator MUST reject a malformed bundle with the **failing field named**: a missing or overlapping root, an unknown `stage_category`, an unknown interval category, an overlapping or non-reconciling ledger, a `code`/`review` snapshot missing any `candidate` field, an artifact record missing `producer_stage` or `digest`, a zero-valued usage slot where the mapping reported the field absent, an `evaluator_access` event out of authorized order, a stop outcome with `publication_eligible: true`, a duplicate dispatch ordinal, and an unsupported `schema_version`. | feature.md Workflow handoff ("must not accept a schema that records only final workflow status or worker self-report"); UAT-09 |
| REQ-F-017 | The closed vocabularies I-05 depends on — `stage_category`, interval category, `artifact_type`, `edge_kind`, `evaluator_access.phase`, stop outcome, and error kind — MUST have exactly one machine-readable owner (`bench/evidence/i05-schema.yaml`). The Go contract validator and every bench guard script MUST read the vocabulary from that file rather than embedding a private copy, so a vocabulary change cannot land in one consumer and not the other. | F05's "single named owner per piece of arithmetic" discipline (`eval-predicate.sh`, `diff-ledgers.sh`); Rule 7 |

### Non-functional requirements

| ID | Requirement | Traces to |
|---|---|---|
| REQ-NF-001 | This feature changes no shark product code. It adds no file under `internal/` or `cmd/`, no schema, no migration, and no service. Its only addition inside the shark Go module is one test-only contract validator, matching E40-F05's REQ-NF-001 posture. | architecture component table ("F06 — New bench contract and guards"); research Scope |
| REQ-NF-002 | With the I-05 schema, guards, and fixtures committed, `make fmt && make lint && make test` at the repository root MUST stay green, and `go list ./...` MUST list no evidence fixture or evaluator-only package. | E40-F05 REQ-NF-002 precedent |
| REQ-NF-003 | The I-05 Go contract validator MUST read in-repo artifacts only — `bench/evidence/**` and `tests/contracts/testdata/e40_i05/**` — and MUST NOT require a populated submodule, because CI's `actions/checkout@v4` does not initialise submodules. `.github/workflows/ci.yml` MUST be unchanged by this feature. | ADR-F05-07 / ADR-F01-05 precedent; `.github/workflows/ci.yml` |
| REQ-NF-004 | Every isolation, validation, and replay guard MUST run offline once the fixtures and caches are present, MUST make **zero provider calls**, and MUST produce byte-identical verdicts across repeated runs at an unchanged bundle, fixture SHA, and toolchain identity. | E40-F05 REQ-NF-004; epic G15 ("dry-run and report operations make no provider calls") |
| REQ-NF-005 | Evidence tooling MUST never touch the live shark database, `.sharkconfig.json`, or the live repository working tree, and MUST never invoke shark project-initialisation commands. All work happens inside caller-supplied roots. | E40-F05 REQ-NF-005; repo config-guardrail hook |
| REQ-NF-006 | The completed Phase 1 surface MUST be byte-unchanged by this feature: `bench/corpus/corpus.yaml`, `bench/scripts/collect-run.sh`, `bench/scripts/verify-clean-checkout.sh`, `bench/scripts/canary-runsurface.sh`, and `tests/contracts/e40_i01_corpus_contract_test.go`. I-04's artifacts (`bench/scenarios/**`, `bench/adapters/**`, `bench/scripts/admit-scenario.sh`, `eval-predicate.sh`, `checkout-scenario-fixture.sh`, `tests/contracts/e40_i04_scenario_contract_test.go`) MUST likewise be unchanged — F06 consumes I-04 read-only. | E40-F05 REQ-NF-006 precedent; interaction map I-04 row ("E40-F06 … treat it as read-only") |
| REQ-NF-007 | The dispatch-boundary check (REQ-F-011) MUST complete fast enough to run before **every** dispatch without materially changing scenario wall time, and its own elapsed time MUST be attributed to the `tool_and_test` ledger category rather than to `provider_active`. | REQ-F-005; epic G16 |

### Acceptance criteria

| ID | Criterion |
|---|---|
| AC-001 | TC-042 asserts, over every fixture bundle under `tests/contracts/testdata/e40_i05/valid/`, that the REQ-F-002/003/005/006/008/009 field inventory — including the required top-level `provider` field — is present and well-typed, that `schema_version` is the version the validator supports, and that every closed vocabulary value used resolves against `bench/evidence/i05-schema.yaml` (REQ-F-017). A snapshot with no `provider` claim at all is rejected as `missing_provider`, never silently accepted. |
| AC-002 | TC-042 asserts all three roots are declared with distinct, pairwise non-nested paths and their required `worker_access` modes; a bundle declaring two roots, or a nested pair, is rejected naming the offending pair. |
| AC-003 | TC-042 asserts the REQ-F-004 completeness split: a fixture whose `stage_matrix.prelude.D03.applicable` is `true` with no D03 snapshot is rejected as a named `missing_stage` for D03; a lifecycle-half fixture with an observed dispatch and no snapshot is rejected as an unmatched dispatch, and one with two snapshots for the same dispatch ordinal is rejected as a duplicate — while a lifecycle half with no `missing_stage` verdict available is accepted, proving the halves are evaluated by different oracles rather than one. |
| AC-004 | `bench/scripts/tests/tc044_time_ledger_reconciliation_test.sh` proves a ledger whose six half-open intervals are disjoint and whose union reconciles within `reconciliation_epsilon_ns` is accepted; overlapping intervals across any two categories are rejected naming both categories; an interval escaping `[stage_start, stage_end)` is rejected; and an un-assigned residual larger than the epsilon is rejected naming the residual magnitude. A companion case proves a residual **within** the epsilon lands in `unclassified` and never in `provider_active`. |
| AC-005 | `tc045_candidate_identity_test.sh` proves a `code` snapshot missing any one of `tree_digest`, `binary_diff_digest`, `changed_path_digest`, `dirty_untracked_manifest`, or `test_suite_digest` is rejected naming that field, and that two snapshots sharing `base_commit` but differing in any one of those fields are reported as distinct candidates — so a matching commit id alone never establishes identity (ADR-009). |
| AC-006 | `tc046_artifact_record_test.sh` proves that a bundle with one artifact carrying `consumers: []` and one artifact with the `consumers` key absent yields two distinct verdicts — `orphan` and `consumption_evidence_missing` — and that neither is coerced into the other in either direction. |
| AC-007 | `bench/scripts/canary-usagemapping.sh` asserts every `anthropic_claude_cli` slot in `bench/evidence/usage-mapping.yaml` resolves against a real captured provider envelope, reusing the committed envelope fixtures under `bench/scripts/testdata/run/` (already asserted envelope-shaped by F02) and accepting an operator-supplied live transcript via `--transcript`. A slot whose envelope path is absent produces one `usage_slot_unavailable` naming the slot and path; the slot is absent from the resulting snapshot and is never zero. `tc047_usage_mapping_canary_test.sh` drives both the agreeing and the drifted case. |
| AC-008 | TC-042 asserts a provider declared `unmapped` in `usage-mapping.yaml` fails closed: a snapshot claiming decoded usage under that provider is rejected naming the provider, and a snapshot recording the slots as absent with a `usage_slot_unavailable` error is accepted. The same test asserts REQ-F-018: every slot carries a `verification_tier`, a slot marked `unverified` is rejected when presented as verified identity, and the same slot recorded as opportunistic evidence is accepted. A snapshot with no top-level `provider` claim at all is rejected as `missing_provider` rather than treated as out of scope for this check. |
| AC-021 | `canary-usagemapping.sh` detects both REQ-F-019 drift classes as distinct named failures: (a) a fixture envelope with one mapped path removed fails naming that slot and path; (b) a transcript whose `---STDOUT---` block is arbitrary **non-JSON** text fails as `envelope_source_unavailable`, naming the transcript path, and is **not** reported as nine per-slot absences. Case (b) is a decoder-robustness assertion over non-envelope input — it makes no claim about any observed or predicted upstream artifact shape, so it does not fabricate an envelope (ADR-F06-11). `tc047_usage_mapping_canary_test.sh` drives both. |
| AC-022 | TC-042 asserts `usage-mapping.yaml` declares `required_identity_slots` and that every listed slot's `verification_tier` is `real_capture`; a mapping listing `provider_session_id` (or any other `unverified` slot) as required is rejected naming the slot. A companion case asserts a snapshot carrying every required slot is accepted as identity-complete while one missing any required slot is rejected naming it — so E40-F09 inherits a decided slot set rather than an ambiguity. A `code`/`review` snapshot with no top-level `provider` claim at all is rejected as `missing_provider` — comparison identity can never be established without knowing which provider produced the candidate, so omission MUST NOT bypass identity completeness. |
| AC-009 | `tc043_root_policy_isolation_test.sh` proves the dispatch-boundary check fails naming the root, the path, and the matched evaluator-only source when an evaluator-only file is planted in (a) the fixture checkout and (b) the scratch Shark project — two independently failing cases, so a guard that walks only `--workdir` cannot pass. A third case proves the check passes on clean roots. A fourth proves the guard aborts **before** any dispatch command is issued: on the failing path a PATH-stubbed dispatcher records **zero** invocations, so "before provider spend" is observed rather than inferred from a non-zero exit status alone. |
| AC-010 | `tc043` additionally proves the admission-time check (REQ-F-010) derives evaluator-only names from the I-04 package at call time: renaming an `oracle_tests[]` entry in a scratch copy of a package changes which name the guard searches for, with no edit to the guard. |
| AC-011 | `tc048_evaluator_access_ordering_test.sh` proves the authorized ordering: a pre-terminal `inject-tests` placement is rejected as `isolation_violation`; a post-terminal placement through `adapter.sh inject-tests` is accepted; an oracle read of `evaluator_only.reference_solution` performed in place is accepted and appends one `evaluator_access` event with accessor, path, digest, phase, and grant time; and the same read performed by first copying the file into the worker checkout before execution completes is rejected naming the violation. |
| AC-012 | `tc049_snapshot_replay_test.sh` replays a stored bundle with no worker rerun and zero provider calls (asserted by a PATH-stubbed provider that fails on invocation), and detects each of the four REQ-F-013 drift kinds independently: modifying a tracked file, adding an untracked file, changing the adapter's normalized test-id set, and deleting one artifact's `consumers` key. Each verdict names the specific path, test id, or artifact. |
| AC-013 | `tc049` proves REQ-F-015 immutability: recomputing `snapshot_digest` over an unmodified stored snapshot reproduces the recorded value, and a one-byte edit to any snapshot field yields `snapshot_mutated` naming the stage. |
| AC-014 | `tc050_partial_evidence_test.sh` proves that for each of the ten named stop outcomes, the bundle retains its partial snapshots, sets `publication_eligible: false`, and carries a non-empty `ineligibility_reasons[]`; and that a bundle pairing any stop outcome with `publication_eligible: true` is rejected. |
| AC-015 | TC-042 asserts each malformed case enumerated in REQ-F-016 exits non-zero with a message naming the failing field, verified by table-driven fixtures under `tests/contracts/testdata/e40_i05/invalid/`. |
| AC-016 | `tc051_evidence_offline_determinism_test.sh` runs every guard with the network disabled and asserts byte-identical verdicts across two consecutive runs over the same bundle, and that no guard issues a provider call (PATH-stubbed provider records zero invocations). |
| AC-017 | `make fmt && make lint && make test` at the repository root is green with the I-05 schema, guards, and fixtures committed; `go list ./...` lists no evidence fixture package; TC-042 passes in CI without `git submodule update --init` and `.github/workflows/ci.yml` is unchanged (REQ-NF-003). |
| AC-018 | A diff review proves REQ-NF-006: `bench/corpus/corpus.yaml`, `collect-run.sh`, `verify-clean-checkout.sh`, `canary-runsurface.sh`, `e40_i01_corpus_contract_test.go`, and every I-04 artifact listed in REQ-NF-006 are byte-unchanged, and no file under `internal/` or `cmd/` is touched. |
| AC-019 | A grep of every generic evidence, isolation, and replay script for `python`, `pytest`, `pip`, `go test`, `golangci-lint`, and `go build` returns hits only inside `bench/adapters/*/`, proving REQ-F-007's language neutrality mechanically rather than by convention — the same mechanical proof F05's AC-012 established. |
| AC-020 | TC-042 asserts REQ-F-017's single-owner rule: every closed vocabulary value the validator accepts is present in `bench/evidence/i05-schema.yaml`, and a value added to the schema file but not to a bundle fixture (or vice versa) surfaces as a named disagreement rather than silently diverging. |

### Out of scope for this feature

- **Capturing live stage evidence.** F06 defines and validates I-05; E40-F08
  populates the runtime records during a real lifecycle ([feature.md](feature.md)
  2026-08-13 amendment). No requirement above observes a running worker.
- Scheduling or dispatching lifecycle work, claiming, heartbeating, or applying
  transitions (E40-F08 / X-11).
- Scoring artifact quality, calibrating a judge, running the execution oracle's
  own comparison, computing comparison identity, or deciding aggregate
  eligibility (E40-F09 / I-08). F06 records `publication_eligible: false` for a
  stop outcome; it does not adjudicate a *passing* run's eligibility.
- Report layouts, baseline commands, spend gates, and the derived
  value-attribution metrics (E40-F10). F06 supplies the raw evidence; it does not
  decide whether a gate or artifact was valuable.
- Defining the interior of I-04 (E40-F05) or of the I-06 replay bundle (E40-F07).
  I-05 references both by path and digest, and types neither.
- Changing `RunResult` / `StageLog`, the transcript byte format, or the I-02
  record schema (X-07, X-08, E40-F02).
- Merging, landing, or depending on E27-F15's unmerged Go implementation, and
  **fixing** the upstream envelope-retention behaviour ADR-F06-11 identifies.
  X-09's obligation is to **verify** the mapping and fail closed, not to ship or
  repair the producer's code; the epic's "triage generic usage-decoder changes
  under their owning epic" rule places the remedy in E27 (see
  [Cross-epic integrations](#cross-epic-integrations) and the proposed Question
  in [Durable unresolved decisions](#durable-unresolved-decisions)).
- Epic-family scenarios and the deferred delivery tail, per epic §"Out of scope".

---

## Architecture

### Component changes

| File | Change |
|---|---|
| `bench/evidence/i05-schema.yaml` | New. The single machine-readable owner (REQ-F-017) of I-05's `schema_version`, required top-level blocks, stage-snapshot field inventory, and every closed vocabulary: `stage_category`, interval category, `artifact_type`, `edge_kind`, `evaluator_access.phase`, stop outcome, and error kind. Plays for I-05 the role `bench/scenarios/scenarios.yaml` plays for I-04, without inheriting any of its fields. |
| `bench/evidence/usage-mapping.yaml` | New. The versioned X-09 binding (REQ-F-009): its own `schema_version`, a `verified_from` provenance block, and one block per provider mapping each semantic slot to a concrete envelope path. `anthropic_claude_cli` is populated from the in-repo confirmed capture; any provider without a verified capture is declared `unmapped` and fails closed. |
| `bench/scripts/verify-evidence-roots.sh` | New. The REQ-F-010 / REQ-F-011 structural guard. `verify-evidence-roots.sh <package.yaml> <fixture_checkout> <scratch_project> <evaluator_root>` derives every evaluator-only path, content digest, and defined test identity from the I-04 package at call time and proves each is absent from **both** agent-visible roots; exits non-zero naming the root, path, and matched source. A **new** I-04-shaped sibling of `verify-clean-checkout.sh`, not a generalization of it — F05 explicitly declined to generalize that corpus.yaml-shaped script (F05 spec, Out of scope). |
| `bench/scripts/verify-stage-evidence.sh` | New. Validates one bundle against `i05-schema.yaml`: field inventory, root policy, completeness split (REQ-F-004), ledger reconciliation (REQ-F-005), candidate identity (REQ-F-006), artifact records including the empty-versus-absent distinction (REQ-F-008), usage slot fail-closed posture (REQ-F-009), access ordering (REQ-F-012), stop-outcome eligibility (REQ-F-014), and snapshot immutability (REQ-F-015). The single named owner of ledger and completeness arithmetic, so F08, F09, and F10 invoke it rather than re-deriving the semantics — the discipline `eval-predicate.sh` established for I-04 and `diff-ledgers.sh` for I-01. |
| `bench/scripts/replay-stage-evidence.sh` | New. Re-evaluates a stored bundle against its named roots with no worker rerun and no provider call, emitting the four REQ-F-013 drift kinds and REQ-F-015's `snapshot_mutated`. |
| `bench/scripts/canary-usagemapping.sh` | New. The X-09 canary (AC-007, AC-021). `canary-usagemapping.sh [--transcript <path>]` asserts every mapped slot resolves against a **real captured envelope**, defaulting to the committed envelope fixtures under `bench/scripts/testdata/run/` and accepting an operator-supplied live transcript. Detects both REQ-F-019 drift classes and keeps them distinct: per-slot `usage_slot_unavailable` versus whole-source `envelope_source_unavailable` (ADR-F06-11). Follows `canary-runsurface.sh`'s discipline verbatim: assert the real shape, never re-derive it from memory, and name the exact drifted field. |
| `bench/scripts/testdata/evidence/` | New. Bundle fixtures for the bench-script test cases: clean roots, each planted-leak case, each ledger case, each drift case, each stop outcome. |
| `bench/scripts/tests/tc043_root_policy_isolation_test.sh` | New. AC-009, AC-010. |
| `bench/scripts/tests/tc044_time_ledger_reconciliation_test.sh` | New. AC-004. |
| `bench/scripts/tests/tc045_candidate_identity_test.sh` | New. AC-005. |
| `bench/scripts/tests/tc046_artifact_record_test.sh` | New. AC-006. |
| `bench/scripts/tests/tc047_usage_mapping_canary_test.sh` | New. AC-007. |
| `bench/scripts/tests/tc048_evaluator_access_ordering_test.sh` | New. AC-011. |
| `bench/scripts/tests/tc049_snapshot_replay_test.sh` | New. AC-012, AC-013. |
| `bench/scripts/tests/tc050_partial_evidence_test.sh` | New. AC-014. |
| `bench/scripts/tests/tc051_evidence_offline_determinism_test.sh` | New. AC-016, AC-019. |
| `bench/scripts/tests/run-all.sh` | Modified. Registers TC-043 through TC-051. The only existing bench file this feature edits. |
| `bench/README.md` | Modified. Adds an "I-05 stage evidence and isolation contract (E40-F06)" section — three-root policy, bundle layout, stage-snapshot field reference, ledger rules, usage slot table, and guard invocation sequence — so E40-F08/F09/F10 read the shape instead of re-deriving it, the role its "Manifest schema" and "I-04 scenario package schema" sections play for I-01 and I-04. |
| `tests/contracts/e40_i05_stage_evidence_contract_test.go` | New, **the only Go file this feature adds** (REQ-NF-001). `package contracts`, TC-042, repository-root-relative artifact reading, in-repo artifacts only (REQ-NF-003), following `tests/contracts/e40_i04_scenario_contract_test.go`. |
| `tests/contracts/testdata/e40_i05/{valid,invalid}/` | New. Table-driven bundle fixtures for AC-001 through AC-003, AC-008, AC-015, AC-020. |

### Data model changes

None. No shark table, column, migration, or `CurrentSchemaVersion` bump. I-05 is
file-backed under `bench/`, consistent with ADR-002 (JSONL/file artifacts are
the only store) and the architecture's "E40 adds no Shark database table."

### API / interface contracts

#### Bundle layout (I-05)

An evidence bundle is one directory under an operator-supplied evidence output
root, following the layout convention `bench/README.md` already documents for
I-02 run directories:

```
<evidence_root>/<scenario_id>/<run_id>/
├── bundle.json                 # identity, roots, stage index, stop outcome, eligibility
├── stages/
│   └── <dispatch_ordinal>-<stage_key>.json   # one immutable snapshot per stage
└── access.jsonl                # append-only evaluator_access events
```

`bundle.json` field inventory. Every field is required unless marked.

| Field | Type | Contract |
|---|---|---|
| `schema_version` | string | The I-05 version `i05-schema.yaml` declares and TC-042 supports. |
| `scenario` | object | `{scenario_id, scenario_version, entity_family}`, copied verbatim from the I-04 package. |
| `run_id` | string | The lifecycle run this bundle belongs to. Opaque to F06; E40-F08 assigns it. |
| `roots` | object | The three REQ-F-002 roots, each `{path, worker_access, identity_digest}`. Pairwise disjoint. |
| `stage_matrix_source` | object | `{package_path, package_digest, prelude, lifecycle}` — the I-04 halves this bundle's completeness is evaluated against (REQ-F-004). A **snapshot taken at run time**, by design: `package_digest` pins the exact package content the run used, so divergence from a later `scenario_version` of the live package is expected evidence of a corpus change, not a bundle error. F06 never rewrites I-04. |
| `stages` | array | Ordered index `{dispatch_ordinal, stage_key, stage_category, snapshot_path, snapshot_digest}`. `dispatch_ordinal` is unique within the bundle. |
| `stop_outcome` | string, optional | Absent on a clean terminal run; one of the ten REQ-F-014 values otherwise. |
| `publication_eligible` | bool | `false` whenever `stop_outcome` is present (REQ-F-014). |
| `ineligibility_reasons` | array of string | Non-empty whenever `publication_eligible` is `false`. |

#### Stage snapshot (I-05)

| Field | Type | Contract |
|---|---|---|
| `dispatch_ordinal` | integer | Matches the bundle index entry; unique per bundle (REQ-F-004). |
| `entity` | object | `{entity_key, entity_type}` for the concrete dispatched entity — never the cascade parent. |
| `stage_key` | string | The workflow step or prelude stage (`D01`–`D05`) this dispatch served. |
| `stage_category` | enum | Closed eight-value set (REQ-F-003). |
| `provider` | string | **Required.** Names the CLI/provider that executed the dispatch (REQ-F-003, REQ-F-009). Its closed set of valid values is owned by `usage-mapping.yaml`, never inlined here (REQ-F-009). A snapshot with no `provider` claim is rejected as `missing_provider` — omitting the field MUST NOT bypass the usage-mapping fail-closed check (REQ-F-009) or the identity-completeness check (REQ-F-018) it gates. |
| `prompt_digest` | string | `sha256` of the rendered prompt as Shark produced it. The prompt text itself is **not** stored in the snapshot. |
| `input_lineage` | array | `{source_kind, path, digest}` for every input the stage consumed — the I-04 `input.agent_visible`, prior-stage artifacts, and fixture state. |
| `replay_lineage` | array, feature family only | `{replay_reference, entry_digest}` pointers into the I-06 bundle. Opaque interior (E40-F07 owns it). |
| `artifacts` | array | REQ-F-008 records. `consumers: []` ≠ absent `consumers`. |
| `usage` | object | REQ-F-009 semantic slots. A slot the mapping could not resolve is **absent**, with a matching `usage_slot_unavailable` entry in `errors[]`. |
| `time_ledger` | object | REQ-F-005: `{stage_start, stage_end, reconciliation_epsilon_ns, intervals: {<category>: [[start, end), …]}}`. |
| `candidate` | object, `code`/`review` only | REQ-F-006 fields. Rejected if identified by `base_commit` alone. |
| `errors` | array | `{kind, detail, …}` where `kind` resolves against `i05-schema.yaml`. Includes `usage_slot_unavailable`, `isolation_violation`, `missing_stage`, `snapshot_mutated`. |
| `rework_count` | integer | Re-entries into this stage for this entity. |
| `evaluator_access` | array | REQ-F-012 events; also appended to the bundle's `access.jsonl`. |
| `snapshot_digest` | string | REQ-F-015 content address, computed over the canonical serialization excluding this field. |

#### Usage slot mapping (X-09)

`bench/evidence/usage-mapping.yaml` binds each semantic slot to a concrete
envelope path per provider. The `anthropic_claude_cli` block is populated from
the in-repo confirmed capture documented at `bench/README.md` §"Confirmed claude
CLI JSON envelope field names" — a real `claude --output-format json` result
envelope captured 2026-08-06 through one real `bench/scripts/run-one.sh`
invocation, not a hand-authored fixture.

| Semantic slot | `anthropic_claude_cli` envelope path | `verification_tier` | Notes |
|---|---|---|---|
| `total_cost` | `total_cost_usd` | `real_capture` | Top-level float. |
| `input_tokens` | `usage.input_tokens` | `real_capture` | The envelope's flat `usage` sub-object (snake_case). |
| `output_tokens` | `usage.output_tokens` | `real_capture` | |
| `cache_read_input_tokens` | `usage.cache_read_input_tokens` | `real_capture` | |
| `cache_creation_input_tokens` | `usage.cache_creation_input_tokens` | `real_capture` | |
| `model_ids` | sorted keys of `modelUsage` | `real_capture` | camelCase object keyed by canonical model ID; the id is the **key**, not a value field. |
| `api_active_duration_ms` | `duration_api_ms` | `real_capture` | Top-level integer, milliseconds. |
| `turn_count` | `num_turns` | `real_capture` | Top-level integer. |
| `provider_session_id` | `session_id` | `unverified` | Corroborated only by E27-F15's hand-authored `internal/runner/testdata/claude-usage-result.json`; **not** listed in the in-repo real-capture confirmation. Fails closed per REQ-F-018 until a canary observes it in a real envelope. |

`required_identity_slots` is `[model_ids, total_cost, input_tokens,
output_tokens, cache_read_input_tokens, cache_creation_input_tokens,
api_active_duration_ms, turn_count]` — every `real_capture` slot, and
deliberately not `provider_session_id` (REQ-F-018, ADR-F06-12).

`openai_codex_cli` is declared `unmapped`: `buildCodexArgs` on `main` does not
pass `--json`, so a codex stage's transcript stdout is not a decodable envelope
today, and every slot fails closed (REQ-F-009). This matches
`collect-run.sh`'s own reserved-but-unused `usage_unavailable` error kind.

Consumers MUST read a slot through its semantic name and MUST NOT hard-code an
envelope path — the same opacity discipline F05 fixed for `toolchain_identity`
(F05 REQ-F-008: an ordered list consumers compare and hash without reading a key
by name). No top-level `model` field has ever been observed in a real capture, so
no `model`-fallback path is specified; a `modelUsage`-absent envelope is its own
`usage_slot_unavailable` for `model_ids` (ADR-F06-04).

### Key technical decisions

ADRs are grouped by subject rather than by number: ADR-F06-11 sits directly after
ADR-F06-04 because it extends the same X-09 decision, and ADR-F06-12 closes it.
The identifiers are stable; only the reading order is topical.

**ADR-F06-01 — I-05 is additive evidence keyed to existing stages, never a
mutation of `RunResult` / `StageLog`.** `internal/runner/controller.go:84-135`
carries only `status`, `action`, `agent_type`, `provider`, `duration_ns`,
`exit_code`, and a truncated `output_summary` populated for successful
`spawn_agent` stages — no tokens, cost, candidate digest, artifact record, or
interval ledger (research finding 1). That field set is an X-07/X-08 contract
`bench/scripts/canary-runsurface.sh` already pins against a real invocation, so
mutating it would break a committed canary and an active E22 surface to serve a
benchmark. I-05 is therefore a separate, separately-addressed document keyed to
the same dispatch. Rejected alternative: widening `StageLog` with usage and
candidate fields, which spends the epic's Go-change budget on evidence plumbing
that no shark user needs and puts benchmark concerns inside the product runner.

**ADR-F06-02 — Stage completeness is split, because the two halves of I-04's
stage matrix have different applicability oracles.** F05's ADR-F05-02 states the
binding consequence verbatim: "with the lifecycle half resolved at run time,
stage completeness is one snapshot per dispatch, and a 'named missing-stage
failure' is only detectable for the prelude half." The prelude half enumerates
`D01`–`D05` with explicit booleans ahead of time, so an applicable stage with no
snapshot is genuinely detectable as `missing_stage`. The lifecycle half declares
`mode: all_dispatched`, resolved against the variant workflow bundle at run time,
so no artifact states which lifecycle stages *should* have been dispatched — only
which *were*. REQ-F-004 encodes both oracles rather than papering over the
difference. Rejected alternative: requiring I-04 to enumerate lifecycle statuses
so one oracle covers both halves — ADR-F05-02 already rejected that as wrong for
every variant but one, and ADR-006 forbids a second drifting copy of workflow
routing.

**ADR-F06-03 — Isolation is a path-presence guard against the real live roots,
modeled on `DefaultDisallowedTools`' posture but not its mechanism.**
`internal/runner/dispatcher.go`'s `DefaultDisallowedTools` is this codebase's one
structural precedent for blocking a category of worker capability by
construction, and its posture — a checkable list, enforced before the worker
acts, failing loud — is exactly right. Its *mechanism* is a command-string
denylist, which cannot express "this file must not exist in this tree" (research
finding 3). REQ-F-011 is therefore a new check that inspects the actual fixture
checkout and the actual scratch Shark project at dispatch time, following
`canary-runsurface.sh`'s "assert the real shape, never re-derive from memory"
discipline (research finding 4). Two consequences are load-bearing. First, the
check must walk **both** agent-visible roots: the architecture's own two-root
warning ("one run spans two roots") means a guard that walks only `--workdir`
misses everything Shark writes into the scratch project. Second,
`verify-clean-checkout.sh` is not extended — it is `corpus.yaml`-shaped and F05
explicitly declined to generalize it, so REQ-F-010/011's guard is a new
I-04-shaped sibling. Rejected alternative: trusting I-04's declared `evaluator/`
versus `input/` path separation, which TC-030 already asserts — that proves the
package *labels* the split correctly, not that the running worker cannot reach
the files, and the feature's acceptance boundary demands the runtime property
(research Decision 5).

**ADR-F06-04 — X-09 binds through a versioned mapping file of semantic slots,
never through envelope field names written inline into the I-05 schema, and never
through E27-F15's Go structs.** Four verified facts force this shape.

*(1) E27-F15 is unmerged and mid-rework.* Its HEAD `b52dfdd9` is not an ancestor
of `main`; `internal/runner/claude_dispatcher.go` and `codex_dispatcher.go` on
`main` perform **zero** JSON decoding — `DispatchResult` is a flat
`{ExitCode, Stdout, Stderr, Duration, Command}` with no `Usage` field — and the
branch's own capture task (E27-F15-002) is still in `development` with open
review findings. Binding I-05 to that branch's structs would bind the benchmark
to code that does not exist on `main` and is not signed off on the branch.

*(2) E27-F15's mapping is a strict subset of what I-05 needs, and disagrees with
the real envelope.* Its `claudeJSONResult` decodes
`{type, session_id, request_id, result, model, total_cost_usd, usage{...}}`.
It carries **no** `num_turns`, **no** `duration_api_ms`, and **no** `modelUsage`
— a repo-wide grep finds zero hits for the first two anywhere on the branch. It
instead reads a top-level `model` field that no real capture has ever contained.
Adopting E27-F15's field set verbatim would silently drop `api_active_duration_ms`,
`turn_count`, and `model_ids` from I-05, and would introduce a fallback path
tested only against a hand-authored fixture. That is precisely the "copy the
parser by assumption" the architecture's Metric-collection section forbids.

*(3) The repo already holds a better audit.* `bench/README.md` §"Confirmed claude
CLI JSON envelope field names" records the field set observed in a **real**
captured envelope (2026-08-06, one real `run-one.sh` invocation), and
`bench/scripts/collect-run.sh` already consumes it with an absent-never-zero
posture. X-09's obligation — "verify the current field mapping; do not invent
missing fields" — is satisfied by that verified capture, not by waiting on a
merge. REQ-F-018's `verification_tier` records exactly which slots that capture
covers and which (only `provider_session_id`) rest on E27-F15's fixture alone.

*(4) Lifecycle v2 dispatches more than one provider.* `buildCodexArgs` on `main`
does not pass `--json`, so codex transcripts are not decodable envelopes today;
`collect-run.sh` already reserves `usage_unavailable` for that case.

A mapping file with semantic slots, per-slot verification tiers, and a real-capture
canary satisfies all four. This is structurally the move F05 already made for
`toolchain_identity` (REQ-F-008: ordered, opaque, compared without reading a key
by name), applied to usage. Rejected alternatives: (a) typing envelope field names
directly into the I-05 schema — F05's ADR-F05-05 deliberately kept I-04 clear of
this so F06 could decide it here; (b) parsing `collect-run.sh`'s Python constants
to prove agreement — brittle source-scraping, where binding the canary to the
committed envelope *fixtures* both scripts already assert against couples the two
through a shared artifact instead; (c) implementing the `model` fallback the
design doc assumed — no real capture has ever carried a top-level `model` field,
so implementing it would mean testing against an unobserved shape, and F02 already
resolved this the same way; (d) taking a merge dependency on E27-F15 — the
architecture assigns F06 *verification*, not delivery of the producer's code, and
epic §"Out of scope" places generic usage-decoder changes under their owning epic.

**ADR-F06-11 — The canary must detect envelope-*availability* drift, not only
field drift, because a plausible upstream change destroys I-05's usage source
entirely.** I-05's usage slots read the retained provider envelope from the stage
transcript's `---STDOUT---` block. That block is a decodable JSON envelope on
`main` only as a byproduct: `buildClaudeArgs` passes `--output-format json` and
neither dispatcher touches `result.Stdout`, so `writeTranscript` persists the raw
envelope. On the `E27-F15-cross-session-usage-tracking` branch this no longer
holds — both `Dispatch()` implementations overwrite `result.Stdout` with the
extracted assistant text *before* `controller.go`'s `maybeWriteTranscript` call,
so the transcript retains prose and the raw envelope is discarded. If that branch
merges unchanged, every I-05 usage slot and F02's whole collector lose their
source at once, and a field-only canary would report nine independent
`usage_slot_unavailable` errors rather than the one true cause. REQ-F-019
therefore makes `envelope_source_unavailable` its own named drift class, decoded
rather than inferred from file presence.

AC-021(b) proves this branch with a transcript whose `---STDOUT---` block is
arbitrary **non-JSON** text. That is deliberately *not* a fabricated envelope: it
asserts what the decoder does with non-envelope input, and makes no claim that any
upstream artifact does or will look a particular way. The distinction matters
because ADR-F06-04(c) rejects testing against an unobserved *shape* — a fixture
asserting "the envelope contains field X" when no capture ever showed X. A
negative case over arbitrary text contains no shape claim to be wrong about, so
it does not fall under that prohibition. If a real post-merge transcript is ever
captured, it becomes an additional fixture; it does not replace this one.

This is ADR-004's canary reasoning
("converts a silent metric corruption into a loud failure") applied one level up,
from field names to the existence of the field-bearing artifact. It is also the
concrete instance of X-09's stated purpose — fail loudly when the upstream
contract changes — rather than a hypothetical guard. The upstream remedy (having
E27-F15 preserve the raw envelope, or expose `UsageObservation` as a second
retained source) belongs to E27 under the epic's "triage under the owning epic"
rule; F06 owns only the detection and the fail-closed consequence, and files the
proposal recorded in [Durable unresolved decisions](#durable-unresolved-decisions).

**ADR-F06-05 — `test_suite_digest` is derived from the adapter's normalized
test-id set, not from test file contents.** F05's REQ-F-007 forbids any generic
lifecycle, evidence, or evaluation component from branching on fixture language,
package manager, or toolchain, and its AC-012 proves that mechanically by grep.
Hashing test *files* would require knowing which paths are tests in Python versus
Go — precisely such a branch. `adapter.sh test` already emits ids normalized to
`<module-or-package>::<test-name>` specifically so no consumer performs
language-aware parsing; digesting that sorted id set plus adapter name, adapter
version, and `toolchain_identity` captures every change that can alter which
tests run, with zero language knowledge in the digest itself. Rejected
alternative: a seventh adapter capability returning a language-computed digest —
ADR-F05-04 already set the bar that a new verb requires a `schema_version` bump
and a consumer that actually needs it, and the existing `test` output suffices.

**ADR-F06-06 — The ledger is half-open disjoint intervals with a named epsilon
and an explicit `unclassified` remainder.** "The intervals reconcile to stage
wall time" is not testable without stating *how closely*: clock granularity,
process-spawn overhead, and the boundary between two adjacent measurements all
produce sub-millisecond residuals that no implementation can drive to exactly
zero. Half-open `[start, end)` intervals make adjacency unambiguous and make
overlap detection a pure comparison rather than a tie-breaking convention;
`reconciliation_epsilon_ns` makes AC-004 a real assertion; and routing every
residual to `unclassified` preserves the architecture's binding rule that "the
collector never assigns an unknown interval to model work." That last property —
unknown time never lands in `provider_active` — is the one UAT-16 actually turns
on. Rejected alternative: a single duration scalar per category, which cannot be
checked for overlap at all and would let two categories silently double-count the
same wall time.

**ADR-F06-07 — An empty consumer array and an absent consumer key are two
distinct, both-legal states.** The architecture states the rule directly: "An
empty set means no observed consumer; a missing set means incomplete evidence."
Collapsing them would make an orphaned planning artifact indistinguishable from
one whose consumption was never instrumented, and UAT-18 exists precisely to
distinguish a consumed artifact from an orphan. The schema therefore treats
`consumers: []` and an absent `consumers` key as different values, and REQ-F-008
forbids coercion in either direction. Rejected alternative: a nullable array with
`null` for "not collected" — JSON-schema-legal but a coercion trap in every
language whose YAML/JSON decoder maps a missing key and an explicit null to the
same zero value.

**ADR-F06-08 — Evaluator access is ordered and recorded, not merely restricted.**
Absence at dispatch is necessary but not sufficient: the same held-back oracle
tests that must be *absent* before dispatch must be *present* after terminal
status, and the post-run oracle must read the reference solution without copying
it into the worker checkout before execution completes (feature.md acceptance
boundary 3). A pure absence rule would either forbid the legitimate post-terminal
injection or permit a pre-terminal copy. REQ-F-012 therefore specifies the
ordering — absent at dispatch, injected only through I-04's `adapter.sh
inject-tests` after terminal status, read in place otherwise — and requires every
access to append an `evaluator_access` event, so the bundle records not just that
isolation held but exactly when and by whom it was lifted. Reusing
`inject-tests` rather than a new copy path keeps the one language-specific
placement rule where F05 put it and nowhere else.

**ADR-F06-09 — Schema validation and execution guards are separate owners.**
`tests/contracts/e40_i05_stage_evidence_contract_test.go` (TC-042) validates
structure from in-repo artifacts only and never requires a populated submodule,
because CI's `actions/checkout@v4` does not initialise submodules; a validator
touching a fixture tree would fail in CI and force a workflow change this feature
does not need. Execution-based isolation, replay, and canary guards live in
`bench/scripts/` and do require real roots. This mirrors ADR-F05-07 and E40-F01's
TC-001/`admit.sh` split rather than inventing a third convention.

**ADR-F06-10 — Q003 is reused and updated, not duplicated.** The open Q003
("Which envelope field names does the E40-F02 transcript parser depend on?") is
the same decision X-09 poses for I-05, framed for the earlier consumer. The
question-management skill's deduplication rule is explicit: reuse or update a
Question that asks the same decision; never create a duplicate merely because it
appears in another design document. Q003 gains E40-F06/X-09 as an affected
surface and this spec plus `bench/evidence/usage-mapping.yaml` as decision
sources. No Q007 is minted for the usage mapping.

**ADR-F06-12 — `provider_session_id` is excluded from required comparison
identity, so no `unverified` slot can gate G14.** Two independent reasons.
*Evidentiary:* the in-repo real-capture confirmation does not list `session_id`;
only E27-F15's hand-authored `claude-usage-result.json` carries it, which is
evidence of what its author needed, not of the live envelope. *Semantic:* the
field is not cross-provider coherent — E27-F15's own decode reads `session_id`
for Claude and `thread_id` for Codex, and `UsageObservation` flattens both into
one `NativeSessionID` whose meaning differs by provider. A slot that means a
different thing per provider cannot serve as a uniform comparison field, which is
exactly what G14 requires ("present and **uniform**"). Excluding it is therefore
correct on its own merits, not a workaround for the missing capture. The
consequence is named rather than left implicit: I-05 asserts provider identity
through `model_ids` plus the cost and token slots, all `real_capture`, so
REQ-F-018's fail-closed rule never renders every run ineligible on day one — the
outcome that would follow if an `unverified` slot were required. Rejected
alternative: keeping `provider_session_id` required and flagging the tier gap as
a live G14 blocker — that would make E40-F09's aggregation unreachable until an
unrelated epic merges, trading a real capability for a field no comparison
actually needs. `provider_session_id` remains **recorded** when the envelope
supplies it (it is useful for tracing a run back to a provider session); it is
simply not part of the identity that must be uniform.

### Integration with existing code

Nothing under `internal/` or `cmd/` is called, imported, or extended
(REQ-NF-001). The integration surfaces are read-only artifacts, executables, and
conventions:

- **I-04 read-only consumption** — the guards read
  `bench/scenarios/scenarios.yaml` and `bench/scenarios/packages/*/package.yaml`
  for `evaluator_only`, `toolchain_identity`, and both `stage_matrix` halves, and
  invoke `bench/adapters/<name>/adapter.sh test` and `inject-tests` and
  `bench/scripts/checkout-scenario-fixture.sh` rather than re-deriving language
  commands. No I-04 artifact is edited (REQ-NF-006).
- **Frozen Phase 1 surface** — `internal/runner/controller.go`'s `RunResult` /
  `StageLog` keys and `internal/runner/transcript.go`'s byte format are
  *referenced* (snapshots point at transcript paths and digests) and never
  reparsed, redefined, or mutated. `canary-runsurface.sh` keeps owning that
  assertion; `canary-usagemapping.sh` owns only the envelope-slot binding.
- **Envelope fixtures as the shared coupling** — `canary-usagemapping.sh` asserts
  against the committed envelope fixtures under `bench/scripts/testdata/run/`
  that F02's own tests already assert are envelope-shaped, so a change to the
  confirmed envelope shape fails both features' tests loudly instead of drifting
  one against the other. `collect-run.sh` itself is not edited or source-scraped.
- **Structural-guard pattern, not code** — `internal/runner/dispatcher.go`'s
  `DefaultDisallowedTools` supplies the posture (checkable, pre-action,
  fail-loud); none of `internal/runner`'s code is imported or copied
  (ADR-F06-03).
- **Contract-test convention** —
  `tests/contracts/e40_i05_stage_evidence_contract_test.go` joins `package
  contracts` and follows the repository-root-relative artifact-reading helper
  style of `tests/contracts/e40_i04_scenario_contract_test.go` and
  `tests/contracts/e40_i01_corpus_contract_test.go`.
- **Bench test-runner convention** — the new TC scripts follow the existing
  `bench/scripts/tests/tcNNN_*.sh` naming and are registered in
  `bench/scripts/tests/run-all.sh`, the only existing bench file this feature
  edits.
- **Scratch-environment discipline** — evidence tooling operates only inside
  caller-supplied roots, never invokes shark project-initialisation commands, and
  never writes to the live tree, database, or `.sharkconfig.json` (REQ-NF-005).

---

## Cross-feature interactions

### Produces: I-05 — Stage evidence and isolation contract

| Property | Contract |
|---|---|
| Consumers | E40-F08 Canonical multi-entity lifecycle runner; E40-F09 Calibrated evaluation and comparison identity; E40-F10 Operator workflow and retained lifecycle baseline |
| Shape source | [Stage evidence and isolation contract](../architecture.md#stage-evidence-and-isolation-contract) |
| Payload | Three-root access policy; immutable stage snapshot; non-overlapping time ledger; exact candidate snapshot; typed artifact producer, consumer, and access records; prompt, input, replay, output, usage, cost, error, rework, digest, and evaluator-access lineage |
| Style | File artifact and access policy |
| Shared contract test | `tests/contracts/e40_i05_stage_evidence_contract_test.go#TC-042` |
| Consumer reads | `bench/evidence/i05-schema.yaml`, `bench/evidence/usage-mapping.yaml`, and the "I-05 stage evidence and isolation contract" section of `bench/README.md` |
| Consumer invokes | `bench/scripts/verify-evidence-roots.sh`, `verify-stage-evidence.sh`, `replay-stage-evidence.sh`, and `canary-usagemapping.sh`, rather than re-deriving isolation, ledger, completeness, or usage-slot semantics |
| Consumer split | E40-F08 writes bundles at run time and calls `verify-evidence-roots.sh` at every dispatch boundary; E40-F09 reads candidate identity, usage slots, artifact records, and eligibility for comparison identity and evaluation; E40-F10 reads the ledger, artifact-consumption edges, and eligibility to derive lifecycle and diagnostic reports |
| Test scope | TC-042 reads in-repo evidence artifacts only and requires no populated submodule, so `.github/workflows/ci.yml` is unchanged (ADR-F06-09); execution guards TC-043 through TC-051 run in `bench/scripts/tests/run-all.sh` |
| Gate mode | `contract-only`, staged by [the interaction map](../E40-interaction-map.md#i-05-staged-edge) — F06's producer necessarily runs before its consumers are decomposed |
| Activation owner | E40-F08; E40-F09; E40-F10 — each closes its own consumption independently at its own UAT |
| Closure key | E40-F08 / E40-F09 / E40-F10, respectively |
| Counterpart status | Read live from Shark at review/UAT time; not copied here as a fact that would go stale |
| Review basis | This spec.md and the interaction map row, present together at F06 task_review |
| Demonstrability disposition | `pending-integration` until each consumer's live wiring closes |
| Closure owner (F06 side) | E40-F06 code-review owner, for the producer half of the contract only |
| Required UAT evidence | UAT-09 (F06): inspect both agent-visible roots immediately before every dispatch and prove references, answer keys, patches, and hidden tests absent; after the stage or run, grant recorded evaluator access and replay the snapshot without rerunning the worker. UAT-16, UAT-18, and UAT-19 additionally validate the ledger, artifact-use, and candidate-identity halves with E40-F08/F09/F10. Each consumer's own UAT proves its live wiring per the interaction map's activation-owner closure requirement. |

E40-F08, E40-F09, and E40-F10 must copy the shape source and the contract-test
pointer above verbatim; the same test proves every side of this contract and no
twin test is created. This `contract-only` staging is a predeclared handoff, not
a waiver — an open internal activation obligation blocks epic completion until
each consumer closes it.

### Consumes: I-04 — Lifecycle scenario package contract

| Property | Contract |
|---|---|
| Producer | E40-F05 Lifecycle scenario corpus and adapter contract |
| Shape source | [Lifecycle scenario package contract](../architecture.md#lifecycle-scenario-package-contract) |
| Payload | Versioned scenario identity, family, stage matrix, fixture and adapter, visible input, replay and evaluator references, resource policy, final predicate, and admission result |
| Style | File artifact |
| Shared contract test | `tests/contracts/e40_i04_scenario_contract_test.go#TC-030` |
| F06's consumption slice | `evaluator_only`, `toolchain_identity`, and **both** stage-matrix halves — assigned verbatim by [the I-04 staged edge](../E40-interaction-map.md#i-04-staged-edge) and F05's spec Consumer-split row |
| How F06 consumes it | `evaluator_only` supplies the path, digest, and test-identity set REQ-F-010/011 prove absent; `toolchain_identity` participates in `test_suite_digest` (REQ-F-007); `stage_matrix.prelude` supplies the enumerated applicable set for `missing_stage`, and `stage_matrix.lifecycle` fixes the one-snapshot-per-dispatch oracle (REQ-F-004, ADR-F06-02) |
| Non-regression obligation | Every I-04 artifact listed in REQ-NF-006 is byte-unchanged by this feature; consumption is read-only |
| Gate mode | `contract-only`, staged by [the I-04 staged edge](../E40-interaction-map.md#i-04-staged-edge); E40-F06 is the activation owner for **its own slice** and closes it at its own UAT (UAT-09) with a real caller chain, shared-contract evidence, a production-path integration test, and a wiring-removal counterfactual |
| Closure key | E40-F06, at its own UAT |

No twin test is created for I-04: TC-030 remains the single shared proof of that
contract, and F06's guards consume the artifacts TC-030 validates.

---

## Cross-epic integrations

### Consumes and validates: X-09 — Provider-usage field mapping

| Property | Contract |
|---|---|
| Producer epic / feature | E27 — Shark Status Viewer (E27-F15 Codex and Claude cross-session usage tracking) |
| Consumer epic / feature | E40 — Shark Bench. **E40-F06 is the contract owner; E40-F08 is the runtime writer** (map row, verbatim) |
| Integration purpose | Reuse the audited provider-usage field mapping for stage evidence and comparison identity; do not invent missing token, cost, model, session, or timing fields |
| Contract / shape source | E40 architecture "Stage evidence and isolation contract"; E27-F15 approved usage metadata contract and implementation artifacts |
| UX / CX handoff notes | Internal. Missing required provider identity invalidates the run instead of degrading to an incomparable aggregate |
| What F06 produces for it | `bench/evidence/usage-mapping.yaml` — the versioned, provenance-carrying binding from I-05's semantic slots to concrete envelope paths (REQ-F-009), plus `canary-usagemapping.sh`, which re-verifies that binding against a real captured envelope |
| What F06 validates | That the mapping resolves against a real capture, that an unresolvable slot yields a named `usage_slot_unavailable` and an **absent** field rather than a zero, and that a provider with no verified mapping is declared `unmapped` and fails closed (AC-007, AC-008) |
| Verification posture | The map notes "E27-F15 is still active, so E40-F06 must verify its current artifact and implementation state before depending on it." **Verified 2026-08-14**: E27-F15 HEAD `b52dfdd9` is not an ancestor of `main`; both dispatchers on `main` decode nothing (`DispatchResult` has no `Usage` field); the branch's decode covers `{type, session_id, request_id, result, model, total_cost_usd, usage{…}}` with no `num_turns`, `duration_api_ms`, or `modelUsage`; and its capture task is still in `development` with open review findings. The repo separately holds a **real-capture** confirmation of the live envelope (`bench/README.md` §"Confirmed claude CLI JSON envelope field names", 2026-08-06) that `collect-run.sh` already consumes. F06 binds to the verified capture and its canary, not to unmerged branch code — satisfying "verify before depending" without taking a merge dependency (ADR-F06-04) |
| Upstream drift risk surfaced | On the E27-F15 branch, both `Dispatch()` implementations overwrite `result.Stdout` with extracted assistant text before the transcript is written, so a merge would discard the raw envelope I-05's usage slots read. F06 detects this as the named `envelope_source_unavailable` drift class (REQ-F-019, ADR-F06-11, AC-021) and fails closed; the upstream remedy belongs to E27 (proposed Question in [Durable unresolved decisions](#durable-unresolved-decisions)) |
| Test coverage | UAT-09 and UAT-14; `bench/scripts/canary-usagemapping.sh` driven by `tc047_usage_mapping_canary_test.sh` (AC-007, AC-021) and TC-042's fail-closed cases (AC-008). The map states "consumer contract tests are owned by the E40-F06/E40-F08 workflows"; these are F06's half |
| Deferral | None. No X-09 obligation is deferred to `docs/product/progress.md` |

E40-F06 produces, consumes, or validates **no other X-## row**. X-07 is owned by
E40-F02, X-08 by E40-F04, X-10 by E40-F07, X-11 and X-13 by E40-F08, and X-12 by
E40-F09. F06 does not invoke the Rider product-design action, the keyed dispatch
loop, or the Question lifecycle, and it does not hash installed Shark-data
content — I-05 *records* a Shark-data content digest field only when E40-F09
supplies it under X-12.

---

## Durable unresolved decisions

Applying the materiality test in `skills/question-management/SKILL.md`. One
existing Question is reused and one new Question is **proposed** to the parent
loop; F06 itself creates, claims, responds to, and resolves nothing — the parent
owns every Question transition (skill §"Record and route", step 5).

1. **Provider-usage envelope field names (X-09).** Material — it changes a
   cross-feature contract and F09's fail-closed identity posture. **Q003 already
   asks exactly this decision** ("Which envelope field names does the E40-F02
   transcript parser depend on?") and remains `open` in Shark. The skill's
   deduplication rule requires reusing it rather than minting a twin. Recommended
   parent-loop action: update Q003 to add E40-F06/X-09 as an affected surface and
   link this spec plus `bench/evidence/usage-mapping.yaml` as decision sources;
   the substantive answer is already recorded in `bench/README.md` §"Confirmed
   claude CLI JSON envelope field names" and is bound here by REQ-F-009 and
   ADR-F06-04. F06 does not itself create, claim, respond to, or resolve any
   Question — the parent loop owns those transitions.
2. **Reconciliation epsilon magnitude.** Non-material. The *existence* of a named
   epsilon is a contract requirement (REQ-F-005, AC-004) and is settled here; its
   numeric value is an implementation tuning constant that changes no scope,
   acceptance criterion, cross-feature contract, or entity gate — any value that
   keeps AC-004's overlap and residual cases discriminating satisfies the
   criterion. Recorded here per the skill's "record the rationale in the working
   document instead" guidance.
3. **Evidence output root location.** Non-material, and settled by the same
   reasoning F05 applied to fixture hosting (ADR-F05-09 / ADR-F01-01): every
   consumer reaches a bundle through the operator-supplied `<evidence_root>` path
   recorded in `bundle.json`, so no contract changes with the answer. E40-F10
   owns the operator-facing default.
4. **Lifecycle-half "should have been dispatched" completeness.** Not open — it
   is *closed as undecidable* by ADR-F05-02 and encoded as REQ-F-004's split
   oracle plus ADR-F06-02. This is a recorded design consequence, not a pending
   decision.
5. **Does E27-F15's merge destroy the retained provider envelope that I-05 and
   I-02 both read? — NEW, proposed to the parent loop.** Material: it changes a
   cross-epic contract (X-09), invalidates a completed feature's collector
   (E40-F02's `stages[].usage`), and determines whether I-05 can carry verified
   usage identity at all. Not a duplicate of Q003, which asks *which field names*
   the envelope carries; this asks *whether the envelope survives to be read*.
   Deduplication check: no existing Question (Q001–Q006) asks it.
   - **Context.** `writeTranscript` persists `DispatchResult.Stdout`. On `main`
     that is the raw `--output-format json` envelope. On
     `E27-F15-cross-session-usage-tracking`, both `ClaudeDispatcher.Dispatch` and
     `CodexDispatcher.Dispatch` overwrite `result.Stdout` with extracted assistant
     text before `controller.go`'s `maybeWriteTranscript` runs, so the transcript
     would retain prose only.
   - **Options.** (a) E27-F15 preserves the raw envelope in the transcript and
     exposes `UsageObservation` additionally; (b) E27-F15's `UsageObservation` /
     `agent_usage_events` becomes a named, tested second source for I-05's slots
     — **already pre-authorized by E40's architecture**, which states "A named,
     tested fallback source is acceptable; silently dropping the field is not",
     so (b) is a permitted resolution rather than a new ask on E40; (c) E40
     accepts the loss and every affected slot fails closed permanently.
   - **Consequence.** Under (c), G14 comparison identity becomes unsatisfiable for
     any run dispatched after the merge, and E40-F02's completed `usage` block
     silently empties.
   - **Affected surfaces.** E27-F15, E40-F02, E40-F06 (this spec, REQ-F-019 /
     ADR-F06-11), E40-F08 (runtime writer), E40-F09 (fail-closed identity).
   - **Evidence paths.** `internal/runner/transcript.go`; `internal/runner/{claude,codex}_dispatcher.go`
     on `main` versus branch `b52dfdd9`; `internal/runner/dispatcher.go`
     (`DispatchResult`, branch `UsageObservation`); `bench/scripts/collect-run.sh`;
     `bench/README.md` §"Confirmed claude CLI JSON envelope field names".
   - **Routing.** Requester `E40-F06 specification`; the decision belongs to E27
     under the epic's "triage generic usage-decoder changes under their owning
     epic" rule. Not `--blocking` for E40-F06: REQ-F-019's detection and
     fail-closed consequence are specifiable and testable today regardless of the
     answer.

Epic-level Q001 and Q002 remain resolved. Q004's Phase 1 cascade-attribution
constraint is superseded for lifecycle v2 by E40-F08's per-dispatch I-05/I-07
evidence (ADR-005) and is not this feature's concern; the underlying `shark run`
defect remains open on its own surface and must not be marked fixed here.

---

## Verification plan

| Requirement | Evidence |
|---|---|
| REQ-F-001 | AC-001, AC-018 — `tests/contracts/e40_i05_stage_evidence_contract_test.go#TC-042` |
| REQ-F-002 | AC-002 |
| REQ-F-003 | AC-001, AC-020 |
| REQ-F-004 | AC-003 — epic G9; UAT-09 |
| REQ-F-005 | AC-004 — epic G16; UAT-16 |
| REQ-F-006 | AC-005 — epic G19; UAT-19 |
| REQ-F-007 | AC-012 (test-suite drift branch), AC-019 |
| REQ-F-008 | AC-006 — epic G18; UAT-18 |
| REQ-F-009 | AC-007, AC-008 — X-09; UAT-14 |
| REQ-F-018 | AC-008 (`verification_tier` branch), AC-022 — X-09; epic G14; UAT-14 |
| REQ-F-019 | AC-021 — X-09; UAT-09, UAT-14 |
| REQ-F-010 | AC-010 |
| REQ-F-011 | AC-009 — epic G9; UAT-09 |
| REQ-F-012 | AC-011 — UAT-09 |
| REQ-F-013 | AC-012 — UAT-09, UAT-19 |
| REQ-F-014 | AC-014 — epic G12 |
| REQ-F-015 | AC-013 |
| REQ-F-016 | AC-015 |
| REQ-F-017 | AC-020 |
| REQ-NF-001, REQ-NF-002 | AC-017, AC-018 |
| REQ-NF-003 | AC-017 |
| REQ-NF-004 | AC-016 |
| REQ-NF-005 | Diff review: no shark initialisation command and no live-root write in `bench/scripts/` or `bench/evidence/` |
| REQ-NF-006 | AC-018 |
| REQ-NF-007 | AC-004 (category-attribution branch) |

---

*Last Updated*: 2026-08-14
