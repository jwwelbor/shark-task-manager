---
feature_key: E40-F02
epic_key: E40
title: "Bench harness: run driver and metric collection"
type: combined-spec
tier: STANDARD (13/27)
date: 2026-08-06
related-docs:
  - docs/plan/E40-shark-bench-workflow-benchmarking-harness/epic.md
  - docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md
  - docs/plan/E40-shark-bench-workflow-benchmarking-harness/shark-bench-design.md
  - docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-interaction-map.md
  - docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-cross-epic-map.md
  - docs/plan/E40-shark-bench-workflow-benchmarking-harness/uat-plan.md
  - docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F02-bench-harness-run-driver-and-metric-collection/research-report.md
  - bench/README.md
---

# E40-F02 Specification — Bench harness: run driver and metric collection

## Context (references, not restatement)

- Business context and Phase 1 scope: epic PRD [§1](../epic.md), [§2](../epic.md) (G1–G7), [§3](../epic.md) (in scope / out of scope), [§4](../epic.md) (constraints).
- System-level decisions: [architecture.md](../architecture.md) — "Run lifecycle and isolation contract", "Corpus and oracle contract", "Metric collection and artifact schema", "Run liveness contract", ADR-001 through ADR-005.
- Per-metric mechanics: [shark-bench-design.md §4](../shark-bench-design.md).
- Feature scope and acceptance: [feature.md](feature.md).
- Capability reuse decisions: [research-report.md](research-report.md) "Capability map" and "Decisions" 1–9 — **binding on this spec**.

This feature adds no business capability beyond what the PRD states. Everything below is incremental to the epic.

### Capabilities reused, not re-implemented

Named from the research report's Capability map; this feature builds none of them.

| Reused capability | Source | What F02 does instead of building it |
|---|---|---|
| `shark run` execution engine | `internal/runner/controller.go` (`RunResult`/`StageLog`) | Invokes and parses unmodified (X-07) |
| Run liveness stream + `run.log` | `internal/runner/liveness.go` (`LivenessRecorder`, landed 2026-08-06) | Reads as a consumer (I-03) |
| Corpus manifest, items, ledgers | `bench/corpus/corpus.yaml`, `bench/corpus/ledgers/<base_sha>/*.json` | Reads directly (I-01) |
| Fixture checkout | `bench/scripts/checkout-fixture.sh <base_sha> <dest_dir>` | Invokes; never re-derives checkout or the clean-checkout verification |
| Ledger diff + toolchain guard | `bench/scripts/diff-ledgers.sh` (`--kind=lint\|test`, `--toolchain-guard`) | Invokes both modes; never re-derives the diff semantics documented in `bench/README.md` |
| Post-run ledger generation | `bench/scripts/build-ledgers.sh <checkout_dir> <output_dir>` | Invokes against the post-run checkout |
| Static test enumeration (anti-forgery cross-check) | `bench/scripts/testenum` | Invokes for the same completeness property `admit.sh`/`build-ledgers.sh` rely on |
| Held-back F2P injection | `bench/scripts/admit.sh:493` `copy_f2p_files` | Reuses the mechanism at terminal status instead of admission time |
| Scratch provisioning + live-repo guardrail | `scripts/shark-scratch-env.sh` | Invokes; never runs project-init commands against the live repo |
| `work_sessions` / `entity_history` | `internal/db/db.go` (entity-generic) | Queries the scratch DB; no migration |
| JSONL append convention | `internal/observability/file_jsonl_exporter.go` | Follows the open-append-encode shape, with one deliberate deviation (ADR-F02-07) |

Explicitly **not** reused: `internal/reporting` (`ScanReport`) — a fixed docs/plan scan schema, unrelated to bench records (architecture.md "Scope and component design").

---

## Requirements

### Functional

**Provisioning**

- **REQ-F-001** — A single command executes one `(corpus item, variant, rep)` end-to-end unattended and terminates in a recorded outcome. It takes the corpus item id, a variant id, a rep number, a timeout cap in seconds, and an output directory; it never prompts.
- **REQ-F-002** — The scratch shark project is provisioned by invoking `scripts/shark-scratch-env.sh`. The driver never invokes shark project-initialisation commands directly and never writes to the live repository, its `.sharkconfig.json`, or the live database.
- **REQ-F-003** — After provisioning, the driver repoints the scratch project's `workflow_config` at the variant's workflow bundle and sets `observability.capture_agent_transcripts` to `true`. Scratch `admin init` defaults to the embedded bundle and to transcripts off; neither default is acceptable, and neither is assumed to have been applied.
- **REQ-F-004** — The fixture checkout is created by invoking `bench/scripts/checkout-fixture.sh <base_sha> <dest_dir>` with `base_sha` read from the corpus manifest. The checkout survives the run and is the working tree every post-run check reads. `shark run --worktree` is never passed (ADR-001).
- **REQ-F-005** — Corpus entities are seeded through the shark CLI from the item's `seed_path` file. `type: task` items require a host epic and feature in the scratch project, which the driver creates first; `type: bug` items are standalone. The driver never specifies an entity key: every key is captured from the create command's `--json` response and recorded in the run manifest block.

**Invocation**

- **REQ-F-006** — The driver invokes `shark run <key> --json --workdir <fixture_checkout>` under a timeout cap. The invocation is placed in its own process group, and cap expiry tears down the whole group — not only the `shark` process — so no agent subprocess survives the cap.
- **REQ-F-007** — The driver records its own monotonic start and end around the invocation, the process exit status, and captures stdout, stderr, and the run's `run.log` to separate files in the run's artifact directory before any parsing occurs.
- **REQ-F-008** — The driver resolves the run's `run_id` from the liveness stream (I-03): the `run.log: <path>` line `LivenessRecorder.Start()` prints once on stderr before any event line, and the `run_id` field carried on every NDJSON event. `RunResult` carries no run id, so this is the only deterministic source. If the liveness stream yields none, the driver falls back to the newest directory under `<scratch_root>/.shark/runs/` and records `run_id_source: "fallback_newest_dir"` on the record.

