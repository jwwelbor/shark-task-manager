# Test Plan: E31-F01 - Shift quality left into development

**Created:** 2026-11-27 (revised 2026-11-27 for the amended spec; **R2** delta 2026-11-27 for the round-2 amendment)
**Spec:** [spec.md](spec.md) (stands in for both feature PRD and task spec)
**Other inputs:** [plan.md](plan.md), [research-report.md](research-report.md)
**Status:** APPROVED

Tasks 001-011 are built (candidate `5d848cab`). **R2** marks what rework task 012 changes; a task 012 developer writes only the R2 checks. TC IDs are stable; TC-027 and TC-028 are new. Where an R2 entry reverses an earlier expectation it says **flips**: change that check first and see it fail on the current tree.

## Conventions

- `<b>` = `internal/sharkdata/default_data/`; `<dr>` = `skills/shark-rider/skills/deep-review/`.
- **Family prompts:** `task/development.md`, `bug/development.md`, `change/development.md`, `tech_debt/triaged.md`, `tech_debt/in_progress.md`.
- **Content-only:** the change is prompt, skill or documentation text with no deterministic runtime behavior. These cases check rendering, include and reference resolution, and literals that are explicit contracts (headings, labels, marker lines, include directives). Policy meaning is a human review item (**R**). Do not add decision-table, mutation or simulated-behavior tests for prose. The spec's prose decision tables (limit resolution, severity, gate commands, verification type) are review checklists, not test tables. The tech-debt status table is runtime behavior and is tested (TC-020).
- Upstream: E31 has no `uat-plan.md`, epic PRD, interaction map or cross-epic map. TCs trace to spec ACs.

## Product critical-path guard

- unresolved prerequisite: docs/product/D01-vision-statement.md missing
- unresolved prerequisite: docs/product/D02-success-criteria.md missing
- unresolved prerequisite: docs/plan/product-delivery-roadmap.md missing
- unresolved prerequisite: docs/plan/product-critical-path.md missing

