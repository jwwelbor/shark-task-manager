# E31-F01 Plan — Shift quality left into development

## Problem

Code review keeps finding craft defects that the development step could have
caught. B082 took 8 review rounds; rounds 4–7 were function length (61 and 53
lines vs a 50 limit), a discarded `Close` error, test isolation (shared
package-level command state) and assertion style. B111 took 11 rounds, mostly
on test design.

Root cause: development and review grade against different rubrics.

- `skills/quality/workflows/review-code.md:13,144` hardcodes limits
  (cyclomatic 10 / file 500 / function 50 / nesting ≥4). B082-r4 records that
  no threshold existed in the project's `coding-standards.md`, so the reviewer
  used its own default.
- `skills/implementation/context/coding-standards.md:259` hardcodes
  `< 50 lines per function`.
- `prompts/task/development.md` has no self-review and no limits in its exit
  gate ("follows CLAUDE.md patterns", "no unrelated changes" only).

Reference: an external development prompt (lightcycle `development.md`) that
loads project standards first, measures limits on the first pass, and treats
"QA should not discover basic failures" as the development step's job.

## Principles (constraints on every change)

1. **Language-agnostic product.** Embedded prompts and skills never name a
   language, linter or test framework. Examples in templates may list
   several stacks side by side, as the existing Quality Gate table does.
   Generic numeric defaults are allowed only in the single fallback location
   defined in principle 3.
2. **Local standards are the only authority for gates and quality.** Commands,
   limits and conventions come from the project's own standards, which
   bootstrap (`/shark-rider project bootstrap`) establishes and maintains:
   `docs/architecture/tech-stack.md` (Quality Gate), and
   `docs/architecture/coding-standards.md` (rules, review rubric, limits).
   CLAUDE.md / AGENTS.md and build targets remain fallbacks for commands only.
3. **Local first, generic defaults as fallback.** A limit declared in local
   standards always wins. Only when a limit is not declared do development and
   review fall back to generic, language-neutral defaults (cyclomatic 10,
   function 50 lines, file 500 lines, nesting 4). Those defaults are defined
   once, in the shared rubric partial (principle 4), not repeated across
   skills. Any finding that relies on a default says so
   ("generic default — not declared in local standards") and recommends
   declaring the limit via a standards refresh. Severity rules are unchanged.
4. **One rubric.** Development self-review and code review read the same
   source (local standards, then the shared defaults), so development sees
   exactly the limits review will apply.

## Scope

### A. Bootstrap establishes the limits (product)

- `skills/quality/workflows/generate-standards.md` and
  `skills/quality/context/standards-template.md`: add a required
  **Quality Limits** section to `coding-standards.md` — cyclomatic complexity,
  function length, file length, nesting depth, the project's command that
  measures them, and whether each is tool-enforced or review-enforced.
  Values are derived from existing linter config when present, otherwise
  proposed and confirmed with the user. Reference Configs wire the declared
  limits into whatever linter the project uses.