**Collection**

- **REQ-F-009** — The collector parses `RunResult` stdout for `entity_key`, `final_status`, `stages_completed`, `outcome`, `total_duration_ns`, `error`, `question_block`, and per-stage `status`, `action`, `agent_type`, `provider`, `duration_ns`, `exit_code`. `outcome` is one of `completed`, `paused`, `failed`, `already_terminal`, `no_action` — five values, not three.
- **REQ-F-010** — The record's harness `outcome` field is the five `RunResult` values plus `timeout`, which is harness-assigned and never appears in `RunResult`. A `paused` outcome carrying a `question_block` is surfaced verbatim and is excluded from the defect rollup: a Question-blocked run is not a defect and not a timeout.
- **REQ-F-011** — For every stage whose `action` is `spawn_agent`, the collector locates the stage transcript under `<scratch_root>/.shark/runs/<run_id>/<stage_n>-<status>-<provider>.log`, extracts the agent JSON envelope from the `---STDOUT---` block, and records token usage, cost, API duration, turn count, and exact model IDs.
- **REQ-F-012** — A transcript whose envelope is missing or has renamed an expected field produces a **named, visible parse error** on that run's JSONL record — never a zero, an empty string, or an absent key masquerading as a measurement. This applies to `spawn_agent` stages only; `advance_status` stages write no transcript and produce no error.
- **REQ-F-013** — The collector reconciles the number of `spawn_agent` stages in `RunResult` against the number of transcript files, and asserts that the k-th transcript's `status` and `provider` filename components match the k-th agent stage. A count mismatch or a component mismatch is a named join error on the record. (`RunController` latches transcript writing off for the remainder of a run after a single write failure, so a silent shortfall is a real failure mode.)
- **REQ-F-014** — Rejections are inferred from `RunResult`: re-entry into a status already seen earlier in `stages[]`, following a gate stage, is one rejection attributed to that gate; the re-entry count is `rework_loops`. The inference is cross-checked against the scratch DB's `entity_history` backward transitions and `work_sessions` outcomes, and a disagreement is recorded as a named discrepancy rather than silently resolved in favour of either source.
- **REQ-F-015** — Post-run checks run in the fixture checkout once the run has ended, in a **pinned order** (ADR-F02-11), because two of them are corrupted by the held-back F2P files if injection happens first:
  1. `diff-ledgers.sh --toolchain-guard --base=<base ledger>` — aborts the whole post-run phase, naming every mismatched axis, before any diff would run.
  2. **LOC** (REQ-F-017) — measured on the agent's tree only.
  3. **Quality gates** (REQ-F-016) and post-run ledger generation via `build-ledgers.sh`, then `diff-ledgers.sh --kind=test` and `--kind=lint` against the base-SHA ledgers.
  4. **F2P injection and the oracle run, last** — the item's held-back files are copied in and the F2P test set is run.
- **REQ-F-016** — Quality gates run in the fixture checkout **before F2P injection** and are recorded individually: formatting cleanliness, `go vet`, and the full test suite. Each gate records its own pass/fail plus its raw output path; a gate that could not be executed is recorded as `null` with a reason, never as a pass.
- **REQ-F-017** — LOC is recorded **before F2P injection**, from `git diff --numstat <base_sha>` over the checkout with untracked files staged as intent-to-add, split into production and `*_test.go` lines added/deleted, plus `files_touched`. Injected corpus files must never appear in this measurement.
- **REQ-F-018** — The run emits exactly one JSONL record, at a deterministic, self-identifying path derived from `(item_id, variant_id, rep)`, so a later batch runner can detect an already-completed pair without re-reading record contents.
- **REQ-F-019** — A record is emitted for every run that starts, including a timed-out run, a crashed run, and a run whose stdout carried no `RunResult`. For a timed-out run, stage attribution comes from the liveness record (I-03), which is appended per event and therefore survives SIGKILL; the `run_end` summary line does **not** exist on that path, and the collector must not require it.
- **REQ-F-020** — An X-07 canary preflight asserts, against a real `shark run` invocation, that the `RunResult` and `StageLog` field set and the transcript byte format (`COMMAND:` / `EXIT:` / `DURATION:<ms>ms` / `---STDOUT---` / `---STDERR---`) are unchanged. A mismatch aborts, naming the changed field; it never downgrades to a warning. `run-one.sh` invokes it by default before provisioning, so the requirement has a caller inside this feature; a `--skip-canary` flag lets a later batch runner (out of scope here) pay for it once per batch instead of once per run.
- **REQ-F-021** — Before the envelope parser is implemented, one real transcript captured from F02's own first live run is inspected and the exact names of `modelUsage`, `num_turns`, and `duration_api_ms` are confirmed and recorded in `bench/README.md`. If `modelUsage` is absent, the exact-model-ID source is the envelope's top-level `model` field, and the record notes which source was used.

### Non-functional

