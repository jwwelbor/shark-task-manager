# F1.a Pilot — `quality/workflows/qa-testing.md`

**Purpose**: Validate the craft-vs-scaffolding decision rule on a high-leverage, mid-coupled workflow file before scaling to ~30–50 files in F1.b.

**Status**: Draft analysis. Non-destructive — the original `~/.claude/skills/quality/workflows/qa-testing.md` is untouched. These three artifacts are proposals for the user to review.

## Files

- **`qa-testing.craft.md`** — Proposed craft-only version of qa-testing.md, with `inputs:` / `outputs:` frontmatter declaring the host contract.
- **`qa-testing.scaffolding.md`** — Sidecar capturing what was extracted, with each block tagged (`# fetch`, `# gate`, `# mutate`, `# advance`, `# preflight`).
- **`in_qa.prompt.md`** — Sketched feature-level prompt that wraps the craft via `{{include:}}` and provides the inputs the craft expects.

## Pre-flight numbers (verified 2026-05-10)

- File: 596 lines (prompt implied 78 hits — actual = 16 shark mentions)
- Coupling depth: shallow — most "shark" references are example snippets in markdown code fences, not embedded state mutations.
- Sections that are pure craft (no shark coupling): Step 5 (Manual Testing + Frontend Visual Verification), Step 5.5 (E2E Reachability), Step 5.6 (Production Caller Signature Match), Step 5.7 (Codex Red-Team — the methodology), Step 5.8 (Integration Test Assertion Quality), Playwright Test Execution, Common Issues, Quality Checklist.
- Sections with embedded shark coupling: Step 1 (`shark get`), Step 2 (path extraction via `shark get --json | jq`), Step 4 (failure note storage), Step 7 (report path conventions, `shark task note add`, `shark task context set --field bug_fix`), Step 8 (state mutations on bugs found).

## Validation against the rule

> *If you removed this, would the activity still be that activity?*

| Block | Test result | Decision |
|-------|-------------|----------|
| "Read feature PRD for requirements" | Removing it changes the activity — QA without requirements is not QA | CRAFT |
| `shark get E07-F08-001 --json` | Removing it doesn't change *what QA is* — just changes *how the inputs arrive* | SCAFFOLDING |
| "ENUMERATE — DO NOT ITERATE" (codex prompt) | Removing it changes the codex methodology fundamentally | CRAFT |
| "Codex red-team is MANDATORY before advancing" | Removing the *mandate* doesn't change codex methodology — codex still works the same way; what changes is the workflow gate | SCAFFOLDING (gate) |
| "Frontend visual verification — page loaded in browser" | Removing it changes the QA activity — frontend QA without visual verification is incomplete | CRAFT |
| `shark task context set --field bug_fix --value true` | Pure state mutation — changes shark's bug-fix routing, not the QA methodology | SCAFFOLDING |

The rule held cleanly across every section. **Grey zones** I noted:

- **Step 8: "Conditional pass is PROHIBITED"** — feels like a gate but is actually QA-craft methodology (quality gatekeeping principle). Stays in craft.
- **Step 5.7 codex MANDATORY framing** — split: methodology stays in craft; the *mandate* moves to the prompt as a gate.
- **Step 7 report file paths** (`docs/plan/$EPIC_ID/$FEATURE_ID/qa_reports/...`) — these use shark filesystem conventions but the *act of writing a structured report* is craft. Resolution: craft says "write a structured QA report"; prompt provides the path via `qa_report_path` input.

## Calibration findings (for F1.b)

- **Effort estimate revision**: this file took ~1 hour of careful reading + ~30 min of split design. Extrapolate ~1.5 hours per non-trivial workflow file. With 30–50 files in scope, that's ~5–8 working days of focused effort — closer to the prompt's "4–6 days" but on the higher end.
- **Heuristic shortcut works**: the prompt's heuristic (any line touching `shark` CLI, status names, parent-context fetching, note storage, status advancement is scaffolding by default) caught 90%+ of the scaffolding cases. The grey zones (gate-like statements that are actually craft, or vice versa) were rare and the decision test resolved them.
- **The `inputs:` contract was sometimes obvious, sometimes not**: for QA-testing the inputs decomposed cleanly (task_id, spec paths, AC list, frontend boolean, codex command). For specification-writing or assessment, where the activity itself is *constructing* the artifact, the inputs may be more abstract (parent context, prior decisions). Validate this in F1.b.
