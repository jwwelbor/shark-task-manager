---
research_schema: 2
entity_key: E40-F01
entity_type: feature
recipe: universal
rigor: standard
categories:
  - backend
  - data
  - documentation
related_work: true
---

# Research report: Benchmark corpus v1 — fixture repo and screened tasks

## Scope

E40-F01 builds the two artifacts every other Phase 1 feature depends on: a
pinned, self-contained Go fixture repo with a real test suite, and a corpus
manifest of ~10 admitted tasks/bugs (issue-style prompt, entity seed spec,
held-back FAIL_TO_PASS tests, PASS_TO_PASS set identifier, reference patch),
plus base-SHA test and lint ledgers. It is the sole producer of **I-01**
(`docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-interaction-map.md`),
whose exact shape is already pinned by the epic's own architecture document,
not something F01 is free to redesign.

F01 does **not** touch shark's Go code — it is one of the three Phase 1
features (F01–F03) the epic constrains to zero Go changes (only F04 changes
`internal/`). Its output is wholly new tooling and data living outside
`internal/`: a fixture repo (location TBD — separate repo or vendored
directory, explicitly deferred to implementation by `feature.md`), a manifest
format, an admission-gate script, and base-SHA ledgers. F01 does not run
`shark run`, install variant bundles, or seed entities via the shark CLI —
that is F02's job (`E40-F02-.../feature.md` §Scope item 2); F01 only produces
the material F02 consumes.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F01-benchmark-corpus-v1-fixture-repo-and-screened-task/feature.md` (Scope items 1–5, Acceptance Criteria, Out of Scope) and `shark-bench-design.md` §2 (Corpus & oracles) and §3 (per-entity oracle matrix) define fixture repo, corpus manifest, admission gate, held-back tests, and base-SHA ledger vocabulary and the Phase 1 (tasks/bugs only) boundary.
- [x] `affected_implementation_or_contract` — Evidence: `architecture.md#corpus-and-oracle-contract` (the exact I-01 payload shape F01 must produce: issue prompt, entity seed spec, held-back F2P files, P2P set id, reference patch, base-SHA test+lint ledgers); `Makefile` lines 86–121 (`test`/`lint`/`fmt`/`vet` targets — the command sequence base-SHA ledgers must mirror against the fixture repo); `.golangci.yml` (linter config shape and version pin, `v2.9.0` per `Makefile:108`) — the kind of config the fixture repo needs its own copy of for `lint_new_issues` to be meaningful.
- [x] `related_work` — Evidence: `architecture.md#corpus-and-oracle-contract` (the I-01 shape F01 must produce) and its Delivery boundaries table (F01 row: "Curator runs the admission gate on a candidate item" → I-01); `E40-interaction-map.md` (I-01 row: producer F01, consumer F02); parent research report `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md` Decision 1 ("Proceed as designed... F01–F03 require zero Go changes") for epic-level backing; sibling `E40-F02-.../feature.md` ("Consumes: I-01, I-03"); sibling `E40-F03-.../feature.md` (G7/UAT-07 replay, which re-invokes F02 against a stored manifest and therefore reads F01's ledgers transitively).
- [x] `pattern_contract` — Evidence: `CLAUDE.md` "Mandatory Quality Gate" (`make fmt && make lint && make test`) and `Makefile:86-121` establish this project's own quality-gate sequence as a local, established pattern; `.github/workflows/ci.yml:121-124` (plain `golangci-lint-action`, no new-issues-only diffing) shows that pattern stops short of what F01 needs — there is no in-repo precedent for diffing lint output to count only *new* issues, so that diff logic is new, not reused. `test-fixtures/uat-test-project/` (git-tracked, `README.md`-documented top-level fixture directory) is the closest existing convention for committing a fixture artifact into this repo, if F01 chooses "vendored directory" over "separate repo."
- [x] `dependency_impact` — Evidence: `E40-interaction-map.md` I-01 row (F01 → F02, "File artifact" style, `architecture.md#corpus-and-oracle-contract` as shape source); `E40-F02-.../feature.md` description ("Consumes: I-01, I-03") and Scope item 4 (post-run checks read the held-back tests and base-SHA ledgers F01 produces); `E40-F03-.../feature.md` Goal paragraph (G7/UAT-07 replay re-invokes F02's single-run command against a stored manifest, so F01's ledger format is a second-order dependency for F03 as well as a first-order one for F02).

## Capability map

| Capability | Evidence | Decision | F01 responsibility |
|---|---|---|---|
| I-01 corpus/oracle contract (fixture repo, manifest, admission gate, held-back tests, base-SHA ledgers) | `architecture.md#corpus-and-oracle-contract` (shape pinned at epic-design time); `E40-interaction-map.md` I-01 row (producer F01, consumer F02); Finding 2 below (zero existing hits for F2P/PASS_TO_PASS/admission-gate outside `docs/plan/E40-*` — nothing to reuse) | **NEW** | F01 creates this capability from nothing; there is no fixture repo, manifest, gate, or ledger anywhere in the repo today. It is not free-form design, though: the shape (fields, held-back-test placement, ledger contents) is pinned by `architecture.md`, so F01 implements rather than invents it. Downstream constraints inherited as build requirements: `E40-F02-.../feature.md` ("Consumes: I-01, I-03") requires the manifest be machine-readable, one documented format, and sufficient to seed/score with no manual step; `E40-F03-.../feature.md` (G7/UAT-07 replay) requires the ledgers stay stable and reproducible on a clean checkout, since replay reaches them transitively through F02. |
| Quality-gate command sequence (`make fmt`/`vet`/`lint`/`test`) | `CLAUDE.md` Mandatory Quality Gate; `Makefile:86-121` | REUSE (pattern, applied to the fixture repo) | Base-SHA ledgers must invoke the equivalent fmt/vet/lint/test sequence inside the fixture repo's own Makefile/tooling — not a bespoke check sequence. |
| "New lint issues only" diffing | `.github/workflows/ci.yml:121-124` (plain golangci-lint-action, no new-issue filtering); design doc `shark-bench-design.md` §4 (`lint_new_issues` diffed against base-SHA ledger) | NEW | No in-repo precedent exists for this diff. F01 must build and commit the base-SHA lint ledger and document the diff method F02 will apply post-run; this is new logic, not reused. |
| Top-level committed fixture directory convention | `test-fixtures/uat-test-project/` (git-tracked, README-documented, used for UAT/demo of shark projects) | EXTEND, conditionally | Different domain (a shark *project* fixture, not a benched Go *service* fixture) — not directly reusable structure, but if F01 chooses "vendored directory" over "separate repo," this is the established local convention (git-tracked, README at the fixture root) to follow rather than inventing a new one. |
| `internal/sharkdata/default_data/manifest.yaml` naming precedent | `internal/sharkdata/default_data/manifest.yaml` (declarative bundle-structure manifest for a different subsystem) | CONTRADICTS (not reusable) | Same word ("manifest"), unrelated schema and purpose (prompt/skill bundle validation, not corpus data) — informative only that "manifest" is an existing vocabulary word in this codebase, not a structure to reuse. |
| `internal/reporting` (`ScanReport`) | Parent research report Capability map row (CONTRADICTS — not applicable, F03's concern) | CONTRADICTS | Confirmed out of scope for F01 as well — corpus/ledger artifacts are plain files, not `ScanReport` output. |

## Findings

1. **F01 implements an already-locked contract, not an open design.** `architecture.md#corpus-and-oracle-contract` fully specifies I-01's payload (issue prompt, entity seed spec with bug severity/repro test, held-back F2P files, P2P set id, reference patch, base-SHA test+lint ledgers) and states admission semantics (F2P red + P2P green at base commit; both green after the reference patch). F01's own `feature.md` restates this identically. There is no daylight between the two documents to resolve during F01 implementation — the only genuinely open choice `feature.md` leaves is the fixture repo's location.

2. **No in-repo vocabulary or tooling exists for F2P/P2P/admission-gate/oracle concepts.** A search across this repo's Go and Markdown sources for `F2P`, `FAIL_TO_PASS`, `PASS_TO_PASS`, and "admission gate" outside `docs/plan/E40-*` returns nothing. This is consistent with the parent research report's framing of these as terms borrowed from SWE-bench/Aider methodology (`shark-bench-design.md` §2), not prior art in this codebase — F01 is greenfield tooling, and there is no existing internal pattern to conform to for the oracle mechanics themselves (only for the quality-gate commands underneath them, see Finding 3).

3. **The project's own quality-gate sequence is the one true local pattern F01's ledgers must mirror, and it stops short of what F01 needs.** `CLAUDE.md`'s Mandatory Quality Gate and `Makefile:86-121` establish `fmt`/`vet`/`lint`/`test` as this repo's own gate, and `.github/workflows/ci.yml` runs `golangci-lint-action` plainly — with no "new issues only" diffing. F01's base-SHA lint ledger (feeding `lint_new_issues` in F02, per `shark-bench-design.md` §4) has no existing diff logic to reuse anywhere in this repo; it must be built new, and documented well enough that F02 can consume it without re-deriving the diff method.

4. **The fixture-repo location decision has one directly relevant local precedent, of limited transferability.** `test-fixtures/uat-test-project/` is a git-tracked, `README.md`-documented directory at repo root, used for a different purpose (a pre-seeded shark project for UAT, not a benched Go service). If F01 chooses "vendored directory" over "separate repo" (`feature.md` Scope item 1 defers this explicitly), this is the closest existing convention for how a committed fixture directory in this repo should be structured and documented — but it is not a reusable schema, since its content is shark project state, not a Go service with a test suite.

5. **F01's ledgers have two downstream consumers, not one.** `E40-F02-.../feature.md` names F01 as a direct dependency ("Consumes: I-01"), and `E40-F03-.../feature.md`'s G7/UAT-07 replay re-invokes F02's single-run command against a stored manifest — meaning F03's reproducibility guarantee depends transitively on the same base-SHA ledgers and manifest shape F01 commits. This raises the cost of getting the ledger format wrong or non-reproducible: it would silently break both F02 (directly) and F03's replay claim (transitively), not just one feature.

6. **`internal/sharkdata/default_data/manifest.yaml` is a false-cognate, not a reusable pattern.** It is the only other "manifest" file in the codebase, but it is a declarative bundle-structure manifest for prompt/skill validation (`internal/sharkdata` bundle validators), unrelated in schema or purpose to a benchmark corpus manifest. Its existence is worth noting only so a future reader does not conflate the two.

## Decisions

1. **Implement the I-01 contract exactly as pinned in `architecture.md`; do not re-derive the manifest or ledger shape during F01.** The corpus manifest fields, held-back-test placement rule (never in the repo the agent sees, injected post-run), and base-SHA ledger contents are epic-level decisions F01 executes, not proposes.

2. **Base-SHA ledger generation reuses this project's own quality-gate command sequence, applied to the fixture repo.** Generate the ledgers by running the fixture repo's own `fmt`/`vet`/`lint`/`test`-equivalent targets (the fixture repo needs its own `Makefile` and `.golangci.yml`, modeled on this repo's, since it is a separate Go module) — do not invent a divergent check sequence for the fixture repo.

3. **Build the "new lint issues only" diff logic as new tooling; there is nothing in this repo to reuse for it.** Document the diff method (how a lint-issue-set diff determines "new" vs. pre-existing) alongside the committed base-SHA ledger so F02's post-run check can apply it without guessing the semantics.

4. **Treat the fixture-repo location choice as a real trade-off, not a formality.** A vendored directory can follow the `test-fixtures/`-style convention (git-tracked, README at the fixture root) already established in this repo; a separate repo avoids bloating this repo's history with an unrelated Go module's commits and test-run artifacts, at the cost of an extra pinned-SHA dependency to manage. `feature.md` defers this to implementation deliberately — this research does not resolve it further, since neither F02 nor F03 depends on which choice is made, only on the manifest correctly pointing at wherever it lives.

5. **No Go changes; F01's entire output is new tooling/data outside `internal/`.** This is confirmed both by the epic's own constraint ("F01–F03 require no changes to shark itself," `epic.md` §4) and by the absence of any F01-relevant hook point in `internal/` found during this research — nothing in `internal/reporting`, `internal/config`, or elsewhere needs to change or be reused for corpus/oracle construction.

## Sources

- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F01-benchmark-corpus-v1-fixture-repo-and-screened-task/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/epic.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/shark-bench-design.md` (§2 Corpus & oracles, §3 per-entity oracle matrix, §4 per-metric mechanics)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md` (Corpus and oracle contract; Metric collection and artifact schema; Delivery boundaries and traceability)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-interaction-map.md` (I-01 row)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md` (parent epic research report)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F02-bench-harness-run-driver-and-metric-collection/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F03-baseline-report-and-noise-band/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F04-shark-run-live-progress-and-per-run-log/feature.md`
- `internal/sharkdata/default_data/research/recipes.yaml`
- `CLAUDE.md` (Mandatory Quality Gate)
- `Makefile` (`fmt`/`vet`/`lint`/`test` targets, lines 86-121)
- `.golangci.yml`
- `.github/workflows/ci.yml` (golangci-lint job, lines 121-124)
- `test-fixtures/uat-test-project/` and its `README.md`
- `internal/sharkdata/default_data/manifest.yaml`

RECOMMENDED OUTCOME: standard
