# Test Plan: E31-F01 - Shift quality left into development

**Created:** 2026-11-27
**Spec:** [spec.md](spec.md) (stands in for both feature PRD and task spec; no tasks exist yet)
**Other inputs:** [plan.md](plan.md), [research-report.md](research-report.md)
**Status:** APPROVED (the gaps below are absorbed into the test cases)

## Conventions

- `<b>` = `internal/sharkdata/default_data/`; `<dr>` = `skills/shark-rider/skills/deep-review/`.
- **Family prompts:** `task/development.md`, `bug/development.md`, `change/development.md`, `tech_debt/triaged.md`, `tech_debt/in_progress.md`.
- **Content-only:** the change is prompt, skill or documentation text with no deterministic runtime behavior. These cases check rendering, include and reference resolution, and literals that are explicit contracts (headings, labels, marker lines, include directives). Policy meaning is a human review item (**R**). Do not add decision-table, mutation or simulated-behavior tests for prose. The spec's three decision tables are review checklists, not test tables.
- Upstream: E31 has no `uat-plan.md`, epic PRD, interaction map or cross-epic map. TCs trace to spec ACs.

## Product critical-path guard

- unresolved prerequisite: docs/product/D01-vision-statement.md missing
- unresolved prerequisite: docs/product/D02-success-criteria.md missing
- unresolved prerequisite: docs/plan/product-delivery-roadmap.md missing
- unresolved prerequisite: docs/plan/product-critical-path.md missing