No gate can be named, so neither the gate relationship nor advancement evidence can be reported. Side-quest call: **proceed** (workflow-quality work on Shark's own development loop; spec §Product critical-path guard). Nothing in this plan counts as golden-path advancement evidence.

## Spec drift and gaps

Drift from `plan.md`: edited-code rule, tech-debt parking and the round-1 fixes are owner decisions recorded in the spec; the dropped 175-character cap (D8), tech-debt scope (D5) and narrowed language neutrality are accepted there. Earlier gaps G-1 to G-5, G-7 and G-10 are closed by the amended spec or the built tasks.

| ID | Gap | Handling |
|---|---|---|
| G-6 | Boundary meaning of a limit | Inclusive maximum. TC-005 R, TC-014 |
| G-8 | REQ-F-018 has no value for work without AC IDs, behavior-changing TCs or a commit | Write `none`, `0/0`, `none`; no commit obligation. TC-008 R |
| G-9 | `internal/templates` imports `sharkdata` | Render TCs live in `internal/templates` |
| G-11 | **R2, resolved.** The spec now states that the engine does not compute the resume target (capability map, lifecycle table, D16). REQ-F-034 corrects the three blocked prompts; the engine half is B091 | TC-027; TC-020e records the unchanged engine behavior |
| G-12 | The edited-code rule makes any edit to an over-limit Go test file force a refactor (for example `embed_test.go`, `b036_prompt_crawl_test.go`, `next_test.go`, `controller_gate_ingest_test.go`) | Put new tests in new or within-limit files. The suite was not run with the new YAML, so task 009 runs it first and lists failing files; an over-limit file needing an edit (candidates: `viewer_service_test.go`, `embed_test.go`, `next_test.go`, `run_apply_result_test.go`, `controller_gate_ingest_test.go`) is a refactor or consultation case. TC-024 |
| G-13 | AC-22 names Makefile mutations only; REQ-F-033(a) also covers CI | TC-025 mutates both |
| G-14 | **R2.** AC-24 says "every table-driven case"; REQ-F-033(c) names the loops | Binding set: the loops REQ-F-033(c) names plus any case loop task 012 adds. A loop of assertions inside one case (for example `assertContainsAll`, the `readListStandards` loop) is not a table case. TC-026, TC-028 |
| G-15 | **R2.** REQ-F-033(d) does not name job-level `if:` or `continue-on-error`, a `//nolint` directive, or a pin moved forward | Not required of the guard. Review items; TC-015 checks the pin's ancestry |

### Traceability

| REQ | AC | TC | Rework task |
|---|---|---|---|
| F-001, F-002, F-025 | 01 | 001 | 007 |
| F-005, F-008, NF-001, NF-002 | 02 | 002 | 007 |
| F-003 | 03 | 003 | regression (built) |
| F-004, F-030 | 04 | 004 | 011 |
| F-005 to 007, F-025, 026, 029, 035, NF-005 | 05 | 005 | 007; **012** |
| F-008, F-009 | 06 | 006 | 007 |
| F-012, F-014, F-029 | 07 | 007 | 008 |
| F-010 to 016, 018, 019, 026, 027, 033(b) | 08 | 008 | 008; **012** |
| F-017, F-019 | 09 | 009 | 008; **012** |
| F-020 | 10 | 010 | 008 (regression) |
| F-019, NF-004 | 11 | 011 | 008; **012** |
| F-021 | 12 | 012 | 011 |
| F-009 | 13 | 013 | 007; **012** |
| F-023, F-024 | 14 | 014, 015 | 010 |
| NF-004, F-031 | 15 | 016, 017 | every task; closes after **012** |
| F-022, F-023, F-024 (header) | 16 | 018 | 010; **012** |
| F-025, 026, NF-006 | 17 | 021 | 007 (rubric), 008 (prompts) |
| F-018 | 18 | 022 | 008; **012** |
| F-012, F-027 | 19 | 023 | 008 |
| F-031, NF-003, NF-007 | 20 | 020 | 009 |
| F-032 | 21 | 024 | 008 (includes), 009 (workflow); closes after **012** |
| F-033(a), (d) | 22 | 025 | 010; **012** |
| F-033(c) | 24 | 026, 028 | 007; **012** |
| F-015 | 24 | 008 (literal 9) | **012** |
| F-034 | 23 | 027 | **012** |
| NF-003, NF-006 | 05, 07 | 019 | 008 |

A task writes tests only for the TCs (or the named part) in its row.

## Acceptance criteria review

All 24 ACs are testable (**R2:** AC-23 and AC-24 are new). AC-02 is closed by the pattern list. The one behavior-bearing field is the tech-debt `status` (AC-20): state-transition technique, TC-020 (and TC-027 for the resume).

## ISTQB technique and ISO 25010 coverage

Columns: Fn functional, Pf performance, Co compatibility, Us usability, Re reliability, Se security, Ma maintainability, Po portability. N/A cells are justified in the last column.

| AC | Technique | Fn | Pf | Co | Us | Re | Se | Ma | Po | N/A reason |
|---|---|---|---|---|---|---|---|---|---|---|
| 01 | Contract surface enumeration | TC-001 | N/A | TC-019 | TC-001 R | N/A | N/A | TC-002 | TC-001 | text only: no runtime, input or secrets |
| 02 | Attack-class enumeration (phrasing classes) + EP (allowed file vs others) | TC-002 | N/A | N/A | N/A | TC-002 counterfactual | N/A | TC-002 | TC-002 | static scan |
| 03 | Contract surface enumeration | TC-003 | N/A | TC-019 | TC-003 R | N/A | N/A | TC-002 | TC-003 R | text only |
| 04 | Contract surface enumeration | TC-004 | N/A | TC-004 | TC-004 R | N/A | N/A | TC-004 | N/A | text only; stack-neutral heading check |
| 05 | Contract surface enumeration + BVA (**R2:** 175 lines) | TC-005 | TC-005 (size) | TC-019 | TC-005 R | TC-005 | TC-019 | TC-002 | TC-005 R | |
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
| 17 | Contract surface enumeration + BVA (recorded value) | TC-021 | N/A | TC-019 | TC-021 R | TC-021 | TC-021 | TC-016 | N/A | prompt text |
| 18 | EP (change and tech-debt vs task and bug) | TC-022 | N/A | TC-011 | N/A | TC-022 (1000-byte cap) | N/A | TC-016 | N/A | prompt text |
| 19 | Contract surface enumeration (five prompts) | TC-023 | N/A | N/A | TC-023 R | TC-023 | TC-019 | TC-016 | N/A | prompt text |
| 20 | State transition (all values, invalid, recovery, state addition) | TC-020 | N/A | TC-020g | TC-020f | TC-020d (run stops), TC-020e | TC-020d (human resolution only) | TC-024 | TC-016 | no performance surface |
| 21 | BVA (500 lines) + contract enumeration | TC-024 | N/A | TC-024 (test names) | N/A | TC-016 | N/A | TC-024 | N/A | test-only refactor |
| 22 | Mutation of guard inputs (deterministic; **R2:** attack-class enumeration of ways to weaken the config) | TC-025 | N/A | TC-015 | N/A | TC-025 | N/A | TC-025 | N/A | config guard |
| 23 | **R2:** EP (three entity prompts vs `task/blocked.md`) + state transition (parked step to resume) | TC-027 | N/A | TC-027 (task prompt unchanged) | TC-027 R | TC-027 runtime | TC-027 (no `--force` on resume) | TC-027 mirror | TC-016 lint | no performance surface |
| 24 | **R2:** Contract surface enumeration (5 REOPENED lines, named loops) | TC-008, 026, 028 | N/A | TC-019 | TC-008 R | TC-028 | N/A | TC-026, 028 | N/A | prompt text and test structure |

**Coverage gaps:** nesting depth and file length are review-enforced (TC-024 steps 2 and 4 for this branch); CI uses golangci-lint `latest` while local pins v2.9.0 (TC-015 observes the PR run); history-based resume is not implemented (**R2:** B091); a persistent tech-debt gate failure can re-dispatch without bound (**R2:** B092, out of scope); the limits guard does not cover the G-15 routes.

## Observability design

| Behavior | Runtime evidence | Assertion |
|---|---|---|
| Lint enforcement | golangci output naming `funlen` or `gocyclo` with file and line; CI lint step result | TC-014, TC-015 |
| Self-review and AC walk | Summary line `ACs verified: …; red-first: …; commit: …` | TC-008 (instruction present) |
| Limit source | Label text in self-review and review findings | TC-005, TC-007 |
| Tech debt parks | Status `blocked`, history row, run outcome `paused`, `next` action `pause` | TC-020d, e |
| Consultation | `blocked` reason naming the six REQ-F-026 fields | TC-021 |
| Resume from a parked step (**R2**) | History row `blocked` to `<step>` carrying the `--reason` text; status equals the step | TC-027 |
| Bug evidence routing (**R2**) | Envelope `evidence[]` entries of kinds `ac_evidence`, `limit_measurement`, `self_review` | TC-022 (instruction present) |
| Blocked on no gate commands | `blocked` outcome with bootstrap recommendation | TC-008 (instruction present) |
| All other edits | internal — no observability (static content) | n/a |

## Caller-path contracts

| TC | Production entrypoint | Lowest mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| 014, 015 | `make lint` from repo root, no extra flags; CI lint job | none | Hand-built temporary config; bare `golangci-lint run -c …` as the only evidence | Config exists but `make lint` or CI never runs it, or an edited file's old overage is hidden |
| 020a-c | `DefaultTechDebtWorkflow()`, `ResolveOutcome(step, outcome)`, `ValidateWorkflow(wf)` | none (embedded YAML) | Hand-built `WorkflowConfig`; temp YAML | YAML still routes `blocked: identified`, or `blocked` is not parking |
| 020d | `RunController.Run(ctx, "TD-001", RunOptions{EntityType: "tech_debt", ...})`, new file `internal/runner/controller_tech_debt_blocked_test.go` | `AgentDispatcher`, plus fakes for the transitioner, action service, persistence and lease | Literal outcomes, outcome roles or actions: the fakes must read them from `workflow.NewService("").ForLevel("tech_debt")` over the embedded bundle, and the transitioner must return the status last written (so the second `Run` sees `blocked`). No mocked `workflow.Service` | Item bounces to `identified` and is dispatched again |
| 020e, f | Candidate `shark status advance|set`, `shark next --json` in a scratch project; renderer for `tech_debt/blocked.md` | none | Live repo database | Parking loads in unit tests but the CLI still dispatches or cannot be resumed |
| 024 | `git diff`, `go test -list`, limits lint on the candidate | none | n/a | A split drops a test or leaves an edited file over a limit |
| 025 | `go test ./internal/sharkdata/` reading the real config, Makefile and CI file | none; **R2:** in-memory mutations of the real files' bytes are allowed | Hand-written fixture configs as the only evidence | Limits loosened, a linter disabled or its failure swallowed while tests stay green |
| 027 (runtime) | **R2:** candidate binary: `shark next <key> --json`, then the rendered `shark status history` and `shark status set` commands, in a scratch project | none | Live repo database; `--force` on the resume step; hand-typed commands that differ from the rendered text | The rendered instruction does not return the item to its parked step |

Content-only cases (justification for each: **content-only**, no deterministic Go behavior changes):

| TC | Renderer or direct-file entrypoint |
|---|---|
| 001, 002, 003, 004, 006, 012 | `sharkdata.ReadEmbedded(path)` or a walk of the embedded tree |
| 027 (static, **R2**) | `templates.NewOrchestratorRenderer(canonicalPromptsDir).Render("<entity>/blocked.md", vars)` |
| 028 (test-structure only, **R2**) | `go test -v -run 'TestE31F01_TC0(19\|20[bc])' ./internal/templates/ ./internal/config/workflow/` |
| 026 (test-structure only) | `go test -v -run TestE31F01 ./internal/sharkdata/` |
| 005 | `sharkdata.ReadEmbedded` and `templates.NewIncludeResolverWithEmbed("").Resolve("{{include: skills/quality/context/craft-rubric.md}}")` |
| 007-011, 019, 021-023 (TC-008 also `sharkdata.ReadEmbedded`) | `templates.NewOrchestratorRenderer(canonicalPromptsDir).Render("<entity>/<status>.md", vars)`, the renderer `shark next` uses |
| 013, 018 | `os.ReadFile` from the repo root |
| 016, 017 | `make fmt && make lint && make test`; Python `unittest` in `<dr>/scripts` |

## R2 anchor literals

These strings are explicit contracts for task 012, so its tests may assert them (REQ-F-012(c)). Search only the stated slice, after collapsing each run of whitespace (and, in R2-4, the `#` markers) to one space, so line wrapping cannot hide a literal.

| ID | Slice | Must contain |
|---|---|---|
| R2-1 | rubric `## Scope of the limits` | `only to named languages`; `every code file is` |
| R2-2 | rubric consultation bullet `The re-dispatched author` | `and the work item's decision notes` |
| R2-3 | `coding-standards.md` `## Quality Limits`, below the table | `Go source only`; `review_workflow.js` |
| R2-4 | `.golangci-limits.yml` leading comment | `every function`; `--whole-files`; `new-from-rev`; `Quality Limits`; not `new code only` (any case) |
| R2-5 | deep-review `SKILL.md` `## Runner 3` | `craft-rubric.md`; `shark skill get quality context/craft-rubric.md`; `skills/quality/context/craft-rubric.md` |
| R2-6 | each rendered bug, change, tech-debt blocked prompt | `shark status history <id>`; `shark status set <id> <parked step> --reason` (`<parked step>` literal) |
| R2-7 | bug DECISION `blocked` clause | `unavailable external evidence`; `missing quality gate`; `architect consultation`; `unclear requirements` |
| R2-8 | bug SUMMARY LINE paragraph | `ac_evidence`; `limit_measurement`; `self_review`; `alongside` |

## Test cases

Automated tests carry the TC ID in the name (`TestE31F01_TC005_…`). Write each new or revised check before its content edit and see it fail on the current tree. Limit refactors need no new red step.

### TC-001 Quality Limits in the standards template (AC-01)

- **Expected:** `skills/quality/context/standards-template.md` Output A contains `## Quality Limits`; names five rows (cyclomatic complexity, function length, file length, nesting depth, coverage marked optional) and five columns (limit, value, applies to, enforcement, measuring command); names both `tool-enforced` and `review-enforced`. `## Testing Standards (Code-Level)` requires test-isolation and assertion-style entries.
- **Expected (revised):** Output A also defines the optional `### Limit exceptions` table under Quality Limits with the columns Unit, Limit, Recorded value, Reason, Decision, and states the REQ-F-025 rules (unit existed before the change; measurement at or below the recorded value; architect or owner decision only; tool suppression cites the row).
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
- **Baseline (verified on the pre-feature tree `d9427496`, Python equivalent):** 30 matching lines, in `implement-api.md:506,529`; `implement-backend.md:492,608`; `implement-tests.md:665,686`; `implementation/context/coding-standards.md:259`; `testing-requirements.md:25-30,423,424,473`; `validation-gates.md:241,248-251,516`; `review-code.md:13,144`; `test-planning.md:335`; `code-review-reference.md:21,24`; `write-task.md:330`; `angle-d-cleanup.md:39,42`.
- **Counterfactual test:** a table test feeds one pre-change literal per pattern ID, including `(typical: >= 80% for new code)` after "coverage", and asserts a match. It asserts no match for `Coverage: 94%`, `Skip deep nesting beyond 3-4 levels`, `50 lines MAX per task file` and `≥80% of new hires`.
- **Positive:** `craft-rubric.md` states 10, 50, 500 and 4. The scan fails if that file is missing or a root yields zero files.
- **R:** diff review for other numeric gates in new wording, and no new stack-specific gate command in the bundle (NF-001).

### TC-003 Generator derives, proposes, confirms, validates (AC-03)

- **Expected:** `generate-standards.md` lists Quality Limits in Output A, names `craft-rubric.md`, contains the label `proposed — confirm`, and states no number (TC-002).
- **R:** Step 1 derives from linter config; Step 2 marks each row derived or proposed; unattended run labels rows; Reference Configs wire limits where the linter supports it; Step 4 fails on a missing section.

### TC-004 Bootstrap marker, fill-gaps and Greenfield (AC-04)

- **Expected:** `bootstrap.md` Phase 4 contains exactly `**Quality Limits**: declared, not declared, or n/a (readiness 2)`; Step 0.2 names `## Quality Limits` and `existing_standards_path`; the Greenfield standards-generation text names `standards-template.md` (or generate-standards), `Quality Limits`, and the test-isolation and assertion-style entries. Scope each assertion to its own section, so a Step 0.2 mention cannot satisfy the Greenfield one.
- **Negative:** no numeric default in the Greenfield text (TC-002).
- **R:** the fill-gaps trigger is the missing heading in `coding-standards.md` or a linked sub-file, not the marker value; only that section is added; a present section with a missing or placeholder row is not a trigger; Greenfield values come from `craft-rubric.md` by reference and are labeled `proposed — confirm`.

### TC-005 Craft rubric (AC-05)

- **Expected:** file exists; **R2 (flips):** at most 175 lines (175 passes, 176 fails; `craftRubricLimit` is 170 today and the file is exactly 170 lines); resolves through the include resolver; contains the labels `generic default — not declared in local standards`, `declared in local standards (§Quality Limits)` and `proposed in local standards, unconfirmed`; contains `../context/review-rubric.md`; states 10, 50, 500 and 4 (TC-002); contains the `Limit exceptions` rule and the checklist.
- **Negative:** no `{{`; none of the command strings forbidden by `next_test.go` `TestRenderedDispatchPromptsDoNotTellWorkersToTransitionSharkState`; the rooted `skills/quality/context/review-rubric.md` is absent; no case-insensitive `substantially expand`; `TestEmbedded_SkillsContainNoBareSharkCLIRefs` passes.
- **Rework note:** the existing assertions on the rooted path and the 150-line limit are replaced.
- **Expected (R2, REQ-F-035):** the section `## Scope of the limits` (cut at the next `## `) contains anchors R2-1. The bullet in `## Architect consultation` that begins `The re-dispatched author` contains anchor R2-2. Assert inside those slices only: both sections already say "not measured" or "decision note" elsewhere.
- **R (R2):** a file outside a declared language scope is unmeasured, is not a "limit not declared" case and never takes a generic default; with no language sentence every code file is measured; the rubric still names no language.
- **R:** resolution is per limit and matches the three rows of spec §Limit resolution (blank or placeholder counts as absent); the four rows of the spec §Severity table appear with the same verdicts (edited file within limit: no finding; covered by exception: no finding, cite it; over with no exception, including pre-existing overage and any source: blocker; unedited file: declared gives non-blocker to triage, default or proposal needs evidenced impact); the admission statement of REQ-F-007; remedy is "add or fill the row, or bootstrap fill-gaps when the section is missing"; inclusive maximum; checklist covers reuse, standards citation, SOLID and architecture, idiom, complexity and size, test design, and "complexity and size" measures edited files in full; names no language, linter or test framework.

### TC-006 review-code defers to the rubric (AC-06)

- **Expected:** `review-code.md` contains `../context/craft-rubric.md`; no numeric threshold (TC-002); retains `triage skill's decision tree` and `unattended review disposition`; `TestEmbedded_InlinedSkillRelativeReferencesResolve` passes.
- **Negative (revised):** no case-insensitive `substantially expand` in `review-code.md`.
- **R:** the `complexity_thresholds` input and Step 7 use the resolved limit; Step 12 Blocker and Non-blocker rows and the decision rule apply the rubric's severity table by reference; Step 6 points to the rubric; no rule contradicts it; report shape is unchanged; the "triage skill's decision tree" wording stays.

### TC-007 Rendered task prompt (AC-07)

- **Expected:** render of `task/development.md` contains the rubric's heading once, the source label, and `TDD PROCESS:` (or the rewritten partial's single heading, with the existing partial tests updated to it) exactly once.
- **Expected (revised):** in every family render, the `Bundled `quality` skill` note appears at an index before `# Craft Rubric`. Counter-factual: a rooted `skills/quality/...` citation in the rubric produces no note and fails this.
- **Negative:** no `{{include:`, `{{template` or `<no value>`; no `09-test-plan.md`.
- **R:** partial covers REQ-F-012 (a) to (d): three artifact-type rows, one test per TC assigned to this work item, with TC-ID and exact Expected Output, caller-path rules, never edit Preconditions, Input or Expected Output (`blocked`), no source-text search except explicit contracts. No leftover unconditional red-first rule (G-4).

