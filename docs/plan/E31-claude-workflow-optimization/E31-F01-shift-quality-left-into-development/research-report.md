---
research_schema: 2
rigor: standard
categories: [workflow_operations, documentation, backend]
related_work: true
---

# E31-F01 Research Report: Shift quality left into development

## Scope

Brownfield check of [plan.md](plan.md), which is the working definition of this feature. `feature.md` and `../epic.md` are unfilled templates. E31 has no parent research report and no sibling features. The epic description cites `docs/workflow/optimization-backlog.md`, which does not exist.

In scope: the embedded bundle (`internal/sharkdata/default_data/`, abbreviated `<bundle>`), the host deep-review skill, bundle tests, and this repo's standards and lint config. The external "lightcycle" prompt was not read.

Vocabulary, mapped to Shark's actual contracts:

| Plan term | Actual contract |
|---|---|
| Task `files:` list | No such field. Task files use `## Scope` with "Files to modify / create" (`<bundle>/prompts/feature/task_generation.md`) |
| TC `Action` / `Expect` | Test cases use `Preconditions`, `Input`, `Expected Output` (`<bundle>/skills/quality/workflows/test-planning.md`) |
| "Shared rubric partial" | Two files already carry rubric roles: `review-code.md` (canonical craft rubric for deep review) and `context/review-rubric.md` (finding admission and severity) |
| Quality Gate | Commands section in the project's `tech-stack.md` |
| Quality Limits | New term; no existing section or field |
| Completion summary of 175 characters or fewer | No source found in the prompts or run verb searched; treat as unverified |

## Research checklist

- [x] `scope_vocabulary` — Evidence: docs/plan/E31-claude-workflow-optimization/E31-F01-shift-quality-left-into-development/plan.md
- [x] `affected_implementation_or_contract` — Evidence: internal/sharkdata/default_data/prompts/task/development.md
- [x] `related_work` — Evidence: docs/plan/changes/CC-047.md
- [x] `pattern_contract` — Evidence: internal/sharkdata/default_data/skills/quality/workflows/review-code.md
- [x] `dependency_impact` — Evidence: internal/templates/includes_test.go

## Capability map

The feature mainly **extends** existing capabilities. The one **new** capability is declared, project-local quality limits.

