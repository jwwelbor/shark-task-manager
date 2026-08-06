---
research_schema: 2
entity_key: E40-F02
entity_type: feature
recipe: universal
rigor: standard
categories:
  - backend
  - data
  - workflow_operations
related_work: true
---

# Research report: Bench harness — run driver and metric collection

## Scope

E40-F02 is Phase 1's run driver: harness scripts (zero shark Go changes) that
provision a scratch shark project plus a harness-owned fixture-repo checkout
at the corpus base SHA (ADR-001), seed corpus entities via the shark CLI,
invoke `timeout <cap> shark run <key> --json --workdir <checkout>`, and
collect four metric sources — `RunResult` stdout, per-stage transcript
envelopes, scratch-DB `work_sessions`/`entity_history`, and post-run checks
in the checkout (F2P/P2P oracle, fmt/vet/lint/test vs. base ledgers, LOC) —
into one JSONL artifact per run (**I-02**). It consumes **I-01** (E40-F01's
corpus/oracle contract) and **I-03** (E40-F04's run-liveness contract, which
**landed today**, 2026-08-06) and inherits **X-07** (E22's `RunResult`/
`StageLog`/transcript surface).

Complexity is pre-scored **STANDARD (13/27)**, recorded as a decision note on
E40-F02, 2026-08-06. This research corroborates that score: F02's remaining
unknowns are field-name and data-shape verification against contracts that
are now largely *built*, not open cross-boundary or alternatives questions —
F01's producer surface (`bench/corpus/`, `bench/scripts/`) is materially
implemented already (Finding 1), and F04's liveness recorder is live code,
not a design doc (Finding 2). The COMPLEX-tier material a wider feature would
need (isolation ADRs, canary strategy, cross-epic seams) already exists
upstream in `architecture.md` and `E40-cross-epic-map.md` and is cited here
rather than re-derived.

Terms used in this report, matching the epic's and siblings' vocabulary:
**envelope** (the raw `claude --output-format json` stdout object, captured
verbatim inside a transcript's `---STDOUT---` block — still undecoded by
`internal/runner/claude_dispatcher.go`), **I-01/I-02/I-03** (the epic's
stable interaction IDs; see `E40-interaction-map.md`), **Q003** (the
envelope's exact field names, unverified — this feature's own obligation),
**Q004** (Phase 2 cascade-attribution constraint from B052), and
**check_or_resume** (unrelated to F02; carried from the epic report for
continuity only).

## Research checklist

- [x] `scope_vocabulary` — Evidence: `feature.md` (Goal, Scope items 1–5,
  Acceptance Criteria, Out of Scope), `epic.md` §3 (Phase 1 scope line for
  F02), and `architecture.md`'s "Run lifecycle and isolation contract",
  "Metric collection and artifact schema", and "Run liveness contract"
  sections define the provision/seed/invoke/collect vocabulary, the I-01/
  I-02/I-03 boundary, and the harness-owned `--workdir` isolation model
  (ADR-001).
