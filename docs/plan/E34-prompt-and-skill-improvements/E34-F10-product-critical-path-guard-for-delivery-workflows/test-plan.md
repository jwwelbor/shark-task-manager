---
feature_key: E34-F10-product-critical-path-guard-for-delivery-workflows
epic_key: E34
title: Product Critical-Path Guard for Delivery Workflows — Test Plan
---

# E34-F10 Test Plan

**content-only justification**: This feature adds one new embedded prompt
partial (`_partials/_product_critical_path_guard.md`) and edits twelve
existing embedded prompt markdown files to reference it, plus documents (but
does not create) two illustrative project-artifact markdown shapes. No Go
source, CLI command, schema, or deterministic runtime behavior changes (per
spec.md's Architecture section and D-F10-01). Per CLAUDE.md's prompt-only
testing guidance and this workflow's own "Prompt-only changes" gate, tests
below use the production template renderer (`internal/templates`), direct
grep/file-content checks against the checked-in bundle, and manual policy
wording review — not caller-path, mutation, or decision-table tests, which
apply only to deterministic runtime behavior this feature does not add.

## AC Test Matrix

| TC | AC | Description | Input/Setup | Expected outcome | Edge cases |
|----|----|--------------|--------------|-------------------|------------|
| TC-001 | AC-1 | New partial exists, defines exactly one template, renders cleanly with all four source files absent | `internal/sharkdata/default_data/prompts/_partials/_product_critical_path_guard.md` present; run through the production renderer (`internal/templates`, sibling table entry to `TestIncludeResolver_BasicInclude` in `includes_test.go`, or a companion `_product_critical_path_guard_test.go`) against this repository's actual project state, which today has none of `docs/product/D01-vision-statement.md`, `docs/product/D02-success-criteria.md`, `docs/plan/product-delivery-roadmap.md`, `docs/plan/product-critical-path.md` | Renderer returns no error; the file contains exactly one `{{define "_product_critical_path_guard"}}` block (grep count = 1, no second `{{define}}` in the same file); rendered output reports all four source files as unresolved | A file with zero or two-or-more `{{define}}` blocks fails the test; a renderer error (broken `{{include}}`/`{{template}}` syntax) fails the test |
| TC-002 | AC-2 | All twelve wired prompts invoke the guard template exactly once; no prompt restates the guard's reporting fields inline | Grep each of the twelve files listed in spec.md's Architecture table (`prompts/sprint/{planning,active,closing}.md`, `prompts/epic/{assessment,decomposition,active}.md`, `prompts/feature/{specification,test_planning,task_generation,task_review,approval}.md`, `prompts/task/development.md`) for the literal string `{{template "_product_critical_path_guard" .}}` | Each of the twelve files contains that literal string exactly once (grep `-c` = 1 per file); none of the twelve files contains inline paraphrased duplicates of the guard's five reporting fields (current gate name, contribution-to-gate relationship, executable advancement evidence, unresolved prerequisite, side-quest disposition) outside the single template invocation — mirrors E34-F06's AC-2 single-source-of-truth pattern and its structural, not exact-string, verification approach | A file with zero or 2+ invocations fails; a file that also spells out "report the current gate" or equivalent prose alongside the template call (duplicated source of truth) fails |
| TC-003 | AC-3 | Rendering any of the twelve prompts against this repository's actual project state produces an unresolved-prerequisite guard block with no error and no change to pre-existing content | Render each of the twelve prompts (production renderer) before-feature (baseline captured from the parent commit / `git show <base>:<path>`) and after-feature; diff the two outputs | The only diff between before/after renders is the new guard block's addition; every existing line of prompt content is byte-identical; the guard block itself reports all four source files as "unresolved prerequisite: `<file>` missing" with no rendering error | Any diff outside the added guard block (reordered section, altered pre-existing line) fails the test; a renderer error/panic on any of the twelve fails the test |
| TC-004 | AC-4 | Guard block names the five disqualified evidence classes verbatim in `specification.md`, `test_planning.md`, and `approval.md` renders | Render `prompts/feature/specification.md`, `prompts/feature/test_planning.md`, `prompts/feature/approval.md`; extract the guard block; grep for the five evidence-class terms from REQ-F-034 (reusing E34-F02's evidence-authenticity vocabulary per research-report.md finding 5): fixture data, a captured/recorded run, a hand-authored test actor, a contract-only test, a component-level test suite | All five terms are present verbatim in the guard block of all three renders (15 assertions: 5 terms × 3 files) | Any of the five terms missing, reworded, or present in only some of the three renders fails the test |
| TC-005 | AC-5 | No wired prompt's rendered guard block contains a bare workflow status name or a `shark` CLI verb | Render all twelve prompts; extract each guard block; (a) grep the block against the full set of `steps:` keys from every entity workflow YAML under `internal/sharkdata/default_data/workflow/*.yaml` (task, feature, epic, bug, change, question, sprint — e.g. `draft`, `in_progress`, `qa`, `approval`, `completed`, `active`, `planning`, `closing`, …); (b) grep the block for `shark ` followed by a known CLI verb (`claim`, `heartbeat`, `release`, `status advance`, `status set`, `next-status`, `set-status`) | Zero matches for both (a) and (b) in every guard block across all twelve renders | A guard block containing a bare status token that also happens to be an ordinary English word requires the check to anchor on the full token set, not a substring — a false-positive-prone naive substring match (e.g. matching "active" inside unrelated prose) must be scoped to the guard block only, not the whole rendered prompt, since the surrounding prompt content legitimately contains status names and `shark` commands outside the guard |

## Caller-Path Contracts

Not applicable in the traditional sense — no test case in this plan drives a
Go function with a production runtime argument shape, because this feature
adds no deterministic runtime code. Per the "Prompt-only changes" exemption,
each test case above instead names its concrete entrypoint:

- TC-001: `internal/templates` production renderer (e.g.
  `NewIncludeResolverWithEmbed("").Resolve(...)` or the equivalent path already
  exercised by `internal/templates/includes_test.go`'s `TestIncludeResolver_*`
  table) — `content-only` justification; the renderer IS the entrypoint under
  test.
- TC-002, TC-004, TC-005: direct grep/file-content check against the rendered
  output (TC-004, TC-005) or the checked-in bundle file (TC-002) —
  `content-only` justification; entrypoint is the rendered prompt text or the
  bundle file itself.
- TC-003: golden-diff of before/after renders through the same production
  renderer as TC-001 — `content-only` justification.

## Integration Scenarios

- **Guard partial → twelve consuming prompts**: `_product_critical_path_guard.md`
  is `{{template}}`-invoked by all twelve prompts named in spec.md's
  Architecture table. Verify via the same renderer path as TC-001 that each of
  the twelve consuming prompts resolves the template invocation without a
  broken-reference error — this is the integration boundary between "the
  guard exists and renders standalone" (TC-001) and "every consuming prompt
  actually wires it in" (TC-002/TC-003).
- **task/development.md scoping**: spec.md requires the guard be placed only
  in `development.md`'s completion-reporting section, not the whole file body.
  TC-003's diff for this one file must additionally confirm the guard block's
  insertion point is inside the completion-reporting section (not, e.g.,
  prepended to the file) — this is a structural-position check layered on top
  of TC-003's byte-identical-elsewhere assertion.
- **Epic UAT scenarios**: `docs/plan/E34-prompt-and-skill-improvements/uat-plan.md`
  does not name E34-F10 or any product-critical-path scenario explicitly
  (confirmed empty via grep 2026-09-04) — this feature does not contribute to
  a named epic UAT scenario; its acceptance criteria are fully self-contained
  in spec.md AC-1 through AC-5.

## Test Infrastructure

- **Existing to reuse**: `internal/templates/includes_test.go`'s
  `TestIncludeResolver_*` table pattern (add entries rather than a new test
  file where practical) for TC-001's standalone-render check and the
  integration-scenario include-resolution checks; the same renderer
  invocation pattern used there is reused for TC-003's before/after diff and
  TC-004/TC-005's render-then-grep checks. `internal/sharkdata/embed_test.go`'s
  existing global gates (`TestEmbedded_SkillsContainNoBareSharkCLIRefs`,
  `TestEmbedded_AgentsDescribeRoleNotWorkflow`) remain in force for the new
  partial file by construction — it is workflow-adjacent guard content, not
  an agent persona, and REQ-NF-003 already requires it contain no bare
  `shark` CLI verb, which is a strict subset of what those gates check at the
  whole-bundle level.
- **New test helpers needed**: a companion test file (or new table entries in
  `includes_test.go`) implementing TC-001 (standalone define-count + clean
  render), TC-002 (per-file invocation-count grep across the twelve paths),
  TC-003 (before/after golden diff across the twelve paths, including the
  development.md structural-position check), TC-004 (evidence-class term grep
  scoped to the guard block in three renders), and TC-005 (bare-status/CLI-verb
  grep scoped to the guard block in all twelve renders, using the full
  `steps:` key set enumerated from `internal/sharkdata/default_data/workflow/*.yaml`
  as the negative-token list). No fixture-execution harness is needed — every
  check is a static render, grep, or diff against checked-in files, consistent
  with E34-F06's and E34-F09's precedent for prompt-only features.

## Cross-feature contract tests (I-##)

None. `spec.md`'s "Cross-feature interactions" section and
`E34-interaction-map.md` both confirm (grep verified 2026-09-04) no I-## row
names E34-F10 as a producer or consumer — F10 is described as "an
independent pre-dispatch product-alignment guard" that does not produce or
consume an I-## payload.

## Cross-epic integration tests (X-##)

None. `spec.md`'s "Cross-epic integrations" section, `E34-cross-epic-map.md`,
and `docs/product/cross-epic-integration-map.md` all confirm (grep verified
2026-09-04) no X-## row is declared for E34-F10.

## Codex Test-Plan Red-Team

**Verdict:** FAILED — both attempts timed out without producing a verdict
**Issues raised:** 0 (no verdict produced)
**Issues addressed before dev:** 0
**Issues deferred:** 0 (nothing to defer — codex never reached the review)

Two attempts were made per Step 7.5's timeout/retry guidance, both against
this file and spec.md, asking codex to evaluate AC-1..AC-5 for
open-endedness, ISTQB technique fit under the content-only exemption,
grep/diff enumeration completeness, ISO 25010 gaps, observability
justification, negative-case presence, and TC entrypoint concreteness.

**Attempt 1:** `codex exec -s read-only -c model_reasoning_effort=high
--skip-git-repo-check "<red-team prompt>"`, wrapped in `timeout 595`.
Backgrounded automatically after 120s (harness timeout), then reported
`status: killed` at the 595s mark with the captured output file containing
only `[killed]` — no codex output was captured at all before the kill.

**Attempt 2:** Same prompt, run directly via the harness's own
`run_in_background` (timeout 600000ms) instead of a shell `timeout` wrapper,
redirected to a log file for inspection. Also killed at the timeout. The
captured log shows codex spent its entire budget on self-directed
exploration before reaching the actual review: it attempted to load a
`brownfield-analysis` skill path that doesn't exist in this environment
(`sed: can't read /home/jwwel/.claude/skills/brownfield-analysis/SKILL.md`),
then pivoted to reading `graphify`'s skill file, this repo's `.codex/memories/MEMORY.md`,
`CLAUDE.md`, and several `.claude/rules/*.md` files as a single combined
shell command — none of which this red-team task required — and was killed
before it ever opened `test-plan.md` or `spec.md`. This is an environment/
tool-invocation issue (codex over-scoping its own orientation pass), not
evidence about this test plan's quality.

Per Step 7.5's degrade rule ("After two failures, log 'Codex test-plan
review: FAILED — [error]' as a non-blocking note in the test plan and
proceed — do not gate on codex availability, but document the gap"), this is
recorded as a non-blocking gap. Both attempts' raw logs are preserved at
`/tmp/claude-1000/-home-jwwel-projects-shark-task-manager--worktrees-E34/3faa3f75-56cb-49e1-8b4a-3aaa23db02d1/tasks/b4rw4yrxr.output`
and `.../scratchpad/codex-f10-attempt2.log` for the next worker to inspect
(session-local paths; re-run rather than assume they persist).

Owner: next worker to touch E34-F10 (development or code-review pass) —
re-run the Step 7.5 prompt with a narrower instruction (explicitly forbid
loading unrelated skills/memory/rules files) so codex spends its budget on
the actual review instead of orientation.
Timeframe: before this feature's code-review gate, or re-attempted and
re-deferred with a fresh reason if still failing at that point.

## Recommendations

- [x] Ready for development — no PRD/spec drift (spec.md traces every
      requirement to feature.md's Triage Breadcrumb and requirements.md Area
      10 verbatim), every AC (AC-1 through AC-5) has at least one concrete
      test case, the content-only exemption is documented and applied
      consistently with E34-F06's/E34-F09's precedent, no I-##/X-## rows are
      declared for this feature (confirmed empty). The one open item is the
      Codex Step 7.5 red-team, deferred per its own degrade rule rather than
      blocking — see the section above.
- [ ] Needs BA refinement
- [ ] Needs tech refinement

*Last Updated*: 2026-09-05
