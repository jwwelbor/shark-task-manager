# Test Plan: E31-F01 - Shift quality left into development

**Created:** 2026-11-27 (revised 2026-11-27 for the amended spec)
**Spec:** [spec.md](spec.md) (stands in for both feature PRD and task spec)
**Other inputs:** [plan.md](plan.md), [research-report.md](research-report.md)
**Status:** APPROVED (G-11 is a spec wording error with binding handling below)

Tasks 001-006 are built (candidate `966333ea`). TC IDs are stable; TC-020 to TC-026 are new. Expected Output marked "(revised)" or rewritten here replaces the earlier text.

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
| G-11 | Spec (capability map and lifecycle table) says parking resume is computed from history. The engine does not do that: `steps.go` only comments it; the parking flow is every workable step in sorted order and `status advance` takes the first. Observed 2026-11-27: change and bug parked from `research` advanced to `code_review`; tech debt (probe workflow) advanced to `identified` | REQ-NF-007 bars a Go fix, and AC-20 does not require history resume, so build as specified. TC-020e asserts real behavior. Resume to the parked step with `status set <key> <step>`. For tech debt the plain advance only redoes earlier steps; for bug and change it can skip development (existing behavior, which consultation blocks will now reach). Follow-up for the parent: file a bug for history-based resume and the blocked prompts' resume instruction, and correct the spec wording |
| G-12 | The edited-code rule makes any edit to an over-limit Go test file force a refactor (for example `embed_test.go`, `b036_prompt_crawl_test.go`, `next_test.go`, `controller_gate_ingest_test.go`) | Put new tests in new or within-limit files. The suite was not run with the new YAML, so task 009 runs it first and lists failing files; an over-limit file needing an edit (candidates: `viewer_service_test.go`, `embed_test.go`, `next_test.go`, `run_apply_result_test.go`, `controller_gate_ingest_test.go`) is a refactor or consultation case. TC-024 |
| G-13 | AC-22 names Makefile mutations only; REQ-F-033(a) also covers CI | TC-025 mutates both |

### Traceability

| REQ | AC | TC | Rework task |
|---|---|---|---|
| F-001, F-002, F-025 | 01 | 001 | 007 |
| F-005, F-008, NF-001, NF-002 | 02 | 002 | 007 |
| F-003 | 03 | 003 | regression (built) |
| F-004, F-030 | 04 | 004 | 011 |
| F-005 to 007, F-025, 026, 029, NF-005 | 05 | 005 | 007 |
| F-008, F-009 | 06 | 006 | 007 |
| F-012, F-014, F-029 | 07 | 007 | 008 |
| F-010 to 016, 018, 019, 026, 027, 033(b) | 08 | 008 | 008 |
| F-017, F-019 | 09 | 009 | 008 |
| F-020 | 10 | 010 | 008 (regression) |
| F-019, NF-004 | 11 | 011 | 008 |
| F-021 | 12 | 012 | 011 |
| F-009 | 13 | 013 | 007 |
| F-023, F-024 | 14 | 014, 015 | 010 |
| NF-004, F-031 | 15 | 016, 017 | every task; closes after last |
| F-022, F-023 | 16 | 018 | 010 |
| F-025, 026, NF-006 | 17 | 021 | 007 (rubric), 008 (prompts) |
| F-018 | 18 | 022 | 008 |
| F-012, F-027 | 19 | 023 | 008 |
| F-031, NF-003, NF-007 | 20 | 020 | 009 |
| F-032 | 21 | 024 | 008 (includes), 009 (workflow); closes after last |
| F-033(a) | 22 | 025 | 010 |
| F-033(c) | n/a | 026 | 007 |
| NF-003, NF-006 | 05, 07 | 019 | 008 |

A task writes tests only for the TCs (or the named part) in its row.

## Acceptance criteria review

All 22 ACs are testable. AC-02 is closed by the pattern list. The one behavior-bearing field is the tech-debt `status` (AC-20): state-transition technique, TC-020.

## ISTQB technique and ISO 25010 coverage

Columns: Fn functional, Pf performance, Co compatibility, Us usability, Re reliability, Se security, Ma maintainability, Po portability. N/A cells are justified in the last column.

