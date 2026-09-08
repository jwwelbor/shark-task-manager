# Plan: Run /code-review on Outstanding PRs

Generated: 2026-08-28 10:02

## Goal

Run `/code-review <PR#>` against every outstanding (open) PR that hasn't already
been reviewed, and track completion here.

## PR Review Checklist

| # | PR | Title | Review status | Done |
|---|----|----|----|:---:|
| 1 | [#205](https://github.com/jwwelbor/shark-task-manager/pull/205) | fix(B052): entity-scoped transcript paths for cascading shark run | Fixed stale doc comment (TD-156/mindepth already covered) + merged `7a340872` | [x] |
| 2 | [#204](https://github.com/jwwelbor/shark-task-manager/pull/204) | fix(B045): sync embedded sprint skills, fix invalid notes command | Fixed (routes via `entity_type` field, not key regex) + merged `1543de4f` | [x] |
| 3 | [#203](https://github.com/jwwelbor/shark-task-manager/pull/203) | fix(B053): filter expected test set by run_selector in admit.sh | Fixed zero-match silent-green bug (HIGH) + README + tc dedup + merged `3e79f39a` | [x] |
| 4 | [#202](https://github.com/jwwelbor/shark-task-manager/pull/202) | fix(B054): validate .git marker before accepting it as project root | Fixed ceiling-bound test + dup path, then `.git`-as-file content validation gap + merged `5bef4c49` | [x] |
| 5 | [#201](https://github.com/jwwelbor/shark-task-manager/pull/201) | fix(B058): credit terminal external dependencies in sprint readiness | Fixed fail-loud + chunking (MED) + dedup/doc (LOW) + merged `93254524` | [x] |
| 6 | [#200](https://github.com/jwwelbor/shark-task-manager/pull/200) | fix(B046): recognize JSON outcome objects in RunController gating | Fixed silent-fallthrough on malformed JSON (MED) + empty-outcome consistency + merged `4ffb83ac` | [x] |
| 7 | [#199](https://github.com/jwwelbor/shark-task-manager/pull/199) | fix(B044): filter claimed items out of sprint next candidates | No findings; merged `7e2cf836` | [x] |
| 8 | [#198](https://github.com/jwwelbor/shark-task-manager/pull/198) | fix(B047): close remaining presentation-layer progress-coherency gap | No new findings (TD-150 covers rest); merged `c1d94254` | [x] |
| 9 | [#197](https://github.com/jwwelbor/shark-task-manager/pull/197) | docs(B050): close out --depends-on wiring bug with correct fix citation | Finding's premise was wrong (B048 link is real, verified); fixed stale path citation + merged `1b74862a` | [x] |

**All 9 PRs resolved and merged as of 2026-08-28 23:37 UTC.**

All 9 were open, non-draft, mergeable, with passing CI, and no prior review
comments/threads as of 2026-08-28 — none skipped.

## Steps per PR

1. Run `/code-review <PR#>` (medium effort default) against the PR.
2. Record findings below (if any); decide fix-now vs. new bug ticket.
3. Check the box in the table once the review is posted/complete.

## Findings Log

(Append one entry per PR as reviews complete.)

### PR #205 — fix(B052): entity-scoped transcript paths for cascading shark run

- INCIDENT: a review subagent ran `gh pr checkout 205 --force`, discarding uncommitted WIP in `.vscode/settings.json` and `CLAUDE.md`. Recovered from VS Code local-history cache; `git status` matches session-start snapshot. Fixed a stray formatting artifact left by the recovery (2-space indent before a `---` rule in CLAUDE.md, CLAUDE.md:67).
- **[LOW]** `bench/scripts/run-one.sh:562` (`cp "$f" "$run_dir/run/transcripts/"`): the postrun transcript copy still flattens the destination — once Phase 2 allows cascade dispatch, two sibling cascade children producing same-named transcripts will silently overwrite each other at the destination, reintroducing (one hop later) the exact collision class B052 fixed at the source. Already acknowledged in the PR's own comments and tracked as deferred debt (TD-156).
- **[LOW]** Same file/area: the removed explicit `run.log` basename exclusion is replaced by an implicit `-mindepth 2` invariant with no local enforcement (e.g. an assertion) that `LivenessRecorder` always writes `run.log` at exactly mindepth 1.
- **[LOW]** `internal/config/config.go:277`: doc comment for `CaptureAgentTranscripts` still describes the pre-PR flat transcript-path layout (`.shark/runs/{run_id}/`) instead of the new `.shark/runs/{run_id}/{entity_key}/...` layout.
- No correctness bugs found; removed-behavior and reuse/duplication angles returned clean (secondary concerns already captured in TD-156).

### PR #204 — fix(B045): sync embedded sprint skills, fix invalid notes command

- `internal/sharkdata/default_data/skills/sprint-analytics/workflows/retro-sprint.md:424` (also synced into `skills/shark-rider/skills/sprint-analytics/workflows/retro-sprint.md`): the new entity-type routing table regex-matches carryover/rejected entity keys against shark key formats (`E##-F##-###`, `B###`, ...), but real `CarryoverEntity.Key` values are `{entity_type}-{entity_id}` (e.g. `task-42`), which never match. Every carryover/rejected entity falls into the "matches none" branch and loses its notes in the retro report, even though `CarryoverEntity.EntityType` already carries the correct type directly. Pre-existing field-name mismatch (`SUMMARY.carryover`/`rejected` vs actual `carryover_entities`) predates this diff, not reported separately.

### PR #202 — fix(B054): validate .git marker before accepting it as project root

- `internal/dbinit/project_root_test.go:666`: `TestFindProjectRoot_EmptyGitDir_NotAcceptedAsMarker` relies on `findProjectRoot` walking unbounded up the real filesystem after rejecting an empty `.git`. Unlike `internal/cli`'s `findProjectRootFrom` (bounded by a `ceiling` param in its sibling test), `dbinit.findProjectRoot` has no ceiling and the test has no host-pollution guard — in a sandbox/devcontainer where `TMPDIR` resolves under a real project root, the walk can pick up a real marker and the test fails/passes for the wrong reason.
- `internal/cli/root.go:315` (and duplicated in `internal/dbinit/project_root.go`): minor — `gitDir := filepath.Join(currentDir, ".git")` recomputes a path already built two lines above for the `os.Stat` call. Not a bug, just duplicated work in both copies.

### PR #199 — fix(B044): filter claimed items out of sprint next candidates

- No findings. Reviewed entity-type translation, claimability filtering, nil-claimReader backward-compat path, and comparison against the established `filterClaimable` pattern — all correct.

### PR #197 — docs(B050): close out --depends-on wiring bug with correct fix citation

- `docs/plan/bugs/B050.research-report.md:143`: cites `docs/plan/bugs/B048.md` and claims a `shark link B050 B048 --type=related_to` was run, but `B048.md` doesn't exist anywhere in the repo (main, this PR, or the abandoned B048 branch). The link/duplicate-bug claim is unverifiable and possibly never executed, despite being stated as completed fact in B050.md's Fix Notes and this report.

### PR #203 — fix(B053): filter expected test set by run_selector in admit.sh

- **[HIGH]** `bench/scripts/admit.sh:451` (`check_p2p_green`): a `run_selector` that matches **zero** enumerated tests now silently makes the whole package look P2P-green with nothing actually run. Old code compared the unfiltered `expected` set against `observed`, so a mistyped/no-match selector left `missing` = the full expected set → package correctly flagged not-clean. New code filters `expected` through the same selector first, so a selector matching nothing yields `expected=observed=missing=failed={}` and `go test -run <no-match>` exits 0 → `package_clean=True` with 0 tests actually run. Nothing validates that the selector matches ≥1 enumerated test. No test (tc096/tc097) covers this zero-match case — they cover the happy-path narrowing and the grammar-rejection path, not this gap.
- **[MED]** `bench/README.md:167` still documents `run_selector` as an unrestricted `-run` regex, but the new `validate_run_selector_grammar` rejects `|`, `[`, `(`, `\` at runtime — a corpus author following the README who writes a grouping/alternation selector gets an undocumented `RuntimeError`/exit 2.
- **[LOW]** `tc096`/`tc097` duplicate ~35–40 lines of near-identical transient-corpus-building Python heredoc between each other instead of sharing one helper.

### PR #201 — fix(B058): credit terminal external dependencies in sprint readiness

- **[MED]** `internal/services/sprint_service.go` (`computeExternalDependencyTerminalStatus`): `taskRepo.GetByKeys` error is silently swallowed (comment only, no log/wrap) — a transient DB error degrades Factor 2 to "all external deps unsatisfied" for every task in the sprint with zero observability, contrary to CLAUDE.md Rule 12 (fail loud).
- **[MED]** Same function builds an unbounded `key IN (?, ?, ...)` clause over every distinct external-dependency key in the sprint with no chunking — a large sprint can exceed SQLite's bound-parameter limit, which (per the item above) then fails silently rather than loudly.
- **[LOW]** `assignedKeys` (uppercased assignment-key set) is built twice per readiness computation — once in `computeExternalDependencyTerminalStatus`, once in `computeReadinessFromData` — wasted work and a latent drift risk between the two copies.
- **[LOW]** The `GetSprintReadiness` doc comment still claims "exactly two repository calls" even though this PR's own edit to that comment block left a third (pre-existing) call undisclosed.

### PR #200 — fix(B046): recognize JSON outcome objects in RunController gating

- **[MED]** `internal/runner/controller.go:921` — the new JSON-outcome branch silently swallows any `json.Unmarshal` error (malformed JSON, wrong-typed `outcome` field) and falls through to "no outcome specified" → pass-first silent advance, instead of surfacing a parse error. This is the same failure class (silent advance past a gate) the bug was filed to fix, and is inconsistent with the sibling `parseQuestionResponseHandoff`, which fails loud on invalid JSON.
- **[LOW]** Whole-stdout-only JSON parsing means a legitimate JSON blob emitted for unrelated reasons (e.g. a result doc with a coincidental `outcome` key) can be misread as a routing directive. Already tracked as TD-152 MAJ-1/MAJ-2 — not net-new.
- **[LOW]** Empty-outcome semantics diverge between the JSON and text-line paths: JSON `{"outcome":""}` silently falls back to pass-first, while the equivalent empty text-line format hard-errors.

### PR #198 — fix(B047): close remaining presentation-layer progress-coherency gap

- The background multi-agent review stalled (finder agents idled without the orchestrator consolidating), so this PR was reviewed directly instead of waiting further.
- The PR fixes two real, well-scoped gaps left by B047's earlier coordinator fix: the "ready for approval" readiness banner (`feature_helpers.go`) and `sortFeatures()`'s "progress" sort key both read the stale persisted `ProgressPct` cache instead of the live-computed weighted progress the table/JSON renderers already show — both now key off the same computed source, with a `cfg == nil` fallback to the cache matching the renderers' own fallback. Both changes carry red/green regression tests (`TestRenderFeatureAggregation_ReadinessMessage`, `TestSortFeatures_ProgressUsesLiveComputedValue`) and existing call sites were updated consistently.
- No new correctness issues found. This PR already went through a prior code-review pass (`docs/review/bugs/b047/code-review-20260827-2348-B047.md`) whose 7 non-blocking findings (perf: map-alloc-per-comparator-call instead of precompute; a 3rd duplicated status-count-map conversion in the same file; `sort.Slice` non-determinism on ties; local `.sharkconfig.json` has no `status_flow` so the new computed-progress branch always takes the cache-fallback path locally) are already filed as `docs/plan/tech-debt/TD-150.md` — all correctly non-blocking.