### TC-008 Family prompt rules (AC-08; REQ-F-010 to 016, 018, 019, 026, 027, 033(b))

- **Input A (source):** `sharkdata.ReadEmbedded("prompts/<family prompt>")`, unrendered. **Input B:** the render.
- **Expected A, each of five prompts, one `t.Run` per prompt and literal:** the template body contains
  1. `If every source is absent or contains only a placeholder`
  2. `project bootstrap`
  3. `Missing quality limits never block.`
  4. `Run the self-review checklist in the Craft Rubric below against your own diff`
  5. `before you change any code, read the latest transition note and the gate report that note cites`
  6. `Every code file you edited is within the resolved limits`
  7. `measured in full`
  8. `consult the architect`
  9. **R2:** `and the work item's decision notes`, asserted on the template line that starts `REOPENED OR REWORK DISPATCH:` (not anywhere in the file; the included rubric also carries the words)
- **R2 (AC-24):** literal 5 stays one unbroken substring of the same line.
- **Expected B:** the READ section contains `CLAUDE.md`, `AGENTS.md`, `docs/architecture/coding-standards.md`, `patterns-catalog.md`, `tech-stack.md` exactly once each (count within READ only), and mentions Quality Limits (with Limit exceptions) and Testing Standards; rubric and `_tdd_process` markers once each; summary token `ACs verified:`; no unresolved directive.
- **Negative:** the bug prompt has no `CLAUDE.md for coding standards` item. Asserting the eight literals on the render is not acceptable: the included rubric could satisfy them.
- **R, per prompt:** all-empty gate sources give `blocked` plus bootstrap with no implementation, and one runnable source runs; limits applied first pass; each edited code file is measured in full, and when the measuring command covers only new code the rest is measured by tool or hand; checklist findings fixed or reasoned; pre-existing failure outside the changed files is fixed or routed via triage and the gate stays unmet (`fail` in envelope prompts), while an overage inside an edited file is handled by the edited-code rule; envelope prompts lead `gate_result.summary` with the summary line; TC rules apply only when a test plan exists; G-8 values.