- [x] `affected_implementation_or_contract` — Evidence: `internal/runner/controller.go:67-135` (`RunProgress`/`RunResult`/`StageLog` — read directly, not from doc paraphrase, surfacing the previously-undocumented `QuestionBlock` field and the five-value `Outcome` enum, Finding 3); `internal/runner/liveness.go` (the landed I-03 producer — `LivenessRecorder`, NDJSON schema, `run.log` sink, all implemented, not designed); `tests/contracts/e40_i03_liveness_contract_test.go` (`TestTC001_I03LivenessContract`, the shared I-03 contract test F02 must reuse verbatim); `tests/contracts/e40_i01_corpus_contract_test.go` (`TestTC001_I01CorpusAndOracleContract`, the shared I-01 contract test F02 must reuse verbatim); `internal/runner/claude_dispatcher.go` (re-read directly — envelope decode still absent, unchanged from epic research); `bench/corpus/corpus.yaml` + `bench/scripts/{admit,build-ledgers,checkout-fixture,diff-ledgers,verify-clean-checkout}.sh` + `bench/README.md` (F01's already-built I-01 producer surface, materially ahead of what the epic-level report described).
- [x] `related_work` — Evidence: epic-level `research-report.md` (Capability map row on the envelope, Decisions 2/6 on verification and the canary); `E40-F01-.../research-report.md` (I-01 producer-side findings); `E40-F04-.../research-report.md` (I-03 producer-side findings, filed the same day I-03 landed); `E40-interaction-map.md` (I-01/I-02/I-03 rows, "Sequencing note on I-01" and "on I-03"); `E40-cross-epic-map.md` (X-07/X-08/X-09); E40-F02's own decision notes (obligations 2139, 2192; complexity note 2211).
- [x] `pattern_contract` — Evidence: `bench/scripts/checkout-fixture.sh` and `diff-ledgers.sh` (F01's already-built, tested, documented checkout and ledger-diff tooling F02 is obligated to invoke, not re-derive — `bench/README.md`'s "Ledger diff method" section); `bench/scripts/admit.sh:493` (`copy_f2p_files` — the established F2P-injection pattern); `internal/observability/file_jsonl_exporter.go` (fail-soft JSONL append pattern, cited by F04's report, directly reusable for I-02 emission); `scripts/shark-scratch-env.sh` (unchanged scratch-provisioning + live-repo-guardrail pattern).
- [x] `dependency_impact` — Evidence: `internal/db/db.go` (`work_sessions`/`entity_history`, confirmed still entity-type/entity_key generic, no migration needed); `git show E27-F15-cross-session-usage-tracking:testdata/claude-usage-result.json` (confirmed a 7-field hand-authored fixture, not a live capture — corroborates none of `modelUsage`/`num_turns`/`duration_api_ms`); `grep -n "run_selector" bench/scripts/diff-ledgers.sh` (zero hits — B053 scoping finding, below); `docs/plan/bugs/B052.md`, `docs/plan/bugs/B053.md`.

## Capability map

| Capability | Evidence | Decision | F02 responsibility |
|---|---|---|---|
| `shark run` execution engine (`RunController.Run`, `RunResult`/`StageLog`/`RunProgress`) | `internal/runner/controller.go:67-135` | REUSE | Parse stdout `RunResult` unmodified via `--json --workdir`. `Outcome` is one of `completed`/`paused`/`failed`/`already_terminal`/`no_action` (not just completed/failed/timeout), and a `paused` outcome may carry a `QuestionBlock` — see Finding 3. |
| I-03 run-liveness stream + `run.log` (`LivenessRecorder`) | `internal/runner/liveness.go` (landed 2026-08-06); `tests/contracts/e40_i03_liveness_contract_test.go`'s `TestTC001_I03LivenessContract` | REUSE, verbatim | F02's own I-01/I-03-consumes task (decision note 2192) must reuse `TestTC001_I03LivenessContract` as shared contract evidence rather than write a second reader. This is F02's primary source for the stalled-stage attribution UAT-5 needs on a timeout. |
| I-01 corpus/oracle contract (manifest, items, ledgers) | `bench/corpus/corpus.yaml` (10 admitted items, 3 negative items, ledgers pinned at base SHA `4c249868…`); `tests/contracts/e40_i01_corpus_contract_test.go`'s `TestTC001_I01CorpusAndOracleContract` | REUSE, verbatim | F01's producer surface is materially built, not just designed (Finding 1). F02's I-01-consumes task (decision note 2139) must reuse `TestTC001_I01CorpusAndOracleContract` verbatim, read `corpus.yaml` directly, and never re-derive the `p2p_set` resolution rule already documented in `bench/README.md`. |
| Fixture checkout + ledger-diff tooling | `bench/scripts/checkout-fixture.sh`, `diff-ledgers.sh`; `bench/README.md` §"Ledger diff method" | REUSE | F02 invokes these scripts directly for its own fixture checkout and post-run `p2p_regressions`/`lint_new_issues` diff (Finding 1, Decision 6) — it does not reimplement checkout or the diff method. |
| Held-back F2P test injection | `bench/scripts/admit.sh:493` (`copy_f2p_files`) | REUSE, as pattern | F02's own post-run F2P injection step should follow the same `shutil.copy2`-from-corpus-item mechanism, applied at terminal status instead of admission time (Finding 8). |
| Claude JSON usage envelope decode | `internal/runner/claude_dispatcher.go` (confirmed unchanged: appends `--output-format json`, never decodes); branch `E27-F15-cross-session-usage-tracking`'s `testdata/claude-usage-result.json` (7-field hand-authored fixture, no `modelUsage`/`num_turns`/`duration_api_ms`) | EXTEND, with an unresolved field-name gap (Q003) | This research could not empirically resolve Q003 (Finding 5) — see Decisions 2–4 for the scoped verification obligation this leaves for spec/test_planning. |
| `work_sessions` / `entity_history` | `internal/db/db.go` (entity-generic schema, unchanged) | REUSE | No migration; F02 queries directly for per-child time windows/outcomes and the backward-transition cross-check. |
| Scratch-project provisioning + live-repo guardrail | `scripts/shark-scratch-env.sh` (re-read, unchanged) | REUSE | Scratch `admin init` still defaults to the embedded workflow bundle — F02 must still explicitly repoint `workflow_config` at the variant after provisioning. |
| JSONL fail-soft append pattern | `internal/observability/file_jsonl_exporter.go` | REUSE, as pattern | F02's I-02 per-run JSONL emission can follow the same open-append-encode-swallow-errors convention already used by this file and by F04's `run.log` sink, rather than inventing a third JSONL-writing convention. |
| `internal/reporting` (`ScanReport`) | Parent epic report Capability map row | CONTRADICTS | Reconfirmed out of scope at feature level — F02 emits raw JSONL records, not a `ScanReport`. |
| B053 (`admit.sh` `run_selector` defect) | `bench/scripts/diff-ledgers.sh` — `grep -n "run_selector"` returns zero hits | CONTRADICTS (not applicable) | Verified directly: `diff-ledgers.sh --kind=test` is a flat base-vs-post ledger identity comparison with no `p2p_set`/`run_selector` filtering. B053 is scoped to `admit.sh`'s per-item admission-time resolution (F01), not F02's post-run diff path (Finding 6). |
| B052 (cascading transcript/StageLog collision) | `docs/plan/bugs/B052.md` (status `draft`); `architecture.md` ADR-005 | Constrained out by phasing, not fixed | Phase 1 benches tasks and bugs only (ADR-005/Q004), so F02 should not need cascade-child attribution this phase. The bug remains unfixed and is a named Phase 2 feature-benching risk (Finding 7), not a Phase 1 blocker. |

## Findings

1. **F01's producer surface is materially built, not just designed.**
   `bench/corpus/corpus.yaml` (10 admitted items, 3 negative items, ledgers
   pinned at base SHA `4c249868…`) and `bench/scripts/{admit,build-ledgers,
   checkout-fixture,diff-ledgers,verify-clean-checkout}.sh` all exist,
   are documented in `bench/README.md`, and carry their own test suite
   (`bench/scripts/tests/`). This changes several capability-map rows from
   "F02 will need to build X" to "F02 invokes X" — most concretely
   `checkout-fixture.sh` and `diff-ledgers.sh`, which F02's own decision note
   (2139) already obligates it to call verbatim rather than re-derive.

2. **I-03 landed today (2026-08-06).** `internal/runner/liveness.go`
   implements the full `LivenessRecorder` — stage_start/heartbeat/stage_end
   NDJSON on stderr in `--json` mode, plain text on stderr otherwise, an
   unconditional `run.log` file sink, a fixed 10s heartbeat, and a
   `Start()`/`Stop()`/`Finish()` lifecycle — and is wired into
   `internal/cli/commands/run.go` (`rec := runner.NewLivenessRecorder(...)`
   at line 157, `opts.Progress = rec.Observe` at line 346,
   `defer func() { rec.Stop(); rec.Finish(runResult) }()` at line 159).
   `tests/contracts/e40_i03_liveness_contract_test.go`'s
   `TestTC001_I03LivenessContract` is the shared contract evidence F02's own
   I-03-consumes task must reuse verbatim per decision note 2192 — not
   re-derive.

3. **`RunResult`'s outcome taxonomy is wider than every upstream E40 document
   states.** `controller.go:97-98` documents `Outcome` as one of
   `completed`/`paused`/`failed`/`already_terminal`/`no_action`, and
   `controller.go:105-107` adds an optional `QuestionBlock` field — "the
   compact I-03 handoff when a directly linked open blocking Question pauses
   this run before dispatch work." Neither `epic.md`, `architecture.md`,
   `shark-bench-design.md`, nor F02's own `feature.md` mentions `paused` or
   `QuestionBlock` as a possible run outcome; feature.md's Scope item 3
   speaks only of "timeout-as-outcome." A collector that maps every
   non-`completed` outcome to a bench failure would misclassify a
   Question-blocked run as a defect.

4. **The claude JSON envelope remains genuinely undecoded, confirmed
   unchanged since epic-level research.** A fresh direct read of
   `internal/runner/claude_dispatcher.go` shows it still only appends
   `--output-format json` and returns raw stdout — no diff from the epic
   report's Finding 2. The only related artifact, branch
   `E27-F15-cross-session-usage-tracking`'s `testdata/claude-usage-result.json`
   (confirmed via `git show`), is a **hand-authored unit-test fixture** — 7
   fields (`type`, `session_id`, `request_id`, `result`, `model`,
   `total_cost_usd`, `usage{input_tokens, cache_read_input_tokens,
   cache_creation_input_tokens, output_tokens}`) — not a captured live
   transcript, and it still corroborates none of `modelUsage`, `num_turns`,
   or `duration_api_ms`.

5. **This research could not empirically resolve Q003.** The `claude` CLI
   (v2.1.223) is installed and reachable on PATH in this environment, but
   this research task's parent-loop ownership contract prohibits spawning
   external AI CLIs from a dispatched research step unless the workflow
   prompt explicitly requests it, which it does not for a live envelope
   capture. Decisions 2–4 below scope the concrete verification obligation
   this leaves for F02's spec/test_planning steps instead, per the dispatch
   context's own instruction to "resolve Q003 ... or scope precisely what
   spec/test_planning must pin down."

6. **B053 does not propagate into F02's own post-run check path.** Direct
   inspection of `bench/scripts/diff-ledgers.sh` (the script F02 is
   obligated to invoke for `p2p_regressions`/`lint_new_issues`) shows it
   never references `run_selector` or `p2p_set` at all — `grep -n
   "run_selector" bench/scripts/diff-ledgers.sh` returns zero hits. Its
   `--kind=test` diff is a flat base-vs-post ledger identity
   (`<package>::<test>`) comparison, exactly as `bench/README.md`'s "Ledger
   diff method" section describes. B053's defect is confined to
   `admit.sh`'s per-item, per-`p2p_set` admission-time resolution (F01's
   corpus-build-time concern) and does not reach F02's full-suite post-run
   diff.

7. **B052 is out of Phase 1's reach by entity-type scoping, not by a fix.**
   The epic's own constraint (ADR-005/Q004: Phase 1 benches tasks and bugs
   only) means F02 should not need cascade-child transcript/`StageLog`
   attribution this phase. B052 itself remains `status: draft` with no fix
   notes — it is constrained out, not resolved — and stays a named risk for
   Phase 2 feature-benching planning, consistent with the epic report's own
   framing of Q004.