- **REQ-N-001** — Zero production Go changes. Nothing under `internal/` or `cmd/` is modified. A Go **contract test** under `tests/contracts/` is permitted and expected, following E40-F01's precedent (F01 shipped `tests/contracts/e40_i01_corpus_contract_test.go` under the same rule).
- **REQ-N-002** — The live repository, its `.sharkconfig.json`, and the live database are untouched by every code path, and the shark-config guardrail hook stays satisfied. Held-back F2P tests are absent from the checkout the agent works in, verified by inspecting the working tree, not by intent.
- **REQ-N-003** — Bounded worst case: a run cannot exceed its cap plus a fixed grace, and cannot leave an orphaned agent process consuming API budget after the cap fires.
- **REQ-N-004** — Record content is deterministic for a fixed input: object keys are emitted sorted, and list-valued fields are emitted in a fixed order, so two runs' records differ only where the measurements differ. This is what makes G7 replay checkable.
- **REQ-N-005** — Fail loud everywhere a measurement could be fabricated: a subprocess's exit code alone is never accepted as evidence that "zero results" occurred, and an unparseable producer output is refused rather than written as a zero. (This is the same defect-class guard `build-ledgers.sh` already carries in its own header.)
- **REQ-N-006** — Scripts match the existing `bench/scripts` conventions: `bash` entry point with `set -euo pipefail`, embedded `python3` (PyYAML available) for YAML/JSON work, machine-readable JSON to stdout, diagnostics to stderr, and a self-test under `bench/scripts/tests/tcNNN_*_test.sh` registered in `bench/scripts/tests/run-all.sh`.
- **REQ-N-007** — Every measurement in a record names its source, so a consumer can tell a `RunResult`-derived number from a transcript-derived one from a DB-derived one without re-deriving the provenance.

### Acceptance criteria

| ID | Criterion |
|---|---|
| AC-01 | Running the single-run command for one admitted corpus item, the default variant, and rep 1 completes with no human interaction and writes exactly one JSONL record at the deterministic path. |
| AC-02 | The emitted record carries all six metric families: time (per-stage wall, API duration, harness wall), tokens and cost per agent stage, rejections per gate, oracle result plus `p2p_regressions`, quality gates (`fmt`/`vet`/`lint_new_issues`/`tests_pass`), and LOC split prod vs test. |
| AC-03 | A run whose stage exceeds the cap is killed, the record carries `outcome: "timeout"`, and the record names the stalled stage sourced from the liveness record — with no `RunResult` and no `run_end` line available. |
| AC-04 | After a timed-out run, no `shark` or agent process from that run remains alive (verified by process-group inspection, not by exit code). |
| AC-05 | A transcript whose envelope is missing an expected field yields a named parse error on the record naming the field and the transcript path; the corresponding metric is absent, not zero. Driven by a synthetic transcript fixture, not a live run. |
| AC-06 | An `advance_status` stage, which writes no transcript, produces no parse error and no missing-transcript error. |
| AC-07 | With one transcript deliberately removed, the collector reports a named join error stating the expected and observed agent-stage/transcript counts. |
| AC-08 | A `RunResult` with `outcome: "paused"` and a populated `question_block` is recorded with that outcome, the question block surfaced verbatim, and is not counted as a defect or a timeout. Each of `already_terminal` and `no_action` likewise round-trips into the record's `outcome` field unchanged. |
| AC-09 | `p2p_regressions` and `lint_new_issues` in the record are byte-identical to the counts `bench/scripts/diff-ledgers.sh` prints for the same base/post ledger pair — the collector performs no independent diff. |
| AC-10 | With a toolchain axis deliberately mismatched, the post-run checks abort naming that axis, and the record records the abort rather than a diff computed under the wrong toolchain. |
| AC-11 | The live repository's `.sharkconfig.json`, working tree, and database are byte-identical before and after a full single run. |
| AC-12 | Inspecting the fixture checkout's working tree at dispatch time shows no held-back F2P file present; the same files are present after post-run injection. |
| AC-13 | The X-07 canary fails, naming the changed field, when run against a `RunResult` fixture with a renamed field, and passes against the current shape. |
| AC-14 | Two independent runs of the collector over the same completed run directory produce byte-identical records. |
| AC-15 | `tests/contracts/e40_i02_artifact_contract_test.go#TC-001` validates a committed golden record against the I-02 schema and passes under `make test` with no submodule, no scratch project, no network, and no API spend. |
| AC-16 | The seeded task item's assigned key in the record matches the key the create command returned, and no create command in the driver passes an explicit key. |
| AC-17 | The confirmed envelope field names, and whichever exact-model-ID source was used, are recorded in `bench/README.md` and reflected on the record. |
| AC-18 | For a run in which the agent changed nothing, `loc.test_added` and `lint_new_issues_count` are both zero **even though the injected F2P files are present in the checkout after the run** — proving LOC and the lint diff were measured before injection. |
| AC-19 | Against a synthetic scratch DB whose `entity_history` backward transitions disagree with the `RunResult`-derived rejection count, the record carries a `crosscheck_disagreement` error naming both counts, and neither source silently wins. The same fixture exercises the `entity_key` → `entity_id` resolution `entity_history` requires. |
| AC-20 | Against a completed run directory with no liveness stream, the collector resolves `run_id` by newest directory, locates the transcripts, and records `run_id_source: "fallback_newest_dir"`. |
| AC-21 | `run-one.sh` invokes the canary by default and aborts before provisioning when it fails; `--skip-canary` suppresses the invocation and is recorded on the run's `meta.json`. |

### Traceability

Every requirement maps to at least one acceptance criterion and to the epic criterion it serves ([epic PRD §2](../epic.md)). Phase 1 exit owns G1–G5 and G7; F02 serves G2 (unattended termination), G4 (complete metrics), and G7 (reproducibility), and supplies the per-run inputs G5 is computed from by F03.