No gate can be named, so neither the gate relationship nor advancement evidence can be reported. Side-quest call: **proceed** (workflow-quality work on Shark's own development loop; spec §Product critical-path guard). Nothing in this plan counts as golden-path advancement evidence.

## Spec drift and gaps

### Drift from plan.md

| Item | Assessment |
|---|---|
| Plan principle 3 "severity rules are unchanged" replaced by REQ-F-007 | Accepted: owner decision note 5527 (Q014, Q013 resolved). The spec's "non-blocking open question" wording is stale; behavior matches |
| 175-character summary cap dropped (D8) | Accepted, justified in spec |
| Tech-debt prompts and `_tdd_process` wiring added | Accepted scope addition (D2, D5) |
| Language-neutrality narrowed to added or changed content | Accepted (RR F3) |

### Gaps found here, with the binding test-plan handling

| ID | Gap | Handling |
|---|---|---|
| G-1 | Spec T1-T3 and AC-15 omit the rendered-prompt golden corpus (`internal/cli/commands/testdata/rendered-prompts/`). Prompt and skill edits fail `make test` until it is regenerated | TC-016 |
| G-2 | Threshold sites missing from rows B1-B9: `testing-requirements.md:25-28, 423-424`; `test-planning.md:335` (`complexity ≤ 5` example); `write-task.md:330` (`≥80% coverage` example) | All fall under REQ-F-008. Fix them; reword the two examples without a number (for example `≥N% coverage`). TC-002 |
| G-3 | AC-02's minimum pattern list misses real forms: `10 / 500 / 50` (`review-code.md:13`), table cells `\| 90% \|`, `--cov-fail-under=80`, `threshold: >= 80%` | Closed list in TC-002 |
| G-4 | The task prompt's inline "TDD IMPLEMENT" red-first steps contradict the artifact-type split once `_tdd_process` is wired in. No AC requires removing them | Reconcile so one verification rule remains. TC-007 R |
| G-5 | AC-08 omits REQ-F-012 and REQ-F-014 for the bug, change and tech-debt prompts, although REQ-F-019 requires them | TC-008 covers all five |
| G-6 | "Over a limit" is undefined at the boundary; today `review-code.md` flags nesting `≥4` | Interpretation: a limit is an inclusive maximum (4 allowed, 5 over), matching gocyclo and funlen. The rubric states this. TC-005 R |
| G-7 | Limit table does not cover a row present with a blank or placeholder value | Interpretation: treat as absent (generic default, labeled). TC-005 R |
| G-8 | REQ-F-018 has no value for work without AC IDs, behavior-changing TCs or a commit; no prompt requires committing | Interpretation: write `none`, `0/0`, `none`. Do not add a commit obligation. TC-008 R |
| G-9 | Spec places render checks in `internal/sharkdata`; `internal/templates` imports `sharkdata`, so an in-package test there cannot import the renderer | Render TCs live in `internal/templates` |
| G-10 | "Extracted from review-code.md Steps 2-4, 6-8" does not say whether review-code keeps a second copy | One checklist (capability map: no competing rubric). `review-code.md` keeps review-only procedure and points to the rubric. TC-006 R |

G-6 to G-8 are the narrowest readings of the spec. If the owner reads them differently, only the named review item changes.

### Traceability

| REQ | AC | TC |
|---|---|---|
| F-001, F-002 | 01 | 001 |
| F-005, F-008, NF-001, NF-002 | 02 | 002 |
| F-003 | 03 | 003 |
| F-004 | 04 | 004 |
| F-005, F-006, F-007, NF-005 | 05 | 005 |
| F-008 | 06 | 006 |
| F-012, F-014 | 07 | 007 |
| F-010, F-011, F-013, F-015, F-016, F-018, F-019 | 08 | 008 |
| F-017, F-019 | 09 | 009 |
| F-020 | 10 | 010 |
| F-019, NF-004 | 11 | 011 |
| F-021 | 12 | 012 |
| F-009 | 13 | 013 |
| F-023, F-024 | 14 | 014, 015 |
| NF-004 | 15 | 016, 017 |
| F-022, F-023 | 16 | 018 |
| NF-003, NF-006 | 05, 07 | 019 |

## Acceptance criteria review

All 16 ACs are testable. Ambiguities are G-5 to G-8 and G-10. No AC is an open-ended robustness assertion; AC-02 is closed by the pattern list below. The spec adds no lifecycle field, so state-transition selection does not apply.

## ISTQB technique and ISO 25010 coverage

Columns: Fn functional, Pf performance, Co compatibility, Us usability, Re reliability, Se security, Ma maintainability, Po portability. N/A cells are justified in the last column.

| AC | Technique | Fn | Pf | Co | Us | Re | Se | Ma | Po | N/A reason |
|---|---|---|---|---|---|---|---|---|---|---|
| 01 | Contract surface enumeration | TC-001 | N/A | TC-019 | TC-001 R | N/A | N/A | TC-002 | TC-001 | text only: no runtime, input or secrets |
| 02 | Attack-class enumeration (phrasing classes) + EP (allowed file vs others) | TC-002 | N/A | N/A | N/A | TC-002 counterfactual | N/A | TC-002 | TC-002 | static scan |
| 03 | Contract surface enumeration | TC-003 | N/A | TC-019 | TC-003 R | N/A | N/A | TC-002 | TC-003 R | text only |
| 04 | Contract surface enumeration | TC-004 | N/A | TC-004 | TC-004 R | N/A | N/A | TC-004 | N/A | text only; stack-neutral heading check |
| 05 | Contract surface enumeration + BVA (150 lines) | TC-005 | TC-005 (size) | TC-019 | TC-005 R | TC-005 | TC-019 | TC-002 | TC-005 R | |
| 06 | Contract surface enumeration | TC-006 | N/A | TC-016 | N/A | TC-017 | N/A | TC-002 | N/A | reference edit |
| 07 | Contract surface enumeration (exactly once) | TC-007 | TC-005 (size) | TC-019 | TC-007 R | TC-007 | TC-019 | TC-016 | TC-016 lint | |
| 08 | Contract surface enumeration (5 prompts x rules) | TC-008 | N/A | TC-019 | TC-008 R | TC-008 | TC-019 | TC-016 | TC-016 lint | prompt size bounded by TC-005 |
| 09 | EP (task vs other family) | TC-009 | N/A | N/A | TC-009 R | N/A | TC-019 | TC-016 | N/A | gate text |
| 10 | EP (three prompts with shortcut) | TC-010 | N/A | N/A | TC-010 R | TC-010 | TC-019 | TC-016 | N/A | gate text |
| 11 | Contract surface enumeration (preserved blocks) | TC-011 | N/A | TC-011 | N/A | TC-011 | N/A | TC-016 | N/A | preservation |
| 12 | Contract surface enumeration | TC-012 | N/A | TC-017 | TC-012 R | N/A | N/A | TC-012 | TC-016 lint | text only |
| 13 | Contract surface enumeration | TC-013 | N/A | TC-017 | TC-013 R | TC-017 | N/A | TC-002 | TC-013 R | |
| 14 | BVA + EP (new, untouched, edited code) | TC-014 | TC-015 (CI time within 10 min job) | TC-015 | TC-014 (linter named) | TC-015 | N/A | TC-014 | TC-015 | lint config has no security surface |
| 15 | Regression suite | TC-016, 017 | N/A | TC-016 | N/A | TC-017 | TC-016 | TC-016 | TC-016 | no runtime change |
| 16 | Contract surface enumeration | TC-018 | N/A | N/A | TC-018 R | N/A | N/A | TC-018 | N/A | repo-local docs |

**Coverage gaps (deferred by spec):** growth inside an existing function is not linted; nesting depth and file length are review-enforced; CI uses golangci-lint `latest` while local pins v2.9.0 (TC-015 observes the PR run).

## Observability design

| Behavior | Runtime evidence | Assertion |
|---|---|---|
| Lint enforcement | golangci output naming `funlen` or `gocyclo` with file and line; CI lint step result | TC-014, TC-015 |
| Self-review and AC walk | Summary line `ACs verified: …; red-first: …; commit: …` | TC-008 (instruction present) |
| Limit source | Label text in self-review and review findings | TC-005, TC-007 |
| Blocked on no gate commands | `blocked` outcome with bootstrap recommendation | TC-008 (instruction present) |
| All other edits | internal — no observability (static content) | n/a |

## Caller-path contracts

| TC | Production entrypoint | Lowest mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| 014, 015 | `make lint` from repo root, no extra flags; CI lint job | none | Hand-built temporary config; bare `golangci-lint run -c …` as the only evidence | Config exists but `make lint` or CI never runs it, or the baseline hides new code |

Content-only cases (justification for each: **content-only**, no deterministic Go behavior changes):

| TC | Renderer or direct-file entrypoint |
|---|---|
| 001, 002, 003, 004, 006, 012 | `sharkdata.ReadEmbedded(path)` or a walk of the embedded tree |
| 005 | `sharkdata.ReadEmbedded` and `templates.NewIncludeResolverWithEmbed("").Resolve("{{include: skills/quality/context/craft-rubric.md}}")` |
| 007-011, 019 | `templates.NewOrchestratorRenderer(canonicalPromptsDir).Render("<entity>/<status>.md", vars)`, the renderer `shark next` uses |
| 013, 018 | `os.ReadFile` from the repo root |
| 016, 017 | `make fmt && make lint && make test`; Python `unittest` in `<dr>/scripts` |

## Test cases

Automated tests carry the TC ID in the name (`TestE31F01_TC005_…`). Write each before its content edit and see it fail on the current tree.

### TC-001 Quality Limits in the standards template (AC-01)

- **Expected:** `skills/quality/context/standards-template.md` Output A contains `## Quality Limits`; names five rows (cyclomatic complexity, function length, file length, nesting depth, coverage marked optional) and five columns (limit, value, applies to, enforcement, measuring command); names both `tool-enforced` and `review-enforced`. `## Testing Standards (Code-Level)` requires test-isolation and assertion-style entries.
- **Negative:** no numeric value in any example row (also TC-002); no single stack presented as required.
- **R:** multi-stack examples are side by side.

### TC-002 Single source of thresholds (AC-02)

- **Input:** walk every file under `<b>/prompts`, `<b>/skills`, `<dr>/references`. Match each line, case-insensitive, against this closed list (RE2):

```text
T01  cyclomatic[^\n]{0,30}?\b10\b
T02  complexity(?:_score)?\b["\'\]\[:, ]{0,8}(?:"error", ?)?(?:≤|<=|<|>|≥|>=)? ?\b(?:10|5)\b
T03  -over[ =]10\b
T04  (?:function[^\n]{0,25}?\b50\b|\b50 lines per function)
T05  \bfiles?\b[^\n]{0,15}?\b500\b
T06  \b10 ?/ ?500 ?/ ?50\b
T07  nesting(?: depth)?[ :]{0,3}(?:≥|>=|>)? ?4\b
T08  (?:coverage[^\n]{0,70}?(?:>=|≥|>) ?\d{2,3} ?%|(?:>=|≥|>) ?\d{2,3} ?% ?(?:code |test )?coverage)
T09  (?:\d{2,3}%\+ ?(?:code |test )?coverage|coverage[^\n]{0,30}?\d{2,3}%\+)
T10  (?:--cov-fail-under[= ]\d+|coverage\.threshold\.\w+[= ]\d+)
T11  ^\s*(?:- |\| )(?:Service [Ll]ayer|API [Ee]ndpoints|Repositories|Utilities|Components(?: \(Frontend\))?|Overall Project)\s*(?::|\|)\s*(?:>=|≥)? ?\d{2,3}%
```

- **Expected:** matches only in `skills/quality/context/craft-rubric.md`. The allowlist (exact file and phrase) starts empty: no document-length rule matches these patterns today. Only a document-length phrase may be added.
- **Baseline (verified on the current tree, Python equivalent):** 30 matching lines, in `implement-api.md:506,529`; `implement-backend.md:492,608`; `implement-tests.md:665,686`; `implementation/context/coding-standards.md:259`; `testing-requirements.md:25-30,423,424,473`; `validation-gates.md:241,248-251,516`; `review-code.md:13,144`; `test-planning.md:335`; `code-review-reference.md:21,24`; `write-task.md:330`; `angle-d-cleanup.md:39,42`.
- **Counterfactual test:** a table test feeds one pre-change literal per pattern ID, including `(typical: >= 80% for new code)` after "coverage", and asserts a match. It asserts no match for `Coverage: 94%`, `Skip deep nesting beyond 3-4 levels`, `50 lines MAX per task file` and `≥80% of new hires`.
- **Positive:** `craft-rubric.md` states 10, 50, 500 and 4. The scan fails if that file is missing or a root yields zero files.
- **R:** diff review for other numeric gates in new wording, and no new stack-specific gate command in the bundle (NF-001).

### TC-003 Generator derives, proposes, confirms, validates (AC-03)

- **Expected:** `generate-standards.md` lists Quality Limits in Output A, names `craft-rubric.md`, contains the label `proposed — confirm`, and states no number (TC-002).
- **R:** Step 1 derives from linter config; Step 2 marks each row derived or proposed; unattended run labels rows; Reference Configs wire limits where the linter supports it; Step 4 fails on a missing section.

### TC-004 Bootstrap marker and fill-gaps (AC-04)

- **Expected:** `bootstrap.md` Phase 4 contains exactly `**Quality Limits**: declared, not declared, or n/a (readiness 2)`; Step 0.2 names `## Quality Limits` and `existing_standards_path`.
- **R:** trigger is the missing heading in `coding-standards.md` or a linked sub-file, not the marker value; only that section is added.

### TC-005 Craft rubric (AC-05)

- **Expected:** file exists; at most 150 lines (150 passes, 151 fails); resolves through the include resolver without error; contains the exact label `generic default — not declared in local standards` plus `declared in local standards (§Quality Limits)` and `proposed in local standards, unconfirmed`; references `skills/quality/context/review-rubric.md` by rooted path (it is inlined into prompts).
- **Negative:** no `{{`; none of the forbidden command strings listed in `next_test.go` `TestRenderedDispatchPromptsDoNotTellWorkersToTransitionSharkState` (not bare status words); `TestEmbedded_SkillsContainNoBareSharkCLIRefs` passes.
- **R:** resolution is per limit and matches all three spec rows (G-7); severity matches REQ-F-007 including already-over code as non-blocker; inclusive maximum (G-6); checklist covers reuse, standards citation, SOLID and architecture, idiom, complexity and size, test design; names no language, linter or test framework.

### TC-006 review-code defers to the rubric (AC-06)

- **Expected:** `review-code.md` contains `../context/craft-rubric.md`; no numeric threshold (TC-002); retains `triage skill's decision tree` and `unattended review disposition`; `TestEmbedded_InlinedSkillRelativeReferencesResolve` passes.
- **R:** `complexity_thresholds` input, Step 7, Step 12 blocker row and decision rules use the resolved limit; Step 6 points to the rubric; no rule contradicts it (G-10); report shape is unchanged.

### TC-007 Rendered task prompt (AC-07)

- **Expected:** render of `task/development.md` contains the rubric's heading once, the source label, and `TDD PROCESS:` (or the rewritten partial's single heading, with the existing partial tests updated to it) exactly once.
- **Negative:** no `{{include:`, `{{template` or `<no value>`; no `09-test-plan.md`.
- **R:** partial covers REQ-F-012 (a) to (d): three artifact-type rows, one test per TC with TC-ID and exact Expected Output, caller-path rules, never edit Preconditions, Input or Expected Output (`blocked`), no source-text search except explicit contracts. No leftover unconditional red-first rule (G-4).

