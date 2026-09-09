---
research_schema: 2
rigor: standard
categories:
  - backend
  - data
  - workflow_operations
  - documentation
related_work: true
---

# Research report: Live I-05 evidence bundle producer for lifecycle-v2 runs

## Scope

E40-F12 closes a spec-versus-implementation gap in an already-shipped
feature: `bench/scripts/run-lifecycle.sh` (E40-F08's host-side lifecycle
controller) computes candidate identity and folds bounded `evidence_refs`
snippets into the I-07 JSONL on every dispatch, but never materializes a
real I-05 bundle (`bundle.json`, `stages/*.json`, `access.jsonl`) at the
`i05_bundle_dir` path its own downstream consumers already expect. The
scope is bounded to: (1) building that producer inside `run-lifecycle.sh`'s
existing per-dispatch loop, reusing the identity/digest data it already
computes; (2) conforming the output exactly to E40-F06's already-shipped
I-05 schema (`bench/evidence/i05-schema.yaml`, `bench/README.md`'s "I-05
stage evidence and isolation contract" section) with no new vocabulary; and
(3) closing the two already-filed, still-open findings this unblocks —
E40-F10 code-review finding TC-082 (`retain_pair()`'s evidence copy) and
TD-132 (`content_root` never populated, so `evaluate-lifecycle.sh`'s
content-digest crosscheck never fires). Out of scope: any change to the
I-05 schema itself, to `verify-stage-evidence.sh`'s validation rules, or to
`retain_pair`'s own copy/fail-closed logic (both already correct — see
Findings).

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F12-live-i-05-evidence-bundle-producer-for-lifecycle-v/feature.md` (Problem/Solution/Impact sections define `i05_bundle_dir`, `bundle.json`/`stages/*.json`/`access.jsonl`, TC-082, and TD-132 as the bounded vocabulary).
- [x] `affected_implementation_or_contract` — Evidence: `bench/scripts/run-lifecycle.sh` — `candidate_identity()` (lines 111-127), `refresh_candidate()` (lines 162-167), `scenario_identity()` (lines 241-256), and the per-dispatch loop building `dispatch["evidence_refs"]` and `stage_record()` (lines 794-871) are the exact functions this feature extends; `bench/evidence/i05-schema.yaml` is the closed-vocabulary contract the new writer output must satisfy.
- [x] `related_work` — Evidence: parent `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md` (`RECOMMENDED OUTCOME: pass`) and `E40-interaction-map.md`'s I-05 row (E40-F06 producer; E40-F08/F09/F10 consumers); sibling `E40-F06-stage-evidence-and-evaluator-isolation/research-report.md` ("F06 defines and validates the schema... E40-F08 populates real bundles during a real lifecycle run"); sibling `E40-F08-canonical-multi-entity-lifecycle-runner/spec.md` REQ-F-010 (the runner "MUST write or reference the I-05 stage snapshot..." — the requirement this feature is closing, not new scope); sibling `E40-F10-operator-workflow-and-retained-lifecycle-baseline/test-plan.md` TC-082 and `docs/plan/tech-debt/TD-132.md`.
- [x] `pattern_contract` — Evidence: `bench/README.md` lines 1142-1228 (I-05 bundle-layout table, stage-snapshot field reference, `verify-stage-evidence.sh` as the single named validation owner) and `bench/evidence/i05-schema.yaml` (the closed `stage_category`/`interval_category`/`artifact_type`/`edge_kind`/`evaluator_access_phase`/`stop_outcome`/`error_kind` vocabularies) are the established contract this producer's output must match byte-for-byte — no parallel schema, no new fields.
- [x] `dependency_impact` — Evidence: `bench/scripts/run-lifecycle-batch.sh` lines 902-913 (`if [[ -z "$i05_bundle_dir" ]]; then ... recorded failed`, then `"$EVALUATE_LIFECYCLE_BIN" --i05 "$i05_bundle_dir" --i07 "$lifecycle_out"`) and line 949 (`retain_pair ... "$i05_bundle_dir"`) are the live, already-wired direct callers waiting on this producer's output; `bench/scripts/lib/retain_pair` lines 264-381 (`copy_dir_artifact("evidence", ...)` / `copy_dir_artifact("transcripts", ...)`) is the direct downstream consumer of the bundle directory's on-disk shape.

## Capability map

| Capability | Brownfield evidence | Decision | E40-F12 responsibility |
|---|---|---|---|
| I-05 bundle producer inside `run-lifecycle.sh`'s per-dispatch loop | `bench/scripts/run-lifecycle.sh` `candidate_identity()`/`refresh_candidate()`/`stage_record()` (lines 111-127, 162-167, 295-305) compute candidate identity and fold bounded `evidence_refs` into I-07 only; no `bundle.json`/`stages/*.json`/`access.jsonl` writer exists anywhere under `bench/scripts/` — every I-05-related test (`tc044` through `tc057`, `tc075`, etc.) hand-builds its own synthetic bundle fixture inline, confirming there is no shared writer to extend | NEW | Build the producer inside the existing dispatch loop, serializing the identity/digest data the loop already computes into real `bundle.json` + `stages/<dispatch_ordinal>-<stage_key>.json` + `access.jsonl` files at `i05_bundle_dir`. |
| I-05 schema and validator (`bundle.json`/`stages/`/`access.jsonl` shape, `verify-stage-evidence.sh`) | `E40-F06-stage-evidence-and-evaluator-isolation/research-report.md`; `bench/evidence/i05-schema.yaml`; `bench/README.md` lines 1142-1228 | REUSE | Producer output must conform exactly to F06's already-shipped schema and pass `verify-stage-evidence.sh` unmodified — no new vocabulary, no parallel validator. |
| `retain_pair`'s evidence/transcripts copy (TC-082) | `bench/scripts/lib/retain_pair` docstring (UAT-R3-01: already refuses to write any artifact and exits nonzero, no `manifest.json`, when a required source is absent — the "empty dir satisfies every check" defect is already closed); `copy_dir_artifact("evidence", i05_bundle_dir, exclude_top_level=("transcripts",))` and `copy_dir_artifact("transcripts", i05_bundle_dir + "/transcripts")` (lines 380-381) | EXTEND (retention side already hardened; needs a real source, not new logic) | Once the producer writes `bundle.json`/`stages/`/`access.jsonl` plus a `transcripts/` subdirectory inside `i05_bundle_dir`, `retain_pair`'s existing copy logic starts succeeding for real — no change to `retain_pair` itself is in scope. |
| `evaluate-lifecycle.sh`'s content-digest crosscheck (TD-132) | `docs/plan/tech-debt/TD-132.md`; `bench/scripts/evaluate-lifecycle.sh` lines 275-278 (`declared_content_root` / walk-based `content_digest()`); `bench/scripts/run-lifecycle.sh` line 255 (`shark_content_digest` = whole-repo `git rev-parse HEAD^{tree}` hash, no `content_root` ever emitted) | CONTRADICTS (two incompatible digest schemes) | TD-132.md names this an open design decision — what `content_root` should point at (installed `internal/sharkdata/default_data` bundle vs. an installed scratch-project copy) and which digest scheme wins — not a mechanical field fill-in; the spec step must record the choice explicitly. |
| Preflight's `scenario_roots.<id>.i05_bundle_dir` configuration (P5) | `feature.md` Problem section, 2026-09-04 preflight run: every scenario blocked on `scenario_roots.<id>.i05_bundle_dir is not configured` | NEW | A real, per-scenario `i05_bundle_dir` path must be threaded from `run-lifecycle-batch.sh`'s `dispatch_pair()` into `run-lifecycle.sh`'s dispatch loop before the preflight gate can pass. |

## Findings

1. This is a spec-versus-shipped-implementation gap in an already-completed
   feature, not newly invented scope: `E40-F08-canonical-multi-entity-lifecycle-runner/spec.md`
   REQ-F-010 already required "After each applicable dispatch, the runner
   MUST write or reference the I-05 stage snapshot, time ledger, candidate
   snapshot, artifact producer/consumer events, provider usage, and
   evaluator-access ordering" — `run-lifecycle.sh` was spec'd to do this and
   never did.

2. No shared bundle-writer helper exists anywhere under `bench/scripts/`.
   Every I-05-related test script (`tc044_time_ledger_reconciliation_test.sh`
   through `tc057_non_applicable_record_test.sh`, `tc075_content-identity_x12_test.sh`,
   and others) hand-builds its own synthetic `bundle.json`/stage-snapshot
   fixture independently. This confirms the producer is NEW capability, not
   an extension of an existing writer.

3. `run-lifecycle.sh` already computes every field a stage snapshot's
   `candidate` block needs. `candidate_identity()` (lines 111-127) derives
   `base_commit`/`tree_digest`/`binary_diff_digest`/`changed_path_digest`/
   `dirty_untracked_manifest`/`test_suite_digest`; `refresh_candidate()`
   (lines 162-167) adds `scratch_content_digest`; the dispatch loop (lines
   794-871) already threads `prompt_sha256`/`candidate_snapshot_digest`
   through `dispatch["evidence_refs"]` per dispatch. The producer's job is
   to serialize this already-computed data into the I-05 file layout, not
   to derive new identity data.

4. TC-082 is already half-fixed. `lib/retain_pair`'s own docstring records
   that a prior UAT round (UAT-R3-01) closed the "an empty dir satisfies
   every downstream check" defect: the script now refuses to write ANY of a
   pair's artifacts and exits nonzero without writing `manifest.json` when a
   required source (including `evidence/`) does not exist on disk. What
   remains open is only that no producer ever populates `i05_bundle_dir` for
   `retain_pair` to copy from — the retention-side hardening itself needs no
   further change.

5. TD-132 is a design decision, not a one-line wiring fix. `TD-132.md`
   states the two digest schemes are structurally incompatible:
   `run-lifecycle.sh`'s `shark_content_digest` is a whole-repo git-tree hash
   (`sha256(git rev-parse HEAD^{tree})`, line 255), while
   `evaluate-lifecycle.sh`'s `content_digest()` (invoked at line 277) walks
   a directory and hashes file contents. Reconciling them requires deciding
   what `content_root` should point at (the installed
   `internal/sharkdata/default_data` bundle vs. an installed copy in the
   scratch project) and switching one producer's scheme to match the other,
   or building an explicit two-scheme compatibility path.

6. `retain_pair`'s `copy_dir_artifact` calls (lines 380-381) expect
   `i05_bundle_dir` to contain a `transcripts/` subdirectory in addition to
   the `bundle.json`/`stages/`/`access.jsonl` layout `bench/README.md`'s I-05
   section documents. A producer that writes only the three documented
   top-level artifacts will make `retain_pair`'s "transcripts" artifact copy
   fail. This undocumented but load-bearing directory-shape requirement is
   only visible by reading `retain_pair` directly, not from the I-05 bundle-
   layout table in `bench/README.md`.

7. `run-lifecycle-batch.sh`'s `dispatch_pair()`/`retain_pair()` call sites
   (lines 823-826, 902-913, 949) are live, already-wired direct consumers
   waiting on this producer today: `evaluate-lifecycle.sh --i05
   "$i05_bundle_dir"` (line 913) and `retain_pair`'s evidence/transcripts
   copy currently only ever receive an empty or nonexistent directory,
   which is exactly why the 2026-09-04 preflight (feature.md) reports every
   scenario blocked and `run-lifecycle-batch.sh:902-905` treats a missing
   bundle as fatal per pair.

## Decisions

1. **Treat the producer as NEW capability**, built inside
   `run-lifecycle.sh`'s existing per-dispatch loop using the identity/digest
   data it already computes (Finding 3), rather than introducing a separate
   bundle-writer library — there is no existing writer to extend or
   duplicate (Finding 2).

2. **Reuse the I-05 schema exactly** (`bench/evidence/i05-schema.yaml`,
   `bench/README.md`'s bundle-layout table) — no new fields, no parallel
   validator. The producer's acceptance bar is `verify-stage-evidence.sh`
   passing on its real output, the same guard F06/F08/F09/F10 already share.

3. **Include a `transcripts/` subdirectory inside `i05_bundle_dir`**
   (Finding 6) — omitting it will make `retain_pair`'s existing
   `copy_dir_artifact("transcripts", ...)` call fail even though nothing in
   the documented I-05 bundle-layout table mentions it.

4. **Defer the TD-132 digest-scheme reconciliation to an explicit design
   decision in the spec step** (Finding 5) — do not silently pick a
   `content_root` target without recording the choice; feature.md's "wire
   the resulting bundle through... so the crosscheck fires" understates the
   design work this requires.

5. **No change to `lib/retain_pair` itself is in scope** (Finding 4) — its
   fail-closed behavior is already correct; this feature only needs to give
   it a real source directory to copy from.

RECOMMENDED OUTCOME: standard

## Sources

- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F12-live-i-05-evidence-bundle-producer-for-lifecycle-v/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-interaction-map.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F06-stage-evidence-and-evaluator-isolation/research-report.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F08-canonical-multi-entity-lifecycle-runner/spec.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F10-operator-workflow-and-retained-lifecycle-baseline/test-plan.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F11-six-family-lifecycle-benchmark-readiness-and-cover/research-report.md`
- `docs/plan/tech-debt/TD-132.md`
- `bench/README.md` (I-05 stage evidence and isolation contract; retention layout sections)
- `bench/evidence/i05-schema.yaml`
- `bench/scripts/run-lifecycle.sh`
- `bench/scripts/run-lifecycle-batch.sh`
- `bench/scripts/evaluate-lifecycle.sh`
- `bench/scripts/lib/retain_pair`
- `internal/research/validator.go` (research-report schema/validator confirming standard-tier module requirements)