- Testing Standards section must declare test-isolation and assertion-style
  conventions (content is the project's, the heading is required).
- `research/workflows/bootstrap.md` Phase 4 marker: record whether Quality
  Limits are declared. Existing-marker update mode (Step 0.2) offers to add
  the section for projects bootstrapped before this change.

### B. Review resolves limits local-first (product)

- Replace the scattered hardcoded defaults in `review-code.md`
  (`complexity_thresholds` input and Step 7) and
  `implementation/context/coding-standards.md` with a reference to the shared
  rubric partial, which owns the one copy of the generic defaults.
- Resolution order per limit: local Quality Limits section → generic default.
  Findings cite which source applied. The pre-existing vs new-complexity
  blocker rule is unchanged.
- Defect-class sweep: audit the embedded bundle for any other hardcoded gate
  thresholds or stack-specific gate commands (e.g. coverage "≥ 80%" in
  `implementation/workflows/implement-*.md`) and route each to local
  standards.

### C. Development self-reviews against the same rubric (product)

- Extract the craft checklist from `review-code.md` into one shared partial,
  used by both review and the development prompts.
- `prompts/task/development.md`:
  - Load local standards explicitly (coding-standards incl. Quality Limits and
    Testing Standards, patterns-catalog, tech-stack Quality Gate).
  - Verification by artifact type: TDD for behavior-changing code;
    green-before/green-after for refactors; the TC's planned check for
    prompts/docs/config. No source-text-search tests unless a literal is an
    explicit contract.
  - TC rules: one test per TC (variants separate), TC id in the name, assert
    `Expect` exactly; never edit a TC's Action/Expect — return `blocked`.
  - Reopened by a feature gate: read latest transition note and gate report.
  - Write within declared limits on the first pass; measure with the
    project's declared command.
  - SELF-REVIEW block (shared partial) before exit; fix all findings.
  - Pre-existing gate failure: fix if straightforward, otherwise file a bug
    and the gate still fails (`fail`).
  - Exit gate adds: AC evidence walk, declared limits met, diff within
    declared `files:`.
  - Structured completion summary:
    `ACs verified: <ids>; red-first: N/N; commit: <sha>` (≤175 chars).
- Rewrite `_partials/_tdd_process.md` so it matches the artifact-type split.
- Apply the same blocks to `bug/development.md` and `change/development.md`.

### D. Review findings compound into local standards (product)

- `defect-class-sweep` structural-guard step: a new craft defect class becomes
  a local standards entry, and a linter rule in the project's own config where
  the project's tooling supports it.

### E. Dogfood in this repo (local, not shipped — Go-specific is fine here)

- Add a Quality Gate section to this repo's `docs/architecture/tech-stack.md`
  (currently missing) and a Quality Limits section to
  `docs/architecture/coding-standards.md`, including the testify/require and
  shared-command-state rules B082 hit.
- Measure the current baseline, then enforce in `.golangci.yml`
  (`gocyclo`, `funlen`, possibly `nestif`) with thresholds or `new-from-rev`
  chosen so existing code does not break the gate.

### Out of scope

- Self-routing, `amendment_pending`, and the "already complete → DONE"
  shortcut from the reference prompt (conflicts with `_resume_preamble`).
- Anchored skill includes (`SKILL.md#Section`) — needs a resolver change in
  `internal/templates`; file separately as tech debt.

## Open decision

When the development step finds **no** declared quality-gate commands (no
tech-stack Quality Gate, nothing in CLAUDE.md/AGENTS.md or build targets):
return `blocked` and recommend `/shark-rider project bootstrap`, or proceed
with a noted gap? Proposed: `blocked`. Missing *limits* never block — the
generic defaults apply (principle 3).

## Sizing and task split

Feature is **L** — `task_generation` must split it into tasks of M or smaller.
Suggested split:

| # | Task | Size |
|---|------|------|
| 1 | Quality Limits + Testing Standards in standards template, generate-standards, bootstrap marker and update mode (A) | M |
| 2 | Remove hardcoded thresholds from review and implementation skills; bundle sweep (B) | S |
| 3 | Shared craft-review partial + `task/development.md` rewrite + `_tdd_process` (C) | M |
| 4 | Port development blocks to bug and change prompts (C) | S |
| 5 | Defect-class-sweep structural guard wording (D) | XS |
| 6 | Dogfood: local standards + golangci limits after baseline measurement (E) | S |

## Verification

- Rendered prompts for task, bug and change development contain no language
  or tool names, and generic limit defaults appear only via the shared rubric
  partial (bundle validator / golden tests).
- A project with declared Quality Limits gets its own numbers in both the
  development and review renders; a project without them gets the defaults,
  labeled as such.
- Includes resolve; existing prompt tests pass (`internal/templates`,
  `internal/sharkdata`, `internal/cli/commands/next_test.go`).
- Representative-input check: render the task development prompt for a
  code task and a prompt-only task; confirm the verification branch and
  self-review read local standards.
- Dogfood: `make lint` reports a function over the declared limit before
  review would.
- `make fmt && make lint && make test` green.
