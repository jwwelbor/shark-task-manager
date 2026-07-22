# Test plan: E34-F03 — Deliverable feature decomposition and staged integration acceptance

**Updated:** 2026-07-21
**Feature spec:** `spec.md`
**Test scope:** Embedded prompts, skills, and documentation references

## Goal

Confirm that the shipped prompt bundle remains mechanically valid after this
content change. Review the staged-integration policy as prose; do not model it
as application behavior.

## Automated checks

| Test case | Check | Entry point | Expected result |
|---|---|---|---|
| TC-001 | Render changed prompt templates | `templates.NewOrchestratorRenderer` with `goldenVars()` | Each changed template renders with its includes resolved. |
| TC-002 | Render the complete prompt corpus | `TestRenderedPromptsGolden` | The bundle renders deterministically and changed goldens are intentional. |
| TC-003 | Verify documented handoff files | `TestE34F03PromptBundleAndReferences` | The E34 interaction map and E34-F03/E34-F02 feature files exist. |
| TC-004 | Run the repository quality gate | `make fmt`, `make lint`, `make test` | Formatting, linting, and tests pass. |

These checks validate paths, includes, and rendering. They do not assert
individual policy phrases, simulate acceptance decisions, or mutate prompt text
to manufacture a failing test.

## Manual policy review

Review the changed prompt and skill wording against `spec.md` before approval.
Confirm that the change:

- keeps `live` as the default interaction mode;
- describes `contract-only` as a documented handoff, not proof of live wiring;
- preserves security, integrity, and current-feature acceptance gates;
- keeps assessor verdicts separate from owner decisions; and
- assigns activation closure to the feature that owns the live wiring.

This is a human judgment step. It is not a CI decision table.

## Out of scope

- Runtime caller-path tests, API tests, database tests, and workflow-transition
  tests.
- Wording-mutation tests or exhaustive source-to-policy matrices.
- Tests that treat prompts as executable policy engines.

## Approval criteria

Approve when all automated checks pass, the changed-file inventory stays within
the embedded bundle, documentation, prompt-test, and golden-file surfaces, and
the manual policy review finds no material conflict with `spec.md`.
