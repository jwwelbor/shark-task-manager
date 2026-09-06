# Tier Matrix

Canonical SIMPLE/STANDARD/COMPLEX artifact and gate contract, consumed by
assessment, task generation, code review, QA, and UAT prompts. This is the
single source of truth for the tier contract — do not restate this table
elsewhere; reference this file by path instead.

## Tier contract

| Tier | Planning source | Test source | Same-model gate | Separate QA | Final UAT |
|---|---|---|---|---|---|
| SIMPLE | `feature.md` and validated `research-report.md` | Inline task ACs and test cases | Combined code review and QA | No | Yes |
| STANDARD | `spec.md` and `test-plan.md` | Test-plan cases and caller paths | Combined code review and QA | No | Yes |
| COMPLEX | `spec.md` and `test-plan.md` | Test-plan cases and caller paths | Craft code review | Yes | Yes |

The tier determines required artifacts and gate division, not a lower evidence standard.
Missing artifacts are failures only when the selected tier requires them.

## Executable gate evidence

Every gate report must cite, per gate:

- The exact command run, verbatim — discovered from project guidance such as
  `docs/architecture/tech-stack.md`'s Quality Gate section or equivalent,
  never a project-specific tool hardcoded into this bundle.
- The working directory the command ran from.
- The exit status.
- Runner-native pass/fail/error/skip counts — not a hand-tallied or
  prose-only total.
- An expected-skip comparison: if the project declares an expected-skip
  list, compare against it; if it does not, state "no expected-skip list
  declared" as its own reportable fact rather than silently ignoring the
  question.
- A bounded pointer to the retained log or artifact, not the full log
  inlined.

A prose-only total, an omitted exit status, a missing declared test case, or
an unexplained unexpected skip fails the gate.

## Pinned E40 benchmark scenarios

Non-blocking note, not a gate: E40 pins four scenario categories against
this tier contract as benchmark follow-up work — tier routing, evidence
fidelity, defect-class recurrence, and integration closure. These scenarios
measure how faithfully a workflow run honors the tier contract above; they
are not a delivery prerequisite for this feature or any feature that
references this file, and their absence never fails a gate.