| Requirement | AC | Epic criterion |
|---|---|---|
| REQ-F-001 single-run command | AC-01 | G2 |
| REQ-F-002 scratch provisioning | AC-11 | G2 (constraint: never the live repo) |
| REQ-F-003 workflow_config + transcripts | AC-01, AC-02 | G4 |
| REQ-F-004 fixture checkout via F01 script | AC-12 | G1 (consumed), G4 |
| REQ-F-005 seeding, keys captured | AC-16 | G2 |
| REQ-F-006 invocation under a process-group cap | AC-03, AC-04 | G2 |
| REQ-F-007 raw capture before parsing | AC-14 | G7 |
| REQ-F-008 run_id from I-03 (+ fallback) | AC-02, AC-20 | G4 |
| REQ-F-009 RunResult parse, five outcomes | AC-02, AC-08 | G4 |
| REQ-F-010 outcome superset, question_block | AC-08 | G2, G4 |
| REQ-F-011 envelope parse | AC-02 | G4 |
| REQ-F-012 fail-loud, agent stages only | AC-05, AC-06 | G4 |
| REQ-F-013 stage/transcript reconciliation | AC-07 | G4 |
| REQ-F-014 rejections + DB cross-check | AC-02, AC-19 | G4 |
| REQ-F-015 pinned post-run order + oracle | AC-02, AC-09, AC-10, AC-18 | G4 |
| REQ-F-016 quality gates | AC-02, AC-18 | G4 |
| REQ-F-017 LOC | AC-02, AC-18 | G4 |
| REQ-F-018 deterministic artifact path | AC-01 | G2 (batch skip), G7 |
| REQ-F-019 record emitted for every run | AC-03 | G2, G4 |
| REQ-F-020 X-07 canary | AC-13, AC-21 | G4, G7 (X-07) |
| REQ-F-021 Q003 capture + model fallback | AC-17 | G4, G7 |
| REQ-N-001 zero production Go changes | AC-15 | epic constraint §4 |
| REQ-N-002 live repo/DB untouched | AC-11, AC-12 | epic constraint §4 |
| REQ-N-003 bounded worst case | AC-03, AC-04 | G2 |
| REQ-N-004 deterministic record content | AC-14 | G7 |
| REQ-N-005 fail loud, never fabricate | AC-05, AC-07, AC-10, AC-19 | G4 |
| REQ-N-006 bench script conventions | AC-15 (CI-safe) + the three `tcNNN` self-tests | — |
| REQ-N-007 every metric names its source | AC-15 | G4 |

### Out of scope for this feature