| AC | Technique | Fn | Pf | Co | Us | Re | Se | Ma | Po | N/A reason |
|---|---|---|---|---|---|---|---|---|---|---|
| 01 | Contract surface enumeration | TC-001 | N/A | TC-019 | TC-001 R | N/A | N/A | TC-002 | TC-001 | text only: no runtime, input or secrets |
| 02 | Attack-class enumeration (phrasing classes) + EP (allowed file vs others) | TC-002 | N/A | N/A | N/A | TC-002 counterfactual | N/A | TC-002 | TC-002 | static scan |
| 03 | Contract surface enumeration | TC-003 | N/A | TC-019 | TC-003 R | N/A | N/A | TC-002 | TC-003 R | text only |
| 04 | Contract surface enumeration | TC-004 | N/A | TC-004 | TC-004 R | N/A | N/A | TC-004 | N/A | text only; stack-neutral heading check |
| 05 | Contract surface enumeration + BVA (170 lines) | TC-005 | TC-005 (size) | TC-019 | TC-005 R | TC-005 | TC-019 | TC-002 | TC-005 R | |
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
| 22 | Mutation of guard inputs (deterministic) | TC-025 | N/A | TC-015 | N/A | TC-025 | N/A | TC-025 | N/A | config guard |

**Coverage gaps:** nesting depth and file length are review-enforced (TC-024 steps 2 and 4 for this branch); CI uses golangci-lint `latest` while local pins v2.9.0 (TC-015 observes the PR run); history-based resume is not implemented (G-11).

## Observability design

| Behavior | Runtime evidence | Assertion |
|---|---|---|
| Lint enforcement | golangci output naming `funlen` or `gocyclo` with file and line; CI lint step result | TC-014, TC-015 |
| Self-review and AC walk | Summary line `ACs verified: …; red-first: …; commit: …` | TC-008 (instruction present) |
| Limit source | Label text in self-review and review findings | TC-005, TC-007 |
| Tech debt parks | Status `blocked`, history row, run outcome `paused`, `next` action `pause` | TC-020d, e |
| Consultation | `blocked` reason naming the six REQ-F-026 fields | TC-021 |
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
| 025 | `go test ./internal/sharkdata/` reading the real config, Makefile and CI file | none | Fixture copies of those files | Limits loosened or `--whole-files` removed while tests stay green |

Content-only cases (justification for each: **content-only**, no deterministic Go behavior changes):

| TC | Renderer or direct-file entrypoint |
|---|---|
| 001, 002, 003, 004, 006, 012 | `sharkdata.ReadEmbedded(path)` or a walk of the embedded tree |
| 026 (test-structure only) | `go test -v -run TestE31F01 ./internal/sharkdata/` |
| 005 | `sharkdata.ReadEmbedded` and `templates.NewIncludeResolverWithEmbed("").Resolve("{{include: skills/quality/context/craft-rubric.md}}")` |
| 007-011, 019, 021-023 (TC-008 also `sharkdata.ReadEmbedded`) | `templates.NewOrchestratorRenderer(canonicalPromptsDir).Render("<entity>/<status>.md", vars)`, the renderer `shark next` uses |
| 013, 018 | `os.ReadFile` from the repo root |
| 016, 017 | `make fmt && make lint && make test`; Python `unittest` in `<dr>/scripts` |

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

- **Expected:** file exists; at most 170 lines (170 passes, 171 fails); resolves through the include resolver; contains the labels `generic default — not declared in local standards`, `declared in local standards (§Quality Limits)` and `proposed in local standards, unconfirmed`; contains `../context/review-rubric.md`; states 10, 50, 500 and 4 (TC-002); contains the `Limit exceptions` rule and the checklist.
- **Negative:** no `{{`; none of the command strings forbidden by `next_test.go` `TestRenderedDispatchPromptsDoNotTellWorkersToTransitionSharkState`; the rooted `skills/quality/context/review-rubric.md` is absent; no case-insensitive `substantially expand`; `TestEmbedded_SkillsContainNoBareSharkCLIRefs` passes.
- **Rework note:** the existing assertions on the rooted path and the 150-line limit are replaced.
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
- **Expected B:** the READ section contains `CLAUDE.md`, `AGENTS.md`, `docs/architecture/coding-standards.md`, `patterns-catalog.md`, `tech-stack.md` exactly once each (count within READ only), and mentions Quality Limits (with Limit exceptions) and Testing Standards; rubric and `_tdd_process` markers once each; summary token `ACs verified:`; no unresolved directive.
- **Negative:** the bug prompt has no `CLAUDE.md for coding standards` item. Asserting the eight literals on the render is not acceptable: the included rubric could satisfy them.
- **R, per prompt:** all-empty gate sources give `blocked` plus bootstrap with no implementation, and one runnable source runs; limits applied first pass; each edited code file is measured in full, and when the measuring command covers only new code the rest is measured by tool or hand; checklist findings fixed or reasoned; pre-existing failure outside the changed files is fixed or routed via triage and the gate stays unmet (`fail` in envelope prompts), while an overage inside an edited file is handled by the edited-code rule; envelope prompts lead `gate_result.summary` with the summary line; TC rules apply only when a test plan exists; G-8 values.