### TC-009 Exit gates (AC-09)

- **Expected:** the task EXIT GATE contains (a) `for each AC ID`; (b) `Every code file you edited is within the resolved limits`, with measured in full, overages that predate the change, source label, and the covering limit exception; (c) `## Scope` "Files to modify" or justification; (d) `Self-review is recorded with no unresolved finding`. Bug, change and both tech-debt prompts contain (a), (b) and (d). Existing own items remain.
- **Negative:** no source template or render contains `Every new or changed function and file`.
- **R2 (flips for bug):** the bug EXIT GATE evidence-walk item names `evidence[]`; the bug gate contains neither `compactly` nor `stays within its size limit`. It still contains `for each AC ID`.
- **R:** item (c) is task-only and covers files created by a limit refactor.

### TC-010 Shortcuts tightened (AC-10)

- **R on rendered `change/development.md`, `tech_debt/triaged.md`, `tech_debt/in_progress.md`:** every "advance immediately" or equivalent resume path requires quality gate, self-review and exit gate; the gate-command `blocked` rule applies before it.
- **Negative:** no sentence allows advancing on gate pass alone.

### TC-011 Bug contracts preserved (AC-11)

- **Expected (R2, flips: DECISION was frozen):** compared with `d9427496`, `<b>/prompts/bug/development.md` has no removed or altered text in the approved-plan and coverage-index paragraphs, the decision-constraint paragraph, the `route_rework` role line, `_gate_result_directive` and REQUIRED OUTER EVIDENCE. Permitted edits are the inserted blocks, the REQ-F-011 READ fix (with its renumbering), the PRECEDENCE line, the EXIT GATE evidence-walk item and the DECISION `blocked` clause. Use `git diff d9427496 -- <file>`.
- **Automated (R2):** in the rendered DECISION paragraph, the text up to and including `and blocked ` equals the `d9427496` text, and the text from `. For route_rework outcomes` to the end equals the `d9427496` text. Copy both pieces into the test as string constants; the test must not read git history (the CI test job uses a shallow checkout). The clause between them contains anchors R2-7. This pins `pass`, `review_plan`, `needs_research`, `replan` and `fail` word for word.
- **Automated (R2):** the bug source template contains `"evidence": [` exactly once (`strings.Count`), so the REQUIRED OUTER EVIDENCE example stays the last one that `TestRenderedPromptLint` reads.
- **Automated:** `gate_result_role_prompt_consistency_test.go` and `TestRenderedPromptLint` (bug evidence kinds) pass; `TestBugPlanAdmissionThroughGateIngestion` stays green.
- **R (R2):** `blocked` now covers a missing quality gate, an architect consultation and unclear requirements, and no other bug-prompt sentence sends those cases to `fail`.