### TC-008 Family prompt rules (AC-08; REQ-F-012, 014, 019)

- **Input:** render each family prompt.
- **Expected, per prompt:** READ list names `CLAUDE.md`, `AGENTS.md`, `docs/architecture/coding-standards.md`, `patterns-catalog.md`, `tech-stack.md`, and mentions Quality Limits and Testing Standards; rubric and `_tdd_process` markers exactly once each; summary token `ACs verified:`; no unresolved directive.
- **R, per prompt:** empty or placeholder-only sources give `blocked` plus bootstrap, with no implementation; at least one runnable source runs; missing limits never block; limits applied first pass and the measuring command run; every checklist finding fixed or reasoned; rework reads latest transition note and cited gate report beside defect-class sweep; pre-existing outside-diff failure is fixed or routed via triage and the gate stays unmet (`fail` in envelope prompts); envelope prompts put the summary line first in `gate_result.summary`; TC rules conditional on a test plan; G-8 values.

### TC-009 Exit gates (AC-09)

- **R on rendered output:** task EXIT GATE has (a) AC evidence walk per AC ID, (b) resolved limits with source labeled, (c) diff within `## Scope` "Files to modify" or justified, (d) self-review recorded with none unresolved. Bug, change and both tech-debt prompts have (a), (b), (d).
- **Negative:** existing task exit-gate items (TC-ID naming, caller-path, quality gate) remain.