8. **`bench/scripts/admit.sh:493`'s `copy_f2p_files` is a directly reusable
   pattern for F02's own F2P-injection step.** It copies each item's
   `f2p.paths` file(s) via `shutil.copy2` into the matching package
   directory inside a checkout, validating the package/test-name pairing
   from `f2p.test_names` first. F02 performs the same copy at terminal
   status instead of admission time (`shark-bench-design.md`'s "inject
   held-back F2P tests + run" mechanic) — same mechanism, different trigger
   point.

9. **`work_sessions`/`entity_history` are unchanged and confirmed
   entity-generic.** `internal/db/db.go` still carries `entity_type`/
   `entity_key` columns on both tables (migrated under the schema history's
   "E36 metrics: rebuild work_sessions entity-generic" note) — no migration
   needed for F02's cross-check queries, corroborating the epic and F01
   reports' framing.

10. **The dispatch prompt's named catalog path does not exist in this
    repo.** `shark-data/research/recipes.yaml` is absent;
    `internal/sharkdata/default_data/research/recipes.yaml` is the actual
    canonical v2 catalog read for this report — the same discrepancy F04's
    report already documented. Per project memory, the embedded copy is
    canonical and `shark-data/` is a deployed, overwritable copy.

## Decisions

1. **Rigor stays STANDARD (13/27).** Nothing found argues the tier is
   wrong. F02's remaining unknowns are field-name and outcome-shape
   verification against contracts that are now largely built (I-01, I-03),
   not open cross-boundary or alternatives questions — the COMPLEX-tier
   material a wider feature would need already exists upstream in
   `architecture.md` (ADR-001/004/005) and `E40-cross-epic-map.md` and is
   cited here, not re-derived.

2. **Scope Q003's verification precisely for F02's spec/test_planning
   steps**, since this research could not resolve it empirically (Finding
   5): before writing the envelope parser, capture one real
   `.shark/runs/<run_id>/<n>-<status>-anthropic.log` transcript from an
   actual dispatched claude stage — naturally available once F02's own run
   driver executes real runs, so no separate live `claude` invocation is
   needed at spec time — and confirm the presence and exact names of
   `modelUsage`, `num_turns`, and `duration_api_ms` specifically, the three
   fields with zero in-repo corroboration. The seven fields
   `E27-F15`'s fixture decodes are a hand-authored unit-test fixture, not
   live-capture evidence, and must not be treated as confirmation of
   anything beyond what that branch's own tests need.

3. **Name the exact-model-ID fallback now, as a spec obligation.** If
   `modelUsage` is absent from a captured envelope, the parser's fallback
   source is the envelope's top-level `model` field — the only other
   in-fixture candidate. `architecture.md` already requires a fallback be
   named rather than the field silently dropped, and exact model IDs are
   required manifest data for G7 reproducibility; F02's spec should state
   this fallback explicitly rather than leave it as an implementation-time
   surprise.

4. **Specify a fail-loud parse posture for missing/renamed envelope
   fields**, consistent with the epic's ADR-004 canary stance: a stage
   transcript whose envelope is missing an expected field should produce a
   named, visible parse error attached to that run's JSONL record, not a
   silently-zeroed metric. F02's spec should state this as an explicit
   acceptance criterion.

5. **Model `RunResult.Outcome` and `QuestionBlock` correctly in F02's
   collection logic** (Finding 3): the collector's rollup must handle five
   outcomes (`completed`/`paused`/`failed`/`already_terminal`/`no_action`),
   not the three feature.md's Scope item 3 implies (completed/failed/
   timeout), and must surface `QuestionBlock` when present rather than
   folding a Question-blocked pause into a generic "failed" bucket
   indistinguishable from a genuine defect or a harness timeout.