### TC-012 Sweep records standards (AC-12)

- **Expected:** `defect-class-sweep.md` Guard selection names `coding-standards.md`, contains `proposed — confirm`, and names each craft category (one `t.Run` each): `ALTITUDE`, `CODING-STANDARD`, `COMPLEXITY`, `DIP`, `DRY`, `ERROR-HANDLING`, `IDIOM`, `ISP`, `LOGGING`, `LSP`, `NAMING`, `OCP`, `SIMPLIFICATION`, `SRP`, `TESTS`, `TYPES`, `YAGNI`. Existing `TestDefectClassSweep*` tests pass. Match whole tokens (`SRP` must not be satisfied inside another word).
- **R:** the section states that `ARCH`, `ARCHITECTURE`, `CONTRACT`, `CORRECTNESS`, `REMOVED-BEHAVIOR`, `RISK`, `ROUNDTRIP`, `SECURITY`, `SIBLING`, `WIRING` never edit standards; the 17 plus 10 equal the `review-code.md` §B categories; "proposed — confirm" applies unless the owner confirms in the same session; linter rule where supported; the entry never replaces the verified guard.

### TC-013 Deep review (AC-13)

- **Expected:** `<dr>/SKILL.md` reviewer load instruction names `context/craft-rubric.md`; `consolidator.md` and `angle-d-cleanup.md` reference the rubric; no T01-T11 match in `SKILL.md` or references.
- **Expected (revised):** none of `SKILL.md`, `consolidator.md`, `angle-d-cleanup.md` matches case-insensitive `substantially expand`.
- **R:** `SKILL.md` tells the host, before any runner dispatches a reviewer, to confirm the rubric is readable by either route and otherwise to stop, report the review incomplete and name the missing rubric (no review without it); Blocker and Non-blocker rows and the angle-D text apply the severity table by reference; `review_workflow.js` does not contradict this.
- **Expected (R2, G-1):** the `## Runner 3` section of `SKILL.md` (cut at the next `## `) contains anchors R2-5. A match in the precondition section or in Runner 2 does not count.
- **R (R2):** the anchors sit in the list of Runner 3 prompt contents, so the fallback reviewer receives the rubric name and both read routes.

### TC-014 Lint limits (AC-14)

Temporary `.go` files or temporary one-line edits in a real package on the branch; revert afterwards. Entrypoint `make lint`.

| Case | Input | Expected |
|---|---|---|
| a | new function, 50 body lines | pass |
| b | new function, 51 body lines | fail, `funlen` |
| c | cyclomatic complexity 10 | pass |
| d | cyclomatic complexity 11 | fail, `gocyclo` |
| e | 51-line function in a `_test.go` file | fail, `funlen` |
| f | 50 lines plus 2 comment-only lines | pass |
| g | 50 lines plus 1 blank line | fail, `funlen` |
| h | all temporary changes removed | pass |
| i | feature applied; over-limit code in files unchanged since the pin | pass |
| j | the feature's own new test code | pass |
| k | one line added to an otherwise untouched Go file (unchanged since the pin) that already holds an over-limit function, outside that function | fail on the pre-existing function (`funlen` or `gocyclo`) |

- **Setup check for k:** confirm `git diff --quiet <new-from-rev> -- <file>` before the edit, and name the file, function and linter output in the evidence.
- **Counterfactual:** without `--whole-files`, row k passes.
- **Standards wording:** the REQ-F-023 function-length entry keeps the counting rule (blank lines count, comment-only lines do not).
- **Negative:** `.golangci.yml` is unchanged; a main-config failure still fails `make lint`.
- **"Unchanged main":** after merge, `make lint` passes on `main` (post-merge observation).

### TC-015 Baseline and CI (AC-14, REQ-F-024)

- **Expected:** `new-from-rev` is a full SHA; after a fetch, `git merge-base --is-ancestor <sha> refs/remotes/origin/main` succeeds (use the full ref: bare `main` is ambiguous in git commands here, and a local ref can be stale); `git cat-file -e <sha>:.golangci-limits.yml` fails (predates the config); only gocyclo and funlen enabled, `statements` disabled; Makefile runs the limits config after the main run with `--whole-files`; CI lint job has `fetch-depth: 0` and a second step with `--config=.golangci-limits.yml --whole-files`; both steps pass on the PR.
- **Edge, `main` moves before merge:** every line that reaches `main` after the pin is checked, including other branches' squash merges that were never held to these limits, and PR CI lints the merge commit. At the final rebase, re-pin to the current `origin/main` head (still predates the config, so the checks above hold). Fix a failure caused by code outside this branch by re-pinning, never by editing that code or relaxing a limit.
- **Negative:** an unreachable SHA must not pass silently. An earlier scratch probe showed golangci warns `bad revision`, scans everything and exits 1; record the CI log showing no such warning.

### TC-016 Gate and golden corpus (AC-15)

- **Steps:** regenerate with `go test ./internal/cli/commands/ -run TestRenderedPromptsGolden -update`; re-pin `task/development.md` in `productCriticalPathGuardBaselineSHA256` (in whichever file holds it after the split; **R2:** `product_critical_path_guard_test.go`, and it is the only family prompt pinned there, so the REOPENED edit breaks one hash); update `_tdd_process` expectations; run `make fmt && make lint && make test`.
- **Expected:** all pass, none skipped. The golden diff holds only files traceable to an edited bundle file, including the new `tech_debt/blocked.golden`; review it line by line (`-update` alone proves nothing).
- **Named regressions:** `TestRenderedPromptsGolden`, `TestF02_AC3_OutputEqualsGoldenCorpus`, `TestRenderedPromptLint`, `TestB036_ExhaustivePromptCrawl_WritesVerificationArtifacts` (covers the new prompt), `TestTwelvePromptsAddGuardAsPureAddition`, `TestDevelopmentPromptGuardSitsInCompletionReportingSection`, `TestRenderedDispatchPromptsDoNotTellWorkersToTransitionSharkState`, the gate-result role consistency test, `TestEmbedded_SkillsContainNoBareSharkCLIRefs`, `TestEmbedded_BundleReferencesResolve`, `internal/config/workflow` and `internal/runner` packages.
- **Edge (R2, flips):** the three corrected `blocked.md` prompts are pause prompts and stay out of the dispatch-prompt ban list in `next_test.go`, so their `shark status history` and `shark status set` lines are expected. The guard-block and rubric bans on `shark status set` do not cover them. Do not add them to a ban list.
- **Expected (R2) golden diff:** `bug/`, `change/` and `tech_debt/blocked.golden` change, plus goldens of changed family prompts and goldens that inline the rubric (`# Craft Rubric`: the five development goldens today; a `code_review` golden that only mentions `craft-rubric.md` does not change); `task/`, `feature/` and `epic/blocked.golden` do not.