- Matrix/batch orchestration across items, variants, or reps, and the already-completed-pair skip that UAT-01 needs (F03 and Phase 2). F02 owns only the deterministic naming that makes the skip implementable (REQ-F-018).
- Variant **bundle authoring** and A/B comparison (Phase 2, G6/UAT-03/UAT-04). Phase 1's only variant is the default bundle; F02 builds the repoint mechanism and records the variant identity, nothing more.
- Aggregation, the noise band, and the G7 replay command (E40-F03).
- Core instrumentation G1–G3 — usage/outcome fields on `StageLog`, `outcome_source`, `--stage-timeout` (Phase 2).
- Codex-dispatched steps: the codex path emits no usage envelope (G4, Phase 3). A codex stage records its `RunResult`-derived fields and an explicit `usage_unavailable` reason, not a zero.
- Cascade-child attribution. Phase 1 benches tasks and bugs only (ADR-005/Q004), so no cascade path is exercised. **B052** (sibling children collide on transcript filename; child stages flatten into the parent with no entity key) is constrained out by that scoping, not fixed, and remains a named prerequisite risk for Phase 2 feature benching.
- **B053** (`admit.sh`'s per-item `run_selector` resolution) is F01-scoped and does not reach F02: `diff-ledgers.sh` never references `run_selector` or `p2p_set`, and its `--kind=test` diff is a flat base-vs-post ledger identity comparison. Verified directly; recorded here so Phase 2 need not re-derive the scoping.

---

## Architecture

### Component changes

All new files live outside shark's Go production tree, except one contract test.

| Path | Change | Responsibility |
|---|---|---|
| `bench/scripts/run-one.sh` | New | Driver: provision → seed → invoke → hand off to the collector. Owns the scratch project, the fixture checkout, the timeout cap, and the run directory layout. |
| `bench/scripts/collect-run.sh` | New | Collector: a **pure function of a completed run directory** → one JSONL record. Reads only files already captured by the driver plus the scratch DB and the fixture checkout. Never invokes `shark run`. |
| `bench/scripts/canary-runsurface.sh` | New | X-07 preflight (REQ-F-020). Asserts the `RunResult`/`StageLog` field set and transcript byte format against a real invocation. |
| `bench/scripts/tests/tc014_run_one_smoke_test.sh` | New | Driver smoke test against a stubbed `shark` on PATH — no API spend. |
| `bench/scripts/tests/tc015_collect_run_record_test.sh` | New | Collector over committed synthetic run directories: five outcomes, timeout path, missing envelope field, missing transcript, `advance_status` stage, absent liveness stream (`run_id` fallback), and a scratch DB whose `entity_history` disagrees with the `RunResult` rejection inference. |
| `bench/scripts/tests/tc016_canary_runsurface_test.sh` | New | Canary passes on the current shape, fails naming the field on a mutated fixture. |
| `bench/scripts/tests/run-all.sh` | Modify | Register the three new tests. |
| `bench/scripts/testdata/run/` | New | Synthetic run directories (stdout `RunResult` JSON, stderr NDJSON, `run.log`, transcripts) driving the collector tests with no live run. |
| `tests/contracts/e40_i02_artifact_contract_test.go` | New | `TC-001`: the shared I-02 contract test. Validates the committed golden record against the schema. |
| `tests/contracts/testdata/e40_i02_golden_record.jsonl` | New | The committed golden record TC-001 validates. |
| `bench/README.md` | Modify | Add "Run driver and artifact schema": the run directory layout, the I-02 record schema field reference, and the Q003-confirmed envelope field names. |

Nothing under `internal/` or `cmd/` is touched (REQ-N-001).

### Data model changes

**No shark database change**: no table, no column, no migration, no `CurrentSchemaVersion` bump (ADR-002). `work_sessions` and `entity_history` are already `entity_type`-generic and are read as-is.

The data model this feature introduces is the **I-02 JSONL record**. One record per run, one line, sorted keys.

| Block / field | Type | Source | Notes |
|---|---|---|---|
| `schema_version` | string | constant | Pinned; TC-001 asserts a supported value. |
| `manifest.item_id` | string | `corpus.yaml` | |
| `manifest.item_type` | string | `corpus.yaml` | `task` or `bug`. |
| `manifest.variant_id` | string | driver arg | Phase 1: `default`. |
| `manifest.rep` | integer | driver arg | 1-based. |
| `manifest.run_key` | string | derived | `<item_id>::<variant_id>::rep<rep>` — the self-identifying key behind REQ-F-018. |
| `manifest.fixture_base_sha` | string | `corpus.yaml` `fixture.base_sha` | |
| `manifest.corpus_schema_version` | string | `corpus.yaml` `schema_version` | |
| `manifest.p2p_set` | string | item's `p2p_set` | |
| `manifest.variant_bundle_sha256` | string | derived | Content hash over the installed workflow bundle, sorted by path. Pins the variant for G7 without requiring the bundle to be a git object. |
| `manifest.shark_version`, `manifest.shark_binary_sha256` | string | driver | G7 reproducibility. |
| `manifest.model_ids` | array of string | envelope | Exact IDs, deduplicated, sorted. |
| `manifest.model_id_source` | string | envelope | `modelUsage` or `model` (REQ-F-021). |
| `manifest.timeout_cap_s` | integer | driver arg | |
| `manifest.seeded_keys` | object | create `--json` | Assigned keys for the epic, feature, and benched entity. Never harness-chosen. |
| `manifest.run_id`, `manifest.run_id_source` | string | I-03 stream / fallback | |
| `outcome` | string | derived | One of the five `RunResult` values plus `timeout`. |
| `timeout_detail` | object | liveness stream / scratch DB | Present **only** when `outcome == "timeout"`, otherwise absent (never a zero-valued object). `{stage_index, status, action, agent_type, provider, source}`, where `source` is `"liveness_stream"` or `"scratch_db_status_fallback"` (REQ-F-019, AC-03; schema gap closed on the test-plan.md-pinned decision, 2026-08-06). |
| `runresult.final_status`, `.stages_completed`, `.total_duration_ns`, `.error` | — | `RunResult` | Copied verbatim. |
| `runresult.question_block` | object \| null | `RunResult` | Surfaced verbatim when present. |
| `stages[]` | array | `RunResult.Stages` + transcripts | Per stage: `index`, `status`, `action`, `agent_type`, `provider`, `duration_ns`, `exit_code`, and for agent stages a `usage` sub-object (`input_tokens`, `output_tokens`, `cache_read_input_tokens`, `cache_creation_input_tokens`, `total_cost_usd`, `duration_api_ms`, `num_turns`, `model_ids[]`) plus `transcript_path`. |
| `timing.harness_wall_ns` | integer | driver t0/t1 | Includes process spawn and teardown; distinct from `total_duration_ns`. |
| `rejections.by_gate` | object | `RunResult` | Gate status → count. |
| `rejections.rework_loops` | integer | `RunResult` | |
| `rejections.crosscheck` | object | scratch DB | `entity_history_backward_transitions`, `work_session_outcomes`, `agrees` (bool). |
| `oracle.f2p_resolved` | bool \| null | post-run | `null` only with a recorded reason. |
| `oracle.repro_confirmed` | bool \| null | post-run | Bug items only; the item's own `f2p.test_names` is the repro set. |
| `oracle.p2p_regressions[]`, `.p2p_regressions_count`, `.removed[]`, `.removed_count` | — | `diff-ledgers.sh --kind=test` | Copied from the script's stdout, not recomputed. |
| `quality.fmt_clean`, `.vet_ok`, `.tests_pass` | bool \| null | post-run | |
| `quality.lint_new_issues[]`, `.lint_new_issues_count` | — | `diff-ledgers.sh --kind=lint` | Copied from the script's stdout. |
| `quality.toolchain_guard` | string | `diff-ledgers.sh --toolchain-guard` | `pass`, or the named mismatched axes. |
| `loc.prod_added/.prod_deleted/.test_added/.test_deleted/.files_touched` | integer | `git diff --numstat` | |
| `errors[]` | array | collector | Named, visible failures: `envelope_parse_error`, `stage_join_error`, `transcript_missing`, `crosscheck_disagreement`, `postrun_check_aborted`, `usage_unavailable`. Each carries `kind`, `detail`, and where applicable `stage_index` and `path`. Empty array on a clean run. |
| `sources` | object | collector | Per metric family, which of `runresult` / `transcript` / `scratch_db` / `postrun` / `liveness` produced it (REQ-N-007). `sources.stalled_stage` (or equivalent key naming which family `timeout_detail` came from) is `"liveness"` when resolved from the stream, `"scratch_db"` when resolved from the DB status fallback. |

### Interface contracts

```
bench/scripts/run-one.sh --item <id> --variant <id> --rep <n> \
                         --timeout <seconds> --out <artifact_root> \
                         [--corpus <corpus.yaml>] [--keep-scratch]

bench/scripts/collect-run.sh --run-dir <dir>          # prints one JSON object to stdout
bench/scripts/canary-runsurface.sh [--corpus <corpus.yaml>]
```

Run directory layout, created by the driver and consumed by the collector:

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

The split matters: `collect-run.sh` reads only this directory (plus the scratch DB and the fixture checkout paths named in `meta.json`), so every collector behaviour is testable from committed fixtures with no API spend.

Envelope extraction rule, pinned against `internal/runner/transcript.go`'s documented byte format: split the transcript on the **first** occurrence of `\n---STDOUT---\n` and the **last** occurrence of `\n---STDERR---\n`; the text between is the agent's raw stdout and is parsed as one JSON object. A transcript that does not match this shape is a named parse error, never a skipped stage.

### Key technical decisions

- **ADR-F02-01 — Driver and collector are separate scripts.** The collector is a pure function of a completed run directory. Rationale: every collector rule (five outcomes, timeout path, missing field, missing transcript, join mismatch) becomes testable from committed synthetic fixtures with no live agent and no API spend, and the same collector re-runs over a stored run directory for G7 replay. Follows the existing `bench/scripts` split between `build-ledgers.sh` (produce) and `diff-ledgers.sh` (interpret).
- **ADR-F02-02 — `run_id` is resolved from I-03, not from stdout.** `RunResult` has no run id field — `runID := generateRunID()` in `internal/cli/commands/run.go:90` is a uuid that never reaches stdout. `LivenessRecorder.Start()` prints `run.log: <abs path>` on stderr once before any event, and every NDJSON line carries `run_id`. This makes I-03 a hard dependency for the **tokens/cost/model-ID family**, not only for timeout stage attribution as the interaction map's sequencing note frames it. The newest-directory fallback is recorded on the record because it is racy under any concurrency and must be visible, not silent.
- **ADR-F02-03 — Timeout kills the process group, not just `shark`.** `claude` is a child of `shark`; SIGTERM to `shark` alone can orphan an agent that keeps consuming budget, defeating the bounded-worst-case claim the batch depends on. The invocation runs in its own session/process group with a kill-after grace, and cap expiry signals the group. Consequence: on the kill path `run.go`'s `defer rec.Stop(); rec.Finish(runResult)` never runs, so **no `run_end` line exists** — the timeout path reads the per-event-appended `run.log`/stderr instead (REQ-F-019).
- **ADR-F02-04 — Harness `outcome` is a superset in its own field.** `timeout` is harness-assigned and cannot come from `RunResult`; the five `RunResult` values are copied unchanged. `paused` + `question_block` is a first-class outcome excluded from the defect rollup. Folding a Question-blocked pause into `failed` would make it indistinguishable from a real defect — the exact misclassification the research report's Finding 3 identifies.
- **ADR-F02-05 — Fail-loud parse posture, scoped to agent stages.** Consistent with ADR-004's canary stance: a missing or renamed envelope field is a named, visible error, never a silently-zeroed metric. Scoped to `spawn_agent` stages because `maybeWriteTranscript` is called only from `handleSpawnAgent` and `recordDispatchFailure` (`internal/runner/controller.go:1191/1244/1447`) — an unscoped rule would manufacture a spurious error for every `advance_status` stage.
- **ADR-F02-06 — Invoke F01's scripts; never re-derive.** `checkout-fixture.sh`, `build-ledgers.sh`, `diff-ledgers.sh` (both diff modes and the toolchain guard), and `testenum` are built, tested, and documented. A second implementation of the checkout, the diff semantics, or the toolchain comparison would fork an already-owned contract. The collector copies the diff scripts' stdout into the record rather than recomputing counts (AC-09).
- **ADR-F02-07 — The record write is verified, not fail-soft.** This deviates from `internal/observability/file_jsonl_exporter.go`'s swallow-errors convention, which the research report's Capability map offered as a pattern. Rationale: for an observability exporter a dropped line is a lost log entry; here it is a lost run and a hole in the noise band that F03 would average over. The append shape is reused; the error swallowing is not. Surfacing this deliberately rather than blending the two conventions.
- **ADR-F02-08 — Seeding creates a synthetic host epic and feature.** `seed.yaml` carries only `type`, `title`, `description`, and `severity` for bugs — no parent scaffold. `shark create task` requires an epic and a feature, so the driver creates both in the scratch project first and captures every assigned key from the `--json` response (never specifying keys, per the cloud-DB key-assignment rule). A `no_action` outcome on a freshly seeded entity is the diagnostic signal for a seeding or workflow-config misconfiguration, and is recorded as such rather than treated as a benign no-op.
- **ADR-F02-09 — The rejection cross-check resolves key → id.** `work_sessions` carries `entity_key` directly, but `entity_history` keys on `entity_type` + `entity_id INTEGER` (`internal/db/db.go:219-230`). The cross-check resolves the seeded entity's row id from the scratch DB first. A disagreement between the `RunResult` inference and the DB is recorded as a named discrepancy, not silently resolved — the two sources measure the same thing by different means, and a divergence is a finding.
- **ADR-F02-10 — The X-07 canary is a harness preflight, not a CI test.** architecture.md requires it to assert "against a real invocation", which needs a scratch project and a dispatch. It aborts the batch; it is deliberately not a `make test` resident. `tests/contracts/e40_i02_artifact_contract_test.go` is the opposite: schema-only over a committed golden record, so it runs in CI with no submodule, no scratch project, and no API spend — the same CI-safety property F01's ADR-F01-05 pins for TC-001.
- **ADR-F02-11 — LOC and the lint diff are measured before F2P injection; the oracle runs last.** The naive order — inject, then measure — silently corrupts two metric families. `git add -A -N` + `git diff --numstat <base_sha>` cannot distinguish a corpus-injected test file from an agent-authored one, so every run's `test_added` would carry a constant inflation the noise band would then read as signal. And `build-ledgers.sh` over a checkout that already contains the injected files puts any lint issue *inside those files* into the post ledger, where `diff-ledgers.sh --kind=lint` reports it as a new issue attributable to the agent. (`--kind=test` is unaffected: a test absent at base produces no regression entry by construction.) Rejected alternative: inject once and exclude corpus-injected paths from numstat and the lint ledger by path — it works, but it puts a path-filter rule in two places instead of an ordering rule in one, and a filter that drifts fails silently while an ordering violation is caught by AC-18.

### Integration with existing code

Read-only integration points, with the exact surfaces this feature depends on:

| Surface | Location | Dependency |
|---|---|---|
| `RunResult`, `StageLog` JSON tags | `internal/runner/controller.go:82-135` | Field names parsed verbatim; pinned by the X-07 canary. |
| `Outcome` enum, `QuestionBlock` | `internal/runner/controller.go:96-107` | Five values; optional `question_block`. |
| Transcript byte format and path layout | `internal/runner/transcript.go:10-25`, `relTranscriptPath` at `:51` | `.shark/runs/<run_id>/<stage_n>-<status>-<provider>.log`; the five markers. |
| Transcript write sites and the run-scoped latch | `internal/runner/controller.go:1191`, `:1244`, `:1447`, `maybeWriteTranscript` at `:1479` | Agent stages only; one warning then all further writes suppressed → REQ-F-013. |
| Liveness NDJSON schema and `run.log` sink | `internal/runner/liveness.go` (`ndjsonLine`, `LogPath`, `Start`, `writeLogLine`) | 11 fields, per-event append, path announced on stderr at start. |
| Liveness wiring | `internal/cli/commands/run.go:157-159`, `:346` | Confirms the recorder runs for every invocation, `--json` or not. |
| `--workdir` flag | `internal/cli/commands/run.go:72`, `:285` | Agent-process cwd override; threaded to `RunOptions.WorkingDir`. Never `--worktree`. |
| `capture_agent_transcripts` config key | `internal/config/config.go:277-279`, `internal/config/manager.go:152` | Written into the scratch project's config. |
| `work_sessions` columns | `internal/db/db.go:1356-1370` | `entity_type`, `entity_key`, `session_id`, `started_at`, `ended_at`, `outcome`. |
| `entity_history` columns | `internal/db/db.go:219-230` | `entity_type`, `entity_id`, `from_status`, `to_status`, `changed_at`. |
| Scratch provisioning | `scripts/shark-scratch-env.sh` | Prints the scratch dir; copies the binary into it. |
| Corpus manifest and items | `bench/corpus/corpus.yaml`, `bench/corpus/items/*/`, `bench/corpus/ledgers/<base_sha>/{tests,lint}.json` | Read per `bench/README.md` "Manifest schema" and "`p2p_set` resolution rule". |
| F2P injection mechanism | `bench/scripts/admit.sh:493` `copy_f2p_files` | Same copy mechanism, invoked at terminal status. |

---

## Cross-feature interactions

### Consumes: I-01 — Corpus and oracle contract

| Property | Contract |
|---|---|
| Producer | E40-F01 Benchmark corpus v1: fixture repo and screened tasks |
| Shape source | `architecture.md#corpus-and-oracle-contract` |
| Contract test | `tests/contracts/e40_i01_corpus_contract_test.go#TC-001` |
| Consumer reads | `bench/corpus/corpus.yaml`, `bench/corpus/items/*/`, `bench/corpus/ledgers/<base_sha>/*.json`, and the diff method in `bench/README.md` |
| Consumer invokes | `bench/scripts/checkout-fixture.sh` and `bench/scripts/diff-ledgers.sh` (both modes), rather than re-deriving the checkout or the diff method |
| Gate mode | `live`, as assigned by [the interaction map](../E40-interaction-map.md) |

Shape source and contract test are copied verbatim from F01's `spec.md`. The same test proves both sides; no twin test is created. This discharges the spec-side half of the obligation recorded on E40-F02 (decision note, 2026-08-05); `task_generation` must still create at least one task declaring `I-01: consumes` that owns the real caller path.

### Consumes: I-03 — Run liveness contract

| Property | Contract |
|---|---|
| Producer | E40-F04 shark run live progress and per-run log |
| Shape source | `architecture.md#run-liveness-contract` |
| Contract test | `tests/contracts/e40_i03_liveness_contract_test.go#TC-001` |
| Consumer reads | The stderr NDJSON stream and `.shark/runs/<run_id>/run.log` under the scratch project root |
| Real caller path | `run_id` resolution for locating stage transcripts (REQ-F-008, ADR-F02-02) and stalled-stage attribution for a timed-out run (REQ-F-019, UAT-05) |
| Gate mode | `live`, as assigned by [the interaction map](../E40-interaction-map.md) |

Shape source and contract test are copied verbatim from F04's `spec.md` "Cross-feature interactions". This discharges the spec-side half of the obligation recorded on E40-F02 (decision note, 2026-08-06); `task_generation` must still create at least one task declaring `I-03: consumes`.

Note for the interaction map's "Sequencing note on I-03": the note frames I-03 as strongly preferred but not a hard dependency, on the grounds that transcript numbering bounds the last completed stage. That fallback holds for **stage attribution**. It does not hold for **transcript location**: `RunResult` carries no run id, so without the liveness stream the collector can only guess `<run_id>` by directory mtime. F02 therefore consumes I-03 for two distinct purposes, and records which source it used.

### Produces: I-02 — Metric collection and artifact schema

| Property | Contract |
|---|---|
| Consumer | E40-F03 Baseline report and noise band |
| Shape source | `architecture.md#metric-collection-and-artifact-schema` |
| Contract test | `tests/contracts/e40_i02_artifact_contract_test.go#TC-001` |
| Payload | One JSONL record per run: manifest block (item, variant, rep, SHAs, exact model IDs, timeout cap), per-stage records, post-run check results, and a rollup |
| Style | File artifact |
| Gate mode | `live`, as assigned by [the interaction map](../E40-interaction-map.md) |

TC-001 validates the committed golden record `tests/contracts/testdata/e40_i02_golden_record.jsonl` against the schema above: required blocks present, `schema_version` supported, `outcome` within the closed six-value set, `errors[]` entries carrying `kind` and `detail`, and every metric family declaring a source. It reads in-repo artifacts only — no submodule, no scratch project, no network, no API spend — matching F01's ADR-F01-05 CI-safety property.

**Consumer-side mirror obligation.** E40-F03 has not entered `task_generation`, so no F03 task exists today to declare `I-02: consumes`. This mirrors the map's "Sequencing note on I-01" precedent: build-order sequencing, not an open gap. The obligation is recorded as a decision note on E40-F03 — F03's `task_generation` must create at least one task declaring `I-02: consumes` that copies the shape source `architecture.md#metric-collection-and-artifact-schema` and the contract test `tests/contracts/e40_i02_artifact_contract_test.go#TC-001` **verbatim**, and owns the real caller path that aggregates records. The uat-plan's I-02 scenario also binds F03: a record missing a metric family must fail aggregation loudly rather than be silently averaged away.

---

## Cross-epic integrations

### Consumes: X-07 — `shark run` as the sole execution engine

- **Integration purpose**: Consume `shark run` as the sole execution engine: the `--json` `RunResult`/`StageLog` field set, the documented transcript byte format, and the `--workdir` agent-cwd override, all parsed unmodified by the bench collector.
- **Producer**: E22 — External Orchestration Runner (E22-F08 Simplify RunController to match architecture v2).
- **Contract / shape source**: E40 architecture "Run lifecycle and isolation contract" and "Metric collection and artifact schema"; `internal/runner/controller.go` `RunResult`/`StageLog`; `internal/runner/transcript.go` byte format; E22-F08 owns this surface today — its title names RunController, `T-E22-F08-001/002/003` touch the dispatch functions (`handleAdvanceStatus`/`handleSpawnAgent`) that populate `StageLog`, and `T-E22-F08-007` covers worktree isolation.
- **UX / CX handoff notes**: Not user-facing. Operator-visible only when the canary trips: a bench batch fails loudly with the changed field named instead of publishing silently wrong metrics.
- **Owning feature / status**: E40-F02 Bench harness: run driver and metric collection; `assigned`.
- **Test coverage**: E40 uat-plan.md X-07 canary scenario; UAT-01, UAT-07. Implemented in this feature as `bench/scripts/canary-runsurface.sh` (REQ-F-020, AC-13) — a harness preflight against a real invocation, deliberately not a CI-resident test (ADR-F02-10).

Row copied verbatim from [E40-cross-epic-map.md](../E40-cross-epic-map.md), which mirrors [docs/product/cross-epic-integration-map.md](../../../product/cross-epic-integration-map.md). No value is altered here.

**No other X-## obligation.** F02 produces no X-08 row — that is E40-F04's, with F04 as its own activation owner. X-09 stays `proposed` and Phase 2: it proposes reusing E27-F15's usage decoder to move envelope decoding into `StageLog`, which is G1 work explicitly out of Phase 1 scope, and it is gated on Q003 closing first. F02's harness-side parser is the Phase 1 stand-in, not an X-09 implementation, and the producing branch `E27-F15-cross-session-usage-tracking` remains unmerged.

---

## Durable unresolved decisions

**Q003 — Which envelope field names does the E40-F02 transcript parser depend on?** (`open`, reused; no new Q### minted.)

This feature's research could not resolve Q003 empirically: the only in-repo artifact, `E27-F15-cross-session-usage-tracking`'s `testdata/claude-usage-result.json`, is a hand-authored 7-field unit-test fixture and corroborates none of `modelUsage`, `num_turns`, or `duration_api_ms`. It is not confirmation of anything beyond what that branch's own tests need.

Q003 is carried here as a **requirement with an ordering constraint** and a named closure mechanism, not as an undecided placeholder:

- **Closure mechanism**: REQ-F-021. Capture one real `.shark/runs/<run_id>/<n>-<status>-anthropic.log` from F02's own first live run — naturally available once the driver executes a real run, so no separate `claude` invocation is needed — and confirm the exact presence and spelling of `modelUsage`, `num_turns`, and `duration_api_ms` before the envelope parser is written.
- **Named fallback, decided now**: if `modelUsage` is absent, exact model IDs come from the envelope's **top-level `model` field**, the only other in-fixture candidate. Exact model IDs are required manifest data for G7, so the field is never dropped, and `manifest.model_id_source` records which source was used (ADR-F02-05, REQ-F-021).
- **Failure posture, decided now**: a missing or renamed field is a named parse error on the record, never a zero (REQ-F-012, AC-05).
- **Escalation**: epic PRD §4 states that if the envelope assumption proves false, G1 moves into Phase 1 scope. If the capture shows the usage data is not present in the transcript at all, that is a scope escalation to raise before implementation continues, not a gap to work around.
- **Closure owner**: the task that implements the envelope parser; the confirmed names are recorded in `bench/README.md` (AC-17) and Q003 resolved against it.

**Q004 and B052** (cascade attribution; sibling transcript filename collision) are constrained out of Phase 1 by entity-type scoping (tasks and bugs only, ADR-005), not fixed. Named here so Phase 2 feature-benching planning does not re-derive the scoping. **B053** is F01-scoped and verified not to reach F02's post-run diff path. None of the three is an open decision for this feature.

---

*Last Updated*: 2026-08-06