### TC-009 Exit gates (AC-09)

- **Expected:** the task EXIT GATE contains (a) `for each AC ID`; (b) `Every code file you edited is within the resolved limits`, with measured in full, overages that predate the change, source label, and the covering limit exception; (c) `## Scope` "Files to modify" or justification; (d) `Self-review is recorded with no unresolved finding`. Bug, change and both tech-debt prompts contain (a), (b) and (d). Existing own items remain.
- **Negative:** no source template or render contains `Every new or changed function and file`.
- **R:** item (c) is task-only and covers files created by a limit refactor.

### TC-010 Shortcuts tightened (AC-10)

- **R on rendered `change/development.md`, `tech_debt/triaged.md`, `tech_debt/in_progress.md`:** every "advance immediately" or equivalent resume path requires quality gate, self-review and exit gate; the gate-command `blocked` rule applies before it.
- **Negative:** no sentence allows advancing on gate pass alone.

### TC-011 Bug contracts preserved (AC-11)

- **Expected:** `git diff d9427496 -- <b>/prompts/bug/development.md` shows no removed or altered text (the only permitted removals are the separate `CLAUDE.md for coding standards` READ item with its renumbering, and the PRECEDENCE line) in: the approved-plan and coverage-index paragraphs, the decision-constraint paragraph, `DECISION:`, the `route_rework` role line, `_gate_result_directive`, and REQUIRED OUTER EVIDENCE. READ and EXIT GATE may gain lines.
- **Automated:** `gate_result_role_prompt_consistency_test.go` and `TestRenderedPromptLint` (bug evidence kinds) pass.

### TC-012 Sweep records standards (AC-12)

- **Expected:** `defect-class-sweep.md` Guard selection names `coding-standards.md`, contains `proposed — confirm`, and names each craft category (one `t.Run` each): `ALTITUDE`, `CODING-STANDARD`, `COMPLEXITY`, `DIP`, `DRY`, `ERROR-HANDLING`, `IDIOM`, `ISP`, `LOGGING`, `LSP`, `NAMING`, `OCP`, `SIMPLIFICATION`, `SRP`, `TESTS`, `TYPES`, `YAGNI`. Existing `TestDefectClassSweep*` tests pass. Match whole tokens (`SRP` must not be satisfied inside another word).
- **R:** the section states that `ARCH`, `ARCHITECTURE`, `CONTRACT`, `CORRECTNESS`, `REMOVED-BEHAVIOR`, `RISK`, `ROUNDTRIP`, `SECURITY`, `SIBLING`, `WIRING` never edit standards; the 17 plus 10 equal the `review-code.md` §B categories; "proposed — confirm" applies unless the owner confirms in the same session; linter rule where supported; the entry never replaces the verified guard.

### TC-013 Deep review (AC-13)

- **Expected:** `<dr>/SKILL.md` reviewer load instruction names `context/craft-rubric.md`; `consolidator.md` and `angle-d-cleanup.md` reference the rubric; no T01-T11 match in `SKILL.md` or references.
- **Expected (revised):** none of `SKILL.md`, `consolidator.md`, `angle-d-cleanup.md` matches case-insensitive `substantially expand`.
- **R:** `SKILL.md` tells the host, before any runner dispatches a reviewer, to confirm the rubric is readable by either route and otherwise to stop, report the review incomplete and name the missing rubric (no review without it); Blocker and Non-blocker rows and the angle-D text apply the severity table by reference; `review_workflow.js` does not contradict this.

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