### TC-017 Review-output and deep-review regressions (AC-15)

- **Expected pass:** `TestConsolidatorRoutesNonBlockersThroughTriage`, `TestUnattendedReviewDispositionHandoff`, the rest of `review_output_policy_test.go`, `TestEmbedded_ReviewFindingsRouteThroughTriage`, `TestEmbedded_NonBlockersCarryStableFingerprint`.
- **Also:** `cd <dr>/scripts && python3 -B -m unittest discover -p 'test_*.py'` (not part of `make test`; pre-feature baseline: 114 tests pass in about 210 s).

### TC-018 Dogfood docs (AC-16)

- **Expected:** `tech-stack.md` has `## Quality Gate` with columns Step, Command, When and `make fmt`, `make lint`, `make test`; no leftover `### Quality Gate (Mandatory)`. `coding-standards.md` has `## Quality Limits` with 10/gocyclo/tool, 50/funlen/tool, 500/review, 4/review, production and test, `make lint`. `testing.md` forbids shared package-level command or flag state between tests.
- **Expected (revised):** the Quality Limits text states that `make lint` measures every function in each Go file changed since the pinned revision, and that file length and nesting depth are checked on every edited file by self-review and code review.
- **Negative:** the old "only functions whose declaration line changed" baseline wording is gone.
- **Expected (R2, F-110):** the `## Quality Limits` section contains anchors R2-3 after the last table row. The `Applies to` cells stay `production and test`.
- **Expected (R2, F-115):** the leading comment block of `.golangci-limits.yml` (lines before `version:`) satisfies anchors R2-4. TC-015 and TC-025 stay green after the edit.
- **R:** the require/assert rule is still present.

### TC-019 Project without standards; no bypass (NF-003, NF-006)

- **Input:** render all five family prompts with the embedded bundle, no override, no project docs.
- **Expected:** no render error; default label present.
- **R:** no new text permits skipping or weakening the quality gate; no marker field or `patterns-catalog.md` is required.

### TC-020 Tech-debt status lifecycle and parking (AC-20; REQ-F-031, NF-003, NF-007)

**Technique:** state transition over the closed table in spec §Tech-debt status lifecycle. No I-##/X-## axis names this field.

**a. Values and the new state** (`DefaultTechDebtWorkflow()`)
- Sorted status set equals exactly `blocked, cancelled, identified, in_progress, research, resolved, triaged, wont_fix` (set equality, so adding a status fails). `ValidateWorkflow` returns nil. Start is `identified`; complete statuses stay `cancelled, resolved, wont_fix`.
- `blocked`: phase `blocked`, color `red`, display token `BLK`, responsibility `human`, action `pause`, parking true, prompt `tech_debt/blocked.md`, no outcomes. No step has `rework_outcomes`.

**b. Entry and exit transitions** (`ResolveOutcome` and `StatusFlow`)

| From | pass | fail | blocked | cancelled | wont_fix | StatusFlow |
|---|---|---|---|---|---|---|
| identified | research | identified | blocked | cancelled | wont_fix | 5 unique, `research` first |
| research | triaged | identified | blocked | cancelled | wont_fix | 5 unique, `triaged` first |
| triaged | in_progress | identified | blocked | cancelled | wont_fix | 5 unique, `in_progress` first |
| in_progress | resolved | identified | blocked | cancelled | wont_fix | 5 unique, `resolved` first |
| blocked | none | none | none | none | none | `[identified, in_progress, research, triaged]` |
| resolved, wont_fix, cancelled | none | none | none | none | none | empty |

**c. Invalid transitions:** the `blocked` outcome of the four workable steps never resolves to `identified`; every outcome from `blocked` and from the three terminal steps fails to resolve.

**d. Run loop parks and does not re-dispatch** (once from `triaged`, once from `in_progress`)
- **Input:** `RunController.Run` on `TD-001`; the dispatcher returns a gate result `blocked` with a `no_kickback_reason`.
- **Expected:** transition to `blocked`; run outcome `paused`; dispatcher called exactly once. A second `Run` while `blocked` returns `paused` with zero dispatches.

**e. Production CLI and recovery** (candidate binary, scratch project from `scripts/shark-scratch-env.sh`, never the live repo)
- Create a tech debt, `status advance --outcome pass` to `research`, then `--outcome blocked`: status `blocked`; `next <key> --json` has `action: pause` and the rendered blocked prompt.
- `status advance <key> --outcome pass` exits non-zero ("has no outcomes"); status unchanged.
- `status advance <key>` (no outcome) moves to `identified`, the first entry of the parking flow, **not** the parked-from step (existing engine behavior, B091; **R2:** the prompts no longer tell anyone to do this).
- Park again, then `status set <key> research` (no `--force`): status `research`.
- A second item parked from `identified` (`--outcome blocked`) reaches `blocked`.
- **Evidence so far:** a 2026-11-27 scratch run with the pre-change binary and a probe workflow carrying this parking step produced these results. Re-run on the candidate.

**f. Prompt:** `tech_debt/blocked.md` renders with and without `blocked_reason`; mirrors `change/blocked.md` apart from the entity noun (**R2:** TC-027 extends the mirror to bug); golden present (TC-016).

**g. Compatibility:** no migration; items in the seven existing statuses keep them (diff review: no Go production change, no data rewrite).

**Test updates:** `workflow_multilevel_test.go` expects 7 statuses in two places and 4 transitions per workable step, and its StatusFlow comment says blocked routes to `identified`. Update all of them (8, 5, comment) in the split files (TC-024).

### TC-021 Limit exceptions and architect consultation (AC-17; REQ-F-025, 026, NF-006)

- **Expected:** `craft-rubric.md` and each rendered family prompt contain `consult the architect` (prompt) and the exception rule (rubric, included).
- **R:** exception covers only a pre-existing unit, only at or below the recorded value, added or changed only by architect or owner; new units never excepted; a worker never records one; a worker in doubt returns `blocked` naming file, unit, limit and source label, measured value, considered refactor and safety concern (`gate_result.no_kickback_reason` in envelope prompts); the overage is not left in place; rework re-reads the decision (TC-008 literal 5). No text permits bypassing the gate.
- **Edge (R):** value equal to the recorded value is covered; one above needs a new decision.