| Capability / related work | Decision | Note |
|---|---|---|
| Parent E31 shipped work (rewritten code-review craft prompt with triage tail, 2-strikes loop guard, augmented test planning; per epic description) | EXTEND | No parent research report exists |
| Quality Limits in generated standards and bootstrap marker | NEW | Added inside the existing generator: extend `generate-standards.md`, `standards-template.md`, `research/workflows/bootstrap.md` |
| Local-standards loading for gate commands | REUSE | The source list in the task, bug, change, tech-debt, QA and feature-review prompts already names `tech-stack.md`, `coding-standards.md` and others |
| Craft review rubric (`review-code.md`) | EXTEND | Share its checklist with development; do not add a competing rubric |
| Impact-based finding admission (B088, PR #297; `review-rubric.md`) | CONTRADICTS | See Finding 2 |
| Report and coverage contracts (B085, PR #298) | REUSE | Do not change report shape |
| Defect-class sweep (E34-F06) | EXTEND | It already selects a guard from the project toolchain; add "record a local standards rule" |
| Mandatory dev quality gate (CC-047, draft) | EXTEND | Task, bug and tech-debt prompts already require it, and change development does too. CC-047 looks largely covered; link it to this feature or close it (owner decision) |
| Caller-path compliance from development (CC-044 item 1) | REUSE | Task exit gate already requires it; CC-044 stays separate |
| Complexity not lint-enforced (TD-215, resolved) | EXTEND | TD-215 noted the gap; Scope E closes it |
| `_tdd_process` partial | CONTRADICTS | See Finding 4 |
| Plan claim that this repo lacks a Quality Gate | CONTRADICTS | See Finding 3 |

No sibling features exist under E31.

## Findings

### 1. Root cause is confirmed, with a second cause

- `review-code.md:13,144` defaults to cyclomatic 10, file 500, function 50, nesting 4. `implementation/context/coding-standards.md:259` states under 50 lines. `task/development.md` states no limits and has no self-review step.
- B082 round 4 blocked on a 61-line function and records that `coding-standards.md` documents no threshold (`docs/review/bugs/b082/code-review-2026-11-19-B082-r4.md`). Round 5 blocked on a 53-line test and an ignored close error.
- Second cause: the round 5 close error and the round 7 assertion style broke rules that were **already documented** (`docs/architecture/coding-standards.md` Error Handling; `docs/architecture/coding-standards/testing.md` require/assert). The task prompt's READ list names only CLAUDE.md for standards; `coding-standards.md` appears later, for test architecture and gate commands. Loading standards (Scope C) matters as much as declaring limits.
- B111 reports (v6 to v11 in `docs/review/bugs/b111/`) show repeated test-design findings. They review a Python backend and B111 is not an entity in this database; rounds 1 to 5 are not in that folder.

### 2. The bundle already conflicts on size and complexity severity

| Source | Rule |
|---|---|
| `review-code.md` Step 7 | File and function size are "guide, not hard rule", yet new complexity above threshold is a blocker |
| `review-code.md` Step 11, `review-rubric.md` (B088) | Do not create findings from rubric thresholds; require evidenced impact |
| `skills/shark-rider/skills/deep-review/references/angle-d-cleanup.md:45` | Size is a judgment guide, not an automatic blocker |
| `.../deep-review/references/consolidator.md:63` | "new complexity over threshold" is a blocker |

B088 merged 2026-11-26, after B082 (2026-11-19), so it may already reduce the B082-r4 pattern. This was not tested. Plan principle 3 ("severity rules are unchanged") preserves the conflict. Specification must choose one rule. Recommendation: a **locally declared** limit is a documented standard and may block; a **generic default** is a signal that needs evidenced impact. This follows B088, the newer contract, and `review-code.md`'s own "do not invent standards" rule.

### 3. Plan corrections

- `docs/architecture/tech-stack.md:80` already has `### Quality Gate (Mandatory)`. It is an H3 under Build & Quality Tools, not the `## Quality Gate` table that bootstrap generates. Scope E should normalize it, not add it.
- This repo's `coding-standards.md` is an index with sub-files. Limit and testing-convention resolution must allow linked files.
- The "no language or tool names in rendered prompts" check fails on existing content: `bug/development.md` includes `skills/debugging/SKILL.md`, which names React and Vue (lines 30-31). `review-code.md` also names languages. Narrow the criterion to "introduces no stack-specific gate commands or limits", or add that cleanup to scope.

### 4. Scope gaps

- No shipped prompt invokes `_tdd_process`; only tests reference it. Rewriting it changes no dispatch unless the plan also wires it in or removes it.
- `tech_debt/triaged.md` and `tech_debt/in_progress.md` are implementation prompts with the same gate and no self-review. The plan omits them; decide in or out.
- `change/development.md` and both tech-debt prompts let existing work advance once the gate passes. That bypasses a new self-review. The plan rules out importing such a shortcut but does not address these existing ones.

### 5. Hardcoded threshold inventory (Scope B)

- Craft limits: `review-code.md:13,144`; `implementation/context/coding-standards.md:259`; cyclomatic 10 in `quality/context/code-review-reference.md` and `angle-d-cleanup.md:39-43`.
- Coverage: `implementation/context/validation-gates.md:241-251,516`; `testing-requirements.md:473`; `implement-backend.md:492`; `implement-api.md:506`; `implement-tests.md:665,686`.
- Document-length limits (task, design docs) are artifact rules, not code quality gates; leave them.

The search was pattern-based over the embedded bundle and shared host skills, so it may not be exhaustive.

### 6. Pattern fit and include behavior

- Dispatch prompts resolve `{{include:}}` recursively; `{{template}}` partials are Go templates for prompts.
- `shark skill get` also resolves includes unless `--raw` is passed (`internal/services/content_bundle_service.go:142-149`). Deep review reads the rubric this way (`deep-review/SKILL.md:202`), but also permits an extracted file, which would show the directive unresolved. It would not expand a `{{template}}` call.
- No skill file currently uses `{{include:}}`. Recommendation: put the shared checklist and defaults in one file under `skills/quality/context/`, include it from development prompts, and reference it by relative path from `review-code.md`, as `code-review-reference.md` is referenced now. Verify that deep-review workers load it.

### 7. Dependency impact

| Dependency | Effect |
|---|---|
| `internal/templates/includes_test.go` | SHA-256 pin on `task/development.md` (guard removed); any rewrite needs a re-pin. Guard must stay after "When done: stop and summarize" |
| `internal/cli/commands/next_test.go` (about line 560) | Dispatch prompts, including development ones, must not contain status or claim command strings |
| `internal/cli/commands/gate_result_role_prompt_consistency_test.go` | Bug and change outcome wording must match workflow roles |
| `internal/templates/{partials,orchestrator_partials,task_execution_templates}_test.go` | Reference `_tdd_process` |
| `internal/sharkdata` tests | Include and reference resolution; no bare CLI invocations in craft skills including `quality/` |
| `review-code.md` consumers | Feature, bug, change and shared code review prompts, plus deep review |
| Project `shark-data/overrides/` | An override of a changed prompt or skill hides the update |

### 8. Dogfood baseline

Measured with golangci-lint 2.9.0, only these linters, temporary config outside the repo, thresholds gocyclo 10, funlen 50 lines (statements off), nestif 5:

| Linter | Production | Test |
|---|---|---|
| gocyclo | 374 | 488 |
| funlen | 99 | 683 |
| nestif | 106 | 27 |

Plain thresholds would break `make lint`, so use `new-from-rev` or an equivalent baseline. `nestif` scores complexity, not nesting depth 4. Current `.golangci.yml` excludes `Close` calls from errcheck and disables errcheck in tests, so lint would not catch the B082-r5 close error.

### Risks and unknowns

- Open decision (block vs. gap when no gate commands exist): greenfield readiness 2 writes a `tech-stack.md` whose Quality Gate is "To be defined". Blocking is compatible with the other fallbacks only if it triggers after all listed sources are empty.
- Projects bootstrapped earlier will have no limits; defaults must work without a migration.

## Decisions

Recommendations for specification; none is owner-approved beyond the entity's existing decision notes.

1. Extend the existing review rubric and standards generator; create one shared context file, not a second rubric.
2. Resolve the severity conflict as in Finding 2 and apply it to `review-code.md` and `consolidator.md` together.
3. Make standards loading explicit in every in-scope development prompt.
4. Decide tech-debt prompts, existing "already done" shortcuts, and `_tdd_process` wiring.
5. Rewrite the `files:`, Action/Expect and language-name criteria to match actual contracts.
6. Size remains L; split at task generation.

## Sources

- `plan.md`, `feature.md`, `../epic.md`; E31-F01 notes 5521-5523
- `internal/sharkdata/default_data/`: `prompts/{task,bug,change,tech_debt}/`, `prompts/_partials/`, `skills/quality/`, `skills/implementation/`, `skills/research/workflows/`
- `skills/shark-rider/skills/deep-review/`
- `internal/services/content_bundle_service.go`, `internal/templates/`, `internal/sharkdata/embed_test.go`
- `docs/architecture/{tech-stack.md,coding-standards.md,coding-standards/testing.md}`, `.golangci.yml`
- `docs/review/bugs/b082/` (r4, r5) and B082 notes; `docs/review/bugs/b111/`
- Shark records CC-047, CC-044, TD-215, B085, B087, B088; git commits c233d643, fda465cb

RECOMMENDED OUTCOME: standard