### TC-010 Shortcuts tightened (AC-10)

- **R on rendered `change/development.md`, `tech_debt/triaged.md`, `tech_debt/in_progress.md`:** every "advance immediately" or equivalent resume path requires quality gate, self-review and exit gate; the gate-command `blocked` rule applies before it.
- **Negative:** no sentence allows advancing on gate pass alone.

### TC-011 Bug contracts preserved (AC-11)

- **Expected:** `git diff <base> -- <b>/prompts/bug/development.md` shows no removed or altered text in: the approved-plan and coverage-index paragraphs, the decision-constraint paragraph, `DECISION:`, the `route_rework` role line, `_gate_result_directive`, and REQUIRED OUTER EVIDENCE. READ and EXIT GATE may gain lines.
- **Automated:** `gate_result_role_prompt_consistency_test.go` and `TestRenderedPromptLint` (bug evidence kinds) pass.

### TC-012 Sweep records standards (AC-12)

- **Expected:** `defect-class-sweep.md` Guard selection names `coding-standards.md`; the existing `TestDefectClassSweep*` tests pass.
- **R:** applies to rubric `rule` categories; linter rule where supported; entry never replaces the verified guard.

### TC-013 Deep review (AC-13)

- **Expected:** `<dr>/SKILL.md` reviewer load instruction names `context/craft-rubric.md`; `consolidator.md` and `angle-d-cleanup.md` reference the rubric; no T01-T11 match in `SKILL.md` or references.
- **R:** blocker row: declared limit blocks; generic default needs evidenced impact; already-over is non-blocker. Both forms (skill read and extracted file) are usable.