- **Steps:** regenerate with `go test ./internal/cli/commands/ -run TestRenderedPromptsGolden -update`; re-pin `task/development.md` in `productCriticalPathGuardBaselineSHA256` (in whichever file holds it after the split); update `_tdd_process` expectations; run `make fmt && make lint && make test`.
- **Expected:** all pass, none skipped. The golden diff holds only files traceable to an edited bundle file, including the new `tech_debt/blocked.golden`; review it line by line (`-update` alone proves nothing).
- **Named regressions:** `TestRenderedPromptsGolden`, `TestF02_AC3_OutputEqualsGoldenCorpus`, `TestRenderedPromptLint`, `TestB036_ExhaustivePromptCrawl_WritesVerificationArtifacts` (covers the new prompt), `TestTwelvePromptsAddGuardAsPureAddition`, `TestDevelopmentPromptGuardSitsInCompletionReportingSection`, `TestRenderedDispatchPromptsDoNotTellWorkersToTransitionSharkState`, the gate-result role consistency test, `TestEmbedded_SkillsContainNoBareSharkCLIRefs`, `TestEmbedded_BundleReferencesResolve`, `internal/config/workflow` and `internal/runner` packages.
- **Edge:** `tech_debt/blocked.md` is a pause prompt and stays out of the dispatch-prompt ban list, so its `shark status advance` line (mirrored from change) is expected.

### TC-017 Review-output and deep-review regressions (AC-15)

- **Expected pass:** `TestConsolidatorRoutesNonBlockersThroughTriage`, `TestUnattendedReviewDispositionHandoff`, the rest of `review_output_policy_test.go`, `TestEmbedded_ReviewFindingsRouteThroughTriage`, `TestEmbedded_NonBlockersCarryStableFingerprint`.
- **Also:** `cd <dr>/scripts && python3 -B -m unittest discover -p 'test_*.py'` (not part of `make test`; pre-feature baseline: 114 tests pass in about 210 s).

### TC-018 Dogfood docs (AC-16)

- **Expected:** `tech-stack.md` has `## Quality Gate` with columns Step, Command, When and `make fmt`, `make lint`, `make test`; no leftover `### Quality Gate (Mandatory)`. `coding-standards.md` has `## Quality Limits` with 10/gocyclo/tool, 50/funlen/tool, 500/review, 4/review, production and test, `make lint`. `testing.md` forbids shared package-level command or flag state between tests.
- **Expected (revised):** the Quality Limits text states that `make lint` measures every function in each Go file changed since the pinned revision, and that file length and nesting depth are checked on every edited file by self-review and code review.
- **Negative:** the old "only functions whose declaration line changed" baseline wording is gone.
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
- `status advance <key>` (no outcome) moves to `identified`, the first entry of the parking flow, **not** the parked-from step (G-11).
- Park again, then `status set <key> research` (no `--force`): status `research`.
- A second item parked from `identified` (`--outcome blocked`) reaches `blocked`.
- **Evidence so far:** a 2026-11-27 scratch run with the pre-change binary and a probe workflow carrying this parking step produced these results. Re-run on the candidate.

**f. Prompt:** `tech_debt/blocked.md` renders with and without `blocked_reason`; mirrors `change/blocked.md` apart from the entity noun; golden present (TC-016).

**g. Compatibility:** no migration; items in the seven existing statuses keep them (diff review: no Go production change, no data rewrite).

**Test updates:** `workflow_multilevel_test.go` expects 7 statuses in two places and 4 transitions per workable step, and its StatusFlow comment says blocked routes to `identified`. Update all of them (8, 5, comment) in the split files (TC-024).

### TC-021 Limit exceptions and architect consultation (AC-17; REQ-F-025, 026, NF-006)

- **Expected:** `craft-rubric.md` and each rendered family prompt contain `consult the architect` (prompt) and the exception rule (rubric, included).
- **R:** exception covers only a pre-existing unit, only at or below the recorded value, added or changed only by architect or owner; new units never excepted; a worker never records one; a worker in doubt returns `blocked` naming file, unit, limit and source label, measured value, considered refactor and safety concern (`gate_result.no_kickback_reason` in envelope prompts); the overage is not left in place; rework re-reads the decision (TC-008 literal 5). No text permits bypassing the gate.
- **Edge (R):** value equal to the recorded value is covered; one above needs a new decision.

### TC-022 Evidence routing in envelope prompts (AC-18; REQ-F-018)

- **Expected:** rendered `change/development.md`, `tech_debt/triaged.md`, `tech_debt/in_progress.md` contain `evidence[]` within the exit-gate or summary-line instructions.
- **Negative:** their EXIT GATE no longer asks for the compact per-AC walk inside `gate_result.summary`.
- **R:** one entry per AC or record with `kind`, `pointer`, `summary`; limit source labels and self-review record go there; `gate_result.summary` holds only the summary line and verdict. Bug REQUIRED OUTER EVIDENCE is unchanged (TC-011).

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

