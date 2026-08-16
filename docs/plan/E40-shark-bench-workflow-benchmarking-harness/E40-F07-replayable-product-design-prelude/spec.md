---
feature_key: E40-F07-replayable-product-design-prelude
epic_key: E40
title: Replayable product-design prelude — combined requirements and architecture
date: 2026-08-15
---

# E40-F07 Specification: Replayable product-design prelude

Business context is not restated here. See epic PRD [epic.md](../epic.md)
§"Success gates" (G10, G12, G18) and §"Feature breakdown" (E40-F07), and
[feature.md](feature.md) for this feature's outcome, scope, and acceptance
boundary. System-level decisions are in [architecture.md](../architecture.md) —
this spec implements the shape already fixed by
[Product-design replay contract](../architecture.md#product-design-replay-contract),
ADR-006, ADR-007, ADR-008, and the I-06 row plus
[I-06 staged edge](../E40-interaction-map.md#i-06-staged-edge) of
[E40-interaction-map.md](../E40-interaction-map.md).

Capability reuse is settled by the validated
[research report](research-report.md) Capability map. In summary: the Shark
Rider product-design action and the bundled D01-D14 methodology are **wrapped,
never forked** — `skills/shark-rider/verbs/product-design.md` and
`internal/sharkdata/default_data/skills/product-design/**` are byte-frozen by
REQ-NF-006, which is what makes "do not copy the methodology" mechanical rather
than aspirational (row 1); `docs/product/progress.md` remains the wrapped
action's own derived record and F07 adds no second progress surface (row 2);
`DefaultDisallowedTools` supplies the *posture* of a checkable, pre-action,
fail-loud capability block but has no substitution path, so the substitute
answer source is new benchmark-adapter code and **not** a change to
`internal/runner` (row 4); I-04's `scenarios.yaml` and `package.yaml` are
**read-only inputs** (row 5); I-05's three-root model is **inherited, not
extended with a fourth root** (row 6); X-13's durable Question lifecycle is
**not this feature's mechanism** and belongs to E40-F08 (row 7);
`bench/scripts/replay-manifest.sh` supplies the epic's *replay-and-verify
discipline as a pattern only*, not code to copy (row 8).

**Scope boundary that binds every requirement below.** F07 delivers the I-06
*schema, replay bundle, resolver, prelude runner, guards, and documented
contract* for D01-D05 on the **feature** family, plus the explicit
non-applicable record for the other three families. It does not start the keyed
Shark entity lifecycle (E40-F08 / I-07), does not evaluate artifact quality
(E40-F09 / I-08), and does not publish a report (E40-F10). I-06 references I-05
stage snapshots and I-04 packages **by path and digest and types neither** — the
mirror of the line F06's spec already wrote about I-06.

---

## Requirements

### Functional requirements

| ID | Requirement | Traces to |
|---|---|---|
| REQ-F-001 | I-06 MUST be **two** schema-versioned, file-backed documents with distinct roles, and the spec, schema, and `bench/README.md` MUST name them unambiguously so no consumer looks for the wrong file: (a) the **replay bundle** — the committed, versioned *input*, the file I-04's `replay_reference` points at; and (b) the **replay result** — the per-run *output* document E40-F08 consumes, which the interaction map names "product-design replay result". Both MUST declare `schema_version`. Neither may redefine, retype, or duplicate any I-04 or I-05 field; both reference those contracts by path and digest only. | architecture Product-design replay contract; interaction map I-06 row; research Decision 4; ADR-F07-08 |
| REQ-F-002 | The replay bundle MUST carry `schema_version`, `bundle_version`, a `scenario_binding` of `{scenario_id, scenario_version}`, and an ordered `entries[]`. Each entry MUST carry `{entry_id, stage, ordinal, request_kind, topic_key, required, response, response_digest, entry_digest}`. `stage` MUST be one of `D01`–`D05`; `request_kind` MUST be `human_question` or `research_query`; `ordinal` MUST be unique within its stage; `response` MUST be either an inline string or a `{path, digest}` reference resolved relative to the bundle file. Every entry is **single-use**: an entry supplied once is consumed and MUST NOT be supplied again within the same prelude run. | feature.md Scope 2; architecture ("an *unused* authorized entry"); ADR-F07-04 |
| REQ-F-003 | `entry_digest` MUST be a `sha256` over the entry's canonical serialization excluding the digest field itself, and it is the **single join key** between I-06 and I-05: E40-F08 writes it verbatim into each stage snapshot's `replay_lineage[].entry_digest` alongside the bundle path as `replay_reference`. Its computation MUST be defined in `bench/replay/i06-schema.yaml` and MUST be recomputable by any consumer from the stored bundle alone. A result whose recorded `entry_digest` does not recompute from the bundle MUST be rejected as `replay_bundle_mutated` naming the entry. | I-05 stage-snapshot `replay_lineage` row ("Opaque interior (E40-F07 owns it)"); ADR-F07-08 |
| REQ-F-004 | The set of tools a scored prelude dispatch MUST NOT reach — the **live-egress set** — MUST be a closed, machine-readable set owned by exactly one file (`bench/replay/live-egress-tools.yaml`), currently `AskUserQuestion`, `WebSearch`, and `WebFetch`. Enforcement MUST be **tool-name-scoped and session-wide**, never a per-call-site list: the harness MUST pass every member of the set as a denial argument on every scored prelude dispatch, and MUST fail the dispatch before it starts if any member is missing from the constructed argument vector. `WebFetch` is in the set even though the bundle's own "Tools Used" section does not name it, because "cannot reach live research" is a reachability property of the session, not a documentation property of the bundle. | feature.md Acceptance boundary 2; research Finding 2 as refined by [X-10 verification](#consumes-x-10--shark-rider-product-design-action-and-progress-record); ADR-F07-02 |
| REQ-F-005 | The **binding** proof that no live interaction occurred MUST be observational, not an assumption about the provider CLI's denial semantics: the scored run's retained transcript MUST contain **zero** tool-use records naming any live-egress-set member, and a single such record MUST stop the scenario with the named terminal outcome `live_interaction_reached`, identifying the tool name and the stage. REQ-F-004's argument-vector check is belt-and-braces; a specification or test that treats the denial flag alone as proof MUST be rejected. | feature.md Acceptance boundary 2; F06 AC-009 precedent (observe, do not infer); ADR-F07-03 |
| REQ-F-006 | A single named component — `bench/scripts/replay-answer.sh` — MUST own request matching, response supply, and consumption recording; no other script, test, or consumer may re-derive those semantics. It MUST accept `--bundle <path> --stage <D0X> --kind <human_question\|research_query> --topic <key>` and MUST supply the response of the **lowest unconsumed `ordinal` for that stage** only when the entry's `request_kind` and `topic_key` both equal the caller-supplied values. It MUST make no network call, MUST append exactly one consumption record per successful supply, and MUST never supply a partial, nearest, or fuzzy match. | F05/F06 "single named owner per piece of arithmetic" discipline (`eval-predicate.sh`, `verify-stage-evidence.sh`); ADR-F07-05 |
| REQ-F-007 | Matching MUST be **ordinal-primary with a topic assertion**, never a match against the model's literal request text, which is not reproducible across runs. Three outcomes are named and distinct: a match supplies the response; a caller `--topic`/`--kind` that disagrees with the entry at the current ordinal MUST fail as `replay_desync` naming both the expected and supplied topic; and no remaining unconsumed entry for the stage MUST fail as `unresolved_gate`. Two runs of the same scenario against the same bundle MUST therefore consume a byte-identical response sequence. | feature.md Acceptance boundary 1; architecture ("supplies a response only when the current action and request match an unused authorized entry"); ADR-F07-04 |
| REQ-F-008 | `unresolved_gate` MUST stop the prelude. The harness MUST NOT invent, paraphrase, infer, or degrade to a default answer, MUST retain the partial replay result, MUST set `publication_eligible: false` with a non-empty `ineligibility_reasons[]`, MUST increment `unresolved_gate_count`, and MUST name the stage, `request_kind`, and `topic_key` that went unanswered. | feature.md Acceptance boundary 3, Scope 6; epic G10, G12; ADR-008 |
| REQ-F-009 | The resolver's consumption ledger MUST be the **single writer** of consumption records, and every artifact's claimed lineage MUST reconcile against it. Two failures are named and MUST remain distinguishable, because they have different causes and different owners downstream: `unresolved_gate` means the bundle lacked or had exhausted an authorized entry (missing input); `unattributed_artifact` means a stage produced an artifact having consumed **zero** of its `required: true` entries, or claimed a consumption absent from the ledger (fabrication or resolver bypass). A validator that collapses the two into one verdict MUST be rejected. | feature.md Acceptance boundary 3 ("never invents an answer"); ADR-F07-05 |
| REQ-F-010 | Every D01-D05 artifact MUST be recorded as `{stage, artifact_type, path, digest, size_bytes, produced_at, revision_index, prompt_digest, input_digests[], consumed_entries[], consumers[]}`. `consumers[]` is an array of typed edges `{consuming_stage, edge_kind, observed_at}` with `edge_kind` ∈ `read` \| `modified` \| `referenced`, recording which **later** D01-D05 stage consumed each earlier artifact. The present-but-empty array means "no consumer observed" (an orphan) and the absent key means "consumption evidence was not collected"; the two MUST be distinguishable at schema level and MUST NOT be coerced in either direction — the same rule I-05 fixed for its own artifact records, adopted verbatim rather than re-decided. | feature.md Scope 5; epic G18; UAT-18; ADR-F06-07 (inherited) |
| REQ-F-011 | Interaction-volume evidence MUST live in a `replayed_interaction_proxies` object carrying a required `measurement_kind: "replayed_interaction_proxy"` discriminator and exactly the closed field set `{authorized_request_count, authorized_response_count, request_bytes_total, response_bytes_total, revision_or_replacement_count, replay_wait_ns, replay_wait_category, unresolved_gate_count}`. `replay_wait_category` MUST be I-05's `replay_or_human_gate_wait`. `replay_wait_ns` is **the harness's own resolution latency**; the harness MUST NOT synthesize, pad, or model a human-latency delay. Any field outside the closed set, any `measurement_kind` other than the discriminator, and any field name or unit expressing human time, stakeholder minutes, or cognitive effort MUST be rejected naming the offending field. | feature.md Scope 5 and 2026-08-13 amendment; epic G18; UAT-18; ADR-F07-06 |
| REQ-F-012 | The replay bundle, and any copy or transformation of it, MUST be absent from both agent-visible roots at every prelude dispatch; only the **matched entry's response** crosses the boundary, one entry at a time. A bundle or bundle-derived file present in the fixture checkout or the scratch Shark project at dispatch MUST be a named `bundle_bulk_disclosure` violation identifying the root and path. The rationale is lineage integrity, not truth-hiding: bulk disclosure would let a session answer out of order or without the resolver, destroying the consumption ledger REQ-F-009 reconciles against. | feature.md Acceptance boundary 1 and 2; ADR-F07-07 |
| REQ-F-013 | When every `stage_matrix.prelude.D01`–`.D05` entry of an I-04 package is `applicable: false`, the harness MUST NOT invoke the Rider action at all and MUST still write a replay result whose `terminal_outcome` is `not_applicable` and whose per-stage records carry `{applicable: false, reason}` copied **verbatim** from the package. An absent replay result for such a scenario MUST be a named failure, never an accepted absence — an explicit non-applicable record is the deliverable. | feature.md Acceptance boundary 4; architecture ("record those stages as non-applicable"); ADR-F07-09 |
| REQ-F-014 | A **read-only** consistency assertion over I-04 MUST hold and MUST be checked before any prelude dispatch: a package with any `prelude.D0X.applicable: true` MUST carry a non-empty `replay_reference` resolving to a bundle whose `scenario_binding.scenario_id` equals the package's `scenario_id`; a package whose prelude stages are all `applicable: false` MUST NOT carry `replay_reference`. Either violation MUST be rejected naming the package and the offending field. No I-04 file is edited to satisfy this (REQ-NF-006). | I-04 `replay_reference` field; feature.md Contracts ("Consumes I-04") |
| REQ-F-015 | The prelude dispatch's working directory MUST be I-05's `roots.scratch_shark_project`. D01-D05 artifacts and `docs/product/progress.md` are planning documents written there — never into the agent-visible fixture checkout and never into the evaluator-only root — and the replay result MUST record the root path and its identity digest it wrote into, so E40-F08 and E40-F09 read the placement rather than inferring it. F07 introduces **no fourth root**. | research Finding 6 / Decision 6; architecture three-root table; ADR-007 |
| REQ-F-016 | F07 MUST invoke the existing Rider product-design action through X-10 unmodified. Interaction routing MUST be supplied by a benchmark-owned, digest-pinned preamble (`bench/replay/preamble.md`) prepended to the dispatch, whose digest is recorded in the replay result as `preamble_digest`. The preamble routes interaction only: it directs the session to resolve elicitation and research through `replay-answer.sh` and MUST NOT restate, extend, reorder, or alter any D01-D05 methodology instruction. Produced artifact filenames MUST conform to the bundle's own Output Standards (`D0X-*.md`), asserted positively against the produced set. | feature.md Scope 1; SKILL.md "Checkpoint boundary"; research Finding 1 / Decision 1; ADR-F07-01 |
| REQ-F-017 | `terminal_outcome` MUST be exactly one value of the closed set `complete` \| `not_applicable` \| `unresolved_gate` \| `replay_desync` \| `live_interaction_reached` \| `unattributed_artifact` \| `bundle_bulk_disclosure` \| `resource_limit` \| `error` \| `cancellation` \| `worker_failure` \| `timeout`. Any value other than `complete` or `not_applicable` MUST retain partial evidence, set `publication_eligible: false`, and carry a non-empty `ineligibility_reasons[]`. A result pairing a stop outcome with `publication_eligible: true` MUST be rejected. **Four of these values are I-06-local diagnostics that I-07's stop vocabulary does not contain** — `replay_desync`, `live_interaction_reached`, `unattributed_artifact`, and `bundle_bulk_disclosure`. F07 MUST NOT widen E40-F08's enum; instead the result MUST also carry `i07_stop_mapping`, a required field on every stop outcome naming the I-07 bucket E40-F08 propagates (`unresolved_gate` for `replay_desync`, `error` for the other three), while the specific F07 value is preserved verbatim in `terminal_outcome` and in `ineligibility_reasons[]`. The eight remaining values map to themselves. | feature.md Acceptance boundary 3; epic G12; ADR-008; architecture Lifecycle run record contract (I-07 stop set); F06 REQ-F-014 (inherited posture) |
| REQ-F-018 | The closed vocabularies I-06 depends on — `stage`, `request_kind`, `artifact_type`, `edge_kind`, proxy field set, `terminal_outcome`, error kind, and the live-egress-set file reference — MUST have exactly one machine-readable owner (`bench/replay/i06-schema.yaml`). The Go contract validator and every bench guard MUST read the vocabulary from that file rather than embedding a private copy. | F06 REQ-F-017 precedent; Rule 7 |
| REQ-F-019 | A schema validator MUST reject a malformed bundle or result with the **failing field named**: an unsupported `schema_version`, a duplicate `ordinal` within a stage, an unknown `stage`/`request_kind`/`edge_kind`/`terminal_outcome`/error kind, an entry whose `entry_digest` does not recompute, a response `{path, digest}` that does not resolve or does not match, a proxy block with an extra field or a non-discriminator `measurement_kind`, an artifact record missing `digest` or `producer`-stage identity, a stop outcome paired with `publication_eligible: true`, a `not_applicable` result missing a per-stage `reason`, and a consumption claim absent from the resolver ledger. | feature.md Workflow handoff; F06 REQ-F-016 precedent; UAT-10 |

### Non-functional requirements

| ID | Requirement | Traces to |
|---|---|---|
| REQ-NF-001 | This feature changes no shark product code. It adds no file under `internal/` or `cmd/`, no schema, no migration, and no service. Its only addition inside the shark Go module is one test-only contract validator, matching E40-F05's and E40-F06's posture. In particular, the disallow-plus-substitute mechanism is **benchmark-adapter code under `bench/`**, not an extension of `internal/runner`'s `DefaultDisallowedTools`, because F07 dispatches host-side over the Rider action and never through `shark run`. | research Finding 3 / Decision 3; ADR-006; F06 REQ-NF-001 precedent |
| REQ-NF-002 | With the I-06 schema, bundle, resolver, guards, and fixtures committed, `make fmt && make lint && make test` at the repository root MUST stay green, and `go list ./...` MUST list no replay fixture package. | E40-F05 / E40-F06 REQ-NF-002 precedent |
| REQ-NF-003 | The I-06 Go contract validator MUST read in-repo artifacts only — `bench/replay/**`, `bench/scenarios/packages/*/evaluator/replay/**`, and `tests/contracts/testdata/e40_i06/**` — and MUST NOT require a populated submodule, because CI's `actions/checkout@v4` does not initialise submodules. `.github/workflows/ci.yml` MUST be unchanged by this feature. | ADR-F06-09 / ADR-F05-07 precedent; `.github/workflows/ci.yml` |
| REQ-NF-004 | Every validator, guard, resolver invocation, and replay-verification MUST run offline once fixtures are present, MUST make **zero provider calls**, and MUST produce byte-identical verdicts across repeated runs at an unchanged bundle and package. The scored prelude dispatch (`run-prelude.sh`) is the single provider-calling path in this feature and is exercised by no test in it; every test in this feature runs against a PATH-stubbed dispatcher that records its invocations. | epic G15; F06 REQ-NF-004 precedent |
| REQ-NF-005 | Replay tooling MUST never touch the live shark database, `.sharkconfig.json`, or the live repository working tree. Because F07 — unlike F06 — needs a **live scratch Shark project**, the sanctioned mechanism for standing one up is `scripts/shark-scratch-env.sh`; no F07 script, test, or documented procedure may invoke a shark project-initialisation or cloud-initialisation command directly. | E40-F05 / E40-F06 REQ-NF-005; repo config-guardrail hook |
| REQ-NF-006 | The following surfaces MUST be byte-unchanged by this feature. **(a) The wrapped methodology** — `skills/shark-rider/verbs/product-design.md` and every file under `internal/sharkdata/default_data/skills/product-design/**`. This is the mechanical form of the feature's first scope rule ("wrap; do not copy or fork"), and it is the strongest single constraint in this spec. **(b) Phase 1** — `bench/corpus/corpus.yaml`, `collect-run.sh`, `verify-clean-checkout.sh`, `canary-runsurface.sh`, `replay-manifest.sh`, and `tests/contracts/e40_i0{1,2,3}_*_test.go`. **(c) I-04** — `bench/scenarios/**`, `bench/adapters/**`, `admit-scenario.sh`, `eval-predicate.sh`, `checkout-scenario-fixture.sh`, `tests/contracts/e40_i04_scenario_contract_test.go`, **with exactly one carve-out**: `bench/scenarios/packages/py-feature-recurring-tasks/evaluator/replay/reference-bundle.json`, whose interior F05 explicitly deferred to E40-F07/I-06 in the file's own header and in `package.yaml`'s comment. Its containing `package.yaml` is **not** changed — the pointer stays as F05 wrote it. **(d) I-05** — `bench/evidence/**`, `verify-evidence-roots.sh`, `verify-stage-evidence.sh`, `replay-stage-evidence.sh`, `canary-usagemapping.sh`, `tests/contracts/e40_i05_stage_evidence_contract_test.go`. F07 consumes I-05 by path and digest and types nothing in it. | feature.md Scope 1 and Out of scope; interaction map I-04 row ("E40-F07 … treat it as read-only"); F06 spec Out of scope ("Defining the interior of … the I-06 replay bundle (E40-F07)") |
| REQ-NF-007 | Resolver latency MUST be bounded and MUST be attributed to I-05's `replay_or_human_gate_wait` interval category, never to `provider_active`. A local file read is the intended cost; no F07 component may introduce a deliberate delay. | REQ-F-011; epic G16; UAT-16; F06 REQ-F-005 |

### Acceptance criteria

| ID | Criterion |
|---|---|
| AC-001 | TC-052 asserts, over every fixture under `tests/contracts/testdata/e40_i06/valid/`, that both I-06 documents carry a supported `schema_version` and the REQ-F-001/002/010/011/017 field inventory, well-typed; that every closed vocabulary value used resolves against `bench/replay/i06-schema.yaml` (REQ-F-018); and that the bundle and the result are recognised as **distinct document kinds** — a result supplied where a bundle is expected, or the reverse, is rejected naming the expected kind rather than silently half-validating. |
| AC-002 | TC-052 asserts REQ-F-003: `entry_digest` recomputes from the stored bundle for every entry; a one-byte edit to any entry field yields `replay_bundle_mutated` naming the entry; and a result whose `replay_lineage`-facing `entry_digest` values are not a subset of the bundle's recomputed set is rejected — so the I-05 join key is proven, not asserted. |
| AC-003 | `tc053_live_egress_denial_test.sh` proves both halves of REQ-F-004/REQ-F-005 independently. (a) **Structural:** the argument vector `run-prelude.sh` constructs contains a denial argument for every member of `bench/replay/live-egress-tools.yaml`, captured from a PATH-stubbed dispatcher that records its argv; removing a member from the file changes the argv with no edit to the script, and a stubbed argv missing a member fails before dispatch. (b) **Observational and binding:** a retained transcript fixture containing one `WebSearch` tool-use record yields `live_interaction_reached` naming the tool and stage, and a clean transcript passes — proving the gate holds independently of any assumption about the provider CLI's denial semantics. |
| AC-004 | `tc054_replay_resolver_test.sh` proves REQ-F-006/REQ-F-007's three named outcomes over one bundle: a matching `--stage`/`--kind`/`--topic` supplies the lowest unconsumed ordinal's response and records exactly one consumption; the same call repeated after exhaustion of that stage's entries fails as `unresolved_gate` naming stage, kind, and topic; and a call whose `--topic` disagrees with the entry at the current ordinal fails as `replay_desync` naming both expected and supplied topic. A fourth case asserts the resolver never supplies a nearest or partial match. |
| AC-005 | `tc054` additionally proves REQ-F-007's reproducibility: two consecutive resolver-driven passes over the same bundle and the same recorded call sequence produce byte-identical supplied responses and byte-identical consumption ledgers. |
| AC-006 | `tc055_lineage_reconciliation_test.sh` proves REQ-F-009's two distinct verdicts: a result whose artifact claims a consumption absent from the resolver ledger is rejected as `unattributed_artifact` naming the entry; a stage that produced an artifact having consumed zero `required: true` entries is rejected as `unattributed_artifact` naming the stage; and a run that stopped for a missing entry is reported as `unresolved_gate` — never as `unattributed_artifact`, and never the reverse. |
| AC-007 | TC-052 asserts REQ-F-010's artifact records and the empty-versus-absent distinction: one artifact carrying `consumers: []` and one with the `consumers` key absent yield two distinct verdicts, `orphan` and `consumption_evidence_missing`, with no coercion in either direction; and a downstream edge from a later D0X stage to an earlier stage's artifact is recorded with its `edge_kind`, so UAT-18's reused-versus-orphan distinction is available to E40-F10. |
| AC-008 | TC-052 asserts REQ-F-011: a proxy block missing `measurement_kind`, carrying a value other than `replayed_interaction_proxy`, carrying any field outside the closed set, or carrying a field whose name or declared unit expresses human time (`human_minutes`, `stakeholder_minutes`, `cognitive_effort`, or any duration attributed to a person) is rejected naming the field. A companion case asserts `replay_wait_category` is `replay_or_human_gate_wait` and that `replay_wait_ns` reflects observed resolver latency only — a fixture whose `replay_wait_ns` exceeds a declared plausibility ceiling for a local file read is rejected as a synthesized delay. |
| AC-009 | `tc056_bundle_disclosure_test.sh` proves REQ-F-012: planting the replay bundle, or a copy of it under a different name with the same content digest, in (a) the fixture checkout and (b) the scratch Shark project each fails as `bundle_bulk_disclosure` naming the root and path — two independently failing cases, so a guard walking only one root cannot pass. A third case proves clean roots pass while the resolver still supplies one entry, demonstrating that entry-at-a-time disclosure is the permitted path. A fourth proves the check aborts **before** any dispatch: on the failing path a PATH-stubbed dispatcher records zero invocations. |
| AC-010 | `tc057_non_applicable_record_test.sh` proves REQ-F-013 for each of the three non-feature seed packages: the Rider action is never invoked (PATH-stubbed dispatcher records zero invocations), a replay result is written with `terminal_outcome: not_applicable`, and each of D01-D05 carries `applicable: false` with the `reason` string copied verbatim from that package. A fourth case proves that producing **no** result for a non-applicable scenario is itself a named failure. |
| AC-011 | `tc057` additionally proves REQ-F-014's read-only consistency assertion: the feature package passes; a scratch copy with `replay_reference` removed is rejected naming the field; a scratch copy of a bug package with a `replay_reference` added is rejected naming the field; and a bundle whose `scenario_binding.scenario_id` disagrees with the package is rejected naming both ids. No file under `bench/scenarios/` is modified by the test. |
| AC-012 | `tc058_prelude_placement_test.sh` proves REQ-F-015/REQ-F-016 against a scratch Shark project created by `scripts/shark-scratch-env.sh` with a PATH-stubbed dispatcher: the dispatch working directory is the declared `roots.scratch_shark_project`; the result records that root's path and identity digest; no D01-D05 artifact or `docs/product/progress.md` write lands in the fixture checkout or the evaluator-only root; the constructed prompt contains the `bench/replay/preamble.md` content and the result's `preamble_digest` equals that file's `sha256`; and the produced artifact filenames match the bundle's own `D0X-*.md` Output Standard. |
| AC-013 | TC-052 asserts REQ-F-017: for each stop outcome in the closed set, the result retains its partial stage and artifact records, sets `publication_eligible: false`, and carries a non-empty `ineligibility_reasons[]`; a result pairing any stop outcome with `publication_eligible: true` is rejected; and an unknown `terminal_outcome` value is rejected naming it. A companion case asserts the I-07 seam: every stop outcome carries `i07_stop_mapping`, each of the four I-06-local diagnostics maps to the bucket REQ-F-017 fixes, a stop outcome with `i07_stop_mapping` absent is rejected naming the field, and a mapping naming a value outside I-07's own stop vocabulary is rejected — so E40-F08 inherits a decided propagation rule rather than an ambiguity. |
| AC-014 | TC-052 asserts each malformed case enumerated in REQ-F-019 exits non-zero with a message naming the failing field, verified by table-driven fixtures under `tests/contracts/testdata/e40_i06/invalid/`. |
| AC-015 | TC-052 asserts REQ-F-018's single-owner rule: every closed vocabulary value the validator accepts is present in `bench/replay/i06-schema.yaml`, and a value added to the schema file but not to a fixture (or the reverse) surfaces as a named disagreement rather than silently diverging. |
| AC-016 | `tc059_replay_offline_determinism_test.sh` runs every F07 guard, validator, and resolver invocation with the network disabled and asserts byte-identical verdicts across two consecutive runs, and that a PATH-stubbed provider records **zero** invocations across the whole suite — so no test in this feature depends on the single provider-calling path (REQ-NF-004). |
| AC-017 | `make fmt && make lint && make test` at the repository root is green with the I-06 schema, bundle, resolver, guards, and fixtures committed; `go list ./...` lists no replay fixture package; TC-052 passes in CI without `git submodule update --init`; and `.github/workflows/ci.yml` is unchanged (REQ-NF-003). |
| AC-018 | A diff review proves REQ-NF-006 group by group: `skills/shark-rider/verbs/product-design.md` and every file under `internal/sharkdata/default_data/skills/product-design/**` are byte-unchanged; every Phase 1, I-04, and I-05 file listed is byte-unchanged; `package.yaml` is byte-unchanged while `evaluator/replay/reference-bundle.json` is the single carve-out this feature writes; and no file under `internal/` or `cmd/` is touched. |
| AC-019 | A grep of every generic F07 script (`run-prelude.sh`, `replay-answer.sh`, `verify-replay-result.sh`, `verify-replay-isolation.sh`) for `python`, `pytest`, `pip`, `go test`, `golangci-lint`, and `go build` returns no hits, proving F07 adds no language-aware generic component — the same mechanical proof F05's AC-012 and F06's AC-019 established. |

### Out of scope for this feature

- **D06-D14 artifacts and any new product-design methodology.** F07 wraps
  D01-D05 only, and REQ-NF-006(a) freezes the bundle and the Rider verb so a
  methodology change cannot land here by accident.
- **The keyed Shark entity lifecycle after the prelude** — dispatch, claim,
  heartbeat, transition, release, and generated-task execution (E40-F08 / I-07 /
  X-11).
- **Shark's durable Question lifecycle (X-13).** D01-D05's human elicitation is
  an in-session tool call inside one interactive Rider action, not a durable
  Question entity keyed to a Shark entity. X-13 is assigned to E40-F08 in the
  cross-epic map and is the surface for the *post-prelude* keyed lifecycle.
  Routing prelude responses through it would be scope creep into F08's
  territory (research Finding 5 / Decision 5).
- **Estimating stakeholder cognitive effort or elapsed human work.** REQ-F-011
  makes this structurally unavailable rather than merely discouraged.
- **Scoring the produced D01-D05 artifacts, calibrating a judge, or deciding
  aggregate eligibility** (E40-F09 / I-08). F07 records `publication_eligible:
  false` for a stop outcome; it does not adjudicate a *passing* run.
- **Report layouts and the derived artifact-reuse metrics** (E40-F10). F07
  supplies typed producer/consumer edges and labelled proxies; it does not
  decide what they are worth. The "no report labels these as human minutes"
  half of UAT-18 is enforced here at the data level and by F10 at the
  presentation level.
- **Defining or modifying the interior of I-04 (E40-F05) or I-05 (E40-F06).**
  F07 references both by path and digest and types neither.
- **Any change to `internal/runner`'s `DefaultDisallowedTools` or dispatcher
  argument construction** (REQ-NF-001). If a generic Shark need for
  disallow-plus-substitute is ever identified, it is triaged under E22 per the
  epic's "triage generic changes under their owning epic" rule.
- Epic-family scenarios and the deferred delivery tail, per epic §"Out of
  scope".

---

## Architecture

### Component changes

| File | Change |
|---|---|
| `bench/replay/i06-schema.yaml` | New. The single machine-readable owner (REQ-F-018) of I-06's `schema_version`, both document shapes, the `entry_digest` computation, and every closed vocabulary: `stage`, `request_kind`, `artifact_type`, `edge_kind`, the proxy field set, `terminal_outcome`, and error kind. Plays for I-06 the role `bench/evidence/i05-schema.yaml` plays for I-05, without inheriting any of its fields. |
| `bench/replay/live-egress-tools.yaml` | New. The closed live-egress set (REQ-F-004): `AskUserQuestion`, `WebSearch`, `WebFetch`, each with the reason it is denied. Sourced at call time by `run-prelude.sh` and by the REQ-F-005 transcript scan, so adding a tool changes both enforcement and detection with no script edit. |
| `bench/replay/preamble.md` | New. The digest-pinned interaction-routing preamble (REQ-F-016). Directs the dispatched session to resolve every elicitation and research request through `replay-answer.sh` with an explicit `--stage`/`--kind`/`--topic`, and to stop on a non-zero resolver exit. Contains no methodology. |
| `bench/scenarios/packages/py-feature-recurring-tasks/evaluator/replay/reference-bundle.json` | **Modified — the single I-04 carve-out in REQ-NF-006.** F05 committed this file as an explicitly untyped placeholder whose header defers its interior to E40-F07/I-06; F07 replaces that interior with a conforming replay bundle for the recurring-tasks feature scenario. `package.yaml`'s `replay_reference` pointer is unchanged. |
| `bench/scripts/replay-answer.sh` | New. The resolver and **single named owner** of request matching, response supply, and consumption recording (REQ-F-006). Ordinal-primary matching with a topic assertion; three named outcomes (`supplied`, `replay_desync`, `unresolved_gate`); appends exactly one consumption record per supply; makes no network call. Follows F06's bench-script conventions: self-contained `python3` heredoc, `ScriptError` versus violation exit-code split, and `resolve_within`-style containment for every bundle-relative response path. |
| `bench/scripts/run-prelude.sh` | New. The host-side prelude adapter. Resolves the I-04 package, enforces REQ-F-014's consistency assertion before dispatch, constructs the argument vector with every live-egress denial (REQ-F-004), prepends `preamble.md`, dispatches the unmodified Rider product-design action with the scratch Shark project as its working directory (REQ-F-015), and writes the replay result. **The only provider-calling path in this feature** (REQ-NF-004). REQ-F-012's disclosure check is `verify-replay-isolation.sh`'s three-arg bulk-disclosure mode (see that row below); it is invoked by the **caller's** dispatch loop (E40-F08's future work) against the live in-flight roots, immediately before every scored dispatch — `run-prelude.sh` has no internal call to it and no `--fixture-checkout`/root-path flags of its own, matching `bench/README.md`'s "Bundle-disclosure guard" and "Tier 2 guard invocation sequence" sections. For an all-non-applicable package it writes the `not_applicable` record and dispatches nothing (REQ-F-013). **Dispatch shape** (fixed here so every PATH-stub acceptance criterion is implementable without the task author inventing it): a subprocess invocation of a named provider CLI binary resolved from `PATH`, in non-interactive print mode, with one `--disallowedTools <tool>` argument per live-egress-set member and the prompt formed as `preamble.md` followed by the Rider product-design action invocation. Resolving the binary from `PATH` is what makes AC-003(a), AC-012, and AC-016's argv-recording and zero-invocation stubs possible. |
| `bench/scripts/verify-replay-result.sh` | New. Validates one replay result's lineage reconciliation against the resolver's own consumption ledger (REQ-F-009): two named, distinguishable verdicts (`unattributed_artifact` for a fabricated or resolver-bypassed claim, or for a stage that consumed zero `required: true` entries; a genuine `unresolved_gate` stage is reported by passing the result's own `terminal_outcome` straight through, never manufactured as `unattributed_artifact`). This is the **single named owner of REQ-F-009's reconciliation arithmetic only** — it does not implement the rest of Document B's field inventory. Field inventory and vocabulary (REQ-F-001/002), `entry_digest` recomputation against the bundle (REQ-F-003), artifact records with the empty-versus-absent distinction (REQ-F-010), proxy-block closure and the human-time prohibition (REQ-F-011), non-applicable completeness (REQ-F-013), and stop-outcome eligibility plus the `i07_stop_mapping` requirement (REQ-F-017) are TC-052's job (the Go contract validator) over the static fixtures under `tests/contracts/testdata/e40_i06/{valid,invalid}/` — the same schema-validator/execution-guard split ADR-F07-10 fixes, matching AC-001/002/007/008/013/014/015 and `bench/README.md`'s own "Lineage reconciliation" and "Tier 2 guard invocation sequence" sections. F08 and F10 invoke this script for REQ-F-009 and TC-052 (`go test ./tests/contracts/...`) for the rest, rather than re-deriving either. `revision_or_replacement_count` is not derived anywhere in this feature: `run-prelude.sh`'s `write_complete_result` emits it as an honest zero placeholder (REQ-F-011 proxy arithmetic left to later wiring); TC-052 validates only that the closed proxy field set and its declared units are well-formed (AC-008), never that the count was computed correctly. |
| `bench/scripts/verify-replay-isolation.sh` | New. The REQ-F-012 bulk-disclosure guard and the REQ-F-005 transcript scan. Proves the bundle and every content-digest-identical copy are absent from both agent-visible roots, and that the retained transcript carries zero tool-use records for any live-egress-set member. A **new** guard, not an edit to F06's `verify-evidence-roots.sh` — the replay bundle is not `evaluator_only` material, so it falls outside that guard's contract by design (ADR-F07-07), and F06's script is frozen by REQ-NF-006(d). |
| `bench/scripts/testdata/replay/` | New. Bundle, result, transcript, and root fixtures for the bench test cases: clean and planted roots, each resolver outcome, each proxy-violation case, each stop outcome, each non-applicable package copy. |
| `bench/scripts/tests/tc053_live_egress_denial_test.sh` | New. AC-003. |
| `bench/scripts/tests/tc054_replay_resolver_test.sh` | New. AC-004, AC-005. |
| `bench/scripts/tests/tc055_lineage_reconciliation_test.sh` | New. AC-006. |
| `bench/scripts/tests/tc056_bundle_disclosure_test.sh` | New. AC-009. |
| `bench/scripts/tests/tc057_non_applicable_record_test.sh` | New. AC-010, AC-011. |
| `bench/scripts/tests/tc058_prelude_placement_test.sh` | New. AC-012. |
| `bench/scripts/tests/tc059_replay_offline_determinism_test.sh` | New. AC-016, AC-019. |
| `bench/scripts/tests/run-all.sh` | Modified. Registers TC-053 through TC-059. The only existing bench file this feature edits. |
| `bench/README.md` | Modified. Adds an "I-06 product-design replay contract (E40-F07)" section — the two-document split, bundle and result field references, the live-egress set and its two-half proof, resolver invocation and matching rule, proxy-field closure, artifact placement, and the guard invocation sequence — so E40-F08 and E40-F10 read the shape instead of re-deriving it. |
| `tests/contracts/e40_i06_product_design_replay_contract_test.go` | New, **the only Go file this feature adds** (REQ-NF-001). `package contracts`, TC-052, repository-root-relative artifact reading, in-repo artifacts only (REQ-NF-003), following `tests/contracts/e40_i05_stage_evidence_contract_test.go`. |
| `tests/contracts/testdata/e40_i06/{valid,invalid}/` | New. Table-driven bundle and result fixtures for AC-001, AC-002, AC-007, AC-008, AC-013, AC-014, AC-015. |

### Data model changes

None. No shark table, column, migration, or `CurrentSchemaVersion` bump. I-06 is
file-backed under `bench/`, consistent with ADR-002 and the architecture's
"E40 adds no Shark database table."

### API / interface contracts

#### Document A — replay bundle (I-06 input)

The committed, versioned file `package.yaml`'s `replay_reference` points at.

| Field | Type | Contract |
|---|---|---|
| `schema_version` | string | The I-06 version `i06-schema.yaml` declares and TC-052 supports. |
| `bundle_version` | string | Bumped whenever any entry changes; recorded in the result so a rerun against a different bundle is visible rather than silent. |
| `scenario_binding` | object | `{scenario_id, scenario_version}`; `scenario_id` MUST equal the owning package's (REQ-F-014). |
| `entries` | array | Ordered authorized entries; see below. |

Entry fields:

| Field | Type | Contract |
|---|---|---|
| `entry_id` | string | Unique within the bundle; the id the result's lineage references. |
| `stage` | enum | `D01`–`D05`. |
| `ordinal` | integer | Unique within its stage. Matching consumes the lowest unconsumed ordinal for the stage (REQ-F-007). |
| `request_kind` | enum | `human_question` (D01, D02, D05 elicitation) or `research_query` (D03, D04 research). |
| `topic_key` | string | The assertion the caller must supply. Not a lookup key — a mis-pairing guard (ADR-F07-04). |
| `required` | bool | `true` entries must be consumed for the stage's artifact to be attributable (REQ-F-009). |
| `response` | string or object | Inline text, or `{path, digest}` resolved relative to the bundle file and contained within its directory. |
| `response_digest` | string | `sha256` of the resolved response bytes. |
| `entry_digest` | string | REQ-F-003 content address; the I-05 `replay_lineage` join key. |

#### Document B — replay result (I-06 output, consumed by E40-F08)

| Field | Type | Contract |
|---|---|---|
| `schema_version` | string | As above. |
| `scenario` | object | `{scenario_id, scenario_version, entity_family}`, copied verbatim from the I-04 package. |
| `run_id` | string | Opaque to F07; E40-F08 assigns it. |
| `replay_bundle` | object | `{replay_reference, bundle_path, bundle_digest, bundle_version}` — the exact input this run consumed. |
| `preamble_digest` | string | `sha256` of `bench/replay/preamble.md` as dispatched (REQ-F-016). |
| `artifact_root` | object | `{path, identity_digest, root_kind: "scratch_shark_project"}` (REQ-F-015). |
| `stages` | array | One record per `D01`–`D05`: `{stage, applicable, reason?, artifacts[], consumed_entries[]}`. `reason` is required and copied verbatim when `applicable` is `false` (REQ-F-013). |
| `stages[].artifacts` | array | REQ-F-010 records. `consumers: []` ≠ absent `consumers`. |
| `stages[].consumed_entries` | array | `{entry_id, entry_digest, request_kind, topic_key, supplied_at, request_bytes, response_bytes}`, written **only** by the resolver ledger (REQ-F-009). |
| `replayed_interaction_proxies` | object | REQ-F-011 closed field set with its `measurement_kind` discriminator. |
| `artifact_consumption_edges` | array | `{producer_stage, artifact_path, consuming_stage, edge_kind}` — the flattened cross-stage view E40-F10 reads for reuse and orphan counts (REQ-F-010). |
| `terminal_outcome` | enum | REQ-F-017 closed set. |
| `i07_stop_mapping` | string, required on every stop outcome | The I-07 bucket E40-F08 propagates for an I-06-local diagnostic: `unresolved_gate` for `replay_desync`; `error` for `live_interaction_reached`, `unattributed_artifact`, and `bundle_bulk_disclosure`; otherwise the same value as `terminal_outcome` (REQ-F-017). Keeps F07 from widening F08's enum while preserving the specific cause. |
| `publication_eligible` | bool | `false` for every outcome other than `complete` and `not_applicable`. |
| `ineligibility_reasons` | array of string | Non-empty whenever `publication_eligible` is `false`. |

#### Live-egress set and its two-half proof

`bench/replay/live-egress-tools.yaml` is the one owner of the denied set.
Enforcement and detection both read it:

| Half | Mechanism | Binding? |
|---|---|---|
| Structural denial | `run-prelude.sh` emits one denial argument per set member and refuses to dispatch if any is missing from the constructed argv | No — belt-and-braces |
| Observational detection | `verify-replay-isolation.sh` scans the retained transcript for tool-use records naming any set member; one hit is `live_interaction_reached` | **Yes** — this is the contract (REQ-F-005, ADR-F07-03) |

#### Resolver interface

```
replay-answer.sh --bundle <path> --stage <D01|…|D05> \
                 --kind <human_question|research_query> --topic <key>
```

Exit `0` prints the response bytes on stdout and appends one consumption record.
Exit non-zero prints a named outcome on stderr: `replay_desync` (topic or kind
disagrees with the entry at the current ordinal, naming both), `unresolved_gate`
(no unconsumed entry remains for the stage, naming stage, kind, and topic), or a
`ScriptError` for a malformed bundle or unresolvable response path. No other
outcome exists; there is no nearest-match, default, or degraded path.

### Key technical decisions

**ADR-F07-01 — The replay adapter sits at the Rider-adapter layer, and "wrap,
don't fork" is enforced by a byte-freeze rather than by intent.** The bundled
skill's own "Checkpoint boundary" section states that it "does not own ordering,
resumability, progress records, CLI commands, or retrieval of another skill,"
and assigns that ownership to `skills/shark-rider/verbs/product-design.md`
(research Finding 1). Interception belongs at that same adapter layer;
intercepting inside the bundle's workflow files would duplicate an ownership
split the codebase drew deliberately. Because "do not fork" is easy to state and
easy to violate incrementally, REQ-NF-006(a) freezes both the verb and the whole
bundle tree at the byte level and AC-018 checks it in a diff review. The routing
the session needs therefore arrives as a benchmark-owned, digest-pinned
preamble prepended at dispatch — outside the frozen tree, recorded in the result
so prompt identity is pinned for E40-F09. Rejected alternatives: (a) editing the
bundle's `Tools Used` section to name the resolver — a fork of the methodology
by any reading, and it would leak benchmark concerns into every human's
product-design run; (b) a benchmark-only copy of the bundle — the exact
duplication the feature file's first scope rule forbids.

**ADR-F07-02 — Enforcement is tool-name-scoped and session-wide, not
site-enumerated, and the set includes `WebFetch`.** Direct inspection refines
the research report's "five call sites" claim: `AskUserQuestion` is mandated in
`SKILL.md:110` ("Elicit, don't invent … Use `AskUserQuestion`") and `SKILL.md:133`
("All elicitation, approvals, and human decisions"), and `WebSearch` in
`SKILL.md:136` ("Market research (D03), technical research (D04)"). Of the five
stage-level interaction points, only two carry the literal tool name in a
workflow file — `d01-vision.md:11` and `d03-market-research.md:22`. D02's
elicitation sequence, D04's research prompts, and D05's mode choice and
interview synthesis are prose that the SKILL-level mandate turns into tool calls
at run time. A guard enumerating grep sites would therefore already be
incomplete today, and would silently rot the first time a workflow file adds a
sixth elicitation point. Enforcing on the **tool name**, session-wide, is
complete by construction over the same behaviour. `WebFetch` is added for the
same reason the enforcement surface is tool-scoped: "a scored run cannot reach
live research" is a property of what the session can reach, not of what the
bundle documents, and `WebSearch` denial alone leaves a live URL fetch open. The
five stage points remain the correct description of *where* the surface is
exercised — REQ-F-013's stage matrix and the bundle's per-stage entries are
organised by them — but they are not the enforcement unit. Rejected alternative:
a generic "any tool call" policy, which would block `Read` and `Write` and make
the prelude unrunnable.

**ADR-F07-03 — Denial is belt-and-braces; the binding gate is observational.**
This spec makes no claim about whether a given provider CLI's disallow mechanism
covers a builtin elicitation tool, or whether such a tool is even offered in a
non-interactive invocation. That is a platform property that can change under
us, and a contract resting on it would be unfalsifiable from inside this repo.
REQ-F-005 therefore makes the acceptance gate an assertion about the **retained
transcript**: zero tool-use records naming any live-egress-set member, with
`live_interaction_reached` as the named failure. This holds regardless of the
platform's denial semantics, and it is the same move F06's AC-009 made when it
required a PATH-stubbed dispatcher to record **zero** invocations rather than
inferring "before provider spend" from a non-zero exit status. The denial
arguments remain required (REQ-F-004) because a mechanism that works is cheaper
than a detection that fires; they are simply not the proof. Rejected
alternative: specifying "the CLI blocks these tools" as a requirement, which
would be a requirement on someone else's software that this feature cannot
verify or fix.

**ADR-F07-04 — Matching is ordinal-primary within a stage, with `topic_key` as
an assertion rather than a lookup key.** The architecture requires that a
response be supplied "only when the current action and request match an unused
authorized entry," which needs a match key that is both deterministic across
runs and resistant to mis-pairing. The model's literal question text satisfies
neither: it is regenerated each run and would make the same bundle supply
different sequences, breaking the acceptance boundary's "two runs consume the
same response sequence" outright. Stage plus ordinal is fully deterministic.
Ordinal alone, however, would silently mis-pair if the session asked about
success metrics where the bundle expected a baseline, so each entry declares a
`topic_key` the caller must restate; disagreement is `replay_desync`, a named
stop, not a supplied answer. This is deterministic sequencing with a semantic
tripwire, rather than semantic lookup with nondeterministic keys. Rejected
alternatives: (a) fuzzy or embedding-based matching on question text —
nondeterministic by construction and it would let a near-miss be answered
confidently, the fabrication mode this feature exists to prevent; (b) requiring
the bundle to enumerate exact question strings — brittle against any bundle
wording change, and it would couple the frozen methodology to the benchmark.

**ADR-F07-05 — Missing input and unattributed output are two named failures, and
the resolver ledger is their single arbiter.** `unresolved_gate` (the bundle
lacked or exhausted an authorized entry) and `unattributed_artifact` (a stage
produced an artifact having consumed none of its required entries) have
different causes, different remedies, and different downstream owners: the first
means the corpus is incomplete and the curator must add an entry; the second
means the session bypassed the resolver and the run's lineage cannot be trusted.
Collapsing them would leave E40-F10 unable to distinguish "we did not have the
answer" from "the answer was invented." Making the resolver's ledger the single
writer of consumption records — with the artifact's claimed lineage reconciled
*against* it rather than trusted alongside it — is what makes the second failure
detectable at all: routing through a directed preamble is a prompt-compliance
mechanism, and prompt compliance is exactly the thing a benchmark must not
assume. This applies F05's and F06's "single named owner per piece of
arithmetic" discipline to consumption. Rejected alternative: trusting the
session's self-reported lineage, which is the "worker self-report as oracle"
posture the epic's non-functional evidence section forbids outright.

**ADR-F07-06 — Proxies carry a discriminator and a closed field set, and no
human latency is ever synthesized.** The feature file and epic G18 both require
interaction volume to be reproducible *and* unmistakable for human effort. A
prose disclaimer would not survive being read by F10's report generator, so the
prohibition is structural: a required `measurement_kind:
"replayed_interaction_proxy"` discriminator, a closed field set that contains no
human-attributed duration, and a validator that rejects any field name or unit
expressing stakeholder minutes or cognitive effort. The one time-shaped field,
`replay_wait_ns`, is the harness's own resolution latency for a local file read
and is classified into I-05's existing `replay_or_human_gate_wait` category
(REQ-NF-007); AC-008 rejects a value large enough to look like a modelled human
delay. Rejected alternative: recording a simulated think-time per entry to make
lifecycle wall time "realistic" — that would manufacture precisely the human
minutes the amendment forbids, and would corrupt I-05's ledger reconciliation
with fabricated intervals.

**ADR-F07-07 — The replay bundle is authorized input, not evaluator-only truth,
and bulk disclosure is prohibited for lineage integrity rather than secrecy.**
`replay_reference` is a **top-level** `package.yaml` field, deliberately outside
`evaluator_only.{reference_solution, oracle_tests, answer_keys}`, so F06's
REQ-F-010/011 dispatch-boundary guard does not cover it — and F07 does **not**
move it there. Adding it to `evaluator_only` would be an I-04 edit (forbidden by
REQ-NF-006(c)) and would make F06's guard fail the very dispatch F07 needs,
since the harness must read the bundle to serve it. The correct property is
narrower and is stated honestly: the session will legitimately see every
consumed response by the end of D05, so this is not truth-hiding; what must not
happen is bulk disclosure, which would let the session answer out of order or
without the resolver and destroy the ledger REQ-F-009 reconciles against.
REQ-F-012 therefore requires entry-at-a-time disclosure and names bulk presence
`bundle_bulk_disclosure`, enforced by a new F07-owned guard rather than an edit
to F06's frozen script — the same "new sibling, not a generalization" call F06
itself made about `verify-clean-checkout.sh`.

**ADR-F07-08 — I-06 is two documents, and `entry_digest` is the one field two
contracts join on.** The interaction map names I-06 the "product-design replay
result," while I-04's pointer is called `replay_reference` and points at an
*input*. Leaving that ambiguous would send E40-F08 looking for the wrong file,
so REQ-F-001 names both roles explicitly and TC-052 rejects one supplied where
the other is expected. The join to I-05 is equally load-bearing and equally easy
to leave implicit: I-05's stage snapshot already reserves `replay_lineage[]` as
`{replay_reference, entry_digest}` with an "opaque interior (E40-F07 owns it)"
note, so `entry_digest` is the single field crossing the two contracts. REQ-F-003
therefore fixes its computation in `i06-schema.yaml` and makes it recomputable
from the stored bundle alone, so E40-F08 populates I-05 from a defined rule and
E40-F09 can verify the join without either feature re-deriving it.

**ADR-F07-09 — A non-applicable family produces an explicit record, not an
absence.** All three non-feature seed packages already declare
`prelude.D01`–`.D05` as `applicable: false` with a per-stage `reason`. The
tempting implementation is to skip the prelude and write nothing, but an absent
file is indistinguishable from a harness that crashed before writing one, and
the feature's acceptance boundary asks for "explicit non-applicable stage
records." REQ-F-013 therefore requires a `terminal_outcome: not_applicable`
result carrying each stage's verbatim `reason`, and AC-010 makes a missing
result its own failure. Copying the reason verbatim, rather than regenerating
it, keeps I-04 the single source of the applicability rationale.

**ADR-F07-10 — Schema validation and execution guards are separate owners.**
`tests/contracts/e40_i06_product_design_replay_contract_test.go` (TC-052)
validates structure from in-repo artifacts only and never requires a populated
submodule, because CI's `actions/checkout@v4` does not initialise submodules.
Execution-based resolver, isolation, placement, and determinism guards live in
`bench/scripts/` and do require real roots and a scratch Shark project. This
mirrors ADR-F06-09 and ADR-F05-07 rather than inventing a third convention.

### Integration with existing code

Nothing under `internal/` or `cmd/` is called, imported, or extended
(REQ-NF-001). The integration surfaces are read-only artifacts, executables, and
conventions:

- **Wrapped Rider action (X-10)** — `run-prelude.sh` invokes the existing
  `/shark-rider project product-design` action, which itself retrieves the
  bundle through `shark skill get product-design`. F07 neither retrieves the
  bundle itself nor bypasses the adapter, preserving the ownership split
  `SKILL.md`'s "Checkpoint boundary" and the verb's "Never let the bundle invoke
  the CLI or retrieve another skill" line both state. `docs/product/progress.md`
  remains the wrapped action's own derived record.
- **I-04 read-only consumption** — the guards read
  `bench/scenarios/packages/*/package.yaml` for `entity_family`,
  `stage_matrix.prelude`, and `replay_reference`. The only I-04 file this
  feature writes is the `reference-bundle.json` carve-out named in
  REQ-NF-006(c).
- **I-05 by reference only** — the result names its `artifact_root` as the
  scratch Shark project root and supplies the `entry_digest` values E40-F08
  writes into `replay_lineage[]`. F07 reads I-05's root vocabulary and interval
  category names from `bench/evidence/i05-schema.yaml` at call time (REQ-F-018's
  single-owner discipline applied across the contract boundary) and edits
  nothing under `bench/evidence/`.
- **Structural-guard pattern, not code** — `internal/runner/dispatcher.go`'s
  `DefaultDisallowedTools` supplies the posture (a checkable list, enforced
  before the worker acts, failing loud) and the `--disallowedTools` naming
  convention for consistency; none of `internal/runner`'s code is imported,
  copied, or changed, and the substitution half has no precedent there
  (ADR-F07-03, research Finding 3).
- **Contract-test convention** — the new Go file joins `package contracts` and
  follows the repository-root-relative artifact-reading style of
  `tests/contracts/e40_i05_stage_evidence_contract_test.go`.
- **Bench script and test-runner conventions** — the new scripts follow F06's
  self-contained `python3` heredoc form, its `ScriptError`-versus-violation exit
  split, and its `resolve_within`-style containment checks; the new tests follow
  `bench/scripts/tests/tcNNN_*.sh` naming and are registered in `run-all.sh`,
  the only existing bench file this feature edits.
- **Scratch-environment discipline** — F07 is the first E40 lifecycle-v2 feature
  that needs a live scratch Shark project. It is stood up only through
  `scripts/shark-scratch-env.sh`; no F07 script, test, or documented procedure
  invokes a shark project-initialisation or cloud-initialisation command
  (REQ-NF-005).

---

## Cross-feature interactions

### Produces: I-06 — Product-design replay contract

| Property | Contract |
|---|---|
| Consumers | E40-F08 Canonical multi-entity lifecycle runner (sole consumer) |
| Shape source | [Product-design replay contract](../architecture.md#product-design-replay-contract) |
| Payload | Authorized replay response sequence, D01-D05 artifact references and digests, request and response volume, revision and unresolved-gate counts, downstream artifact-consumption lineage, and terminal prelude outcome |
| Style | File artifact |
| Document split | **Input:** the versioned replay bundle at I-04's `replay_reference`. **Output:** the per-run replay result E40-F08 consumes. Named distinctly by REQ-F-001 so no consumer opens the wrong file (ADR-F07-08) |
| Join key to I-05 | `entry_digest` — computed per REQ-F-003, recomputable from the stored bundle alone, written verbatim by E40-F08 into each stage snapshot's `replay_lineage[].entry_digest` alongside the bundle path as `replay_reference` |
| Shared contract test | `tests/contracts/e40_i06_product_design_replay_contract_test.go#TC-052` |
| Consumer reads | `bench/replay/i06-schema.yaml`, `bench/replay/live-egress-tools.yaml`, and the "I-06 product-design replay contract" section of `bench/README.md` |
| Consumer invokes | `bench/scripts/run-prelude.sh`, `replay-answer.sh`, `verify-replay-result.sh`, and `verify-replay-isolation.sh`, rather than re-deriving matching, consumption, lineage, or isolation semantics — proxy-block and full Document A/B field-shape validation is TC-052's job (`go test ./tests/contracts/...`), invoked separately (see the `verify-replay-result.sh` row above) |
| Consumer split | E40-F08 runs the prelude ahead of the keyed lifecycle for a feature scenario, carries `terminal_outcome` and `publication_eligible` into I-07, and writes `entry_digest` into I-05's `replay_lineage[]`. E40-F10 reads the proxy block and `artifact_consumption_edges` for reuse and orphan reporting through I-07 — it does not read I-06 directly |
| Test scope | TC-052 reads in-repo replay artifacts only and requires no populated submodule, so `.github/workflows/ci.yml` is unchanged (ADR-F07-10); execution guards TC-053 through TC-059 run in `bench/scripts/tests/run-all.sh` |
| Gate mode | `contract-only`, staged by [the I-06 staged edge](../E40-interaction-map.md#i-06-staged-edge) — F07's producer necessarily runs before its consumer is decomposed |
| Activation owner | E40-F08 |
| Closure key | E40-F08, at its own UAT |
| Counterpart status | Read live from Shark at review/UAT time; not copied here as a fact that would go stale |
| Review basis | E40-F07's completed specification and this map row, present together at F07 task_review |
| Demonstrability disposition | `pending-integration` until E40-F08's live wiring closes |
| Closure owner (F07 side) | E40-F07 code-review owner, for the producer half of the contract only |
| Required UAT evidence | UAT-10 (F07): run a feature scenario through the existing Rider D01-D05 action using only its versioned response bundle; record each consumed response and artifact lineage; prove live research and human input are unreachable; prove a missing response stops as `unresolved_gate`; and prove the other three families record D01-D05 as non-applicable. UAT-18 additionally validates the interaction proxies and artifact-use edges with E40-F06/F08/F10 |

E40-F08 must copy the shape source and the contract-test pointer above verbatim;
the same test proves every side of this contract and no twin test is created.
This `contract-only` staging is a predeclared handoff, not a waiver — an open
internal activation obligation blocks epic completion until E40-F08 closes it.

### Consumes: I-04 — Lifecycle scenario package contract

| Property | Contract |
|---|---|
| Producer | E40-F05 Lifecycle scenario corpus and adapter contract |
| Shape source | [Lifecycle scenario package contract](../architecture.md#lifecycle-scenario-package-contract) |
| Payload | Versioned scenario identity, family, stage matrix, fixture and adapter, visible input, replay and evaluator references, resource policy, final predicate, and admission result |
| Style | File artifact |
| Shared contract test | `tests/contracts/e40_i04_scenario_contract_test.go#TC-030` |
| F07's consumption slice | `entity_family`, `stage_matrix.prelude`, and `replay_reference` — the slice [the I-04 staged edge](../E40-interaction-map.md#i-04-staged-edge) assigns to E40-F07 |
| How F07 consumes it | `stage_matrix.prelude` decides whether the prelude runs at all and supplies each non-applicable stage's verbatim `reason` (REQ-F-013); `entity_family` corroborates it; `replay_reference` locates the bundle and is cross-checked for consistency with the prelude matrix (REQ-F-014) |
| Non-regression obligation | Every I-04 artifact is byte-unchanged except `py-feature-recurring-tasks/evaluator/replay/reference-bundle.json`, the interior F05 explicitly deferred to E40-F07/I-06 in that file's own header and in `package.yaml`'s comment. `package.yaml` itself is unchanged (REQ-NF-006(c), AC-018) |
| Carve-out safety, verified 2026-08-15 | Two ways the carve-out could break a completed feature were checked and both are clear. (a) TC-030 validates `replay_reference` through `e40I04CheckPathField(..., requireExists: true)` — non-empty, package-relative, contained in the package directory, and existing. It asserts **nothing** about the file's content: no digest, no `_note` key, no byte length. Rewriting the interior therefore breaks no committed assertion and requires no edit to TC-030. (b) `package.yaml`'s `admission:` block records `status`, `base_outcome`, `reference_outcome`, and `toolchain_identity` (interpreter and tool versions plus `pyproject_sha256`, a fixture digest). **No recorded digest covers the package directory**, so writing the bundle does not invalidate the recorded admission and no re-admission is required |
| Gate mode | `contract-only`, staged by [the I-04 staged edge](../E40-interaction-map.md#i-04-staged-edge); E40-F07 is the activation owner for **its own slice** and closes it at its own UAT (UAT-10) with a real caller chain, shared-contract evidence, a production-path integration test, and a wiring-removal counterfactual |
| Closure key | E40-F07, at its own UAT |

No twin test is created for I-04: TC-030 remains the single shared proof of that
contract, and F07's guards consume the artifacts TC-030 validates.

---

## Cross-epic integrations

### Consumes: X-10 — Shark Rider product-design action and progress record

| Property | Contract |
|---|---|
| Producer epic / feature | E36 — Project Layer and Consult Bridge (E36-F02 Project namespace and progress record) |
| Consumer epic / feature | E40 — Shark Bench (E40-F07), the sole owner of this seam |
| Integration purpose | Invoke the existing Shark Rider product-design action and progress record for D01-D05 rather than copying the methodology into the benchmark |
| Contract / shape source | E36-F02 feature contract; Shark Rider product-design adapter; E40 architecture "Product-design replay contract" |
| UX / CX handoff notes | Scored feature scenarios use frozen inputs, but generated artifacts remain ordinary product-design artifacts and progress remains derived from disk |
| What F07 supplies to it | A digest-pinned interaction-routing preamble, a live-egress denial set, and a local resolver — all outside the frozen adapter and bundle trees. No production Rider behaviour changes |
| Verification posture | **Verified 2026-08-15.** E36-F02 is `completed`, so F07 takes no unmerged-branch dependency (research Finding 7). `skills/shark-rider/verbs/product-design.md` is the owning adapter: it reads `docs/product/progress.md`, retrieves the bundle via `shark skill get product-design`, owns checkpointing and the D04 stack-feedback response, and states "Never let the bundle invoke the CLI or retrieve another skill." `internal/sharkdata/default_data/skills/product-design/SKILL.md` §"Checkpoint boundary" confirms the bundle "does not own ordering, resumability, progress records, CLI commands, or retrieval of another skill" — the seam F07's replay adapter occupies. **One refinement of the research report was found by direct inspection and is recorded rather than left as a silent disagreement:** the report's "five call sites" enumeration is a *behavioural* surface, not an enumerable enforcement surface. `AskUserQuestion` is mandated at `SKILL.md:110` and `:133`, `WebSearch` at `SKILL.md:136`; of the five stage-level interaction points only `d01-vision.md:11` and `d03-market-research.md:22` carry a literal tool name, while D02, D04, and D05 elicit through prose that the SKILL-level mandate turns into tool calls at run time. REQ-F-004 therefore enforces on tool names session-wide and adds `WebFetch`, which is complete by construction over the same behaviour (ADR-F07-02) |
| Test coverage | UAT-10 and UAT-18; `tc053_live_egress_denial_test.sh` (AC-003), `tc054_replay_resolver_test.sh` (AC-004, AC-005), `tc058_prelude_placement_test.sh` (AC-012), and TC-052's structural cases. The REQ-NF-006(a) byte-freeze plus AC-018's diff review is the mechanical proof that the wrapped methodology was not forked |
| Deferral | None. No X-10 obligation is deferred to `docs/product/progress.md`. No generic Rider change is required by this feature; if one were identified it would be triaged under E36 per the epic's "any missing generic production behavior is separate work under the producing epic" rule |

E40-F07 produces, consumes, or validates **no other X-## row**. X-07 is owned by
E40-F02, X-08 by E40-F04, X-09 by E40-F06, X-11 and X-13 by E40-F08, and X-12 by
E40-F09. In particular, **X-13 is explicitly not used by this feature**: D01-D05's
human elicitation is an in-session tool call inside one interactive Rider action,
not a durable Shark Question entity, and the cross-epic map assigns X-13 to
E40-F08's post-prelude keyed lifecycle (research Finding 5 / Decision 5).

---

## Durable unresolved decisions

Applying the materiality test in
`internal/sharkdata/default_data/skills/question-management/SKILL.md` (F06's
spec cites this skill at `skills/question-management/SKILL.md`, which does not
exist in the working tree; the canonical embedded path is used here). **No
Question is created or proposed by this feature**; the items below are recorded
as non-material with their rationale, per the skill's "record the rationale in
the working document instead" guidance. F07 creates, claims, responds to, and
resolves nothing — the parent loop owns every Question transition.

1. **Whether the provider CLI's disallow mechanism covers builtin
   `AskUserQuestion` / `WebSearch`, and whether `AskUserQuestion` is offered at
   all in a non-interactive invocation.** Non-material *by construction*: the
   acceptance criterion is observational (REQ-F-005, AC-003(b)), so the contract
   holds either way and no requirement changes with the answer. Recorded here
   because a future reader may otherwise assume the denial flag is the gate.
   ADR-F07-03 carries the reasoning.
2. **Whether the dispatched session reliably routes through the resolver rather
   than answering from context.** Non-material as a *contract* question, because
   REQ-F-009's ledger reconciliation converts non-compliance into the named,
   detectable `unattributed_artifact` failure rather than a silent fabrication.
   It remains an implementation-quality concern for the preamble's wording,
   which is a tuning matter that changes no scope, acceptance criterion, or
   cross-feature contract.
3. **The `replay_wait_ns` plausibility ceiling (AC-008).** Non-material. The
   *existence* of a ceiling is the contract requirement; its numeric value is a
   tuning constant, and any value that keeps AC-008's synthesized-delay case
   discriminating satisfies the criterion. Same disposition F06 recorded for its
   reconciliation epsilon.
4. **Where the replay bundle for future scenario packages lives.** Non-material,
   and already settled by I-04: every consumer reaches it through the package's
   `replay_reference` pointer, so no contract changes with the answer. The seed
   package's `evaluator/replay/` location is F05's choice and is preserved.
5. **Whether `bundle_bulk_disclosure` should instead be enforced by adding
   `replay_reference` to I-04's `evaluator_only` block.** Not open — it is
   *closed as incorrect* by ADR-F07-07: it would require an I-04 edit forbidden
   by REQ-NF-006(c) and would make F06's dispatch-boundary guard fail the very
   dispatch F07 needs. Recorded as a design consequence, not a pending decision.

Epic-level Q001 and Q002 remain resolved; Q003 and the E27-F15 envelope question
belong to E40-F06's X-09 surface and are not this feature's concern.

---

## Verification plan

| Requirement | Evidence |
|---|---|
| REQ-F-001 | AC-001 — `tests/contracts/e40_i06_product_design_replay_contract_test.go#TC-052` |
| REQ-F-002 | AC-001, AC-004 |
| REQ-F-003 | AC-002 — I-05 `replay_lineage` join |
| REQ-F-004 | AC-003(a) — epic G10; UAT-10 |
| REQ-F-005 | AC-003(b) — epic G10; UAT-10 |
| REQ-F-006 | AC-004 |
| REQ-F-007 | AC-004, AC-005 — UAT-10 |
| REQ-F-008 | AC-004, AC-013 — epic G10, G12; UAT-10 |
| REQ-F-009 | AC-006 |
| REQ-F-010 | AC-007 — epic G18; UAT-18 |
| REQ-F-011 | AC-008 — epic G18; UAT-18 |
| REQ-F-012 | AC-009 |
| REQ-F-013 | AC-010 — UAT-10 |
| REQ-F-014 | AC-011 |
| REQ-F-015 | AC-012 |
| REQ-F-016 | AC-012, AC-018 |
| REQ-F-017 | AC-013 — epic G12 |
| REQ-F-018 | AC-015 |
| REQ-F-019 | AC-014 |
| REQ-NF-001, REQ-NF-002 | AC-017, AC-018 |
| REQ-NF-003 | AC-017 |
| REQ-NF-004 | AC-016 |
| REQ-NF-005 | AC-012; diff review: no shark initialisation command and no live-root write in `bench/scripts/` or `bench/replay/` |
| REQ-NF-006 | AC-018 |
| REQ-NF-007 | AC-008 (`replay_wait_category` branch) |

---

*Last Updated*: 2026-08-15