### TC-014 Lint limits (AC-14)

Use temporary `.go` files in a real package; delete them afterwards. Entrypoint `make lint`.

| Case | Input | Expected |
|---|---|---|
| a | new function, 50 body lines | pass |
| b | new function, 51 body lines | fail, `funlen` |
| c | cyclomatic complexity 10 | pass |
| d | cyclomatic complexity 11 | fail, `gocyclo` |
| e | 51-line function in a `_test.go` file | fail, `funlen` |
| f | 50 lines plus 2 comment-only lines | pass |
| g | 50 lines plus 1 blank line | fail, `funlen` |
| h | all temporary files removed | pass |
| i | branch with the feature applied, existing over-limit code untouched | pass |
| j | the feature's own new test code | pass |

- **Probe evidence:** a scratch module with golangci-lint 2.9.0 confirmed b, d, e, g flagged and a, c, f not flagged; an untouched over-limit function was not flagged; growing its body was not flagged; editing its declaration line was flagged. These are scratch results; the table must still run through `make lint`.
- **Standards wording:** the REQ-F-023 function-length entry states the tool's counting (blank lines count, comment-only lines do not).
- **Negative:** `.golangci.yml` is unchanged; a main-config failure still fails `make lint`.
- **"Unchanged main":** after merge, `make lint` passes on `main` (post-merge observation).