### TC-025 Limits config guard (AC-22; REQ-F-033(a))

- **Expected:** `internal/sharkdata/dogfood_standards_test.go` parses `.golangci-limits.yml` and asserts funlen `lines` 50, `statements` -1, gocyclo `min-complexity` 10, `new-from-rev` matching `^[0-9a-f]{40}$`; the Makefile limits invocation and the CI limits step each carry both `--config=.golangci-limits.yml` and `--whole-files` (checked on that invocation or step, not anywhere in the file).
- **Mutations** (apply one, `go test ./internal/sharkdata/` must fail, revert, `git diff` clean): funlen `lines` 500; gocyclo 100; `statements` 0; short `new-from-rev`; delete `--whole-files` from Makefile; delete the Makefile limits line; delete `--whole-files` from CI; delete the CI limits step.

### TC-026 Subtests in craft rubric tests (REQ-F-033(c))

- **Entrypoint:** `go test -v -run TestE31F01 ./internal/sharkdata/`.
- **Expected:** every table-driven loop in `craft_rubric_test.go` reports one named subtest per case (for example 11 banned-pattern subtests).
- **R:** read the file; no table loop without `t.Run`. No source-scanning test.

## Integration scenarios

| Boundary | Verify | TC |
|---|---|---|
| Rubric to family prompts (`{{include:}}`) | Same checklist and severity for development and review; skill note precedes it | 005, 007, 008 |
| Rubric to review-code (relative path) to code_review renders | Reference resolves; render stable | 006, 016 |
| Rubric to deep-review workers and consolidator | Loaded, severity applied, stop when unreadable | 013, 017 |
| generate-standards and bootstrap to project standards to prompts | Heading names agree (`Quality Limits`, `Limit exceptions`, `Testing Standards`) | 001, 003, 004, 008 |
| Standards limits to golangci config to Makefile and CI | Values, scope and flags agree | 014, 015, 018, 025 |
| Worker `blocked` to gate ingestion to tech-debt parking to run loop | Parks, no re-dispatch, human resume | 020 |

No epic UAT scenarios exist to reference.

## Test infrastructure

| Need | Existing | New |
|---|---|---|
| Scan and rubric tests | `internal/sharkdata/craft_rubric_test.go` | extend; stay at or under 500 lines |
| Render and literal tests | `internal/templates/development_prompts_test.go` | extend |
| Workflow lifecycle | `internal/config/workflow/workflow_multilevel_test.go` | split into `workflow_bug_change_test.go`, `workflow_tech_debt_test.go` |
| Run-loop parking | fakes and mocks in `internal/runner/controller_gate_ingest_test.go`, `controller_harness_test.go` | `controller_tech_debt_blocked_test.go` |
| Config guard | `internal/sharkdata/dogfood_standards_test.go` | extend |
| Goldens | `internal/cli/commands/next_golden_test.go` | `tech_debt/blocked.golden` |
| Partial tests | `partials_test.go`, `orchestrator_partials_test.go` | update expectations |

Automated tests keep the `TestE31F01_TC0xx_` naming. No database or service mocks are needed. This repo has no `shark-data/overrides/`.

Before the next deep-review round, install Shark from the candidate (`make install-shark`), or the REQ-F-009 precondition stops the review.

## Cross-feature contract tests (I-##)

None. E31 has one feature and declares no I-##.

## Cross-epic integration tests (X-##)

None. The spec declares no X-##, and the product map has no E31 row; there is nothing to defer.

## Codex Test-Plan Red-Team

**Verdict:** not run. The dispatch supplied no `codex_command`, and the worker ownership contract bars external AI CLIs. Non-blocking. Author self-check (not independent): AC-02 closed by T01-T11; AC-20 covers values, invalid, recovery and state addition; negative cases on revised ACs; runtime TCs 014, 015, 020, 024, 025 carry caller paths; the rest are content-only.

Residual risk: R items rely on code review and QA judgment. Reviewers should record a per-item result.

## Recommendations

- [x] Ready for rework task generation (007-011, spec §Sizing and task split). Each task regenerates goldens for its own edits.
- [ ] Parent follow-up for G-11.