6. **Invoke `checkout-fixture.sh` and `diff-ledgers.sh` directly; do not
   re-derive either** (Finding 1, reaffirming decision note 2139's
   obligation) — both are built, tested, and documented, and a second,
   divergent implementation of either would fork an already-owned contract.

7. **Reuse `admit.sh:493`'s `copy_f2p_files` pattern for F02's own
   F2P-injection step** (Finding 8) — same copy mechanism, invoked at
   terminal status instead of admission time.

8. **Carry both consumer-side mirror obligations into F02's
   `task_generation` exactly as recorded on E40-F02's decision notes**: an
   I-01-consumes task copying `architecture.md#corpus-and-oracle-contract`
   and `tests/contracts/e40_i01_corpus_contract_test.go#TC-001` verbatim
   (note 2139), and an I-03-consumes task copying the shape source and
   contract-test pointer from F04's `spec.md` "Cross-feature interactions"
   section verbatim, gate mode `live` (note 2192). This research does not
   discharge either obligation — that is `task_generation`'s job — but
   confirms both producer sides (I-01, I-03) are now real, inspectable
   artifacts rather than forward-looking placeholders.

9. **Track B053 as F01-scoped (does not block F02) and B052 as
   phased-out-of-reach for Phase 1 but a named Phase 2 risk** (Findings 6,
   7). Neither requires action inside F02's own spec, but both should be
   named explicitly there so a future Phase 2 feature-benching planner does
   not need to re-derive this scoping.