### TC-015 Baseline and CI (AC-14, REQ-F-024)

- **Expected:** `new-from-rev` is a full SHA; after a fetch, `git merge-base --is-ancestor <sha> refs/remotes/origin/main` succeeds (use the full ref: bare `main` is ambiguous in git commands here, and a local ref can be stale); `git cat-file -e <sha>:.golangci-limits.yml` fails (predates the config); only gocyclo and funlen enabled, `statements` disabled; Makefile runs the limits config after the main run; CI lint job has `fetch-depth: 0` and a second step with `--config=.golangci-limits.yml`; both steps pass on the PR.
- **Edge, `main` moves before merge:** every line that reaches `main` after the pin is checked, including other branches' squash merges that were never held to these limits, and PR CI lints the merge commit. At the final rebase, re-pin to the current `origin/main` head (still predates the config, so the checks above hold). Fix a failure caused by code outside this branch by re-pinning, never by editing that code or relaxing a limit.
- **Negative:** an unreachable SHA must not pass silently. The probe showed golangci warns `bad revision`, scans everything and exits 1; record the CI log showing no such warning.

### TC-016 Gate and golden corpus (AC-15)

- **Steps:** regenerate with `go test ./internal/cli/commands/ -run TestRenderedPromptsGolden -update`; re-pin `task/development.md` in `productCriticalPathGuardBaselineSHA256`; update `_tdd_process` expectations; run `make fmt && make lint && make test`.
- **Expected:** all pass, none skipped for these checks. Golden diff is limited to files traceable to an edited bundle file; expected: the five family goldens, `{_shared,bug,change,feature}/code_review.golden`, `feature/test_planning.golden`, `feature/task_generation.golden`. Review that diff line by line; `-update` alone proves nothing.
- **Named regressions:** `TestRenderedPromptsGolden`, `TestF02_AC3_OutputEqualsGoldenCorpus`, `TestRenderedPromptLint`, `TestTwelvePromptsAddGuardAsPureAddition`, `TestDevelopmentPromptGuardSitsInCompletionReportingSection` (the line `When done: stop and summarize` must survive the REQ-F-018 edit, guard after it as a standalone paragraph), `TestRenderedDispatchPromptsDoNotTellWorkersToTransitionSharkState`, the gate-result role consistency test, `TestEmbedded_SkillsContainNoBareSharkCLIRefs`, `TestEmbedded_BundleReferencesResolve`.