### TC-022 Evidence routing in envelope prompts (AC-18; REQ-F-018)

- **Expected:** rendered `change/development.md`, `tech_debt/triaged.md`, `tech_debt/in_progress.md` and (**R2, flips**) `bug/development.md` contain `evidence[]` in both the EXIT GATE and the SUMMARY LINE paragraph. In `development_prompts_test.go`, `bug/development.md` joins the `routed` set; today it is asserted to have no `evidence[]`. Only the task prompt stays unrouted.
- **Expected (R2):** the bug SUMMARY LINE paragraph contains anchors R2-8.
- **Negative:** no routed EXIT GATE asks for the per-AC walk inside `gate_result.summary` (`compactly`, `stays within its size limit` absent).
- **Negative (R2):** the kinds the routing text assigns to the new entries are exactly those three; the text adds no `"kind": "` literal and no second `"evidence": [` (TC-011), and sits before the REQUIRED OUTER EVIDENCE heading. Naming `plan_review_report` or the `bug_` prefix to forbid reusing them is allowed (REQ-F-018).
- **R:** one entry per AC or record with `kind`, `pointer`, `summary`; limit source labels and self-review record go there; `gate_result.summary` holds only the summary line and verdict. **R2:** the bug entries sit alongside the required entries even though the required block says it "replaces the empty array"; the block itself is unchanged (TC-011).

### TC-023 TDD partial wording and PRECEDENCE (AC-19; REQ-F-012, 027)

- **Expected:** each family render contains `one test per TC assigned to this work item` (case-insensitive) and `any included skill`; each PRECEDENCE line contains `behavior-changing` and keeps "preserve the code and history".
- **Negative:** the partial does not claim a skill is included (`the included skills` absent from the partial source).
- **R:** refactor and prompt, documentation or configuration changes follow the verification table.

### TC-024 This branch complies with the edited-code rule (AC-21; REQ-F-032)

Entrypoint: the final candidate; run after the last rework task.

1. Before each split, record `go test -list . <package>` for `internal/templates` and `internal/config/workflow`.
2. `git diff --name-only --diff-filter=d d9427496 -- '*.go'`: every listed file has at most 500 lines (`wc -l`).
3. `golangci-lint run --config=.golangci-limits.yml --whole-files`: 0 issues. Also run it with `--new-from-rev=d9427496250d889dc7584d71778364b8a79ab766` if the pin has been re-pinned, so the measured set still covers step 2.
4. Manual nesting pass over the same files: no depth over 4; record the result per file.
5. `includes_test.go` is split into the three files and `workflow_multilevel_test.go` into the three files named in spec §How the rule applies to this branch (groupings are guidance; limits are binding). The after name list is a superset of the before list; removed names: none.

- **Negative:** no `//nolint` for funlen or gocyclo and no limit exception row added for this branch.
- **R2:** re-run steps 2-4 on the final candidate after task 012. Current sizes: `craft_rubric_test.go` 441 lines (59 of headroom for the R2 assertions; if they do not fit, put them in a new `_test.go` file), `development_prompts_test.go` 324, `workflow_tech_debt_test.go` 293, `dogfood_standards_test.go` 201 (TC-025 adds the most here). New subtests keep nesting at or under 4 and functions within funlen and gocyclo.

### TC-025 Limits config guard (AC-22; REQ-F-033(a))

- **Expected:** `internal/sharkdata/dogfood_standards_test.go` parses `.golangci-limits.yml` and asserts funlen `lines` 50, `statements` -1, gocyclo `min-complexity` 10, `new-from-rev` matching `^[0-9a-f]{40}$`; the Makefile limits invocation and the CI limits step each carry both `--config=.golangci-limits.yml` and `--whole-files` (checked on that invocation or step, not anywhere in the file).
- **Expected (R2, F-112):** the guard is a check over the contents of the three files. One test runs it on the real files and finds no violation. A table test derives each mutation below in memory from the real files' bytes, asserts the mutation changed the input, and asserts the check reports a violation: one named `t.Run` per row. Keep the table at package level or in helpers so each function stays within funlen 50.
- **Baseline assertions (R2):** `linters.default` is `none`; `linters.enable` is exactly `funlen` and `gocyclo`; no `linters.disable`; no `run.tests: false`; no `linters.exclusions` or `issues.exclude*` entry naming `funlen`, `gocyclo` or a test-file path; the Makefile limits line is not commented, has no leading `-`, and has no `|| true` or `; true`; the CI limits step has no `continue-on-error` and no `if:`.