## Sources

- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F02-bench-harness-run-driver-and-metric-collection/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/epic.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md` (Run lifecycle and isolation contract; Metric collection and artifact schema; Run liveness contract; ADR-001/002/003/004/005; Delivery boundaries and traceability)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/shark-bench-design.md` (§1 architecture, §4 per-metric mechanics)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-interaction-map.md` (I-01/I-02/I-03 rows, sequencing notes)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-cross-epic-map.md` (X-07/X-08/X-09)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md` (epic-level report)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F01-benchmark-corpus-v1-fixture-repo-and-screened-task/research-report.md` (sibling)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F04-shark-run-live-progress-and-per-run-log/research-report.md` (sibling)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F03-baseline-report-and-noise-band/feature.md`
- `internal/sharkdata/default_data/research/recipes.yaml` (the actual v2 catalog read; `shark-data/research/recipes.yaml` named in the dispatch prompt does not exist — same discrepancy F04's report documented)
- `internal/runner/controller.go` (`RunProgress`/`RunResult`/`StageLog`, lines 67-135; `Outcome` values and `QuestionBlock`)
- `internal/runner/liveness.go` (landed `LivenessRecorder` — I-03 producer)
- `internal/runner/claude_dispatcher.go` (confirmed undecoded envelope, re-read)
- `internal/runner/transcript.go` (on-disk transcript byte format, unchanged)
- `internal/cli/commands/run.go` (`LivenessRecorder` wiring at lines 157-159, 346)
- `internal/db/db.go` (`work_sessions`/`entity_history` schema)
- `internal/observability/file_jsonl_exporter.go` (JSONL fail-soft append pattern)
- `scripts/shark-scratch-env.sh`
- `tests/contracts/e40_i03_liveness_contract_test.go` (`TestTC001_I03LivenessContract`, `TestTC002_X08StdoutPurity`)
- `tests/contracts/e40_i01_corpus_contract_test.go` (`TestTC001_I01CorpusAndOracleContract`, `TestTC002_I01FixturePackageVisibilityContract`)
- `bench/README.md`
- `bench/corpus/corpus.yaml`, `bench/corpus/ledgers/4c24986844b09122e2d516f9bc1ec470b155b441/{tests,lint}.json`
- `bench/scripts/admit.sh` (`copy_f2p_files` at line 493)
- `bench/scripts/checkout-fixture.sh`, `bench/scripts/diff-ledgers.sh` (`grep -n "run_selector"` — zero hits, B053 scoping)
- `docs/plan/bugs/B052.md`, `docs/plan/bugs/B053.md`
- `git show E27-F15-cross-session-usage-tracking:testdata/claude-usage-result.json` and `git grep claudeJSONResult E27-F15-cross-session-usage-tracking -- '*.go'` (confirms the fixture's field set and its hand-authored nature)
- `shark feature get E40-F02 --json` / `shark feature notes E40-F02` (decision notes 2139, 2192, 2211)
- `shark feature get E40-F01 --json`, `shark feature get E40-F03 --json` (sibling status/order)

RECOMMENDED OUTCOME: standard