### TC-017 Review-output and deep-review regressions (AC-15)

- **Expected pass:** `TestConsolidatorRoutesNonBlockersThroughTriage`, `TestUnattendedReviewDispositionHandoff`, the rest of `review_output_policy_test.go`, `TestEmbedded_ReviewFindingsRouteThroughTriage`, `TestEmbedded_NonBlockersCarryStableFingerprint`.
- **Also:** `cd <dr>/scripts && python3 -B -m unittest discover -p 'test_*.py'` (not part of `make test`; baseline on the current tree: 114 tests pass in about 210 s).

### TC-018 Dogfood docs (AC-16)

- **Expected:** `tech-stack.md` has `## Quality Gate` with columns Step, Command, When and `make fmt`, `make lint`, `make test`; no leftover `### Quality Gate (Mandatory)`. `coding-standards.md` has `## Quality Limits` with 10/gocyclo/tool, 50/funlen/tool, 500/review, 4/review, production and test, `make lint`. `testing.md` forbids shared package-level command or flag state between tests.
- **R:** the require/assert rule is still present.

### TC-019 Project without standards; no bypass (NF-003, NF-006)

- **Input:** render all five family prompts with the embedded bundle, no override, no project docs.
- **Expected:** no render error; default label present.
- **R:** no new text permits skipping or weakening the quality gate; no marker field or `patterns-catalog.md` is required.

## Integration scenarios

| Boundary | Verify | TC |
|---|---|---|
| Rubric → family prompts (`{{include:}}`) | Same checklist development and review use | 005, 007, 008 |
| Rubric → review-code (relative path) → four code_review renders | Reference resolves; render stable | 006, 016 |
| Rubric → deep-review workers and consolidator | Loaded and severity applied | 013, 017 |
| generate-standards/bootstrap → project standards → prompts | Heading names agree (`Quality Limits`, `Testing Standards`) | 001, 003, 004, 008 |
| Standards limits → golangci config → CI | Values and counting agree | 014, 015, 018 |

No epic UAT scenarios exist to reference.

## Test infrastructure

| Need | Existing pattern | New |
|---|---|---|
| Scan and rubric tests (TC-001-006, 012) | `internal/sharkdata/embed_test.go`, `bundle_references_test.go` (walk and counterfactual style) | `internal/sharkdata/craft_rubric_test.go` |
| Render tests (TC-007, 008, 019) | `internal/templates/includes_test.go` (`newCanonicalOrchestratorRenderer`) | `internal/templates/development_prompts_test.go` |
| Goldens | `internal/cli/commands/next_golden_test.go` | none |
| Partial tests | `partials_test.go`, `orchestrator_partials_test.go` | update expectations |

New test files must satisfy the new limits (TC-014 j). No database, mocks or services are involved. Overrides under `shark-data/overrides/` are absent in this repo.

## Cross-feature contract tests (I-##)

None. E31 has one feature and declares no I-##.

## Cross-epic integration tests (X-##)

None. The spec declares no X-##, and the product map has no E31 row; there is nothing to defer.

## Codex Test-Plan Red-Team

**Verdict:** not run. The dispatch supplied no `codex_command`, and the worker ownership contract bars external AI CLIs. Non-blocking. Author self-check against the seven questions (not independent):

| Check | Result |
|---|---|
| Open-ended ACs | AC-02 closed by T01-T11 |
| Technique fit | Table above |
| Enumeration | Five prompts, 30 baseline lines, ten lint boundary rows |
| ISO 25010 | No empty cells |
| Observability | Static content marked internal |
| Negative cases | In TCs for runtime AC-14 and regression-prone content |
| Caller path | AC-14 only runtime; rest justified content-only |

Residual risk: R items rely on code review and QA judgment. Reviewers should record a per-item result.

## Recommendations

- [x] Ready for task generation. Tasks must carry G-1, G-2, G-4 (task 2 owns G-2; tasks 2-4 each regenerate goldens for their own edits; task 3 owns G-4; task 6 owns TC-014/015).