| Group | Mutation rows (one `t.Run` each) |
|---|---|
| Values (built) | funlen `lines` 500; gocyclo 100; `statements` 0; short `new-from-rev` |
| Flags (built) | delete `--whole-files` from Makefile; delete the Makefile limits line; delete `--whole-files` from CI; delete the CI limits step |
| Linter set (**R2**) | remove funlen from `enable`; remove gocyclo; add a third linter; `default: standard`; `disable: [funlen]`; `disable: [gocyclo]`; `run.tests: false` |
| Exclusions (**R2**) | `linters.exclusions.rules` naming funlen; naming gocyclo; with a `_test\.go` path; `linters.exclusions.paths` with a `_test\.go` pattern; `issues.exclude-rules` naming funlen |
| Makefile (**R2**) | tab then `#` before the limits command (today's `makeRecipe` accepts this); `-` prefix, also as `@-`; `\|\| true` appended; `; true` appended |
| CI (**R2**) | `continue-on-error: true` on the limits step; `if: false` on the limits step |

- **On-disk evidence (QA):** apply at least one row of each group to the real file, see `go test ./internal/sharkdata/` fail, revert, `git diff` clean.
- **Negative:** the check reports nothing for the real files, so a comment-only header edit (TC-018) does not trip it.

### TC-026 Subtests in craft rubric tests (REQ-F-033(c))

- **Entrypoint:** `go test -v -run TestE31F01 ./internal/sharkdata/`.
- **Expected:** every table-driven loop in `craft_rubric_test.go` reports one named subtest per case (for example 11 banned-pattern subtests).
- **R:** read the file; no table loop without `t.Run`. No source-scanning test. **R2:** new rubric and deep-review cases follow the same rule.

### TC-027 Blocked prompts resume the parked step (AC-23; REQ-F-034) (R2)

**Static** (content-only), in `internal/templates/tech_debt_blocked_prompt_test.go`; one `t.Run` per entity (`bug`, `change`, `tech_debt`) and per reason variant (with, without):

- **Expected:** each render contains anchors R2-6 with the item's own ID, in place of the old last line.
- **Negative:** no `shark status advance`; no `--force`; no `<no value>`.
- **Mirror:** extend the TC-020f test so tech debt and bug each equal the change render with only the entity noun swapped. The new resume lines call it `the item` and never repeat the entity noun, which stays in line 1 only.
- **Unchanged:** regeneration leaves `task/blocked.golden` unchanged (TC-016). Review step, not test code: `git diff 5d848cab -- <b>/prompts/task/blocked.md` is empty.
- **R:** the two lines read in order (find the parked step, then set it). "Parked from" means the old status of the newest history row that entered `blocked`.

**Runtime** (QA, candidate binary). Unset `SHARK_DB_BACKEND`, `SHARK_DB_URL` and `SHARK_AUTH_TOKEN_FILE`; run `make shark` first (the scratch script reuses an existing `bin/shark`); create the project with `scripts/shark-scratch-env.sh` and confirm its backend is `local`.

| Case | Park from | Expected after the rendered commands |
|---|---|---|
| a (AC-23) | change at `development` | history newest row `development` to `blocked`; `status set <id> development --reason "..."` exits 0; status `development`; new history row `blocked` to `development` carries the reason |
| b | bug at `development` | same |
| c | tech debt at `triaged`, then at `in_progress` | returns to the parked step |
| d | any item at its first workable step | returns to that step |

- **Procedure:** reaching the starting step may use `status set --force` (setup only). Park with `status advance <id> --outcome blocked`. Take the commands from the `shark next <id> --json` prompt and substitute only `<parked step>` and the reason.
- **Counterfactual:** a second change parked from `development` and resumed with the old plain `status advance` lands at `code_review` (B091).
- **Pre-change evidence (2026-11-27, binary `5d848cab`, scratch project):** CC-001 and B001 parked from `development`, TD-001 from `triaged` and `in_progress`: each `status set` without `--force` returned to the parked step, and history recorded the `--reason` text in the new row's notes. Plain advance from `blocked` moved CC-001 to `code_review`. This answers the spec's second R2 risk bullet. Setup used `--force`, and the commands were typed from REQ-F-034, not rendered. Re-run on the candidate.

### TC-028 Named subtests for the round-2 loops (AC-24; REQ-F-033(c)) (R2)

- **Entrypoint:** `go test -v` as in the content-only table. Test structure only; no source-scanning test and no new red step.
- **Expected:**
  - `TestE31F01_TC019_...`: one named subtest per bundle source (checked-in, embedded fallback) containing the five prompt subtests, shaped `TC019_.../<source>/<prompt>`. No name ends in `#01` (today both sources reuse the prompt name).
  - `TestE31F01_TC020b_...`: `.../<step>/<outcome>` for 4 steps x 5 outcomes (loop at about line 261).
  - `TestE31F01_TC020c_...`: `.../<workable step>` for the loop at about line 277; `.../<non-workable step>/<outcome>` for the loop at about line 284.
- **Negative:** the same assertions run as before; no test name disappears from `go test -list`.
- **R:** apply G-14 when judging other loops.

## Integration scenarios

| Boundary | Verify | TC |
|---|---|---|
| Rubric to family prompts (`{{include:}}`) | Same checklist and severity for development and review; skill note precedes it | 005, 007, 008 |
| Rubric language scope to this repo's Go-only sentence (**R2**) | The standards sentence takes the form the rubric honors, so non-Go files are unmeasured | 005, 018 |
| Blocked prompt text to parking targets and status history (**R2**) | Rendered commands resume the parked step | 020b, 027 |
| Bug prompt evidence routing to runner admission (**R2**) | Added kinds do not collide with reserved kinds; gate ingestion looks up reserved kinds by name and has no kind allowlist (grep of `internal/runner`, `gatepersist`, `workercontrol`, 2026-11-27), so no new runtime TC | 011, 022, 016 |
| Rubric to review-code (relative path) to code_review renders | Reference resolves; render stable | 006, 016 |
| Rubric to deep-review workers and consolidator | Loaded, severity applied, stop when unreadable | 013, 017 |
| generate-standards and bootstrap to project standards to prompts | Heading names agree (`Quality Limits`, `Limit exceptions`, `Testing Standards`) | 001, 003, 004, 008 |
| Standards limits to golangci config to Makefile and CI | Values, scope and flags agree | 014, 015, 018, 025 |
| Worker `blocked` to gate ingestion to tech-debt parking to run loop | Parks, no re-dispatch, human resume | 020 |

No epic UAT scenarios exist to reference.

## Test infrastructure

| Need | Existing | New |
|---|---|---|
| Scan and rubric tests | `internal/sharkdata/craft_rubric_test.go` | extend; stay at or under 500 lines (**R2:** 441 now) |
| Render and literal tests | `internal/templates/development_prompts_test.go` | extend |
| Workflow lifecycle | `internal/config/workflow/workflow_multilevel_test.go` | split into `workflow_bug_change_test.go`, `workflow_tech_debt_test.go` |
| Run-loop parking | fakes and mocks in `internal/runner/controller_gate_ingest_test.go`, `controller_harness_test.go` | `controller_tech_debt_blocked_test.go` |
| Config guard | `internal/sharkdata/dogfood_standards_test.go` | extend |
| Goldens | `internal/cli/commands/next_golden_test.go` | `tech_debt/blocked.golden` |
| Partial tests | `partials_test.go`, `orchestrator_partials_test.go` | update expectations |
| Blocked prompt tests (**R2**) | `internal/templates/tech_debt_blocked_prompt_test.go` | extend (TC-027) |
| Scratch project (**R2**) | `scripts/shark-scratch-env.sh` | none |

Automated tests keep the `TestE31F01_TC0xx_` naming. No database or service mocks are needed. This repo has no `shark-data/overrides/`.

Before the next deep-review round, install Shark from the candidate (`make install-shark`), or the REQ-F-009 precondition stops the review.

## Cross-feature contract tests (I-##)

None. E31 has one feature and declares no I-##.

## Cross-epic integration tests (X-##)

None. The spec declares no X-##, and the product map has no E31 row; there is nothing to defer.

## Codex Test-Plan Red-Team

**Verdict:** not run. The dispatch supplied no `codex_command`, and the worker ownership contract bars external AI CLIs. Non-blocking. Author self-check (not independent): AC-02 closed by T01-T11; AC-20 covers values, invalid, recovery and state addition; **R2:** AC-23 has static and runtime coverage, AC-22 mutations are automated, AC-24 names its loops; negative cases on revised ACs; runtime TCs 014, 015, 020, 024, 025, 027 carry caller paths; the rest are content-only.

Residual risk: R items rely on code review and QA judgment. Reviewers should record a per-item result.

## Recommendations

- [x] Ready for rework task generation (007-011 built; **R2:** task 012, spec §Sizing and task split). Task 012 regenerates goldens for its own edits.
- [x] **R2:** G-11 follow-up done: spec corrected, REQ-F-034 added, engine half tracked as B091.
